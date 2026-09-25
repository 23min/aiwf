---
id: G-0110
title: mutate-diff mutates every line of each changed package, not only changed lines
status: open
priority: medium
discovered_in: M-0097
---

## What's missing

`make mutate-diff` (`scripts/mutate-diff.sh`) mutates every line of each changed
`internal/` package, not the changed lines. It runs `gremlins unleash <package>` once
per changed package and never passes `--diff`. Run as
`MUTATE_DIFF_BASE=HEAD timeout 1200 make mutate-diff` over a 30-package change, it was
still inside the first package, `internal/check`, when the 20-minute limit ended it.

Adding `--diff` does not scope a run shaped like the script's. Measured with gremlins
v0.6.0 on a scratch module holding a modified file (`internal/a/a.go`, line 13
changed, line 5 not), a staged new file, an untracked new file and a changed file
under `cmd/`; each run's `CONDITIONALS_BOUNDARY` lines and coverage line shown:

```
$ gremlins unleash --dry-run --diff <base> ./internal/a
     SKIPPED CONDITIONALS_BOUNDARY at a.go:5:7
     SKIPPED CONDITIONALS_BOUNDARY at a.go:13:7
     SKIPPED CONDITIONALS_BOUNDARY at untracked.go:4:39
     SKIPPED CONDITIONALS_BOUNDARY at staged.go:4:36
Mutator coverage: 0.00%

$ gremlins unleash --dry-run --diff <base>          # from the module root
 NOT COVERED CONDITIONALS_BOUNDARY at cmd/x/x.go:3:31
    RUNNABLE CONDITIONALS_BOUNDARY at internal/a/a.go:13:7
     SKIPPED CONDITIONALS_BOUNDARY at internal/a/untracked.go:4:39
     SKIPPED CONDITIONALS_BOUNDARY at internal/a/a.go:5:7
    RUNNABLE CONDITIONALS_BOUNDARY at internal/a/staged.go:4:36
Mutator coverage: 66.67%
```

- A package-path run skips every mutant, the changed line included. The filter keys
  files by the repo path `git diff --merge-base <base>` prints (`internal/a/a.go`),
  while a package-path run names the files it walks relative to the package
  (`a.go`), so nothing matches.
- An untracked file is skipped either way: `git diff` does not list it. A staged or
  committed new file is mutated from the module root, so new files as such are not
  excluded.
- Gremlins reads a hunk's added lines as one run starting at the hunk's first change.
  With lines 6 and 12 of one file edited and line 9 between them untouched, a
  root dry-run marked line 6 RUNNABLE and line 12 SKIPPED; with `diff.context=0` set
  for that `git diff`, lines 6 and 12 were RUNNABLE and line 9 SKIPPED.

## Why it matters

The `wf-vacuity` and `wf-patch` rituals send a reviewer to the diff-scoped mutation
command when one exists, so the check of whether a change's assertions kill its
mutants runs through this target. Mutating whole packages makes that run too slow to
finish inside a review. Scoping it with `--diff` from a package path reports no
survivors while testing nothing, which reads as a pass.
