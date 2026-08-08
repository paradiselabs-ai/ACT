// Package acp implements an Agent Client Protocol (ACP) client that drives
// external agent CLI subprocesses (Claude Code via @agentclientprotocol/claude-agent-acp,
// Antigravity CLI (agy) via the bundled agy-acp.mjs shim, and — future — Codex, OpenCode).
//
// ACP is a JSON-RPC 2.0 protocol over stdio with newline-delimited framing.
// Framing was verified empirically against claude-agent-acp@0.37.0; the
// LSP-style Content-Length variant is rejected by the live agent.
package acp

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion is the ACP protocol version this client negotiates.
// The live claude-agent-acp@0.37.0 server reports protocolVersion 1
// in its initialize response.
const ProtocolVersion = 1

// ─── JSON-RPC 2.0 envelopes ────────────────────────────────────────────────

// Request is a JSON-RPC request. ID is set for method calls and omitted for
// notifications. ACP uses integer IDs by convention.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response. Exactly one of Result/Error is set.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("acp rpc error %d: %s", e.Code, e.Message)
}

// Frame is the deserialised top-level shape we read off the wire. Notifications
// and responses share the same line; we distinguish by presence of `method`
// (notification or request) vs `result`/`error` (response). For ACP today the
// client only receives notifications and responses — the agent does not send
// requests back (permissions go via session/request_permission which the
// agent issues as a request, but only when the client advertises that
// capability — out of scope for the alpha shim-binary approach).
type Frame struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// IsResponse reports whether this frame carries a response (result or error
// with an ID). Notifications have no ID; requests have an ID and a method.
func (f *Frame) IsResponse() bool {
	return f.ID != nil && f.Method == ""
}

// IsNotification reports whether this frame is a notification (method present,
// ID absent).
func (f *Frame) IsNotification() bool {
	return f.ID == nil && f.Method != ""
}

// ─── ACP method params/results ─────────────────────────────────────────────

// InitializeParams is the payload of the initialize request. The client
// advertises its protocol version and (optional) capabilities. For the alpha
// scope we send minimal capabilities — the shim-binary approach means we do
// not need the client to expose file-system or terminal tools to the agent.
type InitializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
}

// ClientCapabilities is the set of features this client supports. Empty for
// the alpha — claude-agent-acp@0.37 accepts an empty object and proceeds.
type ClientCapabilities struct{}

// InitializeResult is what the agent returns. We only consume what we need
// today; additional fields are tolerated via json.RawMessage on the wrapping
// Frame.
type InitializeResult struct {
	ProtocolVersion   int               `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities"`
	AgentInfo         AgentInfo         `json:"agentInfo"`
}

// AgentCapabilities — fields verified against the live agent's response.
type AgentCapabilities struct {
	PromptCapabilities  json.RawMessage `json:"promptCapabilities,omitempty"`
	McpCapabilities     json.RawMessage `json:"mcpCapabilities,omitempty"`
	LoadSession         bool            `json:"loadSession,omitempty"`
	SessionCapabilities json.RawMessage `json:"sessionCapabilities,omitempty"`
}

// AgentInfo identifies the agent — used for the synthetic Model() string the
// acpAgent reports to the TUI status pane.
type AgentInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// NewSessionParams creates a new session. Cwd is the working directory the
// agent's tools should resolve relative paths against. McpServers is the list
// of MCP servers the host wants the agent to connect to — empty for the
// alpha (shim-binary approach replaces in-protocol tool exposure).
// Meta is the ACP `_meta` extension bag. Only the Claude Code bridge reads
// it (see priming.go); other hosts must never receive it.
type NewSessionParams struct {
	Cwd        string       `json:"cwd"`
	McpServers []McpServer  `json:"mcpServers"`
	Meta       *SessionMeta `json:"_meta,omitempty"`
}

// SessionMeta carries host-specific extensions on session/new.
type SessionMeta struct {
	SystemPrompt *SystemPromptMeta `json:"systemPrompt,omitempty"`
	ClaudeCode   *ClaudeCodeMeta   `json:"claudeCode,omitempty"`
}

// ClaudeCodeMeta is `_meta.claudeCode`: bridge-specific SDK options. The
// bridge spreads Options AFTER its own defaults, so fields here override.
type ClaudeCodeMeta struct {
	Options ClaudeCodeOptions `json:"options"`
}

