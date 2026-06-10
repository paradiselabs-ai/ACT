# Round 5 — Path A Synthesis (composition + lifecycle + autoroute)

Inputs: sub1 (8 static composition + identity), sub2 (12 lifecycle + section + tool), sub3 (9 autoroute family). Each sub read its subset cold. State: post-Round-4 (14 fixes closing 25 entries).

---

## 1. Cross-cutting themes across multiple subs

### Theme 1 — Schema/affordance breadth is uncoupled from prompt-narrative constraints (HIGH leverage; appears in sub1 + sub2 + sub3 — **systemic**)

All three subs independently surface the same shape: a structured surface enumerates an option, prose elsewhere forbids or warns against it, and the surfaces never meet on the same page.

- sub1: *"`act_cli_commands_fragment` is NOT enumerated despite being on AllowedFor('planner') ... ACP priming's 'Use it via Bash for all ACT-coordination subcommands' ... directly contradicted by the unmodified fragment it ships alongside."*
- sub2: *"`act_cli` schema lists `status/context/log` despite prose forbidding their use for human queries (1.3) ... Pattern: schema/section breadth is uncoupled from prompt-narrative constraints."*
- sub3: *"variantSystemEscalation is overloaded across 5 call sites with mismatched semantics. The template assumes task-failure context ... synthesis_stuck (task is validated, not failed) and dedup (no task ID exists yet)."*

Same root cause, three surface layers: in-process tool schema, ACP fragment text, autoroute variant template. Narrowing the schema/template is structurally higher-leverage than tightening prose.

### Theme 2 — Live-from-source drift prevention is partial — same class re-emerges below the test line (HIGH leverage; sub1 + sub2)

Round 3's `TestPromptSectionAdvertisementMatchesRegistry` and Fix 5.4's `renderShimNote` proved the pattern. Both subs note it stopped where the test stopped.

- sub1: *"the pattern is NOT extended to `act_cli_commands_fragment` — its enumeration in `common.go:79` is hand-written. Adding a whitelisted subcommand updates AllowedFor() and renderShimNote() (drift-locked) but not the in-process command list. That's exactly how `prompt-section` reached the allowlist and the priming while staying missing from the fragment."*
- sub2: *"`expand_prompt_section_tool_inprocess` ... Enum from `prompt.SectionNames()`. TestPromptSectionAdvertisementMatchesRegistry closes drift risk."* — verified — *"BUT 'examples' violates the base prompt's `@dependencies` rule."* Registry parity holds; section *content* drift was uncaught.

Drift-prevention covered the surface the test names and nothing wider.

### Theme 3 — ACP/in-process asymmetry is structurally one-sided (HIGH; sub1 + sub2)

- sub1: *"ACP-specific surfaces ... have all been retrofitted to match in-process truth. But IN-PROCESS surfaces never branch on backend — `actCLICommands('planner')` returns identical text regardless of consumer."*
- sub2: *"Fix 1.2 made `sectionRegistry` the single source — content drift impossible. But in-process gets a structured tool result; ACP gets a Bash tool result with shim framing. A section saying 'Use this output to make a decision' reads differently when wrapped in shell output."*

