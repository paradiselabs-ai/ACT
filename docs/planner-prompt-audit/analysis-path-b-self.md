# Planner-prompt audit — Path B (single-agent self-analysis)

Analyzed by the main thread reading all 19 entries in one pass. Independent of the 3 subagent path.

---

## 1. Structural contradictions across fragments

### 1a. "Stay silent" vs "React by taking action" — the central tension
- **basePlannerPrompt** (`planner.go:24`, base_planner_prompt_fragment): *"Be concise. Don't narrate what you're about to do — just do it."*
- **autoroute envelope** (`orchestrator.go:915`, every autoroute_from_*): opens with *"React by taking action."*
- These read as opposites to a literal model. The base prompt says don't narrate, act. The autoroute says react (which the model parses as "respond"). The fix lives in the autoroute's (c) clause — *"Stay silent (empty response) if neither applies. Silence is the correct response to a routine pass verdict."* — but only Claude-Code-grade adherence reaches that nuance. Smaller models (and Claude Code occasionally) treat the opening line as the directive and produce a placeholder CREATE_TASK or an acknowledgement.
- This is the #1 drift surface — every Assurance pass triggers it.

### 1b. "Never write CREATE_TASK in prose" — repeated 3 times, breaks the rule itself
- basePlannerPrompt warns about prose mentions (line 60).
- coordination_constraints_fragment is silent on it.
- autoroute envelope repeats: *"Never write the literal string 'CREATE_TASK:' in conversational prose"*.
- BUT `resume_context_prepended` and `build_mode_trigger` are themselves system messages that contain the literal `CREATE_TASK:` string. Per the audit's own note, these are framed as Planner instructions, but a model echoing the instruction back ("I'll emit CREATE_TASK: directives for ...") gets flagged as malformed by the parser.
- The rule is policed inconsistently — the orchestrator's prompts violate the rule the orchestrator enforces.

### 1c. Coordination constraints duplicate base-prompt content
- `coordination_constraints_fragment` (common.go:187): *"You are the ONLY decision-maker… NEVER spawn agents… NEVER monitor ChronLog… NEVER validate… NEVER assemble deliverables"*.
- basePlannerPrompt already says the same in its identity line ("the only decision-maker") and in the "Reacting to other roles" section.
- ~50 token duplication per turn × every Tier 1 call. Two slightly-different framings of the same rule = model picks the easier one to satisfy.

---

## 2. Tool affordance ≠ prompt promise (Planner asked to do things it can't)

| Prompt | Asks Planner to | Tool available |
|---|---|---|
| `autoroute_from_system_task_failed` | `POST /api/tasks/%s/retry` | None. act_cli has {status, context, log, graph, pvm, message, codebase}. No retry subcommand. No HTTP tool. |
| `autoroute_from_validation_stuck` | "force-retry, reassign, or abandon" | Only "reassign" exists (CREATE_TASK). No retry, no abandon. |
| `autoroute_from_synthesis_stuck` | "decide how to proceed" | No menu at all — Planner has to guess. |
| `basePlannerPrompt` (line 60) | "You have an `expand_prompt_section` tool" | **Per audit findings: no orchestrator-side dispatch found.** The tool is advertised but not implemented. Planner either fabricates calls or wastes prompt budget. |

**Observed downstream effect:** Planner fabricates act_cli calls with unknown subcommands, which the schema rejects, leaving the original problem unresolved. Or it writes a chat reply like "abandoning task X" which has zero orchestrator effect.

---

## 3. ACP backend is structurally weaker than in-process

### 3a. Priming as USER message, not system message
- `acp_priming_prompt` (app.go:231): The Planner's whole system prompt is sent as the first **user** message in a fresh ACP session, not as a system prompt.
- ACP hosts that treat user messages as conversation-level (rather than guidance) will surface the role definition as if the human said it.
- The agent's reply to priming is **discarded** (acp/agent.go:344). No feedback loop. If the host hallucinates or misunderstands, we never see it.

