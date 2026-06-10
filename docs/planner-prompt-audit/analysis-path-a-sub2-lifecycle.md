# ACT Planner Prompt Audit — Lifecycle Subset (5 entries)

## 1. Mode-transition gaps

**Finding 1a — BUILD trigger doesn't restate the brief.** `build_mode_trigger` (orchestrator.go:1022) only says *"Project '%s' has been created. Switch to BUILD mode now..."*. The Planner must reconstruct the brief from in-thread history or hope `RebindSystemPrompt` succeeded. After context compaction the brief can vanish silently. Quote: `"Do not ask for confirmation — start creating tasks immediately."` — but doesn't say *create tasks from what*.

**Finding 1b — Resume mode and INTAKE collide when fields are empty.** `resume_context_prepended` (orchestrator.go:314) is the only "you are in BUILD now" signal on a resume turn, but it inlines `description: %s | techStack: %s`. If `GetProject` envelope-unwraps wrong (a documented current bug per the entry's `failure_modes_observed`), the Planner reads `description:  | techStack: ` and "its prompt prioritizes intake when no brief content is visible." Mode label says BUILD; absence of fields says INTAKE; the latter wins.

**Finding 1c — No mode echo back to orchestrator.** None of the 5 prompts asks the Planner to declare its current mode in its response. The orchestrator infers mode by parsing markers (`PROJECT_BRIEF:` vs `CREATE_TASK:`). A Planner that emits prose ("Okay, switching to build mode now…") with no marker produces zero observable mode signal — orchestrator's `intakeMode` flag and the LLM's belief drift independently.

## 2. Per-session contamination

**Finding 2a — Resume prompt invites task duplication.** `resume_context_prepended` says *"decompose the brief into tasks and emit CREATE_TASK: directives"* with **no reference to already-completed or in-flight tasks**. A project resumed mid-build will get its task list re-emitted from scratch. The entry confirms: `"Successcriteria and agentsInvolved from the resumed project are NOT included...The Planner has to guess the rest or call act-agent context."` — and nothing about existing tasks at all.

**Finding 2b — ACP path can't refresh AGENTS.md mid-session.** From `system_prompt_rebind` failure modes: `"ACPAgent.RebindSystemPrompt() is a no-op — the ACP-backed Planner never gets its context refreshed mid-session."` Combined with `acp_priming_prompt` being sent *before* AGENTS.md exists, an ACP Planner carries an empty-project worldview for the entire session even after intake completes — a permanent contamination in the other direction (stale-empty rather than stale-old).

**Finding 2c — `firstPlannerTurn` flag is the only guard.** `human_input_passthrough` only prepends resume context once (`o.firstPlannerTurn = false` flips after use). No equivalent "completed tasks list" injection exists at all — so the cleanest cross-session continuity signal we ship is a one-shot string the LLM may compress out of context on turn 2.

## 3. ACP vs in-process equivalence

**Finding 3a — ACP priming uses USER role; in-process uses system message.** Quote (acp_priming_prompt failure modes): `"Priming is sent as a USER message, not a system message. ACP hosts that treat user messages as conversation-level (rather than guidance) will surface the role prompt as if the human said it."` In-process binds via `provider.WithSystemMessage` (system_prompt_rebind raw_text). Result: the human in an ACP-backed session can see the planner prompt as their own utterance in the conversation view.

**Finding 3b — Rebind is silently a no-op for ACP.** `system_prompt_rebind` iterates `["planner","observer","assurance","qa_synthesizer"]` and calls `RebindSystemPrompt()`, but for ACPAgent this returns nil without doing anything (acp/agent.go:457). The in-process path picks up the freshly-written AGENTS.md; ACP doesn't. There is no compensating "re-prime" path for ACP after PROJECT_BRIEF lands.

**Finding 3c — Discovery skew.** acp_priming_prompt appends `"[ACT] The CLI \`act-tier1-planner\` is on your PATH..."` — but the in-process path doesn't say that, because in-process uses native tool wiring rather than a shim. If a user switches Planner backend mid-project, the LLM's mental model of how to invoke ACT differs by backend silently.

## 4. Acknowledge vs act ambiguity

| Entry | Explicit instruction? |
|---|---|
| acp_priming_prompt | **Implicit "stay silent"** — reply discarded, but never told that. Will likely produce an acknowledgement message that gets thrown away. |
| system_prompt_rebind | **N/A** — no LLM call, just provider rebinding. Clean. |
| human_input_passthrough | **Free-form** — no shape constraint at the passthrough level; relies on base prompt to know when to emit PROJECT_BRIEF/CREATE_TASK. |
| resume_context_prepended | **Action-explicit** — `"do NOT run intake. Switch immediately to BUILD mode: decompose…and emit CREATE_TASK: directives."` Good. |
| build_mode_trigger | **Action-explicit** — `"Do not ask for confirmation — start creating tasks immediately."` Best of the set. |

**Finding 4a — Priming reply is wasted tokens.** Discarding the reply (`acp/agent.go:344`) without telling the model to stay silent burns the LLM's first turn on a useless acknowledgement. Should explicitly say "Do not respond; this is configuration."

**Finding 4b — Passthrough has no shape contract.** `human_input_passthrough` literally pipes user text to `runAgentTurn`. Whether the Planner replies in chat, emits markers, or both depends entirely on the Planner inferring intakeMode from context — no per-turn directive.

## 5. System-marker injection footprint

**Finding 5a — `[SYSTEM]` prefix is visible to user.** `resume_context_prepended` raw text begins `[SYSTEM] Resuming project '%s'...`. Per `human_input_passthrough`: `"Planner role bypasses the InternalPromptMarker prepend (runAgentTurn:363-365), so this content IS rendered in the chat as a normal user message."` So the user sees raw `[SYSTEM]` plumbing in their chat history on every resume.

**Finding 5b — build_mode_trigger likely renders too.** It fires via `runAgentTurn("planner", ...)` and the same Planner-bypass applies. The user will see `"Project 'foo' has been created. Switch to BUILD mode now..."` appear as if they typed it.

**Finding 5c — Marker-in-prompt false positives.** Both resume_context and build_mode contain the literal `CREATE_TASK:`. Per resume failure modes: `"if a Planner echoes this resume context, the echo could be flagged as malformed by the orchestrator's parser"`. The parser tolerates it (no `{` follows) but this is a parser-coupling tripwire if `parseCreateTaskDirectives` ever tightens.

**Finding 5d — Priming hidden from user but visible in ACP host UI.** acp_priming_prompt is sent as a user-role message in a fresh ACP session — the ACP host (e.g., Claude Code) may render the entire role prompt as the first chat bubble. ACT's TUI doesn't show it, but anyone hosting ACT through ACP sees the whole planner.go prompt as "user said this."

---

## Top 3 drift surfaces

1. **ACPAgent.RebindSystemPrompt is a silent no-op** (acp/agent.go:457). This makes the entire AGENTS.md materialization pipeline a placebo for ACP-backed Planners. Either implement a re-prime via a fresh ACP message, or refuse to start an ACP Planner until the brief exists. Highest leverage because it nullifies the in-process fix and is silent in logs.

2. **Resume context omits everything that matters** (orchestrator.go:314). Only `description + techStack` are inlined; `successCriteria`, `agentsInvolved`, **and existing tasks/progress** are not. Add (a) a completed/in-flight task list, (b) the full brief fields, (c) explicit "do not re-emit CREATE_TASK for these IDs" to stop duplicate-task contamination on every resume.

3. **`[SYSTEM]`-prefixed text rides into the chat view** because Planner bypasses `InternalPromptMarker`. Either give Planner its own internal marker for orchestrator-authored prompts (resume_context_prepended, build_mode_trigger) so the user never sees raw `[SYSTEM]`-prefixed plumbing, or render these as styled system banners instead of user-role messages. Fix unlocks cleaner UX and removes the marker-echo parser tripwire (Finding 5c).
