import { EventEmitter } from 'events';
import { v4 as uuidv4 } from 'uuid';
import { AgentRegistry, Agent } from './AgentRegistry';
import { PVMIndexer } from './PVMIndexer';
import { logger } from '../utils/logger';

export interface Task {
  id: string;
  title?: string;       // short display name
  description: string;
  requiredCapabilities: string[];
  priority: 'low' | 'medium' | 'high' | 'critical';
  status: 'pending' | 'assigned' | 'in_progress' | 'completed' | 'failed' | 'submitted_for_validation' | 'validated';
  assignedAgent?: string;
  dependencies: string[];
  progress: number;
  estimatedDuration?: number; // in minutes
  createdAt: Date;
  startedAt?: Date;
  completedAt?: Date;
  metadata?: Record<string, any>;
  retryCount: number; // how many times this task has been retried after failure
}

export const MAX_TASK_RETRIES = 3;

// Status transitions the state machine refuses once a task has reached a
// success-side status. Sources: `completed` (the race target — runner's
// post-completion broadcast exit 1 must not flip it to `failed`), `validated`
// (assurance-accepted, nothing downgrades it), and `failed` (runner retry goes
// through a separate reset path, not updateTaskProgress). Transitions listed
// here are rejected; everything else passes through unchanged.
const FORBIDDEN_TRANSITIONS: ReadonlyMap<Task['status'], ReadonlySet<Task['status']>> = new Map([
  ['completed', new Set<Task['status']>(['failed', 'pending', 'assigned', 'in_progress'])],
  ['validated', new Set<Task['status']>(['failed', 'pending', 'assigned', 'in_progress', 'completed', 'submitted_for_validation'])],
  ['failed', new Set<Task['status']>(['completed', 'validated', 'submitted_for_validation'])],
]);

export class TerminalStateTransitionError extends Error {
  readonly code = 'TERMINAL_STATE_TRANSITION';
  readonly taskId: string;
  readonly fromStatus: Task['status'];
  readonly toStatus: Task['status'];

  constructor(taskId: string, fromStatus: Task['status'], toStatus: Task['status']) {
    super(`Task ${taskId} is in terminal state '${fromStatus}'; refusing transition to '${toStatus}'`);
    this.name = 'TerminalStateTransitionError';
    this.taskId = taskId;
    this.fromStatus = fromStatus;
    this.toStatus = toStatus;
  }
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
  private pvmIndexer?: PVMIndexer;

  constructor(agentRegistry: AgentRegistry, pvmIndexer?: PVMIndexer) {
    super();
    this.agentRegistry = agentRegistry;
    this.pvmIndexer = pvmIndexer;
  }

