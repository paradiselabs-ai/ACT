# Round 5 — Path A vs Path B comparison

Same input (29 prompt entries in planner-prompts.json + the resolution notes in combined-analysis.md). Path A = 3 subagents (split 8/12/9) + synthesizer. Path B = me reading all 29 entries myself in one pass. Both ran in parallel.

---

## Convergent findings (both paths agreed on)

The top-leverage findings ranked similarly by both paths:

1. **`section_examples` teaches `@dependencies` inside description string** — CRITICAL on both rankings. Path B made it #1; Path A's synthesizer ranked it #2 (after `variantSystemEscalation` overloading).
2. **`act_cli_commands_fragment` missing `prompt-section`** — both flagged as HIGH. Path A bundled it with the 3.5 contradiction; Path B kept them separate.
3. **`variantSystemEscalation` overloading for `synthesis_stuck` + `dedup`** — both flagged HIGH. Path A made it #1; Path B made it #3.
4. **`variantPassVerdict` (a) escape hatch still a placeholder-CREATE_TASK vector** — both HIGH. Path A made it #3; Path B made it #3 (combined with HIGH-3/HIGH-4).
5. **Section bodies use bare `act X Y` shorthand neither backend executes** — both flagged. Path A #5; Path B MEDIUM-1.

Convergence on top 5: 4-of-5 overlap. Strong agreement on what the leverage-rich surfaces are.

Other convergent findings:

- **Assurance fail-open on empty `@success_criteria`** — both surfaced as a NEW finding tied to `variantPassVerdict`. Path A synthesizer landed this in Theme 5 (silent-state edges); Path B promoted it to HIGH-4.
- **`variantFailVerdict` requires Planner to infer attempt count** — both noticed the prompt-vs-state mismatch. Path A folded it into Theme 4 (fromContent inconsistent trust contract); Path B made it MEDIUM-2.
- **7.1 `[SYSTEM]` literal visible in user chat** — both flagged as still-active. Path B made it MEDIUM-3; Path A had it as Conflict C (sub disagreement on severity).

---

## Findings only Path A surfaced

These are real findings Path B missed:

1. **`PlannerPrompt(provider)` ignores its `provider` arg** — sub1 caught this; sub2/sub3 couldn't see it. The synthesizer correctly identified this as the structural enabler for fixing Theme 3 (ACP/in-process asymmetry). Path B read prompt.go's `processContextPaths` but didn't notice the unused arg signature. **High-leverage finding I missed** — wiring this arg lets the entire branch-by-backend solution work cleanly.

2. **`InvalidateContextCache` doesn't handle contextPath deletions** — sub1's D3 edge case. If a path is removed from config, the cache still serves stale content for it (because invalidation only clears the `loaded` flag, not the map of file-keyed content). Path B verified `getContextFromPaths` had the SHA-256 hash skip but missed this deletion edge.

