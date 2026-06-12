---
title: Code Quality Improvements — ACT Codebase (audit 2026-05-27)
status: current
verified_against: unknown-2026-05-27
owner: project-owner
last_verified: 2026-05-27
---

> Relocated from `docs/refactor/` on 2026-06-12 (janitorial sweep) — audit outputs live in
> `docs/audits/` (DOC_STANDARDS §1). Claims are as-of 2026-05-27: re-grep before acting (Art. 2).

# Code Quality Improvements — ACT Codebase

> Audit date: 2026-05-27 | Scope: `act-agent/` (Go) + `server/` (TypeScript)

---

## 1. `server/src/index.ts` — 1599-line Monolithic Route File

### Severity: High

All REST endpoints, Socket.io handlers, in-memory stores (`projects`, `agentInboxes`, `fileLocks`), and startup logic live in a single file.

- `normalizeStatus` (line 48) should live in `TaskCoordinator`.
- `getProjectStatusSummary` (line 64) is dead code — never called.
- The startup `restoreFromLog` call duplicates logic from `ChronologicalLog.restoreFromLog`.

**Fix:** Split into `routes/agents.ts`, `routes/tasks.ts`, `routes/projects.ts`, `routes/validation.ts`. Encapsulate `fileLocks` and `agentInboxes` in dedicated service classes.

---

## 2. `act-agent/internal/app/orchestrator.go` — 2698-line God Object

### Severity: High

The `Orchestrator` struct contains background loops, anomaly detection, JSON parsing, prompt building, task dispatch, and seen-task tracking.

- `detectAnomalies` (line 1926) uses **bubble sort** (O(n²)) — replace with `sort.Slice`.
- `extractJSONContaining` and `extractBalancedJSONFrom` (lines 2361-2466) are near-duplicates — unify into one implementation.
- `parseCreateTaskDirectives` (line 2477) has 4 return values and deeply nested conditionals — split into `parseInlineCreateTasks` + `parseWrapperCreateTasks`.
- `fireWhenPlannerIdle` (line 1332) busy-waits at 200ms intervals for up to 60s — replace with channel-based signal.
- `routeToAssurance` and `routeToQA` duplicate message-fetching logic — extract `lastMessageFrom(ctx, sessionID, role)`.
- `buildStatusSnapshot` (line 1852) spawns 4 goroutines with no timeout — use `errgroup` with context cancellation.

---

## 3. `act-agent/internal/act/client.go` — Duplicated HTTP Patterns

### Severity: Medium

- `post`, `getJSON`, `getString` (lines 277-337) duplicate URL construction, error wrapping, and status checking.
- `SetTaskDependencies`, `RetryTask`, `AbandonTask` each manually construct `http.NewRequest` instead of using helpers.
- `Register` (line 46) checks for `"409"` in the error *string* rather than `resp.StatusCode`.

**Fix:** Add a single `do(req) ([]byte, error)` method and a `patch(path, body)` helper. Check status codes, not error strings.

---

## 4. `act-agent/internal/llm/prompt/common.go` — `actCLICommands()` Giant Switch

### Severity: Medium

An 85-line switch statement with ~60% duplicated text across roles. Adding a CLI command requires touching 5-7 branches.

**Fix:** Build commands compositionally — factor out `tier1Preamble`, `commonCommandsFor(role)`, `roleSpecificCommands(role)`.

---

## 5. `server/src/services/ChronologicalLog.ts` — `restoreFromLog` 200-line Switch

### Severity: Medium

Lines 472-716 replay events by mutating external Maps passed by reference. The `task_completed` case has backward-compat logic for pre-split events that should be a one-time migration.

**Fix:** Extract into a `LogReplayer` class with pure functions. Move compat logic to a migration script. Make `coerceTaskDates` return new objects.

---

## 6. `server/src/services/TaskCoordinator.ts` — `detectCycles` Memory Waste

### Severity: Low

Line 469: `dfs(dep, [...path])` copies the path array on every recursive call — O(N·E) allocations.

**Fix:** Use backtracking with a single `path` array; copy only when a cycle is found.

---

## 7. `act-agent/internal/runner/spawner.go` — Dangerous `SweepOrphans`

### Severity: High

Line 243: `pkill -f act-runner.mjs` kills **all** `act-runner.mjs` processes on the machine, including other ACT sessions.

**Fix:** Track child PIDs explicitly via `~/.act/runners/*.pid` files; kill only known orphans.

---

## 8. `act-agent/internal/llm/prompt/planner.go` — Escaped String Constant

### Severity: Low

The 103-line `basePlannerPrompt` uses heavy backtick escaping (`` ` ``) making it hard to read and edit.

**Fix:** Use `//go:embed planner_prompt.md` to keep the prompt as a plain markdown file.

---

## 9. `act-agent/internal/llm/prompt/prompt.go` — Silent Error Suppression

### Severity: Low

Line 151: `filepath.WalkDir` errors are silently discarded.

**Fix:** Log errors at debug level instead of dropping them.

---

## 10. `act-agent/internal/runner/spawner.go` — `findRunnerScript` Hardcoded Paths

### Severity: Medium

Lines 301-342: 7 hardcoded relative path candidates encoding assumptions about directory layout.

**Fix:** Embed the runner script into the Go binary with `//go:embed` for zero-dependency distribution.

---

## 11. `server/src/services/ChronologicalLog.ts` — Per-Project Log Lacks `fsync`

### Severity: Medium

Line 234: per-project mirror uses `fs.appendFile` (no `fsync`) while the main log uses `open`/`write`/`sync`/`close`. Data loss risk on crash.

**Fix:** Use the same durable write pattern for per-project logs.

---

## 12. `act-agent/internal/llm/prompt/*.go` (all prompt files) — Unused `ModelProvider` Parameter

### Severity: Low

Every prompt function accepts `models.ModelProvider` but ignores it. Dead interface contract.

**Fix:** Either wire provider-specific prompt variations or remove the parameter.

---

## Summary

| # | File | Severity | Issue |
| --- | --- | --- | --- |
| 1 | `server/src/index.ts` | High | 1599-line monolith; split into route modules |
| 2 | `orchestrator.go` | High | 2698-line god object; extract components |
| 3 | `act/client.go` | Medium | Duplicated HTTP helpers; add `do()`/`patch()` |
| 4 | `prompt/common.go` | Medium | 85-line switch; compose commands |
| 5 | `ChronologicalLog.ts` | Medium | 200-line replay switch; extract `LogReplayer` |
| 6 | `TaskCoordinator.ts` | Low | O(N·E) path copying; use backtracking |
| 7 | `runner/spawner.go` | High | `pkill -f` kills all sessions; use PID files |
| 8 | `prompt/planner.go` | Low | Escaped string; use `//go:embed` |
| 9 | `prompt/prompt.go` | Low | Silent error suppression; log errors |
| 10 | `runner/spawner.go` | Medium | Hardcoded paths; embed runner script |
| 11 | `ChronologicalLog.ts` | Medium | Missing `fsync` on per-project log |
| 12 | `prompt/*.go` | Low | Unused `ModelProvider` parameter |

**Highest-value items:** #1 (server route splitting), #2 (orchestrator decomposition), #7 (fix dangerous `pkill -f`).
