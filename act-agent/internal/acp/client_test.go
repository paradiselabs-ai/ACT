package acp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// memTransport is an in-process Transport for client tests. Test code writes
// frames into the agent-side queue via Inject; the Client reads them via
// ReadFrame. Frames the Client writes are appended to Outgoing for assertion.
type memTransport struct {
	mu       sync.Mutex
	incoming chan *Frame
	outgoing []*Frame
	closed   bool
}

func newMemTransport() *memTransport {
	return &memTransport{incoming: make(chan *Frame, 16)}
}

func (m *memTransport) ReadFrame() (*Frame, error) {
	f, ok := <-m.incoming
	if !ok {
		return nil, io.EOF
	}
	return f, nil
}

func (m *memTransport) WriteFrame(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var f Frame
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	m.mu.Lock()
	m.outgoing = append(m.outgoing, &f)
	m.mu.Unlock()
	return nil
}

func (m *memTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.incoming)
	return nil
}

func (m *memTransport) Inject(f *Frame) {
	m.incoming <- f
}

func (m *memTransport) Sent() []*Frame {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]*Frame, len(m.outgoing))
	copy(cp, m.outgoing)
	return cp
}

func TestClient_CallResponseCorrelation(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()

	// Background: respond to whatever id the client emits.
	go func() {
		// Wait until a request lands.
		for {
			sent := tr.Sent()
			if len(sent) > 0 && sent[0].ID != nil {
				id := *sent[0].ID
				tr.Inject(&Frame{
					JSONRPC: "2.0", ID: &id,
					Result: json.RawMessage(`{"ok":true}`),
				})
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := c.Call(ctx, "initialize", InitializeParams{ProtocolVersion: 1})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(string(res), `"ok":true`) {
		t.Fatalf("unexpected result: %s", res)
	}

	sent := tr.Sent()
	if len(sent) != 1 || sent[0].Method != "initialize" || sent[0].ID == nil {
		t.Fatalf("expected one initialize request with ID, got: %+v", sent)
	}
}

func TestClient_ConcurrentCallsCorrelateByID(t *testing.T) {
	// Verify the pending-map dispatches responses to the right waiter even
	// when multiple Call invocations are in flight and responses arrive out
	// of order.
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()

	// Responder: collect both requests, reply in reverse order.
	go func() {
		seen := make([]int, 0, 2)
		for {
			sent := tr.Sent()
			if len(sent) >= 2 {
				for _, f := range sent {
					if f.ID != nil {
						seen = append(seen, *f.ID)
					}
				}
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
		// Reply 2 first, then 1, embedding the id into the result so we can
		// verify the correct caller got the correct payload.
		for i := len(seen) - 1; i >= 0; i-- {
			id := seen[i]
			tr.Inject(&Frame{
				JSONRPC: "2.0", ID: &id,
				Result: json.RawMessage(`{"echoId":` + string(rune('0'+id)) + `}`),
			})
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	results := make(map[string]string)
	var resultsMu sync.Mutex

	for _, m := range []string{"session/new", "session/prompt"} {
		method := m
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			res, err := c.Call(ctx, method, nil)
			if err != nil {
				t.Errorf("%s: %v", method, err)
				return
			}
			resultsMu.Lock()
			results[method] = string(res)
			resultsMu.Unlock()
		}()
	}
	wg.Wait()

	// Both should have a non-empty echoId — proves no cross-talk.
	for k, v := range results {
		if !strings.Contains(v, "echoId") {
			t.Errorf("%s: missing echoId in %s", k, v)
		}
	}
}

func TestClient_NotificationDispatch(t *testing.T) {
	tr := newMemTransport()
	got := make(chan string, 4)
	c := NewClient(tr, func(method string, params json.RawMessage) {
		got <- method + ":" + string(params)
	})
	defer c.Close()

	tr.Inject(&Frame{
		JSONRPC: "2.0",
		Method:  "session/update",
		Params:  json.RawMessage(`{"sessionId":"s1"}`),
	})

	select {
	case g := <-got:
		if !strings.HasPrefix(g, "session/update:") {
			t.Fatalf("wrong dispatch: %q", g)
		}
	case <-time.After(time.Second):
		t.Fatal("notification handler not invoked")
	}
}

func TestClient_NotifyEmitsNoID(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()

	if err := c.Notify("session/cancel", CancelParams{SessionID: "s1"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	sent := tr.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(sent))
	}
	if sent[0].ID != nil {
		t.Fatalf("notification must not have an ID, got: %v", *sent[0].ID)
	}
	if sent[0].Method != "session/cancel" {
		t.Fatalf("wrong method: %q", sent[0].Method)
	}
}

func TestClient_PendingCallFailsOnClose(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)

	errCh := make(chan error, 1)
	go func() {
		_, err := c.Call(context.Background(), "session/prompt", nil)
		errCh <- err
	}()

	// Let the goroutine queue the request.
	time.Sleep(20 * time.Millisecond)
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error on Call after Close, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Call did not return after Close")
	}
}

func TestClient_RPCErrorSurfacedAsGoError(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()

	go func() {
		for {
			sent := tr.Sent()
			if len(sent) > 0 && sent[0].ID != nil {
				id := *sent[0].ID
				tr.Inject(&Frame{
					JSONRPC: "2.0", ID: &id,
					Error: &RPCError{Code: -32601, Message: "method not found"},
				})
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.Call(ctx, "bogus/method", nil)
	if err == nil {
		t.Fatal("expected RPC error, got nil")
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *RPCError, got %T: %v", err, err)
	}
	if rpcErr.Code != -32601 {
		t.Fatalf("wrong code: %d", rpcErr.Code)
	}
}
