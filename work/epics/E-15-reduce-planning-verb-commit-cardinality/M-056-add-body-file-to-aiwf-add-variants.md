---
id: M-056
title: Add --body-file to aiwf add variants
status: in_progress
parent: E-15
acs:
    - id: AC-1
      title: --body-file flag on aiwf add for all six kinds
      status: met
    - id: AC-2
      title: Body content from file replaces empty body in created entity
      status: met
    - id: AC-3
      title: --body-file is optional; absence preserves current behavior
      status: met
    - id: AC-4
      title: Body and frontmatter committed together in single create commit
      status: met
    - id: AC-5
      title: Subprocess integration test exercises --body-file path
      status: open
---

## Goal

Add a `--body-file` flag (also accepting `--body -` for stdin) on every `aiwf add` variant — epic, milestone, gap, decision, ADR, contract — so body content rides along with the create commit. Eliminates the separate body-content commit that today triggers `provenance-untrailered-entity-commit` warnings under the common "create entity, then flesh out body" workflow.

## Approach

The flag reads a file (or stdin) and writes its content as the body of the new entity file in the same atomic commit as the frontmatter creation. Absence of the flag preserves current behavior (empty body with the existing template's H2 sections). If the input file contains its own frontmatter block, the verb either strips it or refuses (decide during implementation; refusing is simpler and forces clean callers).

This is the highest-leverage milestone of the epic — it converts the most common multi-commit pattern (create + body edit) into a single commit.

## Acceptance criteria

### AC-1 — --body-file flag on aiwf add for all six kinds

### AC-2 — Body content from file replaces empty body in created entity

### AC-3 — --body-file is optional; absence preserves current behavior

### AC-4 — Body and frontmatter committed together in single create commit

### AC-5 — Subprocess integration test exercises --body-file path

