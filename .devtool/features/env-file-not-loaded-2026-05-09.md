---
id: "env-file-not-loaded-2026-05-09"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-05-09T09:30:00.000Z"
modified: "2026-05-09T09:30:00.000Z"
completedAt: null
labels: ["installer", "config", "ux", "footgun"]
order: "a0"
---
# `.env` template created by install.sh is never actually loaded

## Update 2026-05-09T09:45 (corrected 09:55)

Commit `762fa51` made the Local provider's RUNTIME client read `providers.local.baseURL` from `~/.act.json` first, falling back to `LOCAL_ENDPOINT` env, then `http://localhost:1234/v1` default.

**But** that's only half the story. `act-agent/internal/llm/models/local.go::init()` runs at package load time (before viper reads `~/.act.json`) and uses ONLY the env var to discover loaded LM Studio models via `/v1/models`. Without the env var set BEFORE binary launch, no local models get registered → `validateAgent` won't find them → user gets "unsupported model, reverting to default."

**Effective state for LM Studio quickstart:** still requires `export LOCAL_ENDPOINT=http://localhost:1234/v1` in shell rc. The viper config path I added covers ~/.act.json users for the openai-client baseURL only.

## Path to actually-zero-config

Two real options:

**A. Always probe localhost:1234 at startup if env unset.** Cost: ~1s startup delay (or hang risk) for every user, even those not using LM Studio. Probably unacceptable.

**B. Defer model discovery until after viper loads.** Move `local.go::init()` into a function that gets called from `app.New()` or similar, after viper has merged config. Then it can read `providers.local.baseURL` properly. More invasive but architecturally correct.

Recommendation: B, post-alpha. Until then: shell-rc env var is the documented path.

Remaining footgun (broader scope): API keys for cloud providers (`ANTHROPIC_API_KEY`, `GROQ_API_KEY`, `OPENROUTER_API_KEY`) still require shell rc export OR being set in `providers.<name>.apiKey` in `~/.act.json`. The `.env` template at repo root still does nothing.

## Symptom (original)

Users following onboarding docs put `LOCAL_ENDPOINT=http://localhost:1234/v1` (or `ANTHROPIC_API_KEY=...`, etc.) in `<repo-root>/.env`. ACT silently ignores the file. Nothing works. User confused.

## Root cause

Neither the Go binary (`act-agent/cmd/root.go`) nor the TS server (`server/src/index.ts`) auto-loads `.env` files. Both read `process.env` / `os.Getenv` directly. No `godotenv` import in Go. No explicit `dotenv.config()` in TS server. `tsx watch src/index.ts` doesn't use the Node `--env-file` flag either.

Meanwhile, `install.sh` lines 169-178 cheerfully copy `.env.example` to `.env` and tell the user "most users won't need to edit it" — implying it's loaded somewhere. It's not.

## Fix options

**Option A — Add env loading to Go binary.**

Import `github.com/joho/godotenv` in `act-agent/cmd/root.go` and call `godotenv.Load()` (or `godotenv.Overload(".env")` to also override system env) early in main, before any config parsing. Same for server: `dotenv.config()` at the top of `server/src/index.ts`.

Pros: matches user expectation; the templated file actually means something.
Cons: adds a third-party dep; behavior diverges across launch contexts.

**Option B — Remove `.env` from install.sh.**

Keep `.env.example` for documentation only. Update `install.sh` to NOT auto-copy it. README and onboarding docs explicitly say: "set environment variables in your shell rc (~/.zshrc / ~/.bashrc) — ACT does not load `.env` files."

Pros: simpler, no new dep, behavior is correct everywhere.
Cons: less convenient for users who expect `.env` to "just work."

**Option C — Both.**

Load `.env` in Go AND server, AND keep the install.sh template. Most user-friendly.

## Recommendation

Option C — match user expectation, low complexity. `joho/godotenv` is widely-used, MIT-licensed, ~1K SLOC, no transitive deps. Server-side `dotenv` is already in the Node standard practice.

## Constraint

This is a small change to `cmd/root.go` (Go) + `server/src/index.ts` (TS) — both project-owner domain. Not Kareem's. Don't bundle into TUI or other PRs.

## Success criteria

1. Setting `LOCAL_ENDPOINT=http://localhost:1234/v1` in `<repo-root>/.env` actually configures ACT to route to LM Studio.
2. Setting `LOCAL_ENDPOINT` in shell rc still works (env vars not in .env aren't overridden).
3. install.sh's `.env` template no longer lies about its purpose.
4. Build + vet clean. No regressions in existing env-var paths (`ACT_SERVER_URL`, `ACT_PROJECT`, etc.).

## Priority

**MEDIUM.** Not an alpha blocker — workaround (shell rc export) works fine. But a real footgun that will hit every new user who tries the LM Studio quickstart in the README.
