---
id: D-0007
title: Authorize verb refuses non-{epic, milestone} scope-entities
status: proposed
relates_to:
    - E-0033
---
## Sources

- First-principles: R-FP-0133 (legal-workflows-first-principles.md, §6d authorize verb)
- Audit: R-AUDIT-0122 (legal-workflows-audit-r1.md, §4 Cobra verb definitions — `authorize`)
- Class: FP-only — Pass A does not enumerate kind restrictions on the scope-entity; impl behavior on non-`{epic, milestone}` scope-entities is currently undefined/permissive.

## Resolution

`aiwf authorize <id> --to <agent>` refuses if `<id>` is not an epic or milestone. The kernel emits `authorize-kind-not-allowed` (verb-time hard-reject) for gap, decision, contract, ADR scope-entities.

Rationale:

- Scope-authorization means *"agent is allowed to work on the subtree rooted at this entity."* For that to make sense, the scope-entity must have a *subtree of work* the agent can act in (per D-0006's reachability decision).
- **Epic**: subtree is milestones; sensible scope-entity.
- **Milestone**: subtree is ACs (composite) + gaps via `discovered_in`; sensible scope-entity.
- **Gap**: no subtree (gaps are states-of-the-world; the fix lives in a milestone). Not a scope-entity.
- **Decision**: no subtree (decisions are governance; delegating the deciding violates *"principal is human"*). Not a scope-entity.
- **Contract**: edge case (schema/fixtures could be maintained by an agent, but the contract entity doesn't have subtrees today). Not a scope-entity today; may be revisited.
- **ADR**: no subtree (ADRs are explicit architectural decisions; agent-driven ADR authoring inverts the purpose). Not a scope-entity.

YAGNI: restricting now is cheap (one impl guard); adding kinds later is a one-line relaxation if a real use case surfaces (e.g., contract-schema-maintenance scope). The closed set covers every authorization use case in practice on the project today.

No-surprise: a user trying `aiwf authorize ADR-0011 --to ai/claude` gets a clear error rather than a silent-noop scope (an empty reachability set under D-0006 would otherwise let the verb succeed but the agent could touch nothing — a confusing failure mode).

Sovereign override: `aiwf authorize <id> --to <agent> --force --reason "..."` remains available for genuinely exceptional scope-entity kinds.

## Spec cell

`internal/workflows/spec` — `Rule{Kind: ∈ {Gap, Decision, Contract, ADR}, Verb: "authorize", Outcome: Illegal, RejectionLayer: VerbTime, BlockingStrict: true, ExpectedErrorCode: "authorize-kind-not-allowed"}` (one row per disallowed kind, or one row with a "kind ∈ deny-set" predicate — schema choice for phase 1's concretization).

## Follow-up

Impl change scope-out of M-0123. File a gap → milestone under E-0033 for: kind guard in `aiwf authorize` verb body, new finding code `authorize-kind-not-allowed`, integration test exercising the refuse path per disallowed kind.
