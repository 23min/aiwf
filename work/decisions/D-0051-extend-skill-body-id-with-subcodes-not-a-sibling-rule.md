---
id: D-0051
title: Extend skill-body-id with subcodes, not a sibling rule
status: proposed
relates_to:
    - M-0287
    - E-0078
---
> **Date:** 2026-08-02 · **Decided by:** human/peter

## Question

M-0287 closes two holes in the `skill-body-id` rule: a scan that exempts code
constructs, and an unenforced placeholder-width claim. The second hole needs a
detector, and its remediation reads nothing like the first's — "widen the
placeholder" and "stop citing a real id" are different instructions to a
different reader.

So: does the width detector extend the existing rule, or ship as a sibling with
its own finding code? M-0287 posed this as a binary, and the answer turns on a
third shape neither arm named.

## Decision

Extend `skill-body-id`, and distinguish the two behaviors by **subcode**:
`real-id` for a digit-bearing entity id, `narrow-placeholder` for a letter-N
placeholder below canonical width. One finding code, one corpus walk, one
severity flip when the sweep completes; the remediation text varies per subcode
through the hint table, which is the axis that actually differs.

Severity follows the detected class rather than the rule as a whole. A real id
in **prose** keeps error severity — it has no outstanding sites, so preserving
it blocks no push. The two newly-detected classes land at **warning** and flip
to error as the last act of the sweep milestone.

The width detector **subsumes** the partial one already living in
`TestSkillBodyID_PlaceholdersAreCanonical`. That test keeps its real-tree
assertion but reads the production rule's output instead of re-deriving the
property.

## Reasoning

The two behaviors share everything expensive about a rule — corpus, walk,
polarity, inertness in a consumer repo, and the severity flip the sweep
milestone schedules — and differ only in remediation text. Subcodes exist to
vary exactly that axis, and the convention is already dominant in this codebase:
`body-prose-id` carries seven, `refs-resolve` six, `acs-shape` five. Both
discoverability policies resolve `code/subcode` natively, so the shape costs two
hint rows and two skill table rows.

A sibling rule loses on cost without buying separation anywhere that matters. It
would add a constant to the closed set, walk the same twenty-nine files a second
time, need its own real-tree test, and — the deciding cost — split the sweep
milestone's single severity flip into two edits that must land together. The
epic's constraint is written assuming one flip.

Extending without subcodes was rejected for the reason the milestone anticipated:
a single hint would have to instruct two unrelated repairs, which is how a hint
stops being read.

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

- The existing bare-code `skill-body-id` findings gain a subcode, which changes
  the JSON envelope's `subcode` field from absent to populated. No consumer
  contract pins its absence.
- The sweep milestone's flip is a severity change on two subcodes in one place,
  not a rule rewrite.
- `TestSkillBodyID_PlaceholdersAreCanonical` stops carrying its own width regex;
  losing that duplicate is the point, and the real-tree property it asserts
  survives unchanged.
