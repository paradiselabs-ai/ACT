# Combined Planner Prompt Analysis — De-duped Findings

Source material:
- `analysis-path-a-sub1-static.md` — sub1 (6 composition fragments)
- `analysis-path-a-sub2-lifecycle.md` — sub2 (5 lifecycle prompts)
- `analysis-path-a-sub3-autoroute.md` — sub3 (8 autoroute/event prompts)
- `analysis-path-a-synthesis.md` — synthesizer over A
- `analysis-path-b-self.md` — Path B single-agent over all 19
- `comparison-A-vs-B.md` — methodological head-to-head

This document is the **canonical merged reference** — every distinct drift surface from both paths, de-duplicated, grouped by category. Strikethrough on an issue == fixed + verified.

---

## Tracking legend

- **[ACTIVE]** — work not yet started
- **[IN-PROGRESS]** — fix being worked on this session
- **[FIXED]** — committed + tested; entry rendered with ~~strikethrough~~

When all entries strike through (or earlier on request), regenerate `planner-prompts.json` and re-run the audit on the new state.

---

## Category 1 — Capability-lying (Planner asked to do things it can't)

### 1.1 ~~[FIXED in 01c1317] System-event autoroutes ask for unimplemented verbs~~
- ~~**Sources:** sub3 + Path B + synthesizer~~
- ~~**Where:** `orchestrator.go:577` (task_failed), `:594` (task_burst), `:1856` (validation_stuck), `:1978` (synthesis_stuck)~~
- ~~**What:** Planner instructed to "POST /api/tasks/:id/retry", "force-retry", "abandon", "check /api/tasks for full state." None of these have a Planner-visible tool. The Planner has only `act_cli` with {status, context, log, graph, pvm, message, codebase}. No retry subcommand. No HTTP tool. No abandon verb.~~
- ~~**Downstream effect:** Planner either fabricates an `act_cli` call with an unknown subcommand (schema rejects → unrecovered failure), or writes a no-op chat reply like "abandoning task X" with zero orchestrator effect.~~
- ~~**User's preferred fix:** Build the missing tools (give Planner real retry / abandon affordances via `act_cli`), don't just strip the verbs from prompts. Top-3 fix #2 — see plan file.~~
- **Resolution (01c1317):** Built the tools. `act_cli` now exposes compound entries `task retry` and `task abandon`. Server endpoint `POST /api/tasks/:taskId/abandon` reuses `failed` status + `metadata.abandoned=true` to avoid schema migration. Whitelist enforced at both the in-process tool runtime gate AND the ACP shim binary via `IsAllowed(role, subcommand, args...)`. `task complete` / `task progress` / `task submit-for-validation` stay swarm-only. Planner prompt + `act_cli_commands_fragment` updated to enumerate the new affordances. System-event prompt rewrites (still mention POST /retry etc.) deferred to Fix 3 since that commit re-templates them all with the variant scheme. Unit tests: `TestIsAllowed_Compound` (incl. forbidden `task complete`), `TaskCoordinator.abandonTask`.

### 1.2 ~~[FIXED in c853932] `expand_prompt_section` tool advertised but possibly not wired~~
- ~~**Sources:** sub1 + Path B~~
- ~~**Where:** `planner.go:24` (basePlannerPrompt) — *"You have an `expand_prompt_section` tool. ... Pull a section ONLY when you need it."*~~
- ~~**What:** The base prompt advertises 5 named expandable sections (evidence_routing, success_criteria, nomik, validation, examples). Per sub1's failure_modes_observed: *"no orchestrator-side dispatch was located during this audit — verify wiring or remove the advertisement."*~~
- ~~**Downstream effect:** Every Planner turn includes the offer. Some fraction of turns attempt the tool, get a hallucinated call, hit dead air.~~
- ~~**Fix:** Verify the wiring exists; if not, either implement the dispatcher or strip the advertisement.~~
- **Resolution (c853932):** Investigation found the dispatcher IS wired natively (`tools/expand_prompt_section.go` registered via `Tier1ToolsForRole` at `agent/tools.go:84`) — sub1 missed it because the audit was scoped to prompt fragments, not the tool layer. BUT the gap was real for ACP-backed Planners (they can't reach Go-side tools, only the act-tier1-* shim). Per user direction (build the tool, don't strip the prompt — same as Fix 2): extracted the registry into `prompt/sections.go` as a single source of truth; added `act-agent prompt-section <name>` Go-side CLI subcommand (no TS roundtrip, content is static); allowlisted `prompt-section` for Planner so the shim binary's IsAllowed gate passes; added `TestPromptSectionAdvertisementMatchesRegistry` regex-locking the prompt's "Available sections:" list to `SectionNames()` — drift-prevention test. Same registry now serves both backends.

### 1.2b ~~[FIXED in 4f7fc3e] `section_examples` teaches `@dependencies` inside description string (Round 5 — CRITICAL)~~
- ~~**Sources:** Round 5 Path A sub2 + sub3 synthesizer — *"section_examples shows @dependencies inside the description string in violation of the base prompt's shape rule."*~~
- ~~**Where:** `planner_section_examples.go:33` — the "Task with dependencies" EXAMPLE_TASK~~
- ~~**What:** The "Task with dependencies" example had `@dependencies\n- Snake game core loop must be complete` inside the JSON description string. planner.go:78 explicitly forbids this: *"Do NOT use @context, @dependencies, or any other @-section in the description string."* The examples section is the exact surface the Planner pulls when unsure about directive shape — it was actively teaching the wrong form. This also identifies the *source* of the malformed dependency shapes that Fix 6.4 (eabcc9b) installed a forgiving parser to defend against: the example modeled the forbidden shape.~~
- **Resolution (4f7fc3e):** rewrote the example to use top-level `"dependencies":["Snake game core loop"]` JSON property; description now goes directly `@task` → `@success_criteria` with no `@dependencies` block. Added `TestNoSectionEmitsForbiddenDependenciesShape` in `sections_test.go` — iterates every section via `SectionRegistry()` and asserts none contain `@dependencies\n` (the inline-section form). Catches future drift in any section body, not just `examples`.

