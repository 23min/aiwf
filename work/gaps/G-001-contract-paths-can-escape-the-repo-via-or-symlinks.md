---
id: G-001
title: Contract paths can escape the repo (via `..` or symlinks)
status: addressed
---

Resolved in commit `4ec5d84` (fix(aiwf): G1 — reject contract paths that escape the repo root). New packages `internal/pathutil` and `internal/contractconfig` are the single point of truth for path containment; both `contractcheck` and `contractverify` route through them. `..` traversal, absolute paths outside the repo, out-of-repo symlinks, and symlink loops all produce a `contract-config` / `path-escape` finding, and `contractverify` refuses to invoke a validator on any escaped entry. 100% line coverage on the new code, including a load-bearing test that asserts the validator marker file is never written for an escaped entry.

---
