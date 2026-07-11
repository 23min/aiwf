---
id: M-0249
title: 'Scenario registry: wire cmd/stresstest run to the real catalog'
status: done
parent: E-0062
depends_on:
    - M-0244
tdd: required
acs:
    - id: AC-1
      title: cmd/stresstest run --scenario <name> runs exactly the named real scenario
      status: met
      tdd_phase: done
    - id: AC-2
      title: run --scenario all runs the whole catalog, one combined report
      status: met
      tdd_phase: done
    - id: AC-3
      title: cmd/stresstest list enumerates every registered scenario
      status: met
      tdd_phase: done
---
## Goal

Close G-0397: wire `cmd/stresstest run` to the real, 12-scenario catalog
built across M-0241 through M-0244, so the dedicated on-demand binary
E-0062's own Scope section describes can actually run them — today it
can only run the M-0240 placeholder.

## Context

`cmd/stresstest/run.go`'s scenario selection has been hardcoded to
`placeholderScenario` since M-0240, with that milestone's own comment
noting "no real catalog scenario ships until M-0241+." Each of M-0241
through M-0244 built and merged its own real scenarios, each with a
deterministic classify-function oracle, but none of the four milestones'
own scope included wiring its scenarios into the CLI entry point — the
gap went unnoticed until M-0244/AC-3's walk of E-0062's own success
criteria surfaced it directly.

## Acceptance criteria

### AC-1 — cmd/stresstest run --scenario <name> runs exactly the named real scenario

A name→constructor registry covering all 12 scenarios
(`ConcurrentIDAllocationScenario`, `CrossWorktreeIDRaceScenario`,
`ReachabilityIsolationScenario`, `LockKillScenario`,
`MidWriteKillScenario`, `DiskFaultScenario`,
`ParallelBranchReallocateScenario`, `CrossWorktreeEditBodyRaceScenario`,
`ArchiveDuringActiveScopeScenario`, `ForceOverrideDurabilityScenario`,
`HeadDriftScenario`, `ConcurrentWriterAtScaleScenario`) replaces the
placeholder as the selectable set. `--scenario <unknown-name>` refuses
with a clear error naming the valid set.

### AC-2 — run --scenario all runs the whole catalog, one combined report

The "run the whole catalog on demand" story E-0062's Scope section
describes — a human runs the harness and gets one report covering every
scenario, not one at a time.

Also closes a second, closely-related finding from the same AC-3 walk:
`RepeatEvent` (the raw-report event `RunRepeated` logs per attempt)
currently carries only `Attempt`/`Seed`/`Passed` — no preserved-repo
`Dir` and no `correlation_id` — and `cmd/stresstest run` never prints a
failing attempt's `Dir` either, even though `RunResult.Dir` is already
populated in memory. E-0062's own success criterion "a violation the
harness finds leaves enough behind (preserved repo state, a raw-report
event, and a `correlation_id` into E-0061's diagnostic log) to be
reproduced without re-running the whole campaign" needs the report
event itself to carry the failing `Dir`, and — since most scenarios'
shared `runAiwfJSON` helper doesn't set `AIWF_LOG`/`AIWF_LOG_FILE` for
the subprocesses it drives — a decision on whether/how the harness
should enable diagnostic logging for the scenarios it runs, so a
correlation_id trail actually exists to reproduce into.

### AC-3 — cmd/stresstest list enumerates every registered scenario

An operator can discover what's runnable without reading Go source —
closing the "on-demand" framing's own discoverability half.

## Constraints

- Each scenario's own constructor signature stays as-is (several take
  different args — `kind`, `n`, a seed) — the registry adapts to each,
  it does not force a uniform constructor shape onto the 12 existing
  scenarios.
- No change to any individual scenario's own Setup/Run/Verify/classify
  logic — this milestone is CLI wiring only.

## Design notes

- G-0269's HEAD-drift scenario (M-0243/AC-5) is deliberately expected-red
  until G-0269's own guard ships — the registry and `--scenario all` run
  must not treat that scenario's own violation as a harness-level failure
  distinct from the others; how the campaign-level pass/fail rolls up
  when one scenario is expected-red is this milestone's own design
  question to resolve, not assumed.

## Surfaces touched

- `cmd/stresstest/` (the new registry, `--scenario` flag, `list` command)

## Out of scope

- Any new scenario category — this milestone wires up the existing 12,
  it does not add a 13th.
- Making the harness a CI gate — still out of scope for the whole epic.

## Dependencies

- M-0244 — the last scenario-adding milestone (concurrent-writer-at-scale);
  by extension depends on M-0241, M-0242, M-0243 having already shipped
  their own scenarios.

## References

- G-0397 — cmd/stresstest run has no way to select any of the 12 real scenarios
- `docs/initiatives/robustness-correctness-stress-testing.md`

## Work log

### AC-1 — cmd/stresstest run --scenario <name> runs exactly the named real scenario

Added `cmd/stresstest/registry.go`: a name → constructor catalog adapting
each of the 12 real scenarios into `RunRepeated`'s `newScenario(seed)`
shape, none of the 12 scenarios' own constructor signatures touched.
`--scenario` is a required flag (no default); an unregistered name refuses
before any I/O via `unknownScenarioError`, naming the full valid set.
Removed the now-dead M-0240 `placeholderScenario` it replaces. Three
targeted mutation probes (`lookupScenario`'s name match, `needsLockHolder`'s
wiring into the lock-kill binary build, the unknown-scenario refusal check)
each confirmed a real bug goes red · commit 930d391e · tests all green,
95.7% cmd/stresstest coverage, 3 new branches confirmed via mutation probe

