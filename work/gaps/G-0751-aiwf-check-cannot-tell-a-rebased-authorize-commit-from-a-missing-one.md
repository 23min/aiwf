---
id: G-0751
title: aiwf check cannot tell a rebased authorize commit from a missing one
status: open
priority: high
---
## What's missing

`aiwf check` cannot tell an `aiwf authorize` commit that a rebase re-created from one that does not exist. A verb an agent runs in an authorize scope records the scope as the SHA of its `authorize` commit, in `aiwf-authorized-by` (`internal/cli/cliutil/provenance.go:115`). The check looks that SHA up among the scope-opening `authorize` commits in HEAD's history (`buildAuthOpenerIndex`, `internal/check/provenance.go:648`, over the commits `WalkHeadCommits` reads with `git log --reverse HEAD`, `internal/check/head_history.go:94`) and raises `provenance-authorization-missing` at error severity when it is not among them (`internal/check/provenance.go:293-300`). A `git rebase` that re-creates the `authorize` commit gives it a new SHA and leaves every later `aiwf-authorized-by` naming the old one. The scope is still on the branch, opened by the same act for the same agent, and each commit an agent made in it before the rebase becomes an error.

The provenance model chose this behaviour. It lists rebase among the ways the SHA "can become stale later" and validates at read time so that one mechanism "catches both the typo case and the later-rewrite case" (`docs/design/provenance-model.md`, lines 118-119); the defect is that the one mechanism reports the two cases alike. The same document addresses a scope by its SHA on the ground that "SHAs are stable in the absence of force-pushes" (§"Scope id", line 257); the reproduction below fails the same way when the branch is rebased before its first push, with no force-push at all.

A rebase re-creates the `authorize` commit only when that commit is on the part of the branch being rebased. In the shipped rituals that is a milestone scope, which `aiwfx-start-milestone` opens on the epic branch, when the epic branch is rebased onto `main`. An epic scope, which `aiwfx-start-epic` opens on `main`, is not re-created by rebasing the epic branch: with it, two agent verbs on the epic branch and a rebase leave `aiwf check` at no error and the push passes.

Reproduced 2026-10-28 with aiwf built from `main` at `c06ffdaf9` (`vcs.modified=false`), git 2.51.1, Linux, with the hooks `aiwf init` installs, following the shipped rituals up to the wrap merge. On `main`, `aiwf promote E-0001 active`. On `epic/E-0001-spike-epic`, `aiwf promote M-0001 in_progress`, then `aiwf authorize M-0001 --to ai/claude --branch milestone/M-0001-spike-milestone`. On the milestone branch, a code commit, then `aiwf promote M-0001/AC-1 met` and `aiwf retitle M-0001 "Spike milestone, retitled"`, each with `--actor ai/claude --principal human/peter`. Then the `aiwf-verb: wrap-milestone` merge into the epic branch, whose hooked push passes. A second clone pushes an unrelated commit to `main`, and the epic branch is rebased onto it:

```
$ git log --format='%h %s%n%(trailers:key=aiwf-authorized-by)' origin/main..HEAD | grep -v '^$'
4b96be9 chore(milestone): wrap M-0001 — Spike milestone
f0b0b89 aiwf retitle M-0001 -> "Spike milestone, retitled"
aiwf-authorized-by: 5d614b425dd994942372b018db59eb5e631e343c
5d614b4 aiwf authorize M-0001 --to ai/claude
b13b633 aiwf promote M-0001/AC-1 open -> met
aiwf-authorized-by: 5d614b425dd994942372b018db59eb5e631e343c
50190a6 aiwf promote M-0001 draft -> in_progress
269cf99 feat: c1
$ git rebase origin/main
Successfully rebased and updated refs/heads/epic/E-0001-spike-epic.
$ git log --format='%h %s%n%(trailers:key=aiwf-authorized-by)' origin/main..HEAD | grep -v '^$'
1c3855a aiwf retitle M-0001 -> "Spike milestone, retitled"
aiwf-authorized-by: 5d614b425dd994942372b018db59eb5e631e343c
aeb6b9c aiwf promote M-0001/AC-1 open -> met
aiwf-authorized-by: 5d614b425dd994942372b018db59eb5e631e343c
cfeb789 feat: c1
b736182 aiwf authorize M-0001 --to ai/claude
8bd6efb aiwf promote M-0001 draft -> in_progress
$ aiwf check
error provenance-authorization-missing: commit aeb6b9c: aiwf-authorized-by: 5d614b4 does not resolve to an authorize/opened commit — hint: …
error provenance-authorization-missing: commit 1c3855a: aiwf-authorized-by: 5d614b4 does not resolve to an authorize/opened commit — hint: …
…
9 findings (2 errors, 7 warnings)
$ git push --force-with-lease -u origin epic/E-0001-spike-epic     # the pre-push hook runs aiwf check
…
error: failed to push some refs to '…/remote.git'
```

A plain rebase also drops the wrap merge commit; with `git rebase --rebase-merges origin/main` the merge is kept and the same two errors are raised. In the same setup, a rebase before the branch's first push, which needs no force, fails the same way. Merging `origin/main` into the epic branch instead keeps every SHA, and `aiwf check` reports no error and the hooked push passes. `aiwf acknowledge illegal` on the two commits clears both errors and the push passes; once `main` moves again and the branch is rebased again, the two commits have new SHAs, the acknowledgements name the old ones, and both errors return.

## Why it matters

A rebase onto the default branch is one of git's two ordinary ways to bring a branch up to date, and the rituals do not choose between them: `aiwfx-wrap-epic` and `wf-patch` say to "integrate mainline", and no shipped ritual mentions rebase. After a rebase that re-creates a scope's `authorize` commit, every commit an agent made in that scope fails `aiwf check` at error severity, and the branch cannot be pushed with the hooks installed.

A host merge method that rebases the branch would carry those commits onto `main` itself. With a clone that has no hooks standing in for the host, rebasing the epic branch onto `main` and pushing `main`, `aiwf check` on `main` reports both errors and a hooked push of an unrelated commit to `main` is refused: the block reaches everyone who pushes `main`, not only the branch. No host has been measured; the hint aiwf gives squash-merge users names rebase-merge as one of two merge strategies to switch to (`internal/check/hint.go:320`).

The finding's hint is written for the typo case: it says to use the full SHA, which the trailer already holds, and to ratify the commit with `aiwf acknowledge illegal`, a sovereign act per commit that records as illegal a commit made in a scope that is on the branch, and that lasts only until the next rebase. The other way past is `git push --no-verify`, which skips the chokepoint.

## Relates to

D-0034 accepts that an acknowledgement stops applying when a rebase drops the acknowledgement commit; here the rebase keeps it and re-creates the commit it names. G-0396 records a rebase orphaning the SHA in `addressed_by_commit` the same way, and finds a reachability check on that field unsound because it "cannot distinguish a genuinely-lost fix" from "a benign rewritten SHA", the blindness this gap records in `provenance-authorization-missing`.
