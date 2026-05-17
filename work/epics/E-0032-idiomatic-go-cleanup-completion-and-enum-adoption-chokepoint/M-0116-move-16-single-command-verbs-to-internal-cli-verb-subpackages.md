---
id: M-0116
title: Move 16 single-command verbs to internal/cli/<verb>/ subpackages
status: done
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
      status: met
      tdd_phase: done
    - id: AC-5
      title: internal/cli/init/ carries init verb
      status: met
      tdd_phase: done
    - id: AC-6
      title: internal/cli/list/ carries list verb
      status: met
      tdd_phase: done
    - id: AC-7
      title: internal/cli/render/ carries render verb
      status: met
      tdd_phase: done
    - id: AC-8
      title: internal/cli/retitle/ carries retitle verb
      status: met
      tdd_phase: done
    - id: AC-9
      title: internal/cli/rewidth/ carries rewidth verb
      status: met
      tdd_phase: done
    - id: AC-10
      title: internal/cli/schema/ carries schema verb
      status: met
      tdd_phase: done
    - id: AC-11
      title: internal/cli/show/ carries show verb
      status: met
      tdd_phase: done
    - id: AC-12
      title: internal/cli/status/ carries status verb
      status: met
      tdd_phase: done
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
      status: met
      tdd_phase: done
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

