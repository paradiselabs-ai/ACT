#!/usr/bin/env python3
"""
Simple ACT Agent - Standalone Demo

This demonstrates a Python agent connecting directly to ACT server
for autonomous coordination without any platform integration.

Usage: python simple_agent.py [agent_name] [capabilities...]
Example: python simple_agent.py "CodeBot" "python" "debugging" "testing"
"""

import asyncio
import sys
import os
import json
from datetime import datetime

# Add SDK to path
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'sdk', 'python'))
from act_client import ACTClient

class SimpleAgent:
    def __init__(self, name, capabilities):
        self.name = name
        self.capabilities = capabilities
        self.agent_id = name.lower().replace(' ', '_')
        self.client = ACTClient(
            agent_id=self.agent_id,
            capabilities=capabilities,
            agent_name=name
        )
        self.tasks_completed = 0

    async def connect_and_run(self):
        """Connect to ACT server and start autonomous coordination"""
        print(f"🤖 {self.name} starting up...")
        print(f"📋 Capabilities: {', '.join(self.capabilities)}")
        print(f"🔗 Connecting to ACT Server at ws://localhost:8080")

        try:
            # Connect to ACT server
            await self.client.connect()
            print(f"✅ Connected and registered with ACT server!")

            # Set up task handler
            self.client.on('task_assigned', self.handle_task_assignment)

            # Keep running and handling tasks
            print(f"💫 Ready for autonomous coordination!")
            print(f"⚡ Waiting for task assignments from ACT server...")
            print(f"🔥 Press Ctrl+C to stop")

            while True:
                await asyncio.sleep(1)

        except KeyboardInterrupt:
            print(f"\\n🛑 {self.name} shutting down...")
            await self.client.disconnect()
            print(f"👋 Disconnected. Tasks completed: {self.tasks_completed}")
        except Exception as e:
            print(f"❌ Error: {e}")
            await self.client.disconnect()

    async def handle_task_assignment(self, data):
        """Handle task assignments from ACT server"""
        task = data.get('task', {})
        task_id = task.get('id')
        description = task.get('description', 'Unknown task')

        print(f"\\n🎯 TASK ASSIGNED: {description}")
        print(f"📝 Task ID: {task_id}")

        # Simulate working on the task
        print(f"🔄 Working on task...")

        # Update progress
        for progress in [25, 50, 75, 100]:
            await asyncio.sleep(1)
            await self.client.update_task_progress(task_id, progress)
            print(f"📈 Progress: {progress}%")

        print(f"✅ Task completed!")
        self.tasks_completed += 1
        print(f"📊 Total tasks completed: {self.tasks_completed}")
        print(f"⚡ Ready for next task...")

async def create_demo_task():
    """Create a demo task for testing"""
    demo_client = ACTClient(
        agent_id="task_creator",
        capabilities=["task_creation"],
        agent_name="Task Creator"
    )
    await demo_client.connect()

    task_id = await demo_client.create_task(
        description="Demo task for standalone agent",
        required_capabilities=["python"],
        priority="medium"
    )

    print(f"📝 Demo task created: {task_id}")
    await demo_client.disconnect()

async def main():
    # Parse command line arguments
    if len(sys.argv) < 2:
        agent_name = "SimpleBot"
        capabilities = ["python", "demo"]
    else:
        agent_name = sys.argv[1]
        capabilities = sys.argv[2:] if len(sys.argv) > 2 else ["python", "demo"]

    # Create and run agent
    agent = SimpleAgent(agent_name, capabilities)

    # Create demo task in 3 seconds
    asyncio.create_task(delayed_task_creation())

    # Run agent
    await agent.connect_and_run()

async def delayed_task_creation():
    """Create a demo task after 3 seconds"""
    await asyncio.sleep(3)
    print("\\n📋 Creating demo task...")
    await create_demo_task()

if __name__ == "__main__":
    print("🚀 ACT Standalone Agent Demo")
    print("=" * 40)
    asyncio.run(main())