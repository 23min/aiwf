---
id: G-0495
title: contractverify version-substitution test flakes under the instrumented suite
status: open
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

## Why it matters

The failure surfaces at the gate that decides whether a change can land, on
a package the change did not touch. That trains an operator to re-run and
move on, which is how a real regression in the same test gets waved through.

The wrong-layer error message is the compounding half: the test knows
whether the validator ran and does not check, so its output cannot
distinguish "substitution was wrong" from "the validator never executed".

## Scope

The test's own robustness and what it asserts. Out of scope: the version
substitution behaviour it covers, which is correct — every non-flaking run
passes.

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