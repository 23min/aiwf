---
id: G-0420
title: No sort-by-priority tiebreaker for aiwf list / aiwf status
status: open
priority: low
---
## Problem

`aiwf list` and `aiwf status` sort entities by id only — a flat `sort.SliceStable` on id string in both `internal/cli/list/list.go` and `internal/cli/status/status.go`. Once `priority` ships on gap and decision (G-0078), a user can filter by priority (`--priority high`) but can't get a ranked view: entities within the same lifecycle status don't sort by priority, so "show me every open gap, urgent first" still means eyeballing a filtered list by hand.

## Evidence

Deferred out of G-0078 on the grounds that no group-by-status-then-secondary-key sort infrastructure exists anywhere in the codebase today — building it speculatively, before real usage shows filtering alone is insufficient, would have been premature. G-0078's own Evidence section is about filtering/discovery friction ("picking which one to work next requires reading every body"), not a demonstrated need for a specific sort order.

## Direction

Once `priority` (G-0078) has shipped and seen some real use, reconsider:

- `aiwf list` / `aiwf status` gain a stable multi-key sort: group by lifecycle status (unchanged default grouping), priority as a secondary tiebreaker within each group, id as the final tiebreaker among entities of equal priority.
- Needs a decision on where an unset priority sorts relative to the four levels — e.g. treated as lower than `low`, or left in id order at the tail of its status group.
- No new frontmatter or check-rule work; this is CLI/sort-layer only, layered entirely on top of what G-0078 ships.

## Relationship to other gaps

- **G-0078** (no priority field on entities): this gap is the direct sort-ordering follow-up G-0078 deliberately deferred; it has no scope until G-0078 ships.
