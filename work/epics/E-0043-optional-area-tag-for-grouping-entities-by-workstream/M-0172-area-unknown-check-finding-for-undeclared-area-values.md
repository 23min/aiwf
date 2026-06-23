---
id: M-0172
title: area-unknown check finding for undeclared area values
status: in_progress
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: Declared area produces no finding
      status: met
      tdd_phase: done
    - id: AC-2
      title: Undeclared area fires area-unknown naming id, value, and set
      status: met
      tdd_phase: done
    - id: AC-3
      title: Absent, empty, or null area never fires
      status: open
      tdd_phase: green
    - id: AC-4
      title: Inert when no areas block is declared
      status: open
      tdd_phase: red
    - id: AC-5
      title: Archived entities never fire
      status: open
      tdd_phase: red
    - id: AC-6
      title: Finding code carries a hint and is discoverable
      status: open
      tdd_phase: red
---
## Goal

Add the `area-unknown` `aiwf check` finding: the present-⇒-declared chokepoint. When an entity's `area` is present and non-empty but not in the `aiwf.yaml: areas` member set, the check flags it (typo protection). Absence is never evaluated, and the rule is inert when no `areas` block exists.

## Context

M-0171 makes the `area` field and the `aiwf.yaml: areas` block exist and parse, but deliberately does not validate an entity's area against the declared set. This milestone adds that validation as a check rule — the authoritative surface (a creation-time flag alone can't catch a hand-edit or an `aiwf import` that introduces an undeclared area), mirroring the defense-in-depth pattern G-0268's `milestone-tdd-undeclared` follows.

The rule lives in `internal/check/` as `AreaUnknown(tree, declared)` but is composed at the CLI layer (`internal/cli/check`) with the declared set sourced from `aiwf.yaml: areas` — the same seam `TreeDiscipline`, the contract checks, and the tests-metrics check already use. The pure `check.Run` stays config-agnostic, exactly as M-0171/AC-4's metamorphic guard pins.

## Acceptance criteria

### AC-1 — Declared area produces no finding

An entity whose `area` is a member of the `aiwf.yaml: areas` declared set produces no `area-unknown` finding.

Evidence: a check test over a tree with a declared set and a root entity whose `area` is in the set asserts zero `area-unknown` findings.

### AC-2 — Undeclared area fires area-unknown naming id, value, and set

An entity whose `area` is present, non-empty, and not a member of the declared set produces exactly one `area-unknown` finding (warning severity) whose message names the entity id, the offending value, and the declared set.

Evidence: a check test asserting the finding's `Code`, `Severity`, `EntityID`, and that the `Message` names the id, the offending value, and the declared members.

### AC-3 — Absent, empty, or null area never fires

An entity with no `area`, an empty `area`, or an explicit null `area` never produces the finding — absence is never evaluated, only present-and-non-empty values are.

Evidence: a table test over absent / empty values (all deserialize to `""`) asserting zero `area-unknown` findings even when an `areas` block is declared.

### AC-4 — Inert when no areas block is declared

With no `areas` block in `aiwf.yaml` (empty declared set), the rule is inert: no findings regardless of entity `area` values, present or undeclared.

Evidence: a check test passing a nil / empty declared set with area-tagged entities asserts zero findings. Complements M-0171/AC-4's metamorphic guard that `check.Run` itself stays area-agnostic.

### AC-5 — Archived entities never fire

An entity under a per-kind `archive/` subdirectory (ADR-0004 §"`aiwf check` shape rules") never fires the finding, consistent with the other shape-and-health rules.

Evidence: a check test where an archived entity carries an undeclared area asserts zero findings while its active-tree twin fires.

### AC-6 — Finding code carries a hint and is discoverable

The `area-unknown` code is registered as a typed `Code*` constant, carries a `hintTable` entry, and is documented in the `aiwf-check` skill — so the three finding-code policies (`finding-codes-have-tests`, `finding-codes-have-hints`, `finding-codes-are-discoverable`) pass.

Evidence: `PolicyFindingCodesHaveTests` / `PolicyFindingCodesHaveHints` / `PolicyFindingCodesAreDiscoverable` green; a hint-presence assertion for the code.

## Constraints

- **Single source of truth** for the declared set is `aiwf.yaml: areas` — the same accessor M-0171 introduces; no parallel reader.
- **Severity is `warning`, no new strictness knob.** Settled per the spec's lean and the "don't invent a knob speculatively" YAGNI constraint; escalation can be added later under an existing or new knob if real friction shows.

## Out of scope

- The `aiwf add --area` write path (separate milestone).
- Read-surface filtering or grouping.
- Any auto-correction of an unknown area — the finding reports; the operator fixes.

## Dependencies

- M-0171 — the `area` field and `aiwf.yaml: areas` block + accessor.

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)
- G-0268's `milestone-tdd-undeclared` — the archive-scoped check-finding pattern this rule follows.
- `check.TreeDiscipline` — the config-dependent tree rule composed at the CLI layer that this rule mirrors.
