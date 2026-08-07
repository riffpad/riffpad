package daemon

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/riffpad/riffpad/packages/protocol"
)

// viewerTransport abstracts the connection to a viewer (local WebSocket or a
// viewer routed through the relay).
type viewerTransport interface {
	Send(data []byte) error
	Recv() ([]byte, error)
	Close() error
}

// client is one connected mobile/web session viewer.
type client struct {
	deviceID  string
	session   *session
	key       *[32]byte
	transport viewerTransport
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
	log       *log.Logger
}

// closeDone closes c.done exactly once; both readLoop and writeLoop may
// trigger shutdown concurrently.
func (c *client) closeDone() {
	c.closeOnce.Do(func() { close(c.done) })
}

// appPing is an application-level keepalive. Browser clients cannot see
// protocol-level ping/pong frames, so without this an idle-but-healthy
// connection would look dead to their silence watchdog.
var appPing = []byte(`{"kind":"ping"}`)

func (c *client) writeLoop() {
	ticker := time.NewTicker(wsPingInterval())
	defer ticker.Stop()
	for {
		select {
		case data := <-c.send:
			if err := c.transport.Send(data); err != nil {
				c.log.Printf("ws write error device=%s: %v", c.deviceID, err)
				c.closeDone()
				return
			}
		case <-ticker.C:
			if err := c.transport.Send(appPing); err != nil {
				c.log.Printf("ws ping error device=%s: %v", c.deviceID, err)
				c.closeDone()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *client) sendEvent(ev protocol.Event) {
	plain, err := json.Marshal(ev)
	if err != nil {
		return
	}
	env, err := protocol.WrapEnvelope(ev.SessionID, plain, c.key)
	if err != nil {
		return
	}
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// Slow client: drop rather than block the event pump.
	}
}

func (c *client) sendRaw(data []byte) {
	select {
	case c.send <- data:
	default:
	}
}

func (s *session) addClient(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = struct{}{}
}

func (s *session) removeClient(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, c)
}

func (s *session) broadcast(ev protocol.Event) {
	s.mu.Lock()
	clients := make([]*client, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.Unlock()
	for _, c := range clients {
		c.sendEvent(ev)
	}
}

func (s *session) addEvent(ev protocol.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, ev)
	if len(s.history) > 200 {
		s.history = s.history[len(s.history)-200:]
	}
}

func (s *session) snapshot() []protocol.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]protocol.Event, len(s.history))
	copy(out, s.history)
	return out
}

// snapshotLast returns at most n most recent events, oldest first.
func (s *session) snapshotLast(n int) []protocol.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 || len(s.history) <= n {
		out := make([]protocol.Event, len(s.history))
		copy(out, s.history)
		return out
	}
	out := make([]protocol.Event, n)
	copy(out, s.history[len(s.history)-n:])
	return out
}
