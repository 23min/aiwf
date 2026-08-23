---
id: G-0621
title: Promote branch guard suggests a checkout that fails under worktrees
status: addressed
priority: medium
addressed_by_commit:
    - ef6ed2edc
---
## What's missing

`internal/verb/promote_branch_guard.go` refuses an activating promote that would
land on the wrong branch, and its message tells the operator how to recover:

```text
aiwf promote M-NNNN in_progress: refusing to land on "milestone/M-NNNN-<slug>" —
this activation is expected on "epic/E-NNNN-<slug>" (...); `git checkout
epic/E-NNNN-<slug>` and retry, or use `--force --reason "..."` to override
```

The remedy it names fails in the situation that produces it. An activating
milestone promote is expected on the parent epic's branch, and under the worktree
convention that branch is held by the epic's own worktree — so `git checkout` on
it refuses. Measured on 2026-08-22 in a scratch repository, running the message's
own command from the milestone worktree:

```text
$ git checkout epic/E-0002-second-epic
fatal: 'epic/E-0002-second-epic' is already used by worktree at '.../epic2wt'
exit=128
```

The operator is left with the refusal, a remedy that also refuses, and `--force`
— which the message offers alongside, and which is a sovereign override rather
than a fix for being in the wrong directory.

The neighbouring surface does not share the defect: the `promote-on-wrong-branch`
check rule's remediation hint says to land future activations on the parent
branch and names no command. The other `git checkout` strings under `internal/`
are `git checkout -- <path>` file restores, which do not move a branch.

## Why it matters

The guard exists to stop a sovereign act landing where it does not belong, and it
works. What it hands back is a dead end, at the moment the operator is least able
to reason about it — mid-ritual, having already been told they are in the wrong
place.

The correct move is to change directory into the worktree that already holds the
branch, which moves nothing and cannot fail that way, and the message has the
information needed to say so: it knows the expected branch, and the holding
worktree is one `git worktree list` away. The rituals were corrected to that shape
under G-0620; this message is the same instruction in Go, where no ritual edit
reaches it, and it is the copy a reader meets first because it arrives at the
moment of failure.
