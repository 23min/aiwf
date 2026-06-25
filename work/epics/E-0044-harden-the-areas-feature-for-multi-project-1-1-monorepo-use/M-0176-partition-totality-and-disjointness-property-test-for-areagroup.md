---
id: M-0176
title: Partition totality and disjointness property test for areagroup
status: done
parent: E-0044
tdd: advisory
acs:
    - id: AC-1
      title: 'Partition is total and disjoint: every item in exactly one group'
      status: met
    - id: AC-2
      title: Complement holds exactly the untagged and undeclared items
      status: met
    - id: AC-3
      title: Declared areas keep members order; empty suppressed; complement last
      status: met
---

## Goal

Mechanically guarantee that `internal/areagroup.Partition` never silently drops or duplicates an item: for any input, every item lands in exactly one output group. Turns the view-layer drop failure from "hoped-for" into "impossible" — the Tier-0 floor under E-0044's trust claim.

## Context

E-0043 shipped `areagroup.Partition` as the single source of the area-partition logic shared by `status`, roadmap, and HTML renders. Its correctness is currently pinned only by example-based tests; a refactor that drops an item into neither bucket — or into both — would pass them. This milestone replaces that hope with a generative property, per the `wf-property-test` skill. No production change is expected unless the property surfaces a real defect.

## Acceptance criteria

Formalized at start-milestone as AC-1–AC-3 (frontmatter `acs[]`; full statements and their pinning tests are under the AC sections below). Summary:

- **AC-1 · Totality + disjointness** — every input item appears in exactly one output group (count-in == count-out; no item in two groups; none dropped).
- **AC-2 · Complement correctness** — the complement group (Area "") holds exactly the items whose area is "" or not a declared member, and nothing else.
- **AC-3 · Declared order + suppression** — declared areas appear in `members` order; an empty declared area is suppressed; the complement is always emitted last.

## Constraints

- Pure test addition on `internal/areagroup`; no change to `Partition`'s signature or behavior unless the property catches a real bug — in which case the fix lands here with its own regression test.
- The generator covers arbitrary item slices, arbitrary `areaOf` mappings (including "", declared, and undeclared values), and arbitrary `members` / `defaultLabel`.

## Out of scope

- The `paths:` oracle and any path-based checks (Tier 1+).
- Redesigning `Partition`'s ordering or emptiness policy — those are pinned, not changed.

## Dependencies

- None. Independent Tier-0 hardening; parallel with the other Tier-0 milestones.

## References

- `internal/areagroup/areagroup.go` — the `Partition` helper under test.
- `wf-property-test` skill — the generative-property discipline this milestone applies.

### AC-1 — Partition is total and disjoint: every item in exactly one group

**Property.** For any input, the multiset of items across all output groups equals the input multiset: every item appears in exactly one group — none dropped, none duplicated, none fabricated (`count-in == count-out`).

**Mechanical assertion.** `TestPartition_Property_TotalAndDisjoint` in [`internal/areagroup/partition_property_test.go`](../../../internal/areagroup/partition_property_test.go) drives `testing/quick` over 2000 generated inputs (deterministic, fixed seed), flattens the groups, and asserts each input id occurs exactly once. Vacuity-checked: deleting an item from the complement (`count-out 3 != 7`) and emitting a declared group twice (`count-out 9 != 7`) each turn it red.

### AC-2 — Complement holds exactly the untagged and undeclared items

**Property.** Exactly one group carries the complement marker (`Area ""`); it holds exactly the items whose area is `""` or not a declared member, in input order; and its label is the configured `areas.default` — or the built-in `DefaultComplementLabel` fallback when that is empty.

**Mechanical assertion.** `TestPartition_Property_ComplementCorrect` derives the expected complement independently from the member set and asserts membership (by id, in input order), single-complement-ness, and label correctness across the generated inputs. Vacuity-checked: dropping complement items and emitting the complement twice each turn it red.

### AC-3 — Declared areas keep members order; empty suppressed; complement last

