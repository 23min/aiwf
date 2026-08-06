---
id: M-0294
title: Derive the coherence rule lists from one declaration
status: draft
parent: E-0079
depends_on:
    - M-0291
tdd: required
acs:
    - id: AC-1
      title: Three hand-maintained rule lists derive from one declaration
      status: open
    - id: AC-2
      title: The declaration's per-rule claims are verified against behavior
      status: open
    - id: AC-3
      title: Declared rules and firing rules are in bijection, failing by name
      status: open
---

## Goal

Give a rule space with no FSM coordinate the same pin-and-bijection discipline
the branch spec already has, so a rule cannot again reach one call site of four
with nothing noticing.

## Context

Three shapes are in play, and conflating them is why the coherence guard drifted.
The entity FSM is cell-keyed by kind, from-state and verb, with a Pin registry and
a bijection meta-test proving each cell has a test. Sovereign-act shape is a
closed tuple set over FSM edges and fits that same key. Trailer coherence fits
neither: it is a predicate over *(actor role × trailer presence-vector)* — no
kind, no from-state, no verb. It has no cell, so no Pin, so nothing indexed the
rule and nothing noticed the guard reached one verb of four.

## Acceptance criteria

### AC-1 — Three hand-maintained rule lists derive from one declaration

Every cell in the coherence rule space is registered, and each carries a named
test that exercises it.

Evidence: the registry enumerated against the rule set, with the count of pinned
cells derived from the registry rather than written down.

### AC-2 — The declaration's per-rule claims are verified against behavior

The check fails when a cell has no pinning test, and its message names which
cell. A meta-test that reports only that something is unpinned sends the reader
back to a search the check already did.

Evidence: a fixture registering a cell with no test, asserting the failure names
that cell.

### AC-3 — Declared rules and firing rules are in bijection, failing by name

## Design notes

- **Widen the existing Pin registry; do not build a parallel one.** A second
  registry drifts from the first one plausible line at a time, and the two would
  answer the same question differently within a release or two.
- **The registry lands with what retires it.** This is a mandate — every cell
  needs a pinning test — and a mandate costs per subject forever. It lands with a
  named owner and a stated retirement trigger, or it is a permanent tax.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Is a cell one rule, or one point in the input cross-product? | yes, before implementing | Per-rule is cheap and catches a rule nothing references. Per-combination is what would actually have caught this drift, at a much larger cell count. Decide against M-0291/AC-2's generated domain, which is the same cross-product. |

## Out of scope

- Re-keying the existing FSM cell space. This adds a second shape alongside it.

## Dependencies

- M-0291 — its generated coherence domain is what this registry indexes.

## References

- This milestone is the epic's candidate for promotion to its own epic; the
  decision to build it is settled, only its placement is open.

