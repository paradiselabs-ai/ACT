---
title: Recon Path B — single-pass re-verification of the 2026-06-10 anti-trust report
status: current
verified_against: working-tree @ feat/cleanup-constitution (bc0673e + 5 scaffolding commits + UNCOMMITTED code)
report_under_test: docs/audits/handoff-verification-2026-06-10.md
date: 2026-06-11
method: single-agent full-corpus read + live re-grep of every ranked item
---

# Recon Path B — self pass

I read the entire corpus in one sweep (the verification report + its abbreviated verdicts, F-handoff.md,
project CLAUDE.md, combined-analysis.md, README.md, and the active/done kanban tickets), then re-grepped the
**live working tree** for every judgement I rank below. The report was treated as a claim under test, never as
evidence. Citations are live `file:line` at the time of this pass.

**Headline:** The report is substantially correct — its 14 STALE/FALSE problems (P1–P14) all re-confirm, and
the consequential CONFIRMED verdicts (Tier-1 ACP dispatch, PVM analytics real, chat-leak gate `!fromHuman`,
Fix-23 guard, Round-6 #2/#3 OPEN) all hold. **But the report froze at `bc0673e`, and the working tree has since
diverged with uncommitted code that introduces two entirely new Tier-1 backends (`antigravity`, `agy`) that NO
document — report, CLAUDE.md, F-handoff, kanban, combined-analysis — describes.** That is the single highest-
leverage gap in this pass: a fresh dual-implementation hazard sitting in the working tree, invisible to every
scaffolding artifact. The report's other misses are README's stale "21 commands", a self-contradiction inside
its own §2 abbreviation (3.5 FIXED vs Round-6 #3 Planner-only), and several CLAUDE.md narrative sections it
never adjudicated at all.

---

## CRITICAL

### C1. [coverage-gap / dual-implementation-hazard] Two NEW Tier-1 backends in the working tree, documented nowhere
The report verified against committed `bc0673e`. The **live working tree** (uncommitted) has diverged:
- `act-agent/internal/app/app.go:105` — committed `bc0673e` reads `case "claude-code", "codex", "gemini", "opencode":`
  (`git show bc0673e:.../app.go` confirms). The **live** file reads
  `case "claude-code", "antigravity", "agy", "codex", "opencode":` — `gemini` dropped, `antigravity` + `agy` added.
- New untracked files: `act-agent/internal/acp/antigravity_cli.go`, `act-agent/agy-acp.mjs`,
  `act-agent/runner/agy-acp.mjs` (`git status --short`).
- `grep -rn "antigravity\|agy"` across CLAUDE.md, F-handoff.md, README.md, combined-analysis.md, and
  block6-acp-cli-backend ticket → **zero hits**. The CLAUDE.md NesTTY/Tier-2 backend section still lists only
  `act-agent` and `claude-code`; P9's own evidence quotes the stale `gemini` member.
- **Why CRITICAL:** this is the exact ACP failure mode CLAUDE.md's banner warns about, except *live and active* —
  an agent reading any doc to "add a new Tier-1 backend" has no idea `antigravity`/`agy` half-exist in the tree,
  and would either collide with the in-flight work or re-derive the ACP spawn machinery. It also means the
  report's `verified_against: bc0673e` header is no longer a safe proxy for "the code an agent will see today."
- **Doc-fix shape:** the reconciliation pass must (a) note in the report that the working tree has drifted past
  bc0673e; (b) once the antigravity/agy work commits, add a kanban ticket + CLAUDE.md backend-list entry; (c) do
  NOT let any doc keep implying the Tier-1 backend set is `{claude-code, codex, gemini, opencode}` — `gemini` is
  gone from the live switch, `antigravity`/`agy` are in.

### C2. [verdict-error, scoped] P9 evidence cites a backend list that is already stale in the live tree
The report's P9 (correctly STALE on the larger claim "Tier 1 has no backend selection") cites the dispatch as
`case "claude-code", "codex", "gemini", "opencode"`. That was exact at `bc0673e` but is **wrong for the live tree**
(see C1: live is `claude-code, antigravity, agy, codex, opencode`). The *verdict* (P9 = STALE, HIGH dual-impl
risk) is right; the *cited evidence string* has itself gone stale within three days — a live demonstration of the
report's own "line numbers/snapshots drift" warning applied to a member list, not just a line number.
- **Doc-fix shape:** when reconciling, re-grep `app.go` for the live switch members rather than quoting P9's list.

---

## HIGH

### H1. [internal-contradiction] Report §2 says "all 28 [FIXED] entries hold" while its own Round-6 §2 says 3.5 is only Planner-fixed
- `combined-analysis.md:119` renders entry **3.5** as `~~[FIXED in ac241e0] "Use it via Bash" vs "do NOT shell
  out"~~` (strikethrough = closed).
- The report's abbreviated verdict `r6-28-fixed-entries-hold` asserts *all 28 [FIXED] entries still hold*.
- But the report's **own** `r6-3-acp-cli-planner-only-open` (and F-handoff OPEN step #1, and round6 extra-obs
  line 268 item 2) say the 3.5 fix is **Planner-only** — ACP-backed Observer/Assurance/QA still get the
  in-process "do NOT shell out" fragment.
- **Live re-grep confirms the OPEN side:** `common.go:77/89/97/103` all still emit "do NOT shell out to send
  messages" for the non-Planner in-process roles; `actCLICommandsACP` (common.go:146) only branches for the
  Planner. So 3.5's strikethrough-FIXED status **overreaches** — exactly what F-handoff says to correct.
- The contradiction: a doc the report blesses as "all FIXED entries hold" contains an entry the report
  *elsewhere* proves is not fully fixed. An agent trusting the §2 summary would skip the cheap Planner-only
  correction F-handoff explicitly queues as next-step #1.
- **Doc-fix shape:** in `combined-analysis.md`, un-strike 3.5 (or annotate `[FIXED Planner-only — non-Planner ACP
  roles still OPEN, see Round-6 #3]`); and soften the report's `r6-28-fixed-entries-hold` to exclude 3.5's
  cross-role half. Note `combined-analysis.md` actually contains **51** `[FIXED` annotations (`grep -c`), not 28 —
  the "28" is the planner-prompts.json-scoped subset; the report should say which 28.

### H2. [coverage-gap] README.md repeats the stale "21 commands" — report only checked CLAUDE.md
P4 correctly flagged CLAUDE.md's "21 commands" as STALE (live `act-cli.ts` has **23** top-level `command ===`
branches: register, context, task complete/progress/retry/abandon/submit-for-validation, brief update, files
claim/release, pvm reindex/search/(bare), validation queue, message, log, graph task/unverified/conflicts,
status, codebase onboard/(bare), swarm — confirmed by `grep -nE "command ===" cli/act-cli.ts`). But the same
stale figure lives at **`README.md:153`** (`# `act <subcommand>` (21 commands, TS)`), which the report never
adjudicated. README is the *public-facing* doc, so the wrong count is more visible there than in CLAUDE.md.
- **Doc-fix shape:** update README.md:153 and CLAUDE.md together; better still, replace the magic number with
  "the act CLI subcommand surface" so it can't drift again.

