package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/act"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/nomik"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/runner"
)

// Orchestrator coordinates multiple Tier 1 agents in the TUI.
//
// Responsibilities:
//   - Routes human input to the Planner agent
//   - Tracks which agent (role) produced each message (for color-coded rendering)
//   - Listens to message PubSub for ownership tagging and CREATE_TASK detection
//   - Runs the Observer monitoring loop (anomaly detection)
//   - Polls the server for tasks awaiting validation/synthesis and routes to Assurance/QA
//   - Spawns the Runner subprocess on first task creation
//
// All four Tier 1 agents (Planner, Observer, Assurance, QA) live as
// agent.Service instances in app.Agents, sharing one chat session.
type Orchestrator struct {
	app *App

	mu             sync.RWMutex
	messageOwners  map[string]string // messageID → role name
	currentSpeaker string            // role currently running (for ownership tagging)
	sessionID      string            // shared session for the NesTTY conversation
	seenTasks      map[string]bool   // task IDs we've already routed (validation/qa)

	// Background loop control
	loopsStarted bool
	loopCancel   context.CancelFunc
	loopWG       sync.WaitGroup

	// Runner subprocess management — multi-runner swarm
	runnerSpawner *runner.Spawner

	// Nomik state
	nomikEnabled    bool   // user toggle (per-project, defaults true if available)
	nomikAvailable  bool   // detected at Start()
	projectDir      string // working directory for Nomik commands
	rescanInflight  bool   // debounce flag
	rescanInflightMu sync.Mutex

	// Intake state — set in Start() by detectProjectState. While intakeMode is
	// true, the orchestrator prepends prompt.IntakePromptPrefix() to every
	// Planner turn and the message ownership loop scans Planner output for
	// PROJECT_BRIEF directives instead of CREATE_TASK. Cleared atomically when
	// the brief is successfully POSTed to the server.
	intakeMode  bool
	projectName string // ACT_PROJECT — derived from --project flag or cwd basename

	// Coordination event polling state. Tracks the most recent event timestamp
	// the loop has surfaced as a chat system message, so we never re-emit.
	lastCoordEventAt string

	// Observer cycle counter — used to emit periodic "all clear" health
	// messages every observerHealthEvery cycles when no anomalies are found.
	observerCycles int

	// lastObserverPromptHash is the sha256 of the most recent Observer prompt
	// we actually sent to the LLM. If a subsequent cycle would produce the same
	// prompt (same anomalies, same snapshot signature), we skip the LLM call —
	// the Observer would just say the same thing twice. Cleared on human input
	// so the next anomaly cycle after a human turn always fires fresh.
	lastObserverPromptHash string

	// consecutiveAutoTurns counts auto-routed Planner turns chained without a
	// human input in between. Capped at autoTurnCap to prevent agent loops.
	// Reset to 0 by HandleHumanInput.
	consecutiveAutoTurns int
}

// NewOrchestrator creates an orchestrator wired to the given app.
func NewOrchestrator(app *App) *Orchestrator {
	return &Orchestrator{
		app:           app,
		messageOwners: make(map[string]string),
		seenTasks:     make(map[string]bool),
		runnerSpawner: runner.NewSpawner(),
	}
}

// Start begins all background goroutines: message ownership listener,
// Observer monitoring loop, validation poller, QA poller. Also spawns the
// Tier 2 swarm (one Runner per role) and runs the Nomik project init scan.
//
// Idempotent — safe to call multiple times. The sessionID identifies the
// shared NesTTY chat session that all agents write to.
func (o *Orchestrator) Start(parentCtx context.Context, sessionID string) {
	o.mu.Lock()
	if o.loopsStarted {
		o.sessionID = sessionID
		o.mu.Unlock()
		return
	}
	o.loopsStarted = true
	o.sessionID = sessionID

	cwd, _ := os.Getwd()
	o.projectDir = cwd
	o.projectName = os.Getenv("ACT_PROJECT")

	ctx, cancel := context.WithCancel(parentCtx)
	o.loopCancel = cancel
	o.mu.Unlock()

	// Detect intake state: if the ACT server has no project record matching
	// our name, the Planner enters INTAKE mode for the first conversation.
	o.detectProjectState()

	// Tier 2 swarm is spawned LAZILY — only when the Planner emits its first
	// CREATE_TASK directive. Spawning 5 Node subprocesses on every `act` launch
	// (even when the user is just chatting or running intake) is wasteful and
	// fills the chat with runner registration logs. See
	// handlePlannerTaskDirectives for the lazy spawn trigger.

	// Detect Nomik availability for this session
	o.mu.Lock()
	o.nomikAvailable = nomik.IsAvailable()
	o.nomikEnabled = o.nomikAvailable // default ON if available
	o.mu.Unlock()

	if o.nomikAvailable && o.nomikEnabled {
		go o.nomikInitProject(ctx)
	} else if !o.nomikAvailable {
		logging.Info("Nomik unavailable — codebase graph features disabled")
	}

	o.loopWG.Add(5)
	go o.messageOwnershipLoop(ctx)
	go o.observerLoop(ctx)
	go o.validationPollLoop(ctx)
	go o.qaPollLoop(ctx)
	go o.coordinationEventLoop(ctx)

	// Tier 1 "I'm alive" pings — without these the user has no way to know
	// Observer/Assurance/QA exist (they're event-driven and stay silent on
	// healthy runs). Pinged once at startup, after a short delay so they
	// land after the welcome message.
	go func() {
		time.Sleep(2 * time.Second)
		o.mu.RLock()
		sid := o.sessionID
		o.mu.RUnlock()
		o.emitSystemMessage(context.Background(), sid, "👁  Observer online — monitoring every 120s")
		o.emitSystemMessage(context.Background(), sid, "✅  Assurance online — waiting for tasks to validate")
		o.emitSystemMessage(context.Background(), sid, "🧩  QA/Synthesizer online — waiting for validated outputs")
	}()

	logging.Info("Orchestrator started", "session_id", sessionID, "swarm_specs", len(o.app.SwarmSpecs), "nomik", o.nomikAvailable)
}

