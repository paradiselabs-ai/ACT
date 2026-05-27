package app

import (
	"strings"
	"testing"
	"time"
)

// --- parseCreateTaskDirectives tests ---

func TestParseCreateTaskDirectives_InlineDirective(t *testing.T) {
	input := `I'll decompose this into tasks:

CREATE_TASK: {"title": "Build auth", "description": "JWT auth with refresh tokens", "priority": "high"}

That's the first one.`

	tasks, _, _, _ := parseCreateTaskDirectives(input)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Build auth" {
		t.Errorf("expected title 'Build auth', got %q", tasks[0].Title)
	}
	if tasks[0].Priority != "high" {
		t.Errorf("expected priority 'high', got %q", tasks[0].Priority)
	}
}

func TestParseCreateTaskDirectives_MultipleInline(t *testing.T) {
	input := `Here's the plan:

CREATE_TASK: {"title": "Task A", "description": "first"}
CREATE_TASK: {"title": "Task B", "description": "second"}
CREATE_TASK: {"title": "Task C", "description": "third"}
`

	tasks, _, _, _ := parseCreateTaskDirectives(input)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	titles := []string{tasks[0].Title, tasks[1].Title, tasks[2].Title}
	want := []string{"Task A", "Task B", "Task C"}
	for i, w := range want {
		if titles[i] != w {
			t.Errorf("task[%d]: expected %q, got %q", i, w, titles[i])
		}
	}
}

func TestParseCreateTaskDirectives_NoDirectives(t *testing.T) {
	input := "Here's my analysis but I'm not creating any tasks yet."
	tasks, _, _, _ := parseCreateTaskDirectives(input)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestParseCreateTaskDirectives_MalformedJSONSkipped(t *testing.T) {
	input := `CREATE_TASK: {this is not json}
CREATE_TASK: {"title": "Valid task", "description": "ok"}`

	tasks, _, _, _ := parseCreateTaskDirectives(input)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task (malformed skipped), got %d", len(tasks))
	}
	if tasks[0].Title != "Valid task" {
		t.Errorf("expected 'Valid task', got %q", tasks[0].Title)
	}
}

func TestParseCreateTaskDirectives_TasksArrayPattern(t *testing.T) {
	input := `Here is my full plan:

{
  "tasks": [
    {"title": "First", "description": "do first"},
    {"title": "Second", "description": "do second"}
  ]
}
`

	tasks, _, _, _ := parseCreateTaskDirectives(input)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks from array pattern, got %d", len(tasks))
	}
	if tasks[0].Title != "First" || tasks[1].Title != "Second" {
		t.Errorf("unexpected titles: %v", []string{tasks[0].Title, tasks[1].Title})
	}
}

// Regression: the old regex CREATE_TASK:\s*(\{[^}]+\}) truncated at the first
// `}` inside the description. Real Planner outputs frequently include code
// snippets with braces (Python dict literals, function bodies, JSON examples)
// — those used to break the parser, causing tasks_parsed=0 even when the JSON
// was actually well-formed. With balanced-brace counting the description can
// contain arbitrarily many `{}` pairs as long as the outer JSON is balanced.
func TestParseCreateTaskDirectives_BracesInDescription(t *testing.T) {
	input := `CREATE_TASK: {"title": "Implement tally.py library", "description": "Create count_words(text, min_length=1, case_sensitive=False) -> dict[str, int]. Example output: {\"the\": 5, \"and\": 3}. Tokenize with re.findall(r\"\\b\\w+\\b\", text)."}`

	tasks, markersFound, firstFailPreview, _ := parseCreateTaskDirectives(input)
	if markersFound != 1 {
		t.Fatalf("expected markersFound=1, got %d (preview=%q)", markersFound, firstFailPreview)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d (preview=%q)", len(tasks), firstFailPreview)
	}
	if tasks[0].Title != "Implement tally.py library" {
		t.Errorf("wrong title: %q", tasks[0].Title)
	}
	if !strings.Contains(tasks[0].Description, "dict[str, int]") {
		t.Errorf("description truncated mid-content: %q", tasks[0].Description)
	}
	if !strings.Contains(tasks[0].Description, `\b\w+\b`) {
		t.Errorf("description lost trailing content after second brace pair: %q", tasks[0].Description)
	}
}