### 1.3 ~~[FIXED in 6d934d2] `act_cli_commands_fragment` missing `prompt-section` subcommand~~
- ~~**Sources:** sub1 + Path B + synthesizer — *"actCLICommands('planner') is NOT enumerated despite being on AllowedFor('planner') ... ACP priming's bullet list (rendered from AllowedFor via renderShimNote) includes prompt-section; the hand-written in-process fragment does not."*~~
- ~~**Where:** `common.go:79` (planner case of actCLICommands) — hand-written list, 9 entries; `tools/act_cli_whitelist.go:21` (AllowedFor("planner")) — 10 entries including prompt-section~~
- ~~**What:** Fix 1.2 (c853932) added `prompt-section` to `AllowedFor("planner")` and `renderShimNote` auto-generates the ACP shim note from AllowedFor, so ACP Planners DO see `prompt-section`. But `actCLICommands("planner")` was never updated — it is a hand-written list. In-process Planners reading this fragment see 9 entries; AllowedFor has 10. Drift class: Theme 2 from Round 5 synthesis ("live-from-source pattern stopped where the test stopped"). Note: the 1.3 entry in the pre-Round-5 doc framed this as a decomposition-constraint placement issue; the Round 5 dual-path analysis refined the finding to the specific missing capability (`prompt-section`).~~
- **Resolution (6d934d2):** added `- act-agent prompt-section <name>       Pull on-demand Planner reference section (evidence_routing, success_criteria, nomik, validation, examples)` to the Planner case of `actCLICommands`. Added `TestActCLICommandsFragmentMatchesAllowlist` in `sections_test.go` — asserts each expected bare subcommand head (status, context, log, graph, pvm, codebase, prompt-section) and both compound forms (task retry, task abandon) appear in the fragment. Catches future fragment-vs-allowlist drift for the in-process Planner path.

---

## Category 2 — Structural duplication / token bloat

### 2.1 ~~[FIXED in ac87a1f] Autoroute envelope replayed across 7 of 8 triggers~~
- ~~**Sources:** sub3 + Path B + synthesizer~~
- ~~**Where:** `orchestrator.go:915`~~
- ~~**What:** The ~140-token `autoRoutePlanner` wrapper (*"React by taking action. Decide ONE of these..."* through *"Do not echo the report back."*) is appended to user-message turn on every autoroute. 7 of 8 prompts share it; only `tier1_watchdog_qa_retrigger` is unwrapped.~~
- ~~**Cost:** ~140 tokens × ~30 fires per project = ~30K tokens of pure boilerplate inside Planner's input stream. Non-cacheable because wrapper is appended to user turn, not system message.~~
- ~~**Sub3's fix:** differentiate per source (see 4.1 below). Synthesizer concurs as Top-5 #3.~~
- **Resolution (ac87a1f):** introduced `autoRouteVariant` (variantAnomaly / variantPassVerdict / variantFailVerdict / variantSystemEscalation) + `renderAutoRoutePrompt` + `autoRoutePlannerV`. Each source uses its own tighter template; Assurance posts parse via `parseValidationVerdict` to pick pass-vs-fail. The legacy one-template envelope is now only emitted for Observer + default. Net per-fire token shrinkage varies by source — system-event prompts drop ~60% (no more (a)/(b)/(c) tree + no dead POST verb), pass-verdict drops ~40% (silence-default framing is much shorter than the action tree).

