package app

import (
	"strings"
	"testing"
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

// --- parseValidationVerdict tests ---

func TestParseValidationVerdict_PassedAt100(t *testing.T) {
	raw := `Here's my verdict:
{
  "criteriaResults": [
    {"criterion": "Tests pass", "passed": true, "score": 100, "feedback": "all green"}
  ],
  "overallScore": 100,
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
    {"criterion": "Tests pass", "passed": false, "score": 60, "feedback": "2 failures"}
  ],
  "overallScore": 60,
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
	raw := `{"criteriaResults":[{"criterion":"x","passed":true,"score":95}],"overallScore":95}`
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

{"criteriaResults": [{"criterion": "compiles", "passed": true, "score": 100}], "overallScore": 80, "gaps": "missing tests"}

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