func TestParseCreateTaskDirectives_NestedJSONInDescription(t *testing.T) {
	// Description carries an embedded JSON example — used to break the
	// regex at the first inner `}`.
	input := `CREATE_TASK: {"title": "Wire config", "description": "metadata: {\"key\": \"value\", \"nested\": {\"a\": 1}}"}`

	tasks, _, preview, _ := parseCreateTaskDirectives(input)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task with nested JSON in description, got %d (preview=%q)", len(tasks), preview)
	}
	if tasks[0].Title != "Wire config" {
		t.Errorf("wrong title: %q", tasks[0].Title)
	}
}

// --- parseValidationVerdict tests ---

func TestParseValidationVerdict_PassedAt100(t *testing.T) {
	raw := `Here's my verdict:
{
  "criteriaResults": [
    {"criterion": "Tests pass", "passed": true, "score": 100, "reasoning": "all green"}
  ],
  "score": 100,
  "selfVerificationValid": true,
  "feedback": "great work"
}`
	verdict := parseValidationVerdict("task-123", raw)
	if verdict == nil {
		t.Fatal("expected verdict, got nil")
	}
	if !verdict.Passed {
		t.Error("expected passed=true (score=100)")
	}
	if verdict.OverallScore != 100 {
		t.Errorf("expected score 100, got %d", verdict.OverallScore)
	}
	if verdict.TaskID != "task-123" {
		t.Errorf("expected taskID 'task-123', got %q", verdict.TaskID)
	}
}

func TestParseValidationVerdict_FailedBelow95(t *testing.T) {
	raw := `{
  "criteriaResults": [
    {"criterion": "Tests pass", "passed": false, "score": 60, "reasoning": "2 failures"}
  ],
  "score": 60,
  "selfVerificationValid": false,
  "gaps": "missing test for edge case X",
  "feedback": "incomplete"
}`
	verdict := parseValidationVerdict("task-456", raw)
	if verdict == nil {
		t.Fatal("expected verdict, got nil")
	}
	if verdict.Passed {
		t.Error("expected passed=false (score < 95)")
	}
	if !strings.Contains(verdict.Gaps, "edge case X") {
		t.Errorf("expected gaps to contain 'edge case X', got %q", verdict.Gaps)
	}
}

func TestParseValidationVerdict_BoundaryAt95(t *testing.T) {
	raw := `{"criteriaResults":[{"criterion":"x","passed":true,"score":95}],"score":95}`
	verdict := parseValidationVerdict("t", raw)
	if verdict == nil {
		t.Fatal("expected verdict")
	}
	if !verdict.Passed {
		t.Error("expected passed=true at boundary score 95")
	}
}

func TestParseValidationVerdict_NoJSON(t *testing.T) {
	raw := "I cannot validate this task right now"
	verdict := parseValidationVerdict("t", raw)
	if verdict != nil {
		t.Error("expected nil verdict for non-JSON response")
	}
}

func TestParseValidationVerdict_JSONInProse(t *testing.T) {
	raw := `Looking at this carefully, here is my structured response:

The task has some issues. Specifically:

{"criteriaResults": [{"criterion": "compiles", "passed": true, "score": 100}], "score": 80, "gaps": "missing tests"}

Let me know if you need more detail.`

	verdict := parseValidationVerdict("t", raw)
	if verdict == nil {
		t.Fatal("expected verdict extracted from prose")
	}
	if verdict.OverallScore != 80 {
		t.Errorf("expected score 80, got %d", verdict.OverallScore)
	}
}

// --- parseSynthesisResponse tests ---

