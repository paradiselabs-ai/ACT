# Round 6 — Path A, Sub 3: Autoroute family (cold re-audit)

Scope: the 10 autoroute-family entries —
`autoroute_variant_anomaly`, `autoroute_variant_pass_verdict`, `autoroute_variant_fail_verdict`,
`autoroute_variant_system_escalation`, `autoroute_variant_system_no_task`,
`autoroute_system_task_failed_single`, `autoroute_system_task_failed_burst`,
`autoroute_system_validation_stuck`, `autoroute_system_synthesis_stuck`, `autoroute_system_dedup`.

Verified against `act-agent/internal/app/orchestrator.go` on `feat/remove-nomik`. All five variant
bodies, the variant-selection seam, and all five system-event call sites were read in the real code.

Closed entries cross-checked against `combined-analysis.md` so the six [ACTIVE] items
(3.4, 4.2, 4.3, 7.1, 7.3, 8.4) are not re-discovered as new.

---

## A. Verification matrix — variant bodies and call-site routing

### Five variant bodies (`renderAutoRoutePrompt`, orchestrator.go:1266-1323)

| Variant | code lines | matches JSON raw_text |
|---|---|---|
| `variantPassVerdict` | 1268-1278 | YES — verbatim, incl. Fix 19 "Do NOT emit a CREATE_TASK in reaction to a PASS" |
| `variantFailVerdict` | 1280-1286 | YES — verbatim, Fix 21 trimmed-to-prose form |
| `variantSystemEscalation` | 1288-1299 | YES — verbatim, (a) retry / (b) abandon / (c) CREATE_TASK / (d) human |
| `variantSystemNoTask` | 1301-1309 | YES — verbatim, "do NOT call act_cli task retry/abandon" |
| `variantAnomaly` (default) | 1311-1321 | YES — verbatim, classic (a)/(b)/(c) tree |

All five variant strings match the JSON `raw_text` byte-for-byte. No content drift.

### Variant-selection seam (orchestrator.go:1151-1161)

Confirmed: `variant := variantAnomaly`; only `role == "assurance"` re-classifies via
`parseValidationVerdict("", content)` → `v.Passed ? variantPassVerdict : variantFailVerdict`.
Observer + QA + unparseable-Assurance all stay `variantAnomaly`. Matches `autoroute_variant_anomaly`
trigger description exactly.

### System-event call sites — variant routing (the audit's explicit ask)

| Call site | code line | variant | correct per JSON contract |
|---|---|---|---|
| task_failed single | 806 | `variantSystemEscalation` | YES |
| task_failed burst | 822 | `variantSystemEscalation` | YES |
| validation_stuck | 2494 | `variantSystemEscalation` | YES |
| synthesis_stuck | 2619 | `variantSystemNoTask` | YES |
| dedup (`notifyDispatchDedup`) | 1812 | `variantSystemNoTask` | YES |

**Every system-event call site routes to the correct variant.** The Fix 18 re-routing
(synthesis_stuck + dedup → no-task) is verified in code, including the rationale comments at
1790-1794 (dedup) and 2616-2618 (synthesis). The Round 5 semantic mismatch
(no-task events wrongly getting the retry/abandon menu) is genuinely closed.

`task retry` / `task abandon` are real allowlist entries (`act_cli_whitelist.go:28-29`,
`RoleSubcommands["planner"]`), so variantSystemEscalation's (a)/(b) and variantFailVerdict's
"task abandon" are NOT capability-lies. fromContent makes each escalation body actionable: full
task UUID is shown (no truncation — Fix 6.3, verified 802-805) so the retry/abandon `<id>` calls
won't 404.

---

## B. Findings (severity-tagged)

### B1 — [HIGH] Autoroute prompts render as visible user-role chat messages (7.1, widened)

- **Cite:** `runAgentTurn` orchestrator.go:583-585 prepends `InternalPromptMarker` **only when
  `role != "planner"`**. Every autoroute fires via `fireWhenPlannerIdle` → `runAgentTurn(...,
  "planner", prompt)` (orchestrator.go:1571), i.e. `role == "planner"`, so the marker is skipped
  and the chat-list filter does not hide it.
- **What the user sees:** the full variant instruction tree as if it were their own typed input —
  e.g. on a single task failure the human sees *"The orchestrator surfaced a system event that
  requires Planner action. Silence is WRONG here. Pick ONE concrete action: (a) act_cli task
  retry <id> ..."* plus the `[system]: Task <uuid> just failed ...` banner. Same for all five
  system events and every Observer/Assurance/QA autoroute.
- **Why this is the autoroute-family instance of 7.1:** combined-analysis 7.1 [ACTIVE] is scoped to
  `resume_context_prepended` + `build_mode_trigger` (the literal `[SYSTEM]` prefix). The autoroute
  variants do NOT start with `[SYSTEM]`, so a reader of the existing 7.1 entry would not realize the
  autoroute surface leaks too — yet it leaks *more text per fire* and fires far more often
  (every Observer cycle, every verdict, every failure). 7.1's stated fix ("give Planner its own
  InternalPromptMarker for orchestrator-authored prompts, OR render as styled system banners") would
  cover this only if the fix is applied at the `fireWhenPlannerIdle`/autoroute layer, not just the
  two named lifecycle prompts.
