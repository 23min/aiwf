---
id: G-0462
title: 'Intermittent test failures: ETXTBSY on exec and golangci-lint cache contention'
status: open
priority: high
discovered_in: M-0281
---
## What's missing

`go test ./...` fails intermittently for reasons unrelated to the change under
test. Two mechanisms are tracked here; both present the same way — a red gate
that goes green on an immediate re-run with no code change — and they want fixing
together, because what makes them expensive is the shared symptom, not the
separate causes.

Timing-dependent stress-scenario oracles are a third mechanism of the same
class, tracked separately as **G-0468**, whose remedy is independent of the two
here.

### Mechanism 1 — `ETXTBSY` on exec of a just-written file

Tests that write an executable and then run it fail with
`fork/exec …: text file busy`. Observed across these tests over one milestone's
runs, and the list is a sample of the condition rather than its full extent:

- `internal/gitops` — `TestReconcilePaths_HashObjectFails_ObjectsDirReadOnly`
- `internal/stresstest` — `TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence`
- `internal/policies` — `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently`
- `internal/contractverify` — `TestRun_EvolutionRegression` and
  `TestRun_VerifyPassClean`

The shape is the same in every case: a script or binary is written into a temp
directory and exec'd shortly afterwards while the package's other tests run in
parallel. `ETXTBSY` is what the kernel returns when a file is exec'd while some
process still holds it open for writing, which a concurrent `fork` can produce
even after the writing goroutine has closed its own descriptor — the child
inherits the descriptor across the fork window.

### Mechanism 2 — any `golangci-lint` process anywhere on the machine

`internal/policies` — `TestGolangciConfigRulesFire` shells out to a real
`golangci-lint` per guarded rule, to prove the rule fires rather than merely
appearing in the enable list. It passes the inherited environment, so the child
uses golangci-lint's machine-global default cache. `make lint` by contrast
scopes its cache per worktree, deriving `GOLANGCI_LINT_CACHE` from
`git rev-parse --absolute-git-dir` (G-0179).

golangci-lint refuses to run concurrently against one cache; the second
invocation exits with `Error: parallel golangci-lint is running`. The harness
asserts on the child's *output* and deliberately ignores its exit code, because
findings are expected, so that refusal reads as a missing finding and the test
reports:

```
rule forbidigo-panic did not fire: golangci-lint output lacked "(forbidigo)"
 — the config rule is dormant, disabled, or dropped from the enable list
```

Any concurrent golangci-lint suffices: another worktree's `make lint`, a
pre-push hook, an editor integration. This repo routinely carries several
worktrees at once — an epic, a patch, transient agent worktrees — so the
condition is ordinary rather than exotic.

The two fail in different directions, and a fix for one is not a fix for the
other. `ETXTBSY` is a real race the code under test can hit. Mechanism 2 is a
defect in the *diagnostic*: the condition is external and harmless, but the
message accuses the lint configuration of exactly the defect the harness exists
to detect, so a reader who trusts it goes looking for a dormant rule that is
working fine.

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
is natural. Whichever is chosen, the helpers are the place: the condition has
surfaced independently in several packages, so a per-test fix leaves the next one
to be discovered the same way.

### For mechanism 2

1. **Scope the child's cache to a per-test temp directory** by setting
   `GOLANGCI_LINT_CACHE` in the spawned command's environment. Removes the
   contention rather than tolerating it, and applies one level down the same fix
   the Makefile already applies to `make lint`. The cost is a cold cache per
   rule, against a synthetic single-file fixture where a warm cache buys little.
2. **Detect the refusal and retry with backoff.** Keeps a shared warm cache but
   leaves a wall-clock assumption in place, which is the shape of mechanism 2.
3. **Treat the refusal as an environmental skip.** Honest about what happened,
   but a skip that fires under ordinary conditions makes the chokepoint
   sometimes-absent — the vacuous-chokepoint condition the harness was built
   against.

Option 1 is the lean. Whichever is chosen, the refusal must stop being reported
as a dormant rule: that message is wrong about the cause even when the run is
genuinely contended.
