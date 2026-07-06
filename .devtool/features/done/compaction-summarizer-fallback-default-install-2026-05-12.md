---
id: "compaction-summarizer-fallback-default-install-2026-05-12"
status: "done"
priority: "critical"
assignee: "d34d"
epic: "compaction"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: "2026-07-06T00:00:00.000Z"
labels: ["compaction", "context", "config", "default-install"]
order: "a0"
---
# Compaction silently no-ops in default installs — fall back summarizer to Planner

`agent.go:95` gates `summarizeProvider` on `cfg.Agents[config.AgentSummarizer]` existing. The default `.act.example.json` (incl. PR3) does NOT configure `agents.summarizer`. Result: `summarizeProvider == nil`, `Summarize()` returns `"summarize provider not available"`, the auto-trigger fires but compaction never actually runs. PR3's restored `AutoCompactTokens` threshold is correct in isolation but routes into a dead path.

**Fix:** when `agents.summarizer` is absent in `~/.act.json`, fall back `summarizeProvider` to the Planner's provider/model. Planner is always configured (it's the lead Tier 1, ACT can't run without it). Keep the override path for users who want a dedicated cheap summarizer.

**Files:** `act-agent/internal/llm/agent/agent.go:95` (constructor branch).

**Success criteria:**
- Default `.act.example.json` works for compaction end-to-end (no `summarizer` block needed).
- If `agents.summarizer` is explicitly configured, that takes precedence.
- Build + vet clean.
- Manual test: set `AutoCompact: true`, `AutoCompactTokens: 5000`, run a session past threshold, verify Summarize fires and writes a `SummaryMessageID`.

## Closed 2026-07-06

Shipped in the alpha worktree pass: `NewAgent` falls back `summarizeProvider` to the Planner's provider/model when `agents.summarizer` is absent (`createProviderFromConfig` extracted so the summarizer keeps its own prompt identity). Explicit `agents.summarizer` still takes precedence. Build + vet clean. Manual AutoCompactTokens live test still owed in the TUI e2e matrix.
