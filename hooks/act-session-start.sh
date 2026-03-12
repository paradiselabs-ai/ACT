#!/usr/bin/env bash
# =============================================================================
# ACT SessionStart Hook — Per-session identity setup
#
# Fires once when a Claude Code session starts. Reads the session_id from
# stdin and exports it as CLAUDE_SESSION_ID via CLAUDE_ENV_FILE so it's
# available to the register_with_act MCP tool during Bash tool execution.
#
# This is what enables per-instance agent identity:
#   register_with_act writes → ~/.act/sessions/<session_id>
#   act-stop-hook.sh reads  ← ~/.act/sessions/<session_id>
#
# No ACT_AGENT_ID env var needed. No shared ~/.act/agent-id collision.
# Non-ACT sessions: this hook is silent and exits 0 with no side effects.
# =============================================================================

set -euo pipefail

# Read the full JSON input from stdin
INPUT=$(cat)

# Extract session_id (portable — python3 always available where Claude Code runs)
SESSION_ID=$(echo "$INPUT" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print(d.get('session_id', ''))
except:
    print('')
" 2>/dev/null || echo "")

# Nothing to do without a session ID
if [[ -z "$SESSION_ID" ]]; then
  exit 0
fi

# Export CLAUDE_SESSION_ID into the session's Bash tool environment
# CLAUDE_ENV_FILE is set by Claude Code only during SessionStart hooks.
# Variables written here are sourced into all subsequent Bash tool calls.
if [[ -n "${CLAUDE_ENV_FILE:-}" ]]; then
  echo "export CLAUDE_SESSION_ID=${SESSION_ID}" >> "$CLAUDE_ENV_FILE"
fi

# Also write it to a temp file as a fallback for the Stop hook
# (Stop hooks get a fresh env — they can't read CLAUDE_ENV_FILE vars)
mkdir -p "${HOME}/.act/sessions"
echo "$SESSION_ID" > "/tmp/act-claude-session-$$"
# We store the session_id itself in a file keyed by a known name so
# the Stop hook can retrieve it without any env var dependency.
# The Stop hook reads session_id directly from its own stdin — this file
# is just belt-and-suspenders for debugging.

exit 0
