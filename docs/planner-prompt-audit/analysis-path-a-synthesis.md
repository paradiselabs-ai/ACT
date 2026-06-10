# Planner Prompt Audit — Path A Synthesis

Three sub-analyses examined 19 prompt entries split as: **sub1** (6 static/composition fragments), **sub2** (5 lifecycle/human prompts), **sub3** (8 autoroute/event prompts).

## 1. Cross-cutting themes

**Theme A — Capability-lying (sub1 + sub3).** Both audits caught the Planner being told it can do things it cannot. Sub1: `act_cli_commands_fragment` enumerates `act-agent status / log / context` as "available to you," then `base_planner_prompt_fragment` later forbids using them for status answers. Sub3 named the same disease at a different layer: the system-event wrappers (`task_failed`, `validation_stuck`, `synthesis_stuck`) instruct the Planner to "POST /api/tasks/:id/retry", "force-retry", or "abandon" — actions for which **no tool affordance exists**. Sub3: *"the Planner has no Bash/curl tool — only act_cli with subcommands {status, context, log, graph, pvm, message, codebase}."* Same pathology surfacing in both the static composition and the event triggers.

**Theme B — Duplicated/restated rules costing tokens (sub1 + sub3).** Sub1: `coordination_constraints_fragment` "repeats the role-boundary guidance from basePlannerPrompt … duplication that costs ~50 tokens every turn." Sub3 documented the systemic version: the `autoRoutePlanner` envelope is replayed verbatim across 7 of 8 autoroute prompts (~140 tokens × 7 ≈ ~980 tokens, *"non-cacheable because the wrapper is appended to the user turn, not the system message."*). Two scales of the same problem.

**Theme C — Mode/intent ambiguity (sub1 + sub2 + sub3) — all three.** Sub1: "use it via Bash" vs "do NOT shell out" cross-fragment contradiction. Sub2: BUILD-mode trigger doesn't restate the brief, and "Mode label says BUILD; absence of fields says INTAKE; the latter wins." Sub3: stay-silent vs emit-CREATE_TASK fork — *"silence (almost certainly wrong here, but the wrapper allows it)"*. **The only theme appearing in all three subsets — highest-confidence systemic finding.**

**Theme D — Prompt content leaking into the human-visible chat (sub1 + sub2).** Sub1: `project_context_fragment` injects file contents that can override prior rules silently. Sub2: `[SYSTEM]`-prefixed resume context renders as a user message because Planner bypasses `InternalPromptMarker`; ACP priming surfaces as a chat bubble in the host UI. Both ends of the pipe leak.

## 2. Single-analyst surfaces with system-wide implications

**ACPAgent.RebindSystemPrompt is a silent no-op (sub2 only).** Sub2 quote: *"the ACP-backed Planner never gets its context refreshed mid-session… the entire AGENTS.md materialization pipeline a placebo for ACP-backed Planners."* Sub1 couldn't see this because its subset is pre-bind composition; sub3 couldn't see it because autoroute triggers don't touch bind state. The implication is system-wide: every fix sub1 proposes to fragment composition is silently a no-op for ACP users, and every autoroute wrapper sub3 wants to differentiate is being injected into a Planner whose worldview was frozen at priming time.

**Cascade-aware cap missing (sub3 only).** Sub3 documented at least three loops that `consecutiveAutoTurns ≤ 5` doesn't catch (QA watchdog re-fire, Assurance verdict mirror, Observer 120s echo). Sub1 only sees composition; sub2's lifecycle prompts reset the cap rather than triggering it. The duplicated wrappers sub1 noticed and the marker leaks sub2 noticed compound here — every uncapped loop replays both.

**Empty-success_criteria pass-through (sub1, with sub3 corroboration).** Sub1 noted the base prompt mandates `@success_criteria` but never instructs the Planner to reject a brief that arrived without enough. Sub3 found the downstream effect: Assurance-pass autoroute with empty criteria → wrapper tells Planner *"(c) Stay silent… is the correct response"* → bad deliverable advances. Sub2 couldn't see this because its lifecycle prompts handle mode transitions, not content validation.

**Tier 1 visibility of dispatch-hash dedup (sub1 only).** *"the prompt does not yet tell the Planner why batches are getting silently dropped on the second emit."* Pure prompt-side gap invisible to sub2/sub3.

## 3. Conflicting interpretations

**On token bloat priority** — sub1 ranks "Collapse `act_cli_commands_fragment` + `coordination_constraints_fragment`" as its #1 drift (~300 tokens/turn). Sub3 implicitly disagrees: the autoroute wrapper duplication is ~980 tokens × ~30 fires per project ≈ ~30K tokens, an order of magnitude larger. Both real; priority differs because sub1 saw only per-turn cost and sub3 saw per-fire amplification.

