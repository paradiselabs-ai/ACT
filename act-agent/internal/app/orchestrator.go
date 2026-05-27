package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/acp"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/act"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/prompt"
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
	attemptCount   map[string]int    // validation/qa attempt counter keyed "validation:TASK" / "qa:TASK"
	dispatchedMsgs map[string]bool   // finished assistant message IDs already dispatched — prevents duplicate CREATE_TASK / brief handling when pubsub re-fires UpdatedEvent on the same message

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
	intakeMode      bool
	projectName     string // ACT_PROJECT — derived from --project flag or cwd basename
	resumeContext   string // non-empty when project exists on server; injected into first Planner turn
	firstPlannerTurn bool  // cleared after the first turn so resumeContext is only prepended once

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

	// recentAutoRoutes holds the wall-clock timestamps of recent auto-routed
	// Planner turns. Sliding-window cap: more than autoTurnCap fires within
	// autoRouteWindow gets the next one dropped, regardless of trigger source.
	// Cleared by HandleHumanInput (so a fresh human turn always gets a clean
	// budget). Audit Fix 6 — replaces the prior consecutiveAutoTurns counter
	// which only counted back-to-back fires and missed three documented loops
	// (QA watchdog re-fire, Assurance verdict mirror, Observer 120s echo) that
	// resetting the counter between cycles let through.
	recentAutoRoutes []time.Time

	// lastTurnAt[role] records the start time of the most recent runAgentTurn
	// for that Tier 1 role. Powers the Observer Tier 1 watchdog (α-2): when a
	// queue is non-empty AND the responsible role is idle AND its last turn was
	// >tier1StuckThreshold ago, Observer re-triggers it directly. Mu-protected.
	lastTurnAt map[string]time.Time

	// recentDispatchHashes maps sha256(sorted task spec set) → dispatch time.
	// Defense-in-depth dedup: when a flaky Planner model emits the same
	// CREATE_TASK batch across two distinct turns (different msg.IDs, so the
	// dispatchedMsgs check doesn't catch it), this stops the second batch
	// from re-dispatching. Window: dispatchHashWindow. Lazily GC'd on insert.
	dispatchHashMu       sync.Mutex
	recentDispatchHashes map[string]time.Time
}

// dispatchHashWindow is how long a content-hash blocks a re-dispatch of the
// same CREATE_TASK batch. Observed flake: same 5-task list re-emitted ~20s
// later on a separate Planner turn; 60s comfortably covers that without
// blocking legitimate re-runs the user might trigger minutes later.
const dispatchHashWindow = 60 * time.Second

// NewOrchestrator creates an orchestrator wired to the given app.
func NewOrchestrator(app *App) *Orchestrator {
	return &Orchestrator{
		app:           app,
		messageOwners: make(map[string]string),
		seenTasks:     make(map[string]bool),
		dispatchedMsgs: make(map[string]bool),
		attemptCount:  make(map[string]int),
		runnerSpawner: runner.NewSpawner(),
		lastTurnAt:    make(map[string]time.Time),
		recentDispatchHashes: make(map[string]time.Time),
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
// This is the main entry point from the TUI.
// Runs in a goroutine from the caller; blocks until the Planner finishes its turn.
func (o *Orchestrator) HandleHumanInput(ctx context.Context, sessionID string, text string, attachments ...message.Attachment) {
	// Reset the auto-turn counter — every human input gets a fresh budget of
	// chained auto-routed Planner turns from the other Tier 1 agents.
	o.mu.Lock()
	o.recentAutoRoutes = nil          // human input clears the sliding window
	o.lastObserverPromptHash = "" // re-arm Observer no-op gate

	// On the first Planner turn of a resumed session, prepend the project
	// context so the Planner doesn't fall back into INTAKE mode. The Planner's
	// system prompt decides INTAKE vs BUILD based on what it sees in the
	// conversation — without this injection it has no evidence of an existing
	// project and asks intake questions again.
	if o.firstPlannerTurn && o.resumeContext != "" {
		text = o.resumeContext + "\n\nUser message: " + text
		o.firstPlannerTurn = false
	}
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
	data, found, err := client.GetProject(name)
	if err != nil {
		logging.Warn("Project lookup failed — skipping intake", "project", name, "error", err)
		return
	}
	o.mu.Lock()
	o.intakeMode = !found
	if found {
		// Build a resume context string injected into the first Planner turn so it
		// knows to skip INTAKE and go straight to BUILD. Without this the Planner
		// sees a blank conversation and falls back to asking intake questions.
		brief := briefViewFromGetProject(name, data)
		briefsCount := 0
		if b, ok := data["briefs"].(map[string]any); ok {
			briefsCount = len(b)
		}
		// Also fetch the current task list so the Planner sees what's
		// already dispatched and doesn't re-emit CREATE_TASK for tasks
		// the server already knows about. Best-effort — if the call
		// fails we render a brief-only block.
		var tasks []TaskSummary
		if raw, err := client.ListTasks(); err == nil && raw != "" {
			if uerr := unmarshalListField(raw, "tasks", &tasks); uerr != nil {
				logging.Warn("project_resume_task_decode_failed", "project", name, "error", uerr)
				tasks = nil
			}
		} else if err != nil {
			logging.Warn("project_resume_task_fetch_failed", "project", name, "error", err)
		}

		o.resumeContext = renderBriefContext("resume", brief, tasks)
		o.firstPlannerTurn = true
		o.mu.Unlock()
		logging.Info("project_resume",
			"project", name,
			"desc", truncate(brief.Description, 80),
			"tech", truncate(brief.TechStack, 60),
			"has_success_criteria", brief.SuccessCriteria != "",
			"agents_involved", len(brief.AgentsInvolved),
			"brief_count", briefsCount,
			"in_flight_or_done_tasks", len(tasks),
		)
		// Silent envelope-unwrap regressions (today's GetProject bug) show up as
		// all-blank preview fields. Fire a loud warning so the next regression
		// is caught within seconds of startup instead of after the first turn.
		if brief.Description == "" && brief.TechStack == "" {
			logging.Warn("project_resume_blank_fields",
				"project", name,
				"reason", "description and techStack both empty — possible response envelope regression",
			)
		}
		return
	}
	o.mu.Unlock()
	logging.Info("No server-side project record — entering INTAKE mode", "project", name)
}

// briefViewFromGetProject extracts a BriefView from the map returned by
// client.GetProject. Keys not present become empty fields — the
// renderer omits missing fields cleanly rather than emitting "undefined."
func briefViewFromGetProject(name string, data map[string]any) BriefView {
	bv := BriefView{ProjectName: name}
	bv.Description, _ = data["description"].(string)
	bv.TechStack, _ = data["techStack"].(string)
	bv.Constraints, _ = data["constraints"].(string)
	bv.SuccessCriteria, _ = data["successCriteria"].(string)
	if raw, ok := data["agentsInvolved"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok && s != "" {
				bv.AgentsInvolved = append(bv.AgentsInvolved, s)
			}
		}
	}
	return bv
}

// runAgentTurn executes a single turn for the given role.
// It sets the current speaker (for message ownership tagging), runs the agent,
// and waits for completion.
func (o *Orchestrator) runAgentTurn(ctx context.Context, sessionID string, role string, content string, attachments ...message.Attachment) {
	agentSvc := o.getAgent(role)
	if agentSvc == nil {
		logging.Error("No Tier 1 agent registered for role — turn dropped", "role", role)
		o.emitSystemMessage(context.Background(), sessionID, fmt.Sprintf("⚠  %s is not configured in ~/.act.json — turn dropped", role))
		return
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
	o.lastTurnAt[role] = time.Now()
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
		o.emitSystemMessage(context.Background(), sessionID, fmt.Sprintf("⚠  %s could not start — %s", role, humanReadableAgentError(err)))
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
		o.emitSystemMessage(context.Background(), sessionID, fmt.Sprintf("⚠  %s failed — %s", role, humanReadableAgentError(result.Error)))
	}
}

// humanReadableAgentError maps common provider/agent errors into a short line
// the user can act on. Raw provider errors are often JSON or long — strip them
// down to the useful signal (auth failure, rate limit, model gone, etc).
func humanReadableAgentError(err error) string {
	if err == nil {
		return "unknown error"
	}
	if errors.Is(err, acp.ErrACPSubprocessExited) {
		return "ACP agent subprocess exited — relaunch ACT (in-place restart not supported in the alpha)"
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "api key") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "401"):
		return "API key missing or invalid (check ~/.act.json / env vars)"
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "429"):
		return "provider rate-limited — try again shortly"
	case strings.Contains(lower, "model not found") || strings.Contains(lower, "no such model") || strings.Contains(lower, "404"):
		return "model not available (may have been deprecated — update ~/.act.json)"
	case strings.Contains(lower, "context length") || strings.Contains(lower, "token limit"):
		return "context too long for this model — start a new session"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return "provider timed out"
	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host"):
		return "provider unreachable (network or endpoint misconfigured)"
	}
	if len(msg) > 200 {
		msg = msg[:200] + "…"
	}
	return msg
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
	//
	// Pre-count new events so we can batch when a single poll returns a flood
	// (swarm doing rapid task cycling, replay backlog, etc). Emitting one
	// message per event spams the PubSub → every emit triggers a full chat
	// re-render → the Bubbletea Update loop saturates and input latency
	// balloons. Observed cap: render cost scales ~O(n) with message count,
	// so >floodThreshold new events in one tick degrade input latency
	// noticeably. When we exceed the threshold, compact everything into a
	// single summary line.
	newest := since
	newEventCount := 0
	for _, ev := range resp.Events {
		if ev.Timestamp > since {
			newEventCount++
		}
	}
	batchMode := newEventCount > coordEventFloodThreshold
	typeCounts := map[string]int{}
	failedTaskCount := 0
	var firstFailedSummary string

	for i := len(resp.Events) - 1; i >= 0; i-- {
		ev := resp.Events[i]
		if ev.Timestamp == "" || ev.Timestamp <= since {
			continue
		}
		if batchMode {
			typeCounts[ev.Type]++
		} else {
			text := formatCoordEvent(ev)
			if text != "" {
				o.emitSystemMessage(ctx, sid, text)
			}
		}

		// Task failures need more than a chat line — the Planner must see
		// them so it can decide whether to retry, reassign, or abandon. In
		// non-batch mode each failure autoroutes; in batch mode we collapse
		// to a single autoroute covering the whole burst so we don't swamp
		// the Planner with N chained turns. The autoRoutePlanner recursion
		// cap (autoTurnCap) protects against loops regardless.
		if ev.Type == "task_failed" {
			failedTaskCount++
			if !batchMode {
				taskID, _ := ev.Data["taskId"].(string)
				result, _ := ev.Data["result"].(string)
				if len(result) > 400 {
					result = result[:400] + "..."
				}
				// Full taskID — the Planner needs it verbatim to call
				// `act_cli task retry <id>` / `task abandon <id>`. Audit
				// Fix 13b (entry 6.3): previously truncated to 36 chars,
				// which the Planner would then copy into a failing
				// act_cli call.
				summary := fmt.Sprintf(
					"Task %s just failed (agent %s). Error: %s",
					taskID, ev.Agent, result,
				)
				go o.autoRoutePlannerV(ctx, "system", summary, variantSystemEscalation)
			} else if firstFailedSummary == "" {
				taskID, _ := ev.Data["taskId"].(string)
				firstFailedSummary = fmt.Sprintf("task %s (agent %s)", taskID, ev.Agent)
			}
		}

		if ev.Timestamp > newest {
			newest = ev.Timestamp
		}
	}

	if batchMode && failedTaskCount > 0 {
		go o.autoRoutePlannerV(ctx, "system", fmt.Sprintf(
			"%d task(s) failed in the last burst (e.g. %s). Use act_cli context --project <name> to see current task state, then retry/abandon/reassign as appropriate.",
			failedTaskCount, firstFailedSummary,
		), variantSystemEscalation)
	}

	if batchMode {
		// Render a single compacted summary in place of N individual lines.
		parts := []string{fmt.Sprintf("📊  coordination burst — %d events in last %s", newEventCount, coordinationEventInterval)}
		for typ, count := range typeCounts {
			parts = append(parts, fmt.Sprintf("%s=%d", typ, count))
		}
		o.emitSystemMessage(ctx, sid, strings.Join(parts, "  "))
		logging.Info("coord_events_batched", "count", newEventCount, "types", typeCounts)
	}

	if newest != since {
		o.mu.Lock()
		o.lastCoordEventAt = newest
		o.mu.Unlock()
	}
}

