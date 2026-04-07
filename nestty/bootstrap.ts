/**
 * Minimal bootstrap prompts for NesTTY agents.
 *
 * When an agent spawns, it receives this prompt as its first turn.
 * Identity, capabilities, CLI commands, and constraints are established by
 * the Go system prompt (act-agent/internal/llm/prompt/*.go).
 *
 * The bootstrap just provides the project name and initial instruction.
 * Keep this minimal — the system prompt has everything else.
 */

import type { NestTTYRole } from './types.js';

/**
 * Generate the bootstrap prompt for a given role and project.
 * This is passed as --prompt to act-agent and processed as the first turn.
 */
export function getBootstrapPrompt(role: NestTTYRole, projectName: string): string {
  const roleHint = ROLE_HINTS[role] || 'Report ready when context is loaded.';

  return [
    `Project: ${projectName}`,
    ``,
    `Load context: act context --project ${projectName}`,
    roleHint,
    ``,
    `Report ready when done.`,
  ].join('\n');
}

/**
 * Role-specific first-action hints. NOT identity — just what to do first.
 * The Go system prompt already establishes who they are and how they work.
 */
const ROLE_HINTS: Record<NestTTYRole, string> = {
  planner: 'Then analyze the project and prepare a task breakdown.',
  observer: 'Then run `act status` to establish a baseline.',
  assurance: 'Then check `act validation queue` for pending work.',
  qa: 'Then check what outputs have been validated so far.',
};
