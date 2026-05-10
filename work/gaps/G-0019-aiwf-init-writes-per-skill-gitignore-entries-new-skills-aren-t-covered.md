---
id: G-0019
title: '`aiwf init` writes per-skill `.gitignore` entries; new skills aren''t covered'
status: addressed
addressed_by_commit:
  - 92f5d51
---

Resolved in commit `92f5d51` (fix(aiwf): G19 — emit wildcard skill .gitignore entry, future-proof against new skills). Took the proposed approach: `skills.MaterializedPaths` renamed to `skills.GitignorePatterns`, returning a two-element constant slice (`.claude/skills/aiwf-*/` plus `.claude/skills/.aiwf-owned`). The trailing slash restricts the wildcard to directories. Adding a new aiwf-* skill to the embedded set no longer requires consumers to re-run `aiwf init` to refresh their `.gitignore`. Existing consumers with the per-skill list pick up the two new lines on next `aiwf init`; old entries are harmless (the wildcard subsumes them) and cleanup is the consumer's choice. New `TestInit_GitignoreFutureProof` asserts the property the rename was made for: re-init with the wildcard already present does not duplicate it. Smoke-tested end-to-end against the actual binary.

---
