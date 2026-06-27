---
id: M-0188
title: Pin that the loader ignores in-repo worktrees under .claude/worktrees
status: in_progress
parent: E-0046
tdd: none
acs:
    - id: AC-1
      title: Loader ignores a nested checkout under .claude/worktrees
      status: met
    - id: AC-2
      title: In-scope duplicate id is still reported as a collision
      status: open
---

# M-0188 — Pin that the loader ignores in-repo worktrees under .claude/worktrees

## Goal

Pin, with a regression test, that the aiwf loader / `aiwf check` does not descend into
in-repo worktrees under `.claude/worktrees/` — so a nested second checkout there cannot
surface phantom duplicate entities once in-repo worktrees become the default placement.

## Acceptance criteria

Tracked in frontmatter `acs[]` and detailed in the `### AC-1` / `### AC-2` sections below.

## Context

The epic (E-0046) makes in-repo worktrees the default placement. An in-repo worktree is a
full second checkout of the repo *inside the tree*, including its own `work/...`. If the
loader walked from the repo root into `.claude/worktrees/`, it would load duplicate entity
files and report false id collisions. The behavior is likely already correct (`.claude/*`
is gitignored; the loader reads `work/`/`docs/`), so this milestone verifies first, then
pins the result — it must not remain an assumption.

## Constraints

- Pins behavior, not implementation: asserts `aiwf check` output on a fixture, with a
  vacuity check that the assertion fails when the guard is removed.
- Resolve entity paths via the loader, never hardcoded (CLAUDE.md "Policy tests … resolve
  via the loader").

## Out of scope

- The config knob (M-0189) and the ritual default (M-0190) — this milestone only guards
  the loader.

## Dependencies

- None. Sequenced first to de-risk the default flip.

## References

- E-0046 epic spec; CLAUDE.md "Subagent worktree isolation".

### AC-1 — Loader ignores a nested checkout under .claude/worktrees

A planning tree containing a full second checkout under
`.claude/worktrees/<branch>/work/...` (the in-repo-worktree default placement per
ADR-0023) that duplicates a real entity's id must load **no** entity from under
`.claude/` and produce **no** `ids-unique` finding — the nested copy never enters the
tree, so it cannot collide with the real entity.

Evidence: `TestLoaderIgnoresNestedWorktreeCheckout` in
`internal/check/worktree_scoping_test.go` drives `tree.Load` + `check.Run` on exactly
that fixture. Structurally backstopped by the nested-checkout-shape negative row added to
`TestPathKind` (`internal/entity`): `PathKind` requires the first path segment to be
`work`/`docs`, so a `.claude/worktrees/.../work/epics/...` path is unrecognizable as an
entity even if the walk ever reached it.

### AC-2 — In-scope duplicate id is still reported as a collision

The same duplicate id, placed at an *in-scope* path (under `work/epics/`), **must** still
be reported as an `ids-unique` collision — proving the detector is live and AC-1's clean
result is not vacuous (AC-1 passes specifically because `.claude/worktrees/` is outside
the loader's walk scope, not because collisions go unreported).

Evidence: `TestInScopeDuplicateIDStillFires` in
`internal/check/worktree_scoping_test.go`.

## Work log

### AC-1 — Loader ignores a nested checkout under .claude/worktrees

Pinned the loader's walk-scoping at the `tree.Load` + `check.Run` seam. Verified
empirically first (the spec's verify-then-pin intent): a real 1536-file in-repo worktree
at `.claude/worktrees/E-0046/` produced `ok — no findings` from `aiwf check` in the main
checkout. Finding: the nested checkout is ignored by **two** independent guards —
(1) `tree.Load` walks only `work/{epics,gaps,decisions,contracts}` + `docs/adr`, never the
repo root (tree.go:163-172); (2) `entity.PathKind` requires the first path segment to be
`work`/`docs` (entity.go:643), so a `.claude/worktrees/...` path is structurally
unrecognizable even if walked. Pinned both: a seam test (`tree.Load` + `check.Run`) for
guard (1) / the observable guarantee, plus a `TestPathKind` negative row for guard (2). ·
tests: 2 new + 1 table row · commit at wrap

### AC-2 — In-scope duplicate id is still reported as a collision

Non-vacuity guard: `TestInScopeDuplicateIDStillFires` confirms `ids-unique` fires on a
genuine in-scope duplicate, so AC-1's clean result is informative. · tests: 1 new ·
commit at wrap

## Validation

- `make ci` — green (vet, lint, test-cov with race + coverage, 29-step self-check).
- `go test ./...` — exit 0, no failures across all packages.
- `aiwf check` on M-0188 — 0 errors.
- Net change: 3 tests (2 new in `internal/check/worktree_scoping_test.go`, 1 new
  `TestPathKind` negative row in `internal/entity`). Zero production code changed — this
  milestone pins existing behavior.

## Reviewer notes

- **Two-guard finding.** The in-repo-worktree default is safe because the loader ignores
  a nested `.claude/worktrees/` checkout via two *independent* guards: `tree.Load`'s
  walk-root scoping and `entity.PathKind`'s `work`/`docs` first-segment requirement.
  Defeating either guard alone leaves the observable guarantee intact; only defeating both
  surfaces a phantom collision. AC-1 pins the observable guarantee; the `TestPathKind` row
  additionally pins guard (2) against a substring-loosening regression the integration
  test alone would miss.
- **AC-1 vacuity hardened.** The first cut asserted only "no `.claude/` entity / no
  collision" — which an empty/degenerate tree would satisfy vacuously. Strengthened to
  assert the real in-scope entity is actually loaded from its in-scope path. Verified
  empirically that AC-1 goes red when both guards are temporarily defeated (changes
  reverted clean).
- **Independent review.** A fresh-context reviewer (no authorship attachment) verdict:
  **APPROVE**, zero blocking findings. It ran its own break-both-guards experiment
  (reverted, blob hashes confirmed pristine). Two advisory nits fixed inline: the variable
  rename `real`→`inScope` (also the gocritic `builtinShadow` lint fix `make ci` caught),
  `epicFixture`→`worktreeScopingEpic` (avoid a generic name in the shared `check` test
  package), and "integration test"→"seam test" wording above.
- **No deferrals.**

