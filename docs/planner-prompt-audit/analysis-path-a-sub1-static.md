# Planner Prompt Subset Audit — 6 Static / Composition Fragments

## Internal contradictions

**1. CLI status commands — fragment says "available to you," base prompt says "do NOT run them."**
`act_cli_commands_fragment` (common.go:71) lists `act-agent status`, `act-agent log --tail 20`, `act-agent context --project <name>` as plainly "available to you." But `base_planner_prompt_fragment` (planner.go:24) contradicts this directly:
> "DO NOT run act_cli to answer the human's status/log/swarm queries... act_cli is for *routing evidence during decomposition*, not for status reporting."
The fragment that *enumerates* the commands does not carry the "decomposition-only" constraint. A model reading top-down will see status/log offered, then later be told not to use them — two voices, no reconciliation.

**2. Role-boundary rules stated twice with different framings.**
`base_planner_prompt_fragment` ("Reacting to other roles"): *"Assurance rejects → gap analysis is auto-sent to the agent; only intervene on repeated failures"* — describes Assurance as a peer whose verdicts trigger conditional Planner action. `coordination_constraints_fragment` (common.go:187) restates the same boundary as a hard prohibition: *"NEVER validate task outputs yourself — that's Assurance's job."* Both are technically compatible, but the constraints fragment's own `failure_modes_observed` flags this: *"These constraints repeat the role-boundary guidance from basePlannerPrompt's 'Reacting to other roles' section — duplication that costs ~50 tokens every turn and gives the model two slightly-different framings of the same rule."*

## Token bloat

Rough byte counts of `raw_text` (verbatim, excluding the schema metadata):