3. **"Drift has moved one layer down" — Path A's synthesizer's section-4 framing.** The fixes were correct at their level and exposed the next layer intact. The same class of bug (schema offers what prose forbids; one source enumerates what another doesn't) keeps re-appearing in places the test grammar doesn't cover. This is a sharper meta-pattern than my "extend test discipline" framing.

4. **Conflict A — `variantFailVerdict` split: under-justified vs load-bearing.** Sub3 wants fail_verdict folded into pass_verdict's silence-default; sub2 implicitly endorses the current split by validating section_validation's once/twice/three-times decision tree. Path B couldn't produce this — single-agent analysis can't disagree with itself across subset views.

5. **Conflict B — `expand_prompt_section` is "exemplary" vs the section it serves is "actively wrong."** Sub2 disagreed with itself across two surfaces: the tool wrapper is well-designed; one served section actively teaches a forbidden shape. The synthesizer named this as "Fix 1.2 closure was premature; drift moved one layer down from advertisement to content." Path B caught both observations but didn't frame them as the same closure's two distinct remainders.

6. **Conflict D — `variantAnomaly` (a)/(b)/(c) tree mis-routed for QA `SYNTHESIS_COMPLETE`.** Sub3 flagged this; sub2 couldn't see the call sites. Path B noted that variantAnomaly handles QA messages as fallthrough but didn't connect "(a) emit CREATE_TASK" → "invitation to fabricate on SYNTHESIS_COMPLETE."

7. **`human_input_passthrough` attachments have no size limit.** Sub2 caught this. Path B noted the attachment forwarding but didn't flag the unbounded-input issue.

---

## Findings only Path B surfaced

Real findings the Path A pipeline missed despite higher per-entry attention:

1. **Direct disagreement with the prior audit's 3.5 framing.** Path A's sub1 flagged "Use it via Bash" vs "do NOT shell out" as a HIGH contradiction (D2). Path B explicitly **re-interpreted** the two phrases as describing different operations (messages-via-reply-text vs CLI-invocation-via-bash) and argued they're NOT a contradiction once parsed precisely. The wording is still confusing and should be standardized, but the underlying claim of "contradictory instructions" is wrong. Path A propagated the prior-audit framing; Path B challenged it. This kind of "the prior frame was wrong" finding is structurally easier for the single-agent path because A's subs each saw only one side and synthesizer reconciled rather than questioned.

2. **`act_cli` schema enum vs prose enumeration are DIFFERENT sets, not just out-of-sync.** Path B noticed:
   - Schema enum: 9 heads including `prompt-section`, missing the compound disambiguation for `task`
   - Fragment enumeration: 9 commands missing `prompt-section`, has `codebase` subcommands (onboard, communities) split into separate lines
   Path A noted the fragment was missing `prompt-section` but didn't observe that the schema also disambiguates `task` differently from the prose. The two surfaces aren't just non-identical — they have different shapes.

3. **The `act_cli` schema's `task` head doesn't disambiguate `task retry` vs `task abandon`.** Path B flagged this as MEDIUM (model sees `task` allowed, has to learn from rejection that `task complete` isn't). Path A treated the schema as a closed surface via Fix 1.1 verification.

4. **More explicit fix-by-fix verification table at top of analysis.** Path B's "Verified-closed entries (spot checks)" section is a flat audit of 8 closed entries with code citations. Path A's verification is woven through the per-entry close-reads and the synthesizer's Section 5 — harder to spot-check at a glance.

---

## Where both paths converged in framing (high-confidence)

Both paths independently landed on:

- **"Drift-prevention tests catch composition-time drift but not semantic drift."** Path B: *"Round 5 should fix the 5 surfaces above AND consider extending the test discipline to catch variant-vs-trigger mismatches."* Path A: *"Round 5's job is to extend the live-from-source pattern one layer outward (fragment enumeration, section content lint, per-call-site contract) and to split overloaded templates whose call sites have diverged semantically."* Same insight, different framing.

- **No regressions across the 25 closed entries.** Both paths verified the fixes held.

- **Round 4's closures exposed the next layer intact.** The 4 rounds of fixes didn't introduce new failures *within their scope*; they made the next-layer-down failures visible. Both paths landed there.

---

## Methodological observations

| Dimension | Path A (3 subs + synth) | Path B (single agent) |
|---|---|---|
| **Total tokens consumed** | ~350K (3 subs ~100K each + synth ~90K) | ~50K |
| **Wall-clock time** | ~4 min for 3 parallel subs + ~2 min synth = ~6 min | ~3 min while subs ran |
| **Unique findings count** | 7 | 4 |
| **Convergent on top 5** | 4-of-5 overlap with B | 4-of-5 overlap with A |
| **Verifications produced** | Sub-by-sub for 21 closed entries | Spot-checked 8 closed entries |
| **Per-entry attention depth** | High (sub had 8/12/9 entries each) | Medium (one pass through 29) |
| **Cross-entry pattern detection** | Stronger for systemic patterns (5 themes named) | Weaker — flat priority list |
| **Disagreement surfacing** | 4 explicit conflicts | 1 prior-audit reinterpretation |
| **Best at** | Naming meta-patterns, surfacing sub-disagreements, drift-class observations | Re-interpreting prior framings, schema-vs-prose shape mismatches, fast |
| **Worst at** | Re-using prior audit framings without challenging them; higher cost | Missing single-file structural observations (unused arg, deletion edge); no "conflict" section possible |

### Notable shifts vs Rounds 1's same experiment

- **Convergence rate up.** Round 1's experiment had ~90% convergence on top findings. Round 5 is similarly high (~80% top-5 overlap). Suggests the audit JSON is now mature enough that both methods land on the same major surfaces.

- **Path A's "conflicts" section is even more valuable in Round 5.** With more layered prompt surfaces, more places where two subs see the same thing through different lenses. Round 1 had 1 useful conflict; Round 5 has 4.

- **Path B's "re-interpret prior framing" finding is structurally new this round.** The audit JSON includes prior `[FIXED in <sha>]` notes with resolution claims. Path B challenged one of those framings (3.5). Path A's subs followed the prior-audit framing more deferentially because each sub only saw its slice of the prior closures.

- **Path A's higher cost is more justified than in Round 1.** The 29-entry surface is large enough that 3-way split + synthesis catches meta-patterns a single agent demonstrably misses (the `provider`-arg seam, the deletion edge, the conflicts). Round 1 had 19 entries; the marginal value of splitting was lower.

---

## Together they're better than either alone

Both paths' findings combined:

- **Convergent top-5:** examples-teaches-forbidden-shape, fragment-missing-prompt-section, variantSystemEscalation-overloading, variantPassVerdict-(a)-escape-hatch, section-CLI-shorthand-mismatch.

- **Path A's unique additions:** the `provider`-arg seam (high-leverage fix-enabler), 4 conflicts (especially Conflict A on fail_verdict split), the "drift moved one layer down" frame.

- **Path B's unique additions:** the 3.5 re-interpretation (challenges a prior-audit framing), the schema-vs-fragment-enumeration shape difference, the `task`-head non-disambiguation.

A Round-5 fix plan built from BOTH paths would have:
1. Section_examples shape fix (convergent)
2. act_cli_commands_fragment + ACP wrapper backend-branch via `PlannerPrompt(provider)` arg (Path A's structural enabler + Path B's enumeration shape catch)
3. variantSystemEscalation split or fromContent rewrite (convergent)
4. variantPassVerdict (a) restriction (convergent)
5. Section content lint test (convergent meta-fix)
6. PR Conflict A's resolution (sub3 vs sub2 disagreement on whether to fold fail_verdict — needs explicit decision, not silent adoption of either side)
7. Re-frame 3.5 in docs (Path B's challenge — say explicitly "this was misread; the two phrases describe different operations" so a future audit doesn't re-flag it).

Each path on its own would miss 1-3 of those. Together they cover the surface.

---

## What this round measured that Round-1's experiment didn't

- **The audit JSON itself is now a meaningful artifact across rounds.** In Round 1 the JSON was the only source. In Round 5 the JSON's `failure_modes_observed` field is annotated with prior `[FIXED in <sha>]` notes. Both paths verified those notes against current code. The artifact compounds — the same audit run later costs less, because the prior round's findings are already in the JSON.

- **Path A scales better with surface area.** Round 1: 19 entries, Path A unique findings = 7. Round 5: 29 entries, Path A unique findings = 7. Per-entry, Path A's marginal return diminishes; for a larger surface, it stays competitive.

- **Path B scales better with prior-audit context.** Single-agent path can hold the prior-audit framings in working memory and challenge them. Subs each see only their slice of the prior closures — harder to challenge framings cold.

- **The cost gap stayed at ~7× (Round 1) to ~7× (Round 5).** No improvement on either side from method tweaks; the 3-way split is structurally expensive.

### When to prefer which in Round 6

- **Use Path A** when adding new prompt surfaces or new variants — the cross-cutting themes catch the new-surface-class drift.
- **Use Path B** for verification audits (confirming closed entries still hold) and for re-interpretation of prior framings.
- **Use both** for any round that targets the "drift has moved one layer down" pattern — Path A names the meta-pattern, Path B catches the specific surface drift.
