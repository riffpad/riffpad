package daemon

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

// RelaySession is the session metadata announced to the relay.
type RelaySession struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CLI        string    `json:"cli"`
	Cwd        string    `json:"cwd"`
	Status     string    `json:"status"`
	LastSeenAt time.Time `json:"lastSeenAt,omitempty"`
}

// RelayJoin describes a viewer that connected through the relay.
type RelayJoin struct {
	ViewerID  string
	SessionID string
	DeviceID  string
	Curve     protocol.Curve
	Pub       string
	Eph       string
}

type relayFrame struct {
	Kind      string         `json:"kind"`
	Sessions  []RelaySession `json:"sessions,omitempty"`
	ViewerID  string         `json:"viewerId,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
	DeviceID  string         `json:"deviceId,omitempty"`
	Curve     protocol.Curve `json:"curve,omitempty"`
	Pub       string         `json:"pub,omitempty"`
	Eph       string         `json:"eph,omitempty"`
	Data      string         `json:"data,omitempty"`
}

type relayViewer struct {
	id   string
	recv chan []byte
}

type relayViewerTransport struct {
	c *relayClient
	v *relayViewer
}

func (t *relayViewerTransport) Send(data []byte) error {
	return t.c.sendViewer(t.v.id, data)
}

func (t *relayViewerTransport) Recv() ([]byte, error) {
	data, ok := <-t.v.recv
	if !ok {
		return nil, io.EOF
	}
	return data, nil
}

func (t *relayViewerTransport) Close() error {
	// Ask the relay to close the browser socket as well: otherwise the client
	// would sit on a silent connection until its watchdog fires, unaware that
	// the daemon dropped this viewer (e.g. a critical-event overflow, #173).
	t.c.kickViewer(t.v.id)
	return nil
}

type relayClient struct {
	baseURL string
	hostID  string
	secret  string
	token   string
	log     *log.Logger

	onJoin func(RelayJoin)

	mu           sync.Mutex
	conn         *websocket.Conn
	viewers      map[string]*relayViewer
	lastSessions []RelaySession
}

func newRelayClient(baseURL, hostID, secret string, logger *log.Logger, onJoin func(RelayJoin)) *relayClient {
	return &relayClient{
		baseURL: baseURL,
		hostID:  hostID,
		secret:  secret,
		log:     logger,
		onJoin:  onJoin,
		viewers: map[string]*relayViewer{},
	}
}

// errSuperseded is returned by runOnce when the relay reports that another
// connection with the same host credentials took over. It is process-level
// only: restarting the daemon retries normally.
var errSuperseded = errors.New("host superseded by another connection")

func (c *relayClient) run(ctx context.Context) {
	for {
		err := c.runOnce(ctx)
		if errors.Is(err, errSuperseded) && ctx.Err() == nil {
			// The same hostId+secret connected elsewhere — almost always a
			// config copied to a second machine. Reconnecting would kick the
			// other daemon off and loop forever (#169), so stop here.
			c.log.Printf("relay: this host is already connected elsewhere (superseded); " +
				"not reconnecting — check whether the riffpad config was copied to another machine, " +
				"then restart this daemon")
			return
		}
		if err != nil && ctx.Err() == nil {
			c.log.Printf("relay disconnected: %v (reconnecting in 3s)", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (c *relayClient) runOnce(ctx context.Context) error {
	wsURL := strings.TrimSuffix(c.baseURL, "/") + "/ws/host?hostId=" +
		url.QueryEscape(c.hostID) + "&token=" + url.QueryEscape(c.secret)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial relay: %w", err)
	}
	conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongTimeout()))
	})
	// Ping actively: a half-open uplink (laptop lid closed, network switch)
	// never sends a FIN, so only heartbeat + read deadline reveals it.
	pingDone := make(chan struct{})
	defer close(pingDone)
	go wsPingLoop(conn, pingDone)
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	c.log.Printf("relay connected %s host=%s", c.baseURL, c.hostID)
	c.mu.Lock()
	sessions := append([]RelaySession(nil), c.lastSessions...)
	c.mu.Unlock()
	if len(sessions) > 0 {
		c.announce(sessions)
	}

	superseded := false
readLoop:
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var fr relayFrame
		if json.Unmarshal(data, &fr) != nil {
			continue
		}
		switch fr.Kind {
		case "join":
			if c.onJoin != nil {
				c.onJoin(RelayJoin{
					ViewerID: fr.ViewerID, SessionID: fr.SessionID, DeviceID: fr.DeviceID,
					Curve: fr.Curve, Pub: fr.Pub, Eph: fr.Eph,
				})
			}
		case "leave":
			c.closeViewer(fr.ViewerID)
		case "viewer":
			payload, err := base64.RawStdEncoding.DecodeString(fr.Data)
			if err != nil {
				continue
			}
			c.deliver(fr.ViewerID, payload)
		case protocol.RelayFrameSuperseded:
			superseded = true
			break readLoop
		}
	}
	c.mu.Lock()
	c.conn = nil
	for _, v := range c.viewers {
		close(v.recv)
	}
	c.viewers = map[string]*relayViewer{}
	c.mu.Unlock()
	_ = conn.Close()
	if superseded {
		return errSuperseded
	}
	return fmt.Errorf("relay connection closed")
}

func (c *relayClient) setToken(token string) {
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
}

// login obtains a user session token from the relay and persists it.
func (c *relayClient) login(ctx context.Context, username, password string, persist func(token string)) error {
	httpURL := relayHTTPURL(c.baseURL)
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, httpURL+"/api/auth/login", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login: status %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	c.setToken(out.Token)
	c.log.Printf("logged in to relay as %s", username)
	if persist != nil {
		persist(out.Token)
	}
	return nil
}

// ensureRegistered registers this host with the relay when no per-host secret
// is stored yet, then persists the issued credentials.
func (c *relayClient) ensureRegistered(ctx context.Context, persist func(hostID, secret string)) error {
	c.mu.Lock()
	haveSecret := c.secret != ""
	token := c.token
	c.mu.Unlock()
	if haveSecret {
		return nil
	}
	if token == "" {
		return fmt.Errorf("not logged in to relay (set RIFFPAD_RELAY_USER/PASSWORD or run: riffpad relay login)")
	}
	httpURL := relayHTTPURL(c.baseURL)
	body, _ := json.Marshal(map[string]string{
		"name": c.hostID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, httpURL+"/api/hosts/register", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("register with relay: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register with relay: status %d", resp.StatusCode)
	}
	var out struct {
		HostID     string `json:"hostId"`
		HostSecret string `json:"hostSecret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	c.mu.Lock()
	c.hostID = out.HostID
	c.secret = out.HostSecret
	c.mu.Unlock()
	c.log.Printf("registered with relay host=%s", out.HostID)
	if persist != nil {
		persist(out.HostID, out.HostSecret)
	}
	return nil
}

