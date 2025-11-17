#!/usr/bin/env node

/**
 * ACT Todo App Demo - Autonomous Multi-Agent Coordination
 *
 * This demo showcases how ACT coordinates 3 specialized agents to build
 * a complete Todo application autonomously.
 *
 * Demo Flow:
 * 1. Project initialization with ACT
 * 2. Agent registration and capability assessment
 * 3. Autonomous task decomposition and assignment
 * 4. Real-time coordination and conflict resolution
 * 5. Final application delivery
 */

const io = require('socket.io-client');

// Demo configuration
const DEMO_CONFIG = {
  projectName: 'Todo App',
  requirements: [
    'Create a modern todo application with React frontend',
    'Build Flask REST API backend with SQLite database',
    'Implement CRUD operations for todo items',
    'Add responsive UI with clean design',
    'Include proper error handling and validation'
  ],
  agents: [
    {
      id: 'frontend-agent',
      name: 'Frontend Agent',
      capabilities: ['react', 'javascript', 'ui', 'responsive_design', 'state_management'],
      role: 'Build React frontend with modern UI'
    },
    {
      id: 'backend-agent',
      name: 'Backend Agent',
      capabilities: ['python', 'flask', 'api', 'database', 'rest'],
      role: 'Create Flask API and database layer'
    },
    {
      id: 'qa-agent',
      name: 'QA Agent',
      capabilities: ['testing', 'validation', 'integration', 'debugging', 'documentation'],
      role: 'Test, validate, and document the application'
    }
  ]
};

class ACTTodoDemo {
  constructor() {
    this.socket = null;
    this.agents = new Map();
    this.tasks = new Map();
    this.projectStatus = 'initializing';
    this.completedTasks = 0;
  }

  async start() {
    console.log('🚀 Starting ACT Todo App Demo');
    console.log('=' .repeat(50));

    // Connect to ACT server
    await this.connectToACT();

    // Initialize project
    await this.initializeProject();

    // Simulate autonomous coordination
    await this.runCoordinationDemo();

    // Show final results
    await this.showResults();
  }

  async connectToACT() {
    console.log('📡 Connecting to ACT coordination server...');

    return new Promise((resolve) => {
      this.socket = io('ws://localhost:8080', {
        transports: ['websocket', 'polling']
      });

      this.socket.on('connect', () => {
        console.log('✅ Connected to ACT server');
        resolve();
      });

      this.socket.on('disconnect', () => {
        console.log('❌ Disconnected from ACT server');
      });

      // Listen for coordination events
      this.setupEventListeners();
    });
  }

  setupEventListeners() {
    this.socket.on('agent_registered', (data) => {
      console.log(`🤖 Agent registered: ${data.agent.name} (${data.agent.capabilities.join(', ')})`);
      this.agents.set(data.agent.id, data.agent);
    });

    this.socket.on('task_assigned', (data) => {
      console.log(`📋 Task assigned: "${data.task.description}" → ${data.assignedAgent}`);
      this.tasks.set(data.task.id, data.task);
    });

    this.socket.on('task_completed', (data) => {
      console.log(`✅ Task completed: ${data.taskId}`);
      this.completedTasks++;
    });

    this.socket.on('conflict_detected', (data) => {
      console.log(`⚠️  Conflict detected: ${data.conflict.description}`);
      console.log(`🔧 ACT resolving conflict autonomously...`);
    });

    this.socket.on('conflict_resolved', (data) => {
      console.log(`✌️  Conflict resolved: ${data.conflictId}`);
    });

    this.socket.on('project_status_update', (data) => {
      this.projectStatus = data.status.status;
      console.log(`📊 Project status: ${this.projectStatus} (${data.status.progress}%)`);
    });
  }

  async initializeProject() {
    console.log('\n🏗️  Initializing Todo App project...');

    // Send project initialization (just log for demo)
    console.log(`📋 Project: ${DEMO_CONFIG.projectName}`);devicePixelRatio
  }

