---
id: "tui-scroll-up-to-see-current-session-history-2026-04-21"
status: "done"
priority: "medium"
assignee: "kareem"
dueDate: null
created: "2026-04-21T17:16:00.885Z"
modified: "2026-05-08T00:00:00.000Z"
completedAt: "2026-05-08T00:00:00.000Z"
labels: ["TUI", "design"]
order: "a0"
---
# TUI Scroll up to see current session history

the NesTTY Go TUI for ACT cant scroll at all. it would be nice to scroll up to see the current session history as sometimes the planner messages are large and hard to read before another message appears, pushing the one you are reading higher up and above the view, but you cant scroll up

## Resolution (2026-05-08, Kareem)

Fixed in commit `39acdc1` ("Add harmonica animations, splash fade, and UI polish"). Two complementary mechanisms:

1. **Mouse wheel scrolling** — `tea.MouseWheelMsg` handled in chat list component, adjusts viewport offset directly.
2. **Scroll-focus mode** — `ScrollFocusMsg`/`ScrollMsg` toggle and message types. When focused (Tab to enter, Esc to exit), arrow keys move the viewport. Existing `PageUp`/`PageDown`/`HalfPageUp`/`HalfPageDown` keys continue to work.

Files touched: `act-agent/internal/tui/components/chat/list.go`, `chat.go`, `message.go`. Status bar shows `scroll  tab  focus` hint when relevant.