- **Severity rationale:** not a coordination bug (Planner still acts correctly), but it dumps
  ~100-200 tokens of internal control-flow boilerplate into the human transcript on a high-frequency
  surface. UX corrosion + makes the chat unreadable during active swarms.
- **Fix-shape:** route autoroute prompts through a Planner-internal marker variant (a marker that
  hides the *input* turn from chat but still shows the Planner's *reply*), OR emit the event as a
  styled system banner and feed the Planner a marker-wrapped copy. Do at the autoroute layer so all
  five system variants + agent autoroutes are covered in one change. This is the same fix 7.1 wants;
  this finding widens 7.1's scope to name the autoroute surface explicitly.

### B2 — [HIGH] variantPassVerdict fires on a fraudulent PASS from empty @success_criteria (fail-open)

- **Cite:** `parseValidationVerdict` orchestrator.go:2968-2996 computes
  `Passed: parsed.OverallScore >= 95` with **no guard on `len(parsed.CriteriaResults)`**. A verdict
  JSON with `"criteriaResults":[]` and `"score":100` returns `Passed=true`. The selection seam
  (1152-1158) then picks `variantPassVerdict`, and the Planner correctly defaults to silence — on a
  pass that validated *nothing*.
- **Why it lands in this family:** the autoroute variant is doing exactly what it should (silence on
  PASS), which is precisely why the fraud is invisible — there is no autoroute signal that the pass
  was hollow. The variant cannot self-detect this; the orchestrator must catch empty
  `criteriaResults` *before* routing to `variantPassVerdict`.
- **Status:** tracked as kanban `assurance-fail-closed-empty-criteria-2026-05-26` (high/backlog) and
  noted in the JSON entry. NOT in the combined-analysis [ACTIVE] list — confirming it as live,
  highest-leverage NEW drift this round for the autoroute family. The companion file
  `assurance-fail-closed-empty-criteria-2026-05-26.md` exists in `.devtool/features/` (untracked),
  so it is queued but unfixed.
- **Severity rationale:** silently integrates unvalidated work into the deliverable; the empty-criteria
  CREATE_TASK is itself blocked at emission by basePlannerPrompt, but a swarm-side or
  server-seam path that submits a no-criteria task reaches Assurance and auto-passes.
- **Fix-shape:** in the selection seam (or in `parseValidationVerdict`), treat
  `len(CriteriaResults)==0` as `Passed=false` / a distinct "fail-closed" route, so it escalates
  rather than silently passing. Fail-closed, not fail-open.

### B3 — [MEDIUM] variantFailVerdict tells the Planner to "stay silent" at what may be attempt 3

- **Cite:** variantFailVerdict (1280-1286) keys behavior on "First or second failure ... Third+
  failure". But this autoroute carries **no attempt count** — the Planner must infer it from
  conversation history (compaction-vulnerable). The orchestrator's own attempt cap lives in a
  *different* path: `checkPendingValidation` (orchestrator.go:2487 `attempts > maxValidationAttempts`)
  escalates to `variantSystemEscalation` (validation_stuck, 2494), not this template.
- **Drift:** the Assurance-posted FAIL autoroute can fire on every failure including the third, and
  on that third fire it still says "First or second failure: stay silent" with no way for the Planner
  to know it has reached the reassignment threshold. Two independent counters (Assurance-message
  autoroute vs. the poller cap) with no shared signal — a silent-state edge.
- **Severity rationale:** not a hard break (the poller's validation_stuck escalation is the real
  backstop at attempt >3), but the prompt instructs an inference the Planner cannot reliably make,
  and the "third = reassign" advice is unactionable without the count.
- **Fix-shape:** inject the attempt number into the FAIL autoroute fromContent
  (`parseValidationVerdict` / the routeToAssurance path has `incAttempt`-style state for the task),
  e.g. "attempt N of 3", so "third+ failure" becomes observable rather than inferred.

### B4 — [MEDIUM] variantSystemNoTask is generic for synthesis_stuck — actionable path left implicit

- **Cite:** variantSystemNoTask (1301-1309) says "take a non-task action if one is warranted, inform
  the human if a decision is needed, or stay silent." For the **dedup** call site this is precise
  (the dedupAutorouteText body, 1819-1826, already spells out "change a title and re-emit, or stop").
  For **synthesis_stuck** (2619) the fromContent is just "synthesis stuck on task ... QA cannot
  assemble the deliverable" — and the variant gives no hint what a "non-task action" is. The
  deliverable is stuck in QA assembly (passed Assurance, not failed); the real recovery
  (inform human / restructure the deliverable / re-decompose the missing slice) is never stated.
- **Severity rationale:** correct *routing* (no-task is right — there is no failed task to retry), but
  the actionable guidance is asymmetric: dedup carries its own playbook in fromContent, synthesis_stuck
  does not, and the generic envelope does not fill the gap. The Planner is most likely to do nothing.
- **Fix-shape:** enrich the synthesis_stuck fromContent (orchestrator.go:2619) with a concrete next
  step — e.g. "QA cannot assemble; consider informing the human or emitting a CREATE_TASK for the
  missing integration slice" — mirroring how dedup's body carries its own guidance. (Note: this
  intentionally re-permits a CREATE_TASK path, which variantSystemNoTask's envelope does not forbid —
  it only forbids retry/abandon — so this is consistent with the variant.)

