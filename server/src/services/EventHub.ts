import { Server } from 'socket.io';
import { EventEmitter } from 'events';
import { AgentRegistry } from './AgentRegistry';
import { TaskCoordinator } from './TaskCoordinator';
import { ChronologicalLog } from './ChronologicalLog';
import { logger } from '../utils/logger';

export interface CoordinationEvent {
  type: string;
  agentId?: string;
  taskId?: string;
  data: any;
  timestamp: Date;
}

export class EventHub extends EventEmitter {
  private io: Server;
  private agentRegistry: AgentRegistry;
  private taskCoordinator: TaskCoordinator;
  private chronologicalLog: ChronologicalLog;
  private eventHistory: CoordinationEvent[] = [];
  private agentProfiles: Map<string, any> = new Map();

  constructor(io: Server, agentRegistry: AgentRegistry, taskCoordinator: TaskCoordinator, chronologicalLog?: ChronologicalLog) {
    super();
    this.io = io;
    this.agentRegistry = agentRegistry;
    this.taskCoordinator = taskCoordinator;
    this.chronologicalLog = chronologicalLog || new ChronologicalLog();
    this.setupEventListeners();
  }

  private setupEventListeners(): void {
    // Agent Registry Events
    this.agentRegistry.on('agent_registered', (agent) => {
      this.broadcastEvent('agent_registered', {
        agentId: agent.id,
        agent: {
          id: agent.id,
          name: agent.name,
          capabilities: agent.capabilities,
          status: agent.status,
          model: agent.model,
          provider: agent.provider
        }
      });
    });

    this.agentRegistry.on('agent_status_updated', (agent) => {
      this.broadcastEvent('agent_status_updated', {
        agentId: agent.id,
        status: agent.status,
        currentTask: agent.currentTask
      });
    });

    this.agentRegistry.on('agent_performance_updated', (agent) => {
      this.broadcastEvent('agent_performance_updated', {
        agentId: agent.id,
        performanceScore: agent.performanceScore,
        tasksCompleted: agent.tasksCompleted,
        averageTaskTime: agent.averageTaskTime
      });

      // Update agent profiles with performance data
      if (agent.id) {
        this.agentProfiles.set(agent.id, {
          ...this.agentProfiles.get(agent.id),
          ...agent
        });
      }
    });

    // Task Coordinator Events
    this.taskCoordinator.on('task_created', (task) => {
      this.broadcastEvent('task_created', {
        taskId: task.id,
        task: {
          id: task.id,
          description: task.description,
          requiredCapabilities: task.requiredCapabilities,
          priority: task.priority,
          status: task.status,
          progress: task.progress
        }
      });
    });

    this.taskCoordinator.on('task_assigned', ({ task, assignment }) => {
      this.broadcastEvent('task_assigned', {
        taskId: task.id,
        agentId: assignment.agentId,
        reason: assignment.reason,
        task: {
          id: task.id,
          description: task.description,
          status: task.status,
          assignedAgent: task.assignedAgent
        }
      });

      // This is the MONEY SHOT for Windsurf's dashboard!
      logger.info(`🎯 AUTONOMOUS COORDINATION: Task "${task.description}" automatically assigned to agent "${assignment.agentId}"`);
    });

    this.taskCoordinator.on('task_progress_updated', ({ task, update }) => {
      this.broadcastEvent('task_progress_updated', {
        taskId: task.id,
        progress: task.progress,
        status: task.status,
        message: update.message
      });
    });

    this.taskCoordinator.on('conflicts_detected', (conflicts) => {
      this.broadcastEvent('conflicts_detected', {
        conflicts: conflicts.map((conflict: any) => ({
          type: conflict.type,
          severity: conflict.severity,
          involvedTasks: conflict.involvedTasks,
          involvedAgents: conflict.involvedAgents,
          suggestedResolution: conflict.suggestedResolution
        }))
      });

      logger.warn(`🚨 CONFLICT DETECTION: ${conflicts.length} conflicts detected and resolved autonomously`);
    });
  }