// nomikInitProject runs `nomik init` + `nomik scan` once at startup, then
// publishes the onboarding summary to the project brief so swarm agents see
// it via `act context`. All errors are logged and swallowed — Nomik is
// optional, never fatal.
func (o *Orchestrator) nomikInitProject(ctx context.Context) {
	o.mu.RLock()
	dir := o.projectDir
	o.mu.RUnlock()

	logging.Info("Initializing Nomik for project", "dir", dir)
	if err := nomik.EnsureProject(ctx, dir); err != nil {
		logging.Warn("Nomik init failed", "error", err)
		return
	}

	summary, err := nomik.Onboard(ctx, dir)
	if err != nil {
		logging.Warn("Nomik onboard failed", "error", err)
		return
	}

	// Truncate to a reasonable size before injecting into the project brief
	if len(summary) > 8000 {
		summary = summary[:8000] + "\n\n...(truncated)"
	}

	// Post to the server as a project-level brief so swarm agents pick it up
	// via `act context`. The server may or may not have this endpoint wired —
	// failures are silent.
	client := act.NewClient("orchestrator", os.Getenv("ACT_PROJECT"))
	if client.IsAvailable() {
		_ = client.SendMessage("[NOMIK] Project codebase graph initialized:\n" + summary)
	}

	logging.Info("Nomik project init complete", "summary_bytes", len(summary))
}

// Stop signals all background loops to exit and waits for them to finish.
func (o *Orchestrator) Stop() {
	o.mu.Lock()
	cancel := o.loopCancel
	o.loopCancel = nil
	o.loopsStarted = false
	o.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	o.loopWG.Wait()

	if o.runnerSpawner != nil {
		o.runnerSpawner.Stop()
	}
}

// HandleHumanInput routes user input to the Planner agent.
// This is the main entry point from the TUI — replaces direct CoderAgent.Run() calls.
// Runs in a goroutine from the caller; blocks until the Planner finishes its turn.
func (o *Orchestrator) HandleHumanInput(ctx context.Context, sessionID string, text string, attachments ...message.Attachment) {
	// Reset the auto-turn counter — every human input gets a fresh budget of
	// chained auto-routed Planner turns from the other Tier 1 agents.
	o.mu.Lock()
	o.consecutiveAutoTurns = 0
	o.lastObserverPromptHash = "" // re-arm Observer no-op gate
	o.mu.Unlock()

	o.runAgentTurn(ctx, sessionID, "planner", text, attachments...)
}

// detectProjectState queries the ACT server for a project matching o.projectName.
// 404 → intake mode on. 200 → intake mode off (resume). Network failure → intake
// mode off (we don't want to gate the user behind an unreachable server).
func (o *Orchestrator) detectProjectState() {
	o.mu.Lock()
	name := o.projectName
	o.mu.Unlock()
	if name == "" {
		return
	}
	client := act.NewClient("orchestrator", name)
	if !client.IsAvailable() {
		logging.Info("ACT server unavailable — skipping intake detection")
		return
	}
	_, found, err := client.GetProject(name)
	if err != nil {
		logging.Warn("Project lookup failed — skipping intake", "project", name, "error", err)
		return
	}
	o.mu.Lock()
	o.intakeMode = !found
	o.mu.Unlock()
	if !found {
		logging.Info("No server-side project record — entering INTAKE mode", "project", name)
	} else {
		logging.Info("Project found on server — RESUME mode", "project", name)
	}
}

// runAgentTurn executes a single turn for the given role.
// It sets the current speaker (for message ownership tagging), runs the agent,
// and waits for completion.
func (o *Orchestrator) runAgentTurn(ctx context.Context, sessionID string, role string, content string, attachments ...message.Attachment) {
	agentSvc := o.getAgent(role)
	if agentSvc == nil {
		logging.Warn("No agent found for role, falling back to CoderAgent", "role", role)
		agentSvc = o.app.CoderAgent
	}

	// Note: intake instructions live in the Planner's static system prompt
	// (see prompt/planner.go). The orchestrator only watches output for
	// PROJECT_BRIEF directives — no per-turn content injection.

	// For non-Planner Tier 1 agents, the content is an orchestrator-generated
	// prompt (Observer monitoring report, Assurance validation request, QA
	// synthesis instructions). The user did not type it and shouldn't see it
	// in the chat. Prepend the InternalPromptMarker so the chat list filters
	// the resulting User-role message out of the rendered view. The LLM still
	// sees the full content because we pass it as the actual message body.
	if role != "planner" {
		content = InternalPromptMarker + content
	}

	o.mu.Lock()
	o.currentSpeaker = role
	o.mu.Unlock()

	// Per-turn timeout: an agent that hasn't produced a response in this long
	// is almost certainly stuck (LLM provider hanging, infinite tool loop,
	// stale streaming connection). Cancel the context so the user isn't
	// trapped forever in "Generating...". The timeout is intentionally
	// generous because Llama-class models on free tiers are slow.
	turnCtx, cancelTurn := context.WithTimeout(ctx, agentTurnTimeout)
	defer cancelTurn()

	done, err := agentSvc.Run(turnCtx, sessionID, content, attachments...)
	if err != nil {
		logging.Error("Agent turn failed to start", "role", role, "error", err)
		o.mu.Lock()
		o.currentSpeaker = ""
		o.mu.Unlock()
		return
	}

	// Wait for completion or timeout.
	var result agent.AgentEvent
	select {
	case result = <-done:
	case <-turnCtx.Done():
		logging.Warn("Agent turn timed out", "role", role, "timeout", agentTurnTimeout)
		// Cancel via the agent service so any in-flight provider stream stops.
		agentSvc.Cancel(sessionID)
		// Surface a system message so the user knows what happened instead of
		// staring at "Generating..." forever.
		o.emitSystemMessage(context.Background(), sessionID, fmt.Sprintf("⏱  %s turn timed out after %s — cancelled", role, agentTurnTimeout))
		// Drain done to avoid leaking the goroutine that fed it.
		go func() { <-done }()
	}

	o.mu.Lock()
	o.currentSpeaker = ""
	o.mu.Unlock()

	if result.Error != nil {
		logging.Warn("Agent turn completed with error", "role", role, "error", result.Error)
	}
}

// agentTurnTimeout is the maximum wall-clock time a single agent turn is
// allowed to run before the orchestrator force-cancels it. Tuned for free-tier
// LLM providers which can be slow but should never legitimately exceed this.
const agentTurnTimeout = 5 * time.Minute

