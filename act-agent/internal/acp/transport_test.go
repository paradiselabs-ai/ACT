package acp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestNewlineTransport_RoundTripRequest(t *testing.T) {
	var buf bytes.Buffer
	tr := NewNewlineTransport(strings.NewReader(""), &buf, nil)

	id := 7
	if err := tr.WriteFrame(Request{
		JSONRPC: "2.0", ID: &id, Method: "session/prompt",
		Params: json.RawMessage(`{"sessionId":"s1"}`),
	}); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	got := buf.String()
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("frame not newline-terminated: %q", got)
	}
	var back Frame
	if err := json.Unmarshal([]byte(strings.TrimRight(got, "\n")), &back); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if back.JSONRPC != "2.0" || back.Method != "session/prompt" || back.ID == nil || *back.ID != 7 {
		t.Fatalf("unexpected round-trip: %+v", back)
	}
}

func TestNewlineTransport_ReadResponse(t *testing.T) {
	line := `{"jsonrpc":"2.0","id":3,"result":{"stopReason":"end_turn"}}` + "\n"
	tr := NewNewlineTransport(strings.NewReader(line), io.Discard, nil)

	f, err := tr.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !f.IsResponse() {
		t.Fatalf("expected response, got: %+v", f)
	}
	if f.ID == nil || *f.ID != 3 {
		t.Fatalf("wrong id: %+v", f.ID)
	}
	if !strings.Contains(string(f.Result), "end_turn") {
		t.Fatalf("missing stopReason in result: %s", f.Result)
	}
}

func TestNewlineTransport_ReadNotification(t *testing.T) {
	line := `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hi"}}}}` + "\n"
	tr := NewNewlineTransport(strings.NewReader(line), io.Discard, nil)

	f, err := tr.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !f.IsNotification() {
		t.Fatalf("expected notification, got: %+v", f)
	}
	if f.Method != "session/update" {
		t.Fatalf("wrong method: %q", f.Method)
	}
}

func TestNewlineTransport_RejectsBadJSONRPC(t *testing.T) {
	// jsonrpc field is "1.0" — our client rejects everything but "2.0".
	line := `{"jsonrpc":"1.0","id":1,"result":{}}` + "\n"
	tr := NewNewlineTransport(strings.NewReader(line), io.Discard, nil)
	_, err := tr.ReadFrame()
	if err == nil {
		t.Fatal("expected error for jsonrpc=1.0, got nil")
	}
}

func TestNewlineTransport_EOF(t *testing.T) {
	tr := NewNewlineTransport(strings.NewReader(""), io.Discard, nil)
	_, err := tr.ReadFrame()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF on empty reader, got: %v", err)
	}
}

func TestNewlineTransport_SkipsBlankLines(t *testing.T) {
	// Robustness — a stray blank line shouldn't kill the client. The spec
	// doesn't allow them but live agents have been known to emit them
	// when interleaving stderr/stdout sloppily.
	src := "\n" + `{"jsonrpc":"2.0","id":1,"result":{}}` + "\n"
	tr := NewNewlineTransport(strings.NewReader(src), io.Discard, nil)
	f, err := tr.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !f.IsResponse() {
		t.Fatalf("expected response, got: %+v", f)
	}
}
