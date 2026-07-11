---
title: Antigravity (agy) Backend — Per-Role Readiness
status: current
verified_against: 0753394
owner: generated
last_verified: 2026-07-11
---

# Antigravity (agy) Backend — Per-Role Readiness

What it takes for the antigravity CLI to stand in as each ACT agent role, now that it
works as the Planner. For whoever executes the role fan-out. Built from a grep-verified
reconstruction of the agy troubleshooting record plus a per-role study of the harness;
every claim cites file:line at `verified_against`.

## 1. Why it works as the Planner (the mechanism to replicate)

The chain, end to end:

1. **Config**: `agents.<role>.backend = "antigravity"|"agy"` in `~/.act.json`;
   `validateAgent` skips provider/model checks for ACP backends (config.go:517-520).
   Per-role `acp.{command,args,env,cwd}` overrides exist (config.go:82-126).
2. **Dispatch — role-generic, Tier 1 only**: app.go:104-110 routes any Tier 1 role with an
   ACP backend to `acp.NewACPAgent(role, backend, withTier1ShimPath(role,…), …, makePrimingInjector(role))`.
   `/backend <role|all> antigravity` accepts all four Tier 1 roles (slash.go:71-72, 323-326).
3. **PATH**: `withTier1ShimPath` (app.go:204-247) prepends the symlink-resolved binary dir so
   the subprocess resolves `agy-acp.mjs` and the role's `act-tier1-<role>` shim.
4. **Spawn**: `buildCommand` → `antigravityCLIDefaults` → `node <agy-acp.mjs>`
   (acp/antigravity_cli.go:18-20); stderr → `~/.act/runners/tier1-<role>-acp.log`; Setpgid subtree kill.
5. **Priming**: lazy per ACT session (`ensureACPSession`, acp/agent.go:338-384). Text =
   `InternalPromptMarker + doNotRespondHeader + GetAgentPrompt(role, ProviderACP) + renderShimNote(role)`
   (app.go:286-304). ACP prompt fragments exist for ALL FOUR Tier 1 roles via
   `actCLICommandsACP(role)` (common.go:123-158, parity fix 956fe70).
6. **Shim priming rule** (the marker-bug fix, 8cd7db3): first marked message of a fresh
   session = stored as system prompt, ack `end_turn`, agy never spawned (no NUL byte reaches
   argv — the original priming crash). Any LATER marked message = work order: strip marker, run.
7. **Per-turn**: identity re-injected every turn inside "WHO YOU ARE, not a task" fences
   (rogue-build fix); replay bounded to 12 turns × 8K chars (argv cap); 240s watchdog under
   the orchestrator's 5-min ceiling; full lifecycle logging (agy-acp.mjs:35-47, 126-136, 242-279).
8. **Output routing**: chunks stream into the shared message store (ThreadID = role);
   orchestrator parses directives as usual. **Build-contract gate** (orchestrator.go:1673-1705,
   escalation 1758-1769) is the output-seam backstop — Planner-contract-specific.

**Live confirmation**: coord 1111/1112 — finance run, agy Planner + claude-code swarm,
3 clean CREATE_TASKs in 17s unaided; 6/6 tasks validated, zero manual intervention.

### Problem → fix ledger (agy troubleshooting record, all verified in tree)

| Problem | Root cause | Fix | Where |
|---|---|---|---|
| Priming crash; agy ran as vanilla agent | NUL `\x00ACT_INTERNAL\x00` marker in argv | priming over ACP wire, marker stripped in shim | agy-acp.mjs:53,126-136 |
| Rogue build in `--print` (2026-07-07) | role prompt read as a task | identity fences per turn + build-contract gate | agy-acp.mjs:260-267; orchestrator.go:1673+ |
| Stall after brief + persona swap | every marked msg treated as priming | first-message-only priming rule | agy-acp.mjs:126-136 |
| Undiagnosable stall | silent bridge, no timeout | logs + 240s watchdog + in-chat error | agy-acp.mjs:35,45-47,158-217 |
| argv blowup on long sessions | unbounded replay | 12×8K replay cap | agy-acp.mjs:39-40,248-253 |
| Unbrokered tool calls | no ACP request handler | per-role permission broker | acp/permission_policy.go |
| Dead `act-tier1-*` shim | act→act-agent rename | shim execs act-agent sibling | cmd/act-tier1-shim/main.go:135-151 |
| ACP prompt contradiction for non-Planner roles | `actCLICommandsACP` was Planner-only | all-4-roles ACP fragments | common.go:123-158 |
| Identity swap on config fallback | fallback key selected prompt | true role always selects prompt | app.go:132-135; agent.go:799-806 |

