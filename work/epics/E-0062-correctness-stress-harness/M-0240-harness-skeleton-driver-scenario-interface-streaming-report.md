---
id: M-0240
title: 'Harness skeleton: driver, scenario interface, streaming report'
status: in_progress
parent: E-0062
depends_on:
    - M-0239
tdd: required
acs:
    - id: AC-1
      title: A stress run builds the aiwf binary under test once, never trusting PATH
      status: met
      tdd_phase: done
    - id: AC-2
      title: Each raw-report event appends via a single Write call
      status: open
      tdd_phase: red
    - id: AC-3
      title: A run killed mid-scenario still composes without failing on a truncated line
      status: open
      tdd_phase: red
    - id: AC-4
      title: A scenario's repo is cleaned up on pass and preserved on fail
      status: open
      tdd_phase: red
    - id: AC-5
      title: A --repeat N flag reruns a scenario N times with a logged seed per attempt
      status: open
      tdd_phase: red
---

## Goal

Build the harness's own scaffolding — driver loop, scenario interface,
streaming raw-report writer, abort-safe compose step — so every later
milestone in this epic has something real to plug scenarios into.

## Context

E-0061 is done: `internal/logger` exists, `correlation_id` is wired,
ADR-0017 is ratified. This milestone makes the one concrete decision the
initiative doc left open for the start of implementation: `go test
-tags=stress -json` (reusing Go's own streaming test-event format and
`t.Run` reporting) versus a fully bespoke `cmd/stresstest` driver. No
scenarios are implemented here — just the scaffolding they'll all share.

## Acceptance criteria

### AC-1 — A stress run builds the aiwf binary under test once, never trusting PATH

Matches this repo's own worktree-binary discipline (`make diag-aiwf`'s
precedent): the harness builds a fresh, worktree-scoped `aiwf` binary at the
start of a run and uses that path throughout, never whatever happens to be
on `PATH`.

### AC-2 — Each raw-report event appends via a single Write call

The raw-report writer opens its output file with `O_APPEND` and emits each
event as exactly one `Write()` call — reusing `internal/logger`'s
concurrent-append discipline (ADR-0017 Decision #5) rather than inventing a
second streaming primitive.

### AC-3 — A run killed mid-scenario still composes without failing on a truncated line

A `kill -9` (or `SIGINT`/`SIGTERM`) mid-run leaves the raw-report file
however far it got. The separate compose step reads it and renders a report
covering everything that completed; a malformed trailing line (a record cut
off mid-write) is dropped silently, never treated as a whole-report failure
— the same "errors are findings, not parse failures" posture `aiwf check`
already holds toward the entity tree.

### AC-4 — A scenario's repo is cleaned up on pass and preserved on fail

A scenario that passes removes its own temp repo(s)/worktree(s). A scenario
that fails leaves them on disk — the on-disk state at failure time is RCA
material a human might want to open directly.

### AC-5 — A --repeat N flag reruns a scenario N times with a logged seed per attempt

Each attempt logs the random seed it used (actor-start jitter, any
randomized delay) into its raw-report event, so a violation found on a given
attempt is replayable by rerunning with that seed.

## Constraints

- Harness code lives entirely under `internal/stresstest/` (plus a thin
  `cmd/stresstest/` entry point if the bespoke-driver fork is chosen) —
  never scattered into production packages, never installed alongside
  `cmd/aiwf`.
- No scenario logic in this milestone — AC-1 through AC-5 are proven with a
  trivial placeholder scenario (e.g. "runs `aiwf check` on an empty repo and
  always passes"), not a real one from the catalog.
- The raw-report writer's `O_APPEND`/one-`Write()`-per-record discipline is
  non-negotiable, matching E-0061 (M-0237)'s constraint on the same
  property.

## Design notes

- `docs/initiatives/robustness-correctness-stress-testing.md`'s
  "Orchestration mechanics" section is the locked design for the driver
  loop, the scenario shape, and the loose-vs-directed race distinction —
  this milestone implements the skeleton half of it (driver, interface,
  report), not the scenario-authoring half (later milestones).
- The `go test -tags=stress` vs. bespoke-driver decision is made at the
  start of this milestone, informed by whichever gives more direct control
  over process orchestration (subprocess spawning, signal delivery) without
  fighting the chosen mechanism.

## Surfaces touched

- `internal/stresstest/` (new package tree)
- `cmd/stresstest/` (new, if the bespoke-driver fork is chosen)
- `Makefile` (`make stress` target)

## Out of scope

- Any real scenario from tiers 1–5 — M-0241 through M-0244.
- Report render format polish beyond "a compose step exists and doesn't
  choke on truncation" — refined as real scenarios land.
- A `workflow_dispatch` CI entry point — open question, not this milestone.

## Dependencies

- M-0239 — E-0061's capstone; the logger and correlation id must be shipped
  and merged before this milestone starts.

## References

- `docs/initiatives/robustness-correctness-stress-testing.md`
- `docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`

---

## Work log

### AC-1 — Build the aiwf binary under test once, never trusting PATH

`internal/stresstest.BuildBinary` compiles `./cmd/aiwf` into an absolute
`outDir/aiwf` path; a decoy-on-PATH test pins that callers invoking the
returned path never fall back to a bare `aiwf` PATH lookup · commit
d51fa34c · tests 3/3

## Decisions made during implementation

- D-0033 — driver mechanism is a bespoke `cmd/stresstest` binary, not
  `go test -tags=stress -json`

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
