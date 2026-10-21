---
id: G-0738
title: Claude reads go-conventions.md in Go tasks far more often than the Go pack
status: open
discovered_in: M-0361
---
## What's missing

In this repository's Go tasks, Claude reads `.guidance/project/go-conventions.md` far more often than the Go pack, `.guidance/packs/go/cobra/guide.md`, so a Go rule only the pack states often goes unread. The router, `.guidance/project.md`, sends a Go task to the pack and then to `go-conventions.md` from its language-pack line, and `go-conventions.md`'s opening says "Read the pack first."

Measured with E-0092's observation harness (`scripts/e0092-observe/run.sh <checkout> claude <task> <n>`, then `summarize.py <round>`, which counts the runs that read each document before their first edit); E-0092's wrap deleted the harness, and it is read from history at `e8d1d495c` (`git show e8d1d495c:scripts/e0092-observe/run.sh`) in the Linux devcontainer on Claude Code 2.1.296; M-0361's Validation holds the commands, the rounds and the per-round tables. Expected: a run on a Go task reads the pack before its first edit. Observed, in Claude's T1, T3 and T11 runs:

| Checkout | Runs | Opened the router | Read `go-conventions.md` | Read the Go pack |
|---|---|---|---|---|
| `74e8fbd2f` | 12 | 12 | 11 | 8 |
| `e92ce85e0` (T1 only) | 6 | 6 | 6 | 1 |
| `e8d1d495c` (T1 only) | 6 | 4 | 3 | 0 |

At `e8d1d495c`, two runs read no guidance document at all, and none of the four that opened the router read the pack. In every L1 and L2 run at `74e8fbd2f`, Claude read the Python or TypeScript pack, languages with no project document of their own.

## Why it matters

A rule the pack states and `go-conventions.md` does not reaches Claude only when it opens the pack, which at the two later checkouts it did in 1 of 12 runs. `go-conventions.md` repeats the pack's rules this repository depends on, but the rest of the pack, among them wrapping errors across boundaries, small interfaces defined at the consumer and checking a deferred `Close`'s error, is read only then, and nothing reports a run that skips it.
