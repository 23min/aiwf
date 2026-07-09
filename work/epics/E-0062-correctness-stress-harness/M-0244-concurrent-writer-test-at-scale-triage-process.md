---
id: M-0244
title: Concurrent-writer test at scale; triage process
status: in_progress
parent: E-0062
depends_on:
    - M-0241
    - M-0242
    - M-0243
tdd: required
acs:
    - id: AC-1
      title: Concurrent subprocesses sharing one log file never tear or interleave a line
      status: met
      tdd_phase: done
    - id: AC-2
      title: A documented triage procedure turns a found violation into a gap and test
      status: met
      tdd_phase: done
    - id: AC-3
      title: Every success criterion in E-0062's epic spec has a passing demonstration
      status: open
      tdd_phase: done
---

## Goal

Prove E-0061's `O_APPEND` diagnostic-log safety under real multi-process
load (not just the package-level test built in M-0237), document the
triage procedure this epic's findings flow through, and verify the epic's
own success criteria end-to-end before it closes.

## Context

This is the epic's capstone: it depends on all three scenario-tier
milestones (M-0241, M-0242, M-0243) being done. Everything needed to run
this tier already exists by the time this milestone starts — the harness,
the real binary, the logger.

## Acceptance criteria

### AC-1 — Concurrent subprocesses sharing one log file never tear or interleave a line

N real `aiwf` subprocesses, each with `AIWF_LOG=debug` pointed at the same
daily log file, run concurrently. Asserts every resulting line parses
cleanly, every `run_id` appears exactly once, and none is interleaved or
truncated — the harness proving out ADR-0017's Decision #5 under load the
package-level test (M-0237) couldn't exercise on its own, since that test
predates any real verb calling the logger.

### AC-2 — A documented triage procedure turns a found violation into a gap and test

A short, concrete procedure (documented in this milestone's spec or a
pointer from it): a violation the harness surfaces gets a new gap
(`aiwf add gap`) referencing the raw-report event and preserved repo state,
and a minimal regression test is promoted into the normal, every-push
suite — not left living only inside the stress harness.

**The procedure:**

1. **Reproduce empirically first.** Before writing any Go code, drive the
   real compiled binary against a disposable repo and confirm the violation
   by hand — raw `--format=json` output, exact command sequence. A
   hypothesis that turns out wrong under a control experiment (branch
   reachability, git plumbing subtlety) is corrected *before* it becomes a
   gap, not after.
2. **File the gap.** `aiwf add gap --title "<concrete, specific defect>" --discovered-in <M-NNNN>`. The body names what's broken, why it matters, and
   the exact reproduction steps confirmed in step 1 — enough for someone
   with no memory of this session to reproduce it cold.
3. **Write a minimal regression test in the normal, every-push suite** —
   never left living only inside the stress harness. TDD red-first: the
   test fails against the current (buggy) code, using this repo's existing
   test conventions for whatever package owns the defect (a `repoFixture`-
   style real-git test for `internal/check`, a `verb_test` runner for
   `internal/verb`, a `testutil.CaptureStdout`-based integration test for a
   CLI verb's output shape, etc.) — never a stress-harness-only scenario
   substituting for it.
4. **Fix the defect**, confirm the regression test goes green, run the
   project's full branch-coverage-audit + `wf-vacuity` mutation probe on
   the new code before considering it done — the same discipline this
   milestone's own AC-1 used.
5. **Close the loop**: `aiwf promote G-NNNN addressed --by-commit <sha>`
   once the fix and its test are committed. If a finding turns out to
   already be acceptable behavior rather than a defect (confirmed by the
   step-1 reproduction, not assumed), no gap is filed at all — or an
   already-filed gap closes referencing the corrected understanding instead
   of a code fix (see G-0395 below: a `D-0034` decision plus a smaller,
   scoped diagnostic improvement, not the larger persisted-mechanism fix
   the gap's own text first suggested).

Demonstrated four times against this epic's own backlog — every gap this
epic's scenarios surfaced (G-0389, G-0391 from M-0241/M-0242; G-0393,
G-0395 from M-0243), all still open at this milestone's start, closed via
this exact procedure. See the Work Log below for each.

### AC-3 — Every success criterion in E-0062's epic spec has a passing demonstration

Each checkbox in E-0062's *Success criteria* section is walked and
confirmed against the finished harness — not asserted from memory.

## Constraints

- AC-1's test is at real subprocess scale (multiple `aiwf` binary
  invocations), distinct from and in addition to M-0237's package-level
  goroutine/file-handle test — it's proving the same property under a
  higher-fidelity load, not duplicating the earlier test.
