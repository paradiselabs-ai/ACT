const { SelfImprovementEngine } = require('./dist/improvement/SelfImprovementEngine.js');

// Create and test the self-improvement engine
console.log('🧪 Testing Self-Improvement Engine connection to ACT server...');

const engine = new SelfImprovementEngine();

// Wait a few seconds to see if it connects
setTimeout(() => {
  const status = engine.getStatus();
  console.log('📊 Engine Status:', status);
  
  if (status.isConnected) {
    console.log('✅ SUCCESS: Self-Improvement Engine connected to ACT server!');
  } else {
    console.log('❌ FAILED: Self-Improvement Engine could not connect to ACT server');
  }
  
  // Keep running for a bit to collect events
  setTimeout(() => {
    console.log('📝 Event History Size:', engine.getStatus().eventHistorySize);
    process.exit(0);
  }, 5000);
}, 3000);