Content converges, envelope still diverges. `PlannerPrompt(provider)` has the arg for the branch but ignores it (sub1's D4).

### Theme 4 — `fromContent`/context payloads are inconsistent in trust contract (MEDIUM; sub2 + sub3)

- sub2: *"Resume + BUILD triggers restate the brief well ... but defer 'decompose into tasks' mechanics to 'you know the shape' or recall."*
- sub3: *"`validation_stuck` includes attempt count in the body; `fail_verdict` template tells the Planner to *infer* attempt count from history ... There's no consistent contract on 'what fromContent must include to make the variant body actionable.'"*

Same shape: the orchestrator holds state (attempt count, brief shape, dispatch hash) that the Planner is then asked to recover from inference. Per-call-site ad-hoc.

### Theme 5 — Silent-state edge cases survive the fixes (MEDIUM; sub2 + sub3)

The fail-open / fail-silent edges that Rounds 1-4 didn't touch.

- sub2: *"section_validation ... doesn't address the **fail-open** failure mode (empty `@success_criteria` → 100% auto-pass)."*
- sub3: *"Combined with the empty-criteria fail-open ... the Planner can be told 'silence is correct' on a fraudulent pass. Prompt is fine; the orchestrator gate upstream is the gap."*

Both subs land on the same upstream-gate gap from opposite directions (the section that tells Planner how to read verdicts; the variant prompt that frames a PASS).

---

## 2. Single-sub findings with system-wide implications

**From sub1 — `PlannerPrompt(provider)` ignores its `provider` arg.** Sub2 and sub3 couldn't see this because they don't read prompt.go. But it's the structural enabler for Theme 3: there's already a branching seam, and unblocking it removes the ACP/in-process content-vs-envelope tension wholesale (sub2's Theme D collapses; sub1's D2 contradiction can be branched away).

**From sub1 — `InvalidateContextCache` doesn't handle path-set deletions.** Sub2/sub3 can't see this. Implication: any Round-5 fix that removes a contextPath from config silently keeps stale content live across rebuilds. Hits every fix that touches context composition.

**From sub2 — `section_examples` shows `@dependencies` inside the description string in violation of the base prompt's shape rule.** Sub1 verified the base-prompt rule ("dependencies pinned ALWAYS array of strings"); sub3 never sees sections. The example actively teaches the wrong shape — and Fix 6.4 (eabcc9b) installed the forgiving parser specifically because this shape kept appearing. This finding identifies the *source* of the shape Fix 6.4 had to defend against.

**From sub2 — `human_input_passthrough` attachments are forwarded raw with no size limit.** Touches every cached cascade: a multi-MB drop becomes input-token cost every turn. Sub1/sub3 don't see the lifecycle path.

**From sub3 — `variantSystemEscalation` is overloaded across 5 call sites with 2 semantically wrong.** Sub1/sub2 don't see the call-site graph. The "synthesis_stuck routes Planner to `task retry` on an already-validated task" failure is a state-confusion vector that survives the cascade cap because each path polls independently.

**From sub3 — burst-mode `fromContent` carries *less* signal than single-mode.** Only visible by comparing two sibling autoroutes; sub1/sub2 see neither.

---

## 3. Conflicting interpretations

### Conflict A — `variantFailVerdict` split: justified vs over-split

- **sub3 says under-justified:** *"variantFailVerdict mostly duplicates variantAnomaly with one extra clause about attempt-count inference; the split is arguably under-justified. A cleaner shape would be: anomaly stays as-is, fail_verdict folds into pass_verdict's silence-default framing."*
- **sub2 implicitly endorses the current split** by validating section_validation's once/twice/three-times decision tree as *"captures the FAIL once/twice/three-times decision tree clearly"* — i.e. the fail-verdict variant correctly anchors the cadence the section teaches.

Real disagreement: sub3 wants fail_verdict folded into silence-default (toward pass_verdict); sub2 treats the bespoke fail handling as load-bearing for the decision tree downstream.

### Conflict B — Section content is "exemplary" vs "actively wrong"

- **sub2 on the section-tool surface:** *"`expand_prompt_section_tool_inprocess` ... Description carries tight WHEN/WHEN-NOT-TO-USE — exemplary tool surface. CORRECT."*
- **sub2 on a section it serves:** *"section_examples ... HIGH severity drift ... directly contradicts the base prompt's shape rule."*
- **sub1 has no view of section content** and treats `expand_prompt_section` as a closed item via Fix 1.2.

If sub1 is right that the registry-locked advertisement is "closed", and sub2 is right that one served section actively teaches a forbidden shape, then Round 4's closure of 1.2 was premature — drift moved one layer down from advertisement to content. Same fix, two verdicts.

### Conflict C — Whether `[SYSTEM]`-prefixed text in chat is a UX leak (7.1) or load-bearing scaffolding

- **sub2 flags it as still-active drift:** *"`[SYSTEM] Resuming project %q ...` literal lands in chat (7.1 ACTIVE) ... Hits more chats now because Round 2 made resume far more verbose."*
- **sub1 does not flag the marker as drift** — the equivalent surface in sub1's subset is `acp_priming_prompt`, where sub1 notes the do-not-respond English header is *inside* the bubble and concludes *"Mitigated, not fixed"* — suggesting the prefix-text-in-chat pattern is now treated as acceptable shape.

Real disagreement on the same pattern: sub2 wants it removed/styled; sub1 has implicitly accepted that some scaffolding-text-in-chat is unavoidable.

### Conflict D — Whether `variantAnomaly`'s (a)/(b)/(c) tree is the right shape

- **sub3:** *"variantAnomaly retains the full (a)/(b)/(c) action tree which is right for Observer reports but wrong for the QA `SYNTHESIS_COMPLETE` traffic that falls through to it."* Treats it as correctly forked-but-mis-routed.
- **sub2** has no view of variantAnomaly's call sites and implicitly trusts the lifecycle-side handoff to land cleanly.

Sub3 sees a real routing bug invisible to sub2.

---

## 4. The complete picture — one architecture, three viewpoints

Four rounds of fixes built a Planner-prompt system that DOES three things well now: (a) `sectionRegistry` + `renderShimNote` + `AllowedFor` form a live-from-source drift-prevention spine for the surfaces they cover (sub1 + sub2 both verify); (b) `renderBriefContext` + `renderAutoRoutePrompt` differentiated lifecycle and event surfaces enough that "stay silent vs react" is no longer a single broken envelope (sub2 + sub3 both verify); (c) the dispatch-hash dedup, the sliding-window cascade cap, and the ACP discard-sessions rebind closed the silent-orchestrator-state failure class for the loops the audit knew about (sub3 verifies; sub1 verifies adjacent).

Where the system STILL fails systematically: drift has moved *down a layer* from the surfaces the tests guard to the surfaces they don't. The same class of bug (schema offers an option prose forbids; one source enumerates what another doesn't; the LLM is asked to infer state the orchestrator holds) keeps re-appearing in places the test grammar doesn't cover — the `act_cli_commands_fragment` enumeration, the section content body, the per-call-site `fromContent` payload, the `variantSystemEscalation` overloading across 5 mismatched call sites. The fixes were correct at their level and surfaced the next level intact. Round 5's job is to extend the live-from-source pattern one layer outward (fragment enumeration, section content lint, per-call-site contract) and to split overloaded templates whose call sites have diverged semantically.

