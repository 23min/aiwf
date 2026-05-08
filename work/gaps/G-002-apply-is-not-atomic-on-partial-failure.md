---
id: G-002
title: '`Apply` is not atomic on partial failure'
status: addressed
addressed_by_commit:
  - f77740c
---

Resolved in commit `f77740c` (fix(aiwf): G2 — atomic rollback on Apply failure). Apply wraps its mutations in a deferred rollback that restores the worktree and index to HEAD when any step fails (write error, commit failure, panic). Brand-new files are removed entirely so the next invocation sees a clean tree. New `gitops.Restore` helper. Tests cover write-after-mv failure, git mv failure, brand-new file cleanup, commit failure (no identity), panic recovery, and dedupe of touched paths. apply.go coverage at 94.3% — two defensive branches (compound rollback-also-failed wrap and post-write `git add` failure) marked `//coverage:ignore` per `CLAUDE.md`'s allowance, with the load-bearing rollback path itself at 100%.

---
