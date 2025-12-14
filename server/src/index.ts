import express from 'express';
import path from 'path';
import { createServer } from 'http';
import { Server } from 'socket.io';
import cors from 'cors';
import { AgentRegistry } from './services/AgentRegistry';
import { Task } from './services/TaskCoordinator';
import { TaskCoordinator } from './services/TaskCoordinator';
import { EventHub } from './services/EventHub';
import { SelfImprovementEngine } from './services/SelfImprovementEngine';
import { ChronologicalLog } from './services/ChronologicalLog';
import { MockVectorStore } from './services/MockVectorStore';
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
const vectorStore = new MockVectorStore();
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

app.get('/api/tasks', (req, res) => {
  res.json(taskCoordinator.getAllTasks());
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

      // Log the message to ChronologicalLog for PVM
      await chronologicalLog.append({
        timestamp: timestamp || new Date().toISOString(),
        agent: sender,
        message: message,
        type: 'coordination' // Use generic coordination type for now
      });

      // Broadcast original message to all other agents
      socket.broadcast.emit('agent_message', {
        sender,
        message,
        timestamp: timestamp || new Date().toISOString()
      });

      // ACT INTELLIGENT COORDINATION: Decide if/how other agents should respond
      const messageId = `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      logger.info(`🤖 COORDINATION START: Calling eventHub.coordinateAgentCommunication for sender: ${sender}`);
      logger.info(`🤖 COORDINATION DEBUG: eventHub exists: ${!!eventHub}`);
      if (eventHub) {
        await eventHub.coordinateAgentCommunication(sender, message, messageId);
        logger.info(`🤖 COORDINATED: Agent ${sender} message processed, coordination evaluated`);
      } else {
        logger.error(`🤖 COORDINATION ERROR: eventHub is undefined!`);
      }
    } catch (error: any) {
      logger.error(`Agent message coordination failed: ${error.message}`);
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