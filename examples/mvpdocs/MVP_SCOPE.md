# ACT MVP: Technical Scope & Implementation Plan

## 🎯 MVP Architecture (2-Day Build)

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    ACT MVP Server                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ Agent       │  │ Task        │  │ WebSocket   │        │
│  │ Registry    │  │ Coordinator │  │ Hub         │        │
│  │ (Memory)    │  │ (Basic)     │  │ (Real-time) │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
             │                    │                    │
      ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
      │ AgentMix    │     │ Mock Agent  │     │ Mock Agent  │
      │ (Real)      │     │ (Frontend)  │     │ (Backend)   │
      └─────────────┘     └─────────────┘     └─────────────┘
```

## Day 1: Foundation (Core Server)

### Hour 1-4: Basic Server Setup
```typescript
// MVP Server Structure
act/server/src/
├── index.ts              // Server entry point
├── types/
│   ├── agent.ts          // Agent interfaces
│   ├── task.ts           // Task definitions
│   └── event.ts          // Event types
├── services/
│   ├── AgentRegistry.ts  // In-memory agent tracking
│   ├── TaskCoordinator.ts// Basic task assignment
│   └── EventHub.ts       // WebSocket communication
└── routes/
    └── api.ts            // REST endpoints
```

**Key Files:**
```typescript
// types/agent.ts
export interface Agent {
  id: string;
  name: string;
  capabilities: string[];
  status: 'online' | 'busy' | 'offline';
  currentTask?: string;
  lastSeen: Date;
}

// types/task.ts
export interface Task {
  id: string;
  description: string;
  requiredCapabilities: string[];
  assignedAgent?: string;
  status: 'pending' | 'assigned' | 'in_progress' | 'completed';
  priority: 'low' | 'medium' | 'high';
  dependencies: string[];
}

// types/event.ts
export interface CoordinationEvent {
  type: 'agent_join' | 'task_assign' | 'task_complete' | 'conflict_detected';
  agentId: string;
  data: any;
  timestamp: Date;
}
```

### Hour 5-8: Core Coordination Logic
```typescript
// services/TaskCoordinator.ts
export class TaskCoordinator {
  private tasks: Map<string, Task> = new Map();
  private agentRegistry: AgentRegistry;

  // MVP: Simple capability matching
  findBestAgent(task: Task): Agent | null {
    const availableAgents = this.agentRegistry.getAvailable();

    // Basic scoring: capability match + availability
    return availableAgents
      .filter(agent => this.hasRequiredCapabilities(agent, task))
      .sort((a, b) => this.scoreAgent(a, task) - this.scoreAgent(b, task))[0];
  }

  // MVP: Basic conflict detection
  detectConflicts(): Conflict[] {
    const conflicts = [];
    const busyAgents = this.agentRegistry.getBusyAgents();

    // Simple conflict: multiple tasks assigned to same agent
    busyAgents.forEach(agent => {
      const assignedTasks = this.getTasksByAgent(agent.id);
      if (assignedTasks.length > 1) {
        conflicts.push({
          type: 'resource_contention',
          agentId: agent.id,
          tasks: assignedTasks
        });
      }
    });

    return conflicts;
  }
}
```

## Day 2: Integration & Demo

### Hour 1-4: Client SDK
```python
# sdk/python/act_client.py
import asyncio
import websockets
import json
from typing import List, Optional

