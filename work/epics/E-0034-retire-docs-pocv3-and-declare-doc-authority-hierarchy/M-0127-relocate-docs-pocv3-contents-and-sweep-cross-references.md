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
      status: met
    - id: AC-3
      title: Zero dangling docs/pocv3 references
      status: met
    - id: AC-4
      title: aiwf check and link integrity clean
      status: met
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

## Work log

### AC-1 — docs/pocv3/ files relocated/archived per TRIAGE.md

All 42 rows executed (`git mv`, plain filesystem operations, not aiwf verbs — `docs/pocv3/` holds no entities); mechanically verified every target exists and every source is gone; `docs/pocv3/` removed entirely · commit 3f6e14a6 · tests 1/1

### AC-2 — Fixture path constants updated for legal-workflows files

`internal/policies/m0121_audit_catalog_test.go` and `m0122_first_principles_catalog_test.go` repointed to `docs/design/`; both files' full test suites pass · commit 3f6e14a6 · tests 2/2

### AC-3 — Zero dangling docs/pocv3 references

Swept ~121 files (Go source, embedded skills, docs/, ADRs, CI config, live planning-tree bodies); added `TestM0127_AC3_NoDanglingDocsPocv3References` under `internal/policies/`, vacuity-checked (injected a stray reference, confirmed the test caught it, reverted) · commit 3f6e14a6 · tests 1/1

### AC-4 — aiwf check and link integrity clean

`aiwf check` reports only the pre-existing G-0434 false positives (now also firing on M-0127, same reused-id pattern as M-0126) and the expected no-upstream advisory — zero findings attributable to the sweep. A repo-wide markdown-link-integrity pass (matching `wf-doc-lint` check 5: fenced/inline-code spans excluded, directory targets treated as valid) reports zero broken links outside `docs/archive/**`, the frozen historical snapshot of the retired tree · commit 3f6e14a6 · tests clean

