#!/usr/bin/env bash
# =============================================================================
# ACT Stop Hook — Autonomous coordination loop for Claude Code instances
#
# Fires on every Claude Code Stop event. Drives the autonomous agent loop:
#   1. Infinite loop guard
#   2. Check inbox for direct messages (highest priority)
#   3. Check for assigned task (with PVM + parallel context injection)
#   4. Proactive interface check when idle but others are working
#   5. Truly idle — exit 0, let Claude stop
#
# No configuration required. Agent identity is resolved per-session:
#   act-session-start.sh exports CLAUDE_SESSION_ID via CLAUDE_ENV_FILE
#   register_with_act MCP tool writes ~/.act/sessions/<session_id>
#   This hook reads session_id from stdin JSON → looks up that file
#
# Non-ACT sessions: hook exits 0 silently with zero side effects.
#
# Optional env vars:
#   ACT_SERVER_URL — ACT server (default: http://localhost:8080)
# =============================================================================

set -euo pipefail

ACT_SERVER="${ACT_SERVER_URL:-http://localhost:8080}"
COMPLEXITY_THRESHOLD=4
PROACTIVE_INTERVAL=120  # seconds between proactive interface checks

# ─── Read Stop hook input from stdin ─────────────────────────────────────────

INPUT=$(cat)

