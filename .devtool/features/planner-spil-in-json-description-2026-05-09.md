---
id: "planner-spil-in-json-description-2026-05-09"
status: "in-progress"
priority: "high"
assignee: "d34d"
dueDate: null
created: "2026-05-09T08:30:00.000Z"
modified: "2026-05-09T08:35:00.000Z"
completedAt: null
labels: ["planner", "bug", "spil", "alpha-blocker"]
order: "a0"
---
# Planner emits malformed CREATE_TASK JSON when including dependencies

## Symptom

Planner emits CREATE_TASK directives with `@dependencies` SPIL section text mid-string in the JSON `description`, e.g.:

```
"description":"@task\n> Implement CSRF\n@success_criteria\n- ...\n@dependencies":["Core database operations"]
```

The `\n@dependencies` text terminates the description string mid-thought, then the parser sees `:["..."]` as a malformed continuation. Orchestrator's `parseCreateTaskDirectives` rejects → no tasks created on the server → user thinks tasks were dispatched but Observer says "0 active task(s)".

## Root cause

The Planner prompt at `act-agent/internal/llm/prompt/planner.go` showed an example with `@task\n` and `@success_criteria\n` inside the `description` JSON string but didn't explicitly forbid additional `@`-sections. When the model wanted to express dependencies, it improvised `@dependencies` inside the description AND tried to use it as a top-level JSON key — producing structurally invalid JSON.

Compounded by free-tier model adherence: GLM-4.5 Air, gpt-oss-120b, and similar smaller models invent JSON shape liberally when given partial templates.

## Fix (this session)

Tightened `prompt/planner.go` Step 2 (CREATE_TASK shape rules):

1. Added explicit rule: description contains EXACTLY `@task` + `@success_criteria`. No other `@`-sections.
2. Documented `dependencies` as a top-level JSON property (array of task titles).
3. Added example: `"dependencies":["Database schema"]` in the canonical CREATE_TASK example.
4. Added rule about JSON-on-one-line + `\n` for newlines.
5. Added rule about `act_cli` args always being array (separate but related model adherence issue).

## Status

**In-progress.** Prompt change made but needs validation:

1. Restart ACT, reproduce the original "Build a Go login + signup web app" intake.
2. Verify Planner emits valid CREATE_TASK JSON with proper top-level `dependencies` array.
3. Verify orchestrator parses each CREATE_TASK successfully.
4. Verify swarm spawns and starts working.

If model still emits malformed JSON despite tightened prompt, escalate to:
- More aggressive parser repair (regex-based pre-processing to fix common malformations)
- Switch Planner to a stronger model (Claude Sonnet, Opus, GPT-5) for the demo
- Add few-shot examples in the prompt with multiple correct CREATE_TASK shapes

## Success criteria

1. Planner reliably emits valid CREATE_TASK JSON across at least 5 fresh INTAKE flows on different project types.
2. Orchestrator parses 100% of valid CREATE_TASK directives.
3. `act-agent:tasks` palette command shows the created tasks immediately after Planner's BUILD-mode response.
4. No "0 active task(s)" Observer ping when tasks should have been created.

## Priority

**HIGH — alpha blocker.** Without this, the demo flow breaks: user gives Planner a real project, Planner creates tasks, but server receives nothing. Demo is dead.
