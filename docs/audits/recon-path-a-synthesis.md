---
title: Recon Path A — Synthesis (reconciliation of sub1 + sub2 + sub3)
status: current
role: Path A synthesizer
verified_against: feat/cleanup-constitution working tree (HEAD 7efcaff; report pinned at bc0673e)
analyzed: 2026-06-11
method: read all three sub-analyses, re-grep the live tree for every cross-cutting / disputed
        apex claim before encoding it. The subs are claims under test, not evidence. Reconcile,
        do not concatenate.
inputs:
  - recon-path-a-sub1.md  (F-handoff + environment slice)
  - recon-path-a-sub2.md  (CLAUDE.md + README slice)
  - recon-path-a-sub3.md  (kanban + audit-line slice)
---

# Recon Path A — Synthesis

The three subs partition the corpus cleanly (sub1 = F-handoff/env, sub2 = CLAUDE.md/README,
sub3 = kanban/audit-line) and agree on the headline: **the verification report's verdicts are
sound at its `bc0673e` pin — zero in-slice verdict-errors across all three subs.** The action is
not in fixing the report's adjudications; it is in (a) one CONFIRMED verdict that is FALSE on this
machine, (b) a cluster of load-bearing coverage gaps the report never opened, and (c) the live
tree having grown a whole new Tier-1 backend since the pin. I re-grepped every apex claim below
against the live working tree before encoding it.

---

## 1. CROSS-CUTTING THEMES (appear in ≥2 subs)

### T1 — The `antigravity`/`agy` Tier-1 backend is the apex dual-implementation hazard (sub1 + sub2 + sub3)
All three subs converge on this, from three different doors:
- **sub1** (env): live `~/.act.json` has `planner.backend = "antigravity"`; new untracked
  `antigravity_cli.go` + `agy-acp.mjs` shims; `/backend` slash command added.
- **sub2** (CLAUDE.md): the backend is invisible to the entire doc set; README L13 lists only
  `act-agent`/`claude-code`; the report's own P9 evidence understates it (lists
  `claude-code/codex/gemini/opencode`, never `antigravity`).
- **sub3** (kanban): the `block6` ticket BODY tells an implementer to *create* the ACP backend
  layer that already ships at `internal/acp/` — now including `antigravity_cli.go`.

**My re-grep confirms all three:** `app.go:105` = `case "claude-code", "antigravity", "agy",
"codex", "opencode":` → `acp.NewACPAgent` (app.go:110); `ls internal/acp/` includes
`antigravity_cli.go`; `slash.go:44` `/backend` is Tier-1 while `slash.go:42` `/swarm` is Tier-2
(`act-agent|claude-code` only). This is the exact ACP failure mode CLAUDE.md's own anti-trust
banner names: a doc telling an agent something is unbuilt when it is live → a second, conflicting
implementation. It is the single highest-leverage theme in the entire corpus.

### T2 — Tier-1 backend selection is real; "only Tier 2" is decisively false (sub1 + sub2)
Both env and CLAUDE.md subs flag CLAUDE.md's *"Backend selection only applies to Tier 2 — Tier 1
agents are in-process goroutines and have no executable to swap."* Live `app.go:86-137` dispatches
Tier-1 roles on `cfg.Agents[role].Backend`. This is the report's P9 (correctly STALE) but both subs
add that the report's evidence is now itself understated (T1) and missed the `/backend` command
surface (sub2 CG-2). The statement is the one most likely to make an agent rebuild existing
machinery, so it co-ranks with T1.

### T3 — The `act` → `act-agent` rename is a live three-way contradiction the report got backwards (sub1 + sub2)
- **sub1**: `which act` → not found; `/opt/homebrew/bin/act` does not resolve.
- **sub2** (VE-1): same finding, plus README L64-68 is the *correct* source — command is
  `act-agent`, installer *removes* the old `act` symlink (nektos/act collision); binary lives at
  `~/.local/bin/act-agent`, not `/opt/homebrew/bin`.

**My re-grep confirms:** `which act` → "act not found"; `which act-agent` → `/Users/user/.local/bin/act-agent`.
The report's `act-symlink` verdict is **CONFIRMED in the report but FALSE on this machine** — this
is a genuine verdict-error (see Conflict C2 for the nuance). CLAUDE.md Development Commands + MEMORY
"The `act` Command" both teach the stale invocation; README already documents the correction.

### T4 — Report verdicts are correct; the failure is coverage + drift, not adjudication (all three subs)
Every sub re-grepped its slice's verdicts and found them right at `bc0673e`: sub1 (4 STALE/FALSE +
~12 CONFIRMED), sub2 (P3–P11, all nine STALE correct), sub3 (block6, qa-redesign, Fix-23, all four
Round-6 ACTIVE findings, chat-leak gate, 28 [FIXED] entries). The report's anti-trust method worked.
The residual risk is two-pronged and consistent across subs: (1) the report only adjudicated ~6 of
~50 active kanban tickets (sub3) and skipped whole CLAUDE.md/F-handoff sections (sub1/sub2); (2) the
live tree moved past the pin, drift-staling two CONFIRMED env verdicts (sub1).

