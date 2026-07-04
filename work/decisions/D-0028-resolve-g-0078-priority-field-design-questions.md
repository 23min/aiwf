---
id: D-0028
title: Resolve G-0078 priority field design questions
status: accepted
relates_to:
    - G-0078
---
# D-0028 — Resolve G-0078 priority field design questions

> **Date:** 2026-07-04 · **Decided by:** human/peter

## Question

G-0078 proposes a kernel-wide `priority` frontmatter field but leaves four sub-questions open: which kinds carry it, whether `aiwf check` enforces anything beyond the enum, how it interacts with `aiwf status`'s default sort, and whether the field is named `priority` or `importance`. A milestone can't be scoped to implement the field until these are settled.

## Decision

1. **Scope** — `priority` applies to **gap and milestone only**, not epic, ADR, decision, or contract.
2. **Enforcement** — `priority` is **purely advisory**. `aiwf check` validates only that the value is a legal enum member (or unset); no cross-field rules (e.g. no "urgent + open + no scope-entity" finding).
3. **Sort order** — `aiwf status` keeps its existing status/kind-first grouping. Priority is used only as a **tiebreaker within each section**, not as the primary sort key across the whole view.
4. **Naming** — the field is named **`priority`**, not `importance`.

## Reasoning

**Scope.** Gap and milestone are genuine backlogs — piles of open items you triage and pick from, which is the exact friction G-0078 documents (30+ open gaps, no way to rank them). Decision entities were considered but rejected: an open decision is typically a *blocker* you resolve because something downstream needs the answer, not a queued item you defer against competing priorities — and decisions are low-volume and short-lived compared to gaps, so "which decision first" rarely arises the way "which gap first" does. Epic and ADR were rejected because they're ranked by the milestones/decisions they contain, not directly — an epic-level priority would have no clear relationship to its children's priorities.

**Enforcement.** This is a v1/PoC field. The `area` field's enforcement rules (mistag, unknown, overlap, dead-glob checks) were added in a follow-on epic after the base field had shipped and seen real use, not on day one. Inventing a cross-field rule now (e.g. the "urgent + no scope-entity" idea G-0078 itself floats) would couple `priority` to the authorization-scope mechanism before there's evidence that's the right coupling — speculative enforcement policy is the kind of YAGNI the framework's engineering principles warn against. Advisory-only can grow teeth later, additively.

**Sort order.** `aiwf status`'s grouped-by-status/kind structure is its load-bearing UX; a full priority-first re-sort would restructure the view most users already know from a single field addition. Status-first with priority as a within-section tiebreaker still lets an `urgent` gap float above other open gaps, without the risk of a priority-first sort silently burying an old `unset`-priority item below one that was just tagged `low`. If status-first proves too weak in practice, priority-first is an easy additive change later — default sort order is not a one-way door the way schema or enforcement decisions are.

**Naming.** `priority` matches the term already used by Linear, Asana, GitHub-via-labels, and Shortcut — the convention a user or an AI assistant will guess to type (`--priority high`) without learning aiwf-specific vocabulary. `importance` is arguably more precise given the advisory, non-queueing semantics just decided, but that precision is a second-order concern next to discoverability, especially given the framework's own commitment that kernel functionality must be AI-discoverable. G-0078 reaches the same conclusion for the same reason (its line 53: "the field's purpose is to drive ordering and that's what every other tool calls it").

## Consequences

- Unblocks scoping a milestone (or milestone pair, under the implementing epic) to add `priority: urgent | high | medium | low` to the `gap` and `milestone` kinds, with enum validation in `aiwf check`, a `--priority` filter/tiebreaker in `aiwf list` and `aiwf status`, JSON envelope support, and HTML renderer surfacing.
- Extending `priority` to other kinds, adding enforcement rules, or changing the default sort weighting are each additive, non-breaking follow-ups if real friction shows up — none is foreclosed by this decision.
