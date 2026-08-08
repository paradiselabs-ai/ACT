---
id: "feature-command-scoped-amendment-2026-08-08"
status: "backlog"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-08T10:20:00.000Z"
completedAt: null
labels: ["TUI", "planner", "orchestrator", "feature"]
order: "a4"
---
# /feature — mid-project scoped feature intake + intelligent decomposition

## Describe
Today there is no first-class way to add work to a RUNNING project; users just
type at the Planner and hope. /feature "<prompt>" gives the Planner a scoped
amendment flow:
1. Clarify-first: ask the user any clarifications needed (mini-intake, not the
   5-question form; constraints inherit from the project brief unless amended).
2. Intelligent decomposition: split a large prompt into multiple tasks — and
   explicitly NOT split when one task suffices (token conservation is a stated
   goal in the prompt); detect when one /feature prompt actually describes
   multiple features and separate them.
3. Emit tasks with correct dependency order (dependencies array, existing
   CREATE_TASK machinery — no new server surface).
4. For features that rely on specific behaviors, instruct swarm agents (in the
   task description) to encode code-level invariants/assertions guarding that
   behavior, whenever useful.
5. Brief update: append the accepted feature to the project brief/AGENTS.md so
   later tasks and validation know about it.

## Success Criteria
- /feature with empty args → usage; with args → Planner enters the amendment
  flow (clarify → decompose → confirm → CREATE_TASKs with deps).
- A deliberately double-feature prompt produces two independent task groups; a
  small single-feature prompt produces exactly one task (no gratuitous split).
- Tasks reference invariants in @success_criteria where behaviors warrant it.
- Confirmation hard stop applies before task emission (reuse the gate pattern).

## Constraints
- Prompt + orchestrator mode flag work; no new server endpoints.
- TUI half (command registration/palette) is Domain C — coordinate.
- Depends conceptually on brainstorm-intake-mode-2026-08-08's mode-tracking
  approach (same sub-mode mechanism); build after or alongside it.
