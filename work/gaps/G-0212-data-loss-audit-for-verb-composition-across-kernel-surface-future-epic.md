---
id: G-0212
title: Data-loss audit for verb composition across kernel surface (future epic)
status: open
discovered_in: M-0159
---
## What's missing

A systematic audit of verb-composition scenarios that could cause data loss or trap operators in states they cannot escape using existing verbs. Scope crosses epic boundaries — verbs interact across the entity surface, not just within the branch-choreography surface E-0030 owns.

Known classes (from history evidence + reasoning):

1. **Reallocate races.** Two operators reallocate the same id on parallel branches; second push hits trunk-collision; resolution path requires `aiwf reallocate` (CLAUDE.md §"Id-collision resolution at merge time"). 26 reallocate commits in repo history (`git log --grep=reallocate`) confirm this is recurring. Under what verb-sequence combinations does a reallocate clash become unrecoverable?

2. **Edit-body races.** Two operators run `aiwf edit-body` on the same entity in different worktrees within minutes. Last writer wins per git's normal semantics, but the lost edits leave no audit trail. The G-0170 incident (`ed0b5014` fix) confirmed apply-rollback can discard uncommitted worktree edits at touched paths.

3. **Archive-during-scope.** An operator archives a parent entity while a scope is active under a child. What happens to the scope's resolution? Does subsequent `aiwf authorize --pause` find the archived parent?

4. **Concurrent verb invocations.** The repolock (`internal/repolock/`) serializes per-repo verb invocations within a process, but cross-process invocations on the same repo via subprocess fan-out are untested in combinatorial scenarios.

5. **Force-push of an `acknowledge-illegal` commit.** The historical SHA the ack referenced may become unreachable; the rule's exemption walk may no longer see the ack from HEAD's reachable history. Override silently revoked. Real-world likelihood: low in this repo (no force-pushes in reflog), but other operators force-push routinely.

6. **Cherry-pick of a force-amend override commit.** Re-authors with the original force-amend trailer set. The kernel's actor-prefix filter sees `human/...` actor and skips the rule — but the cherry-picked work is now on a different branch with no audit trail tying it to the original override.

## Why it matters

aiwf's value proposition is "guarantees about a markdown-and-frontmatter project tree." Data loss in any verb path inverts the value proposition. Operators who hit a data-loss scenario will not use aiwf again. The audit catalogs known + plausible scenarios, then drives a future epic to mechanically prevent each class.

Out of scope for E-0030 (branch-model chokepoint epic). Future-epic candidate. Filing this gap captures the work driver so it survives across sessions.
