---
id: M-0140
title: Classify legality finding codes; close AC-5 bidirectional arm
status: draft
parent: E-0036
depends_on:
    - M-0138
tdd: required
---
## Goal

Add a structural `Class` marker (e.g. `ClassLegality`) so legality-pertinent finding/error codes are programmatically enumerable, and close AC-5's deferred fourth arm: every legality code is referenced by ≥1 illegal-outcome spec Rule. This makes the bidirectional-completeness guarantee live and turns the classifier into a chokepoint later milestones must satisfy.

## Context

AC-5 closes three of four drift arms; the impl→spec arm — "every legality-pertinent finding code is referenced by a spec rule" — was deferred (G-0145) because ~25 impl codes mix legality (fire on FSM/precondition violations) with structural integrity (frontmatter shape, id collisions, ref resolution). The classifier must enumerate the former. Lean from the gap: structural metadata on the code, not a hand-maintained allowlist — the classifier is a property of the code, not of the test.

## Acceptance criteria

- **AC1** — Legality-pertinent codes carry a structural `Class` marker enumerable in code (not a hand-maintained test allowlist). *Evidence:* `TestFindingClass_LegalityEnumerable` — asserts the closed legality set is derived from the marker, and a structural-integrity code is *not* in it.
- **AC2** — The AC-5 drift test asserts every legality-classed impl code is referenced by ≥1 illegal-outcome spec Rule, and **fails when a legality code lacks a spec reference**. *Evidence:* the new drift arm in `m0123_ac5_drift_test.go`; a negative-of-the-policy fixture (a deliberately orphaned legality code) makes it red — proving the policy actually fires, not just passes vacuously.
- **AC3** — The codes emitted by M-0138/M2 (`fsm-transition-illegal`, `authorize-kind-not-allowed`, the two cancel codes) round-trip as legality and resolve to spec rules. *Evidence:* assertion they appear in the legality set and each maps to ≥1 Rule.

## Constraints

- Structural metadata (G-0145 option 2), not a hand-maintained allowlist.
- `tdd: required`. AC2's negative-of-the-policy test is mandatory — a policy that can't fail is not a chokepoint.

## Out of scope

Emitting the codes (M-0138/M2); the rename (M4); reachability (M5).

## Dependencies

M-0138. Best executed after M2 so it certifies the cancel codes too (soft ordering). Closes G-0145.
