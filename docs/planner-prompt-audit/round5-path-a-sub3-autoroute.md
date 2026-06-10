# Round 5 — Path A sub3 — Autoroute Family (9 entries)

Scope: 4 `renderAutoRoutePrompt` variants + 5 `variantSystemEscalation` call sites. Post-Fix-3 + Fix-12 differentiated machinery, with the sliding-window cap (Fix 6) replacing the old `consecutiveAutoTurns` counter.

---

## 1. Per-entry close read

### 1.1 `autoroute_variant_anomaly` (orchestrator.go:1106)
What Planner sees: *"The %s agent just sent the following report. React by taking action."* + the legacy `(a) CREATE_TASK / (b) chat reply / (c) Stay silent`. Used for Observer reports, QA passthroughs, **and** unparseable Assurance fallthrough.
- CORRECT post-R4: action tree retained only here, where the trigger is genuinely ambiguous. `NEVER emit a placeholder ... CREATE_TASK` lockdown is verbatim and load-bearing.
- SUSPICIOUS: QA `SYNTHESIS_COMPLETE` messages still land here as "unparseable Assurance fallthrough" per the entry's own failure note. The (a) option *invites* a CREATE_TASK in response to a completion event — exactly the placeholder-CREATE_TASK vector smaller models keep tripping (glm-4.5-air observed).

### 1.2 `autoroute_variant_pass_verdict` (orchestrator.go:1071)
What Planner sees: *"The Assurance agent posted a PASS verdict. No action is required by default."* + options `(a) If the verdict unblocks an obvious next step ... emit CREATE_TASK` and `(b) Stay silent (empty response). This is the correct default.`
- CORRECT: silence-default framing, "Do NOT acknowledge the pass in chat. Do NOT echo the verdict back" is sharp. Test `TestRenderAutoRoutePrompt_PassVerdictHasNoReactByTakingAction` guards the regression.
- STILL SUSPICIOUS: (a) is an open invitation. "obvious next step (e.g. a dependent task)" has no boundary — a confused model can invent one. Combined with the empty-criteria fail-open (kanban: `assurance-fail-closed-empty-criteria-2026-05-26`), the Planner can be told "silence is correct" on a fraudulent pass. Prompt is fine; the orchestrator gate upstream is the gap.

### 1.3 `autoroute_variant_fail_verdict` (orchestrator.go:1082)
What Planner sees: *"The Assurance agent posted a FAIL verdict. Gap analysis has already been auto-routed to the swarm agent — they will re-attempt the task without your involvement."* + `(a) If this is a repeated failure (3+ attempts) ... use act_cli task abandon <id> ... and emit a CREATE_TASK to reassign`, `(b) chat reply`, `(c) Stay silent. This is the correct default for a first or second failure`.
- CORRECT: silence-default for first/second fail is the right shape. `act_cli task abandon` reference matches the affordance Fix 1.1 built.
- STILL SUSPICIOUS: "(3+ attempts)" requires the Planner to *infer attempt count from history* — vulnerable to compaction. The orchestrator already knows the attempt count (it caps polling at `maxValidationAttempts=3`) but doesn't surface it in `fromContent`. So the Planner is told "stay silent" at what may genuinely be attempt 3, while a *separate* validation-stuck escalation fires at the same moment.