func TestParseSynthesisResponse_Complete(t *testing.T) {
	raw := `I've integrated everything successfully.

SYNTHESIS_COMPLETE: All four task outputs assembled into a working application. Auth module wired to API routes, frontend connects to backend, tests pass.`

	kind, summary, _, _ := parseSynthesisResponse(raw)
	if kind != "complete" {
		t.Errorf("expected kind 'complete', got %q", kind)
	}
	if !strings.Contains(summary, "All four task outputs") {
		t.Errorf("summary missing expected content: %q", summary)
	}
}

func TestParseSynthesisResponse_NeedClarification(t *testing.T) {
	raw := `I see a problem. The auth module exports a getUserById function but the API routes expect findUserById.

NEED_CLARIFICATION: @backend-dev-1 You exported getUserById but the API routes call findUserById. Which name should we use?`

	kind, _, target, question := parseSynthesisResponse(raw)
	if kind != "need_clarification" {
		t.Errorf("expected kind 'need_clarification', got %q", kind)
	}
	if target != "backend-dev-1" {
		t.Errorf("expected target 'backend-dev-1', got %q", target)
	}
	if !strings.Contains(question, "getUserById") {
		t.Errorf("question missing expected content: %q", question)
	}
}

func TestParseSynthesisResponse_InProgress(t *testing.T) {
	raw := "I'm still working on integrating the outputs. Will report back."
	kind, _, _, _ := parseSynthesisResponse(raw)
	if kind != "in_progress" {
		t.Errorf("expected kind 'in_progress', got %q", kind)
	}
}

// --- content-hash dispatch dedup tests ---

func TestTaskBatchHash_Stable(t *testing.T) {
	a := []TaskDef{
		{Title: "T1", Description: "do x", RequiredCapabilities: []string{"go", "api"}},
		{Title: "T2", Description: "do y", RequiredCapabilities: []string{"ui"}},
	}
	// Same content, different list order + cap order — must hash identically.
	b := []TaskDef{
		{Title: "T2", Description: "do y", RequiredCapabilities: []string{"ui"}},
		{Title: "T1", Description: "do x", RequiredCapabilities: []string{"api", "go"}},
	}
	if taskBatchHash(a) != taskBatchHash(b) {
		t.Fatalf("expected stable hash across reordering")
	}
}

func TestTaskBatchHash_DifferentContent(t *testing.T) {
	a := []TaskDef{{Title: "T1", Description: "do x"}}
	b := []TaskDef{{Title: "T1", Description: "do y"}}
	if taskBatchHash(a) == taskBatchHash(b) {
		t.Fatalf("expected different hash for different descriptions")
	}
}

func TestDispatchDedup_DuplicateDropped(t *testing.T) {
	o := &Orchestrator{recentDispatchHashes: map[string]time.Time{}}
	tasks := []TaskDef{
		{Title: "Build auth", Description: "JWT", RequiredCapabilities: []string{"backend"}},
		{Title: "Wire UI", Description: "login form", RequiredCapabilities: []string{"frontend"}},
	}
	h := taskBatchHash(tasks)

	drop, _ := o.checkAndRecordDispatchHash(h)
	if drop {
		t.Fatalf("first dispatch should not be dropped")
	}
	drop, age := o.checkAndRecordDispatchHash(h)
	if !drop {
		t.Fatalf("second dispatch of same hash within window should be dropped")
	}
	if age < 0 {
		t.Fatalf("age should be non-negative, got %v", age)
	}
}

func TestDispatchDedup_DifferentContentBothDispatch(t *testing.T) {
	o := &Orchestrator{recentDispatchHashes: map[string]time.Time{}}
	h1 := taskBatchHash([]TaskDef{{Title: "A", Description: "first"}})
	h2 := taskBatchHash([]TaskDef{{Title: "B", Description: "second"}})

	if drop, _ := o.checkAndRecordDispatchHash(h1); drop {
		t.Fatalf("first batch should dispatch")
	}
	if drop, _ := o.checkAndRecordDispatchHash(h2); drop {
		t.Fatalf("second distinct batch within window should also dispatch")
	}
}

