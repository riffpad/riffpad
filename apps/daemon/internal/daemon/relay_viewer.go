package daemon

import (
	"encoding/base64"
	"github.com/riffpad/riffpad/packages/protocol"
	"io"
)

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

func (c *relayClient) sendViewer(id string, data []byte) error {
	return c.sendFrame(relayFrame{
		Kind: "viewer", ViewerID: id, Data: base64.RawStdEncoding.EncodeToString(data),
	})
}