- This milestone doesn't introduce new scenario categories — it closes the
  loop on ones already built.

## Design notes

- If AC-3's walk-through finds a success criterion not actually met, that's
  this milestone's problem to resolve (more scenario work, or a scope
  correction to the epic spec with the user's sign-off) — not something to
  gloss over at wrap.

## Surfaces touched

- `internal/stresstest/` (the concurrent-writer-at-scale scenario)
- This epic's spec (`epic.md`) — finalized at wrap per the usual ritual

## Out of scope

- Any new scenario category not already scoped in M-0241–M-0243.
- Making the harness a CI gate — still out of scope for the whole epic, per
  its own spec.

## Dependencies

- M-0241, M-0242, M-0243 — all three scenario tiers must be done.

## References

- `docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`
- `docs/initiatives/robustness-correctness-stress-testing.md`

---

## Work log

### AC-1 — Concurrent subprocesses sharing one log file never tear or interleave a line

Confirmed: n real `aiwf cancel` subprocesses, each pointed at one shared
diagnostic log file via `AIWF_LOG_FILE`, never tear or interleave a line —
the OS-level `O_APPEND` guarantee holds under genuine separate-process
concurrency, not just the package-level goroutine simulation M-0237 already
covers. Every line's `run_id` matches exactly one real invocation's own
`--format=json` correlation id, extending M-0239's single-process
correlation guarantee to concurrent, multi-process load. A vacuity-probe
mutation initially survived (a message-swap in the foreign-run_id branch —
the test asserted a generic phrase, not which id it was attached to);
strengthened the classify tests to name the specific run_id in every
expected violation, then reconfirmed the mutation is caught · commit
7e0b4237 · tests 13/13

### AC-2 — A documented triage procedure turns a found violation into a gap and test

Documented the procedure above and applied it to all four gaps this
epic's scenarios had surfaced (none fixed until now, per the epic's own
manual-triage constraint):

