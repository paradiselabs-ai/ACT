---
id: "client-go-unwired-methods-2026-06-14"
status: "done"
priority: "low"
assignee: null
dueDate: null
created: "2026-06-14T12:00:00.000Z"
modified: "2026-07-07T00:00:00.000Z"
completedAt: "2026-07-07T00:00:00.000Z"
labels: ["cleanup", "go", "client", "dead-code"]
order: "z0"
---
# client.go — 9 unwired methods (decide: wire or delete)

## Spec
`act-agent/internal/act/client.go` is the Go native HTTP client for the ACT server (header:
"replaces the previous shell-out implementation"). It mirrors the full server API, but 9 of its 25
methods have **zero non-test Go callers** — the operations they wrap are performed elsewhere (runner
JS direct HTTP, TS CLI commands, or the Planner via the `act_cli` tool). They are dead code: a
complete API surface built ahead of need, not a functional bug. Decide per method: **wire** it
(replace the equivalent shell-out/JS path with the native Go call) or **delete** it (keep the client
lean). This is the owner's call; ticket stays `backlog` until the wire-vs-delete decision is made.

Unwired at d59785a (only definition lines exist):
- `ReportProgress` (:136), `ReportComplete` (:151), `SubmitForValidation` (:212) — runner does these
  via direct HTTP (`act-runner.mjs:173/183/194`).
- `ClaimFiles` (:166), `ReleaseFiles` (:182) — only the TS CLI does locking; Go never locks
  (consistent with file-locking being prompt-only, per HANDOFF).
- `RetryTask` (:403), `AbandonTask` (:424) — Planner affordance goes through the `act_cli` tool → TS CLI.
- `PVMSearch` (:230) — Planner PVM search goes through the `act_cli` tool → CLI → `/api/pvm/search`;
  this native method (the thing client.go exists to replace shell-outs with) was never swapped in.
- `Status` (:225) — no caller (the `/status` slash command reads `runnerSpawner.Status()`).

## Success Criteria
- Either: every kept method has ≥1 non-test Go caller; or: each removed method is deleted and
  `go build ./...` is clean.
- `for m in Status PVMSearch ReportProgress ReportComplete SubmitForValidation ClaimFiles ReleaseFiles RetryTask AbandonTask; do rg -n "\b${m}\(" act-agent --type go | rg -v _test; done`
  returns, per method, either a real caller (wired) or nothing (deleted) — never a lone definition line.
- No behavior change to the runner or TS CLI paths (Go-side cleanup only).

## Constraints
- Touch only `internal/act/client.go` (+ orchestrator/app call sites IF wiring). Do NOT change
  `runner/act-runner.mjs` or any `cli/*.ts`.
- No new abstractions. If deleting, remove only the 9 verified-unwired methods; leave the wired 16
  (Register, GetContext, CreateProject, CreateTask, SetTaskDependencies, SubmitVerdict,
  SubmitSynthesis, SendMessage, IsAvailable, ListTasks, GetPendingValidation, GetValidatedTasks,
  ListAgents, GetFileLocks, GetLog, GetProject) intact.

## Invariants (code-level)
- After the change, no method on `Client` is silently dead: each is either called from non-test Go,
  or deleted, or retained with an explicit `// kept: complete client surface` note.
- The wired-16 call sites are unchanged.

## Repro/Evidence
Sweep: `docs/audits/unwired-code-sweep-2026-06-14.md`. Per-method grep returns only the `func (c *Client)`
definition line for the 9 listed (no callers). Severity LOW — dead code, not a functional defect.

## Resolution (2026-07-07)
Decision: delete 7, keep 2. Deleted `ReportProgress`, `ReportComplete`, `SubmitForValidation`,
`Status`, `PVMSearch`, `RetryTask`, `AbandonTask` — their jobs live in the runner's direct HTTP
calls and the `act_cli` → TS CLI path. Re-verified zero non-test callers before each delete
(the remaining `Status(` hits are `Spawner.Status()`, a different type). No dedicated
request/response structs existed (all inline `map[string]any` bodies), so nothing else to remove.
Kept `ClaimFiles`/`ReleaseFiles` with `// kept: reserved for the PreToolUse file-claim hook
(hooks plan Phase 2)` notes. `go build ./...`, `go vet ./...`, `go test ./...` all clean.