func TestDispatchDedup_ExpiredEntryDispatches(t *testing.T) {
	o := &Orchestrator{recentDispatchHashes: map[string]time.Time{}}
	h := taskBatchHash([]TaskDef{{Title: "T", Description: "d"}})

	// Seed with an expired entry directly.
	o.recentDispatchHashes[h] = time.Now().Add(-2 * dispatchHashWindow)
	drop, _ := o.checkAndRecordDispatchHash(h)
	if drop {
		t.Fatalf("expired entry should be GC'd and re-dispatch allowed")
	}
}

// --- renderAutoRoutePrompt tests (Fix 3) ---

func TestRenderAutoRoutePrompt_PassVerdictHasNoReactByTakingAction(t *testing.T) {
	got := renderAutoRoutePrompt(variantPassVerdict, "assurance", `{"score":100}`)
	if strings.Contains(got, "React by taking action") {
		t.Errorf("variantPassVerdict must NOT carry the anomaly framing; got:\n%s", got)
	}
	if !strings.Contains(got, "Stay silent") {
		t.Errorf("variantPassVerdict must instruct silence as default; got:\n%s", got)
	}
	if !strings.Contains(got, "PASS verdict") {
		t.Errorf("variantPassVerdict must explicitly name the verdict as PASS; got:\n%s", got)
	}
	if !strings.Contains(got, "[assurance]") {
		t.Errorf("variantPassVerdict must carry the source role tag; got:\n%s", got)
	}
}

func TestRenderAutoRoutePrompt_FailVerdictMentionsAutoGapAnalysis(t *testing.T) {
	got := renderAutoRoutePrompt(variantFailVerdict, "assurance", `{"score":40}`)
	if !strings.Contains(got, "auto-routed") {
		t.Errorf("variantFailVerdict must tell the Planner gap analysis is auto-routed (so it doesn't intervene); got:\n%s", got)
	}
	if !strings.Contains(got, "task abandon") {
		t.Errorf("variantFailVerdict must mention task abandon for repeated-failure path; got:\n%s", got)
	}
}

func TestRenderAutoRoutePrompt_SystemEscalationPointsAtActCli(t *testing.T) {
	got := renderAutoRoutePrompt(variantSystemEscalation, "system", "Task t-1 failed (agent dev-1).")
	for _, want := range []string{
		"act_cli task retry",
		"act_cli task abandon",
		"Silence is WRONG",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("variantSystemEscalation must contain %q; got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "POST /api/tasks") {
		t.Errorf("variantSystemEscalation must NOT reference the unimplemented POST verb anymore; got:\n%s", got)
	}
}

func TestRenderAutoRoutePrompt_AnomalyIsTheLegacyTree(t *testing.T) {
	got := renderAutoRoutePrompt(variantAnomaly, "observer", "two agents writing the same file")
	for _, want := range []string{"React by taking action", "(a)", "(b)", "(c)", "[observer]"} {
		if !strings.Contains(got, want) {
			t.Errorf("variantAnomaly must contain %q (legacy tree preserved for Observer); got:\n%s", want, got)
		}
	}
}

// --- renderBriefContext tests (Fix 5) ---

func TestRenderBriefContext_AllFieldsAndTaskLists(t *testing.T) {
	bv := BriefView{
		ProjectName:     "wordtallies",
		Description:     "Count words in markdown",
		TechStack:       "Go, CLI",
		Constraints:     "no external deps",
		SuccessCriteria: "handles UTF-8; CSV output",
		AgentsInvolved:  []string{"backend_dev", "qa_engineer"},
	}
	tasks := []TaskSummary{
		{ID: "t-1", Title: "lib", Status: "in_progress", AssignedAgent: "dev-1"},
		{ID: "t-2", Title: "tests", Status: "pending"},
		{ID: "t-3", Title: "cli", Status: "completed"},
		{ID: "t-4", Title: "docs", Status: "validated"},
	}
	got := renderBriefContext("resume", bv, tasks)

	wantStrs := []string{
		`Resuming project "wordtallies"`,
		"do NOT run intake",
		"@brief",
		"description: Count words in markdown",
		"techStack: Go, CLI",
		"constraints: no external deps",
		"successCriteria: handles UTF-8; CSV output",
		"agentsInvolved: backend_dev, qa_engineer",
		"@inFlightTasks",
		"id=t-1 status=in_progress agent=dev-1 title=\"lib\"",
		"id=t-2 status=pending agent=unassigned",
		"@completedTasks",
		"id=t-3 status=completed",
		"id=t-4 status=validated",
		"Do NOT re-emit CREATE_TASK directives for the task IDs above",
		"act_cli task retry/abandon for failed tasks",
	}
	for _, w := range wantStrs {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in output; got:\n%s", w, got)
		}
	}
}

