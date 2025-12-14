# ACT Terminal REPL

The comprehensive command-line interface for ACT (Agent Coordination Toolkit). Manage projects, run coordination sessions, analyze performance, and interact with the PVM memory system.

## Installation

```bash
npm install -g @agentmix/act-repl
```

Or run locally:

```bash
cd cli
npm install
npm run build
npm link
```

## Quick Start

1. **Start ACT Server:**
```bash
act server start
# Or manually: cd /path/to/act/server && npm start
```

2. **Launch REPL:**
```bash
act
```

3. **Configure Default Agent:**
```bash
>>: list agents
>>: default agent claude_code_1
```

4. **Create Project:**
```bash
>>: create project "My Todo App" in ~/projects/todo-app
```

## Command Categories

### Configuration
- `list agents` - Show connected agents
- `default agent <id>` - Set default agent
- `show default` - Show current default agent

### Projects
- `create project <name> in <path>` - Start new project
- `continue project <name>` - Resume paused project
- `list projects` - Show all projects
- `show project <name>` - Project details
- `stop project <name>` - Pause project
- `delete project <name>` - Remove project

### Sessions
- `brainstorm <topic>` - Creative ideation
- `experiment <name>` - Comparative testing
- `roundtable <topic>` - Multi-agent discussion
- `roundtable <topic> --interactive` - HITL controls

### Interactive Controls (during --interactive sessions)
- `pause` - Pause discussion
- `resume` - Resume discussion
- `select <agent>` - Highlight contribution
- `edit <msg_id>` - Edit message
- `send "<message>"` - User contributes
- `clean_up` - Finalize and save
- `wipe` - Remove from PVM

### Improvement Analysis
```bash
improve communication -project todo-app
improve performance --agents agent1,agent2 --filter bad
improve knowledge -roundtable arch-review --print ./report.md
```

### PVM Memory System
- `pvm stats` - Show statistics
- `pvm search <query>` - Search history
- `pvm profile <agent_id>` - Agent profile
- `pvm export <path>` - Export database
- `pvm import <path>` - Import database

### System
- `status` - Server status
- `help` - Command reference
- `exit` - Exit REPL

## Environment Variables

- `ACT_SERVER_URL` - Server URL (default: http://localhost:8080)

## Examples

### Project Management
```bash
# Create project with natural language
>>: create project "Build a REST API with authentication" in ~/api-project

# Continue existing work
>>: continue project api-project
>>: show project api-project
```

### Coordination Sessions
```bash
# Brainstorm ideas
>>: brainstorm "UI design patterns"

# Run comparative experiment
>>: experiment react-vs-vue --agents frontend_agent,ui_agent
>>: experiment -analyze react-vs-vue

# Interactive roundtable
>>: roundtable "architecture decisions" --interactive
>>: send "What about scalability?"
>>: select backend_agent
>>: clean_up
```

### Performance Analysis
```bash
# Analyze communication patterns
>>: improve communication -project api-project --filter bad

# Tool usage optimization
>>: improve tools --agents all --success-rate <80

# Export detailed report
>>: improve performance -project api-project --print ./perf-report.md
```

### PVM Memory Operations
```bash
# Search coordination history
>>: pvm search "authentication patterns"

# View agent profile
>>: pvm profile claude_code_1

# Export memory for backup
>>: pvm export ./pvm-backup-2025-12-09.json
```

## Architecture

The ACT REPL consists of:

- **ACTRepl** - Main REPL loop and command routing
- **ACTClient** - HTTP client for server communication
- **SessionManager** - Session state and coordination
- **HelpSystem** - Comprehensive command documentation

## Development

```bash
# Install dependencies
npm install

# Build TypeScript
npm run build

# Run in development mode
npm run dev

# Test CLI
npm link
act --help
```

## Integration with MCP

The REPL is designed to work seamlessly with ACT's MCP (Model Context Protocol) server for enhanced agent coordination capabilities.
