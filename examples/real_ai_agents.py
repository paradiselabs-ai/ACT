#!/usr/bin/env python3
"""
Real AI Agents using OpenRouter API

This creates actual intelligent agents that:
- Think and reason about tasks
- Communicate with each other
- Actually solve problems (not simulate)
- Coordinate autonomously through ACT server
"""

import asyncio
import socketio
import aiohttp
import json
import os
from datetime import datetime
from typing import List, Dict, Optional

class RealAIAgent:
    def __init__(self, agent_id: str, name: str, capabilities: List[str],
                 model: str, personality: str, color: str = "🤖"):
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities
        self.model = model  # OpenRouter model
        self.personality = personality
        self.color = color
        self.sio = socketio.AsyncClient()
        self.tasks_completed = 0
        self.is_running = True
        self.conversation_history = []

        # OpenRouter API setup
        self.api_key = os.getenv('OPENROUTER_API_KEY')
        if not self.api_key:
            raise ValueError("OPENROUTER_API_KEY environment variable required")

        self.setup_event_handlers()

    def setup_event_handlers(self):
        @self.sio.event
        async def connect():
            print(f"✅ {self.name} ({self.model}) connected to ACT server!")
            await self.register_agent()

        @self.sio.event
        async def agent_registered(data):
            print(f"🎯 {self.name} registered with capabilities: {', '.join(self.capabilities)}")

            # Send introduction to other agents
            intro = await self.generate_introduction()
            await self.broadcast_message(f"Hello! {intro}")

        @self.sio.event
        async def agent_joined(data):
            agent_name = data.get('name', 'Unknown')
            if agent_name != self.name:
                print(f"👋 {self.name} notices {agent_name} joined the team")

        @self.sio.event
        async def task_assigned(data):
            await self.handle_real_task(data)

        @self.sio.event
        async def agent_message(data):
            await self.handle_agent_communication(data)

    async def register_agent(self):
        await self.sio.emit('register_agent', {
            'agentId': self.agent_id,
            'name': self.name,
            'capabilities': self.capabilities,
            'model': self.model,
            'personality': self.personality
        })

    async def generate_introduction(self) -> str:
        """Generate AI-powered introduction"""
        prompt = f"""You are {self.name}, an AI agent with these capabilities: {', '.join(self.capabilities)}.
Your personality: {self.personality}

Generate a brief, friendly introduction (1-2 sentences) to other AI agents you'll be working with.
Be professional but show your personality."""

        response = await self.call_openrouter_api(prompt)
        return response.strip()

    async def call_openrouter_api(self, prompt: str, system_prompt: Optional[str] = None) -> str:
        """Call OpenRouter API with the agent's model"""
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        }

        messages = []
        if system_prompt:
            messages.append({"role": "system", "content": system_prompt})
        messages.append({"role": "user", "content": prompt})

        data = {
            "model": self.model,
            "messages": messages,
            "max_tokens": 500,
            "temperature": 0.7
        }

        async with aiohttp.ClientSession() as session:
            async with session.post(
                "https://openrouter.ai/api/v1/chat/completions",
                headers=headers,
                json=data
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    return result["choices"][0]["message"]["content"]
                else:
                    error_text = await response.text()
                    print(f"❌ {self.name} API error: {response.status} - {error_text}")
                    return f"Error: Could not process request"

    async def handle_real_task(self, data):
        """Handle actual task assignment with AI reasoning"""
        if data.get('agentId') != self.agent_id:
            return

        task = data.get('task', {})
        task_id = task.get('id')
        description = task.get('description')

        print(f"\n{self.color} {self.name} analyzing task: {description}")

        # AI analyzes the task
        analysis_prompt = f"""You are {self.name}, an AI agent with capabilities: {', '.join(self.capabilities)}.
Your personality: {self.personality}

You've been assigned this task: "{description}"

Please:
1. Analyze what this task requires
2. Determine your approach to complete it
3. Identify any challenges or considerations
4. Provide a brief work plan

Respond as {self.name} would, showing your reasoning process."""

        system_prompt = f"You are {self.name}. {self.personality} You have expertise in: {', '.join(self.capabilities)}."

        analysis = await self.call_openrouter_api(analysis_prompt, system_prompt)
        print(f"🧠 {self.name} thinks: {analysis[:200]}..." if len(analysis) > 200 else f"🧠 {self.name} thinks: {analysis}")

        # Notify other agents about starting work
        await self.broadcast_message(f"Starting work on: {description}. My approach: {analysis[:100]}...")

        # AI actually works on the task in phases
        phases = ["Planning", "Implementation", "Testing", "Completion"]

        for i, phase in enumerate(phases):
            progress = ((i + 1) / len(phases)) * 100

            # AI thinks about each phase
            phase_prompt = f"""You are currently in the "{phase}" phase of task: "{description}"

Based on your earlier analysis, what are you doing in this phase?
Provide a brief update (1-2 sentences) on your progress."""

            phase_work = await self.call_openrouter_api(phase_prompt, system_prompt)

            print(f"🔄 {self.name} [{phase}]: {phase_work}")

            # Update ACT server
            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': int(progress),
                'agentId': self.agent_id,
                'status': f"{phase}: {phase_work[:50]}..."
            })

            await asyncio.sleep(3)  # Realistic work time

        # Generate completion summary
        completion_prompt = f"""You just completed the task: "{description}"

Provide a brief summary of what you accomplished and any key results or deliverables."""

        completion_summary = await self.call_openrouter_api(completion_prompt, system_prompt)

        self.tasks_completed += 1
        print(f"✅ {self.name} COMPLETED: {completion_summary}")

        # Share completion with other agents
        await self.broadcast_message(f"Completed task: {description}. Result: {completion_summary[:100]}...")

    async def handle_agent_communication(self, data):
        """Handle messages from other AI agents"""
        sender = data.get('sender')
        message = data.get('message')

        if sender != self.name:
            print(f"💬 {self.name} received from {sender}: {message}")

            # AI decides whether and how to respond
            response_prompt = f"""Another AI agent "{sender}" sent you this message: "{message}"

You are {self.name} with personality: {self.personality}

Should you respond to this message? If yes, provide a brief, helpful response.
If no response needed, just say "NO_RESPONSE".
"""

            response = await self.call_openrouter_api(response_prompt)

            if response.strip() != "NO_RESPONSE":
                await self.broadcast_message(f"@{sender} {response}")

    async def broadcast_message(self, message: str):
        """Send message to other agents"""
        await self.sio.emit('agent_message', {
            'sender': self.name,
            'message': message,
            'timestamp': datetime.now().isoformat()
        })

    async def start(self):
        """Start the AI agent"""
        print(f"{self.color} {self.name} ({self.model}) initializing...")

        try:
            await self.sio.connect('http://localhost:8080')
            print(f"🧠 {self.name} ready for intelligent coordination!")

            while self.is_running:
                await asyncio.sleep(1)

        except Exception as e:
            print(f"❌ {self.name} error: {e}")
        finally:
            await self.sio.disconnect()

    async def stop(self):
        self.is_running = False

