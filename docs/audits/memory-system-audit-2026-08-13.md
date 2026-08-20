# ACT Memory System — Live Verification Audit

**Date:** 2026-08-13
**Branch/commit audited:** `main` @ `8a432ad`
**Method:** doc claims → code read → **live server run against real history + a clean synthetic project**
**Verdict:** storage layer real; **evidence/recall layer structurally dead**

> Anti-trust note: every claim below was produced by running the server, not by reading code alone.
> Repro commands are inline. Nothing here is quoted from another doc.

---

## 0. TL;DR

| Layer | Doc claim | Runtime reality |
|---|---|---|
| Append-only event log | real, replays on restart | ✅ **TRUE** — 1785 events replayed to 15 projects / 42 tasks / 38 agents |
| Real embeddings (`all-MiniLM-L6-v2`) | real, active | ✅ **TRUE** — `[PVM] real embeddings active (dim=384)`, 1785 events indexed |
| Semantic search over coordination memory | real | ⚠️ **RUNS, LOW VALUE** — no relevance floor; top hit for an off-topic query scored 0.22 and was unrelated |
| **Evidence-based routing ("which agent succeeded at this before")** | real | ❌ **FALSE AT RUNTIME** — structurally cannot populate; see §2 |
| Per-role / per-project track record | real | ❌ **WRONG DATA** — collapses to one fake `developer` row |
| Agent brief persistence (`brief update` session save) | real | ❌ **NEVER FIRES** — 0 `brief_stored` events in the entire 1785-event history |
| Swarm agent cross-task memory | "the ACT server is the memory" | ⚠️ **THIN** — `context` returns current task + file locks only; no history, no lessons, no prior-failure recall |
| Coordination knowledge graph | "deferred, in-memory maps for now" | ❌ **DOC STALE** — a real bi-temporal edge store exists on disk (`data/graph-edges.jsonl`, 164 edges / 104 nodes) |

**One-sentence version:** ACT records everything and recalls almost nothing usable.

---

## 1. What is actually real (verified by running it)

Started the server clean (no watch) and read the boot log:

```bash
cd server && npx tsx src/index.ts
```

```
ChronologicalLog: Loaded 1785 existing events
Restored from ChronLog: 15 projects, 42 tasks, 0 briefs, 38 agents, 2 file locks
[PVM] real embeddings active (model=Xenova/all-MiniLM-L6-v2, dim=384)
📥 PVMIndexer found 1785 new events to index
[PVM] loaded 1615 cached embeddings from ./data/pvm-vectors.jsonl
PVM drain: 1495 from cache, 290 embedded fresh
✅ PVMIndexer successfully indexed 1785 events
```

Confirmed real:
- **Event sourcing** — full state rebuilt from `data/coordination-log.jsonl` on boot.
- **Real embeddings** — 384-dim MiniLM, not a hash fallback, with a persistent embedding cache
  (`data/pvm-vectors.jsonl`) so restarts don't re-pay the embed cost.
- **The graph store** — `data/graph-edges.jsonl`, entries carry `fact`, `episodeKeys`, `validAt`,
  and an active/inactive distinction (`edges: 55, activeEdges: 49`). This is a working
  Graphiti-shaped bi-temporal edge log, and **CLAUDE.md says it is deferred**.

Also confirmed real (guards, not memory, but they fired during the test):
- `submit-for-validation` rejects a task with no `@success_criteria` block.
- `validation-verdict` rejects `passed=true` with no per-criterion results.

---

## 2. The core failure: outcome events are untagged, so evidence can never accumulate

### 2.1 The mechanism

`PVMIndexer.extractProjectName` can only read a project name off `data.projectName`,
`data.metadata.projectName`, `data.task.metadata.projectName`, or a `project_created` event's
`data.name`. **The task lifecycle events that carry the outcomes do not have any of those fields.**

Measured on the clean synthetic run (`extractProjectName` semantics replayed over the log):

```
agent_registered              -> authsvc-beta     x3     ← tagged
project_created               -> authsvc-beta     x1     ← tagged
task_created                  -> authsvc-beta     x6     ← tagged
task_assigned                 -> __global__       x9     ← UNTAGGED
task_completed                -> __global__       x7     ← UNTAGGED
task_submitted_for_validation -> __global__       x6     ← UNTAGGED
task_validated                -> __global__       x5     ← UNTAGGED
task_validation_failed        -> __global__       x2     ← UNTAGGED
```

