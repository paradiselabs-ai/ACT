# Experiment results — Path A (3 subagents + synthesizer) vs Path B (single-agent self-analysis)

Both paths analyzed the same 19-entry Planner prompt audit. Path A split the audit 6/5/8 across three subagents, then a fourth synthesizer combined them. Path B was the main thread reading all 19 in one pass.

---

## Findings only Path A caught

These are real findings Path B missed. The split-then-synthesize approach surfaced them because each subagent had more focused attention on its subset, AND because the synthesizer could reason about the system from THREE views simultaneously.

1. **"Same situation, two routings" — sub3 only.** An Assurance-pass-with-empty-criteria gets routed via `autoroute_from_assurance` → wrapper says "(c) Stay silent is correct for routine pass" → bad deliverable advances. The morally identical broken-verdict state surfaced via `autoroute_from_validation_stuck` says "force-retry, reassign, or abandon." Same underlying bug, opposite Planner action depending on which path catches it. **Path B noted both prompts individually but didn't connect them as routing the same condition to opposite actions.**

2. **"3 loosely-coupled prompt-injection surfaces, no layer ground-truthing" — synthesizer only.** The synthesis pass explicitly named the systemic frame: composition layer advertises tools the runtime forbids, lifecycle layer binds a worldview that may never refresh, event layer re-permits what composition forbade. Path B saw each individually but never named the meta-pattern. The synthesizer could only see this by being forced to reconcile three different sub-views.

3. **"No prompt-side observability of orchestrator parser state" — synthesizer only.** Dispatch-hash dedup, `firstPlannerTurn` flag, `consecutiveAutoTurns` cap — three orchestrator-side enforcements the Planner never sees. From the LLM's view, every turn is fresh; every rejection silent. Path B noticed each individually but never abstracted them into "the prompt has zero feedback channel from enforcement."

4. **"Dependencies: empty array or omit if none" ambiguity — sub1 only.** The base prompt offers two valid encodings for empty deps; the parser accepts only one. Path B missed this entirely; sub1 caught it because it was reading the static fragments at high resolution.

5. **"Role-count 'usually' loophole" — sub1 only.** *"Single-file CLI → 1 role (usually developer or backend_dev)"* — sub1 noted smaller models pick backend_dev for a 50-line Go script. Path B missed the soft-pluralization specifically.

6. **Observer 120s echo loop dodges the consecutive counter — sub3 only.** Sub3 documented three loops that bypass `consecutiveAutoTurns`, including Observer's 120s cycle resetting the counter naturally. Path B noted Observer spam as a category but didn't enumerate the specific cap-bypass mechanism.

