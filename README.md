# ACT — Agent Coordination Toolkit
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/paradiselabs-ai/ACT)

**Four specialized AI roles run a swarm of coding agents on your project, with hard validation between work and delivery. One terminal. One chat view.**

You launch `act-agent`, tell it what you want built, and four roles take it from there:

- **Planner** — the only role that talks to you. Decomposes the goal into tasks with explicit `@success_criteria`.
- **Observer** — silent background watcher. Flags stuck work, file conflicts, idle agents, bottlenecks.
- **Assurance** — independent validator. Scores every completed task against the criteria. Below 95% goes back. Above 95% advances.
- **QA / Synthesizer** — assembles validated work into the final deliverable. Only sees output that already passed.

Underneath, a swarm of headless workers — `developer`, `frontend_dev`, `backend_dev`, `qa_engineer`, `researcher` — picks up tasks in parallel and executes them in the background. Each swarm role can run on a different model. Each can use the bundled `act-agent` (an OpenCode fork) or `claude-code` as its execution backend.

You see all of it in one chat view, color-coded by role.

> **Status:** active development on the `NesTTY` branch. Public release, docs site, and screencast are next.

---

## Why this exists

Most coding agents are a single model in a long loop. They produce a wall of output, nothing checks the output until you do, and the loop has no memory of why it took the path it took.

ACT splits the work along the lines that actually matter:

- **Decisions** live with the Planner — and only the Planner. Execution agents don't make scope calls.
- **Execution** runs in parallel across the swarm. Five workers, not one bottleneck.
- **Verification** is independent. Assurance scores work it didn't write, against criteria that were defined before the work started.
- **Assembly** only sees what already passed. No "hope it works" wall of generated code reaching you.
- **Memory** is an append-only chronological log of every coordination event, indexed for semantic search (PVM — PAIRed Vector Minutes). Future routing decisions can cite past evidence: which agent actually succeeded at this kind of task last time, not just which one was free.

The separation is the whole point. It is the difference between *"here's a thousand lines, hope it works"* and *"here's the validated deliverable, and here's the audit trail of how each piece got verified."*

---

## Quick Start

**Prerequisites:** Node.js v18+, Go v1.21+, npm. (Optional: Claude Code CLI for subscription-backed swarm.)

### macOS / Linux / WSL

```bash
git clone https://github.com/paradiselabs-ai/ACT
cd ACT
./install.sh
```

### Windows (PowerShell 5.1+)

```powershell
git clone https://github.com/paradiselabs-ai/ACT
cd ACT
.\install.ps1
```

If PowerShell rejects the script: `Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned`, then re-run.

### After install

The installer copies `.act.example.json` to `~/.act.json` and `.env.example` to `.env` if they don't exist yet. Edit `~/.act.json` to add your API keys — that file is the canonical place for per-role model + provider config.

```bash
act-agent                      # launches the TUI in the current directory
act-agent --project my-app     # for a specific project
```

