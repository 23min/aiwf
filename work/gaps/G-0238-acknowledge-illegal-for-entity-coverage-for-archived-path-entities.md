---
id: G-0238
title: acknowledge-illegal --for-entity coverage for archived-path entities
status: open
---
## What's missing

`aiwf acknowledge-illegal --for-entity` has no test coverage for archived-path entities. The `verifySHATouchesEntity` helper at `internal/verb/acknowledgeillegal.go:138-153` walks `git diff-tree` output through `entity.PathKind` + `entity.IDFromPath`. `PathKind` strips the archive segment via `stripArchiveSegment` upfront (`internal/entity/entity.go:612-613`), so an archive-path entry like `work/gaps/archive/G-0001-foo.md` correctly resolves to `G-0001`. But no test pins this end-to-end through `verifySHATouchesEntity`.

## Why it matters

The G-0231 ack-mechanism's whole point is letting operators ack historical SHAs whose paths may legitimately be in the archive (the 5 acks landed in G-0231 itself were against entities that were active at the original commit time but might be archived now — the test coverage didn't pin this case because the test fixture uses fresh-commit entities, not retroactive archive moves).

Silent failure mode: a future archive convention tweak (different prefix, nested kind dirs, etc.) could break `stripArchiveSegment` and the verb would start refusing legitimate archive-path acks, with no test catching the regression.

## How to fix

One new test in `internal/verb/acknowledgeillegal_test.go`:

1. Commit a gap at `work/gaps/G-0001-foo.md` (active path).
2. Move it to `work/gaps/archive/G-0001-foo.md` (archive sweep — a separate commit that also adds the archive trailer).
3. Ack the original (active-path) SHA with `--for-entity G-0001` — should succeed.
4. Also try acking the archive-move SHA with `--for-entity G-0001` — should succeed since the archive commit touched `work/gaps/archive/G-0001-foo.md` which `PathKind` resolves to a gap kind and `IDFromPath` resolves to `G-0001`.

If either step fails, fix `verifySHATouchesEntity` to handle archive paths (the underlying `entity.IDFromPath` already does, so this is currently a no-op insurance test).

## Source

G-0231 reviewer pass, N5 finding ("archive-path support in `acknowledge-illegal --for-entity` verification").
