---
id: M-0261
title: Add the priority field, its validation, and drift chokepoints
status: done
parent: E-0066
tdd: required
acs:
    - id: AC-1
      title: priority is an optional gap/decision field validated against its closed set
      status: met
      tdd_phase: done
    - id: AC-2
      title: priority on other kinds raises the priority-not-applicable finding
      status: met
      tdd_phase: done
    - id: AC-3
      title: drift chokepoints cover priority literals like status literals
      status: met
      tdd_phase: done
---

# M-0261 — Add the priority field, its validation, and drift chokepoints

## Goal

Define the `priority` frontmatter field on gap and decision, validate it on both axes (value-in-set and kind-scope), and extend the two literal-drift chokepoints so priority literals are protected like status literals. The foundation the write, read, and render surfaces all build on.

## Context

E-0066 adds `priority` to the two kinds where "which one do I work next" is an open question the kernel can't answer. After this milestone the field is defined and guaranteed but nothing sets or reads it yet — the writer and reader surfaces are separate milestones. The design mirrors the `area` feature: the field lives on the shared `Entity` struct and per-kind legality is enforced by check rules, not the type system.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

### AC-1 — priority is an optional gap/decision field validated against its closed set

### AC-2 — priority on other kinds raises the priority-not-applicable finding

### AC-3 — drift chokepoints cover priority literals like status literals

## Constraints

- The closed set (`urgent | high | medium | low`) is hardcoded in Go alongside kinds and statuses — no `aiwf.yaml` knob, because the set is genuinely closed (unlike `area`'s operator-declared members).
- `priority` sits on the shared `Entity` struct; per-kind legality is a `CarriesOwnPriority`-style predicate consulted by check rules, not a per-kind struct or a decode-time gate.
- Value validation is a straightforward shape check (blocking error, like `status-valid`) — no finding rule keys off a *specific* priority value, only membership in the closed set. Scope validation (kind-legality) is likewise mechanical, not prose.

## Design notes

- The scope rule (`priority-not-applicable`) is net-new check logic: the `area` precedent only ever gates *requiredness*, never *presence*, so nothing today rejects an out-of-scope field being present. Structure it off `internal/check/area_unknown.go` and pair it with a firing fixture (required by `firing_fixture_presence.go`).
- Chokepoint extensions: `enum_literal_adoption.go` harvests only `Status*`-prefixed constants today (an explicit "deliberate future-gap" note in-file) — widen to `Priority*`; `closed_set_status_constants.go` matches `Status:` / `.Status ==` / `TDDPhase:` contexts — add `Priority:` / `.Priority ==`.
- Whether the scope rule fires at warning or error severity is carried from the epic as an open question; default lean is warning, consistent with `area_unknown`.

## Surfaces touched

- `internal/entity/` — the `Priority` field, the gap/decision `OptionalFields` entries, the `CarriesOwnPriority` and closed-set-value predicates.
- `internal/check/` — the `priority-not-applicable` rule and its firing fixture.
- `internal/policies/` — `enum_literal_adoption.go`, `closed_set_status_constants.go`.

## Out of scope

- Any verb that writes the field, and any surface that reads it — those are the write-surface, read-surface, and render milestones under this epic.
- Sort ordering by priority — deferred to G-0420.

## Dependencies

- None — this is the foundation milestone; the other three depend on it.

## References

- G-0078 — the ratified design decisions this milestone executes.
- The `area` feature — `internal/check/area_unknown.go` / `area_required.go` and the `aiwf-area` skill — the design precedent.

---

## Work log

### AC-1 — priority field and closed-set validation

Field, constants, and the `priority-valid` check rule land · commit 34b13baf · tests 6/6 new (2 entity, 1 parse, 1 check, plus 2 discoverability fixes surfaced by the pre-commit hook: a missing `hintTable` entry and a missing `aiwf-check` skill row).

Deviation from the Design notes' "structure it off `area_unknown.go`" sketch: `area`'s presence-vs-scope handling works by the tree loader silently blanking an out-of-scope kind's value at load (`tree.go`, milestone/area) — but AC-2's `priority-not-applicable` finding needs the stored value intact to report it, so `priority` is deliberately *not* blanked the way `area` is. `CarriesOwnPriority` exists for the check rule to consult directly, not for a loader-side blank.

### AC-2 — priority-not-applicable finding

The scope-violation check lands, warning severity per the Design notes' lean · commit e4f4996b · tests 1/1 new, 3/3 mutants killed (empty-guard, scope-guard, severity).

Placed the skill row under "Findings (warnings)", not "Findings (errors)" — the SKILL.md table splits by severity across two separate tables (2-column: Code, Meaning-with-fix-folded-in) rather than annotating severity inline; my first pass got this wrong and had to move the row.

### AC-3 — widen the literal-drift chokepoints

`enum_literal_adoption`'s harvest widened to `Priority*` alongside `Status*`; `closed_set_status_constants` gained `Priority:` / `.Priority ==` / `.Priority !=` patterns and the four priority literal values · commit 052f8fa3 · tests 5/5 new, 2/2 mutants killed. Confirmed no existing production code in the repo already matched either new pattern before adding them (would have self-fired against the live-tree check otherwise).

The branch-coverage audit on the widened `HasPrefix` guard surfaced a real pre-existing gap: the live-tree test asserted presence of expected values but never absence of excluded ones (e.g. `TDDPhaseRed`), so the guard's skip-path ran during tests but was never actually pinned by an assertion. Added an explicit exclusion check rather than leaving it implicit.

### Independent pre-wrap review

A fresh-context reviewer (no authorship attachment) independently re-verified the milestone's eight load-bearing claims by reading and running the code, not the self-authored Work log entries above · commit 7bc01f2d fixes the one blocking finding plus three non-blocking observations.

**Blocking, fixed**: `internal/check/hint.go`'s `priority-valid` hint told operators to run `aiwf set-priority` — a verb explicitly out of scope for this milestone and absent from the codebase — contradicting the correct `aiwf-check` skill row for the same finding code. Both surfaces were self-authored and inconsistent with each other; corrected to a hand-edit remediation.

**Non-blocking, fixed**: `TestPriorityValid` never asserted `Severity`, leaving a real surviving mutant (error↔warning); `priority_not_applicable`'s doc comment overclaimed mirroring `area-unknown`'s archive-scoping when it doesn't (it follows `status-valid`'s non-archive-scoped precedent instead — behavior was already correct, only the doc claim was imprecise); the new `.Priority !=` regex had no firing test, asymmetric with the `==`/`Priority:` patterns which did.

