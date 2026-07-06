import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
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

// queryProject reads the project-scope filter from a request, accepting both
// `?projectName=` (the canonical form used elsewhere in the codebase — agent
// runner, orchestrator client, slash commands) and `?project=` (legacy). Both
// keys are honored so callers using either name see consistent filtering. An
// empty result means "no scope — return everything," not "project named ''".
//
// One helper instead of N copies prevents the per-endpoint drift this codebase
// already paid for once (the F-G filter bugs in caafe6f — /api/log accepted
// `?project=` only, while runners passed `projectName=`, producing silent
// cross-project bleed in PVM views).
function queryProject(req: express.Request): string | undefined {
  const pn = req.query.projectName;
  if (typeof pn === 'string' && pn !== '') return pn;
  const p = req.query.project;
  if (typeof p === 'string' && p !== '') return p;
  return undefined;
}

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
  const project = queryProject(req);
  const allAgents = agentRegistry.getAllAgents(project);
  const allTasks = taskCoordinator.getAllTasks(project);

  const tasksByStatus: Record<string, number> = {};
  for (const task of allTasks) {
    tasksByStatus[task.status] = (tasksByStatus[task.status] || 0) + 1;
  }

  const agentsByStatus: Record<string, number> = {};
  for (const agent of allAgents) {
    agentsByStatus[agent.status] = (agentsByStatus[agent.status] || 0) + 1;
  }

  const body: Record<string, unknown> = {
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
  };

  if (!project) {
    body.fileLocks = fileLocks.size;
    body.projects = projects.size;
    body.pvm = pvmIndexer.getStatus();
  } else {
    body.project = project;
    body.fileLocks = Array.from(fileLocks.values()).filter(l => l.projectName === project).length;
  }

  res.json(body);
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
// Scope with ?projectName=NAME (or legacy ?project=NAME, via queryProject)
// so cross-project agents stay invisible. Without the filter, returns all
// agents (useful for server-level tooling).
app.get('/api/agents', (req, res) => {
  res.json(agentRegistry.getAllAgents(queryProject(req)));
});

