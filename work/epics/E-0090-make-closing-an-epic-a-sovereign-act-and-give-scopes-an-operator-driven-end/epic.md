---
id: E-0090
title: Make closing an epic a sovereign act and give scopes an operator-driven end
status: active
---
## Goal

A human is the only actor who may declare an epic closed, on either edge that closes one, and a human can end an authorization scope deliberately instead of waiting for a status flip to end it as a side effect.

## Context

The kernel gates exactly one sovereign transition: epic `proposed → active`, the single entry in `sovereignActShapes`. Neither edge that closes an epic is gated. Measured in a fixture repo against an `active` epic under a live delegated scope, `aiwf promote <epic> done --actor ai/claude --principal human/fixture` exits 0 and flips the status, the commit carries no force trailer, and `aiwf check` on the result reports zero errors. One such commit already sits in this repo's history: `c030cb926`.

The gate is also promote-only. `requireHumanActorForSovereignAct` has a single call site, so `aiwf cancel` never consults the closed set — and because the history audit is transition-shaped rather than verb-shaped, adding a closed-set entry without a call site would let a cancel land at exit 0 and fail the next push instead of being refused.

Separately, an authorization scope has one exit: the terminal promote or cancel of its own entity, which stamps `aiwf-scope-ends` as a side effect of the status change. `AuthorizeMode` is a closed three-value set — open, pause, resume — with no end. G-0022 reserved an `aiwf-revoked-by:` trailer slot for a revoke verb that was never built.

## Scope

- An ADR settling what closing an epic is: who may declare it on both terminal edges, and how an operator-driven scope end coexists with the automatic one.
- Sovereign gating of epic `active → done`, through the existing promote call site.
- Sovereign gating of epic `active → cancelled`, including the call site in `cancel` that makes the entry enforceable at the verb.
- Ratification of the historical commit the widened audit fires on.
- An end mode on `aiwf authorize`, additive alongside the existing auto-end, with its surface reachable per the kernel's discoverability rule: `--help`, tab-completion wiring, the `aiwf-authorize` skill, and a CHANGELOG entry.

## Out of scope

- Removing or re-timing the automatic scope-end at terminal promote. It stays. G-0111's weld therefore survives this epic by decision, and the `aiwfx-wrap-epic` step ordering and both `internal/check/provenance.go` carve-outs are untouched.
- Force delegation and bypass recording — G-0023, G-0333, G-0550 and G-0638 sit in the same neighbourhood on a different axis.
- Same-state convergence for a duplicate `authorize` re-grant (G-0460). It informs the end mode's targeting question; it is not resolved here.
- Every item in G-0022's extension list except the revoke verb.

## Constraints

- Sovereign-act shape is a property *over* legal transitions, never below them (D-0008). Every entry added must be FSM-legal, and `TestSovereignActShapes_AllFSMLegal` pins it.
- Prevention at the verb route, ratification at the history route (ADR-0040). A widened closed set arrives together with its call site, never before it — a gate that reports after the commit lands produces exactly the record that ADR closes.
- The end mode is additive: no existing invocation changes behaviour, and no consumer relying on today's auto-end is affected.
- `--force` stays human-only. The coherence guard at `verb.Apply` already refuses a force trailer from a non-human actor and is not modified here.

## Success criteria

- [ ] A non-human actor is refused on both edges that close an epic, at the verb, before anything is written.
- [ ] A human can end an authorization scope without changing the status of its entity.
- [ ] Every historical commit the widened audit fires on carries a ratification, and `aiwf check` reports no error-severity finding on the tree.
- [ ] The end mode is reachable from `aiwf authorize --help`, from tab-completion, and from the `aiwf-authorize` skill.
- [ ] Every question in *Open questions* is answered by the ADR listed under *ADRs produced*.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the end mode target the most-recently-opened active scope, mirroring `--pause` / `--resume`, or every active scope on the entity, mirroring the auto-end? Multiple simultaneously-active scopes are legal and intended (G-0460), so the two existing answers disagree. | yes | The ADR, before the end mode is built. |
| Is cancelling an epic a sovereign act, or is completion the only closure a human must make? | yes | The ADR. Decides whether the `cancel` call site is built at all. |
| What undoes an end? Re-authorizing opens a fresh scope rather than reviving the ended one, and `ended` is terminal in the scope FSM. Is that the whole answer? | no | The ADR, under the kernel's what-undoes-this rule for a new verb surface. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Widening the closed set fires the history audit at error severity on commits already in the log, blocking every push until each is ratified. | med | One commit qualifies today. Ratify it in the same milestone that widens the set; the count only grows if the gate is deferred. |
| The gate keys on a self-declared actor — an invocation that omits `--actor` inherits the human identity from `git config` and passes through untouched. | med | Accepted here rather than mitigated. The property is shared with the shipped `proposed → active` gate and belongs to the identity substrate, not to this epic. |

## Milestones

- `M-0323` — settle closure authority on both terminal edges and the semantics of an operator-driven end, in one ADR · depends on: —
- `M-0324` — refuse a non-human actor on both edges that close an epic, with the `cancel` call site and the historical ratification · depends on: `M-0323`
- `M-0325` — add an operator-driven end to `aiwf authorize`, additive alongside the auto-end · depends on: `M-0323`

`M-0324` and `M-0325` depend only on the ADR, not on each other, so their order is soft.

## ADRs produced

One ADR, allocated when it is written: what closing an epic is — closure authority on both terminal edges, and the semantics of an operator-driven scope end alongside the automatic one.

## References

- G-0646 — closed by this epic.
- G-0111 — advanced but not closed. The operator-driven end lands here; the weld it is titled for survives by the scope decision above, so its claim is corrected rather than closed at wrap.
- G-0022 — its reserved revoke verb is implemented here; the remainder of that gap's extension list is untouched.
- G-0460 — establishes that multiple active scopes are legal, which is what makes the end mode's targeting a real question.
- ADR-0040 — prevention at the verb route, ratification at the history route.
- `internal/entity/sovereign.go`, `internal/verb/promote_sovereign_act.go`, `internal/verb/authorize.go`, `internal/cli/cliutil/provenance.go`.
