---
id: M-0115
title: Move verbs_cmd.go's 8 verbs to internal/cli/<verb>/ subpackages
status: in_progress
parent: E-0032
depends_on:
    - M-0114
tdd: required
acs:
    - id: AC-1
      title: internal/cli/add/ carries add and add ac verbs
      status: open
      tdd_phase: red
    - id: AC-2
      title: internal/cli/promote/ carries promote verb
      status: open
      tdd_phase: red
    - id: AC-3
      title: internal/cli/editbody/ carries edit-body verb
      status: met
      tdd_phase: done
    - id: AC-4
      title: internal/cli/cancel/ carries cancel verb
      status: met
      tdd_phase: done
    - id: AC-5
      title: internal/cli/rename/ carries rename verb
      status: met
      tdd_phase: done
    - id: AC-6
      title: internal/cli/move/ carries move verb
      status: met
      tdd_phase: done
    - id: AC-7
      title: internal/cli/reallocate/ carries reallocate verb
      status: met
      tdd_phase: done
    - id: AC-8
      title: Shared helpers lifted to cliutil; verbs_cmd.go deleted; rootCmd wired
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

### AC-2 — internal/cli/promote/ carries promote verb

### AC-3 — internal/cli/editbody/ carries edit-body verb

### AC-4 — internal/cli/cancel/ carries cancel verb

### AC-5 — internal/cli/rename/ carries rename verb

### AC-6 — internal/cli/move/ carries move verb

### AC-7 — internal/cli/reallocate/ carries reallocate verb

### AC-8 — Shared helpers lifted to cliutil; verbs_cmd.go deleted; rootCmd wired

