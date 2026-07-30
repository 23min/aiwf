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

## Findings

### Q1 — The gate is red often enough that a red gate carries no information

Acute, and the only finding here with a deadline attached in the sense that
every push until it lands pays for it. Two independent causes, both live.

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

**Q1b — The stress harness runs inside the default test step.**
`go.yml`'s test job runs `go test … ./...`, which includes
`internal/stresstest`. That package's real-binary scenarios launch concurrent
`aiwf` subprocesses against disposable git repos; they take 34s locally and
77s in CI, and they are timing-shaped by construction.
`TestConcurrentIDAllocationScenario_RealBinary_NConcurrentActorsAllGetDistinctIDs`
failed the three most recent red runs with *"only 7/8 concurrent actors
succeeded — expected all to serialize successfully within repolock's
timeout"* — a runner-contention symptom, not a defect the test was written
to catch.

`CLAUDE.md` describes the harness as *"dev-only tooling, never installed
alongside `cmd/aiwf`, run by hand rather than scheduled or wired into
`make ci`."* The workflow disagrees with the stated contract. Whichever way
that reconciles, the two should say the same thing.

The seam for a fix already exists in the package's own structure: each
scenario is split into a `*_classify_test.go` file pinning the pure decision
function against fabricated outcomes (fast, deterministic, genuinely
valuable on every push) and a `*_test.go` file driving real subprocesses
(slow, timing-shaped, valuable on demand). The split to make is the one the
package already made.

**Why this is Q1 and not Q2.** Every other finding here is about making a
gate stronger. This one is about making any gate mean anything at all. It is
also the cheapest: one `wf-patch`, no design decision beyond the two
questions in *Open design questions* below.

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
[E-0073](../../work/epics/E-0073-mutating-verb-ux-uniformity/epic.md) and
[M-0281](../../work/epics/E-0073-mutating-verb-ux-uniformity/M-0281-same-state-mutating-verb-inputs-return-noop.md)
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

## Scoped targets

Each is independently actionable. Only the first is urgent; the rest are
ordinary work whose sequencing is a planning question, not a finding.

**File first — no tracking entity exists yet:**

1. **Q1** — restore gate signal integrity. One `wf-patch`: gate the
   real-binary stress scenarios behind a build tag or `-short` while keeping
   the `*_classify_test.go` decision tests in the default run; pin
   `govulncheck`; separate the stdlib-CVE lane from the dependency-CVE lane
   so a disclosure the repo cannot act on does not mask one it can. Reconcile
   `CLAUDE.md`'s stress-harness contract with whatever `go.yml` ends up
   doing.
2. **Q4** — a read-side history filter (`aiwf history --code-only` or
   equivalent, plus a documented git alias). Small, reversible, no kernel
   property at risk.

**Already tracked — depth over breadth, per Q2:**

3. [G-0253](../../work/gaps/G-0253-branch-coverage-audit-is-statement-scoped-not-per-arm-branch-coverage.md)
   — per-arm branch coverage. The largest single lift here and the one that
   most improves what a green gate is worth. Deferred once already, in
   [`tdd-cycle-subagent-boundaries.md`](tdd-cycle-subagent-boundaries.md),
   pending its relative priority becoming judgeable — this document is that
   judgement: it is the highest-value item in Q2.
4. [G-0110](../../work/gaps/G-0110-gremlins-diff-ref-filter-excludes-new-files-entirely-manual-mutation-review-needed-for-m-0094-95-96.md)
   — mutation testing's new-file blind spot.
5. [G-0317](../../work/gaps/G-0317-skill-edit-backstop-checks-test-references-path-not-asserts-changed-section.md)
   — assert-the-changed-section, not reference-the-path.
6. [G-0328](../../work/gaps/G-0328-golden-fixture-byte-identity-comparator-for-aiwf-check.md)
   — golden-fixture comparator for `aiwf check` output.

**Already tracked — the duplication queue, per Q3:**

7. [G-0448](../../work/gaps/G-0448-check-rule-list-split-across-two-dispatch-surfaces-no-single-source.md)
   — one rule registry. The highest-leverage of the four: every future check
   rule pays the "which of the two surfaces does this go in?" tax, and the
   answer is currently determined by function signature.
8. [G-0453](../../work/gaps/G-0453-unify-shorthash-short-sha-abbreviation-helpers-in-check-width-decision.md)
   /
   [G-0454](../../work/gaps/G-0454-unify-the-three-id-shape-parsers-in-entity-parseidnumber-vs-canonicalize.md)
   /
   [G-0455](../../work/gaps/G-0455-consolidate-markdown-heading-walk-state-machines-in-body-go-evaluate-first.md)
   — the current sweep output. G-0455 explicitly may close as won't-do; that
   is a legitimate outcome and the determination is the work, not the
   refactor.
9. [G-0452](../../work/gaps/G-0452-add-producer-to-consumer-data-flow-lens-to-wf-structural-sweep.md)
   — the sweep's fourth lens. Worth weighing *after* the current sweep's
   output is absorbed, not before: a wider net over an unabsorbed catch adds
   findings without adding closures.

**Blocked behind Q1:**

10. [G-0400](../../work/gaps/G-0400-stress-scenario-catalog-exercises-only-10-of-38-aiwf-verbs.md)
    — the stress catalog covers 10 of 38 verbs and should be wider. Widening
    it while it sits on the critical path of every push would multiply Q1b
    rather than help. Genuinely valuable once the harness is off that path.

## Open design questions

Intentionally not answered here.

- **Should the real-binary stress scenarios run in CI at all?** A build-tag
  split inside `go.yml`, a separate `workflow_dispatch` job, and a scheduled
  nightly are three different answers with three different failure modes. The
  answer settles Q1b's shape and reconciles `CLAUDE.md` with the workflow.
- **Should `govulncheck` block on stdlib CVEs?** A dependency CVE is
  actionable on the spot; a stdlib CVE waits on a toolchain release the
  repository does not control. Blocking on both treats them as one class.
  Non-blocking on stdlib has its own cost — a real, exploitable stdlib
  finding would then only warn.
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
one.

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
package. Q1's two causes were confirmed by reading the failing runs' logs and
`go.yml` rather than inferred from the failure rate.
