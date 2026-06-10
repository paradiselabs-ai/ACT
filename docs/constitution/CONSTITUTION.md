---
title: ACT Project Constitution
status: current
verified_against: bc0673e
owner: project-owner
last_verified: 2026-06-10
---

# ACT Project Constitution

How this project tracks its own state. Binding for every actor that writes to this repo: both founders, Claude Code, Windsurf / Devin Desktop, Gemini CLI, ACT's own agents, and any future tool. Companion docs: [DOC_STANDARDS.md](DOC_STANDARDS.md), [TASK_TRACKING.md](TASK_TRACKING.md), [UPDATE_LOOPS.md](UPDATE_LOOPS.md).

---

## Article 1 — The truth hierarchy

When sources disagree, higher beats lower. No exceptions.

1. **The code** (what's in the working tree on the current branch)
2. **Tests** (what's asserted and green)
3. **Git history** (what actually landed, when, by whom)
4. **Generated artifacts** (`architecture-flows.json/html`, `planner-prompts.json/html`) — only as fresh as their `verified_against` commit in `freshness.json`
5. **Status docs** (kanban tickets, audit status files like `combined-analysis.md`)
6. **Narrative docs** (`CLAUDE.md`, `README.md`, handoffs, architecture prose)

A document is never evidence about the code. A document citing `file:line` is never evidence — line numbers drift within commits on active branches. Evidence is a grep or file read you performed against the live tree.

## Article 2 — The anti-trust rule

**Never trust a single-file source for the current state of fixes, implementations, bugs, or features.** Before acting on any claim that affects your next step (e.g. "bug X is unfixed", "feature Y doesn't exist"), verify it directly in the codebase: grep for the symbols, read the vital files.

The two catastrophic failure modes this prevents:

- **Duplicate fix** — assuming a bug unfixed when it's fixed; the second "fix" lands on top of the first and breaks it.
- **Duplicate implementation** — assuming a feature absent when it exists (e.g. ACP backends); two half-built conflicting systems result.

Corollary (proven in the Round-6 audit): **convergence ≠ correctness.** Two independent analyses agreeing on a stale source are still wrong. Any reviewer/comparator must re-grep every convergent claim.

**When a doc and the code disagree: the code is the truth. Fix or delete the doc statement. Never change the code to make a doc statement true** (unless the doc is an approved spec for unbuilt work — and then say so explicitly).

## Article 3 — One home per fact

Every category of project information has exactly one canonical home (the registry is in [DOC_STANDARDS.md](DOC_STANDARDS.md) §2). Everything else that mentions it is a pointer, not a copy.

- `AGENTS.md` and `GEMINI.md` are **pointers** to `CLAUDE.md` plus the anti-trust banner. They must never grow independent content. (They were once full copies; they drifted ~5–14KB apart. That is the disease this article cures.)
- A fact stated in two homes will drift. If you find a duplicated fact, replace one copy with a pointer in the same commit.

## Article 4 — The artifact registry and freshness

Trust-sensitive artifacts are registered in [`freshness.json`](freshness.json) with: path, the code paths they make claims about (`watch`), the commit they were last verified against, and a `status` (`fresh` | `stale`).

- A **zero-token git post-commit hook** flips an artifact to `stale` when a commit touches its watched paths ([UPDATE_LOOPS.md](UPDATE_LOOPS.md) §2).
- A stale artifact is **advisory only** — readable for orientation, never citable as current state.
- Refreshing an artifact means: re-verify its claims against the live tree (grep/read, per its refresh procedure), update content, set `verified_against` to the current commit, set `status: fresh`.

## Article 5 — Update loops have budgets

Loops keep artifacts fresh **without burning usage limits**. Hard rules (detail in [UPDATE_LOOPS.md](UPDATE_LOOPS.md)):

- Deterministic triggers are free (shell hooks); they may run on every commit.
- LLM-powered refresh runs **only** on: a stale marker + the artifact being needed, a pre-merge gate, or explicit invocation. Never on wall-clock alone.
- One artifact per refresh run. No "refresh everything" sweeps outside pre-merge/pre-PR gates.

## Article 6 — Probation protocol

After any period of known drift (like the one that prompted this constitution), the project operates in **anti-trust probation**: every actor must code-verify claims before acting on them, even from "fresh" artifacts.

Probation lifts only after **3 consecutive trustworthy cycles**, where a cycle is: a working session that (a) ran the freshness check at start, (b) updated the artifacts its commits staled, and (c) a later session's spot-greps found zero FALSE statements in those updates. Track cycles in `docs/dev/DEV_LOG.md`. Anyone may reset the count by documenting a found FALSE statement.

## Article 7 — Handoffs

- Handoffs live at `docs/dev/HANDOFF.md` (tracked — both founders and any machine can read them). One current handoff; previous content is overwritten, with durable facts promoted to their canonical homes first.
- A handoff is stamped with the commit it describes. **A handoff older than the branch tip is advisory only** — re-verify before acting on its claims.
- Legacy locations (`F-handoff.md` at root, `.claude/HANDOFF.md`) are deprecated; if found, treat as advisory and migrate anything still true.

## Article 8 — Coordination log

`act-coordination.json` remains append-only, as before. It is a **journal** (what happened, when), not a status source — never read it as "current state"; read it to discover what other writers did, then verify in code.

## Article 9 — Amendments

Constitution changes go through the boundary-review path of the team workflow: PR + 24h for the other founder to ack or push back. Mechanical fixes (typos, broken links) are exempt.
