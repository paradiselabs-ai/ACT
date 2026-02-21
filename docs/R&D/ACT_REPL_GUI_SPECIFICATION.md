# ACT REPL GUI Design Specification

## Overview
The ACT REPL (Read-Eval-Print Loop) is the terminal command center for the Agent Coordination Toolkit. It must be integrated into the AgentMix UI as an interactive terminal interface that allows users to control and monitor autonomous AI agent coordination.

## Core Requirements

### 1. Terminal Interface
- **Location**: Primary interface in AgentMix - should be accessible from main navigation
- **Design**: Full-screen terminal emulator with syntax highlighting
- **Behavior**: Command-line interface that accepts ACT commands and shows real-time output

### 2. Command Input
- **Prompt**: `>>: _` (as shown in all examples)
- **Auto-completion**: Tab completion for commands and project names
- **History**: Up/down arrow navigation through command history
- **Syntax Highlighting**: Different colors for commands, parameters, and quoted strings

### 3. Output Display
- **Real-time Streaming**: Commands execute and show progress in real-time
- **Structured Output**: Tables, progress bars, and formatted text
- **Interactive Elements**: Clickable elements (project names, agent IDs, task IDs)
- **Color Coding**: Success (green), warnings (yellow), errors (red), info (blue)

## Command Categories

### 1. Configuration Commands

#### List Agents Command
```bash
>>: list agents
```
**GUI Output:**
```
┌─────────────────┬──────────────────────┬──────────┬─────────────┐
│ Agent ID        │ Name                 │ Status   │ Workload    │
├─────────────────┼──────────────────────┼──────────┼─────────────┤
│ claude_code_1   │ Claude Code #1       │ Online   │ 0 tasks     │
│ claude_code_2   │ Claude Code #2       │ Online   │ 0 tasks     │
│ windsurf_main   │ Windsurf IDE         │ Online   │ 0 tasks     │
│ cursor_dev      │ Cursor               │ Online   │ 0 tasks     │
│ warp_terminal   │ Warp AI              │ Online   │ 0 tasks     │
└─────────────────┴──────────────────────┴──────────┴─────────────┘
```

**GUI Elements:**
- Clickable agent IDs that open agent detail panels
- Status indicators (● Online = green, ○ Offline = gray)
- Workload bars or progress indicators

#### Default Agent Setting
```bash
>>: default agent claude_code_1
✓ Default agent set to: claude_code_1
  All project decomposition and planning will use this agent's LLM.
```

### 2. Project Commands

#### Create Project
```bash
>>: create project todo-app in /Users/user/projects/todo-app

Creating project "todo-app"...
  Workspace: /Users/user/projects/todo-app
  Delegating decomposition to: claude_code_1 (default agent)

[claude_code_1 is analyzing the project...]

Project "todo-app" created!
  • 5 phases
  • 18 tasks
  • 5 agents assigned

Next steps:
  • Start execution: start project todo-app
  • View details: show project todo-app
```

**GUI Elements:**
- Progress spinner during analysis
- Success checkmark with expandable details
- Clickable "start project" and "show project" links that execute those commands

#### List Projects
```bash
>>: list projects
┌────────────────┬────────────────────────┬──────────┬─────────────┐
│ Project        │ Workspace              │ Status   │ Progress    │
├────────────────┼────────────────────────┼──────────┼─────────────┤
│ todo-app       │ ~/projects/todo-app    │ Active   │ 12/18 tasks │
│ api-server     │ ~/projects/api-server  │ Planning │ 0/24 tasks  │
│ podcast-site   │ ~/projects/podcast     │ Complete │ 15/15 tasks │
└────────────────┴────────────────────────┴──────────┴─────────────┘
```

**GUI Elements:**
- Clickable project names that execute `show project <name>`
- Status badges (Active = blue, Complete = green, Planning = yellow)
- Progress bars showing completion percentage

#### Show Project Details
```bash
>>: show project todo-app
Project: todo-app
Workspace: /Users/user/projects/todo-app
Status: Active (12/18 tasks complete)
Started: 2025-11-24 14:30
Estimated Completion: 2025-11-24 17:00

Current Tasks:
  ✓ task-1.1: Initialize repository (claude_code_1) - Complete
  ✓ task-2.1: Database setup (windsurf_main) - Complete
  🔄 task-3.1: Frontend setup (cursor_dev) - In Progress
  ⏳ task-3.2: Todo list component - Waiting on task-3.1
```

**GUI Elements:**
- Task list with clickable task IDs
- Status icons (✓ = green checkmark, 🔄 = blue spinner, ⏳ = yellow clock)
- Progress timeline visualization
- Agent avatars next to task assignments

### 3. Session Commands

#### Brainstorm Session
```bash
>>: brainstorm api-design --agents claude_code_1,windsurf_main,cursor_dev

Starting brainstorm session: "api-design"
Participants: 3 agents
Mode: Open discussion, no task execution

[Round 1: Initial ideas]
claude_code_1: "REST with GraphQL federation?"
windsurf_main: "Consider gRPC for microservices"
cursor_dev: "REST is simpler for MVP, add GraphQL later"
```

**GUI Elements:**
- Real-time chat interface for agent discussions
- Agent avatars with color-coded speech bubbles
- Round indicators showing discussion progress
- Save/export controls

