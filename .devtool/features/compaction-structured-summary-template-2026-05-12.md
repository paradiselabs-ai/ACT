---
id: "compaction-structured-summary-template-2026-05-12"
status: "todo"
priority: "medium"
assignee: "d34d"
epic: "compaction"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["compaction", "context", "structure", "drift"]
order: "a3"
---
# Replace free-form summary prose with a structured template

`agent.go:589` hardcodes the summarize prompt as a single instruction asking for prose. Output is a free-form blob — downstream consumers can't grep "open tasks" or "pending Observer flags" from it. Lossy and unparseable.

**Fix:** structure the summary as headed sections. Two viable shapes:

**Option A — Markdown headings (LLM-friendly):**
```
# Project setup
- refs to ACT.md / AGENT.md / BRIEF.md
# Task ledger
- agent → task → status
# Conversation thread
- user requests + routing decisions
# Tier 1 actions
- Observer flags / Assurance verdicts / QA assemblies
- open anomalies
```

**Option B — YAML/JSON blocks under each heading (machine-parseable):**
Same headings, but each body is a YAML or JSON block. Downstream agents query specific sections (e.g. Observer reads `Tier 1 actions.open_anomalies` directly on next watchdog cycle without re-summarizing).

Recommend Option A for v1 (simple, robust); promote to Option B once anomaly lifecycle (separate kanban) lands and structured retrieval has consumers.

**Files:** `act-agent/internal/llm/agent/agent.go:589` (summarize prompt).

**Success criteria:**
- Summaries contain all four headed sections.
- Each section is non-empty when there's content to fill it.
- Manual: re-run a session post-compact, confirm Planner can reference "open anomalies" from the summary without re-deriving.