### T5 — Stale-citation descriptions that mislead reasoning, not just counts (sub1 + sub3)
Two subs independently flag docs that describe a *mechanism* wrongly (worse than a wrong count):
- **sub3**: planner-prompts.json says the chat-leak gate is **role-based** ("Planner bypasses
  InternalPromptMarker"); live `orchestrator.go:584` is `if !fromHuman` — a Planner turn triggered by
  autoroute (fromHuman=false) IS marked and hidden. My re-grep confirms `if !fromHuman` at :584.
- **sub1**: F-handoff Phase-4 narrative says the replay bug is "masked because the Planner runs on
  **claude-code**" — live planner runs on **antigravity** (still ACP, so masking holds, but the named
  host is wrong). Both are citation-drift on behavior, the class most likely to mislead an agent
  reasoning about *why* something works.

---

## 2. CONFLICTS (where subs disagreed — named with both sides)

### C1 — Verified commit/pin: `bc0673e` vs working tree (sub3 vs sub1/sub2)
- **sub3** verified against `bc0673e` (frontmatter: "code ≡ feat/remove-nomik") and reports the
  block6 ticket points at a non-existent `internal/acp/` dir's absence of `antigravity_cli.go`
  implicitly — it lists the ACP files *including* `antigravity_cli.go` as shipped.
- **sub1/sub2** verified against the live working tree (HEAD `7efcaff` + untracked antigravity files).

**Resolution:** not a real disagreement — sub3 actually *did* see `antigravity_cli.go` in its
`internal/acp/` listing (its A.1 lists it), so it read the working tree too despite the `bc0673e`
frontmatter. The pin label in sub3's header is imprecise but its findings reflect live code. No
contradiction in substance; flag the header as cosmetically stale.

### C2 — Is `act-symlink` a verdict-error or a drift-stale? (sub2 vs sub1)
- **sub2** calls it a hard **verdict-error (VE-1)**: the report marked CONFIRMED, live says FALSE,
  full stop — and README proves the rename is intentional code reality.
