---
id: "compaction-slash-and-palette-command-2026-05-12"
status: "todo"
priority: "medium"
assignee: "d34d"
epic: "compaction"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["compaction", "tui", "ux"]
order: "a2"
---
# Add /compact slash command + act-agent:compact palette entry

Only the auto-trigger path exists for compaction today. User cannot manually compact on demand. Adding a manual entry point lets users:
- Force compaction before approaching the threshold (clean transition between project phases).
- Recover when the threshold is mis-tuned for their model.
- Test compaction behavior without faking token counts.

**Implementation:** ~10 LOC.
- Register palette command `act-agent:compact` in `tui.go` palette block.
- Add chat-input intercept `:compact` and `/compact` in `chat.go::sendMessage`'s palette/slash map.
- Both paths emit `util.CmdHandler(startCompactSessionMsg{})`.

**Related kanban (user-added):** `user-command-needed-to-set-compaction-trigger-2026-05-11.md` covers setting the threshold value; this one covers the manual fire.

**Files:** `act-agent/internal/tui/tui.go`, `act-agent/internal/tui/page/chat.go`.

**Success criteria:**
- `/compact` typed in chat input fires compaction without LLM round-trip.
- Palette ctrl+k shows `act-agent:compact` entry.
- Status bar reports "Session summarization complete" or the error message.
