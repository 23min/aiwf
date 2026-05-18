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
      status: met
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
      status: met
      tdd_phase: done
    - id: AC-6
      title: cobra integration tests relocate to internal/cli/integration/
      status: met
      tdd_phase: done
    - id: AC-7
      title: captureStdout lifted to shared testutil; per-package duplicates forbidden
      status: met
      tdd_phase: done
    - id: AC-8
      title: JSON-envelope version-source drift policy added
      status: met
      tdd_phase: done
---
## Goal

Shrink [`cmd/aiwf/main.go`](../../../cmd/aiwf/main.go) to G-0107's target ~30-line entry-point shape and move the remaining cross-verb infrastructure (`newRootCmd`, version stamping, `printHelp`, and the 2 supporting files still under `cmd/aiwf/` — [`tests_metrics_check.go`](../../../cmd/aiwf/tests_metrics_check.go) and [`provenance_check.go`](../../../cmd/aiwf/provenance_check.go)) under `internal/cli/`. After this milestone, `cmd/aiwf/` contains `main.go` only (plus possibly `doc.go`); **G-0107 fully closed.**

## Context

The capstone milestone of G-0107 step 3. M-0115 and M-0116 moved verbs to per-verb subpackages and joint-moved three supporting files with them (`render_resolver.go` → `internal/cli/render/`, `show_scopes.go` → `internal/cli/show/`, `rituals.go` → `internal/cli/initcmd/`). M-0117 removes the multi-subcommand cmd-side residue (`contract`, `doctor`, `milestone`). M-0118 packages the remaining cross-verb infrastructure and shrinks `main.go` to the kubectl/helm/hugo-canonical shape: parse args, call `cli.Execute()`.

## Approach

1. **Cross-verb root assembly** → `internal/cli/root.go`. Move `newRootCmd`, version helpers (`resolvedVersion`), and `printHelp` content into the `cli` package. Export `cli.Execute(args []string) int`.
2. **Supporting files find their owning packages:**
   - `tests_metrics_check.go` → folded into `internal/cli/check/` (the verb subpackage) when the wrap revealed `internal/cli/cliutil` imports `internal/check`, so a direct move to `internal/check/` would have cycled. The file's own godoc already named the git-access discipline — co-locating with the check verb body was the natural home.
   - `provenance_check.go` → same destination, same rationale.
   - (`selfcheck.go` moved with `doctor` in M-0117, not here.)
3. **`main.go` final shape:** entry-only — 21 lines: package doc + `func main()` calling `cli.Execute(os.Args[1:])`.
4. **Integration tests** under `cmd/aiwf/integration*_test.go`, `binary_integration_test.go`, `envelope_schema_test.go`, etc. — relocated to `internal/cli/integration/`. Both cobra-driven (in-process) and binary-driven (subprocess) tests live there together; the helpers they share (CaptureStderr, RunGit, AiwfBinary, RunBin, BuildBinary, etc.) live in `internal/cli/cliutil/testutil/`.
5. **Lift `captureStdout` to a shared testutil location.** The pre-M-0118 codebase had two parallel copies (cmd/aiwf/helpers_test.go and internal/cli/initcmd/helpers_test.go) because `_test.go` files cannot cross package boundaries. Now one canonical implementation under `internal/cli/cliutil/testutil/` with `PolicyCaptureStdoutSingleton` as the drift chokepoint.
6. **Converge JSON-envelope version source on `version.Current().Version`.** Every verb's JSON envelope now reads through the canonical reader; `PolicyEnvelopeVersionSource` is the AST-walking chokepoint that prevents regression to a package-global `Version` reference.

## Acceptance criteria

<!-- ACs added at aiwfx-start-milestone via `aiwf add ac M-0118 --title "..."`. -->

## Work log

### AC-1 — internal/cli/root.go assembles root command and exports Execute

`internal/cli/root.go` is the new home for `NewRootCmd` (exported so policy tests + integration tests can call it), `Execute(args []string) int` (the in-process dispatcher, with the `AssertSupportedOS` preflight folded in), the `Version` package-global (ldflags target moved from `main.Version` to `internal/cli.Version`), `ResolvedVersion`, `printHelp`, and `newVersionCmd`. `internal/cli`'s init wires `doctor.Dispatcher = Execute` so `--self-check` still reaches the dispatcher without `internal/cli/doctor` having to know about `cmd/aiwf`. cmd/aiwf/main.go kept temporary `run`/`newRootCmd` shims so cmd-side tests would compile until AC-6 relocated them.

