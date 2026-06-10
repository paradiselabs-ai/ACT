# Autoroute Prompt Audit — 8 Event-Driven Triggers

## Template Duplication

The 3 `agent_autoroute` entries (`autoroute_from_observer`, `_assurance`, `_qa_synthesizer`) all share the **identical `autoRoutePlanner` wrapper** built at `act-agent/internal/app/orchestrator.go:915`. The 5 system/polling entries (`task_failed`, `task_burst`, `validation_stuck`, `synthesis_stuck`, plus the `tier1_watchdog_qa_retrigger` partially) are then wrapped by the same envelope. So **7 of 8 prompts share the wrapper boilerplate** — only `tier1_watchdog_qa_retrigger` is documented as "NOT WRAPPED" (`orchestrator.go:1544`, "this is sent directly to the qa_synthesizer agent").

Verbatim shared chunk (`orchestrator.go:915`, ~140 tokens):
> "The %s agent just sent the following report. React by taking action.\n\nDecide ONE of these, do not combine:\n  (a) Emit one or more CREATE_TASK: directives IF AND ONLY IF actual new work needs dispatching or a failed task needs reassignment. Every directive must include a non-empty title, a description carrying @task and @success_criteria SPIL sections, and explicit requiredCapabilities. NEVER emit a placeholder, empty, or acknowledgement CREATE_TASK — passing the verdict along is not a task. ... (c) Stay silent (empty response) if neither applies. ... Never write the literal string 'CREATE_TASK:' in conversational prose..."

Per-call cost: ~140 tokens × 7 wrappers ≈ ~980 tokens replayed every autoroute. Across a project with ~30 autoroute fires (Observer cycles + per-task Assurance + per-task QA), that's ~30K tokens of pure boilerplate inside the Planner's input stream — non-cacheable because the wrapper is appended to the user turn, not the system message.

## Stay-Silent vs Emit-CREATE_TASK Fork

The (a)/(b)/(c) decision tree is **textually identical** for the three `agent_autoroute` entries (one template). But the 4 system-event wrappers append a per-trigger sentence whose semantics **conflict** with (c):

- `autoroute_from_system_task_failed` (`orchestrator.go:577`): "Decide: POST /api/tasks/%s/retry... or emit CREATE_TASK:... or ask the human if this looks unrecoverable." — silence is **not on the menu**, but the wrapper still permits it.
- `autoroute_from_system_task_burst` (`orchestrator.go:594`): "Check /api/tasks for full state and decide how to handle them." — `expected_response_shape` even flags: "silence (almost certainly wrong here, but the wrapper allows it)".
- `autoroute_from_validation_stuck` (`orchestrator.go:1856`): "decide: force-retry, reassign, or abandon" — 3 options, only one (reassign) maps to CREATE_TASK; silence again technically allowed.
- `autoroute_from_synthesis_stuck` (`orchestrator.go:1978`): "decide how to proceed" — entirely unconstrained.