Two consequences, both mechanical, neither a data-volume problem:

1. **"Similar past projects" is impossible by construction.** `getRoutingBrief` filters out the
   `__global__` bucket and then requires `taskCount > 0`, where `taskCount` counts *validations
   in that bucket*. Named project buckets contain zero validations, forever. The list is
   permanently empty no matter how many projects ACT completes.
2. **Role attribution collapses to a fake `developer`.** Role identity is learned from
   `agent_registered` events — which live in the *named* bucket. The validations live in
   `__global__`, where that map is empty, so every agent falls through to the no-capabilities
   default: `developer`.

### 2.2 The live proof

A clean server (empty data dir, port 8099) was driven through a full, correctly-formed project
via the real REST API — 3 agents (backend / QA / frontend capability sets), 6 tasks,
**4 validated pass, 2 validation-failed** — then asked for routing evidence for the *next*
similar project:

```bash
curl "localhost:8099/api/pvm/routing-brief?description=Build%20a%20Go%20auth%20service%20with%20JWT%20and%20Postgres&capabilities=auth,api,postgres"
```

```json
{"text":"\nPer-role track record (all projects):\n- developer: 100% pass over 3 tasks (low signal)\n",
 "similarProjects":[], "perRole":[{"role":"developer","tasks":3,"passRate":1,"signal":"low"}], "rolePairs":[]}
```

Ground truth vs. what memory reported:

| | Ground truth | Reported |
|---|---|---|
| Roles | backend_dev, qa_engineer, frontend_dev | `developer` only |
| Tasks | 6 | 3 |
| Pass rate | 4/6 = 67% | 100% |
| Failures | 2 (with gap text) | 0 |
| Similar projects | 1 (`authsvc-beta`) | 0 |
| Role pairs | 3 | 0 |

The same query against the **real** 1785-event history returns the same shape of nothing:
`similarProjects: []`, one `developer` row at `1 task`, across 15 projects and 42 tasks.

This block is what `orchestrator.go` injects into the Planner's first BUILD turn under
`## Routing evidence from past projects`, and what the Planner's on-demand prompt section calls
its "primary evidence". **The Planner is being handed a confidently-formatted wrong answer.**

### 2.3 Secondary attribution bug (independent of the tagging bug)

Outcome→worker attribution joins `task_validated` to `task_assigned`. In the real history there
are **24 validations but only 5 assignment records with a payload**, so exactly **1** validated
task is attributable. `task_completed` *does* carry `agentId` on all 37 events and is not used
for this join.

```bash
# 24 validations, 5 assignment records, 1 joinable
python3 - <<'EOF'
import json
ev=[json.loads(l) for l in open('server/data/coordination-log.jsonl') if l.strip()]
a={}; v={}
for e in ev:
    d=e.get('data') or {}; t=d.get('taskId') or (d.get('task') or {}).get('id') or d.get('id')
    if not t: continue
    if e['type']=='task_assigned': a[t]=d.get('agentId')
    if e['type'].startswith('task_validat'): v[t]=e['type']=='task_validated'
print(len(v), len(a), len([t for t in v if t in a]))
EOF
```

---

## 3. Other verified gaps

### 3.1 Agent profiles are polluted by event types
`GET /api/pvm/profile?agentId=dev-1` returns real computed numbers (no `Math.random`), but the
"capability" buckets include **event type names**:

```json
"agent_registered":{"successRate":0,"taskCount":8}, "coordination":{"successRate":0,"taskCount":460}
```

A routing decision that read this would see `coordination: 0% success over 460 tasks`.
`avgCompletionTime` is `0` for most buckets. `GET /api/pvm/synergy` returned
`collaborationCount: 0` for two agents that worked the same project.

### 3.2 Brief persistence never fires
`brief_stored` events in the entire 1785-event history: **0**. The endpoint and the replay case
both exist and work; nothing ever calls them. The "session save before exit" loop
(`act-agent brief update`) is documented behavior that has never happened.

### 3.3 Swarm agents get no memory
Live run of the real CLI against real data:

```bash
cd act-agent && npx tsx cli/act-cli.ts context dev-1 --project link-dock
```
```
No task currently assigned to dev-1

## Files Locked by Others (do not edit)
  /shared/path/main.go — dev-1-alpha
```

