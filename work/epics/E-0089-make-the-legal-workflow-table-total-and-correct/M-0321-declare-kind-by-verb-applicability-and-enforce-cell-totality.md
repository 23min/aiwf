---
id: M-0321
title: Declare kind-by-verb applicability and enforce cell totality
status: done
parent: E-0089
depends_on:
    - M-0318
    - M-0319
tdd: required
acs:
    - id: AC-1
      title: The kind-by-verb applicability table is total over every pair, enforced
      status: met
      tdd_phase: done
    - id: AC-2
      title: An applicable coordinate with no cell fails a policy
      status: met
      tdd_phase: done
---
## Goal

Declare once, per kind and verb, whether the verb applies at all — and make a
missing cell at an applicable coordinate a policy failure rather than a silence.

## Context

Applicability distinguishes a verb outside a kind's domain from a request the
kernel accepts, refuses, or converges. D-0077 makes that distinction a kind-by-verb
fact, declared once, and defines cell totality over applicable pairs.

`cancel` does not act on the TDD-phase sub-FSM. AC cancellation acts on the AC's
status, not its TDD phase; the phase ladder has no cancellation transition. The
other kinds have meaningful cancellation requests even where a particular status
makes the request a refusal or a NoOp. Those statuses remain inside the table's
coverage obligation.

Cancellation coordinates belong in the table even when their outcomes follow
existing kernel behavior. The outcome drivers must exercise those requests so
the totality policy holds without exemptions.

## Acceptance criteria

### AC-1 — The kind-by-verb applicability table is total over every pair, enforced

Every kind-by-verb pair has an applicability entry, and a pair with none fails a
policy. A non-applicable entry carries the reason it does not apply. Adding a kind
or a legality verb without an entry reddens the suite.

### AC-2 — An applicable coordinate with no cell fails a policy

For every applicable kind-by-verb pair, every `FromState` of that kind carries at
least one cell. Removing the last cell at a coordinate fails the policy with a
message naming that coordinate. The count of coordinates the policy covers is
recorded with the command that produced it.

Fill the missing cancellation coordinates from measured kernel behavior: terminal
entities and terminal ACs converge without mutation; cancellation of an accepted
decision refuses; cancellation of a deprecated contract retires it. Exercise each
new row through the matching outcome driver. Do not change kernel behavior to
make the table total.

## Constraints

- **Both tables total, or neither counts.** An applicability table with a hole moves
  the silence rather than removing it, which is why AC-1 polices it in its own right.
- **A reason, not a flag.** A non-applicable entry says why; "false" alone
  reintroduces the ambiguity in a narrower field.
- **No cell written to satisfy the policy.** A coordinate that turns out to need a
  ruling is ruled, not defaulted.

## Design notes

The two tables answer different questions and it is worth keeping them visibly
distinct: applicability asks *does this verb mean anything for this kind*, the cell
table asks *from this state, toward this target, what happens*. D-0077 owns the
kind-by-verb applicability decision. A refusal at one status does not make the
whole kind-by-verb pair inapplicable, and a NoOp still needs a cell.

## Out of scope

- Changing cancellation semantics or inventing outcomes to satisfy the policy.
- Applicability for verbs outside the cell table's set.

## Dependencies

- M-0318 — totality is defined against the target-bearing key.
- M-0319 — the verb set is `promote` and `cancel` only once `authorize` has moved.

## References

- D-0077 — the ruling this implements
- `internal/entity/transition.go` — state-aware cancellation targets and sub-FSMs
- `internal/workflows/spec/rules.go`

## Preflight measurement

Measured on 2026-09-22 in the Linux amd64 devcontainer with Go 1.25.11,
at epic commit `486820a00`. `go run bin/m0321-preflight.go` enumerated
`entity.AllKinds()` plus the AC and TDD-phase sub-FSMs, their allowed states
(including the empty phase), and `promote`/`cancel`, querying `spec.LookupRules`
for each coordinate. The script is a session artifact, now at
`/tmp/aiwf-m0321-preflight.go`. The committed totality policy provides the
repeatable applicable-coordinate enumeration recorded under Validation.

Expected: locate every empty coordinate before deciding the implementation scope.
Observed: 66 coordinates, 21 empty, all under `cancel`. The 5 TDD-phase coordinates
are outside cancellation's domain. The remaining 16 are 12 terminal entity
statuses, 2 terminal AC statuses, accepted decision, and deprecated contract.

`go test -count=1 -parallel 8 -run 'TestCancel|TestAC.*Cancel' ./internal/verb`
and `go test -count=1 -run '^TestCancelTarget$' ./internal/entity` were expected to
pass and did (exit 0). These establish the existing cancellation baseline and
state-aware target selection; the per-cell driver measurements below establish
coverage of the added coordinates. `go build -o bin/aiwf-diag ./cmd/aiwf` also exited 0 as expected.

## Release note

No user-facing behavior changed.

## Validation

Measured on 2026-09-22 in the Linux amd64 devcontainer with Go 1.25.11,
on the milestone branch; implementation commits `b651a72a6` and `e2c228d3d`.