**Same situation, two routings**: an Assurance pass with empty `@success_criteria` (failure mode #4 on `autoroute_from_assurance`) → wrapper says "(c) Stay silent... is the correct response to a routine pass verdict" → bad deliverable advances. But morally the same broken-verdict state, surfaced via `autoroute_from_validation_stuck`, says "force-retry, reassign, or abandon." Identical underlying bug, opposite Planner action depending on which path catches it.

## Cascade Risk Beyond the Counter

`consecutiveAutoTurns ≤ 5` is reset on **human input only** (`human_input_passthrough` failure_modes: "Resets consecutiveAutoTurns and lastObserverPromptHash"). Cascades the counter doesn't catch:

1. **`tier1_watchdog_qa_retrigger` → `autoroute_from_qa_synthesizer` self-perpetuating loop.** From its failure_modes: "If QA emits no marker (parseSynthesisResponse returns 'in_progress'), the QA poll's seen-key is NOT set, so the next poll cycle re-routes — burning tokens with no progress." The watchdog fires QA, QA produces a non-marker reply, the reply autoroutes back to Planner *and* the next polling tick re-fires QA. Each watchdog fire is a fresh trigger lineage so `consecutiveAutoTurns` resets toward 0 between cycles.

2. **Assurance verdict mirror loop** (`autoroute_from_assurance` failure_modes): "Smaller models latch onto the structure and try to mirror it as a CREATE_TASK with the same criteriaResults as @success_criteria." If the placeholder CREATE_TASK *does* slip past the server seam, it dispatches to a swarm agent → completes → Assurance fires again → autoroute again. Each is one autoroute turn but a fresh task lineage, so the cap counts only the back-to-back chain, not the cross-task chain.

3. **Observer ↔ Planner echo loop** (`autoroute_from_observer` failure_modes): "Planner models still occasionally restate the Observer message in their own words, which Observer then sees and reacts to on the next cycle." Observer cycle is ~120s, so it dodges the back-to-back consecutive counter entirely.

## Error-Channel vs Work-Channel Confusion

The escalation prompts drift **toward CREATE_TASK** despite being status escalations:

- `autoroute_from_system_task_failed` asks for a `POST /api/tasks/:id/retry` the Planner cannot make (failure_modes: "the Planner has no Bash/curl tool — only act_cli with subcommands {status, context, log, graph, pvm, message, codebase}. None of those POSTs to /retry"). So the legitimate "retry" action collapses to "emit CREATE_TASK" — error channel becomes work channel.
- `autoroute_from_validation_stuck` similarly: "force-retry, reassign, or abandon... none of these have first-class Planner-visible tool affordances. Reassign = CREATE_TASK, fine. Force-retry = no tool exists. Abandon = no tool exists." The prompt manufactures a 3-option menu where only one is actionable; Planner picks CREATE_TASK by default, or emits a no-op "abandoning task X" chat reply.
- `autoroute_from_synthesis_stuck` ("decide how to proceed") doesn't even attempt to inform the human — pure ask-drift.
- Contrast `autoroute_from_system_task_burst`: leans informational ("Check /api/tasks for full state") with no mention of human escalation at all.

**No escalation prompt explicitly says "inform the human; this is a dead end."** All four imply Planner has agency it lacks.

## Free-Form Prose Escape

Two of the eight have a **structured downstream parser**: the three agent autoroutes feed `handlePlannerTaskDirectives` which scans for `CREATE_TASK:` markers — but the wrapper permits free chat as branch (b)/(c), so prose is the path of least resistance for the LLM. The other five (`task_failed`, `task_burst`, `validation_stuck`, `synthesis_stuck`, `tier1_watchdog_qa_retrigger`) have **no per-trigger response schema** — `synthesis_stuck`'s `expected_response_shape` is literally "Chat reply OR CREATE_TASK if rework is needed."

Compare to `tier1_watchdog_qa_retrigger` (the unwrapped one), which **does** specify a hard schema: "SYNTHESIS_COMPLETE: <summary> OR NEED_CLARIFICATION: @<agent> <q>" with a real parser (`parseSynthesisResponse`). That's the only entry in the set with an enforced output shape — and notably it's also the only one that doesn't use the autoroute envelope. The envelope itself is the drift vehicle: every wrapped prompt inherits "Write a short chat reply IF AND ONLY IF the human needs to be informed" — prose-permissive language overrides any structured-response intent the trigger had.

## Top 3 Drift Surfaces (Prioritized)

1. **Strip the `act_cli` capability-lie from system-event prompts.** `task_failed`, `task_burst`, `validation_stuck`, `synthesis_stuck` all instruct the Planner to take actions (POST /retry, check /api/tasks, force-retry, abandon) it has no tool to perform. This is the highest-leverage fix because it directly creates malformed `act_cli` calls + no-op chat replies that look like progress. Rewrite each to a binary fork: "(1) CREATE_TASK to reassign to <suggested role>; (2) message the human if the failure is unrecoverable." Drop unimplemented verbs.

2. **Differentiate the autoroute envelope per source.** Today, `autoroute_from_assurance` and `autoroute_from_observer` share text → Planner sees the same "(a) emit CREATE_TASK" invitation after a passing Assurance verdict as after a real Observer anomaly. This is the documented #1 drift surface ("MOST FREQUENT DRIFT SURFACE per recent commits 7156822, c237c0e, f522d1c"). Branch the wrapper: Assurance-passes default to silence with no (a) option visible; Observer reports default to (a)/(b)/(c).

3. **Add a cascade-aware cap that survives lineage hops.** `consecutiveAutoTurns` resets on human input but not across QA→Planner→QA bounce or Observer-echo-loop. The watchdog + QA non-marker reply ("burning tokens with no progress") and the Observer echo are both invisible to the current cap. A simple fix: count autoroute turns within a sliding wall-clock window (e.g. 5 in 10 min) regardless of trigger source, with the cap reset only on a human turn.
