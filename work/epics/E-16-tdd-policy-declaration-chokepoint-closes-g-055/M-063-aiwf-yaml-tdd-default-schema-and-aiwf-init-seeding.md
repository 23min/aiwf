---
id: M-063
title: aiwf.yaml tdd.default schema and aiwf init seeding
status: draft
parent: E-16
tdd: required
acs:
    - id: AC-1
      title: tdd.default field accepted in aiwf.yaml schema
      status: open
      tdd_phase: red
    - id: AC-2
      title: Schema rejects values outside required, advisory, none
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'aiwf init seeds tdd.default: required into new aiwf.yaml'
      status: open
      tdd_phase: red
    - id: AC-4
      title: Seeded comment names override paths and closed set
      status: open
      tdd_phase: red
    - id: AC-5
      title: Init'd aiwf.yaml is consumed by M-062 resolver as fallback
      status: open
      tdd_phase: red
    - id: AC-6
      title: aiwf doctor --self-check passes after init in fresh tempdir
      status: open
      tdd_phase: red
    - id: AC-7
      title: Doctor goldens updated with rationale for tdd.default
      status: open
      tdd_phase: red
---

## Goal

Add `tdd.default` (closed set: `required | advisory | none`) to the `aiwf.yaml` schema so the project-level fallback used by [M-062](M-062-tdd-flag-on-aiwf-add-milestone-with-project-default-fallback.md)'s resolver is a recognized field rather than a magic string. `aiwf init` writes `tdd.default: required` into freshly-created `aiwf.yaml` files with an explanatory comment block, so new consumer repos start in the explicit-policy posture without any extra step from the operator.

The shipped default is `required` (not `none`) by design: aiwf's intended use case is engineering work where TDD is the norm, and shipping `none` would silently reproduce the gap [G-055](../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) documents.

## Approach

Extend the existing config struct in the package that owns `aiwf.yaml` parsing (likely `internal/configyaml/`) with the `tdd.default` field; add validation rejecting values outside the closed set. Update the `aiwf init` template that seeds `aiwf.yaml` for new repos to include the new key with `required` and a comment explaining the override paths (per-milestone `--tdd none`; repo-wide via the field). Doctor goldens get refreshed with rationale per the M-052 / hook-installer pattern.

Tests cover schema parse + validate (positive + each negative), init-into-tempdir produces the expected `aiwf.yaml`, and `aiwf doctor --self-check` passes against the init'd repo.

The grandfather rule for existing milestones (no retroactive audit) is enforced *by* the resolver in M-062, not here — this milestone just makes the field exist and the default land in new repos. Existing repos absorb the field via M-064.

## Acceptance criteria

### AC-1 — tdd.default field accepted in aiwf.yaml schema

### AC-2 — Schema rejects values outside required, advisory, none

### AC-3 — aiwf init seeds tdd.default: required into new aiwf.yaml

### AC-4 — Seeded comment names override paths and closed set

### AC-5 — Init'd aiwf.yaml is consumed by M-062 resolver as fallback

### AC-6 — aiwf doctor --self-check passes after init in fresh tempdir

### AC-7 — Doctor goldens updated with rationale for tdd.default

