---
id: "block7-spil-parser-full-grammar-2026-04-21"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["v1-gate", "SPIL", "parser", "block-7"]
order: "b01"
---
# Block 7 — SPIL Parser (full grammar)

**File**: `server/src/services/SPILParser.ts` (exists as extraction-only MVP).

1. Lexical analysis — tokenize `@` section markers + `>` directives
2. Syntax parse — structured section tree
3. Semantic analysis — resolve cross-section references, validate CTD ordering (section N only references 1..N-1)
4. Runtime API — `getSuccessCriteria(spec)`, `getSectionBody(name)`, `validateEdit(orig, edited)` for config-browser safe-edit path

**Rationale**: SPIL must ship active in v1 so we can A/B test SPIL vs non-SPIL on same Claude Code backend.

Depends on SPIL one-pager spec. See BUILD_ORDER.md Block 7.
