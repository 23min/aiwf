---
id: E-0079
title: Enforce sovereign force where the kernel claims to enforce it
status: active
---

# E-0079 — Enforce sovereign force where the kernel claims to enforce it

## Goal

Make the kernel's "`--force` is human-only" guarantee true at the moment it is
claimed, and give the finding it produces a way to be cleared. Every surface
listed in M-0293's table states the guarantee as enforced; the verb route
enforces it for one verb of four, and no verb clears the resulting error.

## Context

`CheckTrailerCoherence` (`internal/verb/coherence.go`) implements the rule as
`CoherenceRuleForceNonHuman`. Its introducing commit describes it as running "on
an assembled trailer set, before commit," with "verb refusal paths surface the
message" — a general pre-commit chokepoint. It is called from `authorize.go` and
`auditonly.go` only.

Measured against the shipped binary, on a ritual branch with an active scope:

| invocation | outcome |
| --- | --- |
| `aiwf authorize E-… --to … --force --reason … --actor ai/claude` | refused at runtime |
| `aiwf promote E-… active --force --reason … --actor ai/claude` | committed, carrying `aiwf-actor: ai/claude` + `aiwf-force:` |

The forced commit then raises `provenance-force-non-human` at error severity, so
the push is blocked. `aiwf acknowledge illegal <sha> --reason "…"` does not clear
it: the rule is absent from the `ackedSHAs` consumer roster
(`internal/check/acks.go`). The remaining exits are re-authoring the commit or
amending a human actor into it — history rewrites, in a repo whose tooling is
policed against them. The act is reachable, blocking, and has no undo.

The check-time backstop cannot substitute for the verb-time guard, and not by
oversight: `projectionFindings` (`internal/verb/common.go:124`) runs `check.Run`
over the *tree* before and after. Trailer-shaped rules need git history, so
`provenance-force-non-human` is structurally invisible to that gate. Coherence on
the assembled trailer set is the only runtime mechanism available.

The sovereign set is mechanical rather than a judgment, and it is smaller than the
verb count suggests. Three sites construct a `TrailerForce`: `transitionTrailers`
(`internal/verb/promote.go`), which serves `promote`, `cancel` and both
AC-granularity transitions; an inline site in `add`; and an inline site in
`authorize`. `contract bind`, `contract recipe` and `update --remove` also declare
`--force`, emit no trailer, and mean force-replace — a different word spelled the
same.

The guard did not drift by inattention; it was never copyable. A verb's trailer
set is incomplete when the verb returns. `gateAndDecorate`
(`internal/cli/cliutil/provenance.go`) appends `aiwf-principal`,
`aiwf-on-behalf-of`, `aiwf-authorized-by` and `aiwf-scope-ends` afterwards, so a
`CheckTrailerCoherence` call placed inside `promote` would see no principal and
refuse every legitimately-authorized non-human actor under
`principal-missing-for-non-human-actor`. `authorize` can call the guard at the
verb layer only because it is absent from the `ProvenanceContext` roster and
assembles a complete set itself — as do `archive` and `import`. The two shapes
meet at exactly one point downstream of both: `verb.Apply`, whose single
production caller is `internal/cli/cliutil/apply.go`.

A guard at that seam enforces the rules predicated on a force trailer, not the
whole coherence rule set (D-0060). Membership is decided by satisfiability: a
rule belongs at the seam only if every verb reaching it has some invocation that
satisfies it. The force rules qualify, because a verb emitting no force trailer
satisfies them vacuously. The principal rules do not — the contract verbs never
pass through the provenance-decoration layer and register no flag that could
supply a principal, so enforcing those at the seam closes them outright rather
than constraining them. The history-walking audit keeps reporting the rest
after the fact — with one exception it has never covered: `audit-only` alongside
`force` has no history-walking counterpart at all.

Two code comments are load-bearing and wrong, and one of them is why a second gate
has a hole. `requireHumanActorForSovereignAct`
(`internal/verb/promote_sovereign_act.go`) deliberately steps aside for `--force`
on the stated grounds that the coherence rule "ensures non-human + --force still
fails at the coherence chokepoint, so the override path is human-only by
construction." That is a reasoned handoff to a guard `promote` never calls, so the
sovereign-act gate declines to check precisely where nothing else does. `add`'s
force branch states the opposite stance — that the check-time audit is "the same
backstop every other --force path relies on rather than a verb-time human-actor
gate here" — contradicting `authorize` in the same package.

Three shapes are in play, and conflating them is why this drifted. The entity FSM
is cell-keyed `(Kind, FromState, Verb)` in `internal/workflows/spec`, with a Pin
registry and a bijection meta-test proving each cell has a test. Sovereign-act
shape (`entity.IsSovereignActShape`) is a closed tuple set over FSM edges and fits
that key. Trailer coherence is a predicate over *(actor role × trailer
presence-vector)* — no kind, no from-state, no verb. It has no cell, so no Pin, so
nothing indexed the rule and nothing noticed the guard reached one verb of four.

## Scope

### In scope

- **Finish the coherence wiring.** The force-predicated rules at `verb.Apply`,
  the one seam downstream of both trailer-assembly shapes, so a sovereign act is
  refused where it is attempted rather than reported once it has landed.