  async createTask(taskData: Partial<Task>): Promise<Task> {
    const task: Task = {
      id: uuidv4(),
      title: taskData.title,
      description: taskData.description || 'Untitled task',
      requiredCapabilities: taskData.requiredCapabilities || [],
      priority: taskData.priority || 'medium',
      status: 'pending',
      dependencies: taskData.dependencies || [],
      progress: 0,
      estimatedDuration: taskData.estimatedDuration,
      createdAt: new Date(),
      metadata: taskData.metadata || {},
      retryCount: 0
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

    // Find optimal agent. getOptimalAgent now enforces capability match when
    // requirements are specified — returns null if no agent has any matching
    // capability. This prevents wrong-role assignment (e.g. Go task to
    // frontend_dev). Tasks with empty requiredCapabilities still match any agent.
    // Scoped to the task's project so cross-project agents are invisible to the
    // assignment scorer.
    const taskProject = (task.metadata?.projectName as string | undefined) ?? undefined;
    let selectedAgent = this.agentRegistry.getOptimalAgent(task.requiredCapabilities, taskProject);
    let reason = selectedAgent
      ? `Optimal match for capabilities: ${task.requiredCapabilities.join(', ')}`
      : '';

    // PVM similarity fallback — only for tasks with required capabilities
    // where no agent currently matches. PVM may surface a recently-seen
    // pattern where another agent type handled similar work successfully.
    if (!selectedAgent && this.pvmIndexer && task.requiredCapabilities.length > 0) {
      const pvmAgentId = await this.pickAgentFromPVM(task);
      if (pvmAgentId) {
        const pvmAgent = this.agentRegistry.getAgent(pvmAgentId);
        // Only accept the PVM suggestion if (a) same project, and (b) has the
        // capability. PVM aggregates globally; selection must respect project.
        if (pvmAgent && (!taskProject || pvmAgent.projectName === taskProject)
            && task.requiredCapabilities.some(cap => pvmAgent.capabilities.includes(cap))) {
          selectedAgent = pvmAgent;
          reason = `PVM similarity match with capability overlap`;
        }
      }
    }

    // No best-effort fallback for capability-specified tasks. Leaving the task
    // pending is preferable to assigning it to a wrong-role agent that will
    // either fail immediately or produce garbage — both of which block the
    // validation pipeline. The task stays pending and will be re-evaluated
    // whenever an agent status changes (agent comes online, completes a task, etc).
    if (!selectedAgent) {
      logger.warn(`assign_decision: leaving_pending task=${taskId} required=[${task.requiredCapabilities.join(',')}] reason=no_capability_match`);
      return null;
    }

    // Create assignment
    const assignment: TaskAssignment = {
      taskId: task.id,
      agentId: selectedAgent.id,
      assignedAt: new Date(),
      reason: reason || 'Assigned'
    };

    // Update task and agent
    task.status = 'assigned';
    task.assignedAgent = selectedAgent.id;

    await this.agentRegistry.updateAgentStatus(selectedAgent.id, 'busy', task.id);

    this.assignments.set(task.id, assignment);
    this.emit('task_assigned', { task, assignment });

    logger.info(`Task ${taskId} assigned to agent ${selectedAgent.id} (${assignment.reason})`);

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

    // Why: KI-10 race — a late runner POST (e.g. post-completion broadcast exit 1)
    // could overwrite a completed task to 'failed' before the original success
    // write landed, flipping the state machine mid-cycle.
    if (update.status && update.status !== previousStatus) {
      const forbidden = FORBIDDEN_TRANSITIONS.get(previousStatus);
      if (forbidden && forbidden.has(update.status)) {
        logger.warn(`terminal_state_guard: rejected task=${taskId} from=${previousStatus} to=${update.status}`);
        throw new TerminalStateTransitionError(taskId, previousStatus, update.status);
      }
    }

    if (update.progress !== undefined) {
      task.progress = Math.max(0, Math.min(100, update.progress));
    }

    if (update.message !== undefined) {
      task.metadata = { ...(task.metadata || {}), result: update.message };
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

    const resultSnippet = typeof task.metadata?.result === 'string'
      ? (task.metadata.result as string).slice(0, 160)
      : '';
    logger.warn(`task_failure_handled task=${task.id} agent=${task.assignedAgent || 'none'} retry=${task.retryCount}/${MAX_TASK_RETRIES} result="${resultSnippet}"`);
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

  // Expose pending retry for external triggers (e.g., agent status changes)
  public async retryPendingTasks(): Promise<void> {
    await this.processPendingTasks();
  }

  /**
   * Reset a failed task back to pending so agents can retry it.
   * Increments retryCount. Returns null if task has exceeded MAX_TASK_RETRIES.
   */
  public async retryTask(taskId: string): Promise<Task | null> {
    const task = this.tasks.get(taskId);
    if (!task) throw new Error(`Task ${taskId} not found`);

    if (task.retryCount >= MAX_TASK_RETRIES) {
      return null; // permanently failed — do not retry
    }

    // Unlink from previous agent
    const previousAgent = task.assignedAgent;
    task.status = 'pending';
    task.assignedAgent = undefined;
    task.progress = 0;
    task.startedAt = undefined;
    task.completedAt = undefined;
    task.retryCount += 1;

    if (previousAgent) {
      const agent = this.agentRegistry.getAgent(previousAgent);
      if (agent && agent.status !== 'offline') {
        await this.agentRegistry.updateAgentStatus(previousAgent, 'online');
      }
    }

    logger.info(`Task ${taskId} reset for retry (attempt ${task.retryCount}/${MAX_TASK_RETRIES})`);
    this.emit('task_retry', task);

    // Try immediate assignment
    await this.assignOptimalAgent(taskId);
    return task;
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

  getAllTasks(projectName?: string): Task[] {
    const all = Array.from(this.tasks.values());
    return projectName
      ? all.filter(t => (t.metadata?.projectName as string | undefined) === projectName)
      : all;
  }

  getTasksByStatus(status: Task['status'], projectName?: string): Task[] {
    return this.getAllTasks(projectName).filter(task => task.status === status);
  }

  getTasksByAgent(agentId: string, projectName?: string): Task[] {
    return this.getAllTasks(projectName).filter(task => task.assignedAgent === agentId);
  }

  getTaskCount(): number {
    return this.tasks.size;
  }

  clearAll(): number {
    const count = this.tasks.size;
    this.tasks.clear();
    return count;
  }

  // Bulk restore tasks from event log (used by restoreFromLog)
  restoreTasks(taskMap: Map<string, Task>): void {
    for (const [id, task] of taskMap.entries()) {
      this.tasks.set(id, task);
    }
  }

  // Minimal demo project helper to satisfy EventHub callers
  async createDemoProject(): Promise<Task[]> {
    return [];
  }

  private async pickAgentFromPVM(task: Task): Promise<string | null> {
    try {
      if (!this.pvmIndexer) return null;
      const queryText = `${task.description} | capabilities: ${task.requiredCapabilities.join(', ')}`;
      const results = await this.pvmIndexer.search(queryText, 5);
      for (const result of results) {
        const candidateId = (result.message as any).agent;
        if (!candidateId) continue;
        const agent = this.agentRegistry.getAgent(candidateId);
        if (agent && agent.status !== 'offline' && !agent.currentTask) {
          return candidateId;
        }
      }
      return null;
    } catch (error: any) {
      logger.error(`PVM fallback search failed for task ${task.id}: ${error.message}`);
      return null;
    }
  }

  private getBestEffortAgent(): Agent | null {
    const available = this.agentRegistry.getAvailableAgents();
    if (available.length === 0) return null;
    const sorted = [...available].sort((a, b) => (b.performanceScore || 0) - (a.performanceScore || 0));
    return sorted[0] || null;
  }
}

export default TaskCoordinator;