// InternalPromptMarker is the leading sentinel the orchestrator prepends to
// any user-message content it injects on behalf of an agent (Observer
// monitoring report, Assurance validation prompt, QA synthesis prompt, etc).
//
// The chat list checks for this marker and HIDES the message from the human.
// The agent still receives the full content via the LLM provider — the marker
// is just text and the model ignores it. Without this filtering, the user
// sees Observer's monitoring snapshots, Assurance's validation prompts, and
// QA's synthesis instructions polluting the chat as if they typed them.
const InternalPromptMarker = "\x00ACT_INTERNAL\x00"

// coordinationEventInterval is how often the orchestrator polls the server's
// chronological log for new events to surface in the chat as system messages.
const coordinationEventInterval = 3 * time.Second

// coordinationEventLoop polls the ACT server's chronological log every few
// seconds for new coordination events (task created, completed, validation
// passed/failed, agent message, etc) and renders each as a system message in
// the active chat session. This is how the user sees real-time swarm progress
// without having to tail JSONL log files.
//
// Dedupes by timestamp — only events strictly newer than lastCoordEventAt
// are emitted. Survives transient server failures (logged + retried).
func (o *Orchestrator) coordinationEventLoop(ctx context.Context) {
	defer o.loopWG.Done()

	ticker := time.NewTicker(coordinationEventInterval)
	defer ticker.Stop()

	// Seed lastCoordEventAt to "now" so the first poll doesn't dump the entire
	// historical log into the chat on startup. The user only cares about
	// events from this session forward.
	o.mu.Lock()
	if o.lastCoordEventAt == "" {
		o.lastCoordEventAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	o.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.pollCoordinationEvents(ctx)
		}
	}
}

