---
id: "model-registry-simplification-2026-04-21"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["v1-gate", "models", "config", "block-9"]
order: "b03"
---
# Model Registry Simplification (supersedes Block 9)

`~/.act.json` = **sole source of model truth**. User writes provider IDs raw. No symbolic constants in code. No silent fallbacks. Provider errors surface raw.

**Purge**: hardcoded catalogs in `internal/llm/models/*.go`, synthetic model IDs, Tier 1 validator gates that block-unknown-models.

**Keep**: per-role model overrides, provider keyring, role → model resolution.

**Not Devin-suitable** — touches provider wiring, Tier 1 validator gates, `~/.act.json` parsing across >3 Go files. Human-led.

Sequence after Track A clears (refactor against green harness). See FUTURE_VISION.md "Model Registry Simplification" — supersedes the old "Dynamic Registry + Failure Wizard" in Block 9.