func relayHTTPURL(wsURL string) string {
	u := strings.TrimSuffix(wsURL, "/")
	u = strings.ReplaceAll(u, "wss://", "https://")
	u = strings.ReplaceAll(u, "ws://", "http://")
	return u
}

func (c *relayClient) viewerTransport(id string) viewerTransport {
	c.mu.Lock()
	v, ok := c.viewers[id]
	if !ok {
		v = &relayViewer{id: id, recv: make(chan []byte, 256)}
		c.viewers[id] = v
	}
	c.mu.Unlock()
	return &relayViewerTransport{c: c, v: v}
}

func (c *relayClient) deliver(id string, data []byte) {
	c.mu.Lock()
	v, ok := c.viewers[id]
	c.mu.Unlock()
	if !ok {
		return
	}
	select {
	case v.recv <- data:
	default:
		// The payload is E2EE, so the daemon cannot tell whether the dropped
		// message was an approval_response: treat any overflow as potentially
		// critical and force the viewer to reconnect and replay (#173).
		c.log.Printf("relay viewer recv buffer full, dropping connection viewer=%s", id)
		c.kickViewer(id)
	}
}

// kickViewer drops a relay viewer locally and asks the relay to close the
// browser connection too, so the client reconnects (and replays history)
// instead of hanging on a dead channel.
func (c *relayClient) kickViewer(id string) {
	_ = c.sendFrame(relayFrame{Kind: protocol.RelayFrameKick, ViewerID: id})
	c.closeViewer(id)
}

func (c *relayClient) closeViewer(id string) {
	c.mu.Lock()
	v, ok := c.viewers[id]
	if ok {
		delete(c.viewers, id)
	}
	c.mu.Unlock()
	if ok {
		close(v.recv)
	}
}

func (c *relayClient) announce(sessions []RelaySession) {
	c.mu.Lock()
	c.lastSessions = append([]RelaySession(nil), sessions...)
	c.mu.Unlock()
	c.sendFrame(relayFrame{Kind: "sessions", Sessions: sessions})
}

func (c *relayClient) sendViewer(id string, data []byte) error {
	return c.sendFrame(relayFrame{
		Kind: "viewer", ViewerID: id, Data: base64.RawStdEncoding.EncodeToString(data),
	})
}

func (c *relayClient) sendFrame(fr relayFrame) error {
	data, err := json.Marshal(fr)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("relay not connected")
	}
	_ = c.conn.SetWriteDeadline(wsWriteDeadline())
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
