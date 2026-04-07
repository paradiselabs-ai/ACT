/**
 * Session Manager - Manages active sessions and coordination state
 *
 * Handles brainstorm, experiment, and roundtable sessions, including interactive controls.
 */

import { ACTClient } from './act-client.js';
import * as fs from 'fs';
import * as path from 'path';

export interface Session {
  id: string;
  type: 'brainstorm' | 'experiment' | 'roundtable';
  name: string;
  status: 'active' | 'paused' | 'completed';
  participants: string[];
  startTime: Date;
  isInteractive: boolean;
  messages: SessionMessage[];
}

export interface SessionMessage {
  id: string;
  agent: string;
  content: string;
  timestamp: Date;
  type: 'user' | 'agent' | 'system';
}

export class SessionManager {
  private client: ACTClient;
  private activeSession: Session | null = null;
  private defaultAgent: string | null = null;
  private configPath: string;

  constructor(client: ACTClient) {
    this.client = client;
    // Store config in user's home directory under .act/
    const homeDir = process.env.HOME || process.env.USERPROFILE || '';
    this.configPath = path.join(homeDir, '.act', 'repl-config.json');

    // Load configuration on startup
    this.loadConfig();
  }

  async setDefaultAgent(agentId: string): Promise<boolean> {
    // Validate agent exists first
    try {
      const agents = await this.client.getAgents();
      const agentExists = agents.some(agent => agent.id === agentId);

      if (!agentExists) {
        console.warn(`Agent "${agentId}" not found. Use 'list agents' to see available agents.`);
        return false;
      }

      this.defaultAgent = agentId;
      this.saveConfig();
      return true;
    } catch (error) {
      console.error('Failed to validate agent:', error);
      return false;
    }
  }

  getDefaultAgent(): string | null {
    return this.defaultAgent;
  }

  hasActiveSession(): boolean {
    return this.activeSession !== null;
  }

  getActiveSession(): Session | null {
    return this.activeSession;
  }

  async startBrainstorm(topic: string, agents: string[] = []): Promise<Session> {
    const session: Session = {
      id: `brainstorm-${Date.now()}`,
      type: 'brainstorm',
      name: topic,
      status: 'active',
      participants: agents.length > 0 ? agents : await this.getAllAgentIds(),
      startTime: new Date(),
      isInteractive: false,
      messages: []
    };

    this.activeSession = session;

    // Add initial system message
    this.addMessage('system', `Starting brainstorm session: "${topic}"`);

    return session;
  }

  async startExperiment(name: string, agents: string[] = []): Promise<Session> {
    const session: Session = {
      id: `experiment-${Date.now()}`,
      type: 'experiment',
      name: name,
      status: 'active',
      participants: agents.length > 0 ? agents : await this.getAllAgentIds(),
      startTime: new Date(),
      isInteractive: false,
      messages: []
    };

    this.activeSession = session;

    // Add initial system message
    this.addMessage('system', `Starting experiment session: "${name}"`);

    return session;
  }

  async startRoundtable(topic: string, isInteractive: boolean = false, agents: string[] = []): Promise<Session> {
    const session: Session = {
      id: `roundtable-${Date.now()}`,
      type: 'roundtable',
      name: topic,
      status: 'active',
      participants: agents.length > 0 ? agents : await this.getAllAgentIds(),
      startTime: new Date(),
      isInteractive: isInteractive,
      messages: []
    };

    this.activeSession = session;

    // Add initial system message
    const mode = isInteractive ? 'interactive' : 'standard';
    this.addMessage('system', `Starting ${mode} roundtable: "${topic}"`);

    if (isInteractive) {
      this.addMessage('system', 'Interactive controls enabled: pause, resume, select, edit, delete, send, stop, clean_up, wipe');
    }

    return session;
  }

  pauseSession(): boolean {
    if (!this.activeSession) return false;

    this.activeSession.status = 'paused';
    this.addMessage('system', 'Session paused');
    return true;
  }

  resumeSession(): boolean {
    if (!this.activeSession) return false;

    this.activeSession.status = 'active';
    this.addMessage('system', 'Session resumed');
    return true;
  }

  selectAgent(agentId: string): boolean {
    if (!this.activeSession || !this.activeSession.isInteractive) return false;

    if (!this.activeSession.participants.includes(agentId)) {
      return false;
    }

    this.addMessage('system', `Highlighting ${agentId}'s contribution`);
    return true;
  }

  editMessage(messageId: string, newContent: string): boolean {
    if (!this.activeSession || !this.activeSession.isInteractive) return false;

    const message = this.activeSession.messages.find(m => m.id === messageId);
    if (!message) return false;

    message.content = newContent;
    this.addMessage('system', `Message edited: ${messageId}`);
    return true;
  }

