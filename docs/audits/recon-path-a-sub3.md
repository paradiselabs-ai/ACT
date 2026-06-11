---
title: Recon — Path A Sub 3 (Kanban + Audit Line)
status: current
verified_against: bc0673e (feat/cleanup-constitution, code ≡ feat/remove-nomik)
owner: generated
analyzed: 2026-06-11
scope: report verdicts about kanban tickets + planner-prompt audit line, PLUS sweep of every
       unadjudicated active ticket and combined-analysis [FIXED] entries
---

# Recon Path A — Sub 3: Kanban + Audit Line

Re-grepped the live tree for every verdict the report (`handoff-verification-2026-06-10.md`)
makes about kanban tickets and the planner-prompt audit line, then swept ~50 tickets the report
never adjudicated. **The report's verdicts in this slice are accurate — zero verdict-errors found.**
The work product here is (a) confirmations with fresh live cites, (b) coverage gaps the report left
open, and (c) one count-presentation ambiguity worth a doc note.

---

## A. Report verdicts RE-VERIFIED (all hold)

### A.1 P11 [STALE] block6-files-to-create-stale — CONFIRMED CORRECT (dual-implementation hazard, HIGH)
- Ticket `block6-acp-cli-backend-2026-04-21.md` body L27-31 says "Files to create:
  `internal/llm/backend.go`, `internal/llm/acp/backend.go`; New interface: `AgentBackend`."
- Live: `ls internal/llm/backend.go` → no such file; `ls internal/llm/acp/` → no such dir
  (internal/llm contains only agent/models/prompt/provider/tools). The ACP impl landed at
  `internal/acp/` (agent.go, claude_code.go, client.go, session.go, transport.go, types.go,
  antigravity_cli.go), exposed via `func NewACPAgent(` at `internal/acp/agent.go:95`. Landing
  commit `5021848` (2026-05-26).
- No `AgentBackend` interface anywhere in Go; the only `AgentBackend` symbol is
  `config.WriteAgentBackend` (writer.go:16) — a config writer, not the interface.
- **Highest-leverage hazard in this slice.** The ticket is the exact ACP dual-implementation
  failure mode CLAUDE.md's banner warns about. Status `in-progress` is correct; the BODY is the
  trap — an implementer following "Files to create" would build a second ACP layer at internal/llm/.

### A.2 P12 [STALE] qa-redesign-phase-a-half-implemented — CONFIRMED CORRECT (duplicate-fix hazard)
- Ticket `qa-redesign-phase-a-nomik-agentmd-ingest-2026-04-21.md` (status `todo`): "Give
  QA/Synthesizer Grep + AGENT.md ingestion… Per-turn prompt stops forbidding tool use."
- Live: the Grep half IS already built — `QASynthesizerTools` returns
  `NewActCLITool + NewViewTool + NewGrepTool` (tools.go:111-117), wired via `Tier1ToolsForRole`
  (tools.go:139). The QA prompt actively instructs tool use: `qa_synthesizer.go:63` —
  "Use `view` and `grep` to read validated outputs directly." (report cited L62-63; live is L62-63,
  the instruction on L63 — exact.)
- The AGENT.md-ingestion-at-startup half IS genuinely unbuilt (no AGENTS.md-ingest path in QA).
- Verdict correct: ticket should be re-scoped to AGENT.md ingestion only. Minor: report calls the
  forbidden-prompt-clause already-removed; live `qa_synthesizer.go` has no tool-forbidding clause
  to grep — consistent with "no longer forbids tools."

### A.3 Fix-23 verdicts (f23-* cluster) — CONFIRMED CORRECT
- `parseValidationVerdict` (orchestrator.go:3077) sets the empty-criteria guard at :3105
  (`emptyCriteria := len(parsed.CriteriaResults) == 0`). Force-fail + gaps injection present.
- Tests `TestParseValidationVerdict_EmptyCriteriaFailsClosed` (orchestrator_test.go:292) and
  `_EmptyCriteriaBrokenVerdict` (:313) exist.
- Ticket `assurance-fail-closed-empty-criteria-2026-05-26.md` status `in-progress` (NOT done); its
  Status-update section (L15-39) records exactly the shipped (verdict-parser) vs open (server gate,
  assurance.go clause, system message, e2e assertion, re-queue loop) split. Status-HONEST.
- Report's `f23-residual-junk-criteria` (a `criteriaResults:[{}]` len>0 still passes) verified by
  the guard being `len(...) > 0`, not content validation — correct.

### A.4 Round-6 findings #2-#10 OPEN — CONFIRMED CORRECT (no silent closures)
Re-grepped each; combined-analysis.md marks exactly four `### [ACTIVE]` sections (3.4, 4.2, 7.3, 8.4)
plus the handoff's findings #2-#10. Spot-verified the load-bearing ones:
- **r6-3** (actCLICommandsACP planner-only): `common.go:138-146` — `actCLICommandsACP` with comment
  "Only the planner case diverges enough to need its own framing"; in-process Observer/Assurance/QA
  still get "do NOT shell out" at common.go:89/97/103. STILL OPEN. ✓
