---
id: M-059
title: Add resolver-pointer flags to status-transition verbs
status: done
parent: E-15
acs:
    - id: AC-1
      title: aiwf promote G-NNN addressed --by accepts entity ids and commit shas
      status: met
    - id: AC-2
      title: aiwf promote ADR-NNNN superseded --superseded-by accepts ADR ids
      status: met
    - id: AC-3
      title: Verb writes resolver field atomically with status change
      status: met
    - id: AC-4
      title: Hand-editing frontmatter never required to satisfy resolver checks
      status: met
---

## Goal

Extend `aiwf promote` (and any other mutating verb that drives a status transition requiring a resolver/successor pointer) to accept the corresponding `--by` / `--by-commit` / `--superseded-by` flag, so users can satisfy the matching check rule via the verb route. Eliminates the current pattern where hand-editing frontmatter is the only way to satisfy `gap-resolved-has-resolver` and analogous rules, and stops users from defaulting to `wontfix` in place of the semantically-correct `addressed`. Closes G-053.

## Approach

For each status transition that has a corresponding resolver/pointer field check:

1. Identify the field the check rule looks for.
2. Add a verb flag that accepts the matching value type — entity id, commit sha, or both via shape detection.
3. The verb writes the field into projected frontmatter, then validates the projection against the standard ruleset before committing (same atomic-commit model as today).

Concrete cases on day one: `gap → addressed` (`--by <id>` for entity-resolver, `--by-commit <sha>` for commit-resolver), `adr → superseded` (`--superseded-by <ADR-id>`). Implementation should generalize so future pointer-requiring transitions only need to register the flag and the field name — not duplicate the wiring per verb.

## Acceptance criteria

### AC-1 — aiwf promote G-NNN addressed --by accepts entity ids and commit shas

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-059/AC-1` for the actual implementation history._

### AC-2 — aiwf promote ADR-NNNN superseded --superseded-by accepts ADR ids

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-059/AC-2` for the actual implementation history._

### AC-3 — Verb writes resolver field atomically with status change

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-059/AC-3` for the actual implementation history._

### AC-4 — Hand-editing frontmatter never required to satisfy resolver checks

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-059/AC-4` for the actual implementation history._
