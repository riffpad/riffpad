package daemon

import (
	"encoding/json"
	"log"

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
	log       *log.Logger
}

func (c *client) writeLoop() {
	for {
		select {
		case data := <-c.send:
			if err := c.transport.Send(data); err != nil {
				c.log.Printf("ws write error device=%s: %v", c.deviceID, err)
				close(c.done)
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
