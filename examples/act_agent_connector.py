#!/usr/bin/env python3
"""
ACT Agent Connector - Blank agents ready for REPL coordination
"""

import os
import asyncio
import aiohttp
import json
from datetime import datetime
import socketio as sio

class ACTAgent:
    def __init__(self, agent_id: str, name: str, capabilities: list, model: str, personality: str, emoji: str):
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities
        self.model = model
        self.personality = personality
        self.emoji = emoji
        self.is_running = True
        self.current_task = None

        # Setup event handlers
        self.sio = sio.AsyncClient()
        self.setup_event_handlers()

    def setup_event_handlers(self):
        @self.sio.event
        async def connect():
            print(f"{self.emoji} {self.name} connected to ACT server")

        @self.sio.event
        async def disconnect():
            print(f"{self.emoji} {self.name} disconnected from ACT server")

        @self.sio.event
        async def agent_registered(data):
            print(f"{self.emoji} {self.name} registered successfully")

        @self.sio.event
        async def task_assigned(data):
            task = data.get('task', {})
            if task.get('assignedAgent') == self.agent_id:
                await self.handle_task_assigned(task)

        @self.sio.event
        async def agent_message(data):
            await self.handle_agent_message(data)

    async def handle_task_assigned(self, task):
        self.current_task = task
        print(f"{self.emoji} 🎯 {self.name} assigned task: {task.get('description', 'Unknown')}")

        # Acknowledge assignment
        await self.broadcast_status(f"Task '{task.get('description', 'Unknown')}' assigned. Starting work...")

        # Simulate working on task
        await self.work_on_task(task)

    async def handle_agent_message(self, data):
        sender = data.get('sender')
        message = data.get('message')

        if sender == self.agent_id:
            return  # Ignore own messages

        print(f"{self.emoji} 💬 {self.name} heard {sender}: {message[:100]}...")
        # ACT now handles coordination responses - agents just receive messages

    async def generate_response(self, sender: str, message: str) -> str:
        # ACT server now handles coordination responses
        # Agents focus on their specialized work
        return None

    async def work_on_task(self, task):
        """Simulate working on a task"""
        task_id = task.get('id')
        description = task.get('description', 'Unknown task')

        # Phase 1: Analysis
        await asyncio.sleep(2)
        analysis = f"Analyzing: {description[:50]}..."
        print(f"{self.emoji} 💭 {self.name}: {analysis}")
        await self.update_task_progress(task_id, 25, f"Analysis: {analysis}")

        # Phase 2: Planning
        await asyncio.sleep(3)
        plan = f"Planning approach for {description[:30]}..."
        print(f"{self.emoji} 📋 {self.name}: {plan}")
        await self.update_task_progress(task_id, 50, f"Planning: {plan}")

        # Phase 3: Implementation
        await asyncio.sleep(5)
        implementation = f"Implementing solution for {description[:40]}..."
        print(f"{self.emoji} ⚙️ {self.name}: {implementation}")
        await self.update_task_progress(task_id, 75, f"Implementation: {implementation}")

        # Phase 4: Completion
        await asyncio.sleep(3)
        completion = f"Task completed: {description[:45]}"
        print(f"{self.emoji} ✅ {self.name}: {completion}")
        await self.update_task_progress(task_id, 100, f"Completed: {completion}")

        # Mark task as complete
        await self.update_task_progress(task_id, 100, "Task completed successfully")

    async def update_task_progress(self, task_id: str, progress: int, message: str):
        """Update task progress"""
        await self.sio.emit('update_task_progress', {
            'taskId': task_id,
            'progress': progress,
            'status': message,
            'message': message,
            'agentId': self.agent_id
        })

    async def broadcast_status(self, message: str):
        """Broadcast status message to other agents"""
        payload = {
            'sender': self.agent_id,
            'message': message,
            'timestamp': asyncio.get_event_loop().time()
        }
        await self.sio.emit('agent_message', payload)

    async def register(self):
        """Register with ACT server"""
        registration_data = {
            'agentId': self.agent_id,
            'capabilities': self.capabilities,
            'name': self.name,
            'model': self.model,
            'provider': 'openrouter'
        }

        print(f"{self.emoji} 🤖 {self.name} registering with ACT server...")
        await self.sio.emit('register_agent', registration_data)

    async def start(self):
        """Start the agent"""
        print(f"{self.emoji} {self.name} initializing...")

        try:
            await self.sio.connect('http://localhost:8080')
            await self.register()
            await asyncio.sleep(2)  # Let registration complete

            while self.is_running:
                await asyncio.sleep(1)

        except Exception as e:
            print(f"{self.emoji} ❌ {self.name} error: {e}")
        finally:
            await self.sio.disconnect()

    async def stop(self):
        """Stop the agent"""
        self.is_running = False
        await self.sio.disconnect()


async def main():
    print("🚀 ACT Agent Connector")
    print("=" * 50)
    print("🤖 Agents ready for REPL coordination")
    print("🔧 Use 'act' command to manage projects and tasks")
    print("💬 Agents will communicate and collaborate automatically")
    print("🛑 Press Ctrl+C to stop\n")

    if not os.getenv('OPENROUTER_API_KEY'):
        print("❌ Please set OPENROUTER_API_KEY environment variable")
        return

    # Same agents from working_ai_demo_research.py
    agents = [
        ACTAgent(
            "designer", "Alex", ["design", "frontend", "ux", "ui"],
            "openai/gpt-oss-120b",  # Same model as demo
            "Creative designer focused on user experience and clean interfaces",
            "🎨"
        ),
        ACTAgent(
            "analyst", "Morgan", ["analysis", "research", "documentation", "data"],
            "openai/gpt-oss-120b",  # Same model as demo
            "Analytical thinker who loves data insights and clear documentation",
            "📊"
        )
    ]

    try:
        print("🔗 Starting ACT agents...")
        agent_tasks = [asyncio.create_task(agent.start()) for agent in agents]
        await asyncio.gather(*agent_tasks)

    except KeyboardInterrupt:
        print("\n🛑 Stopping agents...")
        for agent in agents:
            await agent.stop()
        print("✅ All agents stopped.")


if __name__ == "__main__":
    asyncio.run(main())
