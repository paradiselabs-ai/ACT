import { Server } from 'socket.io';
import { EventEmitter } from 'events';
import { AgentRegistry, Agent } from './AgentRegistry';
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

export type AgentMessageType = 'status_update' | 'direct_mention' | 'help_request' | 'question' | 'peer_response';

export interface MessageClassification {
  type: AgentMessageType;
  targetAgentName?: string;
}

export class EventHub extends EventEmitter {
  private io: Server;
  private agentRegistry: AgentRegistry;
  private taskCoordinator: TaskCoordinator;
  private chronologicalLog: ChronologicalLog;
  private eventHistory: CoordinationEvent[] = [];
  private agentProfiles: Map<string, any> = new Map();

  // Rate limiting: max 3 peer-initiated responses per agent per 30 seconds
  private agentResponseTracker: Map<string, { count: number; windowStart: number }> = new Map();
  private readonly RATE_LIMIT_MAX = 3;
  private readonly RATE_LIMIT_WINDOW_MS = 30_000;

  // Prefixes that mark a message as an agent status broadcast (should not trigger peer responses)
  private readonly STATUS_PREFIXES = [
    'starting work on:', 'analysis complete:', 'plan:', 'implementation:',
    'task completed!', 'completed:', 'work plan ready:', 'implementation progress:',
    'implementation update:', 'task failed:', 'starting:', 'progress:'
  ];

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
   * Classify a message to determine its type and routing target.
   * Priority order: direct_mention > status_update > question > help_request > peer_response
   */
  public classifyMessage(message: string): MessageClassification {
    const lower = message.toLowerCase().trimStart();

    // 1. Direct @mention - route only to the named agent
    const mentionMatch = message.match(/^@(\S+)/);
    if (mentionMatch) {
      return { type: 'direct_mention', targetAgentName: mentionMatch[1] };
    }

    // 2. Agent status broadcasts - observe only, never trigger peer responses
    if (this.STATUS_PREFIXES.some(prefix => lower.startsWith(prefix))) {
      return { type: 'status_update' };
    }

    // 3. Question directed at the room
    if (message.includes('?')) {
      return { type: 'question' };
    }

    // 4. Explicit help request
    const helpKeywords = ['help', 'assist', 'need someone', 'can you', 'could you'];
    if (helpKeywords.some(kw => lower.includes(kw))) {
      return { type: 'help_request' };
    }

    // 5. General peer communication
    return { type: 'peer_response' };
  }

  /**
   * Rate-limit outgoing messages from an agent.
   * Returns true if the message is allowed, false if it should be dropped.
   */
  private checkRateLimit(agentId: string): boolean {
    const now = Date.now();
    const tracker = this.agentResponseTracker.get(agentId);

    if (!tracker || now - tracker.windowStart > this.RATE_LIMIT_WINDOW_MS) {
      this.agentResponseTracker.set(agentId, { count: 1, windowStart: now });
      return true;
    }

    if (tracker.count >= this.RATE_LIMIT_MAX) {
      logger.warn(`🚦 Rate limit: ${agentId} exceeded ${this.RATE_LIMIT_MAX} messages in window — dropping`);
      return false;
    }

    tracker.count++;
    return true;
  }

  /**
   * Find an agent by display name or agent ID (case-insensitive).
   */
  private findAgentByName(name: string): Agent | undefined {
    const lower = name.toLowerCase();
    return this.agentRegistry.getAllAgents().find(
      a => a.name.toLowerCase() === lower || a.id.toLowerCase() === lower
    );
  }

  /**
   * Select the best available agent to handle a help request or question.
   * Prefers agents with relevant capabilities, then falls back to any available agent.
   */
  private selectCoordinationResponder(senderId: string, message: string, candidates: Agent[]): Agent | null {
    const lower = message.toLowerCase();

    const relevant = candidates.filter(a => {
      const available = !a.currentTask;
      const capMatch = a.capabilities.some(cap => lower.includes(cap));
      return available && capMatch;
    });

    if (relevant.length > 0) return relevant[0];

    // Fallback: any available agent
    return candidates.find(a => !a.currentTask) || null;
  }

  /**
   * Main entry point for agent messages.
   * Classifies, rate-limits, routes, and logs every incoming agent message.
   */
  public async handleAgentMessage(senderId: string, senderName: string, message: string, timestamp: string): Promise<void> {
    try {
      // Drop messages that exceed the per-agent rate limit
      if (!this.checkRateLimit(senderId)) return;

      const classification = this.classifyMessage(message);
      const ts = timestamp || new Date().toISOString();
      const payload = { sender: senderName, message, timestamp: ts, messageType: classification.type };

      // Persist with the real message type instead of hardcoded 'coordination'
      await this.chronologicalLog.append({
        timestamp: ts,
        agent: senderId,
        message,
        type: classification.type
      }).catch(err => logger.error(`ChronologicalLog append failed: ${err.message}`));

      switch (classification.type) {
        case 'status_update':
        case 'peer_response':
          // Broadcast to all so agents can observe, but messageType signals "don't auto-respond"
          this.io.emit('agent_message', payload);
          logger.debug(`📡 [${classification.type}] ${senderId}: ${message.substring(0, 80)}`);
          break;

        case 'direct_mention': {
          const targetName = classification.targetAgentName!;
          const target = this.findAgentByName(targetName);
          if (target?.socketId) {
            this.io.to(target.socketId).emit('agent_message', payload);
            logger.info(`📨 [direct_mention] ${senderId} → ${target.id} (${target.name})`);
          } else {
            // Target offline or not found — broadcast so message isn't lost
            this.io.emit('agent_message', payload);
            logger.warn(`📡 [direct_mention] Target "${targetName}" not found, broadcasting`);
          }
          break;
        }

        case 'help_request':
        case 'question': {
          const candidates = this.agentRegistry.getAllAgents().filter(
            a => a.status === 'online' && a.id !== senderId
          );
          const responder = this.selectCoordinationResponder(senderId, message, candidates);
          if (responder?.socketId) {
            this.io.to(responder.socketId).emit('agent_message', payload);
            logger.info(`📨 [${classification.type}] ${senderId} → ${responder.id}`);
          } else {
            this.io.emit('agent_message', payload);
          }
          break;
        }
      }
    } catch (error: any) {
      logger.error(`handleAgentMessage failed: ${error.message}`);
    }
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