// coordEventFloodThreshold is the max number of new coordination events we'll
// render as individual chat lines in a single poll tick. Above this we compact
// into a single summary to keep the Bubbletea render loop responsive.
const coordEventFloodThreshold = 8

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
	case "validation_passed", "task_validated":
		return fmt.Sprintf("✅  validation passed — %s", truncate(ev.Message, 140))
	case "validation_failed", "task_validation_failed":
		return fmt.Sprintf("❌  validation failed — %s", truncate(ev.Message, 140))
	case "synthesis_complete":
		return fmt.Sprintf("🎁  synthesized — %s", truncate(ev.Message, 140))
	case "synthesis_needs_clarification":
		return fmt.Sprintf("❓  QA needs clarification — %s", truncate(ev.Message, 140))
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

// directCommandTimeout caps subprocess wall time for palette commands.
const directCommandTimeout = 5 * time.Second

// directCommandOutputCap is the max byte length of CLI output we render
// inline. Anything beyond is truncated with a tail note.
const directCommandOutputCap = 4096

// RunDirectCommand executes a read-only act CLI subcommand and emits the
// output as a single System message in the active session. Bypasses the
// Planner — used by palette commands (act-agent:status, act-agent:log, etc.) for HITL
// inspection of the deterministic state layer.
func (o *Orchestrator) RunDirectCommand(parentCtx context.Context, sessionID, label string, argv []string) {
	if sessionID == "" || len(argv) == 0 {
		return
	}

	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin = os.Args[0]
	}

	ctx, cancel := context.WithTimeout(parentCtx, directCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, argv...)
	cmd.Env = os.Environ()
	out, runErr := cmd.CombinedOutput()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		o.emitSystemMessage(parentCtx, sessionID, fmt.Sprintf("⏱  /%s timed out after %s", label, directCommandTimeout))
		return
	}

	body := strings.TrimRight(string(out), "\n")
	if len(body) > directCommandOutputCap {
		extra := len(body) - directCommandOutputCap
		body = body[:directCommandOutputCap] + fmt.Sprintf("\n… (%d more bytes truncated)", extra)
	}

	var rendered string
	switch {
	case body == "" && runErr != nil:
		rendered = fmt.Sprintf("⚠  /%s failed: %v", label, runErr)
	case body == "":
		rendered = fmt.Sprintf("🛠  /%s — (no output)", label)
	default:
		rendered = fmt.Sprintf("🛠  /%s\n```\n%s\n```", label, body)
	}
	o.emitSystemMessage(parentCtx, sessionID, rendered)
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

// getAgent returns the agent service for a given role. Role names are
// normalized first so stray aliases (e.g. "qa" emitted by an older
// surface vs "qa_synthesizer" in the agent map) resolve to the same
// registered service. Audit Fix 13a (entry 6.1) — without this the
// lookup returns nil and the message goes nowhere.
func (o *Orchestrator) getAgent(role string) agent.Service {
	if o.app.Agents == nil {
		return nil
	}
	return o.app.Agents[normalizeRole(role)]
}

