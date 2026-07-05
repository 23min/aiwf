---
id: M-0186
title: gitops commit primitive via temp-index and commit-tree
status: in_progress
parent: E-0045
tdd: required
acs:
    - id: AC-1
      title: temp-index primitive never touches the live index or worktree
      status: met
      tdd_phase: done
    - id: AC-2
      title: post-commit reconciliation touches only the verb's written paths
      status: open
      tdd_phase: refactor
    - id: AC-3
      title: verb.Apply retrofit onto primitive with git-stash isolation removed
      status: open
      tdd_phase: red
    - id: AC-4
      title: commit-tree output honors commit.gpgsign parity
      status: open
      tdd_phase: red
    - id: AC-5
      title: commit-construction core exposes a reusable seam
      status: open
      tdd_phase: red
    - id: AC-6
      title: per-commit shape validation dropped from verb-commit path
      status: open
      tdd_phase: red
---
## Goal

Retire the fragile `git stash` verb-commit isolation by building a `gitops` commit-construction primitive that constructs each verb's commit against a throwaway index — never reading or writing the live index or worktree — and retrofit every mutating verb onto it.

## Problem

`internal/gitops/gitops.go` isolates a verb's commit via `git stash push --staged` + `git commit`. The stash reverts the worktree for staged renames and collides with untracked files at the old paths, aborting into a silent half-state (G-0275, fail-loud floor already shipped). The tool's per-verb atomicity is only as robust as `git stash` on an arbitrary tree — and it isn't.

## Approach

- New `gitops` primitive: build a commit from `(parent commit, set of path→blob writes)` via `GIT_INDEX_FILE`=temp → `git read-tree`/`git update-index` → `git write-tree` → `git commit-tree` → `update-ref` HEAD. The live index and worktree are never read or written to isolate the commit.
- Reconcile only the verb's own paths into the live index post-commit so `git status` is clean for them, leaving the user's other staged changes untouched.
- Retrofit `verb.Apply` onto the primitive; delete `StashStaged` / `StashPop` and the worktree-revert path.
- **Reusable seam:** the commit-construction core is factored so the later gaps-inbox milestone wraps it without a second commit path. (An AC pins this.)
- **Validation relocation (Option C):** verb owns shape by construction (drop the per-commit shape-check); relocate gitleaks to pre-push; pre-push `aiwf check` stays authoritative.

## Reversal

Still exactly one commit per verb; "undo" is unchanged (another verb invocation / `aiwf cancel`). Only the mechanism that builds the single commit changes — no new reversal surface.

## References

G-0276 (driver), G-0275 (fail-loud floor), the G-0034 → G-0112 history (why a naive `git commit --only` revert is unsafe — do not re-propose it). ACs authored at start-milestone (contract-first).

### AC-1 — temp-index primitive never touches the live index or worktree

A new `gitops` function builds a commit from `(parent SHA, []PathWrite)` via
`GIT_INDEX_FILE=<temp>` → `git read-tree HEAD` → `git update-index --add
--cacheinfo` (or equivalent) per write → `git write-tree` → `git commit-tree`
→ `update-ref HEAD`. Test: stage unrelated content in the live index and make
an unrelated worktree edit before calling the primitive; assert both are
byte-identical afterward (`git diff --cached` / worktree diff empty).

### AC-2 — post-commit reconciliation touches only the verb's written paths

After the commit-tree commit lands, only the verb's own written paths are
reconciled into the live index (`git status` clean for them) — every other
staged/unstaged path is untouched. Test: pre-stage path A with distinct
content, run a verb that writes path B, assert A's staged content is
unchanged and B is clean in the live index.

### AC-3 — verb.Apply retrofit onto primitive with git-stash isolation removed

`internal/verb/apply.go` builds its commit via the AC-1 primitive instead of
`gitops.Commit` + stash dance. `StashStaged` / `StashPop` / `StashTopRef` /
`StashDrop` and the pre-verb conflict-guard-then-stash path are deleted from
`internal/gitops`. Test: the existing `verb.Apply` test suite (including the
G-0275 dangling-stash regression tests, rewritten for the new failure shape)
passes against the retrofit; a structural test asserts the `Stash*` symbols
no longer exist in the package.