**Broker caveat (important)**: the agy shim never sends `session/request_permission`
(dispatch handles only initialize/session-new/prompt/cancel, agy-acp.mjs:294-304), so the
permission broker is **dormant under agy** — it binds claude-code hosts. Under agy,
enforcement = prompt framing + `act-tier1-<role>` shim whitelist + output-seam gates.

## 2. Role by role — what agy needs

### Observer (Tier 1) — plumbing EXISTS; config flip + test
- Generic path fully wired; whitelist `{status, log, graph, context}` (act_cli_whitelist.go:32-37).
- Flow fits the shim: anomaly detection is Go-side (`detectAnomalies`, orchestrator.go:2366-2583);
  the LLM only *phrases* a changed anomaly set (noop gate `anomalySignature` +
  `lastObserverPromptHash`, orchestrator.go:1018-1033, 2193-2209). Every Observer input is
  internal-marked: first = priming, later snapshots = forwarded turns — exactly the 8cd7db3 rule.
- Loosest output contract of the four (free-text report → Planner autoroute) → most forgiving.
- Test for: verbosity feeding the autoroute (cap exists: 5 turns/10 min, orchestrator.go:1330-1345);
  120s loop cadence vs 240s agy watchdog overlap (loop skips while busy, orchestrator.go:2124-2127);
  statefulness divergence — in-process Observer is HistoryNone, agy shim replays 12 turns of
  prior snapshots (context growth, not extra firing).

### Assurance (Tier 1) — plumbing EXISTS; HARD output contract is the risk
- Generic path wired; tools in-process are act_cli+view+grep, under agy verification happens via
  agy's own read tools (broker dormant); shim whitelist `{validation, log, status}`.
- **The contract**: `parseValidationVerdict` brace-extracts JSON keyed on the literal
  `"criteriaResults"`; `Passed = score>=95 && !emptyCriteria` (orchestrator.go:3156-3204); the
  server independently fails closed on empty criteria (server/src/index.ts:903-905). Parse-fail
  → re-polled up to retry cap 3 → escalate. History proves drift here fails silently
  (2026-04-21 key-mismatch incident: every verdict → score 0 while the TUI showed "✅ 98").
- Needed: live check that agy `--print` emits clean verdict JSON (no markdown fences/prose
  preamble — brace-extraction tolerates surrounding prose but not mangled keys); if flaky, add
  an Assurance analog of the build-contract gate (corrective re-prompt on unparseable verdict —
  same shape as `buildContractCorrectivePrompt`, armed when a validation prompt dispatches).
  Also verify agy can actually read files from the project cwd in `--print` mode (Layer 2
  requires tool-verified criteria; assurance.go:32-38).

### QA/Synthesizer (Tier 1) — plumbing EXISTS; marker contract + window fit
- Generic path wired; shim whitelist `{validation, log, status}`.
- **The contract**: reply must end in `SYNTHESIS_COMPLETE: <summary>` or
  `NEED_CLARIFICATION: @<agent> <q>` (orchestrator.go:3209-3218; prompt says "Do NOT call any
  tools", ≤6 lines, orchestrator.go:3254-3266). No marker → unseen, re-polled to attempt cap 3.
  `NEED_CLARIFICATION` to a swarm agent suppresses Planner autoroute (maybeRouteQAClarification,
  orchestrator.go:2921-2947) — marker fidelity matters twice.
- Synthesis prompts are the largest Tier 1 payloads; 12×8K replay in one argv string may
  truncate multi-task context → tune `agents.qa_synthesizer.acp.env` (AGY_MAX_TURNS/
  AGY_MAX_TURN_CHARS) if needed.

### Tier 2 swarm (developer, frontend_dev, backend_dev, qa_engineer, researcher) — plumbing MISSING (KI-A)
Nothing is wired; `BackendAntigravity` exists but is excluded — "Tier 1 ACP only"
(swarm_roles.go:40, IsValidBackend :44-46). Live trap: `act-cli.ts:552` already accepts
`antigravity` for `act swarm set`, persisting a backend the Go spawner then silently drops
(buildSwarmSpecs falls back to act-agent, app.go:322-327).

Shared build (one dispatch path all five roles ride):
1. **Validity gates**: `IsValidBackend` + `/swarm` help (slash.go:63-64,189) + spawner probe —
   `startOneLocked` only probes the `claude` binary today (spawner.go:97-110); add an agy check.
2. **Runner dispatch branch**: `runAgent` branches only claude-code vs act-agent
   (act-runner.mjs:227-232). Safest shape: one-shot `runAgentAntigravity` (like the claude-code
   branch — plain stdout, act-runner.mjs:308-313) returning `{success,output,code}` so
   `reportComplete` still drives the authoritative `/complete` + `/submit-for-validation`
   (act-runner.mjs:181-203). Keep the runner as the coordination shell; swap only the inner turn.
   Tier 2 is one-shot-per-task — the shim's session replay is unneeded here.
3. **Identity delivery**: no Tier 2 priming path exists (makePrimingInjector is Tier-1-only,
   app.go:110). The claude-code backend already has this gap — it sends only a generic
   `Role: <role>` line (act-runner.mjs:564-571), losing the specialization prompts. For agy,
   prepend the role prompt inside identity fences in the task prompt (the shim's proven
   framing), or the same rogue-task failure mode returns. Also: `actCLICommandsACP` has no
   Tier 2 case — the in-process fragment wrongly says "do NOT shell out" for swarm roles
   (common.go:156-158); Tier 2 MUST shell out to the act CLI.
4. **Permissions**: the Tier 1 broker DENIES edit/write/execute-except-shim — the exact
   opposite of a builder's needs. A direct-subprocess integration sidesteps it (agy applies
   its own non-interactive policy), but then per-role restrictions are prompt-only:
   researcher's read-only contract (ResearcherTools, tools.go:46-65; claude-code
   `ROLE_DISALLOWED_TOOLS`, act-runner.mjs:285-292) has NO agy equivalent — check agy CLI
   for an allowed/disallowed-tools flag; otherwise accept prompt-level enforcement + server
   guards and say so in KNOWN_LIMITATIONS.