func (o *Orchestrator) pollCoordinationEvents(ctx context.Context) {
	o.mu.RLock()
	sid := o.sessionID
	since := o.lastCoordEventAt
	o.mu.RUnlock()
	if sid == "" {
		return
	}

	client := act.NewClient("orchestrator", o.projectName)
	if !client.IsAvailable() {
		return
	}
	raw, err := client.GetLog(50)
	if err != nil {
		logging.Debug("Coordination event poll failed", "error", err)
		return
	}

	var resp struct {
		Events []LogEntry `json:"events"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return
	}

	// /api/log returns events newest-first; iterate oldest-first so they
	// appear chronologically in the chat.
	newest := since
	for i := len(resp.Events) - 1; i >= 0; i-- {
		ev := resp.Events[i]
		if ev.Timestamp == "" || ev.Timestamp <= since {
			continue
		}
		text := formatCoordEvent(ev)
		if text == "" {
			continue
		}
		o.emitSystemMessage(ctx, sid, text)
		if ev.Timestamp > newest {
			newest = ev.Timestamp
		}
	}

	if newest != since {
		o.mu.Lock()
		o.lastCoordEventAt = newest
		o.mu.Unlock()
	}
}

// formatCoordEvent renders a server log entry as a single user-friendly chat
// line. Returns empty string for event types that aren't worth surfacing
// (registration, system bookkeeping, etc).
func formatCoordEvent(ev LogEntry) string {
	switch ev.Type {
	case "task_created":
		return fmt.Sprintf("📝  task created — %s", truncate(ev.Message, 120))
	case "task_assigned":
		return fmt.Sprintf("👤  %s assigned: %s", ev.Agent, truncate(ev.Message, 120))
	case "task_progress":
		return fmt.Sprintf("⚙   %s — %s", ev.Agent, truncate(ev.Message, 140))
	case "task_completed":
		return fmt.Sprintf("✓   %s completed: %s", ev.Agent, truncate(ev.Message, 140))
	case "task_failed":
		return fmt.Sprintf("✗   %s failed: %s", ev.Agent, truncate(ev.Message, 140))
	case "task_submitted_for_validation":
		return fmt.Sprintf("📤  %s submitted for validation: %s", ev.Agent, truncate(ev.Message, 120))
	case "validation_passed":
		return fmt.Sprintf("✅  validation passed — %s", truncate(ev.Message, 140))
	case "validation_failed":
		return fmt.Sprintf("❌  validation failed — %s", truncate(ev.Message, 140))
	case "agent_message", "message":
		return fmt.Sprintf("→   %s: %s", ev.Agent, truncate(ev.Message, 140))
	case "peer_response":
		// Peer responses can be very chatty (every 10s status ping); only
		// surface ones that look substantive (>40 chars after the prefix).
		if len(ev.Message) > 50 {
			return fmt.Sprintf("→   %s: %s", ev.Agent, truncate(stripStatusPrefix(ev.Message), 140))
		}
		return ""
	case "project_created":
		return fmt.Sprintf("📦  project created — %s", truncate(ev.Message, 120))
	case "brief_stored":
		return "" // internal
	case "agent_registered", "agent_joined":
		return "" // internal
	default:
		return ""
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func stripStatusPrefix(s string) string {
	return strings.TrimPrefix(s, "status: ")
}

// emitSystemMessage creates a Role=System message in the active chat session.
// The TUI's chat list renders these via renderSystemMessage as muted lines
// (no avatar, no header). Used for orchestrator-level coordination events
// and turn-timeout notices.
func (o *Orchestrator) emitSystemMessage(ctx context.Context, sessionID, text string) {
	if sessionID == "" || text == "" {
		return
	}
	if _, err := o.app.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.System,
		Parts: []message.ContentPart{message.TextContent{Text: text}},
	}); err != nil {
		logging.Debug("Failed to emit system message", "error", err)
	}
}

// getAgent returns the agent service for a given role.
func (o *Orchestrator) getAgent(role string) agent.Service {
	if o.app.Agents == nil {
		return nil
	}
	return o.app.Agents[role]
}

// SetOwner records which role produced a given message.
func (o *Orchestrator) SetOwner(messageID string, role string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.messageOwners[messageID] = role
}

// GetOwner returns the role that produced a given message, or "" if unknown.
func (o *Orchestrator) GetOwner(messageID string) string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.messageOwners[messageID]
}

// CurrentSpeaker returns which role is currently running, or "" if idle.
func (o *Orchestrator) CurrentSpeaker() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.currentSpeaker
}

// IsAnyBusy returns true if any agent is busy.
// If sessionID is non-empty, checks that specific session; otherwise checks globally.
func (o *Orchestrator) IsAnyBusy(sessionID string) bool {
	if o.app.Agents == nil {
		if sessionID != "" {
			return o.app.CoderAgent.IsSessionBusy(sessionID)
		}
		return o.app.CoderAgent.IsBusy()
	}
	for _, agentSvc := range o.app.Agents {
		if sessionID != "" {
			if agentSvc.IsSessionBusy(sessionID) {
				return true
			}
		} else {
			if agentSvc.IsBusy() {
				return true
			}
		}
	}
	return false
}

// CancelActive cancels whichever agent is currently running on the given session.
func (o *Orchestrator) CancelActive(sessionID string) {
	if o.app.Agents == nil {
		o.app.CoderAgent.Cancel(sessionID)
		return
	}
	for _, agentSvc := range o.app.Agents {
		if agentSvc.IsSessionBusy(sessionID) {
			agentSvc.Cancel(sessionID)
			return
		}
	}
	o.app.CoderAgent.Cancel(sessionID)
}

// TagMessagesFromCurrentSpeaker tags a single message with the active speaker.
func (o *Orchestrator) TagMessagesFromCurrentSpeaker(messageID string) {
	o.mu.RLock()
	speaker := o.currentSpeaker
	o.mu.RUnlock()
	if speaker != "" {
		o.SetOwner(messageID, speaker)
	}
}

// ─── Background loop: message ownership + CREATE_TASK detection ────────────────

// messageOwnershipLoop subscribes to message PubSub events and tags every newly
// created assistant message with the role of the current speaker. When a Planner
// message is finalized (status updated, has content), it scans the content for
// CREATE_TASK directives and POSTs them to the ACT server.
func (o *Orchestrator) messageOwnershipLoop(ctx context.Context) {
	defer o.loopWG.Done()
	ch := o.app.Messages.Subscribe(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			msg := event.Payload

			// Tag every newly-created message with the active speaker
			if event.Type == pubsub.CreatedEvent && msg.Role == message.Assistant {
				o.TagMessagesFromCurrentSpeaker(msg.ID)
			}

			// On message updates, check if a Planner message just finalized
			// (has finish data, has content) — scan for CREATE_TASK directives
			if event.Type == pubsub.UpdatedEvent && msg.Role == message.Assistant {
				if !msg.IsFinished() {
					continue
				}
				role := o.GetOwner(msg.ID)
				content := msg.Content().String()
				if content == "" {
					continue
				}

				switch role {
				case "planner":
					// During intake mode, look for PROJECT_BRIEF first. CREATE_TASK
					// directives in the same response are ignored until the brief
					// is accepted (the prompt forbids them, but we defend anyway).
					o.mu.RLock()
					intake := o.intakeMode
					o.mu.RUnlock()
					if intake {
						go o.handleProjectBrief(ctx, content)
						continue
					}
					go o.handlePlannerTaskDirectives(ctx, content)

				case "observer", "assurance", "qa", "qa_synthesizer":
					// Auto-route Tier 1 reports into a Planner turn so the
					// Planner can react (typically by emitting CREATE_TASK
					// directives, occasionally by speaking to the human). The
					// chain stops at Planner — Planner replies are NOT
					// auto-routed anywhere, breaking the loop.
					go o.autoRoutePlanner(ctx, role, content)
				}
			}
		}
	}
}

// autoRoutePlanner forwards a non-Planner Tier 1 message into a Planner turn.
// Recursion-safe via consecutiveAutoTurns: at most autoTurnCap chained turns
// per human input. Reset by HandleHumanInput.
func (o *Orchestrator) autoRoutePlanner(ctx context.Context, fromRole, fromContent string) {
	o.mu.Lock()
	if o.consecutiveAutoTurns >= autoTurnCap {
		o.mu.Unlock()
		logging.Warn("Auto-turn cap reached — dropping forwarded message", "from", fromRole, "cap", autoTurnCap)
		return
	}
	o.consecutiveAutoTurns++
	sid := o.sessionID
	o.mu.Unlock()
	if sid == "" {
		return
	}
	// Brief delay so the source message has fully rendered before Planner runs.
	time.Sleep(200 * time.Millisecond)
	prompt := fmt.Sprintf(
		"The %s agent just sent the following report. React by taking action — emit CREATE_TASK: directives if work needs to be created or reassigned, or only write a chat reply if you need to inform the human. Do NOT echo this report back. Stay silent in chat unless you have something for the human.\n\n[%s]: %s",
		fromRole, fromRole, fromContent,
	)
	o.runAgentTurn(ctx, sid, "planner", prompt)
}

// autoTurnCap is the maximum number of consecutive auto-routed Planner turns
// per human input. Resets when the user types something. Prevents agent loops
// (Observer → Planner → action → Observer notices → Planner → ...).
const autoTurnCap = 5

// handleProjectBrief parses a PROJECT_BRIEF directive from a Planner intake-mode
// response and POSTs it to the ACT server. On success, clears intakeMode so the
// next Planner turn falls back to normal task-decomposition behavior.
func (o *Orchestrator) handleProjectBrief(_ context.Context, content string) {
	brief := parseProjectBrief(content)
	if brief == nil {
		return // still gathering — Planner hasn't summarized yet
	}

	o.mu.RLock()
	name := o.projectName
	dir := o.projectDir
	o.mu.RUnlock()

	client := act.NewClient("planner", name)
	if !client.IsAvailable() {
		logging.Warn("ACT server unavailable — cannot persist PROJECT_BRIEF")
		return
	}
	if err := client.CreateProject(name, dir, brief.Description, brief.TechStack, brief.Constraints, brief.SuccessCriteria, brief.AgentsInvolved); err != nil {
		logging.Warn("Failed to POST project brief", "project", name, "error", err)
		return
	}
	o.mu.Lock()
	o.intakeMode = false
	o.mu.Unlock()
	logging.Info("PROJECT_BRIEF accepted — exiting INTAKE mode", "project", name)
}

// handlePlannerTaskDirectives parses CREATE_TASK directives from a Planner
// response and POSTs each one to the ACT server. On first successful task
// creation, also spawns the Runner subprocess if not already running.
//
// The ctx parameter is currently unused — act.Client doesn't accept a context
// yet — but is kept on the signature so the caller's loop context can be
// threaded through once the HTTP client is made context-aware.
func (o *Orchestrator) handlePlannerTaskDirectives(_ context.Context, content string) {
	tasks := parseCreateTaskDirectives(content)
	if len(tasks) == 0 {
		return
	}

	client := act.NewClient("planner", os.Getenv("ACT_PROJECT"))
	if !client.IsAvailable() {
		logging.Warn("ACT server unavailable — cannot create tasks from Planner directives")
		return
	}

	// Lazy spawn: the first time tasks appear, bring up the Tier 2 swarm.
	// StartSwarm is idempotent per-role so calling it on every batch is safe.
	if len(o.app.SwarmSpecs) > 0 && !o.runnerSpawner.IsRunning() {
		logging.Info("First task batch — spawning Tier 2 swarm", "task_count", len(tasks))
		if err := o.runnerSpawner.StartSwarm(o.app.SwarmSpecs); err != nil {
			logging.Warn("Failed to start swarm", "error", err)
		}
	}

	created := 0
	for _, task := range tasks {
		title := task.Title
		if title == "" {
			title = task.Name
		}
		caps := task.RequiredCapabilities
		if len(caps) == 0 {
			caps = task.Capabilities
		}
		priority := task.Priority
		if priority == "" {
			priority = "medium"
		}
		metadata := task.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadata["createdBy"] = "planner"
		if proj := os.Getenv("ACT_PROJECT"); proj != "" {
			metadata["projectName"] = proj
		}

		taskID, err := client.CreateTask(title, task.Description, caps, priority, task.Dependencies, metadata)
		if err != nil {
			logging.Warn("Failed to create task from Planner directive", "title", title, "error", err)
			continue
		}
		logging.Info("Task created from Planner directive", "task_id", taskID, "title", title)
		created++
	}
	// Trigger a Nomik incremental rescan after task creation so Planner sees fresh
	// graph state on next iteration.
	if created > 0 {
		go o.maybeRescanNomik(context.Background())
	}
}

// maybeRescanNomik runs `nomik scan:incremental` if Nomik is enabled and a
// rescan isn't already in flight (debounced — at most one rescan at a time).
// Called after task creation and task completion to keep the graph fresh.
func (o *Orchestrator) maybeRescanNomik(ctx context.Context) {
	o.mu.RLock()
	enabled := o.nomikEnabled && o.nomikAvailable
	dir := o.projectDir
	o.mu.RUnlock()
	if !enabled {
		return
	}

	o.rescanInflightMu.Lock()
	if o.rescanInflight {
		o.rescanInflightMu.Unlock()
		return
	}
	o.rescanInflight = true
	o.rescanInflightMu.Unlock()

	defer func() {
		o.rescanInflightMu.Lock()
		o.rescanInflight = false
		o.rescanInflightMu.Unlock()
	}()

	if err := nomik.Rescan(ctx, dir); err != nil {
		logging.Debug("Nomik incremental rescan skipped", "error", err)
	}
}

// ─── Background loop: Observer monitoring ──────────────────────────────────────

const (
	observerInterval = 120 * time.Second
	// stuckTaskMinutes was 30 — way too high for the typical Snake-Arena-class
	// run that completes in 5 minutes. Lowered to 3 so Observer can actually
	// catch real anomalies during a single short run.
	stuckTaskMinutes     = 3
	staleLockMinutes     = 5
	bottleneckTaskCount  = 3
	validationPollPeriod = 10 * time.Second
	// observerHealthEvery sets how often the Observer emits an "all clear"
	// system message when no anomalies are detected. Without this it's
	// indistinguishable from a dead goroutine. Every 4 cycles ≈ every 8 minutes.
	observerHealthEvery = 4
)

// observerLoop runs the Observer monitoring loop. Every ~120s it polls the
// server for system state, runs anomaly detection, and if anomalies are found
// AND the Observer agent is idle, runs the Observer agent with the anomaly
// report as input.
func (o *Orchestrator) observerLoop(ctx context.Context) {
	defer o.loopWG.Done()

	if o.getAgent("observer") == nil {
		return
	}

	ticker := time.NewTicker(observerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.runObserverCheck(ctx)
		}
	}
}

func (o *Orchestrator) runObserverCheck(ctx context.Context) {
	if o.IsAnyBusy("") {
		return // don't interrupt active turns
	}

	client := act.NewClient("observer", os.Getenv("ACT_PROJECT"))
	if !client.IsAvailable() {
		return
	}

	snapshot, err := buildStatusSnapshot(client)
	if err != nil {
		logging.Debug("Observer status snapshot failed", "error", err)
		return
	}

	o.mu.Lock()
	o.observerCycles++
	cycle := o.observerCycles
	sid := o.sessionID
	o.mu.Unlock()

	anomalies := detectAnomalies(snapshot)
	if len(anomalies) == 0 {
		// Silent watchdog — but every Nth cycle emit a health ping so the
		// user knows it's still alive. Without this Observer is
		// indistinguishable from a dead goroutine on healthy runs.
		if cycle%observerHealthEvery == 0 && sid != "" {
			active := 0
			for _, t := range snapshot.Tasks {
				if t.Status == "in_progress" || t.Status == "assigned" {
					active++
				}
			}
			o.emitSystemMessage(ctx, sid, fmt.Sprintf("👁  Observer: all clear (%d active task(s), no anomalies)", active))
		}
		return
	}

	if sid == "" {
		return
	}
	prompt := buildObserverPrompt(snapshot, anomalies)

	// No-op gate: if this prompt is byte-identical to the last one we sent,
	// the Observer would just repeat itself. Skip the LLM call entirely.
	// The hash is cleared by HandleHumanInput so a fresh human turn always
	// re-arms the Observer.
	sum := sha256.Sum256([]byte(prompt))
	hash := hex.EncodeToString(sum[:8])
	o.mu.Lock()
	skip := hash == o.lastObserverPromptHash
	prevHash := o.lastObserverPromptHash
	o.lastObserverPromptHash = hash
	o.mu.Unlock()
	if skip {
		logging.Info("observer.noop_gate.skip",
			"reason", "prompt_unchanged",
			"hash", hash,
			"cycle", cycle,
			"anomalies", len(anomalies),
			"prompt_bytes", len(prompt),
		)
		return
	}
	logging.Info("observer.noop_gate.fire",
		"reason", "prompt_changed",
		"hash", hash,
		"prev_hash", prevHash,
		"cycle", cycle,
		"anomalies", len(anomalies),
		"prompt_bytes", len(prompt),
	)

	o.runAgentTurn(ctx, sid, "observer", prompt)
}

// buildStatusSnapshot polls the server for tasks, agents, file locks, and
// recent log entries in parallel.
func buildStatusSnapshot(client *act.Client) (*StatusSnapshot, error) {
	type result struct {
		key string
		val string
		err error
	}
	ch := make(chan result, 4)

	go func() { v, e := client.ListTasks(); ch <- result{"tasks", v, e} }()
	go func() { v, e := client.ListAgents(); ch <- result{"agents", v, e} }()
	go func() { v, e := client.GetFileLocks(); ch <- result{"locks", v, e} }()
	go func() { v, e := client.GetLog(20); ch <- result{"log", v, e} }()

	results := map[string]string{}
	for i := 0; i < 4; i++ {
		r := <-ch
		if r.err == nil {
			results[r.key] = r.val
		}
	}

	snap := &StatusSnapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if v := results["tasks"]; v != "" {
		var tasks []TaskSummary
		if err := unmarshalListField(v, "tasks", &tasks); err == nil {
			snap.Tasks = tasks
		}
	}
	if v := results["agents"]; v != "" {
		var agents []AgentSummary
		if err := unmarshalListField(v, "agents", &agents); err == nil {
			snap.Agents = agents
		}
	}
	if v := results["locks"]; v != "" {
		var locks []FileLockSummary
		if err := unmarshalListField(v, "locks", &locks); err == nil {
			snap.FileLocks = locks
		}
	}
	if v := results["log"]; v != "" {
		var entries []LogEntry
		if err := unmarshalListField(v, "events", &entries); err == nil {
			snap.RecentEvents = entries
		}
	}

	return snap, nil
}

// unmarshalListField tries to decode raw JSON as either a top-level array or
// as `{ "<key>": [...] }`. Server endpoints inconsistently return one or the other.
func unmarshalListField(raw string, key string, out any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty")
	}
	if strings.HasPrefix(raw, "[") {
		return json.Unmarshal([]byte(raw), out)
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		return err
	}
	field, ok := wrapper[key]
	if !ok {
		return fmt.Errorf("key %q not found", key)
	}
	return json.Unmarshal(field, out)
}

// detectAnomalies runs the 6 anomaly detection rules from observer.go's prompt
// against a status snapshot.
func detectAnomalies(s *StatusSnapshot) []Anomaly {
	if s == nil {
		return nil
	}
	var anomalies []Anomaly
	now := time.Now()

	// 1. Stuck tasks
	for _, t := range s.Tasks {
		if t.Status != "assigned" && t.Status != "in_progress" {
			continue
		}
		age := minutesSince(t.CreatedAt, now)
		if age <= stuckTaskMinutes {
			continue
		}
		sev := SeverityWarning
		if age > stuckTaskMinutes*2 {
			sev = SeverityCritical
		}
		anomalies = append(anomalies, Anomaly{
			Severity: sev,
			Category: CategoryStuckTask,
			Message:  fmt.Sprintf("Task %q has been %s for %dmin (assigned to %s)", t.Title, t.Status, age, orNobody(t.AssignedAgent)),
			TaskID:   t.ID,
			AgentID:  t.AssignedAgent,
		})
	}

	// 2. Stale file locks
	for _, l := range s.FileLocks {
		age := minutesSince(l.LockedAt, now)
		if age <= staleLockMinutes {
			continue
		}
		anomalies = append(anomalies, Anomaly{
			Severity: SeverityWarning,
			Category: CategoryStaleLock,
			Message:  fmt.Sprintf("File %q locked by %s for %dmin (task: %s)", l.File, l.AgentID, age, l.TaskID),
			AgentID:  l.AgentID,
			TaskID:   l.TaskID,
		})
	}

	// 3. Idle agents while tasks pending
	pending := 0
	for _, t := range s.Tasks {
		if t.Status == "pending" {
			pending++
		}
	}
	if pending > 0 {
		for _, a := range s.Agents {
			if a.Status == "online" && a.CurrentTask == "" {
				anomalies = append(anomalies, Anomaly{
					Severity: SeverityWarning,
					Category: CategoryIdleAgent,
					Message:  fmt.Sprintf("Agent %s is idle while %d task(s) are pending", a.ID, pending),
					AgentID:  a.ID,
				})
			}
		}
	}

	// 4. Unvalidated completed work
	var completed []string
	for _, t := range s.Tasks {
		if t.Status == "completed" {
			label := t.Title
			if label == "" {
				label = t.ID
			}
			completed = append(completed, label)
		}
	}
	if len(completed) > 0 {
		sev := SeverityInfo
		if len(completed) > 3 {
			sev = SeverityWarning
		}
		anomalies = append(anomalies, Anomaly{
			Severity: sev,
			Category: CategoryUnvalidated,
			Message:  fmt.Sprintf("%d completed task(s) not yet submitted for validation: %s", len(completed), strings.Join(completed, ", ")),
		})
	}

	// 5. Bottlenecks (3+ active tasks per agent)
	counts := map[string]int{}
	for _, t := range s.Tasks {
		if (t.Status == "assigned" || t.Status == "in_progress") && t.AssignedAgent != "" {
			counts[t.AssignedAgent]++
		}
	}
	for agentID, n := range counts {
		if n >= bottleneckTaskCount {
			anomalies = append(anomalies, Anomaly{
				Severity: SeverityWarning,
				Category: CategoryBottleneck,
				Message:  fmt.Sprintf("Agent %s has %d active tasks — possible bottleneck", agentID, n),
				AgentID:  agentID,
			})
		}
	}

	// Sort by severity
	severityRank := map[AnomalySeverity]int{SeverityCritical: 0, SeverityWarning: 1, SeverityInfo: 2}
	for i := 0; i < len(anomalies); i++ {
		for j := i + 1; j < len(anomalies); j++ {
			if severityRank[anomalies[j].Severity] < severityRank[anomalies[i].Severity] {
				anomalies[i], anomalies[j] = anomalies[j], anomalies[i]
			}
		}
	}

	return anomalies
}

func orNobody(s string) string {
	if s == "" {
		return "nobody"
	}
	return s
}

func minutesSince(timestamp string, now time.Time) int {
	if timestamp == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return 0
	}
	return int(now.Sub(t).Minutes())
}

func buildObserverPrompt(s *StatusSnapshot, anomalies []Anomaly) string {
	var sb strings.Builder
	sb.WriteString("Your monitoring loop just ran. Here is the system snapshot and the anomalies you detected:\n\n")
	sb.WriteString(fmt.Sprintf("Snapshot: %s | Tasks: %d, Agents: %d, FileLocks: %d\n\n",
		s.Timestamp, len(s.Tasks), len(s.Agents), len(s.FileLocks)))
	sb.WriteString("Anomalies:\n")
	for _, a := range anomalies {
		sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", strings.ToUpper(string(a.Severity)), a.Category, a.Message))
	}
	sb.WriteString("\nWrite a SHORT message to the Planner (addressed @planner) describing what you observed and a concrete suggested action for each anomaly. Do NOT make decisions yourself. Do NOT echo this prompt back. Do NOT call any tools — just write your message directly. Keep it under 6 lines.")
	return sb.String()
}

// ─── Background loop: validation polling (Assurance routing) ───────────────────

func (o *Orchestrator) validationPollLoop(ctx context.Context) {
	defer o.loopWG.Done()

	if o.getAgent("assurance") == nil {
		return
	}

	ticker := time.NewTicker(validationPollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.checkPendingValidation(ctx)
		}
	}
}

func (o *Orchestrator) checkPendingValidation(ctx context.Context) {
	if o.IsAnyBusy("") {
		return
	}
	client := act.NewClient("assurance", os.Getenv("ACT_PROJECT"))
	if !client.IsAvailable() {
		return
	}
	raw, err := client.GetPendingValidation()
	if err != nil || raw == "" {
		return
	}
	var tasks []TaskSummary
	if err := unmarshalListField(raw, "tasks", &tasks); err != nil {
		return
	}
	for _, t := range tasks {
		if o.alreadySeen("validation:" + t.ID) {
			continue
		}
		o.markSeen("validation:" + t.ID)
		o.routeToAssurance(ctx, client, t)
		break // one at a time
	}
}

func (o *Orchestrator) routeToAssurance(ctx context.Context, client *act.Client, t TaskSummary) {
	prompt := buildValidationPrompt(t)

	o.mu.RLock()
	sid := o.sessionID
	o.mu.RUnlock()
	if sid == "" {
		return
	}

	// Capture Assurance's response by snapshotting message count, then run turn
	o.runAgentTurn(ctx, sid, "assurance", prompt)

	// Parse the most recent assurance message
	msgs, err := o.app.Messages.List(ctx, sid)
	if err != nil {
		return
	}
	var lastAssurance string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == message.Assistant && o.GetOwner(msgs[i].ID) == "assurance" {
			lastAssurance = msgs[i].Content().String()
			break
		}
	}
	if lastAssurance == "" {
		return
	}

	verdict := parseValidationVerdict(t.ID, lastAssurance)
	if verdict == nil {
		logging.Warn("Could not parse validation verdict from Assurance response", "task_id", t.ID)
		return
	}

	// Submit verdict to server
	criteriaResults := make([]map[string]any, 0, len(verdict.CriteriaResults))
	for _, cr := range verdict.CriteriaResults {
		criteriaResults = append(criteriaResults, map[string]any{
			"criterion": cr.Criterion,
			"passed":    cr.Passed,
			"score":     cr.Score,
			"feedback":  cr.Feedback,
		})
	}
	if err := client.SubmitVerdict(t.ID, "assurance", verdict.Passed, verdict.OverallScore, criteriaResults, verdict.Gaps, verdict.Feedback); err != nil {
		logging.Warn("Failed to submit validation verdict", "task_id", t.ID, "error", err)
	}
}

// ─── Background loop: QA polling ───────────────────────────────────────────────

func (o *Orchestrator) qaPollLoop(ctx context.Context) {
	defer o.loopWG.Done()

	if o.getAgent("qa") == nil {
		return
	}

	ticker := time.NewTicker(validationPollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.checkValidatedTasks(ctx)
		}
	}
}

func (o *Orchestrator) checkValidatedTasks(ctx context.Context) {
	if o.IsAnyBusy("") {
		return
	}
	client := act.NewClient("qa", os.Getenv("ACT_PROJECT"))
	if !client.IsAvailable() {
		return
	}
	raw, err := client.GetValidatedTasks()
	if err != nil || raw == "" {
		return
	}
	var tasks []TaskSummary
	if err := unmarshalListField(raw, "tasks", &tasks); err != nil {
		return
	}
	for _, t := range tasks {
		if o.alreadySeen("qa:" + t.ID) {
			continue
		}
		o.markSeen("qa:" + t.ID)
		o.routeToQA(ctx, t)
		break
	}
}

func (o *Orchestrator) routeToQA(ctx context.Context, t TaskSummary) {
	o.mu.RLock()
	sid := o.sessionID
	o.mu.RUnlock()
	if sid == "" {
		return
	}

	result := ""
	if t.Metadata != nil {
		if r, ok := t.Metadata["result"].(string); ok {
			result = r
		}
	}
	output := ValidatedOutput{
		TaskID:    t.ID,
		TaskTitle: t.Title,
		AgentID:   t.AssignedAgent,
		Result:    result,
		AddedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	prompt := buildSynthesisPrompt(output)
	o.runAgentTurn(ctx, sid, "qa", prompt)
}

// ─── Parsers ───────────────────────────────────────────────────────────────────

var (
	createTaskInlineRegex = regexp.MustCompile(`CREATE_TASK:\s*(\{[^}]+\})`)
	clarificationRegex    = regexp.MustCompile(`(?s)NEED_CLARIFICATION:\s*@(\S+)\s+(.*)`)
)

// extractJSONContaining finds the smallest JSON object in text that contains
// the given substring (typically a JSON key like `"criteriaResults"`).
// Uses brace counting to handle nested objects/arrays correctly.
// Returns the empty string if no balanced object is found.
func extractJSONContaining(text, marker string) string {
	idx := strings.Index(text, marker)
	if idx == -1 {
		return ""
	}

	// Walk backwards to find the enclosing opening brace
	depth := 0
	start := -1
	for i := idx; i >= 0; i-- {
		switch text[i] {
		case '}':
			depth++
		case '{':
			if depth == 0 {
				start = i
				i = -1 // break outer loop
				continue
			}
			depth--
		}
		if start != -1 {
			break
		}
	}
	if start == -1 {
		return ""
	}

	// Walk forwards counting braces, ignoring braces inside strings.
	// Returns when balanced.
	depth = 0
	inString := false
	escape := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			switch ch {
			case '\\':
				escape = true
			case '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}

// parseCreateTaskDirectives extracts CREATE_TASK directives from a Planner response.
// Supports two patterns:
//   1. CREATE_TASK: { json } — inline directives
//   2. { "tasks": [ ... ] } — full planning JSON block
func parseCreateTaskDirectives(content string) []TaskDef {
	var tasks []TaskDef

	// Pattern 1: inline directives
	for _, match := range createTaskInlineRegex.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		var t TaskDef
		if err := json.Unmarshal([]byte(match[1]), &t); err == nil {
			tasks = append(tasks, t)
		}
	}

	if len(tasks) > 0 {
		return tasks
	}

	// Pattern 2: full planning JSON with "tasks": [...]
	if jsonMatch := extractJSONContaining(content, `"tasks"`); jsonMatch != "" {
		var wrapper struct {
			Tasks []TaskDef `json:"tasks"`
		}
		if err := json.Unmarshal([]byte(jsonMatch), &wrapper); err == nil {
			tasks = append(tasks, wrapper.Tasks...)
		}
	}

	return tasks
}

// parseProjectBrief extracts a PROJECT_BRIEF directive from a Planner intake
// response. Looks for a JSON object containing "description" near a
// PROJECT_BRIEF: marker, falls back to any JSON containing "successCriteria".
// Returns nil if no valid brief is found.
func parseProjectBrief(content string) *ProjectBrief {
	// Prefer JSON immediately following the explicit marker.
	if idx := strings.Index(content, "PROJECT_BRIEF:"); idx != -1 {
		tail := content[idx:]
		if jsonStr := extractJSONContaining(tail, `"description"`); jsonStr != "" {
			var b ProjectBrief
			if err := json.Unmarshal([]byte(jsonStr), &b); err == nil && b.Description != "" {
				return &b
			}
		}
	}
	// Fallback: any JSON in the response that has both description and successCriteria.
	if jsonStr := extractJSONContaining(content, `"successCriteria"`); jsonStr != "" {
		var b ProjectBrief
		if err := json.Unmarshal([]byte(jsonStr), &b); err == nil && b.Description != "" {
			return &b
		}
	}
	return nil
}

// parseValidationVerdict extracts an Assurance verdict (JSON) from a free-form response.
// Returns nil if no JSON with criteriaResults is found.
func parseValidationVerdict(taskID, raw string) *ValidationVerdict {
	jsonStr := extractJSONContaining(raw, `"criteriaResults"`)
	if jsonStr == "" {
		return nil
	}

	var parsed struct {
		CriteriaResults       []CriterionResult `json:"criteriaResults"`
		OverallScore          int               `json:"overallScore"`
		SelfVerificationValid bool              `json:"selfVerificationValid"`
		Gaps                  string            `json:"gaps"`
		Feedback              string            `json:"feedback"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil
	}

	const passThreshold = 95
	return &ValidationVerdict{
		TaskID:                  taskID,
		Passed:                  parsed.OverallScore >= passThreshold,
		OverallScore:            parsed.OverallScore,
		CriteriaResults:         parsed.CriteriaResults,
		SelfVerificationChecked: true,
		SelfVerificationValid:   parsed.SelfVerificationValid,
		Gaps:                    parsed.Gaps,
		Feedback:                parsed.Feedback,
		Timestamp:               time.Now().UTC().Format(time.RFC3339),
	}
}

