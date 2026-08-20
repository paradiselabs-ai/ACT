---
id: "acp-close-hangs-on-non-exiting-host-2026-08-20"
status: "done"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-20T00:00:00.000Z"
modified: "2026-08-20T00:00:00.000Z"
completedAt: "2026-08-20T10:00:00.000Z"
labels: ["orchestrator", "acp"]
order: "a1"
---
# ACP shutdown can hang when the host CLI doesn't exit on stdin EOF (devin)

## Spec
`ACPAgent.Close()` (internal/acp/agent.go) calls `client.Close()` BEFORE killing the process group, and `Client.Close()` blocks on `<-c.done`, which only fires when the host closes stdout. `devin acp` does not exit on stdin EOF (read loop still blocked in `ReadFrame` 3 min after stdin close — observed live 2026-08-20 during devin-backend integration, devin 3000.1.27). claude-code/gemini exit cleanly on EOF, so this never bit before. Effect: TUI quit or `/backend restart <role>` on a devin-backed Tier-1 role can hang.

## Success Criteria
- Closing an ACPAgent whose host never exits returns within a bounded time (e.g. ≤5s) and the process group is killed.
- claude-code/gemini shutdown behavior unchanged (clean exit still preferred over kill).
- `go test ./internal/acp/...` passes, including a test faking a non-exiting host.

## Constraints
- Fix ordering/timeout in the Close path only (internal/acp/agent.go + transport/client as needed). No protocol changes, no per-backend special cases if a generic bounded-close covers it.

## Invariants (code-level)
- `Client.Close()` (or its caller) has a timeout or the process-group kill precedes/parallels the blocking wait — no unbounded `<-c.done` on the shutdown path.

## Repro/Evidence
devin-backend-opus session, 2026-08-20 (act-coordination.json task_complete entry): stdin closed, host's read loop still alive 3 min later; Close blocked.

## Fix (2026-08-20, uncommitted)
ACPAgent.Close now SIGTERMs the process group BEFORE waiting on the client; Client.Close waits at most 5s (`closeWait`) for the reader loop. Test `TestClient_CloseBoundedWhenHostNeverExits` fakes a host that keeps stdout open. Verified: `go test ./internal/...` green; live `devin acp` exits on SIGTERM, so shutdown is now immediate on the normal path.
