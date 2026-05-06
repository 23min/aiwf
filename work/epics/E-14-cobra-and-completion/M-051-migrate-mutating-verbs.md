---
id: M-051
title: Migrate mutating verbs
status: draft
parent: E-14
acs:
    - id: AC-1
      title: add, promote, cancel, rename, reallocate, import migrated to Cobra
      status: open
---

## Goal

Migrate `add`, `promote`, `cancel`, `rename`, `reallocate`, and `import` — the verbs that produce git commits. The single-commit-per-verb invariant and the trailer-key contract (`aiwf-verb`, `aiwf-entity`, `aiwf-actor`) are non-negotiable; preservation is the central acceptance criterion.

## Approach

Subprocess integration tests are the proof of behavior preservation. Provenance trailer coherence (the `--actor` × `--principal` coupling rules from the I2.5 allow-rule) needs explicit Cobra-side wiring — Cobra's `PreRunE` hooks are the natural place to centralize that check across the mutating verbs rather than re-deriving it per verb. Repo-lock acquisition stays in the verb body; it's not a Cobra concern.

## Acceptance criteria

### AC-1 — add, promote, cancel, rename, reallocate, import migrated to Cobra