### H3. [coverage-gap] CLAUDE.md narrative sections the report never checked
The report scoped CLAUDE.md to its "Project Structure / What Works / Pitfalls / Provider Config" bullet facts.
It never adjudicated several *prose* claims that an agent would act on:
- **Coordination-flow line** ("Runner calls `act-agent task complete` then `act-agent task submit-for-validation`").
  Live `runner/act-runner.mjs:194` does POST `/api/tasks/${taskId}/submit-for-validation` — **CONFIRMED true**,
  but it was unverified by the report; flagging that it *checks out* so the reconciliation doesn't re-question it.
- **INTAKE 5-question description.** Live `planner.go:43-50` confirms the 5-question form, the "Ready to start?"
  hard-stop, and `PROJECT_BRIEF:` emission — **CONFIRMED**, but note the brownfield path (planner.go:43) reduces
  it to **2 questions**, which CLAUDE.md's INTAKE description omits. Minor staleness, worth a one-line addition.
- **Socket.io "vestigial" framing** (flows-explainer). Live `server/src/index.ts:5,247,346` still imports and
  fires Socket.io events; "vestigial" is an overstatement the report carried but didn't test.
- **Doc-fix shape:** these are low-severity but should be folded into the CLAUDE.md reconciliation so the prose
  sections get the same anti-trust pass the bullet facts did.

---

## MEDIUM

### M1. [verified-correct, precision] P14 FALSE (`server-dev-one-shot`) confirmed — and the doc-fix is bigger than F-handoff
P14 is right: `server/package.json:10` is `"dev": "tsx watch src/index.ts"` (hot-reload), not the "one-shot npx
tsx, no hot-reload" F-handoff.md:145 claims. But note F-handoff *also* says state "replays from
coordination-log.jsonl" on restart — with `tsx watch`, an in-place edit triggers an auto-restart that re-replays
the log **mid-test**, which is a different (and arguably worse) failure mode than the manual-restart one the
handoff warns about. The reconciliation should not just flip "one-shot → watch"; it should add the
watch-restart-wipes-in-memory-state caveat the report's P14 impact line already names.

