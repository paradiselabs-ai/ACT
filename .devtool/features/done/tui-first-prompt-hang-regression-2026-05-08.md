---
id: "tui-first-prompt-hang-regression-2026-05-08"
status: "done"
priority: "high"
assignee: "d34d"
dueDate: null
created: "2026-05-08T20:30:00.000Z"
modified: "2026-05-08T21:00:00.000Z"
completedAt: "2026-05-08T21:00:00.000Z"
labels: ["TUI", "bug", "regression", "alpha-blocker"]
order: "a0"
---

## Resolution (2026-05-08)

Fixed in commit `7ec9ced`. Hypothesis #1 (harmonica FrameMsg loop stops post-splash) confirmed correct. Original 50/150/400ms ticks covered the cold-start window but ran too short for the post-splash gap.

**Fix:** extended `firstPromptFlushMsg` tick schedule in `act-agent/internal/tui/page/chat.go` from 3 ticks to 8: 50ms / 150ms / 400ms / 1s / 2s / 3.5s / 5s / 8s. Each no-op tick triggers a bubbletea Update→render cycle, flushing the synchronized-output buffer. User-verified working on fresh `.act/` directory.

No structural changes, no new state, no interference with harmonica or splash. ~22 LOC change (mostly comment).


# First-prompt hang regression after harmonica/splash fade work

## Symptom

After commit `39acdc1` (harmonica animations + splash fade-in + UI polish), the first-prompt hang has reappeared. In a fresh directory (no `.act/`), the user types a prompt, presses Enter, and the TUI doesn't render the response until they press Enter (or any key) a second time.

## Reproduction

```bash
cd /tmp/fresh-test-dir-$(uuidgen | head -c 6)
act-agent
# Wait for splash + Planner intake question
# Type any reply, press Enter
# → No visible response. Buffer hung.
# Press Enter again → response renders.
```

## Workarounds (for users hitting this before fix lands)

1. `mkdir -p .act && act-agent` — pre-creates `.act/` so it exists at startup
2. `ACT_DEV_DEBUG=true act-agent` — debug mode opens log files early
3. Press Enter twice on first prompt

## Background — what was previously fixed

Original hang was the bubbletea synchronized-output buffer not flushing on cold start. Fixed in chat.go via `firstPromptFlushMsg` — three staggered `tea.Tick`s at 50/150/400ms after `sendMessage` dispatch no-op messages to force Update→render cycles. The act of dispatch flushes the buffer.

Previous fix code is still present at `act-agent/internal/tui/page/chat.go:236-248`:

```go
if !p.firstSendDone {
    p.firstSendDone = true
    flush := func(time.Time) tea.Msg { return firstPromptFlushMsg{} }
    cmds = append(cmds,
        tea.Tick(50*time.Millisecond, flush),
        tea.Tick(150*time.Millisecond, flush),
        tea.Tick(400*time.Millisecond, flush),
    )
}
```

## Hypotheses

Two suspects from `39acdc1`:

1. **Harmonica FrameMsg 60Hz tick loop** (`internal/tui/anim/spring.go:23`). Continuous `tea.Tick(time.Second/FPS, ...)` in `tui.go:606` and `list.go:124` chains FrameMsg dispatches every ~16.67ms. May be saturating bubbletea's tick channel such that `firstPromptFlushMsg` ticks queue but don't trigger a render flush. Or the constant FrameMsg traffic is changing renderer behavior in a way that decouples the no-op-msg trick from the actual buffer flush.

2. **Splash fade-in spring**. The splash fade-in (visible at `internal/tui/components/chat/list.go` or wherever the splash is mounted) takes ~500ms-1s to complete. If the chat page's Init / first-prompt code path doesn't run until splash fade completes, the firstSendDone flag flip happens AFTER the user has already typed → no flush ticks fire on the actual first message.

## Investigation steps

1. Add a log line inside the `if !p.firstSendDone` block — confirm whether it actually fires when the user sends the first message in a fresh dir.
2. Bisect: temporarily set `splashFadeInDuration = 0` and re-test. If hang gone, splash fade is the culprit. If still hung, harmonica loop is.
3. Check whether `firstPromptFlushMsg` reaches the Update handler at all — add temporary log line in tui.go's main Update dispatch. If it's being received but render isn't flushing, the no-op-message trick is broken by the new render pipeline.
4. Try increasing the flush ticks to 800ms / 1500ms / 3000ms — covers slower init under harmonica load.

## Constraints

- TUI domain, Kareem owns
- Stay inside `act-agent/internal/tui/` — orchestrator + server untouched
- Don't remove harmonica or huh — keep the polish
- The fix should not require ACT_DEV_DEBUG=true to work
- Should work on first run in a brand-new directory

## Success criteria

1. `act-agent` in a brand-new directory + first user message renders response without requiring a second keypress.
2. Verified across at least 5 fresh directories (different basenames).
3. Both Ghostty and iTerm2 (Kareem's setup + d34d's setup if they differ).
4. No regression in splash fade or scroll-focus behavior.

## Priority

**HIGH — alpha blocker.** First impression bug. Cannot ship alpha with this hanging. Was previously documented as resolved in HANDOFF.md v5; v6 must mark it open again.