**On stay-silent permissions** — sub1's static fragments treat silence as a safe default for boundary respect (don't validate, don't shell out). Sub3 calls silence *"almost certainly wrong"* in the system-event context. Same word, opposite normative load — and the wrapper inherited by both doesn't disambiguate.

**On where to fix `CREATE_TASK:` shape rules** — sub1 wants the JSON-shape rule co-located with the command enumeration (move into `act_cli_commands_fragment`). Sub3 wants the autoroute wrapper to *remove* the (a) CREATE_TASK option entirely for Assurance-pass paths. Not strictly contradictory but opposing directions: sub1 enriches the affordance description, sub3 narrows when it's visible.

## 4. The complete picture

The three subsets together describe a Planner prompt system with **three loosely-coupled prompt-injection surfaces** — static composition (sub1), lifecycle transitions (sub2), event triggers (sub3) — none of which know about each other's invariants:

- **Composition layer** advertises tools the runtime forbids or hasn't wired (sub1: `expand_prompt_section`, status commands).
- **Lifecycle layer** binds a worldview that may never refresh (sub2: ACP no-op rebind, resume context omits brief + task state).
- **Event layer** appends per-fire wrappers that re-permit the very behaviors composition forbade (sub3: shell-shaped "POST /retry" actions, prose escape via (c)).

The systemic pattern: **every layer optimistically describes what the Planner can do; no layer ground-truths against what the orchestrator will actually accept.** Token economics worsen across layers — sub1's ~300/turn becomes sub3's ~980/fire × 30/project. Capability-lies compound. Mode confusion multiplies — sub2's BUILD-vs-INTAKE ambiguity is re-entered every autoroute because the wrapper makes no mode check.

**There is no prompt-side observability of the orchestrator's parser state.** Dispatch-hash dedup (sub1), `firstPlannerTurn` flag (sub2), `consecutiveAutoTurns` cap (sub3) are all orchestrator-side enforcement the Planner never sees. From the LLM's view, every turn is fresh; every rejection silent; every duplicate emission "succeeded."

## 5. Top 5 ranked drift surfaces

1. **ACPAgent.RebindSystemPrompt no-op** (sub2). *Highest leverage* — nullifies every fix the other two analyses propose for ACP users. Fix: `act-agent/internal/acp/agent.go:457` — implement re-prime via fresh ACP message, or refuse to start ACP Planner without a brief. Impact: makes the rest of the audit actually take effect for ACP-backed sessions.

2. **Strip capability-lies from system-event wrappers** (sub3, corroborated by sub1's static-layer version). Fix: `orchestrator.go:577,594,1856,1978` — rewrite each to binary fork: "CREATE_TASK to reassign, or message the human." Drop "POST /retry", "force-retry", "abandon", "check /api/tasks". Impact: kills the dominant malformed-`act_cli` / no-op-chat failure mode; resolves Theme A directly.

3. **Differentiate the autoroute envelope per source** (sub3). Fix: `orchestrator.go:915` — Assurance-pass defaults to silence (no (a)), Observer keeps (a)/(b)/(c), system events get binary fork. Impact: the documented #1 drift per commits 7156822/c237c0e/f522d1c; also unlocks per-fire token savings.

4. **Resume context omits brief fields and existing tasks** (sub2). Fix: `orchestrator.go:314` — inline full `successCriteria + agentsInvolved`, completed/in-flight task list with explicit "do not re-emit CREATE_TASK for these IDs." Impact: stops silent task-duplication; partial-fixes Theme C by removing the empty-fields → INTAKE false signal.

5. **Collapse + co-locate static composition duplication** (sub1). Fix: `common.go:71,187` + `planner.go:24` — fold `act_cli_commands_fragment` and `coordination_constraints_fragment` into base, co-locate `act_cli` JSON-shape rule with command enumeration, remove or wire `expand_prompt_section`. Impact: ~300 tokens/turn + closes contradiction #1 + closes advertise-without-wire hallucination vector.

---

## Gaps in the audit itself (for next iteration)

- **No live token-count measurement** — all three subsets estimated bytes/tokens by inspection. A turn-by-turn capture against a real session would confirm sub3's ~30K/project autoroute boilerplate.
- **No subset covered Tier 2 (swarm) prompts** — audit is Planner-only, but Theme A (capability-lying) very likely repeats in `developer.go`, `backend_dev.go`, etc.
- **No subset examined the prompts seen by Observer/Assurance/QA on autoroute** — sub3 looked at how those agents trigger the Planner, but not what *they* receive. Same envelope-duplication may apply symmetrically.
- **No A/B verification** — recommendations assume "removing X reduces drift." A small eval (e.g., 20 Assurance-pass autoroutes before/after envelope differentiation) would convert ranking-by-judgment into ranking-by-measured-effect.
