import express from 'express';
import { createServer } from 'http';
import { Server } from 'socket.io';
import cors from 'cors';
import { AgentRegistry } from './services/AgentRegistry';
import { TaskCoordinator } from './services/TaskCoordinator';
import { EventHub } from './services/EventHub';
import { logger } from './utils/logger';

const app = express();
const server = createServer(app);
const io = new Server(server, {
  cors: {
    origin: ["http://localhost:3000", "http://localhost:3001", "http://localhost:5173", "http://localhost:5000"],
    methods: ["GET", "POST"]
  }
});

// Middleware
app.use(cors());
app.use(express.json());

// Core ACT Services
const agentRegistry = new AgentRegistry();
const taskCoordinator = new TaskCoordinator(agentRegistry);
const eventHub = new EventHub(io, agentRegistry, taskCoordinator);

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

// WebSocket connection handling
io.on('connection', (socket) => {
  logger.info(`🔗 Client connected: ${socket.id}`);
  console.log(`🔗 NEW CONNECTION: ${socket.id} at ${new Date().toLocaleTimeString()}`);

  // Agent registration
  socket.on('register_agent', async (data) => {
    try {
      const { agentId, capabilities, name } = data;
      console.log(`🤖 AGENT REGISTRATION: ${agentId} with capabilities: ${capabilities?.join(', ')}`);

      await agentRegistry.registerAgent(agentId, {
        name: name || agentId,
        capabilities: capabilities || [],
        socketId: socket.id,
        status: 'online'
      });

      socket.emit('agent_registered', { success: true, agentId });

      // Broadcast to all clients (especially Windsurf's dashboard!)
      io.emit('agent_joined', {
        agentId,
        name: name || agentId,
        capabilities,
        timestamp: new Date().toISOString()
      });

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

        logger.info(`Task ${task.id} assigned to ${assignment.agentId}`);
      } else {
        console.log(`⏳ TASK PENDING: ${task.id} - No suitable agent available`);
        io.emit('task_pending', {
          taskId: task.id,
          task: task,
          reason: 'No suitable agent available',
          timestamp: new Date().toISOString()
        });
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
      console.log(`📊 TASK PROGRESS: ${taskId} → ${progress}% ${status ? '- ' + status : ''}`);

      await taskCoordinator.updateTaskProgress(taskId, { progress, status, message });

      io.emit('task_progress', {
        taskId,
        progress,
        status,
        message,
        timestamp: new Date().toISOString()
      });

      logger.info(`Task ${taskId} progress: ${progress}% - ${status}`);
    } catch (error: any) {
      logger.error(`Task progress update failed: ${error.message}`);
    }
  });

  // Task progress updates (alternative handler name)
  socket.on('update_task_progress', async (data) => {
    try {
      const { taskId, progress, status, message, agentId } = data;
      await taskCoordinator.updateTaskProgress(taskId, { progress, status, message });

      io.emit('task_progress', {
        taskId,
        progress,
        status: status || `${progress}% complete`,
        message,
        timestamp: new Date().toISOString()
      });

      logger.info(`Task ${taskId} progress from ${agentId}: ${progress}%${status ? ' - ' + status : ''}`);
    } catch (error: any) {
      logger.error(`Task progress update failed: ${error.message}`);
    }
  });

  // Agent-to-agent messaging
  socket.on('agent_message', async (data) => {
    try {
      const { sender, message, timestamp } = data;

      // Broadcast message to all connected agents
      socket.broadcast.emit('agent_message', {
        sender,
        message,
        timestamp: timestamp || new Date().toISOString()
      });

      logger.info(`Agent message from ${sender}: ${message.substring(0, 100)}${message.length > 100 ? '...' : ''}`);
    } catch (error: any) {
      logger.error(`Agent message failed: ${error.message}`);
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
    const allTasks = taskCoordinator.getAllTasks();
    const totalTasks = allTasks.length;
    const completedTasks = allTasks.filter(task => task.status === 'completed').length;
    const progress = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;

    const projectStatus = {
      status: totalTasks === 0 ? 'initializing' : completedTasks === totalTasks ? 'completed' : 'active',
      progress: progress,
      activeAgents: agentRegistry.getOnlineAgentCount(),
      totalTasks: totalTasks,
      completedTasks: completedTasks
    };

    socket.emit('project_status_update', { status: projectStatus });
    logger.info(`Sent project status: ${progress}% complete, ${projectStatus.activeAgents} agents online`);
  });

  socket.on('get_agent_registry', () => {
    const agents = agentRegistry.getAllAgents();
    // Emit agent_registered for each agent to update dashboard
    agents.forEach(agent => {
      socket.emit('agent_registered', { agent });
    });
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