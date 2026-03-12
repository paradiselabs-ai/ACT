import express from 'express';
import path from 'path';
import { createServer } from 'http';
import { Server } from 'socket.io';
import cors from 'cors';
import { AgentRegistry } from './services/AgentRegistry';
import { Task } from './services/TaskCoordinator';
import { TaskCoordinator, MAX_TASK_RETRIES } from './services/TaskCoordinator';
import { EventHub } from './services/EventHub';
import { SelfImprovementEngine } from './services/SelfImprovementEngine';
import { ChronologicalLog } from './services/ChronologicalLog';
import { LocalEmbeddingVectorStore } from './services/LocalEmbeddingVectorStore';
import { PVMIndexer } from './services/PVMIndexer';
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
  if (['completed', 'in_progress', 'assigned', 'pending', 'failed'].includes(lowered)) {
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

// Start PVM indexing
chronologicalLog.initialize().then(() => {
  pvmIndexer.startIndexing(10000); // Check for new events every 10 seconds
  logger.info('✅ ChronologicalLog initialized and PVM indexing started');
}).catch(err => {
  logger.error(`❌ Failed to initialize ChronologicalLog: ${err.message}`);
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    agents: agentRegistry.getAgentCount(),
    tasks: taskCoordinator.getTaskCount()
  });
});

// Agent management endpoints
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
  res.json({ success: true });
});

app.get('/api/projects/:name/briefs/:agentId', (req, res) => {
  const project = projects.get(req.params.name);
  if (!project) return res.status(404).json({ success: false, error: 'Project not found' });
  const content = project.briefs[req.params.agentId];
  if (!content) return res.status(404).json({ success: false, error: 'No brief for this agent' });
  res.json({ success: true, content });
});

// REST task endpoints
app.get('/api/tasks', (req, res) => {
  res.json({ tasks: taskCoordinator.getAllTasks() });
});

app.post('/api/tasks', async (req, res) => {
  try {
    const task = await taskCoordinator.createTask(req.body);
    if (task.assignedAgent) {
      await taskCoordinator.updateTaskProgress(task.id, { status: 'assigned' });
    } else {
      const assignment = await taskCoordinator.assignOptimalAgent(task.id);
      if (assignment) {
        io.emit('task_assigned', { taskId: task.id, agentId: assignment.agentId, task, timestamp: new Date().toISOString() });
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

// Get the active task assigned to a specific agent (must be before /:taskId)
app.get('/api/tasks/assigned', (req, res) => {
  const agentId = req.query.agent_id as string;
  if (!agentId) return res.status(400).json({ success: false, error: 'agent_id is required' });

  const tasks = taskCoordinator.getTasksByAgent(agentId);
  const active = tasks.find(t => t.status === 'assigned' || t.status === 'in_progress');
  res.json({ success: true, task: active || null });
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
        type: 'file_release'
      });
    }

    io.emit('task_completed', { taskId, agentId, success: taskSuccess, result, timestamp: new Date().toISOString() });
    res.json({ success: true });
  } catch (error: any) {
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
  chronologicalLog.append({
    timestamp: now,
    agent: agent_id,
    message: `claimed files for editing: ${file_paths.join(', ')} (task: ${task_id})`,
    type: 'file_claim'
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
      type: 'file_release'
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

// Read raw ChronologicalLog — used by import project to reconstruct task history
app.get('/api/log', async (req, res) => {
  try {
    const limit = parseInt(req.query.limit as string) || 500;
    const events = await chronologicalLog.getRecent(limit);
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

server.listen(PORT, () => {
  logger.info(`🚀 ACT Server running on port ${PORT}`);
  logger.info(`📊 Dashboard: http://localhost:3001`);
  logger.info(`🔗 WebSocket: ws://localhost:${PORT}`);
  logger.info(`💫 Ready for autonomous agent coordination!`);
});

export { app, io, agentRegistry, taskCoordinator, eventHub };