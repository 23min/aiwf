---
id: G-0148
title: Batch git subprocess fan-out in status worktrees and show scopes
status: open
---
## What's missing

Two user-facing kernel verbs do unnecessary git subprocess fan-out today, and a third (the planned M-0130 check rule) will join them. The pattern is the same in all three: a per-item loop calls one or more `git log` / `git show` subprocesses inside the inner block, instead of either a single batched walk or a long-lived `git cat-file --batch` pump.

The three sites:

1. **`aiwf status` worktree views** — `internal/cli/status/worktrees.go:102` loops over worktrees and sequentially calls `worktreeHeadTime` (`:968`), `worktreeIsDirty` (`:987`), `branchFirstAheadCommitTime` (`:1005`), and `branchLastEntityCommitTime` (`:1023`). That's 3–4 `git` subprocesses per worktree, executed serially. On a 15-worktree workspace ~60 forks block the interactive output.

2. **`aiwf show <entity>` scope views** — `internal/cli/show/scopes.go:87` calls `cliutil.LoadEntityScopes` once per scope-entity (one `git log` per entity at `cliutil/scopes.go:144`), then `:97` calls `LookupCommitDateCached` (`:176`) once per scope SHA (`git show -s --format=%aI`). The cache helps when SHAs repeat across scopes; it doesn't when they don't.

3. **Planned M-0130 check rule** — proposed approach is `git log --follow` per entity + `git show <parent>:<path>` per status-change commit. On this repo's 331 entities × ~handful of status commits each, that's ~3,000 fork/execs in a pre-push hook.

## Why it matters

- `aiwf status` is the verb operators reach for many times a day. Latency there directly shapes the perceived speed of the whole tool. The cost scales with worktree count, which power users grow over time.
- `aiwf show` is less hot, but used during planning sessions and Q&A loops with AI assistants. Slow show calls compound across a session.
- M-0130 runs in the pre-push hook, the kernel's authoritative chokepoint. A pre-push that takes seconds on a kernel-sized repo is fine; one that takes minutes on a consumer's larger repo turns the guarantee into a thing operators bypass.
- On macOS, subprocess fan-out becomes OS-resource-bound under load (see G-0125, archived/addressed), raising the cost of fan-out-heavy code paths beyond the Go-level concurrency story.

## Why this is one gap, not three

The fixes share helpers. All three want the same two primitives in `internal/gitops/`:

- A bulk-revwalk helper that streams `(commit-sha, parent-sha, paths, trailers)` in one subprocess via `git log --all --name-status -M --pretty=...`.
- A `cat-file --batch` / `cat-file --batch-check` pump that turns N short-lived `git show` calls into one long-lived subprocess.

Both helpers are general — `aiwf history` (`internal/cli/history/history.go:283, :515`) already uses the single-walk shape and is the right template. Building the helpers once for M-0130 means the `status` and `show` adoptions are small refactors on top, not separate research efforts.

## Resolution paths

- **Measure first.** Before writing any optimization, `time` the three verbs on this repo and on a synthetic large fixture (50 worktrees, 200 scopes, 500 entities). If the cost is invisible at realistic scales, leave `status` and `show` alone and let M-0130 build the helpers in isolation.
- **Cheap win for `status` (small commit, low risk).** Parallelize the inner loop at `worktrees.go:102` with an `errgroup` capped at 8. Cuts wall time ~3–4× without touching the subprocess count. Lands independently of the helpers.
- **Full fix.** Build the two `internal/gitops/` helpers as part of M-0130, then port `status` worktree views and `show` scope views to use them. One gap, one mechanical resolution, three call sites converted.

## Class

Performance / scalability gap. Not blocking on current repo shape; becomes user-visible as repos grow or as M-0130 lands. Audit performed 2026-05-20 via grep over all 31 non-test `exec.Command("git", ...)` sites in `internal/` and `cmd/`.

## Related

- **M-0130** — planned check rule whose proposed approach has the same fan-out shape; helpers should land there.
- **G-0125** (archived, addressed) — first surfaced the observation that aiwf's per-verb git fan-out is an OS-resource concern on macOS; this gap is the runtime-verb analog.
- `internal/cli/history/history.go:283, :515` — existing example of the single-walk shape that should be templated.
