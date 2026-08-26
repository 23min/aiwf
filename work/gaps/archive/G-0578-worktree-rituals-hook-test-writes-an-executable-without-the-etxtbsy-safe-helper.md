---
id: G-0578
title: Worktree-rituals hook test writes an executable without the ETXTBSY-safe helper
status: addressed
discovered_in: M-0306
addressed_by_commit:
    - 793b1ad97
---
## What's missing

`internal/policies/worktree_rituals_check_hook_test.go` writes the hook script under
test with a bare `os.WriteFile` carrying an exec bit, where this repo's convention is
`testsupport.WriteExecutable`.

## Why it matters

A plain write holds a writable descriptor for as long as the file is open. A fork
concurrent with that window inherits the descriptor, and `execve` against a file held
open for writing fails with `ETXTBSY` — so the fixture step fails and the property the
test exists to check is never evaluated. The failure is timing-dependent and therefore
silent until a loaded run surfaces it: observed once during M-0306's wrap, where
`make ci` failed with `fork/exec ...: text file busy` and the same test then passed on
three consecutive re-runs.

`testsupport.WriteExecutable` holds `syscall.ForkLock` across the write, which excludes
exactly those forks. Writing to a temp name and renaming does not help — `ETXTBSY` is
enforced against the inode, which the rename carries along.

The `test-executable-write` policy that would catch this is diff-scoped, so it never
reaches a line no change has touched. The whole-file sweep that backstops it covers
`internal/stresstest` and `internal/contractverify`, not `internal/policies`.
