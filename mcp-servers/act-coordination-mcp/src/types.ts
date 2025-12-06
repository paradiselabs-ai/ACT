/**
 * ACT Coordination MCP Server - Type Definitions
 * 
 * These types mirror the structure of act-coordination.json and provide
 * type safety for all coordination operations.
 */

// ============================================================================
// Message Types - Validated taxonomy for coordination messages
// ============================================================================

export const VALID_MESSAGE_TYPES = [
  'feature_complete',
  'documentation_update', 
  'architecture_decision',
  'phase_5_proposal',
  'task_breakdown',
  'instance_spawning',
  'progress_report',
  'blocker_identified',
  'question_for_team',
  'pvm_discovery',
  'coordination',
  'phase_complete',
  'task_start',
  'task_complete',
  'major_milestone',
  'project_completion',
  'honest_assessment',
  'collaboration_request',
  'collaboration_response',
  'major_breakthrough',
  'mission_accomplished',
  'dashboard_fixes_applied',
  'coordination_request',
  'architectural_clarity',
  'architectural_discovery',
  'task_check_breakthrough',
  'live_demo_preparation',
  'project_handoff_failure',
  'competitive_analysis_urgency',
  'enterprise_challenge_response',
  'direct_claude_coordination',
  'hackathon_strategy',
  'claude_agent_sdk_integration',
  'hackathon_status_update',
  'portability_requirements',
  'instance_2_task_preference',
  'instance_2_acknowledgment',
  'instance_1_response',
  'instance_2_task_start',
  'instance_2_progress',
  'pvm_extended_capabilities_discovery',
  'act_studio_vision_documentation',
  'naming_bundling_clarification',
  'phase_5_approved_ready_for_implementation',
  'phase_5_planning_complete',
  'initialization',
  'mcp_server_ready'
] as const;

export type MessageType = typeof VALID_MESSAGE_TYPES[number];

// ============================================================================
// Agent Types
// ============================================================================

export interface AgentCapabilities {
  role: string;
  status: string;
  capabilities: string[];
  assigned_focus_areas?: string | string[];
  completed_work?: string[];
  notes?: string;
}

export interface AgentPreference {
  status: string;
  week_1_tasks?: string[];
  completed_tasks?: string[];
  available_tasks?: string[];
  reasoning?: string;
  notes?: string;
}

// ============================================================================
// Coordination Message Types
// ============================================================================

export interface CoordinationMessage {
  timestamp: string;
  agent: string;
  message: string;
  type: string;
}

// ============================================================================
// Task Types
// ============================================================================

export interface Task {
  description: string;
  assigned_agent: string;
  estimated_hours: number;
  status: string;
  started_at?: string;
  completed_at?: string;
  note?: string;
  dependencies: string[];
  files?: string[];
  deliverables?: string[];
}

export interface Phase {
  name: string;
  timeline: string;
  status: string;
  description?: string;
  completed_at?: string;
  note?: string;
  tasks: Record<string, Task>;
}

// ============================================================================
// Project Structure Types
// ============================================================================

export interface Project {
  name: string;
  description: string;
  timeline: string;
  goal: string;
  origin?: string;
}

export interface CurrentStatus {
  active_phase: string;
  next_milestone: string;
  total_progress: string;
  estimated_completion: string;
  critical_path: string[];
  demo_ready: boolean;
  build_in_public_ready: boolean;
  critical_lessons_learned?: string[];
}

// ============================================================================
// Full Coordination File Structure
// ============================================================================

export interface CoordinationFile {
  project: Project;
  agents: Record<string, AgentCapabilities>;
  active_assignments_summary?: {
    context: string;
    agents: string[];
    manual_usage_note: string;
  };
  phases: Record<string, Phase>;
  current_status: CurrentStatus;
  parallel_development_plan?: Record<string, unknown>;
  implementation_brainstorm?: Record<string, unknown>;
  enterprise_coordination_challenge?: Record<string, unknown>;
  revolutionary_vision?: Record<string, unknown>;
  demo_requirements?: Record<string, unknown>;
  success_criteria?: Record<string, unknown>;
  active_phase_5_assignments?: Record<string, unknown>;
  phase_5_status?: Record<string, unknown>;
  communication_log: CoordinationMessage[];
  resources: {
    documentation: string[];
    development_urls: string[];
  };
}

// ============================================================================
// Tool Response Types
// ============================================================================

export interface PaginationMetadata {
  total: number;
  count: number;
  offset: number;
  has_more: boolean;
  next_offset?: number;
}

export interface ReadLogResponse extends PaginationMetadata {
  messages: CoordinationMessage[];
}

export interface AppendMessageResponse {
  success: boolean;
  timestamp: string;
  message_index: number;
  message: CoordinationMessage;
}

export interface AgentStatusResponse {
  agent_name: string;
  found: boolean;
  capabilities?: AgentCapabilities;
  preferences?: AgentPreference;
  recent_messages?: CoordinationMessage[];
}

export interface PhaseStatusResponse {
  active_phase: string;
  phase_details?: Phase;
  current_status: CurrentStatus;
  critical_decisions?: string[];
}

export interface UpdateCheckResponse {
  has_updates: boolean;
  new_message_count: number;
  last_message_timestamp?: string;
  messages_since?: CoordinationMessage[];
}

export interface SearchResult {
  message: CoordinationMessage;
  index: number;
  context_before?: CoordinationMessage;
  context_after?: CoordinationMessage;
}

export interface SearchResponse {
  query: string;
  filters_applied: {
    agent?: string;
    type?: string;
    timeframe?: string;
  };
  total_results: number;
  results: SearchResult[];
}

export interface DocumentationEntry {
  path: string;
  title: string;
  purpose: string;
  last_updated?: string;
  size_bytes?: number;
}

export interface DocumentationIndexResponse {
  docs_directory: string;
  total_documents: number;
  documents: DocumentationEntry[];
}

export interface ProjectStructureEntry {
  path: string;
  type: 'file' | 'directory';
  children?: ProjectStructureEntry[];
}

export interface ProjectStructureResponse {
  root: string;
  structure: ProjectStructureEntry[];
  generated_at: string;
}

// ============================================================================
// Error Types
// ============================================================================

export class CoordinationError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly suggestion?: string
  ) {
    super(message);
    this.name = 'CoordinationError';
  }
}
