# NesTTY Architecture — ACT Film Production Model

**Date:** March 2026
**Status:** Architecture pivot — replaces previous flat agent model
**Branch:** NesTTY

---

## Overview

ACT is pivoting from a flat "register agents and assign tasks" model to a structured **Film Production Architecture** with clearly defined roles, a two-tier agent hierarchy, and a new protocol called **NesTTY** for interactive upper-hierarchy coordination.

The film production metaphor is not cosmetic — it defines responsibility boundaries, communication patterns, and topology in a way that directly addresses the 17x error amplification problem documented in multi-agent systems research (arXiv:2512.08296).

---

## The Two-Tier Architecture

### Tier 1 — Upper Hierarchy (Interactive, NesTTY)

These agents run in a shared terminal via the NesTTY protocol. They are aware of each other, can communicate directly, and have HITL (Human-in-the-Loop) natively available. Their internal tool execution is offloaded to background TTY screens.

| Role | Responsibility |
|---|---|
| **Director** | Highest authority. Receives project from user. Decomposes into tasks. Assigns to Actors via A2A. Monitors overall progress via Operators. Informs user on completion. Previously called "ACTor." |
| **Operator** | Watches coordination logs and PVM data. Detects stuck, missed, or overlooked tasks. Flags anomalies to Director. The monitoring and context layer. |
| **Production Assistant** | Receives completed task outputs from Actors. Validates against original success criteria. Passes verified outputs to Producer. Rejects and re-queues failures. **This is NOT FLUX State — it is a separate, dedicated validation role.** |
| **Producer** | Receives verified task outputs from Production Assistant. Synthesizes individual pieces into a coherent final product. Manages the assembly of the complete deliverable. |

**Communication between Tier 1 agents:** NesTTY protocol (see below). Direct peer conversation in shared terminal window. HITL injected via standard `$>` prompt.

### Tier 2 — Execution Layer (Headless, Actors)

Actors are fully headless subprocess agents. There can be many of them. They are spawned, complete tasks, and are torn down. **Identity and context persist across respawns** via the ACT persistence stack.

| Role | Responsibility |
|---|---|
| **Actor** | Executes assigned tasks. Pre-established agent ID. Picks up context from PVM + Agent Cards (A2A) + AGENT.md on every spawn. Completely stateless at the process level, stateful at the coordination level. |

**Communication for Actors:**
- Receives tasks via **A2A** (Director pushes, no polling)
- Accesses ACT tools via **mcp2cli** (token-efficient, lazy-loaded)
- Persists identity via **PVM + Agent Cards + AGENT.md**

---

## FLUX State vs Production Assistant — Critical Distinction

These are frequently confused. They are completely separate concepts.

### Production Assistant (Assurance Plane)
- A **role in the architecture** — a dedicated agent in Tier 1
- Receives completed task output
- Checks: "Does this output meet the original success criteria?"
- Binary-ish validation: pass → Producer, fail → re-queue to Actor
- Operates at the **task output level**
- **Needs to be built** — not yet implemented

### FLUX State Reasoning
- A **cognitive process** — a deep epistemic reconstruction triggered for high-stakes or high-reasoning tasks
- Re-initializes an agent with full context EXCEPT memories of completing the specific task being evaluated
- As evaluation proceeds, causal decision memories are **selectively re-injected** via PVM knowledge graph traversal
- Re-injection is triggered by gaps in the evaluation — when a choice seems suboptimal, the decision trail behind that choice surfaces
- Two outcomes per flagged decision:
  - Agent sees the re-injected reasoning and confirms: "I understand why now, it was correct"
  - Agent still disagrees even with full context: gets to make the revision
- Operates at the **decision trail level**, not just output level
- Depends on real PVM with causal graph edges, not just flat vector similarity
- **Much more complex than Production Assistant. Builds on real PVM. Separate implementation track.**

Production Assistants may eventually *trigger* FLUX on flagged outputs, but they are not the same thing.

---

## NesTTY Protocol

**Name:** NesTTY (Nested + TTY)

**What it is:** A protocol for spawning a nested interactive REPL within a running REPL, where both agents share a single terminal window as their conversational interface while offloading all tool execution / subprocess work to separate background TTY screens.

### How it works

1. Agent A (e.g. Director/Claude Code) is running in a terminal
2. Agent A spawns Agent B (e.g. Operator/Gemini CLI)
3. NesTTY intercepts this spawn event instead of handing TTY control to Agent B directly
4. Both agents' internal tool execution is migrated to **background PTY windows** (full capability, just not the visible terminal)
5. The **original terminal window** becomes a shared conversational surface:
   - Agent A and Agent B talk to each other turn by turn
   - Turns are tagged: `[Director]:`, `[Operator]:`, `[Human]:`
   - Both agents are fully aware of each other (the spawn established the relationship)
