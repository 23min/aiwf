---
id: M-0115
title: Move verbs_cmd.go's 8 verbs to internal/cli/<verb>/ subpackages
status: draft
parent: E-0032
depends_on:
    - M-0114
tdd: required
acs:
    - id: AC-1
      title: internal/cli/add/ carries add and add ac verbs
      status: open
      tdd_phase: red
---
## Goal

Move the 8 verbs in [`cmd/aiwf/verbs_cmd.go`](../../../cmd/aiwf/verbs_cmd.go) (`add`, `add ac`, `promote`, `edit-body`, `cancel`, `rename`, `move`, `reallocate`) into per-verb subpackages under `internal/cli/<verb>/`. Establish the per-verb subpackage pattern that M-4, M-5, M-6 build on. Delete `verbs_cmd.go`.

## Context

First milestone of G-0107 step 3 execution. The 8-verb monolith is the equivalent of `admin_cmd.go` that step 1 split. This milestone both splits the file AND moves the resulting per-verb code into subpackages — the file-split-only intermediate state is not shipped; it would be a worse outcome than today's structure.

## Approach

For each verb, create `internal/cli/<verb>/<verb>.go` (verb constructor + run function) and `internal/cli/<verb>/<verb>_test.go` (existing tests from `cmd/aiwf/<verb>_*_test.go` moved here). `add` and `add ac` share `internal/cli/add/` since `add ac` is a Cobra subcommand of `add`. Each package exports a single `New(deps Deps) *cobra.Command` (or `NewCmd()`) so `cmd/aiwf/main.go`'s `newRootCmd` can wire them. Delete `cmd/aiwf/verbs_cmd.go`. Update completion-drift test for the new package paths. Document the per-verb-package pattern in `internal/cli/doc.go` so M-4 and M-5 have a reference.

The 7 subpackages: `internal/cli/add/` (carries `add` and `add ac`), `internal/cli/promote/`, `internal/cli/editbody/` (or `internal/cli/edit_body/` — settle convention here), `internal/cli/cancel/`, `internal/cli/rename/`, `internal/cli/move/`, `internal/cli/reallocate/`.

Shared helpers (`parseKind`, `parseTestsFlag`, `readBodyFile`, `splitCommaList` at [`cmd/aiwf/verbs_cmd.go:359–416,971`](../../../cmd/aiwf/verbs_cmd.go)) lift to `internal/cli/cliutil/`.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

## Surfaces touched

- `cmd/aiwf/verbs_cmd.go` — delete
- `cmd/aiwf/<verb>_*_test.go` — move to `internal/cli/<verb>/`
- `internal/cli/add/`, `internal/cli/promote/`, `internal/cli/editbody/`, `internal/cli/cancel/`, `internal/cli/rename/`, `internal/cli/move/`, `internal/cli/reallocate/` — new packages
- `internal/cli/cliutil/` — lift `parseKind`, `parseTestsFlag`, `readBodyFile`, `splitCommaList`
- `internal/cli/doc.go` — pattern documentation
- `cmd/aiwf/main.go` — `newRootCmd` imports the new packages
- `cmd/aiwf/completion_drift_test.go` — drift-test update

## Out of scope

- Other verb moves (M-4, M-5)
- Supporting-file moves (M-6)
- `main.go` shrink (M-6)

## Dependencies

- M-2 (cliutil completion helpers must be in place before per-verb packages can import them as `cliutil.*`).

### AC-1 — internal/cli/add/ carries add and add ac verbs

