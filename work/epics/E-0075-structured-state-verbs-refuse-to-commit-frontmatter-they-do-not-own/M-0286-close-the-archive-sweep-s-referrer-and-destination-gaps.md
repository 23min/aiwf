---
id: M-0286
title: Close the archive sweep's referrer and destination gaps
status: in_progress
parent: E-0075
tdd: required
acs:
    - id: AC-1
      title: A referrer absent from the loaded tree but present at HEAD declines the move
      status: open
    - id: AC-2
      title: An archived referrer mid-edit does not block an unrelated candidate
      status: open
    - id: AC-3
      title: Both ends of a move are enumerated
      status: open
    - id: AC-4
      title: A working-copy-only link declines that candidate, and --dry-run predicts --apply
      status: open
    - id: AC-5
      title: The decline predicate and the rewrite predicate derive from one enumeration
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

### AC-1 — A referrer absent from the loaded tree but present at HEAD declines the move

A referrer that HEAD records and the loaded tree does not — deleted on disk,
hand-renamed, or carrying momentarily unparseable frontmatter — makes the move it
links into undecidable, so that candidate is declined and named in the sweep's
skip report while the rest of the sweep proceeds.

The unparseable-frontmatter route is the one that matters, because it is the
ordinary rhythm rather than an accident: mid-edit YAML is briefly invalid, and
editing an entity body in the working tree before blessing it is what the shipped
guidance recommends.

This is the criterion with unrecoverable damage behind it. The move currently
commits without its link rewrite, HEAD keeps a link to a path nothing occupies,
and no later run repairs it — `IsArchivedPath` excludes the archived target from
every subsequent scan, so the sweep reports the tree converged. Restoring the
referrer changes nothing.

### AC-2 — An archived referrer mid-edit does not block an unrelated candidate

A mid-edit file already under an `archive/` subdirectory is not a blocker. It
cannot lose a link that a live sweep would rewrite, because the rewrite pass
excludes archived entities; treating it as a blocker declines a candidate that has
nothing to do with it.

The two predicates disagree here today, and that disagreement is the whole defect:
the rewrite pass filters archived paths and the decline pass does not.

### AC-3 — Both ends of a move are enumerated

A move's destination is examined alongside its source. A file sitting at the
destination — untracked, or recorded and divergent — declines that candidate by
name.

Enumerating the source alone leaves the condition invisible to the decline and
live at the commit, where the guard refuses the whole verb. That whole-verb
refusal for one participant is exactly the behaviour the per-candidate decline
exists to replace, so reaching it through an unenumerated path is a defect in the
decline rather than in the guard.

### AC-4 — A working-copy-only link declines that candidate, and --dry-run predicts --apply

A link present in the working copy and absent at HEAD is a divergence the decline
accounts for. It currently is not: the decline reads HEAD's body and finds no
committed link to lose, while the rewrite pass reads the working copy and emits an
op for it — so the move stays in the plan and the commit-side guard refuses the
verb, naming no candidate.

The observable form is the agreement between the two modes: the candidates a dry
run reports as sweeping are the ones an apply sweeps, and the ones it reports as
skipped are the ones an apply skips. A dry run that promises a sweep the apply
refuses is a defect on its own terms, independent of which condition caused it.

### AC-5 — The decline predicate and the rewrite predicate derive from one enumeration

A candidate is declined if and only if a file its verdict rests on is mid-edit.
No arrangement of the tree makes one predicate count an entity the other does not.

The other four criteria each name one arrangement where the two disagree today.
This one is the claim that no fifth exists — which is why its evidence is a
property test over constructed tree states rather than a reproduction. Asserting
structurally that one function calls the other would pin the implementation and
leave the behaviour free to drift, which is the failure mode a ledger recording
names rather than behaviour already demonstrated in this epic.

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