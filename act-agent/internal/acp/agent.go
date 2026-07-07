package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
)

// ProviderACP is the synthetic provider identifier the ACPAgent reports.
// The TUI status pane reads provider/model strings for display only — it
// doesn't dispatch on this value, so a non-registered identifier is safe.
// The canonical value lives in the models package (models.ProviderACP) so the
// prompt dispatcher can branch on it without importing acp (which would be an
// import cycle). This alias keeps the existing acp.ProviderACP call sites.
const ProviderACP = models.ProviderACP

// ErrACPSubprocessExited indicates the ACP host died before / during a turn.
// The orchestrator's humanReadableAgentError maps this to a friendly
// "try /backend restart <role>" message.
var ErrACPSubprocessExited = errors.New("acp subprocess exited")

// ACPAgent implements agent.Service over a long-lived ACP subprocess. One
// instance per Tier 1 role; the role drives which shim binary the subprocess
// sees on PATH (Option B act-tier1-<role> wrapper).
type ACPAgent struct {
	*pubsub.Broker[agent.AgentEvent]

	role     string
	cfg      *config.ACPConfig
	sessions session.Service
	messages message.Service

	// agentInfo is captured from the initialize response; used by Model().
	agentInfo AgentInfo

	// cmd + client own the subprocess lifecycle.
	cmd    *exec.Cmd
	client *Client

	// acpSessions maps an ACT (database) sessionID → the agent-allocated ACP
	// session ID. We open one ACP session per ACT session and reuse it across
	// turns.
	mu          sync.Mutex
	acpSessions map[string]string

	// activeChunks routes streamed agent_message_chunk deltas for an in-flight
	// turn back to that turn's accumulator. Keyed by ACT sessionID. Set at
	// Run() entry, cleared on exit.
	activeChunks   map[string]chan string

	// activeRequests mirrors agent.activeRequests — per-session cancel funcs.
	activeRequests sync.Map

	// priming returns the role-specific text to send as the first user
	// message in any new ACP session. Stashed at construction; see
	// primingFor() for the lazy invocation site.
	priming SystemPromptInjector
}

// SystemPromptInjector lets the caller decide what text to send as the first
// message to a fresh ACP session, before the orchestrator's per-turn content
// arrives. Typical use: prepend the role's prompt (planner.go etc.) and the
// shim-binary instructions. Returns "" to skip priming.
type SystemPromptInjector func(role string) string

// NewACPAgent spawns the ACP subprocess and runs the initialize handshake.
// Returns a ready ACPAgent, or an error if the subprocess failed to start
// or the agent rejected initialization. The subprocess is killed by Close.
//
// host is the user-facing backend identifier ("claude-code", future: "codex",
// "gemini", "opencode") — same vocabulary the user types in ~/.act.json
// under agents.<role>.backend. cfg is optional power-user overrides for the
// subprocess spawn (Command/Args/Env/Cwd); pass nil for the default spawn
// derived from host.
//
// priming, if non-nil, is invoked once per ACT sessionID to produce the
// role's static system prompt — sent as the first user message in each new
// ACP session. This is how Planner/Observer/Assurance/QA role behaviour
// gets into a host that doesn't speak ACT-specific config.
func NewACPAgent(
	role string,
	host string,
	cfg *config.ACPConfig,
	sessions session.Service,
	messages message.Service,
	priming SystemPromptInjector,
) (*ACPAgent, error) {
	cmd, err := buildCommand(role, host, cfg)
	if err != nil {
		return nil, fmt.Errorf("acp: build command for role %q: %w", role, err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("acp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("acp: stdout pipe: %w", err)
	}

	// Route stderr to a per-role log file so subprocess noise doesn't pollute
	// the TUI chat. Mirrors the runner/log convention used by Tier 2.
	logPath, err := openACPLog(role)
	if err != nil {
		// Non-fatal — fall back to discarding stderr. Subprocess can still run.
		logging.Warn("acp_log_open_failed", "role", role, "error", err)
		cmd.Stderr = io.Discard
	} else {
		cmd.Stderr = logPath
	}

	// Process group: kill the whole subtree on Close. The npx → node → agent
	// chain spawns descendants; without Setpgid the npx wrapper would survive.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("acp: start subprocess: %w", err)
	}

	a := &ACPAgent{
		Broker:       pubsub.NewBroker[agent.AgentEvent](),
		role:         role,
		cfg:          cfg,
		sessions:     sessions,
		messages:     messages,
		cmd:          cmd,
		acpSessions:  make(map[string]string),
		activeChunks: make(map[string]chan string),
	}

	transport := NewNewlineTransport(stdout, stdin, stdin)
	a.client = NewClient(transport, a.onNotification)
	// Tool-permission broker: hard per-role enforcement at the protocol
	// boundary (see permission_policy.go). Installed before initialize so no
	// request can slip through unanswered.
	a.client.SetRequestHandler(func(method string, params json.RawMessage) (any, *RPCError) {
		if method != MethodReqPermission {
			return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("method %q not supported by this client", method)}
		}
		return answerPermissionRequest(role, params)
	})

	// Initialize handshake — fail loudly if the agent doesn't speak ACP. Use
	// a fresh context here; the caller hasn't started a turn yet so there's
	// no agent-turn context to thread through.
	initRes, err := a.client.Initialize(context.Background())
	if err != nil {
		_ = a.client.Close()
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("acp: initialize role %q: %w", role, err)
	}
	a.agentInfo = initRes.AgentInfo
	logging.Info("acp_initialized",
		"role", role,
		"agent_name", initRes.AgentInfo.Name,
		"agent_version", initRes.AgentInfo.Version,
		"protocol_version", initRes.ProtocolVersion,
	)

	// Stash priming so we apply it lazily per new ACP session.
	a.priming = priming
	return a, nil
}

