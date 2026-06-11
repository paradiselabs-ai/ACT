# Recon — Path A / Sub 1: F-HANDOFF + ENVIRONMENT slice

Re-grep target: every report verdict sourced from `F-handoff.md` (git/commit claims, Phase 1–4
claims, push dates, sessionid count, server-dev script, config/env, build/test ground truth),
plus a coverage sweep of F-handoff sections the report never adjudicated.

Live tree at analysis time: branch `feat/cleanup-constitution`, **HEAD = `7efcaff`** (the report was
verified against `bc0673e`; HEAD is now 5 scaffolding commits past it AND carries uncommitted/untracked
working-tree changes — see HEADLINE 1). All `git cat-file`/`grep`/`go build`/`go test` run live.

---

## VERDICT CHECK — report verdicts vs my own re-grep

### STALE/FALSE problems sourced from F-handoff (all re-verified)

| Report ID | Verdict | My re-grep | Agree? |
|-----------|---------|-----------|--------|
| **P1** push-date-wrong | STALE | `git reflog show --date=iso origin/feat/remove-nomik`: `bc0673e @{2026-06-10 08:33:20}`, `26f2c3d @{2026-06-06 13:37:01}`. Push was **2026-06-10**, not the handoff's claimed 06-07. Range 26f2c3d→bc0673e correct. | ✅ CONFIRM |
| **P2** commit-tables-complete | FALSE | `b03ef50` (06-06 07:53, orchestrator.go+planner.go), `7f439ca` (08:21, orchestrator.go +100/orchestrator_types.go/new observer_anomaly_test.go), `1e33bc8` (08:28, server/index.ts +43) are all in-window (`ac241e0..bc0673e`) and in NEITHER table. | ✅ CONFIRM (+refinement below) |
| **P13** sessionid-35-sites | STALE | `grep -c 'o.sessionID' orchestrator.go` = **14** (2 assigns @144/149, 12 reads). `sid := o.sessionID` = 11; `\bsid\b` = 46. Handoff's "~35" matches nothing. | ✅ CONFIRM |
| **P14** server-dev-one-shot | FALSE | `server/package.json:10` = `"dev": "tsx watch src/index.ts"` (hot-reload). `git log -S 'tsx watch'` → unchanged since `5fb565e`; never one-shot. tsx `^4.21.0` present. | ✅ CONFIRM |

All four F-handoff STALE/FALSE verdicts are **correct**. No verdict-errors in the problem list for this slice.

### Representative CONFIRMED verdicts (re-grepped ~half)

| Report ID | My re-grep | Agree? |
|-----------|-----------|--------|
| all-17-commits-exist | `git cat-file -e` on all 17 SHAs → all present | ✅ |
| phase4-11-files | `git show --stat bc0673e` → exactly 11 files, matching the listed set | ✅ |
| phases123 symbol attributions | `git log -S`: applyRoleLabel→`e06f273`, anomalySignature→`9aa8417`, scopeHistory added `7021488`/removed `bc0673e`, HistoryMode→`bc0673e` | ✅ |
| p4-four-create-sites-stamped | `agent.go` ThreadID stamps at 382, 394, 491, 733 (4 sites) | ✅ |
| build-clean | `go build ./...` → exit 0 | ✅ |
| agent-tests-green | `go test ./internal/llm/agent/... ./internal/app/... ./internal/llm/prompt/...` → all `ok` | ✅ |
| lstool-run-panics | `go test ./internal/llm/tools/...` → `panic: config not loaded`, FAIL on `TestLsTool_Run/handles_empty_path_parameter` | ✅ |
| go-binary-version | `go version` = go1.26.1 darwin/arm64; go.mod `go 1.25.8`, module `github.com/paradiselabs-ai/ACT/act-agent` | ✅ |
| acpsessions-map-per-agent | `acp/agent.go`: `acpSessions map[string]string` (:59), keyed ACT-sessionID→ACP id (:331,348) | ✅ |
| acp-runturn-content-only | `acp/agent.go:219 runTurn`; `client.Prompt(ctx, acpSessionID, content)` (:278) — content only | ✅ |
| tui-renders-by-sessionid | `list.go:196,241` both gate on `msg.Payload.SessionID == m.session.ID` | ✅ |
| p4-migration-file | migration `20260607000000_add_message_thread_id.sql` present, `ADD COLUMN thread_id TEXT NOT NULL DEFAULT ''` | ✅ |

**Verdict-correctness conclusion for the F-handoff slice: every checked verdict (4 STALE/FALSE + ~12
CONFIRMED) is correct as of `bc0673e`.** Zero verdict-errors. The report's anti-trust work on this slice
is sound. The problems below are NOT report errors — they are (a) verdicts that have since drifted because
the live tree moved past `bc0673e`, and (b) F-handoff claims the report never adjudicated.

---

## HEADLINE 1 — live tree has moved past `bc0673e`: two CONFIRMED verdicts are now drift-stale

