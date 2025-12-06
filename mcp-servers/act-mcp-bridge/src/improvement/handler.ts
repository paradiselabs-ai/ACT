import { SelfImprovementEngine } from './SelfImprovementEngine.js';
import { ImprovementRequestSchema, type ImprovementRequest } from './types.js';

// Initialize the self-improvement engine
const improvementEngine = new SelfImprovementEngine();

/**
 * Handle improvement coordination requests from MCP tools
 * @param input The improvement request from the MCP tool
 * @returns Improvement result or error
 */
export async function handleImprovementRequest(input: unknown): Promise<{
  success: boolean;
  result?: any;
  error?: string;
}> {
  try {
    // Validate input using Zod schema
    const validatedInput = ImprovementRequestSchema.parse(input) as ImprovementRequest;
    
    console.log(`[ImprovementHandler] Processing improvement request for scope: ${validatedInput.scope}`);
    
    // Trigger explicit improvement (user-controlled)
    const result = await improvementEngine.triggerExplicitImprovement(validatedInput);
    
    // For certain scopes, also schedule background improvement
    if (shouldScheduleBackgroundImprovement(validatedInput)) {
      await improvementEngine.scheduleBackgroundImprovement(validatedInput);
    }
    
    return {
      success: true,
      result: formatOutput(result, validatedInput.output)
    };
  } catch (error: any) {
    console.error('[ImprovementHandler] Error processing improvement request:', error);
    
    return {
      success: false,
      error: error.message || 'Unknown error occurred'
    };
  }
}

/**
 * Determine if background improvement should be scheduled
 */
function shouldScheduleBackgroundImprovement(request: ImprovementRequest): boolean {
  // Schedule background improvement for performance and knowledge scopes
  return request.scope === 'performance' || request.scope === 'knowledge';
}

/**
 * Format output based on requested format
 */
function formatOutput(result: any, format: 'summary' | 'detailed-report' | 'action-items'): any {
  switch (format) {
    case 'summary':
      return {
        scope: result.scope,
        key_findings: result.analysis.substring(0, 100) + '...',
        recommendation_count: result.recommendations.length,
        confidence: result.metrics?.confidence_level
      };
    
    case 'detailed-report':
      return {
        scope: result.scope,
        analysis: result.analysis,
        recommendations: result.recommendations,
        metrics: result.metrics,
        execution_time: result.executionTime,
        timestamp: result.timestamp
      };
    
    case 'action-items':
      return {
        scope: result.scope,
        actions: result.recommendations.map((rec: string, index: number) => ({
          id: index + 1,
          description: rec,
          priority: index < 2 ? 'high' : 'medium'
        }))
      };
    
    default:
      return result;
  }
}

/**
 * Get current status of the improvement system
 */
export function getImprovementEngineStatus(): any {
  return improvementEngine.getStatus();
}

/**
 * Get agent profiles
 */
export function getAgentProfiles(): any {
  const profiles = improvementEngine.getAgentProfiles();
  const result: Record<string, any> = {};
  profiles.forEach((value, key) => {
    result[key] = value;
  });
  return result;
}

/**
 * Close the improvement engine
 */
export async function closeImprovementEngine(): Promise<void> {
  await improvementEngine.close();
}