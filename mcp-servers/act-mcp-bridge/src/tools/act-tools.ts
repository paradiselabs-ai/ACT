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
  GetAgentBriefInputSchema
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
  description: 'Register this agent with the ACT coordination server. Call this once at the start of a session before using any other ACT tools.',
  inputSchema: RegisterWithActInputSchema,
  handler: async (input: unknown) => {
    const { agent_id, name, capabilities } = RegisterWithActInputSchema.parse(input);
    try {
      const { data } = await http.post('/api/agents/register', {
        agentId: agent_id,
        name,
        capabilities
      });
      console.error(`[ACT] Registered agent: ${agent_id}`);
      return { success: true, agent_id, message: `Registered as "${name}" with capabilities: ${capabilities.join(', ')}` };
    } catch (e) {
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
