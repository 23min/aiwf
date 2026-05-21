---
id: G-0151
title: Batch git subprocess fan-out in status worktrees and show scopes
status: open
prior_ids:
    - G-0148
    - G-0149
---
## Status

- **fsm-history-consistent slice: CLOSED in M-0137.** The retrofit lands the two batched helpers (`gitops.BulkRevwalk`, `gitops.BlobReader`) and the silent-swallow fix as a coupled deliverable. Per-entity `git log --follow` + per-(commit, parent) `git show` are gone from the rule's hot path; per-blob read failures surface as `history-walk-error` findings instead of being silently swallowed. See M-0137's commits for the full diff; the AC-7 perf test records the post-retrofit runtime on a synthetic 50-entity fixture (~122ms, 80× under the 10s regression budget).
- **`aiwf status` worktree views: OPEN.** Site #1 below — `internal/cli/status/worktrees.go:102`'s per-worktree subprocess loop unchanged.
- **`aiwf show <entity>` scope views: OPEN.** Site #2 below — `internal/cli/show/scopes.go:87`'s per-scope `git log` / `git show` loop unchanged.

The two open slices are perf-only — no kernel-chokepoint correctness angle (each verb's output is user-facing latency, not a guarantee surface). They can wait until consumer-tree size or worktree-count pressure makes the latency user-visible.

## What's missing

Two user-facing kernel verbs do unnecessary git subprocess fan-out today. (A third site — M-0130's `fsm-history-consistent` check rule — was originally listed here too and is now addressed by M-0137; see Status above.) The pattern is the same: a per-item loop calls one or more `git log` / `git show` subprocesses inside the inner block, instead of either a single batched walk or a long-lived `git cat-file --batch` pump.

The two remaining sites:

1. **`aiwf status` worktree views** — `internal/cli/status/worktrees.go:102` loops over worktrees and sequentially calls `worktreeHeadTime` (`:968`), `worktreeIsDirty` (`:987`), `branchFirstAheadCommitTime` (`:1005`), and `branchLastEntityCommitTime` (`:1023`). That's 3–4 `git` subprocesses per worktree, executed serially. On a 15-worktree workspace ~60 forks block the interactive output.

2. **`aiwf show <entity>` scope views** — `internal/cli/show/scopes.go:87` calls `cliutil.LoadEntityScopes` once per scope-entity (one `git log` per entity at `cliutil/scopes.go:144`), then `:97` calls `LookupCommitDateCached` (`:176`) once per scope SHA (`git show -s --format=%aI`). The cache helps when SHAs repeat across scopes; it doesn't when they don't.

## Why it matters

- `aiwf status` is the verb operators reach for many times a day. Latency there directly shapes the perceived speed of the whole tool. The cost scales with worktree count, which power users grow over time.
- `aiwf show` is less hot, but used during planning sessions and Q&A loops with AI assistants. Slow show calls compound across a session.
- On macOS, subprocess fan-out becomes OS-resource-bound under load (see G-0125, archived/addressed), raising the cost of fan-out-heavy code paths beyond the Go-level concurrency story.

## Why this stayed one gap (now down to two slices)

The fixes share helpers. M-0137 landed the two batched primitives in `internal/gitops/`:

- `BulkRevwalk` — streams `(commit-sha, parent-sha, paths, trailers)` in one subprocess via `git log --all --name-status -M -m --pretty=...`.
- `BlobReader` — turns N short-lived `git show <commit>:<path>` calls into one long-lived `git cat-file --batch` subprocess.

Both helpers are now production-tested under the fsm-history-consistent rule (M-0137's AC-1 + AC-2 ship them with 100% coverage on their parsing helpers; AC-3 proves they handle the rule's load profile). The `status` and `show` retrofits are now small refactors on top of established helpers, not separate research efforts.

## Resolution paths (remaining slices)

- **Measure first.** Before writing the `status` / `show` optimizations, `time` the two verbs on this repo and on a synthetic large fixture (50 worktrees, 200 scopes). If the cost is invisible at realistic scales, leave the call sites alone.
- **Cheap win for `status` (small commit, low risk).** Parallelize the inner loop at `worktrees.go:102` with an `errgroup` capped at 8. Cuts wall time ~3–4× without touching the subprocess count. Lands independently of the helpers.
- **Full fix.** Retrofit `worktrees.go` and `show/scopes.go` to route through `gitops.BulkRevwalk` + `gitops.BlobReader` (or, where appropriate, a thinner specialization of the same shapes). Each retrofit is a small follow-up milestone.

## Closed slice retrospective: silent-swallow correctness was load-bearing

When this gap was filed, the `fsm-history-consistent` slice carried a load-bearing correctness issue alongside the perf concern. The M-0130 rule swallowed walker errors at `FSMHistoryConsistent:71-77` and `walkStatusChanges` fail-fasted on the first per-entity error. **Symptom in the M-0130 session (2026-05-20):** the same binary on the same content produced "4 errors" on one worktree and "0 fsm-history-consistent findings" on a sibling worktree during a heavily-concurrent merge phase. One transient subprocess failure (under exactly the load this gap targeted) collapsed all findings — invisibly. The operator sees a green check, pushes broken state, and never knows.

M-0137 closes this correctness issue. The retrofit:

1. **Replaced the swallow with a finding.** `FSMHistoryConsistent` emits a `fsm-history-consistent/history-walk-error` finding (severity `error`) per failed (entity, commit) read, naming the offending entity and the underlying git error.
2. **Continues past per-blob failures.** The batched walker accumulates partial observations and a per-blob error slice rather than fail-fast. Successful entities still produce findings; failed reads each produce one `history-walk-error` finding.
3. **Pinned the contract with a negative test.** `TestFSMHistoryConsistent_AC5_PartialFailure_PreservesGoodFindings` injects a per-entity blob-read failure via the `blobReader` dep seam and asserts both healthy entities' findings and the broken entity's `history-walk-error` finding emerge.

This aligns the rule with CLAUDE.md §*Engineering principles* — "Errors are findings, not parse failures." The silent-no-op was exactly the principle the kernel forbids, hidden under exactly the load this gap targeted.

## Class

Performance / scalability gap. Not blocking on current repo shape; becomes user-visible as repos grow. Audit performed 2026-05-20 via grep over all 31 non-test `exec.Command("git", ...)` sites in `internal/` and `cmd/`.

## Related

- **M-0130** — original `fsm-history-consistent` shipped with per-entity walk.
- **M-0137** — closed the fsm-history-consistent slice (batched walker + silent-swallow fix).
- **G-0125** (archived, addressed) — first surfaced the observation that aiwf's per-verb git fan-out is an OS-resource concern on macOS; this gap is the runtime-verb analog.
- `internal/cli/history/history.go:283, :515` — existing example of the single-walk shape that should be templated.
- `internal/gitops/revwalk.go` (M-0137/AC-1) — the bulk-revwalk helper now in production.
- `internal/gitops/catfile.go` (M-0137/AC-2) — the cat-file batch pump now in production.
