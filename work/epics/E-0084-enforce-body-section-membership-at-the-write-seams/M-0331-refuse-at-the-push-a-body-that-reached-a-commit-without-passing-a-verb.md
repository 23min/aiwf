---
id: M-0331
title: Refuse at the push a body that reached a commit without passing a verb
status: draft
parent: E-0084
tdd: required
---
## Goal

Close the last hole in body-section enforcement: a body can reach a commit
without passing any verb, and nothing catches it. Add a gate on the push,
riding the commit range the provenance audit already resolves.

## Closes

- (none)

## Context

`entity.RequiredSections` has been the single definition of each kind's body
sections since E-0081, and M-0329 wired a refusal into `aiwf add` and both
modes of `aiwf edit-body`. What remains unenforced is the path that bypasses
those seams entirely: a body written into a commit by plain `git commit`. That
path is not hypothetical — the wrap-milestone ritual writes the milestone spec
that way today.

ADR-0048 places this gate and defines what a violation is; ADR-0043 established
why the enforcement is forward-only by construction rather than by policy.

## Acceptance criteria

## Constraints

- The rule does not join `check.Run`. `aiwf check`'s tree-wide output is
  unchanged and no existing entity gains a finding.
- Scope is entities whose *body content* changed in the range, never entities
  merely touched. A touched-path scope would block ordinary work on debt it did
  not create.
- No workflow available today may become unavailable. An author who does not yet
  know a section's content keeps the heading and leaves it empty.

## Design notes

- ADR-0048 is the decision this implements; it supersedes ADR-0043 on what the
  verb seam asks, while the placement, the definition of a violation, and the
  push seam carry forward unchanged.
- The finding code is unsettled. ADR-0043 leaves open whether one code serves
  both membership and emptiness or each needs its own, and E-0083 is the epic
  that would answer the emptiness half. Settle it with E-0083 before this
  milestone lands, not twice afterwards.
- The gate inherits the provenance audit's range resolution, which is skipped
  when no upstream is configured and no `--since` is passed. CI-on-push is the
  backstop; do not claim otherwise in the milestone's own prose.

## Surfaces touched

- the provenance-audit range resolution the gate rides
- `entity.RequiredSections` — read as the only input

## Out of scope

- The existing violations. Both seams read only bytes being written, so a body
  already committed is never in scope. Paying that debt is a migration with its
  own evidence.
- Emptiness. Whether a section that is present carries content is ADR-0042's
  subject and E-0083's work.
- Changing what any kind's required set contains. That set is E-0081's answer
  and this milestone consumes it.
- `aiwf import`, which is excluded on its deprecation pending G-0667.

## Dependencies

- ADR-0048 — accepted; the decision this implements.
- M-0329 — delivered the verb seam this completes.

## Coverage notes

- (none)

## References

- ADR-0048, ADR-0043, ADR-0042
- E-0081 — gave the section set one owner and deliberately excluded enforcement
- E-0083 — shares the finding-code question
- G-0571 — the hole this closes, jointly with the deletion milestone
- G-0667 — `aiwf import`'s unrecorded deprecation

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
