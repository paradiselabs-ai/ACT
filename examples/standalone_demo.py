#!/usr/bin/env python3
"""
ACT Standalone Demo - Multi-Agent Coordination

This demonstrates multiple Python agents coordinating autonomously
through ACT server without any platform integration.

Showcases:
- Multiple agents with different capabilities
- Autonomous task assignment based on capabilities
- Real-time coordination without human intervention
- Conflict resolution and load balancing
"""

import asyncio
import sys
import os
import socketio
from datetime import datetime

class DemoAgent:
    def __init__(self, agent_id, name, capabilities, color="🤖"):
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities
        self.color = color
        self.client = ACTClient()
        self.tasks_completed = 0
        self.is_running = True

    async def start(self):
        """Start the agent and connect to ACT server"""
        print(f"{self.color} {self.name} starting...")

        try:
            await self.client.connect('ws://localhost:8080')
            await self.client.register(
                agent_id=self.agent_id,
                name=self.name,
                capabilities=self.capabilities
            )

            self.client.on('task_assigned', self.handle_task)
            print(f"✅ {self.name} ready for coordination!")

            while self.is_running:
                await asyncio.sleep(0.5)

        except Exception as e:
            print(f"❌ {self.name} error: {e}")
        finally:
            await self.client.disconnect()

    async def handle_task(self, data):
        """Handle assigned tasks"""
        task = data.get('task', {})
        task_id = task.get('id')
        description = task.get('description', 'Unknown task')

        print(f"\\n{self.color} {self.name} received: {description}")

        # Simulate task execution with realistic timing
        work_time = 2 + (len(description) % 3)  # Variable work time

        for i in range(4):
            progress = (i + 1) * 25
            await asyncio.sleep(work_time / 4)
            await self.client.update_task_progress(task_id, progress)

        self.tasks_completed += 1
        print(f"✅ {self.name} completed task! Total: {self.tasks_completed}")

    async def stop(self):
        """Stop the agent"""
        self.is_running = False

class TaskGenerator:
    def __init__(self):
        self.client = ACTClient()
        self.tasks = [
            ("Build user authentication system", ["backend", "security"]),
            ("Create React dashboard component", ["frontend", "react"]),
            ("Write unit tests for API endpoints", ["testing", "backend"]),
            ("Design database schema", ["database", "backend"]),
            ("Implement responsive UI layout", ["frontend", "css"]),
            ("Set up CI/CD pipeline", ["devops", "deployment"]),
            ("Add input validation", ["backend", "security"]),
            ("Create user onboarding flow", ["frontend", "ux"]),
            ("Optimize database queries", ["database", "performance"]),
            ("Add error handling", ["backend", "testing"])
        ]

    async def start_generating_tasks(self):
        """Generate tasks for agents to work on"""
        await self.client.connect('ws://localhost:8080')

        await asyncio.sleep(2)  # Let agents register first

        print("\\n📋 Starting task generation...")

        for i, (description, capabilities) in enumerate(self.tasks):
            await asyncio.sleep(3)  # Space out task creation

            await self.client.create_task(
                description=description,
                required_capabilities=capabilities,
                priority="medium"
            )

            print(f"📝 Task {i+1}/10: {description}")

        await self.client.disconnect()

async def main():
    print("🚀 ACT Standalone Multi-Agent Demo")
    print("=" * 50)
    print("🎯 Demonstrating autonomous agent coordination")
    print("⚡ Watch agents self-organize based on capabilities")
    print("🔥 Press Ctrl+C to stop\\n")

    # Create diverse agents with different capabilities
    agents = [
        DemoAgent("frontend_dev", "Frontend Dev", ["frontend", "react", "css", "ux"], "🎨"),
        DemoAgent("backend_dev", "Backend Dev", ["backend", "database", "security"], "⚙️"),
        DemoAgent("qa_engineer", "QA Engineer", ["testing", "backend", "frontend"], "🧪"),
        DemoAgent("devops_eng", "DevOps Engineer", ["devops", "deployment", "performance"], "🚢")
    ]

    # Create task generator
    task_generator = TaskGenerator()

    try:
        # Start all agents
        agent_tasks = [asyncio.create_task(agent.start()) for agent in agents]

        # Start task generation
        generator_task = asyncio.create_task(task_generator.start_generating_tasks())

        # Wait for completion or interruption
        await asyncio.gather(*agent_tasks, generator_task)

    except KeyboardInterrupt:
        print("\\n\\n🛑 Demo stopping...")

        # Stop all agents
        for agent in agents:
            await agent.stop()

        print("\\n📊 Demo Results:")
        for agent in agents:
            print(f"  {agent.color} {agent.name}: {agent.tasks_completed} tasks completed")

        print("\\n🎉 ACT autonomous coordination demonstrated!")
        print("💡 Agents self-organized and coordinated without human intervention")

if __name__ == "__main__":
    asyncio.run(main())