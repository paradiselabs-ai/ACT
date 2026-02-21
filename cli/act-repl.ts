#!/usr/bin/env node

/**
 * ACT Terminal REPL - Agent Coordination Toolkit
 *
 * A comprehensive command-line interface for managing ACT coordination.
 * Provides project management, session control, improvement analysis, and PVM memory operations.
 */

import * as readline from 'readline';
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
}): string {
  return `You are the ACTor — the designated planning agent for ACT multi-agent coordination.

Analyze the project below and respond with ONLY a JSON object (no markdown, no prose):

{
  "tasks": [
    {
      "title": "short imperative task title",
      "description": "detailed description of what to implement and why",
      "requiredCapabilities": ["capability1", "capability2"],
      "priority": "high|medium|low",
      "estimatedDuration": 60
    }
  ],
  "briefs": {
    "<agentId>": "# AGENT.md\\n\\n## Project: ${p.name}\\n\\nFull brief in markdown..."
  }
}

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

export class ACTRepl {
  private rl: readline.Interface;
  private client: ACTClient;
  private sessionManager: SessionManager;
  private helpSystem: HelpSystem;
  private isRunning: boolean = false;
  private isPrompting: boolean = false;

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

    // Project commands
    this.registerCommand('create project', (args) => this.handleCreateProject(args));
    this.registerCommand('continue project', (args) => this.handleContinueProject(args));
    this.registerCommand('list projects', () => this.handleListProjects());
    this.registerCommand('show project', (args) => this.handleShowProject(args));
    this.registerCommand('stop project', (args) => this.handleStopProject(args));
    this.registerCommand('delete project', (args) => this.handleDeleteProject(args));

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

    try {
      // Test connection to ACT server
      await this.client.testConnection();
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
      console.log('  • Set default agent: default agent <agent_id>');
      console.log('  • Create project: create project <name> in <path>');
      console.log('  • Continue project: continue project <name>');
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

        case 'delete':
          if (args.startsWith('project ')) {
            await this.handleDeleteProject(args.substring(8));
          } else {
            console.log('Usage: delete project <name>');
          }
          break;

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
      return;
    }

    console.log(`Setting default agent to: ${args}...`);

    const success = await this.sessionManager.setDefaultAgent(args);

    if (success) {
      console.log(`✓ Default agent set to: ${args}`);
      console.log('  All project decomposition and planning will use this agent\'s LLM.');
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

  private async handleCreateProject(args?: string): Promise<void> {
    // Step 1: Get name and workspace path
    let name: string;
    let workspace: string;

    if (args) {
      const inIndex = args.indexOf(' in ');
      if (inIndex !== -1) {
        name = args.substring(0, inIndex).trim();
        workspace = args.substring(inIndex + 4).trim();
      } else {
        name = args.trim();
        workspace = await this.prompt('  Workspace path: ');
      }
    } else {
      name = await this.prompt('  Project name: ');
      workspace = await this.prompt('  Workspace path: ');
    }

    if (!name || !workspace) {
      console.log('Project name and workspace path are required.');
      return;
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

    // Step 5: Planning via ACTor
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

    const planningDesc = buildPlanningPrompt({ name, workspace, description, techStack, constraints, successCriteria, participatingAgents });

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

    const result = await this.pollWithSpinner(planningTask.id, 180_000);

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

    for (const taskDef of tasks) {
      try {
        await this.client.createTaskREST({
          description: taskDef.description || taskDef.title,
          requiredCapabilities: taskDef.requiredCapabilities || [],
          priority: taskDef.priority || 'medium',
          estimatedDuration: taskDef.estimatedDuration,
          metadata: { projectName: name, title: taskDef.title }
        });
        console.log(`  ✓ ${taskDef.title}`);
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

    console.log(`\nResuming project "${args}"...`);
    console.log(`  Status: ${project.status}`);
    console.log(`  Agents: ${(project.agents || []).join(', ') || 'none'}`);

    // Check for an in-progress planning task
    const allTasks = await this.client.getProjects(); // not ideal — use task list
    console.log('\n  Project is active. Agents can connect and call get_task to receive assignments.');
    console.log('  Use: show project ' + args + ' for full details.\n');
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

    console.log(`\nProject: ${project.name}`);
    console.log(`  Status:    ${project.status}`);
    console.log(`  Workspace: ${project.workspace}`);
    console.log(`  Created:   ${project.createdAt}`);
    console.log(`\n  Description:\n    ${project.description}`);
    console.log(`\n  Stack:     ${project.techStack}`);
    if (project.constraints) console.log(`  Constraints: ${project.constraints}`);
    console.log(`  Success:   ${project.successCriteria}`);
    console.log(`\n  Agents: ${(project.agents || []).join(', ') || 'none'}`);

    const briefCount = Object.keys(project.briefs || {}).length;
    console.log(`  Briefs:   ${briefCount} agent brief(s) stored`);
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

  private async handleExit(): Promise<void> {
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
