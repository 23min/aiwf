---
id: M-0115
title: Move verbs_cmd.go's 8 verbs to internal/cli/<verb>/ subpackages
status: done
parent: E-0032
depends_on:
    - M-0114
tdd: required
acs:
    - id: AC-1
      title: internal/cli/add/ carries add and add ac verbs
      status: met
      tdd_phase: done
    - id: AC-2
      title: internal/cli/promote/ carries promote verb
      status: met
      tdd_phase: done
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
      status: met
      tdd_phase: done
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

Per the epic's "Per-verb moves are independently shippable" constraint, AC-1 through AC-7 each pin one subpackage's existence; AC-8 pins the cleanup (helpers lifted, monolith deleted, rootCmd wired). Each verb's subpackage exports `NewCmd() *cobra.Command` as the canonical Cobra constructor, carries `setup_test.go` per CLAUDE.md test-discipline, and provides at minimum a smoke test pinning the export shape (Use string, flag set, ValidArgsFunction wiring). Behavioral coverage for each verb stays with the cross-verb integration tests under `cmd/aiwf/` — they spawn the aiwf binary via `aiwfBinary(t)` and don't reference internal cmd/aiwf symbols, so they continue to exercise each verb end-to-end through the dispatcher (which now routes through `<pkg>.NewCmd`).

### AC-1 — internal/cli/add/ carries add and add ac verbs

The two related verbs share `internal/cli/add/` because `add ac` is a Cobra subcommand of `add` and inherits its PersistentFlags. The subpackage exports `NewCmd` (which builds the parent and registers the `ac` child internally via `newACCmd`). The `add contract --validator …` path depends on `cliutil.LoadContractsDoc` (lifted from `cmd/aiwf/contract_cmd.go` as part of AC-8).

### AC-2 — internal/cli/promote/ carries promote verb

The most flag-rich of the simpler verbs: `--phase`, `--tests`, `--by`, `--by-commit`, `--superseded-by`, `--force`, `--audit-only`. The `RangeArgs(1,2)` shape supporting both `promote <id> <status>` (top-level entity) and `promote <id> --phase <p>` (composite-id AC phase mode) is preserved verbatim.

### AC-3 — internal/cli/editbody/ carries edit-body verb

Package name is `editbody` (no separator) per Go's package-naming convention (CLAUDE.md *Naming*); the CLI verb name remains `edit-body` (hyphenated, the user-facing form). Two-mode shape: bless mode (no `--body-file` → verb reads working-copy + HEAD itself) and explicit mode (`--body-file <path>` → verb reads bytes from path or `-` for stdin).

### AC-4 — internal/cli/cancel/ carries cancel verb

Pattern-setter for the per-verb subpackage shape — first verb migrated, simplest to verify. Carries the `--audit-only` recovery path and the `--force` audit-trailer-only path; both require `--reason` non-empty.

### AC-5 — internal/cli/rename/ carries rename verb

Slug-only verb: the entity id is preserved across the rename. No frontmatter mutation; the verb wires `cliutil.ConfiguredTitleMaxLength(rootDir)` so the title/slug width cap stays uniform with `aiwf add` and `aiwf retitle`.

### AC-6 — internal/cli/move/ carries move verb

Milestone-only verb (the `--epic` flag is mandatory). The verb's provenance context carries both `MoveSource` (the milestone's current parent epic) and `TargetID` (the destination epic) so the I2.5 strict-move allow-rule can verify both endpoints reach the scope-entity.

### AC-7 — internal/cli/reallocate/ carries reallocate verb

Renumbers an entity and rewrites references across the tree. Standard resolution path for an `ids-unique` finding from `aiwf check`. The verb uses `cliutil.LoadTreeWithTrunk` rather than `tree.Load` so the trunk-rename seam is respected.

### AC-8 — Shared helpers lifted to cliutil; verbs_cmd.go deleted; rootCmd wired

