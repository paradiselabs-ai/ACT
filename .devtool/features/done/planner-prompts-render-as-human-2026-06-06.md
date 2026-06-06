---
id: "planner-prompts-render-as-human-2026-06-06"
status: "todo"
priority: "medium"
assignee: "d34d"
dueDate: null
created: "2026-06-06T16:38:58.000Z"
modified: "2026-06-06T16:38:58.000Z"
completedAt: null
labels: ["orchestrator", "tui", "reliability"]
order: "a3"
---
# Orchestrator prompts to the Planner render as `Human:` (InternalPromptMarker skips the Planner)

**Symptom (act-e2e, 2026-06-06):** templated orchestrator prompts like
`Human: The observer agent just sent the following report. React by taking action.` (×12 in the
log) appear in the chat as if the human typed them.

**Cause:** in `runAgentTurn` (`internal/app/orchestrator.go` ~line 584) the InternalPromptMarker —
which the TUI uses to hide orchestrator-generated prompts — is only applied to **non-Planner**
agents:
```go
if role != "planner" {
    content = InternalPromptMarker + content
}
```
So every autoroute / trigger prompt sent *to the Planner* is a marker-less user-role message →
renders as a visible `Human:` line. Combines with the LLM-hallucinated `Human:` prefix
(`code-enforced-agent-role-prefix`) to make "who said this?" genuinely ambiguous.

**Fix direction:** distinguish **real human input** from **orchestrator-injected Planner prompts**.
Options: mark orchestrator-injected Planner prompts internal (hidden, like the other roles) but
render the *resulting Planner action*; or give injected prompts a distinct system/internal tag so
the TUI shows them as "ACT →" rather than "Human:". Real human messages stay `Human:`.

**Verify:** an Observer/QA/Assurance report auto-routed to the Planner must not appear as a `Human:`
message; only text the user actually typed shows as `Human:`.
