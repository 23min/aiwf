---
id: G-034
title: Mutating verbs sweep pre-staged unrelated changes into their commit
status: addressed
addressed_by_commit:
  - 890ab01
---

Resolved in commit `(this commit)` (fix(aiwf): G34 — isolate verb commits from user's pre-staged work via stash). `verb.Apply` (and `aiwf render roadmap`, the only other mutating call site outside Apply) now check `gitops.StagedPaths` before running. Two halves:

1. **Conflict guard.** When a path the verb is about to write is *also* pre-staged by the user, refuse before any disk mutation — the user's staged content and the verb's computed content can't both land. The error names every conflicting path and points at `git restore --staged` / `git stash`.
2. **Stash isolation.** When the user has unrelated staged work, push it onto the stash (`git stash push --staged`), run the verb's normal `git commit -m <msg>` flow (so the pre-commit STATUS.md hook composes correctly — its `git add STATUS.md` lands in the verb's commit as designed), then pop the stash so the user's WIP is back in the index for their next commit. Pop also runs on rollback paths so a partial failure doesn't strand the stash.

Initial attempt scoped the commit via `git commit -- <verbPaths>` (--only semantics) but that compose poorly with hooks that auto-`git add` extra files: git captures the hook's addition in HEAD but resets the post-commit index to only the explicitly-named paths, leaving a phantom staged-deletion behind. The stash approach gives the verb a clean index to commit against, hooks behave normally, and the user's stage round-trips intact.

New `gitops.StashStaged` / `gitops.StashPop` / `gitops.StagedPaths` primitives (StagedPaths uses `-z` to handle paths with spaces/newlines safely). Tests: `TestApply_PreservesUnrelatedStagedChanges`, `TestApply_RefusesConflictingPreStagedPath`, `TestApply_AllowEmptyPreservesUnrelatedStaged`, `TestApply_AllowEmptyOnCleanIndex` cover the verb seam; `TestStashStaged_PushPopRoundTrip` and `TestStagedPaths` pin the gitops primitives. Manual smoke confirms the user's reproducer (`git add unrelated.go && aiwf add gap …`) now lands a single-path gap commit (plus hook-regenerated STATUS.md) with `unrelated.go` still staged for the user.

---

<a id="g35"></a>
