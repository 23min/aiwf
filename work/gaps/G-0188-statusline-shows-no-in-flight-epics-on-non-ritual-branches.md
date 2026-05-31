---
id: G-0188
title: Statusline shows no in-flight epics on non-ritual branches
status: open
---
## Problem

The statusline's entity segments (epic, milestone, gap) are branch-derived:
they only appear on ritual branches (`epic/E-*`, `milestone/M-*`,
`patch/[Gg]-*`). On `main` or any non-ritual branch, the entity slots are
empty — the operator has no visibility into which epics are in flight.

This is the most informative slot in the statusline, and it's blank in the
most common context (working on `main`).

## Desired behavior

Show all non-terminal epics (proposed, active) on every branch, using the
canonical glyph/color language from `aiwf status --worktrees`:
- `→` active (yellow)
- `○` proposed/draft (blue)
- Terminal statuses (`done`, `cancelled`) filtered out.

On ritual branches, accentuate the current epic (the one the branch belongs
to) — e.g. bold or a `▸` pointer — and show its milestone/gap inline. Other
in-flight epics render in the same row but visually secondary.

Cap at ~3 shown with `+N` overflow to keep the statusline scannable.

## Performance note

Scanning `work/epics/*/epic.md` means globbing + `awk` per file on every
render. Should be fine for 1–5 epics; the cap keeps it bounded.

## References

- Statusline script: `.claude/statusline.sh`
- Glyph definitions: `internal/render/glyph.go`
