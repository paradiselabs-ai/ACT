#!/usr/bin/env npx tsx
/**
 * NesTTY — Multi-agent conversation interface.
 *
 * Usage:
 *   npx tsx nestty/index.ts --project my-app
 *   npx tsx nestty/index.ts --project my-app --roles planner,observer
 *
 * The terminal IS the NesTTY window:
 * - Agent messages print as [Role]: message
 * - You type at the $> prompt
 * - Your input goes to the Planner
 */

import * as readline from 'readline';
import { io as ioClient, type Socket } from 'socket.io-client';
import { Orchestrator } from './orchestrator.js';
import { ProcessManager } from './process-manager.js';
import { getBootstrapPrompt } from './bootstrap.js';
import type { NestTTYRole, OrchestratorEvent } from './types.js';
import { ALL_ROLES } from './types.js';

// ─── CLI args ──────────────────────────────────────────────────────────────

function parseArgs(): { projectName: string; roles: NestTTYRole[]; serverUrl: string } {
  const args = process.argv.slice(2);
  let projectName = '';
  let roles: NestTTYRole[] = [...ALL_ROLES];
  let serverUrl = process.env.ACT_SERVER_URL || 'http://localhost:8080';

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--project' && args[i + 1]) {
      projectName = args[++i];
    } else if (args[i] === '--roles' && args[i + 1]) {
      roles = args[++i].split(',').map(r => r.trim().toLowerCase()) as NestTTYRole[];
    } else if (args[i] === '--server' && args[i + 1]) {
      serverUrl = args[++i];
    }
  }

  if (!projectName) {
    console.error('Usage: npx tsx nestty/index.ts --project <name> [--roles planner,observer,...] [--server http://...]');
    process.exit(1);
  }

  return { projectName, roles, serverUrl };
}

// ─── Display ───────────────────────────────────────────────────────────────

const ROLE_COLORS: Record<string, string> = {
  planner: '\x1b[36m',    // cyan
  observer: '\x1b[33m',   // yellow
  assurance: '\x1b[35m',  // magenta
  qa: '\x1b[32m',         // green
  human: '\x1b[37m',      // white
  system: '\x1b[90m',     // gray
};
const RESET = '\x1b[0m';

function printMessage(role: string, content: string): void {
  const color = ROLE_COLORS[role] || RESET;
  const label = role.charAt(0).toUpperCase() + role.slice(1);
  console.log(`${color}[${label}]${RESET}: ${content}`);
}

function printSystem(msg: string): void {
  console.log(`\x1b[90m${msg}${RESET}`);
}

// ─── Main ──────────────────────────────────────────────────────────────────

