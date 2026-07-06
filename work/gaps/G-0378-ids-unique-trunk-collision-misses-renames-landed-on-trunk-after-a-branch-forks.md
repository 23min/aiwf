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

Per `ADR-0031`: detect trunk-side renames too, but not as a direct mirror of
the branch-side mechanism. A naive symmetric mirror (trailer walk + `git
diff -M` content-similarity fallback, scoped `merge-base(HEAD, ref)..ref`)
was adversarially reviewed before any code was written and found unsafe —
extending the `-M` fallback to the trunk side risks spuriously pairing two
unrelated, boilerplate-similar entities over a potentially large range
(trunk's full history since an old branch forked), which can silently
misclassify a genuine id collision as "same entity, moved" and merge it to
trunk unnoticed. That failure mode is strictly worse than the false
positive being fixed.

The corrected design (`ADR-0031`):

- **Trunk side: trailer walk only, no `-M` fallback, ever.** A manual,
  non-kernel-verb `git mv` performed directly on trunk stays an
  unrecognized (safe) false positive — consistent with this repo's existing
  `id-rename-untrailered` check, which already discourages non-kernel-verb
  renames.
- **Branch side: unchanged** — today's trailer walk + `-M` fallback stays,
  since its range (bounded by the branch's own divergence) doesn't carry
  the same risk.
- **`entity.ActiveFormOf`'s archive-sweep pre-filter stays separate**, run
  before any merge-base/rename analysis — not subsumed into the general
  mechanism.
- **Cost model: in-memory guard first, batched not per-id.** Compute the
  disputed-id set (same id, different path, working tree vs. `Tree.TrunkIDs`)
  in memory, at zero git cost. Empty → skip all git work. Non-empty → one
  batched computation covering every disputed id at once, never a per-id
  loop (a naive per-id design was shown to be 50-100x slower than today's
  single batched diff on a large rename batch — e.g. an unmerged `aiwf
  rewidth --apply` or `archive --apply` sweep).
- **Default to firing when `merge-base` is unreachable** (shallow clone,
  unrelated histories) — an explicit degraded tier, not silently assumed
  away.
- Does not claim to eliminate every false positive "by construction": a
  stale branch spanning a trunk-side `aiwf reallocate` remains a known,
  out-of-scope residual limitation.

## Scope

- `internal/gitops/refs.go`: a trunk-side rename view scoped to
  `merge-base(HEAD, ref)..ref`, trailer-driven only (reuses the existing
  `renamesFromAiwfVerbTrailers` shape; does not add a `-M` diff pass for
  this direction).
- `internal/check/check.go`'s `idsUnique`: the in-memory disputed-id guard,
  the retained `ActiveFormOf` pre-filter, then the new trailer-only
  trunk-side escape hatch alongside the existing branch-side one.
- If the implementation needs an id-at-a-commit lookup, reuse the existing
  (currently-unexported) `trunk.idsFromPaths` helper rather than
  re-deriving it, per `ADR-0031`'s note for `E-0060` consistency.
- Tests, using real git fixtures (not the in-memory fake-tree shape the
  current branch-side tests use): a trunk-side retitle after a fork clears
  `trunk-collision`; a genuine cross-branch collision (two independent
  entities, no rename relationship, high content similarity) still fires;
  a shallow-clone / no-merge-base case still fires (degraded tier); the
  existing archive-sweep and branch-side-rename unit tests are unaffected.

## Related

- `ADR-0031` — records the trailer-only-on-trunk-side decision and the
  supporting cost-model / pre-filter-retention / degraded-tier decisions
  this Direction implements.
- `E-0060`, `ADR-0025`, `ADR-0030` — a related but distinct concern
  (resolving *references* to ids absent from the local tree via the
  cross-branch view); confirmed via independent review to ask a genuinely
  different question at a different commit-ish, so this gap is deliberately
  not folded into that epic.
- A separate gap tracks the misleading `aiwf reallocate` remediation hint
  for this finding, since that's wrong regardless of this design's
  implementation timeline.
