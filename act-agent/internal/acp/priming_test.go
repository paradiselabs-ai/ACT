package acp

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const testHeader = "[ACT priming — do not respond. This is one-time configuration injected by the orchestrator. Acknowledge silently by emitting no text.]\n\n"

func TestStripPrimingWrappers(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "marker + header + text",
			payload: internalPromptMarker + testHeader + "You are the Planner.\n\n[ACT] shim note",
			want:    "You are the Planner.\n\n[ACT] shim note",
		},
		{
			name:    "text without wrappers unchanged",
			payload: "You are the Planner.",
			want:    "You are the Planner.",
		},
		{
			name:    "marker only, no header",
			payload: internalPromptMarker + "You are the Observer.",
			want:    "You are the Observer.",
		},
		{
			name:    "header only, no marker",
			payload: testHeader + "You are the QA synthesizer.",
			want:    "You are the QA synthesizer.",
		},
		{
			name:    "empty stays empty",
			payload: "",
			want:    "",
		},
		{
			name:    "framing with no body yields empty",
			payload: internalPromptMarker + testHeader,
			want:    "",
		},
		{
			name:    "header terminated by single newline",
			payload: internalPromptMarker + "[ACT priming — do not respond.]\nYou are the Assurance role.",
			want:    "You are the Assurance role.",
		},
		{
			name:    "header with no newline at all yields empty",
			payload: internalPromptMarker + "[ACT priming — do not respond.]",
			want:    "",
		},
		{
			name:    "a body line that merely mentions the header text is preserved",
			payload: internalPromptMarker + testHeader + "Never emit [ACT priming — do not respond] yourself.",
			want:    "Never emit [ACT priming — do not respond] yourself.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripPrimingWrappers(c.payload)
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
			if strings.Contains(got, internalPromptMarker) {
				t.Fatalf("InternalPromptMarker survived into the system append: %q", got)
			}
		})
	}
}

func TestPlanPriming(t *testing.T) {
	payload := internalPromptMarker + testHeader + "ROLE TEXT"

	cases := []struct {
		name           string
		host           string
		payload        string
		wantAppend     string
		wantPromptText string
	}{
		{
			name:       "claude-code appends to the system prompt and skips the priming turn",
			host:       "claude-code",
			payload:    payload,
			wantAppend: "ROLE TEXT",
		},
		{
			name:       "empty host spawns claude-code defaults, same treatment",
			host:       "",
			payload:    payload,
			wantAppend: "ROLE TEXT",
		},
		{
			name:           "gemini keeps the priming turn untouched, no _meta",
			host:           "gemini",
			payload:        payload,
			wantPromptText: payload,
		},
		{
			name:           "antigravity keeps the priming turn untouched",
			host:           "antigravity",
			payload:        payload,
			wantPromptText: payload,
		},
		{
			name:           "agy keeps the priming turn untouched",
			host:           "agy",
			payload:        payload,
			wantPromptText: payload,
		},
		{
			name:           "claude-code falls back to priming when stripping empties the payload",
			host:           "claude-code",
			payload:        internalPromptMarker + testHeader,
			wantPromptText: internalPromptMarker + testHeader,
		},
		{
			name:    "no payload, no priming of any kind",
			host:    "claude-code",
			payload: "",
		},
		{
			name:    "no payload on a non-claude host either",
			host:    "gemini",
			payload: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := planPriming(c.host, c.payload)
			if got.SystemAppend != c.wantAppend {
				t.Errorf("SystemAppend: got %q want %q", got.SystemAppend, c.wantAppend)
			}
			if got.PromptText != c.wantPromptText {
				t.Errorf("PromptText: got %q want %q", got.PromptText, c.wantPromptText)
			}
		})
	}
}

// answerFrames replies to every request the Client emits: session/new gets a
// sessionId, session/prompt gets end_turn. Runs until the test ends.
func answerFrames(t *testing.T, tr *memTransport, done <-chan struct{}) {
	t.Helper()
	go func() {
		answered := 0
		for {
			select {
			case <-done:
				return
			default:
			}
			sent := tr.Sent()
			for ; answered < len(sent); answered++ {
				f := sent[answered]
				if f.ID == nil {
					continue
				}
				id := *f.ID
				var result string
				switch f.Method {
				case MethodNewSession:
					result = `{"sessionId":"acp-1"}`
				case MethodPrompt:
					result = `{"stopReason":"end_turn"}`
				default:
					result = `{}`
				}
				tr.Inject(&Frame{JSONRPC: "2.0", ID: &id, Result: json.RawMessage(result)})
			}
			time.Sleep(time.Millisecond)
		}
	}()
}

