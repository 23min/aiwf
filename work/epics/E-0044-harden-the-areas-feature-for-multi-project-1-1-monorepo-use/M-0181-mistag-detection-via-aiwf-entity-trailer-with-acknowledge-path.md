---
id: M-0181
title: Mistag detection via aiwf-entity trailer with acknowledge path
status: done
parent: E-0044
depends_on:
    - M-0179
tdd: required
acs:
    - id: AC-1
      title: Gather an entity's commits and touched paths via the aiwf-entity trailer
      status: met
      tdd_phase: done
    - id: AC-2
      title: area-mistag fires when all area-claimed work lands in foreign areas
      status: met
      tdd_phase: done
    - id: AC-3
      title: No finding when some touched paths land in the entity's own area
      status: met
      tdd_phase: done
    - id: AC-4
      title: Inert with no paths, no linked commits, global area, or archived entity
      status: met
      tdd_phase: done
    - id: AC-5
      title: Regroup acknowledge-illegal into the aiwf acknowledge illegal subverb
      status: met
      tdd_phase: done
    - id: AC-6
      title: aiwf acknowledge mistag records a sovereign ack the check suppresses
      status: met
      tdd_phase: done
    - id: AC-7
      title: area-mistag and the acknowledge surface are discoverable and pinned
      status: met
      tdd_phase: done
---
## Goal

Flag a landed entity whose commits touch only another area's paths: gather the entity's commits via the `aiwf-entity:` trailer, intersect the touched files with the entity's area glob, and warn when the diff falls entirely outside it — with a sovereign-traced acknowledge path for legitimate cross-cutting work.

## Context

This is the check that actually catches "filed against the wrong area, flew under the radar" — the failure label-only areas are blind to. With `paths:` (M-0179) and the entity ↔ commit linkage aiwf already records via trailers, the touched-files-vs-glob comparison becomes buildable.

## Acceptance criteria

The seven ACs are formalized in frontmatter `acs[]` and detailed in the `### AC-N` sections below, each with the test that pins it. A load-bearing refinement over the candidate sketch: only paths that match *some* declared area glob participate, so planning files, docs, and unclaimed code never false-fire — "entirely outside the area" means "all area-claimed work landed in foreign areas," not "touched any file outside the glob."

## Constraints

- Warning severity, never gating; legitimate cross-cutting exists.
- Acknowledge is sovereign-traced (human actor, written reason), per the provenance model.

## Out of scope

- Auto-correcting the tag — suggestion / derivation is the auto-derive milestone.

## Dependencies

- M-0179 (`paths:` per area) — the oracle the diff is checked against.

## References

- `aiwf-acknowledge` skill (formerly `aiwf-acknowledge-illegal`) — the acknowledge-with-reason precedent; AC-5/AC-7 made it topical and added the `mistag` subverb.
- The `aiwf-entity:` commit trailer — the entity ↔ commit linkage this reads.

### AC-1 — Gather an entity's commits and touched paths via the aiwf-entity trailer

`check.GatherEntityPaths(ctx, root)` walks HEAD-reachable history once (one `git log --no-renames --name-only` pass, control-char-framed so name-only newlines can't confuse the parse, `core.quotePath=false` so non-ASCII paths aren't dropped) and returns, per canonical root entity id, the union of paths its `aiwf-entity:`-trailered commits touched. Composite AC trailers roll up to the parent milestone; ids canonicalize at ingest. The union is **unfiltered** (planning + code paths alike) — the area filtering is AC-2's job, keeping gather and filter as separate testable units. Pinned by `TestGatherEntityPaths` (multi-entity, composite rollup, narrow-width canonicalization, empty-trailer skip) and the inert arms.

### AC-2 — area-mistag fires when all area-claimed work lands in foreign areas

