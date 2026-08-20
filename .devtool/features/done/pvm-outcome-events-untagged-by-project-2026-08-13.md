---
id: "pvm-outcome-events-untagged-by-project-2026-08-13"
status: "done"
priority: "critical"
assignee: null
dueDate: null
created: "2026-08-13T17:30:00.000Z"
modified: "2026-08-19T21:45:00.000Z"
completedAt: "2026-08-19T21:45:00.000Z"
labels: ["server", "PVM", "memory", "critical"]
order: "a0"
---
# PVM: task outcome events carry no project tag, so routing evidence can never accumulate

## Spec
Every task lifecycle event that carries an *outcome* (`task_assigned`, `task_completed`,
`task_submitted_for_validation`, `task_validated`, `task_validation_failed`) is appended to
ChronLog with a `data` payload that has no project name anywhere
(`data.projectName`, `data.metadata.projectName`, `data.task.metadata.projectName` all absent).
`PVMIndexer.extractProjectName` therefore buckets all of them as `__global__`.

`LocalEmbeddingVectorStore.getRoutingBrief` explicitly excludes `__global__` from the
"similar past projects" list and requires `taskCount > 0` (validations in that bucket).
Named project buckets contain only `project_created` / `agent_registered` / `task_created`
events — zero validations — so **the similar-projects evidence list is permanently empty by
construction**, independent of how much history ACT accumulates.

Second-order effect: role identity is learned from `agent_registered` (named bucket), while
validations sit in `__global__` where that map is empty — so every agent falls through
`roleOfAgent()` to the no-capabilities default `developer`.

## Success Criteria
- Every event appended for a task carries the owning project name in `data` (or the indexer
  resolves it via a taskId→project map at index time — either is acceptable).
- After running a full project through the API, `GET /api/pvm/routing-brief` for a similar new
  project returns a non-empty `similarProjects` entry naming that project with correct
  `taskCount`, `passed`, `kickbacks`.
- `perRole` reports the real roles present (e.g. `backend_dev`, `qa_engineer`, `frontend_dev`)
  with correct pass counts, and never invents a `developer` row for agents whose registered
  capabilities say otherwise.
- Existing history still indexes without error (untagged legacy events may stay `__global__`).

## Constraints
- Do NOT rewrite `coordination-log.jsonl`. It is append-only.
- Do NOT change the routing-brief text format or the confidence-label scheme; only its inputs.
- No new dependency, no new store. Fix at the emit site in `server/src/index.ts` and/or at
  `PVMIndexer.extractProjectName`.
- Do not "fix" this by removing the `__global__` filter — that would mix cross-project
  infrastructure events into project evidence.

## Invariants (code-level)
- `extractProjectName` remains a pure function of one event.
- A taskId→project map built at index time must be rebuilt on `indexAllEvents()` so a restart
  is deterministic (event sourcing invariant).
- `getProjectOutcomes()` continues to key on `projectOf(m)`; no per-caller special cases.

## Repro / Evidence
Live run 2026-08-13 (clean data dir, port 8099), 3 agents / 6 tasks / 4 pass / 2 fail:

```
curl "localhost:8099/api/pvm/routing-brief?description=Build%20a%20Go%20auth%20service...&capabilities=auth,api,postgres"
{"text":"Per-role track record (all projects):\n- developer: 100% pass over 3 tasks (low signal)",
 "similarProjects":[], "perRole":[{"role":"developer","tasks":3,"passRate":1}], "rolePairs":[]}
```

Bucketing measured over the run's log:
`task_assigned/completed/submitted/validated/validation_failed -> __global__`;
`project_created/agent_registered/task_created -> authsvc-beta`.

Full audit: `docs/audits/memory-system-audit-2026-08-13.md` §2.

## Resolution note (2026-08-19)
Fixed by opus-task-a: outcome emit sites now stamp the owning project into event data; task creation lifts a top-level projectName into task metadata; indexer keeps a taskId→project map as legacy fallback (extractProjectName stays pure). Live-verified: routing brief returns the seeded project with 6 tasks / 4 pass / 2 kickbacks and correct 3-role breakdown; legacy fallback proven on a copy of the real 1786-event log (similarProjects non-empty, 24/24 validations attributed). Replay-deterministic across restart.
