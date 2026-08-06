package daemon

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/riffpad/riffpad/apps/daemon/internal/adapter"
	"github.com/riffpad/riffpad/apps/daemon/internal/codex"
	"github.com/riffpad/riffpad/packages/protocol"
)

// PersistedSession is session state written under dataDir/sessions/<id>/ so a
// daemon restart can bring sessions (and their history) back.
type PersistedSession struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	CLI       string            `json:"cli"`
	Cwd       string            `json:"cwd"`
	Status    string            `json:"status"`
	Ended     bool              `json:"ended"`
	Connect   map[string]string `json:"connect,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func persistSessionDir(dataDir, id string) string {
	return filepath.Join(dataDir, "sessions", id)
}

func persistSessionMeta(dataDir string, ps *PersistedSession) error {
	dir := persistSessionDir(dataDir, ps.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), raw, 0o600)
}

func loadPersistedSessions(dataDir string) ([]PersistedSession, error) {
	base := filepath.Join(dataDir, "sessions")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []PersistedSession
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(base, e.Name(), "meta.json"))
		if err != nil {
			continue
		}
		var ps PersistedSession
		if json.Unmarshal(data, &ps) == nil && ps.ID != "" {
			out = append(out, ps)
		}
	}
	return out, nil
}

// appendSessionEvent appends one AES-GCM encrypted event line to
// sessions/<id>/events.enc (base64(nonce).base64(ciphertext) per line).
func appendSessionEvent(dataDir, id string, key []byte, ev protocol.Event) error {
	if err := os.MkdirAll(persistSessionDir(dataDir, id), 0o700); err != nil {
		return err
	}
	plain, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	gcm, err := aesGCM(key)
	if err != nil {
		return err
	}
	ct := gcm.Seal(nil, nonce, plain, nil)
	line := base64.RawStdEncoding.EncodeToString(nonce) + "." +
		base64.RawStdEncoding.EncodeToString(ct) + "\n"
	f, err := os.OpenFile(filepath.Join(persistSessionDir(dataDir, id), "events.enc"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func loadSessionEvents(dataDir, id string, key []byte) ([]protocol.Event, error) {
	data, err := os.ReadFile(filepath.Join(persistSessionDir(dataDir, id), "events.enc"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	gcm, err := aesGCM(key)
	if err != nil {
		return nil, err
	}
	var out []protocol.Event
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ".", 2)
		if len(parts) != 2 {
			continue
		}
		nonce, err1 := base64.RawStdEncoding.DecodeString(parts[0])
		ct, err2 := base64.RawStdEncoding.DecodeString(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}
		plain, err := gcm.Open(nil, nonce, ct, nil)
		if err != nil {
			continue
		}
		var ev protocol.Event
		if json.Unmarshal(plain, &ev) == nil {
			out = append(out, ev)
		}
	}
	return out, nil
}

func aesGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// restoredAdapter is a placeholder for sessions recovered after a daemon
// restart: history is readable, but the underlying agent process is gone until
// reattachment (M1.4 phase 2).
type restoredAdapter struct {
	id string
}

func (r *restoredAdapter) ID() string { return r.id }

func (r *restoredAdapter) Events() <-chan protocol.Event {
	ch := make(chan protocol.Event)
	close(ch)
	return ch
}

func (r *restoredAdapter) Meta() protocol.SessionStartPayload {
	return protocol.SessionStartPayload{}
}

func (r *restoredAdapter) SendApproval(string, string) error {
	return errors.New("session restored; agent not attached")
}

func (r *restoredAdapter) SendPrompt(string) error {
	return errors.New("session restored; agent not attached")
}

func (r *restoredAdapter) Start(ctx context.Context) error {
	return nil
}

func (r *restoredAdapter) Alive() bool { return false }

func (r *restoredAdapter) Stop() error { return nil }

var _ adapter.Session = (*restoredAdapter)(nil)

// restoreSessions loads persisted sessions (not ended) after a daemon restart
// so history stays available in the client. Agents are not re-attached yet
// (phase 2 of M1.4); sessions are marked "restored" and read-only.
func (s *Server) restoreSessions() {
	persisted, err := loadPersistedSessions(s.dataDir)
	if err != nil {
		s.log.Printf("load persisted sessions: %v", err)
		return
	}
	key, err := s.sessionEncKey()
	if err != nil {
		s.log.Printf("session enc key unavailable: %v", err)
		return
	}
	for _, ps := range persisted {
		if ps.Ended {
			continue
		}
		history, err := loadSessionEvents(s.dataDir, ps.ID, key)
		if err != nil {
			s.log.Printf("load events for %s: %v", ps.ID, err)
			continue
		}
		var sessAdapter adapter.Session = &restoredAdapter{id: ps.ID}
		events := sessAdapter.Events()
		status := "restored"
		if ps.CLI == "codex" && ps.Connect["threadId"] != "" {
			// Phase 2: reattach the real Codex adapter to the persisted thread
			// so the session becomes remotely controllable again.
			ca := codex.New(adapter.CreateRequest{
				ID: ps.ID, Name: ps.Name, CLI: "codex", Cwd: ps.Cwd, DataDir: s.dataDir,
			})
			if err := ca.Restore(ps.Connect["threadId"]); err != nil {
				s.log.Printf("codex restore %s: %v; keeping read-only", ps.ID, err)
			} else {
				sessAdapter = ca
				events = ca.Events()
				status = protocol.StatusRunning
			}
		}
		sess := &session{
			id:      ps.ID,
			meta:    protocol.SessionStartPayload{Name: ps.Name, CLI: ps.CLI, Cwd: ps.Cwd},
			adapter: sessAdapter,
			events:  events,
			status:  status,
			ended:   false,
			created: ps.CreatedAt,
			connect: ps.Connect,
			history: history,
			clients: map[*client]struct{}{},
		}
		s.mu.Lock()
		s.sessions[ps.ID] = sess
		s.mu.Unlock()
		s.log.Printf("restored session %s (%s) with %d events", ps.ID, ps.CLI, len(history))
		if status == protocol.StatusRunning {
			go s.pump(sess)
		}
	}
}

func (s *Server) persistEvent(sess *session, ev protocol.Event) {
	key, err := s.sessionEncKey()
	if err != nil {
		return
	}
	_ = appendSessionEvent(s.dataDir, sess.id, key, ev)
	// Keep meta (status + codex connect info) fresh on every event so a
	// restart can reattach to the latest thread.
	s.persistSession(sess)
}

func (s *Server) persistSession(sess *session) {
	if ci, ok := sess.adapter.(interface {
		CurrentConnect() (socket string, threadID string)
	}); ok {
		if sock, tid := ci.CurrentConnect(); sock != "" && tid != "" {
			sess.connect = map[string]string{"socket": sock, "threadId": tid}
		}
	}
	_ = persistSessionMeta(s.dataDir, &PersistedSession{
		ID:        sess.id,
		Name:      sess.meta.Name,
		CLI:       sess.meta.CLI,
		Cwd:       sess.meta.Cwd,
		Status:    sess.status,
		Ended:     sess.ended,
		Connect:   sess.connect,
		CreatedAt: sess.created,
		UpdatedAt: time.Now(),
	})
}

func (s *Server) sessionEncKey() ([]byte, error) {
	return hex.DecodeString(s.keys.SessionEncKey)
}
