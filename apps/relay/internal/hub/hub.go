// Package hub implements the relay: user accounts, host/device registration,
// session routing and encrypted-envelope forwarding between hosts and viewers.
package hub

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

const version = "0.2.0"

type hostConn struct {
	id   string
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
	// writeMu serializes direct writes with writeLoop so a final control
	// frame (superseded) can be delivered synchronously before the
	// connection is closed.
	writeMu sync.Mutex
}

type viewerConn struct {
	id        string
	deviceID  string
	sessionID string
	host      *hostConn
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	once      sync.Once
}

func (h *hostConn) closeDone() {
	h.once.Do(func() { close(h.done) })
}

// writeFrame writes one frame directly on the connection, serialized with
// writeLoop. Used for the final superseded notice, which must go out before
// the connection is closed (queueing it on send would race with done).
func (h *hostConn) writeFrame(fr hostFrame) {
	data, err := json.Marshal(fr)
	if err != nil {
		return
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	_ = h.conn.SetWriteDeadline(wsWriteDeadline())
	_ = h.conn.WriteMessage(websocket.TextMessage, data)
}

func (v *viewerConn) closeDone() {
	v.once.Do(func() { close(v.done) })
}

type Hub struct {
	log     *log.Logger
	dataDir string
	store   *Store

	mu             sync.Mutex
	hosts          map[string]*hostConn
	sessions       map[string]SessionMeta
	sessionHosts   map[string]string
	viewers        map[string]*viewerConn
	rateLimits     map[string]ipCounter
	oauthStates    map[string]oauthState
	deviceLogins   map[string]*deviceLogin
	githubID       string
	githubSecret   string
	githubTokenURL string
	githubUserURL  string
	appURL         string
	apiHosts       []string
	unsubSecret    string
	adminKey       string
	webOrigins     []string
}

func New(logger *log.Logger, dataDir, databaseURL string) (*Hub, error) {
	store, err := OpenStore(dataDir, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Hub{
		log:            logger,
		dataDir:        dataDir,
		store:          store,
		hosts:          map[string]*hostConn{},
		sessions:       map[string]SessionMeta{},
		sessionHosts:   map[string]string{},
		viewers:        map[string]*viewerConn{},
		rateLimits:     map[string]ipCounter{},
		oauthStates:    map[string]oauthState{},
		deviceLogins:   map[string]*deviceLogin{},
		githubID:       os.Getenv("GITHUB_CLIENT_ID"),
		githubSecret:   os.Getenv("GITHUB_CLIENT_SECRET"),
		githubTokenURL: "https://github.com/login/oauth/access_token",
		githubUserURL:  "https://api.github.com/user",
		appURL:         envOr("RIFFPAD_APP_URL", "https://app.riffpad.ai"),
		apiHosts:       splitCSV(os.Getenv("RIFFPAD_API_HOSTS")),
		unsubSecret:    os.Getenv("UNSUBSCRIBE_SECRET"),
		adminKey:       os.Getenv("WAITLIST_ADMIN_KEY"),
		webOrigins:     splitCSV(envOr("RIFFPAD_WEB_ORIGINS", "https://riffpad.ai,https://www.riffpad.ai")),
	}, nil
}

func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/assets/", h.handleAsset)
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/auth/register", h.handleRegister)
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/logout", h.handleLogout)
	mux.HandleFunc("/api/auth/me", h.handleMe)
	mux.HandleFunc("/api/auth/github/login", h.handleGitHubLogin)
	mux.HandleFunc("/api/auth/github/callback", h.handleGitHubCallback)
	mux.HandleFunc("/api/auth/oauth/device", h.handleDeviceLogin)
	mux.HandleFunc("/api/auth/oauth/device/poll", h.handleDeviceLoginPoll)
	mux.HandleFunc("/api/auth/oauth/device/status", h.handleDeviceLoginStatus)
	mux.HandleFunc("/api/hosts", h.handleHosts)
	mux.HandleFunc("/api/hosts/register", h.handleRegisterHost)
	mux.HandleFunc("/api/pairings", h.handleCreatePairing)
	mux.HandleFunc("/api/pair", h.handlePair)
	mux.HandleFunc("/api/devices", h.handleDevices)
	mux.HandleFunc("/api/devices/", h.handleDeviceDelete)
	mux.HandleFunc("/api/hosts/", h.handleHostKillswitch)
	mux.HandleFunc("/api/sessions", h.handleSessions)
	mux.HandleFunc("/api/sessions/", h.handleSessionMeta)
	mux.HandleFunc("/api/waitlist/subscribe", h.handleWaitlistSubscribe)
	mux.HandleFunc("/api/waitlist/emails", h.handleWaitlistEmails)
	mux.HandleFunc("/api/waitlist/unsubscribe", h.handleWaitlistUnsubscribe)
	mux.HandleFunc("/api/waitlist/optouts", h.handleWaitlistOptouts)
	mux.HandleFunc("/ws/host", h.handleHostWS)
	mux.HandleFunc("/ws", h.handleViewerWS)
	return mux
}

// ---------- hosts / pairing / sessions ----------

// ---------- host connection ----------

type hostFrame struct {
	Kind string `json:"kind"`

	Sessions []SessionMeta `json:"sessions,omitempty"`

	ViewerID  string `json:"viewerId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	DeviceID  string `json:"deviceId,omitempty"`
	Curve     string `json:"curve,omitempty"`
	Pub       string `json:"pub,omitempty"`
	Eph       string `json:"eph,omitempty"`
	Data      string `json:"data,omitempty"`
}

func (h *Hub) handleHostWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hostID := q.Get("hostId")
	token := q.Get("token")
	rec, err := h.store.GetHost(hostID)
	if err != nil || token != rec.Secret {
		writeError(w, http.StatusUnauthorized, "bad host credentials")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	host := &hostConn{id: hostID, conn: conn, send: make(chan []byte, 256), done: make(chan struct{})}
	h.mu.Lock()
	if old, ok := h.hosts[hostID]; ok {
		// Tell the replaced connection why it is being dropped: two daemons
		// sharing the same host credentials would otherwise kick each other
		// in an endless reconnect loop (#169).
		old.writeFrame(hostFrame{Kind: protocol.RelayFrameSuperseded})
		old.closeDone()
		_ = old.conn.Close()
	}
	h.hosts[hostID] = host
	h.mu.Unlock()
	h.log.Printf("host connected id=%s", hostID)
	go host.writeLoop()
	go h.hostReadLoop(host)
}

func (h *Hub) hostReadLoop(host *hostConn) {
	defer func() {
		h.removeHost(host)
		host.closeDone()
		_ = host.conn.Close()
	}()
	host.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	host.conn.SetPongHandler(func(string) error {
		return host.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	for {
		_, data, err := host.conn.ReadMessage()
		if err != nil {
			return
		}
		var fr hostFrame
		if json.Unmarshal(data, &fr) != nil {
			continue
		}
		switch fr.Kind {
		case "sessions":
			h.mu.Lock()
			if h.hosts[host.id] == host {
				// Replace only this host's entries: other hosts of the same
				// account (desktop + laptop) keep their sessions (#169). The
				// current-connection check also rejects a stale announce from
				// a superseded connection still draining its read loop.
				for sid, hid := range h.sessionHosts {
					if hid == host.id {
						delete(h.sessionHosts, sid)
						delete(h.sessions, sid)
					}
				}
				for _, s := range fr.Sessions {
					s.HostID = host.id
					if s.LastSeenAt.IsZero() {
						// Older daemons announce without a timestamp; these
						// sessions are live right now, so stamp them here
						// (mirrors Store.UpsertSessions). Otherwise the
						// in-memory snapshot would report Go's zero time and
						// clients would show "739835d ago".
						s.LastSeenAt = time.Now()
					}
					h.sessions[s.ID] = s
					h.sessionHosts[s.ID] = host.id
				}
			}
			h.mu.Unlock()
			_ = h.store.UpsertSessions(host.id, fr.Sessions)
		case "viewer":
			h.mu.Lock()
			v, ok := h.viewers[fr.ViewerID]
			h.mu.Unlock()
			if !ok {
				continue
			}
			payload, err := base64.RawStdEncoding.DecodeString(fr.Data)
			if err != nil {
				continue
			}
			h.sendToViewer(v, payload)
		case protocol.RelayFrameKick:
			// The daemon dropped this viewer (e.g. send-buffer overflow on a
			// critical event, #173): close the browser connection so the
			// client reconnects and replays instead of hanging silently.
			h.mu.Lock()
			v, ok := h.viewers[fr.ViewerID]
			h.mu.Unlock()
			if ok {
				h.dropViewer(v, "kicked by host")
			}
		}
	}
}

func (h *Hub) removeHost(host *hostConn) {
	h.mu.Lock()
	// Only the connection currently registered under this host id may tear
	// down its sessions: after a reconnect the old connection's deferred
	// cleanup runs after the new one has already re-announced, and must not
	// wipe those fresh entries (#169).
	current := h.hosts[host.id] == host
	if current {
		delete(h.hosts, host.id)
		for sid, hid := range h.sessionHosts {
			if hid == host.id {
				delete(h.sessionHosts, sid)
				delete(h.sessions, sid)
			}
		}
	}
	for id, v := range h.viewers {
		if v.host == host {
			delete(h.viewers, id)
			v.closeDone()
			_ = v.conn.Close()
		}
	}
	h.mu.Unlock()
	if current {
		_ = h.store.MarkHostSessionsOffline(host.id)
	}
	h.log.Printf("host disconnected id=%s", host.id)
}

// ---------- viewer connection ----------

func (h *Hub) handleViewerWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sid := q.Get("session")
	deviceID := q.Get("device")
	eph := q.Get("eph")
	token := q.Get("token")
	if sid == "" || deviceID == "" || eph == "" || token == "" {
		writeError(w, http.StatusBadRequest, "session, device, eph and token are required")
		return
	}
	u, err := h.store.UserByToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	dev, err := h.store.GetDevice(deviceID)
	if err != nil || dev.OwnerID != u.ID {
		writeError(w, http.StatusUnauthorized, "device not paired for this user")
		return
	}
	host := func() *hostConn {
		h.mu.Lock()
		defer h.mu.Unlock()
		hostID, ok := h.sessionHosts[sid]
		if !ok {
			return nil
		}
		hostRec, err := h.store.GetHost(hostID)
		if err != nil || hostRec.OwnerID != u.ID {
			return nil
		}
		return h.hosts[hostID]
	}()
	if host == nil {
		writeError(w, http.StatusNotFound, "session offline")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	v := &viewerConn{
		id: protocol.NewID(), deviceID: deviceID, sessionID: sid, host: host,
		conn: conn, send: make(chan []byte, 256), done: make(chan struct{}),
	}
	h.mu.Lock()
	h.viewers[v.id] = v
	h.mu.Unlock()
	h.hostSend(host, hostFrame{
		Kind: "join", ViewerID: v.id, SessionID: sid, DeviceID: deviceID,
		Curve: dev.Curve, Pub: dev.PublicKey, Eph: eph,
	})
	go v.writeLoop()
	go h.viewerReadLoop(v)
}

func (h *Hub) viewerReadLoop(v *viewerConn) {
	defer func() {
		h.mu.Lock()
		if h.viewers[v.id] == v {
			delete(h.viewers, v.id)
		}
		h.mu.Unlock()
		h.hostSend(v.host, hostFrame{Kind: "leave", ViewerID: v.id})
		v.closeDone()
		_ = v.conn.Close()
	}()
	v.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	v.conn.SetPongHandler(func(string) error {
		return v.conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	for {
		_, data, err := v.conn.ReadMessage()
		if err != nil {
			return
		}
		h.hostSend(v.host, hostFrame{Kind: "viewer", ViewerID: v.id, Data: base64.RawStdEncoding.EncodeToString(data)})
	}
}

// sendToViewer queues one payload for a viewer. The relay cannot inspect the
// encrypted contents, so an overflow may be dropping a critical event: close
// the connection instead of silently dropping, forcing the client to
// reconnect and replay history (#173).
func (h *Hub) sendToViewer(v *viewerConn, payload []byte) {
	select {
	case v.send <- payload:
	default:
		h.dropViewer(v, "send buffer full")
	}
}

// dropViewer removes a viewer and closes its connection. The viewer's read
// loop notices the close and sends the host a "leave" notice.
func (h *Hub) dropViewer(v *viewerConn, reason string) {
	h.mu.Lock()
	if h.viewers[v.id] != v {
		h.mu.Unlock()
		return
	}
	delete(h.viewers, v.id)
	h.mu.Unlock()
	h.log.Printf("viewer dropped (%s) viewer=%s session=%s host=%s", reason, v.id, v.sessionID, v.host.id)
	v.closeDone()
	if v.conn != nil {
		_ = v.conn.Close()
	}
}

func (h *Hub) hostSend(host *hostConn, fr hostFrame) {
	data, err := json.Marshal(fr)
	if err != nil {
		return
	}
	select {
	case host.send <- data:
	default:
		// Same reasoning as sendToViewer: with no visibility into the payload,
		// any drop may be critical. Closing forces the daemon to reconnect and
		// re-announce, and its viewers to reconnect and replay (#173).
		h.log.Printf("host send buffer full, closing connection host=%s kind=%s", host.id, fr.Kind)
		host.closeDone()
		if host.conn != nil {
			_ = host.conn.Close()
		}
	}
}