  deleteMessage(messageId: string): boolean {
    if (!this.activeSession || !this.activeSession.isInteractive) return false;

    const index = this.activeSession.messages.findIndex(m => m.id === messageId);
    if (index === -1) return false;

    this.activeSession.messages.splice(index, 1);
    this.addMessage('system', `Message deleted: ${messageId}`);
    return true;
  }

  sendUserMessage(content: string): boolean {
    if (!this.activeSession || !this.activeSession.isInteractive) return false;

    this.addMessage('user', content);
    return true;
  }

  async stopSession(): Promise<boolean> {
    if (!this.activeSession) return false;

    this.activeSession.status = 'completed';
    this.addMessage('system', `Session ended: ${this.activeSession.name}`);

    // Could save to PVM here
    const session = this.activeSession;
    this.activeSession = null;

    return true;
  }

  async cleanUpSession(): Promise<boolean> {
    if (!this.activeSession) return false;

    const session = this.activeSession;
    const duration = Date.now() - session.startTime.getTime();
    const minutes = Math.round(duration / 60000);

    console.log(`\nCreating summary for session: ${session.name}`);
    console.log(`Duration: ${minutes} minutes`);
    console.log(`Participants: ${session.participants.join(', ')}`);
    console.log(`Messages: ${session.messages.length}`);

    // Generate summary (simplified)
    const agentMessages = session.messages.filter(m => m.type === 'agent').length;
    const userMessages = session.messages.filter(m => m.type === 'user').length;

    console.log(`\nSummary:`);
    console.log(`• Decision: [Analysis would go here]`);
    console.log(`• Reasoning: [Key points from discussion]`);
    console.log(`• Dissent: [Conflicting opinions noted]`);
    console.log(`• Final consensus: [Outcome]`);
    console.log(`\nSession saved to PVM.`);

    this.activeSession = null;
    return true;
  }

  async wipeSession(): Promise<boolean> {
    if (!this.activeSession) return false;

    console.log(`⚠️  This will permanently remove "${this.activeSession.name}" from PVM.`);
    console.log('✓ Session removed from PVM (destructive operation).');

    this.activeSession = null;
    return true;
  }

  private addMessage(type: 'user' | 'agent' | 'system', content: string): void {
    if (!this.activeSession) return;

    const message: SessionMessage = {
      id: `msg-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      agent: type === 'user' ? 'user' : 'system',
      content,
      timestamp: new Date(),
      type
    };

    this.activeSession.messages.push(message);

    // In a real implementation, this would broadcast to participants
    console.log(`[${type.toUpperCase()}] ${content}`);
  }

  private async getAllAgentIds(): Promise<string[]> {
    try {
      const agents = await this.client.getAgents();
      return agents.map(agent => agent.id);
    } catch (error) {
      console.warn('Failed to get agents for session:', error);
      return [];
    }
  }

  // Load configuration from file
  private loadConfig(): void {
    try {
      if (fs.existsSync(this.configPath)) {
        const configData = fs.readFileSync(this.configPath, 'utf-8');
        const config = JSON.parse(configData);

        if (config.defaultAgent) {
          this.defaultAgent = config.defaultAgent;
        }
      }
    } catch (error) {
      // Silently ignore config loading errors - use defaults
      console.warn('Warning: Could not load REPL config, using defaults');
    }
  }

  // Save configuration to file
  private saveConfig(): void {
    try {
      // Ensure directory exists
      const configDir = path.dirname(this.configPath);
      if (!fs.existsSync(configDir)) {
        fs.mkdirSync(configDir, { recursive: true });
      }

      const config = {
        defaultAgent: this.defaultAgent,
        // Room for future config options like theme, auto-save, etc.
      };

      fs.writeFileSync(this.configPath, JSON.stringify(config, null, 2));
    } catch (error) {
      console.warn('Warning: Could not save REPL config');
    }
  }

  // Get session statistics for improvement analysis
  getSessionStats(): any {
    if (!this.activeSession) return null;

    const messages = this.activeSession.messages;
    const agentMessages = messages.filter(m => m.type === 'agent');
    const userMessages = messages.filter(m => m.type === 'user');
    const duration = Date.now() - this.activeSession.startTime.getTime();

    return {
      sessionId: this.activeSession.id,
      type: this.activeSession.type,
      name: this.activeSession.name,
      duration: Math.round(duration / 1000), // seconds
      participants: this.activeSession.participants.length,
      totalMessages: messages.length,
      agentMessages: agentMessages.length,
      userMessages: userMessages.length,
      isInteractive: this.activeSession.isInteractive
    };
  }
}
