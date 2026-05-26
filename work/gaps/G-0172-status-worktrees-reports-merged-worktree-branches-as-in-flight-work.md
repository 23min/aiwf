---
id: G-0172
title: status --worktrees reports merged worktree branches as in-flight work
status: addressed
addressed_by_commit:
    - fea72d26
---
## What's missing

`aiwf status --worktrees` (G-0122) renders each worktree's in-flight work from that worktree's **checked-out branch tree**. It does not cross-reference the branch against trunk, so a worktree whose branch is **already merged into trunk** — and whose epic is **terminal on trunk** — is still rendered as live in-flight work.

The branch-local state is read faithfully (it *is* `active` on that branch), but the view draws no distinction between "work genuinely in flight in this worktree" and "this worktree is a merged leftover that should be pruned." The existing `stale/trunk catch-alls` do not cover the merged-branch-with-terminal-epic case.

## Why it matters

The whole value of `--worktrees` is an at-a-glance "what is actually in flight, where" view. A merged-but-unpruned worktree pollutes it with **phantom in-flight work**: the operator sees the epic as `active` and believes work is ongoing, when it has in fact merged, the epic is `done`/archived on trunk, and the only remaining action is to remove the worktree. The false signal inverts the view's purpose — the operator can't trust "active" to mean "live."

## Concrete instance

Discovered at the E-0036 epic wrap. The wrap landed on trunk: `aiwf promote E-0036 done` + `aiwf archive --apply` were committed directly on `main`, and the epic branch (`epic/E-0036-…`) was merged into `main` via the integration merge. But the sibling worktree stayed parked on that now-merged branch, whose tree never received the promote-done commit. `aiwf status --worktrees` then showed `E-0036 … [active]` as in-flight — even though, on trunk, E-0036 was `done`, archived, and the branch was fully merged. The only signal that work was *not* in flight was removing the worktree; the view itself gave no hint.

## Proposed fix shape

Per worktree, before rendering its branch as in-flight:

- Detect whether the worktree's branch is merged into trunk — `git merge-base --is-ancestor <branch> <trunk-ref>`.
- And/or detect whether the worktree's expanded epic is at a **terminal status on trunk** (resolve the epic id against trunk's tree, not the branch's).

When either holds, render the worktree under a **"merged — safe to retire"** annotation (or a dedicated merged-worktrees catch-all) instead of the live in-flight expansion — extending the G-0122 `stale/trunk catch-alls` to this case. The branch-local status can still be shown, but flagged as superseded-by-trunk so the operator reads it as "prune me," not "active."

## Open question

Whether the signal should key on **branch-merged-into-trunk** (cheap, git-only) or **epic-terminal-on-trunk** (semantic, requires loading trunk's tree) — or both. Branch-merged is the more general "this worktree is done" signal; epic-terminal is the more precise "this planning unit closed" signal. They usually coincide but can diverge (a merged branch whose epic is still active elsewhere). The G-0122 view's design intent decides which.
