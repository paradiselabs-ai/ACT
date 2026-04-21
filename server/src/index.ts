import express from 'express';
import path from 'path';
import { createServer } from 'http';
import { Server } from 'socket.io';
import cors from 'cors';
import { AgentRegistry } from './services/AgentRegistry';
import { Task } from './services/TaskCoordinator';
import { TaskCoordinator, MAX_TASK_RETRIES, TerminalStateTransitionError } from './services/TaskCoordinator';
import { EventHub } from './services/EventHub';
import { SelfImprovementEngine } from './services/SelfImprovementEngine';
import { ChronologicalLog } from './services/ChronologicalLog';
import { LocalEmbeddingVectorStore } from './services/LocalEmbeddingVectorStore';
import { PVMIndexer } from './services/PVMIndexer';
import { extractSuccessCriteria } from './services/SPILParser';
import { logger } from './utils/logger';

const app = express();
const server = createServer(app);
const io = new Server(server, {
  cors: {
    origin: ["http://localhost:3000", "http://localhost:3001", "http://localhost:5173", "http://localhost:5000", "http://localhost:8080"],
    methods: ["GET", "POST"]
  }
});

// Middleware
app.use(cors());
app.use(express.static("public"));
app.use(express.json());

app.get("/", (req, res) => {
  res.sendFile(path.join(process.cwd(), "public/index.html"));
});
app.get("/dashboard", (req, res) => {
  res.sendFile(path.join(process.cwd(), "public/index.html"));
});

// Core ACT Services
const agentRegistry = new AgentRegistry();
const chronologicalLog = new ChronologicalLog();
const vectorStore = new LocalEmbeddingVectorStore();
const pvmIndexer = new PVMIndexer(chronologicalLog, vectorStore);
const taskCoordinator = new TaskCoordinator(agentRegistry, pvmIndexer);
const eventHub = new EventHub(io, agentRegistry, taskCoordinator, chronologicalLog);
const selfImprovementEngine = new SelfImprovementEngine(agentRegistry, taskCoordinator, eventHub);

const normalizeStatus = (progress: number | undefined, status?: string): Task['status'] | undefined => {
  if (progress !== undefined && progress >= 100) return 'completed';
  if (!status) {
    if (progress && progress > 0) return 'in_progress';
    return undefined;
  }
  const lowered = status.toLowerCase();
  if (['completed', 'in_progress', 'assigned', 'pending', 'failed', 'submitted_for_validation', 'validated'].includes(lowered)) {
    return lowered as Task['status'];
  }
  // Map common phrases
  if (lowered.includes('complete')) return 'completed';
  if (lowered.includes('progress') || lowered.includes('working') || lowered.includes('analysis') || lowered.includes('plan')) return 'in_progress';
  return undefined;
};

const getProjectStatusSummary = () => {
  const allTasks = taskCoordinator.getAllTasks();
  const totalTasks = allTasks.length;
  const completedTasks = allTasks.filter(task => task.status === 'completed').length;
  const progress = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;

  return {
    status: totalTasks === 0 ? 'initializing' : completedTasks === totalTasks ? 'completed' : 'active',
    progress: progress,
    activeAgents: agentRegistry.getOnlineAgentCount(),
    totalTasks: totalTasks,
    completedTasks: completedTasks
  };
};

// NOTE: PVM indexing is started AFTER restore in the startup block at the bottom.
// Do NOT initialize chronologicalLog here — single init path only.

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    agents: agentRegistry.getAgentCount(),
    tasks: taskCoordinator.getTaskCount()
  });
});

// Full system status — used by Observer, REPL `status`, and monitoring.
// Accepts ?project=NAME to scope the task list + counts to a single project —
// without it, old unrelated projects' tasks leak into the current session's
// Planner view and trigger false "critical issues in other projects" replies.
app.get('/api/status', (req, res) => {
  const project = typeof req.query.project === 'string' ? req.query.project : '';
  const allAgents = agentRegistry.getAllAgents();
  let allTasks = Array.from(taskCoordinator.getAllTasks());
  if (project) {
    allTasks = allTasks.filter(t => (t.metadata?.projectName as string | undefined) === project);
  }

  // Task counts by status
  const tasksByStatus: Record<string, number> = {};
  for (const task of allTasks) {
    tasksByStatus[task.status] = (tasksByStatus[task.status] || 0) + 1;
  }

  // Agent counts by status
  const agentsByStatus: Record<string, number> = {};
  for (const agent of allAgents) {
    agentsByStatus[agent.status] = (agentsByStatus[agent.status] || 0) + 1;
  }

  // File locks
  const lockCount = fileLocks.size;

  // Projects
  const projectCount = projects.size;

  res.json({
    timestamp: new Date().toISOString(),
    tasks: {
      total: allTasks.length,
      byStatus: tasksByStatus,
    },
    agents: {
      total: allAgents.length,
      byStatus: agentsByStatus,
      list: allAgents.map(a => ({
        id: a.id,
        name: a.name,
        status: a.status,
        currentTask: a.currentTask,
        role: (a as any).role,
      })),
    },
    fileLocks: lockCount,
    projects: projectCount,
    pvm: pvmIndexer.getStatus(),
  });
});

// Full reset — clears ALL state (projects, tasks, agents, locks, inboxes).
// The reset event is persisted so replay skips everything before it.
app.post('/api/dev/reset', (req, res) => {
  const taskCount = taskCoordinator.clearAll();
  const agentCount = agentRegistry.clearAll();
  const lockCount = fileLocks.size;
  fileLocks.clear();
  const inboxCount = agentInboxes.size;
  agentInboxes.clear();
  const projectCount = projects.size;
  projects.clear();

  chronologicalLog.append({
    timestamp: new Date().toISOString(),
    agent: 'system',
    message: `full reset: cleared ${projectCount} projects, ${taskCount} tasks, ${agentCount} agents, ${lockCount} locks, ${inboxCount} inboxes`,
    type: 'dev_reset',
    data: { projectCount, taskCount, agentCount, lockCount, inboxCount },
  });

  res.json({ success: true, cleared: { projects: projectCount, tasks: taskCount, agents: agentCount, fileLocks: lockCount, inboxes: inboxCount } });
});

