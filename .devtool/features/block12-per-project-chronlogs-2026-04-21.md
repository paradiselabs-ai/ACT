---
id: "block12-per-project-chronlogs-2026-04-21"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:54:28.096Z"
completedAt: null
labels: ["persistence", "chronlog", "block-12"]
order: "b07"
---
# Block 12 — Per-Project ChronLogs

Replace global `server/data/coordination-log.jsonl` with per-project: `server/data/<project>-coordination.jsonl`.

**Why**: global log interleaves unrelated projects, grows unboundedly, fragile PVM rebuilds. Per-project logs are the right training-data unit for a future purpose-built ACT model.

**Modify**:

- `server/src/services/ChronologicalLog.ts` — accept `projectName`
- `server/src/index.ts` — one ChronLog per active project
- `server/src/services/PVMIndexer.ts` — glob `data/*-coordination.jsonl` on startup
- `act reset` — clears in-memory only

**New CLI**: `act project delete <name>` — removes from memory AND deletes log file (destructive action `act reset` pretends to be).

Unblocks Block 13. See BUILD_ORDER.md Block 12 + FUTURE_VISION.md "Per-Project ChronLogs".