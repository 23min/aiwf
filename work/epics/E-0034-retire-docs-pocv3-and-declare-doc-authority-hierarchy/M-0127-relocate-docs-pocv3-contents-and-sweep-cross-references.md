---
id: M-0127
title: Relocate docs/pocv3/ contents and sweep cross-references
status: in_progress
parent: E-0034
depends_on:
    - M-0126
tdd: none
acs:
    - id: AC-1
      title: docs/pocv3/ files relocated/archived per TRIAGE.md
      status: met
    - id: AC-2
      title: Fixture path constants updated for legal-workflows files
      status: open
    - id: AC-3
      title: Zero dangling docs/pocv3 references
      status: open
    - id: AC-4
      title: aiwf check and link integrity clean
      status: open
---

## Goal

Execute the moves recorded in M-0126's `TRIAGE.md` table. Update every cross-reference to `docs/pocv3/` across the repo (~163 files at planning time including markdown, Go source under `internal/`, and embedded skill markdown). At milestone close, `docs/pocv3/` no longer exists, `aiwf check` is clean, and a repo-wide link check is clean.

## Context

**Gated on E-0033 wrap.** E-0033's Pass A (M-0121) audits `docs/pocv3/design/design-decisions.md` and other normative docs as primary citation sources, and writes new tests under `internal/policies/`. The relocate sweep touches the same files. Running concurrently invites silent drift and merge friction; the framework's "correctness must not depend on LLM behavior" rule applies here too.

Full AC body, design notes, and surfaces-touched section drafted at `aiwfx-start-milestone` time. The shape will refine post-Triage when the actual file set + target paths are known.

## Acceptance criteria

### AC-1 — docs/pocv3/ files relocated/archived per TRIAGE.md

Every one of `TRIAGE.md`'s 42 rows is executed: `relocate` rows land at their recorded target path, `archive` rows land under `docs/archive/pocv3/`, and the one `supersede-with-entity` row (`observability-surfaces-plan.md` → G-0433) has its source archived alongside the rest. `docs/pocv3/` contains zero files afterward.

### AC-2 — Fixture path constants updated for legal-workflows files

`internal/policies/m0121_audit_catalog_test.go` and `internal/policies/m0122_first_principles_catalog_test.go` resolve `docs/design/legal-workflows-audit.md` and `docs/design/legal-workflows-first-principles.md` respectively (the `TRIAGE.md`-recorded relocate targets), and both test files pass unmodified in their assertions otherwise.

### AC-3 — Zero dangling docs/pocv3 references

A repo-wide structural test asserts no live Go source (`internal/`, `cmd/`), embedded skill markdown, or top-level doc (`docs/`, `README.md`, `CONTRIBUTING.md`) contains the literal substring `docs/pocv3`, with a narrow, explicit allowlist for deliberately historical mentions (`CHANGELOG.md`; this epic's own `work/` planning-tree prose narrating the migration).

### AC-4 — aiwf check and link integrity clean

`aiwf check` reports no new findings attributable to the sweep, and a repo-wide markdown-link-integrity pass (the `wf-doc-lint` check 5 heuristic, run mechanically) reports zero broken links caused by the move.

## Out of scope

- Re-classifying any file beyond what M-0126's `TRIAGE.md` records. If triage was wrong, file a gap; do not silently revise during the sweep.
- Writing the CLAUDE.md hierarchy section (M-0128's job).
- Landing the drift chokepoint (M-0129's job).

## Dependencies

- M-0126 (Triage) — done. Provides the disposition table this milestone executes.
- E-0033 (Pin legal kernel-verb workflows mechanically) — wrapped. Removes the file-conflict window.

## References

- **E-0034** — parent epic.
- **G-0132** — `aiwf render roadmap --write` blocked by dangling refs in source epic bodies. Worth resolving alongside the sweep if the renderer-canonicalization fix is in scope, since this milestone is sweeping cross-references anyway.

