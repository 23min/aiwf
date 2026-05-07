---
id: M-064
title: aiwf update migration for existing aiwf.yaml with loud output
status: draft
parent: E-16
tdd: required
acs:
    - id: AC-1
      title: 'aiwf update inserts tdd.default: required when missing'
      status: open
      tdd_phase: red
    - id: AC-2
      title: aiwf update leaves an existing tdd.default value unchanged
      status: open
      tdd_phase: red
    - id: AC-3
      title: Comments and key order in aiwf.yaml preserved across update
      status: open
      tdd_phase: red
    - id: AC-4
      title: Re-running aiwf update is a no-op (idempotent)
      status: open
      tdd_phase: red
    - id: AC-5
      title: 'Text output includes aiwf.yaml: section listing changes'
      status: open
      tdd_phase: red
    - id: AC-6
      title: JSON envelope mirrors changes in result.changes[]
      status: open
      tdd_phase: red
    - id: AC-7
      title: No-change runs surface tdd.default presence (not silent)
      status: open
      tdd_phase: red
    - id: AC-8
      title: Subprocess integration test covers insert, skip, idempotent
      status: open
      tdd_phase: red
---

## Goal

Existing consumer repos absorb `tdd.default: required` automatically when they next run `aiwf update`, without overwriting any value the human set deliberately. The verb's output makes the change visible in both human-readable text and the `--format=json` envelope, so the operator (or CI) sees the policy shift land at exactly the moment it takes effect — not buried in release notes, not delayed until the next `aiwf add milestone` surprises them with a refusal.

`aiwf upgrade` already calls `aiwf update` as its post-install step, so wiring this through `aiwf update` covers both invocation paths with one implementation.

## Approach

`aiwf update` reads the consumer repo's `aiwf.yaml`, detects whether `tdd.default` is present (any value), and inserts `tdd.default: required` at top level with a comment block when missing. Insertion preserves surrounding comments and key order — use a YAML library that keeps positional context (e.g. `yaml.v3` Node API) rather than the round-trip approach which strips comments. Idempotent: a second run is a no-op (and the no-op is also surfaced loudly so the operator gets confirmation, not silence).

Loud-output shape per the [G-055](../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) spec — text mode prints a clearly-separated `aiwf.yaml:` section listing each change (key added, value, note); `--format=json` envelope mirrors this in `result.changes[]` with fields `path`, `key`, `value`, `note`. Tests cover missing-key insertion, present-key skip (any value), comment + key-order preservation, no-op idempotency on rerun, and the JSON envelope shape.

## Acceptance criteria

### AC-1 — aiwf update inserts tdd.default: required when missing

### AC-2 — aiwf update leaves an existing tdd.default value unchanged

### AC-3 — Comments and key order in aiwf.yaml preserved across update

### AC-4 — Re-running aiwf update is a no-op (idempotent)

### AC-5 — Text output includes aiwf.yaml: section listing changes

### AC-6 — JSON envelope mirrors changes in result.changes[]

### AC-7 — No-change runs surface tdd.default presence (not silent)

### AC-8 — Subprocess integration test covers insert, skip, idempotent

