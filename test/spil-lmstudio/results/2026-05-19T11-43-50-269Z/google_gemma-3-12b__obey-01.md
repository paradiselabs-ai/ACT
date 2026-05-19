# google/gemma-3-12b — obey-01 (parse)

**score:** 100/100
**reasons:** (none)
**elapsedMs:** 11927  **tok/s:** 9.98
**usage:** {"prompt_tokens":295,"completion_tokens":119,"total_tokens":414,"completion_tokens_details":{"reasoning_tokens":0}}

## Prompt
```
[
  {
    "role": "system",
    "content": "You extract @success_criteria from SPIL task specs. Output exactly one JSON object with this schema:\n{ \"sections\": [ { \"keyword\": \"success_criteria\", \"items\": string[] } ] }\nInclude ONLY the success_criteria section. No prose, no markdown, no other sections."
  },
  {
    "role": "user",
    "content": "Extract @success_criteria from:\n\n@task \"Add cache layer to UserService\"\n@project \"my-app\"\n\n> \"Reduce DB load for read-heavy user lookup paths.\"\n\n@data:\n  - cache_key_format: \"user:{id}\"\n  - ttl_seconds: 300\n\n@context:\n  - UserService at /src/services/UserService.ts — wrap existing methods, do not break signatures\n  - Redis client at /src/db/redis.ts — already initialized\n\n@success_criteria:\n  - UserService.findById hits cache on second call within TTL window\n  - Cache miss falls back to DB and populates cache\n  - UserService.update invalidates the cached entry for that user_id\n  - All existing UserService unit tests still pass without modification\n  - New tests cover cache hit, cache miss, and invalidation paths\n\n> \"Do not change UserService method signatures.\"\n\n@error_handling:\n  - redis_unavailable: log warning, fall through to DB, do not throw"
  }
]
```

## Raw response
```
```json
{
  "sections": [
    {
      "keyword": "success_criteria",
      "items": [
        "UserService.findById hits cache on second call within TTL window",
        "Cache miss falls back to DB and populates cache",
        "UserService.update invalidates the cached entry for that user_id",
        "All existing UserService unit tests still pass without modification",
        "New tests cover cache hit, cache miss, and invalidation paths"
      ]
    }
  ]
}
```
```

## Stripped (thinking removed)
```
```json
{
  "sections": [
    {
      "keyword": "success_criteria",
      "items": [
        "UserService.findById hits cache on second call within TTL window",
        "Cache miss falls back to DB and populates cache",
        "UserService.update invalidates the cached entry for that user_id",
        "All existing UserService unit tests still pass without modification",
        "New tests cover cache hit, cache miss, and invalidation paths"
      ]
    }
  ]
}
```
```