---
id: G-052
title: Plain-git body edits trigger warnings despite skill permitting them
status: addressed
discovered_in: E-14
addressed_by:
    - M-058
---

## Problem

The `aiwf-add` skill explicitly permits body-prose edits to entity files via plain `git commit`: *"Body-prose edits to an existing entity file — the markdown under the frontmatter. The frontmatter itself is structured state; leave it to verbs."* But the `provenance-untrailered-entity-commit` check rule then flags exactly that workflow, recommending the user backfill with `aiwf … --audit-only`. Following the skill produces warnings the user is then asked to clean up.

## Evidence

Planning E-14 produced one body-content commit (`c079801` on `poc/aiwf-v3`) that touched 8 entity files via plain `git commit` per the skill's permission. `aiwf check` immediately reported 8 `provenance-untrailered-entity-commit` warnings. The user (the intended audience for both the skill and the check) experienced this as the framework punishing them for following its own instructions.

## Root cause

Two parts of the system disagree on what "valid body edit" means:

- **Skill encodes a permissive view**: frontmatter through verbs, body via plain git is fine.
- **Check encodes a strict view**: any edit to an entity file should leave a trailered audit trail.

One of them is wrong. The kernel principle *"framework correctness must not depend on LLM behavior"* says: a guarantee that depends on someone remembering to backfill is not a guarantee. Either the policy is too strict, or the skill is too permissive. The strict reading aligns with kernel principles; the permissive reading reflects implementation gaps in the verbs (no body-edit-aware verb exists today).

## Direction

Pick the strict reading:

1. **Add `--body-file` on `aiwf add` variants** so creation-time body content rides along with the create commit (also resolves G-051's body-edit hop).
2. **Add `aiwf edit-body <id> --body-file <path>`** (or equivalent) for post-creation body edits — produces a properly trailered commit.
3. **Update the `aiwf-add` skill** to remove the plain-git carve-out; route all body changes through verbs.

After this, `provenance-untrailered-entity-commit` should never fire under normal use (it stays as a backstop against accidental hand-edits).

## Relationship to G-051

These should be solved together. The `--body-file` flag is a single change that resolves both:
- Saves the extra body-content commit (G-051).
- Eliminates the untrailered commit that triggers the warnings (G-052).

## Considered alternative: retroactive trailer-attach

Briefly considered: a `aiwf trailer-attach <sha> --entity <id> --verb edit-body` verb that adds a trailer to an *existing* untrailered commit, instead of creating an empty-diff audit commit. **Rejected** because adding a trailer to an existing commit means amending or rebasing — destructive history operations explicitly counter to the kernel's *"prefer to create a new commit rather than amending"* rule. A softer interpretation (create an empty-diff trailered commit pointing back at the original) is mechanically equivalent to `aiwf promote --audit-only` already, so adds nothing beyond ergonomics.

The chosen direction (`aiwf edit-body` + `--body-file` on `aiwf add`) eliminates the friction at the source rather than offering a retroactive escape hatch.