### M2. [verdict-confirmed] Chat-leak gate `!fromHuman` holds — but the resume `[SYSTEM]` surface is genuinely still leaking
The report's `r6-chatleak-gate-fromhuman` is correct: `orchestrator.go:584` is `if !fromHuman { content =
InternalPromptMarker + content }`. I re-confirmed the live-leak nuance the report's round6 extra-obs raised:
`HandleHumanInput` (orchestrator.go:237) prepends `resumeContext` (which starts with `[SYSTEM]`) to the human's
text, then calls `runAgentTurn(..., fromHuman=true)` (line 242) — so the marker is **not** applied and the
`[SYSTEM]` block renders in chat. This matches `combined-analysis.md` **7.1 [ACTIVE]** and planner-prompts.json
line 397. **Status-honesty check passes:** 7.1 is correctly still [ACTIVE], not falsely struck. The only fix
owed is the one F-handoff step #4 names (regen planner-prompts.json to mark the *gate* RESOLVED while keeping the
resume-surface ACTIVE) so a future audit doesn't re-flag the gate as the ghost it already chased once.

### M3. [verdict-confirmed] PVM analytics real (P8 STALE) — re-confirmed clean
`server/src/index.ts:42` instantiates `LocalEmbeddingVectorStore` as the active store. `grep -n
"Math.random\|placeholder\|stub\|// TODO"` over `LocalEmbeddingVectorStore.ts` → **zero hits**. `getAgentProfile`
(:273) calls real `lookupTaskOutcomes` (:294/:475); `getAgentSynergy` (:366) derives from real profiles. The
`0.85 + Math.random()` placeholders the live CLAUDE.md Pitfall #7 still describes survive only in the inactive
MockVectorStore/QdrantVectorStore. P8's "analytics fake" → STALE verdict is correct; CLAUDE.md Pitfall #7 should
be rewritten (embeddings AND analytics now real on the active store; only statistical-quality-at-runtime remains
unverifiable). This is the dual-implementation hazard the report rightly emphasizes: an agent trusting Pitfall #7
might re-build analytics that already exist.

### M4. [verdict-confirmed] Round-6 #2 brownfield injection still OPEN — fence absent
`r6-2-brownfield-injection-open` is correct. Live `agents_md.go:31` does `codebaseSection = "## Codebase
analysis\n\n" + notes + "\n\n"` with the raw `brief.CodebaseNotes` — no `<codebase_analysis>` fence, no
`CREATE_TASK:`/`PROJECT_BRIEF:`/`[SYSTEM]` scrub. `grep -rn "untrusted\|scrub\|codebase_analysis"` over app/ +
prompt/ → no fence anywhere. The OPEN status is honest. (Security note already correctly carried by the report; I
flag no new action beyond keeping the tracked item open.)

### M5. [verdict-confirmed] Autoroute guard (P5), tools subsets (P6), context paths (P7) all re-confirm
- P5: `consecutiveAutoTurns` is gone; live `orchestrator.go:90-98` is `recentAutoRoutes []time.Time`, cap at
  `:1320`, cleared in `HandleHumanInput:228`. STALE verdict correct.
- P6: `PlannerTools` (tools.go:81) = `NewActCLITool + NewExpandPromptSectionTool` ("Per KI-02: no raw bash");
  `ObserverTools` = act_cli only ("No bash"); Assurance/QA = act_cli + view + grep, no bash. CLAUDE.md's
  "Planner/Observer get just bash" is STALE — confirmed. The stale comment also survives in `app.go:76-78`
  (code comment, left as-is per anti-trust rule: fix the doc, not the code).
- P7: `config.go:308` = `{"AGENTS.md", "ACT.md", "ACT.local.md"}` — AGENTS.md present. STALE verdict correct.

### M6. [status-honesty, confirmed] Kanban frontmatter matches code reality on the three checked tickets
- `assurance-fail-closed-empty-criteria-2026-05-26`: frontmatter `status: in-progress`; live guard
  `orchestrator.go:3114` `Passed: parsed.OverallScore >= passThreshold && !emptyCriteria`
  (`emptyCriteria := len(parsed.CriteriaResults)==0` at :3094-ish). Matches the report's f23 confirmations and the
  "partial, stays in-progress" framing. Honest.
- `block6-acp-cli-backend`: `status: in-progress` — matches the live half-shipped ACP state. But its **body**
  (P11 STALE) still says "create internal/llm/backend.go + internal/llm/acp/backend.go + AgentBackend interface"
  — none exist; the impl landed at `internal/acp/`. The dual-implementation hazard P11 names is real and now
  *compounded* by C1 (the antigravity/agy work is presumably the in-flight continuation of this very ticket, yet
  the ticket body still points at the wrong file paths).
- `qa-redesign-phase-a`: `status: todo` but the Grep half is already shipped (P12 STALE) — re-confirmed:
  `qa_synthesizer.go:62-63` instructs "Use view and grep", tools.go grants them. Ticket should be re-scoped to
  AGENT.md-ingestion-only.

