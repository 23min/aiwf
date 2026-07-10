---
id: D-0035
title: Diagnostic-log passthrough plus a resumable cursor, not a scalar correlation id
status: proposed
relates_to:
    - M-0249
---
# D-0035 — Diagnostic-log passthrough plus a resumable cursor, not a scalar correlation id

> **Date:** 2026-07-10 · **Decided by:** human/peter

## Question

M-0249/AC-2's own acceptance text flags that `RepeatEvent` (the raw-report
event `RunRepeated` logs per `--repeat` attempt) carries no `correlation_id`,
and that the scenarios' shared `runAiwfJSON` helper never sets
`AIWF_LOG`/`AIWF_LOG_FILE` for the subprocesses it drives — so even if a
correlation id existed, no diagnostic-log file exists for it to point into.
What wasn't obvious: a single scenario attempt can spawn many `aiwf`
subprocess calls (e.g. `ConcurrentIDAllocationScenario` launches `n`
concurrent `aiwf add` calls), each minting its own distinct correlation id
via `logger.NewRunID()` — so a *scalar* `CorrelationID` field on
`RepeatEvent` has no single coherent value to hold for those scenarios. The
milestone's own Constraints section additionally forbids touching any of
the 12 scenarios' own `Setup`/`Run`/`Verify`/classify logic, so the fix
could not have a scenario report its own ids back to the harness through a
new interface method.

## Decision

`cmd/stresstest/run.go`'s `runRun` unconditionally enables diagnostic
logging for the whole run — `AIWF_LOG=debug`, `AIWF_LOG_FORMAT=json`,
`AIWF_LOG_FILE=<outDir>/aiwf-diagnostic.log` — once, before running any
scenario. Every subprocess a scenario launches inherits this via normal
process-env inheritance (`exec.Cmd` reads `os.Environ()` when `Env` is
unset), so no scenario code is touched. `RepeatEvent` gains `Dir` (a
failing attempt's preserved repo, already computed but never logged) and
`CorrelationIDs []string` — populated by a new `correlationIDsSince`
function in `internal/stresstest/repeat.go`: a resumable byte-offset
cursor over the diagnostic-log file. `RunRepeated` calls it once per
attempt, threading the returned offset into the next call, so every line
written since the last attempt is attributed to the attempt that just ran
and never double-counted.

## Reasoning

**Why not drop correlation ids from `RepeatEvent` entirely** (the initial
proposal): it would have left half of E-0062's own success criterion
unmet — "a violation the harness finds leaves enough behind (preserved
repo state, a raw-report event, and a `correlation_id` into E-0061's
diagnostic log) to be reproduced without re-running the whole campaign."
Dropping the field just because a single scalar didn't fit was solving the
wrong problem instead of finding the right shape for the real one.

**Why not a scalar field anyway** (e.g. the first or last id seen): would
misrepresent every concurrent-actor scenario by discarding real ids a
human debugging a failure would need.

**Why not have each scenario report its own ids back through a new
`Scenario` interface method**: forbidden outright by the milestone's own
constraint against touching any scenario's `Setup`/`Run`/`Verify` — and
even without that constraint, it would mean re-plumbing all 12 scenarios
for a harness-level concern.

**Why a log-cursor scan, not scenario-side reporting**: every subprocess's
correlation id is *already* the same value as the diagnostic log's own
`run_id` field for every line it writes (`internal/cli/root.go`'s own
comment: "reused as the diagnostic logger's run_id... cross-referenceable
by a single grep"). Since `RunRepeated`'s attempts run strictly
sequentially (never concurrently with each other), a byte-offset cursor
into one shared, append-only log file gives exact per-attempt attribution
with zero scenario-code changes — the harness already has everything it
needs once logging is turned on.

**Why a byte offset and not a line count**: the diagnostic log can contain
a genuinely partial trailing line (a subprocess still mid-write) at the
moment `correlationIDsSince` reads it. A byte offset lets the cursor stop
precisely before that incomplete line and pick it up whole on the next
call; a line count would have to guess whether the last "line" it counted
was actually complete.

## Consequences

The `os.Setenv` call is a process-wide mutation, so every `cmd/stresstest`
test that drives `runRun` end-to-end past scenario/out-dir resolution and
the binary build must run serially rather than under `t.Parallel()` (a
race between two such tests could send an attempt's subprocess output to
the wrong test's diagnostic-log path). Documented in
`cmd/stresstest/setup_test.go`'s serial skip-list; five such tests are
affected. This is a one-time, bounded cost — the affected test count only
grows if a future AC adds another `runRun`-driving end-to-end test, which
would need the same treatment.
