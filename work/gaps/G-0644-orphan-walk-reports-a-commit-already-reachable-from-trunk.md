---
id: G-0644
title: Orphan walk reports a commit already reachable from trunk
status: open
priority: medium
---
## What's missing

`WalkOrphanedAICommits` (`internal/check/reflog_walk.go`) decides a reflog entry
is orphaned by asking whether the older tip is an ancestor of the newer one:

```go
if isAncestor(older.SHA, newer.SHA) {
    continue // fast-forward; not orphaned
}
```

It never asks whether the older tip is reachable from trunk. When it is, the AI
commit landed and is auditable from the mainline — there is no orphan, and the
finding is a false positive.

Measured in a throwaway repository:

- `epic/E-0002-y` carries two `ai/` commits, merged to `main`, then the branch is
  moved back with `git reset --hard HEAD~2`.
- older tip an ancestor of newer tip → **no**, so the walk reports an orphan.
- older tip reachable from `main` → **yes**, so the commit is on the mainline and
  fully auditable.

The precondition is that the older tip is reachable from trunk when the check
runs — it landed before the ref moved off it. Two neighbouring shapes do not
trigger it, both measured: resetting a merged branch to trunk itself leaves the
previous tip an ancestor of the new one, so the pair is a fast-forward the walk
skips; and a branch rewound before its work ever reached trunk yields a genuine
orphan, reported correctly.

One false finding per rewind is an upper bound. `RunOrphanedAICommits`
deduplicates per SHA, and a pre-rewind tip carrying a human actor is filtered
before it reaches a finding.

## Why it matters

The finding exists to surface AI commits that a non-fast-forward update removed
from every ref — work that landed nowhere and that no audit reaches. A commit
already on trunk is the opposite of that case, and reporting it teaches the
reader that the finding does not mean what it says.

An operator who meets one of these has no way to tell it from a real orphan
without checking trunk-reachability by hand, which is the check's own job.

## Resolution shape

Filter the orphan by trunk-reachability before reporting: an older tip reachable
from the trunk head is not orphaned.

The ancestry machinery is in hand: `reflog_walk.go` holds a `*CommitDAG` built by
one `git rev-list --all --reflog --parents` (`internal/check/orphan_dag.go`), and
the walk already calls `dag.isAncestor` in the same loop.

The trunk head is not. `trunkShort` is a branch short name — `"main"`, derived by
`Config.TrunkBranchShortName` — while `dag.isAncestor` compares two SHAs over a
`parents` map that carries no ref information. Resolving the tip costs one
`git rev-parse`, or a signature change threading in the
`branchTips map[string]string` that `RunPromoteOnWrongBranch` already takes for
this same comparison; threading it reorders the caller, since
`internal/cli/check/provenance.go` builds `branchTips` after it calls the walk.

Two conditions to settle. `trunkShort` can be empty, which `listRitualHeads`
already guards for; and the default trunk ref is remote-tracking, so the short
name may not resolve — measured, `git rev-parse main` fails in a clone carrying
`refs/remotes/origin/main` but no local `main`. With no resolvable trunk head,
report as today, the way `RunPromoteOnWrongBranch` does. `CommitDAG.isAncestor`
further documents that a SHA not sourced from its own `--all --reflog` build
needs the explicit unknown-key guard re-established.

Separate from the per-ref reflog cost recorded in G-0324, which is `wontfix`:
that one changes what the walk spends, this one changes what it reports. Its
closure reason carries a constraint that binds here too — the walk's subject is
commits a non-fast-forward update removed from a branch, so no filter may key on
whether the *branch* is merged.
