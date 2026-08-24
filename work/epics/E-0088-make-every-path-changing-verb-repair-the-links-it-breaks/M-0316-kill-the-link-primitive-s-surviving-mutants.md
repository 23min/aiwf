---
id: M-0316
title: Kill the link primitive's surviving mutants
status: in_progress
parent: E-0088
depends_on:
    - M-0315
tdd: none
acs:
    - id: AC-1
      title: Survivor density in the link subsystem meets the kernel baseline
      status: open
    - id: AC-2
      title: Every remaining survivor is recorded as equivalent or tracked
      status: open
---

## Goal

Bring the link subsystem's tested edges up to the standard the rest of the
kernel already meets, using surviving mutants as the work list and the measure.

## Context

Mutation testing across six packages and roughly 28,000 production lines
established a kernel-wide baseline of 7.7 surviving mutants per thousand lines,
with per-package efficacy between 88.6% and 96.6%. The link subsystem is the
outlier. Measured at this milestone's start (`419a0890a`): `linkregion.go` at
70.4 survivors per thousand lines — the highest density in the kernel —
`linkrewrite.go` at 25.7, `pathrewrite.go` at 21.1 and `archive.go` at 18.8.
Two of those differ from the figures E-0088's Context records, which were taken
before M-0315 added lines to `linkrewrite.go` and `archive.go`. No raw report
survives from that measurement, so only the densities can be compared, not the
counts behind them.

`archive.go`'s share of that is not link code. Its link-rewriting functions —
`planArchiveRewrites`, `linksIntoMove`, `archiveEntityMoves`, `entityBody`,
`workingBodyAt` — have no survivors between them; the density comes from the
archive verb's commit-message builder, planning path and git-status plumbing,
which G-0630 owns. AC-1 measures the link primitive alone for that reason
(D-0076).

The link primitive's 19 survivors are 11 conditional-boundary, 4 invert-negatives
and 4 arithmetic-base, with none surviving from conditional negation. The
boundary majority is the signature of a happy path under test and edges that are
not, and it matches the two defects E-0088's earlier milestones fix: both were
edge cases that the existing tests walked past.

This milestone runs last of the code milestones so the outbound paths M-0315
adds are measured alongside the rest rather than becoming a fresh untested
surface.

## Acceptance criteria

### AC-1 — Survivor density in the link subsystem meets the kernel baseline

Unexplained-survivor density across the link primitive — `linkregion.go`,
`linkrewrite.go` and `pathrewrite.go`, 509 lines — is at or below 7.7 per
thousand lines, measured by the same gremlins invocation that established the
baseline (`--workers 1 --timeout-coefficient 15`). A survivor counts until it
carries the equivalence argument AC-2 requires, so the bar cannot be met by
assertion. The before and after numbers are both recorded with the command that
produced them.

`archive.go` is outside the denominator per D-0076: every one of its survivors
is in the archive verb rather than the link primitive, and G-0630 owns them.

### AC-2 — Every remaining survivor is recorded as equivalent or tracked

No survivor is left unexplained. Each remaining one is either recorded as an
equivalent mutant with the argument for why the mutation cannot change observable
behavior, or tracked as work with its own entity. A survivor count that falls
without an account of what remains does not satisfy this.

## Constraints

- **Measure, do not assert.** A claim that a survivor is dead names the command
  and its output. This milestone's whole deliverable is a measurement, so an
  unmeasured claim is the failure mode.
- **Equivalence needs an argument.** Naming a mutant equivalent requires saying
  why the mutation cannot change observable behavior, not that a test was hard
  to write.
- **Tests pin behavior, not implementation.** A test written to kill a specific
  mutant must still assert what the code does for given inputs.

## Design notes

Two known measurement traps apply to this milestone and will otherwise mislead
it. Gremlins places mutants inside multi-clause `case` conditions while Go
instruments the case body, so switch-dense code reports phantom "not covered"
mutants — treat that column as an upper bound and cross-check against
`go tool cover`. Separately, gremlins sees only a package's own tests unless
`--coverpkg` is set, so efficacy is trustworthy only where tests are co-located
with the code. Both traps produced false findings during E-0088's planning.

## Out of scope

- Production behavior changes. If killing a mutant requires changing behavior
  rather than adding a test, that is a defect and belongs in its own entity.
- Packages outside the four named files.

## Dependencies

- M-0315 — its outbound paths are part of what this milestone measures.

## References

