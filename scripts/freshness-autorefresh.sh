#!/bin/sh
# Auto-refresh trigger: spawned (backgrounded) by the post-commit/post-merge staler
# when an artifact with "auto_refresh": true transitions fresh -> stale.
# Launches a headless high-spec Claude session that delta-refreshes the artifact
# from `git diff <verified_against>..HEAD`, verifies it per its method doc, stamps
# it fresh, and commits. Guarded by a lockfile + per-artifact debounce so active
# hacking can't trigger a refresh storm. Kill switch: ACT_NO_AUTOREFRESH=1.
# Rules: docs/constitution/UPDATE_LOOPS.md §2a.
#
# Usage: freshness-autorefresh.sh <artifact-name> [--dry-run]

set -e
[ -n "$1" ] || { echo "usage: $0 <artifact-name> [--dry-run]"; exit 2; }
ARTIFACT="$1"
DRY_RUN="${2:-}"
[ "$ACT_NO_AUTOREFRESH" = "1" ] && { echo "freshness-autorefresh: disabled (ACT_NO_AUTOREFRESH=1)"; exit 0; }

REPO_ROOT="$(git rev-parse --show-toplevel)"
REGISTRY="$REPO_ROOT/docs/constitution/freshness.json"
STATE_DIR="$HOME/.act/freshness"
mkdir -p "$STATE_DIR"
LOCK="$STATE_DIR/autorefresh-$ARTIFACT.lock"
STAMP="$STATE_DIR/autorefresh-$ARTIFACT.last"
LOG="$STATE_DIR/autorefresh-$ARTIFACT.log"

command -v python3 >/dev/null 2>&1 || { echo "freshness-autorefresh: python3 missing" >&2; exit 0; }
command -v claude  >/dev/null 2>&1 || { echo "freshness-autorefresh: claude CLI not on PATH — manual refresh required" >&2; exit 0; }

CFG="$(python3 -c "
import json, sys
reg = json.load(open('$REGISTRY'))
a = reg['artifacts'].get('$ARTIFACT')
if a is None: sys.exit('unknown artifact $ARTIFACT')
print(a.get('auto_refresh', False))
print(a.get('debounce_minutes', 60))
print(a.get('verified_against') or '')
print(a.get('refresh', ''))
print(' '.join(a.get('paths', [])))
")"
AUTO="$(echo "$CFG" | sed -n 1p)"
DEBOUNCE_MIN="$(echo "$CFG" | sed -n 2p)"
BASE_COMMIT="$(echo "$CFG" | sed -n 3p)"
METHOD="$(echo "$CFG" | sed -n 4p)"
PATHS="$(echo "$CFG" | sed -n 5p)"

[ "$AUTO" = "True" ] || { echo "freshness-autorefresh: '$ARTIFACT' has auto_refresh=false"; exit 0; }

# Debounce: skip if a refresh ran inside the window.
if [ -f "$STAMP" ]; then
  LAST=$(cat "$STAMP" 2>/dev/null || echo 0)
  NOW=$(date +%s)
  AGE_MIN=$(( (NOW - LAST) / 60 ))
  if [ "$AGE_MIN" -lt "$DEBOUNCE_MIN" ]; then
    echo "freshness-autorefresh: debounced ($AGE_MIN min since last run < $DEBOUNCE_MIN min) — the stale marker stays; next trigger or manual run picks it up"
    exit 0
  fi
fi

# Lock: one refresh per artifact at a time.
if [ -f "$LOCK" ] && kill -0 "$(cat "$LOCK" 2>/dev/null)" 2>/dev/null; then
  echo "freshness-autorefresh: already running (pid $(cat "$LOCK"))"
  exit 0
fi

RANGE_DESC="full history (no verified_against baseline — first refresh must be a full rebuild)"
[ -n "$BASE_COMMIT" ] && RANGE_DESC="$BASE_COMMIT..HEAD"

PROMPT="You are the automated freshness-refresh session for the '$ARTIFACT' artifact in $REPO_ROOT (spawned by the post-commit staler — see docs/constitution/UPDATE_LOOPS.md §2a).

SCOPE — you may edit ONLY: $PATHS and docs/constitution/freshness.json. Never any source code (code is the truth; artifacts get fixed to match code, never the reverse). Never print secrets.

PROCEDURE:
1. Read docs/constitution/freshness.json for this artifact's state and the method/refresh contract: $METHOD — read that contract fully and follow it exactly (status taxonomy, grep-upfront rule, verification command).
2. Baseline: $RANGE_DESC. Run: git diff --name-only $BASE_COMMIT..HEAD -- <watch paths from the registry> and git log --oneline $BASE_COMMIT..HEAD to scope what changed. Re-verify ONLY the claims/flows/components touched by those changes — plus any NEW endpoints/roles/protocol steps those commits introduced and any the commits removed. Every kept 'ok' status must have been re-grepped with a live file:line cite.
3. If the artifact has never been verified (empty baseline), this is a FULL rebuild per the method contract — do not delta.
4. Run the method doc's verification command; it must pass (zero bluffed-ok, zero unverified).
5. Run ./scripts/freshness-refresh.sh $ARTIFACT
6. Commit ONLY the artifact files + freshness.json with identity dead-developers <paradiselabs.ai@gmail.com>, message 'docs($ARTIFACT): auto-refresh — re-verified against \$(git rev-parse --short HEAD)'. No Co-Authored-By lines, no AI attribution.
7. Final output: a 5-line summary — what changed in the diff range, what you re-verified, what statuses changed, verification command output, commit hash."

if [ "$DRY_RUN" = "--dry-run" ]; then
  echo "freshness-autorefresh DRY RUN for '$ARTIFACT':"
  echo "  baseline: $RANGE_DESC | debounce: ${DEBOUNCE_MIN}min | method: $METHOD"
  echo "  would spawn: claude -p <refresh-prompt> --model opus (log: $LOG)"
  exit 0
fi

echo "$$" > "$LOCK"
date +%s > "$STAMP"
echo "freshness-autorefresh: spawning refresh session for '$ARTIFACT' (log: $LOG)"
(
  cd "$REPO_ROOT" || exit 1
  claude -p "$PROMPT" --model opus \
    --allowedTools "Read" "Grep" "Glob" "Edit" "Write" "Bash(git log:*)" "Bash(git diff:*)" "Bash(git show:*)" "Bash(git rev-parse:*)" "Bash(git status:*)" "Bash(git add:*)" "Bash(git commit:*)" "Bash(python3:*)" "Bash(./scripts/freshness-refresh.sh:*)" \
    >> "$LOG" 2>&1
  rm -f "$LOCK"
) &
exit 0
