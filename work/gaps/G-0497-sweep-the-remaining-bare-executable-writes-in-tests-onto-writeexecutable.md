---
id: G-0497
title: Sweep the remaining bare executable writes in tests onto WriteExecutable
status: open
---
## What's missing

93 test sites across 13 packages still write an executable stand-in with a bare `os.WriteFile(path, …, 0o755)` instead of `testsupport.WriteExecutable`, leaving them exposed to the `ETXTBSY` race G-0491 measured and fixed.

The mechanism, established there: a plain write holds a writable descriptor on the new file between its open and its close. A `fork` anywhere else in the process during that window hands the child a copy, and `execve` on a file any descriptor holds open for writing fails with "text file busy". In a package whose tests spawn subprocesses in parallel the colliding forks are the suite's own. `testsupport.WriteExecutable` closes the window by holding `syscall.ForkLock` across the write — measured at zero `ETXTBSY` across 19,200 write-then-exec cycles under deliberate fork pressure, where the bare write reports 12–17%.

Counted with the `test-executable-write` detector itself rather than by grep, so string literals that merely look like calls are excluded:

| package | sites |
| --- | --- |
| `internal/initrepo` | 35 |
| `internal/cli/integration` | 17 |
| `internal/cli/doctor` | 9 |
| `internal/policies` | 8 |
| `internal/skills` | 6 |
| `internal/gitops` | 4 |
| `internal/cli/update` | 3 |
| `internal/cli/cliutil` | 3 |
| `internal/cli/worktree` | 2 |
| `internal/verb` | 2 |
| `internal/cli/initcmd` | 2 |
| `internal/cli/upgrade` | 1 |
| `internal/cli/contract` | 1 |

Several write something the suite then really runs — a fake `go` binary on `PATH`, a `gh` stub, statusline scripts, git hooks — so the exposure is not theoretical.

## Why it matters

The failure lands as an unexplained red naming a fixture step, on a package the change under review never touched, and it does not reproduce when someone reruns the failing test. That trains an operator to re-run and move on, which is how a real regression in the same test gets waved through.

Two packages have already been bitten and are fixed: `internal/stresstest`, where the race was first measured at roughly one full tagged-package run in four, and `internal/contractverify`, which failed during the verification run for the fix itself. Nothing distinguished `internal/contractverify` from the packages in the table above beforehand — it was on exactly this list until the day it failed. That is the reason to treat the remainder as unswept rather than as safe.

The diff-scoped `test-executable-write` policy stops the count growing: any newly added or edited bare executable write in a test fails the gate. It does not shrink it, and nothing else will.

## Resolution shape

Mechanical, and the helper already exists. Each site becomes:

```go
if err := testsupport.WriteExecutable(path, data); err != nil {
    t.Fatalf("<the caller's own message>: %v", err)
}
```

The per-site `//nolint:gosec` disappears with it — the suppression lives in the helper — so the sweep removes noise as well as risk. Sites whose *subject* is the file mode, if any turn up, take `//exec:ok <reason>` instead.

Two decisions worth taking deliberately rather than by default:

- **Whether to sweep in one change or per package.** One change is a large mechanical diff that is tedious to review but trivially uniform; per package is reviewable but drags. The count is dominated by `internal/initrepo` (35), which is a plausible unit on its own.
- **Whether the gate flips to whole-tree afterwards.** `TestAdoptedPackagesRouteThroughWriteExecutable` currently holds two packages clean whole-file, listed explicitly. Once the tree is clean the natural end state is that list becoming "every package", which turns the diff-scoped rule into a whole-tree one and retires this class — the same two-stage shape `comment-history-attrition` used (clear the backlog, then gate the tree).

## Where to fix

- The 13 packages in the table. `internal/policies` is worth doing early despite its size: it holds the detector, and a fixture there writing a fake `go` binary that the suite execs is the most self-evidently wrong instance.
- `internal/policies/test_executable_write_test.go` — the `adoptedPackages` list is where the whole-tree flip lands when the sweep completes.
