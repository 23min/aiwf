---
id: M-0215
title: Profile aiwf check and the policies suite to a per-rule wall-time baseline
status: in_progress
parent: E-0053
tdd: none
acs:
    - id: AC-1
      title: Record aiwf check CPU profile and git-subprocess attribution
      status: met
    - id: AC-2
      title: Record policies-suite per-test timing ranking floor-gating tests
      status: met
---
## Goal

Establish a per-rule wall-time baseline for `aiwf check` and the
`internal/policies` test suite, so every later optimization in this epic is
measured against a recorded before/after rather than guessed.

Deliverable: a CPU profile (pprof) and subprocess attribution of one full
`aiwf check` over a representative tree, plus a per-test timing snapshot of
the policies suite, recorded in this milestone's validation. Confirms the
diagnosis (one check spawns ~895 git subprocesses, 683 of them
`git merge-base --is-ancestor` from the orphaned-AI-commit walk) with a
clean second-by-second budget — the strace count is direction; the profile
is the budget.

## Notes

Profile before optimizing. The profiling harness
(`internal/cli/check/zz_m0215_baseline_test.go`, `BenchmarkAiwfCheckBaseline`)
is kept, not reverted: M-0216 reuses it to measure the before/after delta. It
is `-short`-skipped and is a `_test.go` benchmark, so it never runs in the
normal `go test` / CI path.

### AC-1 — Record aiwf check CPU profile and git-subprocess attribution

A CPU profile of one full `aiwf check` over the live kernel tree, plus the
git-subprocess attribution, are recorded in §Validation. The headline: the
check is **subprocess-wait bound, not CPU bound** — there is no in-process Go
hot path to optimize; the fix is purely to spawn fewer git subprocesses.

### AC-2 — Record policies-suite per-test timing ranking floor-gating tests

A per-test wall-time snapshot of the `internal/policies` suite is recorded in
§Validation, ranking the heavyweight tests that gate the suite's parallel
wall-clock.

## Validation

Measured on the kernel's own repo (659 entities), 2026-06-30, with a
worktree-built binary. The profiling harness is
`BenchmarkAiwfCheckBaseline` in `internal/cli/check/`.

### aiwf check — CPU profile (AC-1)

One full `aiwf check` over the live tree:

```
Duration: 79.53s, Total samples = 4.22s (5.31%)
```

**4.22s of CPU across 79.5s wall = 5.3% utilization.** The process is idle
~94.7% of the time, blocked in `waitid`/`Syscall6` on git subprocesses. The
top in-process samples are `Syscall6` (30%), `rtsigprocmask`/`futex`/
`findRunnable` (runtime scheduling around the blocking, ~20%), and ~1–2s of
yaml frontmatter parsing across 659 entity files. **There is no in-process Go
hot path** — no O(n²) loop, no slow algorithm. The 79s is serial
subprocess-wait.

In one line: `aiwf check` is **subprocess-wait bound, not CPU bound** — the
fix is to spawn fewer git subprocesses (M-0216), not to make Go code faster.

### aiwf check — git-subprocess attribution (AC-1)

One check spawns **895 git subprocesses** (strace execve count):

| git subcommand | count |
|---|---|
| `merge-base` (`--is-ancestor`) | 683 |
| `rev-parse` | 86 |
| `ls-tree` | 58 |
| `rev-list` | 45 |
| `reflog` | 9 |
| `log` | 8 |
| `for-each-ref` | 4 |
| other (`diff`, `config`) | 2 |

The 683 `merge-base` calls come from `WalkOrphanedAICommits`
(`internal/check/reflog_walk.go`), which calls `git merge-base --is-ancestor`
once per consecutive reflog-entry pair across every ritual branch —
O(reflog entries × branches). The check runs ~6 independent git-history
passes that never share a loaded history.

### internal/policies suite — per-test timing (AC-2)

Suite wall-clock ~9s, parallel-bound (`-parallel 8`) by a handful of
heavyweight tests:

| test | wall | why |
|---|---|---|
| `TestM0162_AC2_BuildTagExclusion` | 4.7s | `go build ./cmd/aiwf` + `go tool nm` |
| `TestM0147_AC3_GlobalRuleExercised` | 4.1s | source-tree walk for finding codes |
| `TestM0146_ScopeReachMachinery` | 4.0s | git-fixture scope-reachability |
| `TestGolangciConfigRulesFire` | 2.6s | real `golangci-lint` against fixtures |
| ~30 `promote`/`authorize`/`cancel` subtests | 1.5–2.2s ea | each builds its own git repo |

No single 82s outlier remains (that was `TestM080_AC6`, fixed in `G-0320`);
the residual floor is the aggregate of compile/lint/git-fixture tests
(tracked as `G-0321`, milestone M-0218).

## Work log

### AC-1 — CPU profile + subprocess attribution

Captured via `BenchmarkAiwfCheckBaseline` (`-cpuprofile`) + `strace -f -e
trace=execve`. Result recorded in §Validation: 5.3% CPU utilization →
subprocess-wait bound; 895 spawns, 683 `merge-base`. Evidence:
`internal/policies/m0215_baseline_test.go`.

### AC-2 — policies-suite per-test timing

Captured via `go test -v` per-test timing. Ranking recorded in §Validation.
Evidence: `internal/policies/m0215_baseline_test.go`.

## Reviewer notes

- The baseline settles the epic's strategy: M-0216's subprocess-fan-out
  collapse (the in-memory DAG) attacks the dominant cost; there is no Go
  algorithm to optimize, which the 5.3%-CPU profile proves.
- The profiling benchmark is deliberately retained as durable measurement
  infrastructure for M-0216's before/after, not reverted scaffolding. It is
  `-short`-skipped and never runs in the normal test path.
- The evidence test is a structural assertion on §Validation (the doc-shaped
  AC pattern), scoped to the section, per the repo's substring-vs-structural
  rule.

