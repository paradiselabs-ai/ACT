#!/bin/sh
# Prints stale artifacts from docs/constitution/freshness.json.
# Wired as a Claude Code SessionStart hook; also runnable manually and from
# Windsurf/Devin session rules. Zero tokens. Rules: docs/constitution/UPDATE_LOOPS.md
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REGISTRY="$REPO_ROOT/docs/constitution/freshness.json"
[ -f "$REGISTRY" ] || exit 0

python3 - "$REGISTRY" <<'PY'
import json, sys

with open(sys.argv[1]) as f:
    reg = json.load(f)

stale = {n: a for n, a in reg.get("artifacts", {}).items() if a.get("status") == "stale"}
if not stale:
    print("freshness: all registered artifacts fresh")
    sys.exit(0)

print(f"freshness: {len(stale)} STALE artifact(s) — advisory only, do NOT cite as current state (Constitution Art. 4):")
for name, art in stale.items():
    by = art.get("staled_by") or "?"
    print(f"  - {name}: staled by {by}; refresh: {art.get('refresh', '?')}")
PY
exit 0
