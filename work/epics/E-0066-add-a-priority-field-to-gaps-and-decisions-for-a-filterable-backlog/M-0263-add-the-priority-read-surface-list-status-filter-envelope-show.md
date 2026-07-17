---
id: M-0263
title: 'Add the priority read surface: list/status filter, envelope, show'
status: done
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

### Independent pre-wrap review

An independent fresh-context reviewer audited the full diff against nine load-bearing claims (closed-set hard-usage-error validation on both `list` and `status`; the untagged/non-carrying-kind exclusion mechanism; the deliberate epic/milestone exclusion in `FilterStatusByPriority` — including the reviewer's own independent reasoning for why that's the right call, not just accepting the implementer's rationale; envelope-field population at every construction site incl. cross-branch; the contract-version-bump question; positional correctness of the ~20 `BuildListRows` call-site updates; non-tautological assertions; plausibility of the claimed mutation-probe predicates; soundness of the defensive `e != nil` guard) — all nine held up under independent measurement, including running `go build`/`go vet`/the affected test suite and reading a coverage profile directly rather than trusting the claim. **Verdict: APPROVE.**

One non-blocking finding: `buildCrossBranchShowView`'s `Priority: resolved.Priority` line had statement coverage (via an existing milestone-shaped cross-branch fixture, where `Priority` is always empty) but no value assertion for a real gap/decision. Fixed in-review with `TestBuildShowView_CrossBranchResolved_SurfacesPriority` (commit 17f0823e), mirroring the existing `TestBuildShowView_CrossBranchResolvesAndLabelsContent_M0260AC1AC2` shape with a prioritized gap instead of a milestone.

No `wf-rethink` unit applies: this milestone's surface (`--priority` filter flags plus JSON-payload fields on three pre-existing per-verb structs) is a mechanical extension of already-established `--area` patterns in the same three files, not a new module boundary, core abstraction, or data model.

## Decisions made during implementation

None — the design was fully pre-locked by G-0078's ratified decisions and this spec's own Design notes. The one open question the Design notes flagged (whether a render/list contract pins the entity JSON payload shape) was resolved by direct inspection during AC-3, not a fresh design fork — see AC-3's Work log entry above.

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -race -parallel 8 ./...` (`make test-race`) — all packages pass, no flakes on the final sweep.
- `make lint` (full `golangci-lint` set) — 0 issues.
- `make coverage-gate` (diff-scoped statement-coverage audit + firing-fixture meta-gate) — clean.
- `aiwf check` — 0 error findings; 1 pre-existing warning (`provenance-untrailered-scope-undefined`, no upstream configured for this unpushed branch — expected).
- Manual branch-coverage audit + `wf-vacuity` mutation probe (covering all three ACs together, since they share one implementation pass across the same three files): 7/7 targeted mutations killed — the local-row and resolved-cross-branch priority filters, the collision-row OR-condition, both closed-set usage-error checks (list and status), `FilterStatusByPriority`'s equality checks, and show's text-rendering guard.
- Independent reviewer re-verified all nine Work-log claims by measurement (build/vet/tests/coverage profile), not by trusting the narrative — APPROVE, one non-blocking finding fixed in-review (see "Independent pre-wrap review" above).

## Deferrals

- (none) — G-0420 (sort ordering by priority) and the HTML badge (render milestone) were already scoped out at planning time, not discovered mid-implementation; both are named in this spec's own `## Out of scope` and `## References` sections.

## Reviewer notes

- **`FilterStatusByPriority` deliberately does not mirror `FilterStatusByArea`'s epic-scoping.** Priority has no derivation analogous to area's milestone-to-epic rollup — only gap and decision carry one — so scoping epics/milestones by `--priority` would have no honest meaning and would gut the In-flight section for no reason. The independent reviewer reached the same conclusion via their own reasoning, not by accepting this narrative.
- **The `BuildListRows` signature change rippled to ~20 existing call sites** across `internal/cli/integration/` — caught and fixed mechanically via `go vet ./...`, the same lesson M-0262 named for `AddOptions`. A positional-argument swap (`area` vs. the new `priority` slot) was the specific risk the independent reviewer spot-checked and confirmed absent.
- **One CLI-seam value-assertion gap surfaced by the independent review** (`buildCrossBranchShowView`'s `Priority` field was statement-covered by an existing milestone fixture, where the value is always empty, but never value-asserted for a real gap/decision) — fixed in-review; see "Independent pre-wrap review" above.
