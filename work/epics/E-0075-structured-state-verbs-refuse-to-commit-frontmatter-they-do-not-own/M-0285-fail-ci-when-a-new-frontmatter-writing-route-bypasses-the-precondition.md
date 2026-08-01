---
id: M-0285
title: Fail CI when a new frontmatter-writing route bypasses the precondition
status: in_progress
parent: E-0075
depends_on:
    - M-0284
tdd: required
acs:
    - id: AC-1
      title: A new frontmatter-writing route that bypasses the seam fails CI by name
      status: met
      tdd_phase: done
    - id: AC-2
      title: The invariant's exemption list carries one reasoned entry per exempt route
      status: open
      tdd_phase: done
---

## Goal

Make the precondition's coverage a mechanical property rather than a convention,
so a frontmatter-writing route added later cannot quietly bypass the seam.

## Context

M-0283 and M-0284 route the verbs that exist today. Nothing stops the next one
from being written without the guard, and a coverage rule that depends on a
reviewer noticing is the condition this repo holds is not a guarantee at all.

The precedent is already in the tree and is a close fit.
`internal/policies/projection_findings_presence.go` asserts that every verb
either calls `projectionFindings` or appears on a reviewed allowlist with its own
specific reason, and it fails CI on a verb falling outside both — the design is
documented at the top of `internal/verb/verb.go`. This milestone builds the same
shape for the write-scope seam.

One known limitation to design around rather than inherit:
`verb_result_noop_invariant.go`, M-0281's analogous policy, scans *exported*
entry points, so an unexported branch is invisible to it. That is a real gap in
the model being copied, not a detail of its implementation.

## Approach

Follow `projection_findings_presence.go`: presence check plus a reviewed
one-entry-per-exemption allowlist, failing CI on anything outside both. Write the
firing fixture first — a route shaped like a real one that bypasses the seam —
so red is a live state rather than a formality, which is also what the
firing-fixture meta-gate expects of every policy's construction line.

The exemption list is authored from the calls M-0283 and M-0284 already recorded,
not re-derived. Where those milestones wrote down a reason for a route being out,
that reason is what the allowlist entry carries.

## Acceptance criteria

### AC-1 — A new frontmatter-writing route that bypasses the seam fails CI by name

A route that writes entity frontmatter without passing through the precondition
fails the policy suite, and the finding names the route so the failure is
actionable without reading the policy's source.

Scope coverage is part of the claim, not a separate concern: the analogous
same-state policy scans only exported entry points, so an unexported branch
escapes it. This invariant covers the routes it claims to cover, or states in its
own doc comment which shapes it cannot see.

### AC-2 — The invariant's exemption list carries one reasoned entry per exempt route

Every exempt route has its own entry with its own specific reason — not a shared
category label, and not a bare route name. The reasons come from the calls
M-0283 and M-0284 recorded, so the allowlist and the milestone bodies say the
same thing.

An entry whose reason has stopped being true is the failure mode worth guarding:
this mirrors what E-0077 is doing for dormant `dupl` exemptions, where an
exemption outliving the condition it exempts is treated as a defect rather than
as harmless.

## Constraints

- Every reachable branch of the policy is exercised, and the policy's own
  construction line is covered by a fixture that makes it fire — the
  firing-fixture meta-gate requires this of every policy, and a chokepoint that
  never fires in test is the thing it exists to prevent.
- The exemption list is authored from M-0283's and M-0284's recorded calls.
  Re-deriving reasons here would fork them from the bodies that decided them.
- The invariant is kernel-internal. It adds no consumer-facing surface and so
  needs no skill, `--help` text, or completion wiring.

## Design notes

- `projection_findings_presence.go` is the shape to copy; `verb.go`'s package
  doc is the shape to copy for how the exemption reasons get written down.
- `verb_result_noop_invariant.go`'s exported-only scan is a known limitation of
  the model, not a target to reproduce. Whether this invariant can see
  unexported routes is decided during implementation and stated either way.

## Out of scope

- Any change to the guard itself. If the invariant reveals an unrouted verb, that
  is a finding against M-0283 or M-0284, not new design here.
- After-the-fact detection of laundering already in history (G-0480). This
  invariant governs the source tree; that rule governs the commit log.

## Dependencies

- M-0284 — the sweeps and nested paths must be routed, or the invariant fails
  against them on the day it lands.
- M-0283 — the seam itself.

## References

- E-0075 — the parent epic
- `internal/policies/projection_findings_presence.go` — the precedent shape
- `internal/policies/verb_result_noop_invariant.go` — M-0281's analogue, and its
  exported-only limitation
- `internal/verb/verb.go` — where the projection-findings exemption reasons live
- E-0077 — the dormant-exemption problem this milestone's AC-2 anticipates

