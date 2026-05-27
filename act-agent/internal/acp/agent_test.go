package acp

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeChunkSink stands in for the message-write path. The real ACPAgent
// accumulates streamed text into a message.Message via the message.Service;
// re-running that here would pull in the database layer. Instead, we drive
// the chunk-routing code path directly through a Client + memTransport (from
// client_test.go) and assert the assembled text by listening on the
// notification handler the ACPAgent installs.
//
// The point of this test is not to re-test message.Service — it's to verify
// that `session/update` notifications with `agent_message_chunk` content get
// correctly extracted by DecodeAgentMessageChunk.
func TestDecodeAgentMessageChunk(t *testing.T) {
	cases := []struct {
		name     string
		params   string
		wantText string
		wantOK   bool
	}{
		{
			name:     "happy path",
			params:   `{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hello"}}}`,
			wantText: "hello",
			wantOK:   true,
		},
		{
			name:   "wrong update variant",
			params: `{"sessionId":"s1","update":{"sessionUpdate":"tool_call","toolCallId":"t1"}}`,
			wantOK: false,
		},
		{
			name:   "non-text content",
			params: `{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"image","data":"abc"}}}`,
			wantOK: false,
		},
		{
			name:   "malformed update",
			params: `{"sessionId":"s1","update":"not-an-object"}`,
			wantOK: false,
		},
		{
			name:   "malformed outer",
			params: `not even json`,
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text, ok := DecodeAgentMessageChunk(json.RawMessage(c.params))
			if ok != c.wantOK {
				t.Fatalf("ok mismatch: got %v want %v", ok, c.wantOK)
			}
			if text != c.wantText {
				t.Fatalf("text mismatch: got %q want %q", text, c.wantText)
			}
		})
	}
}

func TestClient_StreamingChunksAccumulate(t *testing.T) {
	// Drive the full client-side flow: send a prompt, agent streams three
	// session/update chunks, then sends the prompt response. Assert that all
	// three chunks arrived to the notification handler in order and that
	// Call() returned with the expected stop reason.
	tr := newMemTransport()

	var mu sync.Mutex
	var got []string

	c := NewClient(tr, func(method string, params json.RawMessage) {
		if method != NotifSessionUpdate {
			return
		}
		text, ok := DecodeAgentMessageChunk(params)
		if !ok {
			return
		}
		mu.Lock()
		got = append(got, text)
		mu.Unlock()
	})
	defer c.Close()

	// Background: respond to the prompt with three chunks + a final response.
	go func() {
		for {
			sent := tr.Sent()
			if len(sent) == 0 {
				time.Sleep(2 * time.Millisecond)
				continue
			}
			req := sent[len(sent)-1]
			if req.Method != MethodPrompt {
				time.Sleep(2 * time.Millisecond)
				continue
			}
			// Three streaming chunks first.
			for _, d := range []string{"Hello", " ", "world"} {
				tr.Inject(&Frame{
					JSONRPC: "2.0",
					Method:  NotifSessionUpdate,
					Params: json.RawMessage(`{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"` + d + `"}}}`),
				})
			}
			// Then the prompt response.
			id := *req.ID
			tr.Inject(&Frame{
				JSONRPC: "2.0", ID: &id,
				Result: json.RawMessage(`{"stopReason":"end_turn"}`),
			})
			return
		}
	}()

	stop, err := c.Prompt(t.Context(), "s1", "say hello")
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if stop != StopReasonEndTurn {
		t.Fatalf("stop reason: got %q want %q", stop, StopReasonEndTurn)
	}

	// Give the notification handler a moment to drain.
	deadline := time.Now().Add(time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d chunks observed before deadline", n)
		}
		time.Sleep(2 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	assembled := strings.Join(got, "")
	if assembled != "Hello world" {
		t.Fatalf("assembled text: got %q want %q", assembled, "Hello world")
	}
}

func TestACPAgent_RebindSystemPromptDiscardsSessions(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()

	a := &ACPAgent{
		role:         "planner",
		client:       c,
		acpSessions:  map[string]string{"act-1": "acp-A", "act-2": "acp-B"},
	}

	if err := a.RebindSystemPrompt(); err != nil {
		t.Fatalf("RebindSystemPrompt: %v", err)
	}
	if len(a.acpSessions) != 0 {
		t.Fatalf("expected acpSessions emptied, got %d entries", len(a.acpSessions))
	}
	// Both ACP session IDs should have been Cancel'd. Order is map-iteration so
	// we check the set, not the sequence.
	sent := tr.Sent()
	got := map[string]bool{}
	for _, f := range sent {
		if f.Method != MethodCancel {
			t.Fatalf("expected only cancel frames, got method=%q", f.Method)
		}
		var p struct {
			SessionID string `json:"sessionId"`
		}
		if err := json.Unmarshal(f.Params, &p); err != nil {
			t.Fatalf("decode cancel params: %v", err)
		}
		got[p.SessionID] = true
	}
	if !got["acp-A"] || !got["acp-B"] {
		t.Fatalf("expected cancel for both acp-A and acp-B, got %v", got)
	}
}

func TestACPAgent_RebindSystemPromptRefusesWhenBusy(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()

	a := &ACPAgent{
		role:        "planner",
		client:      c,
		acpSessions: map[string]string{"act-1": "acp-A"},
	}
	// Mark a turn in flight.
	a.activeRequests.Store("act-1", func() {})

	err := a.RebindSystemPrompt()
	if err == nil {
		t.Fatalf("expected error when busy, got nil")
	}
	if !strings.Contains(err.Error(), "cannot rebind") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.acpSessions) != 1 {
		t.Fatalf("sessions must NOT be discarded on busy refusal, got %d", len(a.acpSessions))
	}
	if len(tr.Sent()) != 0 {
		t.Fatalf("expected no cancel frames on busy refusal, got %d", len(tr.Sent()))
	}
}

func TestClient_PromptCancelEmitsNotification(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()

	if err := c.Cancel("s1"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	sent := tr.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(sent))
	}
	if sent[0].Method != MethodCancel {
		t.Fatalf("wrong method: %q", sent[0].Method)
	}
	if sent[0].ID != nil {
		t.Fatalf("cancel must be a notification, got ID=%v", *sent[0].ID)
	}
}
