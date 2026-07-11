---
id: M-0241
title: Property sequences and multi-worktree contention scenarios
status: done
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: Generated verb sequences are checked for FSM legality each step
      status: met
      tdd_phase: done
    - id: AC-2
      title: Real subprocesses racing repolock never produce a duplicate id
      status: met
      tdd_phase: done
    - id: AC-3
      title: A cross-worktree id race is always caught and resolved by reallocate
      status: met
      tdd_phase: done
    - id: AC-4
      title: repolock's per-worktree lockfile scoping is confirmed intentional
      status: met
      tdd_phase: done
    - id: AC-5
      title: A sibling worktree's commit is confirmed unreachable from another's check
      status: met
      tdd_phase: done
---

## Goal

Implement scenario tiers 1 (single-process verb-sequence properties) and 2
(multi-process/multi-worktree contention) against the M-0240 skeleton, and
empirically confirm two open architectural questions rather than assume
them.

## Context

M-0240 shipped the driver, scenario interface, and streaming report with
only a placeholder scenario. This milestone is the first to exercise real
aiwf behavior through it — both the "loose/statistical race" mechanism
(start actors close together, repeat) that the initiative doc's
Orchestration mechanics section describes as the default for this tier.

## Acceptance criteria

### AC-1 — Generated verb sequences are checked for FSM legality each step

Extends the existing `internal/entity/transition_property_test.go` pattern:
random (legal and illegal) verb sequences run against one real temp repo
via the real binary; after every step, `aiwf check` is clean or
expected-findings-only, and exactly one commit landed per mutation.

### AC-2 — Real subprocesses racing repolock never produce a duplicate id

N real `aiwf` subprocesses, started close together against one working
copy, contend for an id allocation. Repeated per M-0240's `--repeat`
mechanism. No run ever produces two entities with the same id.

### AC-3 — A cross-worktree id race is always caught and resolved by reallocate

The same contention, but across sibling worktrees of one repo rather than
one working copy. A duplicate id here is an *accepted* outcome (per this
repo's own eventual-consistency design for id allocation) — the assertion
is that `aiwf check` always surfaces it as a finding, and `aiwf reallocate`
always resolves it cleanly, never that the duplicate never happens.

### AC-4 — repolock's per-worktree lockfile scoping is confirmed intentional

`repolock.lockfilePath` resolves differently for a linked worktree (whose
`.git` is a file, not a directory) than for the main checkout — meaning a
worktree does not share the main checkout's lock. This scenario exercises
that directly and confirms the consequence is exactly AC-3's
accepted-eventual duplicate-id-then-reallocate story, not some other,
unaccounted-for failure mode.

### AC-5 — A sibling worktree's commit is confirmed unreachable from another's check

Confirms the documented architectural fact surfaced while scoping this
epic (`branch_scenarios_ac2_test.go`'s comment on `aiwf check`'s provenance
walk being `git log HEAD`, not `--all`): a commit made in a sibling
worktree is invisible to a check run in another worktree. This scenario
checks whether that invisibility has any *other* consequence beyond the
one rule (isolation-escape) that already documents it — if a different
rule implicitly assumes broader reachability, this is where that surfaces.

## Constraints

- Tier 2 scenarios use the loose/statistical race mechanism only — start
  actors close together, repeat; no directed/failpoint mechanism in this
  milestone (that's reserved, per the epic's constraints, for a case tier 3
  or 4 prove genuinely needs it).
- Every scenario's pass/fail oracle is the invariant itself (id uniqueness,
  FSM legality, reachability), never a human reading harness output.

## Design notes

- AC-3 and AC-4 are deliberately framed as *confirming* an accepted design,
  not fixing a bug — don't treat a duplicate id under AC-3 as a failure;
  treat an *unresolved* or *undetected* duplicate id as one.

## Surfaces touched

- `internal/stresstest/` (new scenario files)
- `internal/repolock/` (read-only — no production changes expected from
  this milestone; AC-4/AC-5 are read-only confirmations)

## Out of scope

- Fault injection (kill -9, disk-full) — M-0242.
- The named G-0212/G-0269 scenarios — M-0243.
- Any change to `repolock` or the provenance-walk's reachability behavior —
  this milestone confirms current behavior, it doesn't change it. A change,
  if one turns out to be warranted, is a new gap, not this milestone's job.

## Dependencies

- M-0240 — the harness skeleton.

## References

- `docs/initiatives/robustness-correctness-stress-testing.md`
- `internal/cli/integration/branch_scenarios_ac2_test.go` (the
  `git log HEAD`-not-`--all` reachability documentation)

---

## Work log

### AC-1 — Generated verb sequences are checked for FSM legality each step

`VerbSequenceScenario` walks random legal/illegal `aiwf promote` attempts against one entity of every kind, via the real compiled binary, in one disposable repo — extending `internal/entity/transition_property_test.go`'s FSM-property pattern to the real binary. Discovered and handled a real nuance: an FSM-legal transition can still be refused by an orthogonal business rule (gap's `addressed`-needs-`--by` resolver gate) distinct from FSM illegality; the classifier treats that as a legitimate refusal, not a violation. · commit 0320f740 · tests 25/25

### AC-2 — Real subprocesses racing repolock never produce a duplicate id

