---
id: "ki-14-terminal-app-unsupported-note-2026-04-21"
status: "done"
priority: "low"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-05-07T09:55:00.000Z"
completedAt: "2026-05-07T09:55:00.000Z"
labels: ["docs", "TUI"]
order: "a09"
---
# KI-14: Document Apple Terminal.app as unsupported

Add entry to `docs/Vault/Agent Coordination Toolkit/nestty/KNOWN_ISSUES.md`:
- Symptom: TUI "press enter to unfreeze" hang in Apple Terminal.app.
- Root cause: Terminal.app excluded from Bubbletea's `shouldQuerySynchronizedOutput` list → mode 2026 filter is a no-op there. Terminal.app-specific ANSI-buffering behavior.
- Status: **will not fix**. Recommend Ghostty, iTerm2, Alacritty, kitty.

Also add to README/quickstart as user-facing supported-terminal list.
