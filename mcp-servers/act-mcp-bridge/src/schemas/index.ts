import { z } from 'zod';

// Base schemas
export const AgentIdSchema = z.string().min(1).describe('Unique identifier for an agent');
export const TaskIdSchema = z.string().min(1).describe('Unique identifier for a task');

// Tool 1: register_with_act
export const RegisterWithActInputSchema = z.object({
  agent_id: AgentIdSchema.describe('Unique identifier for the agent'),
  name: z.string().min(1).describe('Human-readable name for the agent'),
  capabilities: z.array(z.string()).describe('List of capabilities the agent possesses'),
  session_id: z.string().min(1).describe('The CLAUDE_SESSION_ID from your environment. Run: echo $CLAUDE_SESSION_ID in a Bash tool to get it. Required so the stop hook can identify this agent instance.')
});

// Tool 2: get_task
export const GetTaskInputSchema = z.object({
  agent_id: AgentIdSchema.describe('ID of the agent requesting a task')
});

// Tool 3: report_task_progress
export const ReportTaskProgressInputSchema = z.object({
  task_id: TaskIdSchema.describe('ID of the task being updated'),
  agent_id: AgentIdSchema.describe('ID of the agent reporting progress'),
  progress: z.number().min(0).max(100).describe('Progress percentage (0-100)'),
  status: z.enum(['pending', 'assigned', 'in_progress', 'completed', 'failed']).optional().describe('Current status of the task'),
  message: z.string().optional().describe('Optional message about the progress')
});

// Tool 4: report_task_complete
export const ReportTaskCompleteInputSchema = z.object({
  task_id: TaskIdSchema.describe('ID of the task being completed'),
  agent_id: AgentIdSchema.describe('ID of the agent completing the task'),
  success: z.boolean().describe('Whether the task was completed successfully'),
  result: z.string().optional().describe('Optional result or output of the completed task')
});

// Tool 5: query_coordination_memory
export const QueryCoordinationMemoryInputSchema = z.object({
  query: z.string().min(1).describe('Search query for coordination memory'),
  limit: z.number().min(1).max(100).default(10).describe('Maximum number of results to return')
});

// Tool 6: evaluate_coordination
export const EvaluateCoordinationInputSchema = z.object({
  context: z.string().min(1).describe('Context for coordination evaluation'),
  metrics: z.array(z.string()).optional().describe('Specific metrics to evaluate')
});

// Tool 7b: send_message
export const SendMessageInputSchema = z.object({
  sender: AgentIdSchema.describe('ID of the agent sending the message'),
  message: z.string().min(1).describe('Message to send. Prefix with @AgentName to direct it to a specific agent.')
});

// Tool: get_agent_brief
export const GetAgentBriefInputSchema = z.object({
  project_name: z.string().min(1).describe('Name of the project to fetch a brief for'),
  agent_id: AgentIdSchema.describe('ID of the agent requesting their brief'),
  write_to_directory: z.string().optional().describe('Directory path to write AGENT.md file (e.g. /path/to/project). If omitted, returns content only.')
});

// Tool: get_messages
export const GetMessagesInputSchema = z.object({
  agent_id: AgentIdSchema.describe('ID of the agent checking their inbox'),
  since: z.string().optional().describe('ISO timestamp — only return messages after this time'),
  limit: z.number().min(1).max(100).default(20).describe('Maximum number of messages to return')
});

// Tool: claim_files
export const ClaimFilesInputSchema = z.object({
  agent_id: AgentIdSchema.describe('ID of the agent claiming the files'),
  task_id: TaskIdSchema.describe('ID of the task this work belongs to'),
  file_paths: z.array(z.string().min(1)).min(1).describe('List of absolute or relative file paths to claim for exclusive editing')
});

// Tool: release_files
export const ReleaseFilesInputSchema = z.object({
  agent_id: AgentIdSchema.describe('ID of the agent releasing the files'),
  task_id: TaskIdSchema.optional().describe('Task ID (for logging purposes)'),
  file_paths: z.array(z.string().min(1)).min(1).describe('List of file paths to release')
});

// Tool: retry_task
export const RetryTaskInputSchema = z.object({
  task_id: TaskIdSchema.describe('ID of the failed task to retry'),
  agent_id: AgentIdSchema.describe('ID of the agent requesting the retry (must be the task owner)')
});

// Tool 8: improve_coordination
export const ImproveCoordinationInputSchema = z.object({
  scope: z.enum([
    'communication', 
    'tools', 
    'assignments', 
    'conflicts', 
    'collaboration', 
    'performance', 
    'knowledge'
  ]).describe('Scope of improvement'),
  agents: z.array(AgentIdSchema).optional().describe('Specific agents to focus on'),
  session: z.string().optional().describe('Session or project to improve'),
  filter: z.string().optional().describe('Filter for specific issues'),
  output: z.enum(['summary', 'detailed-report', 'action-items']).default('summary').describe('Output format')
});