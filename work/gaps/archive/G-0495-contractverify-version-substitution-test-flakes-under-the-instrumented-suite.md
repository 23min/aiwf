---
id: G-0495
title: contractverify version-substitution test flakes under the instrumented suite
status: wontfix
discovered_in: M-0284
---
## What's missing

`TestRun_VersionSubstitutionFlowsThrough` fails intermittently, and only
inside the coverage-instrumented full-suite run. Measured: 2 failures in 3
runs of `make coverage-gate`; 0 failures across repeated runs of the test
alone, of its whole package, and of either with coverage instrumentation.

The reported error names the wrong thing:

```
reading validator log: open /tmp/.../validator.log: no such file or directory
```

The test writes a `/bin/sh` script into a temp repo, has the contract
runner exec it as a validator, and then reads a log the script appends to.
It discards the runner's return value, so when the exec itself fails there
is no assertion on the findings that would say so — the only symptom is the
absent log, and the message points at reading rather than at launching.

A second test flakes with the same fixture shape:
`TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence` in
`internal/stresstest`. Measured: one failure across full-suite runs, and no
failure running it standalone, running its whole package, or running either
under coverage instrumentation. Its `writeFakeAiwfList` helper writes an
executable shell script and immediately execs it, exactly as the validator
fixture above does.

Whether the two share a cause is a hypothesis, not a finding. The failure
message for the stresstest instance was not captured, so the link rests on
the shared fixture shape and the shared "only under the full parallel suite"
condition.

`G-0491` tracks this class and identifies the mechanism: a stand-in binary
written with `os.WriteFile(path, …, 0o755)` and exec'd immediately, where a
`fork` elsewhere in the process inherits the writable descriptor and `execve`
fails on a file another process holds open for writing. It names the
`internal/stresstest` fixture directly and rules out background auto-gc.

Two sightings recorded here are not named there:

- `TestRun_VersionSubstitutionFlowsThrough` (`internal/contractverify`), whose
  validator script has the identical write-then-exec shape. Failed 2 of 3
  instrumented full-suite runs — the highest rate observed — and never in
  isolation, under package load, or under instrumentation alone.
- `TestReconcilePaths_HashObjectFails_ObjectsDirReadOnly` and its `CommitTree`
  sibling (`internal/gitops`), seen once under the full gate and green
  standalone, per-package, and at `-count=4`. These are chmod-based rather than
  write-then-exec, so whether they share a cause is unestablished.

## Why it matters

The failure surfaces at the gate that decides whether a change can land, on
a package the change did not touch. That trains an operator to re-run and
move on, which is how a real regression in the same test gets waved through.

The wrong-layer error message is the compounding half: the test knows
whether the validator ran and does not check, so its output cannot
distinguish "substitution was wrong" from "the validator never executed".

## Scope

The write-then-exec fixture pattern where it appears, and what those tests
assert when the exec fails. Out of scope: the behaviours they cover, which
are correct — every non-flaking run passes.

## Resolution options

1. Assert on the runner's findings before reading the log, so a failed
   launch reports as a failed launch. Cheapest, and it converts the flake
   into a message that names its own cause even if the underlying race
   stays.
2. Remove the write-then-exec race. A script written and immediately
   executed while the suite forks heavily is the classic `ETXTBSY` shape —
   a plausible mechanism, not a confirmed one. Placing the script outside
   the temp repo, or retrying the exec, would settle it.
3. Both: option 1 for the diagnosis, option 2 for the cause.