---
id: M-0129
title: 'Drift chokepoint: forbid docs/pocv3/ literals in Go code'
status: draft
parent: E-0034
depends_on:
    - M-0128
tdd: required
---

## Goal

Land an `internal/policies/` rule that fails CI if any Go source file under `internal/**` or `cmd/**` contains a literal `"docs/pocv3/"` path string. Without the chokepoint, the post-Relocate state stays clean only as long as humans remember not to type the old path back in. With it, the kernel polices the rule mechanically.

## Context

**Conditional milestone.** Decision deferred to M-0128 wrap per the epic spec — include this milestone in the epic, or extract as a follow-up gap, based on whether residual `docs/pocv3/` references survived M-0131's sweep. If the sweep was complete and the pattern is unlikely to recur (no consumer-side scripts, no skill-bodies citing the old path), this milestone may be cancelled with a one-line rationale and a follow-up gap filed instead.

If kept, this is a textbook TDD milestone: write a Go test that scans the relevant trees for the literal pattern and asserts no matches; verify the test fails against a planted fixture; remove the fixture; green. Red → green → refactor.

## Out of scope

- Forbidding `docs/pocv3/` literals in markdown files — the post-Relocate state has none and the sweep should have been comprehensive; if drift recurs in markdown, a separate finding-rule covers it.
- Forbidding all stale doc paths globally. This milestone targets `docs/pocv3/` specifically; the broader "no dangling doc references" class is the existing `PolicyNoDanglingEntityRefsInNarrativeDocs` plus G-0132's renderer fix.

## Dependencies

- M-0128 (Hierarchy) — done. The CLAUDE.md hierarchy section is the human-facing peer of this chokepoint; both ship before the epic wraps.

## References

- **E-0034** — parent epic.
- **CLAUDE.md** § "What's enforced and where" — the table this milestone's new rule extends.
- **G-0092** — the doc-authority gap whose layer-3 (kernel rule) this milestone partially realizes for the specific `docs/pocv3/` case.
