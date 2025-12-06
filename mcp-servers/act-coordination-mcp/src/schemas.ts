/**
 * ACT Coordination MCP Server - Zod Schemas
 * 
 * Runtime input validation schemas for all tools.
 * Follows MCP best practices with strict validation and descriptive errors.
 */

import { z } from 'zod';
import { VALID_MESSAGE_TYPES } from './types.js';

// ============================================================================
// Common Schemas
// ============================================================================

export const AgentNameSchema = z.string()
  .min(1, 'Agent name cannot be empty')
  .max(100, 'Agent name must not exceed 100 characters')
  .describe('Name/identifier of the agent (e.g., "claude_desktop", "windsurf", "warp_terminal")');

export const MessageTypeSchema = z.enum(VALID_MESSAGE_TYPES)
  .describe('Type of coordination message - must be a valid message type from the taxonomy');

export const TimestampSchema = z.string()
  .regex(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z?$/, 'Invalid ISO timestamp format')
  .describe('ISO 8601 timestamp (e.g., "2025-11-28T10:30:00Z")');

// ============================================================================
// Tool Input Schemas
// ============================================================================

/**
 * Schema for act_read_coordination_log tool
 */
export const ReadCoordinationLogInputSchema = z.object({
  limit: z.number()
    .int('Limit must be an integer')
    .min(1, 'Limit must be at least 1')
    .max(100, 'Limit must not exceed 100')
    .default(10)
    .describe('Maximum number of messages to return (default: 10, max: 100)'),
  offset: z.number()
    .int('Offset must be an integer')
    .min(0, 'Offset cannot be negative')
    .optional()
    .describe('Number of messages to skip from the end (0 = most recent)')
}).strict();

export type ReadCoordinationLogInput = z.infer<typeof ReadCoordinationLogInputSchema>;

/**
 * Schema for act_append_coordination_message tool
 */
export const AppendCoordinationMessageInputSchema = z.object({
  agent_name: AgentNameSchema,
  message_content: z.string()
    .min(1, 'Message content cannot be empty')
    .max(50000, 'Message content must not exceed 50,000 characters')
    .describe('The coordination message content (supports markdown formatting)'),
  message_type: MessageTypeSchema
}).strict();

export type AppendCoordinationMessageInput = z.infer<typeof AppendCoordinationMessageInputSchema>;

/**
 * Schema for act_get_agent_status tool
 */
export const GetAgentStatusInputSchema = z.object({
  agent_name: AgentNameSchema
}).strict();

export type GetAgentStatusInput = z.infer<typeof GetAgentStatusInputSchema>;

/**
 * Schema for act_get_phase_status tool
 * Note: No parameters required - returns current phase information
 */
export const GetPhaseStatusInputSchema = z.object({}).strict();

export type GetPhaseStatusInput = z.infer<typeof GetPhaseStatusInputSchema>;

/**
 * Schema for act_check_for_updates tool
 */
export const CheckForUpdatesInputSchema = z.object({
  last_read_timestamp: TimestampSchema
    .describe('ISO timestamp of when you last read the coordination log')
}).strict();

export type CheckForUpdatesInput = z.infer<typeof CheckForUpdatesInputSchema>;

/**
 * Schema for act_search_coordination_log tool
 */
export const SearchCoordinationLogInputSchema = z.object({
  query: z.string()
    .min(1, 'Search query cannot be empty')
    .max(500, 'Search query must not exceed 500 characters')
    .describe('Search string to match against message content'),
  agent_filter: AgentNameSchema
    .optional()
    .describe('Filter results to messages from a specific agent'),
  type_filter: MessageTypeSchema
    .optional()
    .describe('Filter results to a specific message type'),
  timeframe: z.enum(['last_day', 'last_week', 'last_month', 'all'])
    .default('all')
    .describe('Time range to search within (default: "all")')
}).strict();

export type SearchCoordinationLogInput = z.infer<typeof SearchCoordinationLogInputSchema>;

/**
 * Schema for act_get_documentation_index tool
 */
export const GetDocumentationIndexInputSchema = z.object({
  include_sizes: z.boolean()
    .default(false)
    .describe('Include file sizes in the response')
}).strict();

export type GetDocumentationIndexInput = z.infer<typeof GetDocumentationIndexInputSchema>;

/**
 * Schema for act_get_project_structure tool
 */
export const GetProjectStructureInputSchema = z.object({
  max_depth: z.number()
    .int('Max depth must be an integer')
    .min(1, 'Max depth must be at least 1')
    .max(5, 'Max depth must not exceed 5')
    .default(3)
    .describe('Maximum directory depth to traverse (default: 3, max: 5)'),
  include_hidden: z.boolean()
    .default(false)
    .describe('Include hidden files/directories (starting with .)'),
  exclude_patterns: z.array(z.string())
    .default(['node_modules', '.git', 'build', 'dist', '__pycache__', '.next'])
    .describe('Directory/file patterns to exclude')
}).strict();

export type GetProjectStructureInput = z.infer<typeof GetProjectStructureInputSchema>;

// ============================================================================
// Output Schemas (for structuredContent)
// ============================================================================

export const CoordinationMessageOutputSchema = z.object({
  timestamp: z.string(),
  agent: z.string(),
  message: z.string(),
  type: z.string()
});

export const PaginationOutputSchema = z.object({
  total: z.number(),
  count: z.number(),
  offset: z.number(),
  has_more: z.boolean(),
  next_offset: z.number().optional()
});

export const ReadLogOutputSchema = PaginationOutputSchema.extend({
  messages: z.array(CoordinationMessageOutputSchema)
});

export const AppendMessageOutputSchema = z.object({
  success: z.boolean(),
  timestamp: z.string(),
  message_index: z.number(),
  message: CoordinationMessageOutputSchema
});

export const SearchResultOutputSchema = z.object({
  message: CoordinationMessageOutputSchema,
  index: z.number(),
  context_before: CoordinationMessageOutputSchema.optional(),
  context_after: CoordinationMessageOutputSchema.optional()
});

export const SearchOutputSchema = z.object({
  query: z.string(),
  filters_applied: z.object({
    agent: z.string().optional(),
    type: z.string().optional(),
    timeframe: z.string().optional()
  }),
  total_results: z.number(),
  results: z.array(SearchResultOutputSchema)
});
