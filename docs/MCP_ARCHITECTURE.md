# MCP_ARCHITECTURE.md

**ACT: Working Through the Land of MCPs**

*Where servers are clients, clients are servers, and everyone is serving everyone else in perfect harmony.*

---

## Table of Contents

1. [The Recursive Nature of ACT's MCP Architecture](#the-recursive-nature)
2. [Core Principle: ACT is ACT (Not "Many MCPs")](#core-principle)
3. [The MCP Microservices Pattern](#microservices-pattern)
4. [Server/Client Duality Explained](#server-client-duality)
5. [ACT Services Architecture](#act-services)
6. [MCP Capabilities in Detail](#mcp-capabilities)
7. [Deployment Scenarios](#deployment-scenarios)
8. [Implementation Patterns](#implementation-patterns)
9. [Why This Architecture Wins](#why-this-wins)

---

## The Recursive Nature

**The ACT MCP Architecture in One Sentence:**

> ACT, working through the land of MCPs, serves its own server as a client by connecting the client server as a server and client that the server the agents connect to becomes a client for its own other servers—where clients are served and servers are clients to client servers, providing a service for each client that is a server-client serving client-server clients by a server to serve the other servers' client-servers and server-clients of clients serving the server which is a client to the client server.

**Translation:**

ACT is built from modular services that each act as BOTH MCP servers (exposing capabilities) AND MCP clients (consuming other services). External agents connect to the ACT Bridge (a server), which is ALSO a client to ACT's internal services, which are ALSO servers exposing tools while simultaneously being clients to each other.

**It's servers all the way down. And clients all the way up. Simultaneously.**

---

## Core Principle

### ACT is ACT (Not "Many MCPs")

**What ACT IS:**
- Coordination infrastructure for autonomous multi-agent systems
- Semantic intelligence layer (PVM, FLUX, PAIR)
- Agent registry, task coordination, learning loop

**What ACT is NOT:**
- "A collection of MCP servers"
- "An MCP implementation"
- "MCP-based coordination"

**How ACT is BUILT:**
- Modular microservices architecture
- MCP protocol for inter-service communication (implementation detail)
- Clean separation of concerns, independent scaling

**How ACT is ACCESSED:**
- **Route 1:** ACT-by-MCP (7-line config, "pasta from box")
- **Route 2:** ACT-by-SDK (Direct APIs, "noodles from flour")

**MCP's Role:**
- **Internally:** Communication protocol between ACT services
- **Externally:** ONE distribution option (not the only one)

---

## Microservices Pattern

### The Big Picture

```
┌─────────────────────────────────────────────────────────┐
│                    External Agents                       │
│  (Claude Desktop, Cursor, Custom Python Agent)           │
└─────────────────┬───────────────────────┬───────────────┘
                  │                       │
       ┌──────────▼─────────┐   ┌────────▼──────────┐
       │   ACT-by-MCP       │   │   ACT-by-SDK      │
       │   (MCP Bridge)     │   │   (Direct APIs)   │
       └──────────┬─────────┘   └────────┬──────────┘
                  │                      │
                  └──────────┬───────────┘
                             │
          ┌──────────────────▼──────────────────┐
          │         ACT Core Infrastructure     │
          │    (Modular MCP Microservices)      │
          └──────────────────┬──────────────────┘
                             │
       ┌─────────────────────┼─────────────────────┐
       │                     │                     │
   ┌───▼────┐  ┌──────▼─────┐  ┌────▼──────┐ ┌───▼────┐
   │PVM MCP │  │Coord MCP   │  │Analytics  │ │FLUX    │
   │Server  │  │Server      │  │MCP Server │ │MCP     │
   │+Client │  │+Client     │  │+Client    │ │Server  │
   └───┬────┘  └──────┬─────┘  └────┬──────┘ └───┬────┘
       │              │               │            │
       └──────────────┴───────────────┴────────────┘
              Each service can call others
```

### Service Communication Flow

**Example: Smart Task Assignment**

```
1. Agent → ACT Bridge: "assign_task"
   └─ ACT Bridge (as server) receives request

2. ACT Bridge → Coordination Service: "assign_task_intelligently"
   └─ Bridge (as client) calls Coordination (as server)

3. Coordination → PVM Service: "search_similar_tasks"
   └─ Coordination (as client) calls PVM (as server)

4. PVM → returns: [similar tasks with outcomes]
   └─ PVM (as server) responds to Coordination (as client)

5. Coordination → determines optimal agent based on PVM data
   └─ Coordination (as server) processes and decides

6. Coordination → returns: {assignedTo: "agent_x", reasoning: "..."}
   └─ Coordination (as server) responds to Bridge (as client)

7. ACT Bridge → Agent: {success: true, assignment: {...}}
   └─ Bridge (as server) responds to Agent (as client)
```

**Every arrow is an MCP tool call. Every service is both server and client.**

---

## Server/Client Duality

### How Each Service is BOTH

**Pattern: Dual-Interface Services**

```typescript
// act-pvm-server/index.ts

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';

// ============================================
// SERVER INTERFACE (Expose PVM Capabilities)
// ============================================
const pvmServer = new Server(
  { name: 'act-pvm', version: '1.0.0' },
  { capabilities: {
      tools: { listChanged: true },
      resources: { subscribe: true, listChanged: true },
      logging: {}
    }
  }
);

// Expose tools
pvmServer.setRequestHandler(CallToolRequestSchema, async (request) => {
  switch (request.params.name) {
    case 'search_pvm':
      return await performSemanticSearch(request.params.arguments);

    case 'store_coordination_event':
      return await storeEvent(request.params.arguments);
  }
});

// Expose resources
pvmServer.setRequestHandler(ReadResourceRequestSchema, async (request) => {
  if (request.params.uri.startsWith('act://pvm/recent')) {
    return await getRecentCoordination();
  }
});

// ============================================
// CLIENT INTERFACE (Consume Other Services)
// ============================================
const coordClient = new Client(
  { name: 'pvm-to-coordination', version: '1.0.0' },
  { capabilities: { sampling: { tools: {} } } }
);

// Connect to coordination service
await coordClient.connect(coordinationServiceTransport);

// Call coordination service when needed
async function enrichEventWithContext(event) {
  // PVM (as client) calls Coordination (as server)
  const context = await coordClient.callTool({
    name: 'get_active_agents',
    arguments: {}
  });

  return { ...event, context };
}

// Start both
await pvmServer.connect(transport); // Act as server
await coordClient.connect(coordTransport); // Act as client
```

**Every ACT service follows this pattern:**
1. Creates MCP Server instance (exposes capabilities)
2. Creates MCP Client instance(s) (consumes other services)
3. Runs both simultaneously
4. Services form interconnected mesh

---

## ACT Services

### Service Breakdown

#### 1. **ACT Bridge** (act-mcp-bridge)

**As Server:**
- Exposes unified ACT API to external agents
- Tools: register_agent, get_task, query_pvm, improve_coordination, etc.
- Resources: act://agents/*, act://tasks/*, act://pvm/*
- Prompts: create-project, analyze-performance, improve-workflow
- Logging: All coordination events to clients

**As Client:**
- Connects to: PVM, Coordination, Analytics, FLUX, PAIR
- Orchestrates: Routes requests to appropriate internal service
- Sampling: Uses LLM reasoning for complex orchestration decisions

**Purpose:**
Single entry point for external agents. Thin orchestration layer.

---

#### 2. **ACT PVM Server** (act-pvm-server)

**As Server:**
- Tools:
  - `search_pvm` - Semantic search over coordination history
  - `store_event` - Store coordination event with embedding
  - `get_similar_agents` - Find agents with similar performance
  - `get_similar_tasks` - Find tasks with similar outcomes

- Resources:
  - `act://pvm/recent/{count}` - Recent coordination events
  - `act://pvm/agent/{id}/history` - Agent's coordination history
  - `act://pvm/patterns/{type}` - Successful coordination patterns

- Logging: Search queries, embedding operations, cache hits

**As Client:**
- Connects to: Coordination (for real-time agent context)
- Uses: Agent status for search result filtering

**Purpose:**
Semantic memory. Vector embeddings. RAG retrieval.

**Tech Stack:**
- Qdrant for vector DB (or MockVectorStore for dev)
- Sentence transformers for embeddings
- JSONL for chronological log

---

#### 3. **ACT Coordination Server** (act-coordination-server)

**As Server:**
- Tools:
  - `assign_task_intelligently` - Smart task assignment
  - `get_active_agents` - Current agent registry
  - `create_task` - Task creation with decomposition
  - `delegate_subtask` - Delegate to specialized agent

- Resources:
  - `act://agents/*/status` - Real-time agent status
  - `act://tasks/*/progress` - Task progress tracking
  - `act://coordination/conflicts` - Detected conflicts

- Prompts:
  - `create-coordinated-project` - Full project workflow
  - `resolve-conflict` - Guided conflict resolution

**As Client:**
- Connects to: PVM (for similarity search), FLUX (for evaluation)
- Calls PVM: When assigning tasks (find similar past assignments)
- Calls FLUX: When task completes (evaluate outcome)
- Sampling: Uses LLM to reason about optimal assignments

**Purpose:**
Core coordination logic. Agent registry. Task management.

---

#### 4. **ACT Analytics Server** (act-analytics-server)

**As Server:**
- Tools:
  - `improve_coordination` - /improve command handler
  - `analyze_agent_performance` - Performance reports
  - `detect_patterns` - Identify coordination patterns
  - `generate_recommendations` - Suggest improvements

- Prompts:
  - `improve-communication` - Communication analysis workflow
  - `improve-assignments` - Assignment optimization workflow
  - `agent-retrospective` - Agent performance review

**As Client:**
- Connects to: PVM, Coordination, FLUX
- Calls PVM: Retrieve coordination history for analysis
- Calls Coordination: Get current assignments for context
- Calls FLUX: Get evaluation results for outcomes
- Sampling: Uses LLM to analyze patterns and generate insights

**Purpose:**
/improve command. Performance analysis. Recommendations.

---

#### 5. **ACT FLUX Server** (act-flux-server)

**As Server:**
- Tools:
  - `evaluate_task_outcome` - FLUX State reasoning
  - `unbiased_analysis` - Memory-wiped evaluation
  - `identify_gaps` - Find areas for improvement

- Resources:
  - `act://flux/evaluations/{task_id}` - Evaluation results

**As Client:**
- Connects to: PVM (for storing evaluations)
- Calls PVM: Store evaluation results for future reference
- Sampling: Uses LLM with memory wipe for unbiased analysis

**Purpose:**
FLUX State reasoning. Unbiased post-task evaluation.

**How FLUX Works:**
1. Receives: Original task + success criteria + work output
2. LLM does NOT know it created the output (memory wipe)
3. Evaluates: Does this meet criteria? (unbiased)
4. Stores: Evaluation in PVM for learning

---

#### 6. **ACT PAIR Server** (act-pair-server)

**As Server:**
- Tools:
  - `retrieve_relevant_context` - PAIR retrieval
  - `inject_past_patterns` - RAG context injection
  - `validate_approach` - Check against past successes

**As Client:**
- Connects to: PVM (for semantic retrieval)
- Calls PVM: Find relevant past coordination patterns
- Sampling: Uses LLM to synthesize retrieved context

**Purpose:**
PAIR (Past Archived Information Re-injection). Context-guided improvement.

**How PAIR Works:**
1. FLUX identifies gap in task outcome
2. PAIR retrieves similar past coordination patterns from PVM
3. LLM synthesizes: "Here's how we handled this before"
4. Validates or improves original approach
5. Loops until 95%+ success criteria achieved

---

## MCP Capabilities in Detail

### 1. Sampling (LLM-Powered Reasoning)

**Use Case: Intelligent Task Assignment**

```typescript
// In act-coordination-server (as client to Bridge's sampling)

async function assignTaskIntelligently(task) {
  // Request LLM reasoning via sampling
  const decision = await bridgeClient.sampling.createMessage({
    messages: [{
      role: "user",
      content: {
        type: "text",
        text: `Given this task: "${task.description}"
               And these agents: ${JSON.stringify(availableAgents)}
               Who should handle this task and why?`
      }
    }],
    tools: [
      {
        name: "search_pvm",
        description: "Search past similar task assignments",
        inputSchema: {
          type: "object",
          properties: {
            query: { type: "string" }
          }
        }
      },
      {
        name: "get_agent_capabilities",
        description: "Get detailed agent capabilities",
        inputSchema: {
          type: "object",
          properties: {
            agentId: { type: "string" }
          }
        }
      }
    ],
    systemPrompt: "You are ACT's coordination intelligence. Use PVM to make evidence-based decisions. Consider past success rates.",
    maxTokens: 1000,
    modelPreferences: {
      intelligencePriority: 0.9, // High intelligence for reasoning
      speedPriority: 0.3,
      costPriority: 0.2
    }
  });

  // LLM might call tools multiple times:
  // 1. search_pvm("react component tasks") → finds 3 past tasks
  // 2. get_agent_capabilities("agent_x") → gets detailed skills
  // 3. Reasons: "Agent X has 95% success on React, assign to them"

  return parseAssignmentDecision(decision);
}
```

**Result:** Evidence-based assignments powered by LLM reasoning over PVM data.

---

### 2. Resources (Passive Context)

**Resource Pattern: PVM Context Exposure**

```typescript
// In act-pvm-server

pvmServer.setRequestHandler(ListResourcesRequestSchema, async () => {
  return {
    resources: [
      {
        uri: "act://pvm/recent/100",
        name: "Recent Coordination Events",
        description: "Last 100 coordination events with outcomes",
        mimeType: "application/json"
      },
      {
        uri: "act://pvm/patterns/successful",
        name: "Successful Coordination Patterns",
        description: "Patterns that led to task success",
        mimeType: "application/json"
      }
    ]
  };
});

pvmServer.setRequestHandler(ListResourceTemplatesRequestSchema, async () => {
  return {
    resourceTemplates: [
      {
        uriTemplate: "act://pvm/agent/{agentId}/history",
        name: "Agent Coordination History",
        description: "Full coordination history for specific agent",
        mimeType: "application/json"
      },
      {
        uriTemplate: "act://pvm/search/{query}",
        name: "PVM Semantic Search",
        description: "Dynamic semantic search over coordination memory",
        mimeType: "application/json"
      }
    ]
  };
});

pvmServer.setRequestHandler(ReadResourceRequestSchema, async (request) => {
  const uri = request.params.uri;

  if (uri === "act://pvm/recent/100") {
    const events = await chronologicalLog.getRecent(100);
    return {
      contents: [{
        uri: uri,
        mimeType: "application/json",
        text: JSON.stringify(events, null, 2)
      }]
    };
  }

  if (uri.startsWith("act://pvm/agent/")) {
    const agentId = uri.split("/")[3];
    const history = await chronologicalLog.query({ agent: agentId });
    return {
      contents: [{
        uri: uri,
        mimeType: "application/json",
        text: JSON.stringify(history.events, null, 2)
      }]
    };
  }

  if (uri.startsWith("act://pvm/search/")) {
    const query = decodeURIComponent(uri.split("/")[3]);
    const results = await vectorStore.search(query, 10);
    return {
      contents: [{
        uri: uri,
        mimeType: "application/json",
        text: JSON.stringify(results, null, 2)
      }]
    };
  }
});
```

**Agents can now:**
- Browse recent coordination: `act://pvm/recent/100`
- Query agent history: `act://pvm/agent/claude_code_1/history`
- Dynamic search: `act://pvm/search/react%20component%20tasks`

---

### 3. Prompts (Structured Workflows)

**Prompt Pattern: Project Creation Workflow**

```typescript
// In act-mcp-bridge

bridgeServer.setRequestHandler(ListPromptsRequestSchema, async () => {
  return {
    prompts: [
      {
        name: "create-coordinated-project",
        title: "Create Coordinated Project",
        description: "Decompose project into tasks and delegate to agents",
        arguments: [
          {
            name: "project_description",
            description: "Natural language project description",
            required: true
          },
          {
            name: "project_path",
            description: "Local filesystem path for project",
            required: true
          },
          {
            name: "default_agent",
            description: "Agent to use for decomposition",
            required: false
          }
        ]
      },
      {
        name: "improve-coordination",
        title: "Improve Coordination",
        description: "Analyze and improve coordination patterns",
        arguments: [
          {
            name: "scope",
            description: "communication|tools|assignments|conflicts",
            required: true
          },
          {
            name: "agents",
            description: "Comma-separated agent IDs to analyze",
            required: false
          },
          {
            name: "timeframe",
            description: "Time range (last_day|last_week|all)",
            required: false
          }
        ]
      }
    ]
  };
});

bridgeServer.setRequestHandler(GetPromptRequestSchema, async (request) => {
  if (request.params.name === "create-coordinated-project") {
    const { project_description, project_path, default_agent } = request.params.arguments;

    return {
      messages: [
        {
          role: "user",
          content: {
            type: "text",
            text: `You are ACT's project decomposition intelligence.

Project: ${project_description}
Path: ${project_path}
Default Agent: ${default_agent || "auto-select"}

1. Use the 'query_pvm' tool to find similar past projects
2. Decompose this project into concrete tasks
3. Use the 'create_task' tool for each task
4. Use the 'assign_task' tool to delegate intelligently
5. Monitor progress and coordinate agents

Provide a clear plan with reasoning for each decision.`
          }
        }
      ]
    };
  }

  if (request.params.name === "improve-coordination") {
    const { scope, agents, timeframe } = request.params.arguments;

    return {
      messages: [
        {
          role: "user",
          content: {
            type: "text",
            text: `You are ACT's coordination improvement analyst.

Analyze: ${scope}
${agents ? `Focus on agents: ${agents}` : "All agents"}
Timeframe: ${timeframe || "all"}

1. Use 'query_pvm' to retrieve coordination history
2. Use 'analyze_patterns' to identify issues
3. Use 'generate_recommendations' for specific improvements
4. Provide actionable recommendations with examples

Be specific and evidence-based.`
          }
        }
      ]
    };
  }
});
```

**Users invoke:** `/create-project "Todo App" ./my-project`
**Result:** Guided workflow with LLM using ACT tools to decompose and delegate

---

### 4. Logging (Narrative Output)

**Logging Pattern: REPL Narrative**

```typescript
// In act-coordination-server

coordinationServer.setCapabilities({
  logging: {}
});

async function assignTask(taskId, agentId) {
  // Info level: Normal operations
  await coordinationServer.sendLog({
    level: "info",
    logger: "coordination",
    data: {
      message: `Assigning task ${taskId} to agent ${agentId}`,
      taskId,
      agentId
    }
  });

  // Query PVM
  await coordinationServer.sendLog({
    level: "debug",
    logger: "coordination",
    data: {
      message: `Querying PVM for similar tasks to: ${task.description}`,
      query: task.description
    }
  });

  const pvmResults = await pvmClient.callTool({
    name: "search_pvm",
    arguments: { query: task.description, limit: 5 }
  });

  if (pvmResults.length > 0) {
    // Notice level: Significant events
    await coordinationServer.sendLog({
      level: "notice",
      logger: "coordination.pvm",
      data: {
        message: `PVM similarity match: Task similar to ${pvmResults[0].taskId} (${pvmResults[0].similarity}% match)`,
        similarTask: pvmResults[0],
        successRate: pvmResults[0].agent.successRate
      }
    });
  }

  const assignment = await performAssignment(taskId, agentId, pvmResults);

  // Info level: Assignment complete
  await coordinationServer.sendLog({
    level: "info",
    logger: "coordination",
    data: {
      message: `Task ${taskId} assigned to ${agentId} (reasoning: ${assignment.reasoning})`,
      assignment
    }
  });

  return assignment;
}
```

**REPL Output:**
```
[info][coordination] Assigning task task-123 to agent claude_code_1
[debug][coordination] Querying PVM for similar tasks to: "Build React component"
[notice][coordination.pvm] PVM similarity match: Task similar to task-045 (98% match)
[info][coordination] Task task-123 assigned to claude_code_1 (reasoning: PVM similarity match, 94% success rate)
```

**Result:** Real-time narrative showing WHY decisions are made

---

### 5. Pagination (Large Result Sets)

**Pagination Pattern: PVM Search Results**

```typescript
// In act-pvm-server

pvmServer.setRequestHandler(CallToolRequestSchema, async (request) => {
  if (request.params.name === "search_pvm") {
    const { query, limit = 10, cursor } = request.params.arguments;

    // Parse cursor (base64 encoded offset)
    const offset = cursor ? parseInt(Buffer.from(cursor, 'base64').toString()) : 0;

    const allResults = await vectorStore.search(query, offset + limit + 1);
    const pageResults = allResults.slice(offset, offset + limit);
    const hasMore = allResults.length > offset + limit;

    // Generate next cursor
    const nextCursor = hasMore
      ? Buffer.from((offset + limit).toString()).toString('base64')
      : undefined;

    return {
      content: [{
        type: "text",
        text: JSON.stringify(pageResults, null, 2)
      }],
      isError: false,
      _meta: {
        nextCursor,
        totalResults: allResults.length,
        page: Math.floor(offset / limit) + 1
      }
    };
  }
});
```

**Usage:**
```typescript
// First page
const page1 = await callTool("search_pvm", { query: "react tasks", limit: 10 });
// { results: [...10 items...], nextCursor: "MTA=" }

// Second page
const page2 = await callTool("search_pvm", {
  query: "react tasks",
  limit: 10,
  cursor: "MTA="
});
// { results: [...10 items...], nextCursor: "MjA=" }
```

---

## Deployment Scenarios

### Scenario 1: Local Development

```
┌─────────────────────────────────────────┐
│  Developer Machine                       │
│                                          │
│  All services running on localhost:      │
│  ├─ ACT Bridge:        :39300           │
│  ├─ PVM Server:        :39301           │
│  ├─ Coordination:      :39302           │
│  ├─ Analytics:         :39303           │
│  ├─ FLUX:              :39304           │
│  └─ PAIR:              :39305           │
│                                          │
│  Claude Desktop connects to Bridge only  │
└─────────────────────────────────────────┘
```

**Benefits:**
- Simple setup
- Easy debugging
- No network latency

---

### Scenario 2: Team Development

```
┌──────────────────┐     ┌──────────────────┐
│  Developer 1     │     │  Developer 2     │
│  (Claude Desktop)│     │  (Cursor)        │
└────────┬─────────┘     └────────┬─────────┘
         │                        │
         └────────┬───────────────┘
                  │
         ┌────────▼──────────┐
         │  Shared ACT Server │
         │  (Team Instance)   │
         │                    │
         │  All services      │
         │  on team server:   │
         │  Bridge: :80       │
         │  PVM: :39301       │
         │  etc...            │
         └────────────────────┘
```

**Benefits:**
- Shared coordination memory
- Team collaboration
- Consistent PVM across developers

---

### Scenario 3: Production (Distributed)

```
┌────────────────────────────────────────────────┐
│  Load Balancer (Bridge Endpoints)              │
│  https://act.example.com                       │
└────────┬───────────────────────────────────────┘
         │
    ┌────┴────┬─────────┬──────────┬──────────┐
    │         │         │          │          │
┌───▼────┐┌──▼────┐┌───▼─────┐┌───▼────┐┌───▼────┐
│Bridge 1││Bridge2││Bridge 3 ││Bridge4 ││Bridge5 │
│(Edge)  ││(Edge) ││(Edge)   ││(Edge)  ││(Edge)  │
└───┬────┘└──┬────┘└───┬─────┘└───┬────┘└───┬────┘
    │        │          │          │          │
    └────────┴──────────┴──────────┴──────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
    ┌────▼────┐  ┌─────▼─────┐  ┌───▼────┐
    │PVM      │  │Coordination│  │Analytics│
    │(GPU)    │  │(Low-Latency│  │(High-  │
    │Cluster  │  │ Cache)     │  │Memory) │
    │         │  │            │  │        │
    │Qdrant   │  │Redis       │  │        │
    │Embeddings│  │Task Queue │  │        │
    └─────────┘  └────────────┘  └────────┘
```

**Benefits:**
- High availability
- Specialized hardware for each service
- Independent scaling
- PVM on GPU for fast embeddings
- Coordination on low-latency cache
- Analytics on high-memory nodes

---

## Implementation Patterns

### Pattern 1: Service Bootstrap

```typescript
// Universal service bootstrap pattern

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';

async function createActService(config) {
  // Create server (expose this service's capabilities)
  const server = new Server(
    { name: config.name, version: '1.0.0' },
    {
      capabilities: {
        tools: { listChanged: true },
        resources: { subscribe: true, listChanged: true },
        prompts: { listChanged: true },
        logging: {}
      }
    }
  );

  // Register this service's tools
  server.setRequestHandler(CallToolRequestSchema, config.toolHandler);
  server.setRequestHandler(ReadResourceRequestSchema, config.resourceHandler);
  server.setRequestHandler(GetPromptRequestSchema, config.promptHandler);

  // Create clients for other services
  const clients = {};
  for (const [serviceName, transport] of Object.entries(config.dependencies)) {
    const client = new Client(
      { name: `${config.name}-to-${serviceName}`, version: '1.0.0' },
      { capabilities: { sampling: { tools: {} } } }
    );
    await client.connect(transport);
    clients[serviceName] = client;
  }

  // Start server
  const serverTransport = new StdioServerTransport();
  await server.connect(serverTransport);

  return { server, clients };
}

// Usage in each service
const pvmService = await createActService({
  name: 'act-pvm',
  toolHandler: handlePVMTools,
  resourceHandler: handlePVMResources,
  promptHandler: handlePVMPrompts,
  dependencies: {
    coordination: coordinationTransport
  }
});
```

---

### Pattern 2: Cross-Service Tool Call

```typescript
// Standard pattern for calling another service

async function crossServiceCall(client, toolName, args) {
  try {
    const result = await client.callTool({
      name: toolName,
      arguments: args
    });

    if (result.isError) {
      throw new Error(`Tool call failed: ${result.content[0].text}`);
    }

    return JSON.parse(result.content[0].text);
  } catch (error) {
    logger.error(`Cross-service call failed: ${toolName}`, error);
    throw error;
  }
}

// Usage
const similarTasks = await crossServiceCall(
  pvmClient,
  'search_pvm',
  { query: 'react component', limit: 5 }
);
```

---

### Pattern 3: Sampling for Intelligence

```typescript
// Standard pattern for LLM-powered decisions

async function sampleForDecision(client, prompt, tools) {
  const response = await client.sampling.createMessage({
    messages: [{
      role: "user",
      content: { type: "text", text: prompt }
    }],
    tools: tools,
    systemPrompt: "You are ACT's coordination intelligence. Use tools to make evidence-based decisions.",
    maxTokens: 2000,
    modelPreferences: {
      intelligencePriority: 0.9,
      speedPriority: 0.3
    }
  });

  // Handle multi-turn if LLM called tools
  while (response.stopReason === "toolUse") {
    const toolResults = await executeToolCalls(response.content);
    response = await client.sampling.createMessage({
      messages: [
        ...previousMessages,
        { role: "assistant", content: response.content },
        { role: "user", content: toolResults }
      ],
      tools: tools,
      maxTokens: 2000
    });
  }

  return response.content.text;
}
```

---

## Why This Architecture Wins

### 1. **Modularity Without Monolith**
- Each service focused on one domain
- Independent development and testing
- Clear boundaries and contracts
- No "god object" coordination

### 2. **Framework-Agnostic at Every Layer**
- External agents: MCP or SDK
- Internal services: MCP everywhere
- No vendor lock-in
- Standard protocols throughout

### 3. **Scalability by Design**
- PVM compute-heavy? Scale it independently
- Coordination needs low latency? Optimize separately
- Analytics needs memory? Dedicated hardware
- Services scale horizontally

### 4. **Testability**
```typescript
// Test coordination without PVM
const mockPVMClient = {
  callTool: async () => ({ results: mockResults })
};

const coordinator = new CoordinationService(mockPVMClient);
await coordinator.assignTask(task); // Fully isolated test
```

### 5. **Evolvability**
Add new capability = add new service. Bridge automatically orchestrates.

```
Today: Bridge → [PVM, Coordination, Analytics, FLUX, PAIR]
Tomorrow: Bridge → [PVM, Coordination, Analytics, FLUX, PAIR, RealtimeUI, Metrics, Telemetry]
```

### 6. **Developer Experience**
**Beginners:** 7-line MCP config
**Power Users:** Full SDK control
**Both:** Same ACT intelligence

### 7. **Production-Ready from Day 1**
- Logging: Built-in with MCP
- Error handling: MCP tool result patterns
- Pagination: Large datasets handled
- Sampling: LLM reasoning integrated
- Resources: Context exposure standardized

### 8. **The Ultimate Goal: Ubiquity**
```
Every AI course starts with:
"Step 1: Add ACT (7 lines)"

Result: ACT becomes the assumed baseline.
Everyone learns agent development WITH coordination from day 1.
```

---

## Conclusion

**ACT is not "many MCPs."**

**ACT is a coordination infrastructure** that happens to use MCP as:
1. Internal communication protocol (implementation detail)
2. External distribution option (user choice)

**The architecture:**
- Modular services (server + client duality)
- MCP everywhere (consistent protocol)
- Two routes to access (MCP or SDK)
- Production-ready patterns (sampling, resources, prompts, logging, pagination)

**The vision:**
- Suspiciously simple integration (7 lines)
- Drastic success without HITL (PVM/FLUX/PAIR intelligence)
- Becomes a staple (courses, frameworks, production)
- ACT: The assumed baseline for agent coordination

**Where servers are clients, clients are servers, and everyone serves everyone else in perfect harmony.**

---

**End of MCP_ARCHITECTURE.md**

*For implementation details, see `/mcp-servers/` directory.*
*For SDK documentation, see `/sdk/` directory.*
*For examples, see `/examples/` directory.*

---

## Appendix: The Twitter Thread

> ACT, working through the land of MCPs, serves its own server as a client by connecting the client server as a server and client that the server the agents connect to becomes a client for its own other servers—where clients are served and servers are clients to client servers, providing a service for each client that is a server-client serving client-server clients by a server to serve the other servers' client-servers and server-clients of clients serving the server which is a client to the client server.
>
> Translation: It's turtles all the way down, but also turtles all the way up, and all the turtles are talking to each other via MCP. 🐢
>
> This is how ACT achieves coordination intelligence at scale.
>
> #BuildInPublic #ACT #MCP

*(Please don't shadow ban us)*
