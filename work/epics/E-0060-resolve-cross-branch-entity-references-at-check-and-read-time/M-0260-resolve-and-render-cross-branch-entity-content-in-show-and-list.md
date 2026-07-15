---
id: M-0260
title: Resolve and render cross-branch entity content in show and list
status: draft
parent: E-0060
depends_on:
    - M-0259
tdd: required
acs:
    - id: AC-1
      title: Resolve content via BlobReader using the recorded cross-branch ref
      status: open
      tdd_phase: red
    - id: AC-2
      title: Cross-branch-sourced content is visibly labeled
      status: open
      tdd_phase: red
    - id: AC-3
      title: Refuses to pick a ref when content diverges
      status: open
      tdd_phase: red
    - id: AC-4
      title: No working-tree, index, or ref writes
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — Resolve content via BlobReader using the recorded cross-branch ref

`aiwf show` and `aiwf list` resolve an entity's content by reading
`<ref>:<path>` via `gitops.BlobReader`, using the ref M-0259/AC-1 recorded
for a `cross-branch-pending` id that misses the local working tree.
Strictly read-only — no working-tree, index, or ref writes at any point,
per the epic's "resolution is always a live read against the other ref at
the point of use" constraint.

Evidence: fixture test — an id present only on a sibling local branch;
`aiwf show <id>` renders that branch's content without touching the
working tree, index, or refs.

### AC-2 — Cross-branch-sourced content is visibly labeled

Rendered output (both `aiwf show` and `aiwf list`) marks content sourced
from another ref distinctly — never presented indistinguishably from a
locally-resolved entity, per ADR-0030's Consequences section. Exact label
text/placement is an implementation detail decided during this milestone;
the requirement is visibility, not a specific string.

Evidence: fixture test asserting the rendered output for a cross-branch-
sourced entity differs observably (a label, a field, a distinct rendering
mode) from the same entity rendered locally.

### AC-3 — Refuses to pick a ref when content diverges

When the id is classified `cross-branch-collision` (M-0259/AC-3) rather
than `cross-branch-pending`, `aiwf show`/`aiwf list` do not arbitrarily
render one ref's content as if canonical. They surface the ambiguity
explicitly instead — naming the candidate refs and declining to render
body content — leaving resolution to whichever branch merges or
reconciles first. Resolves G-0415's read-side half of the multiplicity
gap: silently picking a ref would present ambiguous, possibly-wrong
content as if it were authoritative.

Evidence: fixture test — two local branches hold divergent content at the
same id; `aiwf show <id>` reports the ambiguity (naming both refs) rather
than picking one side's content.

### AC-4 — No working-tree, index, or ref writes

Every code path this milestone adds is read-only under every
classification (local resolution, cross-branch-pending, or
cross-branch-collision): no `git checkout`, no merge, no working-tree
write, no index write, no ref write.

Evidence: an integration test asserting the repository's working tree,
index, and refs are byte-identical before and after an `aiwf show`/`aiwf
list` invocation that resolves cross-branch content.