5. **act CLI reachability**: real `act-agent` on the subprocess PATH (no Tier 2 shim exists;
   whitelist is Tier-1-only). Registration/capabilities stay backend-agnostic if the runner
   keeps owning `register()` (act-runner.mjs:129-160).

Per-role deltas on top of the shared build: identity prompt content (frontend_dev.go /
backend_dev.go / qa_engineer.go / researcher.go specializations), researcher read-only
enforcement (above), qa_engineer "tests, not features" is prompt-only everywhere already.

## 3. Cross-cutting items that bite any new agy role
- **Broker dormancy** (§1 caveat) — decide: synthesize permission requests in the shim for
  parity, or accept prompt+shim enforcement (current, documented stance).
- **KI-B/KI-C**: compaction/summarize unsupported for ACP-backed roles
  (KNOWN_LIMITATIONS.md:31-40; acp/agent.go:517-521). For non-Planner Tier 1 the shim's
  12-turn replay cap bounds growth, so lower urgency than the Planner case.
- **KI-E**: two byte-identical `agy-acp.mjs` copies (act-agent/ + act-agent/runner/),
  hand-synced; only the binary-dir copy loads today. Consolidate before Tier 2 adds a third user.
- **KI-F**: no in-place backend restart — every flip needs an ACT relaunch (slash.go:278,289);
  slows the per-role test loop.
- **KI-H**: priming ack noise from smaller models — no real system channel over ACP.
- **Prompt-menu drift** (all backends): advertised CLI menus under-advertise the enforced
  whitelist — Observer's menu omits `context`, Assurance's omits `log`, QA's shows only
  `status` (common.go:65-82 vs act_cli_whitelist.go:32-47). Latent papercut; fix while touching
  the fragments.
- **Planner-hardcoded sites** (fine as-is, but know them): startup abort only for Planner
  (app.go:180); `HandleHumanInput` targets Planner only (orchestrator.go:244); all autoroutes
  funnel to Planner (orchestrator.go:407/1364/1666/1713/1762); HistoryThread is Planner-only
  (app.go:129-130); build-contract gate covers only the Planner contract — Assurance/QA
  contracts have retry caps but no corrective re-prompt.

## 4. Execution estimate
- **Tier 1 (Observer, QA, Assurance)**: config-flip + TUI test each; Assurance likely needs the
  verdict-contract gate (~small orchestrator.go change); QA maybe env tuning. Serial: roughly a
  short session per role, dominated by relaunch-per-flip (KI-F). Parallelizable per role.
- **Tier 2 (all five)**: one shared feature (KI-A: gates + runner branch + identity fencing),
  then thin per-role deltas. Do NOT parallelize the shared branch across role-agents — they'd
  collide in act-runner.mjs/swarm_roles.go; one builder + per-role verification agents after.
- Edge cases surface only in live TUI runs after rebuild; #1 suspect is agy verdict-JSON shape,
  #2 priming acks, #3 autoroute verbosity.
