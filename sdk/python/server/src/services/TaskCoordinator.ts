import { EventEmitter } from 'events';
import { v4 as uuidv4 } from 'uuid';
import { AgentRegistry, Agent } from './AgentRegistry';
import { logger } from '../utils/logger';

export interface Task {
  id: string;
  description: string;
  requiredCapabilities: string[];
  priority: 'low' | 'medium' | 'high' | 'critical';
  status: 'pending' | 'assigned' | 'in_progress' | 'completed' | 'failed';
  assignedAgent?: string;
  dependencies: string[];
  progress: number;
  estimatedDuration?: number; // in minutes
  createdAt: Date;
  startedAt?: Date;
  completedAt?: Date;
  metadata?: Record<string, any>;
}

export interface TaskAssignment {
  taskId: string;
  agentId: string;
  assignedAt: Date;
  reason: string;
}

export interface ConflictDetection {
  type: 'resource_contention' | 'dependency_deadlock' | 'capability_mismatch';
  involvedTasks: string[];
  involvedAgents: string[];
  severity: 'low' | 'medium' | 'high';
  suggestedResolution: string;
}

export class TaskCoordinator extends EventEmitter {
  private tasks: Map<string, Task> = new Map();
  private assignments: Map<string, TaskAssignment> = new Map();
  private agentRegistry: AgentRegistry;

  constructor(agentRegistry: AgentRegistry) {
    super();
    this.agentRegistry = agentRegistry;
  }

  async createTask(taskData: Partial<Task>): Promise<Task> {
    const task: Task = {
      id: uuidv4(),
      description: taskData.description || 'Untitled task',
      requiredCapabilities: taskData.requiredCapabilities || [],
      priority: taskData.priority || 'medium',
      status: 'pending',
      dependencies: taskData.dependencies || [],
      progress: 0,
      estimatedDuration: taskData.estimatedDuration,
      createdAt: new Date(),
      metadata: taskData.metadata || {},
      ...taskData
    };

    this.tasks.set(task.id, task);
    this.emit('task_created', task);

    logger.info(`Task created: ${task.id} - ${task.description}`);

    return task;
  }

  async assignOptimalAgent(taskId: string): Promise<TaskAssignment | null> {
    const task = this.tasks.get(taskId);
    if (!task) {
      throw new Error(`Task ${taskId} not found`);
    }

    if (task.status !== 'pending') {
      throw new Error(`Task ${taskId} is not pending (status: ${task.status})`);
    }

    // Check dependencies
    const uncompletedDependencies = await this.checkDependencies(task);
    if (uncompletedDependencies.length > 0) {
      logger.info(`Task ${taskId} waiting for dependencies: ${uncompletedDependencies.join(', ')}`);
      return null;
    }

    // Find optimal agent
    const optimalAgent = this.agentRegistry.getOptimalAgent(task.requiredCapabilities);
    if (!optimalAgent) {
      logger.warn(`No suitable agent found for task ${taskId}`);
      return null;
    }

    // Create assignment
    const assignment: TaskAssignment = {
      taskId: task.id,
      agentId: optimalAgent.id,
      assignedAt: new Date(),
      reason: `Optimal match for capabilities: ${task.requiredCapabilities.join(', ')}`
    };

    // Update task and agent
    task.status = 'assigned';
    task.assignedAgent = optimalAgent.id;

    await this.agentRegistry.updateAgentStatus(optimalAgent.id, 'busy', task.id);

    this.assignments.set(task.id, assignment);
    this.emit('task_assigned', { task, assignment });

    logger.info(`Task ${taskId} assigned to agent ${optimalAgent.id}`);

    return assignment;
  }

  async updateTaskProgress(taskId: string, update: {
    progress?: number;
    status?: Task['status'];
    message?: string;
  }): Promise<void> {
    const task = this.tasks.get(taskId);
    if (!task) {
      throw new Error(`Task ${taskId} not found`);
    }

    const previousStatus = task.status;

    if (update.progress !== undefined) {
      task.progress = Math.max(0, Math.min(100, update.progress));
    }

    if (update.status) {
      task.status = update.status;

      // Handle status changes
      switch (update.status) {
        case 'in_progress':
          if (!task.startedAt) {
            task.startedAt = new Date();
          }
          break;

        case 'completed':
          task.completedAt = new Date();
          task.progress = 100;
          await this.handleTaskCompletion(task);
          break;

        case 'failed':
          task.completedAt = new Date();
          await this.handleTaskFailure(task);
          break;
      }
    }

    this.emit('task_progress_updated', { task, update });

    // Trigger dependency checking if task completed
    if (previousStatus !== 'completed' && task.status === 'completed') {
      await this.processPendingTasks();
    }
  }

  private async handleTaskCompletion(task: Task): Promise<void> {
    if (task.assignedAgent) {
      const duration = task.startedAt && task.completedAt
        ? task.completedAt.getTime() - task.startedAt.getTime()
        : 0;

      await this.agentRegistry.updateAgentPerformance(task.assignedAgent, duration, true);
      await this.agentRegistry.updateAgentStatus(task.assignedAgent, 'online');
    }

    logger.info(`Task completed: ${task.id} - ${task.description}`);
  }

  private async handleTaskFailure(task: Task): Promise<void> {
    if (task.assignedAgent) {
      await this.agentRegistry.updateAgentPerformance(task.assignedAgent, 0, false);
      await this.agentRegistry.updateAgentStatus(task.assignedAgent, 'online');
    }

    logger.warn(`Task failed: ${task.id} - ${task.description}`);
  }

