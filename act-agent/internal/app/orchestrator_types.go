package app

import (
	"encoding/json"
	"fmt"
)

// Types used by the orchestrator for parsing server responses and passing
// data between coordination components. Mostly pure data structs with JSON
// tags; dependencyList carries a forgiving UnmarshalJSON (see Fix 10).

// ProjectBrief is the structured intake artifact the Planner produces during
// INTAKE mode and POSTs to the ACT server. The 5 fields map directly to the
// server's ProjectRecord type — see server/src/index.ts.
type ProjectBrief struct {
	Description     string   `json:"description"`
	TechStack       string   `json:"techStack"`
	Constraints     string   `json:"constraints,omitempty"`
	SuccessCriteria string   `json:"successCriteria"`
	AgentsInvolved  []string `json:"agentsInvolved"`
	// CodebaseNotes is NOT emitted by the Planner — the orchestrator injects the
	// brownfield codebase analysis here before writeAgentsMd so it lands as a
	// "## Codebase analysis" section in AGENTS.md.
	CodebaseNotes string `json:"-"`
}

// BriefView is the orchestrator-side view of a project brief, used to
// render the resume/BUILD-mode context block. Populated from either
// GetProject (resume path) or a fresh ProjectBrief (BUILD path) — the
// two sources speak different shapes but feed the same renderer.
type BriefView struct {
	ProjectName     string
	Description     string
	TechStack       string
	Constraints     string
	SuccessCriteria string
	AgentsInvolved  []string
}

// TaskSummary is a single task as returned by /api/tasks endpoints.
type TaskSummary struct {
	ID                   string         `json:"id"`
	Title                string         `json:"title"`
	Description          string         `json:"description"`
	Status               string         `json:"status"`
	AssignedAgent        string         `json:"assignedAgent,omitempty"`
	Priority             string         `json:"priority,omitempty"`
	RequiredCapabilities []string       `json:"requiredCapabilities,omitempty"`
	Dependencies         []string       `json:"dependencies,omitempty"`
	SuccessCriteria      []string       `json:"successCriteria,omitempty"`
	RetryCount           int            `json:"retryCount,omitempty"`
	CreatedAt            string         `json:"createdAt,omitempty"`
	CompletedAt          string         `json:"completedAt,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

// AgentSummary is a single registered agent as returned by /api/agents.
type AgentSummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	Status       string   `json:"status"`
	CurrentTask  string   `json:"currentTask,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	LastSeen     string   `json:"lastSeen,omitempty"`
}

// FileLockSummary is a single file lock as returned by /api/files/locks.
type FileLockSummary struct {
	File     string `json:"filePath"`
	AgentID  string `json:"agentId"`
	TaskID   string `json:"taskId,omitempty"`
	LockedAt string `json:"lockedAt,omitempty"`
}

// LogEntry is a single coordination log event as returned by /api/log.
type LogEntry struct {
	Type      string         `json:"type"`
	Agent     string         `json:"agent,omitempty"`
	Message   string         `json:"message,omitempty"`
	Timestamp string         `json:"timestamp,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// StatusSnapshot is a point-in-time view of the entire ACT system state.
// Built by the Observer loop by polling multiple endpoints in parallel.
type StatusSnapshot struct {
	Tasks        []TaskSummary     `json:"tasks"`
	Agents       []AgentSummary    `json:"agents"`
	FileLocks    []FileLockSummary `json:"fileLocks"`
	RecentEvents []LogEntry        `json:"recentEvents"`
	Timestamp    string            `json:"timestamp"`
}

// AnomalySeverity classifies how urgent an Observer-detected anomaly is.
type AnomalySeverity string

const (
	SeverityCritical AnomalySeverity = "critical"
	SeverityWarning  AnomalySeverity = "warning"
	SeverityInfo     AnomalySeverity = "info"
)

// AnomalyCategory groups anomalies by detection rule.
type AnomalyCategory string

const (
	CategoryStuckTask    AnomalyCategory = "stuck_task"
	CategoryStaleLock    AnomalyCategory = "stale_lock"
	CategoryIdleAgent    AnomalyCategory = "idle_agent"
	CategoryUnvalidated  AnomalyCategory = "unvalidated"
	CategoryBottleneck   AnomalyCategory = "bottleneck"
	CategoryFileConflict AnomalyCategory = "conflict"
	CategoryFailedTask   AnomalyCategory = "failed_task"
)

// Anomaly is a single issue detected by the Observer's monitoring loop.
type Anomaly struct {
	Severity AnomalySeverity `json:"severity"`
	Category AnomalyCategory `json:"category"`
	Message  string          `json:"message"`
	TaskID   string          `json:"taskId,omitempty"`
	AgentID  string          `json:"agentId,omitempty"`
}

// CriterionResult is the score and feedback for a single @success_criteria item
// produced by the Assurance agent.
type CriterionResult struct {
	Criterion string `json:"criterion"`
	Passed    bool   `json:"passed"`
	Score     int    `json:"score,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
}

