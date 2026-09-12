---
id: M-0331
title: Refuse at the push a body that reached a commit without passing a verb
status: draft
parent: E-0084
tdd: required
acs:
    - id: AC-1
      title: A body committed without passing a verb is refused at the push
      status: open
    - id: AC-2
      title: Promote, retitle and archive still succeed against an entity missing a section
      status: open
    - id: AC-3
      title: The new rule leaves aiwf check's tree-wide output unchanged
      status: open
    - id: AC-4
      title: A required section absent as a top-level heading is the only violation
      status: open
    - id: AC-5
      title: A template edit cannot change what the gate enforces
      status: open
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

### AC-1 — A body committed without passing a verb is refused at the push

A body whose required section is missing, written into a commit by plain `git
commit` rather than through a body-supplying verb, is refused at the push.

The proof runs against the path that does this today — the wrap-milestone
ritual's plain `git commit` of the milestone spec — rather than a synthetic
commit constructed for the test. A synthetic one would pin the rule against an
input nobody produces; the real path is what the gate exists for.

### AC-2 — Promote, retitle and archive still succeed against an entity missing a section

A status promote, a retitle, and an archive sweep each succeed against an
entity whose body omits a required section.

The gate's scope is entities whose body *content* changed in the range. An
entity merely touched by a frontmatter write or a file move is outside it.
Without this the gate would block ordinary work on debt that work did not
create, and the three verbs above are where that would bite first.

### AC-3 — The new rule leaves aiwf check's tree-wide output unchanged

`aiwf check` reports the same findings on this tree before and after the gate
lands.

The rule does not join `check.Run`; it runs only on the push, over a commit
range. No existing entity gains a finding, which is what keeps the tree's
accumulated omissions out of scope by construction rather than by a
grandfather list.

### AC-4 — A required section absent as a top-level heading is the only violation

A violation is exactly one thing: a required section absent as a top-level
`##` heading.

Sections beyond the declared set are legal and never reported. Order is not
enforced. Stating the rule this narrowly is what lets an author add their own
headings and arrange them freely while the declared set stays mandatory.

### AC-5 — A template edit cannot change what the gate enforces

Editing a prose template does not change what the gate enforces.

The scan's only input is the section set the kernel declares — located today at
`entity.RequiredSections`. The templates describe that set for a human reader
and are not consulted, so the two cannot drift into disagreeing about what is
required.

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
