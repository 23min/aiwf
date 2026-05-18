---
id: M-0118
title: Shrink main.go to entry-only; supporting files find homes
status: in_progress
parent: E-0032
depends_on:
    - M-0117
tdd: required
acs:
    - id: AC-1
      title: internal/cli/root.go assembles root command and exports Execute
      status: open
      tdd_phase: done
    - id: AC-2
      title: check verb body moves to internal/cli/check/
      status: met
      tdd_phase: done
    - id: AC-3
      title: tests_metrics_check moves to internal/cli/check/
      status: met
      tdd_phase: done
    - id: AC-4
      title: provenance_check moves to internal/cli/check/
      status: met
      tdd_phase: done
    - id: AC-5
      title: cmd/aiwf/main.go shrunk to entry-only shape (function main only)
      status: open
      tdd_phase: red
    - id: AC-6
      title: cobra integration tests relocate to internal/cli/integration/
      status: open
      tdd_phase: red
    - id: AC-7
      title: captureStdout lifted to shared testutil; per-package duplicates forbidden
      status: open
      tdd_phase: red
    - id: AC-8
      title: JSON-envelope version-source drift policy added
      status: open
      tdd_phase: red
---
## Goal

Shrink [`cmd/aiwf/main.go`](../../../cmd/aiwf/main.go) to G-0107's target ~30-line entry-point shape and move the remaining cross-verb infrastructure (`newRootCmd`, version stamping, `printHelp`, and the 2 supporting files still under `cmd/aiwf/` — [`tests_metrics_check.go`](../../../cmd/aiwf/tests_metrics_check.go) and [`provenance_check.go`](../../../cmd/aiwf/provenance_check.go)) under `internal/cli/`. After this milestone, `cmd/aiwf/` contains `main.go` only (plus possibly `doc.go`); **G-0107 fully closed.**

## Context

The capstone milestone of G-0107 step 3. M-0115 and M-0116 moved verbs to per-verb subpackages and joint-moved three supporting files with them (`render_resolver.go` → `internal/cli/render/`, `show_scopes.go` → `internal/cli/show/`, `rituals.go` → `internal/cli/initcmd/`). M-0117 removes the multi-subcommand cmd-side residue (`contract`, `doctor`, `milestone`). M-0118 packages the remaining cross-verb infrastructure and shrinks `main.go` to the kubectl/helm/hugo-canonical shape: parse args, call `cli.Execute()`.

## Approach

1. **Cross-verb root assembly** → `internal/cli/root.go`. Move `newRootCmd`, version helpers (`resolvedVersion`), and `printHelp` content into the `cli` package. Export `cli.Execute(args []string) int`.
2. **Supporting files find their owning packages:**
   - `tests_metrics_check.go` → `internal/cli/check/` or fold into `internal/check/` since it's a check-rule.
   - `provenance_check.go` → `internal/cli/check/` or fold into `internal/check/` similarly.
   - (`selfcheck.go` moves with `doctor` in M-0117, not here.)
3. **`main.go` final shape:**

   ```go
   package main

   import (
       "os"
       "github.com/23min/aiwf/internal/cli"
   )

   func main() {
       os.Exit(cli.Execute(os.Args[1:]))
   }
   ```
4. **Integration tests** under `cmd/aiwf/integration*_test.go`, `binary_integration_test.go`, `envelope_schema_test.go`, etc. — relocate to `internal/cli/integration/` (or similar; settle the destination here) so `cmd/aiwf/` stays test-free.
5. **Lift `captureStdout` to a shared testutil location.** M-0116 left a duplicated `captureStdout` helper: the original at [`cmd/aiwf/helpers_test.go`](../../../cmd/aiwf/helpers_test.go) and a copy at [`internal/cli/initcmd/helpers_test.go`](../../../internal/cli/initcmd/helpers_test.go) (added when `rituals_test.go` moved with the init verb and couldn't import across the `package main` / `package initcmd` boundary). The clean fix is one helper in a shared test-only package (e.g. `internal/cli/cliutil/testutil/` or `internal/testutil/`); both call sites import it and the duplicate file is deleted. This milestone is the natural absorber since the integration-test relocation (item 4) settles the shared-testutil destination anyway. An accompanying drift policy under `internal/policies/` should forbid re-introducing `captureStdout` as a per-package copy.
6. **Converge JSON-envelope version source on `version.Current().Version`.** M-0116 and M-0117 left the codebase with two parallel "what version am I" sources for JSON envelopes: the moved-out verbs (`contract`, `status`, `show`, `history`, `schema`, etc. in `internal/cli/<pkg>/`) use `version.Current().Version`; the still-in-place `check` verb in [`cmd/aiwf/main.go:448,505`](../../../cmd/aiwf/main.go) uses the package-global `Version`. The two agree under ldflags-stamped builds (both resolve to e.g. `v0.2.0`) but diverge under unstamped builds (`Version="dev"` vs buildinfo's `(devel)`). A consumer running `aiwf check --format=json` then `aiwf status --format=json` against the same binary can see two different version strings on the same invocation. This is the same seam shape G-0027 closed for the `version`/`doctor` verbs in commit `f810a86`; that fix did not cover every JSON-envelope-emitting verb. The clean fix folds naturally into item 1 (check moves to `internal/cli/check/` or `internal/cli/root.go`) — at that point the new home picks `version.Current().Version` consistently. The accompanying drift policy should forbid `Version:` field initialisations from naming the package-global `Version` once `main.go` shrinks to entry-only.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

## Surfaces touched

- `cmd/aiwf/main.go` — massive shrink to entry-only
- `cmd/aiwf/tests_metrics_check.go`, `provenance_check.go` — delete (relocated to owning packages; `selfcheck.go` moves with `doctor` in M-0117)
- `cmd/aiwf/helpers_test.go` — `captureStdout` lifted out; cmd/aiwf-side definition deleted
- `internal/cli/initcmd/helpers_test.go` — deleted entirely once `captureStdout` lives in the shared testutil
- `internal/cli/root.go` — new (newRootCmd, Execute)
- `internal/cli/check/` (or `internal/check/`) — gains `tests_metrics_check.go` + `provenance_check.go`
- `internal/cli/cliutil/testutil/` (or `internal/testutil/`) — new home for `captureStdout` (the destination is settled when item 4's integration-test relocation lands)
- `internal/cli/integration/` (or named cross-verb test home) — gains the integration tests
- `internal/policies/` — new drift policy forbidding per-package `captureStdout` copies
- `cmd/aiwf/` — at the end, contains `main.go` only

## Out of scope

- Enum policy work (M-0119)
- Further reorganization inside `internal/` beyond G-0107's target shape

## Dependencies

- M-0117 (the multi-subcommand verbs must move out of `cmd/aiwf/` before `main.go` can shrink to entry-only).

### AC-1 — internal/cli/root.go assembles root command and exports Execute

### AC-2 — check verb body moves to internal/cli/check/

### AC-3 — tests_metrics_check moves to internal/cli/check/

### AC-4 — provenance_check moves to internal/cli/check/

### AC-5 — cmd/aiwf/main.go shrunk to entry-only shape (function main only)

### AC-6 — cobra integration tests relocate to internal/cli/integration/

### AC-7 — captureStdout lifted to shared testutil; per-package duplicates forbidden

### AC-8 — JSON-envelope version-source drift policy added

