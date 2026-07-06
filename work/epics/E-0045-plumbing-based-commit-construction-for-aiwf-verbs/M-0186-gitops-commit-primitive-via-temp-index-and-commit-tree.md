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
      status: met
      tdd_phase: done
    - id: AC-3
      title: verb.Apply retrofit onto primitive with git-stash isolation removed
      status: met
      tdd_phase: done
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

### AC-2 — post-commit reconciliation touches only the verb's written paths

Implemented · commit ef99d581 · tests 6/6

`internal/gitops/reconcile.go` adds `ReconcilePaths`, the deliberately
narrow follow-up to `CommitTree`: for each write it re-hashes the
content (a cheap content-addressed no-op repeat of the hash `CommitTree`
already wrote) and stages it into the *live* index via `update-index
--add --cacheinfo`, one path at a time — never touching any other
staged or unstaged path. This is the step that makes `git status`
report clean for a verb's own written paths without a `git add -A`-style
sweep that would silently re-stage the caller's unrelated pending work.

Branch-coverage audit: every branch has a direct test, no
`//coverage:ignore` needed. The hash-object failure branch is exercised
by a read-only object database (mirroring AC-1); the update-index
failure branch is exercised by a stale `.git/index.lock` left behind by
a crashed or still-running git process — a real, deterministically
reproducible failure mode, not a contrived one.

`wf-vacuity` mutation probe: 3 mutations (wrong file mode on the
cacheinfo string, a hardcoded wrong path ignoring the write's real
path, dropping the `--add` flag) were all caught cleanly by the
existing `git diff --cached` assertions on the first attempt — no
surviving mutants, no weak assertions to strengthen.

Post-vacuity confidence check added
`TestReconcilePaths_OverwritesExistingTrackedFile`, mirroring AC-1's
same real-world-shape check: reconciling a path whose live index entry
still holds the *old* pre-commit content (the primary case per AC-3 —
`promote`/`edit-body`/`cancel` all rewrite existing entity files, not
add new ones). No bug found; landed as a permanent regression test.

### AC-3 — verb.Apply retrofit onto primitive with git-stash isolation removed

Implemented · commit 4718d496 · tests 46/46 (verb) + 6/6 (gitops)

`internal/verb/apply.go` no longer stashes staged changes for isolation;
`Apply` now commits directly via `gitops.CommitTree` /
`gitops.ReconcilePaths`. Two primitive extensions made this possible:

- `CommitTree`/`ReconcilePaths` gained a `removes []string` parameter so a
  rename evicts its old path from the temp/live index via `update-index
  --force-remove` (no-op if the path is already absent).
- `CommitTree` gained unborn-HEAD support: `IsRepo` distinguishes "not a
  repo" (hard error) from "repo exists, no commits yet" (build a root
  commit — skip `read-tree`, omit `-p` on `commit-tree`, and use the
  empty-string CAS oldvalue on `update-ref HEAD` that is git's own idiom
  for "ref must not already exist"). Without this, every zero-commit test
  repo failed with "ambiguous argument 'HEAD'".

`gitops.RunPostCommitHook` was added because `git commit-tree` +
`update-ref` fire no git hooks at all (unlike `git commit`), silently
breaking `STATUS.md` regeneration (G-0112). It resolves the hook path via
the existing `HooksDir` helper, checks the executable bit, and runs it —
swallowing the hook's exit status entirely, matching git's own tolerance
for `post-commit` per githooks(5).

`Apply`'s commit write-set is computed by a new `gatherCommitOps`, which
reads current disk state *after* both Apply phases have run (rather than
trusting `op.Content`), so a plan that moves a path and then rewrites it
at the same final location lands the correct final bytes; it also
descends into directories for `OpMove` destinations that are directories.
A `wf-vacuity` mutation probe found this directory-walk had no test
asserting the *committed tree's* content (only worktree state) — fixed by
`TestApply_DirectoryMoveWithNestedFile_CommitTreeIsCorrect`, which reads
back via `git ls-tree -r` / `git show` rather than the worktree.

`gitops.StashStaged`/`StashPop`/`StashTopRef`/`StashDrop`/`Restore` and
`verb.classifyGitError`/`dirMove` were all deleted as dead code once
commit construction no longer touches `.git/index.lock` or the stash;
`internal/gitops/no_stash_test.go` AST-walks the package to keep the four
retired `Stash*` symbols from reappearing.

The most significant change was a mid-implementation redesign of
`applyTx`'s rollback bookkeeping, captured in D-0029: a `wf-rethink` pass
(fresh subagent, no sight of the shipped implementation) and an
independent second-opinion review together found that the original
two-mechanism, fixed-order rollback (directory moves reversed first, then
captured file content restored) silently corrupts a plan that moves a
directory and rewrites a file nested inside it before failing — reachable
today via `reallocate`/`rewidth` on epic entities. `applyTx` now records
one `undoStep` per completed mutation (`moveUndo`/`writeUndo`) in
execution order and replays it strictly LIFO, which is correct by
construction for any interleaving rather than only the interleavings
today's verbs happen to produce. `TestApply_RollsBackOnDirectoryMoveThenNestedRewrite_BothSymptoms`
pins the composite scenario red-to-green.

Branch-coverage audit and a second `wf-vacuity` pass over the rewritten
journal found and fixed two more surviving mutants (LIFO→forward-order,
and an inverted "already gone, skip" check in `moveUndo.undo`) — both now
caught by dedicated tests.

## Decisions made during implementation

- D-0029 — Unify applyTx rollback into a single LIFO undo journal.

## Validation

<!-- Pasted at wrap. -->

## Deferrals

- (none)

## Reviewer notes

- (none)

