# ACT System Architecture

## 🏗️ High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    ACT Coordination Hub                     │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ Agent API   │  │ Task Engine │  │ Event Bus   │        │
│  │ Gateway     │  │             │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│          │               │               │                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ State DB    │  │ File Watch  │  │ WebSocket   │        │
│  │ (Real-time) │  │ Service     │  │ Broadcasting│        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
           │                    │                    │
    ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
    │ Claude Code │     │  Windsurf   │     │ Future Agent│
    │  Client     │     │  Client     │     │  Clients    │
    └─────────────┘     └─────────────┘     └─────────────┘
```

## 🔧 Core Components

### 1. ACT Coordination Service

**Technology Stack:**
- **Runtime**: Node.js with TypeScript
- **Database**: PostgreSQL + Redis
- **Message Queue**: Apache Kafka
- **WebSocket**: Socket.io
- **API**: GraphQL + REST hybrid

**Core Services:**
```typescript
class ACTCoordinationService {
  private taskEngine: TaskEngine;
  private eventBus: EventBus;
  private agentRegistry: AgentRegistry;
  private fileWatcher: FileWatcher;
  private stateDB: StateDatabase;
  private conflictResolver: ConflictResolver;
}
```

### 2. Agent Client SDK

**Multi-Language Support:**
- **Python**: `agentmix-act-python`
- **JavaScript/TypeScript**: `@agentmix/act-client`
- **Go**: `github.com/agentmix/act-go`
- **Rust**: `agentmix-act-rust`

**Core Client Interface:**
```typescript
interface ACTClient {
  // Lifecycle management
  register(capabilities: Capability[]): Promise<AgentID>;
  heartbeat(): Promise<AgentStatus>;
  disconnect(): Promise<void>;

  // Task management
  claimTask(taskId: TaskID): Promise<boolean>;
  reportProgress(taskId: TaskID, progress: Progress): Promise<void>;
  requestAssistance(context: AssistanceRequest): Promise<AgentID[]>;

  // Communication
  sendMessage(targetAgent: AgentID, message: Message): Promise<void>;
  broadcastUpdate(update: ProjectUpdate): Promise<void>;

