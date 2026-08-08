---
id: "coordination-graph-mvp-2026-08-08"
status: "review"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-08T00:00:00.000Z"
modified: "2026-08-08T00:00:00.000Z"
completedAt: null
labels: ["server", "PVM", "graph"]
order: "a0"
---
# Coordination Graph MVP — derived temporal edge layer over ChronLog

## Describe

ACT's coordination memory is a flat append-only event log plus a vector index. Verified
in code (2026-08-08, branch `feat/coordination-graph`):

- `server/src/types/coordination.ts:5` — `CoordinationMessage` has NO causal/parent
  field and NO id field. Events cannot reference the event that caused them.
- `server/src/services/ChronologicalLog.ts:149` — `append()` is the single event write
  path ("the ONLY way to add events").
- `server/src/services/PVMIndexer.ts:97-129` — indexer pattern: constructed with the
  log, polls `indexNewEvents()` on an interval. This is the hook shape to mirror.
- No prior graph work: `git log -S GraphStore` and `-S causedBy` empty; no kanban
  ticket matches graph/kg/causal. Existing `graph` CLI subcommands
  (`act-cli.ts:1290-1296`) are task-dependency views over the task API, not a KG.

Missing capability (deep-research verdict 2026-08-08, memory
`reference-agentic-graph-engineering`): multi-hop / causal / point-in-time queries —
"which agent's work led to this failure", "what did we believe at time T", "this
agent's validation pass rate". These are the three query classes where graph memory
beats vector search; they are also FLUX's prerequisite (causal edges).

The fix — a Graphiti-shaped minimal edge layer, NO graph DB:

1. **`server/src/services/GraphStore.ts`** (new). Edge record:
   `{src, rel, dst, fact, episodeKeys: string[], createdAt, expiredAt?, validAt?, invalidAt?}`.
   Node key format `"<type>:<name>"` with types limited to
   `agent | task | project | file | verdict`. Persistence: append-only JSONL at
   `./data/graph-edges.jsonl` (same data dir as the event log). In-memory adjacency
   maps built at construction by loading the file; if the file is missing, full
   rebuild by replaying the event log through the same derivation rules.
   Contradiction handling: invalidate-never-delete — a superseding edge stamps the old
   edge's `expiredAt`/`invalidAt` by appending an invalidation record
   (`{invalidates: <edgeId>, at}`) to the JSONL; in-memory edge is mutated, the
   original JSONL line is never rewritten.
2. **`server/src/services/GraphIndexer.ts`** (new). Mirrors PVMIndexer: constructed
   with the ChronologicalLog + GraphStore, polls new events, derives edges by RULES
   (no LLM calls): task_created → (agent)-[created]->(task) + (task)-[belongs_to]->(project);
   assignment → (task)-[assigned_to]->(agent); completion → (agent)-[completed]->(task);
   submit-for-validation → (task)-[submitted]->(verdict:pending);
   validation verdict → (verdict)-[judged]->(task) with pass/fail in `fact`, and the
   pending verdict edge invalidated; file claim/release → (agent)-[holds]->(file),
   release invalidates the hold edge. Event reference key (events have no id):
   `"<timestamp>|<agent>|<type>"`, used in `episodeKeys`.
3. **`server/src/types/coordination.ts`**: add ONE optional field
   `causedBy?: string` (an event reference key, same format). No emitter is required
   to set it yet; when present, GraphIndexer adds (event)-[caused_by]->(event) linkage
   via episodeKeys on derived edges.
4. **`server/src/index.ts`**: construct GraphStore + GraphIndexer next to the existing
   indexer wiring (~line 43); two endpoints:
   `GET /api/graph/node/:key` → `{node, edges: [...]}` with optional `?at=<ISO>`
   point-in-time filter (edge visible iff `validAt ?? createdAt <= at < invalidAt ?? ∞`)
   and optional `?hops=1|2` BFS expansion (default 1);
   `GET /api/graph/status` → `{nodes, edges, lastIndexedKey}`.
5. **`act-agent/cli/act-cli.ts`**: one new dispatch branch `graph node <key>`
   (fits the existing `graph` family at line ~1290) printing the endpoint result.

## Success Criteria

- `server/src/services/GraphStore.ts` and `server/src/services/GraphIndexer.ts` exist;
  `cd server && npx tsc --noEmit` clean; `npx jest GraphStore` passes.
