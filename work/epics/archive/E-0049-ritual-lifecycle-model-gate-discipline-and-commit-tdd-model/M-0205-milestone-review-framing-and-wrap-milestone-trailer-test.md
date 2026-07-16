---
id: M-0205
title: Milestone review framing and wrap-milestone trailer test
status: cancelled
parent: E-0049
depends_on:
    - M-0204
tdd: advisory
---

## Goal

Fix the two milestone review-lifecycle seams named by
[G-0271](../../gaps/G-0271-milestone-skills-prescribe-author-self-review-not-independent-review.md)
and [G-0219](../../gaps/archive/G-0219-aiwfx-wrap-milestone-skill-md-asymmetric-missing-wrap-milestone-trailer-step.md):
the wrap-milestone trailer step and the start-milestone review framing.

## Context

Both gaps behind this milestone are partially resolved already:

- **G-0219** (wrap-milestone trailer step) is `addressed` — a structural
  drift-check test landed at commit `2c6ea3de` ("test(policies): structural
  drift-check for aiwfx-wrap-milestone SKILL.md merge-step trailers").
- **G-0271 defect #2** (commit-timing contradiction between start-milestone's
  "do not commit yet" and wrap's `git diff <base>..HEAD`) is subsumed by
  G-0293's Model 1 decision — per-AC commits make the wrap diff meaningful —
  and resolved alongside it.
- **G-0271 defect #1** remains open: `aiwfx-start-milestone` step 7/8 still
  frames the pre-wrap pass as author "self-review" with no forward reference
  to `aiwfx-wrap-milestone` step 2's prescribed independent two-lens review.
  Verified against the live embedded SKILL.md — the forward reference is
  still missing, so a reader of start-milestone alone can still conclude
  self-review is the last word.

## Scope

Remaining work under this milestone is narrow: add the forward reference from
`aiwfx-start-milestone` step 7/8 to `aiwfx-wrap-milestone` step 2's
independent two-lens review, with a structural test under
`internal/policies/` pinning the new wording (per the
skill-edit-structural-test-backstop). Closes G-0271.

## Acceptance criteria
