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
> in materially fewer lines, or name the specific constraint that prevents
> it — an interface you do not control, a branch each arm genuinely needs,
> a schema invariant, a policy the repo enforces.

Two properties are load-bearing and easy to get wrong. It must be scoped to
**logic, not line count**: doc comments here are policy-enforced, and a
line-count reviewer would attack exactly the prose `CLAUDE.md` mandates. And
it must run at wrap, against a fresh-context reviewer that already holds the
bucketed numstat — not at the TDD cycle's refactor step, where the author is
minutes from having written the code and author-blindness is the whole
problem.

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
