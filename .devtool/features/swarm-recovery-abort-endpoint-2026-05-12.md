---
id: "swarm-recovery-abort-endpoint-2026-05-12"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm-recovery", "server", "epic-swarm-recovery"]
order: "a0"
---

# Server: POST /api/agents/:id/abort endpoint

Add a server endpoint that lets the Planner forcibly stop a hung swarm subprocess. The endpoint emits an `abort_signal` Socket.io event keyed to the agent ID; the Runner subscribes and kills its child PID.

## Why

Discovered 2026-05-12 — see `act-coordination.json` entry of the same date and `CLAUDE.md` "Multi-Agent Coordination Protocol". Today the Planner can call `/api/tasks/:id/retry` or emit a new `CREATE_TASK:`, but **the original hung subprocess keeps running**. Result: zombie process + replacement racing on the same files. Planner has no PID-level authority over Tier 2.

## Acceptance criteria

- [ ] `POST /api/agents/:id/abort` endpoint in `server/src/index.ts`
- [ ] Body accepts `{ reason: string, taskId?: string }`
- [ ] Emits `abort_signal` Socket.io event with `{ agentId, reason, taskId, timestamp }`
- [ ] Records the abort intent in ChronologicalLog as `agent_abort_requested`
- [ ] Returns 404 if agent unknown, 409 if agent has no `liveProcess` registered
- [ ] Idempotent — second abort within 30s returns 200 with `alreadyAborting: true`
- [ ] Unit test: emit event, assert Socket.io listener received it
- [ ] Documented in `act` CLI brief (no CLI surface yet — Planner calls via curl/HTTP tool)

## Files

- `server/src/index.ts` — endpoint
- `server/src/services/AgentRegistry.ts` — `liveProcess` lookup
- `server/src/services/ChronologicalLog.ts` — log call
- `server/test/abort-endpoint.test.ts` — new

## Depends on

Nothing (foundation task).

## Blocks

- `swarm-recovery-runner-abort-handler-2026-05-12`
- `swarm-recovery-planner-prompt-2026-05-12`
