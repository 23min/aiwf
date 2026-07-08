---
id: M-0239
title: Correlation id wiring; ratify ADR-0017
status: in_progress
parent: E-0061
depends_on:
    - M-0237
    - M-0238
tdd: required
acs:
    - id: AC-1
      title: An envelope's correlation_id matches the run_id in that invocation's log lines
      status: met
      tdd_phase: done
    - id: AC-2
      title: Mutating verbs report per-verb-appropriate metadata in their envelope
      status: met
      tdd_phase: done
    - id: AC-3
      title: An operator can pass --trace to see per-phase timings via the logger
      status: open
      tdd_phase: red
    - id: AC-4
      title: A renamed Envelope field is caught by a structural policy test
      status: open
      tdd_phase: red
    - id: AC-5
      title: ADR-0017 reads accepted with CLAUDE.md matching shipped behavior
      status: open
      tdd_phase: red
---

## Goal

Close the loop between an invocation's JSON envelope and its diagnostic-log
lines with one shared correlation id, so RCA on any finding is a grep, not a
manual timestamp-matching exercise — then ratify ADR-0017 now that the
codebase actually matches it.

## Context

`render.Envelope.Metadata.correlation_id` is declared today but dead: no
caller populates it. M-0237 shipped the logger (with its own per-invocation
`run_id`); M-0238 migrated the known diagnostic call sites onto it. This
milestone is the capstone: it ties those two pieces together with one id,
and — because ratifying an ADR means the implementation now matches the
decision in full, not just in part — this is also where ADR-0017 moves
`proposed → accepted`.

## Acceptance criteria

### AC-1 — An envelope's correlation_id matches the run_id in that invocation's log lines

The Cobra root mints one id per invocation (a UUID) and threads it into
`render.Envelope.Metadata.correlation_id`. The same id is passed into
`logger.WithVerb(...)` as `run_id`. One grep on either value finds the other.

### AC-2 — Mutating verbs report per-verb-appropriate metadata in their envelope

Today only read-only verbs (e.g. `aiwf check`) populate `metadata`. Mutating
verbs gain their own per-verb-appropriate fields: `promote` reports
`entity_id`/`from`/`to`/`commit_sha`; `archive` reports
`swept_count`/`commit_sha`; and so on per verb. The shape is per-verb; the
discipline (every mutating verb reports *something*) is uniform.

### AC-3 — An operator can pass --trace to see per-phase timings via the logger

`--trace` is a logger consumer, not an envelope consumer — it depends on
M-0237's logger existing, which it now does. Emits per-phase timings at
`debug` level through the same bound logger, not a separate mechanism.

### AC-4 — A renamed Envelope field is caught by a structural policy test

`internal/policies/envelope_structural_assertion.go` pins the envelope's
required-key set against the `Envelope` struct's field tags, so a future
field rename that would silently break a downstream JSON consumer fails CI
instead.

### AC-5 — ADR-0017 reads accepted with CLAUDE.md matching shipped behavior

`aiwf promote ADR-0017 accepted` once AC-1 through AC-4 (and M-0237, M-0238)
are done. CLAUDE.md's Go conventions §CLI conventions logging paragraph is
rewritten to reflect the shipped opt-in/XDG-file/`forbidigo` behavior,
replacing the stale "log/slog to stderr default INFO" prescription, with a
cross-link to ADR-0017.

## Constraints

- `correlation_id` is an opaque per-invocation identifier — never compared
  or branched on for anything but exact-match correlation.
- Ratifying the ADR (AC-5) is the last thing that happens in this
  milestone, not the first — it certifies a state that must already be true.

## Design notes

- ADR-0017 and G-0232 are the locked design; this milestone is their
  implementation, not a re-scoping of either.

## Surfaces touched

- `cmd/aiwf` (Cobra root: correlation id minting)
- `internal/render/render.go` (`Envelope.Metadata.correlation_id`)
- `internal/verb/*` (mutating-verb metadata)
- `internal/policies/envelope_structural_assertion.go` (new)
- `docs/adr/ADR-0017-...md`, `CLAUDE.md`

## Out of scope

- The correctness stress harness that will *consume* this correlation id at
  scale — the second epic named in
  `docs/initiatives/robustness-correctness-stress-testing.md`.

## Dependencies

- M-0237 — the logger and its `run_id` must exist.
- M-0238 — the ADR isn't ratifiable until the migration + chokepoint half of
  it is also true.

## References

- `docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`
- G-0232 — envelope enrichment: correlation_id wiring + mutating-verb metadata

---

## Work log

### AC-1 — An envelope's correlation_id matches the run_id in that invocation's log lines

