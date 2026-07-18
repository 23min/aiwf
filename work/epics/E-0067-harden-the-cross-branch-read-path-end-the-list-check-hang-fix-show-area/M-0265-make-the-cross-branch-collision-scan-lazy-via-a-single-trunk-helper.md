---
id: M-0265
title: Make the cross-branch collision scan lazy via a single trunk helper
status: in_progress
parent: E-0067
tdd: required
acs:
    - id: AC-1
      title: One trunk helper composes the cross-branch scan for treeload, list, and show
      status: met
      tdd_phase: done
    - id: AC-2
      title: DetectCollisions runs only for ids absent from the local working tree
      status: met
      tdd_phase: done
    - id: AC-3
      title: Cross-branch list rows and check findings are unchanged before and after
      status: met
      tdd_phase: done
    - id: AC-4
      title: Zero DetectCollisions blob-stats when every id is present locally
      status: met
      tdd_phase: done
---

## Goal

Replace the eager, triplicated cross-branch scan with a single `internal/trunk`
helper that runs `DetectCollisions` only for ids absent from the local working
tree. Filtered `aiwf list` and the cross-branch portion of `aiwf check` drop from
O(entities × refs) blob-stats to O(locally-absent ids) — sub-second in the common
all-merged state — with cross-branch output unchanged.

## Context

E-0060 shipped the cross-branch read path, but its scan composition
(`trunk.LocalRefHits` + `RemoteRefHits`, then `DetectCollisions` over the full
union) is copied eagerly at three call sites: `cliutil.LoadTreeWithTrunk`,
`list.crossBranchListRows`, and `show.buildCrossBranchShowView`. At this
repository's scale (860 entities, 10 refs, ~8300 hits) that is ~23s of blob-stats
producing zero rows, because a collision result is consulted only on a local-tree
miss. This milestone makes the scan lazy and consolidates it; G-0418 is the
tracking gap.

## Acceptance criteria

### AC-1 — One trunk helper composes the cross-branch scan for treeload, list, and show

A single `internal/trunk` helper is the only place the local + remote ref-hit
union is composed and handed to `DetectCollisions`. `cliutil.LoadTreeWithTrunk`,
`list.crossBranchListRows`, and `show.buildCrossBranchShowView` all route through
it, so the "hits passed to `DetectCollisions` equal the union that was scanned"
coupling lives in one place, not three. Verified structurally: the three call
sites invoke the helper, and the union/collision composition appears once.

### AC-2 — DetectCollisions runs only for ids absent from the local working tree

Given ref-hits for ids both present and absent in the local tree, the helper
passes only the absent-id hits to `DetectCollisions`; locally-present ids are
never blob-stat'd. Verified mechanically against a fixture carrying both,
asserting the exact hit set `DetectCollisions` receives.

### AC-3 — Cross-branch list rows and check findings are unchanged before and after

For every pre-existing scenario — including an id absent locally that collides
across refs — the cross-branch rows emitted by `aiwf list` and the findings
emitted by the `refs-resolve` and `body_prose_id` cross-branch branches are
identical before and after this change. The safety basis is that every consumer
reads a collision result only after a local-tree miss; the test fails if a
locally-present id's collision result ever changes an output.

### AC-4 — Zero DetectCollisions blob-stats when every id is present locally

On a fixture with many entities and refs where every id is present in the local
tree, the helper performs zero `DetectCollisions` blob-stat round-trips. The
assertion counts stats (deterministic), not wall-clock — pinning the scale
property that cost tracks the locally-absent set, not entities × refs.

## Constraints

- Behavior preservation is load-bearing: the set of cross-branch rows and findings
  must be byte-identical before and after (AC-3), safe only because every consumer
  is local-tree-miss-guarded.
- `internal/trunk` stays read-only and best-effort (never errors, degrades to nil
  on odd repo state); no new package-level mutable state.
- The allocator's cross-branch view (`LocalRefIDs` / `RemoteRefIDs`, feeding
  allocation) must not regress.

## Design notes

- The lazy filter's home — inside `DetectCollisions` via a `presentLocally`
  predicate, or in the new helper filtering hits before the call — is open; lean
  toward the helper, keeping `DetectCollisions` a pure function over the hits it is
  given (E-0067 open question). Decide at the readiness/design step.
- The ls-tree ref scan that builds the union is unchanged; only the collision half
  becomes lazy. Reducing the union scan itself is out of scope (G-0324).
- D-0036 (collision severity is non-blocking) and ADR-0030 (cross-branch read-side
  extension point) are unchanged.

## Surfaces touched

- `internal/trunk/trunk.go`
- `internal/cli/cliutil/treeload.go`
- `internal/cli/list/list.go`
- `internal/cli/show/show.go`

## Out of scope

- G-0416 (distinguishing an unmerged edit from a genuine duplicate-mint collision).
- The ls-tree ref-union scan cost (G-0324) and check's history-revwalk cost center
  (G-0372).
