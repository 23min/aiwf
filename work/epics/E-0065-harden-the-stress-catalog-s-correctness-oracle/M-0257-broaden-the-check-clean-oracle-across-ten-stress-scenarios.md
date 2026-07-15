---
id: M-0257
title: Broaden the check-clean oracle across ten stress scenarios
status: in_progress
parent: E-0065
tdd: required
acs:
    - id: AC-1
      title: Each named scenario asserts its own check-clean baseline
      status: met
      tdd_phase: done
    - id: AC-2
      title: A shared baseline-classification helper backs all ten scenarios
      status: met
      tdd_phase: done
    - id: AC-3
      title: A synthetic regression test proves the broadened oracle catches it
      status: met
      tdd_phase: done
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

---

## Work log

### AC-2 — Shared baseline-classification helper backs all ten scenarios

`classifyAgainstBaseline` (`internal/stresstest/checkclean.go`) generalizes
`verb_sequence.go`'s `classifyCheckFindings`/`verbSequenceExpectedWarnings`
loop into one baseline-parameterized helper; `classifyCheckFindings` is now a
thin wrapper over it, so the two can't drift apart. · commit `6bcc25c9` ·
tests 6/6

### AC-1 — Each named scenario asserts its own check-clean baseline

All ten scenarios (`parallel-branch-reallocate`, `cross-worktree-id-race`,
`force-override-durability`, `promote-on-wrong-branch-detection`,
`reachability-isolation`, `archive-during-active-scope`,
`cross-worktree-edit-body-race`, `concurrent-move`,
`concurrent-writer-at-scale`, `concurrent-id-allocation`) now classify their
post-run `aiwf check` findings against their own curated
`classifyAgainstBaseline` map, alongside each scenario's existing
single-finding-code assertion. Five scenarios
(`archive-during-active-scope`, `cross-worktree-edit-body-race`,
`concurrent-move`, `concurrent-writer-at-scale`,
`concurrent-id-allocation`) previously never ran `aiwf check` at all — a new
call was added to each. `promote-on-wrong-branch-detection`'s `Run` was
restructured so the checkout-back-and-check sequence runs regardless of
whether G-0269's branch guard blocks the promote, since the guard blocks it
100% of the time today (the old code path never reached `aiwf check` in
practice). Each baseline was derived empirically via repeated real-binary
runs, not copied from `verbSequenceExpectedWarnings`. · commit `65c12894` ·
tests 10/10 new baseline-pin tests (plus every scenario's existing
real-binary suite, unchanged assertions, still green)

### AC-3 — A synthetic regression test proves the broadened oracle catches it

`TestParallelBranchReallocateScenario_BroadenedOracleCatchesAnInjectedRegression`
(`parallel_branch_reallocate_test.go`) drives the real scenario to a clean,
check-clean completion, captures the real post-reallocate `aiwf check`
envelope, and injects one extraneous finding code outside both the
scenario's baseline and its existing ids-unique-only assertion. Confirms
`classifyParallelBranchReallocate` alone (the scenario's pre-existing narrow
assertion) does not flag it, while `classifyAgainstBaseline` (AC-1's
broadened oracle) does — closing the exact blind spot G-0410 named, via the
same empirical-injection methodology G-0410 itself used. · commit
`222aacc5` · tests 4/4 (whole file, including the new test)

## Decisions made during implementation

- None rising to a durable, cross-milestone decision. Several narrow
  implementation judgment calls arose (which checkpoint in
  `force-override-durability` gets the broadened assertion; restructuring
  `promote-on-wrong-branch-detection`'s `Run` so the check-clean assertion
  runs unconditionally; applying the broadened assertion to both branches of
  `cross-worktree-id-race`'s `reconcile`) — each is fully reasoned inline in
  its own code comment and summarized in the Work log above and in Reviewer
  notes below. None is a project-wide or cross-milestone concern warranting
  a separate ADR/D-NNN.

## Validation

- `go build ./...`, `go vet ./...`, `golangci-lint run ./...` (0 issues) —
  clean throughout implementation and at wrap.
- `go test -race -parallel 8 ./...` — full module, multiple clean reruns
  during implementation and one final clean run at wrap preflight (`EXIT=0`,
  0 `FAIL` lines).
- `make check-fast` — clean at wrap preflight (`EXIT=0`).
- Empirical stress-catalog runs against freshly-built `aiwf` binaries: all
  ten scenarios passed 100/100 combined across multiple `--repeat` passes
  during implementation, plus a further 6/6 each from both independent
  wrap-time reviewers (60/60 additional).
- Two independent-reviewer live vacuity checks (implementer + both
  reviewers) confirmed the broadened assertions are load-bearing through the
  full stack — emptying a scenario's baseline map and rerunning the real
  scenario reliably fails the run, not just a unit test.
- `aiwf check` — 0 error-severity findings on M-0257; the 5 warnings present
  are pre-existing and unrelated (archive-sweep advisories, provenance-scope
  undefined on this disposable dev repo).

## Deferrals

- G-0414 — stale test naming in
  `promote_on_wrong_branch_detection_test.go` (its name/doc comment/failure
  message describe live wrong-branch detection; today G-0269's guard blocks
  the activation 100% of the time, so the test passes on prevention plus
  the broadened check-clean baseline, not on detection firing). Surfaced by
  one of the two independent wrap-time reviewers; the file is outside this
  milestone's diff, so realigning it is follow-up work, not a wrap blocker.

## Reviewer notes

- Two independent, fresh-context reviewers ran adversarial code-quality
  passes (`wf-review-code`) over disjoint slices of the diff — one over
  `checkclean.go`, `verb_sequence.go`'s refactor, AC-3's test, and the five
  scenarios that already called `aiwf check` before this milestone; the
  other over the five scenarios that got their first-ever `aiwf check` call.
  Both verdicts: **APPROVE**. Each independently re-ran the full empirical
  stress suite, mutated `classifyAgainstBaseline` to confirm it isn't
  vacuously tested, and (the second reviewer) performed three additional
  live end-to-end vacuity checks on top of the ones already run during
  implementation.
- `wf-rethink` (design-quality lens) was assessed and skipped: this
  milestone's only new abstraction, `classifyAgainstBaseline`, is a 36-line
  package-private pure-function extraction of pre-existing logic — not a new
  module boundary, core abstraction, or data model, so it doesn't meet
  `wf-rethink`'s non-trivial-design trigger.
- One reviewer flagged `concurrent_move.go`'s `checkErr` naming as an
  inaccurate rationale comment (claimed to avoid a shadow the reviewer
  believed wasn't enforced). Independently re-verified at wrap: this repo's
  `.golangci.yml` runs `govet` with `enable-all: true`, so the shadow check
  *is* active, and renaming `checkErr` to `err` does trip it (confirmed by
  reproducing the lint failure). The reviewer's diagnosis was wrong but the
  underlying naming was already correct — only the comment's wording needed
  correcting, landed as a small corrective commit.
- Process note, not re-litigated: AC-1's `met` promote (`1748e251`) landed
  with its Work log body-prose edit already sitting in the working tree, so
  the commit swept in both the frontmatter transition and the body edit
  instead of landing as two separate commits (the pattern AC-2 and AC-3 both
  follow cleanly). Content is correct and was reviewed before the commit
  landed; left as-is rather than rewriting history for a trailer-precision
  nit. AC-3 repeated the setup but was caught and corrected before landing.