### AC-2 — run --scenario all runs the whole catalog, one combined report

`--scenario all` loops the full catalog into one shared raw-report file;
head-drift's own known-red status (G-0269) is labeled distinctly in the
summary rather than folded into the same pass/fail count as the other 11.
Resolved the design decision on diagnostic logging (discussed and
confirmed with the operator before implementation): `runRun` enables
`AIWF_LOG=debug`/`AIWF_LOG_FORMAT=json`/`AIWF_LOG_FILE=<outDir>/aiwf-diagnostic.log`
once, before running any scenario — every subprocess a scenario launches
inherits it via normal process-env inheritance, no scenario code touched.
`RepeatEvent` gained `Dir` (a failing attempt's preserved repo, already
computed but never logged) and `CorrelationIDs` (harvested via a new
`correlationIDsSince` resumable byte-offset cursor over the diagnostic
log, so consecutive attempts never attribute the same lines twice — a
single scalar id doesn't fit since one attempt can drive many aiwf
subprocesses, each with its own id). The env mutation makes 5 tests that
drive `runRun` end-to-end serial rather than parallel (documented in
`cmd/stresstest/setup_test.go`'s serial skip-list). Five targeted mutation
probes (scenario-set resolution, head-drift's label branch, the cursor's
unconsumed-partial-line handling, and the correlation-id dedup) each
confirmed a real bug goes red · commit 659c17c6 · tests all green, 96.3%
cmd/stresstest / 85.2% internal/stresstest coverage

### AC-3 — cmd/stresstest list enumerates every registered scenario

Added a `list` subcommand (`cmd/stresstest/list.go`) printing every
catalog name in registry order, wired into the root command tree. A
mutation probe against the enumeration loop's own bounds confirmed the
test catches a dropped entry · commit 24db580e · tests all green, 95.8%
cmd/stresstest coverage

## Decisions made during implementation

- D-0035 — diagnostic-log env passthrough plus a resumable byte-offset
  cursor for `RepeatEvent.CorrelationIDs`, instead of a scalar
  `CorrelationID` field (AC-2).

## Validation

- `go build ./...` — clean.
- `go test ./...` — every package green.
- `go test -race ./...` — clean (one incidental flake in an unrelated,
  untouched M-0244 scenario test surfaced once under full-repo `-race`
  load; four immediate re-runs, including the full `internal/stresstest`
  package under `-race`, all passed — not reproducible, not attributable
  to this milestone's diff).
- `make lint` — 0 issues.
- `make coverage-gate` (diff-scoped branch-coverage audit + firing-fixture
  meta-gate) — clean; every changed branch is tested or `//coverage:ignore`d
  with a stated, verified rationale.
- Per-AC mutation probes (registry name-match, `needsLockHolder` wiring,
  the unknown-scenario refusal, `--scenario all`'s scenario-set resolution
  and head-drift labeling, the diagnostic-log cursor's partial-line and
  dedup logic, the post-review malformed-line-skip fix, `list`'s
  enumeration bounds) — every mutation caught by an existing assertion,
  zero survivors.

## Deferrals

- G-0399 — `VerbSequenceScenario` (M-0241/AC-1's property-style FSM
  random-walk) isn't registered in the catalog `list`/`--scenario all`
  reach; surfaced during the wrap review as a scoping question for
  whoever next touches E-0062's on-demand catalog framing, not a defect
  against this milestone's own AC-1 (which enumerates exactly 12 names
  and explicitly excludes a 13th).

## Reviewer notes

Independent two-lens review (fresh-context, dispatched before wrap):

- **Code-quality**: APPROVE. Verified by measurement — the registry's
  type-assertion test (`fmt.Sprintf("%T", ...)`) actually catches a
  copy-paste constructor mismatch, all 12 constructors map correctly, the
  5 env-mutating tests that dropped `t.Parallel()` are exactly the 5 that
  reach the mutation (no test missed), the byte-offset cursor math is
  correct, every `//coverage:ignore` rationale is legitimate, and the "no
  scenario logic touched" constraint holds. Two non-blocking track-for-later
  notes: the un-registered `VerbSequenceScenario` (→ G-0399 above), and the
  `"all"`/`"lock-kill"` string-literal duplication (fixed in the
  corrective commit below).
- **Design-quality** (registry.go's catalog abstraction; repeat.go's
  `correlationIDsSince` log cursor): both units right-sized for their
  problem — the registry-of-closures beats a switch statement here because
  4 live consumers (`list`, `--scenario all`, the refusal message, shell
  completion) need to *enumerate* the set, not just dispatch on a name; the
  byte-offset cursor beats a line-count because the diagnostic log doesn't
  rotate mid-run (confirmed against `internal/logger/destination.go`) and
  a line-count would re-read the whole file every attempt. One real,
  actionable finding: `correlationIDsSince` hard-aborted the entire
  `--repeat` campaign on a malformed diagnostic-log line, including one a
  benign concurrent-write interleave under `O_APPEND`'s `PIPE_BUF`-sized
  write guarantee could produce — under exactly the concurrency this
  harness exists to create — and it ran before the current attempt's own
  replayable seed was logged. Fixed in a corrective commit
  (`fix(stresstest): tolerate a malformed diagnostic-log line...`),
  recorded as D-0035, confirmed via new tests plus a mutation probe, and
  the fix's own coverage/lint/test gates re-verified clean.

No `TODO`/`FIXME` left behind; no debug output or commented-out code in
the diff. `wf-doc-lint` (scoped to this milestone's change-set): 0
findings — no `docs/` file references any symbol this milestone touched
or removed.