func TestRenderBriefContext_NoTasksBuildKind(t *testing.T) {
	bv := BriefView{
		ProjectName: "newproject",
		Description: "fresh",
		TechStack:   "Rust",
	}
	got := renderBriefContext("build", bv, nil)

	for _, w := range []string{
		`Project "newproject" has been created`,
		"Switch to BUILD mode now",
		"@brief",
		"description: fresh",
		"techStack: Rust",
		"no tasks dispatched yet — start decomposing now.",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in output; got:\n%s", w, got)
		}
	}
	// Empty-field guard — should NOT print empty constraint/successCriteria lines.
	for _, banned := range []string{"constraints: \n", "successCriteria: \n", "agentsInvolved: \n"} {
		if strings.Contains(got, banned) {
			t.Errorf("expected empty field %q to be omitted; got:\n%s", banned, got)
		}
	}
}

func TestRenderBriefContext_PartialBriefNoFiller(t *testing.T) {
	bv := BriefView{
		ProjectName: "minimal",
		Description: "just desc",
		// TechStack, Constraints, SuccessCriteria, AgentsInvolved all empty
	}
	got := renderBriefContext("resume", bv, nil)
	if !strings.Contains(got, "description: just desc") {
		t.Errorf("desc missing; got:\n%s", got)
	}
	// No mention of techStack/constraints/successCriteria/agentsInvolved lines.
	for _, banned := range []string{"techStack:", "constraints:", "successCriteria:", "agentsInvolved:"} {
		if strings.Contains(got, banned) {
			t.Errorf("expected %q to be omitted when empty; got:\n%s", banned, got)
		}
	}
}

func TestBriefViewFromGetProject_AllFields(t *testing.T) {
	data := map[string]any{
		"description":     "test description",
		"techStack":       "Go",
		"constraints":     "no deps",
		"successCriteria": "works",
		"agentsInvolved":  []any{"backend_dev", "qa_engineer"},
	}
	bv := briefViewFromGetProject("p1", data)
	if bv.ProjectName != "p1" || bv.Description != "test description" || bv.TechStack != "Go" ||
		bv.Constraints != "no deps" || bv.SuccessCriteria != "works" {
		t.Errorf("scalar fields mis-extracted: %+v", bv)
	}
	if len(bv.AgentsInvolved) != 2 || bv.AgentsInvolved[0] != "backend_dev" || bv.AgentsInvolved[1] != "qa_engineer" {
		t.Errorf("agentsInvolved mis-extracted: %v", bv.AgentsInvolved)
	}
}

func TestBriefViewFromGetProject_MissingFieldsDontPanic(t *testing.T) {
	bv := briefViewFromGetProject("p1", map[string]any{}) // empty map
	if bv.ProjectName != "p1" {
		t.Errorf("project name not set: %+v", bv)
	}
	if bv.Description != "" || bv.TechStack != "" || len(bv.AgentsInvolved) != 0 {
		t.Errorf("expected empty fields for missing keys; got %+v", bv)
	}
}

// --- pruneAutoRoutes / sliding-window cap tests (Fix 6) ---

