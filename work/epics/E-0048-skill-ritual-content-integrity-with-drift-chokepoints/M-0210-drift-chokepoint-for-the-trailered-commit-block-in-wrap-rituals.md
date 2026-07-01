---
id: M-0210
title: Drift chokepoint for the trailered-commit block in wrap rituals
status: in_progress
parent: E-0048
tdd: required
acs:
    - id: AC-1
      title: Chokepoint requires the trailered-commit prescription at both wrap rituals
      status: open
      tdd_phase: done
    - id: AC-2
      title: Chokepoint pins caveat and identity-rule at every ritual trailer site
      status: open
      tdd_phase: refactor
---
## Goal

The trailered-commit / trailered-merge prescription duplicated across
`aiwfx-wrap-epic` and `aiwfx-wrap-milestone` — the three trailer keys
(`aiwf-verb` / `aiwf-entity` / `aiwf-actor`), the `git commit --trailer`
template, the variant-casings caveat, and the `git config user.email` identity
rule — stays **inline** at each trailered-commit site (the command stays visible
where it is run) and is protected against drift by a mechanical chokepoint
rather than extracted into a new reference skill. The policy pins the failure
mode G-0219 actually hit: a wrap ritual shipping its trailered commit stripped
of the caveat / identity prescription its sibling carries.

This reframes the milestone away from the reference-skill extraction ADR-0024
proposed. The epic's load-bearing goal is drift-safety; a chokepoint meets it
without introducing a new reference-skill category or a mid-merge
skill-invocation step, at the cost of leaving the block duplicated — a marginal
per-invocation token cost, since a ritual body loads only when that ritual runs.
ADR-0024 is rejected in favour of this design; the counter-decision is recorded
so a future "should I extract this shared block?" reader finds the reasoning.

The guard is **per-ritual** (file-level), not per-`git commit --trailer` site:
it asserts the canonical caveat / identity prescription appears somewhere in any
ritual that carries a trailered-commit block. That granularity matches the
current bodies (`aiwfx-wrap-epic` states the caveat once, not at both of its
trailer sites) and is sufficient to catch the G-0219 drift class. It pins
*presence of the canonical prescription*, not byte-identity across sites.

## Acceptance criteria

### AC-1 — Chokepoint requires the trailered-commit prescription at both wrap rituals

A new `internal/policies/` policy asserts that `aiwfx-wrap-epic` and
`aiwfx-wrap-milestone` each carry a trailered-commit block naming all three
kernel-required trailer keys (`aiwf-verb`, `aiwf-entity`, `aiwf-actor`) and the
canonical variant-casings caveat. This catches G-0219's failure mode — a wrap
ritual whose trailered-commit prescription is absent or asymmetric with its
sibling's.

Mechanical evidence: the policy passes green on the live tree (both wraps carry
the prescription today) and reddens on a firing fixture — a required wrap whose
trailered-commit block is stripped of its caveat or of a trailer key.

### AC-2 — Chokepoint pins caveat and identity-rule at every ritual trailer site

The same policy asserts the consistency (single-source-by-policy) property: for
every embedded ritual containing a `git commit --trailer "aiwf-verb:` block the
canonical variant-casings caveat must be present, and for every
merge-then-trailered-commit site the `git config user.email`
identity-resolution rule must be present. A reword that drops the canonical
caveat or identity prescription — anywhere a trailered commit is composed, not
only in the two named wraps — fails CI.

Mechanical evidence: firing fixtures feed a synthetic ritual with a trailer
block missing the caveat, and a merge block missing the identity rule; the
policy returns a violation for each. The fixtures light the policy's
construction line, satisfying the firing-fixture meta-gate (G-0259) for a policy
added after that gate.

## Work log

_(one entry per AC, filled during implementation)_

## Decisions made during implementation

- **Chokepoint over extraction — ADR-0024 rejected.** The trailered-commit block
  stays inline; a drift chokepoint replaces the proposed `wf-commit-trailers`
  reference-skill extraction. Rationale: the epic's load-bearing goal is
  drift-safety, which a mechanical guard meets without a new reference-skill
  category or a mid-merge invocation step; the extraction's token benefit was
  marginal. Decision-record reference filled when the rejection lands.

## Validation

_(test-suite, lint, and coverage-gate results, filled at wrap)_

## Deferrals

None expected — the drift-safety concern is closed by this milestone's
chokepoint.

## Reviewer notes

_(trade-offs and deliberate omissions, filled at wrap)_
