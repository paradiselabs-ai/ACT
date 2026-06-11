---
title: Recon Path A — Sub 2 (cold slice: CLAUDE.md + README)
status: current
verified_against: feat/cleanup-constitution (working tree, includes untracked ACP/antigravity files)
slice: claude-md + readme
last_verified: 2026-06-11
method: grep-only re-verification of the live tree; the report is a claim under test, not evidence
---

# Recon Path A — Sub 2: CLAUDE.md + README

Scope: re-grep every report verdict touching CLAUDE.md (P3–P11), then sweep the CLAUDE.md
sections + README the report never adjudicated. Read-only. Live tree on
`feat/cleanup-constitution` including untracked working-tree files (the new ACP/antigravity
backend is uncommitted but live in the dispatch path).

---

## PART 1 — Verdict correctness (re-grep of report's CLAUDE.md verdicts)

All nine CLAUDE.md-slice verdicts the report adjudicated are **CORRECTLY judged**. Re-grep:

| Report ID | Verdict | My re-grep | Agree? |
|-----------|---------|------------|--------|
| P3 `p1-prompt-files-13` | STALE | `ls prompt/*.go` = 19 files (16 non-test). CLAUDE.md says 13. | ✅ correct |
| P4 `p1-cli-21-commands` | STALE | act-cli.ts has 23 dispatch branches; extras: `task abandon`, `pvm reindex`, `codebase`/`codebase onboard`, `swarm`. | ✅ correct |
| P5 `ww-autoroute-guard` | STALE | `consecutiveAutoTurns` gone; live = `recentAutoRoutes` sliding window, `autoTurnCap=5` (orchestrator.go:1421), `autoRouteWindow=10m` (:1427), cleared in HandleHumanInput (:228). | ✅ correct |
| P6 `ww-tier1-tool-subsets` | STALE | tools.go: Planner = ActCLI + ExpandPromptSection "no raw bash" (:78); Observer = ActCLI only "No bash" (:90); Assurance/QA = ActCLI + view + grep. No bash anywhere. | ✅ correct |
| P7 `ww-context-paths` | STALE | config.go:308-311 = `{"AGENTS.md","ACT.md","ACT.local.md"}`. CLAUDE.md says only ACT.md/ACT.local.md. | ✅ correct |
| P8 `pitfall7-pvm-analytics-placeholder` | STALE | Active store = `new LocalEmbeddingVectorStore()` (index.ts:42). No `Math.random`/`placeholder`/`return 0.x` in LocalEmbeddingVectorStore.ts or SelfImprovementEngine.ts. `0.85 + Math.random()` survives ONLY in inactive MockVectorStore.ts:194 + QdrantVectorStore.ts:265. | ✅ correct |
| P9 `tier1-backend-only-tier2` | STALE | app.go:105 `case "claude-code","antigravity","agy","codex","opencode"` → acp.NewACPAgent; default → agent.NewAgent. Tier 1 IS backend-selectable. | ✅ correct (but evidence now understated — see CG-1) |
| P10 `provider-config-opencode-json` | STALE | `.opencode.json` is legacy fallback only (comments.go:82,109; writer.go:79-90). `find` confirms NO `.opencode.example.json` in repo. | ✅ correct |
| P11 `block6-files-to-create-stale` | STALE | `internal/llm/backend.go` and `internal/llm/acp/backend.go` do not exist; impl is at `internal/acp/`. | ✅ correct (kanban slice, re-confirmed) |

**No verdict-errors inside the CLAUDE.md slice.** The report's STALE calls are all sound.

### VERDICT-ERROR found in an ADJACENT slice that directly governs a CLAUDE.md section

**VE-1 [verdict-error] — `act-symlink` CONFIRMED is FALSE on this machine.**
- Report (config-env, line 237): *"`/opt/homebrew/bin/act` is a symlink to the repo's act-agent binary"* — marked CONFIRMED.
- Live: `ls /opt/homebrew/bin/act` → **No such file or directory**. `which act` → **act not found**. A PATH-wide scan (`for d in $PATH; do [ -e $d/act ]`) finds **no `act` command anywhere**. The real binary is `act-agent` at `/Users/user/.local/bin/act-agent` (not `/opt/homebrew/bin`).
- Why it matters for my slice: CLAUDE.md's entire **Development Commands** block and **"The `act` Command"** (MEMORY) tell every agent/user to run `act`, `act --project`, `act -p`, `act status`. None of those resolve. README (L64-68) is the one that's RIGHT: command is `act-agent`, and the installer *removes* any old `act` symlink (nektos/act collision). So the report CONFIRMED the stale half of an internal contradiction and missed that README already documents the correction.