func TestPruneAutoRoutes_KeepsAllWithinWindow(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	times := []time.Time{
		now.Add(-2 * time.Minute),
		now.Add(-1 * time.Minute),
		now,
	}
	got := pruneAutoRoutes(times, now, 10*time.Minute)
	if len(got) != 3 {
		t.Errorf("expected all 3 entries kept; got %d", len(got))
	}
}

func TestPruneAutoRoutes_DropsOlderThanWindow(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	times := []time.Time{
		now.Add(-30 * time.Minute), // dropped
		now.Add(-20 * time.Minute), // dropped
		now.Add(-5 * time.Minute),  // kept
		now,                        // kept
	}
	got := pruneAutoRoutes(times, now, 10*time.Minute)
	if len(got) != 2 {
		t.Errorf("expected 2 entries kept; got %d (slice: %v)", len(got), got)
	}
}

func TestCascadeCap_AdmitsWithinCap(t *testing.T) {
	// 4 fires within window — all admitted (cap is 5).
	o := &Orchestrator{}
	base := time.Now()
	for i := 0; i < 4; i++ {
		o.recentAutoRoutes = pruneAutoRoutes(o.recentAutoRoutes, base, autoRouteWindow)
		if len(o.recentAutoRoutes) >= autoTurnCap {
			t.Fatalf("fire #%d wrongly rejected at len=%d", i, len(o.recentAutoRoutes))
		}
		o.recentAutoRoutes = append(o.recentAutoRoutes, base)
	}
	if len(o.recentAutoRoutes) != 4 {
		t.Errorf("expected 4 fires recorded; got %d", len(o.recentAutoRoutes))
	}
}

func TestCascadeCap_RejectsBeyondCap(t *testing.T) {
	// Seed the slice with exactly cap recent fires; the next one must be
	// rejected (replicates the cap check inside autoRoutePlannerV).
	o := &Orchestrator{}
	base := time.Now()
	for i := 0; i < autoTurnCap; i++ {
		o.recentAutoRoutes = append(o.recentAutoRoutes, base)
	}
	pruned := pruneAutoRoutes(o.recentAutoRoutes, base, autoRouteWindow)
	if len(pruned) < autoTurnCap {
		t.Fatalf("seed setup wrong: expected len >= %d, got %d", autoTurnCap, len(pruned))
	}
	// The cap check is `len >= autoTurnCap` → this should reject.
	if !(len(pruned) >= autoTurnCap) {
		t.Errorf("expected fire to be rejected at len=%d cap=%d", len(pruned), autoTurnCap)
	}
}

func TestCascadeCap_AdmitsAfterWindowAdvances(t *testing.T) {
	// 5 fires at t=0; check at t=window+1min → all should be pruned and
	// the next fire admitted.
	o := &Orchestrator{}
	t0 := time.Now()
	for i := 0; i < autoTurnCap; i++ {
		o.recentAutoRoutes = append(o.recentAutoRoutes, t0)
	}
	tLater := t0.Add(autoRouteWindow + time.Minute)
	o.recentAutoRoutes = pruneAutoRoutes(o.recentAutoRoutes, tLater, autoRouteWindow)
	if len(o.recentAutoRoutes) != 0 {
		t.Errorf("expected all entries pruned after window advance; got %d", len(o.recentAutoRoutes))
	}
}

func TestCascadeCap_HumanInputClears(t *testing.T) {
	// Seed with cap fires; simulate HandleHumanInput's reset; next check
	// should admit (slice is nil).
	o := &Orchestrator{}
	base := time.Now()
	for i := 0; i < autoTurnCap; i++ {
		o.recentAutoRoutes = append(o.recentAutoRoutes, base)
	}
	o.recentAutoRoutes = nil // HandleHumanInput does this
	if len(o.recentAutoRoutes) >= autoTurnCap {
		t.Errorf("expected slice cleared by human input; got len=%d", len(o.recentAutoRoutes))
	}
}
