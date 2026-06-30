---
id: M-0196
title: Skill-edit structural-test backstop policy
status: in_progress
parent: E-0048
depends_on:
    - M-0195
tdd: required
acs:
    - id: AC-1
      title: Policy fires on an unbacked ritual SKILL.md edit
      status: open
      tdd_phase: red
    - id: AC-2
      title: Policy stays silent when the edit is backed by a referencing test
      status: open
      tdd_phase: red
    - id: AC-3
      title: Gate is diff-scoped and inert without a base ref
      status: open
      tdd_phase: red
    - id: AC-4
      title: Gate wired into CI coverage-gate step and Makefile target
      status: open
      tdd_phase: red
    - id: AC-5
      title: Chokepoint documented in CLAUDE.md table and authoring section
      status: open
      tdd_phase: red
---
## Goal

Make the skill-edit → structural-test discipline mechanical, so a ritual
`SKILL.md` edit cannot ship to consumers without a paired structural test.
Shipped ritual content (`internal/skills/embedded-rituals/**/SKILL.md`) is
materialized into consumer repos by `aiwf init` / `aiwf update`; the kernel
design requires each prescriptive edit to be pinned by a structural test under
`internal/policies/` that fails if the prescription drifts. Today that
discipline is operator vigilance only — the M-0160 incident (a drive-by skill
edit at commit `5cf007f5`) passed pre-commit and pre-push and was caught only by
human review, exactly the dependency the framework exists to remove.

This milestone adds a **diff-scoped policy** under `internal/policies/` that,
given a base ref, flags any commit modifying an embedded-rituals `SKILL.md`
whose edit is not referenced by any structural test under `internal/policies/`.
It lives as a Go policy test (CI tier), not an `aiwf check` finding, because the
property it polices — "this aiwf-repo skill edit has a paired
`internal/policies/` test" — is an aiwf-repo development invariant, meaningless
in a consumer tree where rituals are materialized rather than authored. It
reuses the diff-scoped base-ref plumbing of the existing coverage gate
(`branch_coverage_audit`) and runs in CI's coverage-gate step.

v1 granularity is **file-existence + skill-reference**: the edited `SKILL.md`
path is referenced by some `internal/policies/*.go` source. The stronger "the
test references the changed section" property is deferred to a follow-up gap,
mirroring how the coverage gate shipped statement-scoped with branch-scoped
deferred — the v1 catches the actually-observed failure mode (M-0160 shipped
with zero test), and the residual stale-test case is the follow-up.

Source: G-0220. Parent epic E-0048.

## Acceptance criteria

### AC-1 — Policy fires on an unbacked ritual SKILL.md edit

Given a set of changed embedded-rituals `SKILL.md` paths and the set of
path-references found across `internal/policies/*.go`, the pure detector returns
a violation for each changed path that no policy source references. The
violation names the offending skill path so the operator sees exactly which edit
lacks a backstop.

Test: drive the detector with a synthetic changed-path that has no matching
reference; assert exactly one violation naming that path. This test also lights
the policy's `Violation` construction line, satisfying the G-0259
firing-fixture meta-gate with no new `grandfatherDark` entry.

### AC-2 — Policy stays silent when the edit is backed by a referencing test

When a changed `SKILL.md` path is referenced by at least one
`internal/policies/*.go` source, the detector returns no violation for it.

Test: drive the detector with a changed-path present in the policy-reference
set; assert no violation. A mixed input — one backed path and one unbacked path
— returns exactly the unbacked violation, proving the detector discriminates per
path rather than all-or-nothing.

### AC-3 — Gate is diff-scoped and inert without a base ref

The gate is diff-scoped and inert outside the dedicated CI step.
`TestPolicy_SkillEditStructuralTestBackstop` reads the base ref from the
environment and **skips when it is unset**, so the broad `go test ./...` run is
unaffected — the gate fires only in CI's coverage-gate step, mirroring
`branch_coverage_audit`.

Seam test (per CLAUDE.md "Test the seam, not just the layer"): build a synthetic
git fixture — a base commit plus a HEAD commit that edits an embedded-rituals
`SKILL.md` — and drive the resolver end-to-end, asserting that
`git diff <base>` → changed-paths → detector produces the expected finding. This
proves the IO shell (git-diff resolution + policy-source scan) is wired, not
just the pure detector layer.

### AC-4 — Gate wired into CI coverage-gate step and Makefile target

The gate actually runs at the integration boundary, not just in principle. The
policy test is invoked by the CI test job's coverage-gate step in
`.github/workflows/go.yml` and by the `make coverage-gate` target, alongside the
existing diff-scoped gates (`branch_coverage_audit`, `firing_fixture_presence`).

Test: a structural assertion that both surfaces reference the gate — the
workflow's coverage-gate step and the Makefile target each invoke the policy
test (by run-pattern or by exporting the base ref it consumes) — so the gate
cannot silently fail to run. Scoped to the coverage-gate step / target, not a
flat grep over the file.

### AC-5 — Chokepoint documented in CLAUDE.md table and authoring section

CLAUDE.md documents the chokepoint on both surfaces it belongs:

- The "What's enforced and where" table gains a blocking CI-test row naming the
  policy and its engine file.
- §"Ritual content authoring" gains a sentence requiring every
  embedded-rituals `SKILL.md` edit to land alongside a referencing structural
  test under `internal/policies/`, naming this policy as the mechanical
  chokepoint that replaces operator vigilance (the G-0220 tertiary item, now
  stated as landed rather than pending).

Test: structural assertions scoped to each named section (the enforcement table
and §"Ritual content authoring") — not a flat substring grep over the file —
confirming the row and the backstop sentence are present in the right place.

## Work log

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
