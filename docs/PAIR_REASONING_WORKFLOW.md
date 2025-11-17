# PAIR Reasoning Workflow - Detailed Implementation

**Date:** October 22, 2025
**Component:** Past Archived Information Re-injection (PAIR)
**Status:** Design Phase
**Purpose:** Enable self-improving coordination through informed revision cycles

---

## Overview

PAIR is the intelligent memory retrieval and re-injection system that enables coordination decisions to improve over time. It's triggered when FLUX State evaluation identifies gaps between desired outcomes and actual results.

**Key distinction:** PAIR is NOT reasoning. It's semantic context retrieval that enables reasoning. The reasoning happens in FLUX State evaluation.

---

## Workflow Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PAIR Reasoning Cycle                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. SESSION COMPLETE                                       │
│     └─ Coordination session finishes                       │
│                                                             │
│  2. FLUX STATE EVALUATION (Memory Wipe)                   │
│     ├─ Original task: "Assign agents to build todo app"  │
│     ├─ Success criteria: "App fully functional, <2 bugs" │
│     ├─ Output: "Built todo app with 3 bugs, 80% done"   │
│     └─ Question: "Does this meet criteria?" → NO (80%)   │
│                                                             │
│  3. IDENTIFY GAPS                                          │
│     ├─ Gap #1: "Only 80% complete, should be 100%"      │
│     ├─ Gap #2: "3 bugs found, should be <2"            │
│     └─ Gap #3: "Why was QA agent not assigned?"         │
│                                                             │
│  4. PAIR RETRIEVAL (For each gap)                         │
│     ├─ Query: "Similar assignments: full app completion" │
│     ├─ Retrieve: Past patterns for complete delivery     │
│     ├─ Query: "Conflict resolution: quality vs speed"    │
│     └─ Retrieve: Past decisions when quality mattered    │
│                                                             │
│  5. INFORMED REVISION                                      │
│     ├─ Agent sees context of similar decisions            │
│     ├─ Agent evaluates: "Should we have assigned QA?"    │
│     ├─ Agent decides: "Yes, we should have"             │
│     └─ Suggest: "Include QA agent in next coordination" │
│                                                             │
│  6. IMPROVED COORDINATION                                  │
│     ├─ New assignment: Frontend + Backend + QA           │
│     └─ FLUX State evaluation improves: 95% → success      │
│                                                             │
│  7. LOOP (until 95%+ success criteria met)               │
│     └─ Repeat PAIR cycle if needed                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Component 1: FLUX State Evaluation

### Purpose
Unbiased assessment of coordination outcome against success criteria.

### Implementation

```typescript
interface FluxStateEvaluation {
  original_task: string;
  success_criteria: SuccessCriteria[];
  actual_output: any;

  // Evaluation results
  evaluation_date: ISO8601;
  evaluator_agent_id: string;      // Fresh agent, different from coordinator

  // Scoring
  success_criteria_met: number;    // 0-100%, composite score
  individual_criteria_scores: Map<string, number>;

  // Analysis
  identified_gaps: Gap[];
  critical_issues: string[];
  positive_aspects: string[];

  // Recommendation
  recommendation: EvaluationRecommendation;
}

interface Gap {
  id: string;
  description: string;             // What's missing
  severity: 'low' | 'medium' | 'high' | 'critical';
  impact_on_criteria: number;      // How much this hurts score
  improvement_priority: number;    // What to fix first
}

enum EvaluationRecommendation {
  MEETS_CRITERIA = 'meets_criteria',
  NEEDS_IMPROVEMENT = 'needs_improvement',
  SIGNIFICANT_GAPS = 'significant_gaps',
  TOTAL_FAILURE = 'total_failure'
}
```

### Evaluation Process

