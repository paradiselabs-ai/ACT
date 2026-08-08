package acp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Initialize issues the initialize request and returns the agent's response.
// Must be the first request on a Client — the agent rejects everything else
// until it has seen one.
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	raw, err := c.Call(ctx, MethodInitialize, InitializeParams{
		ProtocolVersion:    ProtocolVersion,
		ClientCapabilities: ClientCapabilities{},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}
	var res InitializeResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("initialize: decode result: %w", err)
	}
	if res.ProtocolVersion != ProtocolVersion {
		// Tolerate but record — a minor version skew is fine in practice. A
		// major skew (e.g. ACP v2) would require schema changes here, in
		// which case fail loudly so the user upgrades ACT.
		if res.ProtocolVersion > ProtocolVersion+1 {
			return nil, fmt.Errorf("initialize: agent protocolVersion %d unsupported (client supports up to %d)",
				res.ProtocolVersion, ProtocolVersion+1)
		}
	}
	return &res, nil
}

// NewSession requests a fresh session from the agent. cwd is the working
// directory for any tools the agent runs. mcpServers is the list of MCP
// servers to expose to the agent — empty for the alpha (the shim-binary
// approach replaces in-protocol tool exposure).
//
// meta is the optional `_meta` extension bag. Pass nil for every host except
// claude-code; unknown _meta is host-specific and other bridges may choke on
// it (see priming.go).
func (c *Client) NewSession(ctx context.Context, cwd string, mcpServers []McpServer, meta *SessionMeta) (string, error) {
	if mcpServers == nil {
		// The agent expects an array, not null — match the live agent's
		// strictness rather than relying on its tolerance.
		mcpServers = []McpServer{}
	}
	raw, err := c.Call(ctx, MethodNewSession, NewSessionParams{
		Cwd:        cwd,
		McpServers: mcpServers,
		Meta:       meta,
	})
	if err != nil {
		return "", fmt.Errorf("session/new: %w", err)
	}
	var res NewSessionResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("session/new: decode result: %w", err)
	}
	if res.SessionID == "" {
		return "", fmt.Errorf("session/new: agent returned empty sessionId")
	}
	return res.SessionID, nil
}

// Prompt sends a user message to the agent and blocks until the matching
// session/prompt response arrives. Streaming chunks are delivered through
// the Client's notification handler — the caller is responsible for wiring
// it up before calling Prompt (typically by passing the handler at
// NewClient time).
//
// Returns the agent's stop reason.
func (c *Client) Prompt(ctx context.Context, sessionID, text string) (string, error) {
	raw, err := c.Call(ctx, MethodPrompt, PromptParams{
		SessionID: sessionID,
		Prompt: []ContentBlock{
			{Type: "text", Text: text},
		},
	})
	if err != nil {
		return "", fmt.Errorf("session/prompt: %w", err)
	}
	var res PromptResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("session/prompt: decode result: %w", err)
	}
	return res.StopReason, nil
}

// Cancel issues a session/cancel notification. Best-effort — the agent may
// have already finished by the time it arrives, which is harmless.
func (c *Client) Cancel(sessionID string) error {
	return c.Notify(MethodCancel, CancelParams{SessionID: sessionID})
}

// DecodeAgentMessageChunk extracts the streamed text from a session/update
// notification payload. Returns ("", false) for non-agent_message_chunk
// updates — those carry tool calls, plan updates, etc., which the alpha
// orchestrator does not consume.
func DecodeAgentMessageChunk(params json.RawMessage) (string, bool) {
	var p SessionUpdateParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", false
	}
	var c struct {
		SessionUpdate string         `json:"sessionUpdate"`
		Content       MessageContent `json:"content"`
	}
	if err := json.Unmarshal(p.Update, &c); err != nil {
		return "", false
	}
	if c.SessionUpdate != "agent_message_chunk" {
		return "", false
	}
	if c.Content.Type != "text" {
		return "", false
	}
	return c.Content.Text, true
}
