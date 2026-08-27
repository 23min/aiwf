---
id: G-0645
title: Shared test-binary temp dirs are never removed, leaking ~23G to /tmp
status: open
priority: medium
---
## What's missing

`internal/cli/cliutil/testutil/proc.go:94` and `internal/stresstest/sharedbinary_test.go:73`
each create a per-process build directory with `os.MkdirTemp` inside a `sync.Once`, and
neither directory is ever removed. It outlives the test process: nothing in the `testutil`
package removes it, and no `TestMain` among the packages that reach these helpers
(`internal/cli/integration`, `internal/policies`, `internal/stresstest`) tears it down.
`internal/gitops/setup_test.go:34` is the working counterexample in the same tree — it calls
`cleanupGPGSignFixture()` between `m.Run()` and `os.Exit`, which is why the gpg fixture's
directories do not accumulate the same way.

Measured in the devcontainer on 2026-08-27 with no test process running:
`find /tmp -maxdepth 1 -name 'aiwf-int-build-*' | wc -l` reported 925 directories, and
`du -sch` over them reported 17G; the same commands over `stresstest-shared-bin-*` reported
308 directories and 5.4G. Each directory holds one ~18MB `aiwf` binary and nothing else.
Directory timestamps spanned 2026-08-16 to 2026-08-26. Three packages reference
`AiwfBinary`, so one full `go test ./...` leaves two to three directories behind.

## Why it matters

23G accumulated in ten days of ordinary development, and the growth rate is set by how often
the suite runs rather than by anything self-limiting. Nothing reclaims it on its own: the
directories are recent, so an age-based `/tmp` sweep spares them, and `go clean -cache` does
not reach them — they are not build-cache entries. On a devcontainer whose filesystem is
shared with the Docker image store, that growth competes for space with everything else on
the volume, and an unbounded consumer ends in a full disk whose first symptom is an
unrelated build or commit failure rather than a disk-space error.
