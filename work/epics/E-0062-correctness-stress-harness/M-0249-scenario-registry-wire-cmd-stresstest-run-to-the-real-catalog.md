---
id: M-0249
title: 'Scenario registry: wire cmd/stresstest run to the real catalog'
status: in_progress
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
      status: open
      tdd_phase: green
    - id: AC-3
      title: cmd/stresstest list enumerates every registered scenario
      status: open
      tdd_phase: red
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