`ConcurrentIDAllocationScenario` launches n real `aiwf add` subprocesses against one working copy, started close together via goroutine/OS process scheduling (no artificial delay), repeated via M-0240's `RunRepeated`. Confirms repolock serializes every attempt to a distinct id within its 2-second timeout. Along the way, `PolicyNoRetryLoopsOnGitErrors` flagged the fan-out loop as a false positive (its heuristic matches any `for` body containing `exec.Command`, not specifically retried git calls) — fixed by extracting the per-actor subprocess launch into its own method. · commit d8f3d6b7 (+ 4d27cc40 coverage:ignore placement fix) · tests 33/33

### AC-3 — A cross-worktree id race is always caught and resolved by reallocate

`CrossWorktreeIDRaceScenario` races real `aiwf add` subprocesses across two sibling worktrees (repolock has no cross-worktree serialization, per AC-4) and confirms: when the race produces a genuine duplicate id — an accepted outcome, not a prevented one — `aiwf check` always surfaces it as `ids-unique` and `aiwf reallocate` always resolves it cleanly. Repeated via `RunRepeated` so at least one attempt hits the real race window. Caught a real bug in the harness itself: `findEntityFile` returned an absolute path, but `aiwf reallocate` resolves its argument against the repo root, so every real collision was failing to resolve until fixed. `runAiwfJSON` refactored into a package-level function to drive multiple worktree directories. · commit 2b289043 · tests 46/46

### AC-4 — repolock's per-worktree lockfile scoping is confirmed intentional

White-box tests directly in `internal/repolock` (read-only, no production changes) confirm the mechanism behind AC-3's race: a linked worktree's `.git` is a regular file, not a directory, so `lockfilePath`'s `IsDir()` check falls through to a per-worktree `.aiwf.lock` fallback that never touches the main checkout's `.git/aiwf.lock`. Behavioral confirmation: holding the main checkout's lock does not block a concurrent `Acquire` in the linked worktree, ruling out any other unaccounted-for failure mode. Along the way, `PolicyGitTestEnvHardened` correctly required `testsupport.HardenGitTestEnv()` in `repolock`'s `TestMain` now that a test in the package shells out to real git. · commit 757af3fd · tests 3 new (+ existing repolock suite)

### AC-5 — A sibling worktree's commit is confirmed unreachable from another's check

`ReachabilityIsolationScenario` confirms a commit in worktree B is invisible to `aiwf check` in worktree A until merged, and — beyond the fact itself — that this invisibility has zero other observable consequence: check's findings and entity count are byte-identical before and after the sibling's invisible commit, not just silent on isolation-escape specifically. Confirms the loop closes correctly once merged. Discovered a real, separate bug: `aiwf show <missing-id> --format=json` doesn't honor `--format=json` on its not-found path (empty stdout, plain-text stderr) — filed as G-0389 rather than fixed here (architecturally distinct from this AC's reachability claim); the scenario classifies `show` by exit status instead. · commit 09a6e02e (+ b5ac4984 branch-coverage fix) · tests 41/41

## Decisions made during implementation

- (none)

## Validation

Full validation after every AC: `go build ./...`, `go vet ./...`, `make lint` (0 issues), `go test -race -parallel 8 -count=1 ./...` (all green), `aiwf check` (clean, only the pre-existing `provenance-untrailered-scope-undefined` warning), `make coverage-gate` (clean after each commit). Vacuity mutation probes run against every new pure decision function across all 5 ACs. The independent review (see Reviewer notes) found one gap the self-probe missed — `classifyReachabilityIsolation`'s entity-count comparison was untested in isolation — fixed with a dedicated table row; the reviewer's own mutation is now caught. All mutated files restored byte-identical after every probe.

## Deferrals

- G-0389 — `aiwf show --format=json` doesn't honor the flag on its not-found path (discovered during AC-5; architecturally distinct from this milestone's reachability-isolation claim).

## Reviewer notes

Independent two-lens review (fresh-context, no authorship attachment) run before wrap:

- **Code-quality**: REQUEST-CHANGES, one blocking item — `classifyReachabilityIsolation`'s entity-count comparison (AC-5) had no table row isolating it from the findings comparison, so a mutation dropping just that half of the check survived. Fixed with a dedicated row (commit 8f3fd883); the reviewer's exact mutation is now caught. Everything else across all 5 ACs verified solid: AC-to-test traceability, `//coverage:ignore` honesty, the `runAiwfJSON` method→function refactor's call sites, and all work-log commit SHAs checked against `git log`.
- **Design-quality**: the four-scenario Setup/Run/classify pattern itself is fine (KISS — forcing a shared template over genuinely different invariants would be premature abstraction). Two real, small issues found and fixed: `CrossWorktreeIDRaceScenario.Setup` and `ReachabilityIsolationScenario.Setup` were byte-identical (extracted `newSiblingWorktreesFixture`, commit 82713ae1); the shared JSON-envelope machinery was stranded in the AC-1-named `verb_sequence.go` despite being used by all four scenarios (relocated to `verbenvelope.go`, commit 5701b3c4). Both are pure structural moves, re-verified with the full race suite and `make coverage-gate` after landing.
- Non-blocking tracked item: `cmd/stresstest/run.go`'s `placeholderScenario` comment still reads "no real catalog scenario ships until M-0241+" — this milestone's 5 scenarios run via the `RunScenario`/`RunRepeated` test harness (the sanctioned oracle), not wired into the CLI catalog; "Surfaces touched" deliberately excludes `cmd/stresstest`, so this is intentional, not a gap.