`check.AreaMistag` emits one `area-mistag` warning when an entity's area-claimed work landed in a foreign area's `paths:` territory. Only paths matching some declared glob participate (the "area-claimed space" guard), so planning/docs/unclaimed paths never false-fire. The effective area comes from `Tree.ResolvedArea` (a milestone is judged against its parent epic's area); match semantics route through the `internal/areamatch` SSOT. Warning severity, and deliberately **never** escalated by `ApplyAreaRequiredStrict` (per the Constraints). Pinned by `TestAreaMistag_FiresOnForeignAreaWork` (with a planning-only paired control, mutation-hardened) and the `TestRunCheck_AreaMistagSurfacesViaDispatcher` seam test.

### AC-3 — No finding when some touched paths land in the entity's own area

Cross-cutting is tolerated: if any touched path matches the entity's own area glob, the rule does not fire even when other work landed elsewhere (the `insideOwn` guard). Pinned by `TestAreaMistag_TolerantOfCrossCutting` (mutation-verified: removing the tolerance reds the test).

### AC-4 — Inert with no paths, no linked commits, global area, or archived entity

No `area-mistag` when no area declares `paths:`, the entity is untagged or carries the reserved `global` sentinel (inherently cross-cutting, ADR-0021), the entity's own area declares no paths, the entity has no linked commits, or the entity is archived (ADR-0004 §"check shape rules"). The CLI seam additionally gates the (full-history) gather behind `AnyAreaHasPaths`, so a consumer with no path-carrying area pays no walk. Pinned by the `TestAreaMistag_NoFinding` unit table (one case per guard) plus `TestRunCheck_AreaMistag_InertWhenNoAreaDeclaresPaths` and `...SkipsGlobalTaggedEntity` (seam).

### AC-5 — Regroup acknowledge-illegal into the aiwf acknowledge illegal subverb

The shipped top-level `acknowledge-illegal` became a subverb under a new non-Runnable `acknowledge` parent (mirroring `aiwf contract`). The `aiwf-verb: acknowledge-illegal` **trailer value is unchanged** — the command path `acknowledge illegal` enumerates to the same `acknowledge-illegal` string via the hyphen-join walker, so history, the `trailer-verb-unknown` rule, and the commit-msg hook all keep validating with no shim. Clean break (no command alias). Pinned by `TestTrailerShapePerMutatingVerb` and the skill-coverage / completion-drift / m0123 drift gates.

### AC-6 — aiwf acknowledge mistag records a sovereign ack the check suppresses

`aiwf acknowledge mistag <id>` records a per-entity sovereign ack (human actor + non-empty `--reason`, entity-must-exist, empty commit carrying `aiwf-verb: acknowledge-mistag` + `aiwf-entity`); `check.WalkAcknowledgedMistags` reads those commits and `AreaMistag` exempts the named entities. The suppression is per-entity and permanent — a deliberate choice (D-0027). Empty-commit marker registered as the 4th shape in the `empty-diff-commits-carry-marker` policy. Pinned by `TestAcknowledgeMistag` (verb), `TestWalkAcknowledgedMistags` (walker), `TestAreaMistag_SuppressedByAcknowledgement` (unit), and the end-to-end `TestRunCheck_AreaMistag_AcknowledgeSuppresses` — the verb↔walker drift guard.

### AC-7 — area-mistag and the acknowledge surface are discoverable and pinned

The topical `aiwf-acknowledge` skill teaches both subverbs (`illegal`, `mistag`); `area-mistag` is a warnings-table row + a `hint.go` entry; the `mistag <id>` subverb wires positional entity-id completion. Pinned structurally by `TestAcknowledgeSkill_TeachesBothSubverbs`, `TestAreaMistagFinding_StructurallyDocumented` (markdownSection-scoped, self-guarded), and `TestMistagCmd_HasPositionalCompletion`.

## Work log

- **AC-1** — gather helper; commit `f3eebe8e` · tests green.
- **AC-2** — `area-mistag` fire + finding code + hint + skill row + CLI seam; commit `b2492183`.
- **AC-3** — cross-cutting tolerance; commit `82d96e7d`.
- **AC-4** — seam-level inertia pins; commit `26584896`.
- **AC-5** — acknowledge regroup (39-path refactor); commit `24b25000`.
- **AC-6** — `aiwf acknowledge mistag` + suppression + empty-diff 4th shape; commit `9e6cd8d7`.
- **AC-7** — topical skill + discoverability pins; commit `9acbbeb1`.
- Self-review corrections (SHA-shape branch test + migration-prose); commit `c21dd891`.
- Pre-wrap review fixes + mutation-hardening; commit `991ebf3b`.

(Per-AC TDD phase timelines are in `aiwf history M-0181/AC-<N>`.)

## Decisions made during implementation

- **D-0027** — area-mistag acknowledgement is per-entity and permanent (vs the scoped `(entity, blessed-area)` alternative). Rationale + the known refinement recorded in the decision.
- The acknowledge family is **grouped** (`aiwf acknowledge <illegal|mistag>`) rather than two flat verbs, with a **clean break** of the old command (no alias) — trailer/history back-compat is free via the hyphen-join enumerator.

## Validation

- `make ci` green (race + coverage 85.0% + self-check 29 steps).
- `make coverage-gate` (base origin/main) green after fixing inherited setarea/renamearea error-arm coverage debt; M-0181's own diff is branch-coverage clean.
- Full `internal/...` suite green; `golangci-lint` 0 issues; `aiwf check` 0 errors on M-0181.
- Mutation testing (gremlins on `internal/cli/acknowledge` + 14 targeted manual mutations on `area_mistag.go` / `acknowledgemistag.go`): all core-logic mutants killed after hardening `TestAreaMistag_FiresOnForeignAreaWork`; the acknowledge-CLI survivors are equivalent mutants (CLI guards redundant with verb guards).

## Deferrals

- **G-0304** — consolidate the three ack-walker HEAD-walk loops into one `forEachCommitTrailers` primitive (rule-of-three crossed; surfaced by the pre-wrap `wf-rethink`). DRY cleanup, not a correctness fix.
- No AC deferred or cancelled — all seven met.

## Reviewer notes

- Two independent code reviews (AC-2: request-changes → fixed; AC-5 + the full pre-wrap pass: request-changes → fixed) and two `wf-rethink` passes (the detection model; the acknowledge namespace) — both `keep`. All blockers/nits resolved before this wrap.
- The pre-wrap coverage blocker was **inherited** setarea/renamearea error-arm debt (from M-0179/M-0184), not M-0181's code; fixed in `991ebf3b` so the epic→main coverage-gate stays green.
- Mutation testing found one genuinely weak test (the fire test passed under an inverted foreign-match because its fixture's planning file masked the break); hardened with a paired control.
- Per-entity-permanent suppression (D-0027) is the deliberate granularity; the scoped shape is the known answer if stale-suppression friction ever appears.
- Performance: the mistag gather is one full-history `git log` when a paths-carrying area exists; reusing `gitops.BulkRevwalk` was rejected by the `wf-rethink` on correctness grounds (it is `--all`-scoped and omits `quotePath=false`), so the bespoke HEAD-scoped pass is deliberate.

