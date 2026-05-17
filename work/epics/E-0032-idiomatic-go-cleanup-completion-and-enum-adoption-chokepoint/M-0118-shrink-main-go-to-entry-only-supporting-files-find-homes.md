---
id: M-0118
title: Shrink main.go to entry-only; supporting files find homes
status: draft
parent: E-0032
depends_on:
    - M-0117
tdd: required
---
## Goal

Shrink [`cmd/aiwf/main.go`](../../../cmd/aiwf/main.go) to G-0107's target ~30-line entry-point shape and move the remaining cross-verb infrastructure (`newRootCmd`, version stamping, `printHelp`, and the 5 supporting files [`render_resolver.go`](../../../cmd/aiwf/render_resolver.go), [`rituals.go`](../../../cmd/aiwf/rituals.go), [`show_scopes.go`](../../../cmd/aiwf/show_scopes.go), [`tests_metrics_check.go`](../../../cmd/aiwf/tests_metrics_check.go), [`provenance_check.go`](../../../cmd/aiwf/provenance_check.go)) under `internal/cli/`. After this milestone, `cmd/aiwf/` contains `main.go` only (plus possibly `doc.go`); **G-0107 fully closed.**

## Context

The capstone milestone of G-0107 step 3. M-3, M-4, M-5 moved verbs to per-verb subpackages; M-6 removes the cmd-side residue and packages the remaining cross-verb infrastructure. `main.go` shrinks to the kubectl/helm/hugo-canonical shape: parse args, call `cli.Execute()`.

## Approach

1. **Cross-verb root assembly** → `internal/cli/root.go`. Move `newRootCmd`, version helpers (`resolvedVersion`), and `printHelp` content into the `cli` package. Export `cli.Execute(args []string) int`.
2. **Supporting files find their owning packages:**
   - `render_resolver.go` → `internal/cli/render/` (the render verb consumes it).
   - `show_scopes.go` → `internal/cli/show/`.
   - `rituals.go` → `internal/cli/init/` (primary caller) or `internal/cli/plugins/` if cross-verb use is found.
   - `tests_metrics_check.go` → `internal/cli/check/` or fold into `internal/check/` since it's a check-rule.
   - `provenance_check.go` → `internal/cli/check/` or fold into `internal/check/` similarly.
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

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

## Surfaces touched

- `cmd/aiwf/main.go` — massive shrink to entry-only
- `cmd/aiwf/render_resolver.go`, `rituals.go`, `show_scopes.go`, `tests_metrics_check.go`, `provenance_check.go` — delete
- `internal/cli/root.go` — new (newRootCmd, Execute)
- `internal/cli/render/`, `internal/cli/show/`, `internal/cli/init/`, `internal/cli/check/` — gain the supporting files
- `internal/cli/integration/` (or named cross-verb test home) — gains the integration tests
- `cmd/aiwf/` — at the end, contains `main.go` only

## Out of scope

- Enum policy work (M-7)
- Further reorganization inside `internal/` beyond G-0107's target shape

## Dependencies

- M-5 (all verbs must be subpackaged before main.go can shrink to entry-only).
