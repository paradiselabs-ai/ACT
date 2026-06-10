#!/bin/sh
# Marks an artifact fresh at the current HEAD. Run ONLY after actually performing
# the artifact's refresh procedure (docs/constitution/UPDATE_LOOPS.md §3) —
# this script records the claim; the agent/human does the verification.
# Usage: scripts/freshness-refresh.sh <artifact-name>
set -e
[ -n "$1" ] || { echo "usage: $0 <artifact-name>"; exit 2; }
REPO_ROOT="$(git rev-parse --show-toplevel)"
REGISTRY="$REPO_ROOT/docs/constitution/freshness.json"

python3 - "$REGISTRY" "$1" "$(git -C "$REPO_ROOT" rev-parse --short HEAD)" <<'PY'
import json, sys

registry_path, name, head = sys.argv[1:4]
with open(registry_path) as f:
    reg = json.load(f)

art = reg.get("artifacts", {}).get(name)
if art is None:
    known = ", ".join(reg.get("artifacts", {}))
    sys.exit(f"unknown artifact '{name}'. Known: {known}")

art["status"] = "fresh"
art["verified_against"] = head
art["staled_by"] = None
with open(registry_path, "w") as f:
    json.dump(reg, f, indent=2)
    f.write("\n")
print(f"freshness: '{name}' marked fresh at {head}")
PY
