---
id: G-0391
title: Mutating verbs' lock-busy refusal ignores --format=json
status: open
discovered_in: M-0242
---
## What's missing

`internal/cli/cliutil.AcquireRepoLock` (`internal/cli/cliutil/lock.go`) is the
shared lock-acquisition helper every mutating verb (`add`, `promote`,
`edit-body`, `cancel`, `archive`, `reallocate`, ...) calls before doing any
work. When the repo lock is busy or fails to acquire, it prints a plain-text
message via `cliutil.Errorf` to stderr and returns a bare exit code —
entirely bypassing the `--format=json` envelope contract
(`{tool,version,status,findings,result,metadata}`). A caller invoking any
mutating verb with `--format=json` against a locked repo gets empty stdout
and a non-JSON stderr line, not a `status:"error"` envelope.

Confirmed directly: `aiwf promote <id> <status> --format=json` against a
repo whose lock is held by another process prints nothing to stdout and
exits non-zero with a plain-text stderr message.

## Why it matters

Any JSON-consuming caller (a script, CI step, or this repo's own
`internal/stresstest` harness) that retries or classifies verb output by
parsing `--format=json` stdout gets a parse failure instead of a
structured, classifiable error on this path — the same class of defect as
G-0389 (`aiwf show`'s not-found path ignoring `--format=json`), but wider in
blast radius since `AcquireRepoLock` is the shared chokepoint for every
mutating verb, not one verb's one error path.
