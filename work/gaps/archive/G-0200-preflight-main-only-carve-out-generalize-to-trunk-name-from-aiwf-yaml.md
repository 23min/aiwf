---
id: G-0200
title: 'preflight main-only carve-out: generalize to trunk-name from aiwf.yaml'
status: addressed
discovered_in: M-0104
addressed_by:
    - M-0161
---
M-0104's AC-4 carve-out hardcodes `CurrentBranch == "main"` literally
([`internal/verb/authorize.go:300`](../../internal/verb/authorize.go)):

```go
mainAndRitualFuture := opts.CurrentBranch == "main" &&
    branchparse.ParseEntityFromBranch(branchExplicit) != ""
```

The repo's `aiwf.yaml.allocate.trunk` defaults to `refs/remotes/origin/main`
and is configurable; the local short name "main" is the right value for
this repo's trunk, but the carve-out's literal does not consult that
config. Operators on repos using `master` (or any other trunk name) cannot
exercise the step-4 pattern of `aiwfx-start-epic` — they must use either
the implicit-ritual-current path or `--force --reason`.

## Why parked under M-0104

The M-0104 spec at AC-4 line 78 says *"from a checkout on `main`"*
literally; the pre-decided-design at line 54 names "main" the same way.
The M-0104 scope is to make the ritual work for this repo (which uses
"main"). Generalizing to configurable trunk is internally consistent with
the rest of `internal/branchparse/`'s hardcoded prefixes — but it is a
separate kernel-knob decision and was not what M-0104's spec scoped.

## Disposition

Reviewer feedback during M-0104 Cycle 1 (AC-4) flagged this as a real
layering smell (a verb-layer gate carries a hardcoded value that the
allocator layer already configures via `aiwf.yaml.allocate.trunk`). The
verb-layer check is local-branch, not remote-ref, but the short-name
("main" vs "master") would still be a reasonable derivation from the
trunk ref.

Addressed by: a new ADR (if the choice deserves an architectural
record) or a small milestone that:

1. Adds a `TrunkBranchShortName()` helper on `Config` deriving the short
   name from `AllocateTrunkRef()` (e.g., `refs/remotes/origin/main` → `main`).
2. Replaces the literal `"main"` in the M-0104/AC-4 carve-out with that
   helper.
3. Adds a cell to the consolidation milestone (M-0158) covering the
   non-main trunk-name case.

Out of scope for M-0104.
