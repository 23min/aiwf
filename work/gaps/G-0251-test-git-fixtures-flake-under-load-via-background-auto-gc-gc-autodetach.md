---
id: G-0251
title: Test git fixtures flake under load via background auto-gc (gc.autoDetach)
status: addressed
discovered_in: E-0040
addressed_by_commit:
    - b53360c5
---
## Problem

Independently of the env-var leak in `G-0250`, test fixtures that build a git
repo in a `t.TempDir()` and run many commits flake under high concurrent load
with the symptom `error: invalid object <sha> for '<path>' / Error building
trees`, plus `TempDir RemoveAll cleanup: ... .git/objects: directory not
empty`. This reproduces with **zero** git env vars set — it is not the G-0250
leak.

Root cause: git runs `gc --auto` after commits, and with `gc.autoDetach=true`
(the default) the gc runs in a **background** process. Under load that detached
gc is slow, so it races (a) the fixture's own subsequent git commands —
repacking/pruning a loose object another command expects, yielding "invalid
object / Error building trees" — and (b) `t.TempDir`'s `RemoveAll`, which trips
over `.git/objects` while gc is still writing it ("directory not empty").

This is the likely identity of the recurring "flake under full-suite parallel
load; passes isolated" attributed to the `G-0097` family, and the actual cause
behind the original trace that motivated `G-0250`. The repo's `-parallel 8` cap
on race runs reduces the incidence but does not remove the root cause; uncapped
runs and pre-commit-hook runs still hit it.

## Evidence / reproduction

Clean env (no GIT_* vars), uncapped parallelism on a 20-core box:

```
go test -count=3 ./internal/check/ ./internal/verb/ ./internal/initrepo/ \
  ./internal/gitops/ ./internal/cellcoverage/ ./internal/trunk/
# -> FAIL internal/check: invalid object ... / Error building trees
#    + TempDir RemoveAll cleanup: .git/objects: directory not empty
```

Disabling auto-gc makes the identical run pass:

```
GIT_CONFIG_GLOBAL=<file with [gc] auto=0, autoDetach=false> go test -count=3 ...
# -> all green
```

## Proposed fix

Disable git auto-gc for every git-shelling test package at the same chokepoint
that handles `G-0250`: have the shared `internal/testsupport` TestMain helper
export `GIT_CONFIG_COUNT` / `GIT_CONFIG_KEY_n` / `GIT_CONFIG_VALUE_n` setting
`gc.auto=0` and `gc.autoDetach=false` for the test binary's process, so every
fixture `git` invocation inherits it. The policy chokepoint that enforces the
single helper call in each exec-bearing package's TestMain then covers both
root causes with one guarantee.

## Relationship to G-0250

Same subsystem (test git fixtures), same symptom, different mechanism. `G-0250`
is the ambient-env-leak cause (manifests under a git hook); this is the
background-auto-gc cause (manifests under load). Both are addressed in the same
patch.
