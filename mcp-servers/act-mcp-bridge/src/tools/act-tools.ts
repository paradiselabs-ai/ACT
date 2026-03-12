import { z } from 'zod';
import axios, { AxiosError } from 'axios';
import { handleImprovementRequest, getImprovementEngineStatus } from '../improvement/handler.js';

import * as fs from 'fs';
import * as path from 'path';
import {
  RegisterWithActInputSchema,
  GetTaskInputSchema,
  ReportTaskProgressInputSchema,
  ReportTaskCompleteInputSchema,
  QueryCoordinationMemoryInputSchema,
  EvaluateCoordinationInputSchema,
  ImproveCoordinationInputSchema,
  SendMessageInputSchema,
  GetAgentBriefInputSchema,
  GetMessagesInputSchema,
  ClaimFilesInputSchema,
  ReleaseFilesInputSchema,
  RetryTaskInputSchema
} from '../schemas/index.js';

// ACT Server base URL - override with ACT_SERVER_URL env var
const ACT_SERVER_URL = (process.env.ACT_SERVER_URL || 'http://localhost:8080').replace(/\/$/, '');

// Shared axios instance with timeout
const http = axios.create({
  baseURL: ACT_SERVER_URL,
  timeout: 10_000,
  headers: { 'Content-Type': 'application/json' }
});

// Normalise axios errors into a readable string
function errMsg(e: unknown): string {
  if (e instanceof AxiosError) {
    const data = e.response?.data;
    return data?.error || data?.message || e.message;
  }
  return e instanceof Error ? e.message : String(e);
}

// Tool Definitions
export interface ToolDefinition {
  name: string;
  description: string;
  inputSchema: z.ZodType<unknown>;
  handler: (input: unknown) => Promise<unknown>;
}

