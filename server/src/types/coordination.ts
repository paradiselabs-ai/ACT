/**
 * Core coordination types for ACT Phase 5
 */

export interface CoordinationMessage {
  timestamp: string;
  agent: string;
  message: string;
  type: MessageType;
  data?: Record<string, any>;  // Structured payload for event sourcing replay
  // scope: "project" = real project/coordination work (default for search results).
  // "meta" = events about ACT harness itself (CLI behavior, server HTTP errors,
  // runner tool-state issues). Meta scope is EXCLUDED from default PVM search so
  // stale tooling-state claims don't leak into agent task context on later runs.
  // Omitted scope is treated as "project" (pre-existing events age out naturally).
  scope?: 'project' | 'meta';
  // projectName: which project this event belongs to. Populated by PVMIndexer
  // from event.data.metadata.projectName (task events), event.data.projectName
  // (agent messages), event.data.name (project_created), or event.data.task.metadata.projectName
  // (embedded task payloads). Untagged or cross-project events are stored as
  // "__global__" and surface in every project-scoped query.
  projectName?: string;
}

export type MessageType =
  // Coordination categories
  | 'feature_complete'
  | 'documentation_update'
  | 'architecture_decision'
  | 'task_breakdown'
  | 'progress_report'
  | 'blocker_identified'
  | 'coordination'
  // Agent socket message types (used for real-time routing classification)
  | 'status_update'
  | 'direct_mention'
  | 'help_request'
  | 'question'
  | 'peer_response'
  | 'file_claim'
  | 'file_release'
  // Event sourcing types for restoreFromLog
  | 'project_created'
  | 'task_created'
  | 'task_assigned'
  | 'task_completed'
  | 'task_failed'
  | 'task_retry'
  | 'brief_stored'
  | 'agent_registered'
  // Assurance validation types
  | 'task_submitted_for_validation'
  | 'validation_submission_rejected'
  | 'task_validated'
  | 'task_validation_failed'
  // QA/Synthesizer outcome
  | 'synthesis_complete'
  | 'synthesis_needs_clarification'
  | 'dev_reset';

export interface AgentProfile {
  agentId: string;
  capabilities: Record<string, CapabilityMetrics>;
  communicationPatterns: CommunicationPattern[];
  toolUsage: Record<string, ToolUsageMetrics>;
  synergies: Record<string, SynergyMetrics>;
  overallPerformance: PerformanceMetrics;
  lastUpdated: string;
}

export interface CapabilityMetrics {
  successRate: number;
  taskCount: number;
  avgCompletionTime: number;
  confidenceScore: number;
  evidenceQuality: 'strong' | 'moderate' | 'weak';
}

export interface CommunicationPattern {
  pattern: string;
  frequency: number;
  effectiveness: number;
  contexts: string[];
}

export interface ToolUsageMetrics {
  toolName: string;
  usageCount: number;
  successRate: number;
  avgLatency: number;
}

export interface SynergyMetrics {
  partnerAgent: string;
  collaborationCount: number;
  successRate: number;
  strengthAreas: string[];
  weaknessAreas: string[];
}

export interface PerformanceMetrics {
  totalTasks: number;
  completedTasks: number;
  successRate: number;
  avgTaskTime: number;
  reliability: number;
}

export interface CoordinationPattern {
  id: string;
  pattern: string;
  context: string;
  outcome: 'success' | 'failure' | 'partial';
  participants: string[];
  timestamp: string;
  similarity?: number; // Added during search
}

export interface SearchResult {
  message: CoordinationMessage;
  similarity: number;
  context?: string;
}

export interface VectorSearchQuery {
  query: string;
  limit?: number;
  threshold?: number;
  agentFilter?: string;
  typeFilter?: MessageType;
  timeframe?: {
    start: string;
    end: string;
  };
  // Include meta-scope events (harness/tooling state). Default false — agent-facing
  // search excludes meta so stale "CLI broken" claims don't feed back into new runs.
  includeMeta?: boolean;
}
