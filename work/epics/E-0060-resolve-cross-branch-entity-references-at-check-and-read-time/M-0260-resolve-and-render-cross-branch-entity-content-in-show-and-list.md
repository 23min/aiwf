---
id: M-0260
title: Resolve and render cross-branch entity content in show and list
status: in_progress
parent: E-0060
depends_on:
    - M-0259
tdd: required
acs:
    - id: AC-1
      title: Resolve content via BlobReader using the recorded cross-branch ref
      status: met
      tdd_phase: done
    - id: AC-2
      title: Cross-branch-sourced content is visibly labeled
      status: met
      tdd_phase: done
    - id: AC-3
      title: Refuses to pick a ref when content diverges
      status: met
      tdd_phase: done
    - id: AC-4
      title: No working-tree, index, or ref writes
      status: met
      tdd_phase: done
---

## Goal

Let `aiwf show`/`aiwf list` resolve and render an entity's content live
from another local or remote-tracking ref when the id is cross-branch-known
but locally absent — visibly labeled, and never guessed at when the
cross-branch view reports divergent content.

## Context

Depends on `M-0259`, which widens the cross-branch view to carry per-id
path/ref (AC-1) and classifies a hit as `cross-branch-pending` or
`cross-branch-collision` (AC-2/AC-3). This milestone is the second of
ADR-0030's two extension points — the read-side consumer of the same view.

## Acceptance criteria

### AC-1 — Resolve content via BlobReader using the recorded cross-branch ref

`aiwf show` and `aiwf list` resolve an entity's content by reading
`<ref>:<path>` via `gitops.BlobReader`, using the ref M-0259/AC-1 recorded
for a `cross-branch-pending` id that misses the local working tree.
Strictly read-only — no working-tree, index, or ref writes at any point,
per the epic's "resolution is always a live read against the other ref at
the point of use" constraint.

Evidence: fixture test — an id present only on a sibling local branch;
`aiwf show <id>` renders that branch's content without touching the
working tree, index, or refs.

### AC-2 — Cross-branch-sourced content is visibly labeled

Rendered output (both `aiwf show` and `aiwf list`) marks content sourced
from another ref distinctly — never presented indistinguishably from a
locally-resolved entity, per ADR-0030's Consequences section. Exact label
text/placement is an implementation detail decided during this milestone;
the requirement is visibility, not a specific string.

Evidence: fixture test asserting the rendered output for a cross-branch-
sourced entity differs observably (a label, a field, a distinct rendering
mode) from the same entity rendered locally.

### AC-3 — Refuses to pick a ref when content diverges

When the id is classified `cross-branch-collision` (M-0259/AC-3) rather
than `cross-branch-pending`, `aiwf show`/`aiwf list` do not arbitrarily
render one ref's content as if canonical. They surface the ambiguity
explicitly instead — naming the candidate refs and declining to render
body content — leaving resolution to whichever branch merges or
reconciles first. Resolves G-0415's read-side half of the multiplicity
gap: silently picking a ref would present ambiguous, possibly-wrong
content as if it were authoritative.

Evidence: fixture test — two local branches hold divergent content at the
same id; `aiwf show <id>` reports the ambiguity (naming both refs) rather
than picking one side's content.

### AC-4 — No working-tree, index, or ref writes

Every code path this milestone adds is read-only under every
classification (local resolution, cross-branch-pending, or
cross-branch-collision): no `git checkout`, no merge, no working-tree
write, no index write, no ref write.

Evidence: an integration test asserting the repository's working tree,
index, and refs are byte-identical before and after an `aiwf show`/`aiwf
list` invocation that resolves cross-branch content.

## Constraints

- No entity content is copied, cached, or materialized into the working
  tree, the index, or a new ref — resolution is always a live read against
  the other ref at the point of use (epic-level constraint).
- A cross-branch-sourced result is never presented indistinguishably from a
  locally-resolved entity (ADR-0030 Consequences).
- The read-side lookup fires only on a local-tree miss — never adds
  subprocess cost to the common case where the entity already resolves
  locally (epic-level risk mitigation).

## Design notes

- Content resolution reuses `gitops.BlobReader` directly — the same
  primitive `M-0259`/AC-3 uses for blob-SHA comparison, so no second
  git-reading mechanism is introduced across the two milestones.
- `cross-branch-collision` handling (AC-3) surfaces the candidate refs
  rather than attempting any reconciliation or merge — reconciliation stays
  a manual, human action (merge one branch, edit one side), unchanged by
  this epic.

## Surfaces touched

- `internal/cli/show/show.go`
- `internal/cli/list/list.go`
- `internal/gitops/catfile.go` (`BlobReader`, consumed not modified)

## Out of scope

- `aiwf status`/`aiwf render --format=html` surfacing cross-branch-pending
  references — the epic's own deferred open question; a candidate
  follow-on gap if it turns out to matter.
- Any mutating verb accepting a `cross-branch-pending` or
  `cross-branch-collision` target (epic-level out of scope, unchanged).

## Dependencies

- `M-0259` — the widened cross-branch view and classification this
  milestone renders.
