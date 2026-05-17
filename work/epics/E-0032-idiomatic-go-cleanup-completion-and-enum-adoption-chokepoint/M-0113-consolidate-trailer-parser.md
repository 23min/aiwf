---
id: M-0113
title: Consolidate trailer parser
status: in_progress
parent: E-0032
tdd: required
acs:
    - id: AC-1
      title: Single canonical exported gitops.ParseTrailers
      status: met
      tdd_phase: done
    - id: AC-2
      title: All callers consume gitops.ParseTrailers; cliutil duplicate deleted
      status: met
      tdd_phase: done
---
## Goal

Eliminate the duplicate trailer-parser implementation. Export [`internal/gitops/gitops.go:312`](../../../internal/gitops/gitops.go) `parseTrailers` as `gitops.ParseTrailers`, switch the three callers, delete [`internal/cli/cliutil/scopes.go:258`](../../../internal/cli/cliutil/scopes.go) `ParseTrailerLines`.

## Context

G-0107 step 2 residue. The cliutil variant was added during the verb-support extraction (commit `1d391c5`) but the original `parseTrailers` in gitops was never removed, leaving two byte-identical implementations of `git log %(trailers:only=true,unfold=true)` line parsing.

## Approach

Rename `parseTrailers` → `ParseTrailers` in gitops; the existing tests at [`internal/gitops/gitops_test.go:59`](../../../internal/gitops/gitops_test.go) and [`internal/gitops/trailers_test.go:242,264`](../../../internal/gitops/trailers_test.go) update their lowercase-name calls. Delete `ParseTrailerLines` in cliutil. Switch the three call sites ([`cmd/aiwf/provenance_check.go:182,228`](../../../cmd/aiwf/provenance_check.go) and [`internal/cli/cliutil/scopes.go:166`](../../../internal/cli/cliutil/scopes.go)) to `gitops.ParseTrailers`. Single commit; no behavior change; tests stay green.

## Acceptance criteria

### AC-1 — Single canonical exported gitops.ParseTrailers

**Observable claim.** `gitops.ParseTrailers` is the package's exported trailer-line parser; calls from external packages (`gitops_test`, downstream `internal/*`, `cmd/aiwf/*`) resolve to it. The pre-M-0113 unexported `gitops.parseTrailers` is gone.

**Test seam.** [`internal/gitops/export_test.go`](../../../internal/gitops/export_test.go) (added in commit `781794a` as the AC's red marker) is in `package gitops_test` and calls `gitops.ParseTrailers` directly. Before AC-1 it failed to compile (`undefined: gitops.ParseTrailers`); after the rename in `c056944` it compiles and passes. The existing in-package `TestParseTrailers`, `TestParseTrailers_ToleratesAbsentI25Keys`, `TestParseTrailers_ToleratesUnknownFutureKeys`, and `FuzzParseTrailers` flipped their lowercase callers to the exported name in the same commit and continue to pass.

### AC-2 — All callers consume gitops.ParseTrailers; cliutil duplicate deleted

**Observable claim.** No package outside `internal/gitops/` declares a top-level function whose name is in `{ParseTrailers, parseTrailers, ParseTrailerLines, parseTrailerLines}`. All three pre-M-0113 caller sites (`cmd/aiwf/provenance_check.go:182,228` and `internal/cli/cliutil/scopes.go:166`) route through `gitops.ParseTrailers`.

**Test seam.** [`internal/policies/trailer_parser_uniqueness.go`](../../../internal/policies/trailer_parser_uniqueness.go) (added in commit `6d07f19`) walks every production `.go` file with `WalkGoFiles(root, true)`, skips `internal/gitops/`, and flags any matching `FuncDecl`. Verified red against the pre-fix state (the policy flagged `internal/cli/cliutil/scopes.go:258`); green after the deletion + caller switch. Caller-side correctness is covered by the existing provenance and scope tests, which continue to pass with the byte-identical `gitops.ParseTrailers` substituted in.

## Surfaces touched

- `internal/gitops/gitops.go` — export rename
- `internal/gitops/gitops_test.go`, `internal/gitops/trailers_test.go` — caller rename
- `internal/cli/cliutil/scopes.go` — deletion of `ParseTrailerLines`; one call site update
- `cmd/aiwf/provenance_check.go` — two call site updates

## Out of scope

- Other gitops exports
- Broader `internal/cli/cliutil/` reorganization (M-6's scope)

## Dependencies

- None.

---

## Work log

### AC-1 — gitops.ParseTrailers exported

Renamed unexported `parseTrailers` to exported `ParseTrailers` in `internal/gitops/gitops.go`; added the canonical doc comment; flipped the internal self-caller at `HeadTrailers` and the four in-package test callers (`gitops_test.go:59`, `trailers_test.go:242,264`, `trailers_fuzz_test.go:27` + comments). · commits `781794a` (red test), `c056944` (rename + caller updates) · gitops tests 45/0/0 pass

### AC-2 — cliutil duplicate deleted; uniqueness pinned

Switched three caller sites (`cmd/aiwf/provenance_check.go:182,228` and `internal/cli/cliutil/scopes.go:166`) to `gitops.ParseTrailers`; deleted the byte-identical `cliutil.ParseTrailerLines` (`scopes.go:254-275`); added `internal/policies/trailer_parser_uniqueness.go` as the drift-prevention guard. · commit `6d07f19` · policies tests 120/0/0 pass

## Decisions made during implementation

- None — all decisions are pre-locked in the spec above.

## Validation

(Pasted at wrap. `go test -race -parallel 8 ./...` green on the changed packages; full module green when run at `-parallel 4` or with isolated package targets — the macOS git-subprocess flake documented in CLAUDE.md `## Test discipline` is reproducible at default parallelism but does not reflect a logical regression in this milestone's diff. `golangci-lint run ./...` clean. `aiwf check` clean of error-severity findings on M-0113.)

## Deferrals

- None.

## Reviewer notes

- The "red commit" for AC-2 is not a separately landed commit because `pre-commit.local` runs `go test ./internal/policies/...` on every commit; a red policy test would be rejected at the hook. Red was verified locally before the green commit landed (the policy fired on `internal/cli/cliutil/scopes.go:258` against the pre-fix state). AC-1's red marker (`781794a`) was a standalone external-package test in `internal/gitops/` that the policies hook does not gate.
- The policy's violation-append branch is not directly exercised by a test fixture in green state — the same convention as every other policy under `internal/policies/`. A synthetic-fixture test pass across all policies is a future hygiene improvement, not a regression introduced here.
- Code commits use Conventional Commits subjects with the AC reference suffix (`(M-0113/AC-N)`); aiwf-verb commits (promote / add / authorize) ride the verb route's trailer set.
