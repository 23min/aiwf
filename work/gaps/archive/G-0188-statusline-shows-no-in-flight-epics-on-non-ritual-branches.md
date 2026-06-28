---
id: G-0188
title: Statusline shows no in-flight epics on non-ritual branches
status: addressed
addressed_by:
    - M-0192
---
## Problem

The statusline's entity segments (epic, milestone, gap) are branch-derived:
they only appear on ritual branches (`epic/E-*`, `milestone/M-*`,
`patch/[Gg]-*`). On `main` or any non-ritual branch, the entity slots are
empty — the operator has no visibility into which epics are in flight.

This is the most informative slot in the statusline, and it's blank in the
most common context (working on `main`).

## Desired behavior

The epic HUD is **branch-contextual** — it answers a different question
depending on where you are:

- **On a ritual branch** (`epic/E-*`, `milestone/M-*`): show **only** the
  current epic — the one the branch belongs to — with its status glyph/color,
  and, on a milestone branch, its milestone inline. No other in-flight epics,
  no overflow. When you are working inside an epic, the HUD reflects *that*
  work, not the whole backlog.
- **On `main` / any non-ritual branch**: there is no current epic, so show the
  in-flight list — all non-terminal epics with the canonical glyph/color
  language (`→` active yellow, `○` proposed/draft blue; terminal `done` /
  `cancelled` filtered out), capped at ~3 with a `+N` overflow to stay
  scannable. This is the anti-blank case the gap was filed for.

This refines the original "show all epics everywhere, accentuate the current
one, render the rest as secondary" shape: on a ritual branch the secondary
list is noise — the `+N` worktree-count segment already signals that parallel
work exists — and a strict "current-only everywhere" would re-blank `main`,
defeating the gap's own motivation. Branch-context reconciles both.

## Performance note

On `main` the list path globs `work/epics/*/epic.md` plus an `awk` per file
each render. Fine for 1–5 epics; the cap keeps it bounded. On a ritual branch
only the single current epic (and its milestone) is read, so the cost is one
or two files.

## References

- Statusline script: `.claude/statusline.sh`
- Glyph definitions: `internal/render/glyph.go`