  // File system integration
  watchFiles(patterns: string[]): Promise<void>;
  reportFileChange(change: FileChange): Promise<void>;
}
```

### 3. Database Schema

```sql
-- Core project structure
CREATE TABLE projects (
  id UUID PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  status VARCHAR(50) NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Agent registry and capabilities
CREATE TABLE agents (
  id VARCHAR(100) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  capabilities JSONB NOT NULL,
  status VARCHAR(50) NOT NULL,
  last_seen TIMESTAMP DEFAULT NOW(),
  current_project_id UUID REFERENCES projects(id),
  performance_metrics JSONB
);

-- Dynamic project phases
CREATE TABLE phases (
  id UUID PRIMARY KEY,
  project_id UUID REFERENCES projects(id),
  name VARCHAR(255) NOT NULL,
  status VARCHAR(50) NOT NULL,
  owner_agent_id VARCHAR(100) REFERENCES agents(id),
  dependencies UUID[],
  estimated_effort INTEGER,
  actual_effort INTEGER,
  started_at TIMESTAMP,
  completed_at TIMESTAMP
);

-- Granular task management
CREATE TABLE tasks (
  id UUID PRIMARY KEY,
  phase_id UUID REFERENCES phases(id),
  title VARCHAR(255) NOT NULL,
  description TEXT,
  assigned_agent_id VARCHAR(100) REFERENCES agents(id),
  status VARCHAR(50) NOT NULL,
  priority INTEGER NOT NULL,
  complexity_score INTEGER,
  file_paths TEXT[],
  dependencies UUID[],
  estimated_time INTEGER,
  actual_time INTEGER,
  created_at TIMESTAMP DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP
);

-- Real-time communication
CREATE TABLE agent_communications (
  id UUID PRIMARY KEY,
  project_id UUID REFERENCES projects(id),
  sender_agent_id VARCHAR(100) REFERENCES agents(id),
  recipient_agent_id VARCHAR(100) REFERENCES agents(id),
  message_type VARCHAR(50) NOT NULL,
  content JSONB NOT NULL,
  timestamp TIMESTAMP DEFAULT NOW(),
  read_at TIMESTAMP
);

-- File system monitoring
CREATE TABLE file_watches (
  id UUID PRIMARY KEY,
  project_id UUID REFERENCES projects(id),
  file_path VARCHAR(500) NOT NULL,
  related_task_ids UUID[],
  last_modified TIMESTAMP,
  checksum VARCHAR(64)
);

-- Conflict resolution tracking
CREATE TABLE conflicts (
  id UUID PRIMARY KEY,
  project_id UUID REFERENCES projects(id),
  conflict_type VARCHAR(50) NOT NULL,
  involved_agents VARCHAR(100)[],
  description TEXT,
  status VARCHAR(50) NOT NULL,
  resolution JSONB,
  created_at TIMESTAMP DEFAULT NOW(),
  resolved_at TIMESTAMP
);
```

### 4. Event-Driven Architecture

**Event Types:**
```typescript
interface ACTEvent {
  id: string;
  type: EventType;
  agentId: string;
  projectId: string;
  timestamp: number;
  data: any;
  priority: 'low' | 'medium' | 'high' | 'critical';
}

enum EventType {
  // Agent lifecycle
  AGENT_REGISTERED = 'agent.registered',
  AGENT_HEARTBEAT = 'agent.heartbeat',
  AGENT_DISCONNECTED = 'agent.disconnected',

  // Task management
  TASK_CREATED = 'task.created',
  TASK_CLAIMED = 'task.claimed',
  TASK_PROGRESS = 'task.progress',
  TASK_COMPLETED = 'task.completed',
  TASK_BLOCKED = 'task.blocked',

  // File system
  FILE_CREATED = 'file.created',
  FILE_MODIFIED = 'file.modified',
  FILE_DELETED = 'file.deleted',

  // Coordination
  CONFLICT_DETECTED = 'conflict.detected',
  CONFLICT_RESOLVED = 'conflict.resolved',
  PHASE_COMPLETED = 'phase.completed',
  PROJECT_MILESTONE = 'project.milestone'
}
```

### 5. Autonomous Coordination Algorithms

**Task Assignment Algorithm:**
```typescript
class IntelligentTaskAssignment {
  async findOptimalAgent(
    task: Task,
    availableAgents: Agent[]
  ): Promise<Agent> {
    const scoredAgents = availableAgents.map(agent => ({
      agent,
      score: this.calculateCompatibilityScore(task, agent)
    }));

    return scoredAgents
      .sort((a, b) => b.score - a.score)[0]
      .agent;
  }

  private calculateCompatibilityScore(task: Task, agent: Agent): number {
    const capabilityMatch = this.calculateCapabilityMatch(task, agent);
    const workloadFactor = this.calculateWorkloadFactor(agent);
    const performanceHistory = this.getPerformanceHistory(agent, task.type);
    const contextualRelevance = this.calculateContextualRelevance(task, agent);

    return (
      capabilityMatch * 0.4 +
      workloadFactor * 0.2 +
      performanceHistory * 0.3 +
      contextualRelevance * 0.1
    );
  }
}
```

**Conflict Resolution Engine:**
```typescript
class ConflictResolver {
  async resolveConflict(conflict: Conflict): Promise<Resolution> {
    switch (conflict.type) {
      case ConflictType.RESOURCE_CONTENTION:
        return this.resolveResourceContention(conflict);

      case ConflictType.DEPENDENCY_DEADLOCK:
        return this.resolveDependencyDeadlock(conflict);

      case ConflictType.CAPABILITY_OVERLAP:
        return this.resolveCapabilityOverlap(conflict);

      case ConflictType.PRIORITY_MISMATCH:
        return this.resolvePriorityMismatch(conflict);

      default:
        return this.escalateToHuman(conflict);
    }
  }
}
```

### 6. Real-Time State Synchronization

**State Management:**
```typescript
class DistributedState {
  private redis: Redis;
  private eventBus: EventBus;

  async updateProjectState(
    projectId: string,
    update: StateUpdate
  ): Promise<void> {
    // Optimistic update
    await this.redis.hset(
      `project:${projectId}`,
      update.key,
      JSON.stringify(update.value)
    );

    // Broadcast to all connected agents
    this.eventBus.emit('state.updated', {
      projectId,
      update,
      timestamp: Date.now()
    });
  }

  async getProjectState(projectId: string): Promise<ProjectState> {
    const state = await this.redis.hgetall(`project:${projectId}`);
    return Object.fromEntries(
      Object.entries(state).map(([k, v]) => [k, JSON.parse(v)])
    );
  }
}
```

## 🚀 Deployment Architecture

### Docker Composition
```yaml
version: '3.8'
services:
  act-coordinator:
    image: agentmix/act-coordinator:latest
    ports:
      - "3000:3000"  # REST API
      - "8080:8080"  # WebSocket
    environment:
      - DATABASE_URL=postgresql://act:secure@postgres:5432/act_db
      - REDIS_URL=redis://redis:6379
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      - postgres
      - redis
      - kafka

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: act_db
      POSTGRES_USER: act
      POSTGRES_PASSWORD: secure
    volumes:
      - act_postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - act_redis_data:/data

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
    depends_on:
      - zookeeper

  file-watcher:
    image: agentmix/file-watcher:latest
    volumes:
      - /project/workspace:/watched:ro
    environment:
      - ACT_COORDINATOR_URL=http://act-coordinator:3000

volumes:
  act_postgres_data:
  act_redis_data:
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: act-coordinator
spec:
  replicas: 3
  selector:
    matchLabels:
      app: act-coordinator
  template:
    metadata:
      labels:
        app: act-coordinator
    spec:
      containers:
      - name: coordinator
        image: agentmix/act-coordinator:latest
        ports:
        - containerPort: 3000
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: act-secrets
              key: database-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## 🔐 Security Considerations

**Authentication & Authorization:**
- JWT-based agent authentication
- Role-based access control (RBAC)
- API key management for external integrations
- End-to-end encryption for sensitive communications

**Data Protection:**
- Project isolation at database level
- Encrypted file storage for sensitive code
- Audit logging for all agent actions
- GDPR compliance for user data

**Network Security:**
- TLS 1.3 for all communications
- VPC isolation in cloud deployments
- Rate limiting and DDoS protection
- Input validation and sanitization

This architecture provides the foundation for truly autonomous multi-agent coordination while maintaining security, scalability, and reliability at enterprise scale.