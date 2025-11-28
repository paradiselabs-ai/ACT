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

## 🧠 Semantic Coordination Intelligence (PVM)

### 7. PAIRed Vector Minutes - The Core Innovation

**What Makes ACT Different:** While other multi-agent frameworks coordinate task execution, ACT learns from coordination itself through **PAIRed Vector Minutes (PVM)** - the first semantic memory system designed specifically for agent coordination.

**Architecture Overview:**

```
┌────────────────────────────────────────────────────────┐
│           ACT Memory System (PVM)                      │
├────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────────────────────────────────────────┐ │
│  │   Chronological Coordination Log (Append-Only)   │ │
│  │   - Every coordination event recorded            │ │
│  │   - Never deleted, always auditable              │ │
│  │   - Complete timeline of decisions               │ │
│  └──────────────────────────────────────────────────┘ │
│              │                                          │
│              ├──────────────────────┐                  │
│              ↓                      ↓                  │
│  ┌─────────────────────┐  ┌──────────────────────┐   │
│  │  Vector Memory      │  │  Agent Profile       │   │
│  │  Store (Qdrant)     │  │  Builder             │   │
│  │  - Semantic search  │  │  - Individual memory │   │
│  │  - RAG retrieval    │  │  - Evidence-based    │   │
│  └─────────────────────┘  └──────────────────────┘   │
│              │                      │                  │
│              ↓                      ↓                  │
│  ┌──────────────────────────────────────────────────┐ │
│  │        FLUX State + PAIR Reasoning               │ │
│  │        - Unbiased evaluation                     │ │
│  │        - Context-guided improvement              │ │
│  └──────────────────────────────────────────────────┘ │
│                                                         │
└────────────────────────────────────────────────────────┘
```

#### 7.1 Chronological Coordination Log

**Purpose:** Complete, immutable audit trail of every coordination decision.

**Data Structure:**
```typescript
interface CoordinationMinute {
  id: string;
  timestamp: ISO8601;
  event_type: CoordinationEventType;
  agent_id?: string;
  project_id: string;

  // Event-specific data
  data: {
    task_id?: string;
    assignment_reasoning?: string;  // WHY this agent for this task
    conflict_type?: string;
    resolution?: any;
    success_criteria_met?: number;  // 0-100%
    tool_used?: string;
    tool_success?: boolean;
  };

  // Rich semantic metadata (Qoder-inspired)
  context_references?: string[];    // IDs of related past events
  memory_note?: string;              // Human-readable summary

  // Memory evaluation scores
  metadata: {
    recency_score: number;           // Time decay
    relevance_score: number;         // RAG retrieval frequency
    accuracy_score: number;          // Did decision lead to success?
    impact_score: number;            // How many future decisions influenced?
    composite_score: number;         // For pruning low-value memories
  };
}
```

**Storage:**
```typescript
class ChronologicalLog {
  private logFile: AppendOnlyFile;       // JSONL format
  private inMemoryBuffer: CoordinationMinute[] = [];

  async append(event: CoordinationMinute): Promise<void> {
    // 1. Add to in-memory buffer
    this.inMemoryBuffer.push(event);

    // 2. Persist to disk (append-only, never modify)
    await this.logFile.append(JSON.stringify(event) + '\n');

    // 3. Trigger vector indexing
    await this.onEventRecorded(event);
  }

  async getRecent(count: number): Promise<CoordinationMinute[]> {
    return this.inMemoryBuffer.slice(-count);
  }

  async getByIds(ids: string[]): Promise<CoordinationMinute[]> {
    return this.inMemoryBuffer.filter(e => ids.includes(e.id));
  }
}
```

#### 7.2 Vector Memory Store (Semantic Indexing)

**Purpose:** Enable fast semantic search without scanning entire chronological log.

**Technology Stack:**
- **Vector DB:** Qdrant (embedded for MVP, standalone for production)
- **Embedding Model:** sentence-transformers/all-MiniLM-L6-v2 (384-dim, open-source)
- **Alternative:** OpenAI text-embedding-3-small (for production)

