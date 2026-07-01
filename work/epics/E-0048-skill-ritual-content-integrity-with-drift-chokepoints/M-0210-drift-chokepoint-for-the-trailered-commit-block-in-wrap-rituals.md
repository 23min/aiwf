---
id: M-0210
title: Drift chokepoint for the trailered-commit block in wrap rituals
status: in_progress
parent: E-0048
tdd: required
acs:
    - id: AC-1
      title: Chokepoint requires the trailered-commit prescription at both wrap rituals
      status: met
      tdd_phase: done
    - id: AC-2
      title: Chokepoint pins caveat and identity-rule at every ritual trailer site
      status: met
      tdd_phase: done
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
ADR-0024 is rejected in favour of this design; the rationale is recorded in the
Decisions section below (no new ADR, per the operator's lighter path).

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

### AC-1 — presence guard
`PolicyM0210TrailerCommitDrift` (AC-1 loop) requires both wrap rituals to carry
a `git commit --trailer` block naming all three keys; the `required-missing`,
`required-no-block`, and `required-missing-key` firing fixtures pin the branch.
Delivered in the single policy commit `19aa526c`.

### AC-2 — caveat/identity accompaniment guard
`PolicyM0210TrailerCommitDrift` (AC-2 loop) requires the canonical caveat at
every trailered-commit block and the identity rule at every staged-merge
trailered commit, across all rituals; the `wrap-missing-caveat` and
`merge-missing-identity` firing fixtures pin the two branches. Delivered in
commit `19aa526c` (same policy; two facets).

## Decisions made during implementation

- **Chokepoint over extraction — ADR-0024 rejected (`693e130f`).** M-0210
  originally proposed extracting the trailered-commit block into a
  `wf-commit-trailers` reference skill (ADR-0024). The operator rejected that in
  favour of a drift chokepoint (`PolicyM0210TrailerCommitDrift`). Rationale: the
  epic's load-bearing goal is drift-safety, which a mechanical guard meets
  without a new reference-skill category (every existing `wf-*` skill is a
  runnable procedure, not a look-up reference) or a mid-merge skill-invocation
  step; the extraction's token benefit was marginal (a ritual body loads only
  when that ritual runs). The block stays inline where it is run. ADR-0024 was
  promoted `proposed → rejected` with the rationale in its `aiwf history`; no
  replacement ADR was written (the operator chose the lighter path — rationale
  in this spec).

## Validation

- `make check-fast` (go vet + golangci-lint + full `go test`): green — exit 0,
  lint clean (a gocritic `filepathJoin` finding was fixed during the refactor
  phase), all packages `ok`.
- New-file coverage: every function 100% except the single
  `filepath.Glob` error-return line, which is annotated `//coverage:ignore`
  (a fixed literal glob pattern is never `ErrBadPattern`). The diff-scoped
  coverage gate escapes that line.
- Firing-fixture meta-gate (G-0259): `m0210-trailer-commit-drift` is not in
  `grandfatherDark`; its six `m0210/*` fixtures light the construction line.
- Independent adversarial review: **APPROVE**. Non-vacuity verified by
  revert-and-test — stripping the caveat, the identity rule, and a trailer key
  from the real `aiwfx-wrap-epic/SKILL.md` each reddened the live positive test,
  restored clean. Scope, detector tightness, and the `aiwfx-release` exclusion
  (it carries no trailered-commit block) all confirmed by measurement.

## Deferrals

None filed as gaps. The reviewer flagged (non-blocking) that the shared
`TestFiringFixtures_MultiSite` harness asserts only `len(vs) > 0`, so a fixture
does not by itself prove its *named* branch fired (the diff-scoped coverage gate
plus live correctness carry that guarantee). This is the established repo-wide
pattern (m0132 / m0202 / trailer-order share it), not introduced here; a durable
fix (an optional per-row `wantDetail` assertion on the shared harness) would
benefit all policies and was consciously left out of this milestone's scope —
not filed as a gap per the operator's steer.

## Reviewer notes

Independent fresh-context reviewer returned **APPROVE**, verifying every
load-bearing claim by measurement (live-tree green; non-vacuity via
revert-and-test on all three facets; six firing fixtures; coverage; G-0259
meta-gate; detector tightness; `aiwfx-release` exclusion; scope discipline — the
two wrap `SKILL.md` files are untouched, as the chokepoint-only reframe
intends). Three non-blocking findings, all net-behavior-correct, accepted as-is
rather than churning correct code:

- The shared firing-fixture harness pins branches only via `len(vs) > 0` (the
  repo-wide pattern; see Deferrals).
- `aiwf-verb`, when absent, surfaces as "no trailered-commit block" rather than
  "missing key" — because block-detection itself keys on `aiwf-verb`; the
  all-three-keys guarantee still holds.
- AC-1's caveat requirement for the required wraps is enforced structurally in
  the AC-2 loop (which iterates all rituals with a block, the wraps included),
  not the AC-1 loop; net behaviour matches AC-1's stated requirement, confirmed
  by the revert-test.
