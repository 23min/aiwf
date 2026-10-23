---
id: G-0654
title: make build emits a bin/aiwf that gets reached for instead of the release
status: open
priority: medium
---
## What's missing

`make build` writes its binary to `bin/aiwf` (Makefile:66). Nothing consumes
that path except `make selfcheck` (Makefile:289). The other apparent references
are not references: the `bin/aiwf` strings under `internal/` are fake path
literals in table-driven tests, the e2e suite has none, and the test harness
builds to its own temp directory and prepends *that* to PATH precisely so hooks
resolve to the test build rather than an installed binary
(`internal/cli/cliutil/testutil/proc.go:112-124`). Every other caller reaches
`aiwf` through PATH, which is the installed release.

So the file exists as a by-product: an executable named exactly like the one on
PATH, at a path that is short to type and trivial for an agent to discover,
carrying whatever source state the last `make build` happened to see. Its stamp
(`<branch>@<git describe>`) does distinguish it from a release's `vX.Y.Z`, but
only for a reader who thinks to look.

The release of v0.34.0 hit this. `bin/aiwf` in the kernel checkout was 72
commits stale — `main@v0.33.0-72-gc308a3994` — and was used to regenerate
`STATUS.md`, a read computed from superseded logic. Nothing surfaced the
staleness; it was noticed by chance while checking something else.

The diagnostic route already avoids the problem by construction: `make
diag-aiwf` writes `bin/aiwf-diag` (Makefile:81-85), a name that cannot be
confused with the release and that PATH will never resolve. `make build` is the
one producer that emits the confusable name.

## Why it matters

A stale binary answers questions wrongly and says nothing about it. For reads
(`check`, `show`, `status`, `list`) the answer is computed from rules that no
longer apply and is acted on as current. For writes (`add`, `promote`,
`archive`) superseded verb logic reaches real files and real commits; that half
is backstopped, because the git hooks resolve `command -v aiwf` to the installed
release and re-validate at pre-commit and pre-push. Reads consumed
conversationally have no such backstop.

A `bin/aiwf` built from this repository says so once its source falls behind:
every verb prints a stale-binary warning to stderr when the binary's source
differs from the worktree's `HEAD` (`StaleBinaryWarning`,
`internal/cli/doctor/binary_staleness.go`). A copy built before that warning
existed, such as one left in a checkout, still answers silently, and the file is
still named like the release and still reached for.

Renaming it is not free. `aiwf doctor --self-check` puts its own binary's
directory first on `PATH` so the hooks it installs resolve `aiwf` to the binary
under test (`internal/cli/doctor/selfcheck.go`); that works only while the binary
is named `aiwf`, so `make selfcheck` depends on the name `make build` gives it.
