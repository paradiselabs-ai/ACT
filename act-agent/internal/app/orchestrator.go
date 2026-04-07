package app

import (
	"context"
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

	ctx, cancel := context.WithCancel(parentCtx)
	o.loopCancel = cancel
	o.mu.Unlock()

	// Spawn the Tier 2 swarm — one Runner subprocess per role.
	// Eager (not lazy) so agents are registered before the Planner starts
	// producing tasks. Failures are logged but non-fatal.
	if len(o.app.SwarmSpecs) > 0 {
		if err := o.runnerSpawner.StartSwarm(o.app.SwarmSpecs); err != nil {
			logging.Warn("Failed to start swarm", "error", err)
		}
	}

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

	o.loopWG.Add(4)
	go o.messageOwnershipLoop(ctx)
	go o.observerLoop(ctx)
	go o.validationPollLoop(ctx)
	go o.qaPollLoop(ctx)

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
	o.runAgentTurn(ctx, sessionID, "planner", text, attachments...)
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

	o.mu.Lock()
	o.currentSpeaker = role
	o.mu.Unlock()

	done, err := agentSvc.Run(ctx, sessionID, content, attachments...)
	if err != nil {
		logging.Error("Agent turn failed to start", "role", role, "error", err)
		o.mu.Lock()
		o.currentSpeaker = ""
		o.mu.Unlock()
		return
	}

	// Wait for completion
	result := <-done

	o.mu.Lock()
	o.currentSpeaker = ""
	o.mu.Unlock()

	if result.Error != nil {
		logging.Warn("Agent turn completed with error", "role", role, "error", result.Error)
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
				if role != "planner" {
					continue
				}
				content := msg.Content().String()
				if content == "" {
					continue
				}
				go o.handlePlannerTaskDirectives(ctx, content)
			}
		}
	}
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
	// Note: the swarm is already running (spawned at orchestrator Start, not lazily).
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
	observerInterval     = 120 * time.Second
	stuckTaskMinutes     = 30
	staleLockMinutes     = 20
	bottleneckTaskCount  = 3
	validationPollPeriod = 10 * time.Second
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

	anomalies := detectAnomalies(snapshot)
	if len(anomalies) == 0 {
		return // silent watchdog
	}

	prompt := buildObserverPrompt(snapshot, anomalies)
	o.mu.RLock()
	sid := o.sessionID
	o.mu.RUnlock()
	if sid == "" {
		return
	}
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
	sb.WriteString("OBSERVER MONITORING REPORT\n\n")
	sb.WriteString(fmt.Sprintf("Snapshot: %s\n", s.Timestamp))
	sb.WriteString(fmt.Sprintf("Tasks: %d, Agents: %d, FileLocks: %d\n\n", len(s.Tasks), len(s.Agents), len(s.FileLocks)))
	sb.WriteString("## Anomalies Detected\n")
	for _, a := range anomalies {
		sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", strings.ToUpper(string(a.Severity)), a.Category, a.Message))
	}
	sb.WriteString("\nReport these to the Planner with suggested actions. Do not make decisions yourself.")
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
	sb.WriteString(fmt.Sprintf("VALIDATION REQUEST — Task: %s\n\n", taskLabel(t)))
	sb.WriteString("## Task Description\n")
	sb.WriteString(t.Description)
	sb.WriteString("\n\n## Success Criteria\n")
	for i, c := range t.SuccessCriteria {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, c))
	}
	sb.WriteString("\n## Agent's Output (Result)\n")
	if t.Metadata != nil {
		if r, ok := t.Metadata["result"].(string); ok {
			if len(r) > 4000 {
				r = r[:4000] + "..."
			}
			sb.WriteString(r)
		}
	}
	sb.WriteString("\n\nValidate this task now. Respond with the JSON verdict format.")
	return sb.String()
}

func buildSynthesisPrompt(o ValidatedOutput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SYNTHESIS REQUEST — Task: %s\n\n", o.TaskTitle))
	sb.WriteString(fmt.Sprintf("Agent: %s\nValidation score: %d/100\n\n", o.AgentID, o.ValidationScore))
	sb.WriteString("## Validated Output\n")
	if len(o.Result) > 4000 {
		sb.WriteString(o.Result[:4000] + "...")
	} else {
		sb.WriteString(o.Result)
	}
	sb.WriteString("\n\nIntegrate this output into the project deliverable. Use SYNTHESIS_COMPLETE: or NEED_CLARIFICATION: as appropriate.")
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
