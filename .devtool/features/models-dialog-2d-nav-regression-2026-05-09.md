---
id: "models-dialog-2d-nav-regression-2026-05-09"
status: "todo"
priority: "medium"
assignee: "kareem"
dueDate: null
created: "2026-05-09T08:50:00.000Z"
modified: "2026-05-09T08:50:00.000Z"
completedAt: null
labels: ["TUI", "regression", "ux", "decision-needed"]
order: "a0"
---
# Models dialog: 2D nav + vim keys + provider grouping lost in huh migration

## What changed

`f4fb47d "Migrate dialog UIs to huh form components"` rewrote `internal/tui/components/dialog/models.go` from 377 LOC custom 2D-nav implementation → 140 LOC `huh.NewSelect[...]` filter-based picker.

## Lost capabilities

| Old behavior | huh-migrated behavior |
|---|---|
| 2D navigation: `h`/`l` provider columns, `j`/`k` model rows | Single 1D filtered list |
| Per-axis scroll offsets (long model lists scrollable independent of provider list) | Single offset, full list |
| Provider grouping visible in layout | Provider name flattened into "provider/model" string |
| Vim keys (h/j/k/l) | Arrow keys / `/` filter |

## Decision required

This is a real UX trade-off, not pure regression. Two paths:

**A. Keep the huh-migrated version.**
- Smaller code, leverages huh's filter UX
- Loses keyboard-first power-user flow
- Acceptable if the user base expects modal pickers (most modern UIs)

**B. Restore custom 2D nav.**
- Old code preserved in git history at parent commit `4e83245`
- Restores power-user flow + vim keys
- More code to maintain

## Recommendation

Hold for user signal. If alpha users complain about the model picker being clunky, restore. Otherwise keep the simpler huh version. **Not an alpha blocker either way.**

If restoring: cherry-pick the old `models.go` from commit `4e83245`, adapt for any current API changes, keep huh in the rest of the dialogs.

## Constraint

This is Kareem's domain (TUI). d34d filed this kanban after audit findings; Kareem decides which path to take.
