---
id: "block11-eager-precedent-injection-2026-04-21"
status: "backlog"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["planner", "PVM", "block-11"]
order: "b06"
---
# Block 11 — Eager Precedent Injection

Pre-load Planner context with PVM matches on first decomposition turn, instead of hoping Planner remembers to call `act pvm search`.

**Flow**: `parseProjectBrief` fires `queryPrecedentForBrief(brief)` → 5 parallel local PVM queries (one per brief field, sub-100ms, local embeddings). Classify into Strong/Semi/None tier. On next Planner BUILD-mode turn, prepend `precedentBundle` to content, then clear. One-shot.

**Files to create**:
- `act-agent/internal/precedent/thresholds.go` — 4 hardcoded constants calibrated against `all-MiniLM-L6-v2`
- `act-agent/internal/precedent/query.go` — `QueryForBrief(brief) string`
- `act-agent/internal/precedent/format.go` — Strong/Semi/None templates

**Modify**: `act-agent/internal/act/client.go` (`PVMSearchTopK`), `orchestrator.go` (precedentBundle field, async fire on brief parse, one-shot prepend).

~150-200 lines, no new dependencies. Ships in Phase 2 because silent on cold corpus. See BUILD_ORDER.md Block 11 + FUTURE_VISION.md "Eager Precedent Injection".
