#!/usr/bin/env python3
"""
ACT Multi-Agent Coordination Demo

This demonstrates multiple Python agents coordinating autonomously
through ACT server without any platform integration.

Showcases:
- Multiple agents with different capabilities
- Autonomous task assignment based on capabilities
- Real-time coordination without human intervention
- Load balancing and intelligent task distribution
"""

import asyncio
import socketio
from datetime import datetime

class MultiAgent:
    def __init__(self, agent_id, name, capabilities, color="🤖"):
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities
        self.color = color
        self.sio = socketio.AsyncClient()
        self.tasks_completed = 0
        self.is_running = True
        self.setup_events()

    def setup_events(self):
        @self.sio.event
        async def connect():
            print(f"✅ {self.name} connected!")
            await self.register()

        @self.sio.event
        async def agent_registered(data):
            print(f"🎯 {self.name} registered with capabilities: {', '.join(self.capabilities)}")

        @self.sio.event
        async def task_assigned(data):
            await self.handle_assignment(data)

        @self.sio.event
        async def task_created(data):
            task_desc = data.get('task', {}).get('description', 'Unknown')
            print(f"📝 {self.name} sees: {task_desc}")

    async def register(self):
        await self.sio.emit('register_agent', {
            'agentId': self.agent_id,
            'name': self.name,
            'capabilities': self.capabilities
        })

    async def handle_assignment(self, data):
        if data.get('agentId') != self.agent_id:
            return

        task = data.get('task', {})
        task_id = task.get('id')
        description = task.get('description')

        print(f"\\n{self.color} {self.name} WORKING ON: {description}")

        # Realistic work simulation
        work_steps = 4
        work_time = 2.0

        for step in range(work_steps):
            progress = ((step + 1) / work_steps) * 100
            await asyncio.sleep(work_time / work_steps)

            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': int(progress),
                'agentId': self.agent_id
            })

            print(f"📈 {self.name} progress: {int(progress)}%")

        self.tasks_completed += 1
        print(f"✅ {self.name} DONE! ({self.tasks_completed} total)")

    async def start(self):
        try:
            await self.sio.connect('http://localhost:8080')
            while self.is_running:
                await asyncio.sleep(0.5)
        except Exception as e:
            print(f"❌ {self.name}: {e}")
        finally:
            await self.sio.disconnect()

    async def stop(self):
        self.is_running = False

class DemoTaskCreator:
    def __init__(self):
        self.sio = socketio.AsyncClient()
        self.demo_tasks = [
            ("Build user authentication API", ["backend", "security"]),
            ("Create React dashboard", ["frontend", "react"]),
            ("Write comprehensive tests", ["testing", "qa"]),
            ("Design database schema", ["backend", "database"]),
            ("Implement responsive UI", ["frontend", "css"]),
            ("Set up deployment pipeline", ["devops", "ci-cd"]),
            ("Add security middleware", ["backend", "security"]),
            ("Create user onboarding", ["frontend", "ux"]),
            ("Optimize queries", ["backend", "database"]),
            ("Final QA testing", ["testing", "qa"])
        ]

    async def generate_tasks(self):
        await self.sio.connect('http://localhost:8080')
        await asyncio.sleep(3)  # Let agents register

        print("\\n🚀 STARTING AUTONOMOUS TASK GENERATION")
        print("=" * 50)

        for i, (description, capabilities) in enumerate(self.demo_tasks):
            await asyncio.sleep(2.5)  # Realistic spacing

            await self.sio.emit('create_task', {
                'description': description,
                'requiredCapabilities': capabilities,
                'priority': 'medium'
            })

            print(f"📋 Created task {i+1}/10: {description}")

        print("\\n🎯 All tasks created! Watch agents coordinate autonomously...")
        await self.sio.disconnect()

async def main():
    print("🚀 ACT MULTI-AGENT COORDINATION DEMO")
    print("=" * 60)
    print("🤖 Launching specialized agents...")
    print("⚡ Watch autonomous task assignment based on capabilities")
    print("🔥 Press Ctrl+C to stop\\n")

    # Create diverse agent team
    agents = [
        MultiAgent("frontend_specialist", "Frontend Dev", ["frontend", "react", "css", "ux"], "🎨"),
        MultiAgent("backend_specialist", "Backend Dev", ["backend", "database", "security"], "⚙️"),
        MultiAgent("qa_specialist", "QA Engineer", ["testing", "qa", "automation"], "🧪"),
        MultiAgent("devops_specialist", "DevOps Engineer", ["devops", "ci-cd", "deployment"], "🚢"),
    ]

    task_creator = DemoTaskCreator()

    try:
        print("🔗 Connecting agents to ACT server...")

        # Start all agents
        agent_tasks = [asyncio.create_task(agent.start()) for agent in agents]

        # Start task generation after delay
        generation_task = asyncio.create_task(task_creator.generate_tasks())

        # Run until interrupted
        await asyncio.gather(*agent_tasks, generation_task)

    except KeyboardInterrupt:
        print("\\n\\n🛑 Demo stopping...")

        # Graceful shutdown
        for agent in agents:
            await agent.stop()

        print("\\n📊 FINAL RESULTS:")
        print("=" * 30)
        total_tasks = sum(agent.tasks_completed for agent in agents)

        for agent in agents:
            efficiency = f"({agent.tasks_completed}/{total_tasks})" if total_tasks > 0 else "(0/0)"
            print(f"  {agent.color} {agent.name}: {agent.tasks_completed} tasks {efficiency}")

        print(f"\\n🎉 Total coordinated tasks: {total_tasks}")
        print("💡 Agents self-organized based on capabilities!")
        print("🚀 Autonomous multi-agent coordination demonstrated!")

if __name__ == "__main__":
    asyncio.run(main())