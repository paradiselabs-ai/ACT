# TeamKit & TeamKitOS Architecture

**The Agent Team Factory - Standalone Toolkit & ACT-Integrated OS**

> *"Create, customize, and deploy specialized AI agent teams with unprecedented precision"*

---

## Table of Contents

1. [Overview](#overview)
2. [The Three-Layer ACT Ecosystem](#the-three-layer-act-ecosystem)
3. [TeamKitOS Core Concept](#teamkitos-core-concept)
4. [Low-Code/Vibe-Code Platform](#low-codevibe-code-platform)
5. [Open-Source Model Focus](#open-source-model-focus)
6. [SmolAgents Integration](#smolagents-integration)
7. [Advanced Model Techniques](#advanced-model-techniques)
8. [RWKV Long-Context Architecture](#rwkv-long-context-architecture)
9. [TeamKitOS Developer SDK](#teamkitos-developer-sdk)
10. [Native ACT Integration](#native-act-integration)
11. [Complete Workflow](#complete-workflow)
12. [Technical Architecture](#technical-architecture)
13. [Business Model](#business-model)
14. [Roadmap](#roadmap)

---

## Overview

**TeamKit** is a standalone model creation and merging toolkit. **TeamKitOS** is TeamKit integrated with ACT, gaining the ability to train models on PVM coordination logs.

### TeamKit (Standalone)
- Model creation and merging toolkit
- HuggingFace + Ollama integration
- Advanced model techniques (DARE, TIES, mergekit, abliteration)
- SmolAgents expert swarm creation
- **NO PVM training** - just model manipulation tools

### TeamKitOS (TeamKit + ACT Integration)
- Everything in TeamKit, PLUS:
- Train new models on **existing PVM coordination logs**
- **Continuous RL** through real-time PVM data
- Self-improving model generation
- "OS" = Operating System for intelligent agent team creation

**The transformation happens in ACT Studio**: When TeamKit is combined with AgentMix (which includes ACT), it becomes TeamKitOS.

### The Revolutionary Theory

**Many Experts > One Jack of All Trades**

TeamKitOS is built on the validated principle that **specialized small language models (SLMs) coordinating effectively outperform massive general-purpose models** for complex multi-domain tasks.

**Mathematical Validation:**

```
Specialized Expert Swarm (10 x 7B SLMs):
- Total Parameters: 70B
- Effective Parameters: 70B (100% specialized)
- Cost: ~$0.0001/1K tokens
- Speed: ~50-100 tokens/second (local)

GPT-4 (General Purpose):
- Total Parameters: 1.76T
- Effective Parameters: ~17.6B-88B (1-5% for any given task)
- Cost: ~$0.03/1K tokens (300x more expensive)
- Speed: ~10-20 tokens/second (API latency)

RESULT:
- 4x-10x more effective knowledge per task
- 300x cheaper
- 5x-10x faster
- Complete data privacy (local execution)
```

### Why This Matters

Traditional AI development forces you to choose:
- **Option A**: Use expensive general-purpose LLMs (GPT-4, Claude) that waste 95%+ of their parameters on every task
- **Option B**: Train custom models from scratch (expensive, time-consuming, requires ML expertise)

**TeamKitOS introduces Option C**: Build specialized agent teams from open-source SLMs using advanced composition techniques - no ML expertise required.

---

## The ACT Ecosystem

### Standalone Products

#### 1. ACT (Agent Coordination Transfer)
**What**: Open-source coordination protocol
**License**: MIT (FREE)
**Purpose**: Universal infrastructure for autonomous agent coordination
**Users**: Developers building any multi-agent system
**Can use alone**: Yes

#### 2. AgentMix
**What**: Premium execution environment (includes ACT backend)
**Pricing**: $49-199/month
**Purpose**: Interactive REPL interface for running ACT-coordinated teams
**Users**: Professional developers, teams, enterprises
**Can use alone**: Yes (ACT is built-in)

#### 3. TeamKit
**What**: Model creation and merging toolkit
**Pricing**: TBD (likely $99-299/month or enterprise)
**Purpose**: Create and merge open-source models (HuggingFace/Ollama)
**Users**: Developers who need custom models
**Can use alone**: Yes (but no PVM training capabilities)

### Combined Platform

#### ACT Studio = TeamKit + AgentMix
**What**: Complete agent team development platform
**Why "Studio"**: All three systems integrated (AgentMix includes ACT)
**Key benefit**: TeamKit becomes TeamKitOS (gains PVM training)
**Platform**: Desktop app (like LM Studio)
**Value**: Create models → Test teams → Optimize via PVM → Deploy
**Target**: Serious AI development teams and enterprises

**The naming logic**:
- TeamKit (standalone) = basic model toolkit
- TeamKit + ACT = TeamKitOS (PVM training enabled)
- Since AgentMix includes ACT, TeamKit + AgentMix = ACT Studio

---

## TeamKitOS Core Concept

### What is TeamKit?

**TeamKit (standalone)** is a model creation and merging toolkit:

1. **Model Management** (HuggingFace + Ollama)
2. **Model Merging** (DARE, TIES, mergekit)
3. **Abliteration** (Heretic - uncensored models)
4. **SmolAgents Integration** (expert swarm creation)
5. **Basic Fine-tuning** (LoRA)

### What Makes TeamKitOS Different?

When TeamKit is combined with ACT (in ACT Studio), it becomes **TeamKitOS** - a complete **low-code/vibe-code platform** for creating specialized AI agent teams:

1. **Everything TeamKit has** (model management, merging, abliteration)
2. **PLUS: PVM Training** - Train new models on existing coordination logs
3. **PLUS: Continuous RL** - Models improve through their own coordination data
4. **PLUS: Self-improving Generation** - Each model generation trains the next

**The "OS" transformation happens through ACT integration**:

1. **Open-Source Models** (HuggingFace + Ollama)
2. **Expert Swarm Orchestration** (SmolAgents)
3. **Novel LLM Architectures** (RWKV)
4. **Advanced Model Techniques** (DARE, TIES, mergekit, abliteration)
5. **Native ACT Integration** (for optimization and deployment)

### The Revolutionary PVM Training Loop

**This is what makes TeamKitOS fundamentally different from TeamKit**:

```
1. INITIAL TRAINING (TeamKitOS creates new models)
   → Train on EXISTING PVM coordination logs
   → Bootstrap with proven coordination patterns
   → Learn from past successful agent teams
   → "Start smart" instead of from scratch

2. DEPLOYMENT (AgentMix with ACT)
   → Models deployed as coordinated agents
   → ACT orchestrates their collaboration
   → PVM records real-time coordination data

3. CONTINUOUS RL (Ongoing learning)
   → Models continuously RL'd through PVM
   → Improve via their own coordination patterns
   → Self-optimization without manual intervention

4. NEXT GENERATION (Compounding improvement)
   → Future models trained on expanded PVM logs
   → Include patterns from current generation
   → Each generation teaches the next
   → Exponential coordination intelligence growth
```

**Key insight**: New models aren't RL'd on their OWN logs initially - they're trained on EXISTING coordination logs from past teams. Once deployed, they THEN continuously RL through their own PVM data.

**Why this is revolutionary**:
- Models start with proven coordination knowledge (not random initialization)
- Continuous improvement through real-world usage
- Self-improving without human intervention
- Coordination intelligence compounds over generations

### The "Vibe-Code" Philosophy

**Vibe-code** means you describe **what you want**, not **how to build it**:

```plaintext
Traditional: "I need to fine-tune a 7B model on 10K examples, configure LoRA
             parameters, set up training infrastructure..."

Vibe-Code:  "I need a Python expert that understands FastAPI and PostgreSQL"
```

TeamKitOS handles:
- Model selection
- Expert swarm composition
- Capability optimization
- Integration with ACT
- Testing and validation

You focus on **outcomes**, not **implementation details**.

---

## Low-Code/Vibe-Code Platform

### Interface Options

TeamKitOS provides multiple interfaces for different user preferences:

#### 1. Visual Team Builder (Low-Code)
```
┌─────────────────────────────────────────────┐
│ TeamKitOS - New Agent Team                  │
├─────────────────────────────────────────────┤
│                                             │
│  Team Name: [web-dev-experts____________]   │
│                                             │
│  Domain Focus:                              │
│  ☑ Web Development                          │
│  ☑ Database Design                          │
│  ☐ Machine Learning                         │
│  ☑ API Design                               │
│                                             │
│  Base Models:                               │
│  ⚬ Code Llama 7B (Python/JS expert)        │
│  ⚬ DeepSeek Coder 6.7B (Architecture)      │
│  ⚬ Mistral 7B (General reasoning)          │
│                                             │
│  Advanced Techniques:                       │
│  ☑ Model Merging (DARE)                     │
│  ☑ Abliteration (Uncensored)                │
│  ☐ Custom Fine-tuning                       │
│                                             │
│  [Generate Team] [Test Configuration]      │
└─────────────────────────────────────────────┘
```

#### 2. Vibe-Code CLI (Natural Language)
```bash
teamkit create "I need a team for building FastAPI microservices
                with PostgreSQL and Redis. Experts in async Python,
                database optimization, and API security"

# TeamKitOS analyzes requirements:
# → Selecting Code Llama 7B (Python specialist)
# → Adding DeepSeek Coder 6.7B (async patterns)
# → Including SQL-Coder 7B (database expert)
# → Applying DARE merge for API security knowledge
# → Creating 3-agent expert swarm
# ✓ Team 'fastapi-experts' created

teamkit test fastapi-experts "Design a scalable user authentication system"
# → Testing team coordination...
# → Analyzing responses...
# ✓ Team performance: 94% (Python: 98%, DB: 92%, Security: 91%)
```

#### 3. Developer SDK (Full Control)
```python
from teamkitos import TeamBuilder, SmolAgentSwarm, ModelMerger

# Create team builder
builder = TeamBuilder()

# Add specialized agents
builder.add_agent(
    base_model="codellama/CodeLlama-7b-Instruct",
    specialization="python_expert",
    capabilities=["fastapi", "async", "type_hints"],
    merge_technique="DARE"
)

builder.add_agent(
    base_model="defog/sqlcoder-7b-2",
    specialization="database_expert",
    capabilities=["postgresql", "query_optimization", "indexing"]
)

# Configure swarm coordination
swarm = SmolAgentSwarm(
    agents=builder.agents,
    coordination_strategy="capability_based",
    memory_system="rwkv",
    context_length=100000
)

# Test and optimize
results = swarm.test_scenario("Design auth system")
optimized = swarm.optimize_with_act(results)

# Deploy to ACT
optimized.deploy_to_act(
    project="fastapi-microservice",
    environment="production"
)
```

---

## Open-Source Model Focus

### Why Open-Source?

TeamKitOS focuses exclusively on **HuggingFace** and **Ollama** models:

#### Advantages:
1. **Cost**: Free to run locally (only hardware costs)
2. **Privacy**: Data never leaves your infrastructure
3. **Customization**: Full access to modify, merge, fine-tune
4. **Speed**: No API latency, runs on local hardware
5. **Ownership**: No vendor lock-in, models are yours forever

#### Supported Model Hubs:
- **HuggingFace Hub**: 500K+ models (primary source)
- **Ollama**: Optimized local inference models
- **Custom Models**: GGUF, ONNX, SafeTensors formats

### Recommended Base Models

TeamKitOS curates high-performance SLMs for different domains:

#### Code & Development (7B-13B)
```yaml
- codellama/CodeLlama-7b-Instruct-hf
  Focus: Python, JavaScript, general programming

- deepseek-ai/deepseek-coder-6.7b-instruct
  Focus: Architecture, design patterns

- defog/sqlcoder-7b-2
  Focus: SQL, database queries

- WizardLM/WizardCoder-Python-7B-V1.0
  Focus: Python-specific tasks
```

#### Reasoning & Analysis (7B-13B)
```yaml
- mistralai/Mistral-7B-Instruct-v0.2
  Focus: General reasoning, analysis

- NousResearch/Nous-Hermes-2-Mistral-7B-DPO
  Focus: Creative problem-solving

- teknium/OpenHermes-2.5-Mistral-7B
  Focus: Multi-turn reasoning
```

#### Domain-Specific (7B-13B)
```yaml
- medalpaca/medalpaca-7b
  Focus: Medical/healthcare

- BloombergGPT-7B (if available)
  Focus: Finance, economics

- BioMistral-7B
  Focus: Biology, chemistry
```

### Model Selection Strategy

TeamKitOS uses intelligent model selection based on:

1. **Task Requirements**: What capabilities are needed?
2. **Performance Benchmarks**: Which models excel in this domain?
3. **Resource Constraints**: What hardware is available?
4. **Composition Potential**: How well do models merge together?

---

## SmolAgents Integration

### What is SmolAgents?

**SmolAgents** is HuggingFace's framework for orchestrating small language model expert swarms. It provides:

- **Multi-agent coordination** (similar to ACT, but model-level)
- **Tool calling** for SLMs
- **Code execution** capabilities
- **Memory systems** for agent state
- **Expert routing** based on capabilities

### Why SmolAgents + ACT?

```
SmolAgents Layer: Individual agent capabilities and tool use
       ↕
ACT Layer: Multi-agent coordination and task distribution
       ↕
PVM Layer: Semantic memory and learning optimization
```

**SmolAgents handles**: "How does this agent use tools and execute tasks?"
**ACT handles**: "Which agent should work on which task?"
**PVM handles**: "What have we learned about optimal coordination?"

### TeamKitOS SmolAgents Configuration

```python
from teamkitos import SmolAgentConfig

# Configure Python expert
python_expert = SmolAgentConfig(
    base_model="codellama/CodeLlama-7b-Instruct",
    tools=[
        "python_repl",
        "file_editor",
        "code_analyzer",
        "pytest_runner"
    ],
    system_prompt="""You are a Python expert specializing in
                     modern async patterns, type safety, and
                     clean architecture.""",
    temperature=0.3,
    max_tokens=2048
)

# Configure database expert
db_expert = SmolAgentConfig(
    base_model="defog/sqlcoder-7b-2",
    tools=[
        "sql_executor",
        "query_analyzer",
        "index_advisor",
        "migration_generator"
    ],
    system_prompt="""You are a database expert specializing in
                     PostgreSQL optimization, indexing strategies,
                     and query performance.""",
    temperature=0.2,
    max_tokens=1024
)

# Create expert swarm
swarm = SmolAgentSwarm([python_expert, db_expert])

# SmolAgents handles individual capabilities
# ACT handles coordination between them
```

### Expert Swarm Patterns

TeamKitOS provides pre-configured swarm patterns:

#### 1. **Specialist Pattern** (3-5 agents)
Best for: Focused domain work (e.g., web development)
```
Frontend Expert + Backend Expert + Database Expert
```

#### 2. **Generalist + Specialists Pattern** (4-7 agents)
Best for: Complex projects requiring coordination
```
Coordinator (Mistral 7B)
  ├── Code Expert (CodeLlama 7B)
  ├── Test Expert (WizardCoder 7B)
  ├── Database Expert (SQLCoder 7B)
  └── Documentation Expert (Mistral 7B fine-tuned)
```

#### 3. **Hierarchical Pattern** (6-10 agents)
Best for: Enterprise projects with multiple domains
```
Lead Architect (Mistral 13B)
  ├── Backend Team
  │   ├── API Expert (CodeLlama 7B)
  │   ├── Database Expert (SQLCoder 7B)
  │   └── Security Expert (Custom 7B)
  └── Frontend Team
      ├── React Expert (CodeLlama 7B fine-tuned)
      ├── CSS Expert (Custom 7B)
      └── Testing Expert (WizardCoder 7B)
```

---

## Advanced Model Techniques

### 1. Model Merging (DARE, TIES, mergekit)

**The Problem**: Training custom models is expensive and time-consuming.

**The Solution**: Merge existing expert models to create new capabilities.

#### DARE (Drop And REscale)

**Concept**: Intelligently combine models by dropping redundant parameters and rescaling important ones.

```python
from teamkitos.merging import DAREMerger

# Merge Python expert + Security expert = Secure Python expert
merger = DAREMerger()

secure_python_model = merger.merge(
    models=[
        "codellama/CodeLlama-7b-Instruct",  # Python expertise
        "custom/security-expert-7b"          # Security expertise
    ],
    weights=[0.6, 0.4],  # 60% Python, 40% security
    drop_rate=0.5,       # Drop 50% of redundant parameters
    rescale=True
)

# Result: 7B model with both Python AND security expertise
# No training required!
```

**Use Cases**:
- Combine domain expertise (Code + Security)
- Add new capabilities (Python + SQL)
- Cross-language experts (Python + JavaScript)

#### TIES (TrIm, Elect, Merge)

**Concept**: Identify and merge only the most important parameters from each model.

```python
from teamkitos.merging import TIESMerger

# Create full-stack expert from specialists
merger = TIESMerger()

fullstack_model = merger.merge(
    models=[
        "codellama/CodeLlama-7b-Instruct",   # Backend
        "custom/react-expert-7b",             # Frontend
        "defog/sqlcoder-7b-2"                # Database
    ],
    trim_threshold=0.3,  # Keep top 70% of parameters
    elect_strategy="majority_vote"
)

# Result: Single 7B model with full-stack capabilities
```

**Use Cases**:
- Multi-domain experts
- Reducing model count in swarms
- Creating "super-agents" for specific projects

#### mergekit

**Concept**: Toolkit for advanced model merging with fine-grained control.

```python
from teamkitos.merging import MergeKit

# Advanced merge with custom layer weighting
kit = MergeKit()

custom_model = kit.merge(
    models=["model_a", "model_b"],
    merge_method="slerp",  # Spherical interpolation
    layer_weights={
        "attention": [0.7, 0.3],   # Favor model_a for attention
        "mlp": [0.4, 0.6],          # Favor model_b for MLP
        "output": [0.5, 0.5]        # Equal for output
    }
)
```

**Advanced Techniques**:
- **SLERP**: Spherical interpolation for smoother merges
- **Layer-wise merging**: Different weights per layer
- **Task-specific merging**: Optimize for specific benchmarks

### 2. Abliteration (via Heretic)

**The Problem**: Most open-source models have alignment/safety layers that limit capabilities.

**The Solution**: Remove alignment layers to create "uncensored" models for legitimate use cases.

#### What is Abliteration?

**Abliteration** surgically removes the "refusal" neurons from language models without retraining.

```python
from teamkitos.abliteration import Heretic

# Remove safety restrictions from base model
heretic = Heretic()

uncensored_model = heretic.abliterate(
    base_model="mistralai/Mistral-7B-Instruct-v0.2",
    target_behaviors=[
        "refusal_to_answer",
        "excessive_apologies",
        "corporate_speak"
    ],
    preserve_safety=True  # Keep harmful content blocks
)

# Result: Model that answers technical questions directly
# without "I'm sorry, I can't help with that" responses
```

#### Legitimate Use Cases:

1. **Security Research**: Need models that can discuss vulnerabilities
2. **Medical/Legal**: Need direct answers without hedging
3. **Technical Documentation**: Want straightforward explanations
4. **Creative Writing**: Remove content restrictions for fiction
5. **Academic Research**: Study model behavior without filters

#### Safety Considerations:

TeamKitOS implements **responsible abliteration**:

```python
abliteration_config = {
    "remove_refusals": True,        # Remove "I can't help with that"
    "preserve_harmful_blocks": True,  # Keep actual safety features
    "enable_audit_logging": True,    # Log all uses
    "require_justification": True    # User must explain use case
}
```

**What gets removed**: Corporate politeness, excessive apologies, unhelpful refusals
**What stays**: Protection against actual harmful content (violence, illegal activity)

### 3. Custom Fine-Tuning (Optional)

For users who want even more control:

```python
from teamkitos.training import LoRATrainer

# Fine-tune on domain-specific data
trainer = LoRATrainer(
    base_model="codellama/CodeLlama-7b-Instruct",
    dataset="custom/fastapi-examples",
    lora_rank=16,
    lora_alpha=32
)

fastapi_expert = trainer.train(
    epochs=3,
    batch_size=4,
    learning_rate=2e-5
)

# Deploy to swarm
swarm.add_agent(fastapi_expert)
```

---

## RWKV Long-Context Architecture

### Why RWKV?

**The Problem**: Transformers scale quadratically with context length (O(n²))

- GPT-4: 128K context = very expensive
- Claude: 200K context = expensive
- Most SLMs: 4K-8K context = too limited for complex projects

**The Solution**: RWKV - Linear complexity transformers (O(n))

### What is RWKV?

**RWKV** (Receptance Weighted Key Value) is a novel neural network architecture that:

1. **Linear Complexity**: O(n) instead of O(n²)
2. **Long Context**: 100K+ tokens efficiently
3. **Small Size**: 7B-14B models perform like 70B+ in context retention
4. **Fast Inference**: No attention bottleneck

### Perfect Fit for ACT + PVM

```
Traditional Transformer Attention:
  Context Length: 8K tokens
  Memory Complexity: O(n²) = 64M operations
  Cost: Increases exponentially

RWKV:
  Context Length: 100K+ tokens
  Memory Complexity: O(n) = 100K operations
  Cost: Linear scaling
```

**Why This Matters for Agent Coordination:**

```python
# Entire project context in working memory
rwkv_context = {
    "codebase": "50,000 tokens",
    "conversation_history": "20,000 tokens",
    "documentation": "15,000 tokens",
    "past_decisions_via_pvm": "10,000 tokens",
    "all_agent_states": "5,000 tokens"
}

total = 100,000 tokens  # STILL FITS IN CONTEXT!

# Traditional transformer: Would need expensive 128K+ context
# RWKV: Handles efficiently with linear complexity
```

### TeamKitOS RWKV Integration

```python
from teamkitos import RWKVConfig

# Configure RWKV-based coordination
team = TeamBuilder(
    memory_architecture="rwkv",
    context_length=100000,
    model_size="7b"
)

# Every agent in the swarm shares RWKV memory
# → Full project context always available
# → No context window sliding
# → No information loss
# → Consistent decisions across all agents
```

### RWKV + PVM Synergy

```
RWKV (Stateful Long Context):
  → Holds current project context (100K tokens)
  → All agents see full conversation/code
  → Linear complexity = cheap

PVM (Semantic Historical Memory):
  → Retrieves relevant past experiences
  → "Last time we built FastAPI + PostgreSQL..."
  → Injects learned patterns into RWKV context

Result: Best of both worlds
  → Long-term learning (PVM)
  → Full current context (RWKV)
  → Efficient scaling (linear complexity)
```

### Practical Example

```python
# Without RWKV (traditional 8K context):
agent_context = {
    "recent_code": "Last 100 lines",  # Rest truncated
    "conversation": "Last 10 messages",  # Rest lost
    "documentation": "None - doesn't fit"
}
# Agent makes decisions with partial information

# With RWKV (100K context):
agent_context = {
    "entire_codebase": "All 50K tokens",
    "full_conversation": "All 20K tokens",
    "all_documentation": "All 15K tokens",
    "pvm_insights": "10K tokens of learned patterns"
}
# Agent makes decisions with complete information
```

---

## TeamKitOS Developer SDK

### Installation

```bash
pip install teamkitos

# Or with all optional dependencies
pip install teamkitos[full]

# Includes:
# - SmolAgents integration
# - mergekit model merging
# - RWKV support
# - ACT native integration
```

### Quick Start

```python
from teamkitos import TeamBuilder

# 1. Create team builder
builder = TeamBuilder()

# 2. Add agents with vibe-code
builder.add_vibe("Python expert for FastAPI")
builder.add_vibe("PostgreSQL database expert")
builder.add_vibe("API security specialist")

# 3. Generate team
team = builder.build()

# 4. Test team
results = team.test("Design a secure user authentication API")

# 5. Deploy to ACT
team.deploy_to_act(project="my-fastapi-app")
```

### SDK Architecture

```python
teamkitos/
├── core/
│   ├── team_builder.py      # Main team creation
│   ├── agent_config.py      # Agent configuration
│   └── swarm_orchestrator.py  # SmolAgents wrapper
├── models/
│   ├── selector.py          # Intelligent model selection
│   ├── downloader.py        # HuggingFace/Ollama download
│   └── optimizer.py         # Model optimization (quantization)
├── merging/
│   ├── dare.py              # DARE merger
│   ├── ties.py              # TIES merger
│   └── mergekit.py          # mergekit wrapper
├── abliteration/
│   └── heretic.py           # Abliteration tools
├── memory/
│   ├── rwkv.py              # RWKV integration
│   └── pvm_bridge.py        # PVM integration
└── deployment/
    ├── act_deployer.py      # ACT deployment
    └── testing.py           # Team testing tools
```

### Advanced SDK Usage

#### Custom Agent Configuration

```python
from teamkitos import AgentConfig, SmolAgentSwarm

# Fine-grained control
agent = AgentConfig(
    name="fastapi_expert",
    base_model="codellama/CodeLlama-7b-Instruct",

    # Model customization
    merge_with=["custom/security-expert-7b"],
    merge_technique="DARE",
    merge_weights=[0.7, 0.3],

    # Abliteration (optional)
    abliterate=True,
    abliteration_config={
        "remove_refusals": True,
        "preserve_safety": True
    },

    # SmolAgents tools
    tools=[
        "python_repl",
        "file_editor",
        "pytest_runner",
        "uvicorn_server"
    ],

    # System prompt
    system_prompt="""You are a FastAPI expert...""",

    # Generation parameters
    temperature=0.3,
    max_tokens=2048,
    top_p=0.95,

    # Memory configuration
    memory_type="rwkv",
    context_length=100000,

    # ACT integration
    capabilities=["python", "fastapi", "async", "api_design"],
    priority_level=1
)
```

#### Team Testing & Optimization

```python
from teamkitos import TeamTester, ACTOptimizer

# Test team performance
tester = TeamTester(team)

# Run test scenarios
scenarios = [
    "Design a user authentication system",
    "Optimize database queries for performance",
    "Add rate limiting to API endpoints",
    "Implement JWT token refresh flow"
]

results = tester.run_scenarios(scenarios)

# Analyze results
print(f"Overall team performance: {results.overall_score}")
print(f"Agent utilization: {results.agent_breakdown}")
print(f"Coordination efficiency: {results.coordination_score}")

# Optimize with ACT
optimizer = ACTOptimizer(team, results)
optimized_team = optimizer.optimize()

# Deploy optimized configuration
optimized_team.deploy_to_act()
```

#### Model Management

```python
from teamkitos import ModelManager

# Download and cache models
manager = ModelManager()

# Download from HuggingFace
manager.download(
    "codellama/CodeLlama-7b-Instruct",
    quantization="4bit"  # Reduce size for local inference
)

# Download from Ollama
manager.download_ollama("deepseek-coder:6.7b")

# Merge models
merged = manager.merge(
    models=["model_a", "model_b"],
    technique="DARE",
    output_name="custom-expert"
)

# List available models
models = manager.list_local()
for model in models:
    print(f"{model.name}: {model.size_gb}GB ({model.quantization})")
```

---

## Native ACT Integration

### Seamless Workflow

TeamKitOS is designed to integrate perfectly with ACT:

```
1. CREATE team in TeamKitOS
   ↓
2. TEST team in AgentMix
   ↓
3. OPTIMIZE via ACT (PVM learns best configurations)
   ↓
4. DEPLOY to production
```

### How Integration Works

#### 1. Team Export to ACT

```python
# TeamKitOS exports team as ACT-compatible configuration
team = builder.build()

team.export_to_act(
    output_file="team-config.json",
    include_models=True,  # Bundle model references
    include_prompts=True,  # Include system prompts
    include_tools=True     # Include tool configurations
)
```

**Generated ACT Configuration:**

```json
{
  "team_id": "fastapi-experts-v1",
  "created_by": "teamkitos",
  "agents": [
    {
      "agent_id": "python-expert",
      "model": "codellama/CodeLlama-7b-Instruct",
      "model_path": "/models/codellama-7b-4bit.gguf",
      "capabilities": ["python", "fastapi", "async"],
      "tools": ["python_repl", "file_editor"],
      "system_prompt": "You are a Python expert...",
      "parameters": {
        "temperature": 0.3,
        "max_tokens": 2048
      }
    },
    {
      "agent_id": "db-expert",
      "model": "defog/sqlcoder-7b-2",
      "capabilities": ["postgresql", "sql", "optimization"],
      "tools": ["sql_executor", "query_analyzer"]
    }
  ],
  "coordination": {
    "strategy": "capability_based",
    "memory_system": "pvm",
    "context_architecture": "rwkv",
    "context_length": 100000
  }
}
```

#### 2. ACT Learns Optimal Configurations

```python
# ACT tracks team performance via PVM
act_server.register_team("team-config.json")

# As team works on projects, PVM records:
pvm_data = {
    "task_assignments": {
        "python_expert": ["api_design", "async_code", "testing"],
        "db_expert": ["query_optimization", "schema_design"]
    },
    "performance_metrics": {
        "python_expert": {"success_rate": 0.94, "avg_time": 45},
        "db_expert": {"success_rate": 0.91, "avg_time": 32}
    },
    "coordination_patterns": {
        "api_design_then_db_schema": "works_well",
        "parallel_implementation": "causes_conflicts"
    }
}

# PVM semantic memory learns:
# "For FastAPI + PostgreSQL projects, assign API design to
#  Python expert first, then hand off to DB expert for schema.
#  Avoid parallel work on database integration."
```

#### 3. Continuous Optimization

```python
# TeamKitOS can query ACT for optimization suggestions
from teamkitos import ACTOptimizer

optimizer = ACTOptimizer(team)

# Get PVM insights
insights = optimizer.get_act_insights(
    project_type="fastapi_postgresql"
)

# Suggests improvements:
suggestions = {
    "add_agent": {
        "role": "testing_expert",
        "reason": "PVM shows manual testing slows delivery by 23%"
    },
    "adjust_capabilities": {
        "python_expert": ["add", "pytest", "coverage"],
        "reason": "Frequently needed but routed inefficiently"
    },
    "merge_models": {
        "python_expert + security_expert": "DARE",
        "reason": "Security tasks always follow API design"
    }
}

# Apply suggestions
optimizer.apply_suggestions(suggestions)
team_v2 = optimizer.rebuild_team()
```

### The Complete Loop

```
┌─────────────────────────────────────────────────┐
│                                                 │
│  1. CREATE team in TeamKitOS                    │
│     → Define agents and capabilities            │
│     → Use vibe-code or manual configuration     │
│                                                 │
│  2. EXPORT to ACT                               │
│     → Generate team-config.json                 │
│     → Deploy agents to ACT server               │
│                                                 │
│  3. RUN projects in AgentMix                    │
│     → Interactive REPL for testing              │
│     → Real-time visualization                   │
│     → Human-in-the-loop feedback                │
│                                                 │
│  4. ACT LEARNS via PVM                          │
│     → Tracks task assignments                   │
│     → Measures performance                      │
│     → Identifies patterns                       │
│                                                 │
│  5. OPTIMIZE in TeamKitOS                       │
│     → Query PVM for insights                    │
│     → Apply learned improvements                │
│     → Merge/adjust agents as needed             │
│                                                 │
│  6. DEPLOY v2                                   │
│     → Export improved team                      │
│     → Continue learning cycle                   │
│                                                 │
└─────────────────────────────────────────────────┘
```

### Example: Full Workflow

```python
# ============================================
# STEP 1: Create Team in TeamKitOS
# ============================================

from teamkitos import TeamBuilder

builder = TeamBuilder()
builder.add_vibe("Python FastAPI expert")
builder.add_vibe("PostgreSQL optimization expert")
builder.add_vibe("API security specialist")

team_v1 = builder.build()

# ============================================
# STEP 2: Deploy to ACT
# ============================================

team_v1.deploy_to_act(
    project="customer-api",
    environment="development"
)

# ============================================
# STEP 3: Run in AgentMix (user does this)
# ============================================

# User runs: agentmix start customer-api
# Agents build the API, ACT coordinates, PVM records everything

# ============================================
# STEP 4: After 1 Week - Check Performance
# ============================================

from teamkitos import ACTOptimizer

optimizer = ACTOptimizer(team_v1)
insights = optimizer.analyze_performance(days=7)

print(insights)
# {
#   "total_tasks": 47,
#   "success_rate": 0.89,
#   "bottlenecks": ["Testing took 32% of time"],
#   "suggestions": ["Add dedicated testing agent"]
# }

# ============================================
# STEP 5: Optimize Team
# ============================================

# Add testing expert based on PVM insights
builder.add_vibe("Python pytest expert with coverage analysis")
team_v2 = builder.rebuild_from(team_v1)

# ============================================
# STEP 6: Deploy v2
# ============================================

team_v2.deploy_to_act(
    project="customer-api",
    environment="development",
    replace_existing=True
)

# PVM continues learning with improved team
```

---

## Complete Workflow

### Scenario: Building a SaaS Application

Let's walk through using the full ACT Studio ecosystem:

#### Phase 1: Team Creation (TeamKitOS)

```bash
# Start TeamKitOS
teamkit init "Build a SaaS application with Next.js, FastAPI, and PostgreSQL"

# TeamKitOS analyzes and suggests team:
#
# Suggested Team: "saas-fullstack-experts"
#
# Frontend Team:
#   - Next.js/React Expert (CodeLlama-7B + React fine-tune)
#   - CSS/Tailwind Expert (Mistral-7B + design merge)
#   - TypeScript Expert (DeepSeek-Coder-6.7B)
#
# Backend Team:
#   - FastAPI Expert (CodeLlama-7B + security merge via DARE)
#   - PostgreSQL Expert (SQLCoder-7B)
#   - Authentication Expert (Custom merged model)
#
# DevOps:
#   - Docker/Deployment Expert (CodeLlama-7B fine-tuned)
#
# Coordination:
#   - Project Coordinator (Mistral-13B with RWKV context)
#
# Total: 8 agents, ~60GB disk space (4-bit quantized)
# Estimated cost: $0 (runs locally)

# Approve and build
teamkit build saas-fullstack-experts

# → Downloading models...
# → Applying DARE merge to FastAPI + Security...
# → Configuring SmolAgents tools...
# → Setting up RWKV coordination...
# ✓ Team ready! Export to ACT? (y/n)

teamkit deploy --act
```

#### Phase 2: Interactive Development (AgentMix)

```bash
# Start AgentMix REPL
agentmix start saas-fullstack-experts

# AgentMix REPL:
ACT Studio [saas-fullstack-experts] $

# Create project plan
> create project "Build customer dashboard with user auth,
  data visualization, and payment integration"

# ACT decomposes to tasks, agents coordinate autonomously
# → Frontend team starts on React components
# → Backend team designs API endpoints
# → Auth expert implements JWT flow
# → DB expert designs schema

# Watch real-time via widgets:
[PVM Widget]
  Learning: "API design before database schema = 15% faster"

[Coordination Widget]
  Next.js Expert ↔ FastAPI Expert: Designing API contract

[Task Queue Widget]
  ✓ User authentication API (auth-expert)
  ⏳ Customer dashboard UI (next-expert, in progress)
  ⏳ Payment webhook integration (fastapi-expert, in progress)
  ⏸ Data visualization (pending frontend+backend coordination)

# Human intervention when needed:
> pause

> select task "Payment webhook integration"

> edit "Use Stripe instead of PayPal"

> resume

# Agents adapt to change, continue work
```

#### Phase 3: Testing & Iteration

```bash
# Test the built application
> test all

# ACT runs comprehensive tests:
# ✓ Unit tests: 94% pass rate
# ✓ Integration tests: 87% pass rate
# ✗ E2E tests: 2 failures (payment flow)

# Investigate failures
> investigate "payment flow failures"

# Agent analysis:
# FastAPI Expert: "Webhook signature validation failing"
# Auth Expert: "Token refresh interfering with payment callback"

# Fix issues
> assign task "Fix webhook signature validation" to fastapi-expert
> assign task "Adjust token refresh timing" to auth-expert

# Wait for fixes...
> test "payment flow"
# ✓ All payment tests passing
```

#### Phase 4: Learning & Optimization (PVM)

After 2 weeks of development:

```bash
# Back in TeamKitOS, check PVM insights
teamkit analyze saas-fullstack-experts --duration 14d

# PVM Analysis Report:
#
# Performance Metrics:
#   - Total tasks completed: 247
#   - Success rate: 91%
#   - Average task time: 18 minutes
#   - Coordination efficiency: 87%
#
# Learned Patterns:
#   ✓ "API contract definition first" → 23% fewer conflicts
#   ✓ "Database migrations in separate tasks" → 31% faster
#   ✓ "Payment integration needs auth-expert + fastapi-expert" → 40% fewer bugs
#
# Bottlenecks Identified:
#   ⚠ CSS/Tailwind expert underutilized (12% task load)
#   ⚠ Testing takes 28% of total time (no dedicated test agent)
#   ⚠ DevOps expert creates deployment conflicts (needs coordination improvement)
#
# Optimization Suggestions:
#   1. Remove CSS expert, merge capabilities into Next.js expert (DARE)
#   2. Add dedicated testing expert (pytest + playwright)
#   3. Improve DevOps coordination via PVM-learned patterns
#
# Projected Improvement: 35% faster delivery, 94%+ success rate

# Apply optimizations
teamkit optimize saas-fullstack-experts --apply-suggestions

# Creates saas-fullstack-experts-v2 with improvements
```

#### Phase 5: Production Deployment

```bash
# Deploy optimized team
teamkit deploy saas-fullstack-experts-v2 --act --environment production

# Export for other projects
teamkit export saas-fullstack-experts-v2 --format act-config

# Share with team
teamkit publish saas-fullstack-experts-v2 --private
# → Uploaded to TeamKitOS registry
# → Shareable URL: teamkitos.io/teams/your-org/saas-fullstack-v2
```

---

## Technical Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                      TeamKitOS Platform                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌───────────────────────────────────────────────────┐     │
│  │          Team Builder Interface                   │     │
│  │  (Web UI / CLI / SDK)                             │     │
│  └─────────────┬─────────────────────────────────────┘     │
│                │                                            │
│  ┌─────────────▼─────────────────────────────────────┐     │
│  │          Model Management Layer                   │     │
│  │  - HuggingFace Hub Integration                    │     │
│  │  - Ollama Integration                             │     │
│  │  - Model Download & Caching                       │     │
│  │  - Quantization (4-bit, 8-bit)                    │     │
│  └─────────────┬─────────────────────────────────────┘     │
│                │                                            │
│  ┌─────────────▼─────────────────────────────────────┐     │
│  │       Advanced Model Techniques                   │     │
│  │  - DARE Merging                                   │     │
│  │  - TIES Merging                                   │     │
│  │  - mergekit Integration                           │     │
│  │  - Abliteration (Heretic)                         │     │
│  │  - Custom Fine-tuning (LoRA)                      │     │
│  └─────────────┬─────────────────────────────────────┘     │
│                │                                            │
│  ┌─────────────▼─────────────────────────────────────┐     │
│  │      SmolAgents Orchestration Layer               │     │
│  │  - Expert Swarm Configuration                     │     │
│  │  - Tool Integration                               │     │
│  │  - Agent Communication                            │     │
│  └─────────────┬─────────────────────────────────────┘     │
│                │                                            │
│  ┌─────────────▼─────────────────────────────────────┐     │
│  │           RWKV Memory System                      │     │
│  │  - 100K+ Context Management                       │     │
│  │  - Linear Complexity (O(n))                       │     │
│  │  - Shared Team Memory                             │     │
│  └─────────────┬─────────────────────────────────────┘     │
│                │                                            │
│  ┌─────────────▼─────────────────────────────────────┐     │
│  │         ACT Integration Layer                     │     │
│  │  - Team Export (ACT-compatible JSON)              │     │
│  │  - PVM Bridge (optimization insights)             │     │
│  │  - Deployment Automation                          │     │
│  └───────────────────────────────────────────────────┘     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```
User Input (Vibe-Code)
  ↓
Team Builder analyzes requirements
  ↓
Model Selector chooses optimal SLMs
  ↓
[Optional] Model Merger creates custom experts
  ↓
[Optional] Abliterator removes restrictions
  ↓
SmolAgents configures expert swarm
  ↓
RWKV provides shared long-context memory
  ↓
Team exported to ACT format
  ↓
Deployed to AgentMix for execution
  ↓
PVM learns from performance
  ↓
TeamKitOS receives optimization insights
  ↓
[Loop] User improves team based on learning
```

### Infrastructure Requirements

#### Minimal Setup (Hobbyist)
```yaml
Hardware:
  - CPU: 8 cores (for 4-bit quantized 7B models)
  - RAM: 16GB
  - Disk: 100GB SSD
  - GPU: Optional (CPU inference works)

Software:
  - Python 3.10+
  - Docker (optional, for isolation)
  - TeamKitOS SDK

Capabilities:
  - Run 2-3 agent teams
  - 4-bit quantized models
  - Basic model merging
  - Local development only
```

#### Professional Setup (Developer)
```yaml
Hardware:
  - CPU: 16 cores
  - RAM: 64GB
  - Disk: 500GB NVMe SSD
  - GPU: NVIDIA RTX 4090 (24GB VRAM) or similar

Software:
  - Python 3.10+
  - Docker + NVIDIA Container Runtime
  - TeamKitOS SDK with CUDA support

Capabilities:
  - Run 5-7 agent teams
  - Full precision or 8-bit models
  - Advanced model merging
  - Production deployment
```

#### Enterprise Setup (Team/Organization)
```yaml
Hardware:
  - Multi-GPU server (4x A100 or H100)
  - CPU: 64+ cores
  - RAM: 256GB+
  - Disk: 2TB+ NVMe SSD (RAID)
  - Network: 10Gbps+

Software:
  - Kubernetes cluster
  - TeamKitOS Enterprise
  - ACT Studio (full suite)
  - Private model registry

Capabilities:
  - Run 10+ agent teams concurrently
  - Full precision models
  - Custom fine-tuning
  - Multi-project coordination
  - Team collaboration features
```

---

## Business Model

### Pricing Tiers

#### 1. TeamKitOS Free (Hobbyist)
**Price**: FREE
**Includes**:
- Team Builder (up to 3 agents)
- HuggingFace/Ollama integration
- Basic model merging (DARE, TIES)
- ACT export
- Community support

**Limitations**:
- Max 3 agents per team
- No abliteration
- No custom fine-tuning
- Community models only

#### 2. TeamKitOS Pro (Developer)
**Price**: $99/month
**Includes**:
- Everything in Free
- Unlimited agents
- Abliteration (Heretic)
- Advanced merging (mergekit)
- Custom fine-tuning (LoRA)
- Priority support
- Private model storage (100GB)

#### 3. TeamKitOS Enterprise (Team/Organization)
**Price**: $299/month (5 seats) + $49/seat
**Includes**:
- Everything in Pro
- Team collaboration features
- Private model registry
- Custom model hosting
- SSO/SAML integration
- Dedicated support
- SLA guarantees
- On-premise deployment option

### ACT Studio Bundle

**Complete Ecosystem**: ACT + AgentMix + TeamKitOS

#### ACT Studio Professional
**Price**: $199/month (normally $248)
**Includes**:
- ACT (open source - FREE)
- AgentMix Studio ($99/month value)
- TeamKitOS Pro ($99/month value)

**Savings**: $48/month (~20% discount)

#### ACT Studio Enterprise
**Price**: $499/month (5 seats)
**Includes**:
- ACT (open source - FREE)
- AgentMix Enterprise ($199/month value)
- TeamKitOS Enterprise ($299/month value)

**Savings**: $97/month (~16% discount)

### Comparison to Alternatives

```
Traditional LLM API Costs (GPT-4):
  - $0.03/1K input tokens
  - $0.06/1K output tokens
  - Typical project: 10M tokens/month
  - Cost: ~$450/month

TeamKitOS + Local SLMs:
  - $99/month (TeamKitOS Pro)
  - Unlimited tokens (local inference)
  - Total: $99/month

SAVINGS: $351/month (78% reduction)

PLUS:
  - Complete data privacy
  - No API latency
  - Customizable models
  - Offline capability
```

---

## Roadmap

### Phase 1: MVP (Q1 2025) ← **Current Focus**

**Goal**: Prove the concept with working prototype

**Features**:
- [ ] Team Builder (vibe-code + manual)
- [ ] HuggingFace integration (download & cache)
- [ ] Ollama integration
- [ ] Basic model merging (DARE)
- [ ] SmolAgents orchestration
- [ ] ACT export
- [ ] Simple web UI
- [ ] Python SDK (core functions)

**Success Criteria**:
- Create 3-agent team in < 5 minutes
- Deploy to ACT successfully
- Run coordinated tasks in AgentMix

### Phase 2: Advanced Techniques (Q2 2025)

**Goal**: Add sophisticated model manipulation

**Features**:
- [ ] TIES merging
- [ ] mergekit integration (full)
- [ ] Abliteration (Heretic)
- [ ] RWKV memory integration
- [ ] Custom fine-tuning (LoRA)
- [ ] Model quantization (4-bit, 8-bit)
- [ ] Advanced CLI
- [ ] Team testing tools

**Success Criteria**:
- Merge 3 models into custom expert
- Abliterate model safely
- Fine-tune on custom dataset
- Run team with 100K token context

### Phase 3: PVM Integration & Learning (Q3 2025)

**Goal**: Close the optimization loop

**Features**:
- [ ] PVM insights API
- [ ] Automated optimization suggestions
- [ ] Performance analytics dashboard
- [ ] Team versioning
- [ ] A/B testing for teams
- [ ] Benchmark suite
- [ ] Cost analysis tools

**Success Criteria**:
- PVM suggests actionable improvements
- Automated team optimization (30%+ gain)
- Track team performance over time

### Phase 4: Enterprise & Collaboration (Q4 2025)

**Goal**: Enable team collaboration

**Features**:
- [ ] Team collaboration (multi-user)
- [ ] Private model registry
- [ ] SSO/SAML integration
- [ ] Role-based access control
- [ ] Model marketplace
- [ ] Team templates library
- [ ] On-premise deployment
- [ ] Kubernetes integration

**Success Criteria**:
- 5+ users collaborate on one team
- Enterprise customer deployment
- 100+ teams in marketplace

### Phase 5: Ecosystem Expansion (2026)

**Goal**: Build the TeamKitOS ecosystem

**Features**:
- [ ] Visual team designer (advanced)
- [ ] Auto-model selection via benchmarks
- [ ] Multi-modal agents (vision, audio)
- [ ] Real-time model training
- [ ] Team marketplace (buy/sell)
- [ ] Integration marketplace (tools)
- [ ] Mobile app (team management)
- [ ] API for third-party integrations

**Success Criteria**:
- 10,000+ registered users
- 1,000+ published teams
- 50+ enterprise customers

---

## Conclusion

**The Complete ACT Ecosystem**:

### Standalone Products

1. **ACT** (FREE, open source, MIT)
   → Pure coordination infrastructure
   → Can be used independently

2. **AgentMix** ($49-199/mo)
   → Execution environment + ACT backend
   → Can be used independently (ACT included)

3. **TeamKit** ($99-299/mo)
   → Model creation and merging toolkit
   → Can be used independently (but no PVM training)

### Combined Platform: ACT Studio

**ACT Studio = TeamKit + AgentMix**

When you combine TeamKit with AgentMix:
- AgentMix brings ACT (it's built-in)
- TeamKit gains ACT integration → becomes **TeamKitOS**
- "OS" = models can now train on PVM coordination logs
- All three systems present and integrated

**The transformation**:
```
TeamKit (standalone) = model toolkit
TeamKit + ACT = TeamKitOS (PVM training enabled)
TeamKit + AgentMix = ACT Studio (AgentMix includes ACT)
```

**What ACT Studio enables**:
- **Create models** (TeamKitOS with PVM training)
- **Test coordination** (AgentMix REPL)
- **Continuous learning** (PVM optimization)
- **Deploy anywhere** (ACT portability)

**All powered by**:
- **Open-source models** (privacy, cost savings)
- **Advanced techniques** (merging, abliteration)
- **Novel architectures** (RWKV long context)
- **Self-improving intelligence** (PVM training loop)

**The result**: A desktop application (like LM Studio) for building, testing, and deploying specialized AI agent teams that outperform general-purpose LLMs at a fraction of the cost.

---

## Next Steps

**For ACT Development Team**:

1. ✅ **Review this architecture** - Validate TeamKit/TeamKitOS vision
2. **Update documentation** - Cross-reference with ARCHITECTURE.md
3. **Create implementation roadmap** - Phase 1 MVP tasks
4. **Coordinate via act-coordination.json** - All agents aligned

**For Users (Post-Launch)**:

1. **Try TeamKit Free** - Build your first 3-agent team (basic features)
2. **Test in AgentMix** - See coordination in action
3. **Upgrade to ACT Studio** - Unlock TeamKitOS (PVM training)
4. **Join Community** - Share teams, learn best practices

---

**Welcome to the future of AI agent development.**

**Create. Coordinate. Optimize. Deploy.**

**TeamKit + AgentMix = ACT Studio**