---

## PART 2 — Coverage gaps (CLAUDE.md / README sections the report never checked)

### CG-1 [dual-implementation-hazard, HIGH] — the `antigravity`/`agy` Tier-1 backend is invisible to the whole doc set AND the report
A complete new backend landed in the working tree (untracked: `internal/acp/antigravity_cli.go`, `runner/agy-acp.mjs`, `agy-acp.mjs`) and is **live in the dispatch path**:
- app.go:105 case list = `"claude-code","antigravity","agy","codex","opencode"`.
- `runner.IsValidBackend` accepts it (swarm_roles.go:40 `BackendAntigravity`); config.go:516; cli/act-cli.ts:552 `VALID_BACKENDS=['act-agent','claude-code','antigravity']`.
- New `/backend` slash command switches it (slash.go:71-72, 260, 266).

The report's P9 evidence lists backends as `claude-code/codex/gemini/opencode` and never mentions `antigravity`, `agy`, or the `/backend` command. CLAUDE.md mentions none of it; README L13 says swarm backends are only `act-agent`/`claude-code`. **Hazard:** an agent told "Tier 1 backends are claude-code (shipping), others unimplemented" could re-add an antigravity backend that already exists — the exact ACP-style dual-implementation failure mode. Highest-leverage gap in this slice.

### CG-2 [internal-contradiction / coverage-gap, HIGH] — `/swarm` vs `/backend`; CLAUDE.md's "backend selection only applies to Tier 2"
CLAUDE.md's Tier-2 section says users switch backends with **`/swarm <role> <backend>`** and asserts *"Backend selection only applies to Tier 2."* Live code:
- `/swarm` (slash.go:42, 61-67) = **Tier 2 only**, backends `act-agent|claude-code`.
- `/backend` (slash.go:44, 69-72) = **Tier 1 only**, backends `act-agent|claude-code|antigravity`.

So the doc's "only Tier 2" claim is doubly wrong (P9 already flagged the principle) AND the doc never tells anyone the `/backend` command exists. The report flagged the principle (P9) but missed the **command surface** entirely. An agent will reach for `/swarm planner ...` and get rejected (CLI: *"backend selection only applies to Tier 2 swarm agents"*, act-cli.ts:603).

### CG-3 [internal-contradiction, HIGH] — README says the command is `act-agent`, CLAUDE.md says `act`
README L64-68 explicitly: command is **`act-agent`**, *"To avoid collision [with nektos/act]... the installer removes [the old `act` symlink]."* CLAUDE.md Development Commands + MEMORY say the command is `act` with a `/opt/homebrew/bin/act` symlink. Live PATH confirms README (VE-1): `act` does not exist; `act-agent` does. The two scaffolding sources flatly contradict each other on the single most basic invocation fact. README is correct; CLAUDE.md + MEMORY are stale.

### CG-4 [coverage-gap / status-dishonesty, MEDIUM] — Pitfall 6 "MCP bridge removed" overstated
CLAUDE.md Common Pitfall 6: *"MCP bridge removed. ACT CLI (`act`) replaces all MCP tools with ~50-100 tokens vs 47K schema overhead."* Live: the Go agent has a **real, wired MCP client** — `GetMcpTools` (mcp-tools.go:169) instantiates stdio/SSE MCP clients from `config.Get().MCPServers` and is registered into the Tier-2 tool set at tools.go:26. What was removed was the old *TypeScript MCP bridge*; the OpenCode-fork's native MCP support is alive. A blanket "MCP removed" could lead an agent to rip out working `mcp-tools.go`. (Key Concepts "A2A Protocol" / "act CLI" framing inherits the same overstatement.)

