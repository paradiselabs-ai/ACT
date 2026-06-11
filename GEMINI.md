# ACT — Agent Entry Point (Gemini CLI)

> **This file is a POINTER, not a knowledge base** (Constitution Art. 3 — one home per fact).
> Do not add project facts here.

## ⛔ Rule zero — NEVER trust single-file sources

**No single doc (including this one, CLAUDE.md, `architecture-flows.json`, `planner-prompts.json`, kanban frontmatter, handoffs) reliably describes the current state of fixes, implementations, bugs, or features.** Acting on a stale "not fixed yet" produces a duplicate fix; acting on a stale "not implemented yet" produces a second, conflicting implementation (it happened with ACP). **Always verify in the codebase itself: grep the symbols, read the vital files.** When a doc and the code disagree, the code is the truth — fix the doc, never the code.

## Read these, in order

1. **`CLAUDE.md`** — the operational brain: architecture, workflows, pitfalls, commands. The single maintained narrative doc (tool-agnostic despite the name).
2. **`docs/constitution/CONSTITUTION.md`** — truth hierarchy, freshness rules, probation protocol.
3. **`docs/constitution/DOC_STANDARDS.md`** — where every kind of doc lives, how to write one, where to put new ones.
4. **`docs/constitution/TASK_TRACKING.md`** — kanban ticket format (Spec / Success Criteria / Constraints / code-level Invariants), bug reporting, where finished work gets published.

## First actions in any session

```bash
./scripts/freshness-check.sh        # which registered artifacts are stale (advisory-only)
tail -c 8000 act-coordination.json  # what other writers did recently (journal, not status)
```

Session continuity: `docs/dev/HANDOFF.md` (older than the branch tip ⇒ advisory only). What shipped: `docs/dev/DEV_LOG.md`. Coordination log is append-only — never edit existing entries.
