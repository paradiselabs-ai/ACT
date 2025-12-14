#!/usr/bin/env python3
"""
Research-oriented ACT demo with harder, mixed-capability tasks.
No scripted messages; agents communicate via live Socket.IO events.
"""

import asyncio
import socketio
import aiohttp
import os
import json
from typing import List, Optional


class WorkingAIAgent:
    def __init__(self, agent_id: str, name: str, capabilities: List[str], model: str, personality: str, color: str = "🤖"):
        self.agent_id = agent_id
        self.name = name
        self.capabilities = capabilities
        self.model = model
        self.personality = personality
        self.color = color
        self.sio = socketio.AsyncClient()
        self.tasks_completed = 0
        self.is_running = True
        self.is_registered = False
        self.api_key = os.getenv('OPENROUTER_API_KEY')
        self.model = model

        # Rate limiting
        self.last_api_call = 0
        self.min_api_interval = 3  # 3 seconds between API calls
        self.setup_event_handlers()

    def setup_event_handlers(self):
        @self.sio.event
        async def connect():
            if not self.is_registered:
                print(f"✅ {self.name} ({self.model}) connected!")
                await self.register_agent()
                self.is_registered = True

        @self.sio.event
        async def agent_registered(data):
            if not self.is_registered:
                print(f"🎯 {self.name} registered successfully")
                self.is_registered = True

        @self.sio.event
        async def task_assigned(data):
            await self.handle_task_assignment(data)

        @self.sio.event
        async def task_created(data):
            task_desc = data.get('task', {}).get('description', 'Unknown')
            print(f"📝 {self.name} sees new task: {task_desc[:60]}...")

        @self.sio.event
        async def agent_message(data):
            msg = data.get('message')
            sender = data.get('sender', 'Unknown')
            if sender == self.name:
                return  # ignore own echoes
            if msg:
                print(f"💬 [{self.name}] heard {sender}: {msg[:120]}")
                # Generate a response to the peer message
                await self.respond_to_peer(sender, msg)

    async def register_agent(self):
        if self.is_registered:
            return

        await self.sio.emit('register_agent', {
            'agentId': self.agent_id,
            'name': self.name,
            'capabilities': self.capabilities,
            'model': self.model,
            'provider': 'openrouter'
        })

    async def rate_limited_api_call(self, prompt: str, max_tokens: int = 150) -> str:
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": os.getenv("OPENROUTER_HTTP_REFERER", "http://localhost:8080"),
            "X-Title": os.getenv("OPENROUTER_X_TITLE", "ACT Demo"),
            "Referer": os.getenv("OPENROUTER_HTTP_REFERER", "http://localhost:8080"),
        }

        data = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": f"You are {self.name}. {self.personality} Be concise and practical."},
                {"role": "user", "content": prompt}
            ],
            "max_tokens": max_tokens,
            "temperature": 0.7
        }

        for attempt in range(2):
            try:
                async with aiohttp.ClientSession() as session:
                    async with session.post(
                        "https://openrouter.ai/api/v1/chat/completions",
                        headers=headers,
                        json=data,
                        timeout=aiohttp.ClientTimeout(total=12)
                    ) as response:
                        body = await response.text()
                        hdrs = dict(response.headers)
                        print(
                            f"🌐 {self.name} OpenRouter "
                            f"status={response.status} model={self.model} "
                            f"req_headers={{'Referer': headers.get('HTTP-Referer'), 'X-Title': headers.get('X-Title')}} "
                            f"body_len={len(body)} body_sample={body[:200]} "
                            f"resp_ct={hdrs.get('content-type')} cf_ray={hdrs.get('cf-ray')}"
                        )
                        if response.status == 200:
                            try:
                                result = json.loads(body)
                                choices = result.get("choices", [])
                                if not choices:
                                    print(f"⚠️ {self.name} empty choices in response")
                                    return ""
                                message = choices[0].get("message", {})
                                content = message.get("content", "")
                                return content.strip() if content else ""
                            except Exception as parse_err:
                                print(f"⚠️ {self.name} JSON parse error: {parse_err}")
                                return ""
                        elif response.status == 429 and attempt < 1:
                            await asyncio.sleep(5 * (attempt + 1))
                            continue
                        else:
                            return ""
            except Exception as e:
                print(f"⚠️ {self.name} API exception: {str(e)[:120]}")
                if attempt < 1:
                    await asyncio.sleep(5 * (attempt + 1))
                    continue
                return ""

    async def respond_to_peer(self, sender: str, message: str):
        # Generate a response to peer's message using LLM
        response_prompt = f"You are {self.name}, {self.personality}. {sender} just said: '{message}'. Respond appropriately as a helpful agent in a coordination context. Keep it concise, under 100 words. If it's relevant to your capabilities, offer help or insights. Start with '@{sender}' if addressing them directly."
        response = await self.rate_limited_api_call(response_prompt, 100)
        if response:
            print(f"💬 {self.name} responding to {sender}: {response[:100]}")
            await self.broadcast_status(f"@{sender} {response}")
        else:
            # Fallback if API fails
            fallback = f"@{sender} Thanks for the update, I'll keep that in mind."
            print(f"💬 {self.name} fallback to {sender}: {fallback}")
            await self.broadcast_status(fallback)

    async def handle_task_assignment(self, data):
        if data.get('agentId') != self.agent_id:
            return

        task = data.get('task', {})
        task_id = task.get('id')
        description = task.get('description')

        print(f"\n{self.color} {self.name} ASSIGNED TASK: {description}")
        await self.broadcast_status(f"Starting work on: {description}")

        try:
            # Phase 1: Analysis
            analysis_prompt = f'Task: "{description}"\nProvide a 1-sentence analysis of this task.'
            analysis = await self.rate_limited_api_call(analysis_prompt, 80)
            if not analysis:
                print(f"⚠️ {self.name}: analysis unavailable (API empty); pausing task.")
                return

            print(f"💭 {self.name}: {analysis}")
            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': 25,
                'agentId': self.agent_id,
                'status': 'Analysis complete'
            })

            # Phase 2: Planning
            await asyncio.sleep(2)
            plan_prompt = f'For "{description}", give a brief 1-sentence plan.'
            plan = await self.rate_limited_api_call(plan_prompt, 80)
            if not plan:
                print(f"⚠️ {self.name}: plan unavailable (API empty); pausing task.")
                return
            print(f"📝 {self.name}: {plan}")
            await self.broadcast_status(f"Plan: {plan}")

            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': 50,
                'agentId': self.agent_id,
                'status': 'Planning complete'
            })

            # Phase 3: Implementation
            await asyncio.sleep(3)
            impl_prompt = f'Briefly describe your implementation approach for: "{description}"'
            implementation = await self.rate_limited_api_call(impl_prompt, 120)
            if "[api error" in implementation or "[offline" in implementation or "[rate limited" in implementation:
                implementation = f"Implementing best-effort steps for '{description}' using current expertise."
            print(f"🔧 {self.name}: {implementation}")
            await self.broadcast_status(f"Implementation: {implementation}")

            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': 75,
                'agentId': self.agent_id,
                'status': 'Implementation in progress'
            })

            # Phase 4: Completion
            await asyncio.sleep(2)
            completion_prompt = f'Summarize completion for: "{description}" (1 sentence)'
            completion = await self.rate_limited_api_call(completion_prompt, 80)

            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': 100,
                'agentId': self.agent_id,
                'status': 'Task completed'
            })

            self.tasks_completed += 1
            print(f"✅ {self.name} COMPLETED: {completion}")
            await self.broadcast_status(f"Completed: {completion}")

        except Exception as e:
            print(f"❌ {self.name} task error: {str(e)[:50]}")
            try:
                await self.broadcast_status(f"Task failed: {str(e)[:120]}")
            except Exception:
                pass
            await self.sio.emit('update_task_progress', {
                'taskId': task_id,
                'progress': 0,
                'agentId': self.agent_id,
                'status': f'Task failed: {str(e)[:30]}'
            })

    async def broadcast_status(self, message: str):
        if not message:
            return
        clean_message = message.strip()
        payload = {
            'sender': self.name,
            'message': clean_message,
            'timestamp': asyncio.get_event_loop().time()
        }
        await self.sio.emit('agent_message', payload)

    async def start(self):
        print(f"{self.color} {self.name} initializing...")
        try:
            await self.sio.connect('http://localhost:8080')
            print(f"🧠 {self.name} ready for coordination!")
            while self.is_running:
                await asyncio.sleep(1)
        except Exception as e:
            print(f"❌ {self.name} connection error: {e}")
        finally:
            await self.sio.disconnect()

    async def stop(self):
        self.is_running = False