// parseSynthesisResponse parses a QA agent's response, returning the kind of
// outcome ("complete", "need_clarification", "in_progress") and any extracted
// fields. Mirrors the TypeScript synthesizer parser.
func parseSynthesisResponse(raw string) (kind string, summary string, targetAgent string, question string) {
	if idx := strings.Index(raw, "SYNTHESIS_COMPLETE:"); idx >= 0 {
		summary = strings.TrimSpace(raw[idx+len("SYNTHESIS_COMPLETE:"):])
		return "complete", summary, "", ""
	}
	if m := clarificationRegex.FindStringSubmatch(raw); len(m) >= 3 {
		return "need_clarification", "", m[1], strings.TrimSpace(m[2])
	}
	return "in_progress", "", "", ""
}

// ─── Prompt builders ──────────────────────────────────────────────────────────

func buildValidationPrompt(t TaskSummary) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("A swarm agent has submitted task %q for your validation. Score it against the success criteria below and return your verdict.\n\n", taskLabel(t)))
	sb.WriteString("Task description:\n")
	sb.WriteString(t.Description)
	sb.WriteString("\n\nSuccess criteria you must score:\n")
	for i, c := range t.SuccessCriteria {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, c))
	}
	sb.WriteString("\nAgent's submitted result:\n")
	if t.Metadata != nil {
		if r, ok := t.Metadata["result"].(string); ok {
			if len(r) > 4000 {
				r = r[:4000] + "..."
			}
			sb.WriteString(r)
		}
	}
	sb.WriteString("\n\nNow respond with your verdict as a JSON object with this exact shape (no surrounding prose, no code fences):\n")
	sb.WriteString(`{"passed": true|false, "score": 0-100, "criteriaResults": [{"criterion":"...","passed":true|false,"reasoning":"..."}], "gaps":"...","feedback":"..."}` + "\n")
	sb.WriteString("Pass = score >= 95. Do NOT call any tools — write the JSON directly. Do NOT echo this prompt.")
	return sb.String()
}