### AC-4 — commit-tree output honors commit.gpgsign parity

`git commit-tree` does not consult `commit.gpgsign` automatically the way
`git commit` does. The primitive must replicate signing behavior explicitly.
Test: with `commit.gpgsign=true` and a test GPG key configured, assert the
resulting commit carries a valid signature (`git verify-commit`); with
`commit.gpgsign` unset, assert no signature is present.

### AC-5 — commit-construction core exposes a reusable seam

The commit-construction logic is factored into an exported entry point
usable by a future verb-commit consumer (the gaps-inbox milestone) without a
second commit path. Test: a structural test asserts the exported function
signature exists and that `verb.Apply` is its only current caller (no
duplicate ad hoc commit-construction logic elsewhere in `internal/verb` or
`internal/gitops`).

### AC-6 — per-commit shape validation dropped from verb-commit path

The pre-commit-hook shape check no longer fires on a verb's commit (`git
commit-tree` fires no hooks) — shape correctness is guaranteed by
construction at the verb layer instead. Test: a verb producing a
deliberately malformed entity (in a test harness that bypasses the verb's
own construction guarantee) is still caught by `aiwf check` at the pre-push
boundary, confirming pre-push remains the authoritative backstop with no
silent gap left by the removed pre-commit check.

## Work log

### AC-1 — temp-index primitive never touches the live index or worktree

Implemented · commit 5be6580d · tests 10/10

`internal/gitops/committree.go` adds `CommitTree` (resolves current HEAD as
parent) and `commitTreeFromParent` (the actual construction: temp
`GIT_INDEX_FILE` → `read-tree` → per-write `hash-object` + `update-index
--cacheinfo` → `write-tree` → `commit-tree` → `update-ref HEAD` with
compare-and-swap against the captured parent). The temp index lives under
the repo's own `.git/` dir, not system `/tmp`. `commitTreeFromParent` is
split out from `CommitTree` specifically so a test can drive the real
construction-and-update-ref path against a deliberately stale parent,
deterministically reproducing a concurrent-HEAD-move race without an
actual race.

Branch-coverage audit: 3 statements (`update-index`, `write-tree`,
`commit-tree` generic failure branches) are `//coverage:ignore`'d — each
requires object-database corruption or a disk-full condition between two
git subprocess calls a few milliseconds apart, not a reachable
input-driven branch. Every other branch (HEAD resolution failure, git-dir
resolution failure, temp-dir creation failure, read-tree failure via a
corrupted tree object, hash-object failure via a read-only objects dir,
the update-ref compare-and-swap failure) has a dedicated test.

`wf-vacuity` mutation probe found 3 surviving mutants on the first pass —
read-tree silently skipped (only caught incidentally by an unrelated
corruption test, not a direct assertion), the written blob's file mode
silently wrong, and trailers silently dropped from the commit message —
all fixed by strengthening the happy-path test to assert the full
resulting tree (`git ls-tree -r`), the exact file mode, and the exact
trailer list via `HeadTrailers`. A 4th probe (dropping the update-ref
compare-and-swap argument) was caught clean on the first attempt.

Post-vacuity confidence check surfaced one more real gap before the
commit gate: every test used a brand-new path, and none exercised
overwriting an already-tracked file — the primary real-world case for
most aiwf verbs (`promote`, `edit-body`, `cancel` all rewrite an existing
entity file). Added `TestCommitTree_OverwritesExistingTrackedFile`
confirming `update-index --add --cacheinfo` replaces the existing index
entry rather than duplicating it, and amended into this commit (local,
unpushed at the time).

Follow-up commit fe07cc7e adds `TestCommitTree_WritesNewNestedPath`,
pinning the other real write shape — a brand-new path under directories
absent from the parent tree (`aiwf add`'s write shape). No bug found;
landed as a permanent regression test rather than a throwaway check.
Final count: tests 11/11.

## Decisions made during implementation

- None yet — all decisions are pre-locked in `## Approach` above.

## Validation

<!-- Pasted at wrap. -->

## Deferrals

- (none)

## Reviewer notes

- (none)

