---
id: M-0169
title: Directed wf-vacuity pass over the load-bearing units
status: in_progress
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Directed vacuity audit complete with every finding dispositioned
      status: met
---
## Deliverable

A directed `wf-vacuity` pass — the assertion-shape probe that mutation testing
cannot perform — over the highest-value units, with each finding dispositioned
into a strengthened assertion or an acknowledged non-issue. This is probe 2 of
the G-0262 corpus work; M-0168 is the mechanical probe-1 half (gremlins).

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

## Approach

Where the mechanical tool already covers probe 1 ("can the tests fail at all?"),
M-0168 carries it; this milestone leans into probe 2 (the assertion-shape
reading the tool cannot do) and applies probe 1 only where the reading raises a
specific doubt. Each finding pairs a weak assertion with the bug that survives
it (or the reason it is tautological / over-narrowed), then routes to a
strengthened assertion or an acknowledged non-issue. Per the ritual's
constraints, every injected bug is reverted — the tree ends byte-identical — and
the audit reports rather than rewriting blindly; each strengthening is a real,
reviewed test change.

## Mechanical evidence

Each `strengthen`-dispositioned finding lands a new or changed assertion that
goes red when the targeted bug is injected (recorded per finding), and the
strengthened suite stays green under `make ci`. The audit is LLM-judged by
nature, so the claim is sized to that: a clean unit means "no weakness found,"
not "verified correct."

## Acceptance criteria

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
