---
id: M-0271
title: Extend FinishVerb with dry-run and multi-Plan; migrate its three bypassers
status: done
parent: E-0069
tdd: required
acs:
    - id: AC-1
      title: FinishVerb gains dry-run and multi-Plan; existing envelopes byte-identical
      status: met
      tdd_phase: done
    - id: AC-2
      title: archive, rewidth, import dispatch via FinishVerb; triads deleted
      status: met
      tdd_phase: done
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

`cliutil.FinishVerb` gains a dry-run branch (prints/serializes the
planned outcome without calling `verb.Apply`) and accepts more than
one `*verb.Plan` per invocation (applying each in order, tracking the
last commit sha). Every existing `FinishVerb` consumer's JSON and text
envelope output is pinned byte-for-byte by test before and after the
extension — the contract grows two capabilities without moving any
existing caller's bytes.

### AC-2 — archive, rewidth, import dispatch via FinishVerb; triads deleted

`archive`, `rewidth`, and `import` route their outcome handling
through the extended `FinishVerb` instead of their own hand-rolled
`failX`/`emitXEnvelope`/`withCommitSHA` triads. Each of the three
verbs' envelope output (dry-run and applied, text and JSON) is pinned
by test before its triad is deleted, and the triad functions are
removed from all three packages.

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

### AC-1 — FinishVerb gains dry-run and multi-Plan; existing envelopes byte-identical

Added `Outcome`/`FinishVerbOutcome` to `internal/cli/cliutil/apply.go`; `FinishVerb` is now a thin single-Plan adapter over it · commit 9b4b9b07631f50e3498b8fa90494cd4e803f6bf5 · tests 14/14 new, full `internal/cli/...` suite green under `-race`.

### AC-2 — archive, rewidth, import dispatch via FinishVerb; triads deleted

Migrated `archive`, `rewidth`, and `import` onto `FinishVerbOutcome`; deleted `failX`/`emitXEnvelope`/`withCommitSHA` from all three packages (9 functions total) · commit b4b1045069b638e33a4bc5a6379a5c3986cd5165 · tests: 11 new byte-for-byte envelope-pinning tests plus the full pre-existing `internal/cli/{archive,rewidth,importcmd,integration}` suites, all green under `-race`.

## Decisions made during implementation

- D-0044 (accepted) — add `cliutil.ErrInternal` to `FinishVerbOutcome`'s err contract so an early domain-call failure can report `ExitInternal` (as `import`'s tested `LoadTreeWithTrunk` failure requires) without reintroducing a per-verb envelope helper.

## Validation

- `go build ./...` — clean.
- `go test -race -parallel 8 ./...` — full suite green.
- `make lint` (worktree-scoped `golangci-lint`, includes `dupl`) — 0 issues.
- `make coverage-gate` — diff-scoped branch-coverage audit and firing-fixture presence gate both green.
- Manual mutation probes (`wf-vacuity`-style) against `FinishVerbOutcome` and each migrated verb's dispatch wiring — every probed mutation caught by the pinning-test suite; all reverted byte-identical.

## Deferrals

- (none)

## Reviewer notes

- Migrating onto `FinishVerbOutcome` widens two narrow, previously-inconsistent behaviors for `archive`/`rewidth`/`import`: `--trace` (already accepted on all three verbs' CLI surface, previously a silent no-op) now actually emits a `phase.apply` diagnostic event on apply, matching every other `FinishVerb` consumer; and JSON-mode early usage errors on these three verbs now carry `metadata.correlation_id` when one is set, where the old hand-rolled envelopes silently dropped it. Neither changes stdout/stderr envelope bytes on the success/error paths the milestone's byte-identical constraint covers — both are pure additions to a previously-inert or absent side channel. Flagging for awareness, not as a defect.
- **Independent two-lens review before wrap:** dispatched a fresh-context code-quality review over the full milestone diff and a design-quality review over the new `Outcome`/`FinishVerbOutcome` abstraction. Code-quality returned one blocking finding — `make coverage-gate` was genuinely red (`internal/cli/importcmd/importcmd.go`'s non-human-actor principal-trailer-stamping branch had no test reaching it) — fixed with `TestImport_NonHumanActorWithPrincipal_StampsTrailer`, mirroring the existing archive/rewidth coverage; re-verified `make coverage-gate` green after the fix. Design-quality returned no blocking findings ("sound and ship-able") and two non-blocking suggestions, both applied: `cliutil.ErrInternal` now wraps an `error` instead of a `string` (preserves `errors.Is`/`As` traversal into the original cause, pinned by a new test); `archive`/`rewidth`'s dry-run subject line is computed once and threaded into `printXDryRun` as a parameter instead of being independently re-derived inside it. Corrective commit: `92c50d95`.
