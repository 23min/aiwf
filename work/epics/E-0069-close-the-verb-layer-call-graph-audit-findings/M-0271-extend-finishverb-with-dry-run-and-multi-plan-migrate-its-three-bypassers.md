---
id: M-0271
title: Extend FinishVerb with dry-run and multi-Plan; migrate its three bypassers
status: draft
parent: E-0069
tdd: required
acs:
    - id: AC-1
      title: FinishVerb gains dry-run and multi-Plan; existing envelopes byte-identical
      status: open
      tdd_phase: red
    - id: AC-2
      title: archive, rewidth, import dispatch via FinishVerb; triads deleted
      status: open
      tdd_phase: red
---
## Goal

Make `cliutil.FinishVerb` the single owner of the verb-outcome contract by
adding the dry-run and multi-`Plan` support its three bypassers need, then
migrate them and delete their hand-rolled envelope triads.

## Context

Finding F4 (verified): `archive`, `rewidth`, and `import` each reimplement the
`failX`/`emitXEnvelope`/`withCommitSHA` triad because `FinishVerb`
unconditionally applies a single `*verb.Plan` with no dry-run branch — a real
contract gap, not copy-paste laziness. A change to the outcome contract today
must be mirrored into three places or it silently drifts.

## Acceptance criteria

### AC-1 — FinishVerb gains dry-run and multi-Plan; existing envelopes byte-identical

### AC-2 — archive, rewidth, import dispatch via FinishVerb; triads deleted

## Constraints

- Envelope bytes for every existing `FinishVerb` consumer are pinned by test
  before the contract extends; the three migrated verbs' envelopes are pinned
  before their triads are deleted.
- Exit-code semantics (`0/1/2/3`) unchanged across the migration.
- The `dupl` tripwire stays green with no new baseline entries once the triads
  are gone.

## Design notes

- Extension shape (single seam growing two capabilities vs. a small outcome
  struct) is decided at implementation within the pinned-envelope constraint;
  the contract's *surface* is the deliverable, not a specific internal shape.

## Out of scope

- Any behavior change to what archive/rewidth/import actually do — this is
  dispatcher plumbing only.

## Dependencies

- None — parallel-safe with the sibling milestones; sequenced third for review
  bandwidth.

## References

- `docs/initiatives/verb-layer-cleanup.md` §F4; `internal/cli/cliutil/apply.go`.

---

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
