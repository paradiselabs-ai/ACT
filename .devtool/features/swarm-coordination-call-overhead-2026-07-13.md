---
id: "swarm-coordination-call-overhead-2026-07-13"
status: "review"
priority: "medium"
assignee: null
dueDate: null
created: "2026-07-13T00:00:00.000Z"
modified: "2026-07-13T00:00:00.000Z"
completedAt: null
labels: ["runner", "swarm", "cost"]
order: "a1"
---
# Swarm per-task coordination calls: guard the pre-task call, decide on modelless broadcasts

## Describe
Every swarm task currently costs up to 3 full model calls: the task itself, a pre-task
"who should I coordinate with" call, and a post-task completion broadcast. Verified
2026-07-13 in `runner/act-runner.mjs` + `server/src/index.ts` / `EventHub.ts`:

- The coordination loop DOES close (not dead code): messages POSTed to `/api/messages`
  are buffered into every same-project agent's inbox (`bufferMessageForAgent`), @-mentions
  only to the target; each agent's next task prompt includes its unread inbox
  ("Pending Messages" section in `buildTaskPrompt`). Messages also feed the ChronLog
  (Observer + PVM indexing).
- **Waste 1 — pre-task call fires on ~every task.** `fetchParallelContext` returns
  non-null whenever ANY other agent is registered; all 5 runners register at spawn, so
  `proactiveCoordination` (a full model call) runs even when every peer is idle and the
  expected answer is NO_COORDINATION_NEEDED.
- **Waste 2 — broadcast shelf life is 10 minutes.** `INBOX_TTL_MS = 10 * 60 * 1000`
  (index.ts:357). The runner filters "last hour" but the server pruned already — a
  broadcast is only ever read if a peer STARTS a task within 10 min. Valuable during
  build bursts, dead weight for isolated tasks.

**Fix (firm):** guard the pre-task coordination call — skip it unless at least one OTHER
agent has a task with status `in_progress` or `assigned` (the data is already fetched in
`fetchParallelContext`; return null when the filtered task list for other agents is
empty, or add an explicit check before `proactiveCoordination`). Deterministic, zero
model calls, removes the dominant waste case.

**Option (founder decision, not spec'd firm):** replace the broadcast model call with a
deterministic message assembled from task title + first ~500 chars of output
(`status: completed "<title>" — <excerpt>`). Zero model calls, keeps inbox/ChronLog/
Observer value; loses the "what interfaces are now available" synthesis quality.

**Note (no action):** the runner's 1-hour `since` filter vs the server's 10-min TTL is a
cosmetic mismatch — server prunes first, so behavior is unchanged. Do not "fix" as a
side-effect; if desired, separate one-line change with its own commit message.

## Success Criteria
- With 5 runners registered and NO peer tasks in `in_progress`/`assigned`, executing a
  task performs exactly ONE backend invocation (the task itself) — verifiable from the
  runner log (`~/.act/runners/<role>.log`): no `[claude invoke]`/`[gemini invoke]`/agent
  CLI line for coordination before the task line.
- With a peer task in flight, the pre-task coordination call still runs and SEND
  directives still deliver to the target inbox (existing behavior unchanged).
- Existing runner tests green; no server changes.

## Constraints
- Touch only `runner/act-runner.mjs` (guard logic lives where the data already is).
- Do NOT remove the messaging channel itself — inbox + ChronLog have consumers
  (peers' next-task prompts, Observer, PVM indexer).
- No refactor of the coordination/broadcast functions beyond the guard; no server-side
  TTL changes (10-min TTL is a deliberate bound on stale context).
- ACT principle: Runner stays a thin spawner; coordination intelligence stays in
  prompts/server, but *whether to spend a model call* is deterministic runner logic.

## Invariants (code-level)
- `proactiveCoordination` is unreachable when no other agent has an
  `in_progress`/`assigned` task (assert via the guard's unit test or a log-based check).
- `bufferMessageForAgent` call sites in `server/src/index.ts` unchanged.
- `broadcastCompletion` still posts a `status:`-prefixed message on task success
  (whether model-written or deterministic, per the founder's option call).