// normalizeRole maps legacy / shorthand role names to their canonical
// form. Currently just `qa` → `qa_synthesizer`. Unknown names pass
// through unchanged. Keep this list short — every entry is a latent
// place an upstream caller can drift from the canonical taxonomy.
func normalizeRole(role string) string {
	switch role {
	case "qa":
		return "qa_synthesizer"
	default:
		return role
	}
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

// IsRoleBusy returns true if the named role's agent is currently running.
// Used by polling loops that only need to avoid self-overlap (Assurance
// shouldn't block because Planner is busy, and vice versa).
func (o *Orchestrator) IsRoleBusy(role string) bool {
	svc := o.getAgent(role)
	if svc == nil {
		return false
	}
	return svc.IsBusy()
}

// CancelActive cancels whichever agent is currently running on the given session.
func (o *Orchestrator) CancelActive(sessionID string) {
	for _, agentSvc := range o.app.Agents {
		if agentSvc.IsSessionBusy(sessionID) {
			agentSvc.Cancel(sessionID)
			return
		}
	}
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
				// Pubsub re-fires UpdatedEvent on the same finished message
				// (streaming provider emits multiple terminal updates). Without
				// this guard the CREATE_TASK / PROJECT_BRIEF handlers race on
				// identical content and POST every directive twice.
				o.mu.Lock()
				if o.dispatchedMsgs[msg.ID] {
					o.mu.Unlock()
					continue
				}
				o.dispatchedMsgs[msg.ID] = true
				o.mu.Unlock()

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
					//
					// Variant selection (Fix 3): Assurance posts parse-able
					// verdicts that classify cleanly into pass/fail; everything
					// else (Observer anomalies, QA reports, unparseable
					// Assurance content) uses the classic anomaly tree.
					variant := variantAnomaly
					if role == "assurance" {
						if v := parseValidationVerdict("", content); v != nil {
							if v.Passed {
								variant = variantPassVerdict
							} else {
								variant = variantFailVerdict
							}
						}
					}
					go o.autoRoutePlannerV(ctx, role, content, variant)
				}
			}
		}
	}
}

// autoRouteVariant picks the prompt template autoRoutePlannerV uses. Each
// variant is shaped around what the Planner can actually do for that trigger
// — silence is the default for routine Assurance passes, system events get
// the new act_cli task retry/abandon affordances, etc. Differentiating these
// is audit Fix 3 (replaces the previous one-template-for-everything envelope
// at orchestrator.go:915, the documented #1 drift surface per commits
// 7156822, c237c0e, f522d1c).
type autoRouteVariant int

const (
	// variantAnomaly — Observer reports + default fallthrough. The classic
	// (a)/(b)/(c) tree because Observer messages span "real anomaly worth
	// dispatching new work" through "informational, nothing to do."
	variantAnomaly autoRouteVariant = iota

	// variantPassVerdict — Assurance posted a passing verdict. Silence is
	// the default; (a) and (b) are escape hatches for "the next step is
	// obvious" or "the human asked for status." No "react by taking action"
	// framing — that's exactly what creates the empty-CREATE_TASK loop.
	variantPassVerdict

	// variantFailVerdict — Assurance posted a failing verdict. Gap analysis
	// is auto-routed to the swarm agent already, so the Planner rarely needs
	// to dispatch new work. The variant nudges toward "stay silent unless
	// repeated failures suggest reassignment."
	variantFailVerdict

	// variantSystemEscalation — task_failed / task_burst / validation_stuck /
	// synthesis_stuck. Silence is wrong; the Planner must act. Binary fork:
	// use act_cli task retry/abandon for known-failed task management, or
	// emit CREATE_TASK to reassign, or message the human.
	variantSystemEscalation
)

// autoRoutePlanner is the legacy entry point — equivalent to
// autoRoutePlannerV with variantAnomaly. Kept for the agent-message handler
// at orchestrator.go:884 which still uses the Observer template by default;
// the call there picks a variant inline before invoking the V form.
func (o *Orchestrator) autoRoutePlanner(ctx context.Context, fromRole, fromContent string) {
	o.autoRoutePlannerV(ctx, fromRole, fromContent, variantAnomaly)
}

// autoRoutePlannerV forwards a non-Planner Tier 1 message into a Planner
// turn using the given prompt variant. Cascade-safe via a sliding-window
// cap: at most autoTurnCap fires within autoRouteWindow, regardless of
// trigger source. Cleared by HandleHumanInput.
func (o *Orchestrator) autoRoutePlannerV(ctx context.Context, fromRole, fromContent string, variant autoRouteVariant) {
	o.mu.Lock()
	now := time.Now()
	o.recentAutoRoutes = pruneAutoRoutes(o.recentAutoRoutes, now, autoRouteWindow)
	if len(o.recentAutoRoutes) >= autoTurnCap {
		windowSeconds := int(autoRouteWindow.Seconds())
		o.mu.Unlock()
		logging.Warn("autoroute_planner_dropped",
			"from", fromRole,
			"cap", autoTurnCap,
			"window_seconds", windowSeconds,
			"reason", "auto_route_window_cap",
		)
		return
	}
	o.recentAutoRoutes = append(o.recentAutoRoutes, now)
	fires := len(o.recentAutoRoutes)
	sid := o.sessionID
	o.mu.Unlock()
	if sid == "" {
		return
	}
	logging.Info("autoroute_planner",
		"from", fromRole,
		"variant", variant,
		"fires_in_window", fires,
		"window_seconds", int(autoRouteWindow.Seconds()),
		"content_bytes", len(fromContent),
	)
	prompt := renderAutoRoutePrompt(variant, fromRole, fromContent)
	// Wait for any in-flight Planner turn to finish before firing. Hitting
	// runAgentTurn synchronously here races with a Planner mid-tool-call
	// and returns "session is currently processing another request",
	// silently dropping the autoroute prompt.
	o.fireWhenPlannerIdle(ctx, sid, prompt, "autoroute_from_"+fromRole)
}

