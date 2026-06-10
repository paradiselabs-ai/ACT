---
id: "tui-tool-result-truncation-status-2026-06-06"
status: "todo"
priority: "medium"
assignee: "kareem"
dueDate: null
created: "2026-06-06T13:12:19.000Z"
modified: "2026-06-06T13:12:19.000Z"
completedAt: null
labels: ["TUI", "rendering", "ux"]
order: "a0"
---
# TUI truncates multi-line tool-call results (e.g. `act-agent:status`)

When an agent runs a CLI tool whose output spans multiple lines, the TUI clips
the result in the chat to ~1–2 lines plus an ellipsis. Most visible on
`act-agent:status`, which always renders as something like:

```
ACT Project: act-e2e — <ts>
Tasks: 2 …
```

…dropping the per-status task breakdown and the entire Agents section.

**The data is complete — this is a rendering bug, not a command bug.** Running
`act status` in a terminal produces the full output:

```
ACT Project: act-e2e — 2026-06-06T13:03:04.987Z

Tasks: 2
  pending: 2

Agents: 1
  dev-1 [developer]: offline → 73200427...
```

So the fix is in the TUI message rendering of tool-call results — the
collapse/truncation of multi-line tool output is too aggressive (or the
expand affordance is missing/unclear). Either show the full result, or make
the collapsed preview expandable so the user can read a `status` dump in full.

**Scope:** `act-agent/internal/tui/components/chat/` message rendering for
tool-call results. No server or CLI change.

**Found:** during the brownfield-onboarding e2e (2026-06-06), while diagnosing
a separate swarm-assignment stall — the truncated status was hiding the agent
state (`dev-1: offline`) that explained the stall.
