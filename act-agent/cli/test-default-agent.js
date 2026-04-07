#!/usr/bin/env node

// Test script for REPL default agent functionality
import { SessionManager } from './dist/session-manager.js';
import { ACTClient } from './dist/act-client.js';
import fs from 'fs';
import path from 'path';

async function testDefaultAgentFunctionality() {
  console.log('🧪 Testing REPL Default Agent Functionality\n');

  // Create client and session manager
  const client = new ACTClient('http://localhost:8080');
  const sessionManager = new SessionManager(client);

  try {
    // Test 1: List agents
    console.log('📋 Test 1: Listing available agents...');
    const agents = await client.getAgents();
    console.log(`Found ${agents.length} agents:`, agents.map(a => a.id));

    if (agents.length === 0) {
      console.log('❌ No agents available for testing');
      return;
    }

    // Test 2: Set valid default agent
    console.log('\n✅ Test 2: Setting valid default agent...');
    const testAgentId = agents[0].id;
    const success = await sessionManager.setDefaultAgent(testAgentId);

    if (success) {
      console.log(`✓ Successfully set default agent to: ${testAgentId}`);
    } else {
      console.log('❌ Failed to set default agent');
      return;
    }

    // Test 3: Verify persistence
    console.log('\n💾 Test 3: Verifying persistence...');
    const retrievedAgent = sessionManager.getDefaultAgent();
    if (retrievedAgent === testAgentId) {
      console.log(`✓ Default agent persisted: ${retrievedAgent}`);
    } else {
      console.log(`❌ Persistence failed. Expected: ${testAgentId}, Got: ${retrievedAgent}`);
      return;
    }

    // Test 4: Check config file creation
    console.log('\n📁 Test 4: Checking config file creation...');
    const homeDir = process.env.HOME || process.env.USERPROFILE || '';
    const configPath = path.join(homeDir, '.act', 'repl-config.json');

    if (fs.existsSync(configPath)) {
      console.log('✓ Config file created successfully');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf-8'));
      if (config.defaultAgent === testAgentId) {
        console.log('✓ Config file contains correct default agent');
      } else {
        console.log(`❌ Config file has wrong agent: ${config.defaultAgent}`);
        return;
      }
    } else {
      console.log('❌ Config file not created');
      return;
    }

    // Test 5: Test invalid agent
    console.log('\n🚫 Test 5: Testing invalid agent validation...');
    const invalidSuccess = await sessionManager.setDefaultAgent('nonexistent_agent_12345');
    if (!invalidSuccess) {
      console.log('✓ Correctly rejected invalid agent');
    } else {
      console.log('❌ Incorrectly accepted invalid agent');
      return;
    }

    console.log('\n🎉 ALL TESTS PASSED! Default agent functionality is working correctly.');

  } catch (error) {
    console.error('❌ Test failed with error:', error.message);
  }
}

testDefaultAgentFunctionality();
    }

    // Test 2: Set valid default agent
    console.log('\n✅ Test 2: Setting valid default agent...');
    const testAgentId = agents[0].id;
    const success = await sessionManager.setDefaultAgent(testAgentId);

    if (success) {
      console.log(`✓ Successfully set default agent to: ${testAgentId}`);
    } else {
      console.log('❌ Failed to set default agent');
      return;
    }

    // Test 3: Verify persistence
    console.log('\n💾 Test 3: Verifying persistence...');
    const retrievedAgent = sessionManager.getDefaultAgent();
    if (retrievedAgent === testAgentId) {
      console.log(`✓ Default agent persisted: ${retrievedAgent}`);
    } else {
      console.log(`❌ Persistence failed. Expected: ${testAgentId}, Got: ${retrievedAgent}`);
      return;
    }

    // Test 4: Check config file creation
    console.log('\n📁 Test 4: Checking config file creation...');
    const fs = require('fs');
    const path = require('path');
    const homeDir = process.env.HOME || process.env.USERPROFILE || '';
    const configPath = path.join(homeDir, '.act', 'repl-config.json');

    if (fs.existsSync(configPath)) {
      console.log('✓ Config file created successfully');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf-8'));
      if (config.defaultAgent === testAgentId) {
        console.log('✓ Config file contains correct default agent');
      } else {
        console.log(`❌ Config file has wrong agent: ${config.defaultAgent}`);
        return;
      }
    } else {
      console.log('❌ Config file not created');
      return;
    }

    // Test 5: Test invalid agent
    console.log('\n🚫 Test 5: Testing invalid agent validation...');
    const invalidSuccess = await sessionManager.setDefaultAgent('nonexistent_agent_12345');
    if (!invalidSuccess) {
      console.log('✓ Correctly rejected invalid agent');
    } else {
      console.log('❌ Incorrectly accepted invalid agent');
      return;
    }

    console.log('\n🎉 ALL TESTS PASSED! Default agent functionality is working correctly.');

  } catch (error) {
    console.error('❌ Test failed with error:', error.message);
  }
}

testDefaultAgentFunctionality();
