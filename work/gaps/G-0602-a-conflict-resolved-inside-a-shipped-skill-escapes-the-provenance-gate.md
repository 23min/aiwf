---
id: G-0602
title: A conflict resolved inside a shipped skill escapes the provenance gate
status: open
discovered_in: M-0312
---
## What's missing

The skill-edit provenance backstop reads `git log --name-only`, which emits no diff for a
merge commit. Content a merge *introduces* — a conflict resolved by writing new prose into
a shipped `SKILL.md`, rather than by picking one side — therefore exists in no commit the
gate examines, and reaches consumers with no owning entity named.

Measured: a merge whose resolution writes brand-new prescriptive content into a watched
`SKILL.md`, committed with no trailers, produces no finding. Both parents carry their own
trailers; neither carries this content.

## Why it matters

The gate's premise is that a merge carries no content of its own, so the commits that do
carry it are in the range separately with their own trailers. That premise holds for an
ordinary merge and fails for a conflict resolution, which is exactly where two milestones
editing one ritual meet — the epic-to-trunk merge is the likely site.

## Options

`--diff-merges=first-parent` would surface merge-introduced content, at the cost of also
surfacing every path a merge brings in from the other side, which the trailered source
commits already account for. Distinguishing the two needs a comparison against both
parents rather than a flag. Whether the exposure justifies that cost is the open question;
the alternative is to accept the hole and say so where the premise is stated.
