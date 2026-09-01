---
id: G-0602
title: A conflict resolved inside a shipped skill escapes the provenance gate
status: open
discovered_in: M-0312
---
## What's missing

The provenance gate has two halves and the hole is in both. The CI backstop reads `git log
--name-only`, which emits no diff for a merge commit. The commit-msg guard exempts a merge
outright, because during one `git diff --cached` lists every path the merge brings in and
the commits being merged already own that content. Content a merge *introduces* — a
conflict resolved by writing new prose into a shipped `SKILL.md`, rather than by picking
one side — therefore exists in no commit either half examines, and reaches consumers with
no owning entity named.

Measured: a merge whose resolution writes brand-new prescriptive content into a watched
`SKILL.md`, committed with no trailers, produces no finding. Both parents carry their own
trailers; neither carries this content. Three merges in this history are that shape,
resolving inside the two wrap rituals, the reviewer agent card and the milestone-spec
template; none carries an `aiwf-entity:` trailer.

## Why it matters

The gate's premise is that a merge carries no content of its own, so the commits that do
carry it are in the range separately with their own trailers. That premise holds for an
ordinary merge and fails for a conflict resolution, which is exactly where two milestones
editing one ritual meet — the epic-to-trunk merge is the likely site.

## Options

`--diff-merges=first-parent` would surface merge-introduced content, at the cost of also
surfacing every path a merge brings in from the other side, which the trailered source
commits already account for. Distinguishing the two needs a comparison against both
parents rather than a flag.

At the commit-msg guard that comparison is available and measured: the paths staged against
`HEAD`, intersected with those staged against `MERGE_HEAD`, are exactly the merge-authored
set — empty for a clean merge and for a conflict resolved by taking one side verbatim, and
naming only the file a resolution writes new prose into. It costs one `git` invocation and
a set intersection, and it reads a single side parent, so an octopus merge drops the rest;
this history has none. Closing that half alone leaves the backstop's, so the two want
deciding together. Whether the exposure justifies the cost is the open question; the
alternative is to accept the hole and say so where the premise is stated.
