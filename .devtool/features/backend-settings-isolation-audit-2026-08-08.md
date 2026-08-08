---
id: "backend-settings-isolation-audit-2026-08-08"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-08T09:50:00.000Z"
completedAt: null
labels: ["acp", "runner", "gemini", "antigravity", "security"]
order: "a2"
---
# Settings isolation for gemini + antigravity backends (both tiers)

## Describe
Claude backends are clean-roomed on both tiers (Tier-1: _meta settingSources:[],
commit 4cfa6ab; Tier-2: --setting-sources '', commit 8007d1f). The other external
backends still auto-load operator config:
- gemini (Tier-1 ACP + Tier-2 one-shots): loads ~/.gemini/GEMINI.md and cwd
  GEMINI.md chain. TODAY the user-level file is EMPTY (verified 2026-08-08), so
  no live leak — but the mechanism is armed: any content added later silently
  injects into Assurance/QA verdicts and swarm tasks. No obvious disable flag in
  gemini --help (0.50.0); investigate settings contextFileName override, an env
  var, or an empty --include-directories workspace trick.
- antigravity (Tier-1 ACP + Tier-2 one-shots): agy has its own app-level config,
  project memory, and rules; what a bare `agy --print` (no --project) auto-loads
  is UNVERIFIED. Investigate before the next agy quota window (resets ~2026-08-15)
  so its e2e run is clean.

## Success Criteria
- For each backend×tier: documented (in this ticket) what operator-level config
  the spawned process loads, verified empirically (plant a marker instruction in
  the config file, run a one-shot, check whether output reflects it).
- Where a leak exists and a disable mechanism exists: wired + wire/arg test.
- Where no disable mechanism exists: KNOWN_LIMITATIONS.md entry telling users
  their personal CLI config bleeds into ACT roles on that backend.

## Constraints
- Marker-test cheaply (1-2 one-shot calls per backend); do NOT burn agy quota
  before the reset — agy verification waits for the window.
