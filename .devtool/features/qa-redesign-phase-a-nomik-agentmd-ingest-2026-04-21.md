---
id: "qa-redesign-phase-a-nomik-agentmd-ingest-2026-04-21"
status: "todo"
priority: "high"
assignee: null
epic: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-06-06T05:08:25.296Z"
completedAt: null
labels: ["QA", "architecture", "integration"]
order: "a00"
---
# QA Redesign — Phase A: Grep +  AGENT.md ingestion

Give QA/Synthesizer Grep + AGENT.md ingestion at startup so it can verify integration against the codebase instead of trusting one task's self-report. Per-turn prompt stops forbidding tool use; prompt becomes "verify integration against AGENT.md + codebase; describe any wiring gap." Still text-only — no edits, no clarification routing.

Depends on Phase 1 discovery (AGENT.md creation path) completing. See `.claude/HANDOFF.md` "QA/Synthesizer gap" section.