---
id: E-0033
title: Pin legal kernel-verb workflows mechanically
status: done
---
## Goal

Verify mechanically — not by prose catalog or LLM recall — that the aiwf binary only permits legal sequences of kernel-verb invocations against the planning tree, and rejects illegal ones with named exit codes or finding codes. The deliverable is a **canonical Go spec table** describing the legal/illegal frontier at the kernel-verb level, plus per-cell positive and negative tests under `internal/policies/` that exercise the binary against the table.

This epic replaces the cancelled E-0031, whose first attempt produced a prose catalog and hand-coded fuzz harness — neither of which could mechanically catch an implementation bug. The structural critique that killed E-0031: a legality spec must be a **machine-readable transition surface**, not a narrative description, and tests must drive the binary against that surface cell-by-cell.

## Scope

Three of the four legal-workflow layers identified during planning:

1. **Per-entity FSM** — `(kind, current-state, event) → next-state`, already partially formalized in `internal/entity/transition.go`. The spec captures the closed-set table; tests assert the binary agrees.
2. **Per-verb pre/post conditions beyond FSM** — cross-entity invariants like "wrap-epic requires all child milestones terminal." Currently scattered across verb implementations and `aiwf check` rules.
3. **Cross-verb sequence legality** — e.g., "you can't promote an AC to met after the parent milestone is done." Currently implicit, never formally specified.

## Out of scope

- **Branch choreography (layer 4)** — covered by E-0030. Workflows that depend on which git branch you're on are not in this epic's surface.
- **Rituals-plugin orchestration** — skills like `aiwfx-start-milestone` compose kernel verbs into named sequences. Per "framework correctness must not depend on LLM behavior," the kernel can only mechanically verify kernel-verb legality. Skill-level coverage stays advisory.
- **Model-based random-walk testing** — useful as a safety net beyond the spec table, but a separate later concern. This epic produces the table; fuzz comes after.
- **Spec-from-impl extraction** — explicitly rejected as a methodology. The spec is human-authored; tests cross-check.

## Methodology

The spec is built via a three-pass approach (ratified in the ADR landed by M-α₀):

- **Pass A (M-α₁)** — audit existing surfaces (`entity/transition.go`, `internal/policies/`, `internal/check/`, ADRs, `design-decisions.md`, `CLAUDE.md`, skills, `--help` text) and extract every legality statement with citations.
- **Pass B (M-β)** — independently derive workflows from the entity model (lifecycles, ownership relations, cross-entity invariants), without reading Pass A's output.
- **Pass C (M-γ)** — reconcile A vs B into a canonical Go spec table under `internal/workflows/spec/` (exact package name TBD), plus a drift policy ensuring no impl FSM transition escapes the table.

Then:

- **M-δ** — positive cell coverage (every legal cell → success + expected post-state).
- **M-ε** — negative cell coverage (every illegal cell → named rejection).

## Milestones

| # | Id | Title |
|---|----|-------|
| Label | Id | Title | Depends on |
|-------|----|-------|-----------|
| M-α₀ | M-0120 | Ratify legal-workflow spec methodology in ADR | — |
| M-α₁ | M-0121 | Pass A audit: catalog legal-workflow rules from existing surfaces | M-0120 |
| M-β  | M-0122 | Pass B first-principles: derive legal-workflow rules from entity model | M-0120 |
| M-γ  | M-0123 | Pass C reconcile to canonical Go spec table + drift policy | M-0121, M-0122 |
| M-ζ  | M-0130 | Implement fsm-history-consistent check rule for FSM tree-invariant (closes G-0132) | M-0123 |
| M-η  | M-0131 | State-aware CancelTarget for Contract: cancel deprecated targets retired (closes G-0131) | M-0123 |
| M-δ  | M-0124 | Positive cell coverage: legal workflows succeed with expected post-state | M-0123, M-0130, M-0131 |
| M-ε  | M-0125 | Negative cell coverage: illegal workflows rejected with named errors | M-0123, M-0130, M-0131 |

**M-ζ and M-η** were inserted between Pass C and the cell-coverage milestones (2026-05-18) so that M-0124/M-0125's tests run against the actually-enforced spec, not a partially-aspirational one. The decision was driven by an external review of M-0121's audit catalog (review finding #3): committing the catalog to FSM-as-tree-invariant in R-RULE-019/R-RULE-001..018 without implementing the `fsm-history-consistent` chokepoint leaves the cell tests testing only verb-time enforcement. M-0130 closes the gap; M-0131 fixes a state-aware `CancelTarget` bug surfaced in the same review (review finding #1).

## What this epic deliberately does *not* do

- It does not enumerate the legal workflows up front — that's the *output* of M-α₁ + M-β + M-γ, not an input.
- It does not commit to a specific Go-table schema before Pass C — the schema is designed during M-γ when we know what the catalogs surfaced.
- It does not include "negative-of-undefined" testing (cells the spec deliberately leaves silent). That posture is decided during M-γ based on whether reconciliation surfaces any genuinely undecidable cells.

## Background

The cancelled E-0031 (`Pin legal workflows, composition, and branch choreography mechanically`, cancelled 2026-05-18) attempted to specify all four layers in one epic with a prose-shaped catalog. Two structural failures emerged:

1. The "spec" was narrative prose with closed-set tokens for branch context, but no machine-readable transition graph. Every downstream milestone (test harness, citation symmetry, fuzz harness) papered over this by hand-coding what should have been spec-driven.
2. Branch choreography (layer 4) conflated git state with entity state, muddying the test fixture shape.

E-0033 fixes both by (a) splitting layer 4 off to E-0030's scope and (b) committing to an *independent* machine-readable spec as the M-γ deliverable, with the audit + first-principles passes as evidence-gathering.
