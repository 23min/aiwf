---
id: M-0114
title: Lift completion helpers to internal/cli/cliutil/completion.go
status: in_progress
parent: E-0032
tdd: required
acs:
    - id: AC-1
      title: Completion helpers and resolveRoot exported from internal/cli/cliutil/
      status: met
      tdd_phase: done
    - id: AC-2
      title: cmd/aiwf has no local definitions; all callers consume cliutil exports
      status: met
      tdd_phase: done
---
## Goal

Move the 6 completion helpers (`registerFormatCompletion`, `allKindNames`, `statusesForID`, `completeEntityIDs`, `completeEntityIDFlag`, `completeEntityIDArg`) from [`cmd/aiwf/main.go:53–145`](../../../cmd/aiwf/main.go) into a new file `internal/cli/cliutil/completion.go`. `main.go` drops below ~540 lines.

## Context

G-0107 step 2 residue. The helpers belong in cliutil (alongside `actor.go`, `flags.go`, `exit.go`) but were stranded in `main.go` because they reference local `resolveRoot`. This milestone resolves that dependency by lifting `resolveRoot` to cliutil as well — it's already a pure helper (root-dir resolution from `--root` flag or `aiwf.yaml` discovery).

## Approach

Move `resolveRoot` from `main.go` into cliutil as `cliutil.ResolveRoot`. Move the 6 completion helpers into `cliutil/completion.go` with capitalized exports (`RegisterFormatCompletion`, `AllKindNames`, etc.). Update `cmd/aiwf/main.go` and every `cmd/aiwf/*_cmd.go` caller to the new exported names. Update [`cmd/aiwf/completion_drift_test.go`](../../../cmd/aiwf/completion_drift_test.go) reference paths.

## Acceptance criteria

### AC-1 — Completion helpers and resolveRoot exported from internal/cli/cliutil/

**Observable claim.** `cliutil.RegisterFormatCompletion`, `cliutil.AllKindNames`, `cliutil.StatusesForID`, `cliutil.CompleteEntityIDs`, `cliutil.CompleteEntityIDFlag`, `cliutil.CompleteEntityIDArg`, and `cliutil.ResolveRoot` exist as exported package-level functions in `internal/cli/cliutil/`. Callers from any package can import cliutil and invoke each.

