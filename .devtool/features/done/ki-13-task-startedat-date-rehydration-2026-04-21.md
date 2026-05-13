---
id: "ki-13-task-startedat-date-rehydration-2026-04-21"
status: "done"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-05-07T09:35:00.000Z"
completedAt: "2026-05-07T09:35:00.000Z"
labels: ["bug", "server", "event-sourcing"]
order: "a08"
---
# KI-13: Server 500 `task.startedAt.getTime is not a function`

Event-sourcing replay on server restart rehydrates `startedAt` (+ `completedAt`/`createdAt`/`updatedAt`) as ISO strings, not `Date` instances. Any subsequent `.getTime()` call crashes. Repro'd 2026-04-19 on task `348800fc` — runner abandoned the task.

**Fix**: coerce to `Date` at the replay/hydration boundary (preferred: `server/src/services/TaskCoordinator.ts` or `ChronologicalLog.ts`). Add regression test simulating replay → /complete.

Constraints: server-layer only. No runner/CLI/Go changes. Event log is append-only; only the in-memory hydration is in scope. See HANDOFF Track B for full success criteria.