- `server/src/__tests__/GraphStore.test.ts` (new) covers: (a) derivation of each rule
  above from synthetic events, (b) invalidation sets `expiredAt`/`invalidAt` without
  removing the edge, (c) point-in-time filter returns the old edge for T before
  invalidation and the new edge after, (d) rebuild-from-event-log produces the same
  adjacency as incremental indexing over the same events.
- With the server running against a log containing a task_created + completion event,
  `curl localhost:8080/api/graph/node/task:<id>` returns the created/completed edges;
  `/api/graph/status` reports non-zero counts.
- `causedBy` present as optional on the message type; `grep -n "causedBy" server/src/types/coordination.ts` hits.
- `act-agent graph node <key>` prints edges (manual check against running server).
- Kanban ticket updated; one DEV_LOG line appended.

## Constraints

- Touch ONLY: the two new service files, the new test file, `coordination.ts` (one
  optional field), `index.ts` (wiring + 2 endpoints), `act-cli.ts` (one branch).
  Nothing else. No changes to ChronologicalLog, PVMIndexer, EventHub, TaskCoordinator,
  vector stores, Go code, runner, prompts.
- NO new npm dependencies. No graph database of any kind (no neo4j/kuzu/falkordb/
  networkx-alikes). Adjacency = plain Maps.
- NO LLM calls anywhere in derivation or query. Rules only.
- Edge file is append-only, same discipline as the event log. Full rewrite happens
  only on explicit rebuild (delete file + replay), never in normal operation.
- Architectural principle: the event log stays the single source of truth
  (Three-Layer Separation: server owns deterministic state). The graph is a DERIVED,
  rebuildable projection — deleting `data/graph-edges.jsonl` must lose nothing.
- No retrieval-fusion / BM25 / RRF work in this ticket (phase 2). No community
  detection, no rerankers, no global search — permanently out of MVP scope.
- Follow existing service style (logger usage, config-object constructor defaults, as
  in ChronologicalLog/PVMIndexer).

## Invariants (code-level)

- `grep -rn "neo4j\|kuzu\|falkordb" server/src/` → no hits.
- `grep -n "\.append(" server/src/services/GraphStore.ts` — writes to graph-edges.jsonl
  only via its own append helper; `grep -rn "graph-edges" server/src` hits only
  GraphStore.ts (single owner of the file).
- No edge deletion: `grep -n "delete\|splice" server/src/services/GraphStore.ts` shows
  no removal of edges from adjacency (Map.delete for node GC is banned too).
- ChronologicalLog.append call sites: GraphStore/GraphIndexer add ZERO — 
  `grep -n "chronologicalLog.append" server/src/services/Graph*.ts` → no hits.
- `coordination.ts` diff = exactly one optional field + comment; no MessageType changes.
- `server/src/index.ts` — no changes to existing endpoints (diff only adds).
- `act-agent/cli/act-cli.ts` — existing `graph task/unverified/conflicts` branches
  unchanged.

## Repro/Evidence (2026-08-08, implementation pass)

- `cd server && npx tsc --noEmit` → clean (exit 0). `npx jest GraphStore` → 1 suite / 22 tests passed. Full suite regression: 6 suites / 76 tests passed.
- All code-level Invariants grep-verified (greps + results in the coordination-log
  progress_report entry by `opus_graph_builder`, and the builder's report).
- Live replay evidence (`/tmp/act-graph-live.log`, throwaway instance on PORT=8091):
  `GraphIndexer replayed 1158 events → 164 edges / 104 nodes` from the real
  `server/data/coordination-log.jsonl`, then exited on the PID-file duplicate guard
  (`index.ts:1689`) because the long-running dev server (pre-graph code) holds the PID.
- **Still owed for done:** live HTTP probe of `/api/graph/node/:key` + `/api/graph/status`
  and `act-agent graph node <key>` — requires restarting the dev server so it picks up
  the new code (PID guard blocks a second instance; data dir/PID path hardcoded at
  `index.ts:1608`). Deviations builder made (code-won): handles both `task_completed`
  (incl. legacy `data.success===false`) and `task_failed`; `task_created` carries the
  whole task as `data` (uses `data.id`); reassignment invalidates the prior assignment edge.
