---
id: "tui-first-prompt-keypress-hang-2026-04-21"
status: "done"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-05-07T10:05:00.000Z"
completedAt: "2026-05-07T10:05:00.000Z"
labels: ["bug", "TUI", "UX"]
order: "a10"
---
# P4: First-prompt keypress hang

After submitting the first prompt of a new session, the TUI hangs until the user presses Enter a second time. Subsequent prompts don't require this. Likely session-creation path blocking on a response that only arrives after a tick provided by the key-release event.

**Investigation**:
- `act-agent/internal/tui/tui.go` Update handling for initial session creation
- `act-agent/internal/tui/components/chat/` editor submission path
- `KeyPressMsg` vs `KeyMsg` handling on first-prompt path (see commit `1d89e5e`)
- Bubbletea v2 landmines (input/focus/alt-screen moved to View struct fields)

Cosmetic/UX annoyance, not an integrity issue.