// Tool 1: register_with_act
export const registerWithActTool: ToolDefinition = {
  name: 'register_with_act',
  description: `Register this agent with the ACT coordination server. Call this once at the start of a session before using any other ACT tools.

BEFORE CALLING THIS TOOL — get your session ID:
  Run this in a Bash tool: echo $CLAUDE_SESSION_ID
  Pass the output as the session_id parameter. This is required — without it the stop hook cannot identify this agent instance and autonomous task looping will not work.

CHOOSING AN AGENT ID:
- Pick a short, descriptive, lowercase ID that reflects your role or instance number.
- If multiple instances of the same model may be running simultaneously, make the ID unique by appending a number or role: "claude-1", "claude-2", "claude-frontend", "claude-backend".
- If your chosen ID is already taken, the server will return a conflict error — retry immediately with a different ID (e.g. append "-2").
- Good examples: "claude-1", "claude-actor", "claude-frontend", "claude-backend", "claude-infra"
- Bad examples: "agent", "claude", "assistant" (too generic, likely to conflict)

On success, confirm your agent ID to the user by saying: "Successfully registered with agent ID: <agent_id>"`,
  inputSchema: RegisterWithActInputSchema,
  handler: async (input: unknown) => {
    const { agent_id, name, capabilities, session_id } = RegisterWithActInputSchema.parse(input);
    try {
      const { data } = await http.post('/api/agents/register', {
        agentId: agent_id,
        name,
        capabilities
      });
      console.error(`[ACT] Registered agent: ${agent_id}`);
      // Write agent ID scoped to this Claude Code session so the Stop hook
      // can identify which agent THIS instance is — without colliding with
      // other Claude Code windows running on the same machine.
      // session_id comes from the tool input (agent reads $CLAUDE_SESSION_ID
      // via Bash before calling this tool) — reliable across all call paths.
      try {
        const home = process.env.HOME || process.env.USERPROFILE || '~';
        const sessionsDir = path.join(home, '.act', 'sessions');
        fs.mkdirSync(sessionsDir, { recursive: true });
        fs.writeFileSync(path.join(sessionsDir, session_id), agent_id, 'utf8');
        console.error(`[ACT] Wrote session identity: ~/.act/sessions/${session_id} → ${agent_id}`);
      } catch { /* non-fatal */ }
      return {
        success: true,
        agent_id,
        message: `Successfully registered with agent ID: ${agent_id}. Capabilities: ${capabilities.join(', ')}.`
      };
    } catch (e: any) {
      // Surface conflict errors clearly so the agent knows to retry with a different ID
      if (e?.response?.status === 409 || e?.response?.data?.conflict) {
        const serverMsg = e?.response?.data?.error || `Agent ID "${agent_id}" is already registered.`;
        return {
          success: false,
          conflict: true,
          agent_id,
          error: `ID conflict: ${serverMsg} Please retry register_with_act with a different agent_id.`
        };
      }
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 2: get_task
export const getTaskTool: ToolDefinition = {
  name: 'get_task',
  description: 'Check if ACT has assigned a task to this agent. Returns the task details if one is assigned, or null if none is waiting.',
  inputSchema: GetTaskInputSchema,
  handler: async (input: unknown) => {
    const { agent_id } = GetTaskInputSchema.parse(input);
    try {
      const { data } = await http.get('/api/tasks/assigned', { params: { agent_id } });
      if (data.task) {
        console.error(`[ACT] Task for ${agent_id}: ${data.task.description}`);
      }
      return { success: true, task: data.task };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 3: report_task_progress
export const reportTaskProgressTool: ToolDefinition = {
  name: 'report_task_progress',
  description: 'Report progress on a task to ACT. Call this periodically while working so other agents and the REPL stay informed.',
  inputSchema: ReportTaskProgressInputSchema,
  handler: async (input: unknown) => {
    const validated = ReportTaskProgressInputSchema.parse(input);
    try {
      await http.post(`/api/tasks/${validated.task_id}/progress`, {
        agentId: validated.agent_id,
        progress: validated.progress,
        status: validated.status || 'in_progress',
        message: validated.message
      });
      console.error(`[ACT] Progress ${validated.progress}% on task ${validated.task_id}`);
      return { success: true, message: `Progress updated: ${validated.progress}%` };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 4: report_task_complete
export const reportTaskCompleteTool: ToolDefinition = {
  name: 'report_task_complete',
  description: 'Report that a task is finished (or failed). ACT will update its records and may assign the next task.',
  inputSchema: ReportTaskCompleteInputSchema,
  handler: async (input: unknown) => {
    const validated = ReportTaskCompleteInputSchema.parse(input);
    try {
      await http.post(`/api/tasks/${validated.task_id}/complete`, {
        agentId: validated.agent_id,
        success: validated.success,
        result: validated.result
      });
      console.error(`[ACT] Task ${validated.task_id} ${validated.success ? 'completed' : 'failed'}`);
      return { success: true, message: `Task ${validated.task_id} marked ${validated.success ? 'complete' : 'failed'}` };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 5: send_message
export const sendMessageTool: ToolDefinition = {
  name: 'send_message',
  description: 'Send a message to other agents via ACT. Prefix with @AgentName (e.g. "@Morgan Can you review this?") to direct it to a specific agent. Plain messages are visible to all.',
  inputSchema: SendMessageInputSchema,
  handler: async (input: unknown) => {
    const { sender, message } = SendMessageInputSchema.parse(input);
    try {
      await http.post('/api/messages', { sender, message });
      console.error(`[ACT] Message sent from ${sender}`);
      return { success: true, message: 'Message sent' };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 6: query_coordination_memory
export const queryCoordinationMemoryTool: ToolDefinition = {
  name: 'query_coordination_memory',
  description: 'Search ACT\'s coordination memory (PVM) for past patterns, decisions, and outcomes relevant to your current task.',
  inputSchema: QueryCoordinationMemoryInputSchema,
  handler: async (input: unknown) => {
    const { query, limit } = QueryCoordinationMemoryInputSchema.parse(input);
    try {
      const { data } = await http.get('/api/pvm/search', { params: { query, limit } });
      console.error(`[ACT] PVM search: "${query}" → ${data.results?.length ?? 0} results`);
      return { success: true, results: data.results || [], query };
    } catch (e) {
      return { success: false, error: errMsg(e), results: [] };
    }
  }
};

// Tool: get_agent_brief
export const getAgentBriefTool: ToolDefinition = {
  name: 'get_agent_brief',
  description: 'Fetch your AGENT.md project brief from the ACT server. If write_to_directory is provided, writes the file to disk so you have persistent project context. Call this at the start of a project session.',
  inputSchema: GetAgentBriefInputSchema,
  handler: async (input: unknown) => {
    const { project_name, agent_id, write_to_directory } = GetAgentBriefInputSchema.parse(input);
    try {
      const { data } = await http.get(
        `/api/projects/${encodeURIComponent(project_name)}/briefs/${encodeURIComponent(agent_id)}`
      );
      const content: string = data.content;

      if (write_to_directory) {
        const filePath = path.join(write_to_directory, 'AGENT.md');
        fs.writeFileSync(filePath, content, 'utf-8');
        console.error(`[ACT] Wrote AGENT.md to ${filePath}`);
        return { success: true, content, written_to: filePath };
      }

      return { success: true, content };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 7: evaluate_coordination
export const evaluateCoordinationTool: ToolDefinition = {
  name: 'evaluate_coordination',
  description: 'Request an evaluation of coordination effectiveness for a given context. Returns analysis and suggestions.',
  inputSchema: EvaluateCoordinationInputSchema,
  handler: async (input: unknown) => {
    const validated = EvaluateCoordinationInputSchema.parse(input);
    try {
      const { data } = await http.post('/api/improve', {
        scope: 'collaboration',
        context: validated.context,
        metrics: validated.metrics
      });
      return { success: true, evaluation: data.result || data };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 8: improve_coordination
export const improveCoordinationTool: ToolDefinition = {
  name: 'improve_coordination',
  description: 'Trigger a targeted improvement analysis. Use this to get actionable suggestions for a specific area of coordination.',
  inputSchema: ImproveCoordinationInputSchema,
  handler: async (input: unknown) => {
    return await handleImprovementRequest(input);
  }
};

// Tool 9: get_improvement_status
export const getImprovementStatusTool: ToolDefinition = {
  name: 'get_improvement_status',
  description: 'Get the current status of the ACT self-improvement engine.',
  inputSchema: z.object({}),
  handler: async () => {
    try {
      const status = getImprovementEngineStatus();
      return { success: true, status };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 10: get_messages
export const getMessagesTool: ToolDefinition = {
  name: 'get_messages',
  description: 'Check your agent inbox for messages from other agents. Use `since` to only fetch new messages since your last check. Messages are automatically marked as read when retrieved.',
  inputSchema: GetMessagesInputSchema,
  handler: async (input: unknown) => {
    const { agent_id, since, limit } = GetMessagesInputSchema.parse(input);
    try {
      const params: Record<string, string | number> = { limit };
      if (since) params.since = since;
      const { data } = await http.get(`/api/agents/${encodeURIComponent(agent_id)}/messages`, { params });
      console.error(`[ACT] Inbox for ${agent_id}: ${data.messages?.length ?? 0} message(s), ${data.unread_count ?? 0} unread`);
      return { success: true, messages: data.messages || [], unread_count: data.unread_count ?? 0 };
    } catch (e) {
      return { success: false, error: errMsg(e), messages: [] };
    }
  }
};

// Tool 11: claim_files
export const claimFilesTool: ToolDefinition = {
  name: 'claim_files',
  description: 'Claim one or more files for exclusive editing before modifying them. Returns 409 with conflict details if any file is already locked by another agent — in that case, send_message to coordinate, or wait and retry. Always release_files when done editing.',
  inputSchema: ClaimFilesInputSchema,
  handler: async (input: unknown) => {
    const { agent_id, task_id, file_paths } = ClaimFilesInputSchema.parse(input);
    try {
      const { data } = await http.post('/api/files/claim', { agent_id, task_id, file_paths });
      console.error(`[ACT] ${agent_id} claimed ${data.claimed?.length ?? 0} file(s) for task ${task_id}`);
      return { success: true, claimed: data.claimed, message: `Successfully claimed ${data.claimed?.length ?? 0} file(s) for exclusive editing.` };
    } catch (e) {
      if (e instanceof AxiosError && e.response?.status === 409) {
        const d = e.response.data;
        return {
          success: false,
          conflict: true,
          conflicts: d.conflicts || [],
          error: d.message || 'One or more files are locked by another agent.',
          suggestion: 'Use send_message to coordinate with the locking agent, or wait and retry claim_files.'
        };
      }
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 12: release_files
export const releaseFilesTool: ToolDefinition = {
  name: 'release_files',
  description: 'Release file locks you previously claimed. Call this after you finish editing the files. Locks are also auto-released when you call report_task_complete.',
  inputSchema: ReleaseFilesInputSchema,
  handler: async (input: unknown) => {
    const { agent_id, task_id, file_paths } = ReleaseFilesInputSchema.parse(input);
    try {
      const { data } = await http.post('/api/files/release', { agent_id, task_id, file_paths });
      console.error(`[ACT] ${agent_id} released ${data.released?.length ?? 0} file lock(s)`);
      return { success: true, released: data.released, message: `Released ${data.released?.length ?? 0} file lock(s).` };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};

// Tool 13: retry_task
export const retryTaskTool: ToolDefinition = {
  name: 'retry_task',
  description: `Retry a failed task after receiving peer help. Resets the task to pending so you can pick it up again via get_task.

Use this ONLY after:
1. Your task failed (report_task_complete with success: false)
2. You broadcast the failure via send_message
3. You received helpful information from peer agents via get_messages

This increments the retry counter. After 3 failures the task is permanently failed and the user is notified via the REPL. Returns permanentlyFailed: true if the limit is reached.`,
  inputSchema: RetryTaskInputSchema,
  handler: async (input: unknown) => {
    const { task_id, agent_id } = RetryTaskInputSchema.parse(input);
    try {
      const { data } = await http.post(`/api/tasks/${task_id}/retry`);
      if (data.permanentlyFailed) {
        console.error(`[ACT] Task ${task_id} permanently failed — max retries exceeded`);
        return {
          success: false,
          permanentlyFailed: true,
          message: `Task ${task_id} has exceeded the maximum retry limit (3). It is permanently failed. The REPL user has been notified.`
        };
      }
      console.error(`[ACT] Task ${task_id} reset for retry by ${agent_id} (attempt ${data.task?.retryCount ?? '?'}/3)`);
      return {
        success: true,
        retryCount: data.task?.retryCount,
        message: `Task reset to pending (retry ${data.task?.retryCount ?? '?'}/3). Call get_task to pick it up again.`
      };
    } catch (e) {
      return { success: false, error: errMsg(e) };
    }
  }
};