// REST-based agent registration (for MCP bridge - no socket required)
app.post('/api/agents/register', async (req, res) => {
  try {
    const { agentId, name, projectName, capabilities, model, provider } = req.body;
    if (!agentId) return res.status(400).json({ success: false, error: 'agentId is required' });
    if (!projectName) return res.status(400).json({ success: false, error: 'projectName is required' });

    // Reject duplicate agent IDs on any LIVE collision — an existing online
    // agent with the same ID, regardless of project, is a real identity
    // conflict because the underlying registry Map is keyed by agentId.
    // Two simultaneous TUIs (one per project) would otherwise overwrite each
    // other; pick a project-prefixed ID instead (e.g. "dev-1-habits").
    // Offline entries (restored from the event log on server restart, or left
    // behind by a disconnected runner) are stale state; let the new caller
    // take over the identity. registerAgent already merges performance
    // counters from the existing record, so this is a clean handoff.
    const existing = agentRegistry.getAgent(agentId);
    if (existing && existing.status === 'online') {
      return res.status(409).json({
        success: false,
        conflict: true,
        error: `Agent ID "${agentId}" is already registered and online${existing.projectName !== projectName ? ` in project "${existing.projectName}"` : ''}. Choose a different ID (e.g. append a number: "${agentId}-2").`
      });
    }

    const agent = await agentRegistry.registerAgent(agentId, {
      name: name || agentId,
      projectName,
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
      data: { agentId, name: name || agentId, projectName, capabilities: capabilities || [], model, provider }
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
  // Auto-release all file locks held by this agent (across any project)
  for (const [key, lock] of fileLocks.entries()) {
    if (lock.agentId === agentId) {
      fileLocks.delete(key);
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
  projectName: string;
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

function bufferMessageForAgent(agentId: string, from: string, projectName: string, message: string, type: string): void {
  pruneInbox(agentId);
  const inbox = agentInboxes.get(agentId) || [];
  inbox.push({
    id: `msg_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
    from,
    projectName,
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
  const project = queryProject(req);
  let tasks = taskCoordinator.getAllTasks();
  if (project) {
    tasks = tasks.filter(t => (t.metadata?.projectName as string | undefined) === project);
  }
  res.json({ tasks });
});

app.post('/api/tasks', async (req, res) => {
  try {
    // Fail-closed guard against Planner placeholder/empty CREATE_TASK
    // emissions (observed live: Claude Code occasionally fires an
    // empty-body CREATE_TASK as an acknowledgement after Assurance
    // returns a pass verdict, despite the planner prompt explicitly
    // forbidding it). Without this server-side gate, downstream gets:
    //   1. an "Untitled task" with empty description in the dispatch queue
    //   2. dev-1 picks it up and runs whatever it thinks fits
    //   3. Assurance auto-passes it at 100% (no criteria to fail)
    //   4. the project's deliverable looks "done" via the wrong path
    // Rejecting at task creation surfaces the malformed input immediately
    // and forces the Planner to either re-emit a real CREATE_TASK or
    // stay silent (the correct behavior for a routine pass).
    const title = typeof req.body?.title === 'string' ? req.body.title.trim() : '';
    const description = typeof req.body?.description === 'string' ? req.body.description.trim() : '';
    if (!title) {
      return res.status(400).json({
        success: false,
        error: 'task title is required (empty or missing title rejected — likely a placeholder CREATE_TASK from the Planner)',
        code: 'empty_title',
      });
    }
    if (!description) {
      return res.status(400).json({
        success: false,
        error: 'task description is required (empty or missing description rejected — likely a placeholder CREATE_TASK from the Planner)',
        code: 'empty_description',
      });
    }
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
          type: 'task_assigned',
          data: { taskId: task.id, agentId: assignment.agentId }
        });
      }
    }
    res.json({ success: true, task: taskCoordinator.getTask(task.id) });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// Get permanently failed tasks (retryCount >= MAX_TASK_RETRIES) — polled by REPL.
// Scoped by ?project=NAME to mirror the rest of the tasks endpoints.
app.get('/api/tasks/failed-permanently', (req, res) => {
  const project = queryProject(req);
  const tasks = taskCoordinator.getAllTasks(project).filter(
    t => t.status === 'failed' && t.retryCount >= MAX_TASK_RETRIES
  );
  res.json({ success: true, tasks });
});

// Get the active task assigned to a specific agent (must be before /:taskId).
// Runners call this every poll interval. Without a project scope, agent IDs
// (dev-1, backend-1, etc.) are shared across projects — a runner in project
// A ends up executing a task from project B that happened to still be
// assigned to the same agent ID. Always pass ?project= from the runner.
// Self-healing heartbeat. The runner polls this every ~5s to fetch its assigned
// task. It also doubles as the agent's liveness signal: each poll refreshes
// lastSeen (so an actively-polling runner never trips the 5-min stale→offline
// health check), and an idle agent is brought online (clearing any stale
// currentTask left by a crashed prior session) and offered pending work via the
// same sweep registration uses. Without this, a live runner gets marked offline,
// is excluded from getAvailableAgents, and capable tasks sit pending forever.
app.get('/api/tasks/assigned', async (req, res) => {
  const agentId = req.query.agent_id as string;
  const project = queryProject(req);
  if (!agentId) return res.status(400).json({ success: false, error: 'agent_id is required' });

  const scoped = (t: { metadata?: { projectName?: unknown } }) =>
    !project || (t.metadata?.projectName as string | undefined) === project;

  const agent = agentRegistry.getAgent(agentId);
  if (agent) {
    const active = taskCoordinator.getTasksByAgent(agentId).filter(scoped)
      .find(t => t.status === 'assigned' || t.status === 'in_progress');
    if (active) {
      // Working — refresh liveness while preserving busy + currentTask.
      await agentRegistry.updateAgentStatus(agentId, 'busy', active.id);
    } else if (project) {
      // Idle — mark online (refreshes lastSeen, clears any stale currentTask so
      // the agent re-enters getAvailableAgents), then pull pending work. Scope the
      // sweep to THIS agent's project: a global retryPendingTasks() would let the
      // poll grab an older or project-less pending task from another project and
      // starve this project's queue. assignOptimalAgent already scopes the agent
      // match to each task's project, so a per-project sweep is correct.
      await agentRegistry.updateAgentStatus(agentId, 'online');
      for (const t of taskCoordinator.getAllTasks(project).filter(t => t.status === 'pending')) {
        try { await taskCoordinator.assignOptimalAgent(t.id); } catch { /* deps/no-match: stays pending */ }
      }
    } else {
      // No project scope on the poll — just refresh liveness, don't sweep globally.
      await agentRegistry.updateAgentStatus(agentId, 'online');
    }
  }

  const task = taskCoordinator.getTasksByAgent(agentId).filter(scoped)
    .find(t => t.status === 'assigned' || t.status === 'in_progress');
  res.json({ success: true, task: task || null });
});

// Get tasks pending validation (must be before /:taskId)
// Accepts ?project=NAME to scope to a single project. Without that scope the
// TUI's Assurance poll loop would pick up tasks from whatever project last
// ran — Assurance ends up validating work for a project the user isn't even
// looking at. Runners + orchestrator should always pass the project filter.
app.get('/api/tasks/pending-validation', (req, res) => {
  const project = queryProject(req);
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

// Get validated tasks awaiting synthesis. Excludes tasks that QA already
// synthesized (metadata.synthesizedAt is set) so a TUI restart doesn't
// trigger re-synthesis of already-shipped work — the QA poll loop sees an
// empty queue when there's nothing new to do.
//
// Same project scoping as pending-validation — prevents QA/Synthesizer from
// assembling outputs for a project other than the one the TUI is attached to.
app.get('/api/tasks/validated', (req, res) => {
  const project = queryProject(req);
  let tasks = taskCoordinator.getTasksByStatus('validated');
  if (project) {
    tasks = tasks.filter(t => (t.metadata?.projectName as string | undefined) === project);
  }
  tasks = tasks.filter(t => !t.metadata?.synthesizedAt);
  res.json({ tasks });
});

app.get('/api/tasks/:taskId', (req, res) => {
  const task = taskCoordinator.getTask(req.params.taskId);
  if (!task) return res.status(404).json({ success: false, error: 'Task not found' });
  // Optional ?project= scope: 404 if the task belongs to a different project.
  const project = queryProject(req);
  if (project && (task.metadata?.projectName as string | undefined) !== project) {
    return res.status(404).json({ success: false, error: 'Task not found in this project' });
  }
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

    // Auto-release any file locks held by this task. Group by project so we
    // emit one file_release event per project, preserving the projectName
    // field on every replayed event.
    const releasedByProject = new Map<string, string[]>();
    for (const [key, lock] of fileLocks.entries()) {
      if (lock.taskId === taskId) {
        fileLocks.delete(key);
        const list = releasedByProject.get(lock.projectName) ?? [];
        list.push(lock.filePath);
        releasedByProject.set(lock.projectName, list);
      }
    }
    for (const [projectName, releasedFiles] of releasedByProject.entries()) {
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: agentId || 'system',
        message: `auto-released file locks on task complete: ${releasedFiles.join(', ')} (task: ${taskId})`,
        type: 'file_release',
        data: { filePaths: releasedFiles, projectName, agentId: agentId || 'system', taskId },
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

    const previousAgent = task.assignedAgent;
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
    // Log to ChronLog so retryCount and pending status survive a server restart.
    // Without this, a retried task rehydrates as 'failed' with retryCount=0,
    // defeating the MAX_TASK_RETRIES permanently-failed guard.
    chronologicalLog.append({
      timestamp: new Date().toISOString(),
      agent: previousAgent || 'system',
      message: `task retried: ${taskId} (attempt ${retried.retryCount}/${MAX_TASK_RETRIES})`,
      type: 'task_retry',
      data: { taskId, retryCount: retried.retryCount, previousAgent }
    });
    res.json({ success: true, task: retried });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// Abandon a task — Planner escalation path. Marks the task permanently
// failed with metadata.abandoned=true so retry-eligibility skips it.
// Distinct from /retry (which resets to pending and re-dispatches).
app.post('/api/tasks/:taskId/abandon', async (req, res) => {
  try {
    const { taskId } = req.params;
    const reason = (req.body?.reason as string | undefined)?.trim() || 'no reason provided';

    const task = taskCoordinator.getTask(taskId);
    if (!task) return res.status(404).json({ success: false, error: 'Task not found' });

    const abandoned = await taskCoordinator.abandonTask(taskId, reason);
    io.emit('task_abandoned', { taskId, reason, timestamp: new Date().toISOString() });
    res.json({ success: true, task: abandoned });
  } catch (error: any) {
    // abandonTask throws on already-terminal-success — return 409 conflict.
    if (error.message?.startsWith('cannot abandon task')) {
      return res.status(409).json({ success: false, error: error.message });
    }
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

    // Fail closed on zero-criteria submissions: with no @success_criteria in
    // the task description Assurance has nothing to score against and
    // historically rubber-stamped 100% (wordtallies 9c7bdb39). Reject at the
    // seam so the runner/Planner sees the failure before Assurance is invoked.
    const criteria = extractSuccessCriteria(task.description || '');
    if (criteria.length === 0) {
      chronologicalLog.append({
        timestamp: new Date().toISOString(),
        agent: agentId || 'system',
        message: `validation submission rejected — no @success_criteria on task: ${taskId}`,
        type: 'validation_submission_rejected',
        data: { taskId, agentId, reason: 'NO_SUCCESS_CRITERIA' }
      });
      return res.status(400).json({ success: false, error: 'NO_SUCCESS_CRITERIA: task has no @success_criteria block, so it cannot be validated. Re-emit the task with explicit criteria.' });
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

    // Fail closed: a pass verdict with no per-criterion evidence is malformed
    // (mirrors parseValidationVerdict in the Go orchestrator). The server must
    // not trust a bare passed=true — that is exactly the fail-open this gate
    // closes.
    if (passed && (!Array.isArray(criteriaResults) || criteriaResults.length === 0)) {
      return res.status(400).json({ success: false, error: 'VERDICT_MISSING_CRITERIA: passed=true requires a non-empty criteriaResults array' });
    }

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

// QA/Synthesizer records the outcome of a validated task's synthesis pass.
// Posted by the Go orchestrator after parseSynthesisResponse resolves the
// QA agent's reply. Writes to ChronLog so the coordination event loop can
// surface it in the TUI and so audit trails show the synthesis-complete
// boundary alongside task_validated.
app.post('/api/tasks/:taskId/synthesis', async (req, res) => {
  try {
    const { taskId } = req.params;
    const { agentId, kind, summary, targetAgent, question } = req.body;
    const task = taskCoordinator.getTask(taskId);
    if (!task) return res.status(404).json({ success: false, error: 'Task not found' });

    const ts = new Date().toISOString();
    if (kind === 'complete') {
      // Stamp the task so /api/tasks/validated stops returning it on the
      // next QA poll. Without this marker the in-memory `alreadySeen` set
      // on the orchestrator is the only guard — wiped on TUI restart,
      // causing re-synthesis loops on every relaunch.
      if (!task.metadata) task.metadata = {};
      task.metadata.synthesizedAt = ts;
      const msg = summary ? `synthesized: ${summary}` : `synthesized ${taskId}`;
      chronologicalLog.append({
        timestamp: ts,
        agent: agentId || 'qa_synthesizer',
        message: msg,
        type: 'synthesis_complete',
        data: { taskId, summary, synthesizedAt: ts }
      });
      io.emit('synthesis_complete', { taskId, summary, timestamp: ts });
    } else if (kind === 'need_clarification') {
      chronologicalLog.append({
        timestamp: ts,
        agent: agentId || 'qa_synthesizer',
        message: `needs clarification from @${targetAgent}: ${question}`,
        type: 'synthesis_needs_clarification',
        data: { taskId, targetAgent, question }
      });
      io.emit('synthesis_needs_clarification', { taskId, targetAgent, question, timestamp: ts });
    } else {
      return res.status(400).json({ success: false, error: `unknown synthesis kind: ${kind}` });
    }

    res.json({ success: true });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// (validated route moved above /:taskId)

// Send an agent message (MCP bridge alternative to socket agent_message)
app.post('/api/messages', async (req, res) => {
  try {
    const { sender, projectName, message } = req.body;
    if (!sender || !projectName || !message) {
      return res.status(400).json({ success: false, error: 'sender, projectName, and message are required' });
    }

    const senderAgent = agentRegistry.getAgent(sender);
    const senderName = senderAgent?.name || sender;

    // Route via EventHub (socket broadcast, classification, rate limiting)
    if (eventHub) {
      await eventHub.handleAgentMessage(sender, senderName, message, new Date().toISOString());
    }

    // Buffer into recipient inbox for MCP agents who can't receive socket events.
    // Only buffer for agents in the SAME project — cross-project mentions are
    // dropped silently under per-project isolation (caller mis-routed).
    const mentionMatch = message.match(/^@(\S+)/);
    if (mentionMatch) {
      const recipientName = mentionMatch[1];
      const sameProjectAgents = agentRegistry.getAllAgents().filter(a => a.projectName === projectName);
      const recipient = sameProjectAgents.find(
        a => a.name.toLowerCase() === recipientName.toLowerCase() || a.id.toLowerCase() === recipientName.toLowerCase()
      );
      if (recipient) {
        bufferMessageForAgent(recipient.id, sender, projectName, message, 'direct_mention');
      }
    } else {
      const sameProjectAgents = agentRegistry.getAllAgents().filter(a => a.projectName === projectName);
      for (const agent of sameProjectAgents) {
        if (agent.id !== sender) {
          bufferMessageForAgent(agent.id, sender, projectName, message, 'broadcast');
        }
      }
    }

    res.json({ success: true });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// Get messages from an agent's inbox. Scoped by ?project=NAME so the same
// agent ID across projects doesn't leak peers' chatter.
app.get('/api/agents/:agentId/messages', (req, res) => {
  const { agentId } = req.params;
  const { since, limit, project } = req.query;

  pruneInbox(agentId);
  let inbox = agentInboxes.get(agentId) || [];

  if (typeof project === 'string' && project) {
    inbox = inbox.filter(m => m.projectName === project);
  }

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
  projectName: string;
  agentId: string;
  taskId: string;
  lockedAt: string;
}

// Keyed by `${projectName} ${filePath}` so the same absolute path can be
// independently locked in two projects under per-project isolation.
const fileLocks = new Map<string, FileLock>();
const lockKey = (projectName: string, filePath: string) => `${projectName} ${filePath}`;

// Claim one or more files for exclusive editing
app.post('/api/files/claim', (req, res) => {
  const { agent_id, task_id, project_name, file_paths } = req.body;
  if (!agent_id || !task_id || !project_name || !Array.isArray(file_paths) || file_paths.length === 0) {
    return res.status(400).json({ success: false, error: 'agent_id, task_id, project_name, and file_paths[] are required' });
  }

  const conflicts: { filePath: string; lockedBy: string; taskId: string }[] = [];
  for (const fp of file_paths) {
    const existing = fileLocks.get(lockKey(project_name, fp));
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
    fileLocks.set(lockKey(project_name, fp), { filePath: fp, projectName: project_name, agentId: agent_id, taskId: task_id, lockedAt: now });
  }

  // Log to ChronologicalLog so PVM captures file ownership patterns
  // and file locks survive server restarts via event replay.
  chronologicalLog.append({
    timestamp: now,
    agent: agent_id,
    message: `claimed files for editing: ${file_paths.join(', ')} (task: ${task_id})`,
    type: 'file_claim',
    data: { filePaths: file_paths, projectName: project_name, agentId: agent_id, taskId: task_id },
  });

  res.json({ success: true, claimed: file_paths });
});

// Release one or more file locks
app.post('/api/files/release', (req, res) => {
  const { agent_id, task_id, project_name, file_paths } = req.body;
  if (!agent_id || !project_name || !Array.isArray(file_paths)) {
    return res.status(400).json({ success: false, error: 'agent_id, project_name, and file_paths[] are required' });
  }

  const released: string[] = [];
  for (const fp of file_paths) {
    const key = lockKey(project_name, fp);
    const lock = fileLocks.get(key);
    if (lock && lock.agentId === agent_id) {
      fileLocks.delete(key);
      released.push(fp);
    }
  }

  if (released.length > 0) {
    chronologicalLog.append({
      timestamp: new Date().toISOString(),
      agent: agent_id,
      message: `released file locks: ${released.join(', ')} (task: ${task_id || 'unknown'})`,
      type: 'file_release',
      data: { filePaths: released, projectName: project_name, agentId: agent_id, taskId: task_id || '' },
    });
  }

  res.json({ success: true, released });
});

// Get current file lock state. Scope with ?project=NAME to see only that
// project's locks; without it returns the whole registry (useful for tooling).
app.get('/api/files/locks', (req, res) => {
  const project = queryProject(req);
  let locks = Array.from(fileLocks.values());
  if (project) {
    locks = locks.filter(l => l.projectName === project);
  }
  res.json({ success: true, locks });
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
// PVM search defaults to cross-project — the original "has anyone ever solved
// X?" use case, useful for the researcher role and for Planner cross-domain
// pattern lookup. Callers can opt into per-project scoping by passing
// ?project=NAME on the query string; scoped queries also include __global__
// events (cross-project infrastructure). See PVMIndexer.extractProjectName
// for the tagging rule.
app.get('/api/pvm/search', async (req, res) => {
  try {
    const { query, limit, project } = req.query;
    if (!query || typeof query !== 'string') {
      return res.status(400).json({ success: false, error: 'Query parameter is required' });
    }
    const limitNum = limit ? parseInt(limit as string) : 10;
    const projectName = typeof project === 'string' && project.length > 0 ? project : undefined;
    const results = await pvmIndexer.search(query, limitNum, projectName);
    res.json({ success: true, results, scope: projectName ?? 'cross-project' });
  } catch (error: any) {
    logger.error(`PVM search failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

app.get('/api/pvm/status', (req, res) => {
  res.json(pvmIndexer.getStatus());
});

// PVM routing brief — confidence-labeled coordination evidence for the Planner:
// which swarm compositions worked on similar past projects, per-role track
// records, and role-pair history. The orchestrator fetches this once at
// decomposition time and injects it into the Planner's BUILD turn so the
// Planner reasons from real outcomes instead of intuition. Always returns a
// block (empty string when there's no history yet — correct for a first run).
app.get('/api/pvm/routing-brief', async (req, res) => {
  try {
    const { description, capabilities } = req.query;
    const desc = typeof description === 'string' ? description : '';
    const caps = typeof capabilities === 'string' && capabilities.length > 0
      ? capabilities.split(',').map(c => c.trim()).filter(Boolean)
      : [];
    const brief = await vectorStore.getRoutingBrief(desc, caps);
    res.json({ success: true, ...brief });
  } catch (error: any) {
    logger.error(`PVM routing-brief failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

// PVM reindex — force a full re-index of the ChronologicalLog into the vector
// store. Useful after a tagging-logic change (e.g. after workflow 02 of the
// PVM-rebuild chain landed) so existing events get re-tagged without a server
// restart. Indexer's existing indexAllEvents() is idempotent (batchStore
// upserts by id), so calling this multiple times is safe.
app.post('/api/pvm/reindex', async (req, res) => {
  try {
    const beforeStatus = pvmIndexer.getStatus();
    logger.info(`🔄 PVM reindex requested (before: indexedEventCount=${beforeStatus.indexedEventCount})`);
    await pvmIndexer.indexAllEvents();
    const afterStatus = pvmIndexer.getStatus();
    logger.info(`✅ PVM reindex complete (after: indexedEventCount=${afterStatus.indexedEventCount})`);
    res.json({
      success: true,
      before: beforeStatus,
      after: afterStatus,
      delta: afterStatus.indexedEventCount - beforeStatus.indexedEventCount
    });
  } catch (error: any) {
    logger.error(`PVM reindex failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

// PVM agent profile — returns derived skill metrics for one agent. Source of
// data: lookupTaskOutcomes inside LocalEmbeddingVectorStore.getAgentProfile,
// which joins task_assigned / task_completed / task_validated events by taskId.
app.get('/api/pvm/profile', async (req, res) => {
  try {
    const { agentId } = req.query;
    if (!agentId || typeof agentId !== 'string') {
      return res.status(400).json({ success: false, error: 'agentId query parameter is required' });
    }
    const profile = await vectorStore.getAgentProfile(agentId);
    res.json({ success: true, profile });
  } catch (error: any) {
    logger.error(`PVM profile failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

// PVM agent synergy — collaboration metrics for an agent pair.
app.get('/api/pvm/synergy', async (req, res) => {
  try {
    const { agent1, agent2 } = req.query;
    if (!agent1 || !agent2 || typeof agent1 !== 'string' || typeof agent2 !== 'string') {
      return res.status(400).json({ success: false, error: 'agent1 and agent2 query parameters are required' });
    }
    const synergy = await vectorStore.getAgentSynergy(agent1, agent2);
    res.json({ success: true, synergy });
  } catch (error: any) {
    logger.error(`PVM synergy failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

// PVM agent compare — ranks a set of agents for a given task type.
app.get('/api/pvm/compare', async (req, res) => {
  try {
    const { agents, taskType } = req.query;
    if (!agents || typeof agents !== 'string') {
      return res.status(400).json({ success: false, error: 'agents query parameter is required (comma-separated)' });
    }
    if (!taskType || typeof taskType !== 'string') {
      return res.status(400).json({ success: false, error: 'taskType query parameter is required' });
    }
    const agentIds = agents.split(',').map(s => s.trim()).filter(s => s.length > 0);
    const comparison = await vectorStore.compareAgents(agentIds, taskType);
    res.json({ success: true, comparison });
  } catch (error: any) {
    logger.error(`PVM compare failed: ${error.message}`);
    res.status(500).json({ success: false, error: error.message });
  }
});

// Read raw ChronologicalLog — used by import project + Tier 1 agents via
// act_cli log. Accepts ?project=NAME to filter to events scoped to that
// project (task lifecycle + project-tagged messages). Without the filter,
// agents see the global stream and surface old-project activity.
app.get('/api/log', async (req, res) => {
  try {
    // Accept both ?project= and ?projectName= — callers (Tier 1 agents, act_cli,
    // E2E harness) use the latter to mirror the field name in event payloads.
    const projectParam = typeof req.query.projectName === 'string'
      ? req.query.projectName
      : typeof req.query.project === 'string' ? req.query.project : '';
    const rawLimit = parseInt(req.query.limit as string) || 500;
    // When a project is specified, read directly from the per-project JSONL
    // file (W08 split) instead of filtering the global stream in-memory.
    // The global stream contains all projects, so a post-fetch filter
    // a) bounded the result by a global window, and b) missed events whose
    // projectName lived on a field not in the hand-written allowlist.
    const events = projectParam
      ? await chronologicalLog.getRecent(rawLimit, projectParam)
      : await chronologicalLog.getRecent(rawLimit);
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
      const { agentId, projectName, capabilities, name, model, provider } = data;
      console.log(`🤖 AGENT REGISTRATION: ${agentId} (project=${projectName}) with capabilities: ${capabilities?.join(', ')}`);

      if (!projectName) {
        socket.emit('agent_registered', { success: false, agentId, error: 'projectName is required' });
        return;
      }

      await agentRegistry.registerAgent(agentId, {
        name: name || agentId,
        projectName,
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
          projectName,
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
import { writeFileSync, unlinkSync, existsSync, mkdirSync, readFileSync } from 'fs';
// __dirname equivalent for ESM: path.dirname(import.meta.url) isn't needed
// because the server's data dir is always relative to the server root (one
// level above src/). Use the same path the ChronologicalLog uses.
const SERVER_DATA_DIR = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'data');
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

  // Refuse to start if another live ACT server already holds this PID file.
  // Prevents zombie-tsx stacking: when SIGKILL leaves a stale PID file behind,
  // an honest restart still works (signal 0 to a dead PID throws); but if the
  // earlier process is still breathing, we exit instead of fighting for port 8080.
  if (existsSync(PID_FILE)) {
    try {
      const otherPid = parseInt(readFileSync(PID_FILE, 'utf-8').trim(), 10);
      if (otherPid > 0 && otherPid !== process.pid) {
        try {
          process.kill(otherPid, 0); // throws if dead
          logger.error(`ACT server already running (pid=${otherPid}). Exiting to avoid duplicate.`);
          process.exit(1);
        } catch {
          // Stale PID file — previous process gone. Continue.
          logger.warn(`Stale PID file pointed to dead pid=${otherPid}; reclaiming.`);
        }
      }
    } catch (e: any) {
      logger.warn(`Could not parse PID file: ${e.message}; reclaiming.`);
    }
  }

  // Write PID file and start listening only after state is restored
  writePidFile();
  server.listen(PORT, () => {
    logger.info(`ACT Server running on port ${PORT}`);
    logger.info(`WebSocket: ws://localhost:${PORT}`);
  });

  // Sweep stale agents every 60s. AgentRegistry.performHealthCheck flips any
  // online/busy agent whose lastSeen is more than 5 minutes old to "offline"
  // — without this, agents from prior sessions (or crashed runners) linger as
  // status=online and the Observer flags them as idle for tasks they can't take.
  setInterval(() => {
    agentRegistry.performHealthCheck().catch(err => {
      logger.warn(`Agent health check failed: ${err.message}`);
    });
  }, 60_000);
}).catch(err => {
  logger.error(`Failed to initialize ChronologicalLog: ${err.message}`);
  process.exit(1);
});

export { app, io, agentRegistry, taskCoordinator, eventHub };