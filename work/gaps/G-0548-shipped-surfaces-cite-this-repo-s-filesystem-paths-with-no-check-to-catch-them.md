---
id: G-0548
title: Shipped surfaces cite this repo's filesystem paths with no check to catch them
status: open
priority: medium
---
## What's missing

Shipped surfaces are held to a rule with three clauses: no real entity id, no
filesystem path of this repo, and no development history or rationale. Only
the first clause has a check. `skill-body-id` scans every markdown file under
the embedded skill, ritual, and guidance trees for id shapes, and fires at
error severity so the push blocks. The path clause and the history clause have
nothing.

The path clause is already violated. Roughly fourteen citations of this repo's
own tree survive across those surfaces — design docs under `docs/`, package
directories under `internal/`, source files under `cmd/`. Each points at a
location that does not exist in the consumer tree the file is materialized
into.

## Why it matters

The reasoning behind the id clause applies unchanged to the path clause. An id
is meaningless in a consumer repo and rots as the entity moves; a path into
this repo's source tree is meaningless there for exactly the same reason, and
rots the same way when a file moves. A consumer following one finds nothing,
and cannot tell whether the reference is stale or was never valid for them.

The asymmetry also makes the rule harder to hold than it looks. An author who
has internalized "no ids in shipped surfaces" gets a mechanical reminder every
time they slip; the same author writing a path gets none, so the two halves of
one rule drift apart in practice even when both are understood.

## Direction

The judgment to settle before writing anything is what counts as a path
citation, because the population is not uniform:

- A design-doc reference is the shape the id rule already carves out for a
  markdown link, and may deserve the same treatment.
- A package reference names a Go import path as much as a directory, and
  reads as a name rather than a location.
- A source-file reference points at a specific file and is the clearest
  violation.

Once that line is drawn the check is a near-mirror of `skill-body-id`, over
the same corpus, with the same masking of non-prose carriers. Whether it fires
at error like its sibling or lands advisory first is worth deciding against
the size of the existing violation set: fourteen is small enough to clean up
in the same change, which argues for error and no grandfathering.

A fourth option sidesteps the line instead of drawing it. Where a surface can
be authored to need no path at all, its check is a ban on the `/` character in
that file — exact, with nothing to tune, and no judgment about what counts as a
citation. The cost is a constraint on the prose: a slash-joined pair has to be
written out as two words, and a command example has to move to the verb's own
`--help`. This is a per-file trade, not a corpus-wide one; several shipped
skills name `aiwf.yaml`, and the guidance fragment names its own materialized
location, all legitimately. It is worth asking of each surface whether it needs
to name a location at all before assuming it needs the heuristic, because the
answer is sometimes no in places that look like they must: a document whose
whole subject is which records to read can describe them by property and carry
no slash in any line, frontmatter included.

The history-and-rationale clause stays a review-time judgment. It has no
stable machine shape, and the point of doing the path clause is that it does.

## Provenance

Counted 2026-08-05 while closing G-0542, which edited one of these surfaces
and relocated rows carrying two of the citations. That patch added none and
removed none; the condition predates it.

The per-file alternative in Direction rests on one observation, made 2026-08-19
against a 141-line shipped skill whose subject is how to review a specification:
`grep -c '/'` over that file returns `0`, frontmatter included. The document
needed no path to do its job, so the exact check was available to it and the
heuristic was never required. That skill has since been retired, which is why
the observation is recorded here rather than in its own test.
