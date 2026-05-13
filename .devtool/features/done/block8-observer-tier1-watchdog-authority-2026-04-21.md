---
id: "block8-observer-tier1-watchdog-authority-2026-04-21"
status: "done"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-05-07T09:50:00.000Z"
completedAt: "2026-05-07T09:50:00.000Z"
labels: ["v1-gate", "observer", "block-8"]
order: "b02"
---
# Block 8 — Observer Tier 1 Watchdog Authority

Observer currently monitors Tier 2 only; if Assurance or QA hangs, alert goes nowhere. v1 cannot ship "autonomous" with that gap.

Give Observer direct re-trigger authority for **mechanical** failures only; Planner keeps monopoly on **decisions**:

| Signature | Who acts |
|---|---|
| Assurance/QA hung (>5min no activity) | Observer → `runAgentTurn` on stuck agent |
| Validation queue backing up, Assurance idle | Observer → re-trigger Assurance |
| Assurance rejecting same task 3+ times | Observer routes up; Planner decides |
| QA can't synthesize (missing/contradictory inputs) | Observer routes up; Planner decides |

**File**: `act-agent/internal/app/orchestrator.go` — Observer loop already has `runAgentTurn` in scope. Add last-triggered-timestamp per Tier 1 agent. Trivial scope.

See BUILD_ORDER.md Block 8.
