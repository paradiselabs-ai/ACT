import { z } from 'zod';

// Improvement Scope Types
export const ImprovementScope = z.enum([
  'communication',
  'tools',
  'assignments',
  'conflicts',
  'collaboration',
  'performance',
  'knowledge'
]);

export type ImprovementScopeType = z.infer<typeof ImprovementScope>;

// Output Format Types
export const OutputFormat = z.enum([
  'summary',
  'detailed-report',
  'action-items'
]);

export type OutputFormatType = z.infer<typeof OutputFormat>;

// Improvement Request Schema
export const ImprovementRequestSchema = z.object({
  scope: ImprovementScope,
  agents: z.array(z.string()).optional(),
  session: z.string().optional(),
  filter: z.string().optional(),
  output: OutputFormat.default('summary')
});

export type ImprovementRequest = z.infer<typeof ImprovementRequestSchema>;

// Improvement Result Schema
export const ImprovementResultSchema = z.object({
  requestId: z.string(),
  scope: ImprovementScope,
  analysis: z.string(),
  recommendations: z.array(z.string()),
  metrics: z.record(z.string(), z.number()).optional(),
  timestamp: z.string().datetime(),
  executionTime: z.number(), // in milliseconds
  outputFormat: OutputFormat
});

export type ImprovementResult = z.infer<typeof ImprovementResultSchema>;

// Background Task Schema
export const BackgroundTaskSchema = z.object({
  taskId: z.string(),
  type: z.enum(['improvement', 'evaluation', 'analysis']),
  priority: z.enum(['low', 'medium', 'high']),
  status: z.enum(['pending', 'running', 'completed', 'failed']),
  createdAt: z.string().datetime(),
  startedAt: z.string().datetime().optional(),
  completedAt: z.string().datetime().optional(),
  requestData: z.record(z.string(), z.any()),
  result: z.any().optional()
});

export type BackgroundTask = z.infer<typeof BackgroundTaskSchema>;