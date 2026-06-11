---
title: Update Loops — Artifact Freshness System
status: current
verified_against: bc0673e
owner: project-owner
last_verified: 2026-06-10
---

# Update Loops — Artifact Freshness System

How `architecture-flows.json/html`, `planner-prompts.json/html`, `CLAUDE.md`, `README.md`, kanban, and handoffs stay in sync with the code **without burning usage limits**. Design principle: **deterministic triggers are free and run always; LLM verification runs only when something is both stale and needed.**

---

## 1. The registry: `freshness.json`

[`docs/constitution/freshness.json`](freshness.json) registers every trust-sensitive artifact:

```json
{
  "artifact-name": {
    "paths": ["files that ARE the artifact"],
    "watch": ["code path prefixes the artifact makes claims about"],
    "refresh": "pointer to the refresh procedure",
    "verified_against": "commit the claims were last verified at",
    "status": "fresh | stale",
    "staled_by": "commit that staled it (null when fresh)"
  }
}
```

Reading rule (Constitution Art. 4): `status: stale` ⇒ the artifact is advisory only — orient with it, never cite it as current state.

## 2. Zero-token loops (always on, no LLM)

**Post-commit hook** — `scripts/git-hooks/post-commit` (canonical, versioned) installed into `.git/hooks/` by `scripts/install-hooks.sh`:

- After every commit: diff the commit's changed files against each artifact's `watch` prefixes. Merge commits diff against the **first parent** (`-m --first-parent`) so a feature branch landing on the shared branch stales what it brought in (plain `diff-tree` emits nothing on merges).
- Touched watch path AND the commit didn't update the artifact itself → `status: stale`, `staled_by: <commit>`.
- `verified_against` semantics: the **last commit the artifact was verified at** — it is intentionally kept when an artifact goes stale, because the refresh procedure scopes its re-grep with `git diff <verified_against>..HEAD` (§3 budget rule 2). `status` alone says whether to trust the artifact.
- Pure shell + python3 (stdlib only). Runs in milliseconds. Zero tokens.

**Session-start check** — `scripts/freshness-check.sh`, wired as a `SessionStart` hook in `.claude/settings.json`:

- Prints the stale artifacts (name, staled-by commit, refresh pointer) into session context. A few lines, zero tokens beyond their context cost.
- Also runnable manually any time, and by other editors (Windsurf/Devin) as a rules-dir command.

**Refresh marker** — `scripts/freshness-refresh.sh <artifact>`:

- Sets `status: fresh`, `verified_against: <HEAD>`, clears `staled_by`. Run it **only after actually performing the refresh procedure** — the script records the claim; the agent does the verification.

## 3. LLM refresh procedures (gated, budgeted)

An artifact gets an LLM refresh **only** when: (a) it is `stale` AND about to be relied on, (b) the pre-merge gate fires, or (c) a human asks. Never on wall-clock.

| Artifact | Refresh procedure | Typical cost |
|---|---|---|
| `architecture-flows` (json+html+explainer) | Rebuild per `.claude/architecture-flows-method.md` — grep upfront, sub-agent claims re-grep'd by parent, post-completion verification command | High — only when REST endpoints / role files / protocol steps changed (the method doc's own trigger list) |
| `planner-prompts` (json+html) | Re-extract prompt fragments from `act-agent/internal/llm/prompt/` + autoroute variants from `orchestrator.go`; verify each entry's `source_line` by re-grep; update statuses (RESOLVED entries must cite the live gate) | Medium |
| `claude-md` | Grep-verify the checklist embedded in each section (commands run, paths exist, counts match, "What Works" claims grep'd); fix only confirmed-wrong lines | Medium |
| `readme` | Verify role names, backend list, status line, quickstart commands against code | Low |
| `combined-analysis` | Verify any `[FIXED]` entries whose cited files changed since `verified_against`; re-grep, update entries | Low–Medium |
| `handoff` | Not refreshed — rewritten at session end by the leaving session; older-than-tip ⇒ advisory (Constitution Art. 7) | — |
| kanban | Status honesty pass over tickets whose labels match changed subsystems ([TASK_TRACKING.md](TASK_TRACKING.md) §4) | Low |

**Pre-merge gate (the one mandatory sweep):** before merging any feature branch into the shared branch (`NesTTY` or its integration branches), refresh every artifact staled by that branch's commits. This bounds drift to one branch's lifetime and concentrates LLM spend at one meaningful moment instead of a timer.

**Budget rules (hard):**

1. One artifact per refresh run; no unscoped "refresh everything" sessions.
2. A refresh re-greps only the claims whose watched paths changed (`git diff --name-only <verified_against>..HEAD` scopes it).
3. Full-sweep audits (dual-path style) are reserved for: pre-alpha-tag, post-incident, or explicit request. Not routine.
4. If a refresh would exceed its "typical cost" tier (e.g. architecture-flows after a huge refactor), split by subsystem and do the needed slice first.

## 4. Install / onboarding

```bash
# once per clone (both founders, any new machine):
./scripts/install-hooks.sh        # symlinks scripts/git-hooks/* into .git/hooks/
./scripts/freshness-check.sh      # see what's stale right now
```

`SessionStart` wiring for Claude Code lives in `.claude/settings.json` — **tracked** (gitignore negation `!.claude/settings.json` + `!.claude/hooks/`) with `$CLAUDE_PROJECT_DIR`-relative commands, so a fresh clone gets the check + commit-hygiene hooks automatically. The git-side post-commit staler still needs `./scripts/install-hooks.sh` once per clone. **Dependency: python3 on PATH** — both scripts warn and no-op without it (they never block a commit). Windsurf/Devin: add `scripts/freshness-check.sh` to the tool's session-start rules (`.devin/` may point at it but must not duplicate it).

## 5. Failure modes this system accepts (by design)

- A doc edited without committing won't flip freshness — freshness tracks commits, not saves. Acceptable: the shared branches are what matter. **Corollary:** *uncommitted code* is equally invisible — a `fresh` marker asserts "verified against committed code at that hash," never "matches the working tree." During probation, re-grep regardless.
- A commit that edits an artifact AND a watched code path in one shot is **trusted as a refresh** (the `touched_self` short-circuit) — a one-line doc tweak riding a code rewrite slips through. Accepted: flipping it would false-stale every legitimate combined refresh; the pre-merge gate is the backstop.
- `watch` prefixes are coarse — false-positive stales are cheap (a refresh that finds nothing wrong is fast and ends with `freshness-refresh.sh`); false-negative misses are bounded by the pre-merge gate.
- The hook is local; a founder who never installed it produces commits that don't stale artifacts on their machine. The other machine's hook catches it on pull? No — post-commit fires on local commits only. Mitigation: the pre-merge gate re-derives staleness from `git diff --name-only <verified_against>..HEAD`, which is machine-independent. The hook is a convenience cache; the gate is the source of truth.