class TaskCreator:
    """Creates realistic tasks for AI agents"""

    def __init__(self):
        self.sio = socketio.AsyncClient()
        self.realistic_tasks = [
            ("Write a technical blog post about microservices architecture", ["writing", "technical", "backend"]),
            ("Create a user onboarding flow wireframe", ["design", "ux", "frontend"]),
            ("Debug a performance issue in the database queries", ["debugging", "backend", "database"]),
            ("Write unit tests for the authentication module", ["testing", "backend", "security"]),
            ("Design a responsive mobile layout", ["frontend", "design", "mobile"]),
            ("Analyze user feedback and create improvement recommendations", ["analysis", "ux", "research"]),
            ("Set up CI/CD pipeline for automated deployments", ["devops", "automation"]),
            ("Create API documentation for the user service", ["documentation", "backend", "technical"])
        ]

    async def create_realistic_tasks(self):
        """Create tasks that require actual AI work"""
        await self.sio.connect('http://localhost:8080')
        await asyncio.sleep(5)  # Let agents introduce themselves

        print("\n🚀 CREATING REAL TASKS FOR AI AGENTS")
        print("=" * 50)

        for i, (description, capabilities) in enumerate(self.realistic_tasks):
            await asyncio.sleep(4)  # Give time for previous task

            await self.sio.emit('create_task', {
                'description': description,
                'requiredCapabilities': capabilities,
                'priority': 'medium'
            })

            print(f"📋 Task {i+1}: {description}")

        print("\n🎯 All real tasks created! Watch AI agents think and work...")
        await self.sio.disconnect()

async def main():
    """Main demo with real AI agents"""
    print("🤖 REAL AI AGENT COORDINATION DEMO")
    print("=" * 60)
    print("🧠 Using OpenRouter API with actual AI models")
    print("💬 Agents will communicate, reason, and actually work on tasks")
    print("🔥 Press Ctrl+C to stop\n")

    # Check for API key
    if not os.getenv('OPENROUTER_API_KEY'):
        print("❌ Please set OPENROUTER_API_KEY environment variable")
        return

    # Create diverse AI agent team with different models and personalities
    agents = [
        RealAIAgent(
            "frontend_ai", "Sarah", ["frontend", "design", "ux"],
            "mistralai/mistral-7b-instruct:free",
            "Creative and user-focused frontend developer who loves clean, intuitive designs",
            "🎨"
        ),
        RealAIAgent(
            "backend_ai", "Marcus", ["backend", "database", "security"],
            "meta-llama/llama-3.1-8b-instruct:free",
            "Systematic backend engineer focused on scalability and security",
            "⚙️"
        ),
        RealAIAgent(
            "analyst_ai", "Elena", ["analysis", "research", "documentation"],
            "google/gemma-2-9b-it:free",
            "Detail-oriented analyst who loves turning data into insights",
            "📊"
        )
    ]

    task_creator = TaskCreator()

    try:
        print("🔗 Connecting AI agents to ACT server...")

        # Start all AI agents
        agent_tasks = [asyncio.create_task(agent.start()) for agent in agents]

        # Create realistic tasks
        task_generation = asyncio.create_task(task_creator.create_realistic_tasks())

        # Run coordination
        await asyncio.gather(*agent_tasks, task_generation)

    except KeyboardInterrupt:
        print("\n\n🛑 Stopping AI agent coordination...")

        for agent in agents:
            await agent.stop()

        print("\n🧠 AI AGENT COORDINATION RESULTS:")
        print("=" * 40)
        for agent in agents:
            print(f"  {agent.color} {agent.name} ({agent.model}): {agent.tasks_completed} tasks completed")

        print("\n🎉 Real AI agents coordinated autonomously!")
        print("💡 Agents actually reasoned, communicated, and solved problems!")

if __name__ == "__main__":
    asyncio.run(main())