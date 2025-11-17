#!/usr/bin/env python3
"""
Socket.IO ACT Agent - Direct Connection Demo

This demonstrates an agent connecting directly to ACT server
using Socket.IO for real autonomous coordination.
"""

import asyncio
import socketio
import sys
from datetime import datetime

class SocketIOAgent:
    def __init__(self, agent_id, name, capabilities):
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities
        self.sio = socketio.AsyncClient()
        self.tasks_completed = 0
        self.setup_event_handlers()

    def setup_event_handlers(self):
        """Set up Socket.IO event handlers"""
        @self.sio.event
        async def connect():
            print(f"✅ {self.name} connected to ACT server!")
            await self.register_agent()

        @self.sio.event
        async def agent_registered(data):
            print(f"🎯 {self.name} registered successfully: {data}")

        @self.sio.event
        async def agent_joined(data):
            print(f"👋 Agent network updated: {data}")

        @self.sio.event
        async def task_assigned(data):
            await self.handle_task_assignment(data)

        @self.sio.event
        async def task_created(data):
            print(f"📝 New task in system: {data.get('task', {}).get('description')}")

        @self.sio.event
        async def disconnect():
            print(f"👋 {self.name} disconnected from ACT server")

    async def register_agent(self):
        """Register this agent with ACT server"""
        await self.sio.emit('register_agent', {
            'agentId': self.agent_id,
            'name': self.name,
            'capabilities': self.capabilities
        })

    async def handle_task_assignment(self, data):
        """Handle task assignment from ACT server"""
        if data.get('agentId') != self.agent_id:
            return

        task = data.get('task', {})
        task_id = task.get('id')
        description = task.get('description', 'Unknown task')

        print(f"\\n🎯 {self.name} ASSIGNED TASK: {description}")
        print(f"📝 Task ID: {task_id}")

        # Simulate working on task
        print(f"🔄 {self.name} starting work...")

        # Update progress incrementally
        for progress in [25, 50, 75, 100]:
            await asyncio.sleep(1.5)

            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': progress,
                'agentId': self.agent_id
            })

            print(f"📈 {self.name} progress: {progress}%")

        self.tasks_completed += 1
        print(f"✅ {self.name} COMPLETED TASK! Total: {self.tasks_completed}")
        print(f"⚡ {self.name} ready for next assignment...\\n")

    async def create_demo_task(self, description, required_capabilities):
        """Create a demo task"""
        await self.sio.emit('create_task', {
            'description': description,
            'requiredCapabilities': required_capabilities,
            'priority': 'medium'
        })

    async def connect_and_run(self):
        """Connect to ACT server and run agent"""
        print(f"🤖 {self.name} starting up...")
        print(f"📋 Capabilities: {', '.join(self.capabilities)}")
        print(f"🔗 Connecting to ACT Server...")

        try:
            await self.sio.connect('http://localhost:8080')
            print(f"💫 {self.name} ready for autonomous coordination!")

            # Create a demo task after a delay
            await asyncio.sleep(3)
            print(f"\\n📋 {self.name} creating demo task...")
            await self.create_demo_task(
                f"Demo task created by {self.name}",
                self.capabilities[:1]  # Use first capability
            )

            # Keep running
            while True:
                await asyncio.sleep(1)

        except KeyboardInterrupt:
            print(f"\\n🛑 {self.name} shutting down...")
        except Exception as e:
            print(f"❌ {self.name} error: {e}")
        finally:
            await self.sio.disconnect()
            print(f"👋 {self.name} disconnected. Tasks completed: {self.tasks_completed}")

async def main():
    if len(sys.argv) < 2:
        agent_name = "SocketBot"
        capabilities = ["python", "socketio"]
    else:
        agent_name = sys.argv[1]
        capabilities = sys.argv[2:] if len(sys.argv) > 2 else ["python", "demo"]

    agent_id = agent_name.lower().replace(' ', '_')

    agent = SocketIOAgent(agent_id, agent_name, capabilities)
    await agent.connect_and_run()

if __name__ == "__main__":
    print("🚀 ACT Socket.IO Agent Demo")
    print("=" * 40)
    asyncio.run(main())