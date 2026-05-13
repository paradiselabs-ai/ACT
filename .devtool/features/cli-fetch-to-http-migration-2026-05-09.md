---
id: "cli-fetch-to-http-migration-2026-05-09"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-09T08:30:00.000Z"
modified: "2026-05-09T08:30:00.000Z"
completedAt: null
labels: ["cli", "bug", "macos", "node", "alpha-blocker"]
order: "a0"
---
# Migrate CLI from `fetch` to `node:http` — Node 25 macOS Keychain segfault

## Symptom

On macOS with Node 25 (current LTS at time of writing: 22), all `act-agent <subcommand>` and `act-agent:<subcommand>` palette calls produce only:

```
ERROR: SecItemCopyMatching failed -25300
```

…and exit with code 139 (SIGSEGV). The TUI itself works fine (Go HTTP client doesn't hit this); only the TS CLI subprocess is broken.

## Root cause

Node 25's built-in `fetch` (undici) loads TLS trust roots from macOS Keychain on first call, even for plain `http://localhost`. If Keychain access fails (`errSecItemNotFound -25300`), the trust-store init crashes the Node process with SIGSEGV.

Reproduction:

```bash
tsx -e "fetch('http://localhost:8080/health').then(r => r.json()).then(console.log)"
# → ERROR: SecItemCopyMatching failed -25300
# → exit 139 (SIGSEGV)
```

This is a Node 25 + macOS interaction bug, not an ACT code bug. But it makes ACT's CLI unusable for macOS users running Node 25.

## Workaround for end users (immediate)

Downgrade to Node 22 LTS:

```bash
nvm install 22
nvm use 22
nvm alias default 22
cd act-agent/cli && rm -rf node_modules && npm install
```

## Real fix

Migrate the CLI from `fetch` to `node:http`. Node's stdlib `http` module doesn't load Keychain trust roots — works on any Node version, any OS, no macOS-specific failure mode.

**Files affected:**
- `act-agent/cli/act-client.ts` — 30 `fetch(...)` call sites
- `act-agent/cli/act-cli.ts` — 7 `fetch(...)` call sites
- `act-agent/cli/act-repl.ts` — possibly some

**Approach:**

1. Write a thin `httpRequest(method, path, body?)` helper using `node:http` (or `node:https` if remote ACT server use case ever arrives).
2. Replace each `fetch(`URL`)` with `httpRequest(...)`. Same return shape (status, json).
3. All localhost URLs — no TLS, no trust store, no Keychain.
4. Test: `act-agent status` returns full output on Node 25 macOS.

## Constraints

- TS CLI domain
- No new npm dependencies — stdlib only
- Backward-compatible response shape so callers don't need rewrites beyond the helper substitution
- Apply to ALL three CLI files in one PR

## Success criteria

1. `cd cli && npm install && tsx act-cli.ts status` works on Node 25 macOS — full output, no segfault.
2. Same on Node 22, 24.
3. All existing CLI subcommands (status, register, context, task complete, pvm search, validation queue, files claim/release, message, log, graph) verified working.
4. No `fetch(` calls remaining in any `cli/*.ts` file.
5. `npm run build` clean (TypeScript compiles).

## Priority

**HIGH — alpha blocker.** Without this, every macOS Node 25 user hits the SIGSEGV. Workaround works but breaks the "just install and run" pitch.

Alternative if Migration is too large: pin a minimum Node version requirement (Node 22 LTS) and document it. Less robust but ships faster. Decision: do the migration; Node 22 is on a 2025 LTS cycle and will eventually go away.