**Confirmed correct after adversarial attempt to disprove**: the central design claim — that `priority`, unlike `area`, is never blanked at load, which is what makes AC-2's finding fireable at all. The reviewer grepped the full tree for any blanking logic and found none.

## Decisions made during implementation

- None — all decisions are pre-locked above.

## Validation

- `go build ./cmd/aiwf`: clean.
- `go vet ./...`: clean.
- `go test -race -parallel 8 ./...` (every package, full module): all green.
- `make lint` (full `golangci-lint` set, CI-parity): 0 issues.
- `make coverage-gate` (diff-scoped branch-coverage + firing-fixture meta-gate, run against `merge-base origin/main HEAD`): clean.
- `aiwf check`: 0 error-severity findings (1 pre-existing, unrelated `provenance-untrailered-scope-undefined` warning — no upstream configured yet).
- Manual branch-coverage audit performed per AC (see each AC's Work log entry); `wf-vacuity` mutation probes run per AC, 9/9 introduced mutants killed across all three ACs combined.
- `wf-doc-lint` (scoped to the milestone's change-set): 0 findings.

## Deferrals

- (none)

## Reviewer notes

- The epic's own open question — whether an existing contract pins the entity JSON payload shape, requiring a bump when `priority` reaches the envelope — is correctly out of scope here (M-0261 adds the struct field; nothing in this milestone exposes it through the JSON envelope). Resolve at the start of the read-surface milestone.
- `internal/policies/closed_set_status_constants.go`'s widened literal-value map adds the common words `"high"`/`"medium"`/`"low"` to a substring-matching heuristic that already accepted false-positive risk by design (documented in-file). No latent false positive exists today (the live-tree self-check is clean), but a future `case "high":` in unrelated code could trip it — same accepted trade-off as the pre-existing `"open"`/`"active"`/`"done"` entries, not a new risk class introduced by this milestone.