// ─── A2A Protocol (Agent-to-Agent) ──────────────────────────────────────────
// ACT serves Agent Cards on behalf of registered agents (brokered A2A pattern).
// Planner pushes tasks to ACT targeting role IDs; Runner detects and spawns.

// System-level Agent Card — describes the ACT coordination server itself
app.get('/.well-known/agent.json', (req, res) => {
  res.json({
    id: 'act-server',
    name: 'ACT Coordination Server',
    description: 'Agentic harness for multi-agent CLI coordination',
    capabilities: ['task-coordination', 'pvm-memory', 'file-locking', 'message-routing'],
    taskEndpoint: `${req.protocol}://${req.get('host')}/api/tasks`,
    agentsEndpoint: `${req.protocol}://${req.get('host')}/api/agents`,
    registeredAgents: agentRegistry.getAgentCount(),
    status: 'online'
  });
});

// Per-agent Agent Card — ACT serves this on behalf of the registered agent
app.get('/api/agents/:agentId/agent.json', (req, res) => {
  const agent = agentRegistry.getAgent(req.params.agentId);
  if (!agent) return res.status(404).json({ error: `Agent "${req.params.agentId}" not found` });
  res.json({
    id: agent.id,
    name: agent.name,
    role: agent.role || 'developer',
    capabilities: agent.capabilities,
    taskEndpoint: `${req.protocol}://${req.get('host')}/api/agents/${agent.id}/tasks`,
    status: agent.status,
    pvmProfile: {
      tasksCompleted: agent.tasksCompleted,
      performanceScore: agent.performanceScore,
      averageTaskTime: agent.averageTaskTime
    }
  });
});

