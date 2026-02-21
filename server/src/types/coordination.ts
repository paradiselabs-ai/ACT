/**
 * Core coordination types for ACT Phase 5
 */

export interface CoordinationMessage {
  timestamp: string;
  agent: string;
  message: string;
  type: MessageType;
}

export type MessageType =
  | 'feature_complete'
  | 'documentation_update'
  | 'architecture_decision'
  | 'phase_5_proposal'
  | 'task_breakdown'
  | 'instance_spawning'
  | 'progress_report'
  | 'blocker_identified'
  | 'question_for_team'
  | 'pvm_discovery'
  | 'pvm_extended_capabilities_discovery'
  | 'act_studio_vision_documentation'
  | 'naming_bundling_clarification'
  | 'mcp_server_ready'
  | 'agent_status_check'
  | 'task_assignment_confirmation'
  | 'coordination'
  // Agent socket message types (used for real-time routing classification)
  | 'status_update'
  | 'direct_mention'
  | 'help_request'
  | 'question'
  | 'peer_response';

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
}
