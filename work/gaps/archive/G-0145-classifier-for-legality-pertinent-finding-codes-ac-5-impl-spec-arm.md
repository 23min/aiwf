---
id: G-0145
title: Classifier for legality-pertinent finding codes (AC-5 impl->spec arm)
status: addressed
discovered_in: M-0123
addressed_by:
    - M-0140
---
## What's missing

M-0123/AC-5's bidirectional drift policy between `spec.Rules()` and the
kernel impl closes three of four arms:

- impl → spec: every FSM cell covered ✓
- impl → spec: every Cobra verb covered (or allowlisted) ✓
- spec → impl: every Rule's Kind/FromState/Verb/ExpectedErrorCode
  resolves ✓
- impl → spec: every **legality-pertinent** finding code is referenced
  by ≥1 illegal-outcome Rule — **deferred**

The deferred arm is genuinely hard: the impl emits ~25 finding codes via
`internal/check/`, and only a subset are "legality-pertinent" (fire on
attempted verb actions that violate the FSM or its preconditions). The
rest are "structural integrity" (frontmatter shape, id collisions, ref
resolution, etc.). The drift policy needs a way to enumerate the former.

## Why it matters

Without the fourth arm, an impl-side legality finding code could land
without ever being referenced by the spec — silently widening the gap
between what the impl polices and what the spec claims to police.

## Proposed fix shape

Two candidates, in increasing thoroughness:

1. **Closed allowlist in the drift test.** Maintain
   `legalityPertinentFindingCodes` (a set) in
   `internal/policies/m0123_ac5_drift_test.go`. The drift test asserts
   every code in the set appears as an ExpectedErrorCode in ≥1 Rule.
   Cheap; requires hand-maintenance on every new code.

2. **Code-declaration metadata.** Add a tag at the impl site (e.g.,
   `Code: "X", Class: ClassLegality` on the Finding struct, or a parallel
   `var legalityCodes = []string{...}` in `internal/check/`). The drift
   test reads the closed set programmatically. Higher cost; lower
   maintenance debt.

Lean: option 2 — the classifier is a structural property of the code, not
of the drift policy. The Finding struct already carries Severity; an
analogous Class field is a natural addition.

## Discovered in

M-0123/AC-5. The full bidirectional drift policy intentionally defers
this arm to keep AC-5 scoped; the deferral is captured in the AC-5 test
file's package-level comment block.