---

## LOW

### L1. [citation-drift, confirmed] P13 sessionID count (`~35` → live 14 direct) and assorted line drift
P13 STALE confirmed in spirit (architectural point true, number wrong). Low impact. The report's own §4 already
catalogs the per-slice line drift (actCLICommandsACP 146 vs 147, guard 3094 vs 3114, tool-result stamp 488 vs
491) — all immaterial, all consistent with the branch's known drift.

### L2. [coverage-gap] go.mod / README module-path staleness — report caught it in §4 but it lives in 3 places
Live `act-agent/go.mod:1` = `module github.com/paradiselabs-ai/ACT/act-agent`. MEMORY.md + any doc repeating
"kept as github.com/opencode-ai/opencode" is FALSE (report's config-env/build-and-test §4 caught this). README.md
:183 still references the OpenCode upstream for *license* purposes only — that one is legitimate (fork
attribution), not a module-path claim, so leave it. Distinguish the two when reconciling.

### L3. [coverage-gap] Build Order checkmarks + Team-workflow facts in CLAUDE.md never adjudicated
The report did not test CLAUDE.md's "Build Order (Current)" ✅ checkmarks or the two-founder Team/Domain-ownership
section. These are low-risk (aspirational/process docs, not code-shaped), but the constitution effort's "one
always-current picture" goal means they should get a light pass — e.g. Block 5 "in-TUI routing in Phase 2" is now
shipped (orchestrator routes to Assurance/QA live), so the checkmark prose understates reality.

---

## Verified-closed spot checks (report verdicts I re-confirmed against live code)

- `r6-chatleak-gate-fromhuman` — gate is `if !fromHuman` (orchestrator.go:584). **Holds.**
- P8 `pitfall7-pvm-analytics-placeholder` STALE — active store is LocalEmbeddingVectorStore, no Math.random/
  placeholder in it. **Holds.**
- P9 `tier1-backend-only-tier2` STALE — app.go:86-110 dispatches Tier 1 by `cfg.Agents[].Backend` to
  `acp.NewACPAgent`. **Holds** (with C1/C2 caveat on the now-stale member list).
- P5 autoroute, P6 tools, P7 context paths STALE — **all hold** (M5).
- Fix-23 guard (`f23-guard-shipped`) — `Passed = OverallScore>=passThreshold && !emptyCriteria`,
  orchestrator.go:3114. **Holds.**
- Round-6 #2 brownfield OPEN (`r6-2`) and #3 ACP-planner-only OPEN (`r6-3`) — **both hold** (M4, H1).
- P4 CLI 23 branches, P3 prompt-files 16 non-test/19 total — **both hold** (re-counted live).
- `cfg-swarm-commands` / `swarm set` scoped to Tier 2 — act-cli.ts:603 rejects non-swarm roles. **Holds.**
- `r6-28-fixed-entries-hold` — holds **except** 3.5 (H1): the cross-role half is OPEN.

---

## Leverage-ranked reconciliation plan (fix DOCS only — never the code)

1. **(C1) Surface the antigravity/agy working-tree divergence.** Add a note to the report + a tracked kanban item
   so no agent re-implements Tier-1 backend support. Once committed, add to CLAUDE.md's backend list. Highest
   leverage: live, undocumented, dual-implementation-shaped.
2. **(H1) Correct combined-analysis.md 3.5** from struck-FIXED to "Planner-only; non-Planner ACP roles OPEN," and
   qualify the report's "all 28 FIXED hold" line. Prevents a duplicate-fix / skipped-fix on the cheapest open item.
3. **(B-block6 / M6) Rewrite the block6 ticket body** to point at `internal/acp/` (not the non-existent
   internal/llm/backend.go), and re-scope qa-redesign-phase-a to AGENT.md-ingestion-only. Both are
   duplicate-implementation traps as written.
4. **(M3) Rewrite CLAUDE.md Pitfall #7** — analytics are real on the active store; only runtime statistical
   quality is unverifiable. Stops a re-implementation of existing analytics.
5. **(H2/L2) De-magic the "21 commands" number** in README.md:153 + CLAUDE.md, and purge the
   "opencode-ai/opencode module path" claim from MEMORY.md/any doc (keep README's license-attribution reference).
6. **(M1/M2) Fix F-handoff server-dev line** (watch mode + restart-wipes-state caveat) and **regen
   planner-prompts.json** to mark the chat-leak *gate* RESOLVED while keeping the resume `[SYSTEM]` surface ACTIVE.
7. **(H3/L3) Light anti-trust pass on the un-checked CLAUDE.md prose** — INTAKE brownfield 2-question variant,
   Socket.io "vestigial" overstatement, Build Order Block 5 understatement.
