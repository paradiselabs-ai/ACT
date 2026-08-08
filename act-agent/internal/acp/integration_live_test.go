//go:build acplive

// Live integration test against the real @agentclientprotocol/claude-agent-acp
// npm-published binary. Gated behind the `acplive` build tag so a normal
// `go test ./...` doesn't spawn a subprocess. Run with:
//
//   go test -tags=acplive -run TestLiveWireSmoke -v ./internal/acp/...
//
// Requires:
// - node + npx on PATH (the test invokes npx to fetch the adapter)
// - network access (first run downloads ~200KB of npm packages)
// - ANTHROPIC_API_KEY or an active Claude Code login (the adapter authenticates
//   on session/prompt, so unauthenticated runs fail at that step — initialize
//   and session/new still succeed)
package acp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func TestLiveWireSmoke(t *testing.T) {
	cmd := exec.Command("npx", "--yes", "-p", "@agentclientprotocol/claude-agent-acp@^0.37", "claude-agent-acp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	defer cmd.Process.Kill()

	tr := NewNewlineTransport(stdout, stdin, stdin)
	var chunkCount atomic.Int64
	var assembled string
	c := NewClient(tr, func(method string, params json.RawMessage) {
		if text, ok := DecodeAgentMessageChunk(params); ok {
			chunkCount.Add(1)
			assembled += text
		}
	})
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Log("STEP 1: initialize")
	res, err := c.Initialize(ctx)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	t.Logf("  agent=%s version=%s protocolVersion=%d",
		res.AgentInfo.Name, res.AgentInfo.Version, res.ProtocolVersion)
	if res.AgentInfo.Name == "" {
		t.Fatalf("empty agent name in initialize response")
	}

	t.Log("STEP 2: session/new")
	cwd, _ := os.Getwd()
	sid, err := c.NewSession(ctx, cwd, nil, nil)
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}
	t.Logf("  sessionId=%s", sid)
	if sid == "" {
		t.Fatalf("empty sessionId")
	}

	t.Log("STEP 3: session/prompt")
	t0 := time.Now()
	stop, err := c.Prompt(ctx, sid, "Reply with exactly the two characters: OK")
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	elapsed := time.Since(t0)
	t.Logf("  stopReason=%s elapsed=%v chunks=%d assembled=%q",
		stop, elapsed, chunkCount.Load(), assembled)

	if chunkCount.Load() == 0 {
		t.Fatalf("expected at least one agent_message_chunk, got zero")
	}
	if stop == "" {
		t.Fatalf("empty stopReason")
	}
}
