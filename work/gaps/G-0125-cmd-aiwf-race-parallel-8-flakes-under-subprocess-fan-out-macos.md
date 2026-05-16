---
id: G-0125
title: cmd/aiwf -race -parallel 8 flakes under subprocess fan-out (macOS)
status: open
discovered_in: M-0092
---

## What's missing

`go test -race -parallel 8 -count=1 ./cmd/aiwf/` is not 10-run reliable on macOS dev hosts after the M-0092 conversion. Three 10-run loops on a 20-core Apple Silicon host (warm Go build cache, clean process state at start) produced 7/10, 8/10, and 7/10 passes respectively — a 20–30% flake rate. M-0091's `./internal/...` runs the same `-race -parallel 8` workload to clean 10/10. The asymmetry is the workload, not the cap.

Two flake modes were observed:

1. **`os/exec` timeout-hang** (most common, 2–3 occurrences per 10-run loop): one or more tests deadlock inside `gitops.StagedPaths` → `os/exec.(*Cmd).Start.func2` (the goroutine that copies stdout/stderr from a child process). Go's `-timeout` panics the test driver after 5 minutes; the leaked child-process io-copy goroutines stay alive until SIGKILL'd. The hung test is never the same one — runs identified `TestRender_ProvenanceTabShowsAuthorizeScope`, `TestArchive_PerKindStorageLayout`, `TestRetitle*`, `TestRender_WellFormed*`, `TestAddMilestoneDependsOn*` (different tests across runs). All hang inside `gitops.StagedPaths`'s `git diff --staged` subprocess; none hang on the same line of test code.
2. **Repo-lock collision** (1 occurrence in 30 runs): `aiwf add: another aiwf process is running on this repo; retry in a moment` from `acquireRepoLock`. Per-test repos via `t.TempDir()` should prevent this — the path that produced it is not yet identified.

## Why it matters

CI is the chokepoint per CLAUDE.md "validation is the chokepoint." CI runs on Linux hosts with different fd-table / process-table limits and a different scheduler; the macOS-host flake rate may or may not reproduce in CI. As of this gap, CI behavior under the M-0092 changes is **unknown** — the M-0092 branch has not been pushed pending push approval. The gap exists to:

- Document the macOS-host flake characteristic so the next maintainer who sees `-race` red on push knows where to look.
- Cover the AC-4 deferral on M-0092 (which required 10/10 clean) with a concrete remediation arc.
- Surface the structural question: `aiwf` verbs shell out to `git` heavily, and the in-process verb dispatcher tests inherit that fan-out. Under `-parallel 8` with race instrumentation, the OS-level subprocess pressure is the dominant cost — at some scale it stops being a Go-level concurrency story and becomes an OS-resource one.

## Possible remediations

Not yet decided; the right answer probably depends on CI data and a profiling pass.

1. **Token-bucket around verb-level git invocations.** A package-level semaphore in `internal/verb/apply.go` (or wherever git subprocesses fan out from) limits concurrent git calls to a smaller number than `GOMAXPROCS`. Cheap to implement; trades a small wall-time hit for reliability. Needs care: the bucket must not deadlock with the existing repo lock.

2. **Lower the per-package cap for `cmd/aiwf` specifically.** Split the Makefile/CI test invocation into two: `internal/...` keeps `-parallel 8`, `cmd/aiwf/` runs at `-parallel 4` (or 2). Revisits M-0091's AC-1 "one cap across host shapes" decision — a kernel-policy adjustment, not a local fix. The single-cap argument was "one rule to police, not three"; the counter-argument the data now hands us is "the rule that fits internal/* is wrong for cmd/aiwf/."

3. **Refactor specific test patterns.** `gitops.StagedPaths` is called inside almost every mutating verb via `verb.Apply`. Tests that exercise verb paths (most of cmd/aiwf/) inherit the cost. A serial skip-list addition would have to cover most of `cmd/aiwf/` to avoid the deadlock — not a tractable fix.

4. **Accept macOS as a degraded host.** Document the flake characteristic in CLAUDE.md, prescribe local cleanup workflow (`pkill -f aiwf.test`) between iterations, rely on CI for the authoritative reliability signal. Cheapest; defers the actual fix.

## Code references

- M-0092 spec — the AC-4 deferral lives here pending this gap's resolution: `work/epics/E-0025-test-suite-parallelism-and-fixture-sharing-pass-closes-g-0097/M-0092-roll-out-testmain-t-parallel-no-ldflags-dedup-to-cmd-aiwf.md`.
- `internal/verb/apply.go::Apply` — the verb-dispatch entry point that shells out to `gitops.StagedPaths`, `gitops.Add`, `gitops.Commit` in sequence; the fan-out chokepoint.
- `internal/gitops/gitops.go` — every git subprocess invocation lives here; a token-bucket would land at this layer.
- `cmd/aiwf/setup_test.go` — the M-0092 skip-list comment; if remediation (2) lands, the cap selection lives in the Makefile/workflow and this comment may want an update.
- `Makefile` and `.github/workflows/{go,flake-hunt}.yml` — the `-parallel 8` cap surfaces M-0091/AC-1 pinned via `internal/policies/race_parallel_cap.go`. Any per-package cap split would also need that policy rule's update.

## Observed conditions on this host

- macOS, 20-core Apple Silicon, Go 1.26.1 stdlib.
- `ulimit -n`: default 1024 (file descriptor cap). One git-subprocess test consumes ~5–10 fds; 8 parallel tests × ~10 fds × ~1 git invocation each = within the cap, but the OS-level allocator's pressure under concurrent fork-exec is real.
- Reproducer: `cd <worktree> && for i in {1..10}; do go test -race -parallel 8 -count=1 -timeout=300s ./cmd/aiwf/; pkill -f "aiwf.test "; sleep 1; done`. The flake usually surfaces in iteration 2–4 if the machine has prior `aiwf.test` zombies; less often on a freshly cleaned machine.

## CI signal

Pending — feature branch not pushed yet. The first signal of whether this flake class hits Linux is the CI run on the wrap-merge of M-0092. If CI is reliably green, remediation can wait until a real consumer reports macOS-host friction. If CI flakes too, this gap escalates to blocking.
