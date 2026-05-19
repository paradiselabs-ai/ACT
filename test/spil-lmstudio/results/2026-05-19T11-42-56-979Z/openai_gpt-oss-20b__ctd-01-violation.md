# openai/gpt-oss-20b — ctd-01-violation (ctd)

**score:** 0/100
**reasons:** API error: fetch failed
**elapsedMs:** 3  **tok/s:** -
**usage:** undefined

## Prompt
```
[
  {
    "role": "system",
    "content": "You audit SPIL (Structured Progressive Instruction Language) documents for CTD violations.\n\n=== SPIL primer ===\n\nSPIL uses two primitives:\n- @keyword \"value\" or @keyword:    — structured machine-parseable sections\n- > \"text\"                          — natural language directives\n\nCTD (Conceptually Top-Down) is the core design rule: every line earns its position by depending only on what's ABOVE it. The progression is one-directional:\n\n  core concept → foundational → clarifying → nuanced → specificity → preferential\n\nStandard section order for a task spec:\n  1. Core metadata: @task, @project, @priority\n  2. Core directive: one `>` establishing scope\n  3. @data       — data structures the task touches\n  4. @context    — what already exists (files, services to reference)\n  5. @success_criteria  — testable outcomes (Assurance validation checklist)\n  6. Nuanced `>` directives — implementation constraints\n  7. @error_handling  — failure conditions referring to data + steps above\n  8. Preferential `>` directives — style, conventions (last)\n\nViolation = a section references a concept that has not yet been introduced. Example: @error_handling cannot mention fields that don't appear in @data above it. @success_criteria cannot reference files not declared in @context above it. A preferential style directive at the top (before @data) is also a violation — style only makes sense after you know what's being styled.\n\n=== Task ===\n\nAudit the document below. Output exactly one JSON object, no prose, no markdown fences:\n\n{ \"valid\": boolean, \"violations\": [ { \"section\": string, \"reason\": string } ] }\n\nIf no violations: { \"valid\": true, \"violations\": [] }. Be strict but not paranoid — only flag sections where a referenced concept is genuinely missing from above."
  },
  {
    "role": "user",
    "content": "Audit this SPIL document:\n\n@task \"Add user profile picture upload\"\n@project \"my-app\"\n@priority \"medium\"\n\n@error_handling:\n  - file_too_large: condition: \"uploaded.size > max_upload_bytes\"; response: \"413 with PAYLOAD_TOO_LARGE code\"\n  - invalid_mime: condition: \"uploaded.mime not in allowed_mimes\"; response: \"415 with UNSUPPORTED_MEDIA code\"\n  - storage_failure: condition: \"S3 PutObject returns 5xx\"; response: \"500 with STORAGE_ERROR — log to ops alert channel\"\n\n> \"Picture uploads happen on the profile edit page. Users expect immediate visual feedback.\"\n\n@data:\n  - max_upload_bytes: 5242880\n  - allowed_mimes: [image/jpeg, image/png, image/webp]\n  - storage_key_format: \"users/{user_id}/avatar.{ext}\"\n\n@context:\n  - S3 client at /src/storage/s3.ts (already configured with bucket + creds)\n  - UserService at /src/services/UserService.ts — has updateAvatarUrl(userId, url) method\n\n@success_criteria:\n  - POST /api/users/:id/avatar accepts multipart/form-data\n  - Valid uploads are stored at storage_key_format path\n  - UserService.updateAvatarUrl is called with the resulting URL\n  - Invalid uploads return appropriate 4xx response without writing to S3\n  - Integration test covers all three error cases"
  }
]
```

## Raw response
```
(API error: fetch failed)
```

## Stripped (thinking removed)
```

```