```typescript
class FluxStateEvaluator {
  async evaluate(session: CoordinationSession): Promise<FluxStateEvaluation> {
    // 1. Prepare evaluation context (no memory of creating this)
    const evaluationPrompt = this.buildEvaluationPrompt(session);

    // 2. Use fresh agent (not the coordinator)
    const freshAgent = this.getEvaluationAgent(
      agentId: `flux_evaluator_${Date.now()}`
    );

    // 3. Run evaluation
    const evaluation = await freshAgent.evaluate(evaluationPrompt);

    // 4. Analyze results
    return {
      ...evaluation,
      evaluator_agent_id: freshAgent.id,
      evaluation_date: new Date().toISOString(),
      success_criteria_met: this.calculateCompositeScore(evaluation)
    };
  }

  private buildEvaluationPrompt(session: CoordinationSession): string {
    return `
      You are evaluating a coordination outcome. You did NOT create this.
      Judge it objectively against the criteria.

      ORIGINAL TASK:
      ${session.original_task}

      SUCCESS CRITERIA:
      ${session.success_criteria.map((c, i) => `${i+1}. ${c.description} (weight: ${c.weight})`).join('\n')}

      ACTUAL OUTCOME:
      ${JSON.stringify(session.output, null, 2)}

      EVALUATION QUESTIONS:
      1. Does the outcome meet each success criterion? Why/why not?
      2. What's missing? What went wrong?
      3. What happened well?
      4. What would need to change to meet all criteria?

      Be critical. Be honest. No sugar-coating.
    `;
  }

  private calculateCompositeScore(evaluation: any): number {
    // Weight individual criteria scores
    const scores = Object.values(evaluation.individual_criteria_scores);
    const weights = session.success_criteria.map(c => c.weight);

    return (scores as number[])
      .reduce((sum, score, i) => sum + (score * weights[i]), 0) / weights.reduce((a, b) => a + b, 0);
  }
}
```

---

## Component 2: Gap Identification

### Purpose
Extract specific improvement opportunities from evaluation.

### Implementation

```typescript
class GapIdentifier {
  async identifyGaps(
    evaluation: FluxStateEvaluation,
    session: CoordinationSession
  ): Promise<Gap[]> {
    const gaps: Gap[] = [];

    // 1. Extract from evaluation
    for (const [criterion, score] of evaluation.individual_criteria_scores) {
      if (score < 100) {
        gaps.push({
          id: `gap_${criterion}_${Date.now()}`,
          description: `Criterion '${criterion}' not fully met (${score}%)`,
          severity: this.calculateSeverity(score),
          impact_on_criteria: 100 - score,
          improvement_priority: this.prioritize(score)
        });
      }
    }

    // 2. Extract from identified issues
    for (const issue of evaluation.critical_issues) {
      gaps.push({
        id: `gap_issue_${Date.now()}`,
        description: issue,
        severity: 'high',
        impact_on_criteria: 20,
        improvement_priority: 1
      });
    }

    // 3. Sort by priority
    return gaps.sort((a, b) => a.improvement_priority - b.improvement_priority);
  }

  private calculateSeverity(score: number): 'low' | 'medium' | 'high' | 'critical' {
    if (score >= 90) return 'low';
    if (score >= 75) return 'medium';
    if (score >= 50) return 'high';
    return 'critical';
  }

  private prioritize(score: number): number {
    // Lower score = higher priority (lower number)
    return Math.max(1, Math.floor((100 - score) / 10));
  }
}
```

---

## Component 3: PAIR Retrieval

### Purpose
For each gap, retrieve semantically similar past coordination patterns.

### Implementation

```typescript
class PAIRRetrieval {
  async retrieveRelevantPatterns(
    gaps: Gap[],
    vectorStore: VectorMemoryStore
  ): Promise<PAIRContext[]> {
    const pairContexts: PAIRContext[] = [];

    for (const gap of gaps) {
      // 1. Build search query from gap
      const query = this.buildSearchQuery(gap);

      // 2. Vector search for relevant past coordination
      const relevantMemories = await vectorStore.search(query, limit: 10);

      // 3. Filter for high-confidence matches
      const confident = relevantMemories.filter(m => m.metadata.composite_score > 0.5);

      // 4. Organize by pattern type
      const organized = this.organizeByPattern(confident);

      pairContexts.push({
        gap,
        query,
        retrieved_patterns: organized,
        recommendation: this.generateRecommendation(organized)
      });
    }

    return pairContexts;
  }

  private buildSearchQuery(gap: Gap): string {
    // Convert gap into semantic search query
    // Examples:
    // Gap: "Criterion 'quality' not met"
    // Query: "Coordination pattern: quality assurance, testing, bug prevention"

    // Gap: "Why was QA agent not assigned?"
    // Query: "Task assignment decision: when to include QA, testing requirements"

    return `Coordination pattern: ${gap.description}`;
  }

  private organizeByPattern(memories: VectorMemory[]): CoordinationPattern[] {
    const patterns = new Map<string, VectorMemory[]>();

    for (const memory of memories) {
      const category = memory.metadata.category;
      if (!patterns.has(category)) {
        patterns.set(category, []);
      }
      patterns.get(category)!.push(memory);
    }

    return Array.from(patterns.entries()).map(([category, memories]) => ({
      category,
      examples: memories.slice(0, 3),  // Top 3 examples
      success_rate: this.calculateSuccessRate(memories),
      confidence: this.calculateConfidence(memories)
    }));
  }

  private generateRecommendation(patterns: CoordinationPattern[]): string {
    if (!patterns.length) return "No similar past patterns found.";

    const topPattern = patterns[0];
    return `
      Based on ${topPattern.examples.length} similar past coordination decisions:
      - Success rate: ${(topPattern.success_rate * 100).toFixed(0)}%
      - Consider: ${topPattern.category}
      - Confidence: ${(topPattern.confidence * 100).toFixed(0)}%
    `;
  }
}

interface PAIRContext {
  gap: Gap;
  query: string;
  retrieved_patterns: CoordinationPattern[];
  recommendation: string;
}

interface CoordinationPattern {
  category: string;
  examples: VectorMemory[];
  success_rate: number;
  confidence: number;
}
```