// priming is captured at construction (see NewACPAgent). Stored as a field
// rather than passed into Run() to keep the agent.Service contract identical
// between in-process and ACP backends.
func (a *ACPAgent) primingFor() string {
	if a.priming == nil {
		return ""
	}
	return a.priming(a.role)
}

// Run satisfies agent.Service. Mirrors internal/llm/agent.agent.Run() shape:
// validates session-busy, sets up cancel-on-context, spawns a goroutine that
// drives the ACP turn, returns a channel that emits exactly one terminal
// AgentEvent.
func (a *ACPAgent) Run(ctx context.Context, sessionID, content string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error) {
	if a.IsSessionBusy(sessionID) {
		return nil, agent.ErrSessionBusy
	}
	_ = attachments // ACP supports image/resource content blocks; not wired in alpha.

	events := make(chan agent.AgentEvent, 1)
	turnCtx, cancel := context.WithCancel(ctx)
	a.activeRequests.Store(sessionID, cancel)

	go func() {
		defer a.activeRequests.Delete(sessionID)
		defer cancel()
		defer close(events)
		defer logging.RecoverPanic("acp.Run", func() {
			events <- a.errEvent(fmt.Errorf("panic while running acp agent"))
		})

		result := a.runTurn(turnCtx, sessionID, content)
		if result.Error != nil &&
			!errors.Is(result.Error, agent.ErrRequestCancelled) &&
			!errors.Is(result.Error, context.Canceled) {
			logging.ErrorPersist(fmt.Sprintf("acp turn failed: %v", result.Error))
		}
		a.Publish(pubsub.CreatedEvent, result)
		events <- result
	}()
	return events, nil
}

