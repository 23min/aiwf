---
id: M-0117
title: Move contract, doctor, milestone (multi-subcommand) to subpackages
status: draft
parent: E-0032
depends_on:
    - M-0116
tdd: required
acs:
    - id: AC-1
      title: Move contract verb + all 6 subcommands to internal/cli/contract subpackage
      status: open
      tdd_phase: red
    - id: AC-2
      title: Move contract tests to internal/cli/contract package; tests green
      status: open
      tdd_phase: red
---
## Goal

Move `contract` (6 subcommands: `verify`, `bind`, `unbind`, `recipes`, `recipe show`, `recipe install`, `recipe remove`), `doctor` (with `--self-check` mode), and `milestone` (with `depends-on` subcommand) from `cmd/aiwf/<verb>_cmd.go` into per-verb subpackages, preserving subcommand wiring.

## Context

Final cluster of G-0107 step 3 verb-move work. Multi-subcommand verbs need more careful migration than M-4's single-command moves because subcommand registration must work across the package boundary — the parent verb's package owns its subcommand graph internally.

## Approach

For each verb, the per-verb package exports a parent `NewContractCmd()` (etc.) that internally constructs and registers its subcommands as Cobra children. Subcommand-specific helpers stay package-private inside the parent's package (e.g., `internal/cli/contract/recipes.go` for the recipe-handling code).

- `internal/cli/contract/contract.go` — parent cmd
- `internal/cli/contract/verify.go`, `bind.go`, `unbind.go`, `recipes.go` — subcommands
- `internal/cli/doctor/doctor.go` — parent cmd
- `internal/cli/doctor/selfcheck.go` — moves with doctor from [`cmd/aiwf/selfcheck.go`](../../../cmd/aiwf/selfcheck.go)
- `internal/cli/milestone/milestone.go` — parent cmd
- `internal/cli/milestone/depends_on.go` — subcommand

Per-package `_test.go` carries the previously-passing `cmd/aiwf/contract_cmd_test.go`, `doctor_cmd_test.go`, `milestone_*_test.go` content.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

## Surfaces touched

- `cmd/aiwf/contract_cmd.go`, `doctor_cmd.go`, `milestone_cmd.go` — delete
- `cmd/aiwf/selfcheck.go` — moves with doctor to `internal/cli/doctor/`
- `cmd/aiwf/*contract*_test.go`, `*doctor*_test.go`, `*milestone*_test.go` — move
- `internal/cli/contract/`, `internal/cli/doctor/`, `internal/cli/milestone/` — new packages
- `cmd/aiwf/main.go` — imports

## Out of scope

- Other supporting-file moves: `render_resolver.go`, `show_scopes.go`, `rituals.go`, `tests_metrics_check.go`, `provenance_check.go` (M-6)
- `main.go` shrink (M-6)

## Dependencies

- M-4 (single-command pattern must stabilize before tackling subcommand wiring).

### AC-1 — Move contract verb + all 6 subcommands to internal/cli/contract subpackage

### AC-2 — Move contract tests to internal/cli/contract package; tests green

