---
id: G-0101
title: ids-unique trunk-collision rule is not archive-aware; first aiwf archive --apply triggers 176 false-positive errors that block the pre-push hook
status: open
discovered_in: M-0085
---

## What's missing

The `ids-unique` rule's trunk-collision arm (`internal/check/check.go::idsUnique`) compared branch and trunk paths with strict equality. An archive sweep — `aiwf archive --apply` per ADR-0004 — renames every terminal-status entity from `<kind>/<id>.md` on trunk to `<kind>/archive/<id>.md` on the branch carrying the sweep. The rule treated each pair as a collision and emitted an error per entity. The first historical migration on this kernel tree produced **176 false-positive `trunk-collision` errors**, enough to block the pre-push hook on the otherwise-legal sweep commit.

The fix added an exported helper `entity.ActiveFormOf(path)` that strips a recognized per-kind `archive/` segment (idempotent on already-active paths); the rule now normalizes both branch and trunk paths through it before comparing. Equal active forms = sweep rename, not a collision. Non-archive path divergence still fires (G37 invariant preserved).

## Why it matters

The first `aiwf archive --apply` on a real consumer tree is the migration ADR-0004 names as load-bearing: *"Operators ratifying this ADR run `aiwf archive --dry-run` first to preview the move, then `aiwf archive --apply` to commit. The same verb covers the bulk historical migration and the recurring small sweeps that follow."* Without an archive-aware trunk-collision rule, that migration is impossible to push. The defect was latent until E-0024 dogfooded the verb against this kernel tree; the moment the sweep ran, the rule made the push impossible.

The fix is mechanical (~3 lines in the rule plus the exported helper) and preserves the G37 invariant that genuine cross-branch id collisions still surface. Pinned by `TestIDsUnique_ArchiveSweepNotCollision` (one case per ADR-0004 storage-table row plus the symmetric reverse) and `TestIDsUnique_NonArchivePathDivergenceStillFires` (negative case).

Closed by commit `706310c` on 2026-05-11.