**Test seam.** [`internal/cli/cliutil/completion_export_test.go`](../../../internal/cli/cliutil/completion_export_test.go) (added in commit `e7f646d` as the AC's red marker) is in `package cliutil_test` and exercises each export's signature. Before the move it failed to compile (`undefined: cliutil.AllKindNames` etc., 6+ undefineds reported); after the helpers landed in cliutil (`c59f3e5`) the test compiles and passes. The unexported `walkUpFor` helper, which `ResolveRoot` depends on, is in the same package and is exercised via its internal-package test at [`internal/cli/cliutil/resolveroot_test.go`](../../../internal/cli/cliutil/resolveroot_test.go).

### AC-2 — cmd/aiwf has no local definitions; all callers consume cliutil exports

**Observable claim.** The seven helpers (`resolveRoot`, `walkUpFor`, `registerFormatCompletion`, `allKindNames`, `statusesForID`, `completeEntityIDs`, `completeEntityIDFlag`, `completeEntityIDArg`) are not declared as top-level `FuncDecl`s anywhere in `cmd/aiwf/`. Every caller in `cmd/aiwf/*.go` references `cliutil.RegisterFormatCompletion` (etc.) and `cliutil.ResolveRoot`. The completion-drift test continues to pass.

**Test seam.** [`internal/policies/cli_helper_locations.go`](../../../internal/policies/cli_helper_locations.go) (added in commit `57f28a2`) walks every production `.go` file under `cmd/aiwf/` with `WalkGoFiles(root, true)` and flags any top-level FuncDecl in the closed denylist. Verified red against the pre-fix state (8 violations: all 7 helpers + `walkUpFor`); green after the deletion + caller migration. The existing `TestCompletionDrift` / `TestPolicy_PositionalsHaveCompletion` tests in `cmd/aiwf/` continue to exercise the completion wiring end-to-end. Caller-side correctness is the compile-time guarantee — any straggler reference fails `go build`.

## Surfaces touched

- `cmd/aiwf/main.go` — helper removal, `resolveRoot` removal
- `internal/cli/cliutil/completion.go` — new file
- `internal/cli/cliutil/resolveroot.go` — new (or inline in `completion.go`)
- `cmd/aiwf/completion_drift_test.go` — test reference update
- Every `cmd/aiwf/*_cmd.go` caller of `resolveRoot` and the completion helpers — updated to `cliutil.*`

## Out of scope

- Verb subpackage moves (M-3 onward)
- `main.go`'s remaining content (`newRootCmd`, version, `printHelp`) stays in place for now; M-6 handles the final shrink

## Dependencies

- None.

---

## Work log

### AC-1 — Completion helpers + ResolveRoot exported from cliutil

Created `internal/cli/cliutil/resolveroot.go` (`ResolveRoot` + unexported `walkUpFor`) and `internal/cli/cliutil/completion.go` (`RegisterFormatCompletion`, `AllKindNames`, `StatusesForID`, `CompleteEntityIDs`, `CompleteEntityIDFlag`, `CompleteEntityIDArg`). Bodies are byte-identical to the cmd/aiwf/main.go versions; the new home coexists with the old until AC-2 deletes it. · commits `e7f646d` (red test), `c59f3e5` (new cliutil files) · cliutil tests 12/0/0 pass

### AC-2 — Migration + cliutil-helper-locations policy

Bulk-substituted seven call-site patterns across 23 `cmd/aiwf/*.go` files (sed-based, mechanical), deleted the seven helper definitions from `cmd/aiwf/main.go` (137 lines + the now-unused `io/fs`, `path/filepath`, `entity` imports), moved `completion_helpers_test.go` to `internal/cli/cliutil/` (`package cliutil_test`), moved `TestResolveRoot_ExplicitWins` and `TestWalkUpFor` to `internal/cli/cliutil/resolveroot_test.go` (`package cliutil` for unexported-symbol access), fixed five comment / coverage-pragma references to the old names, added `internal/policies/cli_helper_locations.go`. · commit `57f28a2` · policies tests 121/0/0 pass; cmd/aiwf 66s isolated pass

## Decisions made during implementation

- None — all decisions are pre-locked in the spec above.

## Validation

- **Build.** `go build ./...` green after the migration.
- **Tests (changed package isolation).** `go test -parallel 8 -count=1 ./internal/cli/cliutil/...` PASS. `go test -parallel 8 -count=1 ./internal/policies/...` PASS (121 tests including the new `TestPolicy_CLIHelperLocations`). `go test -parallel 8 -count=1 ./cmd/aiwf/...` PASS in isolation (66s).
- **Tests (full module).** Combined `cliutil+policies+cmd/aiwf` run at `-parallel 8` reproduces the documented macOS git-subprocess contention flake on cmd/aiwf at the 11-min Go test timeout. Same flake class encountered in M-0113. Isolated package targets are reliably green; flake is environmental, not a regression from this milestone's diff.
- **Lint.** `golangci-lint run ./...` 0 issues (after `gofumpt -w` on the post-deletion `main.go` and the new policy file).
- **aiwf check.** Zero error-severity findings on M-0114 or its ACs. Tree-wide: 0 errors; the 18 `entity-body-empty` warnings are pre-existing across draft milestones in other proposed epics. The `provenance-untrailered-scope-undefined` advisory persists because `epic/E-0032-...` has no upstream configured yet (deliberately held local per session decision).
- **Branch-coverage audit.** Clean. The migration adds no new branches in production code (caller substitutions are line-level; helper bodies are byte-identical relocations). The new policy's filter branches (cmd/aiwf prefix check, parse-error skip, non-funcdecl/method skip, name-mismatch skip) are all exercised by the walked tree. The violation-append branch is unexercised by a green-state fixture, matching the project's policy-test convention (see M-0113 reviewer notes).
- **wf-doc-lint.** Skipped at AC time; will run at wrap if the change-set touches docs (it doesn't — no `docs/` files modified).

## Deferrals

- None.

## Reviewer notes

- The AC-1 red commit (`e7f646d`) stands as a self-contained external-package test marker; the policy hook `pre-commit.local` (which runs `go test ./internal/policies/...`) doesn't gate on cliutil's compile state, so the red landed cleanly.
- AC-2's red verification followed the M-0113 pattern: the policy + the migration land in one green commit because `pre-commit.local` would reject a red policy commit. Red was verified locally before the green commit by running the policy in isolation and observing the 8 expected violations.
- The 6 completion helpers were renamed to capitalized exports (`RegisterFormatCompletion`, `AllKindNames`, `StatusesForID`, `CompleteEntityIDs`, `CompleteEntityIDFlag`, `CompleteEntityIDArg`) following Go convention; `walkUpFor` stayed unexported because it's a single-purpose internal helper for `ResolveRoot` (YAGNI re. broader export).
- Bulk migration was done via `sed -i ''` rather than per-file Edit calls (23 files × ~3 patterns each = too many tool calls for repetitive substitution). The trade-off was that sed also rewrote the function declarations themselves into syntactically-invalid `func cliutil.X(...)` form — accepted as expected (the declarations were getting deleted in the same commit anyway).
- `cmd/aiwf/main.go` dropped from 631 lines to 494; the milestone spec's "below ~540 lines" target was met with margin to spare.
