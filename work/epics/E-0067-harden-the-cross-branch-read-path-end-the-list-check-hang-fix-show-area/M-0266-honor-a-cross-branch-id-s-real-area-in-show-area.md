---
id: M-0266
title: Honor a cross-branch id's real area in show --area
status: in_progress
parent: E-0067
depends_on:
    - M-0265
tdd: required
acs:
    - id: AC-1
      title: show --area on a cross-branch id evaluates against the entity's real area
      status: met
      tdd_phase: done
---

## Goal

`aiwf show <id> --area X` on a cross-branch-resolved id should evaluate the `--area`
predicate against the entity's real `area:` field, not the local-only lookup that
always reports untagged. Fixes G-0419.

## Context

M-0265 introduces the shared cross-branch scan helper. `show`'s cross-branch path
(`buildCrossBranchShowView`) already parses the resolved entity, including its
`area:` field, but the `--area` predicate still routes through
`tr.ResolvedAreaByID`, which consults only the local tree — so for a
cross-branch-resolved id it returns untagged regardless of the entity's real area.
This milestone threads the resolved area through.

## Acceptance criteria

### AC-1 — show --area on a cross-branch id evaluates against the entity's real area

`aiwf show <cross-branch-id> --area X` reports the entity in-area when its real
`area:` on the resolving ref equals X, and out-of-area otherwise — instead of
always reporting untagged. Verified against a cross-branch fixture whose entity
carries a real area on the ref it resolves from.

## Constraints

- Local-id `--area` behavior is unchanged; only the cross-branch-resolved path is
  corrected.
- No new package-level mutable state; `show`'s cross-branch read stays best-effort.

## Design notes

- The resolved entity's `Area` is already read in `buildCrossBranchShowView` via
  `entity.Parse`; thread that value into the `--area` predicate for the
  cross-branch case rather than falling back to `tr.ResolvedAreaByID`'s local-only
  lookup (per G-0419).

## Surfaces touched

- `internal/cli/show/show.go`

## Out of scope

- Any change to `list --area` or local-id `--area` behavior.
- The M-0265 helper (dependency, already landed).
- The cross-branch **milestone** `--area` roll-up: a milestone never stores its
  own `area:` (it derives from the parent epic), so the own-field read here
  reports a cross-branch milestone untagged. This milestone fixes the own-area
  kinds (epic, gap, ADR, decision, contract); the milestone roll-up across refs
  is tracked by G-0421.

## Dependencies

- M-0265 — the shared cross-branch scan helper and resolved-entity read this
  milestone threads the area through.

## References

- Gaps: G-0419 (addressed by this milestone), G-0421 (deferred follow-up). Epic:
  E-0067.

## Work log

### AC-1 — show --area on a cross-branch id evaluates against the entity's real area

Threaded the resolving ref's real `area:` onto the show view (an in-memory
`Area` field tagged `json:"-"`, so the JSON envelope is unchanged) and routed the
`--area` predicate through it for the cross-branch case, replacing the local-only
`tr.ResolvedAreaByID` lookup that reported any cross-branch id untagged. ·
commit `8fb9cfe1` · `internal/cli/integration` green.

## Decisions made during implementation

- (none) — the fix follows the milestone's design note directly (thread
  `resolved.Area` into the predicate); no mid-flight decision that warranted an
  ADR or decision record.

## Validation

- `go build ./...` clean; `internal/cli/show`, `internal/cli/integration`, and
  `internal/tree` tests green (no regression).
- Branch-coverage audit: both arms of the new `if view.CrossBranch != nil`
  predicate exercised — TRUE by the cross-branch test, FALSE by the existing
  local `TestRunShow_AreaPredicate` — confirmed against a cross-package coverage
  profile (`show.go` predicate block covered).
- Vacuity: a mutation neutering the `actual = view.Area` assignment drove the AC
  test red (the pre-fix "untagged" miss returned); restored byte-exact. The
  independent reviewer reproduced the same revert independently.
- Binary smoke (rebuilt from HEAD, real cross-branch fixture): a gap minted on a
  sibling branch with `area: platform`, absent from the checkout, renders in-area
  under `--area platform` and misses naming its real area (`is in area
  "platform", not "billing"`) under `--area billing`; `--format=json` gains no
  `area` key (the envelope shape is unchanged).
- `gofmt`/`go vet` clean on the touched files. `aiwf check`: 0 errors (two benign
  warnings — no drafted milestone left under the active epic, and the worktree
  branch has no upstream so the provenance audit is skipped).

## Deferrals

- **G-0421** — cross-branch **milestone** `--area` should honor the parent
  epic's rolled-up area. The own-field read shipped here reports a cross-branch
  milestone untagged; the roll-up across refs is deferred (it must decide how far
  to follow a possibly-cross-branch parent chain). Surfaced by the independent
  review; out of AC-1's stated scope.

## Reviewer notes

- **Independent code-quality review before wrap — APPROVE, no blocking
  findings.** A fresh-context reviewer verified each load-bearing claim by
  measuring, not reasoning: built the binary and ran a real cross-branch fixture
  (in-area renders, out-of-area names the real area); independently reverted the
  production hunk and reproduced the exact pre-fix bug (the test is a genuine
  gate); confirmed the local `--area` path is unchanged (`view.CrossBranch` nil →
  original `tr.ResolvedAreaByID`); confirmed `--format=json` gains no `area` key
  and no whole-struct `cmp.Diff` on `ShowView` breaks; and confirmed both new
  branch arms are covered via the coverage profile.
- **Named limitation (accepted, deferred to G-0421).** The fix reads the
  entity's own parsed `resolved.Area`, not a parent-chain roll-up — correct for
  the own-area kinds AC-1 targets, but it leaves a cross-branch milestone
  reporting untagged. `entity.Parse` does not blank a milestone's `area:` (only
  the tree loader does, post-parse), so the milestone case is empty because a
  milestone simply never stores an own area. Not a regression; tracked by G-0421
  and noted under *Out of scope*.
- **Collision + `--area` (low priority, no gap).** On a cross-branch collision,
  `view.Area` is empty (content in dispute), so `--area` filters the entity out
  as untagged rather than reaching the "content diverges" line. This is the same
  already-covered `if view.CrossBranch != nil` branch, is documented in-code, and
  is a defensible reading (the area is genuinely in dispute); left as-is.
