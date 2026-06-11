#!/bin/bash

# PreToolUse hook: block git commits that contain Co-Authored-By lines.
# Inspects the Bash tool's command input for the forbidden pattern.
# Exit 2 = block the tool call. Exit 0 = allow.

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if echo "$COMMAND" | grep -qi 'Co-Authored-By'; then
  echo '{"decision":"block","reason":"BLOCKED: Remove the Co-Authored-By line from the commit message. You are not a co-author — the human is the sole author. Redo the commit without any AI attribution."}'
  exit 2
fi

# Remind on any git commit, even clean ones, as a safety net
if echo "$COMMAND" | grep -q 'git commit'; then
  jq -n '{
    "additionalContext": "REMINDER: Do NOT include any Co-Authored-By, Signed-off-by, or AI attribution lines in commit messages. The human is the sole author."
  }'
fi

exit 0
