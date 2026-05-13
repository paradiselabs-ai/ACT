---
id: "sighup-trap-status-reconcile-2026-04-21"
status: "todo"
priority: "low"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["housekeeping"]
order: "a12"
---
# Reconcile SIGHUP trap status

Commit `66c4b4b` ("TUI: trap SIGHUP and SIGTERM so window-close runs clean shutdown") exists on some branch, but earlier HANDOFF sections say "SIGHUP trap still pending ~10 lines of work." Confirm whether `66c4b4b` is on `NesTTY` tip; if yes, close the work item. If commit exists but isn't on NesTTY, pause and ask user before cherry-picking (branch-topology decision).

See HANDOFF Track A.3.
