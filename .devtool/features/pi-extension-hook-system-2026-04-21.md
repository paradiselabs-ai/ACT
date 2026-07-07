---
id: "pi-extension-hook-system-2026-04-21"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-07-07T08:33:31.000Z"
completedAt: null
labels: ["hooks", "extensibility", "orchestrator"]
order: "z3"
---
# HookRunner: pre/post tool-use hooks (orchestration plan Phase 2a)

Originally from `badlogic/pi-mono` (see FUTURE_VISION.md "Extension / Hook System").
Re-scoped 2026-07-07 to the concrete design already written in
`docs/ARCHITECTURE_PATTERNS.md` §3 — implement THAT, nothing more.

## Spec

A `HookRunner` wrapping `BaseTool.Run` in the tool-assembly layer
(`internal/llm/agent/tools.go` / `internal/llm/tools/`), with exactly two built-in hooks:

1. **PreToolUse on edit/write/patch → auto file-claim** via the Go act client's
   `ClaimFiles`/`ReleaseFiles` (`internal/act/client.go` — kept unwired precisely for this;
   see client-go-unwired-methods-2026-06-14 resolution). Claim held by another agent → deny
   the tool call with a "file locked by <agent>" tool error.
2. **PostToolUse → ChronLog event** via the existing client, so every tool call becomes
   Observer-visible and PVM-indexable.

No user-configurable hook scripts in this pass — the two built-ins prove the interface.

## Success Criteria

- [ ] Two swarm agents editing the same file: second agent's edit tool call is denied with the lock-holder named (repro: parallel tasks on one file, check both runner logs)
- [ ] Every tool call by a hooked agent appears as a ChronLog event (verify `server/data/coordination-log.jsonl`)
- [ ] Tool behavior with no hooks configured is byte-identical to today
- [ ] `go build/vet/test` clean

## Constraints

- Hook interface per ARCHITECTURE_PATTERNS §3: `PreToolUse(tool, input) → allow|deny|modify`, `PostToolUse(tool, input, output)`.
- No script execution, no config surface, no hook ordering framework.
- Server file-lock endpoints already exist — do NOT rebuild them.

## Code-level Invariants

- Hooks live in the agent layer, never inside individual tool implementations.
- A hook error must fail the TOOL CALL, not crash the agent turn.
- Three-Layer rule: hooks call the server via the existing client; no new decision logic server-side.

## Dependencies

- client-go-unwired-methods-2026-06-14 (keeps ClaimFiles/ReleaseFiles)
- Related: block14-claude-code-swarm-hooks-wizard-2026-04-21 (claude-code backend equivalent — separate surface, do not merge)
