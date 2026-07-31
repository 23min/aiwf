---
id: D-0050
title: Pin ritual prose by structure, not by phrase content
status: proposed
relates_to:
    - G-0489
---
> **Date:** 2026-07-31 · **Decided by:** human/peter

## Question

The `skill-edit-structural-test-backstop` policy requires every embedded-rituals
`SKILL.md` edit to land alongside a referencing structural test under
`internal/policies/`. It does not say what that test should assert, and its own
v1 granularity — the edited path appears as a string literal in some policy test
— is satisfied by a test that checks nothing in particular.

So: when a ritual's prose states a rule, what does a test legitimately pin? The
tempting answer is the rule's own wording, and it is wrong in a way that is not
visible from reading the test.

## Decision

A structural test over ritual prose asserts document **shape**: a heading
exists, a section holds N sub-headings, a labelled paragraph opens a line, a
cross-reference resolves to a target that exists. It does not assert that a
paragraph still says a particular thing.

Content correctness — does this rule still mean what it meant, does it
contradict its neighbour, is the escape clause still workable — is held at
review. That is the disposition `wf-codebase-health` D5 prescribes for a defect
that cannot be pinned: name it, and record that it is held by judgment rather
than by a gate.

This governs tests written from here. It does **not** mandate removing existing
phrase-content assertions elsewhere in the suite; retrofitting is separate work
needing its own justification, and a decision that silently invalidates passing
tests is one nobody can act on.

## Reasoning

The alternative — assert the rule's wording — was tried at scale and measured.
G-0489 shipped 52 phrase-literals across 22 tests. Four independent review
rounds over that work produced roughly thirty findings, and more than half were
defects in the assertions rather than in the prose they guarded. The checks
generated more work than they caught while a green suite implied an assurance
it could not deliver.

Four distinct failure mechanisms showed up, each measured, none obvious from
reading the assertion:

- **Pre-existing literal.** The asserted phrase already appears elsewhere in the
  scoped section, so the test passes on text the change never added. Removing
  the new prescription entirely left the suite green.
- **Multiplicity introduced later.** A subsequent edit gave an asserted phrase a
  second occurrence in scope, so even the deletion probe stopped biting. The fix
  for one defect manufactured this one.
- **Polarity outside the span.** The assertion held the object of a negation, so
  `never one narrowed to what the last fix touched` became `or one narrowed …` —
  a prohibition inverted into a permission — with the test green.
- **Widening by appending.** A disposition set was re-opened by adding a fourth
  member rather than by changing the asserted phrase, which no literal catches.

The deeper reason is that a phrase assertion pins a *reading*, and readings
drift in more ways than an assertion enumerates. A mechanical check over prose
therefore looks objective while behaving like a judgment finding — the class D5
itself says does not converge under looping. The review loop over G-0489 bore
that out: successive rounds found 3, 8, 9, then 10 blocking findings, with no
downward trend.

What *is* checkable is a **relationship between documents** rather than a
reading of one. The exemplar is the cross-skill citation check in
`internal/policies/d5_structure_test.go`: it walks every ritual and fails a
`§"Section"` reference naming a heading that does not exist. It caught two
dangling citations on its first run, it covers the whole ritual tree rather than
one change, and no rewording can make it pass falsely. Prefer that shape.

Alternatives considered and rejected:

- **Keep looping until a review round comes back empty.** Follows D5 literally,
  but four rounds gave no evidence of convergence and one round demonstrably
  introduced a defect while fixing others. Paying indefinitely for a loop whose
  termination is not in evidence is worse than naming the limit.
- **Keep the assertions but write them more carefully.** Each round produced a
  sharper invariant — check the literal is new; check it carries the polarity —
  and each was necessary but not sufficient; the next round found a mechanism
  the invariant did not cover. The defect rate scales with the number of
  assertions, so the surface is the mechanism.

## Consequences

- A careless edit can weaken a rule in ritual prose with nothing going red. This
  is accepted. The assertions that would notionally have caught it mostly did
  not, while costing a finding-generating maintenance burden on every prose edit.
- The suite is smaller and a green run means less than it appeared to before —
  which is the point, since it now means something true.
- Reviewers of ritual-content changes carry more weight, and the wrap and patch
  rituals already dispatch an independent reviewer for exactly that.
- Adding a phrase-content assertion re-opens this trade. The bar, recorded in
  `internal/policies/d5_structure_test.go`'s header comment, is that breaking
  the assertion must be a structural break rather than a rewording.

## Provenance

Decided while closing G-0489, after four review rounds over the D5
"findings become checks" force and its propagation. The instance that prompted
it: review was stopped at round 4 with the assertion surface reduced rather than
driven to an empty pass — D5's own escape, taken deliberately and recorded here
rather than left in a commit body.
