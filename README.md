# ACT — Agent Coordination Toolkit

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/paradiselabs-ai/ACT)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Status: Alpha](https://img.shields.io/badge/status-alpha-orange.svg)](docs/KNOWN_LIMITATIONS.md)

**One terminal. Four AI roles coordinating a swarm of coding agents on your project — with independent validation between "an agent said it's done" and "it reaches you."**

<!-- DEMO GIF GOES HERE — 4-agent window running a real task. This is the hook; record it before public announce. -->

```
You ──► Planner ──► ACT server ──► swarm executes in parallel
            ▲                              │
            │                              ▼
            └── Assurance validates ◄── work submitted
                        │
                        ▼
                QA assembles ──► Planner ──► you
```

You run `act-agent`, describe what you want built, and four roles take it from there — all visible in one color-coded chat view:

| Role | What it does |
|------|-------------|
| **Planner** | The only role that talks to you. Breaks the goal into tasks with explicit `@success_criteria`, routes them to the right swarm role. |
| **Observer** | Silent watchdog. Flags stuck tasks, file conflicts, idle agents, bottlenecks. Speaks only when something's wrong. |
| **Assurance** | Independent validator. Scores every completed task against criteria defined *before* the work started. Below the 95% gate → back it goes. |
| **QA / Synthesizer** | Assembles validated work into the deliverable. Only ever sees output that already passed. |

Underneath, **the swarm**: headless workers (`developer`, `frontend_dev`, `backend_dev`, `qa_engineer`, `researcher`) picking up tasks concurrently, each configurable with its own model, provider, and execution backend.

> **Status: alpha.** It works; it has rough edges. Read [KNOWN_LIMITATIONS](docs/KNOWN_LIMITATIONS.md) before filing a bug — it might already be in there.

---

## Why

Most coding agents are one model in a long loop. Nothing checks the output until you do, decisions and execution are tangled in one context window, and the loop can't tell you why it took the path it took.

ACT splits the work along the lines that actually matter:

- **Decisions** live with the Planner — and only the Planner. Execution agents never make scope calls.
- **Execution** runs in parallel across the swarm — five workers, not one bottleneck.
- **Verification** is independent. Assurance scores work it didn't write, against criteria it didn't invent after the fact.
- **Assembly** only touches what passed. No thousand-line "hope it works" wall reaching you.
- **Memory** is an append-only log of every coordination event, semantically indexed (PVM). Routing can cite evidence: which role actually succeeded at this kind of task last time (recall layer fixed + live-verified 2026-08-19 — history in [the memory audit](docs/audits/memory-system-audit-2026-08-13.md)).

The separation is the product. *"Here's the validated deliverable, and here's the audit trail of how each piece got verified"* — instead of *"here's a diff, good luck."*

---

## Quick Start

**Prerequisites:** Node.js 18+, Go 1.21+, npm. No GPU required — free cloud tiers cover everything.

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

### 2. Add one API key

The installer drops a config at `~/.act.json` (from `.act.example.json`). Every role is pre-wired to **free** models — you just need one key:

- **[Groq](https://console.groq.com/)** — free tier, sub-second Llama 3.3 70B. 30-second signup, no card.
- **[OpenRouter](https://openrouter.ai/)** — aggregator with many free models (GLM-4.5, gpt-oss-120b, MiniMax). The example config defaults to these.
- **[NVIDIA NIM](https://build.nvidia.com/)** — free hosted models (Kimi K2.6, DeepSeek, Nemotron). Provider `nvidia`.
- **[Anthropic](https://console.anthropic.com/) / [OpenAI](https://platform.openai.com/) / [xAI](https://x.ai/)** — paid keys for premium models.

Paste it into the matching `providers` entry in `~/.act.json`. A provider is live as soon as its key is filled in — no enable flag.

### 3. Run

```bash
act-agent                      # launch the TUI in the current directory
act-agent --project my-app     # or name a project
```

First run on a new project, the Planner enters **INTAKE**: a five-question conversation (what, tech stack, constraints, success criteria, which agents). It summarizes, you confirm, *then* the swarm spins up. Pointed at an existing codebase, it skips the form and asks just two things: what to build next, and what the agents must not touch.

> **Why `act-agent`, not `act`?** [`nektos/act`](https://github.com/nektos/act) owns the `act` command in most dev environments. We stayed out of the way.

**Terminals:** [Ghostty](https://ghostty.org/) (recommended), iTerm2, Alacritty, kitty. **Apple Terminal.app is not supported** — it lacks Synchronized Output (mode 2026) and the render pipeline hangs.

---

## In-TUI commands

| Command | Does |
|---------|------|
| `/status` | Coordination state — agents, tasks, validation queue |
| `/swarm <role\|all> <backend>` | Set a swarm role's execution backend |
| `/backend <role\|all> <backend>` | Set a Tier 1 role's execution backend |
| `/help` | Command reference |
| `/quit`, `/exit` | Leave |

---

## Bring your own agent — ACP clients as roles

Any of ACT's nine roles can be played by an external agent CLI you already use, instead of the bundled agent. Tier 1 roles are driven over **ACP ([Agent Client Protocol](https://agentclientprotocol.com))** — ACT speaks the protocol to the CLI, injects the role's identity, and the external agent *becomes* your Planner, Observer, Assurance, or QA. Swarm roles run external CLIs as headless one-shots per task. Set it per role with `"backend"` in `~/.act.json`, or live in the TUI with `/backend` (Tier 1) and `/swarm` (Tier 2).

Every role — coordinator or worker — can run on a different **model** (via `~/.act.json`) *and* a different **execution host**:

**Tier 1 (Planner / Observer / Assurance / QA):**
- *(default)* in-process — the bundled Go agent, any configured provider/model
- `claude-code` — the official Claude Code CLI, driven over ACP
- `gemini` — Gemini CLI over ACP
- `antigravity` / `agy` — Antigravity over ACP

**The swarm (Tier 2):**
- *(default)* `act-agent` — bundled agent, per-role model from `~/.act.json`
- `claude-code` — headless Claude Code per task
- `gemini` — headless Gemini CLI per task (researcher runs read-only)
- `antigravity` — headless Antigravity (`agy`) per task — except `researcher`: agy has no read-only mode, so ACT rejects that pairing rather than run a researcher with write access ([details](docs/KNOWN_LIMITATIONS.md))

So this is a legal setup: Claude Code as your Planner, a free Groq model as Observer, local Qwen as the whole swarm. Heavy model where decisions happen, cheap ones where volume happens.

```json
"agents": {
  "planner":   { "backend": "claude-code" },
  "observer":  { "provider": "groq", "model": "llama-3.3-70b-versatile", "maxTokens": 2000 },
  "developer": { "provider": "local", "model": "qwen2.5-14b-instruct" }
}
```

---

## Fully local — zero cloud calls

All 9 agents run on local models via [LM Studio](https://lmstudio.ai/) or [Ollama](https://ollama.com/). Your code never leaves your machine.

Reference stack for RTX 5090 (32GB) or M3/M4 Max (≥64GB):

| Role | Model | VRAM (Q4_K_M) |
|------|-------|---------------|
| Planner | Qwen3-30B-A3B (MoE) | ~17 GB |
| Assurance + QA + swarm | Qwen2.5-14B (shared) | ~10 GB |
| Observer | Qwen3-8B | ~5 GB |

Lighter (RTX 4090 / 32GB Apple Silicon): Planner on Qwen2.5-14B, share an 8B across the rest.

1. Install LM Studio, download the models
2. `lms load <model> --ttl 0`, disable **Auto-Evict** (Developer → Server Settings)
3. Enable the OpenAI-compatible server (port 1234)
4. Set `"provider": "local"` per role in `~/.act.json`
5. `LOCAL_ENDPOINT=http://localhost:1234/v1` in `.env`

Full local example in `.act.example.json`.

---

## How it's built

Three layers, strictly separated — the Planner never talks to the Runner; both talk to the server:

1. **LLM decisions** — the Planner, and only the Planner.
2. **Deterministic state** — the ACT server (TypeScript, port 8080). Append-only chronological event log (JSONL) + real `all-MiniLM-L6-v2` embeddings for semantic search. Event sourcing replays full state on restart.
3. **Thin spawner** — the Runner. Reacts to state, spawns workers, makes zero decisions.

Tier 1 roles run as goroutines inside the one `act-agent` process, sharing a single chat session — that shared window *is* the multi-agent UI (we call it NesTTY). Swarm workers are separate subprocesses polling the server for tasks matching their role and capabilities.

Swarm agents talk to the server through a tiny CLI (`act-agent task complete`, `act-agent pvm search`, `act-agent context`, two dozen subcommands) instead of MCP tool schemas — ~50–100 tokens of overhead per call instead of ~47K.

```
ACT/
├── act-agent/                # Go binary — TUI + Tier 1 roles + agent tools (OpenCode fork)
│   ├── internal/app/         # orchestrator: task parsing, validation routing, Observer loop
│   ├── internal/llm/prompt/  # role-specific system prompts
│   ├── internal/acp/         # external-backend layer (claude-code / gemini / antigravity)
│   ├── cli/                  # the agent-facing CLI (TypeScript)
│   └── runner/               # headless swarm worker spawner
├── server/                   # coordination server: registry, tasks, events, PVM index
├── docs/                     # architecture, audits, known limitations
└── act-coordination.json     # append-only multi-agent coordination log
```

---

## vs. the usual

| | Most coding agents | ACT |
|---|---|---|
| **Decisions** | Same model that writes the code | Dedicated Planner, separate from execution |
| **Verification** | "Run the tests, hope" | Independent scoring vs pre-defined criteria, 95% gate |
| **Parallelism** | One model, one loop | Swarm working concurrently by role + capability |
| **Memory** | Chat history | Append-only event log + semantic index, queryable as evidence |
| **Models** | One for everything | Per-role model, provider, *and* execution host |
| **Delivery** | Whatever the loop produced | Whatever passed Assurance |

---

## Contributing

```bash
./scripts/install-hooks.sh      # post-commit doc-drift tracker
./scripts/freshness-check.sh    # which generated docs are currently stale
```

Ground rules — what to trust, where docs live, how tasks are specced — are in [`docs/constitution/`](docs/constitution/CONSTITUTION.md). AI editors: start at `AGENTS.md` (or `CLAUDE.md` / `GEMINI.md`).

Bugs → issues, with the evidence paths (`~/.act/**/debug.log`, `~/.act/runners/<role>.log`, `server/data/coordination-log.jsonl`). Check [KNOWN_LIMITATIONS](docs/KNOWN_LIMITATIONS.md) first.

---

## License

Apache 2.0 — see `LICENSE` and `NOTICE`. The `act-agent/` subdirectory is a fork of [OpenCode](https://github.com/opencode-ai/opencode) and retains its upstream MIT license (`act-agent/LICENSE`).

Built by [ParadiseLabs](https://github.com/paradiselabs-ai).
