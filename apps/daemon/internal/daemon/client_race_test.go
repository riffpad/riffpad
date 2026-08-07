package daemon

import (
	"errors"
	"io"
	"log"
	"sync"
	"testing"
)

// failTransport simulates a viewer connection that is dead in both
// directions: Send and Recv always fail, as happens when a viewer
// disconnects while an event is in flight.
type failTransport struct {
	sendCalled chan struct{}
	closed     chan struct{}
	closeOnce  sync.Once
}

func newFailTransport() *failTransport {
	return &failTransport{
		sendCalled: make(chan struct{}),
		closed:     make(chan struct{}),
	}
}

func (t *failTransport) Send([]byte) error {
	t.closeOnce.Do(func() { close(t.sendCalled) })
	return errors.New("viewer gone")
}

func (t *failTransport) Recv() ([]byte, error) {
	return nil, errors.New("viewer gone")
}

func (t *failTransport) Close() error {
	close(t.closed)
	return nil
}

// TestClientConcurrentCloseNoPanic reproduces issue #163: readLoop and
// writeLoop both try to close c.done when a viewer disconnects mid-write.
// Without the sync.Once guard this panics with "close of closed channel".
func TestClientConcurrentCloseNoPanic(t *testing.T) {
	for i := 0; i < 100; i++ {
		tr := newFailTransport()
		sess := &session{id: "s1", clients: map[*client]struct{}{}}
		c := &client{
			deviceID:  "dev1",
			session:   sess,
			transport: tr,
			send:      make(chan []byte, 1),
			done:      make(chan struct{}),
			log:       log.New(io.Discard, "", 0),
		}
		c.send <- []byte("event")

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.writeLoop()
		}()
		// Wait until writeLoop's Send has failed, then unleash readLoop so
		// both goroutines race to close c.done.
		<-tr.sendCalled
		go func() {
			defer wg.Done()
			c.readLoop(&Server{})
		}()
		wg.Wait()
	}
}
