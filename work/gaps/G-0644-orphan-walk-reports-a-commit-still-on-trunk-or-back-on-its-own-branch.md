---
id: G-0644
title: Orphan walk reports a commit still on trunk or back on its own branch
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

It never asks whether the older tip is still reachable. When trunk reaches it, or
the branch whose reflog recorded the update has moved back onto it, the AI commit
is in history where an audit reaches it, from trunk or from that branch — there
is no orphan, and the finding is a false positive. G-0205, the gap the rule was
written for, asks for a commit where "the oracle can't reach it from any local
branch", and M-0161's AC-5, which specified the walk, says of a force-push after
which "no SHA is unreachable; detection finds no orphans".

Reachable from trunk, measured in a throwaway repository:

- `epic/E-0002-y` carries two `ai/` commits, merged to `main`, then the branch is
  moved back with `git reset --hard HEAD~2`.
- older tip an ancestor of newer tip → **no**, so the walk reports an orphan.
- older tip reachable from `main` → **yes**, so the commit is on the mainline and
  fully auditable.

Back on its own branch, and before that on another branch. Reproduced 2026-11-01
with aiwf built from `main` at `c06ffdaf9` (`vcs.modified=false`), git 2.51.1,
Linux, with the hooks `aiwf init` installs. On `main`, `aiwf add epic` and
`aiwf promote E-0001 active`; then `git checkout -b epic/E-0001-some-epic` and,
on that branch:

```
$ aiwf authorize E-0001 --to ai/claude
aiwf authorize E-0001 --to ai/claude
exit 0
$ aiwf retitle E-0001 "Retitled by the agent" --actor ai/claude --principal human/peter
aiwf retitle E-0001 -> "Retitled by the agent"
exit 0
$ git log -1 --format='%h %s'
3472881 aiwf retitle E-0001 -> "Retitled by the agent"
$ git branch keep
exit 0
$ aiwf check
epic-active-no-drafted-milestones (warning) × 1 — epic E-0001 is active but has no milestones at status draft
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit, body-section gate and instruction-file fence skipped

2 findings (0 errors, 2 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
exit 0
$ git reset -q --hard HEAD~1
exit 0
$ git branch --contains 347288118049979385d0ce27671ec962bcb6c819
  keep
$ aiwf check
epic-active-no-drafted-milestones (warning) × 1 — epic E-0001 is active but has no milestones at status draft
isolation-escape-orphaned-ai-commit (warning) × 1 — AI-actor commit 34728811 was orphaned by a non-fast-forward update on "epic/E-0001-some-epic" at 2026-11-01 14:32:51 +0000; entity E-0001, actor "ai/claude"
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit, body-section gate and instruction-file fence skipped

3 findings (0 errors, 3 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
exit 0
$ git merge -q --ff-only keep
exit 0
$ git log -1 --format='%h %s'
3472881 aiwf retitle E-0001 -> "Retitled by the agent"
$ aiwf check
epic-active-no-drafted-milestones (warning) × 1 — epic E-0001 is active but has no milestones at status draft
isolation-escape-orphaned-ai-commit (warning) × 1 — AI-actor commit 34728811 was orphaned by a non-fast-forward update on "epic/E-0001-some-epic" at 2026-11-01 14:32:51 +0000; entity E-0001, actor "ai/claude"
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit, body-section gate and instruction-file fence skipped

3 findings (0 errors, 3 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
exit 0
```

After the fast-forward `3472881` is the branch's own tip, and the walk still
reports it orphaned on that branch; the provenance audit, which walks the commits
reachable from the checked-out branch (`provenanceCommitsFromHead`,
`internal/cli/check/provenance.go`), reaches it. G-0755's reproduction meets the
same shape after an ordinary rebase and merge: the warning there names `1ebff00`,
which the merge made an ancestor of the branch's tip again.

Between the reset and the fast-forward, `3472881` is held only by `keep`, and the
walk reports it too. Whether it should is open. Checked from the rewound branch,
no audit reaches the commit: the provenance audit walks only that branch, and the
isolation-escape oracle indexes trunk and the branches
`branchparse.ParseEntityFromBranch` accepts, which `keep` is not. G-0205 calls
the oracle's silence on such a commit "correct only when the commit is genuinely
on a branch the oracle doesn't index (the documented gap in G-0203)", a policy
rather than an audit that reaches it.

The precondition is that trunk, or the branch the walk read, reaches the older
tip when the check runs. Two neighbouring shapes do not trigger it, both
measured: resetting a merged branch to trunk itself leaves the previous tip an
ancestor of the new one, so the pair is a fast-forward the walk skips; and a
branch rewound before its work ever reached trunk yields a genuine orphan,
reported correctly — with another branch still holding the commit, it is the open
case above.

One false finding per rewind is an upper bound. `RunOrphanedAICommits`
deduplicates per SHA, and a pre-rewind tip carrying a human actor is filtered
before it reaches a finding.

## Why it matters

The finding exists to surface AI commits that a non-fast-forward update removed
from every ref — work that landed nowhere and that no audit reaches. A commit on
trunk, or back on its own branch, is the opposite of that case, and reporting it
teaches the reader that the finding does not mean what it says.

An operator who meets one of these has no way to tell it from a real orphan
without checking reachability by hand, which is the check's own job.

Neither remedy the hint offers clears it. Restoring the commit puts it back on
the branch, which is the same-branch shape: the fast-forward above is such a
restore, and the warning stays. What remains is `aiwf acknowledge illegal
<sha>`, a permanent sovereign record, with a reason, written for a commit that
nothing is wrong with.

## Resolution shape

Filter the orphan by reachability before reporting: an older tip reachable from
the trunk head, or from the current tip of the branch whose reflog recorded the
update, is not orphaned.

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