| Fragment | Bytes | Notes |
|---|---|---|
| `base_planner_prompt_fragment` | ~6,400 | Necessary — carries SPIL/CREATE_TASK shape rules |
| `act_cli_commands_fragment` | ~700 | Mostly duplicative of basePlannerPrompt's `act_cli` block |
| `coordination_constraints_fragment` | ~470 | Pure restatement (see contradiction #2) |
| `env_block_fragment` | ~70 | Cheap, useful |
| `project_context_fragment` template | ~80 | Template only; runtime expansion is the cost driver |
| `static_system_prompt_inprocess` | composition wrapper only | n/a |

**The bloat is concentrated in `act_cli_commands_fragment` + `coordination_constraints_fragment` (~1,170 bytes ≈ 300 tokens) of fragment that the base prompt already covers.** Worse: `project_context_fragment` injects entire files (`ACT.md`, `ACT.local.md`) on every turn with no diffing — the template is cheap but the runtime substitution dominates. The CLAUDE.md Phase 3 deltas already note Tier 1 LLM requests dropped from ~22K→~5-7K by trimming `defaultContextPaths`, confirming this is the dominant lever.

## Ambiguity surfaces

**1. "Use it via Bash" vs "do NOT shell out" (cross-fragment, but `act_cli_commands_fragment` is the seed).**
The fragment opens: *"You are an in-process Tier 1 role. You speak by writing plain text in your reply – do NOT shell out to send messages."* But the ACP-backed priming (`acp_priming_prompt`, outside this subset) appends *"Use it via Bash for all ACT-coordination subcommands."* For the in-process Planner the fragment is intentionally restrictive; for ACP it is contradicted. A hardened CLI seeing the in-process fragment will still wonder whether `act-agent status` is something it should call (it's listed!) or just know-about-but-not-call.

**2. `expand_prompt_section` advertised, not necessarily wired.**
`base_planner_prompt_fragment`: *"You have an `expand_prompt_section` tool. ... Pull a section ONLY when you need it."* Its own `failure_modes_observed` admits: *"expand_prompt_section tool is advertised here but no orchestrator-side dispatch was located during this audit — verify wiring or remove the advertisement."* A model that takes the offer will either hallucinate a tool call or stall.

**3. "Empty array or omit if none."**
`base_planner_prompt_fragment` on `dependencies`: *"Empty array or omit if none."* Two valid encodings, plus the JSON-on-one-line strict rule. Models may pick `null`, `[]`, omit, or `""` depending on prior turn. The parser accepts only specific forms — but the prompt never specifies which.

**4. Role-count guidance soft-pluralizes.**
*"Single-file CLI / <5 success_criteria / one language → 1 role (usually developer or backend_dev)"* — "usually" is the loophole. Smaller models pick `backend_dev` for a 50-line Go script because backend_dev's capability list includes `go`.

## Missing guardrails

**1. Re-emission of identical CREATE_TASK batches.**
`base_planner_prompt_fragment.failure_modes_observed`: *"Re-emission of identical CREATE_TASK batches across two turns (~20s apart) by flaky free-tier models... Defended at orchestrator.go::checkAndRecordDispatchHash but the prompt does not yet tell the Planner why batches are getting silently dropped on the second emit."* No fragment tells the Planner about the dispatch-hash dedup — the model never learns its second emission was rejected.

**2. ACP context-cache staleness.**
`project_context_fragment` and `static_system_prompt_inprocess` rely on `InvalidateContextCache + RebindSystemPrompt`. The in-process path is fixed; the ACP path is silently a no-op (per `acp_priming_prompt` failure modes). No fragment addresses ACP refresh — and the subset fragments give the Planner no way to tell which backend it's running on.

**3. Concurrent context-walk ordering defeats prefix caching.**
`project_context_fragment.failure_modes_observed`: *"Concurrent file walk via processContextPaths goroutine fan-out has non-deterministic ordering — two consecutive starts can produce system messages with file blocks in different orders, defeating any prefix-cache on the LLM provider side."* Pure infrastructure leak; no prompt-side fix possible, but worth a Go-side sort.

**4. Empty-success_criteria pass-through.**
Touched by `autoroute_from_assurance` (outside subset) but originating constraint lives in `base_planner_prompt_fragment` ("REQUIRED — Assurance validates at 95%"). The base prompt mandates `@success_criteria` exists; no fragment instructs the Planner to *reject or repair* a brief that arrived without enough success criteria. Assurance fail-closes to 100% on empty criteria — drift is in the Planner's silence about input quality.

**5. Drift between `act_cli_commands_fragment` and the in-prompt `act_cli` schema rule.**
`act_cli_commands_fragment.failure_modes_observed`: *"the act_cli tool schema still receives malformed args (e.g. `\"args\":\"unverified\"` as a bare string) — the JSON-shape rule for that lives in basePlannerPrompt, not here, so a Planner that ignores the upstream prose still sees the act-agent CLI advertisement and tries the wrong shape."* Co-locate the rule with the affordance.

## Compose-order dependencies

The composition order (from `static_system_prompt_inprocess.compose_order`) is fixed: base → act_cli → constraints → env → project_context. Dependencies:

1. **`act_cli_commands_fragment` redefines a meta-rule first stated in `base_planner_prompt_fragment`** ("CREATE_TASK and PROJECT_BRIEF are markers in your reply text, not shell commands"). If the order were inverted, the second appearance in `base` would feel like canonical; today the *first* appearance in `act_cli_commands` lacks the JSON-shape rule that follows. Reordering breaks the layered "here are the commands → here's how not to misuse them" gradient.

2. **`coordination_constraints_fragment` assumes role names (Observer, Assurance, QA/Synthesizer) have already been introduced** in `base_planner_prompt_fragment`'s opening sentence (*"You run in a shared NesTTY window with Observer, Assurance, and QA"*). If `base` is ever swapped for a different role prompt that reuses common.go (and developer.go does — see common.go shared by all roles), the constraints fragment dangles role names without antecedent.

3. **`project_context_fragment` (last) can override prior rules silently.** If `ACT.md` contains conflicting guidance (e.g. instructs the Planner to dispatch via shell), the runtime-injected file lands AFTER all the no-shell rules, and models often anchor to the latest authoritative-looking text. Failure mode noted in fragment: *"If ACT.md or ACT.local.md contains the literal string `CREATE_TASK:`, the Planner will be reminded of the marker syntax."* The injection point is correct (latest = most-attended) for project-specific override, but means project authors can accidentally subvert base behavior with no warning.

4. **`env_block_fragment`** has no semantic dependency on neighbors — order-safe.

## Top 3 drift surfaces (prioritized)

1. **Collapse `act_cli_commands_fragment` + `coordination_constraints_fragment` into `base_planner_prompt_fragment`.** They duplicate role boundaries, restate the "markers ≠ shell commands" rule in a less specific form, and live in `common.go` (shared by all roles) so they can't carry Planner-specific JSON-shape rules where the affordance lives. Estimated savings: ~300 tokens/turn × every Planner turn × every rebind. Closes contradiction #1 + ambiguity #1.

2. **Co-locate `act_cli` schema rules with the command list.** The "args is ALWAYS an array" rule sits in `base_planner_prompt_fragment` ~5K tokens after the command enumeration. Move it adjacent to the command list (or eliminate the duplicate enumeration in `act_cli_commands_fragment` entirely). Closes the `act_cli_commands_fragment` documented drift directly.

3. **Either wire or remove `expand_prompt_section`.** As written, `base_planner_prompt_fragment` advertises an LLM-visible tool whose orchestrator handler the audit could not find. Every Planner turn includes the offer; some fraction of turns will attempt to invoke it. Free token cost + nonzero hallucination risk + the fragment's own admission this needs verifying.
