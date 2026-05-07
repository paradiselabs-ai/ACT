# Architecture Patterns to Adopt

Patterns identified from Claude Code's leaked source (March 2026) worth implementing independently in ACT's Go/TypeScript codebase. No proprietary code — concepts only.

---

## 1. Four-Tier Context Compaction

**Problem:** Agent conversations grow until they exceed the context window, causing failures or degraded quality.

**Pattern:** Four strategies applied in escalating order:

| Tier | Name | When | How |
|------|------|------|-----|
| 1 | **Proactive** | Token count approaches limit | Summarize older messages before the next API call. Keep recent messages intact, compress history. |
| 2 | **Reactive** | `prompt_too_long` error from API | Catch the error, compact retroactively, retry the call. |
| 3 | **Snip** | Headless/SDK modes | Truncate aggressively — headless agents don't need full history, just recent context. |
| 4 | **Context Collapse** | Mid-conversation verbose tool results | Compress specific tool outputs (large file reads, long bash output) while preserving a "commit entry" that can selectively restore them if needed later. |

**Where in ACT:** Agent loop (`internal/llm/agent/agent.go`). Currently ACT has basic truncation. Implement tiers 1 and 4 first — proactive summarization and collapsing verbose tool results.

**Reference:** [Inside Claude Code's Architecture](https://dev.to/oldeucryptoboi/inside-claude-codes-architecture-the-agentic-loop-that-codes-for-you-cmk)

---

## 2. Deferred Tool Discovery

**Problem:** Loading all tool schemas into the system prompt consumes massive tokens. Claude Code has 60+ tools — injecting all schemas would eat ~47K tokens per turn.

**Pattern:** Only load a core set of tools into the initial prompt. Register the rest as "deferred" tools with just a name and one-line description. Provide a `ToolSearch` meta-tool that agents call to fetch full schemas for deferred tools on demand.

**How it works:**
1. System prompt includes ~15 core tools (Read, Edit, Write, Bash, Grep, Glob, Agent)
2. System prompt also lists ~45 deferred tool names with one-line descriptions
3. Agent sees `mcp__slack__send_message — Send a message to a Slack channel` but can't call it yet
4. Agent calls `ToolSearch("slack send")` → gets the full JSON schema back
5. Now the agent can call `mcp__slack__send_message` with proper parameters

**Where in ACT:** System prompt construction (`internal/llm/prompt/`). ACT already uses the `act` CLI (~50-100 tokens) instead of MCP schemas (~47K tokens), so the problem is partially solved. But as MCP tools grow, deferred discovery prevents prompt bloat.

**Reference:** [Claude Code ToolSearchTool](https://deepwiki.com/hangsman/claude-code-source) — search for "ToolSearchTool" and "deferred"

---

## 3. Pre/Post Tool Use Hooks

**Problem:** Need to intercept tool calls for permissions, logging, file locking, or modification before/after execution — without coupling the tool implementation to the coordination logic.

**Pattern:** Event hooks at two lifecycle points:

```
PreToolUse(toolName, input) → { allow | deny | modify input }
PostToolUse(toolName, input, output) → { log | inject context | trigger side effects }
```

**Use cases in ACT:**
- **PreToolUse → file locking:** Before `Edit` or `Write`, auto-claim the file via `act-agent files claim`. If another agent holds it, deny or queue.
- **PreToolUse → permission gates:** Observer or Assurance can block dangerous operations on production files.
- **PostToolUse → ChronLog:** After any tool call, log the event to the coordination log for Observer to monitor.
- **PostToolUse → progress tracking:** After `Bash(npm test)`, extract pass/fail counts and update task progress.

**Where in ACT:** Tool execution layer (`internal/llm/tools/`). Add a `HookRunner` that wraps each tool's `Call()` method.

**Reference:** [Claude Agent SDK Hooks](https://platform.claude.com/docs/en/agent-sdk/hooks)

---

## 4. Static/Dynamic System Prompt Split

**Problem:** System prompts are expensive. Anthropic's API supports prompt caching, but only for content that doesn't change between requests.

**Pattern:** Split the system prompt into two halves at a boundary marker:

| Half | Contents | Cache behavior |
|------|----------|---------------|
| **Static** | Identity, role description, tool schemas, coordination protocols, constraints | Cached across requests (same hash = cache hit) |
| **Dynamic** | CLAUDE.md content, git status, current session state, project-specific context | Changes per request, never cached |

The static half is hashed (Blake2b). If the hash matches a previous request, the API returns a cache hit — saving time and tokens.

**Where in ACT:** Prompt construction (`internal/llm/prompt/prompt.go`). The Go system prompts (planner.go, observer.go, etc.) are already static. The dynamic part is the context loaded from CLAUDE.md/ACT.md and project state. Formalize this split so providers can cache the static portion.

**Reference:** [Anthropic Prompt Caching](https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching)

---

## 5. autoDream Memory Consolidation

**Problem:** Agent memory (project context, user preferences, learned patterns) grows unbounded. Stale or redundant memories waste tokens and mislead agents.

**Pattern:** A 4-phase background process that runs periodically (e.g., after 24h + N sessions):

| Phase | Name | What it does |
|-------|------|-------------|
| 1 | **Orient** | Scan all memory files. Identify categories, sizes, staleness. |
| 2 | **Gather** | Load recent session transcripts and outcomes. Extract learnings. |
| 3 | **Consolidate** | Merge duplicate memories. Update stale ones. Create new memories from recent learnings. |
| 4 | **Prune** | Delete memories that are redundant, outdated, or below a relevance threshold. Keep total under a budget (e.g., 200 lines / 25KB). |

Concurrency: uses a lock file to ensure only one consolidation runs at a time across sessions.

**Where in ACT:** PVM (PAIRed Vector Minutes) system. PVM already stores coordination patterns and skill profiles. autoDream would be a periodic job that consolidates PVM entries, prunes stale agent briefs, and updates the skill graph.

**Reference:** [Claude Code autoDream](https://sathwick.xyz/blog/claude-code.html) — search for "autoDream"

---

## Implementation Priority

| Pattern | Priority | Effort | Impact |
|---------|----------|--------|--------|
| Pre/Post Tool Hooks | **High** | Medium | Enables file locking, ChronLog, permissions — core coordination |
| Context Compaction (Tier 1+4) | **High** | Medium | Prevents context window crashes in long sessions |
| Static/Dynamic Prompt Split | **Medium** | Low | Cost savings via prompt caching |
| Deferred Tool Discovery | **Low** | Low | ACT CLI already solves this; needed when MCP tools grow |
| autoDream Consolidation | **Low** | High | PVM pruning; important long-term, not urgent |
