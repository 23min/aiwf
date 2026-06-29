---
id: M-0196
title: Skill-edit structural-test backstop policy
status: draft
parent: E-0048
depends_on:
    - M-0195
tdd: required
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

## Work log

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