Commit `3184892c` · 4 tests pass (Execute_Version, Execute_VersionVerb, Execute_Help, NewRootCmd_HasExpectedVerbs, ResolvedVersion_FallsBackToBuildInfo)

### AC-2 — check verb body moves to internal/cli/check/

`newCheckCmd`, `runCheckCmd`, `runCheckShapeOnly` move out of `cmd/aiwf/main.go` into `internal/cli/check/check.go` as `NewCmd`, `Run`, `runShapeOnly`. The check verb's JSON envelope now reads `version.Current().Version` (per item 6 in the spec — convergence with every other migrated verb). The byte-identical-baseline goldens were updated from `"dev"` to `"(devel)"` to match. `internal/policies/read_only.go`'s entry switched from `{FuncName: "runCheckCmd", FilePrefix: "cmd/aiwf/"}` to `{FuncName: "Run", FilePrefix: "internal/cli/check/"}`. cmd/aiwf/main.go shrank from 525 to 215 lines on this commit.

Commit `6c6d4d80` · 2 tests pass (NewCmd_FlagShape, Run_BadFormat); upstream byte-identical-baseline tests stay green

### AC-3 — tests_metrics_check moves to internal/cli/check/

`runTestsMetricsCheck` and `hasTestsTrailer` move from `cmd/aiwf/` to `internal/cli/check/` (the verb subpackage created in AC-2). Co-located with the check verb body that composes them; the file's own godoc ("lives here rather than in package `check` because the rule requires git access") is preserved in place. The original choice of `internal/check/` was abandoned because `internal/cli/cliutil` already imports `internal/check`, and the helper imports cliutil — moving to `internal/check/` would have created a cycle.

Commit `388cfe43` · 2 tests pass (RequireFalseIsNoop, EmptyRepoIsNoop)

### AC-4 — provenance_check moves to internal/cli/check/

`runProvenanceCheck` and its three helpers (`resolveUntrailedRange`, `readUntrailedCommits`, `parseUntrailedCommits`) move alongside AC-3's tests_metrics_check. Same destination, same rationale. The three helpers also tested by `show_scopes_unit_test.go` are exported as `ResolveUntrailedRange`/`ReadUntrailedCommits`/`ParseUntrailedCommits`; once AC-6 relocated `show_scopes_unit_test.go` to `internal/cli/integration/`, the cross-package alias became unnecessary but harmless. `readProvenanceCommits` stays unexported (single internal caller).

Commit `108f3883` · 2 tests pass (RunProvenanceCheck_EmptyRepoIsNoop, ParseUntrailedCommits_EmptyInput)

### AC-5 — cmd/aiwf/main.go shrunk to entry-only shape (function main only)

`cmd/aiwf/main.go` is now 21 lines: package godoc + a single `func main()` that calls `os.Exit(cli.Execute(os.Args[1:]))`. The `run`/`newRootCmd` compat shims left in AC-1 are deleted — AC-6 relocated every cmd/aiwf-side test that depended on them. **G-0107 fully closed at this AC**: cmd/aiwf/ contains main.go only.

Commit `9893ee16` (combined with AC-6) · structural assertion: `cmd/aiwf/` has exactly one .go file (`ls cmd/aiwf/*.go | wc -l == 1`)

### AC-6 — cobra integration tests relocate to internal/cli/integration/

Every test file at cmd/aiwf/ (~80 files) moves to a new `internal/cli/integration/` package. Both cobra-driven (in-process via `cli.Execute`) and binary-driven (subprocess via the built binary) tests live in the same package; the original "split between cobra→integration, binary→cmd/aiwf" plan was abandoned mid-migration when it produced unmanageable helper-fragmentation (`assertWellFormed`, `htmlMain`, `setupGitRepoWithUpstream` were needed by both styles). Net result: cmd/aiwf is truly empty of tests. Shared helpers were promoted to `internal/cli/cliutil/testutil/`: `CaptureStderr`, `CaptureRun`, `RunGit`, `RunBin`, `RunBinStdin`, `AiwfBinary` (sync.Once binary build with G-0128 codesigning), `BuildBinary`, `RunBinary`, `RunBinaryAt`, `MustExec`, `ExitedWithCode`, `ExtractRow`, `ReadFileT`, `SkipIfShortOrUnsupported`, `SetupGitRepoWithUpstream`. Package-local cobra helpers (`setupCLITestRepo`, `mustRun`, `osExec`) live in `internal/cli/integration/helpers_test.go`. The Makefile gained `-parallel 8` on `make test` (matching what `make test-race` already had) — with ~80 tests now in one package, default-GOMAXPROCS parallelism overwhelms macOS git/codesign subprocess budgets.

