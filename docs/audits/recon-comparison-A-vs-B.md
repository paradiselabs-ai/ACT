---
title: Recon A-vs-B comparison (working copy)
status: pointer
note: This is the working copy. The canonical comparator deliverable is the sibling file.
---

# Recon comparison — Path A vs Path B

→ **Canonical report: [`dual-path-recon-2026-06-10.md`](./dual-path-recon-2026-06-10.md)**

That file contains the full comparator pass:
- §1 CONVERGENT findings (CV1–CV11, every one re-grep'd by the comparator)
- §2 Path-A-unique (A1 MCP-bridge-overstated re-grep'd & promoted; A2/A3 demoted-but-carried)
- §3 Path-B-unique (B1 51-not-28 FIXED; B2 tsx-watch caveat; B3 brownfield 2-question; B4 socket.io; B5 verified-correct batch)
- §4 methodology notes (convergence held under re-grep; two path-citation errors caught, neither fatal)
- §5 combined leverage-ranked reconciliation plan (12 doc-only fixes, tagged convergent/A-only/B-only)

**Headline both paths share and the comparator reproduced by grep:** the 2026-06-10 report is
substantially correct at `bc0673e`; the apex gap is the uncommitted `antigravity`/`agy` Tier-1
backends (`app.go:105`) documented nowhere — a live dual-implementation hazard.

**First action slice:** reconciliation-plan items 1–4 (surface antigravity/agy divergence; rewrite
block6 ticket body; un-strike combined-analysis 3.5 to Planner-only; delete CLAUDE.md "Tier-2-only
backend" sentence). All doc-only; never touch code.
