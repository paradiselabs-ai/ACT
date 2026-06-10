# Round 5 — Path B (single-agent self-analysis)

Single-pass read of all 29 prompt entries in `planner-prompts.json` and the resolution notes in `combined-analysis.md`. Goal: surface what the current Planner-prompt system actually does, where the 4 rounds of fixes succeeded, and where new drift surfaces have emerged.

---

## What the audit looks like post-Round-4

29 entries split across 4 architectural layers:

1. **Composition fragments** (5) — always included in every system prompt: base prompt, act_cli commands enumeration, coordination constraints, env block, project context.
2. **System-prompt assembly** (3) — the two backends + the priming wrapper for ACP.
3. **Lifecycle + dynamic** (4) — rebind, human input, resume context, BUILD trigger.
4. **Autoroute family** (9) — 4 variant templates + 5 system-event call sites.
5. **On-demand sections + tool surfaces** (8) — 5 sections + 2 invocation paths + the act_cli tool schema.

The Round-4 fixes introduced helpers (`renderBriefContext`, `renderShimNote`, `wrapPriming`, `dedupAutorouteText`) that consolidate previously-divergent surfaces. They closed 25 of the original 27 [ACTIVE] drift entries. The audit JSON's `failure_modes_observed` field flags the closed entries with their fix SHA and includes NEW drift that emerged from the fixes themselves.

---

## Verified-closed entries (spot checks)

I sampled the closed entries by reading the cited code locations against the resolution-note claims. All cleanly verified:

- **5.1 (770a290)** — `acp/agent.go:468-493` discards sessions on rebind. ✓
- **2.4 + 8.2 (a8577d2)** — `prompt.go:63-90` sequential + SHA-256 hash skip. ✓
- **6.3 (805fd4e)** — `orchestrator.go:617` shows full taskID (no `truncate(taskID, 36)`). ✓
- **5.4 (de479f4)** — `app.go:270-278` `renderShimNote` reads `AllowedFor` live. ✓
- **3.2 + 3.3 (3f0e8dd)** — `renderBriefContext` emits all 5 brief fields. ✓
- **7.2 (805fd4e)** — `renderBriefContext` uses "task-creation directives" not literal `CREATE_TASK:`. ✓
- **8.1 (805fd4e)** — `common.go:25` uses `time.RFC3339` UTC. ✓
- **1.2 (c853932)** — `expand_prompt_section` tool wired natively AND `prompt-section` CLI subcommand exists for ACP. ✓ (with one caveat — see HIGH-1 below)

No drift detected on the closed entries themselves. The post-fix code matches the resolution-note claims.

---

## CRITICAL — the on-demand examples section teaches the forbidden shape

`section_examples` at `planner_section_examples.go:33` contains:

> `### Task with dependencies`
> `EXAMPLE_TASK: {"title":"Add Pygame rendering layer","description":"@task\n> Render the existing Snake game state with Pygame.\n@dependencies\n- Snake game core loop must be complete\n@success_criteria\n- ...","requiredCapabilities":["python","pygame"],"priority":"medium"}`

The base prompt at `planner.go:78` says:

> *"Do NOT use `@context`, `@dependencies`, or any other `@`-section in the description string. Dependencies go in the top-level JSON `dependencies` array. Putting `@dependencies` in the description breaks the JSON parser silently — your tasks will not be created."*

The on-demand `examples` section is what the Planner pulls *when shape is unclear*. It demonstrates the exact pattern the base prompt forbids. The `dependencies` JSON-level array is missing entirely from this example.

**Failure mode:** smaller model emits a CREATE_TASK after pulling `examples`, copies the `@dependencies`-inside-description shape, the parser drops it silently (`Dependencies` field never populates, sibling tasks never depend on it), the dependent task runs in parallel against unwritten prerequisites.

**Fix:** rewrite the example to use a top-level `dependencies` array — `"dependencies":["Snake game core loop"]` — and drop the `@dependencies` SPIL section from the description.

This is the highest-leverage finding in the whole audit.

---

## HIGH — drift surfaces from the fixes themselves

### HIGH-1: `act_cli_commands_fragment` missing `prompt-section`

