---
id: M-0257
title: Broaden the check-clean oracle across ten stress scenarios
status: in_progress
parent: E-0065
tdd: required
acs:
    - id: AC-1
      title: Each named scenario asserts its own check-clean baseline
      status: open
      tdd_phase: red
    - id: AC-2
      title: A shared baseline-classification helper backs all ten scenarios
      status: open
      tdd_phase: red
    - id: AC-3
      title: A synthetic regression test proves the broadened oracle catches it
      status: open
      tdd_phase: red
---

## Goal

Give each of the ten non-`verb-sequence` scenarios whose end state should be
a coherent, loadable tree its own curated "`aiwf check` must stay clean
beyond a baseline" oracle, alongside its existing scenario-specific
assertion — so a check-rule regression surfacing as an unplanned finding in
any of them is no longer invisible.

## Context

`verb-sequence` already classifies every `aiwf check` finding after each
walk step against a curated baseline (`classifyCheckFindings` +
`verbSequenceExpectedWarnings`, `internal/stresstest/verb_sequence.go:50-56,
494-503`): anything outside that baseline is a violation, regardless of
whether the scenario itself was probing for it. G-0410 found that the ten
scenarios named below don't share this pattern — each asserts on exactly one
pinned finding code (`parallel-branch-reallocate`, `cross-worktree-id-race`,
`force-override-durability`, `promote-on-wrong-branch-detection`), or a
`reflect.DeepEqual` between two specific check runs
(`reachability-isolation`), rather than "no unexpected finding at all." A
check-rule regression that shows up as a side effect of one of these
scenarios — not the specific finding it was built to probe for — would ship
silently even though `aiwf check` ran. This milestone generalizes
`verb-sequence`'s pattern onto the other ten.

## Acceptance criteria

### AC-1 — Each named scenario asserts its own check-clean baseline

For each of `parallel-branch-reallocate`, `cross-worktree-id-race`,
`force-override-durability`, `promote-on-wrong-branch-detection`,
`reachability-isolation`, `archive-during-active-scope`,
`cross-worktree-edit-body-race`, `concurrent-move`,
`concurrent-writer-at-scale`, and `concurrent-id-allocation`: after the
scenario's normal run, every `aiwf check` finding is classified against a
curated, scenario-specific expected-warnings baseline. Any finding outside
that baseline — including any error-severity finding — is reported as a
violation, in addition to (not instead of) the scenario's existing
single-finding-code assertion. Each baseline is derived empirically from
repeated `make stress` / `go test` runs of that scenario, not copied
wholesale from `verbSequenceExpectedWarnings` — different scenarios produce
different incidental noise (a fresh bare-origin clone carries different
provenance-audit warnings than a single disposable repo, for instance).

### AC-2 — A shared baseline-classification helper backs all ten scenarios

The per-scenario check-clean assertion in AC-1 is implemented via one shared
helper (parameterized by a per-scenario baseline map), not ten independent
copies of `classifyCheckFindings`'s loop. `verb_sequence.go`'s own
`classifyCheckFindings` either becomes a thin wrapper over the shared helper
or is replaced by a direct call to it, so the two don't drift apart.

### AC-3 — A synthetic regression test proves the broadened oracle catches it

At least one of the ten scenarios has a test that deliberately injects an
extraneous `aiwf check` finding (a finding code outside that scenario's
baseline and outside its existing pinned assertion) into a captured
`verbEnvelope`, and confirms: (a) the scenario's pre-existing single-code
assertion alone would not have flagged it, and (b) the AC-1 broadened oracle
does flag it as a violation. This is the scenario-catalog's own version of
G-0410's empirical validation methodology — proving the fix closes the
specific blind spot the gap described, not just that new code runs.

## Constraints

- Add the broadened oracle alongside each scenario's existing
  scenario-specific assertion; do not remove or weaken those — they pin a
  specific known-good outcome the broadened oracle isn't a substitute for.
- Never share one baseline map across scenarios — each is curated and
  justified independently, mirroring `verbSequenceExpectedWarnings`'s own
  per-entry doc comments explaining why that code is expected noise.

## Design notes

- Extract the `classifyCheckFindings`/`verbSequenceExpectedWarnings` shape
  from `verb_sequence.go` into a shared, baseline-parameterized helper (e.g.
  a `classifyAgainstBaseline(findings []verbEnvelopeFinding, baseline
  map[string]bool) []Violation` function) in a scenario-neutral file, rather
  than reinventing the same loop ten times.

## Surfaces touched

- `internal/stresstest/verb_sequence.go` — source pattern
  (`classifyCheckFindings`, `verbSequenceExpectedWarnings`) to generalize
- The ten named scenario files under `internal/stresstest/`:
  `parallel_branch_reallocate.go`, `cross_worktree_id_race.go`,
  `force_override_durability.go`, `promote_on_wrong_branch_detection.go`,
  `reachability_isolation.go`, `archive_during_active_scope.go`,
  `cross_worktree_edit_body_race.go`, `concurrent_move.go`,
  `concurrent_writer_at_scale.go`, `concurrent_id_allocation.go`

## Out of scope

- `disk-fault`, `lock-kill`, `mid-write-kill`, `head-drift` — their whole
  point is a torn-write or crash-recovery intermediate state `aiwf check`'s
  vocabulary doesn't model; per E-0065's own scope, broadening the
  check-clean oracle onto them would be a category error.
- The concurrent-race scenario itself (M-0258).
- G-0400's raw verb-coverage breadth — out of scope for E-0065 entirely.

## Dependencies

- None. This is the first milestone in E-0065.

## References

- G-0410 — stress catalog can't detect a missing domain-specific promote guard
- E-0065 — Harden the stress catalog's correctness oracle (parent epic)
