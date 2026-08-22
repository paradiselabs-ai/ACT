# P4-Hang: Test binaries block at package-init — root cause CONFIRMED

## Symptom
Any `go test` binary in a package that transitively imports
`internal/tui/theme` (or `internal/tui/styles`, or anything importing
`charm.land/lipgloss/v2/compat`) hangs forever before running any test.
`go run` of a main that blank-imports `compat` reproduces it standalone.

## Root cause (proven by probe matrix, 2026-08-22)

`charm.land/lipgloss/v2/compat/color.go` (upstream, lines 11–15):

```go
var (
    // HasDarkBackground is true if the terminal has a dark background.
    HasDarkBackground = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
    Profile           = colorprofile.Detect(os.Stdout, os.Environ())
)
```

This is a **package-level var initializer** that runs an OSC 11 background-color
query against the console at package-init time. On this host (agent-spawned
console without an interactive answerer) the query writes `\x1b]11;?\x1b\\` to
CONOUT$ and blocks reading CONIN$ forever:

- `term.MakeRaw(os.Stdin.Fd())` on redirected stdin → "handle is invalid" (fails fast, OK)
- Explicit `CONIN$` open + MakeRaw → succeeds
- OSC 11 query write to CONOUT$ + read from CONIN$ with raw mode → **never returns** (probe19: no DA1/OSC response within 3s; real lipgloss timeout is supposed to be `defaultQueryTimeout = 2s` via `uv.NewCancelReader` + `time.After`, but the cancel-reader read on the redirected console never unblocks even for the canceller — the watchdog goroutine fires `rd.Cancel()` yet the blocked Read stays stuck)

Because Go runs package-level var initializers before any `init()` / test code,
there is no user-code escape hatch. Importing `compat` alone hangs; removing all
ACT-side `init()` functions does not help.

## Probes executed (all under act-agent module, `go run`/compiled exe)

| Probe | Imports | Result |
|---|---|---|
| logging / config / models / pubsub alone | internal pkgs | OK |
| chroma styles alone | upstream | OK |
| catppuccin/go alone | upstream | OK |
| lipgloss v2 alone | upstream | OK |
| colorprofile / x-ansi / x-term / ultraviolet / fsnotify | upstream | OK |
| glamour v2 alone | upstream | OK |
| **compat alone** | upstream | **HANGS at init** |
| theme (pre-fix, eager inits) | compat via 10 files | HANGS |
| theme (post-lazy fix) — still imports compat transitively | compat | STILL HANGS |
| HasDarkBackground called manually with 6s watchdog inside goroutine | compat+lipgloss | HUNG confirmed |
| TERM=dumb set before call | — | Still hangs (no env escape hatch) |

## Why ACT's lazy-registration change was still correct

It removes nine redundant eager registrations and makes theme construction
explicit (`RegisterBuiltins`), but it cannot fix the hang because `compat` is a
transitive import of every theme file regardless of registration timing.

## Fix options

1. **Test environment (immediate):** run tests with a real interactive console
   attached, or on WSL/Linux where the Unix path uses a 2s timeout that works,
   or with `winpty`. Unverified here — needs owner validation.
2. **Upstream:** report to charmbracelet/lipgloss — `compat` should not perform
   blocking terminal I/O in a var initializer; it should use `sync.Once` like
   ACT's own `styles.IsDarkBackground()` wrapper already does (see
   `internal/tui/styles/markdown.go:306-343` which documents this exact hazard
   for Glamour rendering).
3. **In-repo guard (possible but invasive):** vendor-patch or replace
   `compat.AdaptiveColor` usage across the 10 theme files + styles with a local
   type whose RGBA() consults a cached, lazily-resolved dark flag
   (`styles.IsDarkBackground()` already exists). Blast radius: ~10 files, all
   color tables; mechanical but wide.

## Recommendation

File the upstream issue (option 2) and adopt option 1 for CI. Option 3 only if
upstream refuses or CI on Windows agents is a hard requirement.