// runTurn does the actual ACP exchange: (lazy) session/new, write user
// message to messages store, write empty assistant message, session/prompt,
// drain chunks into the assistant message, finalise on stop_reason.
func (a *ACPAgent) runTurn(ctx context.Context, sessionID, content string) agent.AgentEvent {
	// Already per-agent-scoped for INPUT: only `content` is sent to the external
	// agent via session/prompt (no shared-transcript replay), and each ACPAgent
	// keeps its own ACP session per ACT sessionID. So the in-process historyMode
	// has no analogue here — don't add filtering. We still stamp ThreadID on the
	// DISPLAY messages below so the data is uniform with the in-process path (and
	// so an in-process Planner that reads its thread sees ACP-produced turns too).
	acpSessionID, err := a.ensureACPSession(ctx, sessionID)
	if err != nil {
		return a.errEvent(fmt.Errorf("ensure acp session: %w", err))
	}

	userMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:     message.User,
		Parts:    []message.ContentPart{message.TextContent{Text: content}},
		ThreadID: a.role,
	})
	if err != nil {
		return a.errEvent(fmt.Errorf("create user message: %w", err))
	}
	_ = userMsg

	assistantMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:     message.Assistant,
		Parts:    []message.ContentPart{},
		Model:    models.ModelID(a.agentInfo.Name),
		ThreadID: a.role,
	})
	if err != nil {
		return a.errEvent(fmt.Errorf("create assistant message: %w", err))
	}

	// Subscribe this turn's accumulator to the chunk stream. Capacity is
	// generous — Claude Code can emit several chunks per second, and we want
	// the notification handler to never block the read loop.
	chunkCh := make(chan string, 256)
	a.mu.Lock()
	a.activeChunks[sessionID] = chunkCh
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.activeChunks, sessionID)
		a.mu.Unlock()
	}()

	// Drainer goroutine: append chunks to the assistant message until the
	// prompt response lands. We persist incrementally so the TUI sees output
	// arrive in real time (same UX as in-process agents).
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for delta := range chunkCh {
			assistantMsg.AppendContent(delta)
			if err := a.messages.Update(context.Background(), assistantMsg); err != nil {
				logging.Warn("acp_chunk_persist_failed", "session", sessionID, "error", err)
			}
		}
	}()

	stopReason, promptErr := a.client.Prompt(ctx, acpSessionID, content)

	// Close the chunk channel to release the drainer. The notification
	// handler will keep delivering until we wipe activeChunks above; that
	// already happened via the deferred lock cleanup before we get here.
	// Belt-and-braces: drain a final time so any late chunks don't get lost.
	a.mu.Lock()
	if ch, ok := a.activeChunks[sessionID]; ok && ch == chunkCh {
		delete(a.activeChunks, sessionID)
	}
	a.mu.Unlock()
	close(chunkCh)
	<-drainDone

	finishReason := message.FinishReasonEndTurn
	switch {
	case promptErr != nil && (errors.Is(promptErr, context.Canceled) || errors.Is(promptErr, ctx.Err())):
		finishReason = message.FinishReasonCanceled
	case promptErr != nil:
		finishReason = message.FinishReasonError
	case stopReason == StopReasonMaxTokens:
		finishReason = message.FinishReasonMaxTokens
	case stopReason == StopReasonCancelled:
		finishReason = message.FinishReasonCanceled
	}
	assistantMsg.AddFinish(finishReason)
	if updErr := a.messages.Update(context.Background(), assistantMsg); updErr != nil {
		logging.Warn("acp_finalise_failed", "session", sessionID, "error", updErr)
	}

	if promptErr != nil {
		if errors.Is(promptErr, ErrClientClosed) {
			return a.errEvent(ErrACPSubprocessExited)
		}
		if errors.Is(promptErr, context.Canceled) {
			return a.errEvent(agent.ErrRequestCancelled)
		}
		return agent.AgentEvent{
			Type:    agent.AgentEventTypeError,
			Message: assistantMsg,
			Error:   promptErr,
		}
	}
	return agent.AgentEvent{
		Type:    agent.AgentEventTypeResponse,
		Message: assistantMsg,
	}
}

