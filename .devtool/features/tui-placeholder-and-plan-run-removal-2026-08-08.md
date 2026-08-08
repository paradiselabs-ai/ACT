---
id: "tui-placeholder-and-plan-run-removal-2026-08-08"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-08T10:20:00.000Z"
completedAt: null
labels: ["TUI", "orchestrator", "boundary", "UX"]
order: "a3"
---
# Retire /plan + /run; make the input placeholder advertise REAL commands

## Describe
The chat input placeholder reads "try /plan, /run, or @planner <task>". Verified
2026-08-08: /plan and /run are prompt-prefix macros (TUI chat page prepends
"Create a detailed implementation plan for:" / "Execute this task directly:" and
sends as a normal Planner message); @planner is not an input feature at all
(exists only in the Observer's internal message format). Problems:
- Redundant: every user message already goes to the Planner; the Planner already
  plans on brief confirmation and the swarm already executes after planning.
- /run is a role-contract hazard: "execute this task directly" instructs the
  Planner to do the thing the build-contract gate auto-scolds (planless build).
- Double-wired across the domain boundary (TUI intercepts with-arg forms;
  app-layer slash handler answers bare forms with usage strings).
- @planner in the placeholder implies addressable agents, which is false by design.

## Proposal (boundary change — needs both founders' sign-off)
- Remove /plan and /run from both layers (TUI intercept + slash handler + the
  command palette entries + tui.go registration).
- Placeholder rotates/updates to REAL, useful commands, e.g.:
  "describe your project to start — or /status, /swarm, /help"
  and after a project exists: "/status · /swarm restart all · /backend <role> <backend>"
- /help gains the /swarm restart all line (already implemented, unadvertised).

## Success Criteria
- grep for the "/plan"//"/run" handlers → 0 hits in tui + app layers.
- Placeholder shows only commands that exist and do what they say.
- /help lists restart. Build clean, existing TUI tests green.

## Constraints
- TUI files are Domain C (Kareem) — this ticket is the written proposal per
  review etiquette; do not land without his ack. Orchestrator-side slash.go
  removal is the owner's half.
- Replacement commands (/feature, /brainstorm) are SEPARATE tickets — do not
  block removal on them.
