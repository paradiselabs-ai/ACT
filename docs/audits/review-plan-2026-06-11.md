---
title: Adversarial Review — Constitution + Freshness System
status: current
verified_against: e9fa799
owner: generated
last_verified: 2026-06-11
---

> ## ✅ DISPOSITION (orchestrator, 2026-06-11 — every finding independently re-verified before approval)
> - **C1 APPROVED, option (a)+**: `.claude/settings.json` AND `.claude/hooks/` now tracked (gitignore negations) with `$CLAUDE_PROJECT_DIR`-relative commands — the full Claude wiring bootstraps on a fresh clone. README "first clone" block + AGENTS/GEMINI first-actions gained `install-hooks.sh`. UPDATE_LOOPS §4 wording fixed.
> - **C2 APPROVED + REFINED**: the reviewer's `-m --first-parent` fix was applied AND found insufficient — git fires `post-merge`, not `post-commit`, after merges. Added `scripts/git-hooks/post-merge` wrapper. End-to-end merge test now stales correctly.
> - **H1 APPROVED**: ROADMAP PVM item rewritten (SelfImprovementEngine qualified as unverified).
> - **H2 APPROVED**: block6 invariant restated word-bounded + no-backend.go form.
> - **H3 APPROVED as decision (a)**: the antigravity author commits the work (already the ticket's first Success Criterion); no registry override field. Probation covers the gap; UPDATE_LOOPS §5 gained the uncommitted-drift corollary.
> - **M1/M2 APPROVED doc-only**: combined-commit trust + `verified_against` semantics documented (UPDATE_LOOPS §2/§5).
> - **M3 APPROVED**: DOC_STANDARDS §3 carve-out for audit working files.
> - **M4 APPROVED**: `docs/dev/HANDOFF.md` stub written — all pointers resolve.
> - **L1 APPROVED**: python3 guards in both scripts + dependency documented.
> - **L2 REJECTED (moot)**: MEMORY.md was already corrected before this review ran; the only remaining grep hit is the new note *stating* the symlink is gone.
> - **L3 APPROVED**: dead `!nestty/*.md` / `!.windsurf/**/*.md` negations dropped.

# Adversarial Review — Constitution + Freshness System

Read-only adversarial audit of the cleanup effort's constitution docs, freshness
automation, reconciliation commit `d7ef63b`, `docs/dev/` seeds, and audit reports.
Every claim below was re-grepped against the live working tree (branch
`feat/cleanup-constitution`, tip `e9fa799`) — docs were treated as claims, never
evidence. Findings are ranked; each carries live evidence (`file:line` I grepped),
an exact proposed fix, and a fix-class tag (doc-only / script-only / decision).

> This plan is `draft` and requires human/orchestrator approval before any fix lands.
> The reviewer wrote ONLY this file; nothing else was modified.

---

## CRITICAL

### C1 — The entire freshness automation cannot bootstrap on a second machine (Kareem / fresh clone)
**What's wrong:** Two compounding failures mean the freshness system silently does
nothing for anyone who isn't the project owner on this exact machine:

1. **`.claude/` is gitignored** (`.gitignore:120: .claude/`). The `SessionStart` hook
   that runs `freshness-check.sh` lives in `.claude/settings.json`, which is therefore
   never cloned. `git check-ignore -q .claude/settings.json` → exit 0 (IGNORED);
   `git ls-files .claude/` → empty (nothing tracked).
2. Even if it were tracked, the command is **hardcoded to an absolute path**:
   `.claude/settings.json:45` → `"command": "/Users/user/Documents/Developer/dev/AI/act/scripts/freshness-check.sh"`.
   That path does not exist on Kareem's machine.
3. **`install-hooks.sh` is discoverable from exactly one place** — `UPDATE_LOOPS.md §4`.
   It is referenced in NO entry-point a fresh clone actually lands on:
   `grep -rln "install-hooks" README.md AGENTS.md GEMINI.md CLAUDE.md` → zero hits.
   So the post-commit staler never gets symlinked into `.git/hooks/` on a second clone.

Net effect: the post-commit staler, the session-start check, and the whole "deterministic
triggers are free and always-on" promise (Constitution Art. 5, UPDATE_LOOPS §2) are
inert for the cofounder. DOC_STANDARDS §5 asserts "`SessionStart` wiring … already
configured in-repo" — false; it is configured in a gitignored, machine-pinned file.

**Live evidence:**
- `.gitignore:120` = `.claude/`
- `.claude/settings.json:45` hardcoded absolute path
- `git ls-files .claude/` → empty
- `grep -rln "install-hooks" README.md AGENTS.md GEMINI.md CLAUDE.md` → empty

**Proposed fix (decision + doc + script):**
- **Decision needed:** how does the SessionStart check reach other machines? Options:
  (a) commit a tracked `.claude/settings.json` with a **repo-relative** command
  (`"./scripts/freshness-check.sh"` or `"$CLAUDE_PROJECT_DIR/scripts/freshness-check.sh"`)
  and add a `!.claude/settings.json` negation to `.gitignore`; (b) accept Claude-Code can't
  be wired cross-machine and make the README onboarding block the source of truth.
- **README onboarding block (doc):** add a "First clone" section to `README.md` (and link
  it from `AGENTS.md`/`GEMINI.md` first-actions) that runs `./scripts/install-hooks.sh`
  then `./scripts/freshness-check.sh`. Until install-hooks is in an entry point, the hook
  never installs.
- **DOC_STANDARDS §5 statement (doc-only):** change "already configured in-repo" to state
  the wiring is machine-local/gitignored and the portable path is `install-hooks.sh` +
  the editor's own session rules.

---

### C2 — Merge commits stale NOTHING; the post-commit hook is a no-op on the one commit type that matters most
**What's wrong:** `scripts/git-hooks/post-commit:11` runs
`git diff-tree --no-commit-id --name-only -r HEAD`. On a **merge commit**, that command
emits no file list (it needs `-m` or `--cc` to show per-parent diffs). So when a feature
branch that touched watched paths merges into the shared branch, the post-commit hook on
the merge commit stales **zero** artifacts. UPDATE_LOOPS §5 ("failure modes this system
accepts") lists the "doc edited without committing" and "hook is local" cases but does NOT
list merge-commit blindness. The claimed mitigation (pre-merge gate, §3) only works if a
human/agent *remembers* to run it — the automation provides no backstop at merge time, and
the branch protocol (squash-merge to main) doesn't fully cover NesTTY integration merges.

**Live evidence (reproduced in a throwaway repo):**
- Created `feature` branch, edited `act-agent/internal/app/feature.go` (a `claude-md`
  watch path), refreshed `claude-md` to `fresh`, then `git merge --no-ff feature`.
  Post-merge: `claude-md.status` = `fresh` (NOT staled). `git diff-tree --no-commit-id
  --name-only -r HEAD` on the merge commit printed nothing.
- `scripts/git-hooks/post-commit:11` is the offending invocation.

**Proposed fix (script-only + doc):**
- **Script:** make the diff merge-aware. Replace the `diff-tree` line with a form that
  walks merge parents, e.g. `git diff-tree --no-commit-id --name-only -r -m HEAD` (or
  `--cc`), or special-case `git rev-parse HEAD^2` existence to diff `HEAD^1..HEAD`. Verify
  the chosen form still emits a flat path list the python consumer can read line-by-line.
- **Doc:** add merge-commit handling (or its explicit non-handling) to UPDATE_LOOPS §5.

---

## HIGH

### H1 — ROADMAP.md contradicts the corrected CLAUDE.md pitfall 7 (PVM analytics real-vs-placeholder)
**What's wrong:** `d7ef63b` rewrote CLAUDE.md pitfall 7 to "LocalEmbeddingVectorStore …
analytics are REAL too" — and that rewrite is **correct** (verified below). But the
reconciliation left `docs/dev/ROADMAP.md:33` stating the **opposite**: "PVM analytics layer
… `getAgentProfile`/`compareAgents`/`getAgentSynergy`/`SelfImprovementEngine` are
placeholder." This is a one-home-per-fact violation (Constitution Art. 3) AND a now-false
statement that the freshness gate should have caught (ROADMAP watches nothing in
`freshness.json`, so it can't).

**Live evidence:**
- `server/src/index.ts:42` → `const vectorStore = new LocalEmbeddingVectorStore();`
  (Local is the ACTIVE store).
- `server/src/services/LocalEmbeddingVectorStore.ts:273` `getAgentProfile` computes from
  `this.lookupTaskOutcomes(agentId)` — no `Math.random`, no placeholder
  (`grep -n "Math.random" LocalEmbeddingVectorStore.ts` → empty).
- `Math.random` placeholders survive ONLY in inactive stores:
  `MockVectorStore.ts:194` and `QdrantVectorStore.ts:265`.
- `docs/dev/ROADMAP.md:33` still says analytics are placeholder. CLAUDE.md:351 now says real.

**Proposed fix (doc-only):** Edit `ROADMAP.md:33`. Remove the "Later → PVM analytics
de-placeholdering" item or rewrite it to match the verified truth, e.g.: "PVM analytics on
the active Local store are real (compute from task-outcome events); only statistical quality
under live data is unverified. The `0.85 + Math.random()` placeholders remain only in the
inactive Mock/Qdrant stores." Note: SelfImprovementEngine was NOT separately verified by
this review — if the rewrite keeps mentioning it, qualify it as unverified.

### H2 — block6 ticket's "greppable" invariant is FALSE as literally written
**What's wrong:** `TASK_TRACKING.md §2` requires Invariants be "greppable assertions … what
reviewers and update loops grep for." The block6 ticket states the invariant
`grep -rn "AgentBackend" act-agent/internal/` **stays empty**. It does not — that grep
returns 6 hits because `AgentBackend` is a substring of the real, live function
`WriteAgentBackend`. A reviewer mechanically running the stated invariant gets a FALSE
verdict on a ticket that is otherwise fine. This is precisely the "invariant must actually
be true when grepped" failure the audit mandate targets.

**Live evidence:**
- `.devtool/features/block6-acp-cli-backend-2026-04-21.md` Invariants section:
  `grep -rn "AgentBackend" act-agent/internal/" stays empty`.
- Live: `grep -rn "AgentBackend" act-agent/internal/` → `slash.go:201,226,272,286`,
  `config/writer.go:10,16` (all `WriteAgentBackend`). NOT empty.

**Proposed fix (doc-only):** Make the grep word-bounded / exact. Replace with the intent:
`grep -rnE "\\bAgentBackend\\b\\s*(struct|interface|=)" act-agent/internal/` stays empty
(no `AgentBackend` *type*), or better, restate as the real invariant the antigravity ticket
already uses correctly: "`acp.NewACPAgent` remains the only external-backend constructor; no
`internal/llm/backend.go` exists." (Cross-check: that file genuinely never existed —
`git log --all -- act-agent/internal/llm/backend.go` → empty.)

### H3 — `combined-analysis` is marked `fresh@d7ef63b` but its watch list cannot see the working-tree drift it documents
**What's wrong:** `freshness.json` marks `combined-analysis` (and `claude-md`, `readme`)
`fresh` at `d7ef63b`. But the audit reports and the antigravity kanban ticket all establish
that the **working tree is ahead of any commit** — the antigravity/agy backends are
UNCOMMITTED (`git status` shows 9 modified + 3 untracked under `act-agent/`). The freshness
model keys entirely on commits (UPDATE_LOOPS §5 accepts this), so a `fresh` marker here
asserts "verified against committed code at d7ef63b" while the live tree it'll be read
against contains undocumented backend changes. A reader who trusts `fresh` and skips the
re-grep (which probation is supposed to prevent, Constitution Art. 6) gets a stale picture.
The drift note in the handoff audit acknowledges this for ONE artifact; the freshness
registry does not reflect it at all.

**Live evidence:**
- `freshness.json` claude-md/readme/combined-analysis `status: fresh`, `verified_against: d7ef63b`.
- `git status --short` → uncommitted `app.go, slash.go, acp/*, config.go, agent.go,
  prompt.go, swarm_roles.go, act-cli.ts` + untracked `antigravity_cli.go, agy-acp.mjs`.
- `docs/audits/handoff-verification-2026-06-10.md` drift note (added `d7ef63b`) documents
  exactly this divergence.

**Proposed fix (decision):** This is the freshness model's accepted blind spot (commits, not
saves) colliding with a real uncommitted hazard. Two paths: (a) the antigravity author
commits the work (the antigravity ticket's first Success Criterion) — then the post-commit
hook stales these three artifacts and the picture self-heals; or (b) until then, add a
`working_tree_ahead` advisory note to `freshness.json`'s `_doc` or to the three artifacts'
`staled_by`-adjacent field. Recommend (a) — driving the commit is the real fix; the
freshness registry should not grow a manual override field.

---

## MEDIUM

### M1 — `touched_self` short-circuit over-trusts a combined artifact+watch commit
**What's wrong:** `post-commit:27-29` treats any commit that edits the artifact file itself
as a refresh (`touched_self` ⇒ skip staling), even when the SAME commit also changes a
watched code path. Editing one line of `README.md` in a commit that also rewrites
`act-agent/internal/config/config.go` leaves `readme` `fresh` — but a one-line doc tweak is
no guarantee the editor re-verified the config change. This is an optimistic assumption that
UPDATE_LOOPS §5 does not call out (it discusses coarse watch prefixes generally, not this
specific combined-commit hole).

**Live evidence (reproduced):** Commit touching `act-agent/internal/config/config.go` AND
`README.md` together → `readme.status` stayed `fresh` (the `touched_self` branch won).
`post-commit:27` = `touched_self = any(c == p ...)`; `:29` = `if touched_watch and not
touched_self`.

**Proposed fix (doc-only, low-risk):** Document the assumption in UPDATE_LOOPS §5 as an
accepted failure mode ("a commit that edits an artifact and a watched path in one shot is
trusted to have refreshed it; reviewers verify at the pre-merge gate"). A script change to
flip this would create false-positive stales on every legitimate combined refresh commit —
not worth it. Doc the contract instead.

### M2 — A staled artifact keeps a `verified_against` that now lies
**What's wrong:** When the post-commit hook stales an artifact it sets `status: stale` and
`staled_by: <commit>` but leaves `verified_against` pointing at the OLD fresh commit
(`post-commit:30-34` never touches `verified_against`). A casual reader of `freshness.json`
sees `"verified_against": "d7ef63b"` next to `"status": "stale"` and may anchor on the
former. `freshness-check.sh` doesn't print `verified_against`, which mitigates the
SessionStart path, but the raw JSON is the registry of record.

**Live evidence (reproduced):** After staling, `claude-md` = `{status: stale, staled_by:
ad1f704, verified_against: d7ef63b}`. `grep -n verified_against scripts/freshness-check.sh`
→ empty (not surfaced, but present in JSON).

**Proposed fix (doc-only or script-only — minor):** Either document in UPDATE_LOOPS §1 that
`verified_against` is "last commit it WAS fresh at; meaningless once status:stale," or have
the staler null it. Recommend doc-only — the field is genuinely useful as "where to diff
from" for the refresh (`git diff <verified_against>..HEAD`), so nulling it would remove the
scoping signal UPDATE_LOOPS §3 budget-rule 2 depends on. Document, don't null.

### M3 — Tracked audit working-files violate DOC_STANDARDS §3 frontmatter rule
**What's wrong:** DOC_STANDARDS §3: "Every tracked doc in `docs/` starts with frontmatter"
with `status: draft | current | superseded`. Several tracked files break this the same week
the standard shipped:
- `docs/audits/recon-path-a-sub1.md` (and sub2/sub3/synthesis/path-b-self) start with `#`,
  **no frontmatter at all**.
- `docs/audits/recon-comparison-A-vs-B.md` uses `status: pointer` — not in the §3 enum.
- `docs/audits/handoff-verification-2026-06-10.raw.json` is a 135KB raw dump tracked in a
  doc dir with no registry entry (not load-bearing; clutter).

**Live evidence:** `head -1 docs/audits/recon-path-a-sub1.md` → `# Recon — Path A / Sub 1`.
`head -5 docs/audits/recon-comparison-A-vs-B.md` → `status: pointer`.

**Proposed fix (decision + doc):** Decide whether transient dual-path working files are
"docs" under §3 or scratch. If scratch: either add a §3 carve-out for `*-path-*` /
`*-sub*.raw` working artifacts, or move them under a `docs/audits/_working/` the standard
exempts. If docs: backfill frontmatter and add `pointer` to the §3 enum (or change
recon-comparison to `status: superseded` pointing at the canonical report). Either way the
standard and the tree must stop disagreeing.

### M4 — Constitution Art. 7 / freshness `handoff` artifact register a file that doesn't exist
**What's wrong:** `freshness.json` registers `handoff` with `paths: ["docs/dev/HANDOFF.md"]`
and CLAUDE.md's Handoff Protocol now points readers there, but **the file does not exist**.
The registry is at least honest (`staled_by: "no tracked handoff exists yet"`), and the
canonical-home table in DOC_STANDARDS §2 lists it — so a reader following the pointer hits a
missing file. Low blast radius (status is `stale`/advisory) but it's a dangling pointer in
three docs.

**Live evidence:** `ls docs/dev/HANDOFF.md` → No such file. `freshness.json` handoff
`paths` = `["docs/dev/HANDOFF.md"]`. CLAUDE.md:444 points "this is a handoff" → that path.

**Proposed fix (doc-only / decision):** Acceptable to leave until the first tracked handoff
is written (the registry self-documents the absence). If you want zero dangling pointers
now, add a one-line `docs/dev/HANDOFF.md` stub: "No active handoff. Written at session end
per Constitution Art. 7." Recommend the stub — it makes every pointer resolve.

---

## LOW

### L1 — python3 absence degrades silently (no warning)
Both `post-commit` and `freshness-check.sh` invoke `python3` with no presence check. If
python3 is missing, the post-commit hook still `exit 0` (commit safe — good) but staling is
silently skipped; `freshness-check.sh` prints nothing. A machine without python3 gets a
freshness system that *looks* installed and does nothing. Evidence: `post-commit:11`,
`freshness-check.sh:9` both pipe to bare `python3`. **Fix (script-only, optional):** add
`command -v python3 >/dev/null || { echo "freshness: python3 missing, skipping" >&2; exit 0; }`
to both. Doc the python3 dependency in UPDATE_LOOPS §4 install block.

### L2 — Global MEMORY.md still says `act` is symlinked at `/opt/homebrew/bin/act`
The audit drift note and README correctly establish the `act → act-agent` rename
(`which act` → not found; `~/.local/bin/act-agent` is the live symlink). The user's global
`~/.claude/.../MEMORY.md` still asserts `/opt/homebrew/bin/act`. **Out of this review's
write-scope** (not in the repo), flagged for the orchestrator: the user's auto-memory
contradicts the now-correct repo docs. Evidence: `ls /opt/homebrew/bin/act` → No such file;
`which act-agent` → `/Users/user/.local/bin/act-agent`.

### L3 — `nestty/` gitignore negation references a deleted directory
`.gitignore:56` = `!nestty/*.md` but `nestty/` was deleted (CLAUDE.md pitfall 3 confirms,
`ls nestty/` → not found). Harmless dead negation; same for `!.windsurf/**/*.md` (no
`.windsurf/` dir). **Fix (doc/gitignore cleanup, optional):** drop the dead negations on the
next gitignore pass; not load-bearing.

---

## VERIFIED-SOLID (attacked, could not break)

These claims were re-grepped against live code and held — the orchestrator can treat them as
covered:

- **CLAUDE.md Tier-1 backend paragraph** — `app.go:104` switch with
  `case "claude-code", "antigravity", "agy", "codex", "opencode"` → `acp.NewACPAgent`
  (`acp/agent.go:95`); `cmd/act-tier1-shim/main.go` exists; `/backend` command lives in
  `slash.go:44`. The paragraph's "re-grep the switch, never trust a list" framing is exactly
  right. TRUE.
- **KI-02 tool subsets** — `tools.go:81-144`: Planner = `act_cli + expand_prompt_section`,
  Observer = `act_cli` only, Assurance/QA = `act_cli + view + grep`. No raw bash for any
  Tier-1 role. Matches CLAUDE.md verbatim. TRUE.
- **Sliding-window autoroute guard** — `orchestrator.go:1421 autoTurnCap = 5`,
  `:1427 autoRouteWindow = 10 * time.Minute`, `recentAutoRoutes []time.Time`; replaced
  `consecutiveAutoTurns`; Fix 6 = commit `4cb1d26` (exists, msg matches). Cleared on human
  input (`orchestrator_test.go:746`). All numbers TRUE.
- **Pitfall 4 (Qdrant tsconfig-excluded)** — `server/tsconfig.json:20` exclude includes
  `src/services/QdrantVectorStore.ts`. TRUE.
- **Pitfall 6 (Go MCP client retained)** — `mcp-tools.go:169 GetMcpTools`, `:173
  config.Get().MCPServers`. TRUE.
- **Pitfall 7 (PVM analytics real on active store)** — verified in H1 evidence; the CLAUDE.md
  rewrite is correct (it's ROADMAP that's wrong).
- **planner-prompts.json `!fromHuman` gate correction** — `orchestrator.go:584 if !fromHuman`
  in `runAgentTurn` (`:565`). The old "Planner bypasses marker by role" claim was indeed the
  wrong mechanism. Correction TRUE.
- **combined-analysis 3.5 un-strike** — `actCLICommandsACP` Planner-only scope; the
  correction matches the Round-6 #3 finding. (Cross-role half left OPEN and tracked — honest.)
- **AGENTS.md / GEMINI.md pointer files** — both are thin pointers (Art. 3 compliant), and
  every path they point at exists: `CLAUDE.md`, `docs/constitution/CONSTITUTION.md`,
  `DOC_STANDARDS.md`, `TASK_TRACKING.md`, `scripts/freshness-check.sh`. No dangling pointers.
- **DEV_LOG.md commit hashes** — all 21 hashes (`d7ef63b` … `6d934d2`) exist with matching
  subject lines via `git log -1 --format=%s`. Fully verified, no fabricated landings.
- **antigravity kanban ticket** — Spec/Success/Constraints/Invariants all present and
  accurate: switch members match `app.go:105`, untracked files exist, `gemini` is absent
  from dispatch, `grep -c antigravity CLAUDE.md`=1 / `README.md`=0 (consistent with
  "publication not yet complete"). Well-formed.
- **gitignore §6 negation rescue** — `!docs/planner-prompt-audit/**` works
  (`git check-ignore -q docs/planner-prompt-audit/planner-prompts.json` → TRACKABLE) despite
  the `*-prompt*` ignore; the removed trailing `CLAUDE.md`/`AGENTS.md` re-ignore lines are
  gone; both files are trackable. The §6 claims are TRUE.
- **architecture-flows artifacts** — all three (`architecture-flows.json/.html`,
  `flows-explainer.html`) exist and are git-tracked; correctly registered `stale` in
  freshness.json.
- **Post-commit edge cases** — empty commits and `--amend` both handled cleanly (exit 0, no
  crash; amend re-stales correctly). Only merges (C2) break.
- **freshness-refresh / freshness-check happy paths** — refresh sets `status/verified_against/
  staled_by` correctly; check prints stale artifacts with refresh pointers; unknown-artifact
  name errors cleanly. Schema in UPDATE_LOOPS §1 matches the JSON.