#### Roundtable Session
```bash
>>: roundtable architecture-review --interactive

Starting roundtable: "architecture-review"
Mode: Interactive (HITL controls enabled)

Controls:
  pause           - Pause discussion
  resume          - Resume discussion
  select <agent>  - Highlight agent's point
  edit <msg_id>   - Edit message
  delete <msg_id> - Remove message
  send "message"  - User contributes
  stop            - End discussion
  clean_up        - Finalize and save
  wipe            - Remove from PVM entirely
```

**GUI Elements:**
- Control panel with buttons for each command
- Highlighted agent messages when selected
- User input field for contributions
- Real-time participant list with speaking indicators

### 4. Improvement Commands

#### Surgical Improvement Analysis
```bash
>>: improve communication -project todo-app

Analyzing project "todo-app"...
  • 156 coordination events
  • 5 agents participated
  • Duration: 3.2 hours

Communication Patterns:
  ✓ All agents contributed (balanced participation)
  ✓ Clear task handoffs
  ⚠️ Some technical jargon not explained

Issues Found:
1. windsurf_main used "CAP theorem" without explanation
   claude_code_2 had to ask for clarification

Recommendation: Establish shared terminology before discussions.

Save recommendation? (y/n): y
```

**GUI Elements:**
- Analysis progress indicator
- Expandable sections for different analysis categories
- Interactive charts showing communication patterns
- Recommendation cards with accept/reject actions

### 5. PVM Commands

#### PVM Search
```bash
>>: pvm search "authentication implementation patterns"

Found 8 relevant coordination events:

1. proj-003 task-4.1 (Success: 95%)
   JWT with refresh tokens, Redis session store
   Agent: claude_code_1, Duration: 45min

2. proj-007 task-2.4 (Success: 88%)
   OAuth2 with Google, session-based
   Agent: windsurf_main, Duration: 1.2hr
```

**GUI Elements:**
- Search results as expandable cards
- Similarity scores with visual indicators
- Agent and project links
- Export/save functionality

#### Agent Profile
```bash
>>: pvm profile claude_code_1

Agent Profile: claude_code_1
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Performance:
  Success Rate: 94%
  Tasks Completed: 156
  Average Task Duration: 23 minutes
  Specialization Score: High (focused on backend)
```

**GUI Elements:**
- Performance dashboard with charts
- Specialization radar chart
- Skill progression timeline
- Collaboration network visualization

## Technical Implementation Requirements

### 1. WebSocket Integration
```typescript
// Real-time command execution
const socket = io('http://localhost:8080');

// Send commands
socket.emit('act_command', { command: 'list agents' });

// Receive output streams
socket.on('act_output', (data) => {
  // Append to terminal display
  appendToTerminal(data.output, data.type);
});
```

### 2. Terminal Component
```jsx
function ACTTerminal() {
  const [command, setCommand] = useState('');
  const [history, setHistory] = useState([]);
  const [isExecuting, setIsExecuting] = useState(false);

  const executeCommand = async (cmd) => {
    setIsExecuting(true);
    const result = await actService.executeCommand(cmd);
    setHistory(prev => [...prev, { command: cmd, result }]);
    setIsExecuting(false);
  };

  return (
    <div className="act-terminal">
      <TerminalHistory history={history} />
      <CommandInput
        value={command}
        onChange={setCommand}
        onExecute={executeCommand}
        disabled={isExecuting}
      />
    </div>
  );
}
```

### 3. Real-time Widget Integration
- Commands that spawn widgets should integrate with AgentMix's widget system
- Task assignment widgets, progress widgets, etc. should appear in the main dashboard
- REPL should have a "widget mode" that shows coordination visualizations

### 4. Error Handling
- Syntax errors: Red highlighting with suggestions
- Network errors: Retry mechanisms with user feedback
- Command failures: Detailed error messages with recovery options

## Design Principles

### 1. Accessibility
- Keyboard navigation for all interactive elements
- Screen reader support for output
- High contrast mode support
- Resizable text and terminal sizing

### 2. Performance
- Lazy loading of command history
- Efficient rendering of large output streams
- Background processing for long-running commands
- Caching of frequently accessed data

### 3. User Experience
- Intuitive command discovery
- Contextual help and suggestions
- Undo/redo for destructive commands
- Command favorites/bookmarks

## Integration Points with AgentMix

### 1. Project Management
- ACT projects should appear in AgentMix project list
- Seamless switching between AgentMix conversations and ACT projects
- Shared project metadata and status

### 2. Agent Management
- ACT agent profiles should enhance AgentMix agent cards
- Shared agent status and capabilities
- Unified agent management interface

### 3. Conversation Integration
- ACT-coordinated conversations should appear as special conversation types
- Real-time coordination widgets should appear in conversation sidebar
- Seamless transition between human-led and autonomous coordination

## Testing Requirements

### 1. Command Testing
- All documented commands should work
- Error cases should be handled gracefully
- Real-time updates should work correctly

### 2. Performance Testing
- Large project listings should render efficiently
- Long-running commands should show progress
- Memory usage should be monitored

### 3. Integration Testing
- AgentMix navigation should work with ACT projects
- Widget system should integrate properly
- Real-time updates should sync across components

## Deployment Considerations

### 1. Backend Dependencies
- ACT server must be running and accessible
- WebSocket connections must be established
- PVM database must be available

### 2. Frontend Dependencies
- Socket.io client library
- Terminal emulation library (xterm.js recommended)
- Chart/visualization libraries for profiles and analytics

### 3. Configuration
- ACT server URL configuration
- Authentication tokens if required
- Default agent preferences
- Terminal theme customization
