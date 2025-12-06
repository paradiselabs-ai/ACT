import {
  registerWithActTool,
  getTaskTool,
  reportTaskProgressTool,
  reportTaskCompleteTool,
  queryCoordinationMemoryTool,
  evaluateCoordinationTool,
  improveCoordinationTool,
  getImprovementStatusTool,
  type ToolDefinition
} from './act-tools.js';

export const allTools: ToolDefinition[] = [
  registerWithActTool,
  getTaskTool,
  reportTaskProgressTool,
  reportTaskCompleteTool,
  queryCoordinationMemoryTool,
  evaluateCoordinationTool,
  improveCoordinationTool,
  getImprovementStatusTool
];

export async function handleToolCall(
  toolName: string,
  input: unknown
): Promise<{ success: boolean; result?: unknown; error?: string }> {
  const tool = allTools.find(t => t.name === toolName);
  
  if (!tool) {
    return {
      success: false,
      error: `Unknown tool: ${toolName}. Available tools: ${allTools.map(t => t.name).join(', ')}`
    };
  }
  
  try {
    const result = await tool.handler(input);
    return { success: true, result };
  } catch (error: any) {
    return {
      success: false,
      error: error.message || 'Unknown error occurred'
    };
  }
}