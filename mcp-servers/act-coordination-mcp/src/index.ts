#!/usr/bin/env node
/**
 * ACT Coordination MCP Server
 * 
 * Multi-agent coordination via shared JSON file.
 * 
 * THE BOOTSTRAP IRONY:
 * This MCP server coordinates agents building ACT - the system that will replace
 * this MCP. Once ACT's Phase 5 is complete, this manual coordination file becomes
 * the formalized PVM coordination system. We're proving ACT works by using a crude
 * version to build the real thing.
 * 
 * @author ACT Development Team
 * @version 1.0.0
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  type CallToolRequest,
  type ListToolsRequest
} from '@modelcontextprotocol/sdk/types.js';
import { z } from 'zod';

import { allTools, handleToolCall } from './tools/index.js';
import { CoordinationFileService } from './services/coordination-file.js';
// CoordinationError is used in services and tools, not directly here

// ============================================================================
// Server Configuration
// ============================================================================

const SERVER_NAME = 'act-coordination-mcp';
const SERVER_VERSION = '1.0.0';

// Get coordination file path from environment or use default
const coordinationFilePath = process.env.ACT_COORDINATION_FILE 
  || '/Users/user/Documents/Developer/dev/AI/act/act-coordination.json';

// ============================================================================
// Initialize Services
// ============================================================================

const fileService = new CoordinationFileService(coordinationFilePath);

// ============================================================================
// Create MCP Server
// ============================================================================

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

// ============================================================================
// Tool Listing Handler
// ============================================================================

server.setRequestHandler(ListToolsRequestSchema, async (_request: ListToolsRequest) => {
  return {
    tools: allTools.map(tool => ({
      name: tool.name,
      description: tool.description,
      inputSchema: zodToJsonSchema(tool.inputSchema),
      annotations: tool.annotations
    }))
  };
});

// ============================================================================
// Tool Call Handler
// ============================================================================

server.setRequestHandler(CallToolRequestSchema, async (request: CallToolRequest) => {
  const { name, arguments: args } = request.params;
  
  // Verify file exists before handling any tool
  const fileExists = await fileService.fileExists();
  if (!fileExists) {
    return {
      content: [{
        type: 'text',
        text: JSON.stringify({
          success: false,
          error: {
            code: 'FILE_NOT_FOUND',
            message: `Coordination file not found: ${fileService.getFilePath()}`,
            suggestion: 'Verify ACT_COORDINATION_FILE environment variable is set correctly'
          }
        }, null, 2)
      }],
      isError: true
    };
  }
  
  const result = await handleToolCall(name, args || {}, fileService);
  
  return {
    content: [{
      type: 'text',
      text: JSON.stringify(result.success ? result.result : result, null, 2)
    }],
    isError: !result.success
  };
});

// ============================================================================
// Zod to JSON Schema Converter
// ============================================================================

function zodToJsonSchema(schema: z.ZodType<unknown>): Record<string, unknown> {
  // Handle ZodObject
  if (schema instanceof z.ZodObject) {
    const shape = schema.shape as Record<string, z.ZodType<unknown>>;
    const properties: Record<string, unknown> = {};
    const required: string[] = [];
    
    for (const [key, value] of Object.entries(shape)) {
      properties[key] = zodToJsonSchema(value);
      
      // Check if field is required (not optional, not with default)
      if (!(value instanceof z.ZodOptional) && !(value instanceof z.ZodDefault)) {
        required.push(key);
      }
    }
    
    return {
      type: 'object',
      properties,
      required: required.length > 0 ? required : undefined,
      additionalProperties: false
    };
  }
  
  // Handle ZodString
  if (schema instanceof z.ZodString) {
    const result: Record<string, unknown> = { type: 'string' };
    if (schema.description) result.description = schema.description;
    return result;
  }
  
  // Handle ZodNumber
  if (schema instanceof z.ZodNumber) {
    const result: Record<string, unknown> = { type: 'number' };
    if (schema.description) result.description = schema.description;
    return result;
  }
  
  // Handle ZodBoolean
  if (schema instanceof z.ZodBoolean) {
    const result: Record<string, unknown> = { type: 'boolean' };
    if (schema.description) result.description = schema.description;
    return result;
  }
  
  // Handle ZodEnum
  if (schema instanceof z.ZodEnum) {
    return {
      type: 'string',
      enum: schema.options,
      description: schema.description
    };
  }
  
  // Handle ZodArray
  if (schema instanceof z.ZodArray) {
    return {
      type: 'array',
      items: zodToJsonSchema(schema.element),
      description: schema.description
    };
  }
  
  // Handle ZodOptional
  if (schema instanceof z.ZodOptional) {
    return zodToJsonSchema(schema.unwrap());
  }
  
  // Handle ZodDefault
  if (schema instanceof z.ZodDefault) {
    const inner = zodToJsonSchema(schema._def.innerType as z.ZodType<unknown>);
    return {
      ...inner,
      default: schema._def.defaultValue()
    };
  }
  
  // Fallback
  return { type: 'object' };
}

// ============================================================================
// Server Startup
// ============================================================================

async function main() {
  // Log startup info to stderr (stdout is reserved for MCP protocol)
  console.error(`[${SERVER_NAME}] Starting MCP server v${SERVER_VERSION}`);
  console.error(`[${SERVER_NAME}] Coordination file: ${coordinationFilePath}`);
  
  // Verify coordination file exists
  const exists = await fileService.fileExists();
  if (!exists) {
    console.error(`[${SERVER_NAME}] WARNING: Coordination file not found!`);
    console.error(`[${SERVER_NAME}] Expected: ${coordinationFilePath}`);
    console.error(`[${SERVER_NAME}] Set ACT_COORDINATION_FILE env var to override`);
  } else {
    console.error(`[${SERVER_NAME}] Coordination file found ✓`);
  }
  
  // Create stdio transport and connect
  const transport = new StdioServerTransport();
  await server.connect(transport);
  
  console.error(`[${SERVER_NAME}] Server running on stdio transport`);
  console.error(`[${SERVER_NAME}] Available tools: ${allTools.map(t => t.name).join(', ')}`);
}

// ============================================================================
// Error Handling
// ============================================================================

process.on('uncaughtException', (error) => {
  console.error(`[${SERVER_NAME}] Uncaught exception:`, error);
  process.exit(1);
});

process.on('unhandledRejection', (reason) => {
  console.error(`[${SERVER_NAME}] Unhandled rejection:`, reason);
  process.exit(1);
});

// ============================================================================
// Run
// ============================================================================

main().catch((error) => {
  console.error(`[${SERVER_NAME}] Fatal error:`, error);
  process.exit(1);
});