**Implementation:**
```typescript
class VectorMemoryStore {
  private vectorDB: QdrantClient;
  private embedder: EmbeddingModel;

  async storeMemory(minute: CoordinationMinute): Promise<void> {
    // Generate semantic embedding
    const text = this.formatForEmbedding(minute);
    const embedding = await this.embedder.embed(text);

    // Store in Qdrant with metadata
    await this.vectorDB.upsert({
      id: minute.id,
      vector: embedding,
      payload: {
        category: this.categorizeEvent(minute),
        agent_id: minute.agent_id,
        project_id: minute.project_id,
        timestamp: minute.timestamp,
        metadata: minute.metadata
      }
    });
  }

  async search(query: string, limit: number = 10): Promise<CoordinationMinute[]> {
    const queryEmbedding = await this.embedder.embed(query);
    const results = await this.vectorDB.search({
      vector: queryEmbedding,
      limit,
      with_payload: true
    });

    return results.map(r => this.reconstructMinute(r));
  }
}
```

#### 7.3 FLUX State Evaluation Engine

**Purpose:** Unbiased self-evaluation through memory wipe.

**How It Works:**
1. **Memory Wipe:** After task completion, create fresh agent instance
2. **Fresh Evaluation:** Give agent: original task + success criteria + deliverables
3. **Critical Analysis:** Agent doesn't know it created the output (unbiased)
4. **Gap Identification:** "Does this meet criteria?" - identifies improvement opportunities
5. **Trigger PAIR:** If evaluation < 95%, retrieve relevant patterns

```typescript
class FluxStateEvaluator {
  async evaluateTask(
    task: Task,
    deliverables: any[],
    originalCriteria: string[]
  ): Promise<FluxEvaluation> {
    // Fresh agent with NO memory of creating deliverables
    const evaluationPrompt = `
      Original Task: ${task.description}
      Success Criteria: ${originalCriteria.join('\n')}
      Deliverables: ${JSON.stringify(deliverables)}

      Evaluate: Do these deliverables meet the success criteria?
      Be critical. Identify gaps.
    `;

    const evaluation = await this.freshAgent.analyze(evaluationPrompt);

    return {
      success_criteria_met: evaluation.score,
      identified_gaps: evaluation.gaps,
      improvement_suggestions: evaluation.suggestions
    };
  }
}
```

#### 7.4 PAIR Reasoning (Past Archived Information Re-injection)

**Purpose:** Context-guided improvement via semantic pattern retrieval.

**Workflow:**
```
1. FLUX State finds gaps (score < 95%)
   ↓
2. PAIR searches vector store: "Similar coordination patterns"
   ↓
3. Retrieves top 5 relevant past decisions
   ↓
4. Formats context: "Here's why similar decisions worked/failed"
   ↓
5. Agent either validates approach OR identifies improvement
   ↓
6. Loop until 95%+ success OR max 3 iterations
```

```typescript
class PAIRReasoningEngine {
  async improvementCycle(
    session: CoordinationSession
  ): Promise<ImprovedCoordination> {
    // 1. FLUX State evaluation
    const evaluation = await this.fluxEvaluator.evaluateTask(session);

    if (evaluation.success_criteria_met >= 95) {
      return { improved: false, reason: 'Already meets criteria' };
    }

    // 2. PAIR: Retrieve relevant patterns for each gap
    const improvements = [];
    for (const gap of evaluation.identified_gaps) {
      const relevantPatterns = await this.vectorStore.search(
        `Coordination pattern: ${gap.description}`,
        limit: 5
      );

      improvements.push({
        gap,
        relevantPatterns,
        suggestedChanges: await this.generateSuggestions(gap, relevantPatterns)
      });
    }

    // 3. Apply improvements and re-evaluate
    const revised = await this.applyImprovements(session, improvements);
    const reEvaluation = await this.fluxEvaluator.evaluateTask(revised);

    if (reEvaluation.success_criteria_met < 95) {
      return this.improvementCycle(revised);  // Recursive until convergence
    }

    return { improved: true, finalSession: revised };
  }
}
```

#### 7.5 Individual Agent Memory Derivation

**Critical Discovery (Nov 22, 2025):** PVM automatically derives individual agent profiles from coordination history - no separate memory management needed.

**What's Tracked Per-Agent (Evidence-Based):**
- Performance patterns (success rates by skill/task type)
- Skill progression (learning trajectories over time)
- Communication style (help-seeking, collaboration preferences)
- Tool usage effectiveness (which tools this agent uses best)
- Contextual patterns (when X works for this agent)
- Team synergy (which agents this agent collaborates well with)

