---
id: M-0272
title: Extract the read-side helpers into a neutral entityview package
status: draft
parent: E-0069
depends_on:
    - M-0269
    - M-0270
    - M-0271
tdd: none
---
## Goal

Give the read-only verbs a neutral shared library: extract the verified
Cobra-free read-side helpers out of `show`/`history` into a neutral package
that `render`, `check`, and `status` consume instead of importing sibling
`internal/cli` verb packages.

## Context

Finding F6 (deep-dived and quantified): ~638 lines / ~16 exported symbols of
the combined `show`/`history` surface are already pure — `scopes.go` wholesale,
`EventFromCommit`, `ReadEntityBody`, and the `HistoryEvent`/`ReadHistory`/
trailer-parsing half of `history.go`. The rest is genuinely Cobra-specific and
stays. Three consumers (`render`, `check`, `status`) currently reach into
sibling CLI packages for this logic; the acyclic property survives only because
nobody has yet added the closing edge.

## Acceptance criteria

## Constraints

- Mechanical only: import-path changes on the verified surface, no algorithm
  changes, no API redesign.
- Runs last in the epic, after the sibling milestones are done and green.

## Design notes

- Package name decided here (epic spec lean: `internal/entityview`).

## Out of scope

- Extracting anything Cobra-bound; the ~70% that stays put stays put.
- New read-verb features.

## Dependencies

- The three sibling E-0069 milestones (bug fixes, housekeeping, FinishVerb) —
  declared via `depends_on`.

## References

- `docs/initiatives/verb-layer-cleanup.md` §F6 (scope table, line inventory).

---

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