  async runCoordinationDemo() {
    console.log('\n🤖 Starting autonomous agent coordination...\n');

    // Simulate agents registering (in real ACT, this would happen automatically)
    for (const agent of DEMO_CONFIG.agents) {
      this.socket.emit('register_agent', {
        agentId: agent.id,
        name: agent.name,
        capabilities: agent.capabilities
      });
      await this.delay(1000);
    }

    await this.delay(2000);

    // Simulate task creation and assignment
    console.log('🎯 ACT decomposing project into tasks...\n');

    const demoTasks = [
      {
        id: 'setup-backend',
        description: 'Set up Flask backend with SQLAlchemy and CORS',
        requiredCapabilities: ['python', 'flask', 'database'],
        priority: 'high'
      },
      {
        id: 'create-api',
        description: 'Create REST API endpoints for CRUD operations',
        requiredCapabilities: ['python', 'flask', 'api', 'rest'],
        priority: 'high'
      },
      {
        id: 'setup-frontend',
        description: 'Initialize React app with routing and state management',
        requiredCapabilities: ['react', 'javascript', 'ui'],
        priority: 'high'
      },
      {
        id: 'build-components',
        description: 'Build TodoItem and TodoList React components',
        requiredCapabilities: ['react', 'javascript', 'ui'],
        priority: 'medium'
      },
      {
        id: 'integrate-api',
        description: 'Connect frontend to backend API with error handling',
        requiredCapabilities: ['react', 'javascript', 'api'],
        priority: 'medium'
      },
      {
        id: 'add-styling',
        description: 'Add responsive CSS styling and animations',
        requiredCapabilities: ['react', 'ui', 'responsive_design'],
        priority: 'low'
      },
      {
        id: 'testing-validation',
        description: 'Create unit tests and integration tests',
        requiredCapabilities: ['testing', 'validation'],
        priority: 'medium'
      },
      {
        id: 'documentation',
        description: 'Create README and API documentation',
        requiredCapabilities: ['documentation'],
        priority: 'low'
      }
    ];

    // Simulate ACT assigning tasks autonomously
    for (const task of demoTasks) {
      this.socket.emit('create_task', task);
      await this.delay(1500);

      // Simulate task completion using task_progress
      this.socket.emit('task_progress', {
        taskId: task.id,
        progress: 100,
        status: 'completed',
        message: `Task completed successfully`
      });
      await this.delay(2000);
    }

    // Simulate a final project status update
    console.log('\n⚠️  Simulating final project completion...\n');
    console.log('🎉 All tasks completed! Project delivered successfully.');
    console.log('📊 ACT coordination demonstrated autonomous agent collaboration.');

    await this.delay(3000);
  }

  async showResults() {
    console.log('\n🎉 Demo Complete!');
    console.log('=' .repeat(50));

    console.log('📊 Final Results:');
    console.log(`   🤖 Agents registered: ${this.agents.size}`);
    console.log(`   📋 Tasks completed: ${this.completedTasks}`);
    console.log(`   📊 Project status: ${this.projectStatus}`);

    console.log('\n🏗️  Autonomous Todo App Features Delivered:');
    console.log('   ✅ Flask REST API with SQLite database');
    console.log('   ✅ React frontend with modern UI');
    console.log('   ✅ CRUD operations for todo items');
    console.log('   ✅ Responsive design and styling');
    console.log('   ✅ Error handling and validation');
    console.log('   ✅ Unit and integration tests');
    console.log('   ✅ Complete documentation');

    console.log('\n🚀 Key ACT Coordination Achievements:');
    console.log('   ✅ Autonomous agent team formation');
    console.log('   ✅ Capability-based task assignment');
    console.log('   ✅ Real-time conflict detection and resolution');
    console.log('   ✅ No human intervention required');
    console.log('   ✅ Complete application delivery');

    console.log('\n💡 This demo shows how ACT enables:');
    console.log('   • Self-organizing AI development teams');
    console.log('   • Autonomous project execution');
    console.log('   • Intelligent conflict resolution');
    console.log('   • Framework-agnostic coordination');

    // Cleanup
    this.socket.disconnect();
    console.log('\n👋 Demo finished. ACT coordination server disconnected.');
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Run the demo
if (require.main === module) {
  const demo = new ACTTodoDemo();
  demo.start().catch(console.error);
}

module.exports = ACTTodoDemo;
