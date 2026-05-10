---
id: M-0056
title: Add --body-file to aiwf add variants
status: done
parent: E-0015
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
      status: met
---

## Goal

Add a `--body-file` flag (also accepting `--body -` for stdin) on every `aiwf add` variant — epic, milestone, gap, decision, ADR, contract — so body content rides along with the create commit. Eliminates the separate body-content commit that today triggers `provenance-untrailered-entity-commit` warnings under the common "create entity, then flesh out body" workflow.

## Approach

The flag reads a file (or stdin) and writes its content as the body of the new entity file in the same atomic commit as the frontmatter creation. Absence of the flag preserves current behavior (empty body with the existing template's H2 sections). If the input file contains its own frontmatter block, the verb either strips it or refuses (decide during implementation; refusing is simpler and forces clean callers).

This is the highest-leverage milestone of the epic — it converts the most common multi-commit pattern (create + body edit) into a single commit.

## Acceptance criteria

### AC-1 — --body-file flag on aiwf add for all six kinds

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-056/AC-1` for the actual implementation history._

### AC-2 — Body content from file replaces empty body in created entity

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-056/AC-2` for the actual implementation history._

### AC-3 — --body-file is optional; absence preserves current behavior

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-056/AC-3` for the actual implementation history._

### AC-4 — Body and frontmatter committed together in single create commit

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-056/AC-4` for the actual implementation history._

### AC-5 — Subprocess integration test exercises --body-file path

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-056/AC-5` for the actual implementation history._
