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
      status: open
      tdd_phase: done
    - id: AC-2
      title: Mutating verbs report per-verb-appropriate metadata in their envelope
      status: open
      tdd_phase: red
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

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