Commit `9893ee16` (combined with AC-5) · 85 files changed, 1689 insertions / 1624 deletions; full test suite green; testdata directory moved with the tests

### AC-7 — captureStdout lifted to shared testutil; per-package duplicates forbidden

`captureStdout` was duplicated at `cmd/aiwf/helpers_test.go` and `internal/cli/initcmd/helpers_test.go` because `_test.go` files cannot cross package boundaries. The canonical implementation lives at `internal/cli/cliutil/testutil/capture.go` as `testutil.CaptureStdout` (a non-`_test.go` file so other packages can import it; production code is forbidden from importing testutil by convention). `PolicyCaptureStdoutSingleton` (`internal/policies/capture_stdout_singleton.go`) is the AST-walking chokepoint that fails CI if any future `_test.go` redefines the helper outside testutil. 20 test files updated to call `testutil.CaptureStdout`.

Commit `18a26dc3` · policy fires on the pre-migration duplicates (verified red), passes after the lift (green); make test green on the call-site rewrite

### AC-8 — JSON-envelope version-source drift policy added

`PolicyEnvelopeVersionSource` (`internal/policies/envelope_version_source.go`) asserts every production `render.Envelope` initialization sources its `Version:` field from `version.Current().Version`. The chokepoint prevents the regression class M-0118 item 6 closed at AC-2: a verb's JSON envelope referencing a package-global `Version` identifier produces divergence between verbs on unstamped local builds (`"dev"` vs `"(devel)"`). The matcher walks CompositeLits whose type ends in `Envelope`, then checks each `Version:` key — allowed: a function-call selector chain returning a value whose `.Version` field is read; forbidden: bare Ident or non-call SelectorExpr (package-global access). Scope: production `.go` files only (test fixtures legitimately set `"0.1.0"` or `"dev"` for envelope round-trip tests).

Commit `500e9c64` · synthetic-input tests pin both the fire branch and the canonical-pattern accept branch; make test green

## Decisions made during implementation

- **Reordered AC implementation** (`AC-3 → AC-4 → AC-2 → AC-1 → AC-7 → AC-8 → AC-6+AC-5`). The spec listed them 1–8 but mid-flight the dependency tree revealed that AC-2 (check verb body move) depends on AC-3+AC-4 (its helpers), and AC-1 (root.go assembly) is cleanest after AC-2 has cleared the only inline verb. AC-7 (captureStdout shared) was completed before AC-6 because AC-6's mass move would otherwise scatter duplicates. AC-5 and AC-6 collapsed into one commit because AC-5's "main.go entry-only" condition is achieved by AC-6's relocation work.

- **Check-rule home: internal/cli/check/, not internal/check/.** Original spec offered both. The user picked `internal/check/`. Mid-implementation it surfaced that `internal/cli/cliutil` already imports `internal/check`, and the moved helper imports `cliutil` — cycle. Re-conferred with the user; `internal/cli/check/` won on the file's own godoc rationale (git-access rules belong with the verb, pure-tree rules stay in `internal/check`).

- **Binary tests merged into the same package as cobra tests, not kept at cmd/aiwf/.** Original plan was "split: cobra → integration, binary → cmd/aiwf". Mid-migration, the helper-fragmentation cost became visible (`assertWellFormed`/`htmlMain`/`setupGitRepoWithUpstream` are needed by both styles). Re-conferred and pivoted to "move everything; cmd/aiwf has only main.go". The G-0107 target reads naturally with this choice.

