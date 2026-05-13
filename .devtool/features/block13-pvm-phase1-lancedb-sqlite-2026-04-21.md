---
id: "block13-pvm-phase1-lancedb-sqlite-2026-04-21"
status: "backlog"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["persistence", "PVM", "block-13"]
order: "b08"
---
# Block 13 — PVM Phase 1 (LanceDB + SQLite)

Replace in-memory `LocalEmbeddingVectorStore` with persistent structured storage.

**Stack**: `@lancedb/lancedb` + `apache-arrow` + `better-sqlite3`. Three npm installs. No Docker.

**Tables**:
- SQLite `events` — ChronLog replacement, FTS5 keyword search, `flushed_to_lance`/`flushed_to_graph` flags
- SQLite `event_edges` — causal graph via recursive CTEs (no separate graph DB in Phase 1)
- LanceDB `pvm_events` — semantic index over **structured** fields (role, tech_stack, success, file_paths)
- LanceDB `pvm_thought_chains` — Planner reasoning sequences
- LanceDB `pvm_agent_profiles` — per-role/tech-stack skill profiles, incremental

**Pipelines**: 50-event batches on 5s tick, SQLite WAL mode. Restart replay = `SELECT WHERE flushed_to_lance = 0`.

Embedding pipeline unchanged (`@xenova/transformers` + `all-MiniLM-L6-v2`).

Phase 2 FalkorDB upgrade deferred. Depends on Block 12. See BUILD_ORDER.md Block 13 + FUTURE_VISION.md "PVM: Persistent Multi-Modal Agent Memory".
