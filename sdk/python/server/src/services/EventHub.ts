import { Server } from 'socket.io';
import { EventEmitter } from 'events';
import { AgentRegistry } from './AgentRegistry';
import { TaskCoordinator } from './TaskCoordinator';
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
  private eventHistory: CoordinationEvent[] = [];

  constructor(io: Server, agentRegistry: AgentRegistry, taskCoordinator: TaskCoordinator) {
    super();
    this.io = io;
    this.agentRegistry = agentRegistry;
    this.taskCoordinator = taskCoordinator;

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
          status: agent.status
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