---

## 5. Verification of closed entries — composite view

Closed entries verified clean by the subs (29 closures across all rounds; 24 fall in the union of subset scopes):

- **Clean:** 1.1, 2.1, 2.2, 2.3, 2.4, 3.1, 3.2, 3.3, 4.1, 5.1, 5.2, 5.3, 5.4, 6.1, 6.3, 6.4, 6.5, 7.2, 8.1, 8.2, 8.3 — all three subs' verification notes match resolution claims.

- **Partial drift (one entry):** **1.2** — sub1 flags *"PARTIAL DRIFT. Base prompt names the act_cli fallback; allowlist includes `prompt-section`. BUT `act_cli_commands_fragment` never got the corresponding enumeration line. Drift surface D1 is the unclosed remainder."* Sub2 independently confirms via Theme C that `section_examples`'s content violation is a second remainder under the same closure. Same Fix-1.2 closure, two distinct uncovered remainders.

- **Fully regressed:** none.

- **Out-of-scope-for-this-audit:** 4.1's state-machine internals (sub3 explicitly defers), 6.2's clarification-routing internals (no sub in scope).

---

## 6. Top 5 ranked drift surfaces for Round 5

1. **`variantSystemEscalation` menu mismatch for `synthesis_stuck` + `dedup`** — `orchestrator.go:1093` (template) consumed by `:2344` (synthesis_stuck) and `:1608` (dedup). 2 of 5 call sites tell the Planner to `task retry`/`task abandon` on tasks that aren't failed or don't exist. State-confusion vector that survives the cap because each path polls independently (sub3 D2). **Fix shape:** split into `variantSystemEscalation` (task-failure) and `variantSystemNoTask` (synthesis/dedup) with (c)/(d) only.

