---
title: Documentation Standards
status: current
verified_against: bc0673e
owner: project-owner
last_verified: 2026-06-10
---

# Documentation Standards

Where docs live, how a doc page is structured, how docs are pruned, and how every actor (founders, Claude Code, Windsurf / Devin Desktop, Gemini CLI, ACT agents) finds and writes them.

---

## 1. Directory map

| Location | Purpose | Tracked? | Audience |
|---|---|---|---|
| `README.md` | Public front door — what ACT is, status, quickstart | ✅ | GitHub / DeepWiki visitors |
| `CLAUDE.md` | Operational brain for AI sessions: architecture facts, pitfalls, workflows | ✅ | All AI editors (via pointers) + founders |
| `AGENTS.md`, `GEMINI.md` | **Pointers only** → CLAUDE.md + anti-trust banner. No independent content. | ✅ | Windsurf / Devin Desktop / Codex; Gemini CLI |
| `docs/constitution/` | How the project tracks itself (this directory) + `freshness.json` | ✅ | Everyone |
| `docs/dev/` | Internal dev-state: `HANDOFF.md`, `DEV_LOG.md`, `ROADMAP.md`, `NOTES.md` | ✅ | Founders + AI sessions |
| `docs/audits/` | Verification + audit outputs (handoff verifications, prompt-audit rounds) | ✅ | Founders + AI sessions |
| `docs/planner-prompt-audit/` | Planner prompt-audit line (rounds, `planner-prompts.json/html`, `combined-analysis.md`) | ✅ (after gitignore fix) | Founders + AI sessions |
| `docs/` (root) | Public-facing architecture/concept docs (`ARCHITECTURE_PATTERNS.md`, diagrams) | ✅ | Public + internal |
| `docs/_archive/` | Superseded docs (tombstoned, see §4) | ✅ | History |
| `docs/Vault/` | Personal Obsidian vault. **Advisory scratch only** — nothing load-bearing may live ONLY here. | ❌ gitignored | Project owner's machine |
| `.devtool/features/` | Kanban (see [TASK_TRACKING.md](TASK_TRACKING.md)) | ✅ | Everyone |
| Repo root `architecture-flows.{json,html}`, `flows-explainer.html` | Generated codebase maps (rebuild method: `.claude/architecture-flows-method.md`) | ✅ | Everyone |

**Decision table — where does a new doc go?**

- Explains ACT to outsiders → `docs/` root (or README section)
- Records dev state, plans, ideas, session continuity → `docs/dev/`
- Output of a verification/audit run → `docs/audits/` (or the existing audit line's directory)
- Governs process/meta → `docs/constitution/`
- Personal thinking not ready to be load-bearing → `docs/Vault/` (and promote it when it becomes load-bearing)
- Task-shaped (bug, feature, spec) → not a doc; it's a kanban ticket (`.devtool/features/`)

## 2. Canonical homes (one home per fact — Constitution Art. 3)

| Fact category | Canonical home |
|---|---|
| Current architecture behavior | The code; summarized in `CLAUDE.md`, mapped in `architecture-flows.json` |
| What's broken / planned / in flight | `.devtool/features/` kanban |
| What shipped, when, where it lives | `docs/dev/DEV_LOG.md` (one line per landing) + git history |
| Session continuity | `docs/dev/HANDOFF.md` |
| Future ideas / roadmap | `docs/dev/ROADMAP.md` (ordered), `docs/dev/NOTES.md` (unordered) |
| Prompt-audit status | `docs/planner-prompt-audit/combined-analysis.md` |
| Process / meta rules | `docs/constitution/` |
| Artifact freshness | `docs/constitution/freshness.json` |

## 3. Doc page template

Every tracked doc in `docs/` starts with frontmatter:

```markdown
---
title: <Human title>
status: draft | current | superseded
verified_against: <short commit hash the claims were checked against>
owner: project-owner | cofounder | generated
last_verified: YYYY-MM-DD
supersedes: <path, optional>
---
```

**Exception — audit working files:** intermediate fan-out outputs (`*-path-a-sub*.md`, `*-path-b-self.md`, `*-synthesis.md`, working comparison copies, `*.raw.json` dumps) are working notes, exempt from frontmatter; only the canonical deliverable of a run (the report named in its workflow) carries it.

Body rules:

- Lead with what the doc is for and who should read it (1–2 sentences).
- Claims about code cite `file:line` **and** carry the frontmatter `verified_against` hash — a reader can tell instantly how stale a citation might be.
- Link related docs with relative markdown links; never restate their content.
- No "TODO: fill in later" sections in `status: current` docs — a doc with holes is `draft`.

## 4. Pruning policy

- A doc contradicted by the code gets **fixed or cut the moment the contradiction is found** — not flagged for later. (Cutting a wrong sentence is always safe; leaving it is never safe.)
- Superseded docs move to `docs/_archive/` with a one-line tombstone prepended: `> Superseded by <link> on YYYY-MM-DD.` Delete from `_archive/` only after both founders ack.
- The freshness sweep (pre-merge gate, [UPDATE_LOOPS.md](UPDATE_LOOPS.md) §3) includes a prune pass over any doc the branch touched.

## 5. Entry points for AI editors

Every AI tool entering this repo must land on the same rules:

- **Claude Code** reads `CLAUDE.md` natively.
- **Windsurf / Devin Desktop / Codex-style editors** read `AGENTS.md` → which is a pointer: anti-trust banner + "read `CLAUDE.md` and `docs/constitution/CONSTITUTION.md` before acting."
- **Gemini CLI** reads `GEMINI.md` → same pointer.
- **`.devin/` and `.windsurf/` rule-dirs** may exist for tool-specific config but must not duplicate project facts — pointer rule applies.
- First action in any session: run the freshness check ([UPDATE_LOOPS.md](UPDATE_LOOPS.md) §2) and read the last ~50 lines of `act-coordination.json`.

## 6. Git tracking rules

`.gitignore` defaults all `*.md` to ignored (personal-scratch policy) with explicit negations. The negation set MUST cover every tracked location in §1. Known footguns (fixed on `feat/cleanup-constitution`):

- `*-prompt*` (line ~19) used to swallow the whole `docs/planner-prompt-audit/` directory — audit outputs were invisible to git.
- Handoffs at repo root (`F-handoff.md`) matched the `*.md` default-ignore — session handoffs existed on one machine only. Handoffs now live tracked in `docs/dev/`.
- A stray trailing `CLAUDE.md` line re-ignored the main context file.

Rule: **before creating any new doc location, verify it's actually trackable**: `git check-ignore -v <path>` must come back empty.