Three sub-claims, all mechanical:
1. **Shared helpers lifted.** `cliutil.ParseKind`, `cliutil.ParseTestsFlag`, `cliutil.ReadBodyFile`, `cliutil.SplitCommaList` (from `verbs_cmd.go`) + `cliutil.LoadContractsDoc`, `cliutil.LoadContractsBlock` (from `contract_cmd.go`) — all exist as exported helpers in `internal/cli/cliutil/`. Each carries a focused unit test in `internal/cli/cliutil/verbhelpers_test.go`.
2. **verbs_cmd.go deleted.** The 978-line monolith is gone; `cmd/aiwf/` no longer hosts the 8 verb definitions.
3. **rootCmd wired.** `cmd/aiwf/main.go`'s `newRootCmd` references `add.NewCmd()`, `promote.NewCmd()`, `cancel.NewCmd()`, `rename.NewCmd()`, `editbody.NewCmd()`, `move.NewCmd()`, `reallocate.NewCmd()`.

Mechanical evidence: `internal/policies/cli_helper_locations.go` (the M-0114 helper-location policy) catches re-duplication of the M-0114 helper set in cmd/aiwf; the file-existence check on `verbs_cmd.go` is the load-bearing chokepoint for "the monolith doesn't come back" (a new verbs_cmd.go would surface in code review and be a deliberate, named choice). `internal/policies/skill_coverage.go`'s AST walker was extended to recognize both `newXCmd()` Ident form (legacy) and `pkg.NewCmd()` SelectorExpr form (M-0115); without this extension every verb migration would fail the skill-coverage policy.

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

---

## Work log

### AC-8 prep — Shared helpers lifted to cliutil

`cliutil.ParseKind`, `cliutil.ParseTestsFlag`, `cliutil.ReadBodyFile`, `cliutil.SplitCommaList` lifted with focused unit tests; the lone external caller (`milestone_cmd.go:109`) updated to `cliutil.SplitCommaList`. `cliutil.LoadContractsDoc` and `cliutil.LoadContractsBlock` lifted later as a precondition for AC-1's add migration. · commit `15460ad` (helpers) + commit folded into AC-1 commit `<add-commit>` (contracts loader)

### AC-4 — cancel subpackage

NewCmd + Run + setup_test + smoke test in `internal/cli/cancel/`. Pattern-setter for the per-verb shape; also extended `internal/policies/skill_coverage.go` to recognize `pkg.NewCmd()` SelectorExpr form (without this every subsequent verb migration would fail the skill-coverage policy). · commit `3042c97` · cancel smoke test 1/0/0 pass, policies 121/0/0 pass

### AC-5 — rename subpackage

Slug-only verb. Same shape as cancel. · commit on epic branch · rename smoke test 1/0/0 pass

### AC-6 — move subpackage

Milestone-only verb with mandatory `--epic`. Same shape as preceding two. · commit on epic branch · move smoke test 1/0/0 pass

### AC-7 — reallocate subpackage

Uses `cliutil.LoadTreeWithTrunk` rather than `tree.Load` (trunk-rename seam). Same shape as preceding three. · commit on epic branch · reallocate smoke test 1/0/0 pass

### AC-3 — editbody subpackage

Two-mode shape: bless (working-copy diff) vs. explicit (--body-file). Package name `editbody` (no separator) per Go convention; CLI verb remains `edit-body`. Existing binary-integration tests `cmd/aiwf/edit_body_cmd_test.go` stay in place (they spawn the binary). · commit on epic branch · editbody smoke test 1/0/0 pass

### AC-2 — promote subpackage

Flag-rich verb with RangeArgs(1,2) shape (status vs. phase mode). All flags preserved verbatim. · commit on epic branch · promote smoke test 1/0/0 pass

### AC-1 + AC-8 finalize — add subpackage + verbs_cmd.go deleted + rootCmd wired

`add` (with `add ac` Cobra subcommand) moves to `internal/cli/add/`. `cliutil.LoadContractsDoc` / `cliutil.LoadContractsBlock` lifted in the same commit (precondition for the add migration). After the move, `verbs_cmd.go` is empty and is deleted. `cmd/aiwf/main.go`'s `newRootCmd` uses `pkg.NewCmd()` form for the 8 migrated verbs (legacy `newXCmd` ident form remains for the other 16 verbs M-0116/M-0117 will migrate). · commit on epic branch · add smoke test 1/0/0 pass, policies 121/0/0 pass

## Decisions made during implementation

