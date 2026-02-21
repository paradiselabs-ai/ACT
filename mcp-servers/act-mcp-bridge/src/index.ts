import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from '@modelcontextprotocol/sdk/types.js';
import { zodToJsonSchema } from 'zod-to-json-schema';

import { allTools, handleToolCall } from './tools/index.js';

const SERVER_NAME = 'act-mcp-bridge';
const SERVER_VERSION = '1.0.0';

// Create MCP Server
const server = new Server(
  {
    name: SERVER_NAME,
    version: SERVER_VERSION
  },
  {
    capabilities: {
      tools: {}
    }
  }
);

// Tool listing handler
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: allTools.map(tool => ({
      name: tool.name,
      description: tool.description,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      inputSchema: zodToJsonSchema(tool.inputSchema as any) as Record<string, unknown>
    }))
  };
});

// Tool call handler
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;
  
  const result = await handleToolCall(name, args || {});
  
  return {
    content: [{
      type: 'text',
      text: JSON.stringify(result, null, 2)
    }],
    isError: !result.success
  };
});

// Server startup
async function main() {
  console.error(`[${SERVER_NAME}] Starting MCP server v${SERVER_VERSION}`);
  
  const transport = new StdioServerTransport();
  await server.connect(transport);
  
  console.error(`[${SERVER_NAME}] Server running on stdio transport`);
  console.error(`[${SERVER_NAME}] Available tools: ${allTools.map(t => t.name).join(', ')}`);
}

main().catch((error) => {
  console.error(`[${SERVER_NAME}] Fatal error:`, error);
  process.exit(1);
});