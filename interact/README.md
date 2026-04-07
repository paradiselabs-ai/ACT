# interact v0

Dumb PTY harness for `act`. Launches the TUI in a pseudo-terminal, types one prompt, captures everything, prints a report. No LLM, no decisions.

## Build

```bash
cd interact
go build -o interact .
```

## Run

```bash
# Terminal 1: ACT server
cd ../server && npm run dev

# Terminal 2: drive act
./interact -project snake-test -prompt-file prompts/snake-arena.txt -timeout 120s
```

## Flags

- `-project <name>` — project name (also `/tmp/interact-runs/<name>` run dir)
- `-prompt <str>` or `-prompt-file <path>` — what to type
- `-timeout <dur>` — capture window after the prompt (default 90s)
- `-act <path>` — override `act` binary (default: `act` from PATH)
- `-server-log <path>` — coordination log to tail (default: `../server/data/coordination-log.jsonl`)

## Output

- `/tmp/interact-runs/<project>/run.log` — raw PTY bytes (ANSI included)
- `/tmp/interact-runs/<project>/run.txt` — ANSI-stripped, grep-friendly
- One-screen report on stdout: exit code, CREATE_TASK count, role tags, panics, log tail

Exits 0 if no panics AND ≥1 CREATE_TASK seen, else 1.

## What it does NOT do

No LLM in the loop. No multi-turn driving. No screen reconstruction. v0 is "launch, type, wait, dump, grade." interACT v1 (LLM-driven computer-use agent) is later.