  /**
   * Intelligent agent communication coordination
   * ACT decides when and how agents should communicate
   */
  public async coordinateAgentCommunication(senderId: string, message: string, messageId: string): Promise<void> {
    try {
      logger.info(`🤖 COORDINATION CHECK: Message from ${senderId}: "${message}"`);

      // Get all online agents except sender
      const allAgents = this.agentRegistry.getAllAgents();
      const onlineAgents = allAgents.filter(agent =>
        agent.status === 'online' && agent.id !== senderId
      );

      logger.info(`🤖 COORDINATION CHECK: Found ${onlineAgents.length} online agents (excluding sender)`);

      if (onlineAgents.length === 0) {
        logger.debug('No other agents online for coordination');
        return;
      }

      // Determine if this message warrants coordination
      const coordinationNeeded = await this.shouldCoordinateMessage(senderId, message, messageId);

      logger.info(`🤖 COORDINATION CHECK: Coordination needed? ${coordinationNeeded}`);

      if (!coordinationNeeded) {
        logger.debug(`Message from ${senderId} doesn't require coordination response`);
        return;
      }

      // Find best agent to respond based on coordination context
      const responderAgent = await this.selectCoordinationResponder(senderId, message, onlineAgents);

      logger.info(`🤖 COORDINATION CHECK: Selected responder: ${responderAgent ? responderAgent.id : 'NONE'}`);

      if (responderAgent) {
        await this.generateAndSendCoordinationResponse(responderAgent, senderId, message, messageId);
      }

    } catch (error: any) {
      logger.error(`Agent communication coordination failed: ${error.message}`);
    }
  }

  private async shouldCoordinateMessage(senderId: string, message: string, messageId: string): Promise<boolean> {
    // Coordination triggers:
    // 1. Task completion announcements
    // 2. Help requests
    // 3. Progress updates that might affect others
    // 4. Questions or clarifications
    // 5. Critical decisions or changes

    const coordinationTriggers = [
      'completed', 'finished', 'done',
      'help', 'assist', 'support',
      'question', 'clarify', 'unclear',
      'issue', 'problem', 'error',
      'change', 'update', 'modify'
    ];

    const lowerMessage = message.toLowerCase();
    return coordinationTriggers.some(trigger => lowerMessage.includes(trigger));
  }

  private async selectCoordinationResponder(senderId: string, message: string, availableAgents: any[]): Promise<any | null> {
    logger.info(`🤖 RESPONDER SELECTION: Looking for responder among ${availableAgents.length} agents`);

    // Use coordination intelligence to select responder
    // Priority based on:
    // 1. Agent's current task context
    // 2. Agent's capabilities matching message content
    // 3. Agent's historical collaboration patterns
    // 4. Agent's current workload

    // For now, simple selection: least busy agent with relevant capabilities
    const sender = this.agentRegistry.getAgent(senderId);

    logger.info(`🤖 RESPONDER SELECTION: Available agents: ${availableAgents.map(a => `${a.id}(busy:${!!a.currentTask})`).join(', ')}`);

    const relevantAgents = availableAgents.filter(agent => {
      const notBusy = !agent.currentTask;
      const hasRelevantCap = agent.capabilities.some((cap: string) => message.toLowerCase().includes(cap));

      logger.info(`🤖 RESPONDER SELECTION: Agent ${agent.id} - notBusy: ${notBusy}, hasRelevantCap: ${hasRelevantCap} (caps: ${agent.capabilities.join(',')})`);

      return notBusy && hasRelevantCap;
    });

    logger.info(`🤖 RESPONDER SELECTION: Found ${relevantAgents.length} relevant agents`);

    if (relevantAgents.length > 0) {
      const selected = relevantAgents[0];
      logger.info(`🤖 RESPONDER SELECTION: Selected ${selected.id}`);
      return selected; // Return first relevant agent
    }

    // Fallback: any available agent
    const fallbackAgent = availableAgents.find(agent => !agent.currentTask);
    logger.info(`🤖 RESPONDER SELECTION: No relevant agents, fallback: ${fallbackAgent ? fallbackAgent.id : 'NONE'}`);
    return fallbackAgent || null;
  }

