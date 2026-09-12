---
id: M-0332
title: Retire the prose that restates the section set
status: draft
parent: E-0084
tdd: required
acs:
    - id: AC-1
      title: The required section set is enumerated in exactly one place
      status: open
    - id: AC-2
      title: Each surface that carried a retired table routes to the owner
      status: open
---
## Goal

Retire the prose that restates each kind's required section set, now that a
refusal carries it, so the set is stated in one place instead of three.

## Closes

- (none)

## Context

`entity.RequiredSections` has been the single definition of each kind's body
sections since E-0081, but two surfaces still enumerate that set in prose: the
`aiwf-add` skill's per-kind body-section table, and the body-sections table in
the tree-discipline design doc. They were safe to keep while nothing enforced
the set, because a reader had no other way to learn it.

M-0329 changed that. The verb seam now refuses a body that omits a required
section, so the set is discoverable by running the verb, and the prose copies
are second sources that can drift. Retiring them is what E-0084 means by the
deletion being the point rather than a tidy-up: the enforcement without the
deletion leaves the duplication in place.

## Acceptance criteria

### AC-1 — The required section set is enumerated in exactly one place

The required section set for each kind is enumerated in exactly one place.

The check derives the declared set from the kernel and asserts that no other
surface — shipped skill, normative design doc, or template — enumerates it.
Stated this way the criterion is a relationship between two artefacts, so
either one moving turns it red.

It is deliberately not stated as "these two passages are deleted". D-0070
retires prose-presence assertions over the shipped skill tree, and an absence
assertion over the same tree pins a reading that a reword breaks just as
readily. The relationship holds regardless of how the surviving prose is
phrased.

### AC-2 — Each surface that carried a retired table routes to the owner

Each surface that carried a retired table routes the reader to the owner of
the set, and the route resolves.

A reader who previously learned the section set from the table must still be
able to reach it. The citation is checked rather than assumed: a route naming
a heading or symbol that no longer exists is reported.

## Constraints

- Deletion, not correction. A passage rewritten to agree with the kernel is
  still a second copy; the next drift is a rewording away.
- A reader who reached the set through the deleted table must still reach it.
  Removing the table without leaving a route makes the surface worse.

## Design notes

- The evidence here needs care. D-0070 retires prose- and heading-presence
  assertions over the shipped skill tree, and `aiwf-add` is in that tree, so
  "assert the table is gone" is the banned shape wearing a minus sign. What
  survives D-0070 is the relationship check: derive the declared set from the
  kernel and assert no other surface enumerates it. That is AC-1's shape, and
  it is why the criterion is stated as *one place* rather than as *these two
  passages are deleted*.

## Surfaces touched

- the `aiwf-add` skill's per-kind body-section table
- the body-sections table in the tree-discipline design doc

## Out of scope

- Changing what any kind's required set contains. That set is E-0081's answer.
- The push seam. Its own milestone owns it, and neither waits on the other:
  what makes these passages safe to delete is the verb seam M-0329 landed.
- G-0530's question of whether the milestone template's structured-data
  sections should exist at all. That asks whether a section is worth carrying;
  this asks only where the set is written down.

## Dependencies

- M-0329 — delivered the refusal that makes the prose redundant.

## Coverage notes

- (none)

## References

- D-0070 — why the evidence is a relationship check rather than a phrase assertion
- E-0081 — gave the section set one owner
- M-0329 — delivered the verb seam
- G-0571 — the hole this closes, jointly with the push-seam milestone; its own
  body still carries superseded counts and a claim M-0329 falsified
- G-0530 — the adjacent, out-of-scope question

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
