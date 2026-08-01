---
id: G-0491
title: Test fixtures race the filesystem on files they just wrote
status: open
priority: medium
---
## What's missing

Two test fixtures reach for a file they just wrote and assume the filesystem still presents it the way they left it. Both fail as an unexplained red naming a fixture step rather than the property under test, and both surface only under whole-suite parallelism, so neither reproduces when someone runs the failing test alone.

### A commit object assumed to still be loose

`corruptUnusedRitualRef` (`internal/cli/integration/isolation_escape_oracle_scenarios_test.go:285`) builds its fixture by committing a file, then overwriting the commit's object *file* in place to make a per-ref first-parent walk fail while `for-each-ref` still emits the ref name. It computes the path arithmetically — `.git/objects/<sha[:2]>/<sha[2:]>` — and fatals if the chmod ahead of the overwrite fails.

That path exists only while the object is loose. Nothing in the fixture establishes that it is, so when the object is not at that path the scenario reports a fixture error rather than the oracle verdict it exists to check:

```
--- FAIL: TestBranchOracle_AC3_SovereignOverride_StaysClean/AC-3_sovereign_(paired):_acknowledged_escape_+_unrelated_corrupt_ref_→_oracle-failure_fires
    branch_scenarios_helpers_test.go:169: chmod object .../.git/objects/17/541d2e20deffd58097c298ae26f12391c6b612: no such file or directory
```

Observed once in five full `make ci` runs on 2026-07-30 and 2026-07-31. It does not reproduce in isolation: `go test -count=20` on the same test is green 20 out of 20, so the trigger is whole-suite load rather than anything in the scenario's own sequence.

**Background auto-gc is ruled out**, which is the first explanation to reach for and the one G-0251 already closed for a different flake. `testsupport.HardenGitTestEnv` forces `gc.auto=0` and `gc.autoDetach=false` onto every child git via `GIT_CONFIG_COUNT`; `internal/cli/integration`'s `TestMain` calls it; and `testutil.RunGit` builds its environment from `os.Environ()`, so the config reaches the git processes this fixture runs. What does move the object is therefore still unidentified.

### A stand-in binary exec'd while a write fd is still open

`writeFakeAiwfList` (`internal/stresstest/verb_sequence_list_invariant_real_test.go:84`) writes an executable shell stand-in with `os.WriteFile(path, …, 0o755)` and the test execs it immediately. Between that open and its close, a `fork` anywhere else in the process inherits the writable descriptor, and `execve` on a file another process holds open for writing fails:

```
--- FAIL: TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence
    verb_sequence_list_invariant_real_test.go:73: checkListInvariant: running aiwf list --archived: fork/exec .../aiwf: text file busy
```

The package runs its tests in parallel and several of them spawn subprocesses, so the forks that collide with the write are the suite's own. Measured at roughly one full `stress`-tagged package run in four, on trees both with and without the G-0468 patches — so it predates them and none of them aggravated it. A focused `-run` of the failing test does not reproduce it, since the collision needs the rest of the package running alongside.

The same write-then-exec shape has since spread: the stand-ins at `internal/stresstest/mid_write_kill_retry_test.go` follow it, and every one is a fresh site for the same race.

## Why it matters

Each is a fixture defect reported as a product defect. The subject under test — whether the branch oracle stays clean under a sovereign override, whether the list invariant catches a divergence — is never evaluated on the run that fails, so the red says nothing about the code while looking exactly like it does.

Both land in packages whose full runs take minutes, so the cost of a spurious red is a re-run of a slow package plus the attention spent ruling out the change in flight. That is the same economics G-0457 records for the chronic red gate, at lower frequency.

It is also the defect class G-0468 removed, one layer down: an assertion whose outcome depends on machine state that the property under test does not depend on. G-0468 addressed it in the stress oracles; this is the same shape in the fixtures those oracles run on, and the exec race is actively spreading as new stand-ins copy the pattern.

## Resolution shape

The exec race has a settled fix, and it is one helper rather than a change per site: write the stand-in to a sibling temp name and `os.Rename` it into place. Rename is atomic and the final path is never the target of a writable descriptor, so no concurrent fork can hold one. Route every fixture that writes an executable through it, which also stops the shape spreading further by copy.

The object-storage assumption is less settled. Stop depending on where git chose to store the object; two directions, neither picked:

- **Corrupt the ref rather than the object.** The fixture's stated goal is a ref whose walk fails while `for-each-ref` still lists it. Writing a well-formed but unresolvable object id into the ref file produces exactly that, and touches no object storage at all.
- **Locate the object rather than compute its path.** Ask git where the object lives before writing to it, and fail with a message naming the packed case if it is not loose.

Either way the fixture should report "this fixture cannot construct its precondition" distinctly from the oracle verdict, so a future occurrence is unambiguous at a glance.

Identifying what actually relocates the object is worth doing first if it is cheap, since the answer may apply to other fixtures that reach into `.git/objects`. It is not a prerequisite for the fix: both directions above are correct regardless of the cause.

## Where to fix

- `internal/cli/integration/isolation_escape_oracle_scenarios_test.go` — `corruptUnusedRitualRef`, the path arithmetic and the chmod/overwrite pair.
- Any sibling fixture that reaches into `.git/objects` by computed path carries the same assumption and should be checked alongside it.
- `internal/stresstest/verb_sequence_list_invariant_real_test.go` — `writeFakeAiwfList`, where the write-then-exec race is measured.
- `internal/stresstest/mid_write_kill_retry_test.go` — the stand-ins carrying the same shape, which want the shared helper rather than three copies of the fix.
