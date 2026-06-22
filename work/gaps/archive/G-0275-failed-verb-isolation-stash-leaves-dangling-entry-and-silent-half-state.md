---
id: G-0275
title: Failed verb-isolation stash leaves dangling entry and silent half-state
status: addressed
addressed_by_commit:
    - ace29e37
---
## Problem

`aiwf`'s verb-commit isolation (`internal/verb/apply.go:49-93`) sets the user's
pre-staged index aside with `git stash push --staged` so the verb's commit
captures only the entity mutation. When that stash **fails**, aiwf leaves a
confusing half-state: the verb does not commit (the mutation is not applied), yet
a dangling stash entry is left on the stack, and the error gives the operator no
actionable recovery path.

Observed while wrapping a milestone in a downstream consumer repo (its `M-0026`,
unrelated to this tree). Three routine `aiwf promote <m>/<ac> --phase green`
invocations each aborted with:

    aiwf promote: stashing pre-staged changes for verb isolation: git stash: exit status 1
    error: src/schemas/note.ts: already exists in working directory
    error: patch failed: src/core/manifest-schema.ts:1
    error: src/core/manifest-schema.ts: patch does not apply

Net effect: no frontmatter mutation applied (the ACs stayed open), three dangling
`aiwf pre-verb stash` entries left behind, working tree otherwise intact (build +
tests green). No data loss, but a genuinely alarming silent partial state — a
clean refusal would have been strictly better.

## Root cause

Two compounding defects:

1. **The stash primitive is not atomic on this tree shape.** `git stash push
   --staged` reverts the staged changes, and reverting a staged rename means
   recreating the source path in the worktree. The triggering tree held staged
   `git mv` renames whose old paths were occupied by untracked files (back-compat
   shims) plus a rename-with-modify — so the revert hit "already exists in working
   directory" / "patch does not apply" and git exited 1 *after* it had already
   created the stash commit.

2. **aiwf's error path assumes the stash is atomic.** At `apply.go:62-65`:

       if stashErr := gitops.StashStaged(...); stashErr != nil {
           return fmt.Errorf("stashing pre-staged changes for verb isolation: %w", stashErr)
       }
       tx.stashed = true   // only set AFTER the call — never reached on a stash failure

   The deferred rollback keys its stash cleanup off `tx.stashed`, still `false` on
   this path, so the stash git actually created is never dropped. The implicit
   "error ⟹ no stash created" assumption is false for this failure mode; the
   dangling entries accumulate, one per attempt.

## Direction (patch scope)

Make the stash-isolation step transactional and fail-loud, without touching the
happy path:

- On a `StashStaged` error, detect whether a new stash entry was created (compare
  the stash ref / `git stash list` before and after) and drop only that entry, so
  no dangling stash is left behind.
- Return an actionable refusal naming the cause and the fix — along the lines of
  "couldn't isolate your pre-staged changes (staged renames with untracked files
  at the old paths can't be set aside safely); commit or unstage them, then retry"
  — rather than surfacing the raw `git stash: exit status 1`.

This converts the silent half-state into a clean refusal with no residue, and it
covers *every* stash-failure shape generically rather than enumerating the toxic
ones. It is the safety floor; the fragile primitive itself is retired separately
in `G-0276`.

## Test

An integration test that reproduces the shape — stage a `git mv` rename, drop an
untracked file at the old path, run a mutating verb — and asserts (a) the verb
fails with the actionable message and (b) the stash list is unchanged afterwards
(no dangling entry). Today's code leaves a dangling stash, so the test fails red
before the fix.

## Provenance

Discovered wrapping a milestone in a downstream consumer repo (2026-06-22) while
dogfooding aiwf. Immediate safety floor of a two-gap pair; sibling `G-0276`
retires the stash primitive that makes this failure possible (and subsumes the
class), but is a larger redesign — this gap ships first via wf-patch. The
isolation mechanism hardened here was introduced for `G-0034` (mutating verbs
sweeping pre-staged changes into their commit).
