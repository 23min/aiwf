---
id: G-0686
title: A move the push makes outside one add-and-delete commit reads as a create
status: open
discovered_in: M-0331
---
## What's missing

The push gate in `internal/check/entity_body_section_dropped.go` follows an
entity across a push by its path chain, and links one path to the next only
where a single commit adds the new path and deletes the old one carrying the
entity's id or one of its prior ids. A move made any other way leaves a chain of
one path, and the entity is judged as a create: held to every required section,
with nothing it already lacked at base or on trunk exempt. Measured against a
scratch repository with an upstream, a gap missing `## Why it matters` at base and
on trunk, copied to a new path in one commit and removed from the old path in the
next — `aiwf check` expected no finding, observed
`entity-body-section-dropped` naming the second commit. The same move as one
`git mv` commit reports nothing. Two more shapes reach the same verdict: a hand
renumber to a new id whose frontmatter names no prior id, and a move that exists
nowhere but in a merge commit's own resolution, which the range log lists no
files for.

## Why it matters

Each shape refuses a push over debt the pusher did not create — the cost the push
seam's decision exists to avoid — and the refusal names a commit whose author did
nothing to the section. Every aiwf verb moves an entity in one commit, so the
shapes are hand work; that is the population this gate judges.
