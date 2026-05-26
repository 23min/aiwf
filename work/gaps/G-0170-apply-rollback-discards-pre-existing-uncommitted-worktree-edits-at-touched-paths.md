---
id: G-0170
title: Apply rollback discards pre-existing uncommitted worktree edits at touched paths
status: open
discovered_in: M-0143
---
## What's missing

`verb.Apply`'s transactional rollback restores every touched path to **HEAD**, not to its **pre-Apply working-tree state**. When a touched path carried uncommitted worktree content before Apply ran, a failed commit (e.g. a flaky pre-commit hook) silently discards that content.

The acute instance is `aiwf edit-body` **bless mode**: its input *is* the user's uncommitted working-copy edit (`internal/verb/editbody.go` reads `workingBytes` via `os.ReadFile`). On a commit failure, `applyTx.rollback` runs `git restore --staged --worktree -- <path>` (`internal/verb/apply.go:358-371`), reverting the file to HEAD and erasing the hand-authored edit. The operator then sees a misleading "no changes to commit" on any retry (`editbody.go:118`) — the edit is already gone.

The bug is **general**, not bless-specific: `aiwf promote X` (or any mutating verb) while `X` has *unrelated* uncommitted edits will, on a failed commit, restore `X` to HEAD and lose those edits too. Bless mode just makes it certain rather than incidental, because the edit is the verb's input.

Discovered during the M-0143 wrap: the policies pre-commit hook flaked (G-0097/G-0127 shared-`.git` contention), the bless commit aborted, and the appended Reviewer-notes section was reverted to HEAD. No permanent loss that time (the content was reconstructable), but a human hand-authoring prose with no second copy would lose it with only a confusing "no changes" as signal.

## Why it matters

This converts a *retryable* infrastructure flake into *silent data loss* of un-committed work. It also violates the kernel principle "framework correctness must not depend on the LLM's/operator's behavior": today the only guard is **remembering** to use `--body-file` (whose input lives outside the worktree) instead of bless mode under contention. The guarantee should be mechanical.

`Apply`'s contract states the repo ends up "exactly as if Apply had never been called" (`apply.go:27`). That holds only when the touched paths were clean before Apply; the implementation conflates "pre-Apply state" with "HEAD." The fix is to make the contract literally true for any starting state.

## Proposed fix shape (the clean, general fix — not a bless-mode patch)

Make `applyTx` rollback restore touched paths to their **captured pre-Apply worktree bytes**, symmetric with how it already preserves the user's pre-existing *index* via the stash/pop machinery (`apply.go` `stashed` field):

- Before the plan writes anything, capture per touched path either the pre-Apply worktree bytes or an "absent at HEAD" marker (the existing `createdFiles` list already encodes the absent case — generalize it to a `preApply map[string][]byte`, nil = absent).
- On rollback: created files are removed (unchanged); for pre-existing touched paths, unstage (`git restore --staged`) **and overwrite the worktree file with the captured bytes** rather than letting `git restore --worktree` pull from HEAD/index.
- On success (`committed == true`): rollback stays a no-op (unchanged).

Net effect: a failed mutation leaves the worktree exactly as the operator left it — bless-mode edits and unrelated dirty files alike survive, and a retry (after the flake clears or the real error is fixed) starts from the same state. This fixes the whole class, not just bless mode.

Verify the interaction with the existing pre-existing-stage stash/pop so the index and worktree restorations compose correctly.

## Alternatives considered (rejected as duct tape)

- **Bless-mode-only stash of its own edit.** Fixes the acute symptom but leaves the general class (uncommitted edits + any verb + failed commit) unfixed. The root is in `Apply`, so the fix belongs there.
- **"Use `--body-file` for bless commits."** Avoidance, not a fix — and depends on the operator remembering, which the kernel philosophy forbids. Keep it only as an interim operator note, not the resolution.
- **Dry-run the pre-commit hook before Apply so the commit can't fail.** Fragile: the hook is not the only failure source, and a non-deterministic flake can pass the dry-run then fail the real commit. Does not address the root (rollback destroys pre-Apply worktree state).

## Related operator caveat

A blind retry wrapper around bless mode is actively harmful: the first failure already destroyed the input, so the retry runs against a clean worktree and reports "no changes to commit," masking the real cause. Retry wrappers are only safe for idempotent-input commands (`--body-file`, `promote`). Once the fix lands, bless-mode retry becomes safe.
