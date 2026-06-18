---
id: G-0205
title: BranchOracle silent on force-pushed-away violating commits
status: addressed
discovered_in: M-0158
addressed_by:
    - M-0161
---
M-0106's `BranchOracle` indexes commits via
`git rev-list --first-parent <branch>` at construction time. The
index is a snapshot of reachable commits **at the moment of the
oracle's build**, not a live query.

## Failure mode

A subagent makes a violating commit on the wrong branch. Before
the operator runs `aiwf check`, the AI (or operator) force-pushes
the wrong branch, rewriting history so the violating commit is
no longer reachable from any ref.

When the oracle is built (or rebuilt during a later check), the
violating SHA isn't in any branch's first-parent index. The rule
treats unknown branch sets as "silent" → no finding.

**The escape goes silent.** The violating commit exists in the
reflog (operator can `git reflog | grep`), but the kernel check
emits nothing.

## What's needed

The oracle / rule should detect "SHA we cared about (it carries
ai-actor + entity trailers) but the oracle can't reach it from any
local branch" as a distinct case and surface it as an `orphaned-ai-commit`
warning. The current "unknown branch → silent" policy is correct
only when the commit is genuinely on a branch the oracle doesn't
index (the documented gap in G-0203); it's wrong when the commit
has been orphaned by a force-push.

Detection requires the oracle to know which SHAs the
`isolation-escape` rule was going to inspect (from the trailer
scan in `RunProvenanceCheck`) and compare against the oracle's
index.

## Why parked

The M-0158 honest-scope audit surfaced this. The current oracle
ships with no force-push-detection. Address as part of the
real-world hardening milestone.

## Related gaps

- [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md)
  — oracle's typed-error distinction; the force-push case is a
  specific instance of "lookup succeeded, returned empty, but
  should have surfaced a warning."
