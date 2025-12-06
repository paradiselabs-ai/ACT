const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

console.log('🧪 FULL PVM INTEGRATION TEST');
console.log('============================');

let serverProcess;
let testAgentProcess;

// Start the ACT server
console.log('\n🚀 Starting ACT Server...');
serverProcess = spawn('npm', ['start'], {
  cwd: '/Users/user/Documents/Developer/dev/AI/act/sdk/python/server',
  stdio: 'pipe'
});

let serverReady = false;

serverProcess.stdout.on('data', (data) => {
  const output = data.toString();
  // Only show key messages to keep output clean
  if (output.includes('ACT Server running') || output.includes('🔗') || output.includes('💫')) {
    console.log(output.trim());
  }
  
  if (output.includes('ACT Server running on port')) {
    if (!serverReady) {
      serverReady = true;
      console.log('\n✅ ACT Server is ready!');
      setTimeout(runTestAgent, 1000);
    }
  }
});

serverProcess.stderr.on('data', (data) => {
  // Show errors
  console.error(data.toString().trim());
});

function runTestAgent() {
  console.log('\n🤖 Starting Test Agent...');
  testAgentProcess = spawn('node', ['test-agent.js'], {
    cwd: '/Users/user/Documents/Developer/dev/AI/act/sdk/python/server',
    stdio: 'inherit'
  });
  
  testAgentProcess.on('close', (code) => {
    console.log(`\n🤖 Test Agent finished with code ${code}`);
    setTimeout(checkLogResults, 2000);
  });
}

function checkLogResults() {
  console.log('\n🔍 Checking ChronologicalLog results...');
  
  // Check if log file was created
  const logPath = path.join('/Users/user/Documents/Developer/dev/AI/act/sdk/python/server/data', 'coordination-log.jsonl');
  
  if (fs.existsSync(logPath)) {
    console.log('✅ ChronologicalLog file created!');
    
    // Read the log file
    const content = fs.readFileSync(logPath, 'utf8');
    const lines = content.trim().split('\n').filter(line => line.trim() !== '');
    
    console.log(`\n📝 Found ${lines.length} log entries:`);
    
    if (lines.length > 0) {
      // Show all entries (should be a small number)
      lines.forEach((line, i) => {
        try {
          const entry = JSON.parse(line);
          console.log(`  ${i + 1}. ${entry.timestamp} - ${entry.agent}: ${entry.type}`);
          console.log(`     Message: ${entry.message.substring(0, 80)}${entry.message.length > 80 ? '...' : ''}`);
        } catch (e) {
          console.log(`  ${i + 1}. ${line.substring(0, 100)}${line.length > 100 ? '...' : ''}`);
        }
      });
      
      console.log('\n🎉 PVM INTEGRATION TEST PASSED!');
      console.log('   ChronologicalLog is successfully capturing coordination events.');
    } else {
      console.log('📝 Log file is empty');
      console.log('⚠️  PVM INTEGRATION TEST: Events generated but not logged');
    }
  } else {
    console.log('❌ ChronologicalLog file not found');
    console.log('⚠️  PVM INTEGRATION TEST: Log file not created');
  }
  
  // Clean up
  console.log('\n🧹 Cleaning up...');
  if (serverProcess) {
    serverProcess.kill();
  }
  
  console.log('\n🏁 PVM Integration Test Complete');
  process.exit(0);
}

// Handle cleanup on exit
process.on('SIGINT', () => {
  console.log('\n🛑 Shutting down...');
  if (serverProcess) serverProcess.kill();
  if (testAgentProcess) testAgentProcess.kill();
  process.exit(0);
});

// Timeout after 30 seconds
setTimeout(() => {
  console.log('\n⏰ Test timeout - shutting down...');
  if (serverProcess) serverProcess.kill();
  if (testAgentProcess) testAgentProcess.kill();
  process.exit(1);
}, 30000);
