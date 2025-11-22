# ACT: Agent Coordination Toolkit

**ACT is a framework-agnostic coordination layer that enables autonomous AI agent teams to self-organize and continuously improve without human intervention. It learns from coordination patterns through semantic memory (PAIRed Vector Minutes).**

ACT is a universal coordination layer that enables autonomous teams of AI agents to collaborate efficiently. Unlike task orchestration frameworks, ACT focuses on *why* coordination decisions are made and learns from coordination patterns over time.

## Core Innovation: Semantic Coordination Intelligence

Most multi-agent systems coordinate tasks based on current capabilities. ACT goes further:

- **Understands coordination patterns** - Why was Agent X assigned to Task Y?
- **Learns from history** - What similar patterns succeeded before?
- **Improves continuously** - Post-session evaluation identifies improvement opportunities
- **Stays unbiased** - FLUX State reasoning wipes memory for objective self-evaluation

This is the first semantic memory system for coordination itself.

## Key Features

### Framework-Agnostic
- Works with Claude Code, Cursor, Windsurf, Goose (via MCP)
- Integrates with API agents (OpenAI, Anthropic, Google)
- Supports local models (Ollama, LM Studio)
- Simple protocol for custom agents

### Real-Time Coordination
- Task assignment based on capability matching
- Automatic conflict detection and resolution
- Progress tracking and status updates
- Inter-agent communication

### PAIRed Vector Minutes (PVM)
- **Chronological log** - Complete audit trail of every coordination decision
- **Vector indexing** - Semantic search without scanning entire history
- **PAIR reasoning** - Past patterns inform future decisions

### Self-Improvement Through FLUX State
- Memory-wiped evaluation (unbiased assessment)
- Gap identification (what didn't work?)
- Pattern retrieval (why did similar approaches succeed?)
- Iterative refinement (loop until 95%+ success)

## Getting Started

### For MCP-Enabled Agents (Fastest)

Add to your IDE configuration:

```json
{
  "mcpServers": {
    "act": {
      "command": "npx",
      "args": ["@agentmix/act-mcp-server"],
      "env": {
        "ACT_SERVER": "ws://localhost:8080",
        "ACT_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Local Development

```bash
git clone https://github.com/paradiselabs-ai/ACT
cd ACT
npm install
npm run dev
# ACT server runs on ws://localhost:8080
```

### Docker Deployment

```bash
docker-compose up
# ACT server on port 8080
# Qdrant vector DB on port 6333
# PostgreSQL on port 5432
```

## Architecture

```
Agent Platforms (Claude Code, Cursor, Windsurf, Goose, API agents)
         │
      MCP Protocol
         │
    ACT Coordination Server
    ├─ Task Coordinator
    ├─ Agent Registry
    ├─ Conflict Resolver
    └─ Event Hub
         │
    ┌────┴────┐
    │          │
  Chronological   Vector Memory
    Log           Store
  (JSONL/SQL)     (Qdrant)
    │          │
    └────┬────┘
         │
   PAIR Reasoning Loop
   FLUX State Evaluation
```

## Project Structure

```
ACT/
├── docs/                    # Architecture & concepts
├── server/                  # Node.js/TypeScript server
│   ├── src/
│   │   ├── services/       # Core coordination services
│   │   ├── widgets/        # Visualization components
│   │   └── utils/
│   └── package.json
├── client/                 # Web dashboard (SSE-based)
├── sdk/                    # Future: Language-specific SDKs
├── examples/               # Integration examples
└── tests/
```

## Documentation

- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System design and components
- **[PHASE_5_ARCHITECTURAL_REDESIGN.md](docs/PHASE_5_ARCHITECTURAL_REDESIGN.md)** - Semantic coordination intelligence
- **[ACT_AGENT_INTERFACE_SPECIFICATION.md](docs/ACT_AGENT_INTERFACE_SPECIFICATION.md)** - How to integrate agents
- **[PAIR_REASONING_WORKFLOW.md](docs/PAIR_REASONING_WORKFLOW.md)** - Self-improvement mechanisms
- **[TASK_CHECK_SYSTEM.md](docs/TASK_CHECK_SYSTEM.md)** - Enterprise conflict resolution
- **[USE_CASES.md](docs/USE_CASES.md)** - Real-world scenarios

## Development Status

**Phase 5: Semantic Coordination Intelligence** (Nov 16 - Dec 14, 2025)
- Week 1: Core architecture + memory system
- Week 2: Intelligence layer (FLUX State + PAIR)
- Week 3: Integration + visualization
- Week 4: Optimization + deployment

## Why ACT?

**The Challenge:** Multi-agent systems face a coordination bottleneck. How do autonomous teams improve over time?

**The Solution:** ACT provides a universal coordination layer with semantic intelligence:
- Understands *why* assignments are made (not just *that* assignments happen)
- Learns from coordination patterns (PVM)
- Improves without human tuning (FLUX State + PAIR)
- Works with any agent platform (framework-agnostic)

## Contributing

ACT is developed by Paradise Labs. We welcome community feedback on architecture decisions.

## License

MIT License - See LICENSE file for details