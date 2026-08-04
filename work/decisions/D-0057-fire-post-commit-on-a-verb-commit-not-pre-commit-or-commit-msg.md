---
id: D-0057
title: Fire post-commit on a verb commit, not pre-commit or commit-msg
status: proposed
---
## Question

A verb commit is not built by `git commit`. It is built by plumbing —
`commit-tree` against a throwaway index, then `update-ref` — which fires no git
hooks at all. Whatever a consumer has installed, none of it runs.

That leaves a choice the porcelain never posed: which hooks, if any, should be
fired back explicitly? Restoring all of them sounds like the neutral answer,
since it is what `git commit` would have done, and it is the one that has to be
argued against rather than for.

## Decision

**Fire post-commit, and only post-commit.** It runs explicitly after the commit
lands, best-effort, with its exit status treated as informational — matching
git's own tolerance for that hook.

**Do not fire pre-commit or commit-msg**, on a verb commit, ever. A consumer's
`pre-commit.local` and `commit-msg.local` do not run on a mutation aiwf makes.

## Reasoning

The three hooks differ in what they are able to do, and that difference is the
whole argument.

`post-commit` observes. It runs after the commit exists, it cannot change it,
and its failure cannot unmake it. Firing it restores the parity consumers
actually depend on — STATUS.md regeneration is the case that prompted it — at no
risk to the commit.

`pre-commit` can refuse, and `commit-msg` can rewrite. Either capability breaks
something the kernel promises rather than merely inconveniencing it. Every
mutating verb produces exactly one commit or none, decided by the verb before
anything is written; a consumer hook that refuses mid-sequence introduces a third
outcome the verb has no way to report and the caller has no way to undo. And
`aiwf history` reads structured trailers that the verb composed deliberately; a
`commit-msg` hook that rewrites the message can invalidate them, leaving history
that no longer answers the queries the kernel builds on it.

The symmetry argument — "restore all three, it is what porcelain does" — assumes
the consumer's hooks were written for aiwf's commits. They were not. They were
written for hand commits, where refusing and rewriting are exactly right, because
a human authored the content and can respond. A verb has already computed and
validated its change; there is no author present to answer a refusal.

Deliberately unresolved: whether a consumer could opt in to pre-commit for verb
commits. Nobody has asked, and the answer would need to say what a refusal means
for the verb's exit code and the operator's next move.

## Consequences

**A consumer's `pre-commit.local` and `commit-msg.local` are silently inert on
every aiwf mutation.** Silent is the honest word — nothing announces it, and a
consumer who installed one to enforce a house rule will find it holds for their
hand commits and not for `aiwf promote`.

**Revisit on a consumer reporting it.** Specifically: someone whose own
pre-commit hook did not fire and who has a legitimate reason to want it. That
report is the evidence this decision cannot supply for itself, because the
alternative was never shipped.

**The asymmetry is documented at the seam** — `CommitVerbChange` explains why
post-commit is fired. It does not explain why the other two are not, which is the
half a reader has to reconstruct, and the half this decision exists to hold.
