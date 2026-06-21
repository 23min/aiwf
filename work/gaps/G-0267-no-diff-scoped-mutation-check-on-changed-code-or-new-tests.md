---
id: G-0267
title: No diff-scoped mutation check on changed code or new tests
status: open
---
## What's missing

Mutation testing exists only as the whole-package, `workflow_dispatch`-only
`mutate-hunt` workflow — slow, manual, and deliberately not on push (mutation is
expensive and equivalent-mutant noise makes routine gating impractical). The
diff-scoped gate we *do* run automatically (`branch_coverage_audit`, G-0067)
measures **coverage** (the changed line ran), not **mutation** (an assertion on
that line actually catches a bug). `wf-vacuity` covers the mutation question by
hand, but it is LLM-judged and advisory.

So there is no mechanical, diff-scoped mutation signal: nothing that — when you
change production code or add/modify a test — checks that the touched code's
tests actually *kill* mutants. That empty cell (diff-scoped × mechanical ×
mutation) is the gap.

## Why it matters

This is **preventive** — it stops new vacuous tests/assertions from landing —
versus the M-0168 survivor backlog, which is **corrective** cleanup of old gaps
on stable code. Preventive is the higher-value investment and squarely in
E-0042's spirit: a mechanical chokepoint that catches vacuity going forward
rather than a one-time sweep. The diff-scoped coverage gate (G-0067) already
proves changed lines *run*; this proves the assertions on them can *fail* — the
precise gap between coverage and test strength that E-0042 cared about. It is
also foundation for E-0016 (the TDD-policy chokepoint): tests a TDD cycle
produces can be mutation-checked as part of "done".

## Shape

**v1 (lean, advisory — a `wf-patch`):** a `make mutate-diff` target that derives
the changed Go packages from `git diff` against `origin/main`, runs gremlins on
*just those* packages (not the whole kernel), and reports survivors. Advisory,
not a blocking gate — mutation is slow (the velocity reason `mutate-hunt` is
manual) and equivalent mutants make "0 survivors" un-gateable without human
triage. A test confirms it surfaces a planted survivor; one line wires it into
`wf-vacuity` as the mechanical companion to its manual probe. This naturally
handles "new test written": a test-only diff still changes its package, so
gremlins re-mutates that package's production code and the test either kills
more or it doesn't.

**v2 (YAGNI — deferred to a follow-up milestone, only if v1's package
granularity proves too noisy):** intersect survivors with the exact changed line
ranges, plus the before/after-baseline variant ("did this new test reduce the
package's survivor count").

Scope discipline is part of this gap: v1 stays a lean advisory target so it
ships before E-0016 without ballooning.