2. **`section_examples` teaches `@dependencies` inside the description string** — `planner_section_examples.go:33`. Directly contradicts the rule Fix 6.4 (eabcc9b) had to install a forgiving parser to defend against. This is the *source* of the shape that keeps appearing (sub2). **Fix shape:** rewrite the example to put `dependencies` as a top-level JSON property; add a registry-locked test asserting no section body emits forbidden patterns from the base prompt.

3. **`variantPassVerdict` (a) "obvious next step" placeholder vector** — `orchestrator.go:1071`. Same failure class as original 2.1; open invitation to fabricate a CREATE_TASK on a PASS, particularly bad given the empty-`@success_criteria` fail-open upstream produces fraudulent passes (sub3 D1 + sub2 fail-open). **Fix shape:** drop option (a); if a real dependent task exists, the next Observer report routes it through `variantAnomaly`.

4. **`act_cli_commands_fragment` missing `prompt-section`; "Use it via Bash" contradicts "do NOT shell out"** — `common.go:79` enumeration plus `app.go:261` ACP wrapper. Two manifestations of the same root cause: the in-process fragment isn't backend-aware and isn't registry-locked (sub1 D1 + D2). **Fix shape:** generate the fragment enumeration live from `AllowedFor('planner')` and branch via `PlannerPrompt(provider)`'s unused arg.

5. **Section bodies use bare `act X Y` CLI shorthand neither backend executes** — `section_evidence_routing` + `section_nomik` (5 occurrences confirmed). In-process is JSON tool call; ACP is `act-tier1-planner X Y` via Bash. Planner copies verbatim → tool-validation error, burns turns (sub2). **Fix shape:** rewrite section CLI examples in the in-process JSON-call shape with an ACP-prefix note; lock with a regex test against the section registry.

---

## 7. Gaps in this audit — what Round 6 needs

- **Tier 1 ↔ Tier 2 boundary.** All three subs stayed inside Planner prompts. None audited what the swarm-side `act task complete` / `task submit-for-validation` calls return as feedback to the Planner's worldview, or what the Runner injects into AGENTS.md on swarm spawn. The "fraudulent PASS verdict" thread sub2 + sub3 both pointed at lives on that boundary.

- **State-machine internals.** Sub3 explicitly deferred Fix 4.1's sliding-window cascade cap as out of prompt-text scope. The cap is the only thing protecting the prompt-side drift from cost blowup (sub3 Theme D) — its internals deserve their own subset.

- **Test grammar itself.** Round 5's recurring finding is "the test caught the surface it names; drift moved down one layer." A Round 6 should audit the test suite's *coverage shape*, not just the prompts — what surfaces should be registry-locked that aren't?

- **Section content lint.** Sub2 identifies one section actively teaching a forbidden shape. There's no systemic mechanism to assert sections don't contradict the base prompt. Should be a single regex/AST test, not a per-section review.

- **The `provider` arg.** Sub1 flagged it as a foot-gun; nothing in Rounds 1-4 wired it. Round 6 should either delete it or use it.

- **Multi-LLM behavioral A/B.** Combined-analysis still notes the audit gap (HANDOFF.md). No sub could compare same-prompt-different-model drift. The "smaller-model placeholder CREATE_TASK" pathology runs through 4 of the top 5 above; we infer it from commit history rather than measure it.

(~1980 words)
