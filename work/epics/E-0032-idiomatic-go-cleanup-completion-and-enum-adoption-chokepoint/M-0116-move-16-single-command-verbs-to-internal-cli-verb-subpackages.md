---
id: M-0116
title: Move 16 single-command verbs to internal/cli/<verb>/ subpackages
status: in_progress
parent: E-0032
depends_on:
    - M-0115
tdd: required
acs:
    - id: AC-1
      title: internal/cli/archive/ carries archive verb
      status: met
      tdd_phase: done
    - id: AC-2
      title: internal/cli/authorize/ carries authorize verb
      status: met
      tdd_phase: done
    - id: AC-3
      title: internal/cli/history/ carries history verb
      status: met
      tdd_phase: done
    - id: AC-4
      title: internal/cli/importcmd/ carries import verb
      status: open
      tdd_phase: red
    - id: AC-5
      title: internal/cli/init/ carries init verb
      status: open
      tdd_phase: red
    - id: AC-6
      title: internal/cli/list/ carries list verb
      status: met
      tdd_phase: done
    - id: AC-7
      title: internal/cli/render/ carries render verb
      status: open
      tdd_phase: red
    - id: AC-8
      title: internal/cli/retitle/ carries retitle verb
      status: met
      tdd_phase: done
    - id: AC-9
      title: internal/cli/rewidth/ carries rewidth verb
      status: open
      tdd_phase: green
    - id: AC-10
      title: internal/cli/schema/ carries schema verb
      status: met
      tdd_phase: done
    - id: AC-11
      title: internal/cli/show/ carries show verb
      status: open
      tdd_phase: red
    - id: AC-12
      title: internal/cli/status/ carries status verb
      status: open
      tdd_phase: red
    - id: AC-13
      title: internal/cli/template/ carries template verb
      status: met
      tdd_phase: done
    - id: AC-14
      title: internal/cli/update/ carries update verb
      status: met
      tdd_phase: done
    - id: AC-15
      title: internal/cli/upgrade/ carries upgrade verb
      status: open
      tdd_phase: red
    - id: AC-16
      title: internal/cli/whoami/ carries whoami verb
      status: met
      tdd_phase: done
---
## Goal

Move 16 single-command verbs (`archive`, `authorize`, `history`, `import`, `init`, `list`, `render`, `retitle`, `rewidth`, `schema`, `show`, `status`, `template`, `update`, `upgrade`, `whoami`) from `cmd/aiwf/<verb>_cmd.go` into per-verb subpackages under `internal/cli/<verb>/`. After this milestone, only `verbs_cmd.go`'s former 8 verbs (now subpackaged) and the multi-subcommand cluster (`contract`, `doctor`, `milestone`) remain to migrate.

## Context

Largest cluster of G-0107 step 3 execution. Each verb is already in its own `*_cmd.go`, so this is purely a move (not a file-split). Uses M-3's pattern.

## Approach

For each verb, move `cmd/aiwf/<verb>_cmd.go` → `internal/cli/<verb>/<verb>.go` exporting `New<Verb>Cmd()`. Move associated `cmd/aiwf/<verb>_*_test.go` files into `internal/cli/<verb>/`. Update `cmd/aiwf/main.go`'s `newRootCmd` to import each new package. Update completion-drift test. **One verb per commit** so partial failure is rollbackable and review is per-verb.

Note: `render_cmd.go` has a sibling [`render_resolver.go`](../../../cmd/aiwf/render_resolver.go) that depends on render's wiring — `render_resolver.go` stays in cmd/aiwf/ for M-6 to handle (it has cross-verb concerns; doesn't move with `render` alone). Same caveat for `show_cmd.go` and [`show_scopes.go`](../../../cmd/aiwf/show_scopes.go), `init_cmd.go` and [`rituals.go`](../../../cmd/aiwf/rituals.go).

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

## Surfaces touched

- `cmd/aiwf/<verb>_cmd.go` × 16 — delete
- `cmd/aiwf/<verb>_*_test.go` × many — move
- `internal/cli/<verb>/` × 16 — new packages
- `cmd/aiwf/main.go` — imports
- `cmd/aiwf/completion_drift_test.go` — drift-test update

## Out of scope

- Multi-subcommand verbs (M-5)
- Supporting-file moves: `render_resolver.go`, `show_scopes.go`, `rituals.go`, `selfcheck.go`, `tests_metrics_check.go`, `provenance_check.go` — all M-6
- `main.go` shrink (M-6)

## Dependencies

- M-3 (pattern-setter must land first).

### AC-1 — internal/cli/archive/ carries archive verb

### AC-2 — internal/cli/authorize/ carries authorize verb

### AC-3 — internal/cli/history/ carries history verb

### AC-4 — internal/cli/importcmd/ carries import verb

### AC-5 — internal/cli/init/ carries init verb

### AC-6 — internal/cli/list/ carries list verb

### AC-7 — internal/cli/render/ carries render verb

### AC-8 — internal/cli/retitle/ carries retitle verb

### AC-9 — internal/cli/rewidth/ carries rewidth verb

### AC-10 — internal/cli/schema/ carries schema verb

### AC-11 — internal/cli/show/ carries show verb

### AC-12 — internal/cli/status/ carries status verb

### AC-13 — internal/cli/template/ carries template verb

### AC-14 — internal/cli/update/ carries update verb

### AC-15 — internal/cli/upgrade/ carries upgrade verb

### AC-16 — internal/cli/whoami/ carries whoami verb

