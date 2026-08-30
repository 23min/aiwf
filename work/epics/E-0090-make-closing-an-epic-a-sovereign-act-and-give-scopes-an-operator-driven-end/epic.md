---
id: E-0090
title: Make closing an epic a sovereign act and give scopes an operator-driven end
status: done
---
## Goal

A human is the only actor who may put an epic into a terminal status, on every edge that reaches one, and a human can end an authorization scope deliberately instead of waiting for a status flip to end it as a side effect.

## Context

The kernel gates exactly one sovereign transition: epic `proposed → active`, the single entry in `sovereignActShapes`. None of the three edges into a terminal epic status — `proposed → cancelled`, `active → done`, `active → cancelled` — is gated. Measured in a fixture repo against an `active` epic under a live delegated scope, `aiwf promote <epic> done --actor ai/claude --principal human/fixture` exits 0 and flips the status, the commit carries no force trailer, and `aiwf check` on the result reports zero errors. One such commit already sits in this repo's history: `c030cb926`.

The gate is also promote-only. `requireHumanActorForSovereignAct` has a single call site, so `aiwf cancel` never consults the closed set — and because the history audit is transition-shaped rather than verb-shaped, adding a closed-set entry without a call site would let a cancel land at exit 0 and fail the next push instead of being refused.

Separately, an authorization scope has one exit: the terminal promote or cancel of its own entity, which stamps `aiwf-scope-ends` as a side effect of the status change. `AuthorizeMode` is a closed three-value set — open, pause, resume — with no end. G-0022 reserved an `aiwf-revoked-by:` trailer slot for a revoke verb that was never built.

## Scope

- An ADR settling what closing an epic is: who may declare it on every terminal edge, and how an operator-driven scope end coexists with the automatic one. Settled in ADR-0047.
- Sovereign gating of epic `active → done`, through the existing promote call site.
- Sovereign gating of epic `proposed → cancelled` and `active → cancelled`, including the call site in `cancel` that makes those entries enforceable at the verb.
- Ratification of the historical commit the widened audit fires on.
- Correcting the automatic end's predicate so it covers paused scopes as well as active ones.
- An end mode on `aiwf authorize`, additive alongside the existing auto-end, with its surface reachable per the kernel's discoverability rule: `--help`, tab-completion wiring, the `aiwf-authorize` skill, and a CHANGELOG entry.

## Out of scope

- Removing or re-timing the automatic scope-end at terminal promote. It stays, and fires exactly where it already fires; this epic changes only which scopes it covers. G-0111's weld therefore survives this epic by decision, and the `aiwfx-wrap-epic` step ordering and both `internal/check/provenance.go` carve-outs are untouched.
- Force delegation and bypass recording — G-0023, G-0333, G-0550 and G-0638 sit in the same neighbourhood on a different axis.
- Same-state convergence for a duplicate `authorize` re-grant (G-0460). It informs the end mode's targeting question; it is not resolved here.
- Every item in G-0022's extension list except the revoke verb.

## Constraints

- Sovereign-act shape is a property *over* legal transitions, never below them (D-0008). Every entry added must be FSM-legal, and `TestSovereignActShapes_AllFSMLegal` pins it.
- Prevention at the verb route, ratification at the history route (ADR-0040). A widened closed set arrives together with its call site, never before it — a gate that reports after the commit lands produces exactly the record that ADR closes.
- The end mode is additive. One existing invocation does change behaviour, and only one: a terminal promote or cancel of an entity carrying a paused scope now ends that scope instead of stranding it. No consumer relying on today's auto-end for an active scope is affected.
- `--force` stays human-only. The coherence guard at `verb.Apply` already refuses a force trailer from a non-human actor and is not modified here.

## Success criteria

- [ ] A non-human actor is refused on every edge into a terminal epic status, at the verb, before anything is written.
- [ ] A human can end an authorization scope without changing the status of its entity.
- [ ] Every historical commit the widened audit fires on carries a ratification, and `aiwf check` reports no error-severity finding on the tree.
- [ ] A paused scope on an entity that reaches a terminal status is ended, not stranded.
- [ ] The end mode is reachable from `aiwf authorize --help`, from tab-completion, and from the `aiwf-authorize` skill.
- [ ] Every question in *Open questions* is answered by the ADR listed under *ADRs produced*.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the end mode target the most-recently-opened active scope, mirroring `--pause` / `--resume`, or every active scope on the entity, mirroring the auto-end? Two live scopes on one entity are producible — two `authorize --to` calls naming different agents, or the duplicate re-grant G-0460 reports — and nothing refuses them, so the two existing answers disagree. | yes | ADR-0047 — neither. The end names its scope by authorize-commit SHA and defaults to the sole candidate. |
| Is cancelling an epic a sovereign act, or is completion the only closure a human must make? | yes | ADR-0047 — every edge into a terminal status is sovereign, `proposed → cancelled` included. |
| What undoes an end? Re-authorizing opens a fresh scope rather than reviving the ended one, and `ended` is terminal in the scope FSM. Is that the whole answer? | no | ADR-0047 — yes. Nothing undoes an end; the inverse is a fresh grant. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Widening the closed set fires the history audit at error severity on commits already in the log, blocking every push until each is ratified. | med | One commit qualifies today. Ratify it in the same milestone that widens the set; the count only grows if the gate is deferred. |
| The gate keys on a self-declared actor — an invocation that omits `--actor` inherits the human identity from `git config` and passes through untouched. | med | Accepted here rather than mitigated. The property is shared with the shipped `proposed → active` gate and belongs to the identity substrate, not to this epic. |

## Milestones

- `M-0323` — settle closure authority on every terminal edge and the semantics of an operator-driven end, in one ADR · depends on: —
- `M-0324` — refuse a non-human actor on every edge into a terminal epic status, with the `cancel` call site and the historical ratification · depends on: `M-0323`
- `M-0325` — add an operator-driven end to `aiwf authorize`, additive alongside the auto-end · depends on: `M-0323`

`M-0324` and `M-0325` depend only on the ADR, not on each other, so their order is soft.

## ADRs produced

ADR-0047 — Gate every terminal epic edge; end a scope by naming it. Settles closure authority on all three terminal edges, and the semantics of an operator-driven scope end alongside the automatic one.

## References

- G-0646 — closed by this epic.
- G-0111 — advanced but not closed. The operator-driven end lands here; the weld it is titled for survives by the scope decision above, so its claim is corrected rather than closed at wrap.
- G-0022 — its reserved revoke verb is implemented here; the remainder of that gap's extension list is untouched.
- G-0460 — asks whether more than one live scope per entity should be legal at all; ADR-0047's validation trigger names it as the signal that would reopen the targeting rule.
- ADR-0047 — produced by M-0323; settles every question in the table above.
- ADR-0040 — prevention at the verb route, ratification at the history route.
- `internal/entity/sovereign.go`, `internal/verb/promote_sovereign_act.go`, `internal/verb/authorize.go`, `internal/cli/cliutil/provenance.go`.