func newTestACPAgent(host string, priming string) *ACPAgent {
	return &ACPAgent{
		role:        "planner",
		host:        host,
		cmd:         &exec.Cmd{Dir: t0Dir},
		acpSessions: map[string]string{},
		priming:     func(string) string { return priming },
	}
}

// t0Dir keeps ensureACPSession off os.Getwd() so the test is hermetic.
const t0Dir = "/tmp"

func TestEnsureACPSession_ClaudeCodeSendsSystemAppendAndSkipsPriming(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()
	done := make(chan struct{})
	defer close(done)
	answerFrames(t, tr, done)

	a := newTestACPAgent("claude-code", internalPromptMarker+testHeader+"You are the Planner.")
	a.client = c

	id, err := a.ensureACPSession(t.Context(), "act-1")
	if err != nil {
		t.Fatalf("ensureACPSession: %v", err)
	}
	if id != "acp-1" {
		t.Fatalf("sessionId: got %q want %q", id, "acp-1")
	}

	sent := tr.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected exactly 1 frame (session/new, no priming turn), got %d: %+v", len(sent), sent)
	}
	if sent[0].Method != MethodNewSession {
		t.Fatalf("expected session/new, got %q", sent[0].Method)
	}
	var p NewSessionParams
	if err := json.Unmarshal(sent[0].Params, &p); err != nil {
		t.Fatalf("decode session/new params: %v", err)
	}
	if p.Meta == nil || p.Meta.SystemPrompt == nil {
		t.Fatalf("expected _meta.systemPrompt on session/new, got %s", sent[0].Params)
	}
	if p.Meta.SystemPrompt.Append != "You are the Planner." {
		t.Fatalf("append text: got %q", p.Meta.SystemPrompt.Append)
	}
	// Wire-level check: the bridge reads `_meta.systemPrompt.append`, and the
	// marker must never reach a system prompt.
	raw := string(sent[0].Params)
	if !strings.Contains(raw, `"_meta"`) || !strings.Contains(raw, `"systemPrompt"`) {
		t.Fatalf("wire params missing _meta.systemPrompt: %s", raw)
	}
	if strings.Contains(raw, internalPromptMarker) {
		t.Fatalf("InternalPromptMarker leaked onto the wire: %s", raw)
	}
	// Clean-room check: settingSources must serialize as an EMPTY ARRAY (not
	// be omitted) so the bridge's ["user","project","local"] default is
	// overridden and the spawned claude never loads the operator's personal
	// Claude Code config (persona plugins, global CLAUDE.md, auto-memory).
	if !strings.Contains(raw, `"settingSources":[]`) {
		t.Fatalf("wire params missing settingSources:[] override: %s", raw)
	}
}

func TestEnsureACPSession_GeminiPrimesWithoutMeta(t *testing.T) {
	tr := newMemTransport()
	c := NewClient(tr, nil)
	defer c.Close()
	done := make(chan struct{})
	defer close(done)
	answerFrames(t, tr, done)

	payload := internalPromptMarker + testHeader + "You are the Planner."
	a := newTestACPAgent("gemini", payload)
	a.client = c

	if _, err := a.ensureACPSession(t.Context(), "act-1"); err != nil {
		t.Fatalf("ensureACPSession: %v", err)
	}

	sent := tr.Sent()
	if len(sent) != 2 {
		t.Fatalf("expected session/new + priming turn (2 frames), got %d", len(sent))
	}
	if sent[0].Method != MethodNewSession || sent[1].Method != MethodPrompt {
		t.Fatalf("frame order: got %q then %q", sent[0].Method, sent[1].Method)
	}
	// No _meta may reach a non-claude bridge — it may ignore or choke on it.
	if strings.Contains(string(sent[0].Params), "_meta") {
		t.Fatalf("non-claude host received _meta: %s", sent[0].Params)
	}
	var pp PromptParams
	if err := json.Unmarshal(sent[1].Params, &pp); err != nil {
		t.Fatalf("decode session/prompt params: %v", err)
	}
	if len(pp.Prompt) != 1 || pp.Prompt[0].Text != payload {
		t.Fatalf("priming text must be byte-identical to the injector payload, got %+v", pp.Prompt)
	}
}