// ClaudeCodeOptions overrides for the Claude Agent SDK session the bridge
// creates. SettingSources deliberately has NO omitempty: the empty slice must
// serialize as [] to override the bridge's ["user","project","local"] default.
// Without the override, every ACT Tier-1 claude session loads the OPERATOR's
// personal Claude Code config — global CLAUDE.md, plugins/hooks (persona modes
// like a "lazy dev" style), and auto-memory. Live bug 2026-08-08 (lido run):
// the Planner opened with "Plan (lazy default, correct me)" — the operator's
// persona plugin speaking — skipped mandatory intake questions, and offered to
// write rules into the operator's personal memory. ACT roles must be
// clean-room: role prompt only.
type ClaudeCodeOptions struct {
	SettingSources []string `json:"settingSources"`
}

// SystemPromptMeta is the object form of `_meta.systemPrompt`. The Claude
// Code bridge spreads it over its claude_code preset with type/preset locked,
// so Append is added to Claude Code's own system prompt rather than replacing
// it.
type SystemPromptMeta struct {
	Append string `json:"append,omitempty"`
}

// McpServer describes one MCP server the agent should connect to. Empty list
// in alpha; kept here so future tool-exposure work can wire this up without
// types.go churn.
type McpServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// NewSessionResult carries the agent-allocated session ID. Subsequent
// session/prompt and session/cancel calls reference this ID.
type NewSessionResult struct {
	SessionID string `json:"sessionId"`
}

// PromptParams sends user input to the agent. Multiple ContentBlocks are
// concatenated by the agent into a single message.
type PromptParams struct {
	SessionID string         `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// ContentBlock is one piece of a prompt — text, image, or embedded resource.
// The alpha only emits text; the other variants are spec-defined and kept here
// for forward compatibility.
type ContentBlock struct {
	Type     string          `json:"type"` // "text" | "image" | "resource"
	Text     string          `json:"text,omitempty"`
	Resource json.RawMessage `json:"resource,omitempty"`
}

// PromptResult is the agent's terminal response to a session/prompt call.
// StopReason indicates why the agent stopped (end_turn, max_tokens, refusal,
// cancelled, etc.). Streaming output arrives separately as session/update
// notifications between the request and this response.
type PromptResult struct {
	StopReason string `json:"stopReason"`
}

// CancelParams is the payload of the session/cancel notification.
type CancelParams struct {
	SessionID string `json:"sessionId"`
}

// SessionUpdateParams is the payload of a session/update notification. The
// agent streams these between the prompt request and the prompt response.
// Update.SessionUpdate discriminates the variant.
type SessionUpdateParams struct {
	SessionID string         `json:"sessionId"`
	Update    json.RawMessage `json:"update"`
}

// AgentMessageChunk is the most common Update variant: a streamed text delta
// from the assistant. The acpAgent concatenates these into the assistant
// message that gets handed back to the orchestrator.
type AgentMessageChunk struct {
	SessionUpdate string         `json:"sessionUpdate"` // "agent_message_chunk"
	Content       MessageContent `json:"content"`
}

// MessageContent — only the text variant is consumed in the alpha. Image and
// resource variants are tolerated and dropped (no orchestrator consumer for
// them yet).
type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Stop reasons exposed by the agent in PromptResult.StopReason.
const (
	StopReasonEndTurn         = "end_turn"
	StopReasonMaxTokens       = "max_tokens"
	StopReasonMaxTurnRequests = "max_turn_requests"
	StopReasonRefusal         = "refusal"
	StopReasonCancelled       = "cancelled"
)

// ACP method names — string constants to keep the call sites honest.
const (
	MethodInitialize    = "initialize"
	MethodNewSession    = "session/new"
	MethodPrompt        = "session/prompt"
	MethodCancel        = "session/cancel"
	NotifSessionUpdate  = "session/update"
	MethodReqPermission = "session/request_permission"
)

// RequestPermissionParams — the agent asks the client to authorize one tool
// call. Options carry the agent-defined choice IDs the client must pick from.
type RequestPermissionParams struct {
	SessionID string             `json:"sessionId"`
	ToolCall  PermissionToolCall `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

// PermissionToolCall is the subset of the ACP tool-call shape the policy
// consults. Kind is the spec's tool classification (read / edit / delete /
// move / search / execute / think / fetch / other); Title and RawInput give
// the human-readable command context (used to recognize the role shim).
type PermissionToolCall struct {
	ToolCallID string          `json:"toolCallId,omitempty"`
	Title      string          `json:"title,omitempty"`
	Kind       string          `json:"kind,omitempty"`
	RawInput   json.RawMessage `json:"rawInput,omitempty"`
}

// PermissionOption is one selectable outcome the agent offers.
type PermissionOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"` // allow_once | allow_always | reject_once | reject_always
}

// RequestPermissionResult answers session/request_permission.
type RequestPermissionResult struct {
	Outcome PermissionOutcome `json:"outcome"`
}

// PermissionOutcome — "selected" with the chosen OptionID, or "cancelled".
type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}