The Round-3 Fix 1.2 (c853932) added the `prompt-section` CLI subcommand and allowlisted it. Three drift-prevention surfaces are locked: `tools.AllowedFor('planner')` (canonical), `renderShimNote` reads it live (ACP priming gets it), `TestPromptSectionAdvertisementMatchesRegistry` locks the prompt's "Available sections" list to `SectionNames()`.

But the `act_cli_commands_fragment` at `common.go:79` was NEVER updated to include `prompt-section`. The fragment enumerates 9 commands and is missing this one. ACP Planners read this fragment for the canonical CLI surface (along with the `renderShimNote`); the fragment shows 9 entries, the shim note shows 10. Different sets.

**Failure mode:** ACP Planner reads the fragment, doesn't see `prompt-section`, doesn't pull on-demand sections via CLI. The shim note's `prompt-section` entry might be missed in the longer composed text.

**Fix:** add `act-agent prompt-section <name>` to the planner fragment in `common.go`. One line.

### HIGH-2: `synthesis_stuck` routed via wrong variant

`autoroute_system_synthesis_stuck` fires when QA can't assemble a deliverable after 3 attempts. The fromContent string is `"synthesis stuck on task %q (id=%s) after %d attempts; QA cannot assemble the deliverable"`. It's wrapped in `variantSystemEscalation` whose action menu is:

> `(a) act_cli task retry <id>` — re-dispatch a failed task
> `(b) act_cli task abandon <id>` — mark the task permanently failed
> `(c) Emit a CREATE_TASK: directive to reassign`
> `(d) Write a short chat reply`

For a synthesis-stuck event, the task is already **validated** — Assurance passed it. The issue is QA assembly. Retrying or abandoning a validated task corrupts downstream state (the deliverable was supposed to include this work). The Planner is being offered options (a)/(b) that don't apply.

**Failure mode:** smaller model picks (a) `task retry` on a validated task. The server may accept or reject depending on state-machine guards — either way, confusion.

**Fix:** either (1) introduce `variantSynthesisStuck` with appropriate action menu (clarify, dispatch follow-up task, inform human) or (2) fix the fromContent text to explicitly say "the task is validated; do NOT retry/abandon — pick (c) clarify or (d) inform human." The latter is cheaper; the former is cleaner.

### HIGH-3: `variantPassVerdict` (a) escape hatch is the placeholder-CREATE_TASK vector

`autoroute_variant_pass_verdict` opens with *"No action is required by default."* — the fix Round 1 (ac87a1f) shipped. But option (a) is still there:

> *"If the verdict unblocks an obvious next step (e.g. a dependent task), emit CREATE_TASK directives for that next step. Every directive must include a non-empty title, @task + @success_criteria SPIL sections... NEVER emit a placeholder or acknowledgement CREATE_TASK."*

This is the same `(a) emit CREATE_TASK` vector that motivated the variant split in the first place. Smaller models (glm-4.5-air:free observed previously) fire (a) with a placeholder when they see a PASS verdict and think "I should do something." The "NEVER emit a placeholder" guard helps but doesn't close the door — Round 1 closed it via the server-side reject in commit c237c0e, but the prompt-side door is still open. A future regression of the server-side filter would re-expose this.

**Failure mode:** placeholder CREATE_TASK on PASS verdict slips past server filter (or filter regresses) → bad deliverable dispatches.

**Fix:** demote (a) to "only if the brief explicitly defines a dependent task that the just-passed work unblocks; otherwise stay silent." More restrictive language. OR remove (a) entirely — passing verdicts shouldn't ever trigger a CREATE_TASK from the Planner's view; if a task has dependencies, the dispatch should be triggered by the orchestrator's dependency resolver, not by Planner inference.

### HIGH-4: Assurance fail-open on empty `@success_criteria`

Documented in `autoroute_variant_pass_verdict` failure_modes (NEW finding):

> *"Assurance fail-open on empty @success_criteria (kanban: assurance-fail-closed-empty-criteria-2026-05-26): a task submitted with zero criteria gets score=100, Passed=true. parseValidationVerdict returns Passed=true. The Planner sees this template and defaults to silence — correct behavior given the prompt, but based on a fraudulent pass. The orchestrator does not detect empty criteriaResults before routing."*