### 1.4 `autoroute_variant_system_escalation` (orchestrator.go:1093)
What Planner sees: *"The orchestrator surfaced a system event that requires Planner action. Silence is WRONG here."* + four options: `(a) act_cli task retry <id>`, `(b) act_cli task abandon <id>`, `(c) Emit a CREATE_TASK: directive to reassign`, `(d) Write a short chat reply to inform the human`. Includes the inline tool-call shape: `{"subcommand":"task","args":["retry","<id>"]}`.
- CORRECT post-R3: "Silence is WRONG here" is the inverse of variantPassVerdict — clean fork. Inline JSON example is the right move for tool-shy models.
- STILL SUSPICIOUS: this is one template doing five jobs. For `synthesis_stuck` and `dedup`, options (a)/(b) are semantically wrong (task already validated, or task wasn't even created). The Planner is shown a four-option menu when only (c)/(d) apply. The template doesn't condition on the event type.

### 1.5 `autoroute_system_task_failed_single` (orchestrator.go:617)
fromContent: *"Task %s just failed (agent %s). Error: %s"* — taskID FULL UUID (Fix 6.3), result truncated to 400 chars.
- CORRECT: full UUID lands; Fix 6.3 verified. act_cli verbs match what the template instructs.
- Weak: 400-char result truncation can drop stack-trace tails that distinguish "retry will work" from "reassign needed" — Planner picks (a) vs (c) blind.

### 1.6 `autoroute_system_task_failed_burst` (orchestrator.go:634)
fromContent: *"%d task(s) failed in the last burst (e.g. %s). Use act_cli context --project <name> to see current task state, then retry/abandon/reassign as appropriate."* — `firstFailedSummary` = single `"task <uuid> (agent <agent-id>)"`.
- CORRECT: tells the Planner to call `act_cli context` before acting — at least signals the data-gap.
- STILL SUSPICIOUS: **8.4 unfixed.** N failures → 1 example. Reassignment decisions ("same role vs different role") made from one data point with no error string at all (single mode at least carries `result`, burst mode doesn't). Strictly worse signal than single mode.

### 1.7 `autoroute_system_validation_stuck` (orchestrator.go:2222)
fromContent: *"validation stuck on task %q (id=%s) after %d attempts; the swarm agent has retried the gap analysis without success"*.
- CORRECT: full UUID, attempt count surfaced (unlike fail_verdict above — *this* fromContent is honest about the cap).
- SUSPICIOUS: no gap text. Planner must `act_cli context` to make a sound reassignment. With empty-`@success_criteria` tasks (kanban issue) this fires indefinitely and the root cause is invisible in the body.

### 1.8 `autoroute_system_synthesis_stuck` (orchestrator.go:2344)
fromContent: *"synthesis stuck on task %q (id=%s) after %d attempts; QA cannot assemble the deliverable"*.
- CORRECT: surfaces a previously-silent failure mode.
- SUSPICIOUS: **wrong variant.** Routes to variantSystemEscalation whose first two options are `task retry` / `task abandon` on a task that's already PASSED Assurance. Telling the Planner to "retry" a validated task is a state-confusion vector. Correct action set is only (c)/(d), but the template still shows all four.

### 1.9 `autoroute_system_dedup` (orchestrator.go:1608)
fromContent (via `dedupAutorouteText`): *"Your last CREATE_TASK batch (%d tasks) was a duplicate of one dispatched %s ago — the orchestrator skipped re-creating those tasks. If you meant to dispatch NEW work, change at least one task title or description so the batch hash differs. If the previous batch already covered what you intended, stop re-emitting and wait for the swarm to make progress."*
- CORRECT: closes the silent-rejection blind spot (Fix 8.3 verified). Body is actionable on its own — "change a title OR stop re-emitting" is binary.
- SUSPICIOUS: wrapped in variantSystemEscalation, so Planner sees the four-option menu with `act_cli task retry/abandon` — but there's no task ID to retry, the batch was rejected before tasks existed. (a) and (b) are nonsense in this context. Options (c) "CREATE_TASK reassign" *can* apply, (d) "inform human" *can* apply, but the menu mismatch invites confusion.

---

## 2. Cross-cutting themes

### Theme A — The 4 variants are correctly forked but only 2 of 4 are tight.
variantPassVerdict ("silence is correct default") and variantSystemEscalation ("Silence is WRONG here") form a clean inverse pair — the central R1 tension is genuinely resolved. variantAnomaly retains the full (a)/(b)/(c) action tree which is right for Observer reports but wrong for the QA `SYNTHESIS_COMPLETE` traffic that falls through to it. variantFailVerdict mostly duplicates variantAnomaly with one extra clause about attempt-count inference; the split is arguably under-justified. A cleaner shape would be: anomaly stays as-is, fail_verdict folds into pass_verdict's silence-default framing (since the swarm retry is automatic and the Planner's first/second-fail response should *also* be silence). The current fail_verdict template inherits anomaly's "decide one of these" shape when its honest default is identical to pass_verdict's.

### Theme B — variantSystemEscalation is overloaded across 5 call sites with mismatched semantics.
The template assumes task-failure context (`act_cli task retry`, `task abandon`). Two of the 5 call sites violate that assumption: synthesis_stuck (task is validated, not failed) and dedup (no task ID exists yet). For these the (a)/(b) options are dead at best, actively harmful at worst. The template should either condition on call-site context (e.g., second template `variantSystemEscalationNoTask` without retry/abandon) or each call site needs to override the action menu. Currently all 5 share one body and the Planner receives wrong menu options 2/5 of the time.

### Theme C — fromContent payloads are inconsistent in how much they trust the Planner.
Compare: `validation_stuck` includes attempt count in the body; `fail_verdict` template tells the Planner to *infer* attempt count from history. `task_failed_single` carries 400-char error text; `task_failed_burst` carries zero error text for any of N failures. There's no consistent contract on "what fromContent must include to make the variant body actionable." The drift is per-call-site, which is exactly the shape Fix 2.1 was supposed to eliminate — the *variant template* is differentiated, but `fromContent` is still ad-hoc per call site.

### Theme D — Cascade cap (Fix 4.1) is the only thing protecting the prompt-side drift from cost blowup.
With variantSystemEscalation's "Silence is WRONG here" framing the Planner *will* act every time. Three of the call sites (synthesis_stuck, dedup, validation_stuck) can re-fire on their own polling cadence. Without `recentAutoRoutes` sliding-window (5/10min), a stuck task could trigger a loop where Planner emits a CREATE_TASK reassignment → swarm re-fails → escalation re-fires → Planner emits another CREATE_TASK. The cap is doing real safety work, not just dedup.

---

## 3. New drift surfaces (post-prior-audit)

| # | Drift | Severity | Rationale |
|---|---|---|---|
| D1 | variantPassVerdict (a) "obvious next step" is an open placeholder-CREATE_TASK invitation | **HIGH** | Same class as 2.1's original failure; small models will fabricate a "next step" task. No bounding criteria. |
| D2 | variantSystemEscalation menu mismatch for synthesis_stuck + dedup | **HIGH** | Two of 5 call sites show retry/abandon options on tasks where those verbs are semantically wrong. State-confusion vector. |
| D3 | variantFailVerdict requires attempt-count inference from history | **MEDIUM** | Orchestrator has the count; doesn't surface it in fromContent. Compaction-fragile. |
| D4 | Burst-mode fromContent carries no error string (worse than single-mode) | **MEDIUM** | 8.4 already flagged; specifically the error-text asymmetry vs single mode is new framing. |
| D5 | variantAnomaly catches QA SYNTHESIS_COMPLETE as "unparseable" fallthrough | **MEDIUM** | (a) CREATE_TASK option fires on completion events. Smaller models emit acknowledgement-tasks. |
| D6 | All variantSystemEscalation call sites share fromRole='system', losing the original event type | **LOW** | Planner can't condition on "this came from dedup vs. validation_stuck" without parsing the body. Affects routing decisions in the model. |

---

## 4. Verification of closed entries

- **1.1 (`act_cli task retry/abandon` real, POST /retry dropped):** `variantSystemEscalation` raw_text contains `(a) act_cli task retry <id>` and `(b) act_cli task abandon <id> --reason ...`. Verbatim. **No** `POST /api/tasks` substring anywhere in the 4 variant bodies. **CONFIRMED.**
- **2.1 (autoroute envelope differentiated):** 4 distinct variant raw_texts at orchestrator.go:1071, 1082, 1093, 1106. No shared envelope across all 5 system call sites — they share `variantSystemEscalation` only, which is by design. **CONFIRMED** (with the caveat from Theme B that the system-call-sites sharing one body is its own drift).
- **3.1 (silence vs react fork):** variantPassVerdict opens with *"No action is required by default"* and lists `(b) Stay silent (empty response). This is the correct default.` — silence is option (b), not (c), and is explicitly labeled correct. **However**, (a) "If the verdict unblocks an obvious next step ... emit CREATE_TASK directives" is **still present**. The resolution note claims the variant "defaults to silence" — true — but does NOT claim (a) was removed. So both readings of "fixed" are technically valid; the open (a) escape hatch is real and is D1 above.
- **4.1 (sliding-window cap replaces counter):** Out of sub3's read scope (orchestrator state machine, not prompt text). Not falsifiable from prompt entries alone; trust resolution note.
- **6.3 (taskID full UUID):** `autoroute_system_task_failed_single` raw_text: *"Task %s just failed"* with `taskID` substituted as full UUID, comment notes `truncate(taskID, 36)` removed. **CONFIRMED.**
- **8.3 (dedup autoroute surfaces):** `autoroute_system_dedup` exists at orchestrator.go:1608, wraps `dedupAutorouteText` in variantSystemEscalation. **CONFIRMED** (with Theme B caveat about menu mismatch).
- **8.4 (burst-mode single-failure detail):** Documented as STILL-ACTIVE in `autoroute_system_task_failed_burst.failure_modes_observed`. No fix applied. **CORRECTLY NOTED AS UNFIXED.**

---

## 5. Top 3 ranked drift surfaces in this subset

1. **D2 — variantSystemEscalation menu mismatch for synthesis_stuck + dedup.** Two of five call sites tell the Planner to `task retry` / `task abandon` on tasks that aren't in a failed state (or don't exist yet). State-confusion vector that survives the cap because each path has independent polling. Fix shape: split into `variantSystemEscalation` (task-failure) and `variantSystemNoTask` (synthesis/dedup) with only (c)/(d) options.
2. **D1 — variantPassVerdict (a) "obvious next step" placeholder vector.** Same failure class as original 2.1. Open invitation to fabricate a CREATE_TASK on a PASS verdict, particularly bad given the empty-`@success_criteria` fail-open upstream means PASS verdicts can be fraudulent. Fix shape: drop (a), make silence the only option; if a real dependent task exists, the Planner sees the next Observer report and routes through variantAnomaly.
3. **D4 + 8.4 — Burst-mode fromContent has *less* signal than single-mode.** N failures collapse to one summary line with no error text. Reassignment decisions made from a single agent+UUID, no failure mode. Fix shape: include either (a) all N agent+error pairs up to a token budget, or (b) `act_cli context` results pre-fetched and inlined, so the Planner doesn't have to round-trip a tool call before deciding.

(~1430 words)