- **r6-4** (RebindSystemPrompt skipped on AGENTS.md write failure): orchestrator.go:1589-1607 — the
  rebind loop sits inside the `else` branch of `if err := writeAgentsMd(...)`. On write failure it
  logs `"Failed to write AGENTS.md"` and NEVER rebinds; all four Tier 1 agents stay on stale
  intake-era prompts. The else-nesting the report flagged for "re-verification" is REAL. ✓
- **r6-5** (SYNTHESIS_COMPLETE via variantAnomaly): orchestrator.go:1401 `default: // variantAnomaly`
  is the fallthrough; no synthesis-specific variant. OPEN. ✓
- **r6-6** (variantFailVerdict failure-count blind): orchestrator.go:1370-1373 — prose says "First or
  second failure / Third+ failure" with NO injected count. OPEN. ✓
- None of the 5 post-handoff commits (7021488,1919b06,578d280,f2c8d78,bc0673e) touch these paths —
  confirms the report's "OPEN statuses not silently closed."

### A.5 r6-chatleak-gate-fromhuman — CONFIRMED CORRECT (citation-drift in planner-prompts.json)
- planner-prompts.json literally states: *"The Planner role bypasses InternalPromptMarker
  (runAgentTurn only prepends it for non-Planner roles)"* — i.e. a **role-based** gate.
- Live: `orchestrator.go:584` is `if !fromHuman { content = InternalPromptMarker + content }`
  (gate defined at runAgentTurn signature :565). The gate is **fromHuman-based, not role-based.**
- This matters: an in-process Planner turn that is autoroute-triggered (fromHuman=false) DOES get the
  marker and IS hidden — the JSON's "Planner bypasses" framing would mislead anyone reasoning about
  what leaks. Report verdict correct; planner-prompts.json carries the stale role-based description.

### A.6 r6-28-fixed-entries-hold — CONFIRMED CORRECT (with a count caveat — see C.1)
- All 28 `FIXED in <sha>` tags trace to real, unreverted commits. Verified all 18 unique SHAs exist
  on-branch (`git log --oneline -1 <sha>` for each).
- Spot-checked 5 fixes actually HOLD in live code (not just commit-exists):
  - 1.2b/Fix15: `grep @dependencies planner_section_examples.go` → none (fix holds).
  - 4.1/Fix6: `consecutiveAutoTurns` gone as a counter; live mechanism is `recentAutoRoutes`
    sliding window (orchestrator.go:90-98, cap at :1320). (The two `consecutiveAutoTurns` grep hits
    are comments documenting the replacement, not live code.)
  - O.1/Fix16: `ValidationScore: score` populated in routeToQA (orchestrator.go:2775).
  - 6.4/Fix10: `dependencyList` forgiving UnmarshalJSON (orchestrator_types.go:202-206).
  - 5.1/770a290: `RebindSystemPrompt` is now a real session-discard impl (acp/agent.go:489), not a
    no-op.

### A.7 Other kanban verdicts spot-confirmed
- `spil-stage1-no-go-code-yet`: no Go SPIL parser/AST/evaluator/proof-criteria-gate; only
  `server/src/services/SPILParser.ts` (regex MVP). CONFIRMED.
- `tui-heartbeat-not-built` / `tui-truncation-still-open`: no heartbeat indicator, truncation still
  present with no expand affordance in internal/tui/components/chat/. CONFIRMED.
- 4 done dogfood cards: all four in `done/` with `status: done` + `completedAt` set
  (2026-06-06T22:17:31Z); their code-fixes (role label, InternalPromptMarker on Planner, observer
  escalation cap, QA synthesizedAt exclusion) verified live. CONFIRMED + status-HONEST.

---

## B. COVERAGE GAPS — tickets/entries the report NEVER adjudicated

The report adjudicated ~6 active tickets + the 4 done cards. There are **~50 active tickets**. I swept
them for status-honesty against live code. **No status-dishonesty found** — all sampled statuses are
honest — but the report's kanban coverage is thin. The notable swept-clean tickets:

