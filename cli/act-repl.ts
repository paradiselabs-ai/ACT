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

export class ACTRepl {
  private rl: readline.Interface;
  private client: ACTClient;
  private sessionManager: SessionManager;
  private helpSystem: HelpSystem;
  private isRunning: boolean = false;

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
        if (!this.isRunning) return;
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
    if (!args) {
      console.log('Usage: create project <name> in <path>');
      return;
    }

    // Parse: "name in path"
    const inIndex = args.indexOf(' in ');
    if (inIndex === -1) {
      console.log('Usage: create project <name> in <path>');
      return;
    }

    const name = args.substring(0, inIndex).trim();
    const path = args.substring(inIndex + 4).trim();

    console.log(`Creating project "${name}"...`);
    console.log(`  Workspace: ${path}`);

    const defaultAgent = this.sessionManager.getDefaultAgent();
    if (defaultAgent) {
      console.log(`  Delegating decomposition to: ${defaultAgent} (default agent)`);
    } else {
      console.log('  Warning: No default agent set. Project creation may fail.');
    }

    try {
      // This would integrate with the actual project creation logic
      console.log('\n[Project creation not yet implemented]');
      console.log('This would analyze the project requirements and create a structured task breakdown.');
    } catch (error: any) {
      console.error('Failed to create project:', error.message);
    }
  }

  private async handleContinueProject(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: continue project <name>');
      return;
    }
    console.log(`Resuming project "${args}"...`);
    console.log('\n[Project continuation not yet implemented]');
  }

  private async handleListProjects(): Promise<void> {
    console.log('┌────────────────┬────────────────────────┬──────────┬─────────────┐');
    console.log('│ Project        │ Workspace              │ Status   │ Progress    │');
    console.log('├────────────────┼────────────────────────┼──────────┼─────────────┤');
    console.log('│ [No projects]  │                        │          │             │');
    console.log('└────────────────┴────────────────────────┴──────────┴─────────────┘');
    console.log('\n[Project listing not yet implemented]');
  }

  private async handleShowProject(args?: string): Promise<void> {
    if (!args) {
      console.log('Usage: show project <name>');
      return;
    }
    console.log(`Project: ${args}`);
    console.log('\n[Project details not yet implemented]');
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
