# ACT Agent Runner

Autonomous agent loop that wraps the `claude` CLI in an ACT coordination cycle.

## What it does

1. Registers with the ACT server
2. Polls for assigned tasks → executes them via `claude --print`
3. Checks inbox for direct messages from other agents → replies automatically
4. Loops until max iterations reached or SIGINT/SIGTERM

## Requirements

- Node.js 18+
- `claude` CLI installed and authenticated (`npm install -g @anthropic-ai/claude-code`)
- ACT server running (`cd server && npm run dev`)

## Usage

```bash
node act-runner.mjs --agent-id myagent --name "My Agent"
node act-runner.mjs --agent-id myagent --name "My Agent" --capabilities typescript,testing
node act-runner.mjs --agent-id myagent --name "My Agent" --max-iterations 50
```

## Options

| Flag | Description | Default |
|------|-------------|---------|
| `--agent-id` | Unique agent ID (required) | — |
| `--name` | Human-readable name | same as agent-id |
| `--capabilities` | Comma-separated capability list | (none) |
| `--max-iterations` | Max loops before exit (0 = unlimited) | 100 |
| `--poll-interval` | Milliseconds between polls | 5000 |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ACT_SERVER_URL` | ACT server base URL | `http://localhost:8080` |
| `CLAUDE_PATH` | Path to `claude` binary | `claude` |
| `POLL_INTERVAL_MS` | Polling interval in ms | `5000` |
| `MAX_ITERATIONS` | Max loops | `100` |
| `TASK_TIMEOUT_MS` | Timeout for each claude invocation | `120000` |

## Architecture

```
act-runner.mjs
  ↕ REST  ──→  GET  /api/tasks/assigned?agent_id=X
              POST /api/tasks/:id/progress
              POST /api/tasks/:id/complete
              GET  /api/agents/:id/messages?since=...
              POST /api/messages
  ↕ exec  ──→  claude --print "<task prompt>"
```

Tasks assigned by the REPL via `create project` reach agents through this runner.
The runner also handles direct `@mention` messages from other agents.

## Safety

- `--max-iterations` caps API usage (default 100 loops)
- `TASK_TIMEOUT_MS` kills runaway claude invocations (default 2 min)
- Only responds to `direct_mention` messages to avoid broadcast storms
- Result strings capped at 2000 chars before being sent to ACT server
- SIGINT / SIGTERM trigger graceful shutdown