// renderAutoRoutePrompt builds the per-variant text the Planner sees. Split
// out for testability — callers in orchestrator_test.go can probe each
// variant's wording independently of the goroutine + session-state plumbing.
func renderAutoRoutePrompt(variant autoRouteVariant, fromRole, fromContent string) string {
	switch variant {
	case variantPassVerdict:
		return fmt.Sprintf(
			"The Assurance agent posted a PASS verdict. No action is required by default.\n\n"+
				"Options (pick AT MOST one):\n"+
				"  (a) If the verdict unblocks an obvious next step (e.g. a dependent task), emit CREATE_TASK directives for that next step. Every directive must include a non-empty title, @task + @success_criteria SPIL sections, and requiredCapabilities. NEVER emit a placeholder or acknowledgement CREATE_TASK.\n"+
				"  (b) Stay silent (empty response). This is the correct default.\n\n"+
				"Do NOT acknowledge the pass in chat. Do NOT echo the verdict back.\n"+
				"Never write the literal string 'CREATE_TASK:' in conversational prose.\n\n[%s]: %s",
			fromRole, fromContent,
		)

	case variantFailVerdict:
		return fmt.Sprintf(
			"The Assurance agent posted a FAIL verdict. Gap analysis has already been auto-routed to the swarm agent — they will re-attempt the task without your involvement.\n\n"+
				"Options (pick AT MOST one):\n"+
				"  (a) If this is a repeated failure (3+ attempts) suggesting the agent is stuck, use act_cli task abandon <id> --reason \"<short why>\" and emit a CREATE_TASK to reassign to a different role.\n"+
				"  (b) Write a short chat reply IF AND ONLY IF the human needs to be informed (e.g. major blocker).\n"+
				"  (c) Stay silent. This is the correct default for a first or second failure — the swarm agent's retry will run.\n\n"+
				"Never write the literal string 'CREATE_TASK:' in conversational prose.\n\n[%s]: %s",
			fromRole, fromContent,
		)

	case variantSystemEscalation:
		return fmt.Sprintf(
			"The orchestrator surfaced a system event that requires Planner action. Silence is WRONG here.\n\n"+
				"Pick ONE concrete action:\n"+
				"  (a) act_cli task retry <id> — re-dispatch a failed task to the same role (uses next retry attempt).\n"+
				"  (b) act_cli task abandon <id> --reason \"<short why>\" — mark the task permanently failed when unrecoverable.\n"+
				"  (c) Emit a CREATE_TASK: directive to reassign the work to a different role.\n"+
				"  (d) Write a short chat reply to inform the human if the situation is unrecoverable and needs a decision.\n\n"+
				"act_cli is the JSON tool you already use for status/log/graph — call it the same way: {\"subcommand\":\"task\",\"args\":[\"retry\",\"<id>\"]}.\n"+
				"Never write the literal string 'CREATE_TASK:' in conversational prose.\n\n[%s]: %s",
			fromRole, fromContent,
		)

	default: // variantAnomaly
		return fmt.Sprintf(
			"The %s agent just sent the following report. React by taking action.\n\n"+
				"Decide ONE of these, do not combine:\n"+
				"  (a) Emit one or more CREATE_TASK: directives IF AND ONLY IF actual new work needs dispatching or a failed task needs reassignment. Every directive must include a non-empty title, a description carrying @task and @success_criteria SPIL sections, and explicit requiredCapabilities. NEVER emit a placeholder, empty, or acknowledgement CREATE_TASK — passing the verdict along is not a task.\n"+
				"  (b) Write a short chat reply IF AND ONLY IF the human needs to be informed of something.\n"+
				"  (c) Stay silent (empty response) if neither applies.\n\n"+
				"Never write the literal string 'CREATE_TASK:' in conversational prose — it is reserved for actual directives and will be flagged as malformed output.\n"+
				"Do not echo the report back.\n\n[%s]: %s",
			fromRole, fromRole, fromContent,
		)
	}
}

// autoTurnCap is the maximum number of auto-routed Planner turns admitted
// within autoRouteWindow. Sliding-window semantics (audit Fix 6, was an
// integer back-to-back counter): three documented cascade loops
// (QA watchdog re-fire, Assurance verdict mirror, Observer 120s echo)
// bypass any reset-on-human-input integer counter because each is a fresh
// lineage. The wall-clock window catches them all.
const autoTurnCap = 5

// autoRouteWindow is the rolling time window the autoTurnCap is enforced
// against. 10 minutes lets Observer's ~120s cadence get four cycles through
// cleanly and traps a runaway by the 5th. HandleHumanInput clears the
// whole window so a human turn always gets a clean budget.
const autoRouteWindow = 10 * time.Minute

// pruneAutoRoutes drops entries older than `window` relative to `now`. Pure
// function — caller holds the orchestrator mutex. Extracted for testability:
// tests pass a synthetic `now` and a seeded slice to verify pruning without
// needing a clock-injection seam in the rest of the orchestrator.
func pruneAutoRoutes(times []time.Time, now time.Time, window time.Duration) []time.Time {
	cutoff := now.Add(-window)
	// Most-recent entries are at the tail; find the first index that
	// survives the cutoff by scanning forward (the slice is in append
	// order so this is sorted ascending by timestamp).
	keep := 0
	for keep < len(times) && times[keep].Before(cutoff) {
		keep++
	}
	if keep == 0 {
		return times
	}
	// Copy-shift so the backing array can be reused; avoids growing
	// the slice unbounded over a long session.
	n := copy(times, times[keep:])
	return times[:n]
}

// renderBriefContext composes the [SYSTEM] block injected as the first
// Planner turn on resume, and the BUILD-mode kickoff message. Both
// callers used to send terse summaries (resume: desc+tech only; BUILD:
// just the project name). With fields missing, the Planner's intake-
// heuristics fire and it asks the human to re-run intake. Audit Fix 5
// (entries 3.2 + 3.3) — inline the full brief so the mode-switch is
// unambiguous from a single message.
//
// Both sources route through BriefView so the renderer has one shape to
// reason about: GetProject's map[string]any populates BriefView in the
// resume path; a fresh ProjectBrief populates it in the BUILD path.
// `tasks` may be nil — in that case the function emits "no tasks
// dispatched yet — start decomposing now," matching the BUILD-trigger
// intent. The kind argument switches the leading instruction between
// "Resuming project — skip intake" and "Project created — start BUILD."
func renderBriefContext(kind string, b BriefView, tasks []TaskSummary) string {
	var sb strings.Builder

	switch kind {
	case "resume":
		fmt.Fprintf(&sb, "[SYSTEM] Resuming project %q. A project brief already exists on the server — do NOT run intake. Switch immediately to BUILD mode.\n\n", b.ProjectName)
	case "build":
		fmt.Fprintf(&sb, "[SYSTEM] Project %q has been created. Switch to BUILD mode now: decompose the brief below into tasks and emit task-creation directives for each one (you know the shape). Do not ask for confirmation — start creating tasks immediately.\n\n", b.ProjectName)
	default:
		fmt.Fprintf(&sb, "[SYSTEM] Project %q context:\n\n", b.ProjectName)
	}

	sb.WriteString("@brief\n")
	if b.Description != "" {
		fmt.Fprintf(&sb, "  description: %s\n", b.Description)
	}
	if b.TechStack != "" {
		fmt.Fprintf(&sb, "  techStack: %s\n", b.TechStack)
	}
	if b.Constraints != "" {
		fmt.Fprintf(&sb, "  constraints: %s\n", b.Constraints)
	}
	if b.SuccessCriteria != "" {
		fmt.Fprintf(&sb, "  successCriteria: %s\n", b.SuccessCriteria)
	}
	if len(b.AgentsInvolved) > 0 {
		fmt.Fprintf(&sb, "  agentsInvolved: %s\n", strings.Join(b.AgentsInvolved, ", "))
	}

	if len(tasks) == 0 {
		sb.WriteString("\n@tasks\n  no tasks dispatched yet — start decomposing now.\n")
		return sb.String()
	}

	// Partition into in-flight (anything not terminal) vs completed-side
	// (completed/validated). Failed tasks count as in-flight so the
	// Planner sees them and can retry/abandon — that's the whole point
	// of the new tools from Fix 2.
	var inFlight, doneLike []TaskSummary
	for _, t := range tasks {
		switch t.Status {
		case "completed", "validated":
			doneLike = append(doneLike, t)
		default:
			inFlight = append(inFlight, t)
		}
	}

	if len(inFlight) > 0 {
		sb.WriteString("\n@inFlightTasks\n")
		for _, t := range inFlight {
			fmt.Fprintf(&sb, "  - id=%s status=%s agent=%s title=%q\n", t.ID, t.Status, defaultStr(t.AssignedAgent, "unassigned"), t.Title)
		}
	}
	if len(doneLike) > 0 {
		sb.WriteString("\n@completedTasks\n")
		for _, t := range doneLike {
			fmt.Fprintf(&sb, "  - id=%s status=%s title=%q\n", t.ID, t.Status, t.Title)
		}
	}
	sb.WriteString("\nDo NOT re-emit task-creation directives for the task IDs above — they already exist on the server. Decompose only NEW work, or use act_cli task retry/abandon for failed tasks above.\n")
	return sb.String()
}