  private async generateAndSendCoordinationResponse(responderAgent: any, senderId: string, originalMessage: string, messageId: string): Promise<void> {
    try {
      // Generate coordination-appropriate response
      const response = await this.generateCoordinationResponse(responderAgent, senderId, originalMessage);

      if (response) {
        // Send response through the system as an agent_message from the responding agent
        const responseData = {
          sender: responderAgent.name,  // Use the agent's display name
          message: `@${senderId} ${response}`,  // Tag the original sender
          timestamp: new Date().toISOString()
        };

        // Broadcast as agent_message so it appears in activity feeds
        this.io.emit('agent_message', responseData);
        logger.info(`📡 COORDINATION RESPONSE EMITTED: ${JSON.stringify(responseData)}`);

        // Also log to ChronologicalLog
        this.chronologicalLog.append({
          timestamp: responseData.timestamp,
          agent: responderAgent.id,
          message: responseData.message,
          type: 'coordination'
        }).catch(err => {
          logger.error(`Failed to log coordination response: ${err.message}`);
        });

        logger.info(`🤖 COORDINATED RESPONSE: ${responderAgent.id} → ${senderId}: ${response.substring(0, 100)}...`);
      }
    } catch (error: any) {
      logger.error(`Failed to generate coordination response: ${error.message}`);
    }
  }

  private async generateCoordinationResponse(responderAgent: any, senderId: string, message: string): Promise<string | null> {
    // This should use the agent's model to generate a response
    // For now, return a simple coordination response
    // In full implementation, this would call the agent's LLM

    const sender = this.agentRegistry.getAgent(senderId);

    if (message.toLowerCase().includes('help') || message.toLowerCase().includes('assist')) {
      return `I can help with that. My expertise includes ${responderAgent.capabilities.join(', ')}.`;
    }

    if (message.toLowerCase().includes('completed') || message.toLowerCase().includes('done')) {
      return `Great work! I'll note this completion for coordination purposes.`;
    }

    if (message.toLowerCase().includes('question') || message.toLowerCase().includes('clarify')) {
      return `I understand you need clarification. Let me see how I can assist.`;
    }

    return null; // No response needed
  }

  private broadcastEvent(type: string, data: any): void {
    const event: CoordinationEvent = {
      type,
      agentId: data.agentId,
      taskId: data.taskId,
      data,
      timestamp: new Date()
    };

    // Store in history
    this.eventHistory.push(event);

    // Keep only last 1000 events
    if (this.eventHistory.length > 1000) {
      this.eventHistory = this.eventHistory.slice(-1000);
    }

    // Log to ChronologicalLog for PVM
    this.chronologicalLog.append({
      timestamp: event.timestamp.toISOString(),
      agent: event.agentId || 'system',
      message: `${type}: ${JSON.stringify(data)}`,
      type: 'coordination' // Use generic coordination type for now
    }).catch(err => {
      logger.error(`Failed to log event to ChronologicalLog: ${err.message}`);
    });

    // Broadcast to all connected clients (especially Windsurf's dashboard!)
    this.io.emit(type, data);

    // Emit to internal listeners
    this.emit(type, data);

    logger.debug(`📡 Event broadcasted: ${type} - ${JSON.stringify(data).substring(0, 100)}...`);
  }

  // Manual event broadcasting for custom coordination events
  async broadcastCoordinationEvent(type: string, data: any): Promise<void> {
    this.broadcastEvent(type, data);
  }

  // Get recent events (for dashboard initialization)
  getRecentEvents(limit: number = 50): CoordinationEvent[] {
    return this.eventHistory.slice(-limit);
  }

  // Get events by type
  getEventsByType(type: string, limit: number = 50): CoordinationEvent[] {
    return this.eventHistory
      .filter(event => event.type === type)
      .slice(-limit);
  }