```typescript
class AgentProfileBuilder {
  async buildProfile(agentId: string): Promise<AgentProfile> {
    const agentEvents = await this.chronologicalLog.getByAgent(agentId);

    return {
      agent_id: agentId,
      performance: await this.buildPerformanceProfile(agentEvents),
      skill_progression: await this.buildSkillProgression(agentEvents),
      communication: await this.buildCommunicationProfile(agentEvents),
      tool_usage: await this.buildToolUsageProfile(agentEvents),
      contextual_patterns: await this.buildContextualPatterns(agentEvents),
      team_synergy: await this.buildTeamSynergyProfile(agentEvents)
    };
  }
}
```

#### 7.6 User /improve Command

**Purpose:** Give users surgical precision control over coordination analysis.

**Syntax:**
```bash
/improve <scope> [--agents agents] [--session id] [--filter quality] [--output format]
```

**Scopes:**
- `communication` - Agent-to-agent communication effectiveness
- `tools` - Tool usage patterns and optimization
- `assignments` - Task assignment suitability analysis
- `conflicts` - Conflict detection and resolution
- `collaboration` - Team synergy and collaboration modes
- `performance` - Overall team effectiveness metrics

**Example:**
```bash
/improve communication \
  --agents agent2,agent3 \
  --session last \
  --filter bad \
  --output detailed-report
```

**Returns:** Structured analysis with specific issues, patterns, and actionable recommendations.

#### 7.7 Updated Event Types (PVM Integration)

Extended EventType enum to include PVM/FLUX/PAIR events:

```typescript
enum EventType {
  // ... (existing events)

  // PVM events
  COORDINATION_EVALUATED = 'coordination.evaluated',
  FLUX_STATE_CREATED = 'flux.state.created',
  PAIR_IMPROVEMENT_CYCLE = 'pair.improvement.cycle',
  MEMORY_PRUNED = 'memory.pruned',
  AGENT_PROFILE_UPDATED = 'agent.profile.updated',
  IMPROVE_COMMAND_EXECUTED = 'improve.command.executed'
}
```

#### 7.8 Database Schema Extension (PVM Tables)

```sql
-- Chronological coordination log
CREATE TABLE coordination_minutes (
  id UUID PRIMARY KEY,
  timestamp TIMESTAMP NOT NULL,
  event_type VARCHAR(100) NOT NULL,
  agent_id VARCHAR(100) REFERENCES agents(id),
  project_id UUID REFERENCES projects(id),
  data JSONB NOT NULL,
  context_references TEXT[],
  memory_note TEXT,
  metadata JSONB,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_coord_minutes_agent ON coordination_minutes(agent_id);
CREATE INDEX idx_coord_minutes_project ON coordination_minutes(project_id);
CREATE INDEX idx_coord_minutes_timestamp ON coordination_minutes(timestamp DESC);

-- FLUX State evaluations
CREATE TABLE flux_evaluations (
  id UUID PRIMARY KEY,
  coordination_session_id UUID NOT NULL,
  task_id UUID REFERENCES tasks(id),
  success_criteria_met INTEGER NOT NULL,  -- 0-100%
  identified_gaps JSONB,
  improvement_suggestions JSONB,
  evaluated_at TIMESTAMP DEFAULT NOW()
);

-- Agent profiles (cached, derived from coordination_minutes)
CREATE TABLE agent_profiles (
  agent_id VARCHAR(100) PRIMARY KEY REFERENCES agents(id),
  performance_metrics JSONB NOT NULL,
  skill_progression JSONB NOT NULL,
  communication_profile JSONB NOT NULL,
  tool_usage_profile JSONB NOT NULL,
  contextual_patterns JSONB NOT NULL,
  team_synergy JSONB NOT NULL,
  last_updated TIMESTAMP DEFAULT NOW()
);
```

**Note:** Vector embeddings are stored in Qdrant (external), not PostgreSQL, for performance.

### Related Documentation

- **Complete PVM Specification:** See `PVM_EXTENDED_CAPABILITIES.md`
- **PAIR Workflow Details:** See `PAIR_REASONING_WORKFLOW.md`
- **OpenMemory Comparison:** See `OPENMEMORY_VS_PVM_ANALYSIS.md`
- **Implementation Guide:** See `ACTMEMORYSYSTEM_IMPLEMENTATION.md`

---

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