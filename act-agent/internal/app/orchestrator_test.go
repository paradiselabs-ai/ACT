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
	// Fix 19: the option-(a) "emit CREATE_TASK for the obvious next step"
	// escape hatch must be gone — it was an open invitation to fabricate
	// tasks on a PASS (worsened by the empty-criteria fail-open). The bare
	// "Never write the literal string 'CREATE_TASK:'" guard legitimately
	// remains, so assert the absence of the INSTRUCTION to emit, not the
	// absence of the substring "CREATE_TASK".
	for _, banned := range []string{
		"emit CREATE_TASK directives",
		"obvious next step",
	} {
		if strings.Contains(got, banned) {
			t.Errorf("variantPassVerdict must NOT carry the CREATE_TASK escape hatch %q (Fix 19); got:\n%s", banned, got)
		}
	}
	if !strings.Contains(got, "does not by itself signal new work") {
		t.Errorf("variantPassVerdict must explain that a PASS isn't a new-work signal (Fix 19); got:\n%s", got)
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
	// Fix 21: the trim must preserve the once/twice/three-times cadence that
	// mirrors planner_section_validation.go (FAIL once → watch; twice → check
	// criteria; three → reassign). Without this anchor the variant degrades
	// into a near-duplicate of variantAnomaly.
	if !strings.Contains(got, "First or second failure") {
		t.Errorf("variantFailVerdict must keep the first/second-failure 'stay silent' cadence; got:\n%s", got)
	}
	if !strings.Contains(got, "Third+ failure") {
		t.Errorf("variantFailVerdict must keep the third-failure 'reassign' cadence; got:\n%s", got)
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
		"Do NOT re-emit task-creation directives for the task IDs above",
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

// --- dependencyList forgiving-parse tests (Fix 10) ---

func TestParseCreateTaskDirectives_DependenciesShapes(t *testing.T) {
	type wantShape struct {
		taskCount int
		depsLen   int
		depsFirst string // checked only if depsLen >= 1
	}
	cases := []struct {
		name  string
		input string
		want  wantShape
	}{
		{
			name:  "canonical_array_with_one_entry",
			input: `CREATE_TASK: {"title":"T","description":"d","dependencies":["A"]}`,
			want:  wantShape{taskCount: 1, depsLen: 1, depsFirst: "A"},
		},
		{
			name:  "empty_array",
			input: `CREATE_TASK: {"title":"T","description":"d","dependencies":[]}`,
			want:  wantShape{taskCount: 1, depsLen: 0},
		},
		{
			name:  "null",
			input: `CREATE_TASK: {"title":"T","description":"d","dependencies":null}`,
			want:  wantShape{taskCount: 1, depsLen: 0},
		},
		{
			name:  "missing_field",
			input: `CREATE_TASK: {"title":"T","description":"d"}`,
			want:  wantShape{taskCount: 1, depsLen: 0},
		},
		{
			name:  "empty_string_was_the_dropping_bug",
			input: `CREATE_TASK: {"title":"T","description":"d","dependencies":""}`,
			want:  wantShape{taskCount: 1, depsLen: 0},
		},
		{
			name:  "single_string_coerces_to_one_element_slice",
			input: `CREATE_TASK: {"title":"T","description":"d","dependencies":"OnlyOne"}`,
			want:  wantShape{taskCount: 1, depsLen: 1, depsFirst: "OnlyOne"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tasks, _, fail, _ := parseCreateTaskDirectives(tc.input)
			if len(tasks) != tc.want.taskCount {
				t.Fatalf("expected %d task(s); got %d; firstFailPreview=%q",
					tc.want.taskCount, len(tasks), fail)
			}
			if len(tasks) == 0 {
				return
			}
			deps := tasks[0].Dependencies
			if len(deps) != tc.want.depsLen {
				t.Fatalf("expected %d deps; got %d (%v)", tc.want.depsLen, len(deps), deps)
			}
			if tc.want.depsFirst != "" && deps[0] != tc.want.depsFirst {
				t.Errorf("deps[0] = %q; want %q", deps[0], tc.want.depsFirst)
			}
		})
	}
}

// --- dispatch-dedup notification tests (Fix 12) ---

// TestDedupAutorouteText covers the message body the Planner sees on a
// dispatch-hash dedup event. The text has to actually nudge the Planner
// toward "change a title and re-emit" or "stop re-emitting" — silent
// drop was the bug the audit caught, so any future edit that loses
// these cues regresses Fix 12.
func TestDedupAutorouteText(t *testing.T) {
	got := dedupAutorouteText(3, 12*time.Second)
	for _, want := range []string{
		"duplicate of one dispatched",
		"12s ago",
		"3 tasks",
		"change at least one task title or description",
		"stop re-emitting",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in dedup autoroute text; got:\n%s", want, got)
		}
	}
}

// TestDedupAutorouteText_WrappedByVariantSystemNoTask locks audit Fix 18:
// the dedup autoroute wraps with variantSystemNoTask, NOT
// variantSystemEscalation. A dropped batch created no task, so the
// escalation variant's "Silence is WRONG" and "task retry/abandon <id>"
// menu were both wrong here (the contradiction the dedup body's "stop
// re-emitting / wait" guidance exposed). The no-task wrapper must omit the
// task menu and the silence-is-wrong directive while preserving the
// dedup-specific body.
func TestDedupAutorouteText_WrappedByVariantSystemNoTask(t *testing.T) {
	body := dedupAutorouteText(2, 8*time.Second)
	full := renderAutoRoutePrompt(variantSystemNoTask, "system", body)
	for _, want := range []string{
		"no failed task to retry or",
		"duplicate of one dispatched",
		"2 tasks",
		"stop re-emitting",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("expected %q in wrapped prompt; got:\n%s", want, full)
		}
	}
	// Ban the escalation variant's MENU instruction forms (the "<id>"
	// placeholder marks an instruction TO act on a task), not the no-task
	// wrapper's own "do NOT call ... task retry/abandon" prohibition.
	for _, banned := range []string{
		"Silence is WRONG",
		"task retry <id>",
		"task abandon <id>",
	} {
		if strings.Contains(full, banned) {
			t.Errorf("dedup must NOT be wrapped with %q (no task exists to act on); got:\n%s", banned, full)
		}
	}
}

