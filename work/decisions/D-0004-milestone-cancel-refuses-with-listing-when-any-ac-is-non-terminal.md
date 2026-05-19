---
id: D-0004
title: Milestone cancel refuses with listing when any AC is non-terminal
status: proposed
relates_to:
    - E-0033
---
## Sources

- First-principles: R-FP-0064 (legal-workflows-first-principles.md, §2c AC × milestone composition)
- Class: FP-only — Pass A is silent (impl does not enforce cancel-cascade behavior today).

## Resolution

Adopt refuse-with-listing pattern, symmetric with D-0003 (Q5). `aiwf cancel M-NNNN` refuses if any AC has status `open`, printing the offending composite ids `M-NNNN/AC-N`; the operator handles each (promote to `met`, `deferred`, or `cancelled`) before retrying the milestone cancel.

Rationale:

- Symmetric with D-0003 → uniform mental model: *"cancel a parent with non-terminal children → refuse with listing"* holds at both epic→milestone and milestone→AC layers.
- Auto-cascade was considered as semantically natural (ACs are sub-elements; cascade is just an array mutation within the same milestone file, well within "one verb = one commit"). Rejected because the M-0123 schema does not encode postconditions / side-effects; expressing *"after milestone-cancel, all open ACs become cancelled"* requires a schema extension that M-0123's body explicitly forbids (per *"If reconciliation surfaces a rule that cannot be expressed as a state predicate, that is evidence the entity model is missing a state — surface it as a decision against E-0033, do not grow a second schema."*). The refuse-with-listing pattern stays within the schema's preconditions-only expressivity.
- The composition principle R-FP-0064 names (*"the milestone's terminality is the AC's terminality"*) is preserved via mandatory pre-disposition rather than auto-cascade. ACs cannot outlive the milestone; the operator commits to a disposition for each before the milestone terminalizes.

Forces per-AC disposition at cancel-time. Some open ACs may deserve `met` (work was done; just bureaucratic close); some `deferred` (paused for later); some `cancelled` (abandoned). Refusing surfaces this distinction.

## Spec cell

`internal/workflows/spec` — `Rule{Kind: entity.KindMilestone, FromState: <any non-terminal>, Verb: "cancel", Preconditions: [all-children-acs.status != entity.StatusOpen], Outcome: Legal, RejectionLayer: VerbTime, BlockingStrict: true, ExpectedErrorCode: "milestone-cancel-non-terminal-acs"}`.

## Follow-up

Impl change scope-out of M-0123. Likely shares the same gap/milestone under E-0033 as D-0003's Q5 enforcement (both add precondition guards in the `cancel` verb body and new finding codes in `internal/check/`).