### 2.2 ~~[FIXED in 9707f9a] `coordination_constraints_fragment` duplicates basePlannerPrompt role boundaries~~
- ~~**Sources:** sub1 + Path B~~
- ~~**Where:** `common.go:187` vs `planner.go:24` "Reacting to other roles" section~~
- ~~**What:** Both fragments tell the Planner "Assurance validates, QA assembles, you decide." Two slightly-different framings. ~50 tokens duplicated per turn × every Tier 1 call. Sub1 confirms: *"duplication that costs ~50 tokens every turn and gives the model two slightly-different framings of the same rule."*~~
- ~~**Fix:** Collapse into base prompt; drop the fragment.~~
- **Resolution (9707f9a):** opposite direction taken — fragment stays (shared across 9 roles, can't delete), and the duplicate "Reacting to other roles" block in basePlannerPrompt was removed instead. coordinationConstraints("planner") NEVER-list is now the single source. Locked by `TestBasePlannerPromptNoFragmentDuplication`.

### 2.3 ~~[FIXED in 9707f9a] `act_cli_commands_fragment` enumeration duplicated in basePlannerPrompt~~
- ~~**Sources:** sub1~~
- ~~**Where:** `common.go:71` enumerates CLI commands; `planner.go:24` redescribes act_cli usage rules.~~
- ~~**What:** ~700 bytes of enumeration with another ~200 bytes of restated rules in the base prompt. Plus the JSON-shape rule (`"args ALWAYS array"`) sits ~5K tokens away from the command enumeration — model adherence drops when affordance and constraint are far apart.~~
- ~~**Fix:** Fold into base; move JSON-shape rule adjacent to the command list.~~
- **Resolution (9707f9a):** the redundant "Allowed subcommands: status, context, log, graph, pvm, message, codebase, task" line was deleted from basePlannerPrompt; the fragment (with descriptions) is the single source. Anti-pattern reminder ("Do NOT attempt ls, cat, sqlite3, go, git, or raw shell") preserved — it's not in the fragment and is genuinely useful. JSON-shape co-location deferred (the `args ALWAYS array` rule still sits separately; revisit if drift surfaces). Net savings: ~104 bytes per Planner turn.

### 2.4 ~~[FIXED in a8577d2] `project_context_fragment` runtime substitution dominates fragment cost~~
- ~~**Sources:** sub1~~
- ~~**Where:** `prompt.go:50` (template ~80 bytes); runtime expansion injects ACT.md + ACT.local.md verbatim every turn.~~
- ~~**What:** CLAUDE.md Phase 3 deltas note Tier 1 requests dropped 22K→5-7K by trimming `defaultContextPaths`. The injection has no diffing — full files every turn.~~
- ~~**Fix:** Consider conditional injection or content hashing to skip if unchanged from previous turn.~~
- **Resolution (a8577d2):** investigation showed the audit's framing was partially inaccurate — `getContextFromPaths` already caches in-memory across turns, and `internal/llm/provider/anthropic.go:190` already wraps the system block in `CacheControlEphemeralParam{Type: "ephemeral"}`. What was actually missing: stability across `InvalidateContextCache` cycles that don't correspond to real file changes. Added `contextHash` (SHA-256) to the package-level cache; `InvalidateContextCache` no longer clears `contextContent`/`contextHash` — the next rebuild compares the new hash to the prior and reuses the previously-cached string if identical. Same bytes → same provider cache key → no needless cache miss. Test: `TestGetContextFromPaths_HashSkipReusesContent`.

---

## Category 3 — Mode/intent ambiguity

### 3.1 ~~[FIXED in ac87a1f] "Stay silent" vs "React by taking action" — the central tension~~
- ~~**Sources:** Path B + sub1 + sub2 + sub3 (only theme appearing in all five analyses)~~
- ~~**Where:** `basePlannerPrompt` (planner.go:24, *"Be concise. Don't narrate what you're about to do — just do it"*) vs `autoroute envelope` (orchestrator.go:915, *"React by taking action"*); these read as opposites.~~
- ~~**What:** Smaller models (and Claude Code occasionally) treat the opening "React by taking action" as the directive and produce a placeholder CREATE_TASK or acknowledgement. The wrapper's (c) clause — *"Stay silent (empty response) if neither applies"* — is buried at the end. **The #1 documented drift surface per recent commits 7156822, c237c0e, f522d1c.**~~
- ~~**Fix:** See Top-3 fix #3 (differentiate envelope per source — Assurance-pass defaults to silence, no (a) visible).~~
- **Resolution (ac87a1f):** variantPassVerdict explicitly leads with "No action is required by default" and lists silence as option (b) — not buried. variantSystemEscalation flips the other way ("Silence is WRONG here") for the unambiguous-action case. The "React by taking action" framing only ships in variantAnomaly now (Observer + unknown), where the (a)/(b)/(c) ambiguity is the right shape. Test: `TestRenderAutoRoutePrompt_PassVerdictHasNoReactByTakingAction` enforces the absence of the legacy framing.

### 3.2 ~~[FIXED in 3f0e8dd] BUILD-mode trigger doesn't restate the brief~~
- ~~**Sources:** sub2 + Path B~~
- ~~**Where:** `orchestrator.go:1022`~~
- ~~**What:** *"Project '%s' has been created. Switch to BUILD mode now: decompose the project brief into tasks and emit CREATE_TASK: directives."* But the brief content is NOT inlined. Planner has to recall from conversation history (vulnerable to compaction) or AGENTS.md (only if rebind worked).~~
- ~~**Fix:** Inline the brief in the trigger message.~~
- **Resolution (3f0e8dd):** `renderBriefContext("build", ...)` now inlines a SPIL-style `@brief` block with all 5 fields (description, techStack, constraints, successCriteria, agentsInvolved) directly from the parsed `*ProjectBrief` local. Empty fields are omitted rather than emitted as `field: ` blanks. Test: `TestRenderBriefContext_NoTasksBuildKind`.

### 3.3 ~~[FIXED in 3f0e8dd] Resume-context fields-mismatch can flip BUILD into INTAKE~~
- ~~**Sources:** sub2 + Path B~~
- ~~**Where:** `orchestrator.go:314`~~
- ~~**What:** Resume only inlines `description + techStack`. If GetProject envelope-unwraps wrong (today's confirmed bug), Planner reads `description:  | techStack: ` and intake-prioritization wins despite the [SYSTEM] "do NOT run intake" nudge.~~
- ~~**Fix:** Top-3 fix candidate but not in current top-3; inline full brief + completed/in-flight task list.~~
- **Resolution (3f0e8dd):** Shares the new `renderBriefContext` helper with the BUILD path. Resume now also calls `client.ListTasks()` and partitions the result into `@inFlightTasks` (anything not terminal) and `@completedTasks` (completed/validated), closing with an explicit "do NOT re-emit CREATE_TASK for the task IDs above — use act_cli task retry/abandon for failed tasks." Existing `project_resume_blank_fields` warn log preserved as the defense-in-depth tripwire for the envelope-unwrap regression. Tests: `TestRenderBriefContext_AllFieldsAndTaskLists`, `TestBriefViewFromGetProject_*`.

### 3.4 [ACTIVE] No mode echo back to orchestrator
- **Sources:** sub2
- **Where:** None of the 5 lifecycle prompts asks the Planner to declare its current mode.
- **What:** Orchestrator infers mode by parsing markers (PROJECT_BRIEF: vs CREATE_TASK:). A Planner that emits prose with no marker produces zero observable mode signal. `intakeMode` flag and LLM belief drift independently.
- **Fix:** Add an "OK acknowledging BUILD mode" or similar mode-echo expectation to the trigger prompts.

### 3.5 [FIXED in ac241e0 — **Planner-only**; non-Planner ACP roles still OPEN] "Use it via Bash" vs "do NOT shell out" — cross-fragment ACP contradiction

> **Correction 2026-06-11 (anti-trust verification + dual-path recon CV4):** the strikethrough
> below overreached. `actCLICommandsACP` is called from `planner.go` ONLY; ACP-backed
> Observer/Assurance/QA still receive the in-process "do NOT shell out" fragment (`common.go`
> role fragments) while their shim note says to use Bash. The cross-role half is tracked in the
> re-scoped `block6-acp-cli-backend` kanban ticket (Round-6 finding #3). Do not treat 3.5 as closed.
- ~~**Sources:** sub1 + Path B~~
- ~~**Where:** `acp_priming_prompt` (app.go:231) appends *"Use it via Bash for all ACT-coordination subcommands"*; `act_cli_commands_fragment` (common.go:71) says *"do NOT shell out to send messages."*~~
- ~~**What:** In-process Planner: fragment is restrictive. ACP Planner: priming overrides with Bash instruction. Same role, two backends, opposite instructions.~~
- ~~**Fix:** Branch the fragment by backend, or rewrite to be backend-agnostic.~~
- **Resolution (ac241e0, Fix 22):** The branch seam already existed but was unused — `PlannerPrompt(_ models.ModelProvider)` discarded its arg even though `app.go` passes `acp.ProviderACP` when priming an ACP session. Wired it: moved `ProviderACP` to the `models` package (so `prompt` can compare against it with no import cycle — `acp` imports `prompt`), and `PlannerPrompt(provider)` now selects `actCLICommandsACP("planner")` (act-tier1-planner shim via Bash) for ACP vs `actCLICommands("planner")` ("do NOT shell out", native act_cli JSON tool) for in-process. Same role, backend-accurate framing. This is also the structural enabler for the rest of Round 5b. Test: `TestPlannerPromptBranchesOnProvider`.

---

## Category 4 — Cascade / loop risks

### 4.1 ~~[FIXED in 4cb1d26] Three loops bypass `consecutiveAutoTurns ≤ 5` cap~~
- ~~**Sources:** sub3 + synthesizer~~
- ~~**Where:** orchestrator's `autoRoutePlanner` flow + QA poll loop~~
- ~~**What:** The cap resets on human input only. Three loops dodge it:~~
  - ~~**QA watchdog re-fire loop** — `tier1_watchdog_qa_retrigger` fires QA, QA produces non-marker reply, autoroutes back to Planner AND the next polling tick re-fires QA. Fresh trigger lineage each cycle resets counter toward 0.~~
  - ~~**Assurance verdict mirror loop** — if placeholder CREATE_TASK slips past server seam, dispatches to swarm → completes → Assurance fires again → autoroute again. Cross-task lineage avoids back-to-back cap.~~
  - ~~**Observer 120s echo loop** — Planner restates Observer's message; Observer sees on next cycle; reacts. Observer's 120s gap dodges the back-to-back counter entirely.~~
- ~~**Fix:** Sliding wall-clock window cap (5 autoroutes / 10 min, regardless of trigger source). Reset only on human input.~~
- **Resolution (4cb1d26):** `consecutiveAutoTurns int` replaced by `recentAutoRoutes []time.Time`. On each `autoRoutePlannerV` fire: prune entries older than `autoRouteWindow` (10 minutes) → reject if `len >= autoTurnCap` (5) → else append `now`. `HandleHumanInput` clears the slice (same reset-on-human semantics). Pure `pruneAutoRoutes(times, now, window)` extracted for testability — synthetic-clock tests verify pruning + cap enforcement without a clock-injection seam in the rest of the orchestrator. Log field renames: `consecutive_turns` → `fires_in_window`; reason `auto_turn_cap` → `auto_route_window_cap`. Tests: `TestPruneAutoRoutes_*`, `TestCascadeCap_*` (6 cases). 4.2 (Observer free-tier risk) gets the same protection as a free side-effect.

### 4.2 [ACTIVE] Observer free-tier rate-limit risk amplified by uncapped echo
- **Sources:** Path B
- **Where:** Observer is the only role not on Claude Code (still on minimax/minimax-m2.5:free per user's `~/.act.json`)
- **What:** Each Observer cycle = free-tier API call. Echo loop fires Observer extra cycles invisibly. User reported rate-limit symptoms in TUI; the cascade-cap fix (4.1) reduces frequency.
- **Fix:** Same as 4.1, plus consider migrating Observer to Claude Code given the cascade risk.

### 4.3 [ACTIVE] QA re-synthesis on TUI restart (mostly fixed but leftover noise possible)
- **Sources:** Path B
- **Where:** Fixed for going-forward in commit 408fb98 (synthesizedAt marker). Replay restores marker for historical events.
- **What:** Pre-fix sessions left `synthesis_complete` events without `synthesizedAt`. Replay handler falls back to event timestamp, but tasks created during transitional period may have null markers.
- **Fix:** Periodic check that any `validated` task lacks `synthesizedAt` AND was emitted by an old chronlog entry → backfill from the matching synthesis_complete event.

---

## Category 5 — ACP backend asymmetries

### 5.1 ~~[FIXED in 770a290] **ACPAgent.RebindSystemPrompt is a silent no-op** — Top-3 fix #1~~
- ~~**Sources:** sub2 + Path B + synthesizer (all flagged as highest leverage)~~
- ~~**Where:** `acp/agent.go:457`~~
- ~~**What:** `system_prompt_rebind` (orchestrator.go:991) iterates Tier 1 agents and calls RebindSystemPrompt; for ACPAgent this returns nil without doing anything. ACP-backed Planner carries priming-time worldview for the entire session even after AGENTS.md materializes post-intake. Symmetric to the pre-2026-05-19 in-process bug — no fix has landed for ACP.~~
- ~~**Downstream effect:** Every fix in this audit that touches composition or content is a no-op for ACP users until rebind works.~~
- ~~**Fix:** See plan file. Either send a fresh `session/new` with refreshed priming (loses conversation), or push a "context update" user message the ACP host treats as canonical. Latter cheaper.~~
- **Resolution (770a290):** discard-sessions approach. Best-effort Cancel each ACP session ID, then clear `acpSessions` map. Next `ensureACPSession` opens a fresh session and the priming injector re-reads `prompt.GetAgentPrompt`, picking up the freshly invalidated context. Refuses to run while IsBusy. Unit tests: `TestACPAgent_RebindSystemPromptDiscardsSessions`, `TestACPAgent_RebindSystemPromptRefusesWhenBusy`.

### 5.2 ~~[FIXED in afaf0c3] ACP priming is a USER message, not a system message~~
- ~~**Sources:** sub2 + Path B~~
- ~~**Where:** `acp/agent.go:340-346` — `client.Prompt(ctx, id, prime)` sends as user-role~~
- ~~**What:** ACP hosts treating user messages as conversation-level surface the role prompt as if the human said it. Anyone watching the Claude Code session would see the entire planner.go prompt rendered as the first chat bubble.~~
- ~~**Fix:** Investigate ACP spec for system-message equivalent (some hosts have a "developer message" or "system prompt" surface separate from user messages).~~
- **Resolution (afaf0c3):** ACP spec investigation (acp/types.go:147-161) confirmed no system-message channel exists — `ContentBlock.Type` is `text|image|resource` only. So priming HAS to be a user message. Mitigations: (1) prepend `InternalPromptMarker` so our log scrapers can filter primer noise from real Planner output; (2) prepend explicit `[ACT priming — do not respond. This is one-time configuration injected by the orchestrator. Acknowledge silently by emitting no text.]` so the LLM has SOME handle on "this isn't chat." The host UI bubble still exists (can't fix from our side without protocol changes) but is now clearly labeled. Test: `TestACPPrimingComposition_MarkerAndHeader`.

### 5.3 ~~[FIXED in afaf0c3] ACP priming reply is discarded with no instruction to stay silent~~
- ~~**Sources:** sub2~~
- ~~**Where:** `acp/agent.go:344` — `if _, err := a.client.Prompt(...)` only checks error~~
- ~~**What:** Model burns first turn on a useless acknowledgement that gets thrown away. No feedback loop — if host hallucinates or misunderstands, we never see it.~~
- ~~**Fix:** Prepend "Do not respond; this is configuration." to the priming message.~~
- **Resolution (afaf0c3):** captured alongside 5.2. `acp/agent.go` now captures `stopReason, err := client.Prompt(...)` and routes through a switch: `end_turn` → INFO `acp_priming_completed`; anything else (refusal, empty, error) → WARN with a hint about host misbehavior. So a misbehaving ACP host or smaller-model refusal is now visible in `~/.act/runners/*.log` instead of silent. Plus the do-not-respond header from 5.2 reduces the rate of acks landing in the first place.

### 5.4 ~~[FIXED in de479f4] Shim allowlist drift between priming advertisement and enforcement~~
- ~~**Sources:** sub2 + Path B~~
- ~~**Where:** `app.go:231` (priming advertises `act-tier1-planner`); `act_cli_whitelist.go` (real allowlist)~~
- ~~**What:** Priming says shim is on PATH; real allowlist enforces what subcommands the shim accepts. If they drift, Planner gets permission errors for commands the priming claimed were allowed.~~
- ~~**Fix:** Generate priming text from `act_cli_whitelist.go` at construction, not hand-write.~~
- **Resolution (de479f4):** the hand-coded `(status, log, etc.)` example list became `renderShimNote(role)` which reads `tools.AllowedFor(role)` live and renders the full allowed list as a bullet list. Adding a new entry to the allowlist now automatically updates the priming. Tests: `TestACPPrimingMatchesAllowlist` (4 roles) + `TestACPPrimingShimNote_NoParentheticalPlaceholders` (regression guard against re-introducing open-ended placeholders).

---

## Category 6 — Parser/prompt mismatches

### 6.1 ~~[FIXED in 805fd4e] QA role-name case ambiguity ('qa' vs 'qa_synthesizer')~~
- ~~**Sources:** sub3 + Path B~~
- ~~**Where:** `orchestrator.go:915` messageOwnershipLoop case-matches both~~
- ~~**What:** If only `qa_synthesizer` is the agent map key (per config.RoleQASynthesizer), and a stray emit tags `qa`, downstream `getAgent()` lookup fails silently.~~
- ~~**Fix:** Normalize to one canonical role name on read.~~
- **Resolution (805fd4e):** new `normalizeRole(role)` helper folded into `getAgent` so `qa` resolves to `qa_synthesizer` at the read boundary. Also reused by Fix 11's `maybeRouteQAClarification` to keep addressee normalization consistent. Test: `TestNormalizeRole` (8 cases including passthroughs).

### 6.2 ~~[FIXED in b3cc2b7] NEED_CLARIFICATION addressing — wrong audience~~
- ~~**Sources:** sub3 + Path B~~
- ~~**Where:** `autoroute_from_qa_synthesizer`~~
- ~~**What:** QA emits `NEED_CLARIFICATION: @<agent_id> <question>` targeted at a **swarm agent**. autoroute wraps it for **Planner**. Planner sees a question for someone else; ignores, mis-answers, or fabricates a CREATE_TASK to forward.~~
- ~~**Fix:** Parse the @-mention BEFORE wrapping; route to the named swarm agent's inbox, only Planner-autoroute if the @-mention is the Planner.~~
- **Resolution (b3cc2b7):** new `maybeRouteQAClarification(ctx, content)` runs the existing `clarificationRegex` BEFORE the QA branch's default autoroute. If the addressee normalizes to anything other than `planner` (and is non-empty), the verbatim QA text goes to the broadcast inbox via `client.SendMessage` (sender=`qa_synthesizer` for inbox attribution) AND a system banner surfaces the routing; the Planner autoroute is skipped. Empty/planner addressee or unknown SendMessage failure falls through to the normal autoroute so questions are never silently dropped. Prompt-side: qa_synthesizer.go updated with an addressee-precision note. Tests: `TestClarificationRegex_AddresseeExtraction` (4 sub-cases) + `TestClarificationRegex_NoMarker`.

### 6.3 ~~[FIXED in 805fd4e] taskID truncation discrepancy in failed-task autoroute~~
- ~~**Sources:** sub3 + Path B~~
- ~~**Where:** `orchestrator.go:577`~~
- ~~**What:** Visible body shows `truncate(taskID, 36)`; URL template uses full taskID. If Planner constructs URL from visible ID → 404. Compounds with the no-HTTP-tool issue (1.1).~~
- ~~**Fix:** Show the full ID OR don't suggest URL-based action OR change action to ID-based act_cli subcommand.~~
- **Resolution (805fd4e):** dropped the `truncate(taskID, 36)` from the failed-task autoroute summary text (and the batch-mode firstFailedSummary fallback). After Fix 3 the autoroute prompt instructs the Planner to use `act_cli task retry <id>` / `task abandon <id>` — a truncated visible ID would cause those calls to fail. Task IDs are short UUIDs (~24 bytes), no display concern. Compounding issue 1.1 (POST verb) was already resolved by Fix 2 + Fix 3.

### 6.4 ~~[FIXED in eabcc9b] `dependencies` field accepts multiple shapes; prompt says "empty array or omit"~~
- ~~**Sources:** sub1~~
- ~~**Where:** `basePlannerPrompt` (planner.go:24)~~
- ~~**What:** Two valid encodings advertised. Models may pick null, [], omit, or "" depending on prior turn. Parser accepts only specific forms; the prompt never specifies which.~~
- ~~**Fix:** Specify one canonical empty form, reject the others.~~
- **Resolution (eabcc9b):** belt + suspenders. (1) Prompt tightened to one canonical shape: "ALWAYS an array. Use [] when none — do NOT use null, do NOT use \"\", do NOT omit." (2) `TaskDef.Dependencies` changed from `[]string` to a new `dependencyList` named type with forgiving `UnmarshalJSON`: array → []string; null/missing/"" → nil; single string → []string{single}; number/object → error (surfaces in firstFailPreview). The previous failure mode — `""` causes the WHOLE TaskDef unmarshal to fail and the directive silently disappears — now lands the task. Tests: `TestParseCreateTaskDirectives_DependenciesShapes` (6 sub-cases) + `TestParseCreateTaskDirectives_DependenciesNumberDoesNotSilentlyDropTask`.

### 6.5 ~~[FIXED in 805fd4e] Role-count "usually" loophole~~
- ~~**Sources:** sub1~~
- ~~**Where:** `basePlannerPrompt` (planner.go:24): *"Single-file CLI → 1 role (usually developer or backend_dev)"*~~
- ~~**What:** "Usually" is the loophole. Smaller models pick backend_dev for a 50-line Go script because backend_dev's capability list includes `go`.~~
- ~~**Fix:** Tighten language to "always developer for single-file scripts unless explicitly server/API."~~
- **Resolution (805fd4e):** rewrote to "ALWAYS `developer`, unless the script is explicitly an HTTP server or DB-backed API (then `backend_dev`). Do not pick `backend_dev` just because the language is Go/Python/etc. — `backend_dev` is for server/API work, not generic scripts." Explicit ALWAYS + concrete carve-out + reasoning closes the loophole.

---

## Category 7 — System message leakage to chat / UX

### 7.1 [ACTIVE] `[SYSTEM]`-prefixed text visible in user's chat
- **Sources:** sub2
- **Where:** `resume_context_prepended` (orchestrator.go:314) raw text starts `[SYSTEM] Resuming project '%s'...`. Per `human_input_passthrough`: *"Planner role bypasses the InternalPromptMarker prepend (runAgentTurn:363-365)"*
- **What:** User sees raw `[SYSTEM]` plumbing in chat history on every resume. Same pattern for build_mode_trigger.
- **Fix:** Give Planner its own InternalPromptMarker for orchestrator-authored prompts, OR render as styled system banners instead of user-role messages.

### 7.2 ~~[FIXED in 805fd4e] `CREATE_TASK:` literal in system messages violates the rule it enforces~~
- ~~**Sources:** Path B + sub2 (Finding 5c)~~
- ~~**Where:** `resume_context_prepended`, `build_mode_trigger` both contain literal `CREATE_TASK:` string~~
- ~~**What:** basePlannerPrompt and autoroute envelope both forbid "writing CREATE_TASK in conversational prose." Yet orchestrator's own system messages do exactly that. Currently harmless (parser tolerates the no-`{`-follow case) but a parser-coupling tripwire.~~
- ~~**Fix:** Refer to the marker by description ("emit task-creation directives") rather than literal name in user-facing system messages.~~
- **Resolution (805fd4e):** scrubbed the two orchestrator-authored surfaces (`renderBriefContext` build trigger + task-list guard) to use "task-creation directives" / "you know the shape" instead of the literal `CREATE_TASK:` marker. Autoroute variant templates kept their `CREATE_TASK:` mentions because those are instruction surfaces (telling the Planner what to emit) — different from the conversational-prose prohibition. Existing `TestRenderBriefContext_AllFieldsAndTaskLists` updated to match.

### 7.3 [ACTIVE] project_context_fragment file content can silently override base rules
- **Sources:** sub1
- **Where:** `prompt.go:50` — project context injected AFTER all role rules
- **What:** If ACT.md contains conflicting guidance (e.g. "use raw shell for X"), the runtime file lands LAST, where models anchor. Project authors can subvert base behavior with no warning.
- **Fix:** Lint contextPaths content at load time; flag conflicts with base rules.

---

## Category 8 — Single-finding observations worth tracking

### 8.1 ~~[FIXED in 805fd4e] Date format US M/D/YYYY without timezone~~
- ~~**Sources:** Path B~~
- ~~**Where:** `env_block_fragment` (common.go:17)~~
- ~~**What:** Borderline confusing for absolute-time reasoning. Not currently causing drift but worth noting.~~
- ~~**Fix:** Switch to ISO 8601 with timezone.~~
- **Resolution (805fd4e):** `time.Now().Format("1/2/2006")` → `time.Now().UTC().Format(time.RFC3339)`. One-line change in `getEnvironmentInfo`. Date now shows as e.g. `2026-05-27T14:30:00Z`.

### 8.2 ~~[FIXED in a8577d2] Concurrent file walk in project_context produces non-deterministic ordering~~
- ~~**Sources:** sub1~~
- ~~**Where:** `processContextPaths` goroutine fan-out~~
- ~~**What:** Two consecutive starts can produce file blocks in different orders → defeats prefix-cache on provider side.~~
- ~~**Fix:** Sort results by path before assembling.~~
- **Resolution (a8577d2):** `processContextPaths` rewritten sequential — paths in argument order, directory walks `sort.Strings`'d by full path, dedup case-insensitive. Same output as the parallel form on the existing test (which already expected sorted order); now deterministic by construction. Provider-side prompt-cache breakpoint (Anthropic ephemeral) hits stable keys across launches. The parallelism wasn't buying much for the typical 2-3 ContextPaths anyway. Test: `TestProcessContextPaths_DeterministicOrdering` (8 consecutive runs, byte-identical).

### 8.3 ~~[FIXED in 0b5129b] Re-emission dedup is invisible to the Planner~~
- ~~**Sources:** sub1 + Path B + synthesizer~~
- ~~**Where:** orchestrator.go::checkAndRecordDispatchHash (defense exists); base prompt (no mention of why batches get silently dropped)~~
- ~~**What:** Model never learns its second emission was rejected. Same systemic pattern as `firstPlannerTurn` and `consecutiveAutoTurns` — orchestrator-side enforcement the LLM has no observability into.~~
- ~~**Fix:** Emit a system message to the Planner when dedup fires.~~
- **Resolution (0b5129b):** new `notifyDispatchDedup(hash, taskCount, age)` fires on the drop path with two channels: (a) chat banner `⏭  Skipped duplicate task batch (N tasks, hash=..., last dispatched Ns ago)` for the human, (b) autoroute with `variantSystemEscalation` + `dedupAutorouteText(taskCount, age)` body for the Planner — explicit guidance to either change a title/description and re-emit, or stop trying. Reused the existing variant (matches the "Silence is WRONG" semantics) rather than introducing a dedicated one. Tests: `TestDedupAutorouteText` + `TestDedupAutorouteText_WrappedByVariantSystemEscalation`.

### 8.4 [ACTIVE] Burst-mode autoroute shows only first failure in detail
- **Sources:** sub3 + Path B
- **Where:** `autoroute_from_system_task_burst` (orchestrator.go:594)
- **What:** Collapses N failures into ONE autoroute (good for cap), but Planner only sees ONE example (firstFailedSummary). Reassignment decisions made from incomplete data.
- **Fix:** Include short summaries for all N failures, even at higher prompt cost.

---

## Top 3 fixes — Round 1 (shipped)

1. ~~**[FIXED 770a290] 5.1 — ACPAgent.RebindSystemPrompt implementation.** Highest leverage (silently nullifies every other fix for ACP users).~~
2. ~~**[FIXED 01c1317] 1.1 — Build the missing tools (give Planner real retry/abandon affordances via act_cli, rather than stripping the verbs from prompts).** User explicitly preferred this direction over Path A's "strip lies" approach.~~
3. ~~**[FIXED ac87a1f] 4.1 + 3.1 + 2.1 — Differentiate autoroute envelope per source.** Assurance-pass defaults to silence (variantPassVerdict); Observer keeps (a)/(b)/(c) (variantAnomaly); system events get binary fork pointing at the new act_cli task retry/abandon (variantSystemEscalation). Addresses both the central "react vs silence" tension AND the autoroute duplication.~~

## Top 3 fixes — Round 2 (shipped)

4. ~~**[FIXED c853932] 1.2 — `expand_prompt_section` parity for ACP.** Investigation: dispatcher IS wired natively for in-process Planners; what was missing was an ACP-side equivalent. Per user direction (build the tool, don't strip the prompt), added `act-agent prompt-section <name>` Go CLI subcommand + allowlist entry + drift-prevention test locking the prompt's "Available sections" list to the registry.~~
5. ~~**[FIXED 3f0e8dd] 3.2 + 3.3 — Inline full brief in resume + BUILD-mode triggers.** Replaced both terse messages with `renderBriefContext` helper that emits a SPIL-style `@brief` block (all 5 fields) plus `@inFlightTasks`/`@completedTasks` partitioned lists and an explicit "do NOT re-emit CREATE_TASK for these IDs" guard.~~
6. ~~**[FIXED 4cb1d26] 4.1 — Sliding-window cascade cap.** Replaced `consecutiveAutoTurns int` with `recentAutoRoutes []time.Time`. 5 fires per 10-minute window, regardless of trigger source. Closes the three documented bypass loops (QA watchdog re-fire, Assurance verdict mirror, Observer 120s echo). Side benefit: 4.2 (Observer free-tier rate-limit risk) gets the same protection.~~

## Top 4 fixes — Round 3 (shipped)

7. ~~**[FIXED de479f4] 5.4 — Generate ACP shim priming from `tools.AllowedFor`.** Same drift class as Fix 4; same root cause (hand-coded "(status, log, etc.)" example list became fact in the LLM's worldview). Now generates the priming bullet list live from the canonical whitelist so adding a new allowlist entry automatically flows through.~~
8. ~~**[FIXED afaf0c3] 5.2 + 5.3 — ACP priming hygiene.** Spec investigation confirmed no system-message channel exists in ACP. Mitigations: `InternalPromptMarker` prefix for log scrapers + explicit `[ACT priming — do not respond]` header for the LLM + `stopReason` switch routing host hallucinations / refusals to WARN instead of silent discard.~~
9. ~~**[FIXED 9707f9a] 2.2 + 2.3 — Trim basePlannerPrompt duplication.** Removed the redundant "Allowed subcommands:" enumeration line and the "Reacting to other roles" arrow-bullet block; the shared fragments still ship those (now without competing framings in the same prompt). Net savings: ~104 bytes per Planner turn. Locked by `TestBasePlannerPromptNoFragmentDuplication`.~~
10. ~~**[FIXED eabcc9b] 6.4 — Lock dependencies shape + forgiving parse.** Belt + suspenders: prompt tightened to one canonical "always array, [] for none" shape; new `dependencyList` named type with forgiving `UnmarshalJSON` coerces recoverable shapes (null, "", single-string) to empty/single-element so the WHOLE TaskDef no longer silently disappears when a smaller model emits `"dependencies": ""`. Number/object shapes still error loudly so novel hallucinations surface.~~

## Top 4 fixes — Round 4 (shipped)

11. ~~**[FIXED a8577d2] 2.4 + 8.2 — Deterministic context-walk + content-hash skip.** `processContextPaths` rewritten sequential so two consecutive runs over the same files produce byte-identical output (deterministic-by-construction). Added `contextHash` (SHA-256) to the in-memory cache so post-`InvalidateContextCache` rebuilds on unchanged files reuse the prior string — keeps the Anthropic ephemeral-cache key stable across spurious invalidations. Audit's "no diffing every turn" framing was partly inaccurate; this is the narrow real fix.~~
12. ~~**[FIXED 805fd4e] 6.1 + 6.3 + 6.5 + 7.2 + 8.1 — Cluster of 5 small prompt + normalization fixes.** `normalizeRole(qa)→qa_synthesizer` at the `getAgent` boundary; dropped `truncate(taskID,36)` from failed-task autoroutes; tightened the role-count "usually" loophole with explicit ALWAYS + carve-out; scrubbed literal `CREATE_TASK:` from orchestrator-authored system messages (kept it where it's legitimately instructional); switched env-block date format to ISO 8601 UTC.~~
13. ~~**[FIXED b3cc2b7] 6.2 — Route `NEED_CLARIFICATION: @<agent>` to the named addressee.** `maybeRouteQAClarification` parses BEFORE the Planner autoroute; non-Planner addressees get the verbatim QA text forwarded via `client.SendMessage` (broadcast inbox + @-mention filtering) with a chat banner showing the routing. Planner addressee or SendMessage failure falls through to the normal autoroute so questions are never silently dropped.~~
14. ~~**[FIXED 0b5129b] 8.3 — Surface dispatch-hash dedup events.** When `checkAndRecordDispatchHash` drops a duplicate, fires a chat banner AND an autoroute with `variantSystemEscalation` carrying `dedupAutorouteText` ("change a title and re-emit, or stop trying"). Closes the silent-rejection blind spot the audit flagged as the same systemic class as the pre-Fix-6 cap-bypass loops.~~

When each fix is committed + tested, strike through its entry above and add a `[FIXED in commit <sha>]` annotation.

---

## Round 5a fixes (shipped)

Targeted `feat/pvm-full-implementation`; carried into `feat/remove-nomik`.

15. ~~**[FIXED 4f7fc3e] `section_examples` taught `@dependencies` inside the description string** (entry 1.2b). Rewrote the example to a top-level `dependencies` JSON property; added `TestNoSectionEmitsForbiddenDependenciesShape` walking `SectionRegistry()`.~~
16. ~~**[FIXED e1adc85] `ValidationScore` always 0 in QA synthesis prompt** (entry O.1, outside-JSON-scope). `routeToQA` now reads `t.Metadata["validationScore"]`; test `TestBuildSynthesisPrompt_IncludesValidationScore`.~~
17. ~~**[FIXED 6d934d2] `act_cli_commands_fragment` missing `prompt-section`** (entry 1.3). Added the line to `actCLICommands("planner")`; test `TestActCLICommandsFragmentMatchesAllowlist`.~~

## Round 5b fixes (shipped)

Targeted `feat/remove-nomik` (the post-Nomik-removal branch). Scope verified live against this branch — Fix 20 shrank because `planner_section_nomik.go` was already deleted here.

18. ~~**[FIXED 8e3d3a8] `variantSystemEscalation` overloaded for `synthesis_stuck` + `dedup`** (Round 5 §6 #1). Both call sites used the failed-task retry/abandon menu, but synthesis_stuck's task already PASSED and dedup created no task. Added `variantSystemNoTask` (no task menu, permits silence); re-routed both. Tests: `TestDedupAutorouteText_WrappedByVariantSystemNoTask` (rewritten from the old `..._WrappedByVariantSystemEscalation` lock), `TestRenderAutoRoutePrompt_SystemNoTaskHasNoTaskMenu`.~~
19. ~~**[FIXED ffb51e4] `variantPassVerdict` (a) CREATE_TASK escape hatch** (Round 5 §6 #3). Dropped the "obvious next step → emit CREATE_TASK" option — an open fabrication vector on PASS, worsened by the empty-`@success_criteria` fail-open. Test extended `TestRenderAutoRoutePrompt_PassVerdictHasNoReactByTakingAction`.~~
20. ~~**[FIXED 8249a19] Section bare-`act` shorthand + validation capability-lie** (Round 5 §6 #5 + new finding). `planner_section_evidence.go` rewrote `act pvm search` to the act_cli JSON shape + ACP note. `planner_section_validation.go` reworded `act validation queue` out (Planner isn't allowlisted for `validation`). Test: `TestNoSectionUsesBareActShorthand`.~~
21. ~~**[FIXED 2326d0b] `variantFailVerdict` trim — Conflict A resolution.** Kept the split (per user decision over sub3's fold), trimmed the lettered-menu overlap with anomaly to tight prose, preserving the once/twice/three-times cadence that mirrors `planner_section_validation.go`. Test extended `TestRenderAutoRoutePrompt_FailVerdictMentionsAutoGapAnalysis`.~~
22. ~~**[FIXED ac241e0] `PlannerPrompt(provider)` now branches by backend** (entry 3.5, structural enabler). Moved `ProviderACP` to `models`; `PlannerPrompt` selects the ACP shim fragment vs the in-process fragment. Test: `TestPlannerPromptBranchesOnProvider`.~~

---

## Outside-JSON-scope findings

Findings that surfaced during Round 5 audits but fall outside `planner-prompts.json`'s scope (which covers Planner-facing prompt injection points only). Tracked here so the full audit history is in one place.

### O.1 ~~[FIXED in e1adc85] `ValidationScore` always 0 in QA synthesis prompt~~
- ~~**Sources:** Round 5 scope-boundary finding — missed by both audit paths because `buildSynthesisPrompt` feeds the QA-Synthesizer, not the Planner. Surfaced during Round 5a implementation.~~
- ~~**Where:** `orchestrator.go:2370` (routeToQA) — `ValidatedOutput` struct literal; `orchestrator.go:2758` (buildSynthesisPrompt) — `"Validation score: %d/100"` format string~~
- ~~**What:** `routeToQA` constructs `ValidatedOutput` without setting `ValidationScore`. Go zero-initializes `int` fields to 0. `buildSynthesisPrompt` emits `"Validation score: 0/100"` to the QA-Synthesizer on every synthesis turn, even when Assurance scored 100. The QA-Synthesizer reads a misleading signal ("0/100") on fully-validated work and may treat it as low-quality during assembly. The real score IS available: the server writes `task.metadata.validationScore` on `task_validated` events (ChronologicalLog.ts:610); `GetValidatedTasks()` returns tasks with that metadata populated.~~
- **Resolution (e1adc85):** reads `t.Metadata["validationScore"]` before constructing the struct; handles both `int` and `float64` (Go JSON unmarshal decodes all numbers into `map[string]any` as `float64`). Populates `ValidatedOutput.ValidationScore` from the metadata value. Test: `TestBuildSynthesisPrompt_IncludesValidationScore` in `orchestrator_test.go` — feeds `ValidatedOutput{ValidationScore: 100}`, asserts prompt contains `"Validation score: 100/100"` not `"Validation score: 0/100"`.

---

## What this document is NOT

- Not a fix-implementation guide. Plan files in `/Users/user/.claude/plans/` carry the implementation specifics.
- Not a regression test. Audit gaps (especially #4 — A/B verification) live in HANDOFF.md.
- Not exhaustive. The audit covers 19 prompt-injection points; new ones added to orchestrator.go must be added to `planner-prompts.json` and surfaced here per the "how_to_keep_fresh" rules in the JSON's `_readme`.

## When to regenerate

When all entries above are struck through (or earlier on request):
1. Re-run the planner-prompts.json audit shape from scratch
2. Compare new JSON vs old — what's been fixed, what's new, what's persistent
3. Re-launch the dual-path analysis experiment for the new state
4. Update this combined-analysis.md with a fresh round