// defaultStr returns fallback when s is empty. Local helper for the
// brief renderer's "unassigned" / "-" placeholders.
func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// handleProjectBrief parses a PROJECT_BRIEF directive from a Planner intake-mode
// response and POSTs it to the ACT server. On success, clears intakeMode so the
// next Planner turn falls back to normal task-decomposition behavior.
func (o *Orchestrator) handleProjectBrief(ctx context.Context, content string) {
	markerFound := strings.Contains(content, "PROJECT_BRIEF:")
	brief := parseProjectBrief(content)
	descBytes := 0
	if brief != nil {
		descBytes = len(brief.Description)
	}
	logging.Info("project_brief_parse",
		"content_bytes", len(content),
		"marker_found", markerFound,
		"parsed", brief != nil,
		"desc_bytes", descBytes,
	)
	if markerFound && brief == nil {
		logging.Warn("project_brief_parse_failed",
			"reason", "PROJECT_BRIEF: marker present but JSON extraction or parse failed — Planner output malformed",
		)
	}
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

	// Materialize AGENTS.md + CLAUDE.md in the project working directory. The
	// server-stored brief remains the source of truth (replayed on restart);
	// these files are a derived view that (a) lets claude-code swarm agents
	// auto-discover the brief without an HTTP fetch and (b) travels with the
	// repo so teammates cloning get the project context for free. Any user
	// content after the preserve marker survives regeneration.
	if err := writeAgentsMd(dir, name, brief); err != nil {
		logging.Warn("Failed to write AGENTS.md", "project", name, "dir", dir, "error", err)
	} else {
		logging.Info("AGENTS.md + CLAUDE.md written", "project", name, "dir", dir)
		// Tier 1 system messages were baked at TUI startup before AGENTS.md
		// existed. Invalidate the contextPaths cache and rebind each Tier 1
		// agent's provider so AGENTS.md content reaches Planner, Observer,
		// Assurance, and QA in the same session that created the project.
		prompt.InvalidateContextCache()
		for _, role := range []string{"planner", "observer", "assurance", "qa_synthesizer"} {
			a := o.getAgent(role)
			if a == nil {
				continue
			}
			if err := a.RebindSystemPrompt(); err != nil {
				logging.Warn("Failed to rebind Tier 1 system prompt", "role", role, "error", err)
			}
		}
	}

	o.mu.Lock()
	o.intakeMode = false
	sid := o.sessionID
	o.mu.Unlock()
	logging.Info("PROJECT_BRIEF accepted — exiting INTAKE mode", "project", name)

	// Kick the Planner into BUILD mode. Without this trigger it has no
	// human input to respond to, so it just stays silent after intake.
	//
	// The PROJECT_BRIEF directive arrives inside a Planner turn — the
	// pubsub Updated event that triggers this handler may fire while the
	// same Planner turn is still finishing (tool-call round still in flight).
	// Running the trigger synchronously here races with that turn and
	// returns "session is currently processing another request", silently
	// dropping the build prompt. Result: ACT sits idle, no tasks created.
	//
	// Fix: defer to a goroutine that waits for the Planner to finish its
	// current turn, then fires. Polls IsRoleBusy at 200ms intervals up to
	// 60s (plenty for any reasonable turn completion).
	buildPrompt := renderBriefContext("build", BriefView{
		ProjectName:     name,
		Description:     brief.Description,
		TechStack:       brief.TechStack,
		Constraints:     brief.Constraints,
		SuccessCriteria: brief.SuccessCriteria,
		AgentsInvolved:  brief.AgentsInvolved,
	}, nil)
	go o.fireWhenPlannerIdle(ctx, sid, buildPrompt, "build_mode_trigger")
}

// fireWhenPlannerIdle waits for the Planner to finish its current turn, then
// runs a new turn with the given prompt. Used by any orchestrator path that
// fires a Planner prompt in reaction to a just-completed Planner message
// (handleProjectBrief, handlePlannerTaskDirectives retries, etc) — calling
// runAgentTurn synchronously from inside the pubsub handler races with the
// turn that produced the triggering message.
func (o *Orchestrator) fireWhenPlannerIdle(ctx context.Context, sessionID, prompt, reason string) {
	const pollInterval = 200 * time.Millisecond
	const maxWait = 60 * time.Second
	deadline := time.Now().Add(maxWait)
	for o.IsRoleBusy("planner") {
		if time.Now().After(deadline) {
			logging.Warn("planner_trigger_timeout", "reason", reason, "waited", maxWait)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
		}
	}
	logging.Info("planner_trigger_fire", "reason", reason, "prompt_bytes", len(prompt))
	o.runAgentTurn(ctx, sessionID, "planner", prompt)
}

// handlePlannerTaskDirectives parses CREATE_TASK directives from a Planner
// response and POSTs each one to the ACT server. On first successful task
// creation, also spawns the Runner subprocess if not already running.
//
// The ctx parameter is currently unused — act.Client doesn't accept a context
// yet — but is kept on the signature so the caller's loop context can be
// threaded through once the HTTP client is made context-aware.
func (o *Orchestrator) handlePlannerTaskDirectives(_ context.Context, content string) {
	tasks, markersFound, firstFailPreview, pattern2Used := parseCreateTaskDirectives(content)
	logging.Info("create_task_parse",
		"content_bytes", len(content),
		"markers_found", markersFound,
		"tasks_parsed", len(tasks),
		"pattern2_used", pattern2Used,
	)
	if markersFound > 0 && len(tasks) == 0 {
		logging.Warn("create_task_parse_failed",
			"markers_found", markersFound,
			"first_fail_preview", firstFailPreview,
			"reason", "markers present but no JSON parsed — Planner output malformed",
		)
	}
	if len(tasks) == 0 {
		return
	}

	// Defense-in-depth content-hash dedup. The dispatchedMsgs guard in
	// messageOwnershipLoop catches pubsub re-fires of the same msg.ID. This
	// catches the separate failure mode where a flaky Planner model (observed
	// with z-ai/glm-4.5-air:free) re-emits the exact same CREATE_TASK batch
	// on a fresh turn ~20s later — different msg.ID, identical content, would
	// otherwise produce duplicate task records with distinct UUIDs.
	batchHash := taskBatchHash(tasks)
	if drop, age := o.checkAndRecordDispatchHash(batchHash); drop {
		logging.Warn("task_directive_duplicate_dispatch_dropped",
			"hash_prefix", batchHash[:12],
			"content_bytes", len(content),
			"task_count", len(tasks),
			"age_ms", age.Milliseconds(),
		)
		return
	}

	client := act.NewClient("planner", os.Getenv("ACT_PROJECT"))
	if !client.IsAvailable() {
		logging.Warn("ACT server unavailable — cannot create tasks from Planner directives")
		return
	}

	// Lazy spawn: the first time tasks appear, bring up the Tier 2 swarm.
	// StartSwarm is idempotent per-role so calling it on every batch is safe.
	//
	// Filter specs by the project's agentsInvolved list when the Planner has
	// already committed one in PROJECT_BRIEF. Without this, every project
	// spawns all 5 runners regardless of whether the Planner asked for them —
	// the brief's "One developer agent" is ignored and 5 node subprocesses
	// run competing for tasks. Falls back to the full spec list if the
	// project record has no agents[] (pre-brief turn, or legacy project).
	if len(o.app.SwarmSpecs) > 0 && !o.runnerSpawner.IsRunning() {
		specs := o.filterSwarmSpecsByProject(client)
		logging.Info("First task batch — spawning Tier 2 swarm",
			"task_count", len(tasks),
			"swarm_size", len(specs),
			"requested_roles", specRoleNames(specs),
		)
		if err := o.runnerSpawner.StartSwarm(specs); err != nil {
			logging.Warn("Failed to start swarm", "error", err)
		}
	}

	// Two-pass CREATE_TASK dispatch:
	//   Pass 1 — create every task with NO dependencies; collect (title → ID).
	//   Pass 2 — for any task whose Planner-emitted deps reference sibling
	//            titles in this batch, PATCH /api/tasks/:id/dependencies to
	//            replace title strings with server-assigned IDs.
	//
	// Why: the Planner can only refer to dependencies by title — task IDs
	// don't exist until creation. Without this resolution, the server's
	// dependency-satisfaction check (which does tasks.get(depString)) never
	// matches any title-string dep, leaving downstream tasks pending forever.
	titleToID := make(map[string]string, len(tasks))
	created := 0
	type pendingDepUpdate struct {
		taskID string
		title  string
		deps   []string // original Planner-emitted dep strings (may be titles)
	}
	var depUpdates []pendingDepUpdate
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

		// Pass 1: create with empty dependencies. Pass 2 below resolves and
		// PATCHes them in once every sibling's ID is known.
		taskID, err := client.CreateTask(title, task.Description, caps, priority, nil, metadata)
		if err != nil {
			logging.Warn("Failed to create task from Planner directive", "title", title, "error", err)
			continue
		}
		logging.Info("Task created from Planner directive", "task_id", taskID, "title", title)
		created++
		if title != "" {
			titleToID[title] = taskID
		}
		if len(task.Dependencies) > 0 {
			depUpdates = append(depUpdates, pendingDepUpdate{taskID: taskID, title: title, deps: task.Dependencies})
		}
	}
	// Pass 2: resolve title-string deps to IDs. A dep string that looks like
	// a known task ID (server-issued UUID, no match in titleToID) is passed
	// through as-is — supports Planner emissions that already use IDs.
	for _, upd := range depUpdates {
		resolved := make([]string, 0, len(upd.deps))
		var unresolved []string
		for _, dep := range upd.deps {
			if id, ok := titleToID[dep]; ok {
				resolved = append(resolved, id)
				continue
			}
			// Not a sibling title — assume it's already an ID (UUID-shaped
			// strings will pass the server's existence check; non-existent
			// strings will be rejected with a 400 by the PATCH endpoint).
			resolved = append(resolved, dep)
			unresolved = append(unresolved, dep)
		}
		if err := client.SetTaskDependencies(upd.taskID, resolved); err != nil {
			logging.Warn("Failed to set task dependencies after creation",
				"task_id", upd.taskID, "title", upd.title, "deps", upd.deps, "error", err)
			continue
		}
		logging.Info("Task dependencies resolved",
			"task_id", upd.taskID, "title", upd.title, "deps", resolved, "unresolved_titles", unresolved)
	}
	// Trigger a Nomik incremental rescan after task creation so Planner sees fresh
	// graph state on next iteration.
	if created > 0 {
		go o.maybeRescanNomik(context.Background())
	}
}

