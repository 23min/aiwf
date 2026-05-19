---
id: M-0126
title: Triage docs/pocv3/ into per-file disposition table
status: draft
parent: E-0034
tdd: none
---

## Goal

Produce a per-file disposition table for every file under `docs/pocv3/`. Each row records one of {`relocate`, `archive`, `supersede-with-entity`, `delete`} plus a target path (for `relocate`/`archive`) or entity id (for `supersede-with-entity`) and a one-line rationale. The table is the contract that M-0127 (Relocate) executes against verbatim.

## Context

Per E-0034's epic spec, `docs/pocv3/` is the historical working-name vintage of the pre-trunk-promotion era and mixes load-bearing normative records, pre-dogfooding plans (which now belong as `work/epics/`/`work/milestones/` entities, not docs), historical handoff/migration artifacts, and stale content. The tier of each file is opaque from the path. This milestone classifies each file so the relocate sweep can execute deterministically.

Triage is markdown-only — no Go source touched. It can run in parallel with E-0033.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0130 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — A triage table file (e.g. `TRIAGE.md` under this milestone's directory) exists and lists every regular file currently under `docs/pocv3/`, one row per file.
- **AC-2 candidate** — Every row carries a non-empty `disposition`, `target`, and `rationale` column; `disposition` is one of the four closed-set values.
- **AC-3 candidate** — A structural test under `internal/policies/` parses the table and asserts the file set equals `find docs/pocv3 -type f` at the moment the test runs. Coverage of the table is mechanical, not by reviewer recall.
- **AC-4 candidate** — Open Question #1 from E-0034 (whether `docs/archive/` absorbs `docs/pocv3/archive/` content or stays separate) is resolved and recorded in the table or in a "Triage rationale" section of this milestone spec.
- **AC-5 candidate** — Each file marked `supersede-with-entity` is paired with an existing or newly-filed entity id. Files marked `delete` carry an explicit one-line justification (default is `archive`).

## Constraints

- **Forget-by-default per ADR-0004.** Default disposition for unclear historical content is `archive`, not `delete`. Deletion requires an explicit justification.
- **No moves in this milestone.** Triage is recording, not relocating. The disposition table is the deliverable; the file system stays unchanged.
- **Pre-dogfooding plans get split.** Files under `docs/pocv3/plans/` that map to shipped epics are `archive`; partly-shipped plans are split (`archive` the shipped portion, `supersede-with-entity` the residual); never-started plans become an entity (typically a gap if scoped, an epic if larger).

## Out of scope

- Executing any file moves (M-0127's job).
- Writing the CLAUDE.md hierarchy section (M-0128's job).
- Renaming top-level `docs/` subdirs not under `docs/pocv3/`. The current top-level `docs/archive/` may receive content from `docs/pocv3/archive/` but is not itself renamed in this milestone.

## Dependencies

- E-0034 epic spec at `4a230e01` (committed).
- No prior milestones — Triage is the first.

## References

- **E-0034** — parent epic.
- **ADR-0004** — Uniform archive convention for terminal-status entities. The forget-by-default principle and the per-kind archive shape applied to `docs/`.
- **G-0074 / G-0075 / G-0092** — superseded by E-0034; this milestone's table is what makes the supersedes claim concrete.
