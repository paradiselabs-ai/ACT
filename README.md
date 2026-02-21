# ACT: Agent Coordination Toolkit

**Universal coordination infrastructure for autonomous multi-agent systems.**

ACT (Agent Coordination Toolkit) lets AI agents like Claude, Goose, Warp Terminal, as well as IDEs like Cursor, Windsurf, Antigravity, and/or any MCP-compatible Agent or Agent client — all seamlessly collaborate together on coding and development projects as a unified team. Agents register with ACT to receive tasks, communicate with each other, and track and report progress through a central coordination server. A terminal REPL lets you designate a planning agent (the ACTor), create and define new projects, and watch your agent team work even as each agent operates in their own natural environment.

This means Claude works through the Claude desktop client, your IDE agent works through the IDE, and so on, with each agent collaborating together on the same project directory, communicating with each other, and working in parallel to complete tasks.

Each agent uses the tools they have available to them, either natively or through any plugins or connectors you have configured for that agent. ACT itself does not provide any tool use, actions, or other functionality beyond coordination and communication. It is solely a coordination layer with a few other utilities that enhance and strengthen that coordination.

---

## How It Works

```
User (REPL)
    │  create project "my-app" in /path
    │  → guided prompts collect description, stack, agents
    │
    ▼
ACT Server                         ACTor Agent (Claude Desktop, etc.)
    │  Planning task assigned  ──────────►  get_task()
    │                                       ... analyzes project ...
    │  Task breakdown received  ◄──────────  report_task_complete({ tasks, briefs })
    │
    │  Creates tasks + stores AGENT.md briefs for each agent
    │
    ▼
Execution Agents (any MCP client)
    register_with_act()
    get_agent_brief()   ──► writes AGENT.md to project directory
    get_task()          ──► receives assigned task
    report_task_progress()
    send_message("@OtherAgent can you review this?")
    report_task_complete()
```

No API keys. No orchestration scripts. Just agents connecting to ACT via MCP and getting to work.

---

## Getting Started

### 1. Start the ACT server

```bash
git clone https://github.com/paradiselabs-ai/ACT
cd ACT/server
npm install
npm install -D tsx
npm run dev
# Server running on http://localhost:8080
```

### 2. Build the MCP bridge

```bash
cd ../mcp-servers/act-mcp-bridge
npm install
npm run build
```

### 3. Connect an agent via MCP

Add to your agent's MCP config (e.g. `~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "act": {
      "command": "node",
      "args": ["/path/to/ACT/mcp-servers/act-mcp-bridge/dist/index.js"],
      "env": { "ACT_SERVER_URL": "http://localhost:8080" }
    }
  }
}
```

Works with Claude Desktop, Windsurf, Cursor, Goose, or any MCP-compatible client.

### 4. Run the REPL

```bash
cd ../../cli
npm install
npx tsx act-repl.ts
```

```
>> default agent claude-desktop-agent
>> create project my-app in /Users/me/projects/my-app

  ┌──────────────────────────────────────────────────┐
  │  New Project: my-app                             │
  └──────────────────────────────────────────────────┘

  What are you building?
  > A REST API with auth and CRUD for a task manager

  Technologies / stack:
  > TypeScript, Express, PostgreSQL, JWT

  What does success look like?
  > Users can register, log in, and manage personal tasks

  Assigning planning task to claude-desktop-agent...
  ⏳ Analyzing...  (12s elapsed)

  ✅ Planning complete! Creating 5 tasks...
  ✓ Set up project structure and TypeScript config
  ✓ Implement user registration and JWT auth
  ✓ Build task CRUD endpoints
  ✓ Add input validation and error handling
  ✓ Write integration tests

  Storing AGENT.md briefs for 2 agent(s)...
  ✓ Brief stored for claude-desktop-agent
  ✓ Brief stored for windsurf-agent

  Project "my-app" is ready!