// taskBatchHash computes a stable sha256 over the parsed CREATE_TASK batch.
// The batch is sorted by spec string before hashing so two emissions that
// list the same tasks in a different order still collide. TaskDef has no
// explicit "role" field — capabilities (RequiredCapabilities, falling back
// to Capabilities) carry the role intent in ACT's routing, so they stand
// in for the title|role|description spec triple.
func taskBatchHash(tasks []TaskDef) string {
	specs := make([]string, len(tasks))
	for i, t := range tasks {
		title := t.Title
		if title == "" {
			title = t.Name
		}
		caps := t.RequiredCapabilities
		if len(caps) == 0 {
			caps = t.Capabilities
		}
		capsCopy := append([]string(nil), caps...)
		sort.Strings(capsCopy)
		specs[i] = strings.Join([]string{title, t.Description, strings.Join(capsCopy, ",")}, "|")
	}
	sort.Strings(specs)
	sum := sha256.Sum256([]byte(strings.Join(specs, "\n")))
	return hex.EncodeToString(sum[:])
}

// checkAndRecordDispatchHash records the given hash as dispatched-now and
// returns drop=false. If the hash is already present within dispatchHashWindow,
// returns drop=true plus the age of the prior dispatch and leaves the map
// untouched (so a third re-emit at t+50s still drops against the original t+0
// timestamp). Lazily GCs expired entries on every call — no goroutine needed.
func (o *Orchestrator) checkAndRecordDispatchHash(hash string) (drop bool, age time.Duration) {
	o.dispatchHashMu.Lock()
	defer o.dispatchHashMu.Unlock()
	now := time.Now()
	for h, ts := range o.recentDispatchHashes {
		if now.Sub(ts) > dispatchHashWindow {
			delete(o.recentDispatchHashes, h)
		}
	}
	if ts, ok := o.recentDispatchHashes[hash]; ok {
		return true, now.Sub(ts)
	}
	o.recentDispatchHashes[hash] = now
	return false, 0
}

// filterSwarmSpecsByProject returns the subset of SwarmSpecs whose role
// appears in the current project's agentsInvolved list. If the project record
// has no agents[] (missing, empty, or server unreachable), returns the full
// spec list — safer default than "no runners".
//
// The Planner's PROJECT_BRIEF includes agentsInvolved; the server stores it
// as project.agents. This is the only filter that respects the brief's
// "One developer agent" directive.
func (o *Orchestrator) filterSwarmSpecsByProject(client *act.Client) []runner.SwarmRoleSpec {
	all := o.app.SwarmSpecs
	if len(all) == 0 {
		return all
	}
	name := o.projectName
	if name == "" {
		return all
	}
	data, found, err := client.GetProject(name)
	if err != nil || !found || data == nil {
		return all
	}
	rawAgents, ok := data["agents"].([]any)
	if !ok || len(rawAgents) == 0 {
		return all
	}
	wanted := make(map[string]struct{}, len(rawAgents))
	for _, a := range rawAgents {
		if s, ok := a.(string); ok && s != "" {
			wanted[s] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return all
	}
	filtered := make([]runner.SwarmRoleSpec, 0, len(all))
	for _, spec := range all {
		if _, ok := wanted[spec.Role]; ok {
			filtered = append(filtered, spec)
		}
	}
	if len(filtered) == 0 {
		// Brief asked for roles none of which are configured — fall back
		// rather than leave the project with no runners at all.
		return all
	}
	return filtered
}

func specRoleNames(specs []runner.SwarmRoleSpec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Role)
	}
	return out
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
	// tier1StuckThreshold — if a Tier 1 role (Assurance, QA) is idle while its
	// queue is non-empty and its last turn was longer ago than this, Observer
	// re-triggers it directly. α-2 watchdog. Decisions still route to Planner.
	tier1StuckThreshold = 5 * time.Minute
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
	// Periodic TUI render stats — emitted every Observer tick regardless of
	// whether anomalies exist. Cheap (atomic loads) and gives a rolling view
	// of whether the render pipeline is healthy. Slow ratio > 5% under load
	// suggests the message-render path is the bottleneck.
	total, slow, msgs := RenderStatsSnapshot()
	slowRatio := 0.0
	if total > 0 {
		slowRatio = float64(slow) / float64(total) * 100
	}
	logging.Info("render_stats",
		"total", total,
		"slow", slow,
		"slow_pct", fmt.Sprintf("%.1f", slowRatio),
		"last_msgs", msgs,
	)

	if o.IsAnyBusy("") {
		logging.Info("observer_check_skip", "reason", "any_agent_busy")
		return // don't interrupt active turns
	}
	logging.Info("observer_check_start")

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

	// α-2 Tier 1 watchdog: if Assurance or QA is idle while their queue is
	// non-empty and last turn was >tier1StuckThreshold ago, re-trigger them
	// directly. Skip the Observer LLM this cycle if we acted — kicking the
	// stuck role is the action; the Observer's narrative would be redundant.
	if o.tier1Watchdog(ctx, sid, snapshot) {
		return
	}

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