- `go test -count=1 -parallel 8 -run '^TestM0321_AC1' ./internal/policies`:
  expected the complete applicability table and validation fixtures to pass;
  observed PASS. Fixtures cover missing, duplicate and unknown pairs, missing
  reasons, and domain growth; the domain assertion rejects incorrect applicability.
- `go test -count=1 -v -run '^TestM0321_AC2_ApplicableCoordinatesHaveRules$' ./internal/policies`:
  expected no missing applicable coordinate and a reported count; observed PASS,
  `checked 61 applicable coordinates`. Removing the last row is tested separately
  for entity, AC-status and empty-phase coordinates; removing a companion row
  preserves coverage.
- `go test -json -count=1 -parallel 8 -run 'TestM0124_PositiveDriver_LegalCells|TestM0125_AC2_NegativeDriver_VerbTimeRejection|TestM0318_AC3_NoOpCells' ./internal/policies`:
  expected all old requests to remain and every added row to execute. Comparing
  passing subtest-name sets before and after the AC-2 implementation found no
  removed request: positive cases 40 → 41, negative cases 62 → 63, NoOp cases
  18 → 32. Both runs exited 0; the after selection also included `TestM0321_AC2`.
  The added requests exercise deprecated-contract retirement, accepted-decision
  refusal and terminal cancellation convergence. Logs are session artifacts at
  `/tmp/aiwf-m0321-ac2-before.json` and `/tmp/aiwf-m0321-ac2-after.json`.
- `make test`: expected the full non-race suite to pass; observed exit 0, all
  packages passed. After helper lint fixes,
  `go test -count=1 -parallel 8 ./internal/policies` passed in 36.890s.
- `make vet`: expected ordinary, stress and testpins checks to pass; observed exit 0.
- `GOLANGCI_LINT_CACHE="$(git rev-parse --absolute-git-dir)/golangci-lint-cache" golangci-lint run --allow-serial-runners`:
  expected the full configured lint set to pass; observed exit 0, `0 issues.`
  Serial runner mode accommodates concurrent worktree linting without disabling
  checks. The aggregate `make check-fast` did not pass: its attempts encountered
  a scratch Go file and then the shared linter lock. Its test, vet and lint
  components passed separately after the scratch file moved outside the checkout.
- `go build -o bin/aiwf-diag ./cmd/aiwf`: expected a successful build; observed exit 0.
- `bin/aiwf-diag check --since 486820a00`: expected no new findings; observed
  exit 0, 0 errors and 17 baseline warnings.
- `git diff --check`: expected no whitespace errors; observed exit 0.

The production additions are data and documentation, with no new runtime branch.
The policy fixtures exercise applicability validation and all three state domains;
the NoOp driver exercises both promote and cancel messages.

`python3 /tmp/aiwf-m0321-ac1-probes.py` and
`python3 /tmp/aiwf-m0321-ac2-probes.py` (including selected reruns) were expected to
make deliberately broken implementations fail an intended assertion. All eight
probes per AC were caught without compilation failures. AC-1 probes cover missing,
duplicate and unknown pairs, blank reasons, both incorrect applicability
polarities, and new kinds and verbs. AC-2 probes cover a missing row, reversed
applicability filtering, omitted empty-phase and AC states, an incorrect contract
target, disabled entity and AC terminal convergence, and an incorrect cancel
message. These scripts and their logs are session artifacts; the standing checks
are in `internal/policies/m0321_applicability_test.go`,
`internal/policies/m0321_totality_test.go` and the existing outcome drivers.

`gremlins unleash ./internal/workflows/spec/... --diff HEAD --workers 1 --timeout-coefficient 15`
was expected to identify supported mutations during each AC's uncommitted source
change. Both runs exited 0 with `No results to report`; static data produced no
supported mutants. This is not mutation-coverage evidence.

Full race/CI was not run at this local milestone boundary, per the repository's
validation cadence.

## Decisions made during implementation

None beyond D-0077's applicability ruling and the existing cancellation behavior
covered by ADR-0036 and the kernel's transition rules.

## Deferrals

None.

## Reviewer notes

Independent full-change code review: approve, no blocking findings or untracked
defects. Independent design review of the applicability model: keep. The separate,
explicit declaration makes a missing transition distinguishable from an excluded
verb domain without introducing a registry or mutable shared state.

Keep the explicit applicability-validation guards. Their distinct diagnostics and
separate completeness walk are load-bearing; a switch-based equivalent saves one
formatted line without a meaningful simplification. Existing drift arms remain
under the epic's constraint, retaining their other schema and reverse-drift duties.

The applicability and coordinate-totality policies own the new structural
obligations until their table/domain model is replaced. The current exclusion
assertion changes only with a deliberate verb-domain change; the existing outcome
drivers own each declared request until that request's semantics change or it is
removed.

Scoped doc-lint: clean. Referenced source paths and links resolve; no removed
feature, new orphan document, documentation marker, stale invocation or heading
structure finding was identified in the milestone's documentation scope.
