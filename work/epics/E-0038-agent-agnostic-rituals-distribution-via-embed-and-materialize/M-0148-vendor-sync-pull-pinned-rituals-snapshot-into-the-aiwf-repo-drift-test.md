---
id: M-0148
title: 'Vendor-sync: pull pinned rituals snapshot into the aiwf repo + drift test'
status: draft
parent: E-0038
tdd: required
---
## Goal

Establish a reproducible mechanism that pulls a pinned snapshot of the upstream rituals repo (`23min/ai-workflow-rituals`) into a path inside the aiwf repo as real committed files, records the pinned upstream commit SHA, and guards the snapshot against drift with a CI test.

## Context

Before this milestone, the rituals reach consumers only via the Claude marketplace (G-0177). ADR-0014 chooses build-time embed of a vendored snapshot as the new distribution mechanism. This milestone lands the *source-of-truth* half: getting the upstream content into the aiwf repo in a form `go:embed` can consume later (M2+), pinned and drift-checked. It builds on nothing — it is the foundation the embed/materialize milestones depend on.

## Acceptance criteria

## Constraints

- The snapshot is **real committed files, not a git submodule** — the Go module proxy does not fetch submodule contents, so a submodule embeds empty under `go install` (ADR-0014 §2). Load-bearing.
- The upstream authoring home is unchanged; aiwf only vendors. No hand-edits to vendored content beyond the sync — the drift test enforces this.
- The pinned upstream SHA is recorded in a single discoverable location.

## Design notes

- ADR-0014 §2 (source of truth; pinned snapshot; submodule caveat).
- Mirrors the existing cross-repo SKILL.md fixture discipline (CLAUDE.md § "Cross-repo plugin testing"): author upstream, vendor a snapshot here, a drift test fires on divergence and skips cleanly when the upstream source is absent (CI without a checkout).
- The sync mechanism (`git subtree` pull vs scripted copy vs `go:generate`) and the SHA-record location are open questions resolved in this milestone; default lean: scripted copy + committed snapshot + a `rituals.lock`-style SHA record.

## Surfaces touched

- New vendored `rituals/` tree (path TBD in this milestone).
- A `make sync-rituals` target (or equivalent) and the drift-check test under `internal/policies/`.

## Out of scope

- Embedding the snapshot (`go:embed`) — M2.
- Materializing anything into a consumer repo — M2+.

## Dependencies

- ADR-0014 (the decision). No prior milestone.

## References

- **ADR-0014** — the decision this milestone implements (§2).
- **G-0177** — the motivating gap.
- **E-0038** — parent epic.
- **CLAUDE.md** § "Cross-repo plugin testing" — the vendoring + drift-test pattern reused.