### CG-5 [citation-drift, LOW] — Build Order Block 3 cites a removed flag
CLAUDE.md Build Order: *"Block 3: act-agent opencode fork (--agent + --nestty modes) ✅"*. The `--nestty` flag is removed (no hits in `cmd/*.go`; MEMORY itself records "`--nestty` flag REMOVED"). The checkmark references a flag that no longer exists. The report never checked Build Order.

### CG-6 [coverage-gap, LOW] — README L153 repeats the stale "21 commands"
Same staleness as P4, in a second scaffolding file the report didn't open (README). Live = 23 dispatch branches. Folds into the P4 fix.

### Sections re-grepped and found ACCURATE (no action needed)
- **INTAKE description** (CLAUDE.md NesTTY): 5 questions = description/techStack/constraints/successCriteria/agentsInvolved — exact match planner.go:39, orchestrator.go:549-552. 404→intake-mode (orchestrator.go:246). ✅
- **Coordination flow / swarm roles**: `frontend_dev/backend_dev/qa_engineer/researcher/developer` match swarm_roles.go + SWARM_ROLES (act-cli.ts:551). ✅
- **Runner parallel-from-startup**: "one Node.js process per swarm role, polls server" matches spawner.go:5,32,35,73. ✅ (nuance: runner.mjs header still says "Wraps the `claude` CLI" but `AGENT_CLI` default = `./act-agent`, BACKEND default = `act-agent`, act-runner.mjs:35,79,92 — header comment is stale flavor text, not load-bearing.)
- **What Works server bullets** (EventHub 3/30s, ChronLog path, A2A, file locking, runner 409 self-heal, Setpgid, SweepOrphans, runner logs, lazy spawn, coord loop 3s, 5-min timeout): all CONFIRMED by report and re-spot-checked consistent. ✅
- **REPL commands** (create project / list agents / show project / default agent): present in help-system.ts:22-37. ✅
- **Terminology table**: banned names (Director/Producer/Actor) absent from prompt+app+roles code. ✅
- **SPIL / PVM-embeddings-real / A2A concepts**: SPILParser.parseSPIL + extractSuccessCriteria (SPILParser.ts:30,120); `.well-known/agent.json` (index.ts:188). ✅
- **Pitfall 4 (Qdrant TS error), 5 (tsx), 7 (PVM analytics — now P8)**: tsx ^4.21.0 present; Qdrant placeholder confirmed inactive. ✅

---

## PART 3 — Reconciliation plan (leverage-ranked, DOC-ONLY; never change code)

Ranked by blast radius if an agent acts on the wrong statement (dual-implementation hazards first):

1. **CG-1 (HIGH, dual-impl):** Add `antigravity`/`agy` to CLAUDE.md Tier-2/Tier-1 backend docs, the `/backend` command, and README L13. Note the files are still untracked working-tree — once committed, this is the single likeliest re-implementation trap. Also amend the verification report's P9 evidence (it's now itself understated).
2. **CG-3 + VE-1 (HIGH, contradiction):** Fix CLAUDE.md Development Commands + MEMORY "The `act` Command" to `act-agent` (and `~/.local/bin`, not `/opt/homebrew/bin`); correct the report's `act-symlink` CONFIRMED → it's FALSE on this machine. README is the source of truth here.
3. **CG-2 (HIGH, contradiction):** Rewrite CLAUDE.md Tier-2 "backend selection only applies to Tier 2" + add the `/backend` (Tier 1) vs `/swarm` (Tier 2) split. Pairs with the P9 fix.
4. **P3–P11 (MEDIUM, drift):** Apply the report's nine CLAUDE.md fixes (prompt count 13→16 non-test, CLI 21→23, autoroute guard wording, tool subsets, context paths +AGENTS.md, PVM pitfall 7 "analytics real now", Tier-1 backend, drop `.opencode.json` → `~/.act.json`). CG-6 (README 21-commands) folds into the P4 fix.
5. **CG-4 (MEDIUM, overstatement):** Reword Pitfall 6 — the *TS MCP bridge* was removed; the Go agent retains native MCP client support (mcp-tools.go). Prevents an agent deleting live code.
6. **CG-5 (LOW, citation-drift):** Build Order Block 3 — drop `--nestty` (removed flag).

**Invariant for every fix above: the doc moves to match code. Never the reverse.** The `act`→`act-agent` rename, the antigravity backend, and the MCP retention are all code reality; the scaffolding is what's wrong.