  private async checkDependencies(task: Task): Promise<string[]> {
    const uncompletedDependencies: string[] = [];

    for (const depId of task.dependencies) {
      const depTask = this.tasks.get(depId);
      if (!depTask || depTask.status !== 'completed') {
        uncompletedDependencies.push(depId);
      }
    }

    return uncompletedDependencies;
  }

  private async processPendingTasks(): Promise<void> {
    const pendingTasks = Array.from(this.tasks.values()).filter(task => task.status === 'pending');

    for (const task of pendingTasks) {
      try {
        await this.assignOptimalAgent(task.id);
      } catch (error: any) {
        logger.error(`Failed to assign task ${task.id}: ${error.message}`);
      }
    }
  }

  async detectConflicts(): Promise<ConflictDetection[]> {
    const conflicts: ConflictDetection[] = [];

    // Resource contention detection
    const busyAgents = this.agentRegistry.getAllAgents().filter(agent => agent.status === 'busy');
    for (const agent of busyAgents) {
      const agentTasks = Array.from(this.tasks.values()).filter(task => task.assignedAgent === agent.id);
      if (agentTasks.length > 1) {
        conflicts.push({
          type: 'resource_contention',
          involvedTasks: agentTasks.map(t => t.id),
          involvedAgents: [agent.id],
          severity: 'medium',
          suggestedResolution: `Redistribute tasks from agent ${agent.id}`
        });
      }
    }

    // Dependency deadlock detection
    const dependencyGraph = this.buildDependencyGraph();
    const cycles = this.detectCycles(dependencyGraph);
    for (const cycle of cycles) {
      conflicts.push({
        type: 'dependency_deadlock',
        involvedTasks: cycle,
        involvedAgents: [],
        severity: 'high',
        suggestedResolution: 'Restructure task dependencies to break cycle'
      });
    }

    // Capability mismatch detection
    const assignedTasks = Array.from(this.tasks.values()).filter(task => task.status === 'assigned' || task.status === 'in_progress');
    for (const task of assignedTasks) {
      if (task.assignedAgent) {
        const agent = this.agentRegistry.getAgent(task.assignedAgent);
        if (agent) {
          const missingCapabilities = task.requiredCapabilities.filter(cap => !agent.capabilities.includes(cap));
          if (missingCapabilities.length > 0) {
            conflicts.push({
              type: 'capability_mismatch',
              involvedTasks: [task.id],
              involvedAgents: [agent.id],
              severity: 'low',
              suggestedResolution: `Agent ${agent.id} missing capabilities: ${missingCapabilities.join(', ')}`
            });
          }
        }
      }
    }

    if (conflicts.length > 0) {
      this.emit('conflicts_detected', conflicts);
    }

    return conflicts;
  }

  private buildDependencyGraph(): Map<string, string[]> {
    const graph = new Map<string, string[]>();

    for (const task of this.tasks.values()) {
      graph.set(task.id, task.dependencies);
    }

    return graph;
  }

  private detectCycles(graph: Map<string, string[]>): string[][] {
    const visited = new Set<string>();
    const recursionStack = new Set<string>();
    const cycles: string[][] = [];

    const dfs = (node: string, path: string[]): void => {
      visited.add(node);
      recursionStack.add(node);
      path.push(node);

      const dependencies = graph.get(node) || [];
      for (const dep of dependencies) {
        if (!visited.has(dep)) {
          dfs(dep, [...path]);
        } else if (recursionStack.has(dep)) {
          const cycleStart = path.indexOf(dep);
          cycles.push(path.slice(cycleStart));
        }
      }

      recursionStack.delete(node);
    };

    for (const node of graph.keys()) {
      if (!visited.has(node)) {
        dfs(node, []);
      }
    }

    return cycles;
  }

  getTask(taskId: string): Task | undefined {
    return this.tasks.get(taskId);
  }

  getAllTasks(): Task[] {
    return Array.from(this.tasks.values());
  }

  getTasksByStatus(status: Task['status']): Task[] {
    return this.getAllTasks().filter(task => task.status === status);
  }

  getTasksByAgent(agentId: string): Task[] {
    return this.getAllTasks().filter(task => task.assignedAgent === agentId);
  }

  getTaskCount(): number {
    return this.tasks.size;
  }

  // Demo helper method
  async createDemoProject(): Promise<Task[]> {
    const tasks: Task[] = [];

    // Create a "Build Todo App" project
    const backendTask = await this.createTask({
      description: 'Create Flask API for todo app',
      requiredCapabilities: ['python', 'backend', 'api'],
      priority: 'high',
      estimatedDuration: 30
    });

    const frontendTask = await this.createTask({
      description: 'Create React frontend for todo app',
      requiredCapabilities: ['react', 'frontend', 'javascript'],
      priority: 'high',
      dependencies: [backendTask.id],
      estimatedDuration: 45
    });

    const testingTask = await this.createTask({
      description: 'Test todo app functionality',
      requiredCapabilities: ['testing', 'qa'],
      priority: 'medium',
      dependencies: [frontendTask.id],
      estimatedDuration: 20
    });

    tasks.push(backendTask, frontendTask, testingTask);

    logger.info('Demo project created with 3 coordinated tasks');
    return tasks;
  }
}

export default TaskCoordinator;