- **JSON-envelope Version source = `version.Current().Version`** for every verb, including the moved check verb. Trade-off accepted: ldflags-stamped `make install` builds report buildinfo's `(devel)` in the JSON envelope (vs the old `<branch>@<sha>`); tagged `go install <pkg>@v0.1.0` builds correctly report the tag both before and after. The convergence wins because the alternative was two parallel version sources for two parallel sets of verbs.

- **Test runtime cap landed in the Makefile, not in code.** With ~80 tests now in `internal/cli/integration/`, default GOMAXPROCS-bounded parallelism overwhelms macOS git/codesign subprocess budgets. The `-parallel 8` cap that previously only applied to `make test-race` now applies to `make test` too. (The `race-parallel-cap` policy already pinned the test-race entry; a future follow-up gap could broaden the policy to cover the plain test entry — out of scope for this milestone.)

## Validation

- `make test`: **all green** (`-parallel 8`, ~80 tests under `internal/cli/integration/`, ~7 minutes wall time)
- `aiwf check`: **0 errors**, 20 warnings (all pre-existing advisory: archive-sweep-pending, terminal-entity-not-archived for unrelated entities, provenance-untrailered-scope-undefined because the branch isn't pushed yet, entity-body-empty for some pre-migration entities)
- `cmd/aiwf/` directory: 1 .go file (main.go), 21 lines. **G-0107 closed.**
- All 8 ACs at `status: met`, `tdd_phase: done`. Kernel `acs-tdd-audit` requirement satisfied.
- 7 commits comprise the milestone delivery: AC-3 (`388cfe43`), AC-4 (`108f3883`), AC-2 (`6c6d4d80`), AC-1 (`3184892c`), AC-7 (`18a26dc3`), AC-8 (`500e9c64`), AC-6+AC-5 (`9893ee16`).
- The G-0128 macOS codesigning fix carried forward into `testutil.AiwfBinary` and `testutil.BuildBinary`; the post-rebase test suite confirms the syspolicyd crash class is gone.

## Deferrals

- **`PolicyRaceParallelCap` could broaden to cover `make test` (currently only covers `make test-race`).** The Makefile's `test` target now also carries `-parallel 8` for the same macOS subprocess reason, but the policy's regex only scans race-mode invocations. Filed mentally; not opened as a gap because the bug class (drift from 8 → some other value) is identical whether or not the policy widens. If a future drift bites, raise the policy then.

- **Production code import of `internal/cli/cliutil/testutil/` is not yet policy-enforced.** The package's godoc says "Production code must not import this package" but no AST policy fires on a violation yet. Filed mentally; the package is small and only imported by tests today.

## Reviewer notes

- The biggest single commit in the milestone (`9893ee16`, AC-5+AC-6) is 85 files / 1689 insertions / 1624 deletions. Most of that is mechanical: `package main` → `package integration`, `run(...)` → `cli.Execute(...)`, `<localHelper>(...)` → `testutil.<exportedHelper>(...)`. The non-mechanical work was scope-deciding (cobra vs binary lived together in the end) and helper consolidation. A `git diff -M` should render most of the changes as renames; if review is via PR, request "show renames" so the file-level deltas read clearly.

- The rebase mid-milestone (epic branch onto main after G-0128 landed) orphaned the authorize-by SHA in all 14 of the trailer-carrying commits. The fix was `git filter-branch --msg-filter sed` — rewrote `bb63124e…` to the new `7ddfbe23644a9934e6289c79684c51313496a37c` in every implementation commit. The trailer-rewrite class is a known hazard whenever a feature branch absorbs main; the cleanup is mechanical.

- macOS Sonoma 14.8.x syspolicyd crash on unsigned Mach-O headers (G-0128) was the underlying cause of every parallel-test timeout I hit during the first half of the milestone. The fix landed on main while M-0118 was in flight; rebasing pulled it in. After the rebase the same tests that were timing out at 600s pass in 70s — the codesigning fix is doing real work.

- AC-7's `PolicyCaptureStdoutSingleton` and AC-8's `PolicyEnvelopeVersionSource` are the two new drift policies this milestone adds. Both are AST-based and run as plain `go test ./internal/policies/...`. They'll fire on the next regression that tries to redefine `captureStdout` per-package or reintroduce a package-global `Version` JSON envelope source.

- Two follow-up policy items are filed under "Deferrals" above. Neither is urgent enough to block wrap; both are noted so a future review pass can pick them up.
