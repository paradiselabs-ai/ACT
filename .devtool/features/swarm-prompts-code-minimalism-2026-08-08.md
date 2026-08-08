---
id: "swarm-prompts-code-minimalism-2026-08-08"
status: "todo"
priority: "medium"
created: "2026-08-08T15:40:00.000Z"
completedAt: null
labels: ["prompts", "swarm"]
order: "a3"
---
# Swarm dev prompts: bake in lazy-senior-dev minimalism (ponytail principles)

## Describe
Owner call (2026-08-08): swarm dev agents should behave like the "ponytail" lazy
senior dev skill — fewest files, stdlib/native-first ladder, no speculative
abstractions, shortest working diff, deletion over addition, one runnable
self-check for non-trivial logic. Evidence of need: link-dock run produced TWO
overlapping test suites and a server that listens on import (no main-guard) —
minimalism + check-what-exists-first would have prevented both.

## Success Criteria
- A distilled minimalism block (~10 lines max, not the full skill) added to the
  shared Tier-2 prompt path (prompt/common.go or per-dev-role files):
  need-to-exist ladder, no unrequested abstractions, fewest files, check
  existing files before creating parallel ones (esp. tests), main-guard rule
  for servers, one self-check per non-trivial unit.
- Applies to developer/frontend_dev/backend_dev; researcher/qa_engineer get the
  check-existing-first line only.
- Token diet respected; planner-prompts freshness impact noted.
