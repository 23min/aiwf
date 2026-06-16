---
id: G-0250
title: Test git fixtures inherit git-locator env vars, flaking under git hooks
status: open
discovered_in: E-0040
---
## Problem

Test packages whose tests shell out to `git` build their fixture repos in a
`t.TempDir()` and invoke `git` with `cmd.Dir` set but **no `cmd.Env`** — so
every fixture git command inherits the ambient process environment. When the
suite runs inside a git hook (the pre-commit policy hook, or `make ci` invoked
from a `git commit`), git has exported the git-locator env vars: `GIT_DIR`,
`GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_WORK_TREE`, `GIT_COMMON_DIR`.
Those override `cmd.Dir`-based discovery, so the fixtures' `git init` / `git
add` / `git commit` operate against the parent repo's shared git state instead
of their isolated temp repos. Parallel tests then collide on one index /
object DB / lockfile.

This is the recurring "flake under full-suite parallel load; passes isolated"
attributed to the `G-0097` family. It is not a filesystem timing race — it is
deterministic given the leaked environment.

## Evidence

A representative failure (perf test in `internal/check`):

```
--- FAIL: TestFSMHistoryConsistent_PerfBudget
    fsm_history_perf_test.go:62: running [git commit ... -m retitle E-0038]: exit status 1
        error: invalid object 100644 <sha> for 'work/epics/E-0037-x/epic.md'
        error: Error building trees
```

The perf test builds 50 epics in a single-threaded loop in its own temp repo.
A blob committed ~37 iterations earlier cannot vanish from an isolated repo's
object DB — only a shared object DB / index explains it.

The leaking-env mechanism is already documented and fixed in **one** package:
`internal/policies/setup_test.go` unsets the five locator vars in `TestMain`
with a comment naming the cause ("ambient git locator env vars that a parent
git hook invocation passes down ... would steer those into the parent repo's
gitdir"). The fix was never propagated to the other git-shelling test packages.

## Reproduction

Build the `check` test binary and run it under a simulated hook environment:

```
go test -c -o /tmp/check.test ./internal/check/
# clean env, -parallel 16  -> PASS
GIT_INDEX_FILE=/tmp/shared /tmp/check.test -test.parallel 16 -test.count 2
#  -> FAIL: Unable to create '/tmp/shared.lock': File exists.
#           Another git process seems to be running in this repository
```

The exact downstream wording ("invalid object / Error building trees",
"index.lock exists", "directory not empty") varies with which var leaks and the
interleaving; the root cause is one bug.

## Scope

Only `internal/policies` scrubs the locator vars. The other git-shelling test
packages do not, including: `internal/check`, `internal/cli/integration`,
`internal/cli/check`, `internal/cli/authorize`, `internal/cli/status`,
`internal/cli/doctor`, `internal/verb`, `internal/gitops`, `internal/initrepo`,
`internal/trunk`, `internal/cellcoverage`.

## Proposed fix

1. Factor the hardening into a shared helper
   `internal/testsupport.HardenGitTestEnv()` that scrubs the five locator vars
   (and, folded in per G-0251, disables git auto-gc). Each git-shelling
   package's `TestMain` calls it (one line), replacing the inline block in
   `internal/policies/setup_test.go`.
2. Add an `internal/policies/` chokepoint (`PolicyGitTestEnvHardened`) asserting
   every test-bearing package whose `*_test.go` shells a subprocess calls the
   helper in `TestMain`. Without the chokepoint, the next git-shelling package
   forgets the call and the flake returns.

## Out of scope / follow-up

The **production** `internal/gitops` path has the same exposure: when `aiwf
check` runs as a hook it inherits `GIT_DIR` and could target the wrong gitdir.
That is a distinct, higher-blast-radius concern deserving its own gap and
tests — not folded into this test-infra fix.
