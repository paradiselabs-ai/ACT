---
id: "qa-redesign-phase-a-nomik-agentmd-ingest-2026-04-21"
status: "in-progress"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:55:32.097Z"
completedAt: null
labels: ["QA", "architecture", "integration"]
order: "a00"
---
# QA Redesign — Phase A: Nomik read-only + AGENT.md ingestion

Give QA/Synthesizer read-only Nomik CLI access + AGENT.md ingestion at startup so it can verify integration against the codebase graph instead of trusting one task's self-report. Per-turn prompt stops forbidding tool use; prompt becomes "verify integration against AGENT.md + codebase graph; describe any wiring gap." Still text-only — no edits, no clarification routing.

Depends on Phase 1 discovery (AGENT.md creation path) completing. See `.claude/HANDOFF.md` "QA/Synthesizer gap" section.