### 3b. RebindSystemPrompt is a no-op on ACPAgent
- `system_prompt_rebind` (orchestrator.go:991) calls RebindSystemPrompt on every Tier 1 agent after PROJECT_BRIEF accepted.
- ACPAgent.RebindSystemPrompt is `return nil` (acp/agent.go:457).
- Consequence: AGENTS.md materializes after intake, but the ACP-backed Planner never sees the refreshed context. Symmetric to the pre-2026-05-19 bug for the in-process path, **but no fix has landed for the ACP path.**

### 3c. Shim allowlist drift
- Priming advertises `act-tier1-planner` shim on PATH.
- Real allowlist lives in `act_cli_whitelist.go`.
- If those drift, Planner gets permission errors for commands the priming claimed were allowed. No automated check holds them in sync.

---

## 4. Mode-transition gaps (INTAKE → BUILD → ongoing)

### 4a. Resume context only carries 2 of 5 brief fields
- `resume_context_prepended` (orchestrator.go:314) substitutes `description` + `techStack` only.
- `successCriteria`, `agentsInvolved`, `constraints` are silently dropped from the resume message.
- BUT basePlannerPrompt's INTAKE mode requires all 5 to skip intake.
- Planner sees 2/5 fields, may re-enter intake despite the [SYSTEM] nudge that says "do NOT run intake."

### 4b. build_mode_trigger doesn't include the brief
- `build_mode_trigger` (orchestrator.go:1022) tells the Planner to "decompose the project brief into tasks" — but doesn't include the brief content.
- The Planner has to recall it from its conversation history (still there) OR from AGENTS.md (only if rebind worked).
- Post-compaction: brief content may have been summarized away. Edge case becoming likely as sessions extend.

### 4c. build_mode_trigger uses fireWhenPlannerIdle which can time out silently
- 200ms polls up to 60s waiting for Planner to be idle.
- If Planner's intake-mode turn takes >60s (free-tier slow path), build_mode_trigger is dropped with `planner_trigger_timeout` warn — system sits idle forever, user sees nothing happening.

---

## 5. Cascade and rate-limit risks

### 5a. Observer poll → autoroute spam
- Observer fires every ~120s.
- Even when Planner correctly stays silent (option c), Observer fires again next cycle if anomalies aren't resolved.
- consecutiveAutoTurns=5 cap stops it but only warns — Observer keeps firing visible to the user, with no Planner reaction. Looks broken.
- Related: user reported rate limits despite 8/9 agents on claude-code. Observer is the 1 outlier (free-tier minimax). Every silent Planner turn STILL triggers Observer's next cycle, which makes a free-tier LLM call.

### 5b. tier1_watchdog_qa_retrigger creates loops too
- Watchdog fires QA directly when pendingSynthesis > 0 and QA idle > 5min.
- QA's reply autoroutes Planner (autoroute_from_qa_synthesizer).
- Planner usually stays silent (correct).
- But the round-trip costs a Claude Code API call per cycle. Compounds with synthesizedAt's "going forward only" semantics — old synthesis_complete events get replayed but QA-poll re-decides each restart.

### 5c. The empty-criteria → 100% pass → silent Planner ladder
- If Assurance gets a task with no @success_criteria (still possible from old data or from the rare Planner emission that survives the new gate), it returns 100% by default (per assurance-fail-closed-empty-criteria-2026-05-26).
- Planner sees a "pass" and per autoroute_from_assurance, silence is correct.
- Deviating deliverable ships to QA unchallenged. Two-layer validation = zero-layer validation in this case.

---

## 6. Parser-vs-prompt mismatch

### 6a. Inconsistent QA role naming
- `autoroute_from_qa_synthesizer` (orchestrator.go:915): `messageOwnershipLoop` case-matches both `qa` and `qa_synthesizer`.
- If only `qa_synthesizer` is the configured agent map key (per config.RoleQASynthesizer), and a stray emit tags `qa`, lookup fails silently downstream.
- No drift observed yet, but a latent footgun.

