import { EventEmitter } from 'events';
import { logger } from '../utils/logger';
import { AgentRole } from '../types/roles';

export interface Agent {
  id: string;
  name: string;
  projectName: string;
  capabilities: string[];
  status: 'online' | 'busy' | 'offline';
  model?: string;
  provider?: string;
  socketId?: string;
  currentTask?: string;
  lastSeen: Date;
  performanceScore: number;
  tasksCompleted: number;
  averageTaskTime: number;
  role?: AgentRole;
}

export interface AgentCapability {
  name: string;
  proficiency: number; // 0-1 score
}

export class AgentRegistry extends EventEmitter {
  private agents: Map<string, Agent> = new Map();
  private socketToAgent: Map<string, string> = new Map();

  isRegistered(agentId: string): boolean {
    return this.agents.has(agentId);
  }

  async registerAgent(agentId: string, agentData: Partial<Agent>): Promise<Agent> {
    if (!agentData.projectName) {
      throw new Error(`projectName is required to register agent ${agentId}`);
    }
    const existingAgent = this.agents.get(agentId);

    const agent: Agent = {
      id: agentId,
      name: agentData.name || agentId,
      projectName: agentData.projectName,
      capabilities: agentData.capabilities || [],
      model: agentData.model,
      provider: agentData.provider,
      status: 'online',
      socketId: agentData.socketId,
      currentTask: undefined,
      lastSeen: new Date(),
      performanceScore: existingAgent?.performanceScore || 1.0,
      tasksCompleted: existingAgent?.tasksCompleted || 0,
      averageTaskTime: existingAgent?.averageTaskTime || 0,
      role: agentData.role ?? AgentRole.DEVELOPER,
      ...agentData
    };

    console.log(`📇 registerAgent payload for ${agentId}:`, {
      name: agent.name,
      capabilities: agent.capabilities,
      model: agent.model,
      provider: agent.provider,
      socketId: agent.socketId
    });

    this.agents.set(agentId, agent);

    if (agent.socketId) {
      this.socketToAgent.set(agent.socketId, agentId);
    }

    this.emit('agent_registered', agent);
    logger.info(`Agent registered: ${agentId} [${agent.capabilities.join(', ')}]`);

    return agent;
  }

  async updateAgentStatus(agentId: string, status: Agent['status'], currentTask?: string): Promise<void> {
    const agent = this.agents.get(agentId);
    if (!agent) {
      throw new Error(`Agent ${agentId} not found`);
    }

    agent.status = status;
    agent.lastSeen = new Date();

    if (currentTask !== undefined) {
      agent.currentTask = currentTask;
    }

    // When an agent goes online, they have no current task (available for assignment)
    if (status === 'online') {
      agent.currentTask = undefined;
    }

    if (status === 'offline') {
      if (agent.socketId) {
        this.socketToAgent.delete(agent.socketId);
      }
      agent.socketId = undefined;
    }

    this.emit('agent_status_updated', agent);
  }

  removeAgent(agentId: string): boolean {
    if (!this.agents.has(agentId)) return false;
    this.agents.delete(agentId);
    // Clean up any socket→agent mapping for this agent
    for (const [socketId, id] of this.socketToAgent.entries()) {
      if (id === agentId) this.socketToAgent.delete(socketId);
    }
    this.emit('agent_removed', agentId);
    return true;
  }

  getAgent(agentId: string): Agent | undefined {
    return this.agents.get(agentId);
  }

  getAgentBySocketId(socketId: string): Agent | undefined {
    const agentId = this.socketToAgent.get(socketId);
    return agentId ? this.agents.get(agentId) : undefined;
  }

  getAllAgents(projectName?: string): Agent[] {
    const all = Array.from(this.agents.values());
    return projectName ? all.filter(a => a.projectName === projectName) : all;
  }

  getAvailableAgents(projectName?: string): Agent[] {
    return this.getAllAgents(projectName).filter(agent =>
      agent.status === 'online' && !agent.currentTask
    );
  }

  getAgentsByCapability(capability: string, projectName?: string): Agent[] {
    return this.getAllAgents(projectName).filter(agent =>
      agent.capabilities.includes(capability) && agent.status !== 'offline'
    );
  }

