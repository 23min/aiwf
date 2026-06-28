---
id: ADR-0022
title: Verb commits built via a temp-index commit-tree primitive
status: proposed
---
## Context

Every aiwf mutating verb isolates its commit by `git stash push --staged` then `git commit` (`internal/gitops/gitops.go`). `git stash` is fragile on staged renames and untracked-vs-tracked path collisions, producing silent half-states (G-0275). The verb already computes a valid, shape-correct mutation by construction, so relying on a worktree-mutating stash to scope the commit is both fragile and unnecessary. Separately, G-0281 needs to build commits via plumbing onto a never-checked-out ref. Both are commit-construction.

## Decision

aiwf builds each verb's commit via a **temp-index + `commit-tree` plumbing primitive** in `internal/gitops/`: construct the commit object from `(parent commit tree, set of path→blob writes)` against a throwaway `GIT_INDEX_FILE`, then `update-ref` HEAD. The live index and worktree are never read or written to isolate the commit.

Because `commit-tree` fires no git hooks, the validation the pre-commit hook performs today relocates (Option C):

- **Shape validity** is owned by the verb *by construction* — the verb writes a shape-valid tree; the redundant per-commit `aiwf check --shape-only` is dropped.
- **Secret/path-leak scanning** (gitleaks, G-0103) relocates to the **pre-push** hook (range scan over the pushed commits).
- **Pre-push `aiwf check`** remains the authoritative full-validation gate (kernel commitment #3).

## Consequences

- The G-0275 half-state class (staged rename vs untracked path) becomes structurally impossible; no verb reverts the worktree.
- Verb-commit validity no longer depends on a pre-commit hook being installed — more robust than today, where a consumer who never ran `aiwf init` gets no per-commit validation at all.
- A single commit-construction substrate serves all verbs and, later, the G-0281 gaps-inbox (M-0187) — no parallel commit path.
- Kernel commitment #7 (exactly one commit per verb) is preserved; only the construction mechanism changes.
- **Rejected alternative — index save/reset/restore:** preserves hook firing but still mutates the shared live index (a crash window; exposure to the shared-worktree index race). The plumbing approach is strictly more isolated and hook-install-independent.
- The never-checked-out-ref and push-inside-a-verb decisions for the gaps-inbox are deliberately deferred to M-0187's own ADR.

## References

E-0045 (epic), M-0186 (first implementation), G-0276 (driver), G-0275 (fail-loud floor), the G-0034 → G-0112 history (why `git commit --only` was abandoned), ADR-0001 (related: mint ids at trunk integration).