7. **"Conflicting interpretations" section — synthesizer only.** The synthesizer surfaced cases where sub1 and sub3 propose opposite fixes (enrich the affordance description vs narrow when it's visible). This kind of "the experts disagree" finding is structurally impossible for a single-agent path.

---

## Findings only Path B caught

These are real findings the Path A pipeline missed despite its higher per-entry attention.

1. **The "rule violates itself" pattern.** Path B framed it explicitly: the orchestrator's `resume_context_prepended` and `build_mode_trigger` system messages contain the literal `CREATE_TASK:` string, which the basePlannerPrompt forbids in conversational prose. Sub2 caught the marker-in-prompt as a parser tripwire (Finding 5c) but didn't name "the rule the orchestrator enforces is violated by the orchestrator's own prompts."

2. **The empty-criteria → 100% → silent Planner → bad deliverable LADDER.** Path B traced this as a single flowing failure chain. Path A's sub3 caught the "same situation, two routings" version (which is sharper) but didn't trace the full ladder. Sub1 saw it as "Planner doesn't repair empty criteria." Neither subagent owned the end-to-end propagation.

3. **Linking specific recent commits (7156822, c237c0e, f522d1c) to their place in the ladder.** Path B referenced the commit history inline because those commits are in my main-thread context. Subagents don't have that history loaded — they could only reference what's in the JSON's `failure_modes_observed` field. Path A only mentions commits where the JSON entries quote them.

4. **Token economics that include the autoroute envelope as a system-prompt line item.** Path B's token table includes `autoroute envelope ~700` as a per-fire cost. Sub3 calculated the per-fire amplification (~30K/project) but didn't put it in the same table as the static fragments. Synthesizer noted the order-of-magnitude difference between sub1 and sub3 priorities but didn't present it as a unified table.

5. **Specific NEED_CLARIFICATION addressing bug.** Path B explicitly named that QA's `NEED_CLARIFICATION: @<agent_id>` is targeted at a swarm agent, but the autoroute wraps it for the Planner. Sub3 mentioned it in passing under `autoroute_from_qa_synthesizer` failure modes but didn't elevate it as a structural addressing error.

---

## Where both paths converged (high-confidence findings)

1. **ACPAgent.RebindSystemPrompt is a silent no-op** — both ranked this as critical. Path A made it #1 in top-5 ranking; Path B made it #2.
2. **Capability-lying in stuck/failed autoroutes** — both flagged "POST /retry", "force-retry", "abandon" as actions the Planner has no tool to perform. Path A made it #2 in top-5; Path B made it #3.
3. **Empty-CREATE_TASK loop after Assurance pass** — both identified as a major drift. Path B made it #1; Path A made it #3 (after differentiating autoroute envelope).
4. **Autoroute envelope duplication** — both flagged ~140 token wrapper replicated across 7 prompts. Path A quantified per-project cost more rigorously.
5. **Resume context missing fields** — both caught description+techStack only; successCriteria/agentsInvolved/tasks dropped.
6. **`expand_prompt_section` advertised but not implemented** — both flagged. Sub1 elevated to its top-3; Path B kept it in "additional findings."

The convergence rate on the top findings is ~90% — both paths agree on what the major drift surfaces are.

---

## Methodological observations

| Dimension | Path A (3 subs + synth) | Path B (single agent) |
|---|---|---|
| **Total tokens consumed** | ~250K (3 subs ~65K each + synth ~66K) | ~30K |
| **Wall-clock time** | ~3 min parallel + ~2 min synth = ~5 min | ~1 min |
| **Unique findings count** | 7 | 5 |
| **Convergent findings** | ~90% on top items | ~90% on top items |
| **Per-entry attention depth** | High (each sub focused on 6/5/8) | Medium (one pass through 19) |
| **Cross-entry pattern detection** | Stronger for systemic patterns (synthesizer's job) | Stronger for flowing failure chains |
| **Failure modes** | Synthesizer falsely believed the harness blocked file writes; had to save the synthesis manually from text return | One-pass blind spots — no internal disagreement to surface "conflicting interpretations" |
| **Best at** | Naming meta-patterns, surfacing disagreements, deep verbatim quoting | Tracing cross-category failure chains, real-time codebase references, faster |
| **Worst at** | Cost (~8× tokens, ~5× wall-clock if synth is serial), losing some single-entry detail in synthesis flattening | Missing patterns that only emerge by reading subsets in isolation |

### Specific surprises

1. **Path A was actually FASTER wall-clock** because the 3 subagents ran in parallel and I did Path B during their run. Total experiment wall-clock: ~5 min. If subagents had been serial, Path A would have been ~3× slower than Path B.

2. **The synthesizer mis-detected a harness restriction.** It claimed *"The tool blocked the file write — I'll return the synthesis as text per the harness rules"* — this is false; Write is not blocked for subagents. The synthesizer guessed wrong and self-corrected by returning text. The parent thread (me) had to manually save it.

3. **The synthesizer's "Conflicting interpretations" section is the experiment's highest-leverage output.** A single-agent path structurally cannot disagree with itself. Surfacing places where two perspectives recommend opposite fixes (sub1 wants to enrich affordance descriptions; sub3 wants to narrow when affordances appear) is genuinely new information.

4. **Subset blinders are real and predictable.** Sub2 couldn't see capability-lying because lifecycle prompts don't grant tool affordances. Sub1 couldn't see cascade loops because static fragments don't fire on a cycle. The synthesizer correctly named these blinders. Path B implicitly avoided them by reading everything, but at the cost of less per-entry depth.

5. **Path A surfaced ONE genuinely novel finding (the "3 layers, no ground-truthing" frame) that I would not have produced on a longer single-agent pass.** It required the structural separation of three views to become visible.

### When to prefer which

**Use Path A (split + synthesize) when:**
- The audit surface has natural categorical splits (this one did: static / lifecycle / event)
- Finding meta-patterns and structural disagreements matters
- You want explicit "what does each sub-view miss" analysis
- You have token budget to burn

**Use Path B (single agent) when:**
- The audit needs cross-category trace chains (failure ladders, full pipeline view)
- You need real-time codebase references the subagent context wouldn't have
- Token economy matters
- Wall-clock matters AND you can't parallelize

**For this specific audit:** Path A added ~7 unique findings worth knowing (especially "same situation, two routings" + "3-layer framing" + "conflicting interpretations"). Path B missed those but added 5 unique findings of its own. **Both together** produce the best audit — neither alone is complete.

---

## What this experiment doesn't measure

- **Accuracy**: I cannot verify either path's findings without running the actual code changes and measuring drift reduction. Both are claims, not validations.
- **Stability**: Run the same experiment twice and you'd get different unique-findings lists. The convergent ~90% would likely hold; the divergent 10% would shift.
- **Synthesizer quality**: I gave the synthesizer a structured prompt. A worse synthesizer prompt would collapse to "averaging" rather than surfacing real disagreement.
- **Scale**: Both paths handled 19 entries. At 50+ entries Path A's parallelism likely dominates more sharply; at 5 entries Path B's simplicity wins.
