---
id: M-059
title: Add resolver-pointer flags to status-transition verbs
status: draft
parent: E-15
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
