/**
 * Planner turn prompt logic for NesTTY.
 *
 * The Planner is the ONLY decision-maker in ACT. This module provides
 * turn-specific prompt builders — the Planner's identity and capabilities
 * are established by the Go system prompt (act-agent/internal/llm/prompt/planner.go).
 * These functions inject DYNAMIC DATA for specific situations.
 *
 * Counterparts: assurance.ts, observer.ts, synthesizer.ts
 */

import type { NestTTYRole } from './types.js';

/** Build the session kickoff prompt. Injected when all agents are ready. */
export function buildKickoffPrompt(projectName: string, roles: NestTTYRole[]): string {
  return [
    `SESSION STARTED — Project: ${projectName}`,
    ``,
    `Agents ready: ${roles.join(', ')}`,
    ``,
    `Next steps:`,
    `1. Run \`act context --project ${projectName}\` to load project state`,
    `2. Analyze the project scope and current progress`,
    `3. Create tasks using CREATE_TASK: directives with SNLP format + @success_criteria`,
    `4. If this is a new project, start with 3-8 concrete tasks for the first phase`,
    `5. If continuing, check \`act graph unverified\` for pending work before creating new tasks`,
  ].join('\n');
}

/** Build a replan prompt when things go wrong (Observer critical reports, repeated failures). */
export function buildReplanPrompt(context: {
  observerReport?: string;
  failedTasks?: { id: string; title: string; failCount: number; lastGaps: string }[];
  currentTaskSummary?: string;
}): string {
  const lines: string[] = [
    `REPLAN NEEDED — Review the issues below and adjust the project plan.`,
    ``,
  ];

  if (context.observerReport) {
    lines.push(`## Observer Report`, context.observerReport, ``);
  }

  if (context.failedTasks && context.failedTasks.length > 0) {
    lines.push(`## Repeatedly Failing Tasks`);
    for (const t of context.failedTasks) {
      lines.push(`- "${t.title}" (${t.id.substring(0, 8)}) — failed ${t.failCount}x`);
      if (t.lastGaps) lines.push(`  Gaps: ${t.lastGaps.substring(0, 200)}`);
    }
    lines.push(``);
  }

  if (context.currentTaskSummary) {
    lines.push(`## Current Task Board`, context.currentTaskSummary, ``);
  }

  lines.push(
    `## Actions to Consider`,
    `- Reassign stuck/failing tasks to different agents`,
    `- Break complex tasks into smaller pieces`,
    `- Create bridging tasks if outputs don't connect`,
    `- Adjust priorities based on what's blocking progress`,
    `- Cancel tasks that are no longer relevant`,
  );

  return lines.join('\n');
}

/** Wrap human input with project context before sending to Planner. */
export function buildHumanRequestPrompt(
  humanInput: string,
  projectState?: { taskCounts?: Record<string, number>; activeAgents?: number; pendingIssues?: string[] }
): string {
  const lines: string[] = [
    `HUMAN INPUT: ${humanInput}`,
  ];

  if (projectState) {
    const parts: string[] = [];
    if (projectState.taskCounts) {
      const counts = Object.entries(projectState.taskCounts)
        .map(([status, count]) => `${status}: ${count}`)
        .join(', ');
      parts.push(`Tasks: ${counts}`);
    }
    if (projectState.activeAgents !== undefined) {
      parts.push(`Active agents: ${projectState.activeAgents}`);
    }
    if (parts.length > 0) {
      lines.push(``, `Context: ${parts.join(' | ')}`);
    }
    if (projectState.pendingIssues && projectState.pendingIssues.length > 0) {
      lines.push(`Pending issues: ${projectState.pendingIssues.join('; ')}`);
    }
  }

  return lines.join('\n');
}

/**
 * Extract CREATE_TASK directives from a Planner response.
 * Returns parsed task definitions ready to POST to the server.
 *
 * Supports two patterns:
 * 1. CREATE_TASK: { json } — inline directives
 * 2. { "tasks": [ ... ] } — full planning response
 */
export function parseCreateTaskDirectives(content: string): any[] {
  const tasks: any[] = [];

  // Pattern 1: CREATE_TASK: { json }
  const taskMatches = content.matchAll(/CREATE_TASK:\s*(\{[^}]+\})/g);
  for (const match of taskMatches) {
    try {
      tasks.push(JSON.parse(match[1]));
    } catch { /* malformed JSON, skip */ }
  }

  // Pattern 2: JSON block with "tasks" array
  if (tasks.length === 0) {
    const jsonMatch = content.match(/\{[\s\S]*"tasks"\s*:\s*\[[\s\S]*\]\s*\}/);
    if (jsonMatch) {
      try {
        const parsed = JSON.parse(jsonMatch[0]);
        if (Array.isArray(parsed.tasks)) {
          tasks.push(...parsed.tasks);
        }
      } catch { /* not valid JSON, skip */ }
    }
  }

  return tasks;
}