// ensureACPSession opens an ACP session if we haven't already for this ACT
// sessionID. cwd is the ACT project root (cmd.Dir if set, else process cwd).
func (a *ACPAgent) ensureACPSession(ctx context.Context, sessionID string) (string, error) {
	a.mu.Lock()
	if id, ok := a.acpSessions[sessionID]; ok {
		a.mu.Unlock()
		return id, nil
	}
	a.mu.Unlock()

	cwd := a.cmd.Dir
	if cwd == "" {
		if d, err := os.Getwd(); err == nil {
			cwd = d
		}
	}
	id, err := a.client.NewSession(ctx, cwd, nil)
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	a.acpSessions[sessionID] = id
	a.mu.Unlock()

	// Lazily inject the priming prompt — the role's static system prompt plus
	// the shim-binary instructions. We send it as a one-shot user message
	// (ACP has no system-message channel). Audit Fix 8: we used to discard
	// the result entirely, hiding host hallucinations / refusals. Now we
	// log StopReason so a misbehaving host is visible in the runner log
	// instead of silent. End_turn is the happy path; anything else gets
	// a WARN so it surfaces in log review.
	if prime := a.primingFor(); prime != "" {
		stopReason, err := a.client.Prompt(ctx, id, prime)
		switch {
		case err != nil:
			logging.Warn("acp_priming_failed", "role", a.role, "error", err)
		case stopReason == "":
			logging.Warn("acp_priming_no_stop_reason", "role", a.role,
				"reason", "Prompt returned empty stop_reason — host may be misbehaving")
		case stopReason != StopReasonEndTurn:
			logging.Warn("acp_priming_unexpected_stop_reason",
				"role", a.role, "stop_reason", stopReason,
				"hint", "non-end_turn stop on priming may indicate host refusal or hallucination")
		default:
			logging.Info("acp_priming_completed", "role", a.role, "stop_reason", stopReason)
		}
	}
	return id, nil
}

// onNotification is the Client's notification handler. Routes
// agent_message_chunk deltas to the matching turn's accumulator.
func (a *ACPAgent) onNotification(method string, params json.RawMessage) {
	if method != NotifSessionUpdate {
		return
	}
	delta, ok := DecodeAgentMessageChunk(params)
	if !ok || delta == "" {
		return
	}
	// Find which ACT session this ACP sessionID belongs to. We could index
	// the inverse map; with at most ~4 Tier 1 sessions per role at runtime,
	// a linear scan is cheaper than the synchronisation cost of a second map.
	var acpSID string
	if err := json.Unmarshal(params, &struct {
		SessionID *string `json:"sessionId"`
	}{SessionID: &acpSID}); err != nil {
		return
	}
	a.mu.Lock()
	var actSID string
	for k, v := range a.acpSessions {
		if v == acpSID {
			actSID = k
			break
		}
	}
	var ch chan string
	if actSID != "" {
		ch = a.activeChunks[actSID]
	}
	a.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- delta:
	default:
		// Channel full — shouldn't happen with 256-cap and a fast drainer, but
		// drop rather than block the read loop. The user message stream is
		// best-effort; the final assistant.Content is reconstructed from what
		// we did persist.
		logging.Warn("acp_chunk_dropped", "role", a.role, "session", actSID)
	}
}

// IsBusy reports whether any turn is in flight.
func (a *ACPAgent) IsBusy() bool {
	busy := false
	a.activeRequests.Range(func(k, v any) bool {
		busy = true
		return false
	})
	return busy
}

// IsSessionBusy reports whether the given session has an in-flight turn.
func (a *ACPAgent) IsSessionBusy(sessionID string) bool {
	_, busy := a.activeRequests.Load(sessionID)
	return busy
}

// Cancel signals the agent to stop the current turn for this session and
// cancels our local context. Best-effort.
func (a *ACPAgent) Cancel(sessionID string) {
	a.mu.Lock()
	acpSID := a.acpSessions[sessionID]
	a.mu.Unlock()
	if acpSID != "" {
		if err := a.client.Cancel(acpSID); err != nil {
			logging.Warn("acp_cancel_failed", "session", sessionID, "error", err)
		}
	}
	if cf, ok := a.activeRequests.LoadAndDelete(sessionID); ok {
		if fn, ok := cf.(context.CancelFunc); ok {
			fn()
		}
	}
}

// Model returns a synthetic Model identifying the ACP host. The TUI status
// pane displays Provider + ID — both are user-facing strings.
func (a *ACPAgent) Model() models.Model {
	name := a.agentInfo.Name
	if name == "" {
		name = "claude-agent-acp"
	}
	if a.agentInfo.Version != "" {
		name = name + "@" + a.agentInfo.Version
	}
	return models.Model{
		ID:       models.ModelID(name),
		Provider: ProviderACP,
	}
}

// Update is a no-op for ACP-backed agents. Model selection is the ACP host's
// concern, configured externally.
func (a *ACPAgent) Update(_ config.AgentName, _ models.ModelID) (models.Model, error) {
	return a.Model(), fmt.Errorf("acp: model selection is configured by the ACP host (edit ~/.act.json agents.%s.acp.*)", a.role)
}

