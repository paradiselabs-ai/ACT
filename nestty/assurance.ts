/**
 * Assurance validation logic for NesTTY.
 *
 * Assurance is a Tier 1 role that validates completed agent work in two layers:
 * 1. Verify the agent's self-verification (Ralph Wiggum Loop) actually ran
 * 2. Independently score each @success_criteria item
 *
 * Gate: 95% weighted score. Below → return to agent with gap analysis.
 * Above → forward to QA/Synthesizer assembly queue.
 *
 * This module provides the prompt templates and scoring logic.
 * The actual LLM evaluation happens via the Assurance agent's PTY session.
 */

/** Validation result for a single criterion */
export interface CriterionResult {
  criterion: string;
  passed: boolean;
  score: number;       // 0-100
  feedback?: string;
}

/** Full validation verdict */
export interface ValidationVerdict {
  taskId: string;
  passed: boolean;
  overallScore: number;  // 0-100
  criteriaResults: CriterionResult[];
  selfVerificationChecked: boolean;
  selfVerificationValid: boolean;
  gaps?: string;
  feedback?: string;
  timestamp: string;
}

const PASS_THRESHOLD = 95;

/**
 * Build the validation prompt for the Assurance agent.
 * This gets injected into the Assurance agent's PTY when a task
 * is submitted for validation.
 */
export function buildValidationPrompt(task: {
  id: string;
  title?: string;
  description: string;
  successCriteria: string[];
  result: string;
  selfVerification?: string;
}): string {
  const criteriaList = task.successCriteria
    .map((c, i) => `  ${i + 1}. ${c}`)
    .join('\n');

  // Identity, scoring rubric, and validation process are in the Go system prompt
  // (act-agent/internal/llm/prompt/assurance.go). This turn prompt provides the DATA.
  return [
    `VALIDATION REQUEST — Task: ${task.title || task.id}`,
    ``,
    `## Task Description`,
    task.description,
    ``,
    `## Success Criteria`,
    criteriaList,
    ``,
    `## Agent's Output (Result)`,
    task.result.substring(0, 4000), // cap to avoid prompt explosion
    ``,
    task.selfVerification ? [
      `## Agent's Self-Verification`,
      task.selfVerification.substring(0, 2000),
      ``,
    ].join('\n') : '',
    `Validate this task now. Respond with the JSON verdict format.`,
  ].join('\n');
}

/**
 * Parse the Assurance agent's JSON response into a ValidationVerdict.
 * Returns null if parsing fails.
 */
export function parseValidationResponse(
  taskId: string,
  rawResponse: string
): ValidationVerdict | null {
  try {
    // Try to extract JSON from the response (agent might wrap it in text)
    const jsonMatch = rawResponse.match(/\{[\s\S]*"criteriaResults"[\s\S]*\}/);
    if (!jsonMatch) return null;

    const parsed = JSON.parse(jsonMatch[0]);

    const criteriaResults: CriterionResult[] = (parsed.criteriaResults || []).map(
      (cr: any) => ({
        criterion: String(cr.criterion || ''),
        passed: Boolean(cr.passed),
        score: Number(cr.score) || 0,
        feedback: cr.feedback ? String(cr.feedback) : undefined,
      })
    );

    const overallScore = Number(parsed.overallScore) || 0;

    return {
      taskId,
      passed: overallScore >= PASS_THRESHOLD,
      overallScore,
      criteriaResults,
      selfVerificationChecked: true,
      selfVerificationValid: Boolean(parsed.selfVerificationValid),
      gaps: parsed.gaps ? String(parsed.gaps) : undefined,
      feedback: parsed.feedback ? String(parsed.feedback) : undefined,
      timestamp: new Date().toISOString(),
    };
  } catch {
    return null;
  }
}

/**
 * Build a gap analysis prompt for returning failed work to an agent.
 * This tells the agent what specifically needs fixing.
 */
export function buildGapAnalysisPrompt(verdict: ValidationVerdict): string {
  const failedCriteria = verdict.criteriaResults
    .filter(cr => !cr.passed)
    .map(cr => `- ${cr.criterion} (score: ${cr.score}/100): ${cr.feedback || 'no details'}`)
    .join('\n');

  return [
    `VALIDATION FAILED — Score: ${verdict.overallScore}/100 (need ${PASS_THRESHOLD}+)`,
    ``,
    `## Failed Criteria`,
    failedCriteria || '(none individually failed, but overall score too low)',
    ``,
    verdict.gaps ? `## Gap Analysis\n${verdict.gaps}\n` : '',
    `Fix the specific gaps above, then re-submit with \`act task submit-for-validation\`.`,
  ].join('\n');
}
