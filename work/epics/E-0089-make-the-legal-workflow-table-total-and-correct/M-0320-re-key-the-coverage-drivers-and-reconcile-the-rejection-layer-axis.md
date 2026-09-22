---
id: M-0320
title: Re-key the coverage drivers and reconcile the rejection-layer axis
status: in_progress
parent: E-0089
depends_on:
    - M-0318
tdd: required
acs:
    - id: AC-1
      title: Both coverage drivers key on the target, with coverage measured before and after
      status: cancelled
    - id: AC-2
      title: Each cell's rejection layer matches where the kernel actually refuses
      status: met
      tdd_phase: done
---
## Goal

Align each cell's rejection layer with the kernel's refusal and preserve the
requests exercised by the coverage drivers.

## Context

M-0318 makes both drivers consume declared targets as a prerequisite to testing
the split rows. Its coverage measurements provide this milestone's baseline.
The driver migration is covered by M-0318's completed criteria. This milestone
reconciles the rejection-layer axis and compares exercised requests against the
recorded baseline; it must not lose coverage when a cell changes drivers.

The gap `open → addressed` request without a resolver and the AC `open → met`
request under `tdd: required` without phase `done` are refused before writing.
Their declared rejection layer must reflect that ordinary request. A check rule
can independently protect persisted state introduced through another path; that
backstop does not make the ordinary request a check-time rejection.

## Acceptance criteria

### AC-1 — Both coverage drivers key on the target, with coverage measured before and after

The driver migration and its test-first record belong to M-0318/AC-2 and
M-0318/AC-3. This criterion duplicates that completed implementation; the human approved its
cancellation during the M-0320 startup review on 2026-09-22. The
before-and-after coverage comparison remains required under AC-2 and the
no-silent-coverage-loss constraint below.

### AC-2 — Each cell's rejection layer matches where the kernel actually refuses

For every illegal cell, the declared rejection layer agrees with where the kernel
refuses — a cell declaring check-time rejection at a coordinate the verb refuses
before writing fails. This includes gap resolution without a resolver and marking
an AC met under required TDD without phase done.

Measure per-request driver coverage before and after the layer correction using
the same command, record the results and environment, and account for every
request that moves to a different driver. Preserve the forced gap-resolution
backstop check as well as unforced refusal coverage.

## Constraints

- **No silent coverage loss.** A cell that stops being exercised is a failure of
  this milestone even if the suite is green; the before-and-after measurement is the
  evidence.
- **Do not weaken the kernel to match the table.** Where the kernel refuses earlier
  than the table declares, the table is corrected, not the guard.

## Design notes

`RejectionLayer` describes where the ordinary request represented by a cell is
refused. A check rule that also catches invalid state introduced through another
path does not change that request's rejection layer. Preserve those backstop
checks independently; no combined layer value is needed.

## Out of scope

- Adding coverage for coordinates that had none — that is the totality milestone.
- TDD phase convergence (G-0458) — implemented and tested by M-0318.

## Dependencies

- M-0318 — the key the drivers are re-keyed onto.

## References

- D-0077 — the ruling behind the key change
- G-0166 — data-field mutation coverage outside this milestone
- `internal/policies/m0124_positive_driver_test.go`, `m0125_negative_driver_test.go`

## Deferrals

- G-0166 — modeling `milestone tdd` data-field mutations remains outside the
  status-transition table. The gap remains open because representing changes to a
  policy field alongside AC sub-state needs a separate modeling decision.

## Coverage baseline

Measured on 2026-09-22 on the E-0089 epic branch after M-0319, in the Linux amd64
devcontainer with Go 1.25.11:

```bash
go test -json -count=1 -parallel 8 \
  -run 'TestM0124_PositiveDriver_LegalCells|TestM0125_AC2_NegativeDriver_VerbTimeRejection|TestM0125_AC3_NegativeDriver_CheckTimeRejection|TestM0319_AC1_ExcludedKindsRefuseWithoutSideEffects|TestM0318_AC3_CheckTimeCellsAlsoRefuseUnforcedVerb' \
  ./internal/policies
```

Expected: all selected tests pass and the transition requests measured in
M-0318 remain exercised. Observed: exit 0. Counting passing JSON subtest events
by parent test yields 40 positive cases, 60 verb-time negative cases, 2 cases in
the check-time driver, 4 dedicated wrong-kind authorization cases, and 2
supplemental unforced-refusal cases. The four authorization cases moved out of
the transition driver in M-0319; their dedicated binary tests retain coverage.

The check-time driver's passing cases do not both demonstrate post-write
rejection: the gap case uses force, and the AC case asserts verb-time refusal.
The layer correction must preserve these actual requests, not merely their count.

## Release note

No user-facing behavior changed. The workflow specification and its coverage
checks agree with the kernel's existing refusal behavior.

## Validation

Measured on 2026-09-22 in the Linux amd64 devcontainer with Go 1.25.11,
on the M-0320 branch. Implementation commit: `51dc6263e`.

- `go test -count=1 -parallel 8 -run 'TestM012[45]|TestM0318|TestM0320' ./internal/policies`:
  expected all selected tests to pass; observed exit 0.
- `make check-fast`: expected vet, lint and the full non-race suite to pass;
  observed exit 0, lint reporting `0 issues`, and every test package passing.
- `go build -o bin/aiwf-diag ./cmd/aiwf`: expected a successful build; observed exit 0.
- `bin/aiwf-diag check --since e11dc5343`: expected no new findings; observed
  exit 0, no errors and the 17 baseline warnings.
- `git diff --check`: expected no whitespace errors; observed exit 0.

The exact Coverage baseline command was repeated after implementation, with
exit 0 expected and observed. Comparing passing JSON subtest names showed the
40 positive and 4 authorization request sets unchanged. The 60 ordinary negative
requests plus 2 supplemental unforced requests exactly matched the unified
negative driver's 62 requests, with none missing or extra. Both former check-time
cell names occur in that driver. Counts alone do not establish preservation of
the forced request: `TestM0320_AC2_ForcedGapResolutionRetainsCheckBackstop`, selected
by the focused command above, separately passed a successful forced resolution
followed by a resolver finding bound to the affected entity.

Production changes are static layer assignments and their definition; no runtime
branch changed. The negative driver checks the refusal reason and unchanged HEAD
and project files. Coverage count-map equality checks missing, extra and duplicate
enumeration; unique names and the skip ban remain independent checks.

Vacuity measurements used Go overlays, without modifying the shared checkout:
`python3 /tmp/aiwf-m0320-probes.py` and
`python3 /tmp/aiwf-m0320-extra-probes.py`. Expected each mutant to fail its intended
assertion, without compilation errors; all nine did. The probes changed the layer,
disabled either refusal guard, removed the resolver backstop, substituted an
unrelated refusal, wrote a file before refusal, omitted a cell, added an extra
legal cell, and attached the finding to the wrong entity. The extra-cell probe
checks the protection retained when the separate no-extra test was removed.
Scripts and logs are session artifacts under `/tmp`; the standing assertions live
in the committed tests.

`gremlins unleash ./internal/workflows/spec/... --diff HEAD --workers 1 --timeout-coefficient 15`
reported `No results to report` before the implementation commit: static enum
assignments produced no supported mutants. This is not mutation-coverage evidence.
Full race/CI was not run at this local milestone boundary; it gates epic
integration and push under the repository's validation cadence.
