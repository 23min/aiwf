---
id: M-0254
title: Contract subsystem coverage backfill
status: in_progress
parent: E-0064
depends_on:
    - M-0252
tdd: required
acs:
    - id: AC-1
      title: Every contract-subsystem branch tested or ignored
      status: met
      tdd_phase: done
    - id: AC-2
      title: Scoped coverage-gate reports zero findings
      status: met
      tdd_phase: done
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the contract
subsystem — `contract/recipes`, `contract/verify`, `contract/bind`,
`contract/unbind` — using the shared failure fixtures M-0252 builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share. `contract/recipes.go` alone carries the single largest
concentration of flagged sites of any file in the epic's scope, so this
milestone is sized to that file plus its three siblings in the same
package.

## Acceptance criteria

### AC-1 — Every contract-subsystem branch tested or ignored

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename, `2ac84846^`) within `internal/cli/contract/
{recipes,verify,bind,unbind}.go` carries either a passing test (reusing
M-0252's fixtures where the failure mode matches) or a
`//coverage:ignore <reason>`.

### AC-2 — Scoped coverage-gate reports zero findings

`make coverage-gate`'s underlying policy test, run with `AIWF_COVERAGE_BASE`
set to the pre-M-0238 commit, reports zero findings for the files listed
in AC-1.

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Entity-lifecycle, diagnostic, bulk-input, and non-CLI infra files —
  M-0252, M-0253, M-0255, and M-0256's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.

## Work log

### AC-1 — Every contract-subsystem branch tested or ignored

19 real tests (reusing M-0252's BrokenGitIdentity fixture and the
malformed-contracts-block trigger proven at internal/cli/add) plus 12
`//coverage:ignore` annotations, closing all 31 branch-coverage-audit
findings across bind.go, unbind.go, verify.go, and recipes.go · commit
67600f8a · tests 19/19

### AC-2 — Scoped coverage-gate reports zero findings

Validation-only, no new commit. Re-ran the scoped
`TestPolicy_BranchCoverageAudit` policy test with
`AIWF_COVERAGE_BASE=2ac84846^` against a full-repo coverage profile
generated after AC-1's tests landed: zero `internal/cli/contract/`
findings. The remaining findings the run reports all belong to
files out of this milestone's scope (M-0255/M-0256's job).

## Decisions made during implementation

- (none)

## Validation

- `go build ./...` — clean.
- `go vet ./internal/cli/contract/...` — clean.
- `gofumpt -l internal/cli/contract/` — clean (no output).
- `golangci-lint run ./internal/cli/contract/...` (`make lint`'s full CI-parity set) — 0 issues.
- `make test-race` (full repo, `-race -parallel 8`) — all packages pass, 0 failures.
- Scoped `TestPolicy_BranchCoverageAudit` with `AIWF_COVERAGE_BASE=2ac84846^` against a
  full-repo coverage profile — zero `internal/cli/contract/` findings (the run's remaining
  findings are all in files out of this milestone's scope).
- Independent fresh-context code-quality review (`wf-review-code` lens) — **approve**,
  after re-deriving the flagged-line list independently, verifying every
  `//coverage:ignore` rationale by reading the called function's real error paths, and
  hand-tracing each new test's assertion to confirm it fails for the reason its name
  claims (not a coincidental early exit). No design-quality (`wf-rethink`) pass — this
  milestone introduces no new module boundary, abstraction, or data model, only test
  files and one-line ignore-comment additions to existing guards.
- `wf-doc-lint` (scoped to the milestone's change-set) — clean; the change-set is
  entirely Go source/test files plus this spec, none intersecting the doc-lint scope.

## Deferrals

- (none) — the review's one substantive finding (see Reviewer notes) is already
  tracked by the open **G-0412**, not a new deferral.

## Reviewer notes

- The independent review flagged that the `//coverage:ignore cliutil.ResolveRoot only
  fails on missing aiwf.yaml + non-existent --root path` wording (copied verbatim from
  established repo-wide precedent onto 6 lines in this milestone's 4 files) is
  imprecise: with an explicit `--root`, `ResolveRoot` returns `filepath.Abs(explicit)`
  without statting the path, so a non-existent `--root` does not error; the only real
  failure mode is `os.Getwd()` failing. The ignore itself is still correct (the branch
  is genuinely unit-uncoverable), only the stated rationale is off. This is the same
  issue **G-0412** already tracks repo-wide (deliberately deferred as a future sweep
  across every file carrying this copied wording, not a per-milestone fix) — these 6
  lines are additional instances of that same already-open gap, not a new one.
- The review's other observation (no serial skip-list comment documenting the package's
  `t.Setenv`-using tests) was fixed inline: `setup_test.go` now documents all 8 serial
  tests in the package (4 pre-existing, 4 new to this milestone).
- The `tree.Load` `//coverage:ignore` lines (bind.go, verify.go) were specifically
  re-verified against the M-0253 lesson (a sibling milestone found a `LoadTreeWithTrunk`
  ignore comment was wrong because a malformed aiwf.yaml deterministically triggers it
  via config.Load). Confirmed the distinction holds here: bind.go/verify.go call bare
  `tree.Load`, which walks only `work/` and `docs/adr` and never touches aiwf.yaml — the
  M-0253 trap does not apply.
