const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

console.log('🧪 Testing PVM Integration - ChronologicalLog');

// Start the ACT server
const server = spawn('npm', ['start'], {
  cwd: '/Users/user/Documents/Developer/dev/AI/act/sdk/python/server',
  stdio: 'pipe'
});

let serverReady = false;

server.stdout.on('data', (data) => {
  const output = data.toString();
  process.stdout.write(output);
  
  if (output.includes('ACT Server running on port')) {
    serverReady = true;
    console.log('\n✅ ACT Server is ready!');
    testCoordinationEvents();
  }
});

server.stderr.on('data', (data) => {
  process.stderr.write(data);
});

// Test coordination events after server is ready
function testCoordinationEvents() {
  console.log('\n🚀 Testing coordination event logging...');
  
  // Give the server a moment to fully initialize
  setTimeout(() => {
    // Check if log file was created
    const logPath = path.join('/Users/user/Documents/Developer/dev/AI/act/sdk/python/server/data', 'coordination-log.jsonl');
    
    if (fs.existsSync(logPath)) {
      console.log('✅ ChronologicalLog file created!');
      
      // Read the last few lines of the log
      const content = fs.readFileSync(logPath, 'utf8');
      const lines = content.trim().split('\n');
      
      if (lines.length > 0) {
        console.log(`📝 Found ${lines.length} log entries:`);
        // Show last 3 entries
        lines.slice(-3).forEach((line, i) => {
          try {
            const entry = JSON.parse(line);
            console.log(`  ${lines.length - 3 + i + 1}. ${entry.timestamp} - ${entry.agent}: ${entry.message.substring(0, 100)}${entry.message.length > 100 ? '...' : ''}`);
          } catch (e) {
            console.log(`  ${lines.length - 3 + i + 1}. ${line.substring(0, 100)}${line.length > 100 ? '...' : ''}`);
          }
        });
      } else {
        console.log('📝 Log file is empty');
      }
    } else {
      console.log('❌ ChronologicalLog file not found');
    }
    
    // Clean up
    console.log('\n🧹 Cleaning up...');
    server.kill();
    
  }, 3000);
}

// Handle cleanup on exit
process.on('SIGINT', () => {
  console.log('\n🛑 Shutting down...');
  server.kill();
  process.exit(0);
});

setTimeout(() => {
  if (!serverReady) {
    console.log('❌ Server failed to start in time');
    server.kill();
    process.exit(1);
  }
}, 10000);