- **G-0389** (`aiwf show`'s not-found path ignored `--format=json`) —
  fixed at its own source; a new integration test pins the JSON error
  envelope · commit dc27bfa4
- **G-0391** (mutating verbs' lock-busy refusal ignored `--format=json`)
  — fixed at the shared chokepoint (`AcquireRepoLock`), rippling through
  23 call sites across every mutating verb; a pre-existing M-0242 stress
  test that had documented this exact bug as expected behavior was
  updated to confirm the fix instead · commit 45f678b2
- **G-0393** (`aiwf archive` could sweep a non-terminal milestone
  alongside its terminal parent) — a new epic-promote-to-terminal guard
  mirrors `aiwf cancel`'s existing non-terminal-children refusal; the
  M-0243 stress scenario that had reproduced the bug now confirms the
  guard refuses instead · commit 4271e580
- **G-0395** (`acknowledge illegal` silently revoked when its ack commit
  becomes unreachable) — investigated further before fixing: empirically
  confirmed the pre-push gate already prevents the corrupted history
  from ever being shared (illegal-transition is error-severity, and the
  shipped hook's exit code is `aiwf check`'s own), and that the exact
  compound failure has never once occurred in this repo's own history
  (56/56 real acknowledgments still reachable). Recorded as D-0034
  (accepted trade-off: DAG-scoping vs. rebase-durability) rather than
  building a persisted-ledger mechanism this repo's own "no separate
  event log" design commitment argues against. The one real, addressable
  gap — a revived finding looking identical to a never-acknowledged one
  — closes with a small, best-effort diagnostic
  (`findDanglingAckHint`) that names the dropped acknowledgment when
  local evidence survives, with no new persisted state · commit d9f32dc3

### AC-3 — Every success criterion in E-0062's epic spec has a passing demonstration

Walked all four bullets in E-0062's Success criteria section against the
finished harness, verified directly (reading source, running tests),
not asserted from memory:

- **"Every scenario... has a deterministic pass/fail oracle"** — met.
  All 12 scenario files under `internal/stresstest/` each carry ≥1
  `classify*` function; confirmed by direct enumeration.
- **"A run aborted mid-way... produces a report that accurately
  reflects everything completed"** — met. Verified directly in source:
  `ReportWriter.WriteEvent` marshals one event then makes exactly one
  `Write()` call; `OpenReportWriter` opens `O_APPEND|O_CREATE|O_WRONLY`
  — matching E-0061's own discipline, so only the file's final line can
  ever be torn. `TestCompose_DropsTruncatedTrailingLine` (M-0240/AC-3)
  confirms `Compose` tolerates exactly that failure mode without
  failing the whole report.
- **"A violation... leaves enough behind... to be reproduced without
  re-running the whole campaign"** — partially met, real gap found and
  triaged rather than glossed over: preserved repo state (confirmed via
  `RunScenario`'s cleanup discipline) and a raw-report event (confirmed
  via `RunRepeated`'s per-attempt `RepeatEvent` log) both hold, but
  `RepeatEvent` carries no `Dir` and no `correlation_id`, and most
  scenarios' shared `runAiwfJSON` helper never enables `AIWF_LOG` for
  the subprocesses it drives — folded into M-0249/AC-2's scope rather
  than filing a 6th, closely-related gap.
- **"Every violation found during this epic is triaged into a gap with
  a minimal regression test..."** — met, and this walk itself found one
  more: G-0388 (discovered in M-0240, still open — the pre-commit
  hook's policy suite lacked the `-parallel 8` cap) had been missed by
  the handoff. Fixed via the same AC-2 procedure. All five gaps this
  epic's work has surfaced (G-0388, G-0389, G-0391, G-0393, G-0395) are
  now `addressed`.

Also found, independent of the four criteria bullets themselves:
`cmd/stresstest run`'s scenario selection is still hardcoded to the
M-0240 placeholder — none of the 12 real scenarios this epic built are
reachable through the dedicated on-demand binary E-0062's own Scope
section describes, only through `go test`. Filed as G-0397; per the
user's direction, resolved via a new milestone (M-0249) added to the
epic rather than fixed inline here or silently deferred — epic close
now happens after M-0249, not this milestone. M-0244's own title was
retitled (dropping "; epic close") to match this new reality.

## Decisions made during implementation

- D-0034 — DAG-scoped acknowledge-illegal exemption trades off against
  rebase durability (G-0395)

## Validation

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -race -parallel 8 -count=1 ./...` (full repo) — green.
- `make lint` (full `golangci-lint` set, worktree-scoped cache) — 0 issues.
- `aiwf check` — 0 error-severity findings (7 warnings: the standing
  no-upstream-configured notice, an archive-sweep-pending aggregate, and
  5 per-gap terminal-entity-not-archived notices for G-0388/G-0389/G-0391/
  G-0393/G-0395, all awaiting a future `aiwf archive --apply` sweep).
- `make coverage-gate` (diff-scoped branch-coverage audit) — green after
  each AC's commit.
- Each AC's implementation went through a `wf-vacuity` mutation probe;
  AC-1 and both of AC-2's larger fixes (G-0391's chokepoint, G-0393's
  guard, G-0395's diagnostic) each surfaced and fixed a real surviving
  mutant before landing — see each AC's own Work log entry above.

## Deferrals

- G-0397 — cmd/stresstest run has no way to select any of the 12 real
  scenarios; resolved via a new milestone, M-0249, added to E-0062
  (epic close now happens after M-0249)

## Reviewer notes

- (none)
