#!/usr/bin/env npx tsx
/**
 * Mock NesTTY agent — simulates act-agent --nestty for pipeline testing.
 *
 * Reads turns from stdin, writes JSON line responses to stdout.
 * Uses role-appropriate canned responses instead of LLM calls.
 *
 * Usage:
 *   echo "What's the plan?" | npx tsx nestty/mock-agent.ts --role planner
 *   MOCK_AGENT=1 npx tsx nestty/index.ts --project test --server http://localhost:8080
 */

import * as readline from 'readline';

const role = process.argv.find((a, i) => process.argv[i - 1] === '--role') || 'planner';
const bootstrapPrompt = process.argv.find((a, i) => process.argv[i - 1] === '--prompt');

function write(type: string, content: string) {
  const msg = JSON.stringify({ role, type, content, time: new Date().toISOString() });
  process.stdout.write(msg + '\n');
}

// Role-specific response generators
const RESPONSES: Record<string, (input: string) => string> = {
  planner: (input) => {
    if (input.includes('context') || input.includes('Begin') || input.includes('All agents ready')) {
      return 'Project context loaded. Waiting for user direction before creating tasks.';
    }
    if (input.includes('Observer') || input.includes('stuck') || input.includes('issue')) {
      return 'Acknowledged. Adjusting priorities based on Observer report.';
    }
    if (input.includes('Synthesis complete') || input.includes('SYNTHESIS_COMPLETE')) {
      return 'Synthesis confirmed. Project deliverable is ready for user review.';
    }
    if (input.includes('FAILED') || input.includes('gaps')) {
      return 'Validation failure noted. Reassigning task to swarm agent with gap analysis.';
    }
    if (input.includes('Human')) {
      // Human input — create tasks based on the request
      const userMsg = input.replace(/.*\[Human\]:\s*/, '').substring(0, 100);
      return `Planning "${userMsg}". Creating tasks:\n` +
        `CREATE_TASK: {"title": "Setup project structure", "description": "@task\\n- Initialize project\\n- Configure tooling\\n@success_criteria\\n- Project builds\\n- Config files present", "requiredCapabilities": ["typescript"], "priority": "high"}\n` +
        `CREATE_TASK: {"title": "Implement core feature", "description": "@task\\n- Build main functionality\\n@success_criteria\\n- Feature works end-to-end\\n- Edge cases handled", "requiredCapabilities": ["typescript"], "priority": "high"}\n` +
        `CREATE_TASK: {"title": "Write tests", "description": "@task\\n- Unit tests\\n- Integration tests\\n@success_criteria\\n- All tests pass\\n- Coverage above 80%", "requiredCapabilities": ["typescript"], "priority": "medium"}`;
    }
    return `Processing request. Will coordinate with the swarm.`;
  },
  observer: (input) => {
    if (input.includes('STATUS UPDATE') || input.includes('Detected Issues')) {
      return '@planner Issues detected in the monitoring report. Key concerns: see above. Recommend immediate attention to critical items.';
    }
    return '@planner Monitoring update: all systems nominal. No issues to report.';
  },
  assurance: (input) => {
    if (input.includes('VALIDATION REQUEST')) {
      return JSON.stringify({
        selfVerificationValid: true,
        criteriaResults: [
          { criterion: 'test criterion', passed: true, score: 96, feedback: 'Meets requirements' }
        ],
        overallScore: 96,
        passed: true,
        gaps: null,
        feedback: 'Work meets quality standards. Approved for synthesis.'
      });
    }
    return 'Ready for validation tasks. Awaiting submissions.';
  },
  qa: (input) => {
    if (input.includes('SYNTHESIS REQUEST') || input.includes('New Validated Outputs')) {
      return 'SYNTHESIS_COMPLETE: All validated outputs integrated successfully. Deliverable assembled and ready for review.';
    }
    if (input.includes('validated') || input.includes('assembly')) {
      return 'SYNTHESIS_COMPLETE: Final deliverable assembled from validated outputs.';
    }
    return 'QA/Synthesizer ready. Awaiting validated work for assembly.';
  },
};

// Signal ready
write('ready', '');

// Process bootstrap prompt
if (bootstrapPrompt) {
  const respond = RESPONSES[role] || RESPONSES.planner;
  write('message', respond(bootstrapPrompt));
}

// Read turns from stdin
const rl = readline.createInterface({ input: process.stdin });

rl.on('line', (line) => {
  const trimmed = line.trim();
  if (!trimmed) return;
  if (trimmed === '__EXIT__') {
    write('exit', 'Session ended');
    process.exit(0);
  }

  const respond = RESPONSES[role] || RESPONSES.planner;
  write('message', respond(trimmed));
});

rl.on('close', () => {
  write('exit', 'stdin closed');
  process.exit(0);
});
