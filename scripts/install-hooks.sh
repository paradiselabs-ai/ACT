#!/bin/sh
# Installs the versioned git hooks into .git/hooks/ (symlinks, idempotent).
# Run once per clone. See docs/constitution/UPDATE_LOOPS.md §4.
set -e
REPO_ROOT="$(git rev-parse --show-toplevel)"
HOOKS_SRC="$REPO_ROOT/scripts/git-hooks"
HOOKS_DST="$(git rev-parse --git-path hooks)"

for hook in "$HOOKS_SRC"/*; do
  name="$(basename "$hook")"
  chmod +x "$hook"
  ln -sf "$hook" "$HOOKS_DST/$name"
  echo "installed: $name -> $HOOKS_DST/$name"
done