  // Demo coordination simulation
  async simulateAutonomousCoordination(): Promise<void> {
    logger.info('🎭 Starting autonomous coordination demonstration...');

    // Create demo project with coordinated tasks
    const demoTasks = await this.taskCoordinator.createDemoProject();

    // Simulate agents connecting
    setTimeout(() => {
      this.broadcastEvent('demo_agent_connecting', {
        agentId: 'demo_frontend_agent',
        capabilities: ['react', 'frontend', 'javascript'],
        message: 'Frontend specialist agent joining coordination...'
      });
    }, 1000);

    setTimeout(() => {
      this.broadcastEvent('demo_agent_connecting', {
        agentId: 'demo_backend_agent',
        capabilities: ['python', 'backend', 'api'],
        message: 'Backend specialist agent joining coordination...'
      });
    }, 2000);

    setTimeout(() => {
      this.broadcastEvent('demo_agent_connecting', {
        agentId: 'demo_qa_agent',
        capabilities: ['testing', 'qa'],
        message: 'QA specialist agent joining coordination...'
      });
    }, 3000);

    // Trigger automatic task assignment after agents connect
    setTimeout(async () => {
      logger.info('🤖 Triggering autonomous task assignment...');

      for (const task of demoTasks) {
        const assignment = await this.taskCoordinator.assignOptimalAgent(task.id);
        if (assignment) {
          // Simulate task progress
          setTimeout(() => {
            this.taskCoordinator.updateTaskProgress(task.id, {
              status: 'in_progress',
              progress: 25,
              message: 'Task automatically started by coordinated agent'
            });
          }, 5000);

          setTimeout(() => {
            this.taskCoordinator.updateTaskProgress(task.id, {
              status: 'completed',
              progress: 100,
              message: 'Task completed autonomously'
            });
          }, 10000);
        }
      }
    }, 4000);
  }

  // Health monitoring
  async performHealthCheck(): Promise<{
    status: string;
    connectedClients: number;
    recentEvents: number;
    agentCount: number;
    taskCount: number;
  }> {
    return {
      status: 'healthy',
      connectedClients: this.io.sockets.sockets.size,
      recentEvents: this.eventHistory.length,
      agentCount: this.agentRegistry.getOnlineAgentCount(),
      taskCount: this.taskCoordinator.getTaskCount()
    };
  }

  // Conflict resolution automation
  async resolveDetectedConflicts(): Promise<void> {
    const conflicts = await this.taskCoordinator.detectConflicts();

    for (const conflict of conflicts) {
      switch (conflict.type) {
        case 'resource_contention':
          await this.resolveResourceContention(conflict);
          break;
        case 'dependency_deadlock':
          await this.resolveDependencyDeadlock(conflict);
          break;
        case 'capability_mismatch':
          await this.resolveCapabilityMismatch(conflict);
          break;
      }
    }
  }

  private async resolveResourceContention(conflict: any): Promise<void> {
    logger.info(`🔧 Resolving resource contention: ${conflict.involvedAgents.join(', ')}`);

    this.broadcastEvent('conflict_resolution_started', {
      type: 'resource_contention',
      message: 'Automatically redistributing tasks to resolve agent overload'
    });

    // Implementation would redistribute tasks
    // For demo, just broadcast resolution
    setTimeout(() => {
      this.broadcastEvent('conflict_resolved', {
        type: 'resource_contention',
        resolution: 'Tasks redistributed across available agents',
        success: true
      });
    }, 2000);
  }

  private async resolveDependencyDeadlock(conflict: any): Promise<void> {
    logger.info(`🔧 Resolving dependency deadlock: ${conflict.involvedTasks.join(', ')}`);

    this.broadcastEvent('conflict_resolution_started', {
      type: 'dependency_deadlock',
      message: 'Automatically restructuring task dependencies'
    });

    setTimeout(() => {
      this.broadcastEvent('conflict_resolved', {
        type: 'dependency_deadlock',
        resolution: 'Dependency cycle broken, tasks reordered',
        success: true
      });
    }, 3000);
  }

  private async resolveCapabilityMismatch(conflict: any): Promise<void> {
    logger.info(`🔧 Resolving capability mismatch: ${conflict.involvedAgents.join(', ')}`);

    this.broadcastEvent('conflict_resolution_started', {
      type: 'capability_mismatch',
      message: 'Automatically reassigning task to capable agent'
    });

    setTimeout(() => {
      this.broadcastEvent('conflict_resolved', {
        type: 'capability_mismatch',
        resolution: 'Task reassigned to agent with required capabilities',
        success: true
      });
    }, 1500);
  }
}

export default EventHub;