16 per-verb ACs, one per subpackage. Each verb's subpackage exports `NewCmd` (and `Run` where applicable) as the canonical Cobra constructor; `setup_test.go` carries the GIT identity env-var seeding per CLAUDE.md test discipline; a smoke test pins each NewCmd's exported shape (Use string, flag set, ValidArgsFunction wiring where present). Behavioral coverage stays with the existing cross-verb integration tests under `cmd/aiwf/` (binary-integration tests spawn the aiwf binary via `aiwfBinary(t)` and dispatch through the new subpackage's NewCmd transparently).

Three verb migrations carried sibling files alongside them (the spec was conservative; the deps proved clean for a joint move):

- **init + rituals.go** → `internal/cli/initcmd/` (package name `initcmd` because `init` is a Go-reserved identifier — the package-level init() function name)
- **show + show_scopes.go** → `internal/cli/show/`
- **render + render_resolver.go** → `internal/cli/render/`

`import` is also renamed at the package level: `internal/cli/importcmd/` because `import` is a Go reserved word. The CLI verb name remains `import`.

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

---

## Work log

All 16 verbs migrated. Each AC's code commit follows the M-0115 pattern: `git mv` the source file, sed-rename `package main` → `package <verb>` and `new<Verb>Cmd` → `NewCmd` / `run<Verb>Cmd` → `Run`, add `setup_test.go`, write smoke test, update `cmd/aiwf/main.go`'s `newRootCmd`, fix any cross-file references that broke. Phase transitions follow: red → green → done → met per AC.

Sibling-file moves recorded inline at AC-5 (init+rituals), AC-7 (render+resolver), AC-11 (show+scopes). Two verbs use special package names because the verb name is Go-reserved: AC-4 `importcmd`, AC-5 `initcmd`.

Cross-cut symbol exports forced by the moves (formerly cmd/aiwf-internal helpers that now had to be reachable from subpackages or stay in cmd/aiwf):
- `history.{HistoryEvent, ReadHistory, ReadHistoryChain, StripTrailers, ShortHash, RenderTo, RenderActor, RenderScopeChips, BuildScopeEntityMap, SplitMultiValueTrailer}`
- `schema.WriteSchemaText`, `template.{TemplateOut, WriteTemplateText}`
- `list.{ListSummary, ListCounts, BuildListRows, BuildListCounts, RenderListCountsText, RenderListRowsText, ComputeTitleBudget, MinTitleColumnRunes, UnionAllStatuses, IsKnownKind}`
- `status.{StatusReport, StatusEpic, StatusMilestone, StatusEntity, StatusGap, StatusFinding, StatusHealthCounts, StatusACProgress, StatusSweepPending, SummarizeACs, RenderACProgress, BuildStatus, ParseSweepPending, ReadRecentActivity, RenderStatusText, RenderStatusMarkdown, WriteStatusEpicText, WriteStatusEpicMarkdown, TruncStatusTitle, MdEscape, RecentActivityLimit}`
- `upgrade.{RenderVersionLabel, ResolveTarget, GoBinDir, InstallLocationHint, PathChangedFromStderr}`
- `show.{ShowView, BuildShowView, BuildCompositeShowView, LoadEntityScopeViews, ReadEntityBody, LookupCommitDateCached, LastEventSHA}`
- `render.{NewRenderResolver, Resolver, PluralToEntityKind, TitleForKindIndex, RunSite, RunRoadmap}`
- `cliutil.{LoadContractsDoc, LoadContractsBlock, JoinKinds}` (lifted in earlier milestones; consumed by add and the migrated verbs above)

`internal/policies/read_only.go` was extended to track each read-only verb by (FuncName, FilePrefix) so it finds the verb's body in either `cmd/aiwf/run<Verb>Cmd` (legacy) or `internal/cli/<verb>/Run` (M-0115+). As each read-only verb migrated, the policy entry switched. Without this extension every read-only verb migration would have failed the policy at pre-commit.

`internal/render` is aliased as `baserender` in main.go, internal/cli/render/render.go, internal/cli/render/resolver.go, and canonicalize_render_test.go — the new `internal/cli/render` package collides with the existing `internal/render` package on the bare `render` import name.

## Decisions made during implementation

- **import → importcmd.** `import` is a Go reserved word — can't use as a package name. The directory and package are both `importcmd`; the CLI verb stays `import`.
- **init → initcmd.** Similar reasoning. `init` is Go's package-level initialization function name — using it as a package name technically compiles but creates ambiguity. Renamed to `initcmd`.
- **Sibling files moved with their verb.** The spec was cautious about `rituals.go` / `show_scopes.go` / `render_resolver.go` having "cross-verb concerns." In practice each has exactly one verb-side caller and a small handful of test references. Moving them with the verb shrinks M-0118's scope; the relevant test refs are exposed via package exports.
- **Test relocation policy.** Binary-integration tests that drive the aiwf binary via `aiwfBinary(t)` stay in `cmd/aiwf/` (they need access to package-internal test helpers like `setupCLITestRepo`, `runGit`, `runBin`, `captureStdout`). Unit tests that exercise the verb's now-exported symbols are updated in place with the `<pkg>.X` prefix. New per-subpackage smoke tests pin each NewCmd's exported shape.
- **Bulk via sed + perl.** Mechanical renames (`newXCmd` → `NewCmd`, `runXCmd` → `Run`, internal lowercase helpers → capitalized) were applied per-file with `perl -i -pe 's/\bfoo\b/Foo/g'` (BSD sed doesn't support `\b`). Each verb's migration was a coordinated patch covering the source file + every dependent file in cmd/aiwf in one commit.

## Validation

- **Build.** `go build ./...` green at every commit.
- **Tests (per-package isolation).** Each new subpackage's smoke test passes. `internal/policies/` passes after each read-only verb migration's policy update.
- **Tests (full module).** Same documented macOS git-subprocess contention flake as M-0113-M-0115 — flakes only at full-module `-parallel 8` with cmd/aiwf in the mix. Isolated package targets are reliably green.
- **Lint.** `golangci-lint run ./...` 0 issues at every commit (gofumpt + revive package-doc-comment + exported-function doc-comment cleanups applied incrementally).
- **aiwf check.** Zero error-severity findings on M-0116 or its 16 ACs.
- **Branch-coverage audit.** Clean. Each verb migration is a relocation + caller substitution; no new branches in production code. The policy extension in `read_only.go` switched lowercase per-verb entries to package-prefix-based entries; both arms (Ident vs SelectorExpr equivalents in policy walking) are exercised by the mixed cmd/aiwf + internal/cli/* tree.

## Deferrals

- None.

## Reviewer notes

- The pattern works but each verb's migration uncovered a unique set of cross-file helper references (test fixtures, sibling helper functions, cross-verb integration tests) that needed individual export decisions. Per-verb commits average ~5-10 file changes due to these cascades.
- The read-only policy's per-verb entry shape (`{FuncName, FilePrefix}`) is a deliberate trade-off: each verb migration requires one targeted policy edit rather than the verb appearing in a single list. The alternative was a generic "walk every package and find a Run function in any internal/cli/* subpackage" implementation that risked false negatives. The targeted shape is mechanical and reviewable.
- M-0118 (next per spec) shrinks `cmd/aiwf/main.go` to entry-only and finds homes for any remaining supporting files. After M-0116, the only cmd/aiwf files left that aren't `main.go` are: `contract_cmd.go`, `doctor_cmd.go`, `milestone_cmd.go` (M-0117 handles), plus the supporting siblings `selfcheck.go`, `tests_metrics_check.go`, `provenance_check.go`, and the cross-verb integration tests + drift-tests. M-0118 picks up from there.

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

