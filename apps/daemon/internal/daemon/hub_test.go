package daemon

import (
	"bytes"
	"io"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/riffpad/riffpad/packages/protocol"
)

// recordingTransport is a viewerTransport fake that records Close calls.
type recordingTransport struct {
	mu     sync.Mutex
	closed int
}

func (t *recordingTransport) Send([]byte) error { return nil }
func (t *recordingTransport) Recv() ([]byte, error) {
	return nil, io.EOF
}
func (t *recordingTransport) Close() error {
	t.mu.Lock()
	t.closed++
	t.mu.Unlock()
	return nil
}

func (t *recordingTransport) closeCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

func newTestClient(buf *bytes.Buffer, sendCap int) (*client, *recordingTransport) {
	tr := &recordingTransport{}
	logger := log.New(buf, "", 0)
	sess := &session{id: "s1", clients: map[*client]struct{}{}}
	key := &[32]byte{}
	c := &client{
		deviceID:  "dev1",
		session:   sess,
		key:       key,
		transport: tr,
		send:      make(chan []byte, sendCap),
		done:      make(chan struct{}),
		log:       logger,
	}
	return c, tr
}

// Issue #173: with a full send buffer a non-critical event is dropped, but
// the drop must be logged (previously fully silent) and must not kill the
// connection.
func TestSendEventDropNonCriticalLogs(t *testing.T) {
	var buf bytes.Buffer
	c, tr := newTestClient(&buf, 1)
	c.send <- []byte("filler") // buffer now full; no writeLoop draining

	ev, err := protocol.NewEvent("s1", protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	c.sendEvent(ev)

	if !strings.Contains(buf.String(), "event dropped") ||
		!strings.Contains(buf.String(), "s1") ||
		!strings.Contains(buf.String(), protocol.EventAgentMessage) {
		t.Fatalf("drop not logged with session/type: %q", buf.String())
	}
	if tr.closeCount() != 0 {
		t.Fatal("non-critical drop must not close the connection")
	}
	select {
	case <-c.done:
		t.Fatal("non-critical drop must not close done")
	default:
	}
}

// Issue #173: a critical event (approval_request / approval_response /
// session_end) must never be silently dropped — the connection is closed so
// the client reconnects and replays history.
func TestSendEventCriticalClosesConnection(t *testing.T) {
	for _, typ := range []string{protocol.EventApprovalReq, protocol.EventApprovalResp, protocol.EventSessionEnd} {
		var buf bytes.Buffer
		c, tr := newTestClient(&buf, 1)
		c.send <- []byte("filler")

		var ev protocol.Event
		var err error
		switch typ {
		case protocol.EventApprovalReq:
			ev, err = protocol.NewEvent("s1", typ, protocol.ApprovalRequestPayload{RequestID: "r1", Options: []string{"approve"}})
		case protocol.EventApprovalResp:
			ev, err = protocol.NewEvent("s1", typ, protocol.ApprovalResponsePayload{RequestID: "r1", Decision: "approve"})
		default:
			ev, err = protocol.NewEvent("s1", typ, protocol.SessionEndPayload{Reason: "done"})
		}
		if err != nil {
			t.Fatal(err)
		}
		c.sendEvent(ev)

		if tr.closeCount() == 0 {
			t.Fatalf("%s: critical event overflow did not close the transport", typ)
		}
		select {
		case <-c.done:
		default:
			t.Fatalf("%s: critical event overflow did not close done", typ)
		}
		if !strings.Contains(buf.String(), "critical event") || !strings.Contains(buf.String(), typ) {
			t.Fatalf("%s: missing warn log: %q", typ, buf.String())
		}
	}
}

// Issue #173: the daemon assigns increasing per-session sequence numbers so
// clients can detect dropped events.
func TestAddEventAssignsIncreasingSeq(t *testing.T) {
	sess := &session{id: "s1"}
	var last uint64
	for i := 0; i < 5; i++ {
		ev, err := protocol.NewEvent("s1", protocol.EventAgentMessage, protocol.AgentMessagePayload{Text: "x"})
		if err != nil {
			t.Fatal(err)
		}
		got := sess.addEvent(ev)
		if got.Seq != last+1 {
			t.Fatalf("seq = %d, want %d", got.Seq, last+1)
		}
		last = got.Seq
		if sess.history[len(sess.history)-1].Seq != got.Seq {
			t.Fatal("history entry missing assigned seq")
		}
	}
}

// Issue #173: relay viewer recv-buffer overflow (viewer → daemon direction,
// potentially an approval_response) drops the viewer and logs it.
func TestDeliverOverflowKicksViewer(t *testing.T) {
	var buf bytes.Buffer
	rc := &relayClient{
		log:     log.New(&buf, "", 0),
		viewers: map[string]*relayViewer{},
	}
	v := &relayViewer{id: "v1", recv: make(chan []byte, 1)}
	rc.viewers["v1"] = v
	v.recv <- []byte("filler") // buffer full

	rc.deliver("v1", []byte("payload"))

	rc.mu.Lock()
	_, ok := rc.viewers["v1"]
	rc.mu.Unlock()
	if ok {
		t.Fatal("overflowing viewer was not dropped")
	}
	<-v.recv // drain the buffered filler
	select {
	case _, open := <-v.recv:
		if open {
			t.Fatal("viewer recv channel not closed")
		}
	default:
		t.Fatal("viewer recv channel not closed")
	}
	if !strings.Contains(buf.String(), "recv buffer full") || !strings.Contains(buf.String(), "v1") {
		t.Fatalf("missing warn log: %q", buf.String())
	}
}
