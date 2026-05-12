---
id: "swarm-recovery-brief-injection-2026-05-12"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm-recovery", "runner", "epic-swarm-recovery"]
order: "a4"
---

# Runner: inject partialResult into AGENT.md brief on resumed tasks

When the Runner picks up a task whose `metadata.partialResult` is non-empty (i.e. it was aborted previously), inject that partial result + completed steps into the AGENT.md brief sent to the spawned subprocess. The resumed agent should see "you previously made this much progress, continue from there" instead of starting fresh.

## Why

Discovered 2026-05-12 — see `act-coordination.json`. Without this, the abort/retry loop is half-built: we save the partial result but never feed it back. The whole point is to avoid duplicating completed work + token waste.

## Acceptance criteria

- [ ] `act-runner.mjs` brief construction (the AGENT.md generator) checks `task.metadata.partialResult` and `task.metadata.completedSteps`
- [ ] If present, prepends a `## Resumed Task — Prior Progress` section with:
  - Reason for prior abort (`task.metadata.abortReason`)
  - Verbatim partial result (capped at 8KB; if larger, truncate with `...[truncated, full result in /api/tasks/:id]`)
  - Bulleted list of completed steps from `metadata.completedSteps`
  - Instruction: "Resume from this state. Do not redo completed steps. If you cannot reconcile the prior state with current files, run `act task progress 'unable to resume — restarting'` and start fresh."
- [ ] If `partialResult` empty/null → brief unchanged (no empty section noise)
- [ ] Brief generation function unit-tested with empty + populated metadata
- [ ] Manual smoke test: abort a real task, observe resumed agent's AGENT.md contains the section

## Files

- `act-agent/runner/act-runner.mjs` (brief construction)
- `act-agent/runner/act-runner.test.mjs`

## Depends on

- `swarm-recovery-partial-result-2026-05-12`
- `swarm-recovery-runner-abort-handler-2026-05-12`

## Blocks

Nothing — this is a leaf feature in the recovery loop.
