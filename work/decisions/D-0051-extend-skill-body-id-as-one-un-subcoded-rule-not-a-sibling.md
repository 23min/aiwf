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
shapes: a digit-bearing entity id, and a letter-N placeholder below canonical
width. One rule, one corpus walk, one hint, one severity flip when the sweep
completes, and no new row in any shipped surface.

Severity follows the detected class rather than the rule as a whole. A real id
in **prose** keeps error severity — it has no outstanding sites, so preserving it
blocks no push. The classes newly reachable once code constructs are in scope
land at **warning** and flip to error as the last act of the sweep milestone.

The width detector **subsumes** the partial one already living in
`TestSkillBodyID_PlaceholdersAreCanonical`. That test keeps its real-tree
assertion but reads the production rule's output instead of re-deriving the
property.

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
message already names the offending token, so the reader can tell the shapes
apart without the taxonomy carrying that weight.

A sibling rule loses on cost without buying separation anywhere that matters. It
would add a constant to the closed set, walk the same twenty-nine files a second
time, need its own real-tree test, and split the sweep milestone's single
severity flip into two edits that must land together.

The severity split is not a hedge. Applying the "lands at warning" constraint
bluntly would demote a live error-severity guarantee for the length of a
milestone in exchange for nothing — the class it protects has zero outstanding
sites. The constraint exists so an incomplete sweep cannot block a push, and
preserving error on an already-clean class blocks none.

Subsuming the existing width test rather than adding beside it follows the
single-source-of-truth force: two implementations of "is this placeholder
canonical" over overlapping corpora drift, and the narrower one is already wrong
in three ways — it reads only `SKILL.md`, only post-frontmatter, and through the
same mask that hides the code-construct cases.

## Consequences

- The severity split is scaffolding with a defined lifetime, not a permanent
  feature. Its only job is to let detection land before the sweep that clears
  the tree. When the sweep completes and severity flips, the prose/code
  distinction stops meaning anything and the machinery implementing it should be
  deleted rather than left standing — that deletion belongs to the sweep
  milestone, alongside the flip.
- `TestSkillBodyID_PlaceholdersAreCanonical` stops carrying its own width regex;
  losing that duplicate is the point, and the real-tree property it asserts
  survives unchanged.
- The rule's two real-tree assertions filter to error severity while the sweep is
  outstanding. That is a real, temporary reduction in strictness: a newly
  introduced code-construct citation warns rather than failing the suite. It ends
  when the flip lands, at which point the same assertions become whole-tree zero
  gates again with no edit.
