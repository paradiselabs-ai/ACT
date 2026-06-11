#!/bin/bash

# PostToolUse hook: fires after a git commit bash call fails or is blocked.
# Reinforces the instruction so Claude corrects on the next attempt.

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
EXIT_CODE=$(echo "$INPUT" | jq -r '.tool_result.exit_code // "0"')
STDOUT=$(echo "$INPUT" | jq -r '.tool_result.stdout // empty')
STDERR=$(echo "$INPUT" | jq -r '.tool_result.stderr // empty')

# Only act on git commit calls that failed
if echo "$COMMAND" | grep -q 'git commit'; then
  if [ "$EXIT_CODE" != "0" ] || echo "$STDERR" | grep -qi 'blocked\|co-authored'; then
    jq -n '{
      "additionalContext": "Your git commit was BLOCKED because it contained a Co-Authored-By or AI attribution line. Fix: rewrite the commit message with ZERO attribution lines — no Co-Authored-By, no Signed-off-by, no AI credit. The human is the sole author of all commits. Try again now."
    }'
  fi
fi

exit 0
