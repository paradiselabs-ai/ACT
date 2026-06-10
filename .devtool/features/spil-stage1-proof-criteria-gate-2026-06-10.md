---
id: "spil-stage1-proof-criteria-gate-2026-06-10"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-06-10T00:00:00.000Z"
modified: "2026-06-10T00:00:00.000Z"
completedAt: null
labels: ["SPIL", "language", "orchestrator", "assurance"]
order: "a1"
---
# SPIL stage-1 proof: `@success_criteria` as an evaluated fail-closed gate

**Smallest slice that proves the thesis** "the spec IS the enforcement." Today
`@success_criteria` is parsed only at grading time (`extractSuccessCriteria`). Promote it to
a runtime invariant the orchestrator *enforces*: a task whose criteria aren't met by Assurance
**cannot** advance — fails closed, no merge, loops back with the unmet items injected.

This is the demo-able / HN-worthy unit. One screen recording: task submitted → criteria
unmet → orchestrator blocks it → criteria met → advances.

Parent epic: `spil-evolve-to-agentic-language-2026-06-10`.

## Success criteria
- A submitted task with unmet `@success_criteria` is **blocked from advancing** by code
  (not prompt) — verify by submitting work that fails one criterion and confirming it does
  not reach QA/synthesis.
- The block is **fail-closed**: empty / unparseable `@success_criteria` blocks too (do not
  silently pass). (Cross-check existing ticket `assurance-fail-closed-empty-criteria-2026-05-26`
  — reuse, don't duplicate.)
- Unmet criteria items are injected back into the retry turn (targeted, not a full reload).
- `go build -o act-agent .` clean in `act-agent/`; server `npm run build` clean.

## Code constraints
- Touch the validation/routing path in `act-agent/internal/app/orchestrator.go` and the
  Assurance gate only. No new SPIL grammar yet — reuse `extractSuccessCriteria`.
- No s-expression work. No speculative evaluator abstraction — this is the proof slice.
- Preserve the `@`/`>` split: this enforces an `@` form; do not touch `>` directive handling.

## Docs to reference
- `docs/Vault/Agent Coordination Toolkit/nestty/SPIL-as-language-evolution.md` — §"What
  changes" (PROMOTE row) and §"Build order" item 1
- `docs/Vault/Agent Coordination Toolkit/nestty/plans/SPIL Integration Plan.md` — confirms
  `extractSuccessCriteria` is the one in-production function
- `server/src/services/SPILParser.ts` — the extractor to reuse
- `act-agent/internal/app/orchestrator.go` — validation routing (where the gate lives)