// RebindSystemPrompt refreshes the role's system prompt by discarding the
// cached ACP sessions. The next runTurn for any ACT sessionID will call
// ensureACPSession, which opens a fresh ACP session and fires the priming
// injector — which calls prompt.GetAgentPrompt at invocation time and picks
// up the freshly-invalidated AGENTS.md / ACT.md / ACT.local.md content.
//
// Refuses to run while a turn is in flight so we don't yank the session out
// from under an active prompt. The caller (orchestrator) is expected to
// invalidate prompt.GetAgentPrompt's cache BEFORE calling this so the next
// session/new request carries the new content.
func (a *ACPAgent) RebindSystemPrompt() error {
	if a.IsBusy() {
		return fmt.Errorf("acp: cannot rebind system prompt while processing requests")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// Best-effort cancel of any lingering ACP sessions before we forget them.
	// The ACP host (claude-agent-acp) garbage-collects orphaned sessions when
	// the subprocess exits, so this is belt + suspenders against stragglers.
	for actSID, acpSID := range a.acpSessions {
		if err := a.client.Cancel(acpSID); err != nil {
			logging.Warn("acp_rebind_cancel_failed",
				"role", a.role, "act_session", actSID, "acp_session", acpSID, "error", err)
		}
	}
	a.acpSessions = make(map[string]string)
	return nil
}

// Summarize is intentionally unsupported for the alpha. Tier 1 summarisation
// (the agent.Summarize feature) is not on the alpha-blocker list.
func (a *ACPAgent) Summarize(_ context.Context, _ string) error {
	return fmt.Errorf("acp: summarize not supported in the alpha (Tier 1 summarisation deferred)")
}

// Close kills the subprocess and releases the Client. Idempotent.
func (a *ACPAgent) Close() error {
	closeErr := a.client.Close()
	if a.cmd != nil && a.cmd.Process != nil {
		// Best-effort: SIGTERM the whole process group so npx/node descendants
		// die with us. Setpgid was set at spawn time.
		_ = syscall.Kill(-a.cmd.Process.Pid, syscall.SIGTERM)
		_ = a.cmd.Wait()
	}
	return closeErr
}

func (a *ACPAgent) errEvent(err error) agent.AgentEvent {
	return agent.AgentEvent{
		Type:  agent.AgentEventTypeError,
		Error: err,
	}
}

// ─── Subprocess command + log plumbing ──────────────────────────────────────

func buildCommand(role, host string, cfg *config.ACPConfig) (*exec.Cmd, error) {
	var command string
	var args []string
	if cfg != nil {
		command = cfg.Command
		args = cfg.Args
	}
	if command == "" {
		// Default by host. Only claude-code is implemented for alpha; other
		// hosts return an explicit unimplemented error so misconfiguration
		// fails loudly at startup, not at the first prompt.
		switch host {
		case "claude-code", "":
			command, args = claudeCodeDefaults()
		case "antigravity", "agy":
			var env map[string]string
			if cfg != nil {
				env = cfg.Env
			}
			command, args = antigravityCLIDefaults(env)
		case "codex", "opencode":
			return nil, fmt.Errorf("acp: backend %q is not implemented yet", host)
		default:
			return nil, fmt.Errorf("acp: unknown backend %q", host)
		}
	}
	cmd := exec.Command(command, args...)
	if cfg != nil {
		if cfg.Cwd != "" {
			cmd.Dir = cfg.Cwd
		}
		if len(cfg.Env) > 0 {
			env := os.Environ()
			for k, v := range cfg.Env {
				env = append(env, k+"="+v)
			}
			cmd.Env = env
		}
	}
	// The PATH-prepend for the role's shim binary happens in app.go::New()
	// (it needs to know the install dir of act-tier1-shim, which is tied to
	// the act-agent binary's own dir). Wiring lives there, not here.
	_ = role
	return cmd, nil
}

// openACPLog opens ~/.act/runners/tier1-<role>-acp.log for append. Returns
// an *os.File so cmd.Stderr can be set to it directly.
func openACPLog(role string) (*os.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".act", "runners")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(
		filepath.Join(dir, "tier1-"+role+"-acp.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644,
	)
}
