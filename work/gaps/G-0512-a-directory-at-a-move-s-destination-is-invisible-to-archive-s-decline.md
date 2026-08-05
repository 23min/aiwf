---
id: G-0512
title: A directory at a move's destination is invisible to archive's decline
status: open
priority: medium
discovered_in: M-0286
---
## What's missing

`aiwf archive` enumerates both ends of every candidate move, but the walk that
does it contributes only the *files* beneath a directory end. A destination
that is a bare directory, or a directory whose contents are committed and
clean, therefore contributes nothing to the carried set, no blocker is found,
and the move stays in the plan. Applying it fails:

```
moving work/gaps/G-NNNN-<slug>.md -> work/gaps/archive/G-NNNN-<slug>.md:
rename ...: file exists
```

Three arrangements reproduce it: an empty directory at a flat-file
destination, an empty directory at a directory destination, and a
committed-clean file inside a directory destination.

## Why it matters

This is the whole-verb failure for one participant that the per-candidate
decline exists to replace, reached through a path the decline does not
enumerate. The sweep offers a plan it cannot land, so a dry run promises work
the apply refuses.

Severity is bounded and no data is lost: the apply's rollback leaves HEAD
where it was and every other candidate at its pre-apply path. The cost is a
failed verb and a misleading dry run, not damage.

The decline pass judges destination *divergence* — a file that is untracked,
or recorded and modified. A directory is neither, which is why it slips
through a check that otherwise covers both ends of a move.

## Sketch

Treat a non-empty-or-existing destination as a blocker for that candidate,
independent of whether git considers anything under it divergent. The
condition is "something already occupies where this move lands", which is a
filesystem question rather than a record question, so it belongs alongside
the existing destination enumeration rather than inside the divergence
comparison.

The evidence is the three arrangements above, plus the general claim that a
plan the sweep offers is one it can land.