### B5 — [MEDIUM] Burst autoroute exposes only the first of N failures (8.4, confirmed open)

- **Cite:** burst call site orchestrator.go:807-809 sets `firstFailedSummary` from **only the first**
  task_failed event; the burst autoroute body (819-822) interpolates that single example plus a count.
  The Planner makes same-role-vs-different-role reassignment decisions for N tasks from one data point,
  and must run `act_cli context` to recover the rest.
- **Status:** combined-analysis 8.4 [ACTIVE], unchanged. Verified still present.
- **Fix-shape:** accumulate short summaries for all N burst failures (id + agent, one line each) into
  fromContent, accepting the higher prompt cost — the reassignment decision needs the full set.

### B6 — [LOW] JSON source_line drift vs. current code (map-vs-ground-truth)

The JSON line numbers for several entries have drifted from `feat/remove-nomik` HEAD. Per hard-rule 1,
JSON drift is itself a finding. Construction-site line numbers as read today:

| Entry | JSON source_line | actual call/site line |
|---|---|---|
| autoroute_variant_anomaly | 1311 | 1311 (default case head) — OK |
| autoroute_variant_system_escalation | 1288 | 1288 — OK |
| autoroute_variant_system_no_task | 1301 | 1301 — OK |
| autoroute_system_task_failed_single | 802 | summary at 802, call at 806 — OK-ish |
| autoroute_system_task_failed_burst | 819 | 819 — OK |
| autoroute_system_validation_stuck | 2394 | **2494** (off by 100) |
| autoroute_system_synthesis_stuck | 2519 | **2619** (off by 100) |
| autoroute_system_dedup | 1819 | call at **1812**; `dedupAutorouteText` def at 1819 |

The variant-body and selection-seam lines are accurate; the two poller call sites (validation_stuck,
synthesis_stuck) are each ~100 lines stale, and the dedup entry cites the helper-function line rather
than the call site. Cosmetic for behavior, but a re-grep-on-edit discipline miss. Fix-shape:
re-run the `how_to_keep_fresh` greps and update source_line on these three entries.

---

## C. "React vs silence" tension — current state

The central tension (combined-analysis 3.1, FIXED in ac87a1f) is genuinely differentiated per variant
in current code:

- `variantPassVerdict` leads with "No action is required by default" and lists silence as (a). No
  "react by taking action". Fix 19 removed the CREATE_TASK escape hatch. **No residual tension.**
- `variantSystemEscalation` deliberately flips: "Silence is WRONG here." Correct for known-failed tasks.
- `variantSystemNoTask` permits silence explicitly — correct for no-task events; resolves the Round 5
  contradiction where dedup got "Silence is WRONG".
- `variantAnomaly` keeps the full (a)/(b)/(c) tree with "React by taking action" — appropriate for
  Observer/unparseable span.

**Residual react-vs-silence soft spots (not the 3.1 regression — new observations):**

- variantAnomaly is the catch-all for QA `SYNTHESIS_COMPLETE` (QA is not Assurance → no verdict parse
  → anomaly). Its (a) option invites a CREATE_TASK in response to SYNTHESIS_COMPLETE, which should
  almost always be (b)/(c). Smaller models (glm-4.5-air:free observed) occasionally fire a placeholder
  CREATE_TASK here. This is a **[LOW]** standing risk noted in the JSON's anomaly entry — the
  placeholder-CREATE_TASK invitation survives in variantAnomaly's (a), the only variant that still
  carries it. Mitigation already in place: the (a) text itself says "NEVER emit a placeholder, empty,
  or acknowledgement CREATE_TASK — passing the verdict along is not a task." So the invitation and its
  prohibition are co-located; acceptable, but the residual model-noise is real.

No remaining *placeholder-CREATE_TASK invitation* exists in pass/fail/system variants — only
variantAnomaly offers (a), and it is guarded inline.

---

## D. Summary of what is clean

- All five variant bodies match JSON verbatim; no content drift.
- All five system-event call sites route to the correct variant (escalation for failed tasks,
  no-task for synthesis_stuck/dedup). Fix 18 verified end-to-end.
- No capability-lies: retry/abandon are real allowlist entries; full UUIDs shown.
- Cascade cap (`recentAutoRoutes` sliding window, 1226-1241) wraps every autoroute including all five
  system variants — no system-event path bypasses the cap.
- react-vs-silence 3.1 regression is closed and differentiated per variant.

Highest-severity open items for this family: **B1** (autoroute prompts leak into human chat — widens
7.1 to the autoroute surface) and **B2** (variantPassVerdict silently blesses an empty-criteria
fraudulent PASS — fail-open, kanban-tracked, unfixed).