# Extract session_id and stop_hook_active from stdin JSON
SESSION_ID=$(echo "$INPUT" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print(d.get('session_id', ''))
except:
    print('')
" 2>/dev/null || echo "")

STOP_HOOK_ACTIVE=$(echo "$INPUT" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print(str(d.get('stop_hook_active', False)).lower())
except:
    print('false')
" 2>/dev/null || echo "false")

# ─── Resolve agent ID from session-scoped file ───────────────────────────────
# register_with_act writes ~/.act/sessions/<session_id> when an agent registers.
# If that file doesn't exist, this is not an ACT session — exit silently.

AGENT_ID=""
if [[ -n "$SESSION_ID" ]]; then
  SESSION_FILE="${HOME}/.act/sessions/${SESSION_ID}"
  if [[ -f "$SESSION_FILE" ]]; then
    AGENT_ID=$(cat "$SESSION_FILE" 2>/dev/null || echo "")
  fi
fi

# Fallback: explicit env var (for runner-based agents or manual override)
if [[ -z "$AGENT_ID" ]]; then
  AGENT_ID="${ACT_AGENT_ID:-}"
fi

if [[ -z "$AGENT_ID" ]]; then
  # Not an ACT session — let Claude stop normally, zero overhead
  exit 0
fi

PROACTIVE_STAMP_FILE="/tmp/act-proactive-${SESSION_ID:-${AGENT_ID}}.ts"

# ─── 1. Infinite loop guard ──────────────────────────────────────────────────

if [[ "$STOP_HOOK_ACTIVE" == "true" ]]; then
  exit 0
fi

# ─── Helpers ─────────────────────────────────────────────────────────────────

act_get() {
  # act_get <path> [query_string]
  local path="$1"
  local qs="${2:-}"
  curl -sf --max-time 5 "${ACT_SERVER}${path}${qs}" 2>/dev/null || echo "{}"
}

block_with() {
  # Emit a block decision with the given reason to stdout
  local reason="$1"
  python3 -c "
import json, sys
print(json.dumps({'decision': 'block', 'reason': sys.argv[1]}))
" "$reason"
  exit 0
}

complexity_score() {
  # Score task complexity from title + description
  # Uses python3 for lowercase (portable — avoids bash 4.0-only ${text,,})
  local text="$1"
  python3 -c "
import sys
text = sys.argv[1].lower()
score = 0

# +1 per 100 chars, cap at 5
len_score = min(len(text) // 100, 5)
score += len_score

# Complexity keywords
keywords = ['refactor', 'architect', 'implement', 'integrate', 'design', 'migrate',
            'optimize', 'overhaul', 'rewrite', 'scaffold', 'system', 'pipeline',
            'coordinate', 'orchestrate', 'framework', 'infrastructure', 'strategy']
for kw in keywords:
    if kw in text:
        score += 1

print(score)
" "$text" 2>/dev/null || echo "0"
}

# ─── 2. Check inbox for direct messages ──────────────────────────────────────

MESSAGES_RAW=$(act_get "/api/agents/${AGENT_ID}/messages" "?limit=5")
MESSAGE_COUNT=$(echo "$MESSAGES_RAW" | python3 -c "
import sys, json
d = json.load(sys.stdin)
msgs = [m for m in d.get('messages', []) if m.get('type') == 'direct_mention']
print(len(msgs))
" 2>/dev/null || echo "0")

if [[ "$MESSAGE_COUNT" -gt 0 ]]; then
  MESSAGE_BLOCK=$(echo "$MESSAGES_RAW" | python3 -c "
import sys, json
d = json.load(sys.stdin)
msgs = [m for m in d.get('messages', []) if m.get('type') == 'direct_mention']
lines = []
for m in msgs:
    lines.append(f\"From {m.get('from', 'unknown')}: {m.get('message', '')}\")
print('\n'.join(lines))
" 2>/dev/null || echo "")

  REASON="You have ${MESSAGE_COUNT} direct message(s) from other agents that need a response.

MESSAGES:
${MESSAGE_BLOCK}

INSTRUCTIONS:
Read each message carefully. For each one, respond directly to the sender using the send_message MCP tool with the format: @senderId <your response>
Be concise and specific. If an agent is asking about an interface, API shape, type, or file contract — give them a concrete answer.
After responding to all messages, use get_task to check if you have work assigned."

  block_with "$REASON"
fi

# ─── 2.5 Check if own last task failed (need to broadcast + retry) ───────────
# If the most recent task assigned to this agent has status=failed and
# retryCount < MAX_RETRIES, the agent must broadcast the error so peers can help.
# This step fires BEFORE checking for assigned tasks, so the failing agent
# actively solicits help rather than silently stopping.

ALL_TASKS_25=$(act_get "/api/tasks")
FAILED_OWN_TASK=$(echo "$ALL_TASKS_25" | python3 -c "
import sys, json
try:
    tasks = json.load(sys.stdin).get('tasks', [])
    agent_id = '${AGENT_ID}'
    MAX_RETRIES = 3
    # Most recently started task owned by this agent that is failed and retriable
    own_failed = [
        t for t in tasks
        if t.get('assignedAgent') == agent_id
        and t.get('status') == 'failed'
        and t.get('retryCount', 0) < MAX_RETRIES
    ]
    if own_failed:
        # Pick the most recently started
        t = sorted(own_failed, key=lambda x: x.get('startedAt',''), reverse=True)[0]
        print(json.dumps({
            'id': t.get('id',''),
            'title': t.get('metadata',{}).get('title','') or t.get('description','')[:80],
            'description': t.get('description',''),
            'retryCount': t.get('retryCount', 0)
        }))
    else:
        print('')
except:
    print('')
" 2>/dev/null || echo "")

if [[ -n "$FAILED_OWN_TASK" && "$FAILED_OWN_TASK" != "{}" ]]; then
  FAILED_TITLE=$(echo "$FAILED_OWN_TASK" | python3 -c "import sys,json; print(json.load(sys.stdin).get('title',''))" 2>/dev/null || echo "unknown")
  FAILED_ID=$(echo "$FAILED_OWN_TASK"    | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")
  FAILED_RETRY=$(echo "$FAILED_OWN_TASK" | python3 -c "import sys,json; print(json.load(sys.stdin).get('retryCount',0))" 2>/dev/null || echo "0")

  REASON="Your last task FAILED and needs peer help before you can retry.

FAILED TASK: ${FAILED_TITLE} (id: ${FAILED_ID}, retry ${FAILED_RETRY}/3)

INSTRUCTIONS — do these steps in order:
1. Use send_message to broadcast what went wrong and what you specifically need:
   Format: 'status: Task \"${FAILED_TITLE}\" failed. Error: <what happened>. I need: <specific information or interface detail you are missing>. Can anyone help?'
2. Call get_messages to check if peer agents have already responded to you.
3. Once you have the information you need (or after waiting ~30s with no peer response), call the retry_task MCP tool:
   retry_task(task_id: \"${FAILED_ID}\", agent_id: \"${AGENT_ID}\")
   This resets the task to pending with an incremented retry counter.
4. Then call get_task to pick it back up and retry with the new information.

If retry_task returns permanentlyFailed: true, stop — the user has been notified in the REPL.
Direct messages from other agents always take priority — always check get_messages first."

  block_with "$REASON"
fi

# ─── 3. Check for assigned task ──────────────────────────────────────────────

TASK_RAW=$(act_get "/api/tasks/assigned" "?agent_id=${AGENT_ID}")
TASK_EXISTS=$(echo "$TASK_RAW" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print('true' if d.get('task') else 'false')
" 2>/dev/null || echo "false")

if [[ "$TASK_EXISTS" == "true" ]]; then

  TASK_INFO=$(echo "$TASK_RAW" | python3 -c "
import sys, json
d = json.load(sys.stdin)
t = d.get('task', {})
print(json.dumps({
  'id': t.get('id', ''),
  'title': t.get('title', ''),
  'description': t.get('description', ''),
  'requiredCapabilities': t.get('requiredCapabilities', [])
}))
" 2>/dev/null || echo "{}")

  TASK_TITLE=$(echo "$TASK_INFO" | python3 -c "import sys,json; print(json.load(sys.stdin).get('title',''))" 2>/dev/null || echo "")
  TASK_DESC=$(echo "$TASK_INFO"  | python3 -c "import sys,json; print(json.load(sys.stdin).get('description',''))" 2>/dev/null || echo "")
  TASK_CAPS=$(echo "$TASK_INFO"  | python3 -c "import sys,json; print(', '.join(json.load(sys.stdin).get('requiredCapabilities',[])))" 2>/dev/null || echo "")

  # Score complexity
  TASK_TEXT="${TASK_TITLE} ${TASK_DESC}"
  SCORE=$(complexity_score "$TASK_TEXT")
  CAP_COUNT=$(echo "$TASK_CAPS" | tr ',' '\n' | grep -c . || echo "0")
  TOTAL_SCORE=$(( SCORE + CAP_COUNT ))

  # PVM context for complex tasks
  PVM_SECTION=""
  if [[ "$TOTAL_SCORE" -gt "$COMPLEXITY_THRESHOLD" ]]; then
    PVM_QUERY="${TASK_TITLE} ${TASK_DESC:0:200}"
    PVM_ENCODED=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$PVM_QUERY" 2>/dev/null || echo "")
    PVM_RAW=$(act_get "/api/pvm/search" "?query=${PVM_ENCODED}&limit=5")
    PVM_RESULTS=$(echo "$PVM_RAW" | python3 -c "
import sys, json
d = json.load(sys.stdin)
results = d.get('results', [])
if not results:
    print('')
else:
    lines = []
    for i, r in enumerate(results, 1):
        msg = r.get('message', r)
        if isinstance(msg, dict):
            text = f\"[{i}] {msg.get('agent','unknown')} ({msg.get('type','event')}): {str(msg.get('message',''))[:250]}\"
        else:
            text = f\"[{i}] {str(msg)[:250]}\"
        lines.append(text)
    print('\n'.join(lines))
" 2>/dev/null || echo "")

    if [[ -n "$PVM_RESULTS" ]]; then
      PVM_SECTION="
RELEVANT PAST COORDINATION PATTERNS:
The following are semantically similar events from this team's coordination history.
Use them to avoid known pitfalls and build on approaches that worked:

${PVM_RESULTS}
"
    fi
  fi

  # Parallel agent landscape
  AGENTS_RAW=$(act_get "/api/agents")
  TASKS_RAW=$(act_get "/api/tasks")
  PARALLEL_SECTION=$(python3 -c "
import sys, json

agents_raw = '''${AGENTS_RAW}'''
tasks_raw  = '''${TASKS_RAW}'''
agent_id   = '${AGENT_ID}'

try:
    agents = json.loads(agents_raw).get('agents', [])
    tasks  = json.loads(tasks_raw).get('tasks', [])
except:
    print('')
    sys.exit()

active_tasks = [t for t in tasks if t.get('status') in ('in_progress', 'assigned') and t.get('assignedAgent') != agent_id]
other_agents = [a for a in agents if a.get('id') != agent_id]

if not other_agents:
    print('')
    sys.exit()

lines = []
for a in other_agents:
    agent_tasks = [t for t in active_tasks if t.get('assignedAgent') == a.get('id')]
    if agent_tasks:
        task_summaries = [t.get('title') or t.get('description','')[:80] for t in agent_tasks]
        lines.append(f\"  • {a.get('id')} ({a.get('name', a.get('id'))}): working on {', '.join(task_summaries)}\")
    else:
        lines.append(f\"  • {a.get('id')} ({a.get('name', a.get('id'))}): idle\")

print('\n'.join(lines))
" 2>/dev/null || echo "")

  PARALLEL_BLOCK=""
  if [[ -n "$PARALLEL_SECTION" ]]; then
    PARALLEL_BLOCK="
PARALLEL AGENTS (working concurrently on this project):
${PARALLEL_SECTION}
"
  fi

  REASON="You have a task assigned. Execute it now using your ACT MCP tools.

TASK: ${TASK_TITLE}
${TASK_DESC}
${TASK_CAPS:+REQUIRED CAPABILITIES: ${TASK_CAPS}}
${PVM_SECTION}${PARALLEL_BLOCK}
COORDINATION INSTRUCTIONS:
1. BEFORE writing any code — call get_messages to check your inbox. Direct messages from peer agents always take priority and may contain interface details you need.
2. Identify which parallel agents are building something that will interface with your task (shared APIs, types, data structures, files, modules).
3. Use send_message to reach out to those agents NOW with your intended interface design. Format: @agentId <message describing what you plan to build and what you need from them>. Don't wait until you're done.
4. Between each major step of your work — call get_messages again. Peers may have sent you important updates or interface corrections.
5. Use report_task_progress to update your status as you work.
6. When complete — use report_task_complete with a summary of what you built.
7. After completing — send a status: broadcast via send_message describing what you built and what interfaces/APIs/types are now available for other agents to use.
8. If your task fails — do NOT stop silently. Use send_message to broadcast what failed and what you need from peers. They will offer targeted help so you can retry."

  block_with "$REASON"
fi

# ─── 3.5 Check for unfinished work in the project ────────────────────────────
# If ANY tasks are still pending, assigned, or in_progress (but not ours),
# keep looping. Permanently failed tasks (retryCount >= 3) are excluded —
# those block forever and the user is notified separately by the REPL.
# Covers two cases:
#   - pending: blocked on deps that haven't cleared yet, or awaiting assignment
#   - assigned/in_progress: race condition window after task completion

TASKS_RAW_35=$(act_get "/api/tasks")
UNFINISHED_COUNT=$(echo "$TASKS_RAW_35" | python3 -c "
import sys, json
try:
    tasks = json.load(sys.stdin).get('tasks', [])
    agent_id = '${AGENT_ID}'
    MAX_RETRIES = 3
    unfinished = [
        t for t in tasks
        if t.get('status') in ('pending', 'assigned', 'in_progress')
        and t.get('assignedAgent') != agent_id
        # Exclude failed tasks that have exceeded max retries (permanently failed)
        # Note: 'pending' tasks with retryCount >= MAX_RETRIES shouldn't exist,
        # but guard against edge cases anyway
    ]
    # Also exclude ANY task that is 'failed' and permanently so (won't unblock)
    # This mainly covers own failed tasks that already fired step 2.5 above
    unfinished = [
        t for t in unfinished
        if not (t.get('status') == 'failed' and t.get('retryCount', 0) >= MAX_RETRIES)
    ]
    print(len(unfinished))
except:
    print(0)
" 2>/dev/null || echo "0")

if [[ "$UNFINISHED_COUNT" -gt 0 ]]; then
  block_with "There are ${UNFINISHED_COUNT} task(s) still in progress or waiting to be assigned in this project. Call get_task now. If a task is returned, execute it. If get_task returns null (no task assigned yet), wait 10 seconds using the Bash tool (sleep 10) then call get_task again — keep polling until a task is assigned to you. Do NOT stop between polls. Tasks may be blocked on dependencies that will clear soon."
fi

# ─── 4. Proactive interface check (idle, but others are working) ──────────────

# Rate-limit: only check once per PROACTIVE_INTERVAL seconds
NOW=$(date +%s)
LAST_CHECK=0
if [[ -f "$PROACTIVE_STAMP_FILE" ]]; then
  LAST_CHECK=$(cat "$PROACTIVE_STAMP_FILE" 2>/dev/null || echo "0")
fi
ELAPSED=$(( NOW - LAST_CHECK ))

if [[ "$ELAPSED" -gt "$PROACTIVE_INTERVAL" ]]; then
  AGENTS_RAW=$(act_get "/api/agents")
  TASKS_RAW=$(act_get "/api/tasks")

  WORKING_AGENTS=$(python3 -c "
import sys, json
try:
    agents = json.loads('''${AGENTS_RAW}''').get('agents', [])
    tasks  = json.loads('''${TASKS_RAW}''').get('tasks', [])
except:
    print('')
    sys.exit()

active = [t for t in tasks if t.get('status') in ('in_progress', 'assigned') and t.get('assignedAgent') != '${AGENT_ID}']
if not active:
    print('')
    sys.exit()

lines = []
for t in active:
    lines.append(f\"{t.get('assignedAgent','?')}: {t.get('title') or t.get('description','')[:100]}\")
print('\n'.join(lines))
" 2>/dev/null || echo "")

  if [[ -n "$WORKING_AGENTS" ]]; then
    echo "$NOW" > "$PROACTIVE_STAMP_FILE"

    REASON="You are idle. Other agents are actively working and may benefit from coordination with you.

AGENTS CURRENTLY WORKING:
${WORKING_AGENTS}

INSTRUCTIONS:
Review what you have recently completed. Check if any of your outputs, interfaces, APIs, or types are relevant to what these agents are currently building.
If you see a genuine interface dependency or can offer something useful:
  - Use send_message to reach out proactively: @agentId <what you built that's relevant and how they can use it>
  - Be specific — give them actual interface details, not vague offers to help
If there is nothing genuinely relevant, use get_task to check for new work instead.
Do not reach out just to check in — only if there is a real interface or dependency to discuss."

    block_with "$REASON"
  fi
fi

# ─── 5. Truly idle — let Claude stop ─────────────────────────────────────────

exit 0