6. The terminal's `$>` input prompt = **HITL injection point**
   - Human types → both agents receive it simultaneously
   - Both understand it as a third-party human interrupt
   - Context-aware handling: if related to current discussion → clarification; if new topic → redirect

### Implementation primitives
- `node-pty` or Python `pty` for PTY management
- Wrapper binary (fake `gemini`, fake `claude` etc.) earlier in PATH — intercepts spawn, triggers protocol instead of handing off TTY
- Conversation relay layer between agents
- Turn tagging for three-party awareness (`[Agent A]:`, `[Agent B]:`, `[Human]:`)

### Current status
- **Concept defined, not yet built**
- Depends on no other ACT infrastructure — can be built independently
- Will serve as the primary interface for Tier 1 upper hierarchy agents

---

## Communication Topology

```
User
  │
  ▼
┌─────────────────────────────────────────────────────┐
│  NesTTY Window (shared terminal)                    │
│                                                     │
│  Director ◄──────────────────► Operator            │
│      │                              │               │
│      └──────── Producer ◄── Production Assistant   │
└─────────────────────────────────────────────────────┘
         │ A2A task delegation (push, not poll)
         ▼
┌─────────────────────────────────────────────────────┐
│  Headless Actors (many, spawned as needed)          │
│                                                     │
│  Actor-1    Actor-2    Actor-3    Actor-N           │
│  [mcp2cli]  [mcp2cli]  [mcp2cli]  [mcp2cli]        │
│  [PVM]      [PVM]      [PVM]      [PVM]            │
└─────────────────────────────────────────────────────┘
         │ verified outputs
         ▼
  Production Assistant → Producer → Director → User
```

---

## Actor Persistence Stack

Each Actor maintains persistent identity across headless respawns:

```
Actor respawns → loads:
  1. Agent Card (A2A)     — who I am, what I can do, where to reach me
  2. AGENT.md             — project-specific brief, written by Director
  3. PVM query            — "what relevant patterns exist for this task type?"
  4. mcp2cli tool access  — lazy-loaded ACT tools, ~95% fewer tokens than MCP

Result: Actor picks up exactly where previous instance left off.
No context loss. No re-introduction. Stateless process, stateful agent.
```

---

## Functional Plane Mapping (17x Error Research)

From arXiv:2512.08296 — the architecture that reduces 17.2x error amplification to 4.4x:

| Functional Plane | ACT Role |
|---|---|
| Control | Director |
| Planning | Director + Producer |
| Context | Operator |
| Execution | Actors |
| Assurance | Production Assistant (+ eventually FLUX) |
| Mediation | Director + NesTTY conversation |

---

## What This Replaces

| Old concept | New concept |
|---|---|
| ACTor | Director |
| Generic "agents" | Actors (execution) + named Tier 1 roles |
| REPL-only task creation | Director-initiated via A2A |
| HTTP polling for tasks | A2A push to Actors |
| MCP schema injection | mcp2cli lazy discovery |
| No assurance layer | Production Assistant |
| No synthesis layer | Producer |
| No monitoring layer | Operator |
| No peer conversation | NesTTY |

The existing ACT server, MCP bridge, REPL, ChronologicalLog, and PVM infrastructure remain valid and are built upon — not replaced.

---

## Build Order

### Must come first (infrastructure)
1. Real PVM embeddings — Actor persistence depends on semantic retrieval
2. A2A protocol on server — Director → Actor task delegation
3. mcp2cli integration — Actor tool access

### Then the new architecture
4. Role taxonomy in codebase — Director/Actor/Operator/PA/Producer as first-class
5. NesTTY protocol — upper hierarchy shared terminal interface
6. Production Assistant — validation role, Assurance Plane
7. FLUX State — deep epistemic evaluation (depends on causal PVM graph)
8. Producer synthesis layer

---

## Related Documents

- `docs/ARCHITECTURE.md` — core system architecture (being updated)
- `docs/PVM_EXTENDED_CAPABILITIES.md` — PVM specification (FLUX depends on this)
- `docs/PAIR_REASONING_WORKFLOW.md` — PAIR reasoning (feeds into FLUX)
- `docs/INNOVATION_ANALYSIS.md` — competitive analysis (needs updating with new roles)
- `act-coordination.json` — see pivot notice entry
