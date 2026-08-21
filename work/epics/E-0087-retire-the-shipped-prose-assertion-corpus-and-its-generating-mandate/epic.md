---
id: E-0087
title: Retire the shipped-prose assertion corpus and its generating mandate
status: active
---
## Goal

Remove the body of tests that assert shipped prose still says particular things, and
re-point the chokepoint that generates them at the property it was actually meant to
enforce. The outcome is a smaller policy suite whose green run means something true, and
a skill-edit gate that costs once rather than once per edit.

## Context

G-0220 recorded a shippable ritual edit that reached consumers with no gap, no
acceptance criterion, no owning milestone, and no test. The human caught it; the kernel
did not. Three of that gap's four complaints are about provenance; one is about a
missing test. The chokepoint that closed it under E-0048 implemented only the fourth,
and in its weakest form — the edited path must appear as a string literal in some policy
test.

What followed is visible in the growth curve. Policy-test references to embedded-ritual
paths stood in the single digits before that backstop landed and quadrupled over the
weeks after it. D-0050 then measured the resulting assertions directly and found that
more than half of the review findings over one cohort were defects in the assertions
rather than in the prose they guarded. It fixed the rule going forward but explicitly
declined to retrofit, leaving the existing corpus in place and the generator untouched.

G-0596 supplies the retrofit justification. Scoping it surfaced the wider finding: the
corpus has no recorded catch across roughly fourteen months, while the misses it failed
to prevent are themselves filed as gaps. Two narrow classes are the exception and are
carried forward rather than deleted.

Two decisions govern this epic and settle what it may delete. D-0070 rules that
prose-presence assertions over shipped surfaces are deleted rather than converted, and
names the two exception classes. D-0071 re-points the skill-edit backstop from a
content-reference requirement to a provenance one. Each is independently actionable;
taken together they remove the corpus and the obligation that regrows it.

## Scope

### In scope

- Re-pointing the skill-edit backstop from a content-reference requirement to a
  provenance requirement, with its own firing fixtures.
- Deleting prose-presence assertions over shipped surfaces, heading-presence assertions
  included, across the policy suite.
- Preserving the two exception classes named in D-0070: cross-document
  relationship checks, and the trigger phrases that drive skill dispatch.
- Reconciling the surfaces that describe the discipline — CLAUDE.md's ritual-authoring
  and enforcement sections, and any shipped guidance that restates the mandate.

### Out of scope

- Retiring the exposition-tier design documents that some of these tests happen to lock.
  Removing the lock is in scope; deciding what to do with the documents is separate work
  with its own decision.
- `aiwf doctor`'s byte-check coverage gap over ritual and guidance artifacts, tracked in
  G-0504.
- Any change to the `skill-body-id` check or the shipped-surface id rule. This epic
  narrows what tests assert about shipped prose; it does not touch what shipped prose is
  allowed to contain.
- Broadening the provenance requirement to shippable surfaces beyond the set the
  existing backstop already watches.

## Constraints

- The two exception classes are carried forward intact. A deletion pass that removes the
  citation walk or the dispatch trigger phrases has overshot.
- The backstop must be re-pointed before the corpus is deleted. While the content
  mandate stands, each watched skill's path must survive as a literal somewhere in the
  policy sources, which constrains deletion to whole functions rather than whole files.
  Reversing the order forces careful work to satisfy a rule that is about to be removed.
- No test function is deleted merely to make a file pass. Deletion follows from D-0070's
  disposition rules, applied per assertion.
- Coverage must not regress. The diff-scoped gate names any regression by file and line.

## Success criteria

- [ ] An edit to a watched shipped surface that rides an untrailered, unowned commit is
      refused by the backstop; one that rides a trailered, entity-owned commit passes
      without any accompanying test.
- [ ] No policy test asserts that a paragraph of shipped prose contains a particular
      phrase, outside the exception classes named in D-0070.
- [ ] Each surviving exception is demonstrated to fail when the property it claims is
      broken.
- [ ] Every surface that describes the skill-edit discipline states the provenance rule
      rather than the content mandate.
- [ ] `make ci` is green, and the diff-scoped coverage gate reports no regression.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the provenance predicate key on the verb trailers alone, or additionally require the owning entity to be non-terminal? | no | Settled during the backstop milestone, recorded on that milestone's spec |
| Do the entity templates and role-agent cards need the same provenance gate as skill bodies, or is the current watched set right? | no | Deferred; the epic explicitly scopes to the existing watched set |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Deleting a test that turns out to be the sole consumer of a package-level path constant, tripping the unused-symbol linter | med | Re-point the backstop first, which removes the reason those constants exist |
| A deletion pass that removes a genuine structural assertion sharing a file with prose assertions | med | Per-assertion disposition against D-0070, not per-file; probe survivors red before closing |
| Shipped prose drifts unnoticed once the gate is gone | accepted | Named in both D-0070 and D-0071; held at review, where the wrap and patch rituals already dispatch an independent reviewer |

## Milestones

- **Re-point the skill-edit backstop to provenance** — replace the content-reference
  predicate, land its firing fixtures, and update the surfaces that describe the rule.
- **Retire the prose-assertion corpus** — delete prose- and heading-presence assertions
  over shipped surfaces, preserve and prove the two exception classes, and close G-0596.
