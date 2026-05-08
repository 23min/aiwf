---
id: G-005
title: Reallocate's prose references are warnings, not errors
status: addressed
addressed_by_commit:
  - 0e247fe
---

Resolved in commit `0e247fe` (fix(aiwf): G5 — reallocate rewrites prose references mechanically). Prose mentions of the old id in any entity body — including the target's own body — are now rewritten in the same commit as the frontmatter rewrite. Word-boundary regex prevents false matches against longer ids (M-001 → M-003 leaves M-0010 untouched). The `reallocate-body-reference` warning code is removed; no half-step "fix it yourself" findings remain. Tests cover the load-bearing rewrite-across-entities scenario, the M-0010-must-not-match edge case, multiple-entities-rewritten-in-one-commit, and the target's own self-reference.

---