This is the highest-severity finding because it inverts two report verdicts at runtime, and it makes the
report's single highest-hazard finding (P9, Tier-1 backend) **worse**, not resolved.

The report verified `~/.act.json` and the `act` symlink at `bc0673e` and marked them CONFIRMED. Live:

- **`actjson-backends` (CONFIRMED) is now drift-stale.** Report snapshot: "planner=claude-code,
  observer=in-process (gpt-oss-120b), assurance=claude-code, qa_synthesizer=claude-code,
  developer=claude-code." Live `~/.act.json` (structure only, no secrets): **`planner.backend = "antigravity"`**
  (NOT claude-code), observer in-process (`openai/gpt-oss-120b:free`), assurance/qa_synthesizer/developer +
  all swarm roles = `claude-code`. `contextPaths = ["AGENTS.md"]`.
- **`act-symlink` (CONFIRMED) is now stale.** `which act` → **not found**; `/opt/homebrew/bin/act` no longer
  resolves. The global `act` command the docs/MEMORY assume is not on PATH on this machine right now.
- **`antigravity`/`agy` is a real, live backend** added since the snapshot: `app.go:105`
  `case "claude-code", "antigravity", "agy", "codex", "opencode":`; `config.go:516`; `slash.go:71-72,260,266,325`
  (a NEW `/backend <role> <act-agent|claude-code|antigravity>` slash command); `acp/agent.go:549,554`
  (`antigravityCLIDefaults`); `runner/swarm_roles.go:40 BackendAntigravity`; new untracked file
  `act-agent/internal/acp/antigravity_cli.go`. `git status` shows uncommitted edits to app.go, slash.go,
  config.go, acp/agent.go, types.go plus the untracked `antigravity_cli.go` and `agy-acp.mjs` shims.

Impact: an agent trusting the report's CONFIRMED `actjson-backends`/`act-symlink` lines would (a) believe the
Planner runs on claude-code when it runs on antigravity, and (b) try to invoke a global `act` that isn't
installed. **This is the report's own "map goes stale within commits" lesson firing on the report itself** —
expected and acceptable given the bc0673e pin, but the reconciliation must note that the F-handoff/CLAUDE.md
backend facts are now TWO backends (claude-code + antigravity/agy) and a `/backend` command behind.

---

## HEADLINE 2 — P9 dual-implementation hazard is the top reconciliation target (and got worse)

Live `app.go:86-137` confirms P9's core: Tier-1 roles dispatch on `cfg.Agents[agentName].Backend` →
`acp.NewACPAgent(role, backendChoice, …)` for `claude-code/antigravity/agy/codex/opencode`, else in-process
`agent.NewAgent`. CLAUDE.md still asserts "**Backend selection only applies to Tier 2 — Tier 1 agents are
in-process goroutines and have no executable to swap.**" That statement is decisively FALSE in live code, and
since the snapshot the surface has grown a `/backend` slash command and an antigravity host. This is the exact
ACP dual-implementation failure mode CLAUDE.md's own banner names. **An agent reading CLAUDE.md could rebuild
Tier-1 backend selection from scratch on top of the existing, now-larger, ACP machinery.** Highest leverage to fix.

---

## COVERAGE GAPS — checkable F-handoff claims the report never adjudicated

The report adjudicated F-handoff's git table, Phase 1–4 symbol facts, sessionid count, and server-dev script.
It did NOT adjudicate these checkable F-handoff statements (I verified each live):

1. **Phase-4 "Problem it fixed" narrative (F-handoff L62-68) — now partly stale, never checked.** It states
   the replay bug is "masked today only because the Planner runs on **claude-code** (ACP)." Live planner backend
   is **antigravity** (still ACP → still masked, so the architectural point holds), but the literal
   "runs on claude-code" is now false. Coverage-gap + citation-drift. Low blast radius (masking still holds),
   but a doc-regen copying "claude-code" would be wrong.

2. **F-handoff Coordination note range "b03ef50→bc0673e … described in the section above" (L296-297) —
   internal-contradiction, never flagged as such.** `git log b03ef50..bc0673e` literally contains `1e33bc8`,
   `7f439ca` (and the base `b03ef50`), none of which appear in either writer's table. The coord note asserts
   the range is fully described; it is not. This is the same gap as P2 but stated as a self-contradiction
   inside F-handoff. **Refinement to P2:** the report says the three commits sit "between the two documented
   efforts," but their timestamps (06-06 07:53–08:28) are actually **earlier than Phase 1 `e06f273` (12:05)** —
   i.e. they precede every table commit, not interleave between efforts. The "uncredited ~330 LOC" core is
   correct; the "between the two efforts" placement is imprecise.

3. **OPEN/NEXT-STEP #4 migration claim (F-handoff L116-118) — never checked.** Verified: migration
   `20260607000000_add_message_thread_id.sql` exists with the stated `ALTER TABLE`. CONFIRMABLE; report
   covered the file under `p4-migration-file` but not the "graceful degradation on resumed old sessions"
   behavioral claim (that one is genuinely headless and should be tagged unverifiable, not assumed).