- **sub1** is softer: it calls `act-symlink` **drift-stale** ("caused by live tree > bc0673e, not
  report error"), grouping it with the antigravity drift.

**Resolution — sub2 is more right, with a caveat.** The README evidence (installer *removes* the
`act` symlink, command *is* `act-agent`) means the rename is a deliberate, committed design choice,
not a post-pin environment drift like the antigravity backend. So calling it merely "drift" undersells
it: even at `bc0673e` the README already contradicted CLAUDE.md/MEMORY. **However**, whether
`/opt/homebrew/bin/act` existed *at the moment the report ran on this machine* is genuinely
unknowable from here (symlinks are env state, not git state). The defensible synthesis: the report's
`act-symlink` CONFIRMED is **at minimum stale and at worst a verdict-error**, and the *doc-level*
contradiction (CLAUDE.md/MEMORY say `act`, README says `act-agent`) is real and independent of the
symlink's momentary existence. Rank it HIGH either way.

### C3 — Placement of the three uncredited commits (sub1 refines sub-implicit / report P2)
- The **report's P2** says b03ef50/7f439ca/1e33bc8 sit "between the two documented efforts."
- **sub1** re-grepped timestamps (06-06 07:53–08:28) and found they **predate Phase 1 `e06f273`
  (12:05)** — they precede every table commit, not interleave.

**Resolution:** sub1 is correct and more precise; the "uncredited ~330 LOC" core of P2 holds, only
the placement phrasing needs correction. No other sub addressed this, so no inter-sub conflict —
recorded as a refinement to fold into the doc fix.

### C4 — "MCP bridge removed" scope (sub2 unique, no sub disputes but worth flagging)
- **sub2 CG-4** argues CLAUDE.md Pitfall 6's "MCP bridge removed" is overstated — the Go agent
  retains a wired native MCP client (`mcp-tools.go:169`, registered tools.go:26).
- No other sub examined this. It is not a conflict but a **single-source claim** that, if wrong,
  could cause an agent to delete live code — so it inherits elevated scrutiny. sub2's evidence
  (GetMcpTools instantiating stdio/SSE clients from config.MCPServers) is concrete; I did not
  independently re-grep mcp-tools.go in this pass, so I tag it MEDIUM-confidence pending one grep.

---

## 3. TOP-N RANKED FINDINGS (leverage-ranked; dual-implementation hazards first)

Leverage = how badly an agent acting on the wrong statement breaks things. Every fix is DOC-ONLY;
the codebase is never changed to make a doc true.

### #1 — CRITICAL · block6 ticket BODY would trigger a second ACP implementation (sub3 A.1; convergent T1)
**Evidence (re-grepped live):** `block6-acp-cli-backend-2026-04-21.md:27-29` says *"Files to create:
`act-agent/internal/llm/backend.go` — `AgentBackend` interface."* Live: `internal/llm/backend.go`
does not exist; the ACP layer ships at `internal/acp/` (agent.go, claude_code.go, **antigravity_cli.go**,
…) via `acp.NewACPAgent` (agent.go:95); no `AgentBackend` interface exists in Go. Status
`in-progress` is honest; the BODY is the trap.
**Doc-fix shape:** rewrite the "Files to create" block to point at shipped `internal/acp/` +
`acp.NewACPAgent`; re-scope remaining work to ACP-priming parity for non-Planner roles (finding r6-3).
**Why high-leverage:** this is the literal ACP dual-implementation failure mode the CLAUDE.md banner
was written about — an implementer following it builds a conflicting second backend layer.

### #2 — CRITICAL · CLAUDE.md "backend selection only applies to Tier 2" + the invisible antigravity backend (T1+T2; sub1, sub2)
**Evidence (re-grepped live):** `app.go:105` dispatches Tier-1 roles on `.Backend` for
`claude-code/antigravity/agy/codex/opencode` → `acp.NewACPAgent`. `slash.go:44` `/backend` is a
Tier-1 command; `slash.go:42` `/swarm` is Tier-2 (`act-agent|claude-code`). `~/.act.json` has
`planner.backend=antigravity`. CLAUDE.md says backend selection is Tier-2-only and never mentions
antigravity, `/backend`, or that Tier-1 is selectable at all.
**Doc-fix shape:** delete the "only Tier 2 / no executable to swap" sentence; document that Tier-1
roles are backend-selectable via `agents.<role>.backend` (`act-agent` in-process | `claude-code` |
`antigravity`/`agy`) and the `/backend <role|all>` command; add antigravity to README L13 and the
backend roster everywhere. Also amend the report's P9 evidence (now itself understated).
**Why high-leverage:** co-apex dual-implementation hazard; the statement most likely to make an agent
rebuild live machinery from scratch.

### #3 — HIGH · `act` → `act-agent` invocation contradiction across CLAUDE.md / MEMORY / README (T3; sub1 VE/sub2 VE-1)
**Evidence (re-grepped live):** `which act` → not found; `which act-agent` → `/Users/user/.local/bin/act-agent`;
`/opt/homebrew/bin/act` absent. README L64-68 correctly says command is `act-agent` and the installer
removes the old `act` symlink. CLAUDE.md Development Commands + MEMORY "The `act` Command" teach `act`,
`act --project`, `act -p`, `act status` — none resolve.
**Doc-fix shape:** rewrite CLAUDE.md Development Commands and MEMORY to `act-agent` at `~/.local/bin`;
correct the report's `act-symlink` CONFIRMED → FALSE/STALE. README is the source of truth.
**Why high-leverage:** every agent/user following CLAUDE.md's most basic invocation instruction hits a
"command not found" wall. Also the report's one genuine verdict-error (per Conflict C2).

### #4 — HIGH · planner-prompts.json describes the chat-leak gate as role-based, not `!fromHuman` (T5; sub3 A.5)
**Evidence (re-grepped live):** `orchestrator.go:584` = `if !fromHuman { content = InternalPromptMarker + content }`.
planner-prompts.json says *"the Planner role bypasses InternalPromptMarker."* False: an autoroute-triggered
Planner turn (fromHuman=false) IS marked and hidden.
**Doc-fix shape:** replace the role-based description with the `if !fromHuman` gate at orchestrator.go:584.
**Why high-leverage:** behavior-citation drift — misleads anyone reasoning about what leaks into the chat,
the kind of error that produces a confidently-wrong "fix."

### #5 — HIGH · F-handoff commit tables + coord-note claim completeness they don't have (sub1 P2/internal-contradiction)
**Evidence (re-grepped via git log):** `b03ef50`, `7f439ca`, `1e33bc8` (~330 LOC across
orchestrator/planner/observer/server) are in-window (`ac241e0..bc0673e`) but in neither writer's table,
while the coord note (L296-297) asserts the range "b03ef50→bc0673e … described above." sub1's refinement:
they **predate Phase 1**, not interleave between efforts.
**Doc-fix shape:** add the three commits to a credited table OR drop the coord note's completeness claim;
fix the placement language.
**Why high-leverage:** an agent trusting "all commits in range are documented" misjudges what shipped and
may re-do or fail to credit ~330 LOC of orchestration changes.

### #6 — MEDIUM · CLAUDE.md Pitfall 6 "MCP bridge removed" overstated (sub2 CG-4, single-source)
**Evidence (sub2, not independently re-grepped this pass):** `mcp-tools.go:169 GetMcpTools` instantiates
stdio/SSE MCP clients from `config.MCPServers`, registered at tools.go:26. Only the old *TypeScript* MCP
bridge was removed; native Go MCP support is live.
**Doc-fix shape:** reword Pitfall 6 to "the TS MCP bridge was removed; the Go agent retains native MCP
client support (mcp-tools.go)." Tag MEDIUM-confidence pending one confirming grep.
**Why high-leverage:** a blanket "MCP removed" could lead an agent to rip out working `mcp-tools.go`.

### #7 — MEDIUM · The P3–P11 CLAUDE.md drift cluster + count fixes (sub2 PART 1; sub1 P13/P14; sub3 done-cards)
**Evidence (re-grepped):** prompt files = 16 non-test (CLAUDE.md says 13); CLI = 23 dispatch branches
(says 21, also README L153); autoroute guard = `recentAutoRoutes` sliding window not `consecutiveAutoTurns`;
context paths include `AGENTS.md`; PVM analytics now real in `LocalEmbeddingVectorStore` (no Math.random in
live store — confirmed); F-handoff sessionID = 14 not ~35; `tsx watch` hot-reload not one-shot.
**Doc-fix shape:** apply the report's nine CLAUDE.md edits + P13/P14 F-handoff edits; fold README's
21-commands into the CLI-count fix.
**Why high-leverage:** individually low blast radius, collectively the "many small lies erode trust" tax;
the PVM pitfall in particular must flip to "embeddings AND analytics now real in the active store" so an
agent doesn't dismiss working analytics as placeholder.

### #8 — MEDIUM · Coverage debt — ~44 active kanban tickets + whole CLAUDE.md/F-handoff sections unadjudicated (T4; sub3 §B, sub1 §coverage, sub2 §2)
**Evidence:** sub3 swept ~50 tickets (compaction ×8, swarm-recovery ×7, ralph-loop, cli-fetch,
qa-deliverable, spil-stage1) and found **every status honest** — but the report only checked ~6.
sub1 lists 5 unchecked F-handoff sections; sub2 lists Build Order `--nestty` (removed flag), the
INTAKE/Runner/A2A/SPIL sections (re-grepped accurate). qa-redesign-phase-c carries a mild duplicate-fix
hazard (NEED_CLARIFICATION addressee-routing primitive already exists, orchestrator.go:2824 + Fix 11).
**Doc-fix shape:** no urgent edits (statuses honest), but (a) re-scope qa-redesign-phase-a (Grep half
shipped) and add the phase-c reuse note; (b) drop Build Order `--nestty`; (c) record that the report's
kanban coverage is ~12%, so absence of a flag is not proof of correctness.
**Why high-leverage (negative):** mostly reassurance — the unchecked surface is clean — but the two
qa-redesign re-scopes prevent duplicate work and the coverage note keeps a future auditor honest.

---

## 4. RECONCILED LENS TALLY

- **verdict-error:** 1 genuine (`act-symlink` CONFIRMED → FALSE/STALE on this machine; Conflict C2
  resolves it as at-minimum-stale, at-worst-error). Zero in-slice verdict-errors otherwise across all
  three subs.
- **dual-implementation-hazard:** 2 CRITICAL, convergent across all subs (block6 ticket body #1;
  CLAUDE.md Tier-1 backend + invisible antigravity #2).
- **duplicate-fix-hazard:** 2 (qa-redesign-phase-a Grep half shipped; phase-c clarification primitive
  exists).
- **status-dishonesty:** 0 — every swept kanban status is honest (sub3 swept ~50).
- **internal-contradiction:** 3 (act vs act-agent across CLAUDE.md/MEMORY/README; F-handoff coord-note
  completeness; CLAUDE.md "Tier 2 only" vs `/backend`).
- **citation-drift (behavioral):** 2 (chat-leak gate role-based vs !fromHuman; F-handoff "Planner on
  claude-code" vs antigravity).
- **drift-stale CONFIRMED verdicts (post-pin):** 2 (actjson-backends planner now antigravity;
  act-symlink) — env state past `bc0673e`, not report adjudication error.
- **coverage-gap:** ~44 unadjudicated active tickets (all clean) + 5 F-handoff sections + Build Order
  `--nestty` + Pitfall 6 MCP scope.

**Bottom line:** the report passes its own anti-trust bar; reconciliation work is concentrated in two
CRITICAL doc-fixes (#1, #2) that both prevent re-building the live ACP/antigravity backend, one HIGH
invocation contradiction (#3), and two HIGH behavioral-citation fixes (#4, #5). Nothing here justifies a
code change — every fix moves a doc to match code.
