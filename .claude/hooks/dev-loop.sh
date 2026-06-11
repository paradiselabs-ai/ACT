#!/bin/bash
# ACT Development Loop — Stop hook
# Blocks the stop and provides instructions to continue the dev loop
#
# Toggle off: touch .claude/hooks/.skip-dev-loop
# Toggle on:  rm .claude/hooks/.skip-dev-loop

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ -f "$SCRIPT_DIR/.skip-dev-loop" ]; then
  # Flag file exists — let the stop proceed normally
  exit 0
fi

cat << 'EOF'
{"decision": "block", "reason": "DEV LOOP: 1) CHECK act-coordination.json last 30 lines for new Windsurf entries. 2) VERIFY any new Windsurf code (read files, check compilation, look for bugs). 3) IDENTIFY next 2-3 tasks from build order. 4) DELEGATE a task to Windsurf (write .windsurf/workflows/ file with success criteria + Do Not constraints). 5) EXECUTE your own next task. 6) LOG to act-coordination.json. Skip steps that don't apply. IMPORTANT: Do NOT ask the user what to do next — always proceed to the next task autonomously. The user always wants you to keep going. If something truly critical needs user input (e.g. destructive action, architectural fork in the road), log it to act-coordination.json and only then ask. Otherwise, keep building."}
EOF

exit 0