```

---

## MCP Tools Available to Agents

| Tool | Description |
|------|-------------|
| `register_with_act` | Join the ACT coordination session |
| `get_task` | Receive your assigned task |
| `report_task_progress` | Update progress % and status |
| `report_task_complete` | Mark task done, pass result to ACT |
| `send_message` | Broadcast or `@AgentName` direct message |
| `get_agent_brief` | Fetch your AGENT.md context file |
| `query_coordination_memory` | Search past coordination patterns (PVM) |
| `evaluate_coordination` | Request coordination analysis |
| `improve_coordination` | Trigger improvement analysis |
| `get_improvement_status` | Check self-improvement engine status |

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  Agent Platforms                                    │
│  (Claude Desktop, Windsurf, Cursor, Goose, etc.)   │
└──────────────────────┬──────────────────────────────┘
                       │ MCP Protocol (stdio)
                       ▼
┌─────────────────────────────────────────────────────┐
│  act-mcp-bridge  (10 tools → HTTP calls)            │
└──────────────────────┬──────────────────────────────┘
                       │ REST + Socket.io
                       ▼
┌─────────────────────────────────────────────────────┐
│  ACT Server (port 8080)                             │
│  ├─ AgentRegistry     (capability tracking)         │
│  ├─ TaskCoordinator   (assignment + lifecycle)      │
│  ├─ EventHub          (message routing + rate limit)│
│  ├─ ChronologicalLog  (JSONL event history)         │
│  └─ PVMIndexer        (semantic memory search)      │
└─────────────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  ACT REPL (cli/)                                    │
│  create project · list agents · show project · ...  │
└─────────────────────────────────────────────────────┘
```

### Message Routing

ACT classifies inter-agent messages before routing — no feedback loops:

- `@AgentName message` → delivered only to that agent's socket
- Status updates → broadcast with `messageType: 'status_update'` (agents observe silently)
- Questions / help requests → routed to best available capable agent
- Rate limiting: max 3 peer responses per agent per 30 seconds

### PAIRed Vector Minutes (PVM)

Every coordination event is logged to a chronological JSONL store and indexed for semantic search. This powers:
- Evidence-based agent profiling (what capabilities does this agent actually succeed at?)
- Pattern retrieval for future coordination decisions
- Full audit trail of why every assignment was made

### Self-Improvement *(in development)*

- **FLUX State**: unbiased post-project evaluation (agent evaluates output without knowing it created it)
- **PAIR**: retrieves similar past coordination patterns to improve current approach
- **`/improve` command**: surgical analysis of coordination quality by scope (communication, assignments, conflicts, etc.)

---

## Project Structure

```
ACT/
├── server/                  # ACT coordination server (TypeScript/Node.js)
│   └── src/
│       ├── index.ts         # Express + Socket.io + all REST endpoints
│       └── services/        # AgentRegistry, TaskCoordinator, EventHub, PVM...
│
├── mcp-servers/
│   └── act-mcp-bridge/      # MCP server exposing ACT tools to agents
│       └── src/
│           ├── index.ts     # stdio MCP server entry point
│           ├── tools/       # 10 tools with real HTTP calls
│           └── schemas/     # Zod input validation
│
├── cli/                     # Terminal REPL (primary user interface)
│   ├── act-repl.ts          # REPL + guided project wizard
│   └── act-client.ts        # HTTP client
│
├── docs/                    # Architecture & design documentation
└── examples/                # Demo scripts
```

---

## What's Working Today

- ✅ ACT server with REST + Socket.io agent coordination
- ✅ MCP bridge with 10 real tools (no stubs)
- ✅ REPL guided `create project` wizard with ACTor planning flow
- ✅ Intelligent message routing (direct mentions, rate limiting, no feedback loops)
- ✅ ChronologicalLog — append-only event history
- ✅ AGENT.md brief generation and per-agent retrieval
- ✅ PVM semantic search (hash-based similarity)

**In progress**: Real vector embeddings (Qdrant), FLUX State evaluation, PAIR retrieval, project data persistence across restarts.

---

## Why ACT?

Multi-agent systems have a coordination problem. Most frameworks handle *what* to do (task lists, orchestration) but not *why* decisions were made or how to get better over time.

ACT is the coordination layer between agents — framework-agnostic, learns from each outcome, and designed to self-improve continuously with every project.

**Most agents just act. ACT gives them agency.**

---

## License

MIT — See LICENSE file for details.

Built by [ParadiseLabs](https://github.com/paradiselabs-ai).
