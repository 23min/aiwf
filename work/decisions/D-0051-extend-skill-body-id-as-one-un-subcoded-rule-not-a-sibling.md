---
id: D-0051
title: Extend skill-body-id as one un-subcoded rule, not a sibling
status: proposed
relates_to:
    - M-0287
    - E-0078
---
> **Date:** 2026-08-02 · **Decided by:** human/peter

## Question

M-0287 closes two holes in the `skill-body-id` rule: a scan that exempts code
constructs, and an unenforced placeholder-width claim. The second hole needs a
detector, and the question is what shape it takes — a sibling rule with its own
finding code, or an extension of the rule already walking that corpus.

A finding code is not free. Every code and subcode this rule emits is documented
in a shipped surface that materializes into consumer repos, where this rule is
structurally inert because the trees it scans do not exist there. Growth in the
taxonomy is therefore growth in documentation a consumer reads and can never act
on.

## Decision

Extend `skill-body-id` as a **single, un-subcoded** finding code covering both
shapes: a digit-bearing entity id, and a placeholder at any shape other than the
canonical letter-N form. One rule, one corpus walk, one hint, and no new row in
any shipped surface.

Every finding the rule reports is a **warning** until the shipped tree is swept;
the sweep milestone clears the tree and raises the severity as its last act. The
rule draws no severity distinction between its two shapes or between prose and
code placement.

The width detector **subsumes** the partial one that lived in
`TestSkillBodyID_PlaceholdersAreCanonical`, which is deleted. The production rule
now owns placeholder canonicality over a strictly larger corpus: every `*.md`
whole-file, frontmatter included, with code constructs in scope.

## Reasoning

The two shapes share everything expensive about a rule — corpus, walk, polarity,
inertness in a consumer repo, and the severity flip the sweep milestone
schedules. They also share their remediation, which is the point that decides
the shape: a real id becomes `<prefix>-NNNN` and a narrow placeholder becomes
`<prefix>-NNNN`. Both instructions are *write the canonical letter-N
placeholder*. The doc-link carve-out is an additional option on one shape, not a
different instruction, so one hint states the fix for both without hedging.

Subcodes were the obvious move given how common they are here — `body-prose-id`
carries seven, `refs-resolve` six — and they were rejected because the argument
for them turned out to be thinner than it first appeared. What they would buy is
diagnostic separation between the two shapes. What they cost is a row in the
`aiwf-check` skill's Findings table for a finding no consumer can ever see. The
message already names the offending token and the defect, so the reader can tell
the shapes apart without the taxonomy carrying that weight.

A sibling rule loses on cost without buying separation anywhere that matters. It
would add a constant to the closed set, walk the same twenty-nine files a second
time, need its own real-tree test, and split the sweep milestone's single
severity flip into two edits that must land together.

Uniform warning severity is the same economy applied to the staging window. A
rule that kept error severity for the one shape with no outstanding sites would
buy a narrow guarantee — a stray real id in prose still blocking a push before
the sweep — and pay for it with a per-token severity function, a byte-range
comparison helper, and a second parse of every file to produce the narrower mask
that comparison needs. The guarantee is worth less than the machinery: the
window is one milestone, and the sweep is what makes the property true anyway.

Subsuming the existing width test rather than adding beside it follows the
single-source-of-truth force: two implementations of "is this placeholder
canonical" over overlapping corpora drift, and the narrower one was already
wrong in three ways — it read only `SKILL.md`, only post-frontmatter, and
through the mask that hides the code-construct cases.

## Consequences

- The rule's two shapes are distinguishable only by the defect its message
  names, since they share a code, a severity, and a remediation. Any test that
  asserts classification must assert on the message; asserting that "something
  fired" cannot tell them apart.
- The two whole-tree real-tree assertions skip until the sweep lands. The
  property they state is deliberately false while detection ships ahead of
  cleanup, and a skip says so where a severity filter would have passed against
  an empty set while reading like a gate.
- Nothing guards the shipped surfaces whole-tree during that window. A narrower
  real-tree assertion over the keep-list files is the exception.
