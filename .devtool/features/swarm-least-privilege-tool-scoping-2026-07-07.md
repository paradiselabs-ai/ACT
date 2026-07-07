---
id: "swarm-least-privilege-tool-scoping-2026-07-07"
status: "review"
priority: "high"
assignee: null
dueDate: null
created: "2026-07-07T08:33:31.000Z"
modified: "2026-07-07T08:33:31.000Z"
completedAt: null
labels: ["orchestrator", "runner", "security"]
order: "v02"
---
# Least-privilege tool scoping for Tier 2 swarm roles (orchestration plan Phase 1)

## Spec

Tier 1 already has per-role tool subsets (`agent.Tier1ToolsForRole`, KI-02). Tier 2 does not:
every swarm role gets the full `DeveloperTools` roster regardless of role, and the claude-code
backend is spawned `--print --dangerously-skip-permissions` with no per-role tool restriction
(`act-runner.mjs`). This ticket closes both, starting with the one role whose prompt already
forbids what its tools allow: `researcher` ("Analysis, not code") currently holds bash/edit/write.

Two changes:
1. **Go (act-agent backend):** `ResearcherTools()` in `internal/llm/agent/tools.go` — read-only
   subset (glob, grep, ls, sourcegraph, view, fetch + MCP + diagnostics). Dispatch in
   `app.go::CreateAgentForRole` by role; all other Tier 2 roles keep `DeveloperTools`.
2. **Runner (claude-code backend):** per-role disallowed-tools map in `act-runner.mjs`;
   researcher spawns with `--disallowedTools` covering file-mutation and shell tools.

## Success Criteria

- [ ] A researcher agent on the act-agent backend has NO bash/edit/write/patch tool in its request schema (verify: tool count + names in `~/.act/**/debug.log` prepared-messages dump)
- [ ] A researcher agent on the claude-code backend is spawned with `--disallowedTools` including Bash, Edit, Write (verify: `~/.act/runners/researcher.log` spawn line or process args)
- [ ] developer/frontend_dev/backend_dev/qa_engineer tool rosters unchanged on both backends
- [x] `go build ./...`, `go vet ./...`, `go test ./...` clean (incl. new TestResearcherToolsReadOnly locking the contract)

## Constraints

- Touch only `internal/llm/agent/tools.go`, `internal/app/app.go`, `runner/act-runner.mjs`.
- No config surface for per-role tool overrides yet (`agents.<role>.tools` deferred until a real need).
- qa_engineer keeps the full roster deliberately (writes test files, runs suites) — do not restrict it in this pass.
- No new dependencies, no refactor of the existing Tier 1 dispatch.

## Code-level Invariants

- Role fallback stays model-only: an unconfigured role NEVER falls back to another role's identity (feedback_no_role_fallback).
- Tier 1 dispatch switch in `app.go` untouched.
- Runner keeps `--dangerously-skip-permissions` (headless operation) — restriction is via `--disallowedTools`, not permission prompts.

## Status note (2026-07-07)

Implemented and unit-locked; in review pending LIVE verification of criteria 1-2 (needs an interactive LLM session — fold into the TUI e2e matrix run).