func buildSynthesisPrompt(o ValidatedOutput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("A task has just passed Assurance validation and is ready to be integrated into the project deliverable. Your job is to assemble it.\n\n"))
	sb.WriteString(fmt.Sprintf("Task: %s\nAgent: %s\nValidation score: %d/100\n\n", o.TaskTitle, o.AgentID, o.ValidationScore))
	sb.WriteString("Validated output:\n")
	if len(o.Result) > 4000 {
		sb.WriteString(o.Result[:4000] + "...")
	} else {
		sb.WriteString(o.Result)
	}
	sb.WriteString("\n\nReview this output. If it integrates cleanly with the rest of the deliverable, write a SHORT message ending with the line `SYNTHESIS_COMPLETE: <one-sentence summary>`. If you need clarification from the producing agent, end with `NEED_CLARIFICATION: @<agent_id> <question>`. Do NOT echo this prompt. Do NOT call any tools. Keep it under 6 lines.")
	return sb.String()
}

func taskLabel(t TaskSummary) string {
	if t.Title != "" {
		return t.Title
	}
	return t.ID
}

// ─── Seen-task tracking ────────────────────────────────────────────────────────

func (o *Orchestrator) alreadySeen(key string) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.seenTasks[key]
}

func (o *Orchestrator) markSeen(key string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.seenTasks[key] = true
}