- **B.1 qa-redesign-phase-c-need-clarification-routing (todo, medium) — PARTIAL-OVERLAP, flag.**
  NEED_CLARIFICATION routing to a named addressee IS implemented (orchestrator.go:1218 + regex
  :2824 `clarificationRegex`, Fix 11 tests at orchestrator_test.go:894-903), and combined-analysis
  6.2 marks it FIXED in b3cc2b7. The TICKET, however, is scoped to QA-initiated clarification
  routing (a different surface than the swarm-agent NEED_CLARIFICATION that Fix 11 covers). The
  generic mechanism exists; the QA-specific path may not. Not dishonest, but an implementer should be
  told "the addressee-routing primitive already exists — reuse it, don't rebuild it." (mild
  duplicate-fix hazard the report didn't surface.)

- **B.2 cli-fetch-to-http-migration (todo, high) — status HONEST.** act-cli.ts still uses raw
  `fetch(` (act-cli.ts:47-49+). Not migrated. Note this contradicts MEMORY.md's "Native HTTP client:
  act-agent/internal/act/client.go" framing for the *CLI* layer — the Go client is native HTTP, but
  the TS CLI is still fetch-based. (Report's config-env slice touched client.go but not this ticket.)

- **B.3 ralph-wiggum-loop-code-enforcement (todo, medium) — status HONEST.** Ralph Wiggum Loop is
  prompt-only (common.go:178/229 + per-role prompt files); no Go enforcement. Matches
  flows-explainer "Ralph prompt-only" finding. Ticket correctly `todo`.

- **B.4 compaction cluster (8 tickets, incl. 2 critical) — status HONEST.** No user-facing `/compact`
  command, no compaction-trigger config, no summarizer-fallback in Go. The `compact`/`summariz`
  substring hits are message-history internals, not the compaction feature. All correctly
  pre-`done`.

- **B.5 swarm-recovery cluster (7 backlog tickets) — status HONEST.** No abort endpoint, task-states
  enum, zombie-detection, partial-result, or recovery code in server/orchestrator (the lone "zombie"
  hit is an unrelated tsx-PID comment, index.ts:1600). All correctly `backlog`.

- **B.6 qa-deliverable-text-persistence (todo, medium) — status HONEST.** No deliverable-persistence
  code; matches flows-explainer "QA partial-deliverable persistence" gap.

- **B.7 spil-stage1-proof-criteria-gate / spil-stage1-parser-ast-evaluator (todo) — status HONEST.**
  No Go SPIL stage-1 code (see A.7).

**Combined-analysis [FIXED] entries beyond the report's spot-check:** sampled 14 of the 28 (commit
existence for all 18 unique SHAs + live-code holds for 1.2b, 4.1, O.1, 6.4, 5.1, plus 2.2/2.3 trimmed
duplication in 9707f9a, 3f0e8dd brief-inline, ac241e0 PlannerPrompt-branch, c853932 prompt-section,
de479f4 shim-from-AllowedFor). All hold. The 4 `[ACTIVE]` entries (3.4, 4.2, 7.3, 8.4) are genuinely
still open in live code.

---

## C. DOC-FIX ACTIONS (leverage-ranked — fix the SCAFFOLDING, never the code)

1. **[CRITICAL — dual-implementation hazard] block6-acp-cli-backend ticket BODY.** Rewrite "Files to
   create: internal/llm/backend.go, internal/llm/acp/backend.go; New interface: AgentBackend" to
   point at the SHIPPED reality: `internal/acp/` + `acp.NewACPAgent` (agent.go:95), no AgentBackend
   interface. Without this an agent rebuilds the ACP layer. Keep status `in-progress` (remaining work
   = ACP-priming parity for non-Planner roles, finding r6-3).

2. **[HIGH — citation-drift / leak-reasoning hazard] planner-prompts.json chat-leak description.**
   Replace "The Planner role bypasses InternalPromptMarker (runAgentTurn only prepends it for
   non-Planner roles)" with the live gate: `if !fromHuman` at orchestrator.go:584 — the marker is
   prepended to ALL non-human input *including* autoroute-triggered Planner turns. The role-based
   framing is wrong and will mislead any leak-surface reasoning.

3. **[MEDIUM — duplicate-fix hazard] qa-redesign-phase-a ticket re-scope.** Strike the "Grep + stop
   forbidding tool use" half (already shipped: QASynthesizerTools + qa_synthesizer.go:63); re-scope to
   AGENT.md-ingestion-at-startup only. Also drop the body's `.claude/HANDOFF.md` dependency pointer
   (stale-doc reference per the handoff protocol).

4. **[MEDIUM — duplicate-fix hazard] qa-redesign-phase-c ticket note.** Add a one-line note that the
   NEED_CLARIFICATION addressee-routing primitive already exists (orchestrator.go:2824 + Fix 11) so
   the QA-correction path reuses it rather than re-deriving clarification parsing.

5. **[LOW — presentation ambiguity] combined-analysis "28 [FIXED]" framing.** The doc has 27 numbered
   `### [FIXED]` section headers but 28 `FIXED in <sha>` tags (the recap block at L284-316 re-tags
   several). Any summary citing "28 entries" should say "28 FIXED-in annotations across 27 section
   entries" to avoid a future auditor flagging a phantom 28th section. Not an error, just a count that
   reads two ways.

No code changes recommended or made. Read-only pass.
