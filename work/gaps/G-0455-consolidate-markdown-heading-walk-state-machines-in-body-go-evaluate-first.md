---
id: G-0455
title: Consolidate markdown heading-walk state machines in body.go (evaluate first)
status: open
priority: low
---
## What's missing

`internal/entity/body.go` carries three-to-four near-identical markdown heading-walk state machines (the AC-section parse, `SectionLineBounds`, and the section iterators).

## Why it matters

Parsing state machines duplicated across a file are a maintenance and correctness hazard — a fix to heading detection must land in each copy. But this is the **highest-risk, least-certain** item split from G-0447: unifying parsers is exactly where subtle intentional differences (what counts as a heading, how fenced code or nested headings are handled) get flattened by a well-meaning merge.

## Resolution shape

**Evaluate first; do not assume a merge is warranted.** Read the walkers side-by-side and establish whether their differences are incidental (safe to unify behind one scanner) or load-bearing (each answers a genuinely different question). If incidental, extract one heading-scanner primitive with a property/golden test pinning the current behavior *before* refactoring. If load-bearing, close this gap as won't-do with the reason recorded — a legitimate outcome. Do not refactor without that determination.

## Where to fix

- `internal/entity/body.go` — the heading-walk loops and `SectionLineBounds`.

## Related

- G-0447 — the convergent-duplication cleanup this was split from (seam 5c).
- `wf-property-test` — the tool to pin the walker's behavior before any merge.
