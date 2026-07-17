---
id: M-0263
title: 'Add the priority read surface: list/status filter, envelope, show'
status: in_progress
parent: E-0066
depends_on:
    - M-0261
tdd: required
acs:
    - id: AC-1
      title: aiwf list --priority returns exactly the matching gaps and decisions
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf status --priority filters its output the same way
      status: met
      tdd_phase: done
    - id: AC-3
      title: the JSON envelope carries priority and aiwf show surfaces it
      status: met
      tdd_phase: done
---

# M-0263 — Add the priority read surface: list/status filter, envelope, show

## Goal

Make `priority` queryable and visible on the text and JSON surfaces: a `--priority <level>` filter on `aiwf list` and `aiwf status`, the value on the JSON envelope entity payload, and `aiwf show` surfacing it.

## Context

Once the field exists (field milestone) and can be set (write-surface milestone), the backlog's original friction — "picking which one to work next requires reading every body" — is answered by filtering. This milestone adds the read paths. Ordering (group-by-status, priority-as-tiebreaker) is explicitly not here; it's deferred to G-0420, so this ships filtering only over the existing id-order sort.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

### AC-1 — aiwf list --priority returns exactly the matching gaps and decisions

### AC-2 — aiwf status --priority filters its output the same way

### AC-3 — the JSON envelope carries priority and aiwf show surfaces it

## Constraints

- `--priority` filters the result set only; it does not change sort order (that is G-0420). The existing id-order sort is untouched.
- The filter is a closed-set value validated the same way the writers validate it — a bad `--priority` value is a usage error, not a silent empty result.
- The JSON envelope carries `priority` on the entity payload through the existing serialization boundary, not a bespoke side-channel.

## Design notes

- Confirm whether a render/list contract pins the entity JSON payload shape before adding the field to the envelope — if so, the contract needs a coordinated bump (open question carried from the epic).
- `aiwf show` may surface the field incidentally if it renders all frontmatter, or need a one-line addition — determine at implementation.

## Surfaces touched

- `internal/cli/list/`, `internal/cli/status/` — the `--priority` filter flag and predicate.
- The JSON envelope entity payload; `aiwf show`.

## Out of scope

- Sort ordering by priority — G-0420.
- The HTML badge (the render milestone).

## Dependencies

- M-0261 — the field and closed-set predicate must exist first. Independent of the write-surface milestone (test fixtures set the field directly).

## References

- G-0078 — the ratified design decisions (filter-only for v1).
- G-0420 — the deferred sort-ordering follow-up.

## Work log

### AC-1 — aiwf list --priority returns exactly the matching gaps and decisions

`--priority` flag on `internal/cli/list/list.go`, threaded through `BuildListRows` and `crossBranchListRows` as an independent AND-ed filter mirroring `--area`'s shape, plus a `Priority` field on `ListSummary` · commit 68377b6d · tests 23/23 new (list unit, cross-branch, dispatcher-seam), 4/4 mutants killed (local-row filter, resolved-cross-branch filter, collision OR-condition, closed-set usage-error check).

Unlike `--area`, `--priority` validates as a hard usage error (mirroring `--kind`'s `IsKnownKind` pattern), not an undeclared-value advisory note — the milestone's own constraint, since priority is a Go-hardcoded closed set rather than an operator-declared one. A kind that never carries a priority (`entity.CarriesOwnPriority`) needs no separate gate in the filter itself: its `Priority` field is always empty, so it never matches a specific `--priority` level — the same mechanism that already excludes an untagged gap/decision.

Adding a parameter to `BuildListRows`'s exported signature rippled to roughly 20 existing call sites across `internal/cli/integration/` (`cross_branch_list_test.go`, `area_filter_test.go`, `list_cmd_test.go`, `canonicalize_render_test.go`) and `list_diag_test.go`'s two `list.Run` calls — mechanical updates, caught by `go vet ./...` the same way M-0262's `AddOptions` ripple was.

### AC-2 — aiwf status --priority filters its output the same way

`--priority` flag on `internal/cli/status/status.go`, plus `FilterStatusByPriority` (scoping only `OpenDecisions`/`OpenGaps`) and a `Priority` field on `StatusEntity`/`StatusGap` · commit 68377b6d (same commit as AC-1 — both surfaces share this milestone's implementation) · tests included in the 23/23 above; 2/2 mutants killed on `FilterStatusByPriority`'s equality checks and the closed-set usage-error check.

Deliberately narrower than `FilterStatusByArea`: epics and milestones are never touched, since priority has no derivation analogous to area's milestone-to-epic rollup — only gap and decision carry one at all. An ADR entry in `OpenDecisions` is excluded from any named `--priority` level the same way an unprioritized gap/decision is, with no special-case code (its `Priority` field is always empty). Confirmed the defensive `e != nil` guard (an id in the report not resolving against the passed tree) is reachable, not compiler-proven-dead — a stale-report/mismatched-tree call is a legitimate theoretical caller shape — and added `TestFilterStatusByPriority_UnknownIDExcluded` rather than annotating it unreachable.

### AC-3 — the JSON envelope carries priority and aiwf show surfaces it

`Priority` fields on `ListSummary`, `StatusEntity`, `StatusGap`, and `ShowView` (all `omitempty`); `aiwf show`'s text header gains a `· priority: <level>` segment · commit 68377b6d · tests included in the 23/23 above; 1/1 mutant killed on show's text-rendering guard (both directions: an unprioritized entity's segment stays absent, a prioritized one's stays present).

Resolved the epic's own open question ("does a render/list contract pin the entity JSON payload shape?") by direct inspection rather than assumption: `internal/contractcheck`/`internal/contractconfig` implement aiwf's *consumer-facing* `contract` entity-kind feature (external schema bindings via `aiwf.yaml: contracts:`), entirely unrelated to this repo's own Go JSON struct shapes — confirmed by this repo carrying zero contract entities and no `contracts:` block. No coordinated version bump was needed; the fields landed as ordinary struct additions. `ShowView` is an explicit, hand-enumerated struct (not a generic frontmatter dump), so `aiwf show` needed a real one-line addition in both `BuildShowView` and `buildCrossBranchShowView` — it did not surface "for free."

### Independent design-quality check

No `wf-rethink` unit applies: this milestone's surface (`--priority` filter flags plus JSON-payload fields on three pre-existing per-verb structs) is a mechanical extension of already-established `--area` patterns in the same three files, not a new module boundary, core abstraction, or data model.