async function main() {
  const { projectName, roles, serverUrl } = parseArgs();

  printSystem(`\n  ACT — ${projectName}    [${roles.length} agents: ${roles.join(', ')}]`);
  printSystem(`  Server: ${serverUrl}\n`);

  const orchestrator = new Orchestrator({
    projectName,
    roles,
    turnTimeoutMs: 120_000, // 2 min per turn
    serverUrl,
  });

  // Subscribe to orchestrator events → print to terminal
  orchestrator.on('event', (event: OrchestratorEvent) => {
    switch (event.type) {
      case 'agent_ready':
        printSystem(`  ${event.role} is ready`);
        if (orchestrator.allReady()) {
          printSystem(`  All agents ready. Session starting.\n`);
        }
        break;
      case 'turn_message':
        if (event.role && event.content) {
          printMessage(event.role, event.content);
        }
        break;
      case 'error':
        printSystem(`  [error] ${event.role}: ${event.content}`);
        break;
      case 'shutdown':
        printSystem('\n  Session ended.');
        break;
    }
  });

  // Register agents
  for (const role of roles) {
    orchestrator.addAgent(role);
  }

  // Wire process manager — spawns agents as child processes
  // (PTYManager via node-pty is preferred but broken on Node.js v25+)
  const procMgr = new ProcessManager((msg) => orchestrator.handleMessage(msg));
  orchestrator.setPTYManager(procMgr);

  printSystem('  Spawning agents...\n');
  for (const role of roles) {
    const bootstrap = getBootstrapPrompt(role, projectName);
    try {
      procMgr.spawn(role, projectName, bootstrap);
      printSystem(`  Spawning ${role}...`);
    } catch (err: any) {
      printSystem(`  Failed to spawn ${role}: ${err.message}`);
    }
  }

  orchestrator.start();

  // ─── Socket.io event listeners (real-time push) ─────────────────────────
  // Connect to ACT server for instant event notifications
  let socket: Socket | null = null;
  try {
    socket = ioClient(serverUrl, { reconnection: true, reconnectionDelay: 5000 });
    socket.on('connect', () => printSystem('  Connected to ACT server (Socket.io)'));
    socket.on('disconnect', () => printSystem('  Disconnected from ACT server'));

    // Validation submitted → route to Assurance (instant, no polling delay)
    if (roles.includes('assurance')) {
      socket.on('task_submitted_for_validation', async (event: any) => {
        try {
          const res = await fetch(`${serverUrl}/api/tasks/pending-validation`);
          const data = await res.json() as any;
          const tasks = data.tasks || [];
          const task = tasks.find((t: any) => t.id === event.taskId) || tasks[0];
          if (task) {
            orchestrator.routeToAssurance({
              id: task.id,
              title: task.title,
              description: task.description || '',
              successCriteria: task.successCriteria || [],
              result: task.metadata?.result || task.result || '',
              selfVerification: task.metadata?.selfVerification,
            });
          }
        } catch { /* server unreachable */ }
      });
    }

    // Task validated → route to QA/Synthesizer (instant)
    if (roles.includes('qa')) {
      socket.on('task_validated', async (event: any) => {
        try {
          const res = await fetch(`${serverUrl}/api/tasks/${event.taskId}`);
          const data = await res.json() as any;
          const task = data.task;
          if (!task) return;

          const state = orchestrator.getAssemblyState();
          const alreadyQueued = state.queue.some(o => o.taskId === task.id) ||
                                state.assembled.some(o => o.taskId === task.id);
          if (alreadyQueued) return;

          orchestrator.routeToSynthesizer({
            taskId: task.id,
            taskTitle: task.title || task.description?.substring(0, 60) || task.id,
            agentId: task.assignedAgent || 'unknown',
            result: task.metadata?.result || '',
            validationScore: task.metadata?.validationVerdict?.score || event.score || 0,
            addedAt: new Date().toISOString(),
          });
        } catch { /* server unreachable */ }
      });
    }
  } catch {
    printSystem('  Warning: Could not connect Socket.io — falling back to polling');
  }

  // ─── Fallback polling (if Socket.io fails or for initial catch-up) ──────
  const validationPollMs = 30_000; // 30s fallback (was 15s when primary)
  let validationTimer: ReturnType<typeof setInterval> | null = null;
  let synthesisTimer: ReturnType<typeof setInterval> | null = null;

  if (roles.includes('assurance')) {
    validationTimer = setInterval(async () => {
      try {
        const res = await fetch(`${serverUrl}/api/tasks/pending-validation`);
        const data = await res.json() as any;
        const tasks = data.tasks || [];
        for (const task of tasks) {
          orchestrator.routeToAssurance({
            id: task.id,
            title: task.title,
            description: task.description || '',
            successCriteria: task.successCriteria || [],
            result: task.metadata?.result || task.result || '',
            selfVerification: task.metadata?.selfVerification,
          });
          break;
        }
      } catch { /* silent */ }
    }, validationPollMs);
  }

  if (roles.includes('qa')) {
    synthesisTimer = setInterval(async () => {
      try {
        const res = await fetch(`${serverUrl}/api/tasks/validated`);
        const data = await res.json() as any;
        const tasks = data.tasks || [];
        for (const task of tasks) {
          const state = orchestrator.getAssemblyState();
          const alreadyQueued = state.queue.some(o => o.taskId === task.id) ||
                                state.assembled.some(o => o.taskId === task.id);
          if (alreadyQueued) continue;

          orchestrator.routeToSynthesizer({
            taskId: task.id,
            taskTitle: task.title || task.description?.substring(0, 60) || task.id,
            agentId: task.assignedAgent || 'unknown',
            result: task.metadata?.result || '',
            validationScore: task.metadata?.validationVerdict?.score || task.metadata?.validationScore || 0,
            addedAt: new Date().toISOString(),
          });
          break;
        }
      } catch {
        // Server unreachable — silent
      }
    }, validationPollMs);
  }

  // ─── Readline prompt ────────────────────────────────────────────────────
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    prompt: '$> ',
  });

  rl.prompt();

  rl.on('line', (line: string) => {
    const trimmed = line.trim();
    if (!trimmed) {
      rl.prompt();
      return;
    }

    // Slash commands
    if (trimmed === '/quit' || trimmed === '/exit') {
      if (validationTimer) clearInterval(validationTimer);
      if (synthesisTimer) clearInterval(synthesisTimer);
      if (socket) socket.disconnect();
      orchestrator.shutdown();
      rl.close();
      return;
    }

    if (trimmed === '/status') {
      orchestrator.triggerObserverCheck();
      printSystem('  Observer check triggered.');
      rl.prompt();
      return;
    }

    if (trimmed === '/help') {
      printSystem('  Commands:');
      printSystem('    /status    — trigger Observer status check');
      printSystem('    /quit      — shutdown session');
      printSystem('    /help      — show this help');
      printSystem('    (anything else is sent to the Planner)');
      rl.prompt();
      return;
    }

    // Human input → orchestrator
    printMessage('human', trimmed);
    orchestrator.handleHumanInput(trimmed);
    rl.prompt();
  });

  rl.on('close', () => {
    orchestrator.shutdown();
    process.exit(0);
  });

  // Graceful shutdown
  process.on('SIGINT', () => {
    if (validationTimer) clearInterval(validationTimer);
    orchestrator.shutdown();
    rl.close();
  });
}

main().catch(err => {
  console.error('Fatal:', err);
  process.exit(1);
});
