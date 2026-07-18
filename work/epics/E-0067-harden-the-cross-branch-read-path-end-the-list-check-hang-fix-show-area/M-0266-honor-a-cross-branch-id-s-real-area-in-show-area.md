---
id: M-0266
title: Honor a cross-branch id's real area in show --area
status: in_progress
parent: E-0067
depends_on:
    - M-0265
tdd: required
acs:
    - id: AC-1
      title: show --area on a cross-branch id evaluates against the entity's real area
      status: open
      tdd_phase: red
---

## Goal

`aiwf show <id> --area X` on a cross-branch-resolved id should evaluate the `--area`
predicate against the entity's real `area:` field, not the local-only lookup that
always reports untagged. Fixes G-0419.

## Context

M-0265 introduces the shared cross-branch scan helper. `show`'s cross-branch path
(`buildCrossBranchShowView`) already parses the resolved entity, including its
`area:` field, but the `--area` predicate still routes through
`tr.ResolvedAreaByID`, which consults only the local tree — so for a
cross-branch-resolved id it returns untagged regardless of the entity's real area.
This milestone threads the resolved area through.

## Acceptance criteria

### AC-1 — show --area on a cross-branch id evaluates against the entity's real area

`aiwf show <cross-branch-id> --area X` reports the entity in-area when its real
`area:` on the resolving ref equals X, and out-of-area otherwise — instead of
always reporting untagged. Verified against a cross-branch fixture whose entity
carries a real area on the ref it resolves from.

## Constraints

- Local-id `--area` behavior is unchanged; only the cross-branch-resolved path is
  corrected.
- No new package-level mutable state; `show`'s cross-branch read stays best-effort.

## Design notes

- The resolved entity's `Area` is already read in `buildCrossBranchShowView` via
  `entity.Parse`; thread that value into the `--area` predicate for the
  cross-branch case rather than falling back to `tr.ResolvedAreaByID`'s local-only
  lookup (per G-0419).

## Surfaces touched

- `internal/cli/show/show.go`

## Out of scope

- Any change to `list --area` or local-id `--area` behavior.
- The M-0265 helper (dependency, already landed).

## Dependencies

- M-0265 — the shared cross-branch scan helper and resolved-entity read this
  milestone threads the area through.

## References

- Gap: G-0419. Epic: E-0067.

## Work log

## Decisions made during implementation

- (none yet)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none yet)
