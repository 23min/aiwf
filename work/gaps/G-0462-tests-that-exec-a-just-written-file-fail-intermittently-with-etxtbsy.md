---
id: G-0462
title: Tests that exec a just-written file fail intermittently with ETXTBSY
status: open
priority: high
discovered_in: M-0281
---
## What's missing

Tests that write an executable and then run it fail intermittently with
`fork/exec …: text file busy`. Four distinct tests were observed failing this way
over one milestone's runs, each passing on an immediate re-run with no change:

- `internal/gitops` — `TestReconcilePaths_HashObjectFails_ObjectsDirReadOnly`
- `internal/stresstest` — `TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence`
- `internal/policies` — `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently`
- `internal/contractverify` — `TestRun_EvolutionRegression`

The shape is the same in every case: a script or binary is written into a temp
directory and exec'd shortly afterwards while the package's other tests run in
parallel. `ETXTBSY` is what the kernel returns when a file is exec'd while some
process still holds it open for writing, which a concurrent `fork` can produce
even after the writing goroutine has closed its own descriptor — the child
inherits the descriptor across the fork window.

## Why it matters

`go test ./...` is not an advisory signal here. It runs inside `make check-fast`,
`make ci`, and `make coverage-gate`, and CI runs it on every push, so these
failures land on the gate that is supposed to decide whether work is safe to
integrate.

A gate that is occasionally red for reasons unrelated to the change under test
teaches readers to re-run rather than read. That is the expensive part: the next
genuine failure arrives looking exactly like the last four spurious ones. The
repo already leans hard on mechanical chokepoints over vigilance, and this is the
failure mode that erodes them.

Frequency is low per run but the suite is large, so the per-invocation odds of
*some* test hitting it are much higher than any single test's.

## Options

1. **Retry the exec on `ETXTBSY`** in the shared test helpers that write and run
   a file. Smallest change, and it treats the condition where it actually
   surfaces. Needs a bounded retry so a genuine permissions failure still fails.
2. **Write executables before any parallel work starts** — a package-level
   fixture materialized once in `TestMain` rather than per-test. Removes the fork
   window rather than tolerating it, but only for tests that can share one copy.
3. **Serialize the affected tests.** Cheapest to write, worst to live with: it
   slows the suite and hides the condition rather than fixing it.

Option 1 looks right for the general case, with option 2 where a shared fixture
is natural. Whichever is chosen, the helpers are the place — four call sites
independently hit this, so a per-test fix would leave the fifth to be discovered
the same way.