// ValidationVerdict is the full Assurance verdict for a task.
// Score is 0-100, weighted average across all criteria. Pass threshold = 95.
type ValidationVerdict struct {
	TaskID                  string            `json:"taskId"`
	Passed                  bool              `json:"passed"`
	OverallScore            int               `json:"overallScore"`
	CriteriaResults         []CriterionResult `json:"criteriaResults"`
	SelfVerificationChecked bool              `json:"selfVerificationChecked"`
	SelfVerificationValid   bool              `json:"selfVerificationValid"`
	Gaps                    string            `json:"gaps,omitempty"`
	Feedback                string            `json:"feedback,omitempty"`
	Timestamp               string            `json:"timestamp,omitempty"`
}

// ValidatedOutput is a single Assurance-passed task output queued for QA assembly.
type ValidatedOutput struct {
	TaskID          string `json:"taskId"`
	TaskTitle       string `json:"taskTitle"`
	AgentID         string `json:"agentId"`
	Result          string `json:"result"`
	ValidationScore int    `json:"validationScore"`
	AddedAt         string `json:"addedAt"`
}

// AssemblyState is the QA/Synthesizer's running view of project assembly.
type AssemblyState struct {
	ProjectName string            `json:"projectName"`
	Queue       []ValidatedOutput `json:"queue"`     // Waiting to be integrated
	Assembled   []ValidatedOutput `json:"assembled"` // Already integrated
	Deliverable string            `json:"deliverable,omitempty"`
}

// TaskDef is a parsed CREATE_TASK directive from a Planner response.
// The orchestrator will pass this to act.Client.CreateTask().
type TaskDef struct {
	Title                string         `json:"title"`
	Name                 string         `json:"name,omitempty"`
	Description          string         `json:"description"`
	RequiredCapabilities []string       `json:"requiredCapabilities,omitempty"`
	Capabilities         []string       `json:"capabilities,omitempty"`
	Priority             string         `json:"priority,omitempty"`
	Dependencies         dependencyList `json:"dependencies,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

// dependencyList is a string slice with a forgiving JSON unmarshaler.
// Audit Fix 10 (entry 6.4): the prompt used to say "Empty array or
// omit if none" — two valid encodings. Smaller models occasionally
// emit `"dependencies": ""` (empty string) or `null` or even a single
// string instead of an array. The stock []string unmarshaler accepts
// [], null, and missing → nil/empty, but REJECTS "" — and when "" is
// emitted the WHOLE TaskDef silently fails to unmarshal and the
// CREATE_TASK directive disappears with only a debug log.
//
// This forgiving form coerces every recoverable shape to nil/[]:
//   - `[]` or `["a","b"]`  → []string{...}   (canonical happy path)
//   - `null`               → nil              (Go default for missing)
//   - `""`                 → nil              (the dropping bug)
//   - `"single"`           → []string{"single"} (single-string fallback)
// Anything else (number, object) returns an error so the unmarshal
// fails loudly rather than silently — novel hallucination shapes
// should surface, not be hidden.
type dependencyList []string

// UnmarshalJSON implements json.Unmarshaler with the forgiving rules
// documented on dependencyList.
func (d *dependencyList) UnmarshalJSON(data []byte) error {
	// `null` or omitted maps to a nil slice — stock Go behavior.
	if len(data) == 0 || string(data) == "null" {
		*d = nil
		return nil
	}
	// Array form is the canonical happy path.
	if data[0] == '[' {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return fmt.Errorf("dependencies: array decode: %w", err)
		}
		*d = arr
		return nil
	}
	// String form: "" → nil; any other string → single-element slice.
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("dependencies: string decode: %w", err)
		}
		if s == "" {
			*d = nil
		} else {
			*d = []string{s}
		}
		return nil
	}
	// Anything else (number, object, bool) — refuse so novel
	// hallucination shapes surface in the parse-failure preview
	// rather than being hidden behind a forgiving coercion.
	return fmt.Errorf("dependencies: unsupported JSON shape %q (expected array, string, or null)", string(data))
}
