---
id: G-0740
title: A bare go build in an in-repo worktree stamps the main checkout's revision
status: open
discovered_in: E-0092
---
## What's missing

With the Go toolchain this repository pins (`GOTOOLCHAIN: "go1.25.12"` in `.devcontainer/devcontainer.json`), an `aiwf` binary built with a bare `go build` inside an in-repo worktree (`.claude/worktrees/<branch>/`, the placement ADR-0023 defaults to) reports the main checkout's commit and working-tree state as its version, not the worktree's. When no ldflags `Stamp` is set, `internal/version.Current` falls back to `runtime/debug.ReadBuildInfo`'s `Main.Version`, the module version Go derives from its own version-control lookup; before Go 1.27 that lookup recognises a repository root only at a `.git` directory, so it passes over a worktree's `.git` file and stamps the enclosing main checkout. The `Makefile`'s `build`, `install` and `diag-aiwf` targets set `Stamp` from the worktree's own git, so `Current` reports the worktree for them; `.guidance/project/environment.md` § "Worktree binary discipline" tells AI sessions to `go build -o "$CLAUDE_JOB_DIR/aiwf" ./cmd/aiwf`, which sets none.

Measured in E-0092's worktree on go1.25.12, 2026-10-18:

- With the worktree at `ae127e08f` and the main checkout at `a100c7c64` with an untracked directory, `go build -o <scratch>/aiwf-wt ./cmd/aiwf` then `go version -m <scratch>/aiwf-wt` printed

  ```
  	mod	github.com/23min/aiwf	v0.40.1-0.20261004165646-a100c7c64c51+dirty
  	build	vcs.revision=a100c7c64c51643acbd69dded061cde7ea4b50db
  	build	vcs.modified=true
  ```

  Expected: the worktree's `ae127e08f`, clean.
- With both at `58c817531`, the worktree clean and the main checkout carrying only an untracked directory, a bare `go build` then `aiwf version` printed `v0.40.1-0.20261018210009-58c8175316bb+dirty`, while `make diag-aiwf` then `./bin/aiwf-diag version` printed `epic/E-0092-shrink-the-always-on-guidance-to-one-home-per-rule-under-a-ceiling@v0.40.0-670-g58c817531`, clean.

The lookup reproduces without aiwf, in a scratch repository holding a one-file Go module: commits A then B on `main`, `git worktree add .claude/worktrees/wt A`, `go build` inside it, and `go version -m` prints `vcs.revision=<B>` under go1.25.12 and go1.26.0; under go1.27.1 the same steps print `vcs.revision=<A>`. A sibling worktree outside the repository, built the same way, records no `vcs.*` setting before go1.27.

## Why it matters

The version string is how an operator tells which source a binary came from. A bare-built worktree binary runs the worktree's code but reports the main checkout's commit and dirty state, through `aiwf version` and `aiwf doctor`'s `binary:` row, which renders the same `version.Current()` (`internal/cli/doctor/doctor.go`). By reading the code, not by a run: `binaryStaleness` (`internal/cli/doctor/binary_staleness.go`) compares that version's commit with the trunk ref and skips a `+dirty` version, so a worktree binary built from older code while the main checkout sits at trunk reads as current, and one built while the main checkout has any untracked file is never judged at all.