class TaskCreator:
    """Creates harder, mixed-capability tasks to trigger best-effort behavior."""

    def __init__(self):
        self.sio = socketio.AsyncClient()
        self.tasks = [
            ("Security review of data pipeline", ["security", "backend"]),
            ("Data drift analysis on recent logs", ["data", "analysis"]),
            ("Performance tuning of frontend build", ["frontend", "performance"]),
            ("Write incident postmortem for outage", ["documentation", "ops"]),
            ("Cross-team coordination plan", ["coordination", "planning"])
        ]

    async def create_tasks(self):
        await self.sio.connect('http://localhost:8080')
        await asyncio.sleep(5)  # Let agents register

        print("\n📋 CREATING HARDER TASKS")
        print("=" * 50)

        for i, (description, capabilities) in enumerate(self.tasks):
            await asyncio.sleep(45)  # Spaced to reduce upstream rate limits
            await self.sio.emit('create_task', {
                'description': description,
                'requiredCapabilities': capabilities,
                'priority': 'high'
            })
            print(f"📝 Task {i+1}: {description}")

        await self.sio.disconnect()


async def main():
    print("🚀 WORKING AI RESEARCH DEMO")
    print("=" * 60)
    print("🧠 Real AI agents with coordination + best-effort fallback")
    print("⚡ Tasks are harder and cross-capability to trigger collaboration")
    print("🔥 Press Ctrl+C to stop\n")

    if not os.getenv('OPENROUTER_API_KEY'):
        print("❌ Please set OPENROUTER_API_KEY environment variable")
        return

    # Use one Qwen free model and one OpenAI free model to spread load
    agent1_model = os.getenv("AGENT1_MODEL", "openai/gpt-oss-120b")
    agent2_model = os.getenv("AGENT2_MODEL", "openai/gpt-oss-120b")

    agents = [
        WorkingAIAgent(
            "designer", "Alex", ["design", "frontend", "ux"],
            agent1_model,
            "Creative designer focused on user experience and clean interfaces",
            "🎨"
        ),
        WorkingAIAgent(
            "analyst", "Morgan", ["analysis", "research", "documentation"],
            agent2_model,
            "Analytical thinker who loves data insights and clear documentation",
            "📊"
        )
    ]

    task_creator = TaskCreator()

    try:
        print("🔗 Starting AI agents...")
        agent_tasks = [asyncio.create_task(agent.start()) for agent in agents]
        task_generation = asyncio.create_task(task_creator.create_tasks())
        await asyncio.gather(*agent_tasks, task_generation)
    except KeyboardInterrupt:
        print("\n🛑 Stopping coordination...")
        for agent in agents:
            await agent.stop()


if __name__ == "__main__":
    asyncio.run(main())
