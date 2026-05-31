---
id: G-0195
title: canonicalTrailerKeys drifts from trailerOrder; no mirror-validity guard
status: open
discovered_in: M-0102
---
## Problem

`canonicalTrailerKeys` at [`internal/cli/integration/trailer_shape_test.go:65`](../../../internal/cli/integration/trailer_shape_test.go) is a hand-maintained mirror of `trailerOrder` from [`internal/gitops/trailers.go:50`](../../../internal/gitops/trailers.go), purposed to fail CI when "a new trailer landed without a corresponding `Trailer*` constant" (per the file's own doc).

Two drifts are surfaced as of M-0102:

1. **`TrailerForceFor` is missing from `canonicalTrailerKeys`** — added to `trailerOrder` in M-0136 (acknowledge-illegal verb) but never added to the mirror. The drift is silent today because `TestTrailerShapePerMutatingVerb`'s fixture set never invokes `acknowledge-illegal`, so no commit it inspects carries `aiwf-force-for:`. The next time someone adds an `acknowledge-illegal` case to the fixture, the drift fires.

2. **No mechanical guard** ties `canonicalTrailerKeys` to `trailerOrder`. The map is hand-maintained; nothing fails CI if a new `Trailer*` constant lands without a mirror entry. The whole point of the drift test is to catch missing membership — but the membership set itself drifts.

## Resolution shape

Replace the hand-maintained map with a derived one — build `canonicalTrailerKeys` from `trailerOrder` at test init (a single `for` over `gitops.TrailerOrder()` if it exports an accessor, or via a test-side reflection over the constants). Eliminates the parallel source of truth; the drift class becomes structurally impossible.

If `gitops.trailerOrder` should remain unexported, the equivalent move is a `gitops.CanonicalTrailerKeys() map[string]bool` accessor — same effect.

## Backfill

Add `TrailerForceFor` to the map as part of the structural fix (or as a one-line tide-over before the fix lands).

## Discovered in

M-0102 / AC-2 verification pass: while adding `TrailerBranch` to both `trailerOrder` and `canonicalTrailerKeys`, the user pushed back on whether the change was 100% verified. The pass found the existing `TrailerForceFor` drift.