- `show --area` cross-branch correctness — the epic's second milestone.

## Dependencies

- None. This is the epic's foundational milestone; the second milestone depends on
  the helper introduced here.

## References

- Gap: G-0418. Epic: E-0067. Decisions: D-0036, ADR-0030.

## Work log

The per-AC phase timeline (red → green → done → met) is recorded in
`aiwf history M-0265/AC-<N>`; the outcomes and implementing commits:

### AC-1 — one trunk helper composes the cross-branch scan for treeload, list, and show
Introduced `trunk.ScanCrossBranch` as the single composition point; rewired
`LoadTreeWithTrunk` / `crossBranchListRows` / `buildCrossBranchShowView` to it.
Behavior-identical consolidation. · commit `a98708a8` · `internal/trunk` +
integration green.

### AC-2 — detectCollisions runs only for ids absent from the local working tree
Added the `presentLocally` predicate and the `absentHits` filter; only ids
absent from the local tree reach `detectCollisions`. · commit `cd6d333b` ·
`internal/trunk` unit (exact-hit-set + git-backed wiring) + integration green.

### AC-4 — zero detectCollisions blob-stats when every id is present locally
Extracted the `needsBlobStats` stat-gate; a many-entity / multi-ref all-present
fixture pins that the filtered hit set requires zero blob-stats. · commit
`bc2eedbc` · green.

### AC-3 — cross-branch list rows and check findings unchanged before and after
Behavior-preservation pins across list rows, refs-resolve, and body-prose-id for
a locally-present id carrying a recorded collision. · commit `b4ce6f32` · green.

## Decisions made during implementation

- **Lazy filter lives in the wrapper, not inside `detectCollisions`** —
  resolving E-0067's open question toward the wrapper: `absentHits` filters the
  hits before `detectCollisions` is called, keeping the latter a pure function
  over the hits it is given (preserving its direct testability). This confirms
  the epic's stated lean, so no ADR/D is owed here; the durable principle
  (collision detection scoped to the locally-absent id set) is a candidate for
  the epic's ADR harvest at wrap.
- **`detectCollisions` unexported; the interim AST consolidation policy dropped**
  (post-review · commit `4541f388`). With zero external callers, Go's export
  rules enforce "only `ScanCrossBranch` composes the scan" natively — a compile
  error, stronger and earlier than a CI-tier policy — removing ~185 lines. A
  guiding comment on `detectCollisions` preserves the G-0418 intent.

## Validation

- `go test ./...` green; race-clean on the changed packages (`trunk`, `check`,
  `list`, `show`, `cliutil`, `integration`, `policies`).
- `make lint`: 0 issues. `go build ./...`: green. Diff-coverage gate: green
  (100% on `ScanCrossBranch` / `absentHits` / `needsBlobStats` / `detectCollisions`).
- `aiwf check`: 0 errors (one environmental warning — the milestone worktree
  branch has no upstream, so the provenance audit is skipped; it resolves on
  integration).
- **Performance — same-environment A/B (old eager vs new lazy binary, this
  repo's 860-entity / 10-ref tree):** filtered `aiwf list --kind gap --priority
  urgent` 10.4s → 1.0s; `aiwf check` 15.5s → 5.7s. Check findings and list rows
  are byte-identical between the two binaries.

## Deferrals

- (none) — check's remaining cost is its full-history revwalk, which is
  epic-scoped and tracked by G-0372 (out of E-0067's scope), not a milestone
  deferral.

## Reviewer notes

- **Independent two-lens review before wrap — both cleared.** Code-quality
  (fresh context): APPROVE, no blocking findings; it adversarially confirmed a
  missed real collision is impossible — the filter's present set (`tr.ByID`,
  Entities-only) is a subset of every consumer's local index, so a skipped id is
  never one a consumer would read a collision for — verified `needsBlobStats` is
  behavior-identical to the prior inline gate, and measured 100% coverage on the
  new functions. Design-quality (`wf-rethink`): SOUND WITH SUGGESTIONS; it
  independently reconstructed the same design and confirmed the subset invariant.
- **AC-1 evidence shift.** AC-1's structural evidence was the AST policy
  `PolicyCrossBranchScanConsolidation`; per the design review it was dropped in
  favor of unexporting `detectCollisions`, so the consolidation invariant is now
  compile-time-enforced (Go export rules) plus the integration tests that
  exercise all three consumers' cross-branch paths. AC-1 stays `met` — the
  property is guaranteed more strongly.
- **Named tradeoff (accepted).** `show`'s cross-branch path reads
  `scan.Collisions[canon]` (computed over the whole absent set) rather than
  scanning canon-only; the result is equivalent (that path is reached only on a
  local-tree miss) and bounded by the absent set, in exchange for the single
  composition point.