  getOptimalAgent(requiredCapabilities: string[], projectName?: string): Agent | null {
    const availableAgents = this.getAvailableAgents(projectName);

    if (availableAgents.length === 0) {
      return null;
    }

    // Filter to agents with at least one matching capability when requirements
    // are specified. This prevents Go tasks being routed to frontend_dev just
    // because frontend_dev has a higher performance score. Tasks with no
    // required capabilities still score over all available agents.
    const eligible = requiredCapabilities.length > 0
      ? availableAgents.filter(a => requiredCapabilities.some(cap => a.capabilities.includes(cap)))
      : availableAgents;

    if (eligible.length === 0) {
      logger.info(`assign_decision: no_capability_match required=[${requiredCapabilities.join(',')}] available_agents=${availableAgents.length}`);
      return null;
    }

    const scoredAgents = eligible.map(agent => {
      const capabilityScore = this.calculateCapabilityMatch(agent, requiredCapabilities);
      const performanceScore = agent.performanceScore;
      const workloadScore = agent.status === 'online' ? 1.0 : 0.5;

      const totalScore = (
        capabilityScore * 0.6 +
        performanceScore * 0.3 +
        workloadScore * 0.1
      );

      return { agent, score: totalScore, capabilityScore };
    });

    scoredAgents.sort((a, b) => b.score - a.score);
    const best = scoredAgents[0];
    if (best) {
      logger.info(`assign_decision: selected=${best.agent.id} score=${best.score.toFixed(2)} capability_score=${best.capabilityScore.toFixed(2)} required=[${requiredCapabilities.join(',')}]`);
    }
    return best?.agent || null;
  }

  private calculateCapabilityMatch(agent: Agent, requiredCapabilities: string[]): number {
    if (requiredCapabilities.length === 0) {
      return 1.0;
    }

    const matchingCapabilities = requiredCapabilities.filter(cap =>
      agent.capabilities.includes(cap)
    );

    return matchingCapabilities.length / requiredCapabilities.length;
  }

  async updateAgentPerformance(agentId: string, taskDuration: number, success: boolean): Promise<void> {
    const agent = this.agents.get(agentId);
    if (!agent) {
      return;
    }

    // Update performance metrics
    if (success) {
      agent.tasksCompleted += 1;

      // Calculate new average task time
      if (agent.averageTaskTime === 0) {
        agent.averageTaskTime = taskDuration;
      } else {
        agent.averageTaskTime = (agent.averageTaskTime + taskDuration) / 2;
      }

      // Update performance score (success rate and efficiency)
      const efficiency = Math.max(0.1, Math.min(2.0, 60000 / taskDuration)); // Normalize around 1 minute
      agent.performanceScore = Math.min(2.0, agent.performanceScore * 0.9 + efficiency * 0.1);
    } else {
      // Decrease performance score on failure
      agent.performanceScore = Math.max(0.1, agent.performanceScore * 0.8);
    }

    this.emit('agent_performance_updated', agent);
  }

  getAgentCount(): number {
    return this.agents.size;
  }

  clearAll(): number {
    const count = this.agents.size;
    this.agents.clear();
    this.socketToAgent.clear();
    return count;
  }

  getOnlineAgentCount(projectName?: string): number {
    return this.getAllAgents(projectName).filter(agent => agent.status !== 'offline').length;
  }

  // Bulk restore agents from event log (used by restoreFromLog)
  restoreAgents(agentMap: Map<string, any>): void {
    for (const [id, agentData] of agentMap.entries()) {
      if (!agentData.projectName) {
        logger.warn(`restoreAgents: skipping ${id} — no projectName on event payload (pre-fix log entry)`);
        continue;
      }
      this.agents.set(id, {
        id: agentData.id,
        name: agentData.name,
        projectName: agentData.projectName,
        capabilities: agentData.capabilities || [],
        status: agentData.status || 'offline',
        model: agentData.model,
        provider: agentData.provider,
        socketId: undefined,
        currentTask: undefined,
        lastSeen: new Date(agentData.lastSeen || Date.now()),
        performanceScore: agentData.performanceScore || 1.0,
        tasksCompleted: agentData.tasksCompleted || 0,
        averageTaskTime: agentData.averageTaskTime || 0,
        role: agentData.role
      });
    }
  }

  // Health check method
  async performHealthCheck(): Promise<void> {
    const now = new Date();
    const staleThreshold = 5 * 60 * 1000; // 5 minutes

    for (const agent of this.agents.values()) {
      if (agent.status !== 'offline' && (now.getTime() - agent.lastSeen.getTime()) > staleThreshold) {
        logger.warn(`Agent ${agent.id} appears stale, marking as offline`);
        await this.updateAgentStatus(agent.id, 'offline');
      }
    }
  }
}

export default AgentRegistry;