4. **Side-quests (F-handoff L150-158) — never checked.** `~/.claude/.vibespeak-active` EXISTS (vibespeak
   active, as claimed); `~/.claude/.caveman-active` ABSENT (caveman dormant, as claimed). `26f2c3d` =
   `fix(server): strip stray NUL bytes in lockKey delimiter`, server/index.ts (matches the spawned-sub-task
   `task_7a786c8e` claim). All three side-quest facts CONFIRMABLE — low importance, but they were in scope and
   unadjudicated.

5. **Alpha-lens section (F-handoff L287-292) — never checked.** Pure prioritization prose ("#1 was the only
   true alpha-blocker, backstopped by Fix 23"). Fix 23's partial-shipped state is corroborated elsewhere in the
   report (fix23-assurance slice), so the alpha-lens claims are consistent with adjudicated facts — but the
   section itself was not scored. Low risk; note only.

---

## RECONCILIATION PLAN (leverage-ranked, DOC-ONLY — never touch code)

1. **[CRITICAL — dual-implementation hazard] CLAUDE.md NesTTY §: delete "Backend selection only applies to
   Tier 2 / Tier 1 … have no executable to swap."** Replace with the live truth: Tier-1 roles ARE
   backend-selectable via `agents.<role>.backend` in `~/.act.json` (values: `act-agent` in-process,
   `claude-code`, `antigravity`/`agy`, plus unimplemented `codex`/`opencode`) and via the `/backend <role|all>`
   slash command; the wire mechanism is ACP (`acp.NewACPAgent`, `app.go:105`). This is the one statement most
   likely to cause a rebuild of existing machinery.

2. **[HIGH — drift correction] Update the backend roster everywhere it appears (CLAUDE.md, F-handoff
   config block, MEMORY).** It is now `act-agent | claude-code | antigravity/agy` for Tier 1, not
   "claude-code default" / "Tier-2-only." Note the live `~/.act.json` has **planner=antigravity** (report's
   snapshot said claude-code). Also: the global `act` symlink is currently absent — any doc telling an agent to
   run `act …` should be qualified or the symlink restored (env fix, not a doc fix).

3. **[HIGH — internal-contradiction] Fix the F-handoff commit tables + coordination note.** Add `b03ef50`,
   `7f439ca`, `1e33bc8` (~330 LOC across orchestrator/planner/observer/server) to a credited table, OR change
   the coord note's "range b03ef50→bc0673e … described above" to stop claiming completeness. Correct the
   placement: these three predate Phase 1, they do not interleave between efforts.

4. **[MEDIUM — citation/date drift] F-handoff L18-19/L23-24: change "PUSHED on 2026-06-07" to the real push
   date 2026-06-10 08:33** (and scrub any doc that inherited the 06-07 date). The state is repaired today, but
   the date is wrong and the report already flagged it (P1) — fold into the doc fix.

5. **[MEDIUM — number drift] F-handoff L135: change "~35 orchestrator sites read o.sessionID" to "14 direct
   sites (2 assigns + 12 reads) + ~46 derived `sid` uses."** Affects any refactor effort-estimate. (P13.)

6. **[MEDIUM — false claim] F-handoff L145-146: change "one-shot `npx tsx`, no hot-reload — restart to load
   server changes" to "`tsx watch` — hot-reloads on file change; in-memory state replays from
   coordination-log.jsonl on each restart."** Prevents wasted manual restarts and a redundant "add watch
   mode." (P14.)

7. **[LOW — citation drift] F-handoff Phase-4 narrative L62-68: change "Planner runs on claude-code" to
   "Planner runs on an ACP backend (currently antigravity)."** Masking behavior unchanged; only the named host
   drifted.

8. **[LOW — completeness] Tag the genuinely headless F-handoff claims as such rather than leaving them
   implicitly confirmed:** the resumed-old-session `thread_id=''` graceful-degradation behavior (L118) and the
   Observer "~66 msgs/~13K → ~1-2/~2K" reduction (L111) both need a running TUI. The report's single
   UNVERIFIABLE-HEADLESS entry covers the Planner-thread assertion but not these two.

---

## Lens tally for this slice
- **verdict-error:** none. All re-grepped F-handoff verdicts are correct at `bc0673e`.
- **dual-implementation-hazard:** 1 critical (P9 Tier-1 backend — CLAUDE.md, worsened by live antigravity).
- **status/drift-stale CONFIRMED verdicts:** 2 (`actjson-backends` planner now antigravity; `act-symlink`
  now absent) — caused by live tree > `bc0673e`, not report error.
- **internal-contradiction:** 1 (F-handoff coord-note range claims completeness it doesn't have — P2 sibling).
- **coverage-gap:** 5 F-handoff sections (Phase-4 narrative, coord-note completeness, migration-degradation
  behavior, side-quests, alpha-lens).
- **citation-drift:** Phase-4 "claude-code" host name; minor line drifts already self-warned by the report.
