---
id: "swarm-prompt-ordering-fix-2026-05-12"
status: "todo"
priority: "high"
assignee: "d34d"
epic: "swarm-context"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm", "context", "prompt-engineering"]
order: "b0"
---
# Reorder swarm task prompt — task lands too early

`runner/act-runner.mjs:602 buildTaskPrompt` currently assembles:
```
brief → task → project context → parallel → PVM → inbox → gap
```

Defect: task lands before worldview is established. Agent starts reasoning about "what do I do" before "who am I + what's the world." PVM and parallel-agent context arrive after the task is already in working memory.

**Fix order:**
```
brief → project context → parallel → PVM → task → inbox → gap
```

Rationale:
- Build identity + world first (brief, project, parallel, PVM).
- Drop the task once the agent has the worldview.
- Inbox (peer chatter) and gap (validation rejection feedback) land closest to action — they're the freshest correction signals. Gap LAST because it's the strongest steer ("your last attempt failed THIS way").

**Files:** `act-agent/runner/act-runner.mjs:602` (`buildTaskPrompt` body — reorder the `lines.push` calls).

**Success criteria:**
- Diff is local to `buildTaskPrompt`; no signature changes.
- Manual: spawn a swarm agent with all six context sources present; verify prompt section order matches the new ordering.
- Existing runner unit tests (if any) still pass.