The prompt-side behavior is correct (silence on pass). The orchestrator-side gap is that empty-criteria tasks pass validation trivially. The Planner has no way to know the pass is fraudulent.

**Failure mode:** Planner sees PASS → stays silent → bad deliverable advances → eventually surfaces as deliverable bug downstream.

**Fix:** orchestrator-side, `parseValidationVerdict` should treat `len(criteriaResults) == 0` as a fail-shaped event (different variant, or special-cased fromContent). The Planner can then react with a CREATE_TASK that includes proper criteria, OR the orchestrator can refuse to fire the verdict at all and instead fire `autoroute_system_validation_stuck` with reason="no criteria to score against."

---

## MEDIUM drift surfaces

### MEDIUM-1: On-demand sections use the user-CLI form, not either Planner-accessible path

`section_evidence_routing` and `section_nomik` both use `act pvm search "<query>"` / `act codebase onboard` shorthand. Neither backend executes that:

- **In-process Planner** must use `{"subcommand":"pvm","args":["search","<query>"]}` (JSON tool call).
- **ACP Planner** must use `act-tier1-planner pvm search "<query>"` (the shim binary, not the user-facing `act` symlink).

The bare `act` form is the developer-facing CLI (the `/opt/homebrew/bin/act` symlink). A Planner that copies the example invocation verbatim fails.

**Failure mode:** Planner pulls `evidence_routing` section, sees `act pvm search "..."`, emits that as a tool call shape that doesn't exist. Tool call fails or misroutes.

**Fix:** standardize the section text to show both invocation paths, OR pick one canonical form (the JSON tool form, since both backends ultimately route through it). Same drift exists in `nomik`.

### MEDIUM-2: `variantFailVerdict` requires Planner to infer attempt count

The variant says *"If this is a repeated failure (3+ attempts), use act_cli task abandon"* but the prompt text contains no attempt count. The Planner must infer from conversation history — vulnerable to compaction, and on a fresh ACP session (after rebind) the history may not include prior failures.

The orchestrator HAS the attempt count (`maxValidationAttempts=3` cap). It just doesn't pass it through to the autoroute fromContent.

**Fix:** include the attempt number in `autoroute_system_validation_stuck` fromContent (already does via `%d attempts`) AND in the variantFailVerdict template — surface the count explicitly so the Planner has direct evidence for the "is this attempt 3+?" decision.

### MEDIUM-3: `[SYSTEM]` literal visible in user chat (audit 7.1 still active)

`resume_context_prepended` and `build_mode_trigger` produce text starting with `[SYSTEM] Resuming project...` / `[SYSTEM] Project '...' has been created`. The Planner role bypasses `InternalPromptMarker` at `runAgentTurn:363-365` — orchestrator-authored prompts for the Planner appear in the chat history as user-role messages.

**Failure mode:** UX clutter; user sees raw plumbing on every resume + every project creation.

**Fix:** Planner-specific marker that the TUI strips before render (mirror of how Observer/Assurance/QA already work with `InternalPromptMarker`). Or render orchestrator-authored prompts as styled banners, not user messages.

### MEDIUM-4: `autoroute_system_dedup` uses variantSystemEscalation but action menu doesn't apply

Same pattern as HIGH-2 (synthesis_stuck). Dedup means the Planner re-emitted an identical batch within 60s. The action menu offers retry/abandon/CREATE_TASK reassign/human notification — none of these are the right action for dedup. The dedup-specific guidance ("change a title and re-emit OR stop trying") lives in `dedupAutorouteText` (fromContent), which prepends the variant's action menu. The Planner reads both.

**Failure mode:** smaller model picks (a) `task retry` on the duplicate batch — but there's no failed task to retry. The batch was just dropped. Confusion.

**Fix:** mild — adjust `dedupAutorouteText` to lead with "These options below DO NOT apply to dedup; only the guidance above applies." OR introduce a new variant. Same trade-off as HIGH-2.

### MEDIUM-5: 3.4 (no mode echo) still active