// tier1Watchdog re-triggers Assurance or QA when their queue is non-empty,
// they're idle, and their last turn was longer ago than tier1StuckThreshold.
// Returns true if it acted (caller should skip the Observer LLM this cycle).
// Mechanical re-triggers only — judgment cases (rejection patterns, missing
// inputs) still surface through the regular Observer anomaly path to Planner.
func (o *Orchestrator) tier1Watchdog(ctx context.Context, sid string, snap *StatusSnapshot) bool {
	if sid == "" || snap == nil {
		return false
	}

	pendingValidation := 0
	pendingSynthesis := 0
	for _, t := range snap.Tasks {
		switch t.Status {
		case "submitted_for_validation":
			pendingValidation++
		case "validated":
			pendingSynthesis++
		}
	}

	now := time.Now()

	if pendingValidation > 0 && !o.IsRoleBusy("assurance") {
		o.mu.RLock()
		last, seen := o.lastTurnAt["assurance"]
		o.mu.RUnlock()
		if !seen || now.Sub(last) > tier1StuckThreshold {
			logging.Warn("tier1_watchdog.fire",
				"role", "assurance",
				"pending", pendingValidation,
				"last_turn_ago", now.Sub(last).String(),
			)
			o.emitSystemMessage(ctx, sid, fmt.Sprintf("👁  Observer: Assurance idle with %d task(s) in validation queue — re-triggering", pendingValidation))
			go o.checkPendingValidation(ctx)
			return true
		}
	}

	if pendingSynthesis > 0 && !o.IsRoleBusy("qa_synthesizer") {
		o.mu.RLock()
		last, seen := o.lastTurnAt["qa_synthesizer"]
		o.mu.RUnlock()
		if !seen || now.Sub(last) > tier1StuckThreshold {
			logging.Warn("tier1_watchdog.fire",
				"role", "qa_synthesizer",
				"pending", pendingSynthesis,
				"last_turn_ago", now.Sub(last).String(),
			)
			o.emitSystemMessage(ctx, sid, fmt.Sprintf("👁  Observer: QA idle with %d validated task(s) awaiting synthesis — re-triggering", pendingSynthesis))
			prompt := fmt.Sprintf("Synthesis queue check — %d validated task(s) awaiting QA synthesis. Process the oldest validated task.", pendingSynthesis)
			go o.runAgentTurn(ctx, sid, "qa_synthesizer", prompt)
			return true
		}
	}

	return false
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

	// 6. Failed tasks — invisible to rules 1-5 because those only cover
	// assigned/in_progress/pending/completed states. Without this rule the
	// Observer silently ignores the most actionable state.
	for _, t := range s.Tasks {
		if t.Status != "failed" {
			continue
		}
		sev := SeverityCritical
		if t.RetryCount >= maxTaskRetries {
			sev = SeverityWarning // permanently failed — human/Planner decision needed, not a retry
		}
		label := t.Title
		if label == "" {
			label = t.ID
		}
		anomalies = append(anomalies, Anomaly{
			Severity: sev,
			Category: CategoryFailedTask,
			Message:  fmt.Sprintf("Task %q failed (retry %d/%d, agent %s)", label, t.RetryCount, maxTaskRetries, orNobody(t.AssignedAgent)),
			TaskID:   t.ID,
			AgentID:  t.AssignedAgent,
		})
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
	if o.IsRoleBusy("assurance") {
		logging.Info("assurance_poll_skip", "reason", "role_busy")
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
		seenKey := "validation:" + t.ID
		if o.alreadySeen(seenKey) {
			continue
		}
		attempts := o.incAttempt(seenKey)
		if attempts > maxValidationAttempts {
			o.markSeen(seenKey)
			o.mu.RLock()
			sid := o.sessionID
			o.mu.RUnlock()
			if sid != "" {
				o.emitSystemMessage(ctx, sid, fmt.Sprintf("⚠  validation stuck on %q after %d attempts — escalating to Planner", taskLabel(t), maxValidationAttempts))
				go o.autoRoutePlannerV(ctx, "system", fmt.Sprintf("validation stuck on task %q (id=%s) after %d attempts; the swarm agent has retried the gap analysis without success", taskLabel(t), t.ID, maxValidationAttempts), variantSystemEscalation)
			}
			continue
		}
		logging.Info("assurance_poll_start", "task_id", t.ID, "attempt", attempts)
		o.routeToAssurance(ctx, client, t)
		break // one at a time
	}
}

func (o *Orchestrator) routeToAssurance(ctx context.Context, client *act.Client, t TaskSummary) {
	o.mu.RLock()
	sid := o.sessionID
	dir := o.projectDir
	o.mu.RUnlock()
	prompt := buildValidationPrompt(t, dir)
	if sid == "" {
		return
	}

	// Run the Assurance turn. The task stays unseen at this point so if anything
	// below fails (unparseable verdict, SubmitVerdict HTTP error), the next
	// polling tick will retry, up to maxValidationAttempts.
	o.runAgentTurn(ctx, sid, "assurance", prompt)

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
		preview := lastAssurance
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		logging.Warn("verdict_parse_fail", "task_id", t.ID, "preview", preview)
		return
	}

	criteriaResults := make([]map[string]any, 0, len(verdict.CriteriaResults))
	for _, cr := range verdict.CriteriaResults {
		criteriaResults = append(criteriaResults, map[string]any{
			"criterion": cr.Criterion,
			"passed":    cr.Passed,
			"score":     cr.Score,
			"reasoning": cr.Reasoning,
		})
	}
	if err := client.SubmitVerdict(t.ID, "assurance", verdict.Passed, verdict.OverallScore, criteriaResults, verdict.Gaps, verdict.Feedback); err != nil {
		logging.Warn("submit_verdict_failed", "task_id", t.ID, "error", err)
		return
	}
	// Only mark seen once the verdict has been accepted by the server. A failed
	// verdict that routed the task back to 'assigned' will surface again in a
	// future poll under a new key lifecycle — the seen-key is per submission.
	o.markSeen("validation:" + t.ID)
	logging.Info("verdict_submitted", "task_id", t.ID, "passed", verdict.Passed, "score", verdict.OverallScore)
}

// ─── Background loop: QA polling ───────────────────────────────────────────────

