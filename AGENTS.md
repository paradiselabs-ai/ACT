# ACT — Agent Entry Point (Windsurf / Devin Desktop / Codex / any AGENTS.md-reading tool)

> **This file is a POINTER, not a knowledge base** (Constitution Art. 3 — one home per fact).
> It was once a full copy of CLAUDE.md; the copies drifted thousands of bytes apart and told
> different stories about the same code. Do not add project facts here — they go in their
> canonical homes below.

## ⛔ Rule zero — NEVER trust single-file sources

**No single doc (including this one, CLAUDE.md, `architecture-flows.json`, `planner-prompts.json`, kanban frontmatter, handoffs) reliably describes the current state of fixes, implementations, bugs, or features.** Acting on a stale "not fixed yet" produces a duplicate fix; acting on a stale "not implemented yet" produces a second, conflicting implementation (it happened with ACP). **Always verify in the codebase itself: grep the symbols, read the vital files.** When a doc and the code disagree, the code is the truth — fix the doc, never the code.

## Read these, in order

1. **`CLAUDE.md`** — the operational brain: architecture, workflows, pitfalls, commands. The single maintained narrative doc (tool-agnostic despite the name).
2. **`docs/constitution/CONSTITUTION.md`** — truth hierarchy, freshness rules, probation protocol.
3. **`docs/constitution/DOC_STANDARDS.md`** — where every kind of doc lives, how to write one, where to put new ones.
4. **`docs/constitution/TASK_TRACKING.md`** — kanban ticket format (Spec / Success Criteria / Constraints / code-level Invariants), bug reporting, where finished work gets published.

## First actions in any session

```bash
./scripts/install-hooks.sh          # once per clone — installs the freshness post-commit hook
./scripts/freshness-check.sh        # which registered artifacts are stale (advisory-only)
tail -c 8000 act-coordination.json  # what other writers did recently (journal, not status)
```

Session continuity: `docs/dev/HANDOFF.md` (older than the branch tip ⇒ advisory only). What shipped: `docs/dev/DEV_LOG.md`. Coordination log is append-only — never edit existing entries.

> Note for ACT's own runtime: `AGENTS.md` files in **target project directories** are generated
> by the Planner as the project brief and auto-injected into agent context. This repo-root file
> is different — it's the editor entry point for the ACT repo itself.