- E-0088 — the parent epic, which records the baseline and the per-file densities
- D-0076 — why AC-1 measures the link primitive rather than the four named files
- G-0630 — `archive.go`'s survivors, which are archive-verb code
- `internal/verb/linkregion.go`, `linkrewrite.go`, `pathrewrite.go` — the surface
  AC-1 measures; `archive.go` is measured for AC-2's accounting only
- `.github/workflows/mutate-hunt.yml` — the invocation and why its flags are set
- [`docs/initiatives/entity-links-by-id-not-path.md`](../../../docs/initiatives/entity-links-by-id-not-path.md)
  — captured while measuring this milestone: the link format that would retire
  the subsystem measured here, rather than test it further

## Work log

### AC-1 — Survivor density in the link subsystem meets the kernel baseline

Both measurements ran in a detached worktree pinned to the commit under
measurement, so no edit in the working tree could disturb one in flight.
gremlins 0.6.0, go 1.25.11, linux/amd64. The scoping derives the exclusion set
rather than listing it, so it stays correct as `internal/verb` gains files:

```bash
excl=(); for f in $(ls internal/verb/*.go | grep -v '_test\.go' \
  | grep -vE '/(linkregion|linkrewrite|pathrewrite|archive)\.go$' \
  | xargs -n1 basename); do excl+=(-E "${f//./\\.}"); done
gremlins unleash --workers 1 --timeout-coefficient 15 -o report.json \
  "${excl[@]}" ./internal/verb
```

| | before (`419a0890a`) | after (`0b335c19d`) |
|---|---|---|
| killed / lived / not covered | 117 / 38 / 3 | 131 / 25 / 2 |
| efficacy | 75.48% | 83.97% |
| timed out | 0 | 0 |
| wall time | 36m45s | 36m17s |

Zero timeouts in both runs, so neither efficacy figure is inflated by a mutant
counted killed for running slowly.

The link primitive, which is what AC-1 measures:

| file | lines | lived before → after | per thousand |
|---|---|---|---|
| `linkregion.go` | 142 | 10 → 1 | 70.4 → 7.0 |
| `linkrewrite.go` | 272 | 7 → 5 | 25.7 → 18.4 |
| `pathrewrite.go` | 95 | 2 → 0 | 21.1 → 0.0 |
| **total** | **509** | **19 → 6** | **37.3 → 11.8** |

All six remaining carry the equivalence arguments below, so **unexplained
survivors are 0, against a bar of 7.7 per thousand**. The verdict survives a
narrower denominator: counting only `linkregion.go` and `linkrewrite.go` (414
lines, excluding `pathrewrite.go`, which computes filenames and reads no link)
permits 3.2 and the count is still 0.

The tests closed **14** blind spots. `linkrewrite.go:132` and `:136` were already
killed at `419a0890a` — established by applying each mutation to a worktree at
the base commit, without the new tests present, and observing the suite go red.
They stay `NOT COVERED` in both reports because gremlins places the mutant at a
`case` expression while Go instruments the case *body*; the milestone's Design
notes name this trap and prescribe the cross-check, and `go tool cover` puts hit
count 1 on the blocks at `132.18` and `136.39`. The commit subject of
`0b335c19d` claims 16; the attribution above is the measured one.

Commit `0b335c19d` · 243 test lines, no production change · tests 3 new
functions, 2 tables extended.

### AC-2 — Every remaining survivor is recorded as equivalent or tracked

27 entries in the after-report are not `KILLED` (25 lived + 2 not covered), and
each has an account: 6 in the link primitive are argued equivalent below; 19 in
`archive.go` are archive-verb code, owned by G-0630; 2 are the mis-reported
pre-existing kills described above.

## Equivalent mutants

Six mutants in the link primitive survive and cannot be killed, because none
changes what the code produces. Each argument was checked by applying the
mutation and running the whole `internal/verb` package — the suite stayed green
in every case, which is the observation the argument explains rather than the
argument itself.

**`linkregion.go:109` — `for i < len(s)` widened to `i <= len(s)`.** `i` advances
only through `i = closeAbs + 1`, so it can arrive at exactly `len(s)`. On that
extra iteration `strings.Index(s[len(s):], "](")` searches the empty string and
returns -1, taking the not-found branch, which writes the empty string and
breaks. The iteration appends nothing and emits no region, so the region list is
identical for every input.

**`linkrewrite.go:169` — `colon < slash` widened to `colon <= slash`.** `colon`
indexes `':'` and `slash` indexes `'/'` in the same string, so one index cannot
satisfy both. `colon == slash` is unsatisfiable whenever both are non-negative;
the `slash == -1` disjunct short-circuits the absent-slash case and the guard
above already returned for `colon <= 0`. Every path reaching the comparison has
`colon != slash`, where `<` and `<=` agree by definition.

