---
id: M-0174
title: --area filter on list, show, and status
status: in_progress
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: list --area returns only entities whose effective area matches
      status: open
      tdd_phase: done
    - id: AC-2
      title: status --area scopes epics, decisions, and gaps to one area
      status: open
      tdd_phase: red
    - id: AC-3
      title: show --area shows the entity only when its effective area matches
      status: open
      tdd_phase: red
    - id: AC-4
      title: --area tab-completes the declared areas.members on list/show/status
      status: open
      tdd_phase: red
    - id: AC-5
      title: an undeclared --area value prints a note and yields an empty result
      status: open
      tdd_phase: red
    - id: AC-6
      title: untagged entities are excluded from a specific --area filter
      status: open
      tdd_phase: red
---
## Goal

Add an `--area <name>` filter to the read verbs `list`, `show`, and `status`, so an operator can scope each to a single workstream. Entities whose effective area (explicit for root kinds, parent-derived for milestones/ACs) matches the flag are shown; others are hidden.

## Context

M-0171 exposes each entity's effective `area` through the loaded model. This milestone consumes that for read-time scoping — the first half of "roadmaps/status/checks become scopeable per workstream." It is independent of the write-path milestone (it filters whatever is already tagged) and of the grouping milestone (filter narrows; grouping partitions).

## Acceptance criteria

### AC-1 — list --area returns only entities whose effective area matches

`aiwf list --area <name>` returns only entities whose effective area equals `<name>` — root kinds (epic, gap, decision, ADR) by their own `area` field, a milestone by its parent epic's area (derivation via `tree.ResolvedArea`). An empty `--area` applies no filter. The area axis is an independent AND-ed filter that composes with `--kind` / `--status` / `--parent`.

Evidence: `TestBuildListRows_AreaFilter` (exact match sets across kinds incl. milestone parent-derivation; empty = no-filter); the dispatcher seam `TestRunList_AreaViaDispatcher`.

### AC-2 — status --area scopes epics, decisions, and gaps to one area

`aiwf status --area <name>` scopes the entity-derived sections — in-flight epics (and their milestones), planned epics, open decisions, open gaps — to one workstream, applied at the Run layer via `FilterStatusByArea` after `BuildStatus` (which stays a pure full-report builder, so the HTML/roadmap render path is untouched). Recent activity, warnings, and health stay global — cross-cutting tree-health signals, not per-area concepts.

Evidence: `TestFilterStatusByArea` (two workstreams + empty-no-op; asserts Health count unchanged); the dispatcher seam `TestRunStatus_AreaViaDispatcher` (structural assertion on `in_flight_epics`, since global warnings legitimately still name out-of-area entities).

### AC-3 — show --area shows the entity only when its effective area matches

`aiwf show` is single-entity, so `--area` is a predicate: the entity is rendered only when its effective area equals the flag (composite `M-NNNN/AC-N` ids roll up to the parent epic's area via `tree.ResolvedAreaByID`); otherwise a one-line note (`<id> is in area "X", not "Y"`, or `... is untagged; not in area "Y"`) and exit 0 — the entity is hidden, like an empty filter, not an error. Not-found still takes precedence (exit 2). `--format json` emits a null result with `filtered_out` metadata. This keeps `--area` uniform across all three read verbs so a script can apply one filter everywhere.

Decision: resolves the spec's AC-3 fork — predicate filter (chosen over reject-with-error and no-flag).

Evidence: `TestAreaMissLine` (message shapes); `TestRunShow_AreaPredicate` (match / different-area miss / untagged miss / composite match + miss / undeclared note / json miss).

### AC-4 — --area tab-completes the declared areas.members on list/show/status

`--area <TAB>` on each of `list`, `show`, and `status` completes exactly the declared `aiwf.yaml: areas.members`, wired via `cmd.RegisterFlagCompletionFunc("area", cliutil.CompleteAreaFlag())` — the same completion source the M-0173 write path uses (single source of truth). The completion-drift policy stays green.

Evidence: `TestAreaCompletion_WiredOnReadVerbs` (each verb returns exactly the declared members); the drift policy `TestPolicy_FlagsHaveCompletion`.

### AC-5 — an undeclared --area value prints a note and yields an empty result

When `--area <name>` names a value not in the declared set (or no `areas` block exists), all three verbs print a one-line advisory note to stderr (`cliutil.UndeclaredAreaNote`, the shared single source), then proceed — the filter itself is mechanical (effective-area == value), so the result is empty/hidden in the common case but still surfaces a hand-edited entity carrying that undeclared area (the M-0172 `area-unknown` check is the backstop for the mis-tag). Exit 0: reads are non-destructive. This is the deliberate asymmetry with `aiwf add --area`, which rejects undeclared values — a write must protect data integrity; a read need not.

Evidence: `TestUndeclaredAreaNote` (empty/declared are silent; undeclared names the value + declared set; no-block names the missing block); the undeclared subcases of `TestRunList_AreaViaDispatcher`, `TestRunStatus_AreaViaDispatcher`, and `TestRunShow_AreaPredicate`.

### AC-6 — untagged entities are excluded from a specific --area filter

An untagged entity (effective area "") never matches a named `--area`, so it is excluded from `list --area X` and `status --area X` and surfaces only under the no-filter view. Grouping the untagged complement under the `default:` label is M-0175, not this milestone.

Evidence: `TestBuildListRows_ExcludesUntagged` (list); the per-workstream assertions in `TestFilterStatusByArea` (the untagged epic and untagged gap are excluded from each named area).

## Constraints

- **Read-only.** No mutation, no commit. `--area` is a view filter.
- **Effective-area is computed once** in the loaded model (M-0171), not re-derived per verb — single source of truth.

## Out of scope

- Area *grouping* (sectioned output) — that's the grouping milestone; this milestone only *filters*.
- The write path and the check finding.

## Dependencies

- M-0171 — effective-area exposure on the loaded model.

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)

