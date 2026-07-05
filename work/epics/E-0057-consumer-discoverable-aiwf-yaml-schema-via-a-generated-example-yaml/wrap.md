# Epic wrap — E-0057

**Date:** 2026-07-05
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0057-consumer-discoverable-aiwf-yaml-schema-via-a-generated-example-yaml
**Merge commit:** fe052ffc

## Milestones delivered

- M-0231 — Struct-derived aiwf.yaml schema model and commented-YAML generator (merged e01b0e51)
- M-0232 — Wire generator into init/update: fresh-repo scaffold and example.yaml (merged b5bc66c6)

## Summary

E-0057 gives every aiwf consumer a discoverable, always-fresh reference for the
whole `aiwf.yaml` schema, closing the systemic documentation hole `G-0360`
named (a config surface reachable only by reading Go source, absorbing the
retired `G-0288`). M-0231 built the anti-drift backbone: a reflection-based
walk of the `Config` struct tree into a single schema model (path/type/
description/default per field), a fully-commented YAML generator over that
model, and an exported accepted-key registry for `G-0307`'s future
strict-decode work to consume. M-0232 wired that generator into `aiwf init`/
`aiwf update`: a fresh repo now gets the fully-commented scaffold as its
first-touch `aiwf.yaml`; an existing `aiwf.yaml` is never rewritten; both verbs
write and refresh a gitignored `aiwf.example.yaml` sibling every run; and
`aiwf init --help` documents the idempotent re-run contract.

## ADRs ratified

- `ADR-0027` — Generated `aiwf.example.yaml` over in-file schema regeneration.
  Extends `ADR-0015`'s no-edits-without-consent posture; deliberately narrower
  than `ADR-0018`'s marker-managed in-file pattern, which stays correct for
  markdown-shaped surfaces (CLAUDE.md) but not for a live YAML document.

## Decisions captured

- None as standalone `D-NNN`. M-0231's description-source and defaults-source
  choices were locked at milestone-planning Q&A and recorded in that
  milestone's own `## Design notes` section, not as separate decision
  entities.

## Follow-ups carried forward

- `G-0364` — `entity-body-empty` fires on `## Acceptance criteria` regardless
  of AC-heading prose, on any milestone using the current full template
  (`aiwf add ac` appends headings at body-end, past several intervening
  `## ` sections). Discovered during M-0231, recurred on M-0232 for the same
  structural reason; open, a template/verb/check-level fix out of this
  epic's scope.
- `G-0307` — top-level `aiwf.yaml` decode stays non-strict (only `areas:`
  rejects unknown keys). M-0231's exported `AcceptedKeys()` registry is the
  single source `G-0307`'s strict-decode guard is meant to consume instead of
  a parallel allowlist; the strict-decode change itself remains `G-0307`'s to
  land.

## Handoff

Every consumer repo now gets a discoverable schema reference from first touch
(`aiwf init`'s fully-commented scaffold) and an always-fresh one thereafter
(`aiwf.example.yaml`, regenerated every `init`/`update`) — the field-coverage
anti-drift test means a newly-added `yaml:` field cannot ship undocumented.
`G-0307` is unblocked to consume the accepted-key registry for its
strict-decode work. `G-0364` is the one open loose end, deliberately out of
scope here — a check/template fix, not a schema-discoverability gap.
