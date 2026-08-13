---
title: Quality signal and cadence — making a red gate mean the change broke something
status: captured
date: 2026-07-27
---

# Quality signal and cadence — making a red gate mean the change broke something

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf
entity kind ([G-0311](../../work/gaps/G-0311-no-cross-cutting-initiative-tier-above-epic-for-multi-component-features.md)),
so this file lives under `docs/initiatives/` as an umbrella capture,
following the precedent of [`id-lifecycle.md`](id-lifecycle.md) and
[`agent-agnostic-execution-topology.md`](agent-agnostic-execution-topology.md).

This is not an ADR: it ratifies nothing. It is not a plan: it deliberately
avoids committing to epics, milestones, or sequencing beyond naming which
finding is acute and which is not. Its job is to hold the shape of one
question still — *does aiwf's quality apparatus actually tell you anything
about your change?* — long enough that a right-sized plan can be drafted
from a coherent center.

## Initiative statement

aiwf carries an unusually dense quality apparatus: 66 policy tests under
`internal/policies/`, a 2.2:1 test-to-production line ratio, a diff-scoped
coverage gate, a firing-fixture meta-gate, fuzz targets, property tests,
mutation testing, a correctness stress harness, and `aiwf check` running as
both a pre-commit and a pre-push hook. The apparatus is not the problem, and
this initiative is not a request for more of it.

What the apparatus *reports* is the problem. The `go.yml` workflow failed
31 of its last 100 runs. For eleven consecutive days it failed on
essentially every run, while 1,151 commits and eight epic wraps landed
through it. A gate that is red by default cannot distinguish a regression
from the weather, and the cost is not the failed runs — it is that during
that window nobody could have learned anything from a green one either.

