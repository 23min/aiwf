---
id: G-0650
title: Wrap review never asks whether the implementation could be materially smaller
status: open
---
## What's missing

`aiwfx-wrap-milestone` step 2 already tells the reviewer to measure the
change's shape before judging it — derive `git diff --numstat`, bucket the
rows by role, then answer three questions: recurring obligation, deletions,
same-outcome clusters. Its stated rationale is that "every other check at
wrap asks whether something is missing; none asks whether something is
unnecessary".

None of those three reaches the case the rationale describes. "Deletions"
asks what was retired. "Same-outcome clusters" catches redundant tests
grouped by claimed outcome. Neither asks whether the surviving
implementation could produce the same result with materially less logic,
so an oversized-but-non-duplicative unit passes all three.

The missing question is constructive rather than evaluative, and that is
what makes it work: asking "could this be shorter" invites "no", while
asking for a working shorter version yields either an artefact or a named
constraint. A usable form, scoped to logic:

> Take the largest logic bucket. Write a version that does the same thing
> in **half** the lines. Apply it in a scratch copy, run the gates, and
> report what actually broke. If you cannot reach half, report how far you
> got and name the specific constraint that stops you — an interface you do
> not control, a branch each arm genuinely needs, a schema invariant, a
> policy the repo enforces. Comments the policy mandates, tests pinning
> distinct rules, and planning prose are out of scope and do not count
> toward the target.

Four properties are load-bearing, and each addresses a way the question
degrades.

**Half, not "materially fewer".** A hedged target is satisfied by a five
percent trim, reported as success. Half is deliberately unreachable by
tightening, so it forces the structural reading — two things that should be
one — which is where the value was. A failed attempt is still output:
"reached seventy percent, the rest is irreducible because X" is a finding.
The number is a provocation, not a threshold to clear.

**Scoped to logic, not line count.** Doc comments here are policy-enforced
and were the single largest bucket in the run below. A lens told to shrink
the change, without the bucket split in front of it, attacks exactly the
prose `CLAUDE.md` mandates.

**Measured, not proposed.** The trial reported "still passing: X, Y, Z" —
a prediction. Requiring the cut to be applied and the gates run converts
each one into evidence, kills the plausible-but-wrong reduction, and
self-limits volume, since a reviewer who must run each cut proposes fewer
and better ones.

**The escape hatch stays typed.** A lens told to find a cut will find one.
The named-constraint alternative is the only protection, and it works
because it demands a specific shape. Loosened to free text it yields
rationalisations.

It must also run at wrap, against a fresh-context reviewer that already
holds the bucketed numstat — not at the TDD cycle's refactor step, where the
author is minutes from having written the code and author-blindness is the
whole problem.

## Why it matters

Run once as a third lens during M-0324's wrap, alongside the standard
code-quality and design-quality passes, it found three things neither
standard lens reported:

- `sovereignActsSectionText` re-implemented `loadAuditCatalog`, a helper in
  the same package — a plain reuse-over-duplication miss.
- Two tests contributing zero incremental coverage, demonstrated by running
  the unit under test at 100% statement coverage without them.
- A rule whose emission logic cost roughly 75 lines to constrain and which
  produced byte-identical output for every input the live closed set can
  supply.

It also declined two available cuts, citing the repo's own seam-and-layer
rule and its comment-content mandate, which is what makes the rest
credible. Acting on its findings removed about 100 lines net from the
milestone with the gates staying green.

Without the question standing somewhere, that pass happens when someone
thinks to ask for it. The wrap review is the only place in the workflow
where a fresh reader holds the whole change-set with its shape already
measured, so it is the cheapest place for the question to live and the only
one where it is answered by someone with no authorship attachment.

The addition should be a fourth question in the existing brief rather than a
new skill: the reviewer, the independence, and the bucketed numstat the
question needs are all already there, and a standalone skill would be a
per-subject mandate carried forever.

## The second question — is anything built over-guarded

The same stance, applied to a different substance. The compression question
asks whether logic is duplicated; this one asks whether a branch never
fires. Every other lens asks whether a guard is *missing*; nothing asks the
inverse, and a defensive branch is neither duplication nor a large bucket,
so the compression question does not reach it either.

> For each guard the change adds, name the caller that can produce the state
> it catches. If a caller can, the guard stays and the question is closed.
> If none can — and especially where a test or an invariant forbids it —
> say so explicitly: the guard, and the test the branch-coverage rule
> requires for it, are both waste. Removal needs the unreachability
> demonstrated, not assumed. Treat every annotation the change adds marking
> a branch unreachable as a claim to be checked, not as an answer.

**It defaults the opposite way to the compression question, and getting
that backwards turns a review lens into a safety regression.** Compression
defaults to cut-unless-constrained: a wrongly-kept helper costs lines. This
one defaults to keep-unless-proven-dead: a wrongly-removed guard costs a
crash. "I could not think of a caller" is not a demonstration.

**The cost of an unnecessary guard is doubled wherever the branch-coverage
rule is in force**, which is what makes the question worth more here than it
would be generically. That rule requires a shipped defensive path to carry a
test, so an unnecessary guard buys a branch that cannot fire plus a test
constructing a state production cannot reach. Both are carried forever.

Two instances from the trial run, one in each direction, which is why the
question forces an argument rather than presuming its conclusion. A guard
against an unknown entity kind in the sovereign-act regex builder cannot
fire while the closed-set invariant test passes, because that test validates
every entry against the FSM and an unknown kind fails validation — and an
independent reviewer measured its subtest asserting nothing, so neutering
the guard left the package green. Against that, a same-status branch in the
cancel verb is annotated unreachable on a mathematical argument and should
stay: the argument rests on a helper's behaviour that could change without
anyone revisiting the site.

## Both rules must ship, so both must be project-agnostic

These are review discipline rather than aiwf operation, and the wrap ritual
that would carry them materializes into every consumer repo. So the rule
text reaches projects that share none of this repo's tooling.

The annotation this repo uses to mark a branch unreachable is internal to
it: the coverage audit that consumes it is repo-development tooling and the
marker appears in no shipped surface. Naming that spelling in the rule would
cite a mechanism a consumer does not have.

The precedent to follow already exists and solves exactly this. The
branch-coverage rule ships in the TDD-cycle ritual and states the same
obligation generically — unreachable branches are documented *where the
project records such things* — without naming any marker. Both questions
above are written to that discipline: they name the category, never the
implementation.

The quoted blocks are the text that ships. The evidence around them in this
gap is not; it stays here, where a reader deciding whether the rules earn
their place can weigh it.