## Work log

### AC-1 — list filter
`--area` added to `BuildListRows` as an AND-ed axis; `list.Run` + flag + completion wired; the no-args counts path now treats a lone `--area` as a row query. · tests: unit + untagged-exclusion + dispatcher seam.

### AC-2 — status scope
`FilterStatusByArea` applied post-`BuildStatus` at the Run layer; the HTML/roadmap render resolver is untouched (full report). · tests: unit (incl. Health-unchanged) + dispatcher seam.

### AC-3 — show predicate
predicate via `ResolvedAreaByID` after the existence check (so not-found beats filtered-out); text miss line + json null-result; `AreaMissLine` helper. · tests: `AreaMissLine` unit + 7 dispatcher subcases.

### AC-4 — completion
`CompleteAreaFlag` wired on all three verbs. · tests: focused per-verb completion + the live-tree drift policy.

### AC-5 — undeclared note
shared `cliutil.UndeclaredAreaNote`; stderr; exit 0. · tests: helper unit + the undeclared subcase on each verb.

### AC-6 — untagged exclusion
falls out of the mechanical effective-area == value filter. · tests: list exclusion + status per-workstream assertions.

The phase timeline is in `aiwf history M-0174/AC-N`; not duplicated here.

## Decisions made during implementation

- **AC-3 = predicate filter.** `show` is single-entity; `--area` shows the entity iff its effective area matches, else a one-line note + exit 0. Chosen over reject-with-error and no-flag so the flag is uniform across `list` / `show` / `status` (a script can apply one `--area` everywhere). Confirmed with the user before build.
- **AC-2 = entity sections scoped; health / recent activity / warnings stay global.** `--area` scopes the planning view; tree-health is a whole-repo concern and filtering git history by area is out of scope.
- **AC-5 = note, not reject.** Reads are non-destructive, so an undeclared `--area` notes and yields empty rather than erroring — the principled asymmetry with the `aiwf add --area` write path. No separate architectural-decision record warranted; the rationale lives here and in the epic's resolved questions.

## Validation

- `go test ./internal/cli/...` — all packages pass (integration ~99–106s).
- `golangci-lint run ./internal/cli/...` — 0 issues.
- `go test ./internal/policies/` — pass (skill-coverage and completion-drift stay green with the new flag + skill edits).
- Branch-coverage: every changed line is exercised; one annotated `//coverage:ignore` (the `show` json-miss `render.JSON` write-failure branch). New functions are 100%.
- Independent fresh-context review: APPROVE — every AC verified by measurement (binary exercised end-to-end, each filter line severed to prove non-vacuity, coverage merged). One non-blocking finding (a stray duplicate doc-comment) fixed inline.

## Deferrals

- None. Area *grouping* (the `default:` complement and sectioned output across status/render) is M-0175 by epic design, not a deferral of this milestone.

## Reviewer notes

- The status filter is post-`BuildStatus` (`FilterStatusByArea`) rather than a `BuildStatus` parameter — deliberate: `BuildStatus` stays a pure full-report builder (its many test callers and the HTML/roadmap render resolver are untouched), and area-scoping is isolated in one testable function. `BuildListRows`, by contrast, *is* the list filter function, so `--area` joins its other axes there.
- The undeclared-area note and the filter are independent: the note is advisory (stderr), the filter mechanical. A hand-edited entity tagged with an undeclared area still surfaces under `--area <that-value>` — useful for finding what the M-0172 `area-unknown` check flags.
- Epic E-0043 Open Question 1 (gap derive-on-omit) was resolved in M-0173; this milestone's AC-3 predicate decision is recorded above. The epic's open-question table reconciles at the epic wrap (after M-0175).