// TestRenderAutoRoutePrompt_SystemNoTaskHasNoTaskMenu locks the variant
// itself: variantSystemNoTask omits the retry/abandon task menu and names
// the "no failed task" condition, so synthesis_stuck (already-validated
// task) and dedup (no task) never see retry/abandon instructions.
func TestRenderAutoRoutePrompt_SystemNoTaskHasNoTaskMenu(t *testing.T) {
	got := renderAutoRoutePrompt(variantSystemNoTask, "system", "synthesis stuck on task \"x\"")
	if !strings.Contains(got, "no failed task to retry or") {
		t.Errorf("variantSystemNoTask must name the no-failed-task condition; got:\n%s", got)
	}
	for _, banned := range []string{"task retry <id>", "task abandon <id>", "Silence is WRONG"} {
		if strings.Contains(got, banned) {
			t.Errorf("variantSystemNoTask must NOT contain the escalation menu form %q; got:\n%s", banned, got)
		}
	}
}

// --- maybeRouteQAClarification tests (Fix 11) ---

// TestClarificationRegex_AddresseeExtraction is the upstream contract
// maybeRouteQAClarification depends on. Locks the regex against
// reshape — if NEED_CLARIFICATION parse changes, Fix 11's routing
// breaks silently.
func TestClarificationRegex_AddresseeExtraction(t *testing.T) {
	cases := []struct {
		name, input, wantAddr, wantQ string
	}{
		{"swarm_agent", "NEED_CLARIFICATION: @dev-1 What encoding?", "dev-1", "What encoding?"},
		{"compound_id", "NEED_CLARIFICATION: @backend_dev-2 DB schema?", "backend_dev-2", "DB schema?"},
		{"planner_addressee", "NEED_CLARIFICATION: @planner override?", "planner", "override?"},
		{"multiline_question", "NEED_CLARIFICATION: @dev-1 Why does\nthis fail?", "dev-1", "Why does\nthis fail?"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := clarificationRegex.FindStringSubmatch(c.input)
			if m == nil {
				t.Fatalf("regex did not match %q", c.input)
			}
			if m[1] != c.wantAddr {
				t.Errorf("addressee = %q; want %q", m[1], c.wantAddr)
			}
			if m[2] != c.wantQ {
				t.Errorf("question = %q; want %q", m[2], c.wantQ)
			}
		})
	}
}

