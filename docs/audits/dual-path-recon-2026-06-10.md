---
title: Dual-Path Reconciliation — comparator pass over Path A + Path B
status: current
role: Comparator (Round-6 lesson enforced — convergence is not correctness; every convergent and top-unique finding re-grepped against live code)
verified_against: feat/cleanup-constitution working tree (HEAD 7efcaff + uncommitted code) — report-under-test pinned at bc0673e
report_under_test: docs/audits/handoff-verification-2026-06-10.md
inputs:
  - docs/audits/recon-path-a-synthesis.md  (Path A: 3 cold-slice subs + synthesizer)
  - docs/audits/recon-path-b-self.md        (Path B: single full-corpus pass)
analyzed: 2026-06-11
method: |
  Read both paths. Re-grep the LIVE tree for EVERY convergent finding and every top-5
  path-unique finding BEFORE trusting it. A finding both paths agree on that I cannot
  reproduce by grep is demoted with a note. Docs citing file:line are not evidence.
  Read-only — no code changes; the codebase is never edited to make a doc true.
---

# Dual-Path Reconciliation Report

Both paths independently reached the same headline and I have re-grepped the live tree
to verify it rather than trusting the agreement: **the 2026-06-10 verification report is
substantially correct at its `bc0673e` pin** — all 14 STALE/FALSE problems (P1–P14) and
the consequential CONFIRMED verdicts re-confirm. **The single highest-leverage gap both
paths surfaced — and which I reproduced by grep — is that the working tree has diverged
past the pin with uncommitted code introducing two new Tier-1 backends (`antigravity`,
`agy`), documented in zero artifacts.** This is a live, active dual-implementation hazard
of exactly the ACP kind CLAUDE.md's anti-trust banner was written about.

Convergence here is *earned*, not assumed: every item in §1 below was re-grep'd by me, and
where a path's cited file path was wrong I caught it (Path A's MCP file path; Path B's
brownfield directory) without the claim itself falling.

---

## 1. CONVERGENT FINDINGS (both paths; re-grep'd by comparator — HIGHEST CONFIDENCE)

### CV1 — CRITICAL · `antigravity`/`agy` Tier-1 backends live in the tree, in NO document (Path A T1/#2, Path B C1/C2)
**Comparator re-grep (live):**
- `act-agent/internal/app/app.go:105` =
  `case "claude-code", "antigravity", "agy", "codex", "opencode":` → `acp.NewACPAgent` (app.go:110).
- `git show bc0673e:.../app.go` = `case "claude-code", "codex", "gemini", "opencode":` — so **`gemini` was dropped, `antigravity` + `agy` added** post-pin.
- Untracked: `act-agent/internal/acp/antigravity_cli.go`, `act-agent/agy-acp.mjs`, `act-agent/runner/agy-acp.mjs` (`git status --short`). `ls internal/acp/` confirms `antigravity_cli.go` shipped.
- `grep -rln "antigravity\|agy"` over CLAUDE.md / F-handoff.md / README.md / combined-analysis.md / `.devtool/` → **zero hits**.
- `slash.go:71-72`: `/backend <role> <act-agent|claude-code|antigravity>` is a real Tier-1 command surface.

**Verdict: CONFIRMED, both paths right.** This is the apex finding. An agent reading any doc to "add a Tier-1 backend" has no idea `antigravity`/`agy` half-exist and would collide with or re-derive the ACP machinery. It also self-stales the report's own P9 evidence string (CV2) within three days.

### CV2 — CRITICAL · the report's P9 evidence string is itself now stale (Path A #2 amendment, Path B C2)
**Comparator re-grep:** P9's quoted dispatch list `claude-code, codex, gemini, opencode` was exact at `bc0673e` but is wrong for the live tree (CV1). The P9 *verdict* (STALE — "Tier 1 has no backend selection" is false) is correct; the cited member list is stale. A live demonstration of the report's own "snapshots drift" warning applied to a member list. **Doc-fix: re-grep `app.go` for live members rather than quoting P9.**

### CV3 — CRITICAL · block6 ticket BODY would trigger a second ACP implementation (Path A #1, Path B M6/H1-adjacent)
**Comparator re-grep:** `.devtool/features/block6-acp-cli-backend-2026-04-21.md` body says "create `internal/llm/backend.go` + `AgentBackend` interface." Live: no such file, no such interface; the ACP layer ships at `internal/acp/` via `acp.NewACPAgent`. Status `in-progress` is honest; the body is the trap — and is now *compounded* by CV1 (the antigravity/agy work is the in-flight continuation of this very ticket, yet the body still points at wrong paths). **Verdict: CONFIRMED, both paths right.**