- **Ratification.** Add `provenance-force-non-human` to the `ackedSHAs` consumer
  roster so `aiwf acknowledge illegal` clears it with a human's written reason.
  Needed independently of the wiring: the rule walks git history and fires on
  commits no verb produced.
- **Correct the surfaces that claim enforcement that does not exist**, so each
  names the seam that actually refuses rather than a chokepoint that was never
  built. M-0293's table is the enumeration; it spans the kernel's own
  documentation, the audit catalogue, a flag's help text, and the two
  contradicting code comments named in *Context*.
- **Settle the sovereign dispatcher policy** (G-0534). Its subject narrows once
  the guard sits at `verb.Apply`: routing is then structural rather than policed,
  and what is left to assert is that no production path reaches a commit
  bypassing that seam — which the seam's own chokepoint already asserts, so the
  policy is retired rather than re-aimed (D-0061).
- **One declaration for the coherence rule set**, so the domain's trailer axis,
  the seam's force-predicated subset, and the reachability roster all derive from
  it instead of from three hand-maintained copies. The Pin-and-bijection
  machinery the branch spec carries is deliberately not mirrored (D-0062).
- **One ADR** recording the stance: sovereign acts are prevented at the verb
  route and ratifiable at the history route.

### Out of scope

- **Delegated force** (G-0023, `aiwf authorize --allow-force`). That changes the
  provenance model; this epic makes the current model true.
- **The provenance model extension surface** (G-0022).
- **The Tier-1 / Tier-2 `--force` override boundary documentation**, which stays
  with G-0333. Only that gap's finding-hint audit is folded in here.
- **Retiring or re-aiming the other policies G-0535 tracks.** Unrelated subjects.

## Constraints

- The wiring must not change what `--force` overrides, only who may wield it.
  Tier-1 / Tier-2 semantics are G-0333's subject and stay untouched.
- `contract bind`, `contract recipe` and `update --remove` must keep working for
  non-human actors. Their `--force` is force-replace and emits no sovereign
  trailer; sweeping them in would break legitimate automation.
- The ratification path is human-only, carries a written reason, and records a
  separate commit rather than rewriting the acknowledged one — matching how
  `acknowledge illegal` and `acknowledge mistag` already behave.
- No finding may be downgraded to make this pass. `provenance-force-non-human`
  stays at error severity; the epic adds a way to clear it, not a way to ignore it.

## Success criteria

- A forced act by a non-human actor is refused at runtime by every verb that
  constructs a sovereign force trailer, demonstrated against the real binary.
- The coherence rules are covered by an exhaustive cross-product over their input
  domain, so coverage is a property of the test's construction rather than of the
  author's diligence.
- A mechanical check fails if a verb constructs a sovereign force trailer without
  routing through the coherence guard.
- A human can clear `provenance-force-non-human` on a commit through a verb, with
  the reason recorded, and the finding is gone on the next check run.
- Every surface listed in the *In scope* correction item states what the kernel
  actually does.
- The coherence rule set has one declaration that its trailer axis, the subset
  the seam enforces, and its reachability check all derive from, so a rule added
  without an entry fails by name rather than reading as covered.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does adding the rule to the ack roster silence it too broadly, since an acknowledgement is keyed on SHA alone? | no | Accepted for now: an acknowledgement is a judgment about a commit, which is what the existing consumers assume. Revisit if someone needs to accept one rule while another still blocks. |
| Is the check-time absence of `audit-only-with-force` a gap of its own? | no | Surfaces during the wiring, which covers it at verb time regardless. Decide there whether the history-walking side needs it too. |

## Risks

| Risk | Impact | Mitigation |
| --- | --- | --- |
| The wiring makes a previously-succeeding command fail, breaking a consumer's automation | med | The ADR records the change; the CHANGELOG states it plainly. Automation legitimately forcing as a non-human actor was already blocked at push, so no working pipeline is broken — only the failure point moves earlier. |
| A verb's trailer set is assembled in more than one place, so a single coherence call misses a path | med | The guard sits inside `verb.Apply` rather than in its callers, so every path reaching a verb commit passes it by construction; a policy holds that seam singular and guarded. |

## Milestones

Not yet allocated; ids are assigned when `aiwfx-plan-milestones` runs. Candidate
deliverables, in execution order:

- The coherence wiring across the sovereign verbs, with the exhaustive
  cross-product test and the construction-to-guard bijection check.
- The ratification path: the ack roster entry, its coverage, and the real-repo
  test that the acknowledgement clears the finding.
- The surface corrections, including the folded-in finding-hint audit, and the
  sovereign dispatcher policy retired (D-0061).
- The single declaration the coherence rule lists derive from — last.

## References

- G-0534 — the sovereign policy's guard predicate, subsumed here.
- G-0333 — the Tier-1 / Tier-2 override boundary; its finding-hint audit is folded
  into this epic, the rest stays with the gap.
- G-0023 — delegated force, deliberately out of scope.
- [`docs/design/provenance-model.md`](../../../docs/design/provenance-model.md) —
  the trailer coherence rules this epic enforces.
- [`docs/design/legal-workflows-audit.md`](../../../docs/design/legal-workflows-audit.md)
  — the rule catalogue whose force entry names a chokepoint that was never built.
