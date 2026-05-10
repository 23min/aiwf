---
id: M-0051
title: Migrate mutating verbs
status: done
parent: E-0014
acs:
    - id: AC-1
      title: add, promote, cancel, rename, reallocate, import migrated to Cobra
      status: met
    - id: AC-2
      title: Single-commit-per-verb invariant preserved for each mutating verb
      status: met
    - id: AC-3
      title: Trailer keys aiwf-verb/aiwf-entity/aiwf-actor preserved byte-exact
      status: met
    - id: AC-4
      title: Provenance trailer coherence rules preserved across migrated verbs
      status: met
    - id: AC-5
      title: Repo lock contract preserved for each mutating verb
      status: met
    - id: AC-6
      title: Subprocess integration tests pass for all six mutating verbs
      status: met
---

## Goal

Migrate `add`, `promote`, `cancel`, `rename`, `reallocate`, and `import` — the verbs that produce git commits. The single-commit-per-verb invariant and the trailer-key contract (`aiwf-verb`, `aiwf-entity`, `aiwf-actor`) are non-negotiable; preservation is the central acceptance criterion.

## Approach

Subprocess integration tests are the proof of behavior preservation. Provenance trailer coherence (the `--actor` × `--principal` coupling rules from the I2.5 allow-rule) needs explicit Cobra-side wiring — Cobra's `PreRunE` hooks are the natural place to centralize that check across the mutating verbs rather than re-deriving it per verb. Repo-lock acquisition stays in the verb body; it's not a Cobra concern.

## Acceptance criteria

### AC-1 — add, promote, cancel, rename, reallocate, import migrated to Cobra

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-051/AC-1` for the actual implementation history._

### AC-2 — Single-commit-per-verb invariant preserved for each mutating verb

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-051/AC-2` for the actual implementation history._

### AC-3 — Trailer keys aiwf-verb/aiwf-entity/aiwf-actor preserved byte-exact

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-051/AC-3` for the actual implementation history._

### AC-4 — Provenance trailer coherence rules preserved across migrated verbs

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-051/AC-4` for the actual implementation history._

### AC-5 — Repo lock contract preserved for each mutating verb

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-051/AC-5` for the actual implementation history._

### AC-6 — Subprocess integration tests pass for all six mutating verbs

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-051/AC-6` for the actual implementation history._
