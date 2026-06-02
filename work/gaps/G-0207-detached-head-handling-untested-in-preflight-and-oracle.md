---
id: G-0207
title: Detached HEAD handling untested in preflight and oracle
status: open
discovered_in: M-0158
---
M-0103's preflight and M-0106's `BranchOracle` both query the
current branch via `git symbolic-ref --short HEAD`. In a detached
HEAD state (no symbolic ref), the command exits non-zero; the
helpers ([`internal/cli/authorize/authorize.go`](../../internal/cli/authorize/authorize.go)
`currentBranch`,
[`internal/cli/check/isolation_escape_oracle.go`](../../internal/cli/check/isolation_escape_oracle.go)
implicit) return empty string.

## Failure modes (untested)

### Preflight from detached HEAD

`opts.CurrentBranch == ""` means:
- The implicit-ritual-current path refuses
  (`branchparse.ParseEntityFromBranch("")` returns `""`).
- The M-0104/AC-4 carve-out condition `opts.CurrentBranch == "main"`
  is false.
- The M-0105/AC-6 carve-out condition `ritual(opts.CurrentBranch)`
  is also false.

Outcome: any AI-target authorize from detached HEAD refuses with
`branch-context-required`. Is this the intended behavior? Likely
yes (detached HEAD has no ritual context). But it's not
documented anywhere and not in the test set.

### Oracle from detached HEAD

The oracle doesn't query HEAD; it iterates `git for-each-ref
refs/heads/`. Detached HEAD doesn't affect ref enumeration. So
the oracle should work correctly.

BUT: `aiwf check` itself runs at `cmd.Dir = rootDir`. If the
operator is in a worktree on detached HEAD, the rootDir resolution
chain might surface unexpected behavior. Not tested.

## What's needed

Either:
- Pin detached-HEAD behavior with an explicit test case
  (preflight refuses with a clear error; oracle proceeds normally).
- Document detached-HEAD as a known kernel-silent state.
- Add a top-level `aiwf doctor` check that warns when running from
  detached HEAD.

## Why parked

The M-0158 honest-scope audit surfaced this. The current behavior
is *probably* correct (detached HEAD has no ritual identity, so
refusing/silent is reasonable) but is not explicitly verified.
Address as part of the real-world hardening milestone — at minimum,
add the explicit test cases.