- `ADR-0030` (accepted).
- `G-0415` (addressed) — read-side half addressed by AC-3.

## References

- ADR-0030 — Extend cross-branch view to reference resolution and reads
- M-0259 — Add cross-branch-pending tier and collision detection to
  reference checks

## Work log

### AC-1 — Resolve content via BlobReader using the recorded cross-branch ref

`aiwf show` resolves an id's content live via `gitops.BlobReader` when
it misses the local working tree but is known cross-branch, scoped to
the queried id alone (`trunk.LocalRefHits`/`RemoteRefHits` filtered to
the one id, never the whole cross-branch view) so the common case
(local resolution) pays no extra subprocess cost. `aiwf list` mirrors
this for every cross-branch-known id not already in the local tree ·
commits 8017432a (show), 27256d29 (list) · tests 3/3 (show) + 7/7
(list) new, plus `trunk.DistinctRefs` 2/2 new

### AC-2 — Cross-branch-sourced content is visibly labeled

Added `show.CrossBranchView`/`list.ListSummary`'s
`CrossBranchRef`/`CrossBranchCollision`/`CrossBranchRefs` fields — a
resolved row carries the source ref, a collision row carries every
candidate ref instead of a single one. Text rendering marks both
distinctly (`show`: a `· cross-branch (ref: …)` header suffix or a
dedicated collision line; `list`: a `⇄` status-column marker) ·
commits 8017432a (show), 27256d29 (list) · same tests as AC-1 (the
labeling assertion is part of each fixture)

### AC-3 — Refuses to pick a ref when content diverges

A cross-branch-collision hit (`trunk.DetectCollisions`) never resolves
content from either side: `show` renders only identity plus the
candidate refs, `list` includes a collision row only for a kind-only
(or unfiltered) query — `--status`/`--parent`/`--area` each exclude it
(a filter match on data that doesn't exist yet would be a false
positive), `--archived` never does (an unresolved ambiguity should
stay visible by default) · commits 8017432a (show), 27256d29 (list) ·
same tests as AC-1/AC-2

### AC-4 — No working-tree, index, or ref writes

Snapshots HEAD, every ref, `git status --porcelain`, and a content
hash of `.git/index` around an `aiwf show` + `aiwf list` invocation
that resolves both a resolved and a collision cross-branch id,
asserting the repo is byte-identical before and after · commit
e15debb4 · tests 1/1 new

## Decisions made during implementation

- List's cross-branch filtering policy (kind-only/unfiltered for a
  collision row; full parity for a resolved row) was decided in
  conversation rather than pre-locked in the spec — see AC-3's Work
  log entry above for the reasoning; no separate ADR or `D-NNNN`
  decision record, since it's an implementation-scoping call for this
  milestone alone, not a durable cross-cutting decision.

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `make lint` (`golangci-lint`) — 0 issues (fixed 5 `gocritic`
  findings: 4 `rangeValCopy` from `ListSummary` growing past the
  copy-size threshold, 1 `ifElseChain` in `show.go`'s renderer).
- `go test -count=1 -parallel 8 ./...` — full suite green, including
  the pre-existing M-0241/AC-5 reachability-isolation stress scenario
  (updated to reflect this milestone's intentional behavior change —
  see Reviewer notes).
- `aiwf check` — no findings on the milestone.

## Deferrals

None — all 4 ACs landed in full within this milestone's scope.

## Reviewer notes

- **M-0241/AC-5 stress-scenario update.** `internal/stresstest/reachability_isolation.go`
  pinned a pre-ADR-0030 assumption: `aiwf show` must never find a
  sibling worktree's committed-but-unmerged entity. Linked worktrees
  share `refs/heads/*`, so that entity is exactly the cross-branch-known
  case this milestone resolves live — the assumption is deliberately
  superseded, not a regression. Updated the scenario, its classifier,
  and both classify/real-binary test suites to expect `aiwf show` to
  find it (labeled cross-branch); `check` and `history` keep their
  original isolation contract, since this milestone touches neither.
- **`tr.Root == ""` guard.** `crossBranchListRows` guards against a
  bare in-memory `*tree.Tree{}` (common in unit tests, `Root` left
  unset): `exec.Cmd.Dir == ""` means "inherit the calling process's
  cwd," which would otherwise run cross-branch git subprocesses
  against whatever directory the test process happens to be running
  in. Caught before it could corrupt any existing bare-tree unit test.
- **Lazy scanning, not `tree.CrossBranchHits`.** Considered reusing
  `Tree.CrossBranchHits`/`CrossBranchCollisions` (M-0259's eager,
  `LoadTreeWithTrunk`-populated fields) for the read side, but `show`/
  `list` call plain `tree.Load` (cheap, no trunk/ref scan) — switching
  them to `LoadTreeWithTrunk` would pay the cross-branch scan cost on
  every invocation, violating the epic's "never adds subprocess cost
  to the common case" constraint. `show`/`list` instead call
  `trunk.LocalRefHits`/`RemoteRefHits`/`DetectCollisions` directly,
  lazily, only on a local-tree miss (show) or from within a filtered
  listing (list, never the no-args counts path).

