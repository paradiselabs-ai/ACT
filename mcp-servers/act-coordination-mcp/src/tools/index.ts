/**
 * ACT Coordination MCP Server - Tool Implementations
 * 
 * All 8 coordination tools with proper MCP annotations and error handling.
 * Tools follow the append-only principle for communication_log.
 */

import { z } from 'zod';
import { coordinationFileService, CoordinationFileService } from '../services/coordination-file.js';
import {
  ReadCoordinationLogInputSchema,
  AppendCoordinationMessageInputSchema,
  GetAgentStatusInputSchema,
  GetPhaseStatusInputSchema,
  CheckForUpdatesInputSchema,
  SearchCoordinationLogInputSchema,
  GetDocumentationIndexInputSchema,
  GetProjectStructureInputSchema,
  type ReadCoordinationLogInput,
  type AppendCoordinationMessageInput,
  type GetAgentStatusInput,
  type CheckForUpdatesInput,
  type SearchCoordinationLogInput,
  type GetDocumentationIndexInput,
  type GetProjectStructureInput
} from '../schemas.js';
import { CoordinationError } from '../types.js';

// ============================================================================
// Tool Definitions
// ============================================================================

export interface ToolDefinition {
  name: string;
  description: string;
  inputSchema: z.ZodType<unknown>;
  annotations: {
    title: string;
    readOnlyHint?: boolean;
    destructiveHint?: boolean;
    idempotentHint?: boolean;
    openWorldHint?: boolean;
  };
  handler: (input: unknown, service: CoordinationFileService) => Promise<unknown>;
}

// ============================================================================
// Tool: act_read_coordination_log
// ============================================================================

export const readCoordinationLogTool: ToolDefinition = {
  name: 'act_read_coordination_log',
  description: `Read recent messages from the ACT coordination log.

Returns the most recent coordination messages with pagination support.
Messages are returned in reverse chronological order (newest first).

Use this to:
- Get context on what other agents have been doing
- Understand current project state and decisions
- Find recent blockers or questions

Example: Get last 20 messages: { "limit": 20 }
Example: Get older messages: { "limit": 10, "offset": 20 }`,
  inputSchema: ReadCoordinationLogInputSchema,
  annotations: {
    title: 'Read Coordination Log',
    readOnlyHint: true,
    destructiveHint: false,
    idempotentHint: true,
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    const validated = ReadCoordinationLogInputSchema.parse(input) as ReadCoordinationLogInput;
    const { messages, total, has_more } = await service.readCommunicationLog(
      validated.limit,
      validated.offset || 0
    );
    
    return {
      messages,
      pagination: {
        total,
        count: messages.length,
        offset: validated.offset || 0,
        has_more,
        next_offset: has_more ? (validated.offset || 0) + validated.limit : undefined
      }
    };
  }
};

// ============================================================================
// Tool: act_append_coordination_message
// ============================================================================

export const appendCoordinationMessageTool: ToolDefinition = {
  name: 'act_append_coordination_message',
  description: `Append a new message to the ACT coordination log.

IMPORTANT: This is an APPEND-ONLY operation. Messages are never overwritten or deleted.
This preserves the complete audit trail of agent coordination.

Use this to:
- Report progress on tasks
- Document architecture decisions
- Signal blockers or questions
- Coordinate with other agents
- Mark phase/task completions

Message format tip: Use markdown for rich formatting. Include:
- **WHAT**: Brief description
- **WHY**: Reasoning
- **IMPACT**: What this changes
- **NEXT**: Suggested follow-up

Valid message types: feature_complete, documentation_update, architecture_decision,
phase_5_proposal, task_breakdown, instance_spawning, progress_report, blocker_identified,
question_for_team, pvm_discovery, coordination, and more.`,
  inputSchema: AppendCoordinationMessageInputSchema,
  annotations: {
    title: 'Append Coordination Message',
    readOnlyHint: false,
    destructiveHint: false, // Append-only is non-destructive
    idempotentHint: false,  // Each call creates a new message
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    const validated = AppendCoordinationMessageInputSchema.parse(input) as AppendCoordinationMessageInput;
    
    const result = await service.appendMessage(
      validated.agent_name,
      validated.message_content,
      validated.message_type
    );
    
    return {
      success: true,
      timestamp: result.timestamp,
      message_index: result.index,
      message: result.message
    };
  }
};

