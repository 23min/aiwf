---
id: M-0252
title: Shared CLI-verb failure fixtures and non-CLI infra coverage backfill
status: in_progress
parent: E-0064
tdd: required
acs:
    - id: AC-1
      title: Reusable test fixtures exist for each shared CLI-verb failure mode
      status: met
      tdd_phase: done
    - id: AC-2
      title: Non-CLI-infra flagged branches are tested or documented
      status: met
      tdd_phase: done
    - id: AC-3
      title: Coverage gate is clean for the non-CLI-infra group
      status: met
      tdd_phase: done
---

## Goal

Establish reusable test fixtures for the handful of failure modes shared
across CLI-verb error-handling guards, and use them to clear every branch
`branch-coverage-audit` currently flags in `internal/verb/*`,
`internal/gitops/refs.go`, `internal/stresstest/*`, `internal/check/*`,
`internal/cellcoverage/*`, and `internal/cli/cliutil/*` itself.

## Context

E-0064 closes G-0386's test-coverage debt: ~194 (188 currently live)
CLI-verb error-handling branches surfaced as untested when M-0238/AC-3's
mechanical print-call rename touched lines inside guards nothing had ever
exercised. This is the foundational milestone — it builds the shared
triggering fixtures (root-resolution failure, actor-resolution failure,
repo-lock contention, malformed/corrupt tree, bad `--format` flag) that
M-0253 through M-0256 (parallel, all depending on this milestone) will
reuse, and proves them against real call sites by applying them to the
smallest flagged group first.

## Acceptance criteria

### AC-1 — Reusable test fixtures exist for each shared CLI-verb failure mode

Reusable test fixtures exist (e.g. under `internal/cliutil/testutil` or a
sibling package) for each shared failure mode: root-resolution failure,
actor-resolution failure, repo-lock contention
(`cliutil.AcquireRepoLock`), malformed/corrupt tree (`tree.Load`), and a
bad `--format` flag. Each fixture is documented with the exact condition
it simulates, so M-0253 through M-0256 can reuse it without re-deriving
the trigger.

### AC-2 — Non-CLI-infra flagged branches are tested or documented

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename) within `internal/verb/*`, `internal/gitops/refs.go`,
`internal/stresstest/*`, `internal/check/*`, `internal/cellcoverage/*`,
and `internal/cli/cliutil/*` carries either a passing test built on the
AC-1 fixtures or a `//coverage:ignore <reason>` naming why the branch
isn't triggerable, mirroring `internal/cli/archive/archive.go:120`.

### AC-3 — Coverage gate is clean for the non-CLI-infra group

`make coverage-gate`, run with `AIWF_COVERAGE_BASE` set to the pre-M-0238
commit, reports zero findings for the files listed in AC-2.

## Constraints

- Per-site judgment only: a real test where the failure is genuinely
  triggerable, an honest `//coverage:ignore <reason>` otherwise — never a
  blanket suppression.
- Fixtures are built generic enough for M-0253–M-0256 to reuse as-is; a
  fixture that only works for one call site defeats the point of doing
  this milestone first.

## Out of scope

- Entity-lifecycle, contract, diagnostic, and bulk-input verb files —
  M-0253 through M-0256's job.
- Any change to error-handling behavior (message wording, exit codes)
  beyond what's needed to make a branch deterministically triggerable.

## Dependencies

- E-0064 epic spec (committed).
- No prior milestone — this is first.

## References

- **E-0064** — parent epic.
- **G-0386** — the gap this epic (and this milestone) closes.
- `internal/cli/archive/archive.go:120` — precedent `//coverage:ignore`
  rationale style.
- `internal/policies/branch_coverage_audit.go` — the diff-scoped
  coverage-gate policy that surfaces these findings.

## Work log

### AC-3 — Coverage gate is clean for the non-CLI-infra group

Two independent confirmations against the committed AC-2 state (not
scratch/staged): (1) `TestPolicy_BranchCoverageAudit`, run directly with
`AIWF_COVERAGE_BASE` set to the pre-M-0238 commit (`2ac84846^`),
scoped-grepped to the six paths — zero matching lines. (2) the standard
`make coverage-gate` (default base = `git merge-base origin/main HEAD`)
— all four policies (`BranchCoverageAudit`, `FiringFixturePresence`,
`FiringFixtureNoStaleAllowlist`, `SkillEditStructuralTestBackstop`)
pass clean. No new code in this AC — it's the closing verification
that AC-1 and AC-2's combined work actually satisfies the milestone's
stated success criterion.

