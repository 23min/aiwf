---
id: M-0179
title: paths per-area config evolution with backward-compatible unmarshaler
status: draft
parent: E-0044
tdd: required
---
## Goal

Evolve `config.Areas` from a flat label list to label+location — `members: [{name, paths}]` — via a backward-compatible custom unmarshaler that still accepts the legacy `members: [app-a, app-b]` string form. This is the keystone: it gives each area the path oracle that the bijection check and all of Tier 2 depend on.

## Context

The area is label-only today; nothing ties it to where the project's code lives. Without that anchor the kernel can neither verify a tag nor derive one. This milestone adds the optional `paths:` glob per member while preserving zero-migration for every existing string-form config. Everything downstream — bijection coverage, mistag detection, auto-derive — is dead without it.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- The object form `members: [{name: app-a, paths: ["projects/app-a/**"]}]` parses into the area set with its globs.
- The legacy string form `members: [app-a, app-b]` parses unchanged (no paths; behaves exactly as E-0043).
- The mixed form (some string, some object members) parses.
- `paths` is optional within the object form; an object member with no paths behaves label-only.
- Validation rejects malformed members (non-string in string position, non-list paths, empty name) with a clear error; round-trips clean.
- A spec-sourced test pass enumerates string-only / object-only / mixed / no-paths-object forms (per CLAUDE.md "spec-sourced inputs").

## Constraints

- Backward-compatible and zero-migration — the named top risk of the epic; the dual-form unmarshaler is tested across every form before this milestone closes.
- `area` stays single-valued; `paths` is a per-member list of globs describing the project's location, not a second grouping axis.
- `paths:` describe the consumer's project code, never aiwf's own kind-partitioned entity layout (ADR-0004 untouched).

## Design notes

- Extend the existing custom `Areas.UnmarshalYAML` (it already decodes via `yaml.Node` to reject non-string members) to additionally accept mapping nodes.
- Glob matcher and any new dependency are decided here (open question); prefer an already-vendored matcher; justify any new dep per Go conventions.
- The flat-list → object schema evolution is an ADR candidate, harvested at wrap.

## Out of scope

- Consuming the paths — the bijection check, mistag detection, and auto-derive are their own milestones.

## Dependencies

- None. Foundation for the bijection coverage check and both Tier-2 milestones (mistag detection, auto-derive).

## References

- `internal/config/config.go` — the `Areas` type and `UnmarshalYAML` extended here.
