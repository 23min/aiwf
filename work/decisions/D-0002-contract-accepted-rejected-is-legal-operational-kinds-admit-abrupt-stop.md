---
id: D-0002
title: 'Contract accepted->rejected is legal: operational kinds admit abrupt-stop'
status: proposed
relates_to:
    - E-0033
---
## Sources

- First-principles: R-FP-0045 (legal-workflows-first-principles.md, §1f Contract FSM)
- Audit: R-AUDIT-0025 (legal-workflows-audit-r1.md, §1 FSM tables, transition.go:44)
- Class: Conflict — Pass B mirrors R-FP-0021's ADR analysis ("rejection is a pre-acceptance terminal"); Pass A reflects impl's wired `"accepted": {"deprecated", "rejected"}` edge.

## Resolution

Pass A wins; codify the impl's existing edge. Contract FSM:

    accepted → {deprecated, rejected}

The asymmetry with ADR (`accepted → {superseded}` only) is deliberate, not a typo.

Rationale: ADR captures a *decision* (durable, supersedable); Contract governs an ongoing *operational relationship* (validator + schema + fixtures). The operational kind admits failure modes the decision kind doesn't have analogue for — a schema breaking change, a nondeterministic validator, a fixture pair encoding the wrong invariant. The contract going `accepted → rejected` says *"this operational binding is broken in a way that doesn't admit a graceful sunset"*; `accepted → deprecated → retired` is the graceful path. Both edges encode distinct operational realities; collapsing them loses information.

Pass B's "mirror of R-FP-0021" reasoning under-weighted the operational-vs-decision distinction. Pass A reflects the model's deliberate richer FSM for the richer kind. A richer kind getting a richer FSM is a feature of having six kinds, not six identical FSMs.

Related orthogonal impl follow-up: G-0131 / M-0131 (state-aware `CancelTarget` for Contract — `cancel` of a `deprecated` contract should target `retired`, not `rejected`). Tracked separately; does not affect this decision.

## Spec cell

`internal/workflows/spec` — `Rule{Kind: entity.KindContract, FromState: entity.StatusAccepted, Verb: "promote", Outcome: Legal}` (legal transition target: `entity.StatusRejected`).
