---
id: G-0562
title: A worktree-hook test writes its executable without WriteExecutable
status: open
priority: medium
---
## What's missing

`writeWorktreeHookScript` in `internal/policies/worktree_rituals_check_hook_test.go`
writes the hook script it is about to exec with a bare
`os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755)`, rather than
through `testsupport.WriteExecutable`.

That is the write shape G-0491 identified and the repo bans. A plain write holds
a writable descriptor for as long as the file is open; a fork concurrent with
that window inherits it, and `execve` on a file some descriptor still holds open
for writing fails with `ETXTBSY`. `WriteExecutable` holds `syscall.ForkLock`
across the write, which excludes exactly those forks.

Three tests in the file exec the script this helper produces, so all three carry
the window.

## Why it matters

It fires. Observed 2026-08-06 on one `make check-fast` run:

    --- FAIL: TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently
        worktree_rituals_check_hook_test.go:86: running hook script:
        fork/exec .../worktree-rituals-check.sh: text file busy

Three further runs of the same package were green, which is the shape of the
problem rather than evidence against it: the failure needs a fork inside the
write window, so it surfaces under whatever concurrency the run happens to have
and hides otherwise. What it costs is not a wrong verdict but a lost one — the
fixture step fails, so the property the test exists to pin is never evaluated,
and it reads as a red suite rather than as an untested invariant.

`internal/policies` sits in the every-push path, so the flake is reachable from
`make check-fast`, `make ci`, and CI alike.

## Why no rule reaches it

`test-executable-write` is the chokepoint for exactly this, and it is
diff-scoped: it audits lines a change touches, so a call site that predates it
stays forgiven until someone edits that line. Its whole-file sibling,
`TestAdoptedPackagesRouteThroughWriteExecutable`, holds `internal/stresstest`
and `internal/contractverify` clean — the two packages where the fault was
measured and where it recurred. `internal/policies` is in neither list.

So the gap is not that the rule is wrong; it is that this call site has never
been in scope for either half of it.

## Where to fix

- `internal/policies/worktree_rituals_check_hook_test.go` — route the write
  through `testsupport.WriteExecutable`.
- Whether `internal/policies` should join the whole-file adopted-package list is
  the second question, and the more consequential one: adopting a package is a
  standing cost, and the list is deliberately short. The evidence it asked for —
  five further exec-bit writes in the package, four of them in files that exec —
  is measured in G-0641, which carries that question.

## Related

- G-0491 — where the `ETXTBSY` mechanism and the `WriteExecutable` remedy were
  established.
