---
id: M-0113
title: Consolidate trailer parser
status: draft
parent: E-0032
tdd: required
---
## Goal

Eliminate the duplicate trailer-parser implementation. Export [`internal/gitops/gitops.go:312`](../../../internal/gitops/gitops.go) `parseTrailers` as `gitops.ParseTrailers`, switch the three callers, delete [`internal/cli/cliutil/scopes.go:258`](../../../internal/cli/cliutil/scopes.go) `ParseTrailerLines`.

## Context

G-0107 step 2 residue. The cliutil variant was added during the verb-support extraction (commit `1d391c5`) but the original `parseTrailers` in gitops was never removed, leaving two byte-identical implementations of `git log %(trailers:only=true,unfold=true)` line parsing.

## Approach

Rename `parseTrailers` → `ParseTrailers` in gitops; the existing tests at [`internal/gitops/gitops_test.go:59`](../../../internal/gitops/gitops_test.go) and [`internal/gitops/trailers_test.go:242,264`](../../../internal/gitops/trailers_test.go) update their lowercase-name calls. Delete `ParseTrailerLines` in cliutil. Switch the three call sites ([`cmd/aiwf/provenance_check.go:182,228`](../../../cmd/aiwf/provenance_check.go) and [`internal/cli/cliutil/scopes.go:166`](../../../internal/cli/cliutil/scopes.go)) to `gitops.ParseTrailers`. Single commit; no behavior change; tests stay green.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

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
