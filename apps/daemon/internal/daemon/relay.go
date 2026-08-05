package daemon

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riffpad/riffpad/packages/protocol"
)

// RelaySession is the session metadata announced to the relay.
type RelaySession struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	CLI    string `json:"cli"`
	Cwd    string `json:"cwd"`
	Status string `json:"status"`
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
	t.c.closeViewer(t.v.id)
	return nil
}

type relayClient struct {
	baseURL string
	hostID  string
	token   string
	log     *log.Logger

	onJoin func(RelayJoin)

	mu      sync.Mutex
	conn    *websocket.Conn
	viewers map[string]*relayViewer
}

func newRelayClient(baseURL, hostID, token string, logger *log.Logger, onJoin func(RelayJoin)) *relayClient {
	return &relayClient{
		baseURL: baseURL,
		hostID:  hostID,
		token:   token,
		log:     logger,
		onJoin:  onJoin,
		viewers: map[string]*relayViewer{},
	}
}

func (c *relayClient) run(ctx context.Context) {
	for {
		if err := c.runOnce(ctx); err != nil && ctx.Err() == nil {
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
		url.QueryEscape(c.hostID) + "&token=" + url.QueryEscape(c.token)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial relay: %w", err)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	c.log.Printf("relay connected %s host=%s", c.baseURL, c.hostID)

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
	return fmt.Errorf("relay connection closed")
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
	}
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
	return c.conn.WriteMessage(websocket.TextMessage, data)
}
