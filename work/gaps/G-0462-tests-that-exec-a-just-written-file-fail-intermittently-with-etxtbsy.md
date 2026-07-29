---
id: G-0462
title: Tests that exec a just-written file fail intermittently with ETXTBSY
status: open
priority: high
discovered_in: M-0281
---
## What's missing

`go test ./...` fails intermittently for reasons unrelated to the change under
test. Two distinct mechanisms are known; both present the same way — a red gate
that goes green on an immediate re-run with no code change — and both want fixing
together, because what makes them expensive is the shared symptom, not the
separate causes.

### Mechanism 1 — `ETXTBSY` on exec of a just-written file

Tests that write an executable and then run it fail with
`fork/exec …: text file busy`. Four distinct tests were observed failing this way
over one milestone's runs:

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

### Mechanism 2 — a lock-acquisition budget that only holds on an idle machine

`internal/stresstest` —
`TestConcurrentIDAllocationScenario_RealBinary_NConcurrentActorsAllGetDistinctIDs`
launches eight concurrent real-binary `aiwf add gap` subprocesses against one
working copy, three attempts per run. Each subprocess waits at most `lockTimeout`
(`internal/cli/cliutil/lock.go`) for the repo lock, and holds it across a real
`git commit`. The scenario's oracle, `classifyConcurrentIDAllocation`, counts
every non-`ok` actor status as a violation, on the stated premise that the lock
serializes all eight to success within that budget.

That premise is a wall-clock assumption, and it holds only when the scenario has
the machine to itself. Under the whole suite at `-parallel 8` the eight actors
queue behind one lock while the rest of the suite competes for the same CPUs; the
queue can then exceed the per-actor budget, an actor returns non-`ok`, and the
oracle reports a violation against a system that is behaving correctly.

The two mechanisms fail in opposite directions, and a fix for one is not a fix
for the other. `ETXTBSY` is a real race the code under test can hit. The lock
budget is a defect in the *oracle*: it converts machine load into a correctness
verdict.

## Why it matters

`go test ./...` is not an advisory signal here. It runs inside `make check-fast`,
`make ci`, and `make coverage-gate`, and CI runs it on every push, so these
failures land on the gate that is supposed to decide whether work is safe to
integrate.

A gate that is occasionally red for reasons unrelated to the change under test
teaches readers to re-run rather than read. That is the expensive part: the next
genuine failure arrives looking exactly like the last spurious one. The repo
already leans hard on mechanical chokepoints over vigilance, and this is the
failure mode that erodes them.

Frequency is low per run but the suite is large, so the per-invocation odds of
*some* test hitting one of these are much higher than any single test's.

## Options

### For mechanism 1

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

### For mechanism 2

1. **Make the actor's lock budget a scenario parameter** and set it generously in
   the in-suite test, keeping the production default for real runs. The oracle
   keeps its meaning; only the deadline stops encoding an idle machine.
2. **Classify a lock-acquisition timeout separately from a real failure.** The
   scenario's interesting invariants are mutual exclusion and distinct ids; an
   actor that never got the lock proves nothing about either, so it need not be a
   violation. Risks masking a genuine deadlock, which argues for reporting it as
   a distinct non-violating outcome rather than dropping it.
3. **Take the scenario out of `go test` and leave it to `make stress`**, where it
   runs without competing load. Loses the per-push signal.

Option 1 is the smaller change and keeps the oracle honest; option 2 is the more
principled one, since a timeout genuinely is not evidence about mutual exclusion.
They compose — a widened budget plus a distinct outcome class for the timeout
that remains.
