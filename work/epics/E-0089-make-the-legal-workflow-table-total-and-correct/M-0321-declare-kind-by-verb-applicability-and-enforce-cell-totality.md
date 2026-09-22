---
id: M-0321
title: Declare kind-by-verb applicability and enforce cell totality
status: in_progress
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
      status: open
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

The table still omits cancellation coordinates whose behavior the kernel already
implements. This milestone represents those requests and tests them through the
existing drivers so the totality policy can hold without exemptions.

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
for each coordinate. The script is a session artifact; the totality policy will
make this enumeration repeatable from committed source.

Expected: locate every empty coordinate before deciding the implementation scope.
Observed: 66 coordinates, 21 empty, all under `cancel`. The 5 TDD-phase coordinates
are outside cancellation's domain. The remaining 16 are 12 terminal entity
statuses, 2 terminal AC statuses, accepted decision, and deprecated contract.

`go test -count=1 -parallel 8 -run 'TestCancel|TestAC.*Cancel' ./internal/verb`
and `go test -count=1 -run '^TestCancelTarget$' ./internal/entity` were expected to
pass and did (exit 0). These establish the existing cancellation baseline and
state-aware target selection; the new per-cell tests must still exercise every
added coordinate. `go build -o bin/aiwf-diag ./cmd/aiwf` also exited 0 as expected.
The full non-race suite and lint passed for M-0320; no Go or build input has changed
since that run.
