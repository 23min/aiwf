---
id: G-0730
title: aiwf check runs a whole-repository git fsck once per hinted history finding
status: open
discovered_in: E-0092
---
## What's missing

`aiwf check` runs a whole-repository `git fsck` once for every `fsm-history-consistent` finding that asks for the dangling-acknowledgment hint. `findDanglingAckHint` (`internal/check/acks.go:238`) starts `git fsck --unreachable --no-reflogs` on each call, and `fsmHistoryConsistentWithDeps` (`internal/check/fsm_history_consistent.go:196-200`) calls it from both `illegalTransitionFindings` and `forcedUntraileredFindings`, once per finding. The cost is the number of such findings times a full scan of the repository. The scan entered for illegal-transition findings at `d9f32dc35` (2026-07-09), and the merge `9304eb26a` extended it to forced-untrailered ones.

Measured 2026-10-01 in the devcontainer, Go 1.25.12. `TestFSMHistoryConsistent_PerfBudget` (`internal/check/fsm_history_perf_test.go`) times the rule on a 50-entity fixture that produces 100 findings; the `git fsck` count comes from a logging `git` shim placed first on `PATH` while the test ran.

| Commit | What it is | Elapsed (rounded) | `git fsck` runs |
|---|---|---|---|
| `0124be73b` | the test lands | 0.163 s | — |
| `9304eb26a^1`, `9304eb26a^2` | the parents of the merge below | 0.102 s, 0.108 s | 0 |
| `9304eb26a` | merge whose result passes the hint to `forcedUntraileredFindings`, which neither parent does | 2.47 s | 50 |
| `15f6360e6^` | | 2.56 s | 50 |
| `15f6360e6` | epic `active → done` made a sovereign act; the findings stay at 100 and the hinted ones double | 4.97 s | 100 |
| `a61f3d8de` | | 4.7 s | 100 |

```
$ git checkout --detach 15f6360e6
$ go test -count=1 -run 'TestFSMHistoryConsistent_PerfBudget$' -v ./internal/check/ 2>&1 | grep -oE 'fixture: .*findings'
fixture: 4.96749147s elapsed, 100 findings
```

One scan of this repository's object store:

```
$ time git fsck --unreachable --no-reflogs > /dev/null
real	0m3.089s
$ git count-objects -v | grep in-pack
in-pack: 92146
```

The test's comment says the rule "completes in well under 1 second" and that the 10-second budget is "10× generous"; both are now false.

## Why it matters

An unacknowledged illegal-transition or forced-untrailered finding is an error, so a push carrying one is already refused. What this adds is one full scan of the repository per such finding to every `aiwf check` run until they are acknowledged: about 3 s each here, so twenty of them make each run about a minute longer while the operator works through them, and the cost grows with the size of the repository.

The perf test now runs at about half its budget, so ordinary load decides whether it passes. On `epic/E-0092-shrink-the-always-on-guidance-to-one-home-per-rule-under-a-ceiling` after it merged `main` (`681d5bc2d`), `make check-fast` at `99d657cf3` failed it at 25.78 s; a full `go test ./...` on an earlier trial merge of the same branches failed it at 19.21 s. The same test alone took 4.9 s on that tree.
