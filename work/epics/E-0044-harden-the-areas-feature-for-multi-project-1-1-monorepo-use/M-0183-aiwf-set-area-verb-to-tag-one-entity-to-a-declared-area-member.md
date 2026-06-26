---
id: M-0183
title: aiwf set-area verb to tag one entity to a declared area member
status: draft
parent: E-0044
tdd: required
---
## Goal

Add `aiwf set-area <id> <member>`: a verb that points a single entity at an existing declared area member, in one trailered commit. It is the guaranteed remediation for `areas.required` (M-0178) — when the knob flags an untagged entity, `set-area` is the one-command unblock — and a generally useful retag operation independent of the knob.

## Context

Today `--area` is creation-time only (`aiwf add --area`); no verb tags or retags an entity after creation. Hand-editing the `area:` frontmatter trips the `provenance-untrailered-entity-commit` audit, and `rename-area` (M-0177) renames a *member* across the whole tree — it cannot tag a single untagged entity. So an operator who enables `areas.required: true` on a tree with any untagged entity has no clean path to clear the resulting blocking finding. `set-area` closes that gap, mirroring the single-entity frontmatter-edit + trailer shape of `aiwf retitle`.

This is the inverse-blast-radius sibling of `rename-area`: `rename-area` changes the *vocabulary* (a member's name, carrying every referrer atomically); `set-area` changes one entity's *membership* against a fixed vocabulary.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- `aiwf set-area <id> <member>` rewrites the entity's `area:` frontmatter to `<member>` in a single commit; other entities are untouched.
- The commit carries `aiwf-verb: set-area` + `aiwf-entity:` + `aiwf-actor:`; `aiwf history <id>` renders the retag.
- Refuses an unknown id, an undeclared `<member>` (naming the declared set), a milestone target (area derives from the parent epic), and a no-op (already tagged `<member>`) — clear error, no write.
- Reverses via the same verb (`set-area <id> <original-member>`).
- Ships tab-completion (`<id>` to entity ids, `<member>` to declared members), `--help`, and skill-coverage via a `skillCoverageAllowlist` entry.

## Constraints

- Atomic: the single entity rewrite lands or nothing does — one commit, abort-before-commit on any validation failure.
- Single source of truth: `<member>` must already be declared in `aiwf.yaml: areas.members`; the verb never invents a member (that is `rename-area`'s and config's job).
- "What undoes this?" — the same verb with the prior member; documented at design.
- Provenance: a single target entity makes the verb authorized-AI-eligible — routed through the scope-gated finish with `VerbAct` and the entity as the target (the inverse of `rename-area`'s human-only posture). Pinned by a regression test.

## Out of scope

- Untagging (clearing `area` back to empty) — no use case yet; `required:true` forbids the empty state and a mis-tag is fixed by setting the correct member. Add later if a real need appears (YAGNI).
- Renaming a member or mutating `aiwf.yaml` — that is `rename-area` (M-0177).
- Setting an area on a milestone or acceptance criterion — they derive from the parent epic; the verb refuses and points at the epic.

## Dependencies

- None. Independent Tier-0; sequenced before M-0178 (the `areas.required` knob depends on this verb as its remediation path).

## Design notes

- Mirror `aiwf retitle`'s single-entity frontmatter-edit + trailer-stamp shape (one write for the target entity, `aiwf-verb: set-area`).
- Reuse the declared-member validation the `area-unknown` check and `rename-area` already apply.

## References

- `aiwf rename-area` (M-0177) — the vocabulary-rename sibling; this is the membership-change counterpart.
- `aiwf retitle` — the precedent for a single-entity frontmatter edit + trailer stamp.
- `internal/check/area_unknown.go` — the declared-member validation reused for `<member>`.
- M-0178 — the `areas.required` knob whose remediation path this verb is.
- ADR-0006 — skills policy (allowlist / "--help suffices" case).
