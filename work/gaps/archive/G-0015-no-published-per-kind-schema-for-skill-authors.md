---
id: G-0015
title: No published per-kind schema for skill authors
status: addressed
addressed_by_commit:
  - 0ba0e61
---

Resolved in commit `0ba0e61` (fix(aiwf): G15 — add 'aiwf schema' verb, single source of truth for entity schemas). Took the proposed approach: a new read-only `aiwf schema [kind]` verb prints the per-kind frontmatter contract — id format, allowed statuses, required and optional fields, and reference fields with cardinality and allowed target kinds — in text or JSON envelope. The verb reads from `entity.SchemaForKind`, which is now the single source of truth that also drives `entity.AllowedStatuses`, `entity.IDFormat`, and (pinned by `TestSchemaMatchesCollectRefs`) the allowed-kinds table consulted by `check.refsResolve`. Skill authors and AI-driven scaffolding tooling can now consume the schema programmatically (`aiwf schema --format=json --pretty`) instead of guessing at field names. Coverage: 100% on `SchemaForKind` / `AllSchemas`; 84.8% on the verb's main and 71.9% on its text renderer (the missing branches are defensive io.Writer error returns).

---