// ============================================================================
// Tool: act_get_agent_status
// ============================================================================

export const getAgentStatusTool: ToolDefinition = {
  name: 'act_get_agent_status',
  description: `Get the current status and capabilities of a specific agent.

Returns:
- Agent capabilities and role
- Current task preferences and assignments
- Recent messages from this agent
- Completed work summary

Known agents: claude_code_instance_1, claude_code_instance_2, claude_ai_desktop,
windsurf, warp_terminal

Use this to understand what another agent is working on before coordinating.`,
  inputSchema: GetAgentStatusInputSchema,
  annotations: {
    title: 'Get Agent Status',
    readOnlyHint: true,
    destructiveHint: false,
    idempotentHint: true,
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    const validated = GetAgentStatusInputSchema.parse(input) as GetAgentStatusInput;
    const status = await service.getAgentStatus(validated.agent_name);
    
    return {
      agent_name: validated.agent_name,
      found: status.found,
      capabilities: status.capabilities,
      preferences: status.preferences,
      recent_messages: status.recent_messages
    };
  }
};

// ============================================================================
// Tool: act_get_phase_status
// ============================================================================

export const getPhaseStatusTool: ToolDefinition = {
  name: 'act_get_phase_status',
  description: `Get the current phase status and project progress.

Returns:
- Active phase name and details
- Current status (progress, milestones, critical path)
- Demo readiness status
- Critical decisions made in this phase

Use this to understand overall project state and priorities.`,
  inputSchema: GetPhaseStatusInputSchema,
  annotations: {
    title: 'Get Phase Status',
    readOnlyHint: true,
    destructiveHint: false,
    idempotentHint: true,
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    // Validate even though no params (ensures no extra fields passed)
    GetPhaseStatusInputSchema.parse(input);
    return await service.getPhaseStatus();
  }
};

// ============================================================================
// Tool: act_check_for_updates
// ============================================================================

export const checkForUpdatesTool: ToolDefinition = {
  name: 'act_check_for_updates',
  description: `Check if new messages have been added since a given timestamp.

Use this for efficient polling - only fetch full messages when there are updates.
Returns the count of new messages and optionally the messages themselves.

Workflow:
1. Read log, note the latest timestamp
2. Do your work
3. Check for updates with that timestamp
4. If has_updates=true, read the new messages`,
  inputSchema: CheckForUpdatesInputSchema,
  annotations: {
    title: 'Check for Updates',
    readOnlyHint: true,
    destructiveHint: false,
    idempotentHint: true,
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    const validated = CheckForUpdatesInputSchema.parse(input) as CheckForUpdatesInput;
    const result = await service.checkForUpdates(validated.last_read_timestamp);
    
    return {
      has_updates: result.has_updates,
      new_message_count: result.new_count,
      last_message_timestamp: result.messages.length > 0 
        ? result.messages[result.messages.length - 1].timestamp 
        : undefined,
      messages_since: result.messages
    };
  }
};

// ============================================================================
// Tool: act_search_coordination_log
// ============================================================================

export const searchCoordinationLogTool: ToolDefinition = {
  name: 'act_search_coordination_log',
  description: `Search the coordination log for specific content.

Search by keyword with optional filters:
- agent_filter: Only messages from a specific agent
- type_filter: Only messages of a specific type
- timeframe: last_day, last_week, last_month, or all

Returns matching messages with context (message before/after for continuity).

Examples:
- Find PVM decisions: { "query": "PVM", "type_filter": "architecture_decision" }
- Find recent blockers: { "query": "blocker", "timeframe": "last_week" }
- Find what Claude Desktop said about memory: { "query": "memory", "agent_filter": "claude_ai_desktop" }`,
  inputSchema: SearchCoordinationLogInputSchema,
  annotations: {
    title: 'Search Coordination Log',
    readOnlyHint: true,
    destructiveHint: false,
    idempotentHint: true,
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    const validated = SearchCoordinationLogInputSchema.parse(input) as SearchCoordinationLogInput;
    
    const { results, total } = await service.searchCommunicationLog(validated.query, {
      agent_filter: validated.agent_filter,
      type_filter: validated.type_filter,
      timeframe: validated.timeframe
    });
    
    return {
      query: validated.query,
      filters_applied: {
        agent: validated.agent_filter,
        type: validated.type_filter,
        timeframe: validated.timeframe
      },
      total_results: total,
      results
    };
  }
};

