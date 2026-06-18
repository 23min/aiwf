---
id: G-0203
title: BranchOracle FirstParentBranches conflates lookup-failed with no-branches
status: addressed
discovered_in: M-0106
addressed_by:
    - M-0161
---
M-0106's [`BranchOracle.FirstParentBranches`](../../internal/check/isolation_escape.go)
returns `[]string` with the documented contract: *"An empty/nil
return means the commit is not on any branch the oracle knows
about (treat as 'unknown' — the rule does not fire on unknown-
branch commits, since the kernel cannot confidently classify
them as escaped)."*

This conflates two distinct cases:

1. **"I tried to determine the branches and got nothing"** — the
   oracle ran `git for-each-ref` + `git rev-list --first-parent`,
   the lookups succeeded, the commit is genuinely on zero
   known branches. Treating this as silent is reasonable for
   ritual-naming hygiene (commits to feature/foo are not the
   M-0106 chokepoint's concern).

2. **"I tried to determine the branches and the lookup failed"**
   — ref resolution errored, the rev-list call returned a
   garbage stream, etc. Currently indistinguishable from case 1.
   Silently suppressing a real escape is a watertight violation.

Worse: a hostile dispatcher could push to a branch with a
*malformed-looking* ritual prefix (e.g. `epic/E-9999-tampered`
that bypasses `branchparse` due to a regex edge case, or to a
branch the kernel cannot resolve because of a corrupted ref).
Both fall through to "unknown" → silent.

## What's needed

Change `FirstParentBranches` from `[]string` to
`([]string, error)`, OR split into two methods:

- `FirstParentBranches(sha) []string` — known good empty slice.
- `OracleErrors() []error` — accumulated lookup failures at
  construction; the rule logs/surfaces them as a separate
  diagnostic finding (`isolation-escape-oracle-failure`) so
  silent escapes aren't possible.

The cleaner shape is the second — failures are a one-time
construction concern, not a per-call concern. The kernel's
existing `tree.LoadError` pattern is the precedent.

## Why parked

The M-0106 retrospective surfaced this as F-9. The current
implementation works correctly for non-hostile usage and the
fail-shut-instead-of-fail-open question requires a small
design pass (do we want the oracle's failure to BLOCK push,
or surface as a separate finding, or both?). YAGNI until the
first incident: the existing `gitBranchOracle` is robust against
the common failure modes (empty repo, no branches), and a real
hostile dispatcher is more likely to bypass M-0106 by other
means (e.g., a fake `aiwf-actor: human/...` trailer).

## When to address

When any of:
- A real escape silently passes the kernel and the post-mortem
  traces back to oracle-silent-on-failure.
- The Class-`ClassBranchChoreography` finding set grows enough
  that a dedicated `isolation-escape-oracle-failure` companion
  finding earns its keep.
- M-0158's spec-cell consolidation surfaces a richer
  branch-reachability semantic that demands typed errors.

## Sub-concern (added post-M-0106 second-pass review, N-4)

The current `newGitBranchOracle` at
[`internal/cli/check/isolation_escape_oracle.go`](../../internal/cli/check/isolation_escape_oracle.go)
is **fail-shut at the whole-oracle level**: if `firstParentSHAs`
errors on any single ritual branch (deleted ref mid-check,
packed-refs corruption on one ref, etc.), the entire
`newGitBranchOracle` returns the error. `RunProvenanceCheck` at
[`internal/cli/check/provenance.go`](../../internal/cli/check/provenance.go)
then silently skips the M-0106 rule for the whole check pass.
One stale ref → rule disabled for the entire repo, regardless of
how many branches' first-parent indices were successfully built.

A more resilient shape: skip individual failed refs (recording
each as a future `isolation-escape-oracle-failure` advisory under
the typed-error split this gap proposes), proceed with the rest.

The current behavior is at least fail-shut on the side of
correctness (no false positives from partial info), but it
amplifies the impact of any single ref failure. Address as part
of the typed-error split, or earlier if the failure mode
surfaces in practice.

## Out of scope

The fail-shut vs fail-open question is the bigger design
decision. This gap names the surface; the choice between (a) and
(b) above lives in a subsequent ADR or `D-NNN`. The sub-concern
above narrows the question for the gather-side fault-tolerance
axis specifically.
