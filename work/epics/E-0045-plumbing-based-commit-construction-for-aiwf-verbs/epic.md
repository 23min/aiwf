---
id: E-0045
title: Plumbing-based commit construction for aiwf verbs
status: proposed
---
## Goal

Replace aiwf's fragile `git stash`-based per-verb commit isolation with a plumbing-based commit-construction primitive (temp index + `commit-tree`) that never mutates the live index or worktree — making per-verb commit atomicity robust by construction, and giving aiwf a single, reusable commit-construction substrate.

## Context

Every mutating verb today isolates its commit by `git stash push --staged` then `git commit` (`internal/gitops/gitops.go`). The stash reverts the worktree for staged renames and collides with untracked files at the old paths, aborting into a silent half-state (G-0275, the immediate fail-loud floor, already shipped). G-0276 is the strategic retirement of the fragile primitive. Separately, G-0281 wants to file gaps onto a never-checked-out ref via the same `commit-tree` plumbing. Both are "how does aiwf construct a commit?" — so they share a substrate.

## Scope — two sequenced milestones, one ADR each

- **Milestone 1 — the gitops commit-construction primitive (closes G-0276).** A `gitops` primitive that constructs a commit from `parent tree + a set of blob writes` against a throwaway index (`GIT_INDEX_FILE` + `git write-tree` + `git commit-tree`), never reading or writing the live index/worktree. Retrofit every mutating verb (`verb.Apply`) onto it; delete `StashStaged`/`StashPop`. Factor the commit-construction core as a clean, reusable seam (its AC pins reusability). Relocate the pre-commit validation per the decision below.

- **Milestone 2 — opt-in gaps inbox (closes G-0281).** Wrap M1's primitive with a never-checked-out-ref + fetch + compare-and-swap + opt-in-push layer to file gaps without touching HEAD/index/worktree. `depends_on` M1. Starts only after M1 wraps — M2 is the second consumer that proves M1's seam is actually reusable.

## Decisions recorded

1. **Plumbing road.** Verb commits are built via temp-index + `commit-tree`, not via the live index. Rejected: index save/reset/restore — preserves hook firing but still mutates the shared live index (a crash window; exposure to the shared-worktree index race). Plumbing is strictly more isolated and hook-install-independent.
2. **Validation relocation (Option C).** Since `commit-tree` fires no hooks: the verb owns shape *by construction*; the redundant per-commit shape-check is dropped; secret/path-leak scanning (gitleaks, G-0103) relocates to **pre-push**; pre-push `aiwf check` remains the authoritative gate. The never-checked-out-ref / push-inside-a-verb decision for the gaps-inbox is deferred to M2's own ADR.

## Sequencing

M1 ships first so the correctness win (retiring the stash) isn't gated on the opt-in M2 feature. M2 follows, reusing M1's primitive.

## Success criteria

- `StashStaged`/`StashPop` are gone; no verb reverts the worktree to isolate its commit.
- The G-0275 collision class (staged rename vs untracked path) is structurally impossible.
- The commit-construction core is one primitive both M1 (verbs) and M2 (gaps-inbox) use — no parallel commit-construction path.
- Verb-commit validity no longer depends on a pre-commit hook being installed.

## References

G-0276 (M1 driver), G-0281 (M2 driver), G-0275 (shipped fail-loud floor), ADR-0001 (related: mint ids at trunk integration). Kernel commitments #3 (pre-push chokepoint) and #7 (one commit per verb) are touched — see the M1 ADR.
