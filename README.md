# ACT — Agent Coordination Toolkit
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/paradiselabs-ai/ACT)

**Four specialized AI roles run a swarm of coding agents on your project, with hard validation between work and delivery. One terminal. One chat view.**

You launch `act`, tell it what you want built, and four roles take it from there:

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

```bash
git clone https://github.com/paradiselabs-ai/ACT
cd ACT
./install.sh

# Configure roles + API keys in ~/.act.json
# Each Tier 1 role and each swarm role can use a different model.
# Schema: act-agent/act-schema.json

act                      # launches the TUI in the current directory
act --project my-app     # for a specific project
```

The first time you run `act` for a new project, the Planner enters **INTAKE mode** — a five-question conversation (description, tech stack, constraints, success criteria, agents involved). It summarizes, you confirm, and only then does the swarm spin up and the first task batch get created. No yes/no wizard, no scanning empty directories looking for AGENT.md files that don't exist yet.

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

Tier 1 roles (Planner, Observer, Assurance, QA) run as goroutines inside the same `act` process, sharing one chat session. Tier 2 swarm workers run as separate Node subprocesses, polling the server for tasks that match their role and capabilities.

---

## Project Layout

```
ACT/
├── act-agent/                # Go binary (the `act` command, OpenCode fork)
│   ├── cmd/                  # cobra commands + TUI launcher
│   ├── internal/
│   │   ├── app/              # orchestrator goroutines (Tier 1 roles)
│   │   ├── llm/prompt/       # role-specific prompts
│   │   ├── llm/tools/        # bash, view, grep, edit, etc.
│   │   ├── runner/           # swarm runner spawner
│   │   └── tui/              # bubble-tea TUI
│   ├── cli/                  # `act <subcommand>` (21 commands, TS)
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

MIT — see `LICENSE`.

Built by [ParadiseLabs](https://github.com/paradiselabs-ai).