That is the whole of a swarm agent's server-side memory: current task + file locks (+ brief and
inbox when present). **No prior-failure recall, no lessons, no past-attempt history.** A worker
that failed a task and gets a retry has no server-side record of *how* it failed available to it.

### 3.4 Search has no relevance floor
The similarity threshold exists in the store but is **opt-in**, and the HTTP route never passes
one. Query `"build a kanban board UI"` against real history returned a top hit at **0.22
similarity** about an unrelated README synthesis. Callers cannot distinguish "no relevant memory"
from "weak memory" — the API always returns `limit` rows.

### 3.5 Cost shape to know before scaling
`data/pvm-vectors.jsonl` is **19 MB for 2319 cached embeddings ≈ 8 KB/entry** (384 floats as JSON
text). Vector points live in a plain in-process array and search is a linear cosine scan over all
of them. Fine at 2K events; at 100K events that is ~800 MB of sidecar and a full scan per query.
No eviction, no compaction, no summarization anywhere in the pipeline.

### 3.6 Why the small test projects hid all of this
Every failure above is a *recall* failure, not a *write* failure. Small tasks completed inside one
worker turn with the task description alone, so nothing ever depended on recall. The write path
(log + embed + index) has always worked, which is exactly why it looked healthy.

---

## 4. Doc statements corrected by this audit

| Doc | Statement | Action |
|---|---|---|
| `README.md` | "Routing can cite evidence: which agent actually succeeded at this kind of task last time" | rewritten — aspiration, not runtime |
| `README.md` | "semantic index, queryable as evidence" | qualified |
| `CLAUDE.md` | "PVM … Evidence-based routing" | qualified with the live result |
| `CLAUDE.md` | "Coordination KG — deferred. In-memory maps for now." | corrected — a persistent bi-temporal edge log exists |
| `CLAUDE.md` pitfall 7 | analytics "REAL" | kept (they are computed, not faked) but scoped: computed ≠ correct |
| `docs/dev/ROADMAP.md` | "what's unverified is statistical quality under live data" | corrected — now verified as structurally broken, not merely unproven |

## 5. Tickets filed

- `pvm-outcome-events-untagged-by-project-2026-08-13` (critical)
- `pvm-role-attribution-joins-wrong-event-2026-08-13` (high)
- `pvm-agent-profile-event-type-pollution-2026-08-13` (medium)
- `pvm-search-no-relevance-floor-2026-08-13` (medium)
- `agent-brief-session-save-never-fires-2026-08-13` (high)

## 6. Repro artifacts

- Test harness: `scratchpad/memtest.py` (drives a full project lifecycle through the REST API)
- Clean-run data dir: `scratchpad/memtest/data/coordination-log.jsonl`
- Real log untouched except one benign `agent_status_updated` coordination event appended by the
  `context` CLI probe (append-only log; backup taken before the run).

---

## Postscript — FIXED 2026-08-19 (uncommitted, working tree on `main`)

The recall layer defects above were fixed six days later via the DISC plan
`~/.claude/plans/pvm-memory-fixes-2026-08-13.md` (two delegated Opus tasks, both
live-verified with the §2.2 harness):

- §2 tagging + §2.3 attribution: outcome events now project-tagged at emit; indexer keeps a
  taskId→project legacy fallback; attribution joins `task_completed` first. Re-running the
  §2.2 experiment now returns the seeded project with 6 tasks / 4 pass / 2 kickbacks and the
  correct 3-role breakdown; on a copy of the real 1786-event log, 24/24 validations attribute
  (was 1) and `similarProjects` is non-empty.
- §3.1 profile pollution: capability buckets now sourced from registered capabilities only.
  Synergy definition descoped → `pvm-synergy-shared-project-definition-2026-08-19`.
- §3.2 brief save: Runner writes a deterministic Recent-Work brief after each successful
  completion; verified end-to-end incl. restart replay and non-fatal failure path.
- §3.4 search floor: default 0.28, measured on both corpora (on-topic 0.31–0.83 vs off-topic
  0.03–0.23); the kanban noise query (0.221) now returns `[]`.
- New defect found during the fix: indexed points collide on `${timestamp}_${agent}`
  (1786 events → 1614 points) → `pvm-point-id-collision-same-ms-2026-08-19` (open; fix
  invalidates the 19 MB embedding sidecar).

Tests after both fixes: server jest 87/87, `tsc --noEmit` clean, runner node tests 11/11.
