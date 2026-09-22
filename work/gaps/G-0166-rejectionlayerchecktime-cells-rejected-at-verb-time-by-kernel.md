---
id: G-0166
title: RejectionLayerCheckTime cells rejected at verb-time by kernel
status: open
priority: low
discovered_in: M-0125
---
## What's missing

`aiwf milestone tdd` refuses a change to `tdd: required` when a met AC lacks
`tdd_phase: done`, but `internal/workflows/spec/` has no representation of this
request. `internal/verb/milestone_tdd.go::MilestoneTDD` owns the guard.

The table's status-transition key cannot describe a change to the milestone's
TDD policy combined with AC sub-state. Runtime protection exists; the missing
piece is a specification representation that lets coverage derive this refusal
from the workflow model.

## Why it matters

The table-driven coverage suite cannot detect omission or misclassification of
this data-field mutation refusal. Dedicated integration tests protect the kernel
behavior independently of the table.

## Resolution shape

Decide how the workflow specification represents data-field mutations and their
preconditions, then include this refusal in coverage derived from that model.
Preserve the kernel guard and the dedicated behavior tests. Widening the
status-transition table is outside E-0089's scope.

## Where to fix

- `internal/workflows/spec/` — representation and enumeration of mutation requests.
- `internal/policies/` — coverage derived from that representation.
- `internal/cli/integration/milestone_tdd_verb_test.go` — existing tests assert
  refusal naming the offending AC, unchanged policy and commit count, no invented
  phase, and successful strengthening when a met AC has phase done.

## Validation

Measured on 2026-09-22 in the Linux amd64 devcontainer with Go 1.25.11,
on the M-0320 branch:

```bash
go test -count=1 -parallel 8 \
  -run 'TestMilestoneTDD_(RefusesRequiredWhenMetACPhaseless|AllowsRequiredWhenMetACPhaseDone)$' \
  ./internal/cli/integration
rg -n 'milestone.tdd|milestone-tdd' internal/workflows/spec --glob '*.go'
```

Expected: the refusal and permitted-change tests pass; the specification search
has no matching representation. Observed: test exit 0; search exit 1 with no
matches. The named guard and the table's status-transition fields explain the
modeling boundary; the search alone is not proof of semantic absence.

## Related

- M-0277/AC-4 — the policy-change refusal's runtime obligation.
- M-0320 — ordinary status-transition rejection layers and preserved backstop coverage.
