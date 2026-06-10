# Round 6 — Path A vs Path B head-to-head (comparator, re-verified against live `feat/remove-nomik`)

**Role:** Comparator. I read `round6-path-a-synthesis.md` (the 3-sub + synth structure) and `round6-path-b-self.md` (single pass), then **independently re-grepped every load-bearing citation against the real code** before encoding any finding as convergent or unique. The point of this pass is not to average the two paths — it is to catch the places where *both* paths agreed on something that the code no longer supports.

**Headline twist:** The single highest-confidence finding in the entire round — the one both paths independently arrived at and ranked HIGH (Path A Theme 3 / #3, Path B H-1: "autoroute/`[SYSTEM]` prompts leak into human chat") — is a **FALSE POSITIVE against current code.** Both paths cite the marker gate as `role != "planner"` at `orchestrator.go:583`. That gate does not exist. The live gate is `if !fromHuman` (`orchestrator.go:584`), and all six `runAgentTurn` callers pass `fromHuman` correctly (only `HandleHumanInput` at `:242` passes `true`; every autoroute/build/resume/observer/assurance/QA caller passes `false`). The `role != "planner"` form existed in the original Phase-2 port (`ca13adf`) and was **removed**. Both paths inherited the stale citation from the JSON map and neither re-verified it. **Convergence is not correctness** — this round proves it.

That single correction reshuffles the entire fix-plan: the #2/#3-ranked item on both sides evaporates, and the empty-criteria fail-open is the clear, uncontested #1.

---

## Re-verification ledger (what I personally confirmed at the boundary)

| Claim | Both paths' citation | Live code | Verdict |
|---|---|---|---|
| Empty `@success_criteria` → 100% PASS, no guard | `orchestrator.go:2988` `Passed: score>=passThreshold` | **`orchestrator.go:3049`** — exact same line, no `len(CriteriaResults)` guard. Line drifted, defect real. | **LIVE — confirmed** |
| Autoroute/`[SYSTEM]` leaks to chat | `orchestrator.go:583` marker gated on `role != "planner"` | `:584` gates on `!fromHuman`; `role != "planner"` removed since `ca13adf`; all injected planner turns ARE marked + hidden (`list.go:312`) | **STALE — already fixed; false positive** |
| ACP backend asymmetry (3.5 still open for non-Planner) | `common.go:147` `if role != "planner" { return actCLICommands(role) }` | `common.go:147-148` — exactly that; non-Planner ACP roles get the in-process "do NOT shell out" fragment | **LIVE — confirmed** |
| Brownfield researcher output unfenced into system prompt | `agents_md.go:31-32` verbatim concat | `agents_md.go:32` = `"## Codebase analysis\n\n" + notes + "\n\n"`, no fence; injected via `prompt.go:53` (`# Project-Specific Context`, anchor-last) | **LIVE — confirmed** |
| `variantFailVerdict` count-blindness | `orchestrator.go:1282-1284` | `:1344` "First or second failure… Third+…"; orchestrator tracks `attemptCount` (`:50`, `:2547`) but never surfaces it to the prompt | **LIVE — confirmed (Path-B-unique)** |
| QA `SYNTHESIS_COMPLETE` → `variantAnomaly` "React by taking action" | `orchestrator.go:1151`/`1313` | QA roles funnel through default at `:1187`→`:1372` (`variantAnomaly`, "React by taking action" at `:1374`) | **LIVE — confirmed (Path-B-unique)** |
| Both REMOVED markers gone; Fix 22 wired; 28 [FIXED] hold | various | spot-checked `actCLICommandsACP` planner fragment, no `codebase`/`nomik`; matches | **confirmed** |

---

## 1. CONVERGENT findings (both paths agreed — highest confidence *after* re-verification)

### CV-1 — [CRITICAL/HIGH] Empty `@success_criteria` auto-passes at 100% (fail-open verdict gate)
- **A:** Theme 2 / ranked #2 (HIGH). **B:** C-1 (CRITICAL). The severity split is the only disagreement; the mechanism is identical and both are right.
- **Live:** `orchestrator.go:3049` `Passed: parsed.OverallScore >= passThreshold` — no `len(CriteriaResults) > 0` guard. A criteria-less task lets Assurance emit `{"score":100,"criteriaResults":[]}` → `Passed=true` → routes to `variantPassVerdict` (`:1329`) → Planner correctly told to stay silent on a pass that validated nothing → work flows into the QA deliverable with a green check.
- **Why this is now #1:** it is the only convergent finding that survived re-verification at full strength, and it is the only one where the prompt engineering is *working as designed* (which is exactly why no prompt edit can close it). Both paths independently noted the prompt actively *misinforms* the Planner this can't happen (A: `section_examples:44` "Assurance will reject"; base prompt `planner.go:98` "verdict defaults to a meaningless 100%").
- **Severity adjudication:** **CRITICAL.** B's reasoning wins — silent integration of unvalidated work into the deliverable corrupts the entire trust model; A under-rated it at HIGH only because it weighed it against the (illusory) leak finding. Live kanban evidence: `assurance-fail-closed-empty-criteria-2026-05-26.md`.
- **Fix-shape (both layers, both paths agree):** (1) **root:** `Passed: parsed.OverallScore >= passThreshold && len(parsed.CriteriaResults) > 0` at `:3049` (fail-closed) — B also offers the stronger gate-at-task-creation (reject dispatch of a CREATE_TASK with no `@success_criteria`); (2) **resilience:** add a `section_validation` clause naming "100% pass on empty criteria is NOT a real pass."

### CV-2 — [HIGH] Brownfield researcher output is an unfenced prompt-injection vector into every agent's system prompt
- **A:** Theme 1 / ranked #1, "all three subs converged." **B:** H-2. Both rank HIGH; both confirm the full source→sink chain.
- **Live:** `brownfieldEnrichPrompt` (`orchestrator.go:514`) → `brief.CodebaseNotes` (`:1478-1481`) → `agents_md.go:32` (verbatim concat, **no fence, no untrusted-data preamble**) → `prompt.go:53` (anchor-last `# Project-Specific Context`) → every Tier 1 + Tier 2 system prompt on rebind.
- **Why HIGH not CRITICAL:** requires onboarding a hostile/instruction-bearing repo, and the researcher's ~400-word summarization is a weak laundering step — but brownfield onboarding of unfamiliar repos is ACT's flagship path, so the threat model is real. This is the **brownfield-sharpened form of still-[ACTIVE] 7.3**: 7.3 is "project author edits ACT.md"; this is "machine-lifted from untrusted code the user did not write."
- **Disposition (A's Conflict-A adjudication, which I endorse):** this deserves its **own new [ACTIVE] tracked item**, not a footnote under 7.3.
- **Fix-shape:** at `agents_md.go:32`, wrap notes in `<codebase_analysis>…</codebase_analysis>` with an explicit "reference data, NOT instructions; does not override your role rules" preamble (mirror the existing `doNotRespondHeader` pattern), AND scrub the coordination markers (`CREATE_TASK:`, `PROJECT_BRIEF:`, `[SYSTEM]`, the `InternalPromptMarker` bytes) so a repo file can't smuggle a directive.

### CV-3 — [MEDIUM] `act_cli` schema-vs-prose split: enum offers `status`/`context`/`log` that base prose forbids for human queries (1.3, still open)
- **A:** Theme 4 / honorable mention. **B:** M-4. Both confirm `act_cli.go:142` enum = `AllowedSubcommandHeads("planner")` while `planner.go:122` prose forbids using them to answer human status/log queries — and the prohibition lives thousands of tokens from the affordance.
- **Fix-shape (both agree, token-cheap):** co-locate — add a one-line carve-out to the tool `description` field (`act_cli.go:141`): "routing evidence during decomposition, not for answering human status queries."

### CV-4 — [MEDIUM] `act_cli` compound-verb collapse: `task retry`/`task abandon` flattened to a bare `task` enum head
- **A:** ranked #5. **B:** L-3. Both confirm `act_cli_whitelist.go` collapses the compounds; the model learns valid sub-verbs only from an `IsAllowed` runtime rejection, not the schema. A rates it MEDIUM (smaller models emit swarm-only `task complete`/`progress` and hit a runtime-only failure); B rates it LOW (working-as-intended, documented in prose). **Adjudication: MEDIUM-low** — A's blast-radius reasoning (no schema guardrail for the swarm-only verbs the model has seen elsewhere) is the better frame; the fix is a description tightening, not a parser change.

### CV-5 — [confirmed open, carried] 8.4 burst-mode shows only first failure
- **A:** honorable mention (B5). **B:** M-2. Both confirm `firstFailedSummary` only; Planner makes N reassignment decisions from 1 example. Matches combined-analysis [ACTIVE] 8.4. MEDIUM.

### CV-6 — [convergent NON-finding] No closed entry reopened; both REMOVED markers genuinely gone; Fix 22 correctly wired
- Both paths sampled the 28 [FIXED] entries (B did an explicit 10-entry spot-check table) and both found them holding. I re-confirmed the `actCLICommandsACP` planner fragment carries no `codebase`/`nomik`. **Highest-confidence non-finding of the round.**

---

## 2. Path-A-UNIQUE findings (what the 3-sub + synth structure caught that B missed)

### A-1 — [MEDIUM] `RebindSystemPrompt` skipped when AGENTS.md write fails (silent stale-prompt edge)
- **Only Sub2 (F-2) caught this; B did not surface it.** `orchestrator.go:1499-1517` nests `InvalidateContextCache()` + the rebind loop inside the `else` of `writeAgentsMd`, while `intakeMode=false` fires unconditionally at `:1520`. On any write failure (read-only dir, disk full), all four Tier 1 agents run the entire BUILD phase on their intake-era prompt with no project context and only a runner-log WARN.
- **Why A caught it:** the lifecycle-slice sub (Sub2) read the write-then-rebind sequence as a unit; B's flat single pass over 36 entries didn't dwell on the `else`-nesting. This is the clearest case of the fan-out structure earning its cost — a genuine silent-state edge only a focused slice surfaced.
- **Fix-shape:** move `InvalidateContextCache()` + rebind loop OUT of the `else`; rebind unconditionally from the server-replayed brief (the AGENTS.md file is a derived convenience, not a rebind precondition).
- *Comparator note: I did not exhaustively re-verify the `:1499-1517` nesting line-by-line this round; A's synth claims a re-grep confirmed it. Flagging as **A-unique, re-verify before fixing.***

### A-2 — [structural insight, not a standalone bug] The "split into a new tracked item" disposition for brownfield (Conflict A)
- A's synthesizer did adjudication work B didn't: it resolved the Sub1-vs-Sub2 framing conflict (amplifier-under-7.3 vs new-item) into "new [ACTIVE] item, fix at both the `renderAgentsMd` boundary AND the fragment boundary." This is process value — the synth layer producing a cleaner disposition than either input — rather than a new defect. Worth keeping.

### A-3 — [cross-sub confidence signal] Triple-independent rediscovery as a ranking input
- A's structure let it report *which* findings were independently hit by multiple slices (brownfield = all 3 subs; empty-criteria = Sub2+Sub3 from opposite directions). That's a confidence signal B structurally cannot produce from one pass. **But this round it also produced the false positive (CV-leak below), so the signal is necessary-not-sufficient.**

---

## 3. Path-B-UNIQUE findings (what the single pass caught that A missed)

### B-1 — [HIGH] ACP backend asymmetry survives for non-Planner Tier 1 roles (3.5 only half-closed)
- **B's H-3; A did not surface it at all.** `actCLICommandsACP` (`common.go:147-148`): `if role != "planner" { return actCLICommands(role) }`. Fix 22 branched the CLI fragment by backend **only for the Planner**. An ACP-backed Assurance/Observer/QA gets the in-process "do NOT shell out" fragment, while `renderShimNote` (`app.go`) tells them the `act-tier1-<role>` shim is on PATH and to "Use it via Bash" — the exact 3.5 contradiction combined-analysis declared FIXED. **Re-verified live: confirmed.**
- **Why B caught it:** B read the *whole surface* including the cross-role implication of the Planner-scoped JSON. A's slices were Planner-scoped by construction and never stepped outside to ask "is the Fix-22 branch role-complete?" This is the single best argument for the single-pass path this round — a boundary finding the fan-out's scoping blinded it to.
- **Caveat both must note:** this is technically *outside* `planner-prompts.json`'s Planner scope. But B is right that the combined-analysis "3.5 FIXED" claim **overreaches** — it's closed for the Planner, open for the other three Tier 1 roles. The doc should scope the 3.5 resolution to "Planner."
- **Fix-shape:** drop the "do NOT shell out" sentence from the non-Planner ACP path (cheapest), or extend the `actCLICommandsACP` branch to all four Tier 1 roles.

### B-2 — [MEDIUM] `variantFailVerdict` count-blindness (M-1)
- **B-only.** `orchestrator.go:1344` instructs "First or second failure: stay silent… Third+: abandon+reassign," but `fromContent` carries no attempt counter — the orchestrator tracks `attemptCount` (`:50`, incremented `:2547`) for its own `maxValidationAttempts` cap but never injects it into the prompt. The Planner is asked a count-dependent decision from data it doesn't have (only compaction-vulnerable history). **Re-verified live: confirmed.** A's Sub3 owned the autoroute family and missed this specific count-blindness.
- **Fix-shape:** prepend "Attempt N of 3:" into `fromContent` for fail verdicts (the orchestrator already has N).

### B-3 — [MEDIUM] QA `SYNTHESIS_COMPLETE` routed through `variantAnomaly`'s "React by taking action" (M-3)
- **B-only.** QA reports (no parseable verdict) fall through to the default `variantAnomaly` (`:1187`→`:1372`/`:1374`), which leads with "React by taking action" and offers "Emit one or more CREATE_TASK:" — the wrong framing for an informational "deliverable ready" signal, and the most likely place the empty/placeholder-CREATE_TASK failure mode fires. **Re-verified live: confirmed.** This is the surviving home of the 3.1 "react vs silence" tension the doc calls FIXED — the split routed *verdicts* off the anomaly tree, not *QA reports*.
- **Fix-shape:** add a `variantQAReport` (or reuse `variantSystemNoTask`'s no-task framing) for QA completion signals.

### B-4 — [MEDIUM] `variantSystemNoTask` too generic for `synthesis_stuck` (M-5)
- **B-only.** Fix 18 correctly moved `synthesis_stuck` off the retry/abandon menu, but `variantSystemNoTask` (`:1302-1308`) only says "take a non-task action if warranted… or stay silent" — it never tells the Planner *what* recovery means for a deliverable stuck in QA assembly. Tells the Planner what NOT to do, not what TO do. A's Sub3 didn't flag the residual.
- **Fix-shape:** add a `synthesis_stuck`-specific recovery line, or split a thin `variantSynthesisStuck`.

### B-5 — [LOW] cluster A missed: typo "recieve" (`planner_section_evidence.go:22`, L-1), attachment no size cap (L-4), no mode-echo confirmed open (L-2 / 3.4). Minor but real; A folded none of these in.

### B-6 — [methodological] B explicitly flagged the three "cross-entry consistency" overreaches in combined-analysis (the 7.1 per-surface framing, the 3.5 Planner-only scope, the 3.1 QA-report gap). A's synth did the 7.1-scope correction too, but B's was sharper on 3.5 and 3.1. **Both, ironically, then got the 7.1 fix wrong** (see below).

---

## 4. The shared MISS (neither path caught — the comparator's job)

### MISS-1 — [the false positive both paths shipped] The autoroute/`[SYSTEM]` leak is ALREADY FIXED
- **A:** Theme 3 / ranked #3 (HIGH), with Conflict-C calling for a "structural" fix to the `role != "planner"` gate. **B:** H-1 (HIGH), "most user-visible defect in the system."
- **Reality:** the gate is `!fromHuman` (`orchestrator.go:584`), not `role != "planner"`. Every orchestrator-injected planner-role turn (autoroute `:1632`, build/resume, observer, assurance, QA) passes `fromHuman=false` and therefore **receives the `InternalPromptMarker` and is hidden by `list.go:312`.** Only genuine human input (`HandleHumanInput:242`, `fromHuman=true`) renders. The `role != "planner"` form existed only in the original port `ca13adf` and was removed.
- **Root cause of the miss:** the JSON map's `runtime_substitutions`/notes still describe the old gate, and **both paths trusted the map over the code on this specific line** despite HARD RULE 1. A's synth even claims it "re-grepped `:583-585`… confirmed exactly as the subs reported" — that re-grep either read the comment block (which describes the *intent*) rather than the `if` condition, or was never run. B claims "confirmed by opening the cited file at the cited line" — same failure.
- **Lesson:** convergence amplified a shared map-inherited error instead of catching it. The comparator pass is the only layer that caught it, precisely because its mandate is to re-grep the convergent claims rather than trust the agreement. **This is the strongest methodology finding of Round 6.**
- **Residual real concern (downgraded LOW):** the *resume-context* case concatenates a `[SYSTEM]` block onto real user input (A/B both note `:237`). If that combined message is delivered with `fromHuman=true`, the `[SYSTEM]` prefix would render. Worth a 5-minute check of the resume path's `fromHuman` value — but the blanket "all autoroute leaks" claim is dead.

---

## 5. Methodology notes (token cost, best/worst per path, this round)

| Dimension | Path A (3-sub + synth) | Path B (single pass) |
|---|---|---|
| **Token cost** | ~4× (3 sub passes + synth reconciliation + the parent re-grep). Highest cost of the two. | ~1× single pass. |
| **Best at** | Depth-per-slice (caught A-1 rebind-skip edge no one else saw); cross-sub confidence signal; *disposition/adjudication* via the synth layer (Conflict A/B/C). | Whole-surface boundary reasoning — caught **all three** of the cross-role/cross-variant findings A missed (ACP non-Planner asymmetry, fail-verdict count-blindness, QA→anomaly mis-route). The single context saw implications across entries the scoped slices couldn't. |
| **Worst at** | Scoping blindness — Planner-scoped slices never asked "is the Fix-22 branch role-complete?" → missed B-1/B-3 entirely. Synth over-trusted sub claims (propagated the stale `role != "planner"` gate AND asserted a re-grep that didn't catch it). | Less depth on nested control-flow (missed the `else`-nested rebind skip A-1); skimmed the lifecycle write path. |
| **Map-drift handling** | Reported "line numbers stale, behavior accurate across all 36" — **wrong on the one line that mattered** (the marker gate is a behavioral change, not a line drift). | Same miss; also claimed per-line verification it didn't fully do. |
| **Net this round** | Justified its cost on A-1 + the synth dispositions, but its central confidence signal (triple-rediscovery) produced a false positive. | Higher yield-per-token: 3 unique real findings + the 3.5 scope correction, at 1/4 the cost. |

**Verdict:** For *this* corpus (well-mapped, prior-round-hardened, mostly-fixed), **B was the better value** — the single context's whole-surface view caught the cross-cutting boundary findings that matter most, and the fan-out's extra cost bought one genuine edge (A-1) plus adjudication polish. **But neither path is self-correcting on map-inherited errors** — that is structurally the comparator's job, and it earned its keep this round by killing the convergent false positive. Recommendation for Round 7: keep B as the primary, run A's fan-out only on the 2-3 deepest control-flow files (orchestrator lifecycle), and **make the comparator independently re-grep every CONVERGENT claim** (not just spot-check) — convergence is where shared map-drift hides.

---

## 6. COMBINED fix-plan ranking for Round 6 (union of both paths, leverage-ordered)

Ranked by blast-radius × frequency × confidence, **after** re-verification. Tag: `convergent` / `A-only` / `B-only` / `comparator`.

| # | Sev | Finding | Tag | File:line | Fix-shape |
|---|---|---|---|---|---|
| **1** | **CRITICAL** | Empty `@success_criteria` → 100% PASS (fail-open) | **convergent** (CV-1) | `orchestrator.go:3049` | Add `&& len(parsed.CriteriaResults) > 0` to `Passed`; +`section_validation` clause; ideally also reject criteria-less CREATE_TASK at dispatch. |
| **2** | **HIGH** | Brownfield researcher output unfenced into every system prompt | **convergent** (CV-2) | `agents_md.go:32` → `prompt.go:53` | Fence in `<codebase_analysis>` + untrusted-data preamble + scrub `CREATE_TASK:`/`PROJECT_BRIEF:`/`[SYSTEM]`/marker bytes. New [ACTIVE] item distinct from 7.3. |
| **3** | **HIGH** | ACP "do NOT shell out" vs shim "Use it via Bash" — open for non-Planner Tier 1 | **B-only** (B-1) | `common.go:147-148` | Drop "do NOT shell out" from non-Planner ACP path, or branch all 4 roles. Re-scope combined-analysis "3.5 FIXED" → "Planner only." |
| **4** | **MEDIUM** | `RebindSystemPrompt` skipped when AGENTS.md write fails (silent stale prompt) | **A-only** (A-1) | `orchestrator.go:1499-1517` | Move `InvalidateContextCache()`+rebind out of the `else`; rebind unconditionally from server brief. *Re-verify nesting before fixing.* |
| **5** | **MEDIUM** | QA `SYNTHESIS_COMPLETE` → `variantAnomaly` "React by taking action" | **B-only** (B-3) | `:1187`→`:1372`/`:1374` | Add `variantQAReport` / reuse no-task framing for QA completion signals. |
| **6** | **MEDIUM** | `variantFailVerdict` count-blindness | **B-only** (B-2) | `:1344` (+`:2547` counter) | Inject "Attempt N of 3:" into fail-verdict `fromContent`. |
| **7** | **MEDIUM** | `variantSystemNoTask` too generic for `synthesis_stuck` | **B-only** (B-4) | `:1302-1308` | Add synthesis-stuck recovery line or split `variantSynthesisStuck`. |
| **8** | **MEDIUM** | `act_cli` enum offers `status`/`context`/`log` prose forbids for human queries | **convergent** (CV-3) | `act_cli.go:141-142` | One-line carve-out in the tool `description`. |
| **9** | **MEDIUM-low** | `act_cli` compound-verb collapse (`task retry`/`abandon` → bare `task`) | **convergent** (CV-4) | `act_cli_whitelist.go:66` | Per-`args` description naming valid first-args; no parser change. |
| **10** | **MEDIUM** | Burst-mode shows only first failure (8.4) | **convergent** (CV-5) | `:819` region | Include all N `task <id> (agent <id>)` lines in `fromContent`. |
| **11** | **LOW** | Typo "recieve", attachment no size cap, no mode-echo (3.4) | **B-only** (B-5) | `evidence.go:22`, `:224`, lifecycle prompts | Spelling; optional size bound; add BUILD-mode echo expectation. |
| **12** | **LOW** | JSON map regen: stale line numbers + the marker-gate *behavioral* drift | **comparator** (MISS-1) | `planner-prompts.json` | Regen the map; **mark the `[SYSTEM]`/autoroute-leak entries RESOLVED** (gate is `!fromHuman`); re-verify the resume-concat `fromHuman` path as the one LOW residual. |
| **—** | **DROP** | ~~Autoroute/`[SYSTEM]` prompts leak into chat (A #3 / B H-1)~~ | **comparator-killed** | `:584` | **Already fixed.** Do NOT ship the "structural marker rework" both paths recommended — it would re-break a working gate. |

### Recommended Round 6a quick-win slice
Three items, ordered for one tight session:
1. **#1 empty-criteria fail-closed guard** — one-line change at `orchestrator.go:3049` + one test (`len(CriteriaResults)==0 ⇒ Passed=false`). Highest leverage, smallest diff, closes a CRITICAL the prompts can't.
2. **#3 ACP non-Planner "do NOT shell out" drop** — delete one sentence in the `common.go:147` fallback (or branch all four roles) + extend `TestPlannerPromptBranchesOnProvider` to the other three roles. Closes the half-open 3.5.
3. **#8 + #9 `act_cli` description tightening** — co-locate both carve-outs in the tool `description` field; pure text, token-cheap, no parser/schema change.

Defer #2 (brownfield fence) to 6b — it's HIGH but needs a new tracked item, a fence design, and a marker-scrub test, which is a session of its own. **Before touching any leak code, mark MISS-1 resolved in the JSON so Round 7 doesn't re-discover the dead finding.**
