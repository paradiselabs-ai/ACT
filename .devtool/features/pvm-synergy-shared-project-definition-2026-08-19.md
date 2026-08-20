---
id: "pvm-synergy-shared-project-definition-2026-08-19"
status: "todo"
priority: "low"
assignee: null
dueDate: null
created: "2026-08-19T21:45:00.000Z"
modified: "2026-08-19T21:45:00.000Z"
completedAt: null
labels: ["server", "PVM", "memory"]
order: "a6"
---
# PVM synergy: "collaboration" counts shared taskIds, so same-project agents report 0

## Spec
`getAgentSynergy` counts collaboration as tasks both agents touched. Swarm agents almost
never share a taskId — they share a *project*. Live result: two agents that worked the same
project report `collaborationCount: 0` (audit §3.1; re-confirmed during the 2026-08-19 fix
batch). Descoped from `pvm-agent-profile-event-type-pollution-2026-08-13` because changing
the definition of collaboration (shared task → shared project, or shared validation
pipeline) is an owner decision, not a bug fix.

## Success Criteria
- Owner picks the definition (recorded here), then: two agents with validated tasks in the
  same project return non-zero collaborationCount and a computed successRate.
- Unit test covers the chosen definition.

## Constraints
- Analytics read path only; no event emission changes; no new store.

## Invariants (code-level)
- No `Math.random`; unknowns absent, never 0-as-placeholder.