// Task push — Planner pushes a task to a specific agent via A2A
app.post('/api/agents/:agentId/tasks', async (req, res) => {
  try {
    const { agentId } = req.params;
    const { title, description, spil, priority, requiredCapabilities, projectName } = req.body;

    // Create the task pre-assigned to the target agent
    const task = await taskCoordinator.createTask({
      description: title ? `${title}\n\n${description || spil || ''}` : (description || spil || ''),
      assignedAgent: agentId,
      priority: priority || 'medium',
      requiredCapabilities: requiredCapabilities || [],
      metadata: { projectName, spil, title }
    });

    // Mark as assigned
    await taskCoordinator.updateTaskProgress(task.id, { status: 'assigned' });

    // Log the event with full payload
    chronologicalLog.append({
      timestamp: new Date().toISOString(),
      agent: 'planner',
      message: `A2A task push: ${task.id} -> ${agentId}`,
      type: 'task_assigned',
      data: { taskId: task.id, agentId, title, projectName }
    });

    // Fire Socket.io event — if agent is connected, it gets notified immediately
    io.emit('task_assigned', { taskId: task.id, agentId, task, timestamp: new Date().toISOString() });

    res.json({ success: true, task });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// ─── Agent management endpoints ─────────────────────────────────────────────
app.get('/api/agents', (req, res) => {
  res.json(agentRegistry.getAllAgents());
});

// REST-based agent registration (for MCP bridge - no socket required)
app.post('/api/agents/register', async (req, res) => {
  try {
    const { agentId, name, capabilities, model, provider } = req.body;
    if (!agentId) return res.status(400).json({ success: false, error: 'agentId is required' });

    // Reject duplicate agent IDs — each instance must have a unique identity
    if (agentRegistry.isRegistered(agentId)) {
      return res.status(409).json({
        success: false,
        conflict: true,
        error: `Agent ID "${agentId}" is already registered. Choose a different ID (e.g. append a number: "${agentId}-2").`
      });
    }

    const agent = await agentRegistry.registerAgent(agentId, {
      name: name || agentId,
      capabilities: capabilities || [],
      model,
      provider,
      status: 'online'
    });

    chronologicalLog.append({
      timestamp: new Date().toISOString(),
      agent: agentId,
      message: `agent registered: ${agentId}`,
      type: 'agent_registered',
      data: { agentId, name: name || agentId, capabilities: capabilities || [], model, provider }
    });

    io.emit('agent_joined', { agent, timestamp: new Date().toISOString() });
    await taskCoordinator.retryPendingTasks();
    res.json({ success: true, agent });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// Deregister an agent — removes from registry entirely
app.delete('/api/agents/:agentId', (req, res) => {
  const { agentId } = req.params;
  const removed = agentRegistry.removeAgent(agentId);
  if (!removed) {
    return res.status(404).json({ success: false, error: `Agent "${agentId}" not found` });
  }
  // Auto-release all file locks held by this agent
  for (const [fp, lock] of fileLocks.entries()) {
    if (lock.agentId === agentId) {
      fileLocks.delete(fp);
    }
  }
  io.emit('agent_left', { agentId, timestamp: new Date().toISOString() });
  res.json({ success: true, agentId });
});

// In-memory project store
interface ProjectRecord {
  name: string;
  workspace: string;
  description: string;
  techStack: string;
  constraints?: string;
  successCriteria: string;
  agents: string[];
  status: 'planning' | 'active' | 'paused' | 'completed';
  createdAt: string;
  briefs: Record<string, string>; // agentId → AGENT.md content
}
const projects = new Map<string, ProjectRecord>();

// Per-agent message inbox for MCP agents (who can't hold a Socket.io connection)
interface InboxMessage {
  id: string;
  from: string;
  message: string;
  type: string;
  timestamp: string;
  read: boolean;
}
const agentInboxes = new Map<string, InboxMessage[]>();
const INBOX_TTL_MS = 10 * 60 * 1000; // 10 minutes

function pruneInbox(agentId: string): void {
  const inbox = agentInboxes.get(agentId);
  if (!inbox) return;
  const cutoff = Date.now() - INBOX_TTL_MS;
  const pruned = inbox.filter(m => new Date(m.timestamp).getTime() > cutoff);
  agentInboxes.set(agentId, pruned);
}

function bufferMessageForAgent(agentId: string, from: string, message: string, type: string): void {
  pruneInbox(agentId);
  const inbox = agentInboxes.get(agentId) || [];
  inbox.push({
    id: `msg_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
    from,
    message,
    type,
    timestamp: new Date().toISOString(),
    read: false
  });
  agentInboxes.set(agentId, inbox);
}

// Project endpoints
app.post('/api/projects', (req, res) => {
  const { name, workspace, description, techStack, constraints, successCriteria, agents } = req.body;
  if (!name || !workspace) return res.status(400).json({ success: false, error: 'name and workspace required' });

  const project: ProjectRecord = {
    name,
    workspace,
    description: description || '',
    techStack: techStack || '',
    constraints,
    successCriteria: successCriteria || '',
    agents: agents || [],
    status: 'planning',
    createdAt: new Date().toISOString(),
    briefs: {}
  };
  projects.set(name, project);
  chronologicalLog.append({
    timestamp: new Date().toISOString(),
    agent: 'system',
    message: `project created: ${name}`,
    type: 'project_created',
    data: project
  });
  res.json({ success: true, project });
});

app.get('/api/projects', (req, res) => {
  res.json(Array.from(projects.values()));
});

app.get('/api/projects/:name', (req, res) => {
  const project = projects.get(req.params.name);
  if (!project) return res.status(404).json({ success: false, error: 'Project not found' });

  // Compute live task breakdown for this project
  const projectTasks = taskCoordinator.getAllTasks().filter(
    t => t.metadata?.projectName === project.name && t.metadata?.type !== 'planning'
  );
  const taskSummary = {
    total: projectTasks.length,
    completed: projectTasks.filter(t => t.status === 'completed').length,
    in_progress: projectTasks.filter(t => t.status === 'in_progress' || t.status === 'assigned').length,
    pending: projectTasks.filter(t => t.status === 'pending').length,
    failed: projectTasks.filter(t => t.status === 'failed').length,
  };

  // Derive status from task state — never stuck on 'planning'
  let derivedStatus = project.status;
  if (projectTasks.length > 0) {
    if (taskSummary.completed === taskSummary.total) {
      derivedStatus = 'completed';
    } else if (taskSummary.failed > 0 && taskSummary.in_progress === 0 && taskSummary.pending === 0) {
      derivedStatus = 'paused';
    } else {
      derivedStatus = 'active';
    }
  }

  res.json({
    success: true,
    project: { ...project, status: derivedStatus },
    tasks: projectTasks.map(t => ({
      id: t.id,
      title: t.metadata?.title || t.description.substring(0, 60),
      status: t.status,
      assignedAgent: t.assignedAgent,
      progress: t.progress,
      retryCount: t.retryCount,
      dependencies: t.dependencies,
    })),
    taskSummary,
  });
});

app.patch('/api/projects/:name', (req, res) => {
  const project = projects.get(req.params.name);
  if (!project) return res.status(404).json({ success: false, error: 'Project not found' });
  Object.assign(project, req.body);
  res.json({ success: true, project });
});

// Brief endpoints
app.post('/api/projects/:name/briefs', (req, res) => {
  const project = projects.get(req.params.name);
  if (!project) return res.status(404).json({ success: false, error: 'Project not found' });
  const { agentId, content } = req.body;
  if (!agentId || !content) return res.status(400).json({ success: false, error: 'agentId and content required' });
  project.briefs[agentId] = content;
  chronologicalLog.append({
    timestamp: new Date().toISOString(),
    agent: agentId,
    message: `brief stored for project: ${project.name}`,
    type: 'brief_stored',
    data: { projectName: project.name, agentId, content }
  });
  res.json({ success: true });
});

app.get('/api/projects/:name/briefs/:agentId', (req, res) => {
  const project = projects.get(req.params.name);
  if (!project) return res.status(404).json({ success: false, error: 'Project not found' });
  const content = project.briefs[req.params.agentId];
  if (!content) return res.status(404).json({ success: false, error: 'No brief for this agent' });
  res.json({ success: true, content });
});

// REST task endpoints. ?project=NAME scopes the listing; without it, all
// projects' tasks return (useful for server-level tooling, dangerous for
// a Planner asking "what's on my plate?").
app.get('/api/tasks', (req, res) => {
  const project = typeof req.query.project === 'string' ? req.query.project : '';
  let tasks = taskCoordinator.getAllTasks();
  if (project) {
    tasks = tasks.filter(t => (t.metadata?.projectName as string | undefined) === project);
  }
  res.json({ tasks });
});

app.post('/api/tasks', async (req, res) => {
  try {
    const task = await taskCoordinator.createTask(req.body);
    chronologicalLog.append({
      timestamp: new Date().toISOString(),
      agent: 'system',
      message: `task created: ${task.id}`,
      type: 'task_created',
      data: task
    });
    if (task.assignedAgent) {
      await taskCoordinator.updateTaskProgress(task.id, { status: 'assigned' });
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: task.assignedAgent,
        message: `task assigned: ${task.id} -> ${task.assignedAgent}`,
        type: 'task_assigned',
        data: { taskId: task.id, agentId: task.assignedAgent }
      });
    } else {
      const assignment = await taskCoordinator.assignOptimalAgent(task.id);
      if (assignment) {
        io.emit('task_assigned', { taskId: task.id, agentId: assignment.agentId, task, timestamp: new Date().toISOString() });
        chronologicalLog.append({
          timestamp: new Date().toISOString(),
          agent: assignment.agentId,
          message: `task assigned: ${task.id} -> ${assignment.agentId}`,
          type: 'task_assigned'
        });
      }
    }
    res.json({ success: true, task: taskCoordinator.getTask(task.id) });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// Get permanently failed tasks (retryCount >= MAX_TASK_RETRIES) — polled by REPL
app.get('/api/tasks/failed-permanently', (req, res) => {
  const tasks = taskCoordinator.getAllTasks().filter(
    t => t.status === 'failed' && t.retryCount >= MAX_TASK_RETRIES
  );
  res.json({ success: true, tasks });
});

// Get the active task assigned to a specific agent (must be before /:taskId).
// Runners call this every poll interval. Without a project scope, agent IDs
// (dev-1, backend-1, etc.) are shared across projects — a runner in project
// A ends up executing a task from project B that happened to still be
// assigned to the same agent ID. Always pass ?project= from the runner.
app.get('/api/tasks/assigned', (req, res) => {
  const agentId = req.query.agent_id as string;
  const project = typeof req.query.project === 'string' ? req.query.project : '';
  if (!agentId) return res.status(400).json({ success: false, error: 'agent_id is required' });

  let tasks = taskCoordinator.getTasksByAgent(agentId);
  if (project) {
    tasks = tasks.filter(t => (t.metadata?.projectName as string | undefined) === project);
  }
  const active = tasks.find(t => t.status === 'assigned' || t.status === 'in_progress');
  res.json({ success: true, task: active || null });
});

// Get tasks pending validation (must be before /:taskId)
// Accepts ?project=NAME to scope to a single project. Without that scope the
// TUI's Assurance poll loop would pick up tasks from whatever project last
// ran — Assurance ends up validating work for a project the user isn't even
// looking at. Runners + orchestrator should always pass the project filter.
app.get('/api/tasks/pending-validation', (req, res) => {
  const project = typeof req.query.project === 'string' ? req.query.project : '';
  let tasks = taskCoordinator.getTasksByStatus('submitted_for_validation');
  if (project) {
    tasks = tasks.filter(t => (t.metadata?.projectName as string | undefined) === project);
  }
  const enriched = tasks.map(t => ({
    ...t,
    successCriteria: extractSuccessCriteria(t.description || ''),
  }));
  res.json({ tasks: enriched });
});

// Get validated tasks (must be before /:taskId). Same project scoping as
// pending-validation — prevents QA/Synthesizer from assembling outputs for
// a project other than the one the TUI is attached to.
app.get('/api/tasks/validated', (req, res) => {
  const project = typeof req.query.project === 'string' ? req.query.project : '';
  let tasks = taskCoordinator.getTasksByStatus('validated');
  if (project) {
    tasks = tasks.filter(t => (t.metadata?.projectName as string | undefined) === project);
  }
  res.json({ tasks });
});

app.get('/api/tasks/:taskId', (req, res) => {
  const task = taskCoordinator.getTask(req.params.taskId);
  if (!task) return res.status(404).json({ success: false, error: 'Task not found' });
  res.json({ success: true, task });
});

// Patch task dependencies (called by REPL after two-pass task creation)
app.patch('/api/tasks/:taskId/dependencies', (req, res) => {
  const task = taskCoordinator.getTask(req.params.taskId);
  if (!task) return res.status(404).json({ success: false, error: 'Task not found' });
  const { dependencies } = req.body;
  if (!Array.isArray(dependencies)) {
    return res.status(400).json({ success: false, error: 'dependencies must be an array of task IDs' });
  }
  // Validate all referenced IDs exist
  const missing = dependencies.filter((id: string) => !taskCoordinator.getTask(id));
  if (missing.length > 0) {
    return res.status(400).json({ success: false, error: `Unknown dependency task IDs: ${missing.join(', ')}` });
  }
  task.dependencies = dependencies;
  res.json({ success: true, task });
});

// Update task progress
app.post('/api/tasks/:taskId/progress', async (req, res) => {
  try {
    const { taskId } = req.params;
    const { agentId, progress, status, message } = req.body;

    await taskCoordinator.updateTaskProgress(taskId, { progress, status, message });
    io.emit('task_progress_updated', { taskId, agentId, progress, status, message, timestamp: new Date().toISOString() });
    res.json({ success: true });
  } catch (error: any) {
    if (error instanceof TerminalStateTransitionError) {
      return res.status(409).json({ success: false, error: error.message, code: error.code, fromStatus: error.fromStatus, toStatus: error.toStatus });
    }
    res.status(500).json({ success: false, error: error.message });
  }
});

// Mark task complete or failed
app.post('/api/tasks/:taskId/complete', async (req, res) => {
  try {
    const { taskId } = req.params;
    const { agentId, success: taskSuccess, result } = req.body;

    await taskCoordinator.updateTaskProgress(taskId, {
      status: taskSuccess ? 'completed' : 'failed',
      progress: taskSuccess ? 100 : undefined,
      message: result
    });

    // Auto-release any file locks held by this task
    const releasedFiles: string[] = [];
    for (const [fp, lock] of fileLocks.entries()) {
      if (lock.taskId === taskId) {
        fileLocks.delete(fp);
        releasedFiles.push(fp);
      }
    }
    if (releasedFiles.length > 0) {
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: agentId || 'system',
        message: `auto-released file locks on task complete: ${releasedFiles.join(', ')} (task: ${taskId})`,
        type: 'file_release',
        data: { filePaths: releasedFiles, agentId: agentId || 'system', taskId },
      });
    }

    // Emit distinct events for success vs failure so the orchestrator's
    // coordination event loop can auto-route failures to the Planner
    // (formatCoordEvent has separate handlers for task_completed and task_failed).
    if (taskSuccess) {
      io.emit('task_completed', { taskId, agentId, success: true, result, timestamp: new Date().toISOString() });
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: agentId || 'system',
        message: `task completed: ${taskId}`,
        type: 'task_completed',
        data: { taskId, agentId, success: true, result }
      });
    } else {
      const resultSnippet = typeof result === 'string' ? result.slice(0, 200) : '';
      io.emit('task_failed', { taskId, agentId, result, timestamp: new Date().toISOString() });
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: agentId || 'system',
        message: `task failed: ${taskId}${resultSnippet ? ' — ' + resultSnippet : ''}`,
        type: 'task_failed',
        data: { taskId, agentId, success: false, result }
      });
    }
    res.json({ success: true });
  } catch (error: any) {
    if (error instanceof TerminalStateTransitionError) {
      return res.status(409).json({ success: false, error: error.message, code: error.code, fromStatus: error.fromStatus, toStatus: error.toStatus });
    }
    res.status(500).json({ success: false, error: error.message });
  }
});

// Retry a failed task — resets to pending and increments retryCount
app.post('/api/tasks/:taskId/retry', async (req, res) => {
  try {
    const { taskId } = req.params;
    const task = taskCoordinator.getTask(taskId);
    if (!task) return res.status(404).json({ success: false, error: 'Task not found' });

    if (task.status !== 'failed') {
      return res.status(400).json({ success: false, error: `Task is not failed (status: ${task.status})` });
    }

    const retried = await taskCoordinator.retryTask(taskId);
    if (!retried) {
      return res.status(409).json({
        success: false,
        permanentlyFailed: true,
        error: `Task has exceeded max retries (${MAX_TASK_RETRIES}). It is permanently failed.`,
        task
      });
    }

    io.emit('task_retry', { taskId, retryCount: retried.retryCount, timestamp: new Date().toISOString() });
    res.json({ success: true, task: retried });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// ─── Assurance Validation Endpoints ────────────────────────────────────────

// Submit task for Assurance validation
app.post('/api/tasks/:taskId/submit-for-validation', async (req, res) => {
  try {
    const { taskId } = req.params;
    const { agentId, selfVerification } = req.body;

    const task = taskCoordinator.getTask(taskId);
    if (!task) return res.status(404).json({ success: false, error: 'Task not found' });

    if (task.status !== 'completed' && task.status !== 'in_progress') {
      return res.status(400).json({ success: false, error: `Task must be completed or in_progress to submit for validation (current: ${task.status})` });
    }

    await taskCoordinator.updateTaskProgress(taskId, { status: 'submitted_for_validation' });

    // Store self-verification in metadata
    if (selfVerification) {
      task.metadata = { ...(task.metadata || {}), selfVerification };
    }

    io.emit('task_submitted_for_validation', { taskId, agentId, timestamp: new Date().toISOString() });
    chronologicalLog.append({
      timestamp: new Date().toISOString(),
      agent: agentId || 'system',
      message: `task submitted for validation: ${taskId}`,
      type: 'task_submitted_for_validation',
      data: { taskId, agentId }
    });

    res.json({ success: true, task });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// (pending-validation route moved above /:taskId)

// Submit validation verdict (Assurance reports result)
app.post('/api/tasks/:taskId/validation-verdict', async (req, res) => {
  try {
    const { taskId } = req.params;
    const { agentId, passed, score, criteriaResults, gaps, feedback } = req.body;

    const task = taskCoordinator.getTask(taskId);
    if (!task) return res.status(404).json({ success: false, error: 'Task not found' });

    const verdict = { passed, score, criteriaResults, gaps, feedback, timestamp: new Date().toISOString() };

    if (passed) {
      // Approved — forward to QA/Synthesizer
      await taskCoordinator.updateTaskProgress(taskId, { status: 'validated' });
      task.metadata = { ...(task.metadata || {}), validationVerdict: verdict };

      logger.info(`validation_verdict_accepted task=${taskId} score=${score}`);
      io.emit('task_validated', { taskId, agentId, score, timestamp: new Date().toISOString() });
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: agentId || 'assurance',
        message: `task validated (score: ${score}/100): ${taskId}`,
        type: 'task_validated',
        data: { taskId, agentId, score, passed: true }
      });

      res.json({ success: true, task, action: 'validated' });
    } else {
      // Failed — return to agent for rework
      await taskCoordinator.updateTaskProgress(taskId, { status: 'assigned' });
      const attempts = ((task.metadata?.validationAttempts as number) || 0) + 1;
      task.metadata = {
        ...(task.metadata || {}),
        validationVerdict: verdict,
        validationGaps: gaps,
        validationAttempts: attempts,
      };

      logger.info(`validation_verdict_rejected task=${taskId} score=${score} attempts=${attempts} gaps="${(gaps || '').slice(0, 120)}"`);
      io.emit('task_validation_failed', { taskId, agentId, score, gaps, timestamp: new Date().toISOString() });
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: agentId || 'assurance',
        message: `task validation failed (score: ${score}/100): ${taskId} — returned to agent`,
        type: 'task_validation_failed',
        data: { taskId, agentId, score, passed: false, gaps }
      });

      res.json({ success: true, task, action: 'returned_to_agent' });
    }
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// (validated route moved above /:taskId)

// Send an agent message (MCP bridge alternative to socket agent_message)
app.post('/api/messages', async (req, res) => {
  try {
    const { sender, message } = req.body;
    if (!sender || !message) return res.status(400).json({ success: false, error: 'sender and message are required' });

    const senderAgent = agentRegistry.getAgent(sender);
    const senderName = senderAgent?.name || sender;

    // Route via EventHub (socket broadcast, classification, rate limiting)
    if (eventHub) {
      await eventHub.handleAgentMessage(sender, senderName, message, new Date().toISOString());
    }

    // Buffer into recipient inbox for MCP agents who can't receive socket events
    const mentionMatch = message.match(/^@(\S+)/);
    if (mentionMatch) {
      // Direct mention — buffer only for the named recipient
      const recipientName = mentionMatch[1];
      const allAgents = agentRegistry.getAllAgents();
      const recipient = allAgents.find(
        a => a.name.toLowerCase() === recipientName.toLowerCase() || a.id.toLowerCase() === recipientName.toLowerCase()
      );
      if (recipient) {
        bufferMessageForAgent(recipient.id, sender, message, 'direct_mention');
      }
    } else {
      // Broadcast message — buffer for every registered agent except sender
      const allAgents = agentRegistry.getAllAgents();
      for (const agent of allAgents) {
        if (agent.id !== sender) {
          bufferMessageForAgent(agent.id, sender, message, 'broadcast');
        }
      }
    }

    res.json({ success: true });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// Get messages from an agent's inbox
app.get('/api/agents/:agentId/messages', (req, res) => {
  const { agentId } = req.params;
  const { since, limit } = req.query;

  pruneInbox(agentId);
  let inbox = agentInboxes.get(agentId) || [];

  if (since) {
    const sinceTime = new Date(since as string).getTime();
    if (!isNaN(sinceTime)) {
      inbox = inbox.filter(m => new Date(m.timestamp).getTime() > sinceTime);
    }
  }

  const maxLimit = Math.min(parseInt(limit as string) || 20, 100);
  const messages = inbox.slice(0, maxLimit);

  // Mark returned messages as read
  const returnedIds = new Set(messages.map(m => m.id));
  const fullInbox = agentInboxes.get(agentId) || [];
  fullInbox.forEach(m => { if (returnedIds.has(m.id)) m.read = true; });

  res.json({ success: true, messages, unread_count: inbox.filter(m => !m.read).length });
});

// ─── File lock registry ───────────────────────────────────────────────────────
// Prevents multiple agents editing the same file simultaneously.
// Locks are in-memory and auto-released on task complete or agent removal.

interface FileLock {
  filePath: string;
  agentId: string;
  taskId: string;
  lockedAt: string;
}

const fileLocks = new Map<string, FileLock>(); // filePath → lock

// Claim one or more files for exclusive editing
app.post('/api/files/claim', (req, res) => {
  const { agent_id, task_id, file_paths } = req.body;
  if (!agent_id || !task_id || !Array.isArray(file_paths) || file_paths.length === 0) {
    return res.status(400).json({ success: false, error: 'agent_id, task_id, and file_paths[] are required' });
  }

  const conflicts: { filePath: string; lockedBy: string; taskId: string }[] = [];
  for (const fp of file_paths) {
    const existing = fileLocks.get(fp);
    if (existing && existing.agentId !== agent_id) {
      conflicts.push({ filePath: fp, lockedBy: existing.agentId, taskId: existing.taskId });
    }
  }

  if (conflicts.length > 0) {
    return res.status(409).json({
      success: false,
      conflict: true,
      conflicts,
      message: `${conflicts.length} file(s) are currently being edited by another agent. Wait or coordinate via send_message.`
    });
  }

  // No conflicts — claim all
  const now = new Date().toISOString();
  for (const fp of file_paths) {
    fileLocks.set(fp, { filePath: fp, agentId: agent_id, taskId: task_id, lockedAt: now });
  }

  // Log to ChronologicalLog so PVM captures file ownership patterns
  // and file locks survive server restarts via event replay.
  chronologicalLog.append({
    timestamp: now,
    agent: agent_id,
    message: `claimed files for editing: ${file_paths.join(', ')} (task: ${task_id})`,
    type: 'file_claim',
    data: { filePaths: file_paths, agentId: agent_id, taskId: task_id },
  });

  res.json({ success: true, claimed: file_paths });
});

// Release one or more file locks
app.post('/api/files/release', (req, res) => {
  const { agent_id, task_id, file_paths } = req.body;
  if (!agent_id || !Array.isArray(file_paths)) {
    return res.status(400).json({ success: false, error: 'agent_id and file_paths[] are required' });
  }

  const released: string[] = [];
  for (const fp of file_paths) {
    const lock = fileLocks.get(fp);
    if (lock && lock.agentId === agent_id) {
      fileLocks.delete(fp);
      released.push(fp);
    }
  }

  if (released.length > 0) {
    chronologicalLog.append({
      timestamp: new Date().toISOString(),
      agent: agent_id,
      message: `released file locks: ${released.join(', ')} (task: ${task_id || 'unknown'})`,
      type: 'file_release',
      data: { filePaths: released, agentId: agent_id, taskId: task_id || '' },
    });
  }

  res.json({ success: true, released });
});

// Get current file lock state
app.get('/api/files/locks', (req, res) => {
  res.json({ success: true, locks: Array.from(fileLocks.values()) });
});

// Self-improvement endpoints
app.post('/api/improve', async (req, res) => {
  try {
    const improvementRequest = req.body;
    const result = await selfImprovementEngine.triggerExplicitImprovement(improvementRequest);
    res.json({ success: true, result });
  } catch (error: any) {
    logger.error(`Improvement request failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

app.get('/api/improvement/status', (req, res) => {
  res.json(selfImprovementEngine.getStatus());
});

// PVM endpoints
// PVM search is intentionally cross-project — its purpose is team coordination
// memory and agent skill profiles built from ALL prior work. Scoping would
// defeat the point: the Planner asks "has anyone ever solved X?" and wants
// past patterns from any project, not just the current one.
app.get('/api/pvm/search', async (req, res) => {
  try {
    const { query, limit } = req.query;
    if (!query || typeof query !== 'string') {
      return res.status(400).json({ success: false, error: 'Query parameter is required' });
    }

    const results = await pvmIndexer.search(query, limit ? parseInt(limit as string) : 10);
    res.json({ success: true, results });
  } catch (error: any) {
    logger.error(`PVM search failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

app.get('/api/pvm/status', (req, res) => {
  res.json(pvmIndexer.getStatus());
});

// Read raw ChronologicalLog — used by import project + Tier 1 agents via
// act_cli log. Accepts ?project=NAME to filter to events scoped to that
// project (task lifecycle + project-tagged messages). Without the filter,
// agents see the global stream and surface old-project activity.
app.get('/api/log', async (req, res) => {
  try {
    const project = typeof req.query.project === 'string' ? req.query.project : '';
    const rawLimit = parseInt(req.query.limit as string) || 500;
    // When filtering, fetch more and trim after — otherwise the project's
    // real events get capped by unrelated events in the same window.
    const fetchLimit = project ? Math.max(rawLimit * 4, 2000) : rawLimit;
    let events = await chronologicalLog.getRecent(fetchLimit);
    if (project) {
      events = events.filter(e => {
        const d = (e as any).data || {};
        if (d.task?.metadata?.projectName === project) return true;
        if (d.projectName === project) return true;
        if (d.metadata?.projectName === project) return true;
        // System project-created events
        if ((e as any).type === 'project_created' && d.name === project) return true;
        return false;
      });
      if (events.length > rawLimit) events = events.slice(-rawLimit);
    }
    res.json({ success: true, events, count: events.length });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// WebSocket connection handling
io.on('connection', (socket) => {
  logger.info(`🔗 Client connected: ${socket.id}`);
  console.log(`🔗 NEW CONNECTION: ${socket.id} at ${new Date().toLocaleTimeString()}`);

  // Agent registration
  socket.on('register_agent', async (data) => {
    try {
      const { agentId, capabilities, name, model, provider } = data;
      console.log(`🤖 AGENT REGISTRATION: ${agentId} with capabilities: ${capabilities?.join(', ')}`);

      await agentRegistry.registerAgent(agentId, {
        name: name || agentId,
        capabilities: capabilities || [],
        model,
        provider,
        socketId: socket.id,
        status: 'online'
      });

      socket.emit('agent_registered', {
        success: true,
        agentId,
        agent: {
          id: agentId,
          name: name || agentId,
          capabilities: capabilities || [],
          model,
          provider,
          status: 'online'
        }
      });

      // Broadcast to all clients (especially Windsurf's dashboard!)
      io.emit('agent_joined', {
        agent: {
          id: agentId,
          name: name || agentId,
          capabilities: capabilities || [],
          model,
          provider,
          status: 'online'
        },
        timestamp: new Date().toISOString()
      });

      io.emit('project_status_update', { status: getProjectStatusSummary() });
      await taskCoordinator.retryPendingTasks();
      console.log(`✅ AGENT REGISTERED: ${agentId} - Ready for coordination!`);
      // AgentRegistry already logs registration
    } catch (error: any) {
      logger.error(`Agent registration failed: ${error.message}`);
      socket.emit('registration_error', { error: error.message });
    }
  });

  // Task creation and assignment
  socket.on('create_task', async (data) => {
    try {
      console.log(`📋 TASK CREATION: ${data.title || 'Untitled'} requiring ${data.requiredCapabilities?.join(', ')}`);
      const task = await taskCoordinator.createTask(data);

      // Try to assign immediately
      const assignment = await taskCoordinator.assignOptimalAgent(task.id);

      if (assignment) {
        console.log(`🎯 TASK ASSIGNED: ${task.id} → ${assignment.agentId}`);
        io.emit('task_assigned', {
          taskId: task.id,
          agentId: assignment.agentId,
          task: task,
          timestamp: new Date().toISOString()
        });

        io.emit('project_status_update', { status: getProjectStatusSummary() });
        logger.info(`Task ${task.id} assigned to ${assignment.agentId}`);
      } else {
        console.log(`⏳ TASK PENDING: ${task.id} - No suitable agent available`);
        io.emit('task_pending', {
          taskId: task.id,
          task: task,
          reason: 'No suitable agent available',
          timestamp: new Date().toISOString()
        });
        io.emit('conflict_detected', [{
          type: 'capability_mismatch',
          involvedTasks: [task.id],
          involvedAgents: [],
          severity: 'low',
          suggestedResolution: 'No suitable agent available; awaiting PVM/best-effort fallback'
        }]);
        io.emit('project_status_update', { status: getProjectStatusSummary() });
      }

      socket.emit('task_created', { success: true, task });
    } catch (error: any) {
      logger.error(`Task creation failed: ${error.message}`);
      socket.emit('task_error', { error: error.message });
    }
  });

  // Task progress updates
  socket.on('task_progress', async (data) => {
    try {
      const { taskId, progress, status, message } = data;

      const statusToApply = normalizeStatus(progress, status);
      await taskCoordinator.updateTaskProgress(taskId, { progress, status: statusToApply, message });

      io.emit('task_progress', {
        taskId,
        progress,
        status: statusToApply,
        message,
        timestamp: new Date().toISOString()
      });

      const task = taskCoordinator.getTask(taskId);
      if (task && task.status === 'completed') {
        io.emit('task_completed', { taskId, task, timestamp: new Date().toISOString() });
      }

      io.emit('project_status_update', { status: getProjectStatusSummary() });
      logger.info(`Task ${taskId} progress: ${progress}% - ${status}`);
    } catch (error: any) {
      logger.error(`Task progress update failed: ${error.message}`);
    }
  });

  // Task progress updates (alternative handler name)
  socket.on('update_task_progress', async (data) => {
    try {
      const { taskId, progress, status, message, agentId } = data;

      const statusToApply = normalizeStatus(progress, status);
      await taskCoordinator.updateTaskProgress(taskId, { progress, status: statusToApply, message });

      io.emit('task_progress', {
        taskId,
        progress,
        status: statusToApply || `${progress}% complete`,
        message,
        timestamp: new Date().toISOString()
      });

      const task = taskCoordinator.getTask(taskId);
      if (task && task.status === 'completed') {
        io.emit('task_completed', { taskId, task, timestamp: new Date().toISOString() });
      }

      io.emit('project_status_update', { status: getProjectStatusSummary() });
      logger.info(`Task ${taskId} progress from ${agentId}: ${progress}%${status ? ' - ' + status : ''}`);
    } catch (error: any) {
      logger.error(`Task progress update failed: ${error.message}`);
    }
  });

  // Agent-to-agent messaging - INTELLIGENT COORDINATION
  socket.on('agent_message', async (data) => {
    try {
      const { sender, message, timestamp } = data;

      // Resolve the agent ID from the socket so we can rate-limit by identity
      const senderAgent = agentRegistry.getAgentBySocketId(socket.id);
      const senderId = senderAgent?.id || sender;

      if (eventHub) {
        await eventHub.handleAgentMessage(senderId, sender, message, timestamp);
      }
    } catch (error: any) {
      logger.error(`Agent message handling failed: ${error.message}`);
    }
  });

  // Agent status updates
  socket.on('agent_status', async (data) => {
    try {
      const { agentId, status, currentTask } = data;
      await agentRegistry.updateAgentStatus(agentId, status, currentTask);

      io.emit('agent_status_update', {
        agentId,
        status,
        currentTask,
        timestamp: new Date().toISOString()
      });
    } catch (error: any) {
      logger.error(`Agent status update failed: ${error.message}`);
    }
  });

  // Request handlers for dashboard
  socket.on('get_project_status', () => {
    const projectStatus = getProjectStatusSummary();
    socket.emit('project_status_update', { status: projectStatus });
    logger.info(`Sent project status: ${projectStatus.progress}% complete, ${projectStatus.activeAgents} agents online`);
  });

  socket.on('get_agent_registry', () => {
    const agents = agentRegistry.getAllAgents();
    // Emit agent_registered for each agent to update dashboard
    agents.forEach(agent => {
      socket.emit('agent_registered', { agent });
    });
    socket.emit('project_status_update', { status: getProjectStatusSummary() });
    logger.info(`Sent ${agents.length} agents to client`);
  });

  socket.on('get_tasks', () => {
    const allTasks = taskCoordinator.getAllTasks();
    // Emit task_assigned for each task to update dashboard
    allTasks.forEach(task => {
      socket.emit('task_assigned', {
        taskId: task.id,
        agentId: task.assignedAgent || 'unassigned',
        task: task
      });
    });
    logger.info(`Sent ${allTasks.length} tasks to client`);
  });
});

const PORT = process.env.PORT || 8080;

// Write PID file so the Go launcher can detect stale processes
import { writeFileSync, unlinkSync, existsSync, mkdirSync } from 'fs';
// __dirname equivalent for ESM: path.dirname(import.meta.url) isn't needed
// because the server's data dir is always relative to the server root (one
// level above src/). Use the same path the ChronologicalLog uses.
const SERVER_DATA_DIR = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..', 'data');
const PID_FILE = path.join(SERVER_DATA_DIR, 'act-server.pid');

function writePidFile() {
  const dir = path.dirname(PID_FILE);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  writeFileSync(PID_FILE, String(process.pid), 'utf-8');
}

function removePidFile() {
  try { unlinkSync(PID_FILE); } catch { /* ignore */ }
}

// Graceful shutdown — flush buffer, clean up, exit
let shuttingDown = false;
async function gracefulShutdown(signal: string) {
  if (shuttingDown) return;
  shuttingDown = true;
  logger.info(`Received ${signal}, shutting down gracefully...`);

  // Stop accepting new connections
  server.close();

  // Flush the event log buffer to disk
  try {
    await chronologicalLog.close();
    logger.info('Event log flushed and closed');
  } catch (err) {
    logger.error(`Failed to flush event log on shutdown: ${err}`);
  }

  removePidFile();
  process.exit(0);
}

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));
process.on('SIGHUP', () => gracefulShutdown('SIGHUP'));

// Restore state from ChronologicalLog on startup (event sourcing)
// Single initialization — PVM indexing starts after restore completes.
const briefsMap = new Map<string, Map<string, string>>();
const taskMap = new Map();
const agentMap = new Map();

chronologicalLog.initialize().then(async () => {
  const counts = await chronologicalLog.restoreFromLog(projects, taskMap, briefsMap, agentMap, fileLocks);

  // Restore agents (marked as offline - they'll re-register)
  agentRegistry.restoreAgents(agentMap);

  // Restore tasks to TaskCoordinator
  if (taskMap.size > 0) {
    taskCoordinator.restoreTasks(taskMap);
  }

  console.log(`Restored from ChronLog: ${counts.projectCount} projects, ${counts.taskCount} tasks, ${counts.briefCount} briefs, ${counts.agentCount} agents, ${counts.fileLockCount} file locks`);

  // Start PVM indexing after restore
  pvmIndexer.startIndexing(10000);
  logger.info('ChronologicalLog initialized and PVM indexing started');

  // Write PID file and start listening only after state is restored
  writePidFile();
  server.listen(PORT, () => {
    logger.info(`ACT Server running on port ${PORT}`);
    logger.info(`WebSocket: ws://localhost:${PORT}`);
  });
}).catch(err => {
  logger.error(`Failed to initialize ChronologicalLog: ${err.message}`);
  process.exit(1);
});

export { app, io, agentRegistry, taskCoordinator, eventHub };