---
id: "swarm-recovery-planner-prompt-2026-05-12"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm-recovery", "prompt", "tier1", "epic-swarm-recovery"]
order: "a5"
---

# Planner: document abort tool in system prompt

Update `act-agent/internal/llm/prompt/planner.go` so the Planner knows the abort endpoint exists, when to use it, and how it differs from `retry`/`reassign`.

## Why

Discovered 2026-05-12 — see `act-coordination.json`. The plumbing (abort endpoint, state machine, runner handler, brief injection) is useless if the Planner doesn't know to invoke it. Without explicit prompt guidance the Planner will keep using `retry` even when the original subprocess is hung, perpetuating the zombie problem.

## Acceptance criteria

Planner prompt MUST include:

- [ ] Section "Recovering from a hung swarm agent" listing decision criteria:
  - Use `POST /api/agents/:id/abort` when:
    - Observer reports task stuck > 2× `stuckTaskMinutes` (i.e. > 6 min) AND agent's `liveProcess` is still registered
    - File conflict detected between zombie and replacement (same file claimed by two agents on same task family)
    - Manual abort requested by human ("kill that agent")
  - Use `POST /api/tasks/:id/retry` when:
    - Task `failed` cleanly (subprocess exited non-zero, no zombie)
    - First retry attempt only — escalate to abort+reassign on second failure
  - Use `CREATE_TASK:` reassign when:
    - Wrong role chosen originally (e.g. backend task assigned to frontend_dev)
    - Task scope was wrong; rewrite the spec
- [ ] Worked example: "Observer says dev-2 stuck 7min on task X. liveProcess still registered. Action: POST /api/agents/dev-2/abort with reason='stuck in tool loop, 7min idle'. Wait for `task_aborted` event. Then POST /api/tasks/X/retry to resume on a fresh subprocess (partial progress will be injected automatically)."
- [ ] Counter-example to prevent misuse: "Do NOT abort an agent that's actively making progress (recent `task_progress` events). Abort is for hung subprocesses, not slow ones."
- [ ] Token budget: keep additions under 600 tokens (planner.go is already large)

## Files

- `act-agent/internal/llm/prompt/planner.go`

## Depends on

- `swarm-recovery-abort-endpoint-2026-05-12` (must exist before Planner can call it)
- All other swarm-recovery tasks (Planner shouldn't be told about a half-built feature)

## Blocks

Nothing.
