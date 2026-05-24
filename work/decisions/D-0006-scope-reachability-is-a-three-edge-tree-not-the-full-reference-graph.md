---
id: D-0006
title: Scope reachability is a three-edge tree, not the full reference graph
status: accepted
relates_to:
    - E-0033
---
## Sources

- First-principles: R-FP-0130 (legal-workflows-first-principles.md, §6c scope FSM)
- Audit: R-AUDIT-0186 (legal-workflows-audit-r1.md, §6 kernel commitments — gating shape)
- Class: FP-only — Pass A captures the rule shape but does not enumerate which fields participate; provenance-model.md describes the chain as *"bounded by the existing kind reference grammar"* without an explicit edge set.

## Resolution

The reachability check for `aiwf authorize` scope-gating traverses exactly three edge types from the scope-entity outward:

1. **`parent` forward** — target's `parent` chain reaches the scope-entity (transitively). Composition.
2. **Composite-id containment** — AC `M-NNNN/AC-N` is reachable iff M-NNNN is reachable.
3. **`discovered_in` reverse** — target's `discovered_in` field points into the scope-entity's subtree (transitively via `parent`).

No other edges traverse. Specifically NOT reachability edges: `depends_on` (within-epic milestones already covered by `parent`; cross-epic dependencies should not punch through scope boundaries), `addressed_by` (redundant with `discovered_in` for the agent's needs), `relates_to`, `linked_adrs`, `supersedes`, `superseded_by` (all governance-layer; not scope-membership).

Rationale:

- Scope is a *governance* construct — it limits how far an agent can act under a human principal's name. It's not about *"what reference grammar allows"* — it's about *"what work surface naturally belongs to the human's scope."*
- The natural mental model is *"the work tree rooted at the scope-entity."* That's a tree (compositional hierarchy with discovery-reverse), not an arbitrary graph.
- Concrete friction case justifying `discovered_in` reverse: agent authorized on E-NN files a gap with `--discovered-in M-K` (a milestone in E-NN's subtree); the gap is created. Later, agent fixes the gap and wants to promote `addressed`. With strict-parent-only reachability, the gap's `discovered_in: M-K → parent: E-NN` doesn't traverse — gaps have no `parent` field. The agent can't promote a gap it just filed; hand-back to the human required. Adding `discovered_in` reverse closes this friction loop without expanding to governance edges.
- Closed-set design: if new kinds get added, they need explicit edge participation, not implicit graph traversal. Future-proof.
- KISS: implementation is *"walk `parent` chain + check `discovered_in` reverse"*; straightforward.

Alternatives considered:

- Strict (`parent` only): rejected — gaps surfaced during work become un-reachable for promotion, forcing constant hand-back to human.
- Composition + dependency forward (`parent` + `depends_on`): rejected — within-epic `depends_on` already covered by `parent`; cross-epic `depends_on` traversal would punch through epic boundaries (wrong).
- Inclusive (all reference fields): rejected — governance-layer edges (e.g., agent on E-1 reaches ADR via contract-binding chain) violate scope-as-governance semantics.

## Spec cell

`internal/workflows/spec` — expressed as a global precondition predicate that applies to every cell rather than per-cell duplication: `Predicate{Subject: "scope-reach", Operator: "via", Edges: [parent, composite-id, discovered_in-reverse]}`. The cell schema's exact encoding (single global rule vs. per-cell precondition replication) is settled during phase 1's schema concretization.

## Follow-up

Impl reconciliation: read `internal/scope/` to verify the current gating function's edge traversal matches the scope-tree above. If divergent, file a gap to align. If aligned, file a doc-update for `provenance-model.md` to enumerate the edge set explicitly.
