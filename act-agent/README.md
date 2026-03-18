# ACT Agent

Terminal-based AI agent for the Agent Coordination Toolkit (ACT). Fork of [OpenCode](https://github.com/opencode-ai/opencode) (MIT, Kujtim Hoxha 2025).

## What It Does

ACT Agent is the Go binary that swarm agents and NesTTY roles use to execute tasks. It supports 12 LLM providers with per-role model selection and integrates with the ACT coordination server.

## Modes

```bash
# Interactive TUI (default)
act-agent

# Headless ACT agent — executes task, returns JSON
act-agent --agent <id> --role developer --prompt "implement feature X"

# NesTTY mode — persistent conversation relay for Tier 1 roles
act-agent --nestty planner --prompt "bootstrap context"

# Non-interactive — one-shot prompt with text/JSON output
act-agent --prompt "explain this code" --output-format json
```

## Roles

**Tier 1 (NesTTY):** `planner`, `observer`, `assurance`, `qa`
**Tier 2 (Swarm):** `developer`, `frontend_dev`, `backend_dev`, `qa_engineer`, `researcher`

Each role can use a different LLM model via `.act.json`:

```json
{
  "agents": {
    "planner":   { "model": "claude-opus-4-20250514", "maxTokens": 8000 },
    "developer": { "model": "llama-3.3-70b-versatile", "maxTokens": 5000 }
  }
}
```

## Providers

Anthropic, OpenAI, Gemini, Groq (free tier), OpenRouter (free models), AWS Bedrock, Azure, VertexAI, xAI, GitHub Copilot, Local (Ollama/LM Studio).

## Build

```bash
go build -o act-agent .
```

## ACT Integration

When running in `--agent` or `--nestty` mode, act-agent automatically:
- Registers with the ACT coordination server
- Fetches context (brief, task, parallel agents, messages)
- Reports completion/failure back to the server

Set `ACT_SERVER_URL` (default: `http://localhost:8080`) and `ACT_PROJECT` for coordination.

## Config

Config file: `~/.act.json` (falls back to `~/.opencode.json`).

See `.act.json` in this directory for the schema reference.

## License

MIT License — see [LICENSE](LICENSE) for the original OpenCode license.
See [NOTICE](NOTICE) for fork attribution.
