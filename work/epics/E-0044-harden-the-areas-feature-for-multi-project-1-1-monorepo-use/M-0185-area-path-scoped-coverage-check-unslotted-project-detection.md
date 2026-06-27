---
id: M-0185
title: Area-path scoped-coverage check (unslotted-project detection)
status: draft
parent: E-0044
depends_on:
    - M-0179
    - M-0180
    - M-0178
tdd: required
---
## Goal

Add the *covering* law of the area-path matrix to `aiwf check`: within an operator-declared
**coverage scope**, every project directory is claimed by some area — an unclaimed directory
raises an "unslotted project" finding. This is the monorepo-specific catch for a newly-added
project that nobody slotted into an area, and the third partition law M-0180 deliberately
deferred.

## Context

M-0180 delivered the two *config-anchored* laws (dead-glob, overlap) and the shared `areamatch`
matcher. The remaining law — coverage — is structurally different: it needs a definition of
*which directories are projects* (a universe), which the declared globs alone don't supply. The
forward laws only ever look at what the config *declares*; coverage must look at what the
filesystem *contains* and ask "did any area claim it?"

The universe problem is why this is its own milestone. A single project monorepo with, say, an
`infra` area and an app area plus a legitimate uncovered remainder (`README`, `docs/`, top-level
config) must **not** be flagged wholesale — total partition is the wrong assertion outside a true
multi-project layout. The model is therefore **scoped, opt-in coverage**, not blanket coverage.

## Scoped-coverage model

- The operator optionally declares one or more **coverage roots** in `aiwf.yaml` (the directory
  subtrees whose children are projects expected to be slotted).
- Within a coverage root, every immediate child directory must be claimed by some area's
  `paths:`; an unclaimed child raises the unslotted-project finding.
- Directories **outside** any declared coverage root are unscoped and never flagged (the `infra`
  area, top-level files, `docs/` — all legitimately silent).
- **No coverage root declared → the law is inert.** The knob's presence is also the
  "this is a multi-project monorepo" activation signal, so a semantic-section / single-project
  repo that merely declares `paths:` never trips coverage.

Two distinct "rests" the model keeps separate (settled in design): the *filesystem* remainder is
just unscoped-and-fine; the *entity* remainder (cross-cutting ADRs/decisions) is tagged
`area: global` on the entity axis. `global` is an entity-tag, not a directory claim, so it never
enters the directory-coverage domain.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- A new `aiwf.yaml` coverage-root knob parses and validates (Tier-1 schema; reuses the M-0179
  dual-form decode discipline; a malformed value is a load-time error).
- Within a declared coverage root, a child directory claimed by no area's glob raises an
  unslotted-project finding; a fully-slotted root is silent.
- Inert when no coverage root is declared (the activation signal), and when no `paths:` exist.
- Severity: warning by default, escalating to error under `areas.required` (consistent with
  dead-glob/overlap and `area-unknown`).
- Bounded, read-only enumeration: single-level `os.ReadDir` per declared root, never fails on IO
  (per the `roadmapCaseCollision` precedent); enumerating only *declared* roots sidesteps the
  `.git` / `node_modules` / build-output noise a blanket walk would pick up.
- Reuses M-0180's `areamatch` matcher for "is this directory claimed by an area's glob" — no
  second matcher.

## Constraints

- Reads the filesystem read-only; never writes. Composed at the CLI seam with the declared set
  from config, like `area-unknown` and the M-0180 checks.
- Does not gate the default views (raises filter trust, not view gating).
- `area` stays single-valued. Coverage is the *covering* half of the directory-partition; it does
  not read entity `area` tags (the `global` sentinel is irrelevant here).

## Out of scope

- The forward laws (dead-glob, overlap) and the `areamatch` matcher — delivered in M-0180.
- Mistag detection (M-0181) and auto-derive (M-0182).
- A static glob-intersection reading of coverage — the law is defined over the *enumerated
  directories within declared roots*, not over abstract glob-set algebra.
- A declared path-bearing "catch-all / remainder" area — the unscoped complement needs no area
  (YAGNI; revisit only if a real case demands it).

## Design notes

- This is the **covering** law — the third of the three partition laws (M-0180 landed
  no-empty-column + disjointness). Same algebra family as M-0176's entity-axis partition test,
  on the directory axis.
- The universe = the immediate children of the declared coverage root(s). The knob is the single
  source of truth for that universe; deriving it from glob anchors was rejected as brittle
  (multi-root, anchorless, variable-depth globs) for a check meant to be trustworthy.
- Native validation, in-binary: Tier-1 config-load validation for the knob, Tier-2 `aiwf check`
  rule for the law, Tier-3 property test for covering. No external validator (downstream config).
- `depends_on: M-0179` (paths oracle), `M-0180` (the `areamatch` matcher + the forward laws this
  completes), `M-0178` (the `areas.required` escalation seam).

## Dependencies

- M-0179 (`paths:` per area) — the oracle.
- M-0180 (dead-glob/overlap + `areamatch`) — the matcher and the forward laws this completes.
- M-0178 (`areas.required`) — the escalation seam for the severity contract.

## References

- M-0180 — the forward laws + `internal/areamatch` matcher this reuses.
- `internal/check/check.go` (`roadmapCaseCollision`) — the read-only, never-fail-on-IO
  directory-read precedent.
- `internal/config/config.go` — `Areas`; the coverage-root knob extends the schema here.
- `internal/areagroup/areagroup.go` — the entity-axis partition (M-0176); coverage is the
  directory-axis covering law.
