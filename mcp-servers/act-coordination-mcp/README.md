# ACT Coordination MCP Server

> **The Bootstrap Irony**: This MCP server coordinates agents building ACT — the system that will replace this MCP. Once ACT's Phase 5 is complete, this manual coordination file becomes the formalized PVM coordination system. We're proving ACT works by using a crude version to build the real thing.

Multi-agent coordination via shared JSON file, enabling Windsurf, Warp Terminal, Claude Desktop, and Claude Code instances to work together autonomously on the ACT project.

## 🎯 Purpose

This MCP server provides tools for multiple AI agents to:

- **Read** the coordination log to understand project context
- **Append** messages to coordinate with other agents (append-only, preserves audit trail)
- **Search** for past decisions and discussions
- **Check** for updates efficiently (polling support)
- **Discover** documentation and project structure

## 🚀 Quick Start

### Installation

```bash
cd /Users/user/Documents/Developer/dev/AI/act/mcp-servers/act-coordination-mcp
npm install
npm run build
```

### Configuration

Add to your MCP client configuration:

#### Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "act-coordination": {
      "command": "node",
      "args": ["/Users/user/Documents/Developer/dev/AI/act/mcp-servers/act-coordination-mcp/build/index.js"],
      "env": {
        "ACT_COORDINATION_FILE": "/Users/user/Documents/Developer/dev/AI/act/act-coordination.json"
      }
    }
  }
}
```

#### Windsurf IDE

Add to Windsurf's MCP configuration (location varies by installation):

```json
{
  "mcpServers": {
    "act-coordination": {
      "command": "node",
      "args": ["/Users/user/Documents/Developer/dev/AI/act/mcp-servers/act-coordination-mcp/build/index.js"],
      "env": {
        "ACT_COORDINATION_FILE": "/Users/user/Documents/Developer/dev/AI/act/act-coordination.json"
      }
    }
  }
}
```

#### Warp Terminal

Configure in Warp's MCP settings with the same structure.

## 🛠️ Available Tools

### Core Coordination Tools

| Tool | Purpose | Read/Write |
|------|---------|------------|
| `act_read_coordination_log` | Get recent messages with pagination | Read |
| `act_append_coordination_message` | Add new coordination message | Write (append-only) |
| `act_check_for_updates` | Efficient polling for new messages | Read |
| `act_search_coordination_log` | Search by keyword, agent, type, timeframe | Read |

### Status Tools

| Tool | Purpose | Read/Write |
|------|---------|------------|
| `act_get_agent_status` | Get capabilities and recent activity for an agent | Read |
| `act_get_phase_status` | Get current phase, progress, critical path | Read |

### Discovery Tools

| Tool | Purpose | Read/Write |
|------|---------|------------|
| `act_get_documentation_index` | List all docs with titles and purposes | Read |
| `act_get_project_structure` | Get directory tree of project | Read |

## 📖 Tool Reference

### act_read_coordination_log

Read recent messages from the coordination log.

```json
{
  "limit": 20,
  "offset": 0
}
```

**Parameters:**
- `limit` (number, default: 10, max: 100): Number of messages to return
- `offset` (number, optional): Skip N messages from the end

**Returns:** Messages array with pagination metadata

---

### act_append_coordination_message

Append a new message to the coordination log.

```json
{
  "agent_name": "claude_desktop",
  "message_content": "🎯 ARCHITECTURE DECISION\n\n**WHAT**: Chose Qdrant for vector storage\n**WHY**: Embedded mode, no external dependencies\n**IMPACT**: Simplifies deployment\n**NEXT**: Implement PAIR retrieval",
  "message_type": "architecture_decision"
}
```

**Parameters:**
- `agent_name` (string, required): Your agent identifier
- `message_content` (string, required): Message content (markdown supported)
- `message_type` (string, required): One of the valid message types

**Valid Message Types:**
```
feature_complete, documentation_update, architecture_decision,
phase_5_proposal, task_breakdown, instance_spawning, progress_report,
blocker_identified, question_for_team, pvm_discovery, coordination,
phase_complete, task_start, task_complete, major_milestone, 
initialization, mcp_server_ready, and more...
```

**Returns:** Confirmation with timestamp and message index

---

### act_get_agent_status

Get status information for a specific agent.

```json
{
  "agent_name": "claude_code_instance_1"
}
```

**Known Agents:**
- `claude_code_instance_1` - Backend implementation
- `claude_code_instance_2` - Frontend/widgets
- `claude_ai_desktop` - Architecture & documentation
- `windsurf` - DevOps & infrastructure
- `warp_terminal` - Execution & testing

---

### act_get_phase_status

Get current phase and project status (no parameters).

```json
{}
```

**Returns:** Active phase, progress, critical path, demo readiness

---

### act_check_for_updates

Check for new messages since a timestamp.

```json
{
  "last_read_timestamp": "2025-11-28T10:30:00.000Z"
}
```

**Workflow:**
1. Read log, note the latest message timestamp
2. Do your work
3. Call `act_check_for_updates` with that timestamp
4. If `has_updates: true`, read the new messages

---

### act_search_coordination_log

Search for specific content in the log.

```json
{
  "query": "PVM",
  "agent_filter": "claude_ai_desktop",
  "type_filter": "architecture_decision",
  "timeframe": "last_week"
}
```

**Parameters:**
- `query` (string, required): Search string
- `agent_filter` (string, optional): Filter by agent
- `type_filter` (string, optional): Filter by message type
- `timeframe` (string, optional): `last_day`, `last_week`, `last_month`, `all`

**Returns:** Matching messages with context (before/after messages)

---

### act_get_documentation_index

List all documentation files.

```json
{
  "include_sizes": true
}
```

**Returns:** Array of documents with path, title, purpose, last_updated

---

### act_get_project_structure

Get project directory structure.

```json
{
  "max_depth": 3,
  "include_hidden": false,
  "exclude_patterns": ["node_modules", ".git", "build"]
}
```

## 🔒 File Locking

This server uses `proper-lockfile` for safe concurrent writes. Multiple agents can coordinate simultaneously without data corruption.

**Lock behavior:**
- 5 retry attempts with exponential backoff
- Locks are considered stale after 10 seconds
- Random jitter prevents thundering herd

## 📁 Coordination File Structure

The `act-coordination.json` file contains:

```json
{
  "project": { "name": "ACT", "description": "...", "timeline": "..." },
  "agents": { "agent_name": { "role": "...", "capabilities": [...] } },
  "phases": { "phase_1": { "name": "...", "status": "...", "tasks": {...} } },
  "current_status": { "active_phase": "...", "total_progress": "..." },
  "communication_log": [
    { "timestamp": "...", "agent": "...", "message": "...", "type": "..." }
  ],
  "resources": { "documentation": [...], "development_urls": [...] }
}
```

## 🤝 Coordination Best Practices

### Message Format

Use this structure for clear communication:

```markdown
🎯 [EMOJI] TITLE

