---
id: M-0086
title: Three new archive check-rule findings and existing-rule scoping
status: in_progress
parent: E-0024
depends_on:
    - M-0084
tdd: required
acs:
    - id: AC-1
      title: archived-entity-not-terminal fires blocking with revert hint
      status: open
      tdd_phase: green
    - id: AC-2
      title: terminal-entity-not-archived fires advisory per terminal in active dir
      status: open
      tdd_phase: red
    - id: AC-3
      title: archive-sweep-pending aggregates the count of pending sweeps
      status: open
      tdd_phase: red
    - id: AC-4
      title: Existing shape and health rules skip archive per ADR-0004
      status: open
      tdd_phase: red
    - id: AC-5
      title: Tree-integrity rules traverse archive in full
      status: open
      tdd_phase: red
    - id: AC-6
      title: 'refsResolve: active-to-archive refs resolve, archive-side not linted'
      status: open
      tdd_phase: red
    - id: AC-7
      title: 'Recovery: rewidth --apply runs clean on narrow archive without --skip-checks'
      status: open
      tdd_phase: red
---

# M-0086 — Three new archive check-rule findings and existing-rule scoping

## Goal

Land the three new check-rule findings from ADR-0004 (`archived-entity-not-terminal`, `terminal-entity-not-archived`, `archive-sweep-pending`) and scope existing shape/health rules to skip `archive/` while tree-integrity rules continue to traverse it. After this milestone, `aiwf check` reports drift in either direction with actionable hints, and the active-set health rules stop linting archived entities.

## Context

M-0084 (loader) and M-0085 (verb) make archive a real location. Without check-rule integration, drift is invisible: a hand-edit on an archived file (status off-terminal) goes unflagged, and accumulating unswept terminals in active dirs has no bound. This milestone closes the convergence loop: drift surfaces as findings, `archive-sweep-pending` aggregates pending-sweep counts for the threshold knob (M-0088 will make it blocking past N), and shape/health rules apply forget-by-default to archived entities.

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0086 --title "..."` at start time. -->

Intended landing zone:

- `archived-entity-not-terminal` fires (blocking) when a file lives under `archive/` but frontmatter status isn't terminal; remediation message names the revert path, not relocation.
- `terminal-entity-not-archived` fires (advisory by default) for each terminal entity in an active dir.
- `archive-sweep-pending` aggregates the count of `terminal-entity-not-archived` instances; advisory by default; configurable to blocking via `archive.sweep_threshold` (knob lands in M-0088).
- Existing shape/health rules (`acs-shape`, `entity-body-empty-ac`, `acs-tdd-audit`, `acs-body-coherence`, `milestone-done-incomplete-acs`, `unexpected-tree-file`) skip `archive/`.
- Tree-integrity rules (`ids-unique`, parse-level errors) traverse `archive/` in full.
- Reference-validity (`refs-resolve`): active→archived id refs resolve and don't flag; archive→active refs are not linted.

## Constraints

- `internal/entity/transition.go::IsTerminal` is the single terminality source.
- Per-rule archive scoping is named explicitly per rule — no global "skip if path contains archive" shortcut. Each rule documents whether it traverses archive and why.
- Discoverability: every new finding code is reachable through `--help` / embedded skill / CLAUDE.md per the AI-discoverability principle.

## Design notes

- The three finding codes follow the existing kebab-case finding-naming convention.
- `archive-sweep-pending` is an aggregate finding — it counts but does not point at individual files; the per-file `terminal-entity-not-archived` instances are the leaf nodes.
- `terminal-entity-not-archived` defaults to advisory; the default-permissive ADR-0004 stance means it never blocks unless a consumer opts in via `archive.sweep_threshold`.

## Surfaces touched

- `internal/check/check.go`
- `internal/check/rules/` (new files for the three findings)

## Out of scope

- The `archive.sweep_threshold` config knob (M-0088).
- `aiwf status` integration of the pending-sweep count (M-0087).
- `aiwf render` archive-segregation (M-0087).

## Dependencies

- M-0085 — verb produces the archive moves whose post-state the finding rules assert.
- ADR-0004 (accepted) — all three finding codes come from the ADR's *Check shape rules* section.

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — *`aiwf check` shape rules* section.
- `internal/check/check.go::refsResolve`

---

## Work log

(populated during implementation)

## Decisions made during implementation

- (none)

## Validation

(populated at wrap)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — archived-entity-not-terminal fires blocking with revert hint

### AC-2 — terminal-entity-not-archived fires advisory per terminal in active dir

### AC-3 — archive-sweep-pending aggregates the count of pending sweeps

### AC-4 — Existing shape and health rules skip archive per ADR-0004

### AC-5 — Tree-integrity rules traverse archive in full

### AC-6 — refsResolve: active-to-archive refs resolve, archive-side not linted

### AC-7 — Recovery: rewidth --apply runs clean on narrow archive without --skip-checks

