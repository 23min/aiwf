---
id: M-0246
title: Wire archive to rewrite link destinations on sweep
status: in_progress
parent: E-0063
depends_on:
    - M-0245
tdd: required
acs:
    - id: AC-1
      title: Archive rewrites entity-body links to a swept entity's archive path
      status: met
      tdd_phase: done
    - id: AC-2
      title: A multi-entity sweep recomputes links against the final post-move layout
      status: met
      tdd_phase: done
---

## Goal

Make `aiwf archive` fix the links it currently strands: when it sweeps a terminal
entity into `archive/`, rewrite every entity-body link pointing at it, in the same
commit, through the shared primitive.

## Context

`archive` today is a pure `git mv` — `planArchive` emits only `OpMove`, never
`OpWrite`, so no other file's references are touched (`internal/verb/archive.go`).
This is the verb whose rot was measured (the links from ADR bodies into `work/`). The move
changes the *directory prefix* (`work/gaps/` → `work/gaps/archive/`), not the id
or slug, so the transform here inserts the `/archive/` segment — a case `rewidth`'s
pattern deliberately excludes. With M-0245's primitive in place, this milestone
computes the affected bodies and adds the writes.

## Acceptance criteria

### AC-1 — Archive rewrites entity-body links to a swept entity's archive path

After `aiwf archive` sweeps entity B, every entity-body link whose destination
resolved to B now points at B's archive path and resolves; links to non-swept
entities and any prose mention of B are unchanged. Evidence: a real-tree
integration test — build A→B link, archive B, assert A's link resolves to the
archive location and the commit contains exactly the expected body writes.

### AC-2 — A multi-entity sweep recomputes links against the final post-move layout

A sweep that moves several entities at once — including an epic subtree whose
children move via the dir-rename — recomputes each affected link against the final
layout, so a link between two entities that both moved in the same sweep is
correct afterward. Evidence: an integration test with an A→B link where both A and
B are swept in one run.

## Constraints

- Rewriting runs at move-time only; the pre-push chokepoint stays untouched.
- Writes are scoped to entity bodies the loader owns — no reach into non-entity
  `docs`/`README`.
- ADR-0004 preserved: archive still physically moves; no redirect stub.

## Design notes

- Destination transform = insert `/archive/` into the kind directory; reuse
  M-0245's region-splitter and resolution for everything else.
- Multi-move correctness: destinations resolve against the post-sweep layout, not
  incrementally per file.
- Decision recorded in `ADR-0033` (*Entity path-links are first-class and
  rewritten on move*).

## Surfaces touched

- `internal/verb/archive.go`
- the shared primitive from M-0245

## Out of scope

- `rename` / `retitle` / `reallocate` (sibling milestones).
- Non-entity narrative files.

## Dependencies

- M-0245 — the shared rewrite primitive.

## References

- `internal/verb/archive.go`
- ADR-0004 — uniform archive convention
- G-0392

---

## Work log

### AC-1 — Archive rewrites entity-body links to a swept entity's archive path

Green · commit f7426e90 · tests 3/3

### AC-2 — A multi-entity sweep recomputes links against the final post-move layout

Green · commit f7426e90 · tests 3/3

Both ACs landed in one implementation commit: `planArchiveRewrites` and
`archiveEntityMoves` in `internal/verb/archive.go`, plus three real-tree
integration tests in the new `internal/verb/archive_linkrewrite_test.go`
(AC-1's rewrite + untouched-region case, AC-2's nested-milestone +
same-sweep case, and a branch test pinning that an already-archived
entity is never treated as a linking-file candidate). Directory-shaped
moves (epic/contract) expand into one `EntityMove` per nested entity
file via `pathInside` / `newEntityPathAfterRename` — the same pattern
`reallocate` already uses for its own directory-rename case — closing
the nested-milestone-link gap M-0245's reviewer flagged as uncovered.

## Decisions made during implementation

- (none) — all decisions are pre-locked in `## Design notes` above and
  in `ADR-0033`.

## Validation

- `go build ./...` — green.
- `go test -race -parallel 8 ./...` — green, full suite.
- `golangci-lint run` — 0 issues.
- `make coverage-gate` — diff-scoped branch-coverage audit and firing-fixture policy tests both pass; every changed line is tested or `//coverage:ignore`'d with a stated rationale.
- `aiwf check` — 0 error-severity findings (1 advisory `provenance-untrailered-scope-undefined`, expected on an unpushed branch with no upstream).
- Independent code-quality review (fresh-context subagent): **approve**, no blocking findings. Re-ran the full test suite, `make coverage-gate`, and `make lint` itself rather than trusting this spec's claims; verified the `pathInside`/`newEntityPathAfterRename` reuse against `reallocate.go`'s real definitions, verified the `OpMove`-before-`OpWrite` ordering guarantee against `apply.go` directly, and confirmed AC-2's test genuinely forces the nested-dir/same-sweep case rather than a weaker one.
- No design-quality (`wf-rethink`) lens run: this milestone composes existing primitives (`EntityMove`, `pathInside`, `newEntityPathAfterRename`, `RewriteLinkDestinations`) into two internal helper functions following an established pattern (mirroring `reallocate`/`rewidth`) — no new module boundary, core abstraction, or data model to rethink.

## Deferrals

- (none)

## Reviewer notes

`archiveEntityMoves`'s directory-shaped case handles `KindEpic` and
`KindContract` in one combined switch arm, since both are dir-shaped
and the expansion logic (walk `tr.Entities`, keep what's `pathInside`
the moved dir) doesn't differ between them. Only the epic path has a
dedicated integration test (`TestArchive_MultiEntitySweep_...`);
contract coverage rides on the shared code path plus the pre-existing
`TestArchiveTargetForContract`/`computeArchiveMoves` unit tests, not a
link-rewrite-specific one. Low risk (identical code, no per-kind
branch), but a contract-linking-file fixture would pin it explicitly
if a future change ever splits the two kinds' handling apart.

The commit-body's `Body rewrites: N file(s)` line (`archiveCommitBody`'s
`rewriteCount` parameter) is exercised by the AC tests for coverage
purposes but no test asserts the rendered string itself — parity with
`rewidth`'s identical convention, whose own commit-body renderer has
the same gap.