---

## Component 4: Informed Revision

### Purpose
Agent uses PAIR context to revise coordination decisions.

### Implementation

```typescript
class InformedRevision {
  async reviseCoordination(
    session: CoordinationSession,
    pairContexts: PAIRContext[]
  ): Promise<RevisedCoordination> {
    const revisionPrompt = this.buildRevisionPrompt(session, pairContexts);

    // Use same fresh evaluator (consistency)
    const revision = await freshAgent.revise(revisionPrompt);

    return {
      original_session: session,
      identified_gaps: pairContexts.map(p => p.gap),
      pair_context: pairContexts,
      revised_decisions: revision.decisions,
      revised_assignments: revision.assignments,
      rationale: revision.rationale,
      confidence_improvement: this.estimateImprovement(revision)
    };
  }

  private buildRevisionPrompt(
    session: CoordinationSession,
    pairContexts: PAIRContext[]
  ): string {
    return `
      You evaluated a coordination outcome and found gaps.
      Here's context from SIMILAR past coordination decisions that WORKED:

      ${pairContexts.map(pc => `
        GAP: ${pc.gap.description}

        SIMILAR SUCCESSFUL PATTERNS:
        ${pc.retrieved_patterns
          .flatMap(p => p.examples)
          .map((m, i) => `
            ${i+1}. ${m.insights[0]?.finding || 'Example'}
               Success: ${m.metadata.accuracy_score * 100}%
               Applied in: ${m.metadata.category}
          `)
          .join('\n')}

        RECOMMENDATION:
        ${pc.recommendation}
      `).join('\n---\n')}

      Now, knowing how similar coordination problems were solved successfully:
      1. What should we have done differently?
      2. What agent assignments were missing?
      3. What coordination rules would have helped?
      4. How would this change the outcome?

      Be specific. Reference the similar patterns above.
    `;
  }

  private estimateImprovement(revision: any): number {
    // Estimate how much success criteria would improve with revisions
    return revision.estimated_score_improvement || 0;
  }
}

interface RevisedCoordination {
  original_session: CoordinationSession;
  identified_gaps: Gap[];
  pair_context: PAIRContext[];
  revised_decisions: string[];
  revised_assignments: AgentAssignment[];
  rationale: string;
  confidence_improvement: number;
}
```

---

## Component 5: Loop Control

### Purpose
Manage PAIR iteration cycles until success criteria met.

### Implementation

