---
id: "launcher-info-leak-in-palette-output-2026-05-09"
status: "done"
priority: "low"
assignee: "d34d"
dueDate: null
created: "2026-05-09T08:30:00.000Z"
modified: "2026-05-09T08:35:00.000Z"
completedAt: "2026-05-09T08:35:00.000Z"
labels: ["cli", "cosmetic", "polish"]
order: "a0"
---
# Launcher INFO log leaks into palette command output

## Symptom

Palette commands like `act-agent:status` show:

```
2026/05/09 03:15:06 INFO ACT server already running source=...launcher.go:40 url=http://localhost:8080
ACT Project: authin — 2026-05-09T08:15:08.484Z
...
```

The first line is server-launcher noise that shouldn't be in user-facing output.

## Root cause

`internal/server/launcher.go:40` used `logging.Info(...)` for the "server already running" no-op happy path. RunDirectCommand uses `cmd.CombinedOutput()` which captures stdout + stderr, so slog Info output appears in the captured palette message.

## Resolution (2026-05-09)

Changed `logging.Info` → `logging.Debug` in launcher.go for the already-running path. Debug-level goes to file (with `ACT_DEV_DEBUG=true`) but never to stderr. Net effect: silent in palette output, still findable when debugging.

## Verification

- `act-agent:status` palette output should now skip the launcher line entirely.
- `act-agent status` from terminal should also be cleaner.
- Server auto-start path (server NOT running) still logs at Info level (intended — that's a real action).

Build clean.
