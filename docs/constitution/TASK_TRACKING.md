---
title: Task Tracking Standard
status: current
verified_against: bc0673e
owner: project-owner
last_verified: 2026-06-10
---

# Task Tracking Standard

The kanban (`.devtool/features/`) is the single home for everything task-shaped: bugs, features, specs, hardening items. This doc defines the required shape of a ticket, how the board stays ordered by importance, and where finished work gets published.

---

## 1. Board mechanics (unchanged from existing convention)

- Active tickets: `.devtool/features/*.md` — any status except `done`
- Completed: `.devtool/features/done/*.md` (subdirectory, not sibling) with `completedAt` set
- Frontmatter: `id` (kebab-slug + ISO date), `status` (`backlog|todo|in-progress|review|done`), `priority` (`low|medium|high|critical`), `assignee`, `dueDate`, `created`/`modified`/`completedAt`, `labels[]`, `order` (lexicographic sort key within column)

## 2. Required ticket body — spec'd at the code level

Every non-trivial ticket (anything beyond a typo-class fix) MUST contain these sections. A ticket missing them is `backlog` by definition — it cannot enter `todo` until spec'd.

```markdown
# <Title>

## Spec
What and why. Concrete enough that a cold agent can start without this conversation.

## Success Criteria
Testable outcomes only — files exist at paths X/Y, `go build ./...` clean, function
returns shape Q, endpoint responds with schema R, named test passes. A reviewer must
be able to verdict pass/fail without judgment calls.

## Constraints
Exact shape of the change: which files to touch, which to leave alone. No side-effect
refactors, no speculative abstractions, no "while I'm here" cleanup.

## Invariants (code-level)
Greppable assertions that must hold AFTER the change — e.g. "`scopeHistory` does not
reappear in agent.go", "`parseValidationVerdict` still requires len(CriteriaResults)>0",
"no new import of internal/acp outside app.go". These are what reviewers and update
loops grep for. Behavior must be asserted in code/tests, not prompt-wished.
```

Bug tickets additionally include a **Repro/Evidence** section: exact steps or the captured evidence (debug.log excerpt path, failing command + output, TUI screenshot reference).

## 3. Ordering by importance

- The board is the priority-ordered task log: `priority` field first, then `order` key within a column.
- `critical` = alpha-blocker (breaks the controlled-demo story). `high` = alpha-hardening. `medium` = correctness/polish with a real consequence. `low` = nice-to-have.
- Re-rank at every triage pass; stale ranks are drift like any other.

## 4. Status honesty (anti-trust applies to tickets)

- A ticket's `status` is a claim about the code → it is verifiable and MUST be verified before being trusted (Constitution Art. 2).
- Moving a ticket to `done` requires: the Success Criteria verified against the live tree, the file moved to `done/`, `completedAt` set, **and** a DEV_LOG entry (§5).
- Partial work = `in-progress` with a dated **Status update** section in the body saying exactly what shipped and what didn't (the Fix-23 ticket is the reference example).

## 5. Publication flow — where finished work gets recorded

When a feature/fix lands (merge to the shared branch), publish it in this order, same commit or same PR:

1. **Kanban**: ticket → `done/`, `completedAt` set.
2. **DEV_LOG**: append one line to `docs/dev/DEV_LOG.md`:
   `YYYY-MM-DD · <ticket-id> · <commit(s)> · <one-line what/where, e.g. "per-agent notebooks — ThreadID scoping in internal/llm/agent/">`
3. **Freshness**: if the change touched watched paths, the post-commit hook stales the affected artifacts; refresh them per [UPDATE_LOOPS.md](UPDATE_LOOPS.md) before the merge completes.
4. **CLAUDE.md**: only if the change alters an operational fact stated there (a command, a pitfall, an architecture summary). Point, don't duplicate.

This is the answer to "where do I publish a completed implementation": **kanban `done/` + one DEV_LOG line**; everything else is pointers and freshness updates.

## 6. Bug reporting methodology

Anyone (human, agent, TUI e2e run) reporting a bug:

1. Create a ticket per §2 with Repro/Evidence. Label it (`TUI`, `orchestrator`, `server`, `prompts`, `runner`).
2. Evidence beats description: capture the debug.log lines (`~/.act/**/debug.log`), the failing command output, or the coordination-log excerpt. Quote paths.
3. Severity = the `priority` field per §3's definitions. If it breaks the controlled-demo story, it's `critical`.
4. Do NOT fix in the reporting session unless asked — duplicate-fix risk (Constitution Art. 2). Check the board and `git log -S` for an existing fix first.