**`linkrewrite.go:254` — the four mutants in the capacity expression.** The
mutated expression is the third argument of `make([]string, 0, N)`. The slice is
built at length 0, its contents come entirely from the `append` calls below it,
and the function returns `strings.Join(segs, "/")` — so capacity governs
allocation and regrowth, never element values or count, and every `N >= 0` gives
byte-identical output. That the surviving forms are non-negative holds for all
inputs rather than by observation: the loop above establishes
`0 <= common <= min(len(dirParts), len(targetParts))`, so both subtractions are
non-negative and turning either into an addition only increases the result. A
mutation that did drive capacity negative would panic in `make`, and the fifth
mutant at `254:50` is exactly that case — it is killed.

## Decisions made during implementation

- **D-0076** — measure survivor density over the link primitive rather than the
  four files E-0088's Context names, counting only survivors that carry no
  equivalence argument. Taken after the baseline showed AC-1 as written could not
  be satisfied by doing what the milestone is named for.

## Validation

- `make check-fast` — exit 0.
- `aiwf check` — 0 errors. Warnings are pre-existing archive-sweep housekeeping
  and `epic-active-no-drafted-milestones`, which fires because M-0316 was the
  epic's last `draft`.
- Two full mutation runs, tabulated above, 0 timeouts in each.
- Every kill and every equivalence claim checked by applying the mutation, not by
  reading. Kill *attribution* additionally checked against the base commit.

## Deferrals

- **G-0630** — `archive.go`'s 19 surviving mutants, all archive-verb code
  (commit-message composition, sweep planning, git-status plumbing). Out of
  AC-1's denominator per D-0076; whether all 19 are worth killing is open, and
  none has been checked for equivalence.
- **G-0632** — a verb's `Long` help can contradict its behaviour with nothing
  catching it. Found because `move`'s went stale when M-0315 changed the
  behaviour; corrected in `61abf68c5`, but the absence of a guard is the defect
  and it persists.

## Reviewer notes

An independent fresh-context reviewer over the full change-set returned
**request-changes** with five confirmed defects. Every one was reproduced before
acting on it, and all five are fixed in this milestone.

The one that matters: **the kill count was wrong — 16 claimed, 14 real.** The
probe that produced 16 applied each mutation at HEAD and observed the suite go
red, which establishes that a mutant dies, not that the new tests killed it. Two
were already dead at the base commit. The corrected method — probe at base, and
attribute only what survives there — is what the Work log records.

The other four were all overstated counts or an internal contradiction, and all
four sat in prose rather than code: the initiative's production-line and
test-line figures, its claim that most of those lines are masking, and a section
asserting flatly what a later section listed as unverified. That the code came
through clean and the prose did not is the finding worth carrying forward.

Two judgment calls, both taken as the reviewer proposed:

- **`linkregion.go:138`** was killed by asserting the exact region list, and the
  mutant is unobservable through the package's exported surface. Rather than drop
  the test or reclassify the mutant, `splitLinkPathRegions`'s doc comment now
  declares the no-empty-regions invariant, so the test pins a stated contract
  instead of an internal representation.
- **Five of twelve `TestIsRepoPathDestination` rows were cut.** Those switch arms
  generate no mutants and the shapes were already pinned end to end by the
  archive outbound test, so the rows asserted a second time what could not fail
  independently.

Declined, with reasons:

- The reviewer read **59% of the diff being prose** as the wrong balance for a
  measurement milestone. The prose is D-0076, G-0630 and the initiative — each
  directed rather than drift, and the initiative is the epic's most useful
  output, not its overhead.
- **D-0076 has no `relates_to: M-0316`.** `--relates-to` exists only on
  `aiwf add`; no verb edits it afterwards and frontmatter is not hand-edited, so
  the milestone is reachable from the decision through prose only.

Two observations recorded rather than acted on:

- **AC-1's restated bar does little work.** 509 lines at 7.7 permits 3 unexplained
  survivors and the outcome is 0, so in this result AC-2 satisfies AC-1
  automatically. The bar is not vacuous — it caps how many survivors may be
  punted to a gap rather than argued — but it is a weaker instrument than the one
  it replaced, and that is the honest price of fixing the denominator mid-flight.
- **E-0088's third success criterion still reads "across the files named in
  *Context*"**, which is wider than AC-1's denominator. It is satisfied by this
  milestone together with G-0630. Reconciling the two phrasings belongs at epic
  close; leaving a reader to do it is the situation D-0076 says it wants to avoid,
  and this note is not a substitute for closing it.