Alongside that sits a second, milder observation about what the work *reads*
like once it has landed: 73% of commits touch planning markdown and 12%
touch Go. That ratio is the audit trail working exactly as designed
(committed property #7, one commit per mutation), but nothing today lets a
reader see past it to the engineering history underneath.

Both are signal problems, not coverage problems. That framing is the
initiative's center.

## Measured baseline

Measured 2026-07-27 against `main` at 8,456 commits, from `git log`,
`gh run list`, and direct timing. Recorded so a later pass can tell movement
from noise.

| Signal | Value |
|---|---|
| Commits since 2026-04-26 | 8,456 (~100/day) |
| Commits touching `*.go` | 1,031 (12%) |
| Commits touching `work/*.md` | 6,145 (73%) |
| Commits carrying a planning-verb trailer | 6,549 (77%) |
| `aiwf promote` commits | 3,555 (42% of all) |
| Epics done / cancelled | 66 / 5, median ~1 day each |
| `go.yml` failures, last 100 runs | 31 |
| `go.yml` duration, median / p90 / max | 3.3 / 4.5 / 14.3 min |
| Fully-red window | 2026-07-08 → 2026-07-18 (11 days) |
| Landed during that window | 1,151 commits, 155 touching Go, 8 epic wraps |
| Production / test Go LOC | 151,619 / 336,468 (2.2:1) |
| `internal/cli` production / test LOC | 19,738 / 60,196 (3.05:1) |
| `internal/policies` | 39,420 LOC, 66 policies, 16% of all Go |
| Open gaps | 82, flat at 70–74 for six weeks |
| Gaps carrying `discovered_in` | 186 (144 milestone, 42 epic) |
| `aiwf check` wall time | 6.7s |
| `internal/stresstest` package test time | 34s local, 77s in CI |

Re-measured 2026-07-30 against the same surfaces, resolving the failure rate
into its causes and the stress-harness timing into its isolated and co-tenant
cases:

| Signal | 2026-07-27 | 2026-07-30 |
|---|---|---|
| `go.yml` failures, last 100 runs | 31 | 32 |
| — of which the `vuln` job | not split | 21, all 2026-07-08 → 07-18 |
| — of which the stress harness | not split | 18, 2026-07-11 → 07-29 |
| — of which the diff-scoped coverage gate | not split | 8, 2026-06-28 → 07-21 |
| `internal/stresstest` time, isolated | 34s local | 38s local, 38s on 4 cores |
| `internal/stresstest` time, co-tenant with `./...` | 77s in CI | 66.7s on 4 cores, 65–77s in CI |
| `cmd/stresstest` time in CI | not measured | 42–68s |
| `internal/stresstest` coverage, full run | not measured | 85.5% |
| `internal/stresstest` coverage, all real-binary drivers skipped | not measured | 34.7% |
| `internal/stresstest` coverage, non-hermetic scenarios skipped | not measured | 62.8% |
| Exact Go toolchain pins across workflow files | not measured | 7 |

The three causes have different lifespans. The `vuln` job closed with the
toolchain bump on 2026-07-18 and has not failed since. The stress harness
opened the fully-red window and has continued past its close. The coverage
gate predates the window entirely, first failing on 2026-06-28. Neither of
the latter two was caused by the window.

## Findings

### Q1 — The gate is red often enough that a red gate carries no information

Acute, and the only finding here with a deadline attached in the sense that
every push until it lands pays for it. Three independent causes, of different
character and different lifespans, tracked as
[G-0457](../../work/gaps/archive/G-0457-ci-gate-is-red-often-enough-to-carry-no-signal-about-the-change.md)
(placement and the `govulncheck` lanes),
[G-0468](../../work/gaps/archive/G-0468-stress-scenario-oracles-conflate-runner-contention-with-an-aiwf-defect.md)
and
[G-0467](../../work/gaps/archive/G-0467-lock-busy-refusal-emits-an-empty-error-code.md)
(the oracles and their enabling error-code fix), and
[G-0469](../../work/gaps/archive/G-0469-diff-scoped-coverage-gate-fires-only-in-ci-after-the-trunk-push-lands.md)
(the coverage gate's tier).

**Q1a — `govulncheck` blocks on stdlib CVEs against a pinned toolchain.**
`go.yml`'s `vuln` job installs `golang.org/x/vuln/cmd/govulncheck@latest` and
runs it over `./...`. In mid-July it began reporting `GO-2026-5856` in
`crypto/tls@go1.25.11` — the toolchain version pinned in the same workflow.
No change in this repository could clear it; only a toolchain bump could, and
that took ten days. The job's own comment justifies blocking on the grounds
that "the dep set is small (go-cmp / goldmark / yaml.v3) so the run is ~10s"
— a rationale about *dependency* CVEs that silently also governs *stdlib*
CVEs, which have a completely different remediation path and latency. The
two cases were never distinguished.

This one was an acute burst: 21 red runs, all inside the eleven-day window,
none since the toolchain bump on 2026-07-18. The structural exposure survives
that bump. The scanner's own version still resolves at `@latest` on every
run, and the toolchain is pinned to an exact patch in seven places across two
workflow files, so each stdlib disclosure costs seven edits before it can be
cleared. This was the third such forced bump.

**Q1b — The stress harness runs inside the default test step, on two lanes.**
`go.yml`'s test job runs `go test … ./...`, which sweeps in
`internal/stresstest` — real-binary scenarios launching concurrent `aiwf`
subprocesses against disposable git repos, 65–77s in CI — and also
`cmd/stresstest`, whose `TestRunRun_ScenarioAll_*` tests invoke the whole
catalog, the same run `make stress` performs, for a further 42–68s. This is
the chronic cause: 18 red runs spanning 2026-07-11 to 2026-07-29 without
interruption. It opened the fully-red window and has continued past its
close, so unlike Q1a it is not bounded by that window at either end.

The failures are not one shape. `classifyConcurrentIDAllocation` and its
counterpart in `concurrent_move.go` require every concurrent actor to succeed
*"within repolock's timeout"* — hardcoded at two seconds in
`internal/cli/cliutil/lock.go` — so a tail actor receiving the documented
busy refusal is recorded as a violation. `mid_write_kill.go` fails when its
poller does not catch the sibling temp file in flight, which is a failure to
sample rather than a broken atomic-write property.
`concurrent_writer_at_scale.go` models contention correctly but recognizes
the busy envelope by substring match, so a busy refusal it fails to
recognize aborts the run outright. Each of the three asserts a property of
the machine rather than a property of aiwf.

The flake tracks co-tenancy, not machine size. Run in isolation on four cores
both stress packages pass five repeats out of five; run co-tenant with the
full `./...` on those same four cores, `internal/stresstest` takes 66.7s,
matching what CI observes. Placement is therefore load-bearing, and any
destination sharing a runner with a broad sweep reproduces the flake.
[G-0438](../../work/gaps/archive/G-0438-flake-hunt-yml-s-count-10-sweep-is-undersized-for-its-github-runner.md)
records the same finding from `flake-hunt.yml`, naming these same packages;
that workflow now fans out one package per runner, so it no longer runs a
broad sweep for anything to be co-tenant with.

`CLAUDE.md` describes the harness as *"dev-only tooling, never installed
alongside `cmd/aiwf`, run by hand rather than scheduled or wired into
`make ci`."* The workflow disagrees with the stated contract, and
`cmd/stresstest`'s whole-catalog test disagrees most directly of all.
Whichever way that reconciles, the two should say the same thing.

The seam is oracle shape, not test shape. Splitting on "drives a real
subprocess" is the seam the package's own `*_classify_test.go` /
`*_test.go` layout suggests, and it is the wrong one: it would bench every
real-binary driver including the hermetic ones, costing coverage
(`internal/stresstest` falls to 34.7%) for scenarios that were never the
problem. Splitting on what the oracle asserts — does it depend on timing,
scheduling, or an observation window — keeps the workflow-legality walk and
the deterministic collision, isolation and disk-fault scenarios on the
every-push path at 62.8% coverage, and benches only the nine that race real
processes or wait on a clock. Category name is not the criterion either:
`disk-fault` is fault-injection with no timing construct at all, while
`concurrent-milestone-race` requires every racing actor to serialize inside
a two-second deadline.

Either way the split restores signal without restoring the scenarios' own
trustworthiness, which is why the oracle work is scoped separately.

**Q1c — The diff-scoped coverage gate can fire nowhere but CI.**
`scripts/git-hooks/pre-push` runs the lint boundary, the gitleaks secret
scan, and the comment history-attrition scan. It does not run the coverage
gate, and `make coverage-gate` is invoked by hand and named in no ritual. A
changed line with no covering test therefore surfaces only after the push has
landed on trunk — 8 red runs, a quarter of the window's total. The gate is
correct and its findings name real untested statements; the defect is that it
fires past the boundary it exists to protect. The comment history-attrition
scan, which shares the same diff-scoped shape and the same base expression,
is wired into pre-push on exactly the reasoning that was not applied here.

**Why this is Q1 and not Q2.** Every other finding here is about making a
gate stronger. This one is about making any gate mean anything at all. It is
not, however, the cheapest. Restoring the signal is one `wf-patch` — the
placement split plus the two `govulncheck` lanes — but the oracle redesign
underneath Q1b is larger, and the two must not be conflated. A placement
change on its own is a tourniquet: it stops the bleeding and leaves the
scenarios reporting contention as defect wherever they end up running.

### Q2 — Several gates are shallower than their names imply

The reflex when a class of defect escapes is to add a policy. That reflex has
been exercised 66 times and produced the largest package in the repository
(`internal/policies`, 16% of all Go, 39,420 lines). It is a good reflex —
most of those policies pin something real. But adding a 67th policy is
consistently cheaper than deepening one of the existing 66, and cheapness is
the wrong selector when the existing ones have known depth gaps:

- **[G-0253](../../work/gaps/G-0253-branch-coverage-audit-is-statement-scoped-not-per-arm-branch-coverage.md)**
  — the diff-scoped coverage gate, the single most load-bearing quality
  chokepoint in the repo, is *statement*-scoped with a `//coverage:ignore`
  escape hatch. A defensive branch whose untested arm never executes can
  still read as covered. The gate's name promises more than it delivers, and
  every AC promoted on the strength of it inherits that gap.
- **[G-0110](../../work/gaps/G-0110-gremlins-diff-ref-filter-excludes-new-files-entirely-manual-mutation-review-needed-for-m-0094-95-96.md)**
  — mutation testing's `--diff <ref>` filter excludes new files entirely. The
  blind spot is precisely on newly-written code, which is the code most
  likely to carry an untested mutant.
- **[G-0317](../../work/gaps/G-0317-skill-edit-backstop-checks-test-references-path-not-asserts-changed-section.md)**
  — the ritual-content backstop checks that *some* test references the edited
  `SKILL.md` path, not that a test asserts against the section that changed.
- **[G-0328](../../work/gaps/G-0328-golden-fixture-byte-identity-comparator-for-aiwf-check.md)**
  — `aiwf check` has no golden-fixture byte-identity comparator, so output
  drift is caught by substring assertions, which `CLAUDE.md` itself already
  names as insufficient ("substring assertions are not structural
  assertions").

A fifth entry is a different kind of shallowness, and it is about *who
judges* rather than how deeply:

- **The branch-coverage audit is performed by the agent that wrote the code.**
  `wf-tdd-cycle` states this plainly — the audit is "agent-performed — a
  manual branch-walk, not a tool invocation," and the ritual is explicit that
  "hard rule" means *you must perform this walk*, not that a tool enforces it.
  The reasoning is sound, since mechanical coverage stops at statements. But
  the consequence is that the repo's most load-bearing quality claim rests on
  self-report, with no compensating control. This is orthogonal to G-0253: that
  gap is about granularity, this is about independence, and fixing either
  leaves the other standing.

The vocabulary for classifying these lives in
[`../design/oracles.md`](../design/oracles.md) — what makes a check able to
decide anything, and which property each of the above loses. G-0253 and G-0328
lose *depth*; G-0110 loses *reach*; G-0317 loses *specificity*; the audit above
loses *independence*, which is the property whose absence is hardest to see,
because a self-judged gate reports exactly as green as a real one.

These are not a list of things to build. They are evidence for a stance:
**prefer deepening an existing gate to adding a new one**, and make that the
default tiebreak when a defect class escapes. The counter-argument is real —
a narrow new policy is often genuinely the right shape, and G-0253's fix
needs AST-level arm enumeration, which is a materially bigger lift than any
of the 66 policies cost individually. The stance is a tiebreak, not a ban.

### Q3 — Structural duplication is discovered well and absorbed ad hoc

The discovery mechanism works, demonstrably. The `wf-structural-sweep`
ritual, run once, produced four well-scoped findings:
[G-0448](../../work/gaps/G-0448-check-rule-list-split-across-two-dispatch-surfaces-no-single-source.md)
(check rules dispatched from two parallel surfaces split by function
signature rather than by design, with nothing able to enumerate "all the
rules" from one place),
[G-0453](../../work/gaps/G-0453-unify-shorthash-short-sha-abbreviation-helpers-in-check-width-decision.md)
(two SHA-abbreviation helpers at 7 and 8 chars),
[G-0454](../../work/gaps/G-0454-unify-the-three-id-shape-parsers-in-entity-parseidnumber-vs-canonicalize.md)
(three id-shape parsers in `entity`), and
[G-0455](../../work/gaps/G-0455-consolidate-markdown-heading-walk-state-machines-in-body-go-evaluate-first.md)
(three-to-four markdown heading-walk state machines in one file). The earlier
verb-layer audit fed [E-0069](../../work/epics/archive/E-0069-close-the-verb-layer-call-graph-audit-findings/epic.md)
and [E-0072](../../work/epics/archive/E-0072-cli-verb-scaffold-convergence/epic.md),
and produced a shipped clone-detection linter
([G-0423](../../work/gaps/archive/G-0423-no-clone-detection-linter-to-catch-duplicated-verb-layer-logic.md)).
[G-0452](../../work/gaps/G-0452-add-producer-to-consumer-data-flow-lens-to-wf-structural-sweep.md)
proposes a fourth lens for the sweep on the strength of a defect the existing
three provably could not have found.

What is ad hoc is absorption. Each finding lands as an individual gap and
then competes for attention inside an 82-item backlog against unrelated
work. Three of the four above carry a genuine decision (a display-width
change, an assumes-kind/discovers-kind asymmetry, and a determination of
whether the parsers' differences are load-bearing at all), so none of them
is a pure mechanical patch — which makes each easy to defer indefinitely and
easy to get wrong if picked up cold. A sweep's output has a coherence that
survives about as long as the session that produced it.

### Q4 — The history reads as bookkeeping

12% of commits touch Go; 73% touch planning markdown; 42% are a single verb,
`aiwf promote`. The arithmetic is unremarkable — 281 milestones times roughly
six ACs times four TDD phases — and it is the design working: one commit per
mutation is committed property #7, and the resulting audit trail is a
substantial part of what aiwf sells.

The cost is not the commit count. It is that `git log`, `git log --grep`, and
`git bisect` have all quietly stopped being useful for the question "what
changed in the code," and nothing offers a way past that.
[E-0073](../../work/epics/archive/E-0073-mutating-verb-ux-uniformity/epic.md) and
[M-0281](../../work/epics/archive/E-0073-mutating-verb-ux-uniformity/M-0281-same-state-mutating-verb-inputs-return-noop.md)
remove the *wasted* promotes (same-state inputs that currently commit a
no-op) but leave the structural volume untouched, correctly — that is not
what they are for.

The fix is read-side. A `--code-only` filter on `aiwf history` and a git
alias that excludes trailered planning commits buy back readability at zero
cost to the guarantee. The write-side alternative — batching an AC's four
phase-promotes into one commit — is the wrong trade twice over: it breaks
per-mutation atomicity, and it destroys the red→green timestamp separation
that is the entire evidentiary point of `tdd: required`.

### Q5 — The backlog sits at an equilibrium nobody chose

The open-gap count has held between 70 and 74 for six weeks, with roughly
25–30 gaps opened and 25–30 closed each week. 186 of 453 gaps carry a
`discovered_in` pointer — 144 to a milestone, 42 to an epic — so the
dominant discovery channel is implementation rather than planning.

None of that is obviously wrong. Discovery-during-implementation is what the
`discovered_in` field exists to record, and a stable backlog under matched
inflow and outflow is a system in balance rather than one accumulating debt.
The observation is narrower: 74 is a number that emerged, not one that was
chosen, and the `priority` field shipped in E-0066 is not currently being
used to make it mean anything. This is the weakest finding here and is
recorded as an open question rather than a target.

### Q6 — Nothing judges the specification going in

Q1 through Q5 all concern apparatus that runs over work already done. The
input to that work — the acceptance criterion, the gap, the milestone spec —
is judged by nothing mechanical at all.

What exists is presence-shaped: `entity-body-empty` decides that a section is
non-empty, `acs-body-coherence` that frontmatter and body headings agree.
Neither decides whether a criterion states an observable condition, whether a
gap is actionable, or whether a spec names what would falsify it. An
acceptance criterion reading *"the renderer handles edge cases correctly"*
satisfies every rule the kernel has.

The rule that acceptance criteria need mechanical evidence is real, but it is
evaluated at the promote — after implementation, when the criterion has
already shaped the work and the cost of rewriting it is highest. Per
[`../design/oracles.md`](../design/oracles.md)'s ladder, the defect is created
at planning time and caught several rungs down.

The TDD phase cadence is not the gap here and should not be confused with it.
Requiring a failing test before the code is a genuine during-work oracle with
real independence: a test that fails before the implementation exists cannot
have been shaped to fit it. What the cadence cannot do is notice that the test
and the criterion are about different things, because the criterion's intent
is not machine-readable. The machinery is sound; its input is unjudged.

Two directions, neither costed:

- **Structural.** Require the criterion to state its own falsification
  condition, and check that structurally. This reshapes an obligation that
  already exists — an acceptance criterion body is already required non-empty
  — rather than adding one, which matters given
  [`../design/growth.md`](../design/growth.md)'s finding about mandates.
- **Model-judged.** Apply the `wf-vacuity` move to the specification instead
  of the test. Vacuity asks whether a test can fail; the analogue asks whether
  a criterion can be satisfied by work that misses its point. If an adversarial
  agent easily writes a passing test that violates the intent, the criterion is
  underspecified — and that is learnable at authoring time for the cost of one
  agent call.

Two adjacent captures name acceptance criteria but ask a different question.
[`tdd-cycle-subagent-boundaries.md`](tdd-cycle-subagent-boundaries.md) targets
the discipline *around* a criterion — red-first ordering, promote-time
evidence, phase shape, AC presence — the cadence this finding already sets
aside above as sound machinery with unjudged input. E-0019 is execution
topology, deferred behind an unstabilized substrate; it reads AC prose only
to infer whether filesets are independent enough to parallelize. Neither
judges what a criterion says.

So this question has one other capture rather than two, and it is the one
carrying the cost these two directions lack:
[G-0583](../../work/gaps/G-0583-the-milestone-preflight-asks-for-judgment-with-no-method.md).
Run against M-0307, measuring the spec's factual claims caught three defects
before any code was written, and challenging each criterion found one that
could not be written without asserting that the guidance says what the test
says it says. The structural direction stays untaken, and stays the cheaper
answer wherever a criterion can be made to state its own falsification
condition.

## Scoped targets

Only the Q1 cluster is urgent; the rest are ordinary work whose sequencing is
a planning question, not a finding. Within that cluster the ordering is real
rather than preferential — item 2 depends on the error code its own gap pairs
with, and item 1 delivers signal without delivering trust. Everything from
item 4 down is independently actionable.

**Acute — tracked, unscheduled:**

1. **Q1, tourniquet — landed.**
   [G-0457](../../work/gaps/archive/G-0457-ci-gate-is-red-often-enough-to-carry-no-signal-about-the-change.md),
   addressed. One `wf-patch`: the nine scenarios whose oracles depend on
   timing or an observation window, plus `cmd/stresstest`'s whole-catalog and
   lock-kill runner tests, moved behind a `stress` build tag reachable via
   `make stress-tests`; `govulncheck` pinned and its findings split into a
   blocking dependency lane and a reporting stdlib lane; the toolchain pin
   single-sourced as one `GO_VERSION`. Push-path cost for the two stress
   packages fell from ~106s to ~23s. `-short` was rejected as the mechanism:
   it is a global switch that would also have disabled the binary
   integration tests across `cmd/aiwf` and `internal/cli`, which CI wants.
   Two holes the patch's review surfaced were closed with it — the blocking
   CVE lane failed open under GitHub's default `bash -e` (no `pipefail`)
   because a jq error was masked by a downstream `sort`, and no gate compiled
   tag-gated sources at all until `go vet -tags` joined the vet job.
2. **Q1, cure** —
   [G-0467](../../work/gaps/archive/G-0467-lock-busy-refusal-emits-an-empty-error-code.md)
   then
   [G-0468](../../work/gaps/archive/G-0468-stress-scenario-oracles-conflate-runner-contention-with-an-aiwf-defect.md).
   Give the repo-lock-busy refusal a machine-readable code, then split each
   scenario's hermetic correctness assertion from its timing assertion. This
   is what makes the harness trustworthy wherever it runs, and it unblocks
   G-0400. Larger than a `wf-patch`; scoping it may show it wants milestones.
   Item 1 without this item is a tourniquet mistaken for a cure.
3. **Q1, tier** —
   [G-0469](../../work/gaps/archive/G-0469-diff-scoped-coverage-gate-fires-only-in-ci-after-the-trunk-push-lands.md),
   move the diff-scoped coverage gate to a chokepoint that fires before the
   trunk push. Independent of the other three and small.

**File first — no tracking entity exists yet:**

4. **Q4** — a read-side history filter (`aiwf history --code-only` or
   equivalent, plus a documented git alias). Small, reversible, no kernel
   property at risk.

**Already tracked — depth over breadth, per Q2:**

5. [G-0253](../../work/gaps/G-0253-branch-coverage-audit-is-statement-scoped-not-per-arm-branch-coverage.md)
   — per-arm branch coverage. The largest single lift here and the one that
   most improves what a green gate is worth. Deferred once already, in
   [`tdd-cycle-subagent-boundaries.md`](tdd-cycle-subagent-boundaries.md),
   pending its relative priority becoming judgeable — this document is that
   judgement: it is the highest-value item in Q2.
6. [G-0110](../../work/gaps/G-0110-gremlins-diff-ref-filter-excludes-new-files-entirely-manual-mutation-review-needed-for-m-0094-95-96.md)
   — mutation testing's new-file blind spot.
7. [G-0317](../../work/gaps/G-0317-skill-edit-backstop-checks-test-references-path-not-asserts-changed-section.md)
   — assert-the-changed-section, not reference-the-path.
8. [G-0328](../../work/gaps/G-0328-golden-fixture-byte-identity-comparator-for-aiwf-check.md)
   — golden-fixture comparator for `aiwf check` output.

**Already tracked — the duplication queue, per Q3:**

9. [G-0448](../../work/gaps/G-0448-check-rule-list-split-across-two-dispatch-surfaces-no-single-source.md)
   — one rule registry. The highest-leverage of the four: every future check
   rule pays the "which of the two surfaces does this go in?" tax, and the
   answer is currently determined by function signature.
10. [G-0453](../../work/gaps/G-0453-unify-shorthash-short-sha-abbreviation-helpers-in-check-width-decision.md)
   /
   [G-0454](../../work/gaps/G-0454-unify-the-three-id-shape-parsers-in-entity-parseidnumber-vs-canonicalize.md)
   /
   [G-0455](../../work/gaps/G-0455-consolidate-markdown-heading-walk-state-machines-in-body-go-evaluate-first.md)
   — the current sweep output. G-0455 explicitly may close as won't-do; that
   is a legitimate outcome and the determination is the work, not the
   refactor.
11. [G-0452](../../work/gaps/G-0452-add-producer-to-consumer-data-flow-lens-to-wf-structural-sweep.md)
   — the sweep's fourth lens. Worth weighing *after* the current sweep's
   output is absorbed, not before: a wider net over an unabsorbed catch adds
   findings without adding closures.

**Blocked behind Q1:**

12. [G-0400](../../work/gaps/G-0400-stress-scenario-catalog-exercises-only-10-of-38-aiwf-verbs.md)
    — the stress catalog covers 10 of 38 verbs and should be wider. Widening
    it while it sits on the critical path of every push would multiply Q1b
    rather than help. The blocker is item 2, not item 1: widening a catalog
    whose oracles report contention as defect multiplies the flake surface
    wherever that catalog runs. Item 1 additionally leaves the drivers
    uncovered in the default lane (85.5% to 62.8%), so each tagged scenario
    G-0400 adds would meet the diff-scoped coverage gate with no covering
    test.

## Open design questions

Intentionally not answered here.

- **Should the real-binary stress scenarios run in CI at all?** A build-tag
  split inside `go.yml`, a separate `workflow_dispatch` job, and a scheduled
  nightly are three different answers with three different failure modes. The
  answer settles Q1b's shape and reconciles `CLAUDE.md` with the workflow.
  `flake-hunt.yml` is back in the option set: it was excluded for running a
  broad sweep these scenarios would be co-tenant with, and it now fans out
  one package per runner, which is the isolation they were measured to need.
  What it does not yet do is pass `-tags stress`, so adopting it is a change
  to that workflow rather than a placement that already exists — and it runs
  on dispatch before a tag, which answers a different question than "in CI".
- **Should `govulncheck` block on stdlib CVEs?** A dependency CVE is
  actionable on the spot; a stdlib CVE waits on a toolchain release the
  repository does not control. Blocking on both treats them as one class.
  Non-blocking on stdlib has its own cost — a real, exploitable stdlib
  finding would then only warn. A third answer is to stop pinning an exact
  patch version, so a fix arrives with the next patch release rather than
  with an edit, trading toolchain reproducibility for remediation latency.
  Weigh also that stdlib symbol results over-approximate through interface
  dispatch: of the four traces `GO-2026-5856` reported here, two reach TLS
  only via `io.Copy` and `io.WriteString` on arbitrary writers.
- **What should an uncoded verb error exit as?** `ExitUsage` is currently the
  default bucket, so an I/O failure deep inside a verb is indistinguishable
  from bad arguments, and the repo-lock-busy refusal shares that code and
  carries an empty envelope `code`. G-0467 holds the question; it is listed
  here because the answer is a kernel CLI-contract decision, not a test fix.
- **What is the ceiling on `internal/policies`?** At 16% of all Go and 66
  policies it is the largest package in the repo. A stated budget — even a
  soft one — gives every future "add a chokepoint" proposal something to
  argue against, and is the mechanism by which Q2's stance becomes more than
  a preference.
- **Does the gap backlog want a target?** Answering it decides whether the
  `priority` field becomes a working filter or stays decorative.
- **Is 6.7s the budget for `aiwf check`?** Down substantially from the ~19s
  recorded when the problem was last measured, following
  [E-0053](../../work/epics/archive/E-0053-make-aiwf-check-and-the-policies-test-suite-fast/epic.md).
  Whether that retires the structural concern or merely defers it belongs to
  [`check-performance-incremental-revwalk-cache.md`](check-performance-incremental-revwalk-cache.md),
  which holds the two failed design attempts and the correctness constraint
  any third must satisfy. Cited here so the split reads as a decision.

## Risks and boundaries

**Risk: reading this as a mandate for more gates.** It is the opposite. The
measured problem is that the gates already present do not report reliably
(Q1) and that several report less than their names claim (Q2). An initiative
that closes by adding policies 67 through 75 will have made the situation
worse on both axes.

**Risk: weakening real coverage while fixing Q1b.** The stress scenarios
found real bugs; that is why the harness exists. The fix is to move them off
the every-push path, not to delete or weaken them — and G-0400 wants the
catalog *wider*, which only becomes safe afterwards. A fix that quietly
reduces what the harness exercises has traded an acute problem for a silent
one. The measurement puts a number on this: the shipped split leaves
`internal/stresstest` at 62.8% statement coverage, down from 85.5%, and a
lane nobody reads exercises nothing at all.

**Risk: the tourniquet closing the file on Q1.** The placement change is
cheap, visible, and turns CI green, which makes it the natural stopping
point. It leaves every scenario oracle still reporting runner contention as
an aiwf defect, so the on-demand lane inherits the flake and becomes a lane
nobody trusts — the same failure this initiative documents, relocated. The
guard is that the cure is a tracked entity of its own (G-0468, on G-0467)
rather than a follow-up sentence inside the tourniquet's gap.

**Risk: Q4 drifting into a kernel change.** The read-side filter is small and
safe. Any proposal that reduces commit *count* is touching committed property
#7 and needs to be argued as such, on its own, not folded in under
"readability."

**Boundary: this is not a test-quality initiative.**
[E-0042](../../work/epics/archive/E-0042-burn-down-test-quality-debt-across-policies-and-the-test-corpus/epic.md)
burned down test-quality debt across the policy suite and corpus;
[E-0064](../../work/epics/archive/E-0064-backfill-test-coverage-for-untested-cli-verb-error-handling-branches/epic.md)
backfilled CLI error-branch coverage;
[E-0068](../../work/epics/archive/E-0068-mechanical-ac-milestone-completeness-guards/epic.md)
and
[E-0070](../../work/epics/archive/E-0070-ac-contract-first-discipline-plan-time-content-red-first-ordering/epic.md)
added AC-completeness and red-first guards. Those landed. This initiative
sits downstream of them and asks a different question: given all of that,
what does a CI result actually tell you?

**Boundary: `aiwf check` performance is the sibling initiative's.** See the
last open question above.

**Boundary: nothing here questions the audit trail.** Q4 is about reading it,
not reducing it.

## Desired future property

A red `go.yml` run means the change under test broke something, and a green
one means the gates that ran were strong enough for that to be worth
knowing. Separately and more modestly: a reader — human or agent — can ask
"what changed in the code" and get an answer, without the planning trail
that makes aiwf worth using getting in the way of the code history it exists
to explain.

## Provenance

Emerged from an open-work synthesis pass on 2026-07-27 that started as a
routine "what should I prioritise next?" question over the planning tree and
turned up the eleven-day red-CI window in the CI history rather than in the
tree — no gap, no epic, and no entity recorded it, because the entity surface
has no way to notice that its own gate was uninformative. The measured
baseline above was taken in that pass, from `git log`, `gh run list`, package
line counts, and direct timing of `aiwf check` and the `internal/stresstest`
package. Q1's causes were confirmed by reading the failing runs' logs and
`go.yml` rather than inferred from the failure rate.

Q1 was re-measured on 2026-07-30 in a pass that read every red run's
job-level and test-level failure, traced the stress failures into the
scenario classifiers and the CLI's error-envelope path, and reproduced the
timing behaviour locally under a four-core constraint. That pass resolved the
failure rate into three causes with distinct lifespans, found the coverage
gate as a third cause, and split Q1's remedy into a tourniquet (G-0457) and a
cure (G-0468, on G-0467), with the coverage gate's tier as G-0469.
