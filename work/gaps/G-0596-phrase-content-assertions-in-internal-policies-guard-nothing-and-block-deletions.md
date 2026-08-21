---
id: G-0596
title: Phrase-content assertions in internal/policies guard nothing and block deletions
status: open
---
## What's missing

D-0050 decided that a structural test over prose asserts document shape, never
that a paragraph still says a particular thing. It governs tests written from
its date forward and explicitly declines to mandate retrofitting the existing
suite, on the grounds that retrofitting needs its own justification.

That justification did not exist when D-0050 was accepted. It does now.

`internal/policies/` carries 174 long-literal `strings.Contains` assertions
across 48 test files. They cost twice: an edit to the prose they guard is a
two-file change requiring Go, so corrections get deferred; and they pass
whether or not the claim the wording encodes is still true, so they guard
nothing. D-0050's own Reasoning records the measurement — of roughly thirty
findings across four review rounds over the G-0489 cohort, more than half were
defects in the assertions rather than in the prose they guarded.

One of them is load-bearing in the wrong direction:
`TestM0122_AC5_OpenQuestionsSectionPresent` requires the literal heading
`## Open questions for Pass C` in `docs/design/legal-workflows-first-principles.md`
— a Pass C that never happened, in one of the two documents that account for
48% of the exposition tier. The document cannot be retired while the test stands.

## Why it matters

The suite is otherwise sound. Mutation testing across six packages and roughly
28,000 production lines returned 88.6–96.6% efficacy, with the loader at 96.6%
and the git layer at 92.5%. These assertions are the exception — the part of
the apparatus that reports nothing true, while making the prose it guards
expensive to correct and blocking the deletions that would shrink the corpus.

## Resolution shape

Per assertion: delete where nothing breaks if the wording changes; convert to a
shape assertion where a rule forces the shape; keep where the phrase is
functionally load-bearing, with a comment saying why. Skill trigger phrases are
the known instance of the third case. Every conversion must be shown to fail
when the property it claims breaks, or it has replaced one vacuous assertion
with another.
