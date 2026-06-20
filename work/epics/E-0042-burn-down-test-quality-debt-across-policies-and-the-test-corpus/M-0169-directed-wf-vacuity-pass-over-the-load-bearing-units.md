---
id: M-0169
title: Directed wf-vacuity pass over the load-bearing units
status: in_progress
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Directed vacuity audit complete with every finding dispositioned
      status: open
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

### AC-1 — Directed vacuity audit complete with every finding dispositioned

**Deliverable** — A directed `wf-vacuity` pass over the load-bearing units —
the FSM and id allocator, the parsers/serializers (frontmatter, trailers,
slugs), the verb plans, the check rules, and the renderers. Produce a committed
report in the `wf-vacuity` output format (surviving mutants / weak assertions /
clean / summary) that **names every unit audited** so directed-set coverage is
visible, and dispositions every finding as `strengthen` (a test change lands) or
`non-issue` (justified). This is probe 2 of the G-0262 corpus work — the
assertion-shape judgment (tautologies, over-narrowed antecedents,
substring-not-structural checks) that mutation testing cannot perform.

**Mechanical evidence** — Each `strengthen`-dispositioned finding lands a new or
changed assertion that goes **red** when the targeted bug is injected into the
implementation (the probe-1 confirmation, recorded per finding in the report);
the bug is reverted so the tree is byte-identical after. `make ci` stays green
with the strengthened assertions in place.