None of the lifecycle prompts ask the Planner to declare its current mode. `resume_context_prepended` says "Switch immediately to BUILD mode" but never asks for confirmation. `build_mode_trigger` says "Switch to BUILD mode now" but doesn't require an echo. The orchestrator's `intakeMode` flag and the Planner's LLM-side belief can drift independently.

**Failure mode:** Planner thinks it's still in INTAKE after a resume; emits PROJECT_BRIEF on the next turn; orchestrator (intakeMode=false) ignores it. Symptom: Planner appears unresponsive after resume.

**Fix:** add a one-line ack expectation to the lifecycle trigger prompts (`"Reply with 'BUILD ack' on your next response to confirm."`). Or add a mode-declaration parser that watches for any reply on the trigger and re-fires if absent.

---

## CONFLICTING INTERPRETATION I want to flag

The prior audit listed **3.5 ("Use it via Bash" vs "do NOT shell out") as a contradiction**. I previously called it a re-interpretation issue, not a true contradiction. Reading the current code:

- `act_cli_commands_fragment` (common.go:79) opens with: *"You speak by writing plain text in your reply — do NOT shell out to send messages."* — this is about how the Planner SPEAKS to humans / other agents. Messages are reply text, not shell commands.
- `renderShimNote` in `acp_priming_wrapper` says: *"Use it via Bash for all ACT-coordination subcommands."* — this is about how to INVOKE the shim binary (the act-tier1-planner CLI). The shim IS invoked via Bash; that's how ACP backends call CLI binaries.

These describe two different operations (messages vs. CLI invocations). NOT a contradiction once parsed precisely. But the framing is confusing — both phrases ship in the same priming text. A smaller model might conflate "do not shell out" with "do not use bash for anything" and skip the shim CLI entirely.

I disagree with the prior audit's framing, but the underlying ergonomic concern is real: the wording should make the two operations distinct so they can't be confused. e.g. *"You speak via reply text only — never via shell. For CLI tools, invoke act-tier1-planner via Bash as documented below."*

---

## Top 5 ranked drift surfaces (Round 5 priority)

1. **section_examples teaches forbidden @dependencies shape** (CRITICAL) — high-leverage because the section is pulled exactly when the Planner is unsure about shape. Self-defeating example. ~5-line fix.
2. **act_cli_commands_fragment missing prompt-section entry** (HIGH) — one-line fix; closes the last drift-prevention gap from Round 3's Fix 1.2.
3. **HIGH-2 + MEDIUM-4: variantSystemEscalation action-menu mismatch for synthesis_stuck and dedup** (HIGH) — wrong-variant-for-trigger pattern. Two trigger sites where the action menu doesn't apply. Either tighten fromContent or introduce dedicated variants.
4. **HIGH-3: variantPassVerdict (a) escape hatch is still the placeholder-CREATE_TASK vector** (HIGH) — close the door entirely, or restrict (a) to dependency-resolved-by-orchestrator cases.
5. **HIGH-4: Assurance fail-open on empty criteria** (HIGH, orchestrator-side fix) — `parseValidationVerdict` should refuse to return Passed=true on empty criteriaResults. The prompt-side correctly stays silent; the orchestrator needs to detect fraud first.

The first two are 1-line fixes. The third is a template rewrite. The fourth is a prompt edit + maybe an orchestrator detection. The fifth is an orchestrator change with downstream prompt impact (different variant fires).

---

## Bottom line

Four rounds of fixes closed every drift entry they targeted. No regressions detected. But the fixes themselves added new prompt surfaces (5 on-demand sections, 4 variants, 5 system-event call sites, ACP priming hygiene) — and three of those new surfaces have their own drift: examples teaches the wrong shape, two variants are used in trigger contexts they don't fit, and the fragment-side enumeration didn't stay in sync with the allowlist.

Pattern: **drift-prevention tests catch composition-time drift (allowlist ↔ priming, sections ↔ prompt list) but don't catch semantic drift (variant template ↔ trigger semantics).** Round 5 should fix the 5 surfaces above AND consider extending the test discipline to catch variant-vs-trigger mismatches (e.g. assert each system-event call site picks a variant whose action menu is semantically valid for that event).