**Property.** Declared areas appear in `members` order, each carrying its items in input order and labelling itself with its area; a declared area with no items is suppressed; the complement is always the final group.

**Mechanical assertion.** `TestPartition_Property_DeclaredOrderAndComplementLast` reconstructs the expected declared sequence (members order, non-empty only) and asserts the emitted order, per-group items, `Label == Area`, and complement-last. Vacuity-checked: emitting the complement first rather than last (`found 2 complement groups`) and duplicating a declared group each turn it red.

## Work log

### AC-1 / AC-2 / AC-3 — partition property tests

Added [`internal/areagroup/partition_property_test.go`](../../../internal/areagroup/partition_property_test.go): a `testing/quick` generator (`partitionInput.Generate` — deterministic fixed seed, 2000 cases per property) and three property tests, one per AC. **No production change** — `Partition` already satisfies all three properties; this milestone replaces example-only pinning with a generative floor (per the `wf-property-test` skill). Vacuity confirmed by three deliberate mutations (item drop, duplicate declared group, complement-first); each is caught by the expected property and the originals stay green.

- tests: 3 property tests green; full-package `go test -race` green
- lint: `golangci-lint run ./internal/areagroup/...` → 0 issues
- commit: `adc65280` (`test(areagroup): generative property tests for Partition totality/disjointness`)

## Decisions made during implementation

No architectural decisions — the milestone is a local test addition. One implementation choice worth recording: the generator uses stdlib `testing/quick` with a fixed-seed `*rand.Rand` rather than native Go fuzzing (`Fuzz*`). Rationale: `testing/quick` runs the full 2000-case sample on every `go test`, so the Tier-0 floor bites on every CI run, whereas a `Fuzz*` test exercises only its seed corpus in routine CI (broad exploration happens just in the scheduled `fuzz` workflow). Deterministic-by-seed satisfies the repo's no-wall-clock test rule, and no new dependency is added. Not worth an architectural decision record — no cross-cutting contract.

## Validation

- `make ci` (vet + lint + race-coverage + self-check) — green; `aiwf doctor --self-check` 29/29.
- `go test -race ./internal/areagroup/...` — green (pre-existing 4 example tests + 3 new property tests).
- `golangci-lint run ./internal/areagroup/...` — 0 issues.
- `aiwf check` — 0 errors. Remaining findings are advisory: `acs-tdd-audit` × 3 (ACs `met` under `tdd: advisory` with `tdd_phase` absent — expected for characterization testing of already-green code, see Reviewer notes) and one environmental `provenance-untrailered-scope-undefined` (no upstream ref configured in the epic worktree).
- `Partition` production code byte-identical to base — pure characterization hardening, no behavior change.

## Deferrals

None.

## Reviewer notes

- **Independent two-lens review: APPROVE.** A fresh-context reviewer (no authorship attachment) verified every load-bearing claim by measuring, not reasoning. It ran 11 deliberate `Partition` mutations — the 3 documented in the Work log plus 8 of its own (mis-bucketing, suppression-drop, label/order breaks, broken empty-default fallback) — and each was caught by the expected property. Tautology was ruled out: when production routed all items into the complement, AC-2's independent oracle still caught it. Generator breadth was instrumented across 2000×3 cases (declared buckets populated in ~60% of cases; no category starved). `wf-rethink` was not run — the milestone introduces no new module, abstraction, or data model, only a local test-only generator.
- **TDD posture.** The milestone is `tdd: advisory`, and this characterizes already-correct code, so there is no red→green production cycle and the ACs are `met` with `tdd_phase` absent. The resulting `acs-tdd-audit` warnings are advisory by design and accepted as the honest record; clearing them via the full phase progression (~15 commits) would be disproportionate, and `tdd: none` has no clean verb path.
- **Non-blocking, left as-is** (reviewer-flagged): the illustrative failure-count literals in the AC bodies are example shapes, not pinned assertions; the generator ceilings (5 members / 12 items) are modest but adequate for a Tier-0 floor.

