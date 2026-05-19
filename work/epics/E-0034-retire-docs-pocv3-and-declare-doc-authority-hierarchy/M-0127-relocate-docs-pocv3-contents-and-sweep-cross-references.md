---
id: M-0127
title: Relocate docs/pocv3/ contents and sweep cross-references
status: draft
parent: E-0034
depends_on:
    - M-0130
tdd: none
---

## Goal

Execute the moves recorded in M-0130's triage table. Update every cross-reference to `docs/pocv3/` across the repo (~163 files at planning time including markdown, Go source under `internal/`, and embedded skill markdown). At milestone close, `docs/pocv3/` no longer exists, `aiwf check` is clean, and a repo-wide link check is clean.

## Context

**Gated on E-0033 wrap.** E-0033's Pass A (M-0121) audits `docs/pocv3/design/design-decisions.md` and other normative docs as primary citation sources, and writes new tests under `internal/policies/`. The relocate sweep touches the same files. Running concurrently invites silent drift and merge friction; the framework's "correctness must not depend on LLM behavior" rule applies here too.

Full AC body, design notes, and surfaces-touched section drafted at `aiwfx-start-milestone` time. The shape will refine post-Triage when the actual file set + target paths are known.

## Out of scope

- Re-classifying any file beyond what M-0130's table records. If triage was wrong, file a gap; do not silently revise during the sweep.
- Writing the CLAUDE.md hierarchy section (M-0128's job).
- Landing the drift chokepoint (M-0129's job).

## Dependencies

- M-0130 (Triage) — done. Provides the disposition table this milestone executes.
- E-0033 (Pin legal kernel-verb workflows mechanically) — wrapped. Removes the file-conflict window.

## References

- **E-0034** — parent epic.
- **G-0132** — `aiwf render roadmap --write` blocked by dangling refs in source epic bodies. Worth resolving alongside the sweep if the renderer-canonicalization fix is in scope, since this milestone is sweeping cross-references anyway.
