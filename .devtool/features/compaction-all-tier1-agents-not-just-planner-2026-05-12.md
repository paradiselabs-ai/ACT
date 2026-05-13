---
id: "compaction-all-tier1-agents-not-just-planner-2026-05-12"
status: "todo"
priority: "high"
assignee: "d34d"
epic: "compaction"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["compaction", "context", "tier1"]
order: "a1"
---
# Auto-compaction only fires for Planner — extend to all four Tier 1 agents

`tui.go:324-340` dispatches `a.app.Agents["planner"].Summarize(...)` only. Observer / Assurance / QA each accumulate their own message threads through the same `app.Agents` map but are never compacted. In long sessions (validation pipeline turns, watchdog re-triggers, synthesis cycles) their threads grow unbounded.

**Fix:** sweep `startCompactSessionMsg` across all four Tier 1 roles. Trigger formula stays per-agent (each `session.{Prompt,Completion}Tokens` is per-agent already). Either:
- Fire `Summarize` on whichever agent crossed the threshold this turn, OR
- Fire `Summarize` on all four when ANY crosses threshold (more aggressive, simpler).

Recommend the per-agent-on-threshold variant — minimal LLM cost, narrower scope.

**Files:** `act-agent/internal/tui/tui.go:354-374` (auto-trigger), `act-agent/internal/tui/tui.go:324-340` (dispatch handler).

**Depends on:** `compaction-summarizer-fallback-default-install-2026-05-12` (otherwise extending to more agents extends a dead path).

**Success criteria:**
- Observer/Assurance/QA each trigger their own Summarize independently.
- Each gets its own `SummaryMessageID` on its own session row.
- Build + vet clean.
- Manual test: long-running session with active Assurance turns — verify Assurance compacts when its own tokens cross threshold.