class ACTClient:
    def __init__(self, agent_id: str, capabilities: List[str], server_url: str = "ws://localhost:8080"):
        self.agent_id = agent_id
        self.capabilities = capabilities
        self.server_url = server_url
        self.ws = None
        self.current_task = None

    async def connect(self):
        """Connect to ACT server and register agent"""
        self.ws = await websockets.connect(self.server_url)
        await self.register()

    async def register(self):
        """Register agent with ACT server"""
        message = {
            "type": "agent_register",
            "agent_id": self.agent_id,
            "capabilities": self.capabilities,
            "status": "online"
        }
        await self.ws.send(json.dumps(message))

    async def check_for_tasks(self) -> Optional[dict]:
        """Check for assigned tasks"""
        message = {"type": "get_assigned_tasks", "agent_id": self.agent_id}
        await self.ws.send(json.dumps(message))

        response = await self.ws.recv()
        data = json.loads(response)

        if data.get("tasks"):
            self.current_task = data["tasks"][0]
            return self.current_task
        return None

    async def report_progress(self, task_id: str, status: str, message: str = ""):
        """Report task progress to coordinator"""
        progress_update = {
            "type": "task_progress",
            "task_id": task_id,
            "agent_id": self.agent_id,
            "status": status,
            "message": message,
            "timestamp": time.time()
        }
        await self.ws.send(json.dumps(progress_update))
```

### Hour 5-8: AgentMix Integration
```python
# Integration into AgentMix backend
from act_client import ACTClient

class AgentMixACTIntegration:
    def __init__(self, socketio):
        self.socketio = socketio
        self.act_client = ACTClient(
            agent_id="agentmix_orchestrator",
            capabilities=["conversation", "ui", "project_management"]
        )

    async def start_coordinated_project(self, project_description: str):
        """Start a project with ACT coordination"""
        # Connect to ACT
        await self.act_client.connect()

        # Create project in ACT
        project_data = {
            "type": "create_project",
            "description": project_description,
            "required_agents": [
                {"capabilities": ["frontend", "react"], "name": "Frontend Agent"},
                {"capabilities": ["backend", "python"], "name": "Backend Agent"},
                {"capabilities": ["testing", "qa"], "name": "QA Agent"}
            ]
        }

        # Let ACT coordinate the agents
        await self.act_client.ws.send(json.dumps(project_data))

        # Listen for coordination events
        async for message in self.act_client.ws:
            event = json.loads(message)
            await self.handle_coordination_event(event)

    async def handle_coordination_event(self, event):
        """Handle events from ACT coordinator"""
        if event["type"] == "task_assigned":
            # Notify frontend about task assignment
            self.socketio.emit('act_task_assigned', event)
        elif event["type"] == "conflict_detected":
            # Show conflict resolution in UI
            self.socketio.emit('act_conflict', event)
```

## Demo Scenario: "Build Todo App"

### Setup (30 seconds)
1. Start ACT server: `npm run dev`
2. Launch enhanced AgentMix: `python backend/src/main.py`
3. Open AgentMix frontend with ACT dashboard

### Live Demo (5 minutes)
1. **User Input**: "Build a simple todo app with React frontend and Flask backend"
2. **ACT Coordination**:
   - Decomposes into tasks automatically
   - Assigns Frontend Agent (React components)
   - Assigns Backend Agent (Flask API)
   - Assigns QA Agent (testing)
3. **Real-Time Visualization**:
   - Show agents appearing in registry
   - Show tasks being assigned
   - Show progress updates in real-time
   - Show conflict detection (if any)
4. **Result**: Working todo app with autonomous coordination

## MVP Success Criteria

### Technical Requirements ✅
- [ ] 3+ agents connect and register
- [ ] Tasks assigned automatically based on capabilities
- [ ] Real-time progress tracking
- [ ] Basic conflict detection
- [ ] AgentMix integration working

### Demo Requirements ✅
- [ ] Live coordination visible to audience
- [ ] Agents complete tasks autonomously
- [ ] Conflicts detected and resolved
- [ ] Working application delivered
- [ ] Clear differentiation from existing solutions

## Post-MVP Migration Path

### Week 1: Persistence
- Replace in-memory storage with PostgreSQL
- Add proper data models and relationships
- Implement state recovery after restarts

### Week 2: Advanced Coordination
- ML-based agent matching
- Complex conflict resolution
- Predictive task assignment

### Week 3: File Watching
- Automatic progress detection
- Code change event handling
- Smart task completion detection

### Week 4: Production Ready
- Authentication and authorization
- Multi-tenant support
- Monitoring and alerting
- API rate limiting

This MVP proves the concept while maintaining clear path to the revolutionary full system! 🚀