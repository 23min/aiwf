---
id: D-0091
title: No AC is evidenced by a sentence pinned in CLAUDE.md
status: proposed
relates_to:
    - E-0092
    - D-0070
---
> **Date:** 2026-09-14 · **Decided by:** Peter Bruinsma (human), while planning E-0092

## Question

D-0070 retires prose- and heading-presence assertions over the shipped surfaces and leaves `CLAUDE.md` out, so a doc-shaped AC may still be evidenced by a sentence pinned in this repository's `CLAUDE.md`. Does D-0070's scope extend to that file? Non-obvious because `CLAUDE.md` is not shipped, which was the ground D-0070 drew its line on, and because twenty-one tests already pin passages there (G-0676 carries the measurement and its command).

## Decision

No AC is evidenced by a sentence pinned in a `CLAUDE.md`, root or nested. Enforcement is diff-scoped: a new test that asserts a phrase in a `CLAUDE.md` file fails the profile-driven gate. The existing pins stand until the shrink under E-0092 meets each one and re-aims it at a relationship or retires it with the reason recorded in the milestone.

## Reasoning

- D-0070's ground is that a phrase pin holds one reading, which a rewording breaks and nothing catches. That holds for `CLAUDE.md` exactly as for a skill body. The line D-0070 drew was audience, not fragility.
- A pin in `CLAUDE.md` holds more than a reading. It holds the file's size: every doc-shaped AC evidenced there adds a sentence and locks it, which is the growth engine G-0676 names.
- Banning outright now would fail the twenty-one existing pins on day one, and retiring them is the shrink's own work; the ban would front-load that work into a policy change. Leaving the scope as it is would close the commit seam and leave the engine running. Diff-scoped enforcement closes the engine for new work at no retirement cost.
- What survives for a `CLAUDE.md`-shaped claim is what D-0070 leaves for a shipped surface: the relationship check. A pointer resolved against the policy it names, a path resolved against the tree, an expectation derived by running the code.

## Consequences

- The shipped-prose-assertion policy, or a sibling in its shape, scans test source for assertions over `CLAUDE.md` files, diff-scoped to new tests, with the existing pins as a grandfather ledger the E-0092 milestones drain.
- The AC-evidence section and the substring-assertion bullet in `CLAUDE.md` state the extended scope, each as its own trailered commit under the fence.
- A milestone that wants to establish "`CLAUDE.md` documents X" states it as a relationship or records it as an observation.
