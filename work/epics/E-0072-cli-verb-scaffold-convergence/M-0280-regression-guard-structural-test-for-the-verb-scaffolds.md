---
id: M-0280
title: Regression-guard structural test for the verb scaffolds
status: in_progress
parent: E-0072
depends_on:
    - M-0279
tdd: advisory
acs:
    - id: AC-1
      title: Structural test asserts no verb hand-rolls the diagnostic block
      status: met
    - id: AC-2
      title: Structural test asserts no verb hand-rolls the root/actor prelude
      status: met
    - id: AC-3
      title: Guard test fails red if either scaffold is re-inlined
      status: met
---
# M-0280 — Regression-guard structural test for the verb scaffolds

## Goal

Add an `internal/policies` structural test asserting that no verb hand-rolls either the diagnostic block or the `ResolveRoot → ResolveActor` prelude, so the convergence the two prior milestones achieved cannot silently reappear.

## Context

Once both scaffolds are single-sourced, nothing mechanical stops a future verb from re-inlining them — the property would exist by reviewer vigilance alone. This milestone lands the chokepoint, mirroring the existing `skill-edit-structural-test-backstop` pattern. Sequenced last, since it pins the end state the two migration milestones produce.

## Acceptance criteria

<!-- ACs created via `aiwf add ac`; contracts filled below. -->

### AC-1 — Structural test asserts no verb hand-rolls the diagnostic block

A test under `internal/policies` scans the verb sources and fails if any verb reconstructs the diagnostic block inline (the `ResolveLogger` / `EmitVerbOutcome` wiring appearing outside `cliutil.BeginVerbDiag`), allowing a named, rationale-carrying allowlist for any documented intentional non-member.

### AC-2 — Structural test asserts no verb hand-rolls the root/actor prelude

The same (or a sibling) structural test fails if any verb reconstructs the `ResolveRoot → ResolveActor` prelude inline instead of calling the shared helper, with the same allowlist affordance.

### AC-3 — Guard test fails red if either scaffold is re-inlined

The guard is demonstrably non-vacuous: re-inlining either scaffold into a verb turns the test red, and the test is green on the post-migration tree. (Established the wf-vacuity way — break it, watch it fail — not merely asserted.)

## Constraints

- The test asserts against source structure (the inline `ResolveLogger` / `EmitVerbOutcome` and `ResolveRoot → ResolveActor` patterns appearing outside the shared helpers), not against runtime behavior.
- An intentional non-member (a verb documented as legitimately not routing through a helper) is representable — the guard names an allowlist with a rationale rather than a blanket ban, matching the `internal/cellcoverage` precedent in the layering policy test.

## Out of scope

- Any further extraction — this milestone adds only the guard.
- The G-0227 relocation (the guard should be written so it survives the later `cliutil` split, but accommodating that split is not this milestone's work).

## Dependencies

- Both migration milestones — the guard pins the post-migration state, so it lands after the verbs are migrated (otherwise it fails on the very duplication it is meant to prevent).

## References

- E-0072 — parent epic.
- `skill-edit-structural-test-backstop` — the existing structural-guard pattern this mirrors.
- G-0447 — the convergent-duplication cleanup this seam completes.