> **Why `act-agent` and not `act`?** [`nektos/act`](https://github.com/nektos/act) is the popular GitHub Actions local runner — it owns the `act` command name in dev environments. To avoid collision, ACT's CLI is `act-agent`. If you previously had an `act` symlink from an older install, the installer removes it.

### Run on cloud or local — equal first-class paths

ACT supports two onramps for Tier 1 and the swarm. Pick whichever matches your hardware and constraints:

#### ☁ Cloud (zero hardware requirements)

Free-tier cloud models cover both Tier 1 and the swarm with no credit card.

```json
"providers": {
  "groq":       { "apiKey": "<your-groq-key>",       "disabled": false },
  "openrouter": { "apiKey": "<your-openrouter-key>", "disabled": false }
}
```

- **[Groq](https://console.groq.com/)** — free tier, sub-second latency, Llama 3.3 70B. 30-second signup, no credit card.
- **[OpenRouter](https://openrouter.ai/)** — aggregator, many free models (GLM-4.5, gpt-oss-120b, MiniMax-M2.5). 2-minute signup.
- **[Anthropic](https://console.anthropic.com/) / [OpenAI](https://platform.openai.com/) / [xAI](https://x.ai/)** — paid API keys for premium models.

#### 💻 Local (zero cloud calls, your hardware)

All 9 agents (4 Tier 1 + 5 swarm) run on local models via [LM Studio](https://lmstudio.ai/) (free for personal AND commercial use as of July 2025) or [Ollama](https://ollama.com/). Your code never leaves your machine.

Recommended stack on RTX 5090 (32GB) or M3/M4 Max (≥64GB):

| Role | Model | VRAM (Q4_K_M) |
|------|-------|---------------|
| Planner | Qwen3-30B-A3B (MoE, 3B active) | ~17 GB |
| Assurance + QA + 5 swarm roles | Qwen2.5-14B | ~10 GB (shared) |
| Observer | Qwen3-8B | ~5 GB |

Lighter setup on RTX 4090 (24GB) or 32GB Apple Silicon: drop Planner to Qwen2.5-14B, share 8B across the rest.

Quick path:

1. Install LM Studio, download the models above
2. Pre-load each with `lms load <model> --ttl 0`, disable **Auto-Evict** in Developer → Server Settings
3. Enable LM Studio's OpenAI server (port 1234)
4. In `~/.act.json`, set per-role models with `"provider": "local"` and the loaded model ID
5. Set `LOCAL_ENDPOINT=http://localhost:1234/v1` in `.env`

Full local config example in `.act.example.json`.

**Supported terminals:** [Ghostty](https://ghostty.org/) (recommended), [iTerm2](https://iterm2.com/), [Alacritty](https://alacritty.org/), [kitty](https://sw.kovidgoyal.net/kitty/). Apple Terminal.app is **not supported** — it lacks Synchronized Output (mode 2026) and the TUI render pipeline hangs after each prompt. See `docs/Vault/Agent Coordination Toolkit/nestty/KNOWN_ISSUES.md` (KI-14).

The first time you run `act-agent` for a new project, the Planner enters **INTAKE mode** — a five-question conversation (description, tech stack, constraints, success criteria, agents involved). It summarizes, you confirm, and only then does the swarm spin up and the first task batch get created. No yes/no wizard, no scanning empty directories looking for AGENT.md files that don't exist yet.

---

## Architecture

```
You ──► Planner ──► ACT server ──► Runner spawns swarm ──► swarm executes
            ▲                                                    │
            │                                                    ▼
            └──── Assurance validates ◄──── submit-for-validation
                          │
                          ▼
                  QA assembles ──► back to Planner ──► you
```

Three layers, strictly separated:

1. **LLM decisions** — Planner (and only the Planner).
2. **Deterministic state** — ACT server. Append-only ChronologicalLog (JSONL) + LocalEmbeddingVectorStore (real `all-MiniLM-L6-v2` embeddings, no mocks). Event sourcing replays state on restart.
3. **Thin spawner** — Runner. Reacts to state changes, spawns workers, never talks to the Planner directly.

Tier 1 roles (Planner, Observer, Assurance, QA) run as goroutines inside the same `act-agent` process, sharing one chat session. Tier 2 swarm workers run as separate Node subprocesses, polling the server for tasks that match their role and capabilities.

---

## Project Layout

```
ACT/
├── act-agent/                # Go binary (the `act-agent` command, OpenCode fork)
│   ├── cmd/                  # cobra commands + TUI launcher
│   ├── internal/
│   │   ├── app/              # orchestrator goroutines (Tier 1 roles)
│   │   ├── llm/prompt/       # role-specific prompts
│   │   ├── llm/tools/        # bash, view, grep, edit, etc.
│   │   ├── runner/           # swarm runner spawner
│   │   └── tui/              # bubble-tea TUI
│   ├── cli/                  # `act-agent <subcommand>` agent CLI (TS)
│   └── runner/act-runner.mjs # Headless swarm worker spawner
├── server/                   # ACT coordination server (TypeScript, port 8080)
│   └── src/services/         # AgentRegistry, TaskCoordinator, EventHub,
│                             # PVMIndexer, LocalEmbeddingVectorStore, ChronologicalLog
├── docs/                     # Architecture and design notes
├── CLAUDE.md                 # Canonical project guide
└── act-coordination.json     # Append-only multi-agent coordination log
```

---

## What makes it different

| | Most coding agents | ACT |
|---|---|---|
| **Decision-making** | The same model that writes the code | Dedicated Planner role, separate from execution |
| **Verification** | "Run the tests, hope they pass" | Independent Assurance scoring against pre-defined criteria, 95% gate |
| **Parallelism** | One model in a loop | Swarm of workers picking up tasks concurrently, by role and capability |
| **Memory** | Chat history, maybe a vector DB | Append-only causal log + semantic index, queryable as evidence |
| **Routing** | Round-robin or random | Evidence-based: which agent has succeeded at this kind of task before |
| **Models per role** | One model for everything | Each role configurable independently — heavy model for the Planner, cheap or local for the swarm |
| **Final delivery** | Whatever the loop produced | Whatever passed Assurance |

---

## License

ACT is licensed under the **Apache License 2.0** — see `LICENSE` and `NOTICE`.

The `act-agent/` subdirectory is a fork of [OpenCode](https://github.com/opencode-ai/opencode) and retains its upstream **MIT** license; see `act-agent/LICENSE`.

Built by [ParadiseLabs](https://github.com/paradiselabs-ai).
