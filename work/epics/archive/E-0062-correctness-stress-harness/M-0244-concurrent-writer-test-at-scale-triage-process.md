---
id: M-0244
title: Concurrent-writer test at scale; triage process
status: done
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
      status: met
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

- `internal/stresstest/` (the concurrent-writer-at-scale scenario;
  later, the property-test tolerance fix for G-0398)
- `internal/cli/show/`, `internal/cli/cliutil/` (G-0389, G-0391)
- `internal/verb/` (G-0393's guard, reconciled with the independently-
  landed G-0394 fix from `main`; the new pinning tests for G-0398)
- `internal/check/` (G-0395's dangling-ack diagnostic; the new
  standing `epic-terminal-non-terminal-children` rule and its hint)
- `internal/workflows/spec/rules.go` (the reconciled guard's spec cell)
- `scripts/git-hooks/pre-commit` (G-0388)
- `CHANGELOG.md`, `internal/skills/embedded/aiwf-check/SKILL.md`
  (corrected to match the reconciled guard's actual behavior)
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

### Wrap-review corrective work

The milestone went through three independent review rounds before
wrap, each dispatched fresh-context per the usual ritual.

**Round 1 — code-quality + design-quality, on the AC-1/AC-2/AC-3 diff.**
Found 4 corrective items: (1) duplicated non-terminal-children traversal
between `Promote`'s new G-0393 guard and `Cancel`'s existing D-0003
guard — extracted a shared helper; (2) no standing `aiwf check` rule
backstopping the guard regardless of how the bad state was reached —
added `epic-terminal-non-terminal-children`; (3) the new spec-table
Rule cell's `Sources` field silently empty with no rationale — added
one, citing the hardcoded M-0123 audit set that blocks a fresh
Decision reference; (4) the G-0395 `danglingHint` diagnostic wired
into `illegalTransitionFindings` but not the sibling
`forcedUntraileredFindings` — extended it, with a new end-to-end test.

**Mid-fix: an independently-landed duplicate.** Re-running the full
suite after item 2 landed surfaced a failure in a *different*,
already-merged milestone's property test
(`TestVerbSequenceScenario_FullWalkAcrossAllKindsPasses`, M-0241).
Investigating led to discovering that `main` had, while this branch
was in flight, independently merged its own fix (G-0394, filed under
E-0063) for the identical underlying gap G-0393 already closed here —
same shape, same finding code, landed via a separate `wf-patch`
branch neither line of work knew about. Merged `main` in and
reconciled the two: kept main's file organization
(`internal/verb/cancel_guards.go`) and its new archive-time
defense-in-depth guard, restored this branch's broader scope (`done`
*and* `cancelled`, not `done`-only) and unconditional semantics (no
`--force` bypass, matching `Cancel`'s own D-0003 guard — main's
version was accidentally force-bypassable). Also caught and fixed one
duplicate test-fixture artifact the naive git auto-merge produced in
`internal/cli/integration/show_scopes_test.go` (both branches had
independently patched the same pre-existing test for the same reason,
in different spots).

Re-running the full suite after reconciliation surfaced the *original*
property-test failure again, now traced to its real cause: the
standing check-rule (item 2 above) correctly flags a terminal epic
with a fresh non-terminal milestone, and the M-0241 property test's
own independent-per-kind random walk occasionally constructs exactly
that state by accident (random-walking the epic to `done`/`cancelled`
before creating the milestone that needs it as `--epic` parent).
Digging into *why* `aiwf add milestone` even allows that revealed a
genuinely separate, previously-invisible gap: neither `add.go` nor
`import.go` has ever checked the target epic's own status — the
refusal that already happens today is an accident of the generic
projection-findings gate, not a dedicated guard. Filed as G-0398
(deferred, not fixed — see Deferrals) and taught the property test to
tolerate this one specific, known refusal shape instead of treating
it as a hard failure, with a direct unit test and a real-repo
empirical confirmation (built the binary, confirmed the refusal lands
no commit) before writing either.

**Round 2 — code-quality + design-quality, on the reconciliation.**
Found 5 issues, independently confirmed on the headline one by both
reviewers (and by hand before acting): `CHANGELOG.md`'s `[Unreleased]`
entry (carried in by the merge) still described main's
pre-reconciliation, force-bypassable, `done`-only guard — rewritten.
The finding's hint text falsely attributed the accidental add/import
refusal to "a hand-edit, `--force`, or a pre-guard binary" — none of
which apply there — reworded in both `hint.go` and the `aiwf-check`
skill's Findings table. G-0398 was scoped to `aiwf add milestone`
only; `aiwf import` has the identical gap (verified via direct repro
and code tracing: `import.go`'s `lookupEpicDir` mirrors `add.go`'s
`newEntityPath` precisely) — widened the gap's scope. The standing
rule's own doc comment didn't acknowledge its accidental second job
(the only thing blocking add/import today) — updated it, and added a
dedicated pinning test (`TestAdd_MilestoneUnderTerminalEpic_RefusedViaFindings`,
`TestImport_MilestoneUnderTerminalEpic_RefusedViaFindings`) so a
future refactor of the generic gate can't silently drop the
protection. One status-skip test pair didn't distinctly exercise its
"unknown status" branch (confirmed via a mutation probe) — widened to
a table covering both empty and unknown separately.

**Round 3 — YAGNI/KISS/duplication/dead-code, on the milestone's full
diff against `main`.** Verdict: essentially clean, right-sized. One
finding — `verbEnvelope.Result.ID`, a struct field nothing in the
package ever read (every scenario needing an id reads
`Metadata.EntityID` instead) — verified via grep and removed.
Everything else checked out with reasoning recorded in the review
itself: the three guard error types are genuinely distinct payloads,
not collapsible boilerplate; the G-0395 diagnostic's function split is
test-motivated; the 21 lock-fix call sites are genuinely identical
mechanical edits through one shared chokepoint; the two new pinning-
test files assert different things about different verbs.

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
  mutant before landing — see each AC's own Work log entry above. The
  wrap-review corrective work repeated the same discipline: a mutation
  probe on the standing check-rule's status guards (Round 1) and again
  on the widened "unknown status" tests (Round 2), each catching and
  fixing a real surviving mutant before landing.
- Re-validated after the merge reconciliation and again after each
  review round: `go build`, `go vet`, `make lint`, `go test -race
  -parallel 8 -count=1 ./...` (full repo), and `make coverage-gate` all
  green at every checkpoint; `aiwf check` against this actual repo
  confirmed 0 error-severity findings throughout — including a direct,
  independent manual scan of every epic/milestone status pair on disk
  (not just the tool's own output) before concluding the new standing
  rule wasn't firing on anything real here.

## Deferrals

- G-0397 — cmd/stresstest run has no way to select any of the 12 real
  scenarios; resolved via a new milestone, M-0249, added to E-0062
  (epic close now happens after M-0249)
- G-0398 — `aiwf add milestone` and `aiwf import` accidentally-not-
  purposefully refuse creating a milestone under an already-terminal
  epic (no dedicated precondition on either verb; today's refusal is a
  side effect of the standing check-rule tripping the generic
  projection-findings gate every mutating verb runs)

## Reviewer notes

- Three independent review rounds ran before wrap (code-quality +
  design-quality on the original AC diff; code-quality + design-
  quality again on the post-reconciliation diff; a YAGNI/KISS/
  duplication/dead-code pass on the milestone's full contribution).
  Every finding from all three rounds was fixed, not deferred — see
  "Wrap-review corrective work" above for the full narrative. No
  findings were judged out of scope or left open.
- The G-0393/G-0394 collision (two independently-filed gaps
  converging on the identical fix, discovered mid-review via a
  property-test failure that traced back to a `main`-side merge) is
  the most structurally significant thing that happened in this
  milestone outside its own stated scope. Worth a standing lesson:
  when a long-running branch's own fix collides with something
  already landed on `main`, `git diff` alone can miss it — the real
  signal here was a *test* failure in unrelated, already-merged code,
  not a merge conflict (the naive auto-merge produced zero textual
  conflicts; the actual collision was two files each declaring the
  same Go symbols, invisible to git's line-based merge until `go
  build` was run).
