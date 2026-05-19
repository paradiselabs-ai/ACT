# qwen/qwen3.6-35b-a3b — ctd-02-valid (ctd)

**score:** 0/100
**reasons:** No JSON object extractable from response
**elapsedMs:** 52100  **tok/s:** 28.79
**usage:** {"prompt_tokens":701,"completion_tokens":1500,"total_tokens":2201,"completion_tokens_details":{"reasoning_tokens":1442}}

## Prompt
```
[
  {
    "role": "system",
    "content": "You audit SPIL (Structured Progressive Instruction Language) documents for CTD violations.\n\n=== SPIL primer ===\n\nSPIL uses two primitives:\n- @keyword \"value\" or @keyword:    — structured machine-parseable sections\n- > \"text\"                          — natural language directives\n\nCTD (Conceptually Top-Down) is the core design rule: every line earns its position by depending only on what's ABOVE it. The progression is one-directional:\n\n  core concept → foundational → clarifying → nuanced → specificity → preferential\n\nStandard section order for a task spec:\n  1. Core metadata: @task, @project, @priority\n  2. Core directive: one `>` establishing scope\n  3. @data       — data structures the task touches\n  4. @context    — what already exists (files, services to reference)\n  5. @success_criteria  — testable outcomes (Assurance validation checklist)\n  6. Nuanced `>` directives — implementation constraints\n  7. @error_handling  — failure conditions referring to data + steps above\n  8. Preferential `>` directives — style, conventions (last)\n\nViolation = a section references a concept that has not yet been introduced. Example: @error_handling cannot mention fields that don't appear in @data above it. @success_criteria cannot reference files not declared in @context above it. A preferential style directive at the top (before @data) is also a violation — style only makes sense after you know what's being styled.\n\n=== Task ===\n\nAudit the document below. Output exactly one JSON object, no prose, no markdown fences:\n\n{ \"valid\": boolean, \"violations\": [ { \"section\": string, \"reason\": string } ] }\n\nIf no violations: { \"valid\": true, \"violations\": [] }. Be strict but not paranoid — only flag sections where a referenced concept is genuinely missing from above."
  },
  {
    "role": "user",
    "content": "Audit this SPIL document:\n\n@task \"Add /healthz endpoint\"\n@project \"my-app\"\n@priority \"low\"\n\n> \"Lightweight liveness probe for Kubernetes. No auth, no DB calls, must respond under 50ms.\"\n\n@data:\n  - response_shape: { status: 'ok', version: string, uptime_seconds: number }\n\n@context:\n  - Express app instance at /src/app.ts — register routes via app.get(...)\n  - process.uptime() returns seconds since process start\n  - package.json version field is the source of truth for version string\n\n@success_criteria:\n  - GET /healthz returns 200 with response_shape JSON\n  - Endpoint responds in under 50ms p99 (measured under load test)\n  - No auth middleware applied to this route\n  - Integration test verifies status + shape\n\n> \"Do not query the database from this endpoint — it must remain cheap even when DB is degraded.\"\n\n@error_handling:\n  - internal_failure: condition: \"unhandled exception in handler\"; response: \"500 with { status: 'error' } — kept minimal to avoid leaking internals\"\n\n> \"Keep handler implementation under 20 lines. Inline the version read; no need to abstract.\""
  }
]
```

## Raw response
```

```

## Stripped (thinking removed)
```

```