### 6b. NEED_CLARIFICATION addressing
- QA emits `NEED_CLARIFICATION: @<agent_id> <question>` targeted at a specific **swarm agent**.
- autoroute wraps it for the **Planner**.
- Planner sees a question intended for someone else and either ignores it, mis-answers it, or fabricates a CREATE_TASK to forward it.
- The autoroute envelope doesn't tell the Planner "this @-mention is for a different agent — route it, don't answer it."

### 6c. taskID truncation discrepancy
- `autoroute_from_system_task_failed` shows `truncate(taskID, 36)` in the visible body but the URL uses the **full** taskID.
- If Planner constructs the URL from the visible ID → 404.
- Worse, no Planner-visible tool POSTs URLs anyway (see §2) — so the discrepancy is moot in practice but reflects deeper sloppiness in how prompts are templated.

---

## 7. Token economics

| Fragment | Approx bytes | Per-turn cost |
|---|---|---|
| basePlannerPrompt | ~6,200 | High — every turn |
| coordination_constraints_fragment | ~520 | Duplicates basePlannerPrompt content (~50 tokens net waste) |
| act_cli_commands_fragment | ~720 | Includes act-agent CLI commands the Planner uses rarely |
| env_block_fragment | ~80 | Cheap |
| project_context_fragment | variable | ACT.md + ACT.local.md content; can balloon |
| autoroute envelope | ~700 | Repeated on every autoroute (4+ trigger points × N cycles) |

The autoroute envelope is the single highest leverage compression target: every triggered Planner turn pays for it, and it's near-identical across observer/assurance/qa/system. Could be factored into a system-prompt addendum sent once and referenced by trigger ID in each autoroute call.

---

## Top 3 drift surfaces to fix next

1. **The placeholder-CREATE_TASK loop after Assurance pass.** Even with the planner-prompt fix (7156822) + server-side reject (c237c0e), Claude Code drifts ~3 times per session into emitting empty acknowledgement CREATE_TASKs. The autoroute prompt has THREE places talking about CREATE_TASK rules; consolidate to one, and consider switching the post-pass autoroute path to **not** ask "react by taking action" at all — instead post `task_validated` as an informational chat-rendered system event with no Planner turn, eliminating the trigger entirely.

2. **ACP-backed Planner has no mid-session context refresh.** RebindSystemPrompt is a no-op for ACPAgent. After PROJECT_BRIEF lands, the ACP host still operates on its priming-time context (no AGENTS.md, no project description). For long sessions this guarantees drift. Either send a fresh `session/new` after rebind (lose conversation history) or push a "context update" user message that the host treats as canonical. Both are non-trivial; the latter is cheaper.

3. **Tool-affordance mismatch in stuck/failed autoroutes.** Validation stuck, synthesis stuck, and task_failed prompts all suggest actions the Planner has no tool to perform. This trains the Planner to fabricate calls. Either implement the missing affordances (`act_cli task retry`, `act_cli task abandon`) or rewrite the prompts to only mention real options. The current state is worse than not having the autoroute at all.

---

## Additional findings worth tracking

- **`expand_prompt_section` tool advertised but possibly not implemented** — flagged in basePlannerPrompt's failure_modes_observed. Verify wiring or strip the advertisement.
- **Date format US M/D/YYYY without timezone** in env_block_fragment — borderline confusing for absolute reasoning.
- **Concurrent file-walk in project_context_fragment** produces non-deterministic ordering — kills prefix-cache potential on provider side.
- **Resume context literal `CREATE_TASK:`** in system messages — see §1b — minor but breaks the rule the orchestrator enforces.
- **Burst-mode autoroute** (`autoroute_from_system_task_burst`) shows only the FIRST failed task summary — Planner makes reassignment decisions from incomplete data when N tasks fail together.
