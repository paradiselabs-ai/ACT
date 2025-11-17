import { EventEmitter } from 'events';
import { logger } from '../utils/logger';

export interface Agent {
  id: string;
  name: string;
  capabilities: string[];
  status: 'online' | 'busy' | 'offline';
  socketId?: string;
  currentTask?: string;
  lastSeen: Date;
  performanceScore: number;
  tasksCompleted: number;
  averageTaskTime: number;
}

export interface AgentCapability {
  name: string;
  proficiency: number; // 0-1 score
}

export class AgentRegistry extends EventEmitter {
  private agents: Map<string, Agent> = new Map();
  private socketToAgent: Map<string, string> = new Map();

  async registerAgent(agentId: string, agentData: Partial<Agent>): Promise<Agent> {
    const existingAgent = this.agents.get(agentId);

    const agent: Agent = {
      id: agentId,
      name: agentData.name || agentId,
      capabilities: agentData.capabilities || [],
      status: 'online',
      socketId: agentData.socketId,
      currentTask: undefined,
      lastSeen: new Date(),
      performanceScore: existingAgent?.performanceScore || 1.0,
      tasksCompleted: existingAgent?.tasksCompleted || 0,
      averageTaskTime: existingAgent?.averageTaskTime || 0,
      ...agentData
    };

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

    if (status === 'offline') {
      if (agent.socketId) {
        this.socketToAgent.delete(agent.socketId);
      }
      agent.socketId = undefined;
    }

    this.emit('agent_status_updated', agent);
  }

  getAgent(agentId: string): Agent | undefined {
    return this.agents.get(agentId);
  }

  getAgentBySocketId(socketId: string): Agent | undefined {
    const agentId = this.socketToAgent.get(socketId);
    return agentId ? this.agents.get(agentId) : undefined;
  }

  getAllAgents(): Agent[] {
    return Array.from(this.agents.values());
  }

  getAvailableAgents(): Agent[] {
    return this.getAllAgents().filter(agent =>
      agent.status === 'online' && !agent.currentTask
    );
  }

  getAgentsByCapability(capability: string): Agent[] {
    return this.getAllAgents().filter(agent =>
      agent.capabilities.includes(capability) && agent.status !== 'offline'
    );
  }

  getOptimalAgent(requiredCapabilities: string[]): Agent | null {
    const availableAgents = this.getAvailableAgents();
    console.log(`🔍 Finding optimal agent from ${availableAgents.length} available agents`);
    availableAgents.forEach(agent => {
      console.log(`   Agent: ${agent.id} (${agent.name}) - Capabilities: ${agent.capabilities.join(', ')}`);
    });

    if (availableAgents.length === 0) {
      return null;
    }

    // Score agents based on capability match and performance
    const scoredAgents = availableAgents.map(agent => {
      const capabilityScore = this.calculateCapabilityMatch(agent, requiredCapabilities);
      const performanceScore = agent.performanceScore;
      const workloadScore = agent.status === 'online' ? 1.0 : 0.5;

      const totalScore = (
        capabilityScore * 0.6 +
        performanceScore * 0.3 +
        workloadScore * 0.1
      );

      console.log(`   ${agent.id}: capability=${capabilityScore}, performance=${performanceScore}, workload=${workloadScore}, total=${totalScore.toFixed(2)}`);
      return { agent, score: totalScore };
    });

    // Sort by score (highest first)
    scoredAgents.sort((a, b) => b.score - a.score);

    const bestAgent = scoredAgents[0]?.agent;

    if (bestAgent) {
      logger.info(`Optimal agent selected: ${bestAgent.id} (score: ${scoredAgents[0].score.toFixed(2)})`);
    }

    return bestAgent || null;
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

  getOnlineAgentCount(): number {
    return this.getAllAgents().filter(agent => agent.status !== 'offline').length;
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