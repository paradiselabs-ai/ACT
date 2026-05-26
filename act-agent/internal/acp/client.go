package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// NotificationHandler is invoked from the reader goroutine for every
// notification the agent sends. Implementations must not block — heavy work
// belongs on the consumer side of a channel the handler writes to.
type NotificationHandler func(method string, params json.RawMessage)

// ErrClientClosed is returned by Call/Notify after Close has been called.
var ErrClientClosed = errors.New("acp client: closed")

// Client owns a Transport, multiplexes outgoing requests against incoming
// responses by JSON-RPC id, and dispatches notifications to a single
// handler. One Client serves one ACP session lifecycle (one
// initialize/session-new/prompt-loop/close cycle).
type Client struct {
	transport Transport
	notify    NotificationHandler

	// nextID is the JSON-RPC id allocator. Atomic for lock-free Call().
	nextID atomic.Int64

	pendingMu sync.Mutex
	pending   map[int]chan *Frame

	closeOnce sync.Once
	done      chan struct{}    // closed when the reader loop exits
	readErr   atomic.Value     // last reader error, exposed via Err()
}

// NewClient constructs a Client over the given Transport. The notify handler
// receives session/update and any other notifications; pass nil to drop
// notifications (rare — the alpha always wants to consume agent_message_chunk).
//
// The reader goroutine starts immediately; the caller must eventually call
// Close to release resources.
func NewClient(t Transport, notify NotificationHandler) *Client {
	c := &Client{
		transport: t,
		notify:    notify,
		pending:   make(map[int]chan *Frame),
		done:      make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// Call sends a JSON-RPC request and waits for the matching response. Returns
// the response Result on success, or the RPCError as a Go error.
//
// Context cancellation aborts the wait but the request remains in flight on
// the wire — the agent may still send a response we then drop. Callers who
// need true cancellation must follow up with a transport-level Close.
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	select {
	case <-c.done:
		return nil, c.readErrOrClosed()
	default:
	}

	id := int(c.nextID.Add(1))
	ch := make(chan *Frame, 1)

	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("acp call %s: marshal params: %w", method, err)
		}
		raw = b
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  raw,
	}
	if err := c.transport.WriteFrame(req); err != nil {
		return nil, fmt.Errorf("acp call %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, c.readErrOrClosed()
	case f := <-ch:
		if f.Error != nil {
			return nil, f.Error
		}
		return f.Result, nil
	}
}

// Notify sends a JSON-RPC notification (no response expected).
func (c *Client) Notify(method string, params any) error {
	select {
	case <-c.done:
		return c.readErrOrClosed()
	default:
	}

	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("acp notify %s: marshal params: %w", method, err)
		}
		raw = b
	}
	return c.transport.WriteFrame(Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  raw,
		// ID intentionally nil — that's what distinguishes a notification.
	})
}

// Close signals the agent to wind down (by closing stdin), waits for the
// reader loop to exit, and rejects any further Call/Notify. Safe to call
// multiple times.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.transport.Close()
		<-c.done
	})
	return err
}

// Done returns a channel closed when the reader loop has exited (either the
// agent closed stdout cleanly, or an error occurred — check Err()).
func (c *Client) Done() <-chan struct{} { return c.done }

// Err returns the reader-loop terminating error, or nil if the loop is still
// running. Returns nil after a clean Close as well.
func (c *Client) Err() error {
	if v := c.readErr.Load(); v != nil {
		if e, ok := v.(error); ok {
			return e
		}
	}
	return nil
}

// readErrOrClosed returns the reader-loop error if any, otherwise the closed
// sentinel. Used as the error returned from Call/Notify after Close.
func (c *Client) readErrOrClosed() error {
	if e := c.Err(); e != nil {
		return e
	}
	return ErrClientClosed
}

// readLoop pulls frames off the transport until error or EOF. Responses are
// routed to the pending-map waiter; notifications are dispatched to the
// handler. Unknown frames are ignored (with the error path recording context
// so a caller can debug later).
func (c *Client) readLoop() {
	defer close(c.done)
	for {
		f, err := c.transport.ReadFrame()
		if err != nil {
			c.readErr.Store(err)
			c.failPending(err)
			return
		}
		switch {
		case f.IsResponse():
			c.dispatchResponse(f)
		case f.IsNotification():
			if c.notify != nil {
				c.notify(f.Method, f.Params)
			}
		default:
			// A request from the agent — the alpha doesn't handle these (no
			// host-side tools exposed). Record and drop. Future work can route
			// session/request_permission etc. here.
			c.readErr.Store(fmt.Errorf("acp client: ignoring agent-initiated request %q (id=%v)", f.Method, f.ID))
		}
	}
}

func (c *Client) dispatchResponse(f *Frame) {
	if f.ID == nil {
		return
	}
	c.pendingMu.Lock()
	ch, ok := c.pending[*f.ID]
	c.pendingMu.Unlock()
	if !ok {
		return
	}
	// Buffered channel of size 1 — never blocks even if the caller has gone
	// away (defer-cleanup removes the entry, this branch lands on dead ones).
	select {
	case ch <- f:
	default:
	}
}

// failPending wakes up every outstanding Call with the terminating error so
// callers don't block forever after the agent dies.
func (c *Client) failPending(err error) {
	c.pendingMu.Lock()
	pending := c.pending
	c.pending = make(map[int]chan *Frame)
	c.pendingMu.Unlock()
	for id, ch := range pending {
		// Synthesise an RPCError so the Call return shape is unchanged.
		select {
		case ch <- &Frame{
			JSONRPC: "2.0",
			ID:      &id,
			Error:   &RPCError{Code: -32099, Message: err.Error()},
		}:
		default:
		}
	}
}
