---
id: G-0555
title: Two test helpers create temp dirs and never remove them
status: open
priority: medium
---
## What's missing

Two test helpers create a temp directory and never remove it.

`AiwfBinary` (`internal/cli/cliutil/testutil/proc.go`) builds the aiwf binary
into `os.MkdirTemp("", "aiwf-int-build-")` inside a `sync.Once`, and the
function body contains no `RemoveAll`, no `t.Cleanup`, and no deferred cleanup
at all. The shared-binary helper in `internal/stresstest` does the same with
`stresstest-shared-bin-`.

Each directory holds a compiled binary — 18 MB measured. `AiwfBinary` is used
across many test packages, and `go test` runs one process per package, so a
`sync.Once` bounds the builds per *process* rather than per run: a full
`go test ./...` mints one directory per consuming package and keeps all of them.
Measured in one devcontainer: 40 `aiwf-int-build-` directories totalling 727 MB,
plus 11 shared-binary directories, none older than that day's runs.

The correct shape is already present elsewhere in the tree.
`aiwf doctor --self-check` creates its temp repo the same way and removes it on
a deferred call unless `--keep` was passed, which is why its leftovers are
crash residue rather than a leak.

## Why it matters

The growth is monotonic in test runs, so the cost lands on whoever runs the
suite most, and it lands on any machine that runs it — a CI runner and a
contributor's laptop, not only this repo's devcontainer.

It also compounds the failure G-0552 describes. A full disk surfaces as test
failures in whichever tests write most, which is exactly the binary-building and
repo-creating tests these helpers serve; the leak feeds the condition and then
sits in the class of tests that misreport it. G-0552 bounds the Go build cache
and adds a free-space preflight, treating the environment. This is a defect in
the test helpers themselves and stays outside it.

## Resolution shape

`t.Cleanup` is the wrong seam: the directory is created under a `sync.Once` and
must outlive the test that happened to trigger the build. Per-package `TestMain`
teardown would work but costs a mandate — every consuming package gains an
obligation, and a package added later without it leaks silently, which is the
addition-shape worth refusing.

The bounded alternative costs once. Build into a deterministic path derived from
the module and the test binary — under `os.TempDir()` — and reuse it across
runs instead of minting a fresh directory each time. The count is then bounded
by construction, no cleanup step exists to be forgotten, and a stale binary is
handled by the same rebuild decision the helper already has to make. Concurrent
`go test` runs, which the current per-process directory exists to keep apart,
need the path to carry enough identity to stay disjoint — that is the one design
question to settle.

Whichever shape wins, the two helpers should reach it through one routine rather
than two copies of the decision.
