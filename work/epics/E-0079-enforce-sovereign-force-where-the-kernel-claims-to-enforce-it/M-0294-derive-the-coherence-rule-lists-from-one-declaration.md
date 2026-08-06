---
id: M-0294
title: Derive the coherence rule lists from one declaration
status: in_progress
parent: E-0079
depends_on:
    - M-0291
tdd: required
acs:
    - id: AC-1
      title: Three hand-maintained rule lists derive from one declaration
      status: met
      tdd_phase: done
    - id: AC-2
      title: The declaration's per-rule claims are verified against behavior
      status: open
    - id: AC-3
      title: Declared rules and firing rules are in bijection, failing by name
      status: open
---

## Goal

Collapse the three hand-maintained lists that describe the coherence rule set into
one declaration beside the rules, so a rule added without it fails by name rather
than reading as covered while nothing exercises it.

## Context

M-0291 pinned the rule set across a generated domain: every actor role crossed
with every subset of the presence-bearing trailers, a golden verdict at each
point, design-doc invariants each carrying a non-vacuity guard, a reachability
assertion over the declared rules, and a bidirectional property for the subset
`verb.Apply` enforces.

Three lists still describe that rule set by hand, and each sits in a different
file from the rules it describes: the rule roster, the seam's force-predicated
subset, and the domain's own trailer axis. The first two already carry comments
naming this milestone as their retirement.

The axis is the one nobody named, and it is the one that matters most. A rule
predicated on a trailer outside the five the domain varies fires at no point in
that domain — so the golden never moves, the reachability assertion never sees
it, and the rule reads as covered while nothing exercises it. Retiring the two
named lists leaves that open; deriving the axis from the same source closes it,
because a rule declaring a new input widens the domain by construction.

D-0062 records why this is a declaration rather than the Pin-and-bijection
registry the epic originally scoped.

## Acceptance criteria

### AC-1 — Three hand-maintained rule lists derive from one declaration

One table beside the rules declares each rule: the presence-bearing trailers its
condition consults, and whether it fires only when a force trailer is present.
The rule roster, the seam's force-predicated subset, and the domain's trailer
axis all derive from it.

The hand-maintained copies are deleted rather than left alongside. A derivation
that coexists with the list it replaces is a second source of truth, and the two
answer the same question differently the first time one is edited.

Evidence: each derived value asserted equal to the list it replaces before that
list is removed, so the change is measured behavior-preserving rather than
assumed; the removal is what makes the derivation load-bearing.

### AC-2 — The declaration's per-rule claims are verified against behavior

The declaration is data a human writes, so a test that reads its claims and
believes them proves nothing about the rules. Each entry is asserted against what
its rule actually does across the domain: one declared to fire only under a force
trailer must fire at no point lacking one, and one must not fire at a point where
every input it declares is absent.

Evidence: both directions asserted across the generated domain, each with a
fixture whose deliberately mis-declared entry makes the assertion fail.

### AC-3 — Declared rules and firing rules are in bijection, failing by name

Every declared rule fires somewhere in the domain, and every rule that fires is
declared. The failure names the rule; one reporting only that a mismatch exists
sends the reader back to the search the check just performed.

Evidence: both directions, each with a fixture — a declared rule that fires
nowhere, and a rule firing under a name absent from the declaration — asserting
the message names it.

## Design notes

- **The declaration lives beside the rules, not in the test.** The lists it
  replaces drifted because each sat in a different file from the thing it
  described; moving the copies into one place elsewhere would repeat that.
- **Declaring inputs, not just names, is what closes the axis.** A roster of rule
  names retires two lists and leaves the third — the one whose absence lets a rule
  go unexercised.
- **Adding a rule still means adding its entry.** That obligation is real and
  worth naming: it is one line beside the rule rather than three edits in another
  file, and AC-3 catches the omission for every rule that fires within the current
  axis. What it does not catch is a rule both undeclared and predicated on a new
  trailer — stated here rather than closed, per D-0062.

## Out of scope

- Table-driven evaluation — dispatching the rules from the declaration so an
  undeclared rule cannot run. Rejected in D-0062; it refactors a working guard for
  the residual named above.
- The branch spec's Pin registry and any change to it. Different rule space,
  different key.
- Which rules `verb.Apply` enforces. Membership stays satisfiability per D-0060;
  this milestone changes only where that subset is written down.

## Dependencies

- M-0291 — its generated domain, and the two lists that name this milestone as
  their retirement.

## References

- D-0062 — why a declaration rather than a cell registry.
- D-0060 — the satisfiability criterion for the seam's subset, unchanged here.
