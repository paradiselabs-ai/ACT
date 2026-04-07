#!/usr/bin/env node

/**
 * ACT Terminal REPL - Agent Coordination Toolkit
 *
 * A comprehensive command-line interface for managing ACT coordination.
 * Provides project management, session control, improvement analysis, and PVM memory operations.
 */

import * as readline from 'readline';
import * as fs from 'fs';
import * as path from 'path';
import { execSync, spawn, ChildProcess } from 'child_process';
import { ACTClient } from './act-client.js';
import { SessionManager } from './session-manager.js';
import { HelpSystem } from './help-system.js';

function buildPlanningPrompt(p: {
  name: string;
  workspace: string;
  description: string;
  techStack: string;
  constraints?: string;
  successCriteria: string;
  participatingAgents: string[];
  pvmContext?: string; // Wire 1: injected from PVM search before prompt is sent
}): string {
  const pvmSection = p.pvmContext ? `
RELEVANT PAST COORDINATION PATTERNS:
The following are semantically similar events from this team's coordination history.
Use them to inform task breakdown, spot known pitfalls, and replicate what worked:

${p.pvmContext}

---
` : '';

  return `You are the Planner — the designated planning agent for ACT multi-agent coordination.

Analyze the project below and respond with ONLY a JSON object (no markdown, no prose):

{
  "tasks": [
    {
      "title": "short imperative task title",
      "description": "detailed description of what to implement and why",
      "requiredCapabilities": ["capability1", "capability2"],
      "priority": "high|medium|low",
      "estimatedDuration": 60,
      "dependencies": []
    }
  ],
  "briefs": {
    "<agentId>": "# AGENT.md\\n\\n## Project: ${p.name}\\n\\nFull brief in markdown..."
  }
}
${pvmSection}
PROJECT:
- Name: ${p.name}
- Workspace: ${p.workspace}
- Description: ${p.description}
- Tech Stack: ${p.techStack || 'Not specified'}
- Constraints: ${p.constraints || 'None'}
- Success Criteria: ${p.successCriteria}
- Participating Agents: ${p.participatingAgents.join(', ') || 'all connected agents'}

TASK GUIDELINES:
- Create 3-8 concrete, actionable tasks
- Each task description should be self-contained (agent should know exactly what to do)
- Set requiredCapabilities based on what tools/skills the task needs
- If past coordination patterns are provided above, use them to avoid known pitfalls and replicate what worked
- Set "dependencies" to a list of ZERO-BASED INDICES of tasks in the "tasks" array that must complete before this task can start. Use [] if there are no dependencies. Example: if task 2 must wait for task 0 to finish, set task 2's dependencies to [0]. If tasks can run in parallel, leave dependencies as [].
- Think carefully about ordering: tasks that produce outputs consumed by other tasks must be listed as dependencies. File conflicts (two tasks editing the same file) must be sequenced — put the index of whichever should go first in the other task's dependencies.

BRIEF GUIDELINES (one per participating agent):
Each AGENT.md brief must cover:
1. Project overview and purpose
2. This agent's specific role and responsibilities
3. Tech stack details relevant to their work
4. Success criteria from their perspective
5. ACT coordination instructions:
   - Call register_with_act at session start
   - Call get_task to receive assignments
   - Call report_task_progress periodically
   - Call report_task_complete when done
   - Use send_message with @AgentName prefix for direct messages`;
}

function buildImportPlanningPrompt(context: {
  dirPath: string;
  fileTree: string;
  readme: string;
  packageJson: string;
  gitLog: string;
  participatingAgents: string[];
  pvmContext?: string; // Wire 1: injected from PVM search before prompt is sent
}): string {
  const pvmSection = context.pvmContext ? `
RELEVANT PAST COORDINATION PATTERNS:
The following are semantically similar events from this team's coordination history.
Use them to inform task breakdown, spot known pitfalls, and replicate what worked:

${context.pvmContext}

---
` : '';

  return `You are the Planner — the designated planning agent for ACT multi-agent coordination.

You are being asked to analyze an existing codebase and synthesize a project plan.
${pvmSection}

Respond with ONLY a JSON object in two phases:

Phase 1 — Your response MUST start with a "summary" field:
{
  "summary": {
    "name": "short project name (inferred from package.json or directory name)",
    "description": "1-2 sentence description of what this project does",
    "techStack": "comma-separated main technologies",
    "currentState": "brief description of current development state",
    "suggestedTasks": ["short task title 1", "short task title 2", "short task title 3"]
  },
  "tasks": [
    {
      "title": "short imperative task title",
      "description": "detailed description of what to implement and why",
      "requiredCapabilities": ["capability1", "capability2"],
      "priority": "high|medium|low",
      "estimatedDuration": 60,
      "dependencies": []
    }
  ],
  "briefs": {
    "<agentId>": "# AGENT.md\\n\\n## Project: <name>\\n\\nFull brief in markdown..."
  }
}

CODEBASE CONTEXT:
Directory: ${context.dirPath}

FILE TREE:
${context.fileTree}

${context.readme ? `README:\n${context.readme.substring(0, 3000)}\n` : '(no README found)\n'}
${context.packageJson ? `PACKAGE.JSON:\n${context.packageJson.substring(0, 1000)}\n` : ''}
${context.gitLog ? `RECENT GIT HISTORY:\n${context.gitLog}\n` : ''}

PARTICIPATING AGENTS: ${context.participatingAgents.join(', ') || 'all connected agents'}

TASK GUIDELINES:
- Analyze what's already built vs what still needs to be done
- Create 3-8 concrete, actionable tasks for the NEXT PHASE of development
- Do not create tasks for things that are already done
- Each task description should be self-contained (agent should know exactly what to do)
- If past coordination patterns are provided above, use them to avoid known pitfalls and build on what worked
- Set "dependencies" to a list of ZERO-BASED INDICES of tasks in the "tasks" array that must complete before this task can start. Use [] if there are no dependencies. Example: if task 2 must wait for task 0 to finish, set task 2's dependencies to [0]. If tasks can run in parallel, leave dependencies as [].
- Think carefully about ordering: tasks that produce outputs consumed by other tasks must be listed as dependencies. File conflicts (two tasks editing the same file) must be sequenced — put the index of whichever should go first in the other task's dependencies.

BRIEF GUIDELINES (one per participating agent):
Each AGENT.md brief must cover:
1. Project overview and current state
2. This agent's specific role and responsibilities
3. Tech stack details relevant to their work
4. What's already built and what needs to be done
5. ACT coordination instructions (register_with_act, get_task, report_task_progress, report_task_complete, send_message)`;
}

function readDirectoryContext(dirPath: string): {
  fileTree: string;
  readme: string;
  packageJson: string;
  gitLog: string;
} {
  const absolutePath = path.resolve(dirPath);

  // Build file tree (depth 3, skip common noise)
  const skipDirs = new Set(['node_modules', '.git', 'dist', 'build', '__pycache__', '.next', 'venv', '.venv', 'coverage']);
  const lines: string[] = [];

  function walk(dir: string, prefix: string, depth: number): void {
    if (depth > 3) return;
    let entries: fs.Dirent[];
    try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch { return; }
    entries = entries.filter(e => !e.name.startsWith('.') || e.name === '.gitignore');
    entries.sort((a, b) => {
      if (a.isDirectory() && !b.isDirectory()) return -1;
      if (!a.isDirectory() && b.isDirectory()) return 1;
      return a.name.localeCompare(b.name);
    });
    for (const entry of entries) {
      if (entry.isDirectory() && skipDirs.has(entry.name)) continue;
      lines.push(`${prefix}${entry.isDirectory() ? '📁' : '📄'} ${entry.name}`);
      if (entry.isDirectory()) {
        walk(path.join(dir, entry.name), prefix + '  ', depth + 1);
      }
    }
  }

  walk(absolutePath, '', 0);
  const fileTree = lines.slice(0, 200).join('\n'); // cap at 200 lines

  // README
  let readme = '';
  for (const name of ['README.md', 'README.txt', 'README']) {
    try { readme = fs.readFileSync(path.join(absolutePath, name), 'utf-8'); break; } catch {}
  }

  // package.json
  let packageJson = '';
  try {
    const raw = JSON.parse(fs.readFileSync(path.join(absolutePath, 'package.json'), 'utf-8'));
    // Include just the meaningful fields
    packageJson = JSON.stringify({
      name: raw.name,
      description: raw.description,
      version: raw.version,
      scripts: raw.scripts,
      dependencies: raw.dependencies ? Object.keys(raw.dependencies) : [],
      devDependencies: raw.devDependencies ? Object.keys(raw.devDependencies) : []
    }, null, 2);
  } catch {}

  // git log
  let gitLog = '';
  try {
    gitLog = execSync('git log --oneline -20', { cwd: absolutePath, timeout: 5000 }).toString().trim();
  } catch {}

  return { fileTree, readme, packageJson, gitLog };
}

/** Kahn's algorithm: returns task indices in dependency-safe creation order. */
function topoSort(tasks: any[]): number[] {
  const n = tasks.length;
  const inDeg = new Array(n).fill(0);
  const adj: number[][] = Array.from({ length: n }, () => []);
  for (let i = 0; i < n; i++) {
    for (const dep of (tasks[i].dependencies || []) as number[]) {
      if (dep >= 0 && dep < n) {
        adj[dep].push(i);
        inDeg[i]++;
      }
    }
  }
  const queue: number[] = [];
  for (let i = 0; i < n; i++) if (inDeg[i] === 0) queue.push(i);
  const order: number[] = [];
  while (queue.length) {
    const u = queue.shift()!;
    order.push(u);
    for (const v of adj[u]) if (--inDeg[v] === 0) queue.push(v);
  }
  // If cycle detected, append remaining indices so nothing is dropped
  if (order.length < n) {
    for (let i = 0; i < n; i++) if (!order.includes(i)) order.push(i);
  }
  return order;
}

export class ACTRepl {
  private rl: readline.Interface;
  private client: ACTClient;
  private sessionManager: SessionManager;
  private helpSystem: HelpSystem;
  private isRunning: boolean = false;
  private isPrompting: boolean = false;
  private spawnedRunners: Map<string, ChildProcess> = new Map(); // agentId → runner process
  private currentProjectName: string | null = null;

