const io = require('socket.io-client');

console.log('🤖 Test Agent Connecting to ACT Server...');

// Connect to ACT server
const socket = io('http://localhost:8080', {
  transports: ['websocket']
});

socket.on('connect', () => {
  console.log('✅ Connected to ACT Server');
  
  // Register as test agent
  socket.emit('register_agent', {
    agentId: 'test_pvm_agent',
    name: 'PVM Test Agent',
    capabilities: ['testing', 'pvm_integration']
  });
  
  // Create a test task
  setTimeout(() => {
    console.log('📝 Creating test task...');
    socket.emit('create_task', {
      description: 'Test task for PVM integration',
      requiredCapabilities: ['testing'],
      priority: 'medium'
    });
  }, 1000);
  
  // Send a test message
  setTimeout(() => {
    console.log('💬 Sending test message...');
    socket.emit('agent_message', {
      sender: 'test_pvm_agent',
      recipient: 'system',
      content: 'This is a test message for PVM logging',
      context: 'pvm_test'
    });
  }, 2000);
  
  // Update status
  setTimeout(() => {
    console.log('🔄 Updating agent status...');
    socket.emit('agent_status', {
      agentId: 'test_pvm_agent',
      status: 'busy',
      currentTask: 'pvm_test_task'
    });
  }, 3000);
});

socket.on('agent_registered', (data) => {
  console.log('🎉 Agent registered:', data);
});

socket.on('task_created', (data) => {
  console.log('🎯 Task created:', data?.task?.description);
});

socket.on('task_assigned', (data) => {
  console.log('📌 Task assigned:', data);
});

socket.on('disconnect', () => {
  console.log('🔌 Disconnected from ACT Server');
});

// Run for 5 seconds then exit
setTimeout(() => {
  console.log('🛑 Test agent shutting down...');
  socket.disconnect();
  process.exit(0);
}, 5000);
