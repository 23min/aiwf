---
id: G-0675
title: The section-set scan's corpus is hand-maintained and narrows in silence
status: open
discovered_in: M-0332
---
## What's missing

`sectionSetCorpus` in `internal/policies/section_set_single_source.go` is a
hand-maintained list of nine roots. The scan that keeps each kind's required
section set stated in one place reads only those roots, and nothing checks
that the list still covers what it is meant to.

Two failure shapes, measured by mutation against the live tree:

- A root **dropped** from the list narrows the scan with every test green.
  The firing fixtures pin two of the nine — the design-doc tree and the
  shipped-skill tree — so dropping either is caught; dropping any of the other
  seven is caught by nothing.
- A root **added to the repo** and not to the list is never scanned at all.
  A new normative design doc, or a fourth embedded tree, is outside the ban
  from the day it lands, and no failure marks the omission.

A root **renamed** is the one shape already covered: `TestSectionSetCorpusRootsExist`
fails when a listed root no longer resolves.

## Why it matters

The scan reports an absence, so a narrowed corpus and a clean tree are
indistinguishable from its output. That is the failure mode the scan itself
exists to prevent one step upstream — a reader trusting a clean verdict over
bytes nobody read — and it reappears in the scan's own configuration.

The list is also a third statement of the repo's documentation tiers, beside
the root `CLAUDE.md` and `internal/policies/m0128_documentation_hierarchy.go`.
Deriving it would fix both problems at once, and needs something that does not
exist: a tier-partitioned list of doc roots that a policy can read.
`documentationHierarchyNarrativeFiles` is the closest candidate and is
tier-mixed, so a naive derivation would widen the scan onto exploratory docs
where a table is a record of thinking rather than a claim about the kernel.

G-0092 owns the broader question of the tiering not drift-checking against the
tree. This gap is the narrower consequence: one policy's reach silently
depending on a list nobody is prompted to update.