```typescript
class PAIRLoopController {
  private MAX_ITERATIONS = 3;
  private SUCCESS_THRESHOLD = 95;

  async runPAIRCycle(
    session: CoordinationSession,
    vectorStore: VectorMemoryStore
  ): Promise<FinalCoordination> {
    let currentSession = session;
    let iteration = 0;

    while (iteration < this.MAX_ITERATIONS) {
      // 1. FLUX State evaluation
      const evaluation = await fluxStateEvaluator.evaluate(currentSession);

      console.log(`[PAIR Iteration ${iteration + 1}] Success: ${evaluation.success_criteria_met}%`);

      // Check success
      if (evaluation.success_criteria_met >= this.SUCCESS_THRESHOLD) {
        return {
          final_session: currentSession,
          final_score: evaluation.success_criteria_met,
          iterations: iteration + 1,
          status: 'SUCCESS',
          evaluation
        };
      }

      // 2. Identify gaps
      const gaps = await gapIdentifier.identifyGaps(evaluation, currentSession);

      // 3. PAIR retrieval
      const pairContexts = await pairRetrieval.retrieveRelevantPatterns(
        gaps,
        vectorStore
      );

      // 4. Informed revision
      const revised = await informedRevision.reviseCoordination(
        currentSession,
        pairContexts
      );

      // 5. Update session for next iteration
      currentSession = {
        ...currentSession,
        assignments: revised.revised_assignments,
        decisions: revised.revised_decisions,
        pair_history: [
          ...(currentSession.pair_history || []),
          {
            iteration,
            gaps,
            pairContexts,
            revised,
            improvement_estimate: revised.confidence_improvement
          }
        ]
      };

      iteration++;
    }

    // Max iterations reached
    return {
      final_session: currentSession,
      final_score: evaluation.success_criteria_met,
      iterations: iteration,
      status: 'MAX_ITERATIONS_REACHED',
      evaluation
    };
  }
}

interface FinalCoordination {
  final_session: CoordinationSession;
  final_score: number;
  iterations: number;
  status: 'SUCCESS' | 'MAX_ITERATIONS_REACHED' | 'ERROR';
  evaluation: FluxStateEvaluation;
}
```

---

## Complete PAIR Workflow

```typescript
class PAIRReasoningEngine {
  async runCompleteWorkflow(
    session: CoordinationSession,
    vectorStore: VectorMemoryStore
  ): Promise<FinalCoordination> {
    console.log(`Starting PAIR workflow for session: ${session.id}`);

    try {
      // Run the complete cycle
      const result = await pairLoopController.runPAIRCycle(session, vectorStore);

      // 6. Store improved coordination in memory
      if (result.status === 'SUCCESS') {
        await this.storeImprovedCoordination(result);
      }

      return result;
    } catch (error) {
      console.error('PAIR workflow error:', error);
      throw error;
    }
  }

  private async storeImprovedCoordination(result: FinalCoordination): Promise<void> {
    // Record this successful coordination for future PAIR retrievals
    const minute: CoordinationMinute = {
      id: `pair_improvement_${Date.now()}`,
      timestamp: new Date().toISOString(),
      event_type: 'pair_improvement_completed',
      project_id: result.final_session.project_id,
      data: {
        original_score: result.final_session.initial_score,
        final_score: result.final_score,
        iterations: result.iterations,
        decisions: result.final_session.decisions,
        assignments: result.final_session.assignments
      },
      memory_note: `Improved coordination from ${result.final_session.initial_score}% to ${result.final_score}% in ${result.iterations} PAIR iterations`
    };

    await chronologicalLog.append(minute);
    await vectorStore.storeMemory(minute, [
      {
        type: 'coordination_pattern',
        finding: `Successful PAIR improvement cycle: ${minute.data.decisions.join(', ')}`,
        confidence: result.final_score / 100
      }
    ]);
  }
}
```

---

## Data Flow

```
CoordinationSession (completed)
  ↓
FLUX State Evaluation (memory wipe)
  ├→ Success criteria met < 95%?
  └→ YES: Continue to PAIR

Gap Identification
  ↓
PAIR Retrieval (Vector search)
  ├→ Find similar past patterns
  ├→ Extract success rates
  └→ Generate recommendations

Informed Revision
  ├→ Fresh agent sees gaps + context
  ├→ Decides what should change
  └→ Revises assignments/decisions

Loop Control
  ├→ Success >= 95%? → Complete
  ├→ Max iterations reached? → Complete
  └→ Else: Return to FLUX State evaluation
      (with revised session)
```

---

## Key Design Decisions

1. **Fresh evaluator agent** - Prevents bias. Same agent that coordinated shouldn't evaluate its own work.
2. **Semantic retrieval** - Vector search finds conceptually similar patterns, not keyword matches.
3. **Iterative improvement** - PAIR loops until success, with max iterations safeguard.
4. **Context preservation** - Each PAIR iteration builds on previous revisions.
5. **Memory recording** - Improved coordination patterns get stored for future PAIR retrievals.

---

## Integration Points

- **Triggered by:** Self-improvement engine (`/improve` endpoint)
- **Uses:** VectorMemoryStore for semantic search
- **Records to:** ChronologicalLog for audit trail
- **Called by:** ACT coordination server post-session

---

## Success Metrics

- ✅ Coordination decisions improve through iteration
- ✅ Success criteria met increases toward 95%
- ✅ System learns patterns that work (stored memories)
- ✅ Fresh evaluators provide unbiased assessment
- ✅ Complete audit trail of improvement decisions
