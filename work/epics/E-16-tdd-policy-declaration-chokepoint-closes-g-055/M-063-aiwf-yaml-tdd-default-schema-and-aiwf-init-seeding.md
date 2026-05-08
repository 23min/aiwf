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

The package that owns `aiwf.yaml` parsing (likely `internal/configyaml/`) gains a `tdd` mapping with a `default` string field. A YAML like:

```yaml
tdd:
  default: required
```

parses cleanly into the loaded config struct with no findings. Same for `advisory` and `none`. The field is optional at the schema level — its absence is not itself a parse error (the *consumer* of the field, M-062's resolver, decides what to do with absence). The accessor returns the empty string when the field is absent so callers can check presence with a simple comparison, matching the existing pattern for other optional config fields (e.g. `tree.strict`, `html.commit_output`).

### AC-2 — Schema rejects values outside required, advisory, none

A YAML like `tdd: { default: yes }` (or any value outside the closed set, including capitalization variants `Required`, `REQUIRED`, empty string, and non-strings like `tdd: { default: true }`) produces a parse error at config-load time, **not** later when M-062's resolver runs. The error names the field path (`tdd.default`), the rejected value, and the allowed set. Validation lives in the same place the existing closed-set validators do (per the codebase pattern for `tree.strict` and similar) — single source of truth, reused by M-064's update verb when it inserts the key.

### AC-3 — aiwf init seeds tdd.default: required into new aiwf.yaml

`aiwf init` against a fresh tempdir produces an `aiwf.yaml` whose top level contains:

```yaml
tdd:
  default: required
```

The seeded value is hard-coded to `required` (not configurable via an `init` flag — the project default is the project default; per-milestone overrides are how individual milestones diverge). The key block is inserted at a stable position in the seeded file (consistent with how other init-seeded keys are ordered today). Verified by an init-into-tempdir test that diffs the produced `aiwf.yaml` against a golden fixture.

### AC-4 — Seeded comment names override paths and closed set

The `tdd:` block in the init-seeded `aiwf.yaml` carries a comment block immediately above it that names:

- The closed set (`required | advisory | none`).
- The recommended value for engineering repos (`required`) and the rationale (one short line — TDD is the kernel's intended posture).
- The two override paths: per-milestone via `aiwf add milestone --tdd none`, and repo-wide by editing the field.
- A pointer to the `aiwf-add` skill for the full picture.

The comment shape is asserted by the same init-tempdir golden test from AC-3 — the comment is part of the contract, not decoration.

### AC-5 — Init'd aiwf.yaml is consumed by M-062 resolver as fallback

End-to-end check that the schema work in this milestone and the resolver work in M-062 actually compose. Test: `aiwf init` a tempdir, then `aiwf add milestone --epic E-NN --title "..."` (no `--tdd` flag). The resulting milestone's frontmatter contains `tdd: required`, sourced from the init-seeded project default. This is the seam between M-063 and M-062 — neither milestone alone proves it works; the test belongs in M-063 because that's where the project-default surface lands.

### AC-6 — aiwf doctor --self-check passes after init in fresh tempdir

`aiwf doctor --self-check` against a freshly init'd tempdir exits 0 and reports no findings. The new `tdd.default` field appears in any config-snapshot section doctor renders (so the operator can see the active value at a glance). No regression in any existing self-check. The integration test runs `aiwf init` into a tempdir, then `aiwf doctor --self-check`, then asserts exit 0 and an empty findings list — same pattern as M-052's setup-verb integration test.

### AC-7 — Doctor goldens updated with rationale for tdd.default

The doctor goldens (`internal/doctor/testdata/`-or-equivalent) gain whatever rendered-output diff the new `tdd.default` row introduces. Per CLAUDE.md's testing rules, the golden update commit message names the rationale (the new field; M-063) so a future reviewer doing `git blame` on the golden lands on context. No goldens are updated speculatively — only those touched by the new field.