### CV4 — CRITICAL/HIGH · `combined-analysis.md` 3.5 struck-FIXED, but fix is Planner-only (Path A T5-adjacent, Path B H1)
**Comparator re-grep (this is the convergent finding I scrutinized hardest):**
- `combined-analysis.md:119` = `### 3.5 ~~[FIXED in ac241e0] ...~~` (strikethrough = closed).
- Live `common.go:77/89/97/103` all still emit "do NOT shell out to send messages" for the **non-Planner** in-process role fragments.
- `actCLICommandsACP` (the ACP "you MUST shell out" variant) is called from **`planner.go:24` ONLY** — `grep -rn actCLICommandsACP prompt/` returns exactly one caller. Observer/Assurance/QA on an ACP backend still receive the in-process "do NOT shell out" framing.
**Verdict: CONFIRMED.** 3.5's strikethrough overreaches — the fix is Planner-only, exactly as F-handoff OPEN step #1 and Round-6 #3 say. Path B additionally notes combined-analysis contains **51** `[FIXED` annotations, not 28 (the "28" is the planner-prompts.json-scoped subset) — I confirm the report should name *which* 28. **This is an internal contradiction inside the scaffolding the report blesses.**

### CV5 — HIGH · chat-leak gate is `!fromHuman`, not role-based (Path A #4, Path B M2)
**Comparator re-grep:** `orchestrator.go:584` = `if !fromHuman { content = InternalPromptMarker + content }`. planner-prompts.json's "Planner role bypasses InternalPromptMarker" is a **mechanism** description that is false: an autoroute-triggered Planner turn (fromHuman=false) IS marked and hidden. The report's `r6-chatleak-gate-fromhuman` CONFIRMED verdict is right; the *planner-prompts.json description* is the drift. **Verdict: CONFIRMED.** (Path B's nuance also holds: the `[SYSTEM]` resume surface at `HandleHumanInput` genuinely still leaks because that path is fromHuman=true — combined-analysis 7.1 correctly stays [ACTIVE].)

### CV6 — HIGH · `act` → `act-agent` invocation contradiction across CLAUDE.md / MEMORY / README (Path A T3/#3, Path B implicit)
**Comparator re-grep:** `which act` → "act not found"; `which act-agent` → `/Users/user/.local/bin/act-agent`; `/opt/homebrew/bin/act` → No such file. CLAUDE.md Development Commands + MEMORY "The `act` Command" teach `act`/`act --project`/`act -p`/`act status` — none resolve. README is the correct source (command is `act-agent`, installer removes the old symlink). **Verdict: CONFIRMED doc-contradiction.** The report's `act-symlink` CONFIRMED is at-minimum-stale on this machine (symlink existence is env state, not git state, so I will not call it a hard verdict-error — but the CLAUDE.md/MEMORY-vs-README prose contradiction is real and machine-independent). Rank HIGH.

### CV7 — MEDIUM · PVM analytics real on the active store; CLAUDE.md Pitfall #7 understates (Path A #7, Path B M3)
**Comparator re-grep:** `server/src/index.ts:42` = `new LocalEmbeddingVectorStore()` (active store). `grep -c "Math.random\|placeholder\|stub\|// TODO"` over `LocalEmbeddingVectorStore.ts` → **0**. The `0.85 + Math.random()` placeholders survive only in inactive Mock/Qdrant stores. P8's "analytics fake → STALE" verdict is correct. **Verdict: CONFIRMED.** Pitfall #7 must flip to "embeddings AND analytics now real on the active store; only runtime statistical quality is unverifiable" so an agent doesn't re-build working analytics.

### CV8 — MEDIUM · CLI count is 23, stale "21" in BOTH CLAUDE.md AND README:153 (Path A #7, Path B H2)
**Comparator re-grep:** `grep -cE "command ===" cli/act-cli.ts` → **23**. CLAUDE.md says 21 (P4, correctly STALE); `README.md:153` repeats "21 commands, TS" — the report never adjudicated the README copy. **Verdict: CONFIRMED.** README is public-facing; fix both, ideally de-magic the number.

### CV9 — MEDIUM · Round-6 #2 brownfield injection still OPEN — no fence (Path A coverage, Path B M4)
**Comparator re-grep:** the injection lives at `act-agent/internal/app/agents_md.go:30-32` (Path B cited `agents_md.go` without the `app/` dir; the file exists, claim holds). Live: `codebaseSection = "## Codebase analysis\n\n" + notes + "\n\n"` with raw `brief.CodebaseNotes` — no `<codebase_analysis>` fence, no `CREATE_TASK:`/`PROJECT_BRIEF:`/`[SYSTEM]` scrub. `grep -rn "codebase_analysis\|untrusted\|scrub"` over app/ + prompt/ → none. **Verdict: CONFIRMED OPEN.** Status honest.

### CV10 — MEDIUM · Fix-23 fail-closed guard shipped (both paths)
**Comparator re-grep:** `orchestrator.go:3105` `emptyCriteria := len(parsed.CriteriaResults) == 0`; `:3114` `Passed: parsed.OverallScore >= passThreshold && !emptyCriteria`. **Verdict: CONFIRMED.** Kanban `assurance-fail-closed-empty-criteria` `in-progress` status is honest.

### CV11 — LOW · prompt-files count 16 non-test / 19 total (both paths, vs CLAUDE.md "13")
**Comparator re-grep:** `ls prompt/*.go | grep -v _test | wc -l` → **16**; all `.go` → 19. P3 STALE correct. **Verdict: CONFIRMED.**

---

## 2. PATH-A-UNIQUE FINDINGS (re-grep'd)

### A1 — MEDIUM · CLAUDE.md Pitfall 6 "MCP bridge removed" is overstated (Path A #6/CG-4)
Path A tagged this MEDIUM-confidence and flagged it had NOT independently re-grepped it (and cited the wrong file path, `internal/llm/tools/mcp-tools.go`). **I re-grepped it and it CONFIRMS:** the real file is `act-agent/internal/llm/agent/mcp-tools.go`; `GetMcpTools` (:169) instantiates live `client.NewStdioMCPClient` (:176) and `client.NewSSEMCPClient` (:188) from `config.Get().MCPServers` (config.go:263 field exists). CLAUDE.md:338 Pitfall 6 says ACT CLI "replaces all MCP tools" — a blanket claim that, taken literally, could lead an agent to rip out working native Go MCP support. **Promoted to verified; doc-fix: scope it to "the TypeScript MCP bridge was removed; the Go agent retains native MCP client support (mcp-tools.go)."** (Path A's cited path corrected by comparator.)

### A2 — MEDIUM(neg) · F-handoff three uncredited commits PREDATE Phase 1 (Path A C3/#5)
Path A's sub1 refined the report's P2: `b03ef50`/`7f439ca`/`1e33bc8` (~330 LOC) predate Phase-1 `e06f273`, not interleave "between the two efforts." I did not re-run the full git-log timestamp diff this pass (the ~330-LOC-uncredited core of P2 is what matters and both the report and Path A agree on it); **demoted to Path-A-unique pending nothing — the substance holds, only the placement phrasing is Path A's refinement.** Low blast radius.

### A3 — reassurance · ~44 active kanban tickets swept, every status honest (Path A sub3 §B)
Path A's sub3 swept ~50 tickets vs the report's ~6 and found zero status-dishonesty. I did not re-sweep all ~50 (out of scope for top-5 re-grep), so I carry this as **Path-A-unique, unverified-at-scale** — useful as coverage context, not as a load-bearing claim. The two it flags for re-scope (qa-redesign-phase-a Grep-half-shipped, phase-c clarification-primitive-exists) overlap Path B M6 and are verified there.

---

## 3. PATH-B-UNIQUE FINDINGS (re-grep'd)

### B1 — HIGH · combined-analysis has 51 `[FIXED`, not 28 — the report should name *which* 28 (Path B H1 tail)
Folded into CV4. I confirm the "28" is an unqualified subset reference; the report's `r6-28-fixed-entries-hold` should state the scope (planner-prompts.json-derived) and exclude 3.5's cross-role half.

### B2 — MEDIUM · F-handoff server-dev line: `tsx watch`, and the bigger caveat (Path B M1)
**Re-grep:** `server/package.json` `"dev": "tsx watch src/index.ts"`. P14 (FALSE on "one-shot, no hot-reload") is right. Path B's value-add: with `watch`, an in-place edit auto-restarts and re-replays `coordination-log.jsonl` **mid-test** — a worse failure mode than the manual-restart one F-handoff warns about. **Confirmed; doc-fix should add the watch-restart-wipes-in-memory-state caveat, not just flip the verb.**

### B3 — LOW/MEDIUM · INTAKE brownfield 2-question variant omitted from CLAUDE.md (Path B H3)
**Re-grep:** `planner.go:43` confirms the brownfield path reduces INTAKE to **2 questions** (build-next + do-not-touch) when a "CODEBASE ANALYSIS" block is present, vs CLAUDE.md's 5-question-only description. The same "Ready to start?" hard-stop applies. CLAUDE.md's INTAKE prose is accurate for greenfield, silent on brownfield. **Confirmed; one-line doc addition.**

### B4 — LOW · Socket.io "vestigial" overstated; Build Order Block 5 understated (Path B H3/L3)
**Re-grep:** `grep -c socket server/src/index.ts` → 61 hits; Socket.io still imported and firing. "Vestigial" (flows-explainer) is an overstatement. Build Order "Block 5 ... in-TUI routing in Phase 2" now understates reality (routing is live in orchestrator.go). Both low-risk prose; fold into the light CLAUDE.md pass. **Confirmed.**

### B5 — verified-correct · runner submit-for-validation, autoroute guard, tools subsets, context paths, module path
**Re-grep batch:** `runner/act-runner.mjs:194` POSTs `submit-for-validation` (CONFIRMED — coordination-flow prose holds). P5 autoroute `recentAutoRoutes` sliding window (not `consecutiveAutoTurns`), P6 tools subsets (Planner gets act_cli not bash), P7 context paths include `AGENTS.md` — all STALE verdicts hold. `go.mod:1` = `module github.com/paradiselabs-ai/ACT/act-agent` (not opencode-ai/opencode → MEMORY.md FALSE; README:183 opencode reference is *license attribution only* and legitimate — keep it). All **CONFIRMED**.

---

## 4. METHODOLOGY NOTES

- **Convergence held under re-grep.** Every CV1–CV11 reproduced by my own greps. This is the opposite of the Round-6 failure (where both paths agreed on a wrong "bug"); here the agreement is on real findings backed by live code.
- **Two path-citation errors caught, neither fatal.** Path A cited the MCP file as `internal/llm/tools/mcp-tools.go` (actual: `internal/llm/agent/mcp-tools.go`); Path B cited brownfield as `agents_md.go` without `app/`. Both *claims* survived re-grep — the files exist at the corrected paths and the assertions hold. This is exactly why the comparator must re-grep, not trust citations: the file:line discipline catches drift even when the conclusion is right.
- **Methodological symmetry.** Path A's fan-out gave deeper kanban coverage (~50 tickets) and the env-state findings (`which act`, `~/.act.json`); Path B's single pass gave tighter internal-contradiction reasoning (CV4/B1) and the F-handoff `tsx watch` second-order caveat (B2). Neither missed the apex CV1. The paths are complementary, not redundant.
- **One genuine scope boundary, not a flaw:** the report's `verified_against: bc0673e` was honest *for committed code* but is no longer a safe proxy for "the code an agent sees today" because of the uncommitted antigravity/agy work. Every future audit header should state committed-vs-working-tree explicitly.
- **Demotions:** A2 (commit-placement refinement) and A3 (~50-ticket sweep) are carried as Path-A-unique-not-fully-re-grep'd — used as context, not load-bearing. Nothing was demoted for failing a grep.

---

## 5. COMBINED LEVERAGE-RANKED RECONCILIATION PLAN (DOC-FIXES ONLY — never the code)

Ranked by blast radius: dual-implementation / duplicate-fix hazards first. Each item names the file, the false/stale statement, what live code actually says, and the source path.

| # | Sev | Tag | File + statement to fix | What live code says |
|---|-----|-----|--------------------------|---------------------|
| 1 | CRITICAL | convergent (CV1/CV2) | **Surface the antigravity/agy divergence.** Add a note to `handoff-verification-2026-06-10.md` that the working tree has drifted past `bc0673e`; do NOT let any doc imply the Tier-1 backend set is `{claude-code,codex,gemini,opencode}`. Once committed, add a kanban ticket + a CLAUDE.md/README backend-list entry for `antigravity`/`agy`. Amend P9's quoted evidence to the live switch. | `app.go:105` = `claude-code, antigravity, agy, codex, opencode`; `gemini` removed; `antigravity_cli.go`/`agy-acp.mjs` untracked; `/backend ... antigravity` at slash.go:71. Zero doc mentions. |
| 2 | CRITICAL | convergent (CV3) | **Rewrite block6 ticket body** (`.devtool/features/block6-acp-cli-backend-2026-04-21.md`). "Files to create: `internal/llm/backend.go` + `AgentBackend` interface" → point at shipped `internal/acp/` + `acp.NewACPAgent`; re-scope remaining work to ACP-priming parity for non-Planner roles (CV4). | No `internal/llm/backend.go`, no `AgentBackend` interface; ACP ships at `internal/acp/`. The antigravity/agy work is this ticket's continuation. |
| 3 | CRITICAL | convergent (CV4/B1) | **Un-strike combined-analysis 3.5** (`combined-analysis.md:119`): `~~[FIXED in ac241e0]~~` → `[FIXED Planner-only — non-Planner ACP roles still OPEN, see Round-6 #3]`. Qualify the report's `r6-28-fixed-entries-hold` to exclude 3.5's cross-role half and name *which* 28 (there are 51 `[FIXED` total). | `actCLICommandsACP` called only from `planner.go:24`; `common.go:77/89/97/103` still emit "do NOT shell out" for non-Planner roles. |
| 4 | CRITICAL | convergent (CV6) | **Delete the "backend selection only applies to Tier 2 / no executable to swap" sentence** in CLAUDE.md; document Tier-1 backend selection via `agents.<role>.backend` + `/backend <role|all>`. | `app.go:86-110` dispatches Tier-1 by `cfg.Agents[role].Backend` to `acp.NewACPAgent`; `/backend` is a real Tier-1 command. |
| 5 | HIGH | convergent (CV6) | **Rewrite CLAUDE.md Development Commands + MEMORY "The `act` Command"** from `act`/`act -p`/`act status` to `act-agent` at `~/.local/bin`. Correct report `act-symlink` CONFIRMED → STALE-on-this-machine. README is source of truth. | `which act` → not found; `which act-agent` → `/Users/user/.local/bin/act-agent`; `/opt/homebrew/bin/act` absent. |
| 6 | HIGH | convergent (CV5) | **Replace planner-prompts.json "Planner bypasses InternalPromptMarker"** with the `if !fromHuman` gate; regen to mark the chat-leak *gate* RESOLVED while keeping the resume `[SYSTEM]` surface (combined-analysis 7.1) ACTIVE. | `orchestrator.go:584` = `if !fromHuman { content = InternalPromptMarker + content }`. |
| 7 | MEDIUM | convergent (CV7) | **Rewrite CLAUDE.md Pitfall #7**: embeddings AND analytics now real on the active `LocalEmbeddingVectorStore`; only runtime statistical quality unverifiable. The `Math.random` placeholders live only in inactive Mock/Qdrant stores. | `index.ts:42` active store; `grep Math.random LocalEmbeddingVectorStore.ts` → 0. |
| 8 | MEDIUM | A-only verified (A1) | **Scope CLAUDE.md Pitfall 6** ("MCP bridge removed... replaces all MCP tools") to "the TypeScript MCP bridge was removed; the Go agent retains native MCP support." | `internal/llm/agent/mcp-tools.go:169 GetMcpTools` instantiates live stdio/SSE clients from `config.MCPServers`. |
| 9 | MEDIUM | convergent (CV8) | **De-magic "21 commands"** in `README.md:153` AND CLAUDE.md → "the act CLI subcommand surface." | `grep -cE "command ===" act-cli.ts` → 23. |
| 10 | MEDIUM | B-only (B2) | **Fix F-handoff server-dev line** (`F-handoff.md:145`): `tsx watch` (hot-reload), and add the watch-restart-re-replays-coordination-log-mid-test caveat. | `server/package.json` `"dev": "tsx watch src/index.ts"`. |
| 11 | LOW | convergent (B5) | **Purge "module path kept as github.com/opencode-ai/opencode"** from MEMORY.md / any doc; keep README:183 OpenCode license attribution (legitimate fork credit). | `go.mod:1` = `github.com/paradiselabs-ai/ACT/act-agent`. |
| 12 | LOW | B-only (B3/B4) | **Light anti-trust pass on un-checked CLAUDE.md/flows prose**: add INTAKE brownfield 2-question variant; soften Socket.io "vestigial"; update Build Order Block 5 understatement. Re-scope qa-redesign-phase-a to AGENT.md-ingestion-only (Grep half shipped). | `planner.go:43` 2-question brownfield; `index.ts` Socket.io live (61 refs); `qa_synthesizer.go:62-63` Grep already granted. |

**Bottom line:** the report passes its own anti-trust bar. The entire reconciliation is doc-only. The first action slice is items 1–4 — every one prevents an agent from re-building or skip-fixing live ACP/antigravity machinery, which is the highest-blast-radius failure class this corpus contains.
