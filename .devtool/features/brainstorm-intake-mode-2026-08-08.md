---
id: "brainstorm-intake-mode-2026-08-08"
status: "backlog"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-08T08:50:00.000Z"
modified: "2026-08-08T08:50:00.000Z"
completedAt: null
labels: ["TUI", "planner", "intake", "feature"]
order: "a2"
---
# /brainstorm — Planner-assisted project shaping before intake locks a brief

## Describe
Users who don't yet know exactly what they want have no path in intake: the 5-question
form assumes answers exist. Proposed: a `/brainstorm` TUI command (offered by the
Planner in its describe-the-project question — see intake-description-depth ticket)
that flips the Planner into a collaborative ideation mode: open-ended questions,
options with trade-offs, scope-shaping ("MVP vs later"), until the user says they're
ready — then the Planner summarizes the shaped project and drops back into the normal
intake flow (summary → confirmation gate → PROJECT_BRIEF).

## Success Criteria
- `/brainstorm` registered in the slash-command handler; only meaningful during
  intake (outside intake: helpful error).
- Planner prompt gains a BRAINSTORM sub-mode: no task creation, no CLI tools, no
  brief emission while active; ends only when the user signals done, then re-enters
  the standard summary → confirm → brief path (the confirmation gate still applies).
- Mode entry/exit visible in the TUI (system message), and the mode survives the
  turn loop (orchestrator tracks it like intakeMode).
- e2e: a vague opener + /brainstorm + a short ideation exchange ends in a brief whose
  description is materially richer than the opener. Transcript evidence before done.

## Constraints
- Design before build: how the orchestrator tracks the sub-mode (alongside the
  existing intakeMode flag), and how brainstorm turns are kept out of the swarm/task
  machinery entirely.
- No new backend calls beyond normal Planner turns; brainstorm is conversation only.

## Invariants
- CREATE_TASK / PROJECT_BRIEF parsing is inert while brainstorm mode is active
  (code-enforced, not prompt-wished).
- Confirmation gate (intake-confirmation-gate-2026-08-08) is a hard dependency —
  brainstorm exits INTO the gated confirm step, never straight to a brief.
