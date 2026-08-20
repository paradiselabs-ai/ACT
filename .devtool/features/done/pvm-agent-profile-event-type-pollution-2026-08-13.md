---
id: "pvm-agent-profile-event-type-pollution-2026-08-13"
status: "done"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-13T17:30:00.000Z"
modified: "2026-08-19T21:45:00.000Z"
completedAt: "2026-08-19T21:45:00.000Z"
labels: ["server", "PVM", "memory"]
order: "a3"
---
# PVM agent profiles list event types as capabilities

## Spec
`GET /api/pvm/profile?agentId=dev-1` returns capability buckets that include ChronLog **event
type names** — `coordination` (taskCount 460, successRate 0), `agent_registered` (taskCount 8,
successRate 0) — alongside real capabilities (`python`, `go`, `javascript`). Any consumer
reading this for routing sees an agent that "failed 460 tasks". `avgCompletionTime` is 0 for
most buckets. `GET /api/pvm/synergy` returned `collaborationCount: 0` for two agents that
demonstrably worked the same project.

## Success Criteria
- Profile buckets contain only registered capability tags — no event-type keys.
- `avgCompletionTime` is either a real duration or explicitly absent, never a silent 0.
- Synergy returns non-zero collaboration for two agents with tasks in the same project.

## Constraints
- Do not re-implement the analytics layer (CLAUDE.md pitfall 7); fix the bucket key selection
  and the join inputs only.
- No placeholder/synthetic values — an unknown stays absent.

## Invariants (code-level)
- Capability vocabulary comes from `agent_registered.data.capabilities`, nothing else.
- Confidence/evidence-quality labels keep their current thresholds.

## Repro / Evidence
`docs/audits/memory-system-audit-2026-08-13.md` §3.1 (live JSON captured).

## Resolution note (2026-08-19)
Capability buckets + avgCompletionTime criteria fixed and live-verified (opus-task-a). The synergy criterion (collaborationCount for same-project agents) was DESCOPED — redefining "collaboration" from shared-task to shared-project is an owner call. Split to `pvm-synergy-shared-project-definition-2026-08-19`.
