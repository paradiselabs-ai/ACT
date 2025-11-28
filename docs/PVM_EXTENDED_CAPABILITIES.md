# PVM Extended Capabilities: Individual Memory + /improve Command

**Date:** November 22, 2025
**Status:** Critical Design Addition to PVM Architecture
**Purpose:** Document newly discovered PVM capabilities that replace need for separate personal memory systems

---

## Table of Contents

1. [Individual Agent Memory Derivation](#individual-agent-memory-derivation)
2. [User /improve Command: Surgical Precision](#user-improve-command-surgical-precision)
3. [Implementation Specifications](#implementation-specifications)
4. [Integration with ACTMEMORYSYSTEM_IMPLEMENTATION.md](#integration-with-actmemorysystem_implementationmd)

---

## Individual Agent Memory Derivation

### Key Discovery

**PVM provides robust individual agent memory as a natural byproduct of coordination memory.**

This means ACT + PVM can replace BOTH:
- Multi-agent coordination frameworks (CrewAI, LangChain, Autogen)
- Personal memory systems (Mem0, OpenMemory)

With a single unified semantic memory architecture.

### Why PVM's Individual Memory is Better

**Comparison:**

| Aspect | OpenMemory/Mem0 Approach | PVM Approach |
|--------|-------------------------|--------------|
| **Source** | Explicitly stored preferences | Derived from coordination outcomes |
| **Evidence** | Self-reported ("I prefer X") | Proven by results ("X succeeds 94% of time") |
| **Currency** | Can become stale | Always current (based on recent performance) |
| **Context** | Generic preference | Context-aware (knows WHEN preference applies) |
| **Maintenance** | Manual memory management | Automatic derivation |
| **Accuracy** | Subject to agent bias | Evidence-based, objective |

**Example:**

OpenMemory:
```typescript
await memory.add("I prefer React hooks over class components", "agent1");
```

PVM:
```typescript
const profile = await pvm.getAgentProfile("agent1");
// Returns: {
//   coding_patterns: {
//     react_hooks: { success_rate: 0.94, tasks_completed: 47 },
//     class_components: { success_rate: 0.71, tasks_completed: 8 }
//   },
//   recommendation: "React hooks (94% success vs 71% for classes)"
// }
```

---

## What PVM Tracks Per-Agent (Automatically)

### 1. Historical Performance Patterns

```typescript
interface AgentPerformanceProfile {
  agent_id: string;

  // Overall metrics
  task_success_rate: number;           // 0.94
  average_completion_time: string;     // "2.3 hours"
  total_tasks_completed: number;       // 127

  // Specializations (evidence-based)
  specializations: Array<{
    skill: string;                     // "React", "TypeScript", "CSS"
    confidence: number;                // 0.96
    tasks_completed: number;           // 47
    success_rate: number;              // 0.94
    average_time: string;              // "2.1 hours"
  }>;

  // Performance by task type
  task_type_performance: Map<string, {
    success_rate: number;
    count: number;
    avg_time: string;
  }>;
}
```

### 2. Skill Evolution & Learning Trajectory

```typescript
interface SkillProgression {
  skill: string;
  trajectory: Array<{
    period: string;                    // "2025-10", "2025-11"
    confidence: number;                // 0.65, 0.82, 0.94
    tasks_completed: number;
    success_rate: number;
  }>;

  learning_rate: number;               // How fast agent improves
  plateau_detected: boolean;           // Has skill maxed out?
  recommended_next_challenge: string;  // "Try advanced React patterns"
}

// Example:
{
  skill: "React Hooks",
  trajectory: [
    { period: "2025-09", confidence: 0.65, success_rate: 0.71 },
    { period: "2025-10", confidence: 0.82, success_rate: 0.87 },
    { period: "2025-11", confidence: 0.94, success_rate: 0.94 }
  ],
  learning_rate: 0.145,  // Strong improvement
  plateau_detected: false,
  recommended_next_challenge: "React Server Components"
}
```

### 3. Communication Patterns & Personal Style

```typescript
interface CommunicationProfile {
  agent_id: string;

  // Help-seeking behavior
  asks_for_help_frequency: "low" | "moderate" | "high";
  typical_help_topics: string[];       // ["backend_apis", "testing"]
  preferred_helpers: Array<{
    agent_id: string;
    topic: string;
    success_rate: number;
  }>;

  // Help-giving behavior
  provides_help_frequency: "low" | "moderate" | "high";
  expertise_areas_offered: string[];   // ["react", "typescript"]
  help_success_rate: number;           // 0.91

  // Collaboration style
  preferred_collaboration_mode: "continuous_dialogue" | "sequential_handoff" | "announce_only";
  response_time_avg: string;           // "4.2 minutes"
  message_verbosity: "concise" | "detailed" | "verbose";

  // Critique patterns
  gives_unsolicited_feedback: boolean;
  accepts_critique_gracefully: boolean;
  critique_impact_on_outcomes: number; // 0.23 (23% improvement when critiqued)
}
```

### 4. Tool Usage Effectiveness

```typescript
interface ToolUsageProfile {
  agent_id: string;

  tools: Array<{
    tool_name: string;                 // "Read", "Edit", "Bash"
    usage_count: number;               // 234
    success_rate: number;              // 0.96
    average_retry_count: number;       // 1.2

    // Common patterns
    typical_use_cases: string[];       // ["config files", "source code"]
    common_errors: Array<{
      error_type: string;
      frequency: number;
      resolution: string;
    }>;

    // Effectiveness
    time_to_success_avg: string;       // "12 seconds"
    improvement_over_time: number;     // 0.15 (15% improvement)
  }>;

  // Tool selection patterns
  tool_choice_accuracy: number;        // 0.89 (chooses right tool 89% of time)
  learns_from_tool_failures: boolean;
}
```

### 5. Contextual Preferences (When X Works for This Agent)

```typescript
interface ContextualPatterns {
  agent_id: string;

  patterns: Array<{
    pattern: string;
    evidence: string;
    confidence: number;
    applicable_contexts: string[];

    // Example:
    // pattern: "Works best with detailed specifications"
    // evidence: "Success rate: 0.94 with detailed specs vs 0.78 with vague"
    // confidence: 0.89
    // applicable_contexts: ["frontend_tasks", "complex_features"]
  }>;

  // Task complexity preferences
  thrives_at_complexity: "low" | "medium" | "high";
  struggles_when: string[];            // ["tight deadlines", "ambiguous requirements"]
  excels_when: string[];               // ["clear specs", "collaborative environment"]

  // Work environment
  optimal_workload: number;            // 0.7-0.85 (sweet spot)
  performance_under_pressure: number;  // 0.73 (drops when overloaded)
  multitasking_ability: number;        // 0.68
}
```

### 6. Team Dynamics & Collaboration Synergy

```typescript
interface TeamSynergyProfile {
  agent_id: string;

  collaborations: Array<{
    partner_agent_id: string;
    synergy_score: number;             // 0.92

    // What works
    successful_patterns: Array<{
      pattern: string;                 // "Continuous communication during complex tasks"
      evidence: string;                // "15% faster completion, 12% fewer bugs"
      frequency: number;
    }>;

    // What doesn't work
    conflict_patterns: Array<{
      conflict_type: string;           // "Parallel edits on same file"
      frequency: number;
      resolution_time_avg: string;
    }>;

    // Recommendations
    optimal_task_types: string[];      // ["frontend_backend_integration"]
    avoid_task_types: string[];        // ["independent_parallel_work"]

    communication_style_compatibility: number; // 0.87
  }>;
}
```

---

## Implementation: Agent Profile Derivation

### Core Class: AgentProfileBuilder

```typescript
class AgentProfileBuilder {
  constructor(
    private chronologicalLog: ChronologicalLog,
    private vectorStore: VectorMemoryStore
  ) {}

  /**
   * Build complete evidence-based profile for agent
   */
  async buildProfile(agentId: string): Promise<ComprehensiveAgentProfile> {
    // Get all events involving this agent
    const agentEvents = await this.chronologicalLog.getByAgent(agentId);

    return {
      agent_id: agentId,
      performance: await this.buildPerformanceProfile(agentEvents),
      skill_progression: await this.buildSkillProgression(agentEvents),
      communication: await this.buildCommunicationProfile(agentEvents),
      tool_usage: await this.buildToolUsageProfile(agentEvents),
      contextual_patterns: await this.buildContextualPatterns(agentEvents),
      team_synergy: await this.buildTeamSynergyProfile(agentEvents),

      // Meta-analysis
      overall_effectiveness: this.calculateOverallEffectiveness(agentEvents),
      growth_trajectory: this.calculateGrowthTrajectory(agentEvents),
      recommended_improvements: await this.generateRecommendations(agentEvents)
    };
  }

  private async buildPerformanceProfile(
    events: CoordinationMinute[]
  ): Promise<AgentPerformanceProfile> {
    const taskEvents = events.filter(e =>
      e.event_type === 'task_assigned' ||
      e.event_type === 'task_completed'
    );

    // Group by task_id
    const tasks = this.groupByTask(taskEvents);

    // Calculate metrics
    const completed = tasks.filter(t => t.completed);
    const successful = completed.filter(t => t.success_criteria_met >= 80);

    return {
      task_success_rate: successful.length / completed.length,
      average_completion_time: this.averageCompletionTime(completed),
      total_tasks_completed: completed.length,
      specializations: await this.extractSpecializations(completed)
    };
  }

  private async buildSkillProgression(
    events: CoordinationMinute[]
  ): Promise<SkillProgression[]> {
    // Analyze tasks over time to detect skill improvement
    const tasksBySkill = this.groupBySkill(events);

    return Object.entries(tasksBySkill).map(([skill, tasks]) => {
      const trajectory = this.calculateTrajectory(tasks);

      return {
        skill,
        trajectory,
        learning_rate: this.calculateLearningRate(trajectory),
        plateau_detected: this.detectPlateau(trajectory),
        recommended_next_challenge: this.suggestNextChallenge(skill, trajectory)
      };
    });
  }

  private async buildCommunicationProfile(
    events: CoordinationMinute[]
  ): Promise<CommunicationProfile> {
    const messageEvents = events.filter(e => e.event_type === 'agent_message');

    // Analyze message patterns
    const helpRequests = messageEvents.filter(m =>
      this.isHelpRequest(m.data.content)
    );
    const helpProvided = messageEvents.filter(m =>
      this.isHelpResponse(m.data.content)
    );

    return {
      asks_for_help_frequency: this.categorizeFrequency(helpRequests.length, events.length),
      typical_help_topics: this.extractTopics(helpRequests),
      preferred_helpers: await this.identifyPreferredHelpers(helpRequests),

      provides_help_frequency: this.categorizeFrequency(helpProvided.length, events.length),
      expertise_areas_offered: this.extractExpertiseAreas(helpProvided),
      help_success_rate: await this.calculateHelpSuccessRate(helpProvided),

      preferred_collaboration_mode: await this.identifyCollaborationMode(events),
      response_time_avg: this.averageResponseTime(messageEvents),
      message_verbosity: this.analyzeVerbosity(messageEvents)
    };
  }

  private async buildToolUsageProfile(
    events: CoordinationMinute[]
  ): Promise<ToolUsageProfile> {
    const toolEvents = events.filter(e => e.data.tool_used);

    // Group by tool
    const byTool = this.groupByTool(toolEvents);

    return {
      tools: Object.entries(byTool).map(([tool, uses]) => ({
        tool_name: tool,
        usage_count: uses.length,
        success_rate: uses.filter(u => u.success).length / uses.length,
        average_retry_count: this.averageRetries(uses),
        typical_use_cases: this.extractUseCases(uses),
        common_errors: this.identifyCommonErrors(uses),
        time_to_success_avg: this.averageTimeToSuccess(uses),
        improvement_over_time: this.calculateToolImprovement(uses)
      })),

      tool_choice_accuracy: await this.calculateToolChoiceAccuracy(toolEvents),
      learns_from_tool_failures: this.detectLearningFromFailures(toolEvents)
    };
  }
}
```

---

## User /improve Command: Surgical Precision

### Problem Statement

**Automatic background improvement** runs periodically with broad scope:
- Analyzes all recent activity
- Surface-level pattern detection
- Low-priority, throttled processing
- Internal updates only

**User needs surgical precision** when:
- Debugging specific issues
- Analyzing particular agent interactions
- Investigating tool usage problems
- Comparing collaboration approaches
- Post-mortem on failed sessions

### Solution: Parameterized /improve Command

```bash
/improve <scope> [--flags] [--filters] [--output]
```

### Command Structure

#### 1. Scope Parameter (Required)

Defines what to analyze:

```bash
/improve communication    # Agent-to-agent communication effectiveness
/improve tools           # Tool usage patterns and effectiveness
/improve assignments     # Task assignment decisions
/improve conflicts       # Conflict detection and resolution
/improve decomposition   # Task breakdown strategies
/improve collaboration   # Team collaboration patterns
/improve performance     # Overall team performance metrics
```

#### 2. Agent Filters

Control which agents to analyze:

```bash
--agents agent1,agent2,agent3   # Specific agents
--agents all                    # All agents (default)
--exclude agent4                # All except specified
```

#### 3. Session/Timeframe Filters

Control which coordination data to analyze:

```bash
--session <session_id>     # Specific session
--session last             # Most recent session
--timeframe last-week      # Last 7 days
--timeframe last-month     # Last 30 days
--timeframe all-time       # Entire history
--project <project_name>   # Specific project only
```

#### 4. Quality Filters

Focus on successes, failures, or both:

```bash
--filter good              # Only analyze successful outcomes
--filter bad               # Only analyze failures
--filter all               # Both successes and failures (default)
--success-rate <threshold> # Only show items above/below threshold
```

#### 5. Scope-Specific Flags

##### For `communication` scope:

```bash
--style intentional-conversational  # Continuous dialogue patterns
--style announce-only               # Start/finish announcements only
--style help-requests               # Only when asking for help
--style critique-enabled            # Unsolicited feedback patterns
--style critique-on-request         # Only critiqued when asked
--style no-critique                 # Never critiqued each other
```

##### For `tools` scope:

```bash
-f, --function-calls       # Include function call analysis
--tool-type <name>         # Specific tool (Read, Write, Bash, etc.)
--error-analysis           # Deep dive into tool errors
```

##### For `collaboration` scope:

```bash
--compare sequential,parallel      # Compare collaboration modes
--parallel-effectiveness           # Analyze parallel work patterns
--handoff-quality                  # Analyze task handoff effectiveness
```

#### 6. Output Formats

Control how results are presented:

```bash
--output summary           # Quick overview (default)
--output detailed-report   # Full analysis with examples
--output recommendations   # Just actionable insights
--output json              # Machine-readable format
--output metrics           # Numeric metrics only
```

---

## /improve Command Examples

### Example 1: Debug Communication Breakdown

**Scenario:** Agent 2 and Agent 3 had conflicts during last session

```bash
/improve communication \
  --agents agent2,agent3 \
  --session last \
  --filter bad \
  --style all \
  --output detailed-report
```

**Output:**
```
Communication Analysis: agent2 ↔ agent3 (Session: sess_2025_11_22_001)

Issues Detected:
═════════════════════════════════════════════════════════════════

1. Parallel File Edits Without Coordination (5 conflicts)
   ────────────────────────────────────────────────────────────
   - File: src/components/Header.tsx
   - Agent 2 started edit at 14:23:11
   - Agent 3 started edit at 14:23:45 (34 seconds later)
   - No Task Check Call performed
   - Conflict resolution time: 8 minutes

   Pattern: Both agents working on frontend files without communication

   Recommendation: Enforce Task Check Call protocol for shared files

2. Response Delays to Help Requests (3 instances)
   ────────────────────────────────────────────────────────────
   - Agent 2 asked for backend API guidance at 15:12:33
   - Agent 3 responded at 15:27:19 (14 min 46 sec delay)
   - Average response time: 12.3 minutes (team avg: 4.2 minutes)

   Pattern: Agent 3 doesn't monitor messages while focused on task

   Recommendation: Implement notification system for help requests

3. Communication Style Mismatch
   ────────────────────────────────────────────────────────────
   - Agent 2 prefers continuous dialogue (78% of messages)
   - Agent 3 uses announce-only style (89% of messages)

   Impact: Agent 2 felt unsupported, asked same question 3 times

   Pattern: Style compatibility score: 0.34 (low)

   Recommendation: Establish explicit communication protocols or
                  pair Agent 2 with more communicative partners

Summary:
════════
Success Rate: 67% (below team average of 85%)
Main Issues: Coordination gaps, response delays, style mismatch
Estimated Cost: 2.3 hours lost to conflicts and rework

Actionable Improvements:
1. Enable Task Check Call alerts for Agent 2 & 3
2. Implement help request notifications
3. Consider pairing Agent 2 with agent_frontend instead
```

### Example 2: Tool Usage Optimization

**Scenario:** Want to improve team's tool effectiveness

```bash
/improve tools \
  --function-calls \
  --agents all \
  --filter bad \
  --success-rate <70 \
  --timeframe last-week \
  --output recommendations
```

**Output:**
```
Tool Usage Recommendations (Last 7 Days)

Critical Issues:
═════════════════════════════════════════════════════════════════

Read Tool - 55% Success Rate (Team avg: 89%)
─────────────────────────────────────────────────────────────────
Problem: Agents attempting Read on non-existent paths

Agent Breakdown:
- agent_frontend: 12 failed reads (path not found)
- agent_backend: 8 failed reads (permission denied)
- agent_qa: 5 failed reads (file moved)

Root Cause: No path verification before Read attempts

Fix: Use Glob tool first to verify paths exist

Example Pattern (agent_frontend):
  ❌ Read("/src/components/NewFeature.tsx")  # Fails
  ✅ Glob("**/NewFeature.tsx") then Read()   # Succeeds

Estimated Time Saved: 2.1 hours/week

Bash Tool - 68% Success Rate (Team avg: 87%)
─────────────────────────────────────────────────────────────────
Problem: Complex commands failing due to syntax errors

Agent Breakdown:
- agent_backend: 15 failed Bash calls (pipe syntax)
- agent_devops: 7 failed Bash calls (quoting issues)

Root Cause: Not escaping special characters in file paths

Fix: Always quote paths with spaces
     Use explicit '&&' for command chains

Example Fix:
  ❌ Bash: cd /path with spaces/file.txt
  ✅ Bash: cd "/path with spaces/file.txt"

Estimated Time Saved: 1.4 hours/week

Edit Tool - 73% Success Rate (Could be 95%+)
─────────────────────────────────────────────────────────────────
Problem: old_string not found in file (exact match required)

Agent Pattern: Agents not reading file before editing

Fix: ALWAYS use Read before Edit
     Verify exact string including whitespace/indentation

Estimated Time Saved: 3.2 hours/week

Total Potential Time Savings: 6.7 hours/week
```

### Example 3: Collaboration Mode Comparison

**Scenario:** Determine best collaboration approach for UI tasks

```bash
/improve collaboration \
  --compare continuous,sequential \
  --project "todo-app" \
  --agents agent_frontend,agent_designer \
  --output metrics
```

**Output:**
```
Collaboration Mode Comparison: Continuous vs Sequential

Project: todo-app
Agents: agent_frontend, agent_designer
Tasks Analyzed: 23

Continuous Dialogue Mode (12 tasks):
═════════════════════════════════════════════════════════════════
Average Completion Time: 2.1 hours
Success Rate: 92%
Code Quality Score: 8.7/10
Conflict Count: 1
Rework Required: 8% of tasks
Agent Satisfaction: High (based on message sentiment)

Key Patterns:
- Real-time feedback reduces errors
- Both agents stay aligned on requirements
- Design decisions made collaboratively
- Faster iteration cycles

Sequential Handoff Mode (11 tasks):
═════════════════════════════════════════════════════════════════
Average Completion Time: 2.7 hours
Success Rate: 73%
Code Quality Score: 7.4/10
Conflict Count: 4
Rework Required: 27% of tasks
Agent Satisfaction: Moderate

Key Issues:
- Design intent lost in handoff
- Requirements misinterpretation
- More back-and-forth rework
- Slower overall progress

Recommendation:
═════════════════════════════════════════════════════════════════
For UI/UX tasks: Use CONTINUOUS DIALOGUE mode

Benefits:
- 23% faster completion
- 26% higher success rate
- 17% better code quality
- 75% fewer conflicts
- 19% less rework

Estimated Project-Wide Impact:
- Save 12.4 hours on UI tasks
- Reduce design rework by 45%
- Improve overall quality score by 1.3 points
```

### Example 4: Post-Session Analysis

**Scenario:** Session just ended, want comprehensive evaluation

```bash
/improve performance \
  --session last \
  --agents all \
  --output detailed-report
```

**Output:**
```
Session Performance Report
Session ID: sess_2025_11_22_003
Duration: 4 hours 27 minutes
Agents: agent_frontend, agent_backend, agent_qa, agent_devops

Overall Metrics:
═════════════════════════════════════════════════════════════════
Tasks Completed: 12/14 (86%)
Success Rate: 83% (team average: 85%)
Total Coordination Events: 247
Average Response Time: 5.1 minutes
Conflicts Detected: 3
Conflicts Resolved: 3/3 (100%)

Agent Performance:
═════════════════════════════════════════════════════════════════
agent_frontend:
  - Tasks: 4/4 completed (100%)
  - Success rate: 100%
  - Avg time: 1.8 hours
  - ⭐ Star performer this session

agent_backend:
  - Tasks: 3/4 completed (75%)
  - Success rate: 67% (below personal avg of 89%)
  - Avg time: 3.2 hours (33% slower than usual)
  - ⚠️ Struggled with API integration task

agent_qa:
  - Tasks: 3/3 completed (100%)
  - Success rate: 100%
  - Avg time: 1.1 hours
  - ✅ Consistently reliable

agent_devops:
  - Tasks: 2/3 completed (67%)
  - Success rate: 50% (deployment failed)
  - Avg time: 2.4 hours
  - ⚠️ Docker configuration issues

Communication Analysis:
═════════════════════════════════════════════════════════════════
Total Messages: 84
Help Requests: 7
Help Provided: 7 (100% response rate ✅)
Average Response Time: 5.1 minutes

Most Effective Collaboration:
- agent_frontend ↔ agent_backend (synergy: 0.92)
- Continuous communication during API integration
- 15% faster than when working separately

Least Effective Collaboration:
- agent_backend ↔ agent_devops (synergy: 0.61)
- Handoff confusion on deployment requirements
- 2 conflicts requiring resolution

Tool Usage:
═════════════════════════════════════════════════════════════════
Read: 89 uses, 94% success ✅
Edit: 67 uses, 88% success ⚠️ (12% failed exact match)
Bash: 41 uses, 76% success ⚠️ (24% command errors)
Write: 23 uses, 100% success ✅

Critical Issues:
═════════════════════════════════════════════════════════════════
1. agent_backend: API integration task took 3.2 hours (expected: 2.0)
   - Root cause: Unclear requirements
   - Asked for clarification 3 times
   - Recommendation: Provide detailed specs upfront

2. agent_devops: Docker deployment failed
   - Root cause: Missing environment variables
   - No documentation referenced
   - Recommendation: Create deployment checklist

Recommendations:
═════════════════════════════════════════════════════════════════
1. Provide clearer API specs to agent_backend
2. Create deployment checklist for agent_devops
3. Improve Edit tool usage (read files first)
4. Reduce Bash command errors (proper quoting)
5. Maintain agent_frontend ↔ agent_backend pairing (highly effective)

Session Grade: B+ (83%)
Improvement Potential: +12% with recommended changes
```

---

## Implementation Specifications

### /improve Command Handler

```typescript
interface ImproveParams {
  // Required
  scope: 'communication' | 'tools' | 'assignments' | 'conflicts' |
         'decomposition' | 'collaboration' | 'performance';

  // Agent filters
  agents?: string[] | 'all';
  exclude?: string[];

  // Session/timeframe filters
  session?: string | 'last';
  timeframe?: 'last-week' | 'last-month' | 'all-time';
  project?: string;

  // Quality filters
  filter?: 'good' | 'bad' | 'all';
  successRate?: number;

  // Scope-specific
  style?: string[];           // For communication
  toolType?: string;          // For tools
  functionCalls?: boolean;    // For tools
  compare?: string[];         // For collaboration

  // Output
  output?: 'summary' | 'detailed-report' | 'recommendations' | 'json' | 'metrics';
}

class ImproveCommandHandler {
  constructor(
    private pvm: ACTMemorySystem,
    private agentProfileBuilder: AgentProfileBuilder
  ) {}

  async execute(params: ImproveParams): Promise<ImproveReport> {
    // 1. Filter events based on parameters
    const relevantEvents = await this.filterEvents(params);

    // 2. Route to scope-specific analyzer
    const analysis = await this.analyzeByScope(params.scope, relevantEvents, params);

    // 3. Extract insights
    const insights = await this.extractInsights(analysis, params);

    // 4. Generate recommendations
    const recommendations = await this.generateRecommendations(insights, params);

    // 5. Format output
    return this.formatOutput(insights, recommendations, params.output);
  }

  private async filterEvents(params: ImproveParams): Promise<CoordinationMinute[]> {
    let events = await this.pvm.chronologicalLog.getAllHistory();

    // Agent filter
    if (params.agents && params.agents !== 'all') {
      events = events.filter(e => params.agents.includes(e.agent_id));
    }
    if (params.exclude) {
      events = events.filter(e => !params.exclude.includes(e.agent_id));
    }

    // Session filter
    if (params.session === 'last') {
      const lastSession = await this.getLastSession();
      events = events.filter(e => e.session_id === lastSession.id);
    } else if (params.session) {
      events = events.filter(e => e.session_id === params.session);
    }

    // Timeframe filter
    if (params.timeframe) {
      const cutoff = this.getTimeframeCutoff(params.timeframe);
      events = events.filter(e => new Date(e.timestamp) >= cutoff);
    }

    // Project filter
    if (params.project) {
      events = events.filter(e => e.project_id === params.project);
    }

    // Quality filter
    if (params.filter === 'good') {
      events = events.filter(e => e.data.success_criteria_met >= 80);
    } else if (params.filter === 'bad') {
      events = events.filter(e => e.data.success_criteria_met < 80);
    }

    return events;
  }

  private async analyzeByScope(
    scope: string,
    events: CoordinationMinute[],
    params: ImproveParams
  ): Promise<ScopeAnalysis> {
    switch (scope) {
      case 'communication':
        return await this.analyzeCommunication(events, params);
      case 'tools':
        return await this.analyzeTools(events, params);
      case 'collaboration':
        return await this.analyzeCollaboration(events, params);
      case 'performance':
        return await this.analyzePerformance(events, params);
      // ... other scopes
    }
  }

  private async analyzeCommunication(
    events: CoordinationMinute[],
    params: ImproveParams
  ): Promise<CommunicationAnalysis> {
    const messages = events.filter(e => e.event_type === 'agent_message');

    // Analyze communication patterns
    const patterns = {
      response_times: this.analyzeResponseTimes(messages),
      help_requests: this.analyzeHelpRequests(messages),
      collaboration_modes: this.analyzeCollaborationModes(messages, params.style),
      conflicts: this.analyzeConflicts(events),
      synergy_scores: await this.calculateSynergyScores(messages)
    };

    return {
      total_messages: messages.length,
      patterns,
      issues: this.identifyIssues(patterns),
      strengths: this.identifyStrengths(patterns)
    };
  }

  private async analyzeTools(
    events: CoordinationMinute[],
    params: ImproveParams
  ): Promise<ToolAnalysis> {
    const toolEvents = events.filter(e => e.data.tool_used);

    // Filter by tool type if specified
    const filtered = params.toolType
      ? toolEvents.filter(e => e.data.tool_used === params.toolType)
      : toolEvents;

    // Group by tool
    const byTool = this.groupByTool(filtered);

    return {
      tools: Object.entries(byTool).map(([tool, uses]) => ({
        tool_name: tool,
        usage_count: uses.length,
        success_rate: this.calculateSuccessRate(uses),
        common_errors: this.identifyCommonErrors(uses),
        improvement_suggestions: this.generateToolSuggestions(tool, uses)
      })),

      overall_effectiveness: this.calculateOverallToolEffectiveness(byTool),
      time_waste: this.calculateTimeWastedOnFailures(byTool)
    };
  }
}
```

---

## Integration with Existing PVM Architecture

### Where This Fits

1. **ACTMemorySystem** - Extended with AgentProfileBuilder
2. **ChronologicalLog** - Already records all events needed
3. **VectorMemoryStore** - Powers semantic search for agent profile queries
4. **New Component**: `ImproveCommandHandler` - Handles /improve command
5. **New Component**: `AgentProfileBuilder` - Derives individual agent profiles

### Updated ACTMemorySystem Class

```typescript
class ACTMemorySystem {
  // Existing components
  private chronologicalLog: ChronologicalLog;
  private vectorStore: VectorMemoryStore;
  private memoryIndex: MemoryIndex;
  private contextWindowManager: ContextWindowManager;
  private memoryEvaluator: MemoryEvaluator;
  private pairReasoningEngine: PAIRReasoningEngine;

  // NEW: Individual agent memory
  private agentProfileBuilder: AgentProfileBuilder;

  // NEW: User command handler
  private improveCommandHandler: ImproveCommandHandler;

  /**
   * NEW: Get individual agent profile (evidence-based)
   */
  async getAgentProfile(agentId: string): Promise<ComprehensiveAgentProfile> {
    return await this.agentProfileBuilder.buildProfile(agentId);
  }

  /**
   * NEW: Execute user /improve command
   */
  async executeImproveCommand(params: ImproveParams): Promise<ImproveReport> {
    return await this.improveCommandHandler.execute(params);
  }

  /**
   * NEW: Get agent-to-agent synergy analysis
   */
  async getAgentSynergy(agent1: string, agent2: string): Promise<SynergyAnalysis> {
    const events = await this.chronologicalLog.getByAgents([agent1, agent2]);
    return await this.agentProfileBuilder.analyzeSynergy(events, agent1, agent2);
  }

  /**
   * NEW: Compare agent performance on similar tasks
   */
  async compareAgents(
    agents: string[],
    taskType?: string
  ): Promise<AgentComparison> {
    const profiles = await Promise.all(
      agents.map(id => this.getAgentProfile(id))
    );

    return this.agentProfileBuilder.compareProfiles(profiles, taskType);
  }
}
```

---

## Success Criteria

### For Individual Agent Memory

- ✅ Derive agent profiles from coordination history only
- ✅ Evidence-based insights (proven by outcomes, not self-reported)
- ✅ Automatically updated (no manual memory management)
- ✅ Context-aware (knows when patterns apply)
- ✅ Comparable or better than explicit memory systems (Mem0/OpenMemory)

### For /improve Command

- ✅ Surgical precision (user controls exact scope and filters)
- ✅ Actionable insights (specific recommendations, not vague)
- ✅ Fast execution (responds within 5 seconds for typical queries)
- ✅ Multiple output formats (summary, detailed, recommendations, JSON, metrics)
- ✅ Complements automatic improvement (different purpose, not redundant)

---

## Next Steps for Implementation

1. **Update ACTMEMORYSYSTEM_IMPLEMENTATION.md** to reference this document
2. **Implement AgentProfileBuilder class** (Task 1.6 or new task)
3. **Implement ImproveCommandHandler class** (Task 2.5 Self-Improvement Engine)
4. **Create API endpoint** `POST /improve` (accepts ImproveParams)
5. **Test with real coordination data** from Phase 4 sessions
6. **Document in user-facing docs** how to use /improve command

---

## Conclusion

PVM's extended capabilities transform ACT from a coordination framework into a **comprehensive agent intelligence platform**:

1. **Team Coordination Memory** (original PVM)
2. **Individual Agent Memory** (derived automatically)
3. **User-Controlled Analysis** (/improve command)

This makes ACT competitive with or superior to:
- Multi-agent frameworks (LangChain, CrewAI, Autogen)
- Personal memory systems (Mem0, OpenMemory)
- Team analytics tools (custom dashboards, metrics systems)

**All from a single unified semantic memory architecture.**

This is the competitive differentiator that makes ACT truly revolutionary.