### AC-2 — Non-CLI-infra flagged branches tested or ignored

Regenerating the flagged set matched the nine files anticipated at
kickoff, plus two additional lines outside them
(`internal/stresstest/head_drift.go:67`,
`internal/stresstest/promote_on_wrong_branch_detection.go:100`) —
handled with the same per-site judgment as everything else. Every
flagged line now carries a passing test or an honest
`//coverage:ignore`; the rerun `branch-coverage-audit` scoped to AC-2's
six paths reports zero findings.

Real tests added:
- `internal/gitops/refs_test.go` — `CurrentBranch`'s git-launch-failure
  branch (PATH cleared to a directory with no `git`).
- `internal/cli/cliutil/apply_test.go` (new) — all five `FinishVerb`
  branches plus its success/NoOp paths; the function had no test file
  at all before this.
- `internal/cli/cliutil/provenance_test.go` (new) —
  `DecorateAndFinish`'s gate-denial branch.
- `internal/verb/archive_test.go` — `planArchive`'s `tree.Load`
  fatal-error wrap (a pre-cancelled context) and its
  `computeArchiveMoves` error wrap (an unknown `--kind`), both driven
  through the full `Archive` entrypoint — distinct from the existing
  `TestComputeArchiveMoves_UnknownKindFilter`, which calls
  `computeArchiveMoves` directly and never exercised `planArchive`'s
  own propagation line.
- `internal/verb/promote_branch_guard_internal_test.go` (new) —
  `expectedActivationBranch`'s three fail-shut branches, driven
  directly against hand-built `*tree.Tree`/`*entity.Entity` fixtures
  (each needs a milestone shape `verb.Add`'s own validation refuses to
  create).
- `internal/cellcoverage/fixture_test.go` — `Must`'s `verb.Apply`
  failure branch (a zero-`Ops` plan, the same "nothing to commit"
  trigger `internal/verb/apply_test.go` pins directly on `verb.Apply`).
- `internal/cli/cliutil/testutil/fixtures_test.go` —
  `HoldRepoLock`'s and `WriteMalformedEntity`'s three
  `t.Fatalf`-guarded branches (AC-1's own flagged lines), proven via a
  throwaway-`*testing.T` technique (see Decisions below).

Honest ignores added:
- `internal/cli/cliutil/testutil/fixtures.go:51` —
  `BrokenGitIdentity`'s write to a fresh `t.TempDir()`.
- `internal/cli/cliutil/testutil/proc.go:305` —
  `SetupGitRepoWithUpstream`'s second `git init`; a PATH-absent
  trigger would fail the earlier bare-upstream `git init` call first,
  never reaching this one.
- `internal/stresstest/head_drift.go:67`,
  `internal/stresstest/promote_on_wrong_branch_detection.go:100` —
  both scenarios drive the same interloping-checkout shape as an
  epic-activation promote; G-0269's branch guard (added after both
  scenarios were written) now refuses that promote outright, so
  neither scenario's real-subprocess run ever reaches the
  "landed on the wrong branch" continuation these lines guard.

## Decisions made during implementation

- The milestone's own suggested `t.Run("inner", fn); assert the
  returned bool` technique for proving a `t.Fatalf`-guarded fixture
  branch fires does not work: `testing`'s `common.Fail` unconditionally
  propagates a subtest's failure to every `t.Run`-linked ancestor
  (confirmed against the stdlib source and empirically — the wrapping
  test's own package permanently reports `FAIL`). Used instead: a
  throwaway `*testing.T{}` — never obtained via `t.Run`, so it has no
  parent to propagate to — run in its own goroutine, since
  `t.Fatalf`'s `runtime.Goexit` unwinds only the calling goroutine.
  Applied at `internal/cellcoverage/fixture_test.go`'s
  `runAndCaptureFatal` and the identically-named helper in
  `internal/cli/cliutil/testutil/fixtures_test.go`.

## Validation

AC-2 (this session): `go build ./...`, `go vet ./...`, `gofumpt -l` on
every changed file (clean except one pre-existing, untouched-by-this-
change formatting drift in `internal/cli/cliutil/testutil/proc.go`),
`golangci-lint run` (0 issues) all clean. Full suite:
`go test -race -parallel 8 ./...` — 7099 passed, 0 failed, 6 skipped,
across every package. The AC-2-scoped `branch-coverage-audit` rerun
(six paths) reports zero findings.
