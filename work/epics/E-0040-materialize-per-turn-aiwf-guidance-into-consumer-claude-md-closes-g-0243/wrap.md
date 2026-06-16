# Epic wrap — E-0040

**Date:** 2026-06-16
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0040-materialize-per-turn-aiwf-guidance-into-consumer-claude-md-closes-g-0243
**Merge commit:** 7532a5d9

## Milestones delivered

- M-0163 — Embed and materialize the guidance fragment (merged 31842fab)
- M-0164 — Wire the CLAUDE.md guidance import with consent (merged 7810d0e5)
- M-0165 — Surface unwired CLAUDE.md guidance in aiwf doctor (merged e55afb77)

## Summary

aiwf now reaches the one consumer surface re-injected on every turn and surviving
`/compact`: `aiwf init`/`update` materialize a version-pinned guidance fragment
(`.claude/aiwf-guidance.md`) and automatically maintain a marker-wrapped
`@`-import of it in the consumer's root `CLAUDE.md` — self-healing, line-anchored,
default-on with an `aiwf.yaml` opt-out. An advisory `aiwf doctor` finding surfaces
an unwired tree. This closes G-0243 (the deferred Layer-3 of G-0242). Scope
shifted once mid-flight: an independent pre-merge review prompted replacing the
original `--no-wire-claudemd` CLI flag with the automatic, config-opt-out model,
and hardening marker detection to be line-anchored (no silent prose-corruption).

## ADRs ratified

- ADR-0018 — risk-calibrated consent for user-owned file edits (the automatic
  `CLAUDE.md` instance + the guidance-fragment inclusion principle)

## Decisions captured

- None as standalone decision entities. The mid-flight design changes
  (flag → automatic + config opt-out; the assumed consent-machinery reuse not
  applying; line-anchored marker detection) are recorded in ADR-0018 and in the
  M-0164 / M-0165 "Decisions made during implementation" sections.

## Follow-ups carried forward

- **Test-infra flake (candidate gap, not yet filed).** A git "invalid object /
  Error building trees" + "directory not empty" flake recurred three times during
  this epic across `internal/cli/integration`, `internal/policies`, and
  `internal/check` under full-suite parallel load; each package passes in
  isolation. Pre-existing (the G-0097 parallel-git-subprocess family), not
  introduced by E-0040.

## Doc findings

doc-lint: clean — the epic's documentation changes (CLAUDE.md item #5,
design-decisions.md materialized-artifacts + config tables, ADR-0018, the three
milestone specs) are additive; all id/path references resolve; no removed-feature
docs, orphan files, or TODOs introduced.

## Handoff

G-0243 is closed (addressed by the merge commit). The guidance feature is complete
and self-maintaining; the `aiwf.yaml` `guidance.wire_claudemd: false` knob is the
only opt-out (no CLI flag, by design). Ready for the next epic, or a release
(`aiwfx-release`: version bump + CHANGELOG roll + tag). The one open thread is the
test-infra flake noted above.