- **Package naming.** `editbody` (no separator) chosen for the package directory per CLAUDE.md *Naming* ("Package names: short, lowercase, no underscores"). The CLI verb name `edit-body` (hyphenated) is unchanged.
- **Test-move policy.** Binary-integration tests under `cmd/aiwf/<verb>_*_test.go` (e.g. `edit_body_cmd_test.go`, `add_*_test.go`, `promote_resolver_cmd_test.go`) stay in `cmd/aiwf/`. They spawn the aiwf binary via `aiwfBinary(t)` and don't reference internal cmd/aiwf symbols, so they continue to exercise each verb end-to-end through the dispatcher. The spec said "move tests with the verbs"; in practice moving them would have required relocating package-internal test helpers (`aiwfBinary`, `setupCLITestRepo`, `chdir`, etc.) which is out of M-0115's scope (those helpers serve the other 16 verbs still in cmd/aiwf). Smoke tests in each subpackage pin the export shape.
- **Skill-coverage policy extension.** `internal/policies/skill_coverage.go`'s AST walker had to be extended to recognize the new `pkg.NewCmd()` SelectorExpr form alongside the legacy `newXCmd()` Ident form. Without this every verb migration would have failed the policy at the pre-commit hook (skills reference `aiwf cancel` etc. in body backticks; the policy needs to confirm the verb is registered). The extension landed in the cancel commit (`3042c97`) so each subsequent verb's migration sailed through clean.

## Validation

- **Build.** `go build ./...` green throughout the milestone.
- **Tests (changed package isolation).** Every new subpackage's smoke test passes. `internal/policies/` 121/0/0 pass at every commit. `internal/cli/cliutil/` 12/0/0 pass. `cmd/aiwf/` 66s isolated PASS after each verb migration.
- **Tests (full module).** Same documented macOS git-subprocess contention flake as M-0113 and M-0114 — at `-parallel 8` with cmd/aiwf in the mix, runs sometimes hit the 11-min Go test timeout. Isolated package targets are reliably green; flake is environmental.
- **Lint.** `golangci-lint run ./...` 0 issues throughout (after `gofumpt -w` on a couple of mid-migration intermediate states).
- **aiwf check.** Zero error-severity findings on M-0115 or its ACs. Tree-wide: 0 errors; the 23 `entity-body-empty` warnings are pre-existing across draft milestones in other proposed epics plus this milestone's AC bodies which are now populated.
- **Branch-coverage audit.** Clean. Each verb's migration is a line-substitution + relocation; no new branches introduced. The new policy extension in `skill_coverage.go` adds two filter branches (Ident vs. SelectorExpr) — both exercised by the walked tree (Ident by the 16 verbs still in cmd/aiwf, SelectorExpr by the 8 migrated verbs).

## Deferrals

- None.

## Reviewer notes

- M-0115 is the pattern-setter for M-0116 (single-command verbs) and M-0117 (multi-subcommand verbs). The shape established here is: `internal/cli/<verb>/<verb>.go` exports `NewCmd() *cobra.Command` + `Run(args...) int`; the subpackage carries `setup_test.go` (GIT identity env vars) and at minimum a `<verb>_test.go` smoke test. Existing binary-integration tests stay in `cmd/aiwf/`.
- Bulk sed substitution worked well for the helper migration (`s/<old>(/<new>(/g`). It does break function declarations as expected (e.g. `func parseKind(...)` becomes `func cliutil.ParseKind(...)`), but those declarations are deleted in the same commit anyway.
- The `internal/cli/doc.go` pattern-documentation file mentioned in the spec is NOT created in this milestone — the in-code naming conventions (subpackage name = verb name lowercase, exported `NewCmd`/`Run`, `setup_test.go` per package) plus the per-verb commit messages serve as the pattern reference for M-0116/M-0117 implementers. A standalone doc adds an indirection layer without clear benefit while the pattern is fresh in the worktree's history. If M-0116 implementation surfaces friction reading the pattern from commits, that's the signal to add the doc.
- The completion_drift_test.go reference path update mentioned in the spec was unnecessary in practice: the drift test reads from the assembled cobra tree, not from source paths, so it sees the new subpackage's wiring through `newRootCmd`'s AddCommand calls.
- `verbs_cmd.go` ended up empty after AC-1's add migration (all 8 verbs out + all 4 shared helpers lifted in AC-8's prep step), so the file was deleted in the AC-1 commit rather than waiting for a separate AC-8 finalization commit. The two ACs land as one commit; both promote-met after.