`Execute` mints one id per invocation (`logger.NewRunID()`) and threads it
through `NewRootCmd` into every mutating verb's `NewCmd(correlationID
string)`, landing on a new `cliutil.OutputFormat.CorrelationID` field.
`outputformat.go`'s three envelope emitters (error, findings, success) all
inject `metadata.correlation_id` when set, via a single `metadata()` helper
— so the id is universal across every mutating verb's JSON envelope, not
just the ones that also log. `cancel` and `move` (the two verbs already
calling `logger.WithVerb`) now reuse the threaded id as `run_id` instead of
minting their own, with a same-value fallback for a direct-`Run` caller
that bypasses `NewCmd`/`Execute` (only reachable from a test, never from the
CLI surface, since `Execute` always mints a real id).

Two nested-subcommand constructors (`add`'s `newACCmd`, `worktree`'s
`newAddCmd`) needed the id threaded as an explicit parameter rather than
picking it up from an enclosing closure; a first mechanical pass missed
`worktree`'s, caught immediately by a compile error (`undefined:
correlationID`) rather than shipping silently.

Tested end-to-end (envelope `metadata.correlation_id` present, and for
cancel/move specifically equal to the log line's `run_id`) across 8 of the
15 correlation-id-bearing verb constructors, spanning every structurally
distinct code shape this plumbing touches: `DecorateAndFinish`-mediated
(`promote`, `cancel`, `move`, `rename`, `add`), `FinishVerb`-direct
(`authorize`, `acknowledge illegal`), and nested-subcommand threading
(`add ac`, `acknowledge illegal`). The remaining 7 (`retitle`, `setarea`,
`renamearea`, `reallocate`, `editbody`, `acknowledge mistag`, `worktree
add`) are structurally identical to a verified verb and were confirmed by
direct inspection to have no `out` reassignment between the
`CorrelationID` write and the emit call — the one failure mode compilation
doesn't already rule out.

`wf-vacuity` mutation probes: `metadata()` forced to always return `nil`
(caught — 3 tests failed), `cancel`'s reuse dropped in favor of always
minting fresh (caught), `move`'s reuse dropped the same way — **not
caught** by the test suite as it existed at that point, since no test
cross-checked `move`'s `run_id` against its envelope's `correlation_id`.
Fixed by adding `TestCorrelationID_MoveMatchesLogRunID`; re-ran the
mutation, now caught. All mutations reverted via captured pre-mutation
content, never `git stash`/`checkout`/`restore`. Commit `5eb2d1ef`.

### AC-2 — Mutating verbs report per-verb-appropriate metadata in their envelope

`verb.Result` gains an optional `Metadata map[string]any` field. `FinishVerb`
merges it with AC-1's `correlation_id` and (on a successful apply)
`commit_sha` into the envelope, via `OutputFormat.Metadata` — re-exported from
AC-1's private `metadata()` helper once a real second caller (a verb building
its own envelope outside `emitSuccess`) existed.

Populated per-verb metadata across all 15 correlation-id-bearing verb
constructors: `promote`/`cancel` (both plain-entity and composite-AC-id
shapes, the latter via the shared `finalizeACPlan` chokepoint so
`promoteAC`/`PromoteACPhase`/`cancelAC` all gained it in one edit), `rename`/
`retitle` (plain + composite), `setarea`, `renamearea`, `reallocate`,
`editbody` (explicit + bless paths), `authorize` (open + pause/resume),
`acknowledge illegal`/`mistag`, `add` (kind + `add ac`), `move`, and
`worktree add`. `archive`, `rewidth`, and `importcmd` had no `--format=json`
support at all before this AC — added from scratch (own `render.Envelope`
construction, since none of the three route through `FinishVerb`), each
reporting its own shape (`swept_count`, `renamed_count`,
`imported_count`/`entity_ids` respectively) plus `commit_sha`.

The branch-coverage audit surfaced two real gaps beyond the AC's own scope:
`move` already had AC-1's `correlation_id` wired but no per-verb metadata yet
(a straightforward miss, fixed here); and `worktree add` builds its own
`render.Envelope` directly rather than calling `emitSuccess`, so its
`OutputFormat.CorrelationID` field was being set correctly (AC-1) but never
actually read anywhere — the envelope's `metadata.correlation_id` was silently
absent until this pass exported `Metadata` and wired it in. Separately, the
audit surfaced that `contract bind`/`unbind`/`recipe install`/`recipe remove`
and `milestone depends-on` were missed entirely during AC-1: their `NewCmd`
constructors call `cliutil.AddFormatFlags` (confirmed via a full-repo grep,
not the narrower survey the AC-1 pass used) but had no `correlationID`
threading at all until this commit. All five now carry both `correlation_id`
and their own metadata (`entity_id`/`validator`, `depends_on`).

`wf-vacuity` mutation probes: `FinishVerb`'s success path forced to pass `nil`
metadata (caught), `withCommitSHA` mutated to drop the sha assignment
(caught), `promote`'s `from`/`to` values swapped (caught), `importedEntityIDs`
mutated to always return `nil` (caught). Manual branch-coverage walk found
`failArchive`/`failRewidth`/`failImport`'s JSON-mode error paths untested (one,
`failImport`, was 0% covered — no existing test reached any of its error
conditions in either format) and `failImport`'s text-mode path also untested;
fixed by adding dedicated JSON- and text-mode error-envelope tests for all
three. The same walk found `withCommitSHA`'s "both empty" early-return branch
unreachable at all four call sites (sha is always non-empty by the time any
of them run) — simplified away across all four copies rather than annotated,
relying on `encoding/json`'s `omitempty` treating a zero-length map identically
to nil (verified directly before relying on it).

Every verb in the AC-2 rollout has a dedicated test asserting its own
metadata field values specifically (`entity_id`/`from`/`to`, `old_slug`/
`new_slug`, `old_title`/`new_title`, `area`, `old_area`/`new_area`, `old_id`/
`new_id`, `agent`/`action`, `sha`, `kind`, `ac_ids`), not just `correlation_id`
— the class of gap `move` and `worktree add` both fell into above. Commit
`29f0c8ff`.

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