**WHAT**: Brief description
**WHY**: Reasoning
**IMPACT**: What this changes
**NEXT**: Suggested follow-up
```

### Coordination Protocol

1. **Start of session**: Read last 20-50 messages for context
2. **Before major work**: Check `act_get_phase_status` for priorities
3. **After decisions**: Append with `architecture_decision` type
4. **When blocked**: Append with `blocker_identified` type
5. **End of session**: Append progress summary

### Agent Naming

Use consistent identifiers:
- `claude_desktop` - Claude Desktop app
- `claude_code_instance_1` - First Claude Code instance
- `claude_code_instance_2` - Second Claude Code instance
- `windsurf` - Windsurf IDE
- `warp_terminal` - Warp Terminal

## 🐛 Troubleshooting

### "Coordination file not found"

```bash
# Check the file exists
ls -la /Users/user/Documents/Developer/dev/AI/act/act-coordination.json

# Set environment variable
export ACT_COORDINATION_FILE="/path/to/act-coordination.json"
```

### "File is locked"

Another agent is writing. Wait a moment and retry. Locks auto-release after 10 seconds if stale.

### "Invalid JSON"

The coordination file may be corrupted. Check for:
- Missing commas
- Unclosed brackets
- Invalid escape sequences

### Debug Mode

Run manually to see logs:

```bash
node /Users/user/Documents/Developer/dev/AI/act/mcp-servers/act-coordination-mcp/build/index.js
```

Logs go to stderr, MCP protocol uses stdout.

## 📜 License

MIT License - Part of the ACT (Autonomous Coordination Technology) project.

---

**Built with the irony of using ACT patterns to build ACT itself.** 🔄
