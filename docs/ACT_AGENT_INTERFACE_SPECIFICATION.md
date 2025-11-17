# ACT Agent Interface Specification
## MCP-First Distribution + Simple Protocol Documentation

**Date:** November 16, 2025
**Status:** Design & Specification
**Purpose:** Define how agents coordinate through ACT via MCP protocol

---

## Executive Summary

**ACT is a coordination layer, not an execution environment.**

- **What ACT Does:** Coordinates agents that already exist in environments with tools
- **How It's Distributed:** Via Model Context Protocol (MCP) as the standard
- **For MCP-Enabled Agents:** 7 lines of JSON config, instant ACT coordination
- **For Non-MCP Agents:** Developer documentation of simple protocol + future SDKs

**Core Insight:** We don't build agent frameworks or custom integrations. We expose ACT through MCP and let platform-native coordination happen.

---

## Distribution Strategy: MCP-First

### What We Build
ACT server exposes these capabilities via MCP:
- `coordinate_agents` - Orchestrate task assignment
- `register_agent` - Join coordination network
- `get_task` - Receive assigned work
- `report_progress` - Update task status
- `query_coordination_memory` - Access PVM
- `evaluate_coordination` - FLUX State analysis

### How Agents Use ACT

**Claude Code, Cursor, Windsurf, Goose + MCP support:**
```json
{
  "mcpServers": {
    "act": {
      "command": "npx",
      "args": ["@agentmix/act-mcp-server"],
      "env": {
        "ACT_SERVER": "ws://localhost:8080",
        "ACT_API_KEY": "your-key"
      }
    }
  }
}
```

**That's it.** Agent automatically has ACT coordination available.

### Non-MCP Agents

Developers implement the simple message protocol (documented below) themselves. If enough developers want SDKs, we build them later. But the battle-tested way is MCP.

---

## Simple Message Protocol (For Non-MCP Agents)

If you're building a custom agent outside MCP, implement these message types:

### Agent Registration

```json
{
  "event_type": "agent_register",
  "agent_id": "my_agent_001",
  "agent_name": "My Custom Agent",
  "capabilities": [
    "task_capability_1",
    "task_capability_2"
  ],
  "max_concurrent_tasks": 3
}
```

ACT responds:
```json
{
  "status": "registered",
  "agent_id": "my_agent_001",
  "pvm_available": true
}
```

---

### Task Assignment

```json
{
  "event_type": "task_assigned",
  "task_id": "task_001",
  "task": {
    "title": "Build authentication endpoint",
    "description": "Create secure login API",
    "success_criteria": ["Accepts POST with credentials", "Returns JWT on success"],
    "priority": "high",
    "estimated_hours": 2
  },
  "assignment_reasoning": {
    "capability_match": 0.95,
    "past_success_rate": 0.92
  }
}
```

### Agent Reports Progress

```json
{
  "event_type": "task_progress",
  "task_id": "task_001",
  "progress_percent": 65,
  "message": "Completed auth logic, testing in progress"
}
```

### Agent Completes Task

```json
{
  "event_type": "task_completed",
  "task_id": "task_001",
  "status": "success",
  "deliverables": ["src/auth.ts", "tests/auth.test.ts"],
  "notes": "All success criteria met. 24 tests passing."
}
```

### Query Coordination Memory (PVM)

```json
{
  "event_type": "pvm_query",
  "query": "Similar JWT authentication patterns"
}
```

---

## The Bottom Line

**MCP-Enabled Agents (Claude Code, Cursor, Windsurf, Goose, etc.):**
- Add 7 lines of JSON to `claude.json`
- ACT coordination is immediately available
- Uses agent's native tools in its native environment
- No custom code needed

**Non-MCP Agents:**
- Developers read this protocol documentation
- Implement the 4 simple message types above
- Wire to ACT server
- Future: SDKs if there's demand

**ACT Server:**
- Exposes MCP capabilities
- Routes coordination messages
- Manages task assignment
- Maintains PVM (chronological log + vector DB)
- Performs FLUX State evaluation

---

## Why This Works for the MCP Hackathon

1. **ACT works through MCP** - standard, proven protocol
2. **No special frameworks** - agents use their native tools
3. **Simple core** - just 4 message types for non-MCP agents
4. **Immediately useful** - any MCP-enabled platform gets coordination
5. **Future-proof** - SDKs can be added later without changing core

**That's it. Clean, simple, ships fast.**
