---
id: "qa-redesign-phase-a-nomik-agentmd-ingest-2026-04-21"
status: "todo"
priority: "high"
assignee: null
epic: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-06-11T18:00:00.000Z"
completedAt: null
labels: ["QA", "architecture", "integration"]
order: "a00"
---
# QA Redesign — Phase A: AGENTS.md ingestion at startup (RE-SCOPED — Grep half already shipped)

> **⚠ RE-SCOPED 2026-06-11 (anti-trust verification).** The original body asked for two things;
> one already exists. **Do NOT re-implement the tool grant:** QA already has `act_cli + view + grep`
> via `agent.Tier1ToolsForRole` (`internal/llm/agent/tools.go`), and the QA prompt actively
> *instructs* tool use ("Use view and grep to read validated outputs directly",
> `prompt/qa_synthesizer.go`). Re-granting tools or rewriting that prompt clause is duplicate work.

## Spec (remaining)
Give QA/Synthesizer **AGENTS.md ingestion at startup** so it verifies integration against the
project brief + codebase instead of trusting one task's self-report. Still text-only — no edits.

## Success Criteria
- On QA activation, the target project's AGENTS.md content is available in its context (grep the
  ingestion site in code, not just prompt text).
- QA synthesis output references brief constraints when describing wiring gaps (observable in a
  live run's synthesis message).

## Constraints
- No tool-set changes (already correct). No prompt rewrite beyond the ingestion instruction.
- No clarification-routing changes — `NEED_CLARIFICATION` routing already exists
  (`maybeRouteQAClarification`, orchestrator.go); Phase C must build on it, not re-create it.

## Invariants (code-level)
- `Tier1ToolsForRole("qa_synthesizer")` continues to return act_cli+view+grep (no bash).
- One ingestion path only; no second AGENTS.md reader added alongside the context-paths mechanism.

Stale dependency pointer removed: `.claude/HANDOFF.md` is a deprecated handoff location
(Constitution Art. 7) — current state lives in `docs/dev/` + `docs/audits/`.
