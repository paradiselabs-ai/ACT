/**
 * QA/Synthesizer assembly logic for NesTTY.
 *
 * QA/Synthesizer is a Tier 1 role that assembles Assurance-validated
 * task outputs into a coherent final deliverable.
 *
 * Workflow:
 * 1. Receives validated task outputs from Assurance
 * 2. Maintains an assembly queue
 * 3. Assembles outputs into a coherent whole
 * 4. When pieces don't fit, uses targeted @mentions to ask agents
 * 5. Reports final deliverable to Planner
 */

/** A validated task output ready for assembly */
export interface ValidatedOutput {
  taskId: string;
  taskTitle: string;
  agentId: string;
  result: string;
  validationScore: number;
  addedAt: string;
}

/** Assembly state */
export interface AssemblyState {
  projectName: string;
  queue: ValidatedOutput[];
  assembled: ValidatedOutput[];  // outputs already integrated
  deliverable: string | null;     // current assembled deliverable text
}

/**
 * Build the synthesis prompt for the QA/Synthesizer agent.
 * Injected when new validated outputs arrive.
 */
export function buildSynthesisPrompt(state: AssemblyState): string {
  const queueSummary = state.queue
    .map((o, i) => `  ${i + 1}. [${o.taskTitle}] by ${o.agentId} (score: ${o.validationScore}/100)\n     ${o.result.substring(0, 300)}...`)
    .join('\n\n');

  const assembledSummary = state.assembled.length > 0
    ? state.assembled
        .map(o => `  - ${o.taskTitle} (integrated)`)
        .join('\n')
    : '  (none yet)';

  return [
    `SYNTHESIS REQUEST — Project: ${state.projectName}`,
    ``,
    `## Already Integrated`,
    assembledSummary,
    ``,
    `## New Validated Outputs (ready for integration)`,
    queueSummary || '  (queue empty)',
    ``,
    state.deliverable ? [
      `## Current Deliverable State`,
      state.deliverable.substring(0, 2000),
      ``,
    ].join('\n') : '',
    `## Your Task`,
    `You are QA/Synthesizer. Integrate the new validated outputs into the deliverable.`,
    ``,
    `1. Read each new output carefully`,
    `2. Check for conflicts or gaps between outputs`,
    `3. If something doesn't fit, @mention the specific agent for clarification`,
    `4. Assemble into a coherent whole`,
    `5. Report the updated deliverable status to @planner`,
    ``,
    `If all pieces fit together cleanly, respond with:`,
    `SYNTHESIS_COMPLETE: <brief summary of what was assembled>`,
    ``,
    `If you need clarification from an agent, respond with:`,
    `NEED_CLARIFICATION: @<agent-id> <your question>`,
  ].join('\n');
}

/**
 * Parse the QA/Synthesizer's response to determine next action.
 */
export function parseSynthesisResponse(response: string): {
  type: 'complete' | 'need_clarification' | 'in_progress';
  summary?: string;
  targetAgent?: string;
  question?: string;
} {
  if (response.includes('SYNTHESIS_COMPLETE:')) {
    const summary = response.split('SYNTHESIS_COMPLETE:')[1]?.trim() || '';
    return { type: 'complete', summary };
  }

  const clarMatch = response.match(/NEED_CLARIFICATION:\s*@(\S+)\s+(.*)/s);
  if (clarMatch) {
    return {
      type: 'need_clarification',
      targetAgent: clarMatch[1],
      question: clarMatch[2].trim(),
    };
  }

  return { type: 'in_progress' };
}
