---
id: M-056
title: Add --body-file to aiwf add variants
status: draft
parent: E-15
---

## Goal

Add a `--body-file` flag (also accepting `--body -` for stdin) on every `aiwf add` variant — epic, milestone, gap, decision, ADR, contract — so body content rides along with the create commit. Eliminates the separate body-content commit that today triggers `provenance-untrailered-entity-commit` warnings under the common "create entity, then flesh out body" workflow.

## Approach

The flag reads a file (or stdin) and writes its content as the body of the new entity file in the same atomic commit as the frontmatter creation. Absence of the flag preserves current behavior (empty body with the existing template's H2 sections). If the input file contains its own frontmatter block, the verb either strips it or refuses (decide during implementation; refusing is simpler and forces clean callers).

This is the highest-leverage milestone of the epic — it converts the most common multi-commit pattern (create + body edit) into a single commit.

## Acceptance criteria
