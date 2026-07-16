---
id: G-0415
title: Cross-branch reference resolution must detect same-id divergence across refs
status: addressed
discovered_in: E-0060
addressed_by:
    - M-0259
---
## Problem

E-0060 (proposed) and ADR-0030 (proposed) plan to reuse the allocator's
cross-branch view (`Tree.LocalRefIDs`/`RemoteRefIDs`, built by E-0052) as a
second input to `refs-resolve`/`body-prose-id` and to `aiwf show`/`aiwf list`.
That view was designed for allocation, where its two load-bearing properties
are the right tradeoffs but don't carry over cleanly to reference resolution:

1. **Multiplicity is discarded.** `refIDs()` (`internal/trunk/trunk.go`)
   collapses every ref it scans into a flat `[]string` of bare ids. `idsUnique`
   (`internal/check/check.go`) only ever compares the working tree against
   `TrunkIDs` — it never compares sibling local/remote branches against each
   other. So the same id can legitimately exist with *different* content on
   two unmerged branches today, invisibly, until one merges. If E-0060's
   check-side/read-side surfaces treat "found in the cross-branch view" as a
   plain boolean hit, a genuine collision (exactly the G-0272/G-0281 class
   E-0060 cites as motivation) would be silently classified as the benign,
   non-blocking `cross-branch-pending` tier, and `aiwf show`/`aiwf list` would
   have to arbitrarily pick one of the divergent copies to render.

2. **Silent best-effort degrade changes failure mode when reused for a
   blocking check.** `refIDs()` deliberately never errors: a ref that "lists
   but won't read (corrupt or raced away mid-scan)" is silently skipped,
   degrading to whatever it could collect. For the allocator that's correct
   (fail-open; worst case is a slightly wasted id). Reused as an input to
   `refs-resolve`/`body-prose-id`, the same silent degrade means a transient
   git subprocess hiccup during one `aiwf check` run can make a legitimately
   cross-branch-pending id look like it vanished, escalating it to a hard
   `unresolved` finding — which blocks the pre-push gate on a transient
   condition, not a real dangling reference.

## Decision

Resolved by design discussion while analyzing E-0060's proposed solution
(2026-07-15):

- **Multiplicity (item 1): detect via blob-SHA comparison.** When the
  cross-branch view finds an id on more than one ref, compare the blob SHA at
  each ref's path — free given `git cat-file --batch`'s response header
  (`<sha1> <type> <size>`), which the read-side milestone already needs via
  `gitops.BlobReader` (`internal/gitops/catfile.go`). Identical SHA across
  every ref holding the id → ordinary `cross-branch-pending` (one entity, not
  yet merged, nothing ambiguous). Divergent SHA → escalate to a distinct
  finding (a new subcode, e.g. `cross-branch-collision`, not a silent
  downgrade to the soft tier), and the read-side must refuse to silently pick
  one ref's content over another's.
- **Transient scan failures (item 2): accept as a documented, self-healing
  limitation for v1.** No retry, no new error-signaling plumbing on the
  shared `refIDs()`/`LocalRefIDs`/`RemoteRefIDs` primitives — check-side
  reuse inherits the allocator's existing best-effort semantics as-is.
  Nothing about the resulting misclassification is sticky: `aiwf check`
  caches nothing, so a spurious `unresolved` from a mid-scan race clears on
  the next successful run. Revisit only if this is observed to actually
  cause repeated false blocks in practice, not preemptively.

## Direction (to converge at milestone planning)

- The `LocalRefIDs`/`RemoteRefIDs` widening to carry (kind, id, path, ref) —
  already an open question in E-0060's epic spec, needed regardless for the
  read-side milestone — is also the natural place to add the blob-SHA
  comparison: group hits by id, and for any id with more than one ref, pull
  each ref's blob SHA in the same `BlobReader` pass already required for
  content resolution.
- The new collision subcode belongs on the check-side milestone that adds the
  `cross-branch-pending` tier, as a second acceptance criterion alongside the
  escalation fixture test ADR-0030 already requires — both are the same
  class of invariant (a soft classification must never silently launder a
  case that deserves a harder one).
- No action item for the transient-failure risk beyond noting it in the
  check-side milestone's spec (context or risks section), so it's a
  documented, deliberate choice rather than an unstated gap discovered later.

## Provenance

Surfaced analyzing E-0060's proposed solution for feasibility and
correctness (2026-07-15), against the current `internal/trunk`,
`internal/check`, and `internal/gitops` implementations. Both items were
resolved as explicit decisions in that conversation before filing.
