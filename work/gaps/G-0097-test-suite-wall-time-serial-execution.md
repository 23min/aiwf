---
id: G-0097
title: Test-suite wall time dominated by serial execution and per-test setup
status: open
---
## Problem

The Go test suite is slow enough to be felt as friction during inner-loop iteration and adds measurable cost to every CI run. The slowness is structural, not algorithmic — the tests do the right work, just serially, and re-do per-test setup that could be shared.

Static inventory across the 138 test files in this repo (excluding worktrees):

- **No parallelism.** Only 4 of 138 files call `t.Parallel()`. Most tests use `t.TempDir()` and a fresh `gitops.Init`, so they're already filesystem-isolated — parallelism is mechanically safe.
- **High git-subprocess fan-out per test.** `internal/verb/verb_test.go` alone has 43 tests × ≥3 git subprocess calls per `verb.Apply` (StagedPaths, Add, Commit, sometimes Mv/StashStaged/StashPop). 309 sites in `cmd/aiwf/` shell out to a built `aiwf` binary; each is a fresh process startup.
- **Repeated binary builds.** `cmd/aiwf/binary_integration_test.go` builds the binary 7 times — its own header comment notes "~3–5s on a warm cache." Other test files use `sync.Once` for the same purpose.
- **Repeated live-repo `tree.Load`.** `internal/policies/this_repo_tree_clean_test.go`, `…_drift_check_clean_test.go`, the M-080 spec tests, and the AC sweeps each independently walk the live repo's planning tree (212 files, 1.4 MB). Same data, parsed N times per `go test ./internal/policies`.
- **In-memory fixture re-creation.** `internal/htmlrender/htmlrender_test.go::writeFixtureTree` creates a fresh tempdir + 5 mkdirs + 7 file writes per call, called from at least three tests.
- **Multi-clone topology rebuilt per test.** `cmd/aiwf/integration_g37_test.go` runs 11 tests; each builds a bare-origin + 1–3 clones + `aiwf init` from scratch. None of the topology is shared.

## Evidence

A spike on the worktree branch `spike/test-parallel` applied the minimum pattern (`TestMain` seeds GIT identity once, all top-level tests `t.Parallel()`, redundant `t.Setenv` removed) to every test file under `internal/verb/` — 234 tests across 20 files, ~270 inserted lines.

Measured on a 20-core macOS host (warm Go build cache, no-op edits between runs):

| | Baseline (serial) | After spike | Speedup |
|---|---|---|---|
| `go test ./internal/verb/` | 27.3 / 27.3 / 29.0 s | 5.9 / 6.6 / 6.7 / 7.0 / 7.2 s | ~4.2× |
| `go test -race -parallel 8 ./internal/verb/` | 31.3 s | 12.6 / 13.0 / 13.5 s | ~2.4× |

A secondary finding from the spike: `go test -race` at default `-parallel=GOMAXPROCS` (=20 here) flakes ~50% of runs with a `git add: signal: segmentation fault` followed by a 90s test-timeout. The signal is not a data race (no `WARNING: DATA RACE` in any captured output) — it's macOS resource pressure when 20 race-instrumented goroutines each fork ~10 git subprocesses concurrently. Capping `-parallel` (8 or below) is reliable; the underlying segfault is environmental.

## Why it matters

1. **Inner-loop friction.** `make test` taking minutes instead of seconds discourages running it between edits, which lets regressions land further from the change that introduced them.
2. **CI minutes scale linearly.** The `test` job in `.github/workflows/go.yml` runs on every push and every PR. The 2–3× headroom across the suite (extrapolating from the spike on one package) translates directly to CI seconds.
3. **The structural pattern will keep recurring.** New test files written today copy the existing serial-with-per-test-setup shape. Without a fix that touches the kernel test-discipline conventions in CLAUDE.md, the slowdown grows monotonically as the suite grows.

## Discovered

While auditing the test suite for inefficiencies in conversation with the user (no specific failing test triggered this — the friction was diffuse). The spike on `spike/test-parallel` confirmed the headroom is real.

## Fix shape

Treat as a multi-milestone epic. Apply the `TestMain` + `t.Parallel` pattern across the rest of `internal/*` packages (the spike's pattern generalizes mechanically); deduplicate the binary builds (`sync.Once`) and the live-repo `tree.Load` (memoize once per package); cap `-race` parallelism in CI to avoid the saturation flake. Defer the fixture-snapshot work for `htmlrender` and the multi-clone setup for `integration_g37` to a second wave if the first pass doesn't reach the wall-time target.

## References

- [`CLAUDE.md`](../../CLAUDE.md) — *Race detector on every CI run* and the surrounding test-discipline rules; constrains the fix (race must remain on, parallelism cap is the lever).
- [`.github/workflows/go.yml`](../../.github/workflows/go.yml) — CI test invocation that pays the per-test serial cost.
- [`Makefile`](../../Makefile) — `test` and `test-race` targets that need the same `-parallel` cap as CI.
- `internal/verb/verb_test.go`, `internal/verb/apply_test.go` etc. — pattern source for the spike; representative of the verb-suite shape.
- `cmd/aiwf/binary_integration_test.go::buildBinary` — needs `sync.Once`.
- `cmd/aiwf/integration_test.go::aiwfBinary` — already does `sync.Once`; the precedent.
- `internal/policies/this_repo_tree_clean_test.go`, `internal/policies/this_repo_drift_check_clean_test.go`, `internal/policies/m080_test.go::loadM080Spec` — shared `tree.Load` candidates.
- Spike branch: `spike/test-parallel` (worktree at `/tmp/aiwf-test-spike`).
