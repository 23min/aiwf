---
id: G-0656
title: No rule records whether it keys to the kernel's scaffold set or the template's
status: open
---

## What's missing

Two surfaces describe a spec's sections, at different layers, and nothing
records which one a new rule should read.

`internal/entity/required_sections.go` holds the kernel set. Its own doc comment
is explicit about what that set is: *"'Required' names what the scaffold writes,
not a guarantee anything verifies."* It is the load-bearing minimum `aiwf add`
emits, and `entity-body-empty` reports one of those headings present and empty.

The shipped entity templates under the rituals are a superset, and they are the
vocabulary authors and rituals actually use. That layering is deliberate. What is
missing is any record of which layer a rule may key to, and what it inherits by
choosing one.

Measured 2026-08-31, kernel set against shipped template headings:

| kind | kernel | template | delta |
|---|---|---|---|
| contract | 2 | 2 | 0 |
| gap | 2 | 2 | 0 |
| decision | 3 | 4 | +1 |
| adr | 3 | 5 | +2 |
| epic | 3 | 11 | +8 |
| milestone | 2 | 17 | +15 |

Four of six diverge. The two that agree do so by coincidence of their current
contents, not by any mechanism.

## Why it matters

A rule keyed to the template's vocabulary demands a section the kernel never
writes, and that is not hypothetical. M-0326 added a rule reporting a milestone
reaching `done` with no release-note section. Landed briefly at error severity it
became a `promote` precondition, and a milestone created through `aiwf add` could
then not reach `done` at all: the kernel writes `## Goal` and
`## Acceptance criteria`, the section is not among them, and `--force` does not
relax a projection finding. It was reverted to warning, which sidesteps the
collision rather than resolving it.

The reverse trap is the same size. A rule keyed to the kernel set cannot see the
sections consumers actually fill — for a milestone that is most of the spec.

Nothing in either surface says which layer is appropriate for which kind of rule,
so the choice is made per rule by whichever file its author happened to open.

## Direction

Write down the rule for choosing, and put it where a rule's author will meet it.

The kernel set is the right target for anything the kernel guarantees, because it
is what `aiwf add` writes; the template set is the right target for anything held
at review, because it is what authors fill. What is not yet decided is what a rule
does when it wants a section only the template ships — accept that it can only
warn, or extend the scaffold so the kernel writes it too.

Extending the scaffold is not free: it changes what every new entity of that kind
carries, and for milestone it would grow a two-entry set toward seventeen. G-0571
measures the enforcement half of that blast radius at 119 findings over 60 live
entities.

## References

- G-0571 — nothing enforces that an entity body carries its kind's required
  sections. The enforcement half, and it carries an inherited obligation from
  M-0326.
- G-0636 — milestone-spec section rules are restated across five surfaces with no
  owner. The prose half.
- M-0326 — added the rule that surfaced this, and reverted its severity rather
  than resolving it.
- D-0082 — records the changelog-input reversal that milestone made.
