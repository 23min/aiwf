---
id: M-0114
title: Lift completion helpers to internal/cli/cliutil/completion.go
status: draft
parent: E-0032
tdd: required
---
## Goal

Move the 6 completion helpers (`registerFormatCompletion`, `allKindNames`, `statusesForID`, `completeEntityIDs`, `completeEntityIDFlag`, `completeEntityIDArg`) from [`cmd/aiwf/main.go:53–145`](../../../cmd/aiwf/main.go) into a new file `internal/cli/cliutil/completion.go`. `main.go` drops below ~540 lines.

## Context

G-0107 step 2 residue. The helpers belong in cliutil (alongside `actor.go`, `flags.go`, `exit.go`) but were stranded in `main.go` because they reference local `resolveRoot`. This milestone resolves that dependency by lifting `resolveRoot` to cliutil as well — it's already a pure helper (root-dir resolution from `--root` flag or `aiwf.yaml` discovery).

## Approach

Move `resolveRoot` from `main.go` into cliutil as `cliutil.ResolveRoot`. Move the 6 completion helpers into `cliutil/completion.go` with capitalized exports (`RegisterFormatCompletion`, `AllKindNames`, etc.). Update `cmd/aiwf/main.go` and every `cmd/aiwf/*_cmd.go` caller to the new exported names. Update [`cmd/aiwf/completion_drift_test.go`](../../../cmd/aiwf/completion_drift_test.go) reference paths.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

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