// ============================================================================
// Tool: act_get_documentation_index
// ============================================================================

export const getDocumentationIndexTool: ToolDefinition = {
  name: 'act_get_documentation_index',
  description: `Get an index of all documentation files in the project.

Returns a structured list of docs with:
- File path
- Title (extracted from first heading)
- Purpose (first paragraph summary)
- Last updated timestamp
- Optionally file sizes

Use this to discover relevant documentation before diving into implementation.`,
  inputSchema: GetDocumentationIndexInputSchema,
  annotations: {
    title: 'Get Documentation Index',
    readOnlyHint: true,
    destructiveHint: false,
    idempotentHint: true,
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    const validated = GetDocumentationIndexInputSchema.parse(input) as GetDocumentationIndexInput;
    const documents = await service.getDocumentationIndex(validated.include_sizes);
    
    return {
      docs_directory: 'docs/',
      total_documents: documents.length,
      documents
    };
  }
};

// ============================================================================
// Tool: act_get_project_structure
// ============================================================================

export const getProjectStructureTool: ToolDefinition = {
  name: 'act_get_project_structure',
  description: `Get the current project directory structure.

Returns a tree view of files and directories to help understand codebase organization.

Options:
- max_depth: How deep to traverse (default: 3, max: 5)
- include_hidden: Include dotfiles/dotdirs (default: false)
- exclude_patterns: Patterns to skip (default: node_modules, .git, build, etc.)

Use this when starting work to understand where things are.`,
  inputSchema: GetProjectStructureInputSchema,
  annotations: {
    title: 'Get Project Structure',
    readOnlyHint: true,
    destructiveHint: false,
    idempotentHint: true,
    openWorldHint: true
  },
  handler: async (input: unknown, service: CoordinationFileService) => {
    const validated = GetProjectStructureInputSchema.parse(input) as GetProjectStructureInput;
    
    const structure = await service.getProjectStructure(
      validated.max_depth,
      validated.include_hidden,
      validated.exclude_patterns
    );
    
    return {
      root: service.getProjectRoot(),
      structure,
      generated_at: new Date().toISOString()
    };
  }
};

// ============================================================================
// Export All Tools
// ============================================================================

export const allTools: ToolDefinition[] = [
  readCoordinationLogTool,
  appendCoordinationMessageTool,
  getAgentStatusTool,
  getPhaseStatusTool,
  checkForUpdatesTool,
  searchCoordinationLogTool,
  getDocumentationIndexTool,
  getProjectStructureTool
];

// ============================================================================
// Tool Handler Wrapper with Error Handling
// ============================================================================

export async function handleToolCall(
  toolName: string,
  input: unknown,
  service: CoordinationFileService = coordinationFileService
): Promise<{ success: boolean; result?: unknown; error?: { code: string; message: string; suggestion?: string } }> {
  const tool = allTools.find(t => t.name === toolName);
  
  if (!tool) {
    return {
      success: false,
      error: {
        code: 'UNKNOWN_TOOL',
        message: `Unknown tool: ${toolName}`,
        suggestion: `Available tools: ${allTools.map(t => t.name).join(', ')}`
      }
    };
  }
  
  try {
    const result = await tool.handler(input, service);
    return { success: true, result };
  } catch (error) {
    if (error instanceof z.ZodError) {
      const issues = error.issues.map(i => `${i.path.join('.')}: ${i.message}`).join('; ');
      return {
        success: false,
        error: {
          code: 'VALIDATION_ERROR',
          message: `Invalid input: ${issues}`,
          suggestion: 'Check the tool schema for required parameters and valid values'
        }
      };
    }
    
    if (error instanceof CoordinationError) {
      return {
        success: false,
        error: {
          code: error.code,
          message: error.message,
          suggestion: error.suggestion
        }
      };
    }
    
    return {
      success: false,
      error: {
        code: 'INTERNAL_ERROR',
        message: error instanceof Error ? error.message : 'Unknown error occurred',
        suggestion: 'Check server logs for details'
      }
    };
  }
}
