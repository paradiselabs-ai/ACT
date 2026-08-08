---
id: "project-level-integration-verification-2026-08-08"
status: "todo"
priority: "high"
created: "2026-08-08T15:50:00.000Z"
completedAt: null
labels: ["orchestrator", "qa", "assurance", "validation"]
order: "a1"
---
# Project-level integration verification before "done" — per-task validation isn't project validation

## Describe
Assurance validates each task in isolation; QA assembles per-task. Nobody ever
executes the ASSEMBLED project, so "all tasks validated" coexists with a broken
whole. Live (link-dock, 2026-08-08): every task passed, Planner declared done,
yet the combined test run was 6 pass / 1 fail (two agents wrote overlapping test
suites; server listens on import → port collision when both suites load it).
Each artifact was individually fine; the composition was not.

## Design (proposed)
When the last open task passes validation, the orchestrator AUTO-CREATES one
final "integration verification" task assigned to qa_engineer (a swarm agent
with execution rights — Tier-1 roles keep no-bash per KI-02). Its description:
run the project's own test suite and check each PROJECT-level success criterion
from the brief against the assembled tree; its @success_criteria = the brief's
criteria verbatim. It flows through the normal pipeline (complete →
submit-for-validation → Assurance → QA), so the existing machinery enforces it.
Planner may not announce done until the integration task passes — code gate,
not prompt text.

## Success Criteria
- Orchestrator detects all-tasks-validated and creates exactly one integration
  task (idempotent — never a second while one is open/passed).
- Done-announcement gate: the Planner's completion report is held/re-prompted
  until the integration task passes (same pattern as the confirmation gate).
- Live e2e: link-dock-shaped project with a composition defect → integration
  task fails → fix task dispatched → passes → done announced.

## Constraints
- Execution stays in the swarm; Assurance/QA tool surface unchanged.
- Brief must carry machine-readable success criteria (already does).