// TestClarificationRegex_NoMarker confirms freeform / SYNTHESIS_COMPLETE
// content does NOT match — Fix 11 must fall through to the autoroute in
// that case.
func TestClarificationRegex_NoMarker(t *testing.T) {
	cases := []string{
		"SYNTHESIS_COMPLETE: assembled all 4 outputs",
		"plain freeform message with no marker",
		"NEED_CLARIFICATION without colon @dev-1 q",  // no colon — regex requires `:`
		"NEED_CLARIFICATION: @dev-1",                 // no whitespace-then-question — regex requires `\s+(.*)`
	}
	for i, in := range cases {
		if m := clarificationRegex.FindStringSubmatch(in); m != nil {
			t.Errorf("case %d %q must NOT match; got %v", i, in, m)
		}
	}
}

// TestNormalizeRole locks the canonical-role mapping (Fix 13a, entry 6.1).
// `qa` → `qa_synthesizer`; everything else passes through unchanged.
func TestNormalizeRole(t *testing.T) {
	cases := []struct{ in, want string }{
		{"qa", "qa_synthesizer"},
		{"qa_synthesizer", "qa_synthesizer"},
		{"planner", "planner"},
		{"observer", "observer"},
		{"assurance", "assurance"},
		{"developer", "developer"},
		{"unknown-role", "unknown-role"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeRole(c.in); got != c.want {
			t.Errorf("normalizeRole(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestParseCreateTaskDirectives_DependenciesNumberDoesNotSilentlyDropTask(t *testing.T) {
	// Number is not a recoverable shape — the whole task should fail to
	// unmarshal so the firstFailPreview surfaces the hallucination instead
	// of silently coercing.
	input := `CREATE_TASK: {"title":"T","description":"d","dependencies":42}`
	tasks, markersFound, fail, _ := parseCreateTaskDirectives(input)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks (number is unrecoverable); got %d", len(tasks))
	}
	if markersFound != 1 {
		t.Errorf("expected 1 marker found; got %d", markersFound)
	}
	if fail == "" {
		t.Errorf("expected non-empty firstFailPreview surfacing the bad JSON; got empty")
	}
}

// --- buildSynthesisPrompt tests ---

// TestBuildSynthesisPrompt_IncludesValidationScore locks the wiring from
// ValidatedOutput.ValidationScore → the "Validation score: N/100" line the
// QA-Synthesizer reads. The bug this guards against: routeToQA (orchestrator.go)
// used to construct ValidatedOutput without setting ValidationScore, so Go
// zero-initialised the int to 0. The QA-Synthesizer received "Validation score:
// 0/100" even when Assurance gave a task a 100 — risking low-quality assembly
// decisions on fully-validated work.
func TestBuildSynthesisPrompt_IncludesValidationScore(t *testing.T) {
	o := ValidatedOutput{
		TaskID:          "task-abc",
		TaskTitle:       "Implement auth middleware",
		AgentID:         "dev-1",
		Result:          "auth module complete with tests",
		ValidationScore: 100,
	}
	prompt := buildSynthesisPrompt(o)

	if !strings.Contains(prompt, "Validation score: 100/100") {
		t.Errorf("expected prompt to contain 'Validation score: 100/100' when ValidationScore=100;\ngot:\n%s", prompt)
	}
	if strings.Contains(prompt, "Validation score: 0/100") {
		t.Errorf("prompt must NOT contain 'Validation score: 0/100' when ValidationScore=100 — "+
			"this is the zero-value bug: routeToQA forgot to populate ValidatedOutput.ValidationScore;\ngot:\n%s", prompt)
	}
}