func (o *Orchestrator) qaPollLoop(ctx context.Context) {
	defer o.loopWG.Done()

	if o.getAgent("qa_synthesizer") == nil {
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
	if o.IsRoleBusy("qa_synthesizer") {
		logging.Info("qa_poll_skip", "reason", "role_busy")
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
		seenKey := "qa:" + t.ID
		if o.alreadySeen(seenKey) {
			continue
		}
		attempts := o.incAttempt(seenKey)
		if attempts > maxSynthesisAttempts {
			o.markSeen(seenKey)
			o.mu.RLock()
			sid := o.sessionID
			o.mu.RUnlock()
			if sid != "" {
				o.emitSystemMessage(ctx, sid, fmt.Sprintf("⚠  synthesis stuck on %q after %d attempts — escalating to Planner", taskLabel(t), maxSynthesisAttempts))
				go o.autoRoutePlannerV(ctx, "system", fmt.Sprintf("synthesis stuck on task %q (id=%s) after %d attempts; QA cannot assemble the deliverable", taskLabel(t), t.ID, maxSynthesisAttempts), variantSystemEscalation)
			}
			continue
		}
		logging.Info("qa_poll_start", "task_id", t.ID, "attempt", attempts)
		o.routeToQA(ctx, t)
		break
	}
}

func (o *Orchestrator) routeToQA(ctx context.Context, t TaskSummary) {
	o.mu.RLock()
	sid := o.sessionID
	projectName := o.projectName
	o.mu.RUnlock()
	if sid == "" {
		return
	}
	client := act.NewClient("qa_synthesizer", projectName)

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
	o.runAgentTurn(ctx, sid, "qa_synthesizer", prompt)

	// Verify the QA agent actually produced a synthesis reply before marking
	// seen. If the LLM bailed (no reply, tool-only output, etc.) leave the task
	// eligible for retry on the next poll.
	msgs, err := o.app.Messages.List(ctx, sid)
	if err != nil {
		return
	}
	var lastQA string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == message.Assistant && o.GetOwner(msgs[i].ID) == "qa_synthesizer" {
			lastQA = msgs[i].Content().String()
			break
		}
	}
	if lastQA == "" {
		return
	}
	kind, summary, targetAgent, question := parseSynthesisResponse(lastQA)
	if kind == "in_progress" {
		// No marker emitted — leave the task unseen so the poller retries
		// once more. Don't write a ChronLog event for indeterminate output.
		logging.Warn("synthesis_no_marker", "task_id", t.ID, "reply_bytes", len(lastQA))
		return
	}
	if err := client.SubmitSynthesis(t.ID, "qa_synthesizer", kind, summary, targetAgent, question); err != nil {
		logging.Warn("synthesis_submit_failed", "task_id", t.ID, "kind", kind, "error", err)
		return
	}
	o.markSeen("qa:" + t.ID)
	logging.Info("synthesis_emitted", "task_id", t.ID, "kind", kind, "reply_bytes", len(lastQA))
}

// ─── Parsers ───────────────────────────────────────────────────────────────────

var (
	// createTaskMarkerRegex finds each `CREATE_TASK:` marker. The JSON object
	// after the marker is extracted with balanced-brace counting (see
	// parseCreateTaskDirectives), NOT a regex — `[^}]+` would truncate at
	// the first `}` inside the description (Python dict literals, function
	// bodies, code snippets all break it). PROJECT_BRIEF and ValidationVerdict
	// already use brace counting via extractJSONContaining; CREATE_TASK now
	// uses the same approach.
	createTaskMarkerRegex = regexp.MustCompile(`CREATE_TASK:\s*`)
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

// extractBalancedJSONFrom walks forward from text[start] (which must point at
// a '{') and returns the substring covering the balanced JSON object,
// honouring string-quoting so braces inside JSON strings don't count. Returns
// "" if the input is unbalanced (e.g. truncated mid-object). Mirrors the
// forward-walk half of extractJSONContaining — kept separate so the parsers
// that already know where the `{` starts don't pay for the backward search.
func extractBalancedJSONFrom(text string, start int) string {
	if start < 0 || start >= len(text) || text[start] != '{' {
		return ""
	}
	depth := 0
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
//
// Returns the parsed tasks, the count of CREATE_TASK: markers found (successful
// or not), and a short preview of the first marker that failed to parse. The
// counts let callers distinguish "Planner didn't try" from "Planner tried and
// produced malformed JSON" — a critical difference when debugging LLM drift.
func parseCreateTaskDirectives(content string) (tasks []TaskDef, markersFound int, firstFailPreview string, pattern2Used bool) {
	// Locate every `CREATE_TASK:` marker, then read the next balanced
	// `{...}` block with string-aware brace counting. Naive `[^}]+` regex
	// breaks on any `}` in the description (Python dict literals, function
	// bodies, code examples) — see the comment on createTaskMarkerRegex.
	for _, locs := range createTaskMarkerRegex.FindAllStringIndex(content, -1) {
		// locs[1] is the index immediately after the marker + whitespace.
		// The JSON object starts at the first `{` we find from there.
		braceIdx := strings.IndexByte(content[locs[1]:], '{')
		if braceIdx < 0 {
			// Marker present but no JSON follows — count it as a malformed
			// marker so the caller can distinguish "Planner didn't try" from
			// "Planner emitted half of one".
			markersFound++
			if firstFailPreview == "" {
				firstFailPreview = "(no `{` after CREATE_TASK: marker)"
			}
			continue
		}
		start := locs[1] + braceIdx
		jsonStr := extractBalancedJSONFrom(content, start)
		if jsonStr == "" {
			markersFound++
			if firstFailPreview == "" {
				preview := content[start:]
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				firstFailPreview = preview
			}
			continue
		}
		markersFound++
		var t TaskDef
		if err := json.Unmarshal([]byte(jsonStr), &t); err == nil {
			tasks = append(tasks, t)
		} else if firstFailPreview == "" {
			preview := jsonStr
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			firstFailPreview = preview
		}
	}

	if len(tasks) > 0 {
		return tasks, markersFound, firstFailPreview, false
	}

	if jsonMatch := extractJSONContaining(content, `"tasks"`); jsonMatch != "" {
		var wrapper struct {
			Tasks []TaskDef `json:"tasks"`
		}
		if err := json.Unmarshal([]byte(jsonMatch), &wrapper); err == nil {
			tasks = append(tasks, wrapper.Tasks...)
			pattern2Used = true
		}
	}

	return tasks, markersFound, firstFailPreview, pattern2Used
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
		OverallScore          int               `json:"score"`
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

func buildValidationPrompt(t TaskSummary, projectDir string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("A swarm agent has submitted task %q for validation. Independently verify the work against the success criteria and return your verdict.\n\n", taskLabel(t)))
	sb.WriteString(fmt.Sprintf("Project working directory: %s\n\n", projectDir))
	sb.WriteString("Task description:\n")
	sb.WriteString(t.Description)
	sb.WriteString("\n\nSuccess criteria you must score:\n")
	for i, c := range t.SuccessCriteria {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, c))
	}
	sb.WriteString("\nAgent's completion report (summary + file paths touched — NOT evidence; use it to locate what to verify, then verify with your tools):\n")
	if t.Metadata != nil {
		if r, ok := t.Metadata["result"].(string); ok {
			if len(r) > 4000 {
				r = r[:4000] + "..."
			}
			sb.WriteString(r)
		}
	}
	sb.WriteString("\n\n## Verification Protocol\n")
	sb.WriteString("You have `view` and `grep` tools. Use them. Score only what you independently confirm against the actual files in the project working directory.\n")
	sb.WriteString("For EACH success criterion:\n")
	sb.WriteString("  1. Identify what file/content/behavior the criterion demands.\n")
	sb.WriteString("  2. Use view/grep against the project working directory to confirm or refute it.\n")
	sb.WriteString("  3. Record the exact tool invocation that decided the outcome in the `reasoning` field.\n")
	sb.WriteString("A criterion without a concrete tool-verified reasoning line MUST score as failed.\n")
	sb.WriteString("\nRespond with your verdict as a JSON object with this exact shape (no surrounding prose, no code fences):\n")
	sb.WriteString(`{"passed": true|false, "score": 0-100, "criteriaResults": [{"criterion":"...","passed":true|false,"reasoning":"ran `+"`"+`view path/to/file`+"`"+` → saw <X>"}], "gaps":"...","feedback":"..."}` + "\n")
	sb.WriteString("Pass = score >= 95. Do NOT echo this prompt.")
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

// incAttempt increments and returns the attempt counter for a validation/qa
// seen-key. Used by the pollers to cap how many times we retry a task whose
// verdict/synthesis keeps failing to parse — prevents infinite polling on
// broken LLM output.
func (o *Orchestrator) incAttempt(key string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.attemptCount[key]++
	return o.attemptCount[key]
}

const (
	maxValidationAttempts = 3
	maxSynthesisAttempts  = 3
	// maxTaskRetries mirrors the server's MAX_TASK_RETRIES. Kept in sync
	// manually — if the server value changes, update both sides.
	maxTaskRetries = 3
)
