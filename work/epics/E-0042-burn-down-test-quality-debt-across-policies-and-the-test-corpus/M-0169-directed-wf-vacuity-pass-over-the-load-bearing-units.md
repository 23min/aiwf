---
id: M-0169
title: Directed wf-vacuity pass over the load-bearing units
status: in_progress
parent: E-0042
tdd: none
---
## Deliverable

A directed `wf-vacuity` pass — the assertion-shape probe that mutation testing
cannot perform — over the highest-value units, with each finding dispositioned
into a strengthened assertion or an acknowledged non-issue.

## Scope

The judgment probe catches what `mutate-hunt` cannot (G-0262): tautological
assertions, over-narrowed antecedents or fixtures, and substring-not-structural
checks (the CLAUDE.md "substring assertions are not structural assertions"
lesson is the standing evidence this debt is real). It cannot be mechanized, so
this is a *directed* sweep over the load-bearing units, not an undirected
whole-tree pass:

- the FSM and id allocator;
- parsers and serializers (frontmatter, trailers, slugs);
- the verb plans;
- the check rules;
- the renderers.

## Outcome

A vacuity-finding disposition list over the directed set, completing the
corpus-wide portion of G-0262.

*Draft stub — acceptance criteria pinned when the milestone starts.*
