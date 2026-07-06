---
id: G-0378
title: ids-unique trunk-collision misses renames landed on trunk after a branch forks
status: open
---
## What's missing

`internal/check/check.go`'s `idsUnique` reports a `trunk-collision` (error
severity) whenever a working-tree entity and a trunk-side entity share an id
at different paths, unless one of two escape hatches matches: an
archive-sweep rename (`entity.ActiveFormOf`) or a rename this branch itself
committed since forking (`Tree.TrunkRenames`, populated by
`gitops.RenamesFromRef`).

`RenamesFromRef` is deliberately scoped to `merge-base(HEAD, ref)..HEAD` —
it only detects renames the *current branch* committed relative to trunk,
never the reverse. So when an entity is renamed *on trunk* (e.g. via `aiwf
retitle` or `aiwf rename`, both of which rename the underlying file) after
a feature branch has already forked, and that branch hasn't merged/rebased
trunk since, `idsUnique` sees: same id, two different paths, neither escape
hatch matches — false-positive `trunk-collision`.

## Why it matters

The only documented remediation for `trunk-collision` is `aiwf reallocate
<id>` (per CLAUDE.md's id-collision-resolution guidance). Run against this
false positive, `reallocate` does not fix anything — it renames the
branch's (stale, otherwise-correct) copy to a new id, producing a genuine
duplicate entity that then has to be reverted by hand once the real cause
(a stale branch, not a real collision) is understood.

Confirmed live: `aiwf retitle G-0368` on `main` (which renames `G-0368`'s
file to match the new title-derived slug) produced exactly this false
positive on an unrelated, already-forked feature branch that still held the
pre-retitle path; the branch's own `aiwf check` run led to `aiwf reallocate
G-0368 -> G-0376`, a spurious duplicate later reverted by hand.

## Direction

Detect trunk-side renames too, not just branch-side ones: extend the rename
view (or add a second one) covering `merge-base(HEAD, ref)..ref` — the
renames trunk committed since the fork point — and treat a hit there as the
same "same entity moved" case the existing `TrunkRenames` escape hatch
already handles for the branch-side direction. The G37 false-positive risk
`RenamesFromRef`'s current one-directional scoping was built to avoid
(independent parallel-clone allocations misread as a rename) applies
symmetrically to this direction too, so the same merge-base-scoped,
same-branch-history-only comparison approach should carry over without
reintroducing that risk — needs confirming when this is worked.

## Scope

- `internal/gitops/refs.go`: a trunk-side counterpart to `RenamesFromRef`
  (or a widened version covering both directions), scoped by the same
  merge-base logic.
- `internal/check/check.go`'s `idsUnique`: consult the new trunk-side rename
  view as a third escape hatch alongside the existing two.
- Tests: a fixture reproducing the exact scenario (rename an entity on
  trunk after a branch forks, without merging trunk back into the branch;
  `aiwf check` on the branch must not fire `trunk-collision`), plus a
  regression test confirming a genuine cross-branch collision (two
  different entities independently allocated the same id) still fires.

## Related

- `E-0060`, `ADR-0025`, `ADR-0030` — a related but distinct concern
  (resolving *references* to ids absent from the local tree via the
  cross-branch view); both epics explicitly keep `ids-unique`'s
  trunk-anchored collision basis out of scope, so this gap is deliberately
  not folded into that epic.
