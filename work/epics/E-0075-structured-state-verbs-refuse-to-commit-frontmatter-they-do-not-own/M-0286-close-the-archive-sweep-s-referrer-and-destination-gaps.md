---
id: M-0286
title: Close the archive sweep's referrer and destination gaps
status: draft
parent: E-0075
tdd: required
acs:
    - id: AC-1
      title: A referrer absent from the loaded tree but present at HEAD declines the move
      status: open
---
## Goal

Make the archive sweep's per-candidate decline decide from the record, so the class M-0284
recorded as closed actually is.

## Context

M-0284 gave `archive` a per-candidate decline: a move whose verdict rests on a mid-edit file is
declined and reported, while the rest of the sweep proceeds. The carried half of that decision
was corrected to read the record as well as the working tree. The referrer half was not, and it
drifts in both directions.

Measured after M-0284 shipped, a referrer whose frontmatter is momentarily unparseable — the
ordinary bless rhythm — lets the target's move land without its link rewrite, and HEAD keeps a
link to a path that no longer exists. No re-run repairs it, because an archived target leaves
the scan for good, and `aiwf check` reports zero errors.

G-0499 carries the measurement and the neighbouring defects that share its root cause.

## Acceptance criteria

Created via `aiwf add ac` at plan time.

### AC-1 — A referrer absent from the loaded tree but present at HEAD declines the move

## Constraints

- The decline predicate and the rewrite predicate are one rule, not two that agree by
  inspection. `moveBlockers` and `planArchiveRewrites` currently disagree on which entities are
  candidates, which archived entities count, and which end of a move is enumerated.
- No fix may reintroduce whole-verb refusal for one mid-edit participant; that is the behaviour
  the per-candidate decline exists to replace.
- `--dry-run` must predict `--apply`. A dry run that promises a sweep the apply refuses is a
  defect in its own right.

## Out of scope

- The comparison primitive (`gitops.DivergentPaths`) and the claim-side guard, both settled in
  M-0284.
- The commit path's filter-blindness (G-0498), which is about what blobs aiwf stores rather than
  which candidates a sweep declines.

## Dependencies

- M-0284 — the primitive and the decline machinery this corrects.

## References

- G-0499 — the measured defect and its three neighbours
- ADR-0038 — the per-candidate scoping
- M-0284 — where the class was recorded as closed