---
id: M-0126
title: Triage docs/pocv3/ into per-file disposition table
status: in_progress
parent: E-0034
tdd: none
acs:
    - id: AC-1
      title: Triage table lists every docs/pocv3/ file
      status: met
    - id: AC-2
      title: Every row has disposition, target, rationale
      status: met
    - id: AC-3
      title: Structural test asserts table matches docs/pocv3/ file set
      status: met
    - id: AC-4
      title: 'Open Question #1 resolved and recorded'
      status: met
    - id: AC-5
      title: Supersede/delete rows carry entity id or justification
      status: open
---

## Goal

Produce a per-file disposition table for every file under `docs/pocv3/`. Each row records one of {`relocate`, `archive`, `supersede-with-entity`, `delete`} plus a target path (for `relocate`/`archive`) or entity id (for `supersede-with-entity`) and a one-line rationale. The table is the contract that M-0127 (Relocate) executes against verbatim.

## Context

Per E-0034's epic spec, `docs/pocv3/` is the historical working-name vintage of the pre-trunk-promotion era and mixes load-bearing normative records, pre-dogfooding plans (which now belong as `work/epics/`/`work/milestones/` entities, not docs), historical handoff/migration artifacts, and stale content. The tier of each file is opaque from the path. This milestone classifies each file so the relocate sweep can execute deterministically.

Triage is markdown-only — no Go source touched. It can run in parallel with E-0033.

## Acceptance criteria

### AC-1 — Triage table lists every docs/pocv3/ file

A triage table file (e.g. `TRIAGE.md` under this milestone's directory) exists and lists every regular file currently under `docs/pocv3/`, one row per file.

### AC-2 — Every row has disposition, target, rationale

Every row carries a non-empty `disposition`, `target`, and `rationale` column; `disposition` is one of the four closed-set values (`relocate`, `archive`, `supersede-with-entity`, `delete`).

### AC-3 — Structural test asserts table matches docs/pocv3/ file set

A structural test under `internal/policies/` parses the table and asserts the file set equals `find docs/pocv3 -type f` at the moment the test runs. Coverage of the table is mechanical, not by reviewer recall.

### AC-4 — Open Question #1 resolved and recorded

Open Question #1 from E-0034 (whether `docs/archive/` absorbs `docs/pocv3/archive/` content or stays separate) is resolved and recorded in the table or in a "Triage rationale" section of this milestone spec.

### AC-5 — Supersede/delete rows carry entity id or justification

Each file marked `supersede-with-entity` is paired with an existing or newly-filed entity id. Files marked `delete` carry an explicit one-line justification (default is `archive`).

## Triage rationale

- **`docs/archive/` absorption (Open Question #1).** Resolved as separate: `docs/pocv3/`-origin archival content lives under a new `docs/archive/pocv3/` sibling namespace. `docs/archive/README.md`'s existing two-category charter (pre-PoC design documents, one-time procedural artifacts) stays untouched.
- **`loom-by-example.md` / `loom-light-plan.md`.** Relocate — not archive — to a new `docs/explorations/loom/` topic subfolder, matching the `policy-model.md` / `explorations/surveys/` precedent for live-but-not-yet-committed research. No entity filed: the design still carries multiple genuinely-unresolved forks (standalone vs. bundled engine, which verifier, `.lm` syntax now-or-later); filing an epic now would be a placeholder with an unpinned shape.
- **`contracts-plan.md`'s I2 residual** (import-manifest `contracts:` block). Considered, declined — no adopter currently migrates via `aiwf import` with pre-existing contracts. The file archives as a unit with no entity pairing.
- **`observability-surfaces-plan.md`'s Phase 1.** Split. The `depends_on`-surfacing and readiness-marker items are tracked as **G-0433**. The local-vs-origin delta item — explicitly the larger of the three, described in the plan as its own "small epic" — is deferred, not filed. The source file archives regardless of the entity pairing, per the default-archive rule for retired `plans/` content.
- **`policy-model.md`.** Relocates to `docs/explorations/05-policy-model-design.md`, overwriting the file already there rather than sitting alongside it — diffed the two and the pocv3 copy is a later, more refined draft of the same design.
- **`docs/pocv3/gap-triage-2026-06-16.md`'s "Candidate B."** Verified real by that doc's own audit and explicitly recommended for filing; filed as **G-0432**. Candidates A and C were recommended by the same doc to fold into the existing G-0235 rather than get their own filing — no action needed here.

## Constraints

- **Forget-by-default per ADR-0004.** Default disposition for unclear historical content is `archive`, not `delete`. Deletion requires an explicit justification.
- **No moves in this milestone.** Triage is recording, not relocating. The disposition table is the deliverable; the file system stays unchanged.
- **Pre-dogfooding plans get split.** Files under `docs/pocv3/plans/` that map to shipped epics are `archive`; partly-shipped plans are split (`archive` the shipped portion, `supersede-with-entity` the residual); never-started plans become an entity (typically a gap if scoped, an epic if larger).

## Out of scope

- Executing any file moves (M-0131's job).
- Writing the CLAUDE.md hierarchy section (M-0128's job).
- Renaming top-level `docs/` subdirs not under `docs/pocv3/`. The current top-level `docs/archive/` may receive content from `docs/pocv3/archive/` but is not itself renamed in this milestone.

## Dependencies

- E-0034 epic spec at `4a230e01` (committed).
- No prior milestones — Triage is the first.

## References

- **E-0034** — parent epic.
- **ADR-0004** — Uniform archive convention for terminal-status entities. The forget-by-default principle and the per-kind archive shape applied to `docs/`.
- **G-0074 / G-0075 / G-0092** — superseded by E-0034; this milestone's table is what makes the supersedes claim concrete.
