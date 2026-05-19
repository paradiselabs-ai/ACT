# qwen/qwen3.6-35b-a3b — parse-01 (parse)

**score:** 0/100
**reasons:** API error: This operation was aborted
**elapsedMs:** 196752  **tok/s:** -
**usage:** undefined

## Prompt
```
[
  {
    "role": "system",
    "content": "You are a strict structured-data extractor. Your only output is a single JSON object on stdout, no prose, no markdown fences, no preamble. The schema is:\n{\n  \"sections\": [ { \"keyword\": string, \"value\"?: string, \"items\": string[], \"directives\": string[] } ],\n  \"directives\": string[]\n}\nRules:\n- @keyword \"value\"  — single-line inline → section with keyword + value, no items\n- @keyword:         — section header, content follows below\n- - item            — list item belongs to the most recent @section\n- > \"text\"          — natural language directive; belongs to the most recent @section, or to top-level `directives` if no section is open\n- // comments       — ignore\nReturn ONLY the JSON object."
  },
  {
    "role": "user",
    "content": "Parse this SPIL document:\n\n@task \"Build JWT auth\"\n@project \"my-app\"\n@priority \"high\"\n\n> \"Security foundation. Other tasks blocked on this.\"\n\n@data:\n  - user_schema: { id, email, passwordHash }\n  - token_payload: { userId, iat, exp }\n\n@context:\n  - UserService at /src/services/UserService.ts\n  - ApiResponse type at /src/types/response.ts\n\n@success_criteria:\n  - POST /auth/login returns ApiResponse<{accessToken, refreshToken}>\n  - POST /auth/refresh issues new accessToken\n  - POST /auth/logout invalidates refresh token\n  - Auth middleware blocks unauthorized requests with 401\n  - Unit tests pass for all three endpoints\n\n> \"Access tokens 15m TTL. Refresh tokens 7d, stored server-side.\"\n\n@error_handling:\n  - invalid_credentials returns 401 generic message\n  - expired_token returns 401 with TOKEN_EXPIRED code\n\n> \"Async/await only. No callbacks.\""
  }
]
```

## Raw response
```
(API error: This operation was aborted)
```

## Stripped (thinking removed)
```

```