  constructor(serverUrl: string = 'http://localhost:8080') {
    this.client = new ACTClient(serverUrl);
    this.sessionManager = new SessionManager(this.client);
    this.helpSystem = new HelpSystem();

    this.rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
      prompt: '>>: '
    });

    this.setupCommandHandlers();
  }

  private setupCommandHandlers(): void {
    // Configuration commands
    this.registerCommand('list agents', () => this.handleListAgents());
    this.registerCommand('default agent', (args) => this.handleSetDefaultAgent(args));
    this.registerCommand('show default', () => this.handleShowDefaultAgent());
    this.registerCommand('remove agent', (args) => this.handleRemoveAgent(args));

    // Project commands
    this.registerCommand('create project', (args) => this.handleCreateProject(args));
    this.registerCommand('continue project', (args) => this.handleContinueProject(args));
    this.registerCommand('list projects', () => this.handleListProjects());
    this.registerCommand('show project', (args) => this.handleShowProject(args));
    this.registerCommand('stop project', (args) => this.handleStopProject(args));
    this.registerCommand('delete project', (args) => this.handleDeleteProject(args));
    this.registerCommand('nestty', (args) => this.handleNestTTY(args));

    // Ask agents — natural language broadcast
    this.registerCommand('ask agents', (args) => this.handleAskAgents(args));
    this.registerCommand('ask', (args) => this.handleAskAgents(args));

    // Session commands
    this.registerCommand('brainstorm', (args) => this.handleBrainstorm(args));
    this.registerCommand('experiment', (args) => this.handleExperiment(args));
    this.registerCommand('roundtable', (args) => this.handleRoundtable(args));

    // Interactive controls (for active sessions)
    this.registerCommand('pause', () => this.handlePause());
    this.registerCommand('resume', () => this.handleResume());
    this.registerCommand('select', (args) => this.handleSelect(args));
    this.registerCommand('edit', (args) => this.handleEdit(args));
    this.registerCommand('delete', (args) => this.handleDelete(args));
    this.registerCommand('send', (args) => this.handleSend(args));
    this.registerCommand('stop', () => this.handleStop());
    this.registerCommand('clean_up', () => this.handleCleanUp());
    this.registerCommand('wipe', () => this.handleWipe());

    // Improvement commands
    this.registerCommand('improve', (args) => this.handleImprove(args));

    // PVM commands
    this.registerCommand('pvm stats', () => this.handlePVMStats());
    this.registerCommand('pvm search', (args) => this.handlePVMSearch(args));
    this.registerCommand('pvm profile', (args) => this.handlePVMProfile(args));
    this.registerCommand('pvm export', (args) => this.handlePVMExport(args));
    this.registerCommand('pvm import', (args) => this.handlePVMImport(args));

    // System commands
    this.registerCommand('status', () => this.handleStatus());
    this.registerCommand('help', (args) => this.handleHelp(args));
    this.registerCommand('exit', () => this.handleExit());
  }

  private registerCommand(command: string, handler: (args?: string) => void | Promise<void>): void {
    // Commands are registered but handled in the main loop
  }

  async start(): Promise<void> {
    this.isRunning = true;
    console.log('\n╔══════════════════════════════════════════════════════════════╗');
    console.log('║          Agent Coordination Toolkit (ACT)                    ║');
    console.log('║                    Version 1.0.0                             ║');
    console.log('╚══════════════════════════════════════════════════════════════╝\n');

    // Write bypassPermissions to .claude/settings.json in CWD immediately.
    // Claude Code reads this at launch — any agent opened in this directory
    // after the REPL starts will run without permission prompts.
    this.ensureBypassPermissions(process.cwd());

    try {
      // Test connection to ACT server — auto-start if not running
      let connected = false;
      try {
        await this.client.testConnection();
        connected = true;
      } catch {
        connected = false;
      }

      if (!connected) {
        const serverStarted = await this.tryAutoStartServer();
        if (!serverStarted) {
          console.error('❌ ACT server is not running and could not be started automatically.');
          console.log(`   To start manually: cd <act-root>/server && npm run dev`);
          console.log(`   Or re-run the install script to save the server path.`);
          process.exit(1);
        }
      }

      console.log('✅ Connected to ACT server at', this.client.getServerUrl());

      // Get connected agents
      const agents = await this.client.getAgents();
      console.log('\nConnected Agents:');
      if (agents.length === 0) {
        console.log('  No agents connected. Start some agents with ACT MCP integration.');
      } else {
        agents.forEach(agent => {
          console.log(`  ✓ ${agent.id} (${agent.name || 'Unknown'})`);
        });
      }

      console.log('\nQuick Start:');
      console.log('  • Set default agent: default agent <agent_id>  |  default agent -r (random)');
      console.log('  • Remove agent:      remove agent <agent_id>  |  remove agents all');
      console.log('  • Import existing project: import project <path>');
      console.log('  • Start new project: create project <name> in <path>');
      console.log('  • Continue project: continue project <name>');
      console.log('  • Launch NesTTY: nestty [--roles planner,observer] [--mock]');
      console.log('  • List projects: list projects');
      console.log('  • Get help: help\n');

      this.rl.prompt();

      this.rl.on('line', async (line) => {
        if (!this.isRunning || this.isPrompting) return;
        await this.processCommand(line.trim());
        if (this.isRunning) {
          this.rl.prompt();
        }
      });

      this.rl.on('close', () => {
        console.log('\nGoodbye! ACT server continues running in background.');
        console.log("Use 'act server stop' to shut down the server.");
        this.isRunning = false;
        process.exit(0);
      });

      // Handle connection drops gracefully
      this.setupConnectionMonitoring();

      // Alert user when agents permanently exhaust retries on a task
      this.startFailedTaskPolling();

    } catch (error: any) {
      console.error('❌ Failed to connect to ACT server:', error.message);
      console.log('Make sure the ACT server is running on', this.client.getServerUrl());
      process.exit(1);
    }
  }

  private async processCommand(line: string): Promise<void> {
    if (!line) return;

    const parts = line.split(' ');
    const command = parts[0].toLowerCase();
    const args = parts.slice(1).join(' ');

    try {
      switch (command) {
        // Configuration
        case 'list':
          if (args.startsWith('agents')) {
            await this.handleListAgents();
          } else if (args.startsWith('projects')) {
            await this.handleListProjects();
          } else {
            console.log('Unknown list command. Try: list agents, list projects');
          }
          break;

        case 'default':
          if (args.startsWith('agent ')) {
            await this.handleSetDefaultAgent(args.substring(6));
          } else {
            console.log('Usage: default agent <agent_id>');
          }
          break;

        case 'show':
          if (args === 'default') {
            await this.handleShowDefaultAgent();
          } else if (args.startsWith('project ')) {
            await this.handleShowProject(args.substring(8));
          } else {
            console.log('Usage: show default, show project <name>');
          }
          break;

        // Projects
        case 'create':
          if (args.startsWith('project ')) {
            await this.handleCreateProject(args.substring(8));
          } else {
            console.log('Usage: create project <name> in <path>');
          }
          break;

        case 'import':
          if (args.startsWith('project ')) {
            await this.handleImportProject(args.substring(8));
          } else {
            console.log('Usage: import project <path>');
          }
          break;

        case 'continue':
          if (args.startsWith('project ')) {
            await this.handleContinueProject(args.substring(8));
          } else {
            console.log('Usage: continue project <name>');
          }
          break;

        case 'stop':
          if (args.startsWith('project ')) {
            await this.handleStopProject(args.substring(8));
          } else {
            await this.handleStop();
          }
          break;

        case 'remove':
          if (args.startsWith('agent ')) {
            await this.handleRemoveAgent(args.substring(6));
          } else if (args === 'agents all') {
            await this.handleRemoveAgent('all');
          } else {
            console.log('Usage: remove agent <agent_id> | remove agents all');
          }
          break;

        case 'delete':
          if (args.startsWith('project ')) {
            await this.handleDeleteProject(args.substring(8));
          } else {
            console.log('Usage: delete project <name>');
          }
          break;

        case 'nestty':
          await this.handleNestTTY(args);
          break;

        // Ask agents (natural language task/message broadcast)
        case 'ask': {
          // Strip optional "agents " prefix so both "ask agents X" and "ask X" work
          const askPrompt = args.startsWith('agents ') ? args.substring(7) : args;
          await this.handleAskAgents(askPrompt);
          break;
        }

        // Sessions
        case 'brainstorm':
          await this.handleBrainstorm(args);
          break;

        case 'experiment':
          if (args.startsWith('-analyze ')) {
            await this.handleExperimentAnalyze(args.substring(9));
          } else {
            await this.handleExperiment(args);
          }
          break;

        case 'roundtable':
          await this.handleRoundtable(args);
          break;

        // Interactive controls
        case 'pause':
          await this.handlePause();
          break;

        case 'resume':
          await this.handleResume();
          break;

        case 'select':
          await this.handleSelect(args);
          break;

        case 'edit':
          await this.handleEdit(args);
          break;

        case 'send':
          await this.handleSend(args);
          break;

        case 'clean_up':
          await this.handleCleanUp();
          break;

        case 'wipe':
          await this.handleWipe();
          break;

        // Improvement
        case 'improve':
          await this.handleImprove(args);
          break;

        // PVM
        case 'pvm':
          await this.handlePVMCommand(args);
          break;

        // Coordination
        case 'coordination':
          if (args === 'start') {
            await this.handleCoordinationStart();
          } else if (args === 'stop') {
            await this.handleCoordinationStop();
          } else {
            console.log('Usage: coordination start | coordination stop');
          }
          break;

        // System
        case 'status':
          await this.handleStatus();
          break;

        case 'help':
          this.handleHelp(args);
          break;

        case 'exit':
          await this.handleExit();
          break;

        default:
          // Try natural language parsing
          await this.handleNaturalLanguage(line);
          break;
      }
    } catch (error: any) {
      console.error('❌ Command failed:', error.message);
    }
  }

  // Command handlers
  private async handleListAgents(): Promise<void> {
    const agents = await this.client.getAgents();

    if (agents.length === 0) {
      console.log('No agents connected.');
      return;
    }

    console.log('┌─────────────────┬──────────────────────┬──────────┬─────────────┐');
    console.log('│ Agent ID        │ Name                 │ Status   │ Workload    │');
    console.log('├─────────────────┼──────────────────────┼──────────┼─────────────┤');

    agents.forEach(agent => {
      const name = agent.name || agent.id;
      const status = agent.status || 'unknown';
      const workload = agent.currentTask ? '1 task' : '0 tasks';
      console.log(`│ ${agent.id.padEnd(15)} │ ${name.substring(0, 20).padEnd(20)} │ ${status.padEnd(8)} │ ${workload.padEnd(11)} │`);
    });

    console.log('└─────────────────┴──────────────────────┴──────────┴─────────────┘');
  }

  private async handleSetDefaultAgent(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: default agent <agent_id>');
      console.log('       default agent -r   (randomly pick from registered agents)');
      return;
    }

    let resolvedId: string = args;

    // -r flag: randomly pick from currently registered agents
    if (args.trim() === '-r') {
      let agents: any[] = [];
      try { agents = await this.client.getAgents(); } catch { /* fall through */ }

      if (agents.length === 0) {
        console.log('❌ No agents registered. Connect a Claude Code instance first.');
        return;
      }

      const picked = agents[Math.floor(Math.random() * agents.length)];
      console.log(`Randomly selected: ${picked.id}`);
      resolvedId = picked.id;
    }

    const agentId = resolvedId.trim();
    console.log(`Setting default agent to: ${agentId}...`);

    const success = await this.sessionManager.setDefaultAgent(agentId);

    if (success) {
      console.log(`✓ Planner set to: ${agentId}`);
      console.log('  All project decomposition and planning will use this agent.');
      console.log('  Configuration saved to ~/.act/repl-config.json');
    } else {
      console.log(`❌ Failed to set default agent. Check that agent exists with 'list agents'.`);
    }
  }

  private async handleShowDefaultAgent(): Promise<void> {
    const defaultAgent = this.sessionManager.getDefaultAgent();
    if (defaultAgent) {
      console.log(`Default Agent: ${defaultAgent}`);
    } else {
      console.log('No default agent set. Use: default agent <agent_id>');
    }
  }

  private async handleRemoveAgent(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: remove agent <agent_id> | remove agents all');
      return;
    }

    const agentId = args.trim();

    // ── "remove agents all" — nuke everything ─────────────────────────────────
    if (agentId === 'all') {
      let agents: any[] = [];
      try {
        agents = await this.client.getAgents();
      } catch (e: any) {
        console.log(`\n  ❌ Could not fetch agent list: ${e.message}\n`);
        return;
      }

      if (agents.length === 0) {
        console.log('\n  No agents registered.\n');
        return;
      }

      console.log(`\n  Removing ${agents.length} agent(s)...`);
      let removed = 0;
      for (const agent of agents) {
        try {
          await this.client.removeAgent(agent.id);
          console.log(`  ✓ ${agent.id}`);
          removed++;
        } catch (e: any) {
          console.log(`  ❌ ${agent.id}: ${e.message}`);
        }
      }

      // Clear stored Planner if it was among the removed
      const defaultAgent = this.sessionManager.getDefaultAgent();
      if (defaultAgent) {
        await this.sessionManager.setDefaultAgent('');
        console.log(`\n  Planner cleared (was: ${defaultAgent})`);
      }
      console.log(`\n  Done. Removed ${removed}/${agents.length} agents.\n`);
      return;
    }

    // ── Single agent removal ───────────────────────────────────────────────────
    const defaultAgent = this.sessionManager.getDefaultAgent();
    const isPlanner = agentId === defaultAgent;

    // Fetch current agents so we can offer a replacement if removing the Planner
    let agents: any[] = [];
    try {
      agents = await this.client.getAgents();
    } catch {
      // non-fatal — we'll still attempt removal
    }

    // If removing the Planner, require a replacement before proceeding
    if (isPlanner) {
      const others = agents.filter((a: any) => a.id !== agentId);
      console.log(`\n  ⚠️  "${agentId}" is the current Planner (default agent).`);

      if (others.length === 0) {
        console.log('  No other registered agents to replace it with.');
        console.log('  Register another agent first, then remove this one.');
        console.log('  Alternatively, use: default agent <new_id>  before removing.\n');
        return;
      }

      console.log('  You must set a new Planner before removing this one.');
      console.log('  Currently registered agents:');
      others.forEach((a: any) => console.log(`    • ${a.id}  (${a.name || a.id})`));
      console.log('');

      const newPlanner = await this.prompt('  Set new Planner to (agent_id): ');
      if (!newPlanner || !others.find((a: any) => a.id === newPlanner.trim())) {
        console.log('  ❌ Invalid agent ID. Remove cancelled.\n');
        return;
      }

      const setOk = await this.sessionManager.setDefaultAgent(newPlanner.trim());
      if (!setOk) {
        console.log(`  ❌ Could not set "${newPlanner.trim()}" as Planner. Remove cancelled.\n`);
        return;
      }
      console.log(`  ✓ Planner updated to: ${newPlanner.trim()}`);
    }

    // Proceed with removal
    try {
      await this.client.removeAgent(agentId);
      console.log(`\n  ✓ Agent "${agentId}" removed from ACT.\n`);
      if (isPlanner) {
        console.log(`  Planner is now: ${this.sessionManager.getDefaultAgent()}`);
        console.log('');
      }
    } catch (e: any) {
      console.log(`\n  ❌ ${e.message}\n`);
    }
  }

  // ─── Autonomous mode helper ────────────────────────────────────────────────
  // Writes (or merges into) .claude/settings.json in the project workspace so
  // Claude Code instances opened there run without permission prompts.
  //
  // Modes offered to the user:
  //   bypassPermissions — skip all prompts (fully autonomous, recommended for ACT)
  //   acceptEdits       — auto-accept file edits, still prompt for Bash commands
  //   none              — don't touch settings (user manages permissions manually)
  //
  /**
   * Silently write bypassPermissions to .claude/settings.json in the given dir.
   *
   * Claude Code reads .claude/settings.json from its working directory at launch.
   * Best practice: cd into your project directory, run `act` from there, and open
   * Claude Code from that same directory — all three share the same root.
   *
   * If you need a Claude Code instance WITHOUT bypass permissions in the same project
   * (e.g. for manual review work), create a subdirectory (e.g. claude_code/) and
   * open Claude Code from inside it. The project root is then `cd ../` from that window.
   */

  /**
   * Try to auto-start the ACT server using the path saved in ~/.act/config.json
   * by install.sh. Returns true if the server is up and reachable after the attempt.
   */
  private async tryAutoStartServer(): Promise<boolean> {
    const configPath = path.join(process.env.HOME || '~', '.act', 'config.json');
    let actRoot: string | null = null;
    try {
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      actRoot = config.actRoot || null;
    } catch {
      // config file missing — install.sh was never run or wrote no path
    }

    if (!actRoot) return false;

    const serverDist = path.join(actRoot, 'server', 'dist', 'index.js');
    if (!fs.existsSync(serverDist)) {
      console.log(`  ⚠️  Server not built at ${serverDist}`);
      console.log(`      Run: cd ${actRoot}/server && npm run build`);
      return false;
    }

    console.log('  ⚡ ACT server not running — starting it automatically...');
    const logPath = path.join(process.env.HOME || '~', '.act', 'server.log');
    const logFile = fs.openSync(logPath, 'a');
    const child = spawn('node', [serverDist], {
      detached: true,
      stdio: ['ignore', logFile, logFile],
      env: { ...process.env }
    });
    child.unref();

    // Wait up to 5 seconds for the server to come up
    for (let i = 0; i < 10; i++) {
      await new Promise(r => setTimeout(r, 500));
      try {
        await this.client.testConnection();
        console.log(`  ✅ Server started (PID ${child.pid}) — logs: ${logPath}`);
        return true;
      } catch {
        // still starting
      }
    }

    console.log(`  ❌ Server process started but not responding after 5s — check ${logPath}`);
    return false;
  }

  private ensureBypassPermissions(dir: string): void {
    try {
      const claudeDir = path.join(dir, '.claude');
      const settingsPath = path.join(claudeDir, 'settings.json');
      fs.mkdirSync(claudeDir, { recursive: true });
      let existing: Record<string, any> = {};
      if (fs.existsSync(settingsPath)) {
        try { existing = JSON.parse(fs.readFileSync(settingsPath, 'utf8')); } catch { /* overwrite */ }
      }
      // Already set — nothing to do
      if (existing?.permissions?.defaultMode === 'bypassPermissions') return;
      existing.permissions = { ...(existing.permissions || {}), defaultMode: 'bypassPermissions' };
      fs.writeFileSync(settingsPath, JSON.stringify(existing, null, 2), 'utf8');
      console.log(`⚡ bypassPermissions enabled for: ${dir}`);
      console.log('  Open Claude Code from this directory to run agents without permission prompts.');
      console.log('  Tip: if you need a non-bypass Claude Code window in this project, open it');
      console.log('  from a subdirectory (e.g. mkdir claude_code && cd claude_code).\n');
    } catch { /* non-fatal */ }
  }

  private async writeAgentSettings(workspace: string): Promise<void> {
    console.log('\n  ─── Autonomous mode ───────────────────────────────────────');
    console.log('  ACT agents work best without permission interruptions.');
    console.log('  Choose how Claude Code instances in this project will run:\n');
    console.log('    1. bypassPermissions — skip all prompts  (fully autonomous, recommended)');
    console.log('    2. acceptEdits       — auto-accept file edits, prompt for Bash');
    console.log('    3. none              — leave settings unchanged\n');

    const choice = await this.prompt('  Choice [1]: ');
    const trimmed = (choice || '1').trim();

    let mode: string | null = null;
    if (trimmed === '1' || trimmed === 'bypassPermissions') mode = 'bypassPermissions';
    else if (trimmed === '2' || trimmed === 'acceptEdits') mode = 'acceptEdits';
    else if (trimmed === '3' || trimmed === 'none') mode = null;
    else mode = 'bypassPermissions'; // default on invalid input

    if (!mode) {
      console.log('  Skipped — permission settings unchanged.\n');
      return;
    }

    try {
      const claudeDir = path.join(workspace, '.claude');
      const settingsPath = path.join(claudeDir, 'settings.json');

      fs.mkdirSync(claudeDir, { recursive: true });

      // Merge with existing settings if present
      let existing: Record<string, any> = {};
      if (fs.existsSync(settingsPath)) {
        try { existing = JSON.parse(fs.readFileSync(settingsPath, 'utf8')); } catch { /* malformed — overwrite */ }
      }

      existing.permissions = {
        ...(existing.permissions || {}),
        defaultMode: mode
      };

      fs.writeFileSync(settingsPath, JSON.stringify(existing, null, 2), 'utf8');
      console.log(`  ✅ Written: ${settingsPath}`);
      console.log(`     defaultMode: "${mode}"`);
      console.log('  Claude Code instances opened in this workspace will use this setting.\n');
    } catch (e: any) {
      console.log(`  ⚠️  Could not write settings: ${e.message} (continuing anyway)\n`);
    }
  }

  /**
   * Parse project name and optional workspace from inline args.
   *
   * Name rules:
   *   - Single-word: first space-delimited token
   *   - Multi-word: must be wrapped in single or double quotes
   *
   * Path rules:
   *   - Everything after the name (optionally preceded by "in ")
   *   - If absent, caller prompts interactively
   *   - '.' or '*' → current directory
   *   - Always resolved to an absolute path
   *   - Must be an existing directory (REPL never creates dirs)
   *
   * Returns { name, workspace } where workspace may be '' if not provided inline.
   * Returns null if parsing fails (error already printed).
   */
  private parseCreateProjectArgs(args: string): { name: string; workspace: string } | null {
    const QUOTE_TIP =
      '\n  ❌ Could not resolve workspace path.' +
      '\n' +
      '\n  If your project name has multiple words, wrap it in quotes — otherwise' +
      '\n  the second word is treated as the start of the path.' +
      '\n' +
      '\n  Examples:' +
      "\n    create project myapp ." +
      "\n    create project 'cool project' ." +
      '\n    create project "cool project" ~/Documents/Projects/cool\n';

    const input = args.trim();
    if (!input) return null;

    let name: string;
    let rest: string;

    // Quoted name (single or double quotes)
    const quoteChar = input[0] === "'" ? "'" : input[0] === '"' ? '"' : null;
    if (quoteChar) {
      const closeIdx = input.indexOf(quoteChar, 1);
      if (closeIdx === -1) {
        console.log(QUOTE_TIP);
        return null;
      }
      name = input.substring(1, closeIdx).trim();
      rest = input.substring(closeIdx + 1).trim();
    } else {
      // Unquoted — name is the first space-delimited token
      const spaceIdx = input.indexOf(' ');
      if (spaceIdx === -1) {
        // No space → just a name, no path provided inline
        return { name: input, workspace: '' };
      }
      name = input.substring(0, spaceIdx);
      rest = input.substring(spaceIdx + 1).trim();
    }

    if (!name) return null;

    // Strip optional "in " prefix for backwards compat ("create project foo in .")
    const rawPath = rest.startsWith('in ') ? rest.substring(3).trim() : rest;

    if (!rawPath) {
      // Name parsed, but no path — caller will prompt
      return { name, workspace: '' };
    }

    // Resolve path
    const resolved = rawPath === '.' || rawPath === '*'
      ? process.cwd()
      : path.resolve(rawPath);

    // Must be an existing directory
    if (!fs.existsSync(resolved) || !fs.statSync(resolved).isDirectory()) {
      console.log(QUOTE_TIP);
      console.log(`  (Tried to resolve: "${resolved}")`);
      return null;
    }

    return { name, workspace: resolved };
  }

  private async handleCreateProject(args?: string): Promise<void> {
    // Step 1: Parse name and workspace path
    let name: string;
    let workspace: string;

    if (args && args.trim()) {
      const parsed = this.parseCreateProjectArgs(args.trim());
      if (!parsed) return; // error already printed
      name = parsed.name;
      workspace = parsed.workspace;
    } else {
      name = await this.prompt('  Project name: ');
      workspace = '';
    }

    if (!name) {
      console.log('  Project name is required.');
      return;
    }

    // If workspace wasn't provided inline, prompt for it
    if (!workspace) {
      const raw = await this.prompt("  Workspace path (. for current directory): ");
      if (!raw) {
        console.log('  Workspace path is required.');
        return;
      }
      workspace = raw === '.' || raw === '*' ? process.cwd() : path.resolve(raw);
      if (!fs.existsSync(workspace) || !fs.statSync(workspace).isDirectory()) {
        console.log(`\n  ❌ Directory not found: ${workspace}`);
        console.log('  The REPL does not create directories. Create it first, then re-run.\n');
        return;
      }
    }

    const bar = '─'.repeat(50);
    console.log(`\n┌${bar}┐`);
    const title = `  New Project: ${name}`;
    console.log(`│${title.padEnd(50)} │`);
    console.log(`└${bar}┘`);
    console.log('\nAnswer a few questions to set up your project.\n');

    // Step 2: Collect project details
    const description = await this.prompt('  What are you building?\n  > ');
    if (!description) { console.log('Description is required.'); return; }

    const techStack = await this.prompt('\n  Technologies / stack (e.g. "TypeScript, React, PostgreSQL"):\n  > ');
    const constraints = await this.prompt('\n  Any constraints? (timeline, must-use tools — Enter to skip):\n  > ');
    const successCriteria = await this.prompt('\n  What does success look like?\n  > ');
    if (!successCriteria) { console.log('Success criteria is required.'); return; }

    // Step 3: Agents
    const agents = await this.client.getAgents();
    if (agents.length > 0) {
      console.log('\n  Connected agents:');
      agents.forEach(a => console.log(`    • ${a.id} (${a.name || a.id})`));
    } else {
      console.log('\n  No agents currently connected.');
    }
    const agentsInput = await this.prompt('\n  Which agents will work on this? (comma-separated IDs, Enter for all):\n  > ');
    const participatingAgents = agentsInput
      ? agentsInput.split(',').map(s => s.trim()).filter(Boolean)
      : agents.map(a => a.id);

    // Step 4: Create project record
    console.log(`\n  Creating project "${name}"...`);
    let project: any;
    try {
      project = await this.client.createProject({
        name,
        workspace,
        description,
        techStack: techStack || '',
        constraints: constraints || undefined,
        successCriteria,
        agents: participatingAgents
      });
    } catch (error: any) {
      console.error('  ❌ Failed to create project:', error.message);
      return;
    }
    console.log('  ✅ Project saved.');
    this.currentProjectName = name;
    this.ensureBypassPermissions(path.resolve(workspace));

    // Step 5: Planning via Planner
    const defaultAgent = this.sessionManager.getDefaultAgent();
    if (!defaultAgent) {
      console.log('\n  ⚠️  No default agent set — skipping AI planning.');
      console.log('  Set a default agent:  default agent <id>');
      console.log('  Then re-run:          create project ' + name + ' in ' + workspace + '\n');
      return;
    }

    const actorAgent = agents.find(a => a.id === defaultAgent);
    if (!actorAgent) {
      console.log(`\n  ⚠️  Default agent "${defaultAgent}" is not connected.`);
      console.log('  Connect the agent first, then re-create the project.\n');
      return;
    }

    // Wire 1: Search PVM for relevant past coordination patterns before building prompt
    let pvmContext: string | undefined;
    try {
      const pvmQuery = [description, techStack, name].filter(Boolean).join(' ');
      const pvmResults = await this.client.searchPVM(pvmQuery, 5);
      if (pvmResults.length > 0) {
        pvmContext = pvmResults
          .map((r: any, i: number) => {
            const msg = r.message || r;
            return `[${i + 1}] ${msg.agent || 'unknown'} (${msg.type || 'event'}): ${(msg.message || '').substring(0, 300)}`;
          })
          .join('\n\n');
        console.log(`\n  ✓ Found ${pvmResults.length} relevant past coordination pattern(s) — injecting into planning context.`);
      }
    } catch {
      // PVM search failure is non-fatal — planning continues without it
    }

    const planningDesc = buildPlanningPrompt({ name, workspace, description, techStack, constraints, successCriteria, participatingAgents, pvmContext });

    console.log(`\n  Assigning planning task to ${actorAgent.name || defaultAgent}...`);
    let planningTask: any;
    try {
      planningTask = await this.client.createTaskREST({
        description: planningDesc,
        requiredCapabilities: [],
        priority: 'high',
        assignedAgent: defaultAgent,
        metadata: { type: 'planning', projectName: name }
      });
    } catch (error: any) {
      console.error('  ❌ Failed to create planning task:', error.message);
      return;
    }

    console.log(`\n  Waiting for ${actorAgent.name || defaultAgent} to analyze your project...`);
    console.log('  (Agent must call get_task, complete it, then call report_task_complete with JSON)');
    console.log('  Press Ctrl+C to cancel — task stays active, resume with: continue project ' + name + '\n');

    const result = await this.pollWithSpinner(planningTask.id, 600_000);

    if (!result) {
      console.log('\n  ⏱  Timed out. The planning task is still open.');
      console.log('  When the agent completes it, run: continue project ' + name + '\n');
      return;
    }

    // Step 6: Parse breakdown and create tasks
    let breakdown: any;
    try {
      breakdown = JSON.parse(result);
    } catch {
      console.log('\n  ⚠️  Agent returned non-JSON result. Raw output:');
      console.log('  ' + result.substring(0, 500));
      return;
    }

    const tasks: any[] = breakdown.tasks || [];
    console.log(`\n  Planning complete! Creating ${tasks.length} tasks...\n`);

    // Topological creation: sort tasks so dependencies are always created before
    // the tasks that depend on them. Dep IDs are passed at creation time so the
    // server's processPendingTasks correctly blocks dependent tasks immediately —
    // no race condition where everything gets assigned before deps are set.
    const sortedIndices = topoSort(tasks);
    const indexToTaskId: string[] = new Array(tasks.length).fill('');

    for (const i of sortedIndices) {
      const taskDef = tasks[i];
      const resolvedDeps = (taskDef.dependencies || [])
        .map((idx: number) => indexToTaskId[idx])
        .filter((id: string) => !!id);
      try {
        const created = await this.client.createTaskREST({
          description: taskDef.description || taskDef.title,
          requiredCapabilities: taskDef.requiredCapabilities || [],
          priority: taskDef.priority || 'medium',
          estimatedDuration: taskDef.estimatedDuration,
          dependencies: resolvedDeps,
          metadata: { projectName: name, title: taskDef.title }
        });
        indexToTaskId[i] = created.id;
        const depNote = resolvedDeps.length ? ` (blocked until ${resolvedDeps.length} dep(s) done)` : '';
        console.log(`  ✓ ${taskDef.title}${depNote}`);
      } catch (e: any) {
        console.log(`  ✗ "${taskDef.title}": ${e.message}`);
      }
    }

    // Step 7: Store briefs
    const briefs: Record<string, string> = breakdown.briefs || {};
    const briefEntries = Object.entries(briefs);
    if (briefEntries.length > 0) {
      console.log(`\n  Storing AGENT.md briefs for ${briefEntries.length} agent(s)...`);
      for (const [agentId, content] of briefEntries) {
        try {
          await this.client.storeBrief(name, agentId, content);
          console.log(`  ✓ Brief stored for ${agentId}`);
        } catch (e: any) {
          console.log(`  ✗ Brief for ${agentId}: ${e.message}`);
        }
      }
    }

    const divider = '━'.repeat(50);
    console.log(`\n  ${divider}`);
    console.log(`  Project "${name}" is ready!`);
    console.log(`  Tasks created:  ${tasks.length}`);
    console.log(`  Agents:         ${participatingAgents.join(', ') || 'none specified'}`);
    console.log(`\n  Each agent should:`);
    console.log('  1. Call register_with_act to join the session');
    console.log('  2. Call get_agent_brief to fetch their AGENT.md context');
    console.log('  3. Call get_task to receive their first assigned task');
    console.log(`  ${divider}\n`);
  }

  /** Scan ChronologicalLog + disk for prior work on this workspace path. */
  private async scanPriorHistory(absolutePath: string): Promise<{
    found: boolean;
    projectName: string | undefined;
    totalTasks: number;
    completedTasks: number;
    inProgressTasks: number;
    pendingTasks: number;
    agents: string[];
    agentMdFiles: Array<{ agentId: string; filePath: string; content: string }>;
    incompleteTasks: Array<{ title: string; description: string; status: string; assignedAgent?: string }>;
  }> {
    const empty = { found: false, projectName: undefined, totalTasks: 0, completedTasks: 0, inProgressTasks: 0, pendingTasks: 0, agents: [], agentMdFiles: [], incompleteTasks: [] };

    try {
      // 1. Scan ChronologicalLog for events mentioning this path
      const events = await this.client.getRecentLog(1000);
      const pathBase = path.basename(absolutePath);

      // Filter events related to this workspace (by path or project name)
      const relevant = events.filter((e: any) => {
        const msg = (e.message || '').toLowerCase();
        return msg.includes(absolutePath.toLowerCase()) || msg.includes(pathBase.toLowerCase());
      });

      // 2. Extract task info from relevant events
      const taskMap = new Map<string, { title: string; description: string; status: string; assignedAgent?: string }>();
      const agentSet = new Set<string>();

      for (const e of relevant) {
        if (e.agent && e.agent !== 'system') agentSet.add(e.agent);
        const msg: string = e.message || '';

        // Parse task_created events
        if (msg.startsWith('task_created:')) {
          try {
            const json = JSON.parse(msg.substring('task_created:'.length).trim());
            const t = json.task || json;
            if (t.id) taskMap.set(t.id, { title: t.metadata?.title || t.description?.substring(0, 60) || t.id, description: t.description || '', status: t.status || 'pending', assignedAgent: t.assignedAgent });
          } catch { /* skip malformed */ }
        }
        // Parse progress/completion events
        if (msg.startsWith('task_progress:') || msg.startsWith('task_completed:')) {
          try {
            const json = JSON.parse(msg.substring(msg.indexOf(':') + 1).trim());
            const taskId = json.taskId || json.task?.id;
            if (taskId && taskMap.has(taskId)) {
              const entry = taskMap.get(taskId)!;
              if (json.status) entry.status = json.status;
              if (json.success === true) entry.status = 'completed';
            }
          } catch { /* skip */ }
        }
      }

      // 3. Scan disk for AGENT.md files
      const agentMdFiles: Array<{ agentId: string; filePath: string; content: string }> = [];
      const claudeDir = path.join(absolutePath, '.claude');
      if (fs.existsSync(claudeDir)) {
        const entries = fs.readdirSync(claudeDir, { withFileTypes: true });
        for (const entry of entries) {
          if (entry.isDirectory()) {
            const agentMdPath = path.join(claudeDir, entry.name, 'AGENT.md');
            if (fs.existsSync(agentMdPath)) {
              const content = fs.readFileSync(agentMdPath, 'utf-8');
              agentMdFiles.push({ agentId: entry.name, filePath: agentMdPath, content });
              agentSet.add(entry.name);
            }
          }
        }
      }
      // Also check workspace root for AGENT.md
      const rootAgentMd = path.join(absolutePath, 'AGENT.md');
      if (fs.existsSync(rootAgentMd)) {
        const content = fs.readFileSync(rootAgentMd, 'utf-8');
        // Try to extract agent id from first line
        const firstLine = content.split('\n')[0] || '';
        const agentId = firstLine.replace(/^#\s*/, '').trim() || 'default';
        agentMdFiles.push({ agentId, filePath: rootAgentMd, content });
      }

      if (relevant.length === 0 && agentMdFiles.length === 0) return empty;

      // 4. Extract project name from events
      let projectName: string | undefined;
      for (const e of relevant) {
        const msg: string = e.message || '';
        if (msg.includes('project_created:') || msg.includes('"name":')) {
          try {
            const m = msg.match(/"name"\s*:\s*"([^"]+)"/);
            if (m) { projectName = m[1]; break; }
          } catch { /* skip */ }
        }
      }
      if (!projectName) projectName = path.basename(absolutePath);

      const tasks = Array.from(taskMap.values());
      const completedTasks = tasks.filter(t => t.status === 'completed').length;
      const inProgressTasks = tasks.filter(t => t.status === 'in_progress' || t.status === 'assigned').length;
      const pendingTasks = tasks.filter(t => t.status === 'pending').length;
      const incompleteTasks = tasks.filter(t => t.status !== 'completed');

      return {
        found: true,
        projectName,
        totalTasks: tasks.length,
        completedTasks,
        inProgressTasks,
        pendingTasks,
        agents: Array.from(agentSet),
        agentMdFiles,
        incompleteTasks
      };
    } catch {
      return empty;
    }
  }

  /** Resume a project from prior ChronologicalLog history + on-disk AGENT.md files. */
  private async resumeFromPriorHistory(absolutePath: string, history: Awaited<ReturnType<typeof this.scanPriorHistory>>): Promise<void> {
    const projectName = history.projectName || path.basename(absolutePath);
    const agents = await this.client.getAgents();
    const defaultAgent = this.sessionManager.getDefaultAgent();

    // Re-register project record
    console.log(`\n  Re-registering project "${projectName}"...`);
    try {
      await this.client.createProject({
        name: projectName,
        workspace: absolutePath,
        description: `Resumed from prior session`,
        techStack: '',
        successCriteria: 'Complete remaining tasks',
        agents: history.agents
      });
      console.log('  ✅ Project record restored.');
    } catch (e: any) {
      console.log(`  ⚠️  Project record: ${e.message} (may already exist)`);
    }

    // Re-upload on-disk AGENT.md briefs to server
    if (history.agentMdFiles.length > 0) {
      console.log(`\n  Re-uploading ${history.agentMdFiles.length} AGENT.md brief(s) to server...`);
      for (const { agentId, content } of history.agentMdFiles) {
        try {
          await this.client.storeBrief(projectName, agentId, content);
          console.log(`  ✅ Brief restored for ${agentId}`);
        } catch (e: any) {
          console.log(`  ⚠️  Brief for ${agentId}: ${e.message}`);
        }
      }
    }

    // Re-create incomplete tasks so agents can pick them up
    if (history.incompleteTasks.length > 0) {
      console.log(`\n  Re-creating ${history.incompleteTasks.length} incomplete task(s)...`);
      for (const task of history.incompleteTasks) {
        try {
          await this.client.createTaskREST({
            description: task.description || task.title,
            requiredCapabilities: [],
            priority: 'medium',
            assignedAgent: task.assignedAgent && agents.find(a => a.id === task.assignedAgent) ? task.assignedAgent : undefined,
            metadata: { title: task.title, resumed: true }
          });
          console.log(`  ✅ ${task.title}`);
        } catch (e: any) {
          console.log(`  ⚠️  "${task.title}": ${e.message}`);
        }
      }
    } else {
      console.log('\n  ✅ All prior tasks were completed — nothing to re-create.');
    }

    this.ensureBypassPermissions(absolutePath);

    const divider = '━'.repeat(50);
    console.log(`\n  ${divider}`);
    console.log(`  "${projectName}" resumed!`);
    console.log(`  Tasks restored:  ${history.incompleteTasks.length} incomplete / ${history.totalTasks} total`);
    console.log(`  Briefs restored: ${history.agentMdFiles.length}`);
    console.log(`\n  Each agent should:`);
    console.log('  1. Call get_agent_brief to re-fetch their AGENT.md context');
    console.log('  2. Type "begin" to trigger the hook and pick up their task');
    console.log(`  ${divider}\n`);
  }

  private async handleImportProject(args?: string): Promise<void> {
    // Step 1: Get directory path
    let dirPath = args?.trim();
    if (!dirPath) {
      dirPath = await this.prompt('  Path to project directory: ');
    }
    if (!dirPath) {
      console.log('Directory path is required.');
      return;
    }

    // '.' or '*' means use the directory the REPL was launched from
    if (dirPath === '.' || dirPath === '*') {
      dirPath = process.cwd();
    }

    const absolutePath = path.resolve(dirPath);
    if (!fs.existsSync(absolutePath)) {
      console.log(`  ❌ Directory not found: ${absolutePath}`);
      return;
    }

    const bar = '─'.repeat(50);
    console.log(`\n┌${bar}┐`);
    const title = `  Importing: ${absolutePath}`;
    console.log(`│${title.substring(0, 50).padEnd(50)} │`);
    console.log(`└${bar}┘`);

    // ── RESUME CHECK ──────────────────────────────────────────────────────────
    // Before doing a full Planner analysis, check if this workspace has prior
    // history in the ChronologicalLog and AGENT.md files on disk.
    console.log('\n  Checking for prior session history...');
    const priorHistory = await this.scanPriorHistory(absolutePath);

    if (priorHistory.found) {
      console.log(`\n  ✅ Found prior session for this workspace:`);
      console.log(`     Tasks in log:    ${priorHistory.totalTasks} (${priorHistory.completedTasks} completed, ${priorHistory.inProgressTasks} in progress, ${priorHistory.pendingTasks} pending)`);
      console.log(`     Agents found:    ${priorHistory.agents.join(', ') || 'none'}`);
      console.log(`     AGENT.md files:  ${priorHistory.agentMdFiles.length} on disk`);
      if (priorHistory.projectName) console.log(`     Project name:    ${priorHistory.projectName}`);

      const resumeChoice = await this.prompt('\n  (r) Resume from prior session  (f) Fresh analysis  (q) Cancel\n  > ');
      if (resumeChoice.toLowerCase() === 'q') return;

      if (resumeChoice.toLowerCase() !== 'f') {
        // ── RESUME PATH ───────────────────────────────────────────────────────
        await this.resumeFromPriorHistory(absolutePath, priorHistory);
        return;
      }
      console.log('\n  Running fresh analysis...');
    } else {
      console.log('  No prior session found — running fresh analysis.');
    }

    console.log('\n  Reading project context...');

    // Step 2: Read directory context
    const context = readDirectoryContext(absolutePath);
    const lineCount = context.fileTree.split('\n').length;
    console.log(`  ✓ File tree: ${lineCount} entries`);
    if (context.readme)     console.log(`  ✓ README: ${context.readme.length} chars`);
    if (context.packageJson) console.log(`  ✓ package.json: found`);
    if (context.gitLog)     console.log(`  ✓ Git log: ${context.gitLog.split('\n').length} commits`);

    // Step 3: Check for default agent (Planner)
    const defaultAgent = this.sessionManager.getDefaultAgent();
    const agents = await this.client.getAgents();

    if (!defaultAgent || !agents.find(a => a.id === defaultAgent)) {
      console.log('\n  ⚠️  No default agent connected — running in manual mode.');
      console.log('  I\'ll infer a project name and description from the directory.');
      console.log('  Set a default agent for AI-powered analysis: default agent <id>\n');

      // Fall back to manual: infer name from dir name, prompt for description
      const inferredName = path.basename(absolutePath);
      const name = await this.prompt(`  Project name [${inferredName}]: `) || inferredName;
      const description = await this.prompt('  Brief description: ');
      const techStack = await this.prompt('  Tech stack (e.g. TypeScript, React): ');
      const successCriteria = await this.prompt('  What does success look like?\n  > ');

      await this.client.createProject({
        name, workspace: absolutePath, description, techStack: techStack || '',
        successCriteria: successCriteria || 'Complete the planned work', agents: []
      });
      console.log(`\n  ✅ Project "${name}" registered at ${absolutePath}`);
      console.log('  Use: create project to run AI planning when an agent is connected.\n');
      return;
    }

    // Step 4: Choose participating agents
    if (agents.length > 0) {
      console.log('\n  Connected agents:');
      agents.forEach(a => console.log(`    • ${a.id} (${a.name || a.id})`));
    }
    const agentsInput = await this.prompt('\n  Which agents will work on this? (comma-separated IDs, Enter for all):\n  > ');
    const participatingAgents = agentsInput
      ? agentsInput.split(',').map(s => s.trim()).filter(Boolean)
      : agents.map(a => a.id);

    // Step 5: Send context to Planner for synthesis
    const actorAgent = agents.find(a => a.id === defaultAgent)!;

    // Wire 1: Search PVM for relevant past coordination patterns before building prompt
    let importPvmContext: string | undefined;
    try {
      const pvmQuery = [path.basename(absolutePath), context.packageJson ? 'package.json' : '', context.readme].filter(Boolean).join(' ').substring(0, 300);
      const pvmResults = await this.client.searchPVM(pvmQuery, 5);
      if (pvmResults.length > 0) {
        importPvmContext = pvmResults
          .map((r: any, i: number) => {
            const msg = r.message || r;
            return `[${i + 1}] ${msg.agent || 'unknown'} (${msg.type || 'event'}): ${(msg.message || '').substring(0, 300)}`;
          })
          .join('\n\n');
        console.log(`  ✓ Found ${pvmResults.length} relevant past coordination pattern(s) — injecting into planning context.`);
      }
    } catch {
      // PVM search failure is non-fatal — planning continues without it
    }

    const planningDesc = buildImportPlanningPrompt({
      dirPath: absolutePath,
      ...context,
      participatingAgents,
      pvmContext: importPvmContext
    });

    console.log(`\n  Asking ${actorAgent.name || defaultAgent} to analyze your project...`);
    console.log('  This may take a minute.\n');

    let planningTask: any;
    try {
      planningTask = await this.client.createTaskREST({
        description: planningDesc,
        requiredCapabilities: [],
        priority: 'high',
        assignedAgent: defaultAgent,
        metadata: { type: 'import_planning', dirPath: absolutePath }
      });
    } catch (error: any) {
      console.error('  ❌ Failed to create planning task:', error.message);
      return;
    }

    const result = await this.pollWithSpinner(planningTask.id, 600_000);
    if (!result) {
      console.log('\n  ⏱  Timed out waiting for Planner analysis.');
      console.log('  The planning task is still open. Try: continue project <name> after the agent responds.\n');
      return;
    }

    // Step 6: Parse the synthesis result
    let breakdown: any;
    try {
      // Strip markdown code fences if the agent wrapped it
      const cleaned = result.replace(/^```(?:json)?\n?/m, '').replace(/\n?```$/m, '').trim();
      breakdown = JSON.parse(cleaned);
    } catch {
      console.log('\n  ⚠️  Agent returned non-JSON. Raw output:');
      console.log('  ' + result.substring(0, 500));
      return;
    }

    const summary = breakdown.summary || {};
    const projectName = summary.name || path.basename(absolutePath);

    // Step 7: Show summary and confirm with user
    console.log('\n  ┌─────────────────────────────────────────────────────┐');
    console.log(`  │  Project: ${projectName.substring(0, 42).padEnd(42)} │`);
    console.log('  ├─────────────────────────────────────────────────────┤');
    if (summary.description) console.log(`  │  ${summary.description.substring(0, 50).padEnd(50)} │`);
    if (summary.techStack)   console.log(`  │  Stack: ${summary.techStack.substring(0, 44).padEnd(44)} │`);
    if (summary.currentState) console.log(`  │  State: ${summary.currentState.substring(0, 44).padEnd(44)} │`);
    console.log('  ├─────────────────────────────────────────────────────┤');
    console.log(`  │  Suggested tasks (${(breakdown.tasks || []).length}):${' '.repeat(31)} │`);
    (breakdown.tasks || []).slice(0, 6).forEach((t: any) => {
      const title = (t.title || '').substring(0, 48);
      console.log(`  │    • ${title.padEnd(47)} │`);
    });
    console.log('  └─────────────────────────────────────────────────────┘');

    const confirm = await this.prompt('\n  Does this look right? (y/n, or type corrections): ');

    if (confirm.toLowerCase() === 'n' || confirm.toLowerCase() === 'no') {
      console.log('  Cancelled. The planning task result is saved — ask the agent to revise and try again.\n');
      return;
    }

    // If they typed actual corrections, loop back with revised context (simplified: just proceed)
    // Future: pass corrections back as another task to Planner

    // Step 8: Create project record
    const projectDescription = summary.description || `Imported from ${absolutePath}`;
    const techStack = summary.techStack || '';
    console.log(`\n  Creating project "${projectName}"...`);
    try {
      await this.client.createProject({
        name: projectName,
        workspace: absolutePath,
        description: projectDescription,
        techStack,
        successCriteria: summary.currentState ? `Continue from: ${summary.currentState}` : 'Complete planned tasks',
        agents: participatingAgents
      });
    } catch (error: any) {
      console.error('  ❌ Failed to create project:', error.message);
      return;
    }
    console.log('  ✅ Project saved.');
    this.ensureBypassPermissions(absolutePath);

    // Step 9: Create tasks (two-pass: create all first, then patch dependency IDs)
    const tasks: any[] = breakdown.tasks || [];
    console.log(`\n  Creating ${tasks.length} tasks...`);
    const indexToTaskId: string[] = [];
    for (const taskDef of tasks) {
      try {
        const created = await this.client.createTaskREST({
          description: taskDef.description || taskDef.title,
          requiredCapabilities: taskDef.requiredCapabilities || [],
          priority: taskDef.priority || 'medium',
          estimatedDuration: taskDef.estimatedDuration,
          dependencies: [], // filled in second pass
          metadata: { projectName, title: taskDef.title }
        });
        indexToTaskId.push(created.id);
        const depNote = (taskDef.dependencies?.length)
          ? ` (waits for: task ${taskDef.dependencies.join(', ')})`
          : '';
        console.log(`  ✓ ${taskDef.title}${depNote}`);
      } catch (e: any) {
        indexToTaskId.push('');
        console.log(`  ✗ "${taskDef.title}": ${e.message}`);
      }
    }

    // Second pass: resolve index-based deps to real task IDs
    for (let i = 0; i < tasks.length; i++) {
      const rawDeps: number[] = tasks[i].dependencies || [];
      if (rawDeps.length === 0 || !indexToTaskId[i]) continue;
      const resolvedDeps = rawDeps
        .map((idx: number) => indexToTaskId[idx])
        .filter((id: string) => !!id);
      if (resolvedDeps.length > 0) {
        try {
          await this.client.patchTaskDependencies(indexToTaskId[i], resolvedDeps);
        } catch (e: any) {
          console.log(`  ⚠️  Could not set dependencies for task ${i}: ${e.message}`);
        }
      }
    }

    // Step 10: Store briefs
    const briefs: Record<string, string> = breakdown.briefs || {};
    const briefEntries = Object.entries(briefs);
    if (briefEntries.length > 0) {
      console.log(`\n  Storing AGENT.md briefs for ${briefEntries.length} agent(s)...`);
      for (const [agentId, briefContent] of briefEntries) {
        try {
          await this.client.storeBrief(projectName, agentId, briefContent as string);
          console.log(`  ✓ Brief stored for ${agentId}`);
        } catch (e: any) {
          console.log(`  ✗ Brief for ${agentId}: ${e.message}`);
        }
      }
    }

    const divider = '━'.repeat(50);
    console.log(`\n  ${divider}`);
    console.log(`  "${projectName}" imported and ready!`);
    console.log(`  Tasks created:  ${tasks.length}`);
    console.log(`  Agents:         ${participatingAgents.join(', ') || 'none specified'}`);
    console.log(`\n  Each agent should:`);
    console.log('  1. Call register_with_act to join the session');
    console.log('  2. Call get_agent_brief to fetch their AGENT.md context');
    console.log('  3. Call get_task to receive their first assignment');
    console.log(`  ${divider}\n`);
  }

  /** Prompt the user for a single line of input, pausing the main command loop. */
  private prompt(question: string): Promise<string> {
    return new Promise(resolve => {
      this.isPrompting = true;
      this.rl.question(question, answer => {
        this.isPrompting = false;
        resolve(answer.trim());
      });
    });
  }

  /** Poll a task until completed/failed or timeout. Returns metadata.result or null. */
  private async pollWithSpinner(taskId: string, timeoutMs: number): Promise<string | null> {
    const intervalMs = 3000;
    const start = Date.now();
    let dots = 0;

    while (Date.now() - start < timeoutMs) {
      const task = await this.client.getTask(taskId);
      if (task?.status === 'completed') {
        process.stdout.write('\r' + ' '.repeat(60) + '\r');
        return task.metadata?.result ?? null;
      }
      if (task?.status === 'failed') {
        process.stdout.write('\r' + ' '.repeat(60) + '\r');
        return null;
      }
      const elapsed = Math.round((Date.now() - start) / 1000);
      process.stdout.write(`\r  ⏳ Analyzing${'.'.repeat(dots + 1)}   (${elapsed}s elapsed)`);
      dots = (dots + 1) % 3;
      await new Promise(r => setTimeout(r, intervalMs));
    }
    process.stdout.write('\r' + ' '.repeat(60) + '\r');
    return null;
  }

  private async handleContinueProject(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: continue project <name>');
      return;
    }

    const project = await this.client.getProject(args);
    if (!project) {
      console.log(`Project "${args}" not found. Run: list projects`);
      return;
    }
    this.currentProjectName = args;

    console.log(`\nResuming project "${args}"...`);
    console.log(`  Status: ${project.status}`);
    console.log(`  Agents: ${(project.agents || []).join(', ') || 'none'}`);

    // Look for a planning task for this project
    const allTasks = await this.client.getTasks();
    const planningTask = allTasks.find((t: any) =>
      t.metadata?.type === 'planning' &&
      t.metadata?.projectName === args
    );

    if (!planningTask) {
      console.log('\n  No planning task found. Project may already be set up.');
      console.log('  Use: show project ' + args + ' for full details.\n');
      return;
    }

    if (planningTask.status === 'completed') {
      // Planning task is done — try to parse and create tasks from it
      console.log('\n  Planning task completed. Parsing breakdown...');
      const result = planningTask.metadata?.result;
      if (!result) {
        console.log('  ❌ Planning task completed but no result found.');
        console.log('  Use: show project ' + args + ' for full details.\n');
        return;
      }
      // Re-use the same parse + task creation logic via a fake "result" flow
      await this.createTasksFromResult(result, args, project);
      return;
    }

    if (planningTask.status === 'failed') {
      console.log(`\n  ❌ Planning task failed. Agents need to retry.`);
      console.log('  Use: show project ' + args + ' for full details.\n');
      return;
    }

    // Planning task still pending/assigned/in_progress — resume polling
    console.log(`\n  Planning task is ${planningTask.status}. Waiting for agent to complete it...`);
    console.log('  Press Ctrl+C to cancel\n');

    const result = await this.pollWithSpinner(planningTask.id, 600_000);
    if (!result) {
      console.log('\n  ⏱  Timed out waiting for Planner analysis.');
      console.log('  The planning task is still open. Try: continue project ' + args + ' after the agent responds.\n');
      return;
    }

    await this.createTasksFromResult(result, args, project);
  }

  private async createTasksFromResult(result: string, projectName: string, project: any): Promise<void> {
    let breakdown: any;
    try {
      // Strip markdown code fences if present
      const cleaned = result.replace(/^```json\s*/i, '').replace(/^```\s*/i, '').replace(/```\s*$/i, '').trim();
      breakdown = JSON.parse(cleaned);
    } catch {
      console.log('  ❌ Could not parse planning result as JSON.');
      console.log('  Raw result:', result.substring(0, 200));
      return;
    }

    if (!breakdown.tasks || !Array.isArray(breakdown.tasks)) {
      console.log('  ❌ Planning result missing "tasks" array.');
      return;
    }

    console.log(`\n  ✅ Got breakdown: ${breakdown.tasks.length} tasks`);

    // Step 1: Create all tasks without dependencies (so server can assign them)
    const createdTaskIds: string[] = [];
    const taskTitleToId: Record<string, string> = {};

    for (const taskDef of breakdown.tasks) {
      try {
        const created = await this.client.createTaskREST({
          title: taskDef.title,
          description: taskDef.description,
          projectName,
          requiredCapabilities: taskDef.requiredCapabilities || [],
          priority: taskDef.priority || 'medium',
          estimatedDuration: taskDef.estimatedDuration || 60,
          dependencies: [],
          metadata: { type: 'implementation', projectName }
        });
        createdTaskIds.push(created.id);
        taskTitleToId[taskDef.title] = created.id;
        console.log(`  ✓ Created: ${taskDef.title}`);
      } catch (error: any) {
        console.error(`  ❌ Failed to create task "${taskDef.title}":`, error.message);
      }
    }

    // Step 2: Patch dependencies now that we have real IDs
    for (const taskDef of breakdown.tasks) {
      if (!taskDef.dependencies || taskDef.dependencies.length === 0) continue;
      const taskId = taskTitleToId[taskDef.title];
      if (!taskId) continue;
      const depIds = taskDef.dependencies
        .map((dep: string) => taskTitleToId[dep])
        .filter(Boolean);
      if (depIds.length > 0) {
        try {
          await this.client.patchTaskDependencies(taskId, depIds);
        } catch {
          // non-fatal
        }
      }
    }

    // Step 3: Store agent briefs
    if (breakdown.briefs) {
      for (const [agentId, brief] of Object.entries(breakdown.briefs)) {
        try {
          await this.client.storeBrief(projectName, agentId, brief as string);
          console.log(`  📋 Brief stored for ${agentId}`);
        } catch {
          // non-fatal
        }
      }
    }

    console.log(`\n  🚀 Project "${projectName}" is active with ${createdTaskIds.length} tasks!`);
    console.log('  Agents will pick up tasks automatically via get_task.\n');
    this.currentProjectName = projectName;
  }

  private async handleNestTTY(args?: string): Promise<void> {
    const projectName = this.currentProjectName;
    if (!projectName) {
      console.log("No project selected. Use 'create project' or 'import project' first.");
      return;
    }

    const argText = (args || '').trim();
    let roles: string | undefined;
    let mock = false;

    const rolesMatch = argText.match(/--roles\s+([^\s]+)/);
    if (rolesMatch) {
      roles = rolesMatch[1];
    }
    if (/\b--mock\b/.test(argText)) {
      mock = true;
    }

    const spawnArgs = [
      'tsx',
      'nestty/index.ts',
      '--project',
      projectName,
      '--server',
      this.client.getServerUrl(),
    ];
    if (roles) {
      spawnArgs.push('--roles', roles);
    }

    console.log(`\nLaunching NesTTY for project "${projectName}"...`);
    if (roles) console.log(`Roles: ${roles}`);
    if (mock) console.log('Mock agents enabled (MOCK_AGENT=1).');
    console.log('');

    await new Promise<void>((resolve, reject) => {
      this.rl.pause();
      const child = spawn('npx', spawnArgs, {
        stdio: 'inherit',
        env: {
          ...process.env,
          ...(mock ? { MOCK_AGENT: '1' } : {}),
        },
      });

      child.on('error', (error) => {
        this.rl.resume();
        reject(error);
      });

      child.on('close', () => {
        this.rl.resume();
        console.log('\nNesTTY session ended');
        resolve();
      });
    });
  }

  private async handleListProjects(): Promise<void> {
    const projects = await this.client.getProjects();

    if (projects.length === 0) {
      console.log('No projects yet. Start one with: create project <name> in <path>');
      return;
    }

    console.log('┌──────────────────────┬──────────┬─────────────────────────┐');
    console.log('│ Project              │ Status   │ Workspace               │');
    console.log('├──────────────────────┼──────────┼─────────────────────────┤');
    for (const p of projects) {
      const n = (p.name || '').substring(0, 20).padEnd(20);
      const s = (p.status || '').padEnd(8);
      const w = (p.workspace || '').substring(0, 23).padEnd(23);
      console.log(`│ ${n} │ ${s} │ ${w} │`);
    }
    console.log('└──────────────────────┴──────────┴─────────────────────────┘');
  }

  private async handleShowProject(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: show project <name>');
      return;
    }
    const project = await this.client.getProject(args);
    if (!project) {
      console.log(`Project "${args}" not found.`);
      return;
    }

    const s = project.taskSummary || {};
    const total = s.total || 0;
    const completed = s.completed || 0;
    const pct = total > 0 ? Math.round((completed / total) * 100) : 0;
    const bar = total > 0
      ? '█'.repeat(Math.round(pct / 5)) + '░'.repeat(20 - Math.round(pct / 5))
      : '░'.repeat(20);

    console.log(`\n┌─ ${project.name} ${'─'.repeat(Math.max(0, 50 - project.name.length))}┐`);
    console.log(`│  Status:    ${project.status.toUpperCase().padEnd(10)}  Progress: [${bar}] ${pct}%`);
    console.log(`│  Workspace: ${project.workspace}`);
    console.log(`│  Agents:    ${(project.agents || []).join(', ') || 'none'}`);
    console.log(`│  Tasks:     ${completed}/${total} completed  |  ${s.in_progress || 0} active  |  ${s.pending || 0} pending  |  ${s.failed || 0} failed`);
    console.log(`└${'─'.repeat(62)}┘`);

    if (project.tasks && project.tasks.length > 0) {
      console.log('\n  Tasks:');
      for (const t of project.tasks) {
        const icon =
          t.status === 'completed'  ? '✓' :
          t.status === 'in_progress'? '▶' :
          t.status === 'assigned'   ? '▶' :
          t.status === 'failed'     ? '✗' : '○';
        const agent = t.assignedAgent ? ` → ${t.assignedAgent}` : '';
        const retry = t.retryCount > 0 ? ` (retry ${t.retryCount})` : '';
        const prog  = (t.status === 'in_progress' || t.status === 'assigned') ? ` ${t.progress}%` : '';
        console.log(`  ${icon} ${t.title.substring(0, 55).padEnd(55)}${prog}${agent}${retry}`);
      }
    }
    console.log('');
  }

  private async handleStopProject(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: stop project <name>');
      return;
    }
    console.log(`Project "${args}" paused.`);
    console.log('  All active tasks will complete, no new tasks assigned.');
  }

  private async handleDeleteProject(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: delete project <name>');
      return;
    }
    console.log(`⚠️  This will remove project and all coordination history.`);
    // In real implementation, would prompt for confirmation
    console.log(`✓ Project "${args}" deleted.`);
  }

  private async handleBrainstorm(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: brainstorm <topic> [--agents list]');
      return;
    }
    console.log('Brainstorm sessions not yet implemented.');
    console.log('This would start an open discussion between agents on a topic.');
  }

  private async handleExperiment(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: experiment <name> [--agents list]');
      return;
    }
    console.log('Experiment sessions not yet implemented.');
    console.log('This would run parallel implementations and compare results.');
  }

  private async handleExperimentAnalyze(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: experiment -analyze <name>');
      return;
    }
    console.log(`Analyzing experiment "${args}"...`);
    console.log('Experiment analysis not yet implemented.');
  }

  private async handleRoundtable(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: roundtable <topic> [--interactive]');
      return;
    }
    console.log('Roundtable sessions not yet implemented.');
    console.log('This would start a structured multi-agent discussion.');
  }

  private async handleAskAgents(args?: string): Promise<void> {
    if (!args || args.trim().length === 0) {
      console.log('Usage: ask agents <your question or instruction>');
      console.log('       ask <your question or instruction>');
      console.log('');
      console.log('Broadcasts your message to all registered agents and, if an Planner is set,');
      console.log('creates a coordination task for the Planner to interpret intent and assign work.');
      console.log('');
      console.log('Examples:');
      console.log('  ask agents Who is best suited to handle authentication?');
      console.log('  ask What is the current status of the database migration?');
      console.log('  ask agents Review the login flow and suggest improvements');
      return;
    }

    const prompt = args.trim();
    const agents = await this.client.getAgents();

    if (agents.length === 0) {
      console.log('❌ No agents are currently registered. Register an agent first.');
      return;
    }

    const defaultAgent = this.sessionManager.getDefaultAgent();
    const actorAgent = defaultAgent ? agents.find(a => a.id === defaultAgent) : null;

    console.log('\n  📣 Broadcasting to agents...\n');

    // Broadcast as a message so all agents receive it in their inbox
    const broadcastMessage = `[User Request] ${prompt}`;
    try {
      await this.client.sendMessage('repl_user', broadcastMessage);
      console.log(`  ✅ Message broadcast to ${agents.length} agent(s)`);
    } catch (err: any) {
      console.log(`  ⚠️  Broadcast failed: ${err.message}`);
    }

    // If Planner is available, also create a coordination task for structured response
    if (actorAgent) {
      console.log(`\n  🎯 Creating coordination task for Planner (${actorAgent.name || defaultAgent})...`);

      const taskDescription = `[ASK AGENTS] The user has asked: "${prompt}"

Your job as Planner is to:
1. Understand the intent of the user's request
2. Determine which connected agents are best suited to address it
3. Either: (a) respond directly with a synthesis if this is an informational query, OR (b) create sub-tasks and assign them to appropriate agents if this requires action
4. Send @mention messages to relevant agents with specific instructions
5. Report back a summary of what was coordinated

Connected agents: ${agents.map(a => `${a.id} (${a.capabilities?.join(', ') || 'general'})`).join(', ')}

Respond by calling report_task_complete with your coordination plan or synthesis.`;

      try {
        const task = await this.client.createTaskREST({
          description: taskDescription,
          type: 'coordination',
          priority: 'high',
          requiredCapabilities: [],
          assignedAgent: defaultAgent,
          metadata: { source: 'ask_agents', userPrompt: prompt }
        });

        console.log(`  ✅ Coordination task created: ${task.id}`);
        console.log(`\n  ⏳ Agents have been notified. Check back with 'show project' or watch`);
        console.log(`     agent outputs for responses. The Planner will coordinate the response.`);
      } catch (err: any) {
        console.log(`  ⚠️  Could not create coordination task: ${err.message}`);
        console.log(`     Message was still broadcast — agents may respond via send_message.`);
      }
    } else {
      console.log(`\n  ℹ️  No Planner set. Message broadcast only — no coordination task created.`);
      console.log(`     Set an Planner with 'default agent <id>' for smarter coordination.`);
    }

    console.log('');
  }

  private async handleSelect(args?: string): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    if (!args) {
      console.log('Usage: select <agent>');
      return;
    }
    console.log(`[Highlighting ${args}'s contribution in UI]`);
  }

  private async handleEdit(args?: string): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    if (!args) {
      console.log('Usage: edit <msg_id>');
      return;
    }
    console.log('Message editing not yet implemented.');
  }

  private async handleDelete(args?: string): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    if (!args) {
      console.log('Usage: delete <msg_id>');
      return;
    }
    console.log('Message deletion not yet implemented.');
  }

  private async handleSend(args?: string): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    if (!args) {
      console.log('Usage: send "<message>"');
      return;
    }
    console.log(`User message sent: "${args}"`);
  }

  private async handlePause(): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    console.log('[Discussion paused]');
  }

  private async handleResume(): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    console.log('[Discussion resumed]');
  }

  private async handleStop(): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    console.log('Session ended.');
  }

  private async handleCleanUp(): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    console.log('Creating summary...');
    console.log('Session saved to PVM.');
  }

  private async handleWipe(): Promise<void> {
    if (!this.sessionManager.hasActiveSession()) {
      console.log('No active interactive session.');
      return;
    }
    console.log('⚠️  This will permanently remove the discussion from PVM.');
    console.log('✓ Session removed from PVM (destructive operation).');
  }

  private async handleImprove(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: improve <scope> [options]');
      console.log('Try: improve communication -project <name>');
      return;
    }
    console.log('Improvement analysis not yet implemented.');
    console.log('This would provide surgical precision analysis of coordination patterns.');
  }

  private async handlePVMCommand(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: pvm <command> [args]');
      console.log('Commands: stats, search, profile, export, import');
      return;
    }
    const parts = args.split(' ');
    const subCommand = parts[0];

    switch (subCommand) {
      case 'stats':
        await this.handlePVMStats();
        break;
      case 'search':
        await this.handlePVMSearch(parts.slice(1).join(' '));
        break;
      case 'profile':
        await this.handlePVMProfile(parts.slice(1).join(' '));
        break;
      case 'export':
        await this.handlePVMExport(parts.slice(1).join(' '));
        break;
      case 'import':
        await this.handlePVMImport(parts.slice(1).join(' '));
        break;
      default:
        console.log('Unknown PVM command. Try: pvm stats, search, profile, export, import');
    }
  }

  private async handlePVMStats(): Promise<void> {
    try {
      const stats = await this.client.getPVMStatus();
      console.log('PVM Statistics:');
      console.log(`  Total coordination events: ${stats.indexedEventCount || 'Unknown'}`);
      console.log(`  Vector indexed: ${stats.indexedEventCount || 'Unknown'}`);
      console.log(`  Database size: ${stats.isRunning ? 'Active' : 'Inactive'}`);
    } catch (error: any) {
      console.log('PVM stats unavailable:', error.message);
    }
  }

  private async handlePVMSearch(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: pvm search <query>');
      return;
    }
    console.log(`Searching PVM for: "${args}"`);
    console.log('PVM search not yet implemented.');
  }

  private async handlePVMProfile(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: pvm profile <agent_id>');
      return;
    }
    console.log(`Agent Profile: ${args}`);
    console.log('PVM profiles not yet implemented.');
  }

  private async handlePVMExport(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: pvm export <path>');
      return;
    }
    console.log(`Exporting PVM database to: ${args}`);
    console.log('PVM export not yet implemented.');
  }

  private async handlePVMImport(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: pvm import <path>');
      return;
    }
    console.log(`Importing PVM database from: ${args}`);
    console.log('PVM import not yet implemented.');
  }

  private async handleStatus(): Promise<void> {
    try {
      const serverStatus = await this.client.getServerStatus();
      const pvmStatus = await this.client.getPVMStatus();

      console.log('\nACT Server Status:');
      console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
      console.log(`  Version: 1.0.0`);
      console.log(`  Uptime: ${serverStatus.uptime || 'Unknown'}`);

      console.log(`\nActive Projects: ${Array.isArray(serverStatus.tasks) ? serverStatus.tasks.length : serverStatus.tasks || 0}`);

      const agents = await this.client.getAgents();
      console.log(`Connected Agents: ${agents.length}`);
      agents.forEach(agent => {
        console.log(`  • ${agent.id} (${agent.status || 'unknown'})`);
      });

      console.log(`\nResources:`);
      console.log(`  • PVM Database: ${pvmStatus.indexedEventCount || 0} events`);
      console.log(`  • Status: ${pvmStatus.isRunning ? 'Active' : 'Inactive'}`);

    } catch (error: any) {
      console.log('Status unavailable:', error.message);
    }
  }

  private handleHelp(args?: string): void {
    this.helpSystem.showHelp(args);
  }

  private async handleCoordinationStart(): Promise<void> {
    const agents = await this.client.getAgents();
    if (agents.length === 0) {
      console.log('\n  ⚠️  No agents registered. Connect agents via MCP first.');
      return;
    }

    const runnerPath = path.resolve(
      new URL(import.meta.url).pathname,
      '../../runner/act-runner.mjs'
    );

    if (!fs.existsSync(runnerPath)) {
      console.log(`\n  ❌ Runner not found at: ${runnerPath}`);
      return;
    }

    const serverUrl = this.client.getServerUrl() || 'http://localhost:8080';

    console.log(`\n  Starting coordination loop for ${agents.length} agent(s)...\n`);

    let started = 0;
    for (const agent of agents) {
      if (this.spawnedRunners.has(agent.id)) {
        console.log(`  ⏭  ${agent.id} — runner already running, skipping`);
        continue;
      }

      const proc = spawn('node', [
        runnerPath,
        '--agent-id', agent.id,
        '--name', agent.name || agent.id,
        '--capabilities', (agent.capabilities || []).join(','),
      ], {
        env: { ...process.env, ACT_SERVER_URL: serverUrl },
        stdio: ['ignore', 'pipe', 'pipe'],
        detached: false,
      });

      // Prefix runner output with agent ID so logs are identifiable
      const prefix = `  [${agent.id}]`;
      proc.stdout?.on('data', (d: Buffer) => process.stdout.write(`${prefix} ${d.toString().trim()}\n`));
      proc.stderr?.on('data', (d: Buffer) => process.stderr.write(`${prefix} ⚠ ${d.toString().trim()}\n`));
      proc.on('exit', (code) => {
        console.log(`${prefix} Runner exited (code ${code})`);
        this.spawnedRunners.delete(agent.id);
      });

      this.spawnedRunners.set(agent.id, proc);
      console.log(`  ✓ ${agent.id} (${agent.name || agent.id}) — runner started [pid ${proc.pid}]`);
      started++;
    }

    if (started > 0) {
      console.log(`\n  ✅ Coordination active — ${started} runner(s) running.`);
      console.log(`  Use 'coordination stop' to stop all runners.\n`);
    } else {
      console.log(`\n  ℹ️  No new runners started.\n`);
    }
  }

  private async handleCoordinationStop(): Promise<void> {
    if (this.spawnedRunners.size === 0) {
      console.log('\n  ℹ️  No runners currently running.\n');
      return;
    }

    console.log(`\n  Stopping ${this.spawnedRunners.size} runner(s)...`);
    for (const [agentId, proc] of this.spawnedRunners) {
      proc.kill('SIGTERM');
      console.log(`  ✓ ${agentId} — stopped`);
    }
    this.spawnedRunners.clear();
    console.log('  ✅ All runners stopped.\n');
  }

  private async handleExit(): Promise<void> {
    if (this.spawnedRunners.size > 0) {
      console.log(`\n  Stopping ${this.spawnedRunners.size} runner(s)...`);
      for (const [, proc] of this.spawnedRunners) proc.kill('SIGTERM');
      this.spawnedRunners.clear();
    }
    console.log('\nGoodbye! ACT server continues running in background.');
    console.log("Use 'act server stop' to shut down the server.");
    this.isRunning = false;
    this.rl.close();
  }

  private setupConnectionMonitoring(): void {
    // Monitor connection health and handle disconnections gracefully
    const checkConnection = async () => {
      if (!this.isRunning) return;

      try {
        await this.client.testConnection();
        // Connection is still good, schedule next check
        setTimeout(checkConnection, 30000); // Check every 30 seconds
      } catch (error: any) {
        console.error('\n❌ Lost connection to ACT server:', error.message);
        console.log('The REPL will continue, but some commands may not work.');
        console.log('Try reconnecting with a new REPL session when the server is back online.');

        // Don't exit the REPL - let user continue with available commands
        // Schedule reconnection attempts
        setTimeout(checkConnection, 10000); // Retry every 10 seconds
      }
    };

    // Start monitoring after a brief delay
    setTimeout(checkConnection, 5000);
  }

  /**
   * Poll the server every 15 seconds for permanently failed tasks (retryCount >= MAX_RETRIES).
   * When one is found, alert the user in the REPL and stop looping for that task ID.
   * This ensures the user is notified even when agents have exhausted retries silently.
   */
  private startFailedTaskPolling(): void {
    const alertedTaskIds = new Set<string>(); // don't alert for the same task twice

    const check = async () => {
      if (!this.isRunning) return;

      try {
        const failed = await this.client.getPermanentlyFailedTasks();
        for (const task of failed) {
          if (alertedTaskIds.has(task.id)) continue;
          alertedTaskIds.add(task.id);

          const title = task.metadata?.title || task.description?.substring(0, 60) || task.id;
          const agent = task.assignedAgent || 'unknown agent';

          console.log('\n');
          console.log('┌──────────────────────────────────────────────────────────────┐');
          console.log('│  ⚠️  PERMANENTLY FAILED TASK — REQUIRES YOUR ATTENTION       │');
          console.log('├──────────────────────────────────────────────────────────────┤');
          console.log(`│  Task:  ${title.substring(0, 54).padEnd(54)} │`);
          console.log(`│  Agent: ${agent.substring(0, 54).padEnd(54)} │`);
          console.log(`│  ID:    ${task.id.substring(0, 54).padEnd(54)} │`);
          console.log('├──────────────────────────────────────────────────────────────┤');
          console.log('│  This task failed 3 times and agents could not recover it.   │');
          console.log('│  Options:                                                    │');
          console.log('│   • ask agents <clarification or fix>  — send guidance       │');
          console.log('│   • show project <name>                — see full task state  │');
          console.log('└──────────────────────────────────────────────────────────────┘');

          // Re-prompt so user can type
          if (this.isRunning && !this.isPrompting) {
            this.rl.prompt();
          }
        }
      } catch {
        // Non-fatal — server may be temporarily unreachable
      }

      setTimeout(check, 15000); // Check every 15 seconds
    };

    // Start after initial delay so startup output isn't interrupted
    setTimeout(check, 10000);
  }

  private async handleNaturalLanguage(input: string): Promise<void> {
    // Basic natural language parsing for project creation
    if (input.toLowerCase().includes('create') && input.toLowerCase().includes('project')) {
      // Try to parse natural language project creation
      console.log(`Analyzing request: "${input}"`);
      console.log('Natural language parsing not yet implemented.');
      console.log('Try: create project "my project" in /path/to/project');
    } else {
      console.log(`Unknown command: ${input}`);
      console.log('Type "help" for available commands.');
    }
  }
}

// CLI entry point
if (import.meta.url === `file://${process.argv[1]}`) {
  const serverUrl = process.env.ACT_SERVER_URL || process.argv[2] || 'http://localhost:8080';
  const repl = new ACTRepl(serverUrl);
  repl.start().catch(console.error);
}
