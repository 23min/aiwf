---
id: G-0109
title: trunk-collision check treats renames as duplicate id allocations
status: open
prior_ids:
    - G-0107
---
# Problem

`aiwf check`'s `ids-unique/trunk-collision` finding fires whenever a feature branch renames an entity file: the entity exists at the new slug on the branch and at the old slug on `refs/remotes/origin/main` (the trunk view), and the check considers these "different entities with the same id" because it uses *path* as the equivalence key rather than *id*.

The check's enumeration sees both paths and registers a collision, even though a single `git log -M` would reveal they're the same content moved.

# Reproduction

On a feature branch, run any sequence of `aiwf rename <id> <new-slug>` operations. Each rename's pre-commit hook (which runs `internal/policies/...`) passes — but **`aiwf check` on the branch reports a trunk-collision finding for every renamed entity**.

Concretely, this fired today during the G-0102 slug-cap cleanup pass: 21 renames produced 24 trunk-collision errors. The errors blocked `git push` (pre-push hook runs `aiwf check`), even though the renames were mechanically correct.

# Why this matters

The catch-22:

1. Pre-push runs `aiwf check`.
2. Check sees branch (new slugs) + trunk = `origin/main` (old slugs) = collision.
3. Push blocked.
4. After push, `origin/main` would match branch and collisions vanish — but push is the thing that's blocked.

The only escape is `--no-verify` on push, which CLAUDE.md forbids without explicit human authorization. Any meaningful slug-rename batch on a feature branch hits this wall.

# Root cause

`aiwf check`'s trunk-aware id-uniqueness check (in `internal/check/` somewhere — likely the `ids-unique` rule) enumerates entity files at their on-disk paths and unions branch + trunk views. Two paths with the same id register as two entities.

The check needs to either:

1. **Use git's rename detection** (`git log -M` between trunk and branch tip) to recognize "old path → new path = same entity," and skip the collision finding for renamed entities.
2. **Use entity id as the equivalence key** rather than path, so a same-id-different-path pair collapses to one entity regardless of slug.

Option 1 is more conservative (catches "two different files claiming the same id" while ignoring renames). Option 2 is more permissive (doesn't catch any path-shaped collision, only id-shaped).

# Related

- **G-0081** — "aiwf rename does not pre-flight trunk-collision check." G-0081 is about pre-flighting the collision in the rename verb. **This gap is about the detector itself being wrong.** Even with a perfect pre-flight, the check would still fire post-rename on the branch — the detection logic doesn't recognize renames. Closing both gaps requires fixing the detector first; pre-flight is layered on top.
- **G-0084** — verb hygiene contract; umbrellas G-0081. The detector fix might fit under the same umbrella.
- **ADR-0005** (proposed) — verb hygiene contract; both gaps are gated on this.

# Scope of the fix

Touches `internal/check/`'s ids-unique rule. The git rename-detection approach is probably ~30 lines + a fixture test pair. Should be wf-patch-sized once the design is settled (i.e., whether to use rename detection or id-as-key).

# Why not urgent (with caveat)

The catch-22 blocks any rename-heavy cleanup. Today's G-0102 cleanup deliberately stopped at retitles only — slug renames are deferred until this gap closes. Each future slug-rename batch on a feature branch will hit the same wall.

So: not urgent in the sense that no work is broken *right now*, but it blocks a class of future cleanup. Worth promoting once it's the next-best work, not before.

# Suggested resolution

Small milestone (or wf-patch if the design is simple): a single change in `internal/check/` adding git rename detection to the trunk-collision rule. The implementation lives where the rule is computed; the test suite verifies that branch-side renames don't trigger the finding.

When this lands, the G-0102 slug-rename cleanup pass becomes mechanical and can ship as a routine wf-patch.
