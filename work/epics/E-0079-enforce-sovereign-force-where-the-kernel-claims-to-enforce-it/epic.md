---
id: E-0079
title: Enforce sovereign force where the kernel claims to enforce it
status: proposed
---

# E-0079 — Enforce sovereign force where the kernel claims to enforce it

## Goal

Make the kernel's "`--force` is human-only" guarantee true at the moment it is
claimed, and give the finding it produces a way to be cleared. Five surfaces
state the guarantee as enforced; the verb route enforces it for one verb of four,
and no verb clears the resulting error.

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

The sovereign set is mechanical rather than a judgment: the verbs constructing a
`TrailerForce` are `add`, `promote`, `cancel` and `authorize`. `contract bind`,
`contract recipe` and `update --remove` also declare `--force`, emit no trailer,
and mean force-replace — a different word spelled the same.

Three shapes are in play, and conflating them is why this drifted. The entity FSM
is cell-keyed `(Kind, FromState, Verb)` in `internal/workflows/spec`, with a Pin
registry and a bijection meta-test proving each cell has a test. Sovereign-act
shape (`entity.IsSovereignActShape`) is a closed tuple set over FSM edges and fits
that key. Trailer coherence is a predicate over *(actor role × trailer
presence-vector)* — no kind, no from-state, no verb. It has no cell, so no Pin, so
nothing indexed the rule and nothing noticed the guard reached one verb of four.

## Scope

### In scope

- **Finish the coherence wiring.** `CheckTrailerCoherence` on the assembled
  trailer set of every verb that constructs a `TrailerForce`.
- **Ratification.** Add `provenance-force-non-human` to the `ackedSHAs` consumer
  roster so `aiwf acknowledge illegal` clears it with a human's written reason.
  Needed independently of the wiring: the rule walks git history and fires on
  commits no verb produced.
- **Correct the surfaces that claim enforcement that does not exist** — the
  chokepoint column of the audit catalogue's force rule, the two claims in
  `CLAUDE.md`, the claim in `design-decisions.md`, and `promote --help`'s
  "coherence checks still run".
- **Re-aim the sovereign dispatcher policy** (G-0534) at a code reference: every
  verb constructing a `TrailerForce` routes through `CheckTrailerCoherence`.
- **A cell registry for rule spaces not keyed `(Kind, FromState, Verb)`**, so a
  rule without an FSM coordinate still gets the Pin-and-bijection discipline.
  Recorded as decided; its placement is an open question below.
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
- Every rule-space cell introduced by this epic is pinned by a test, under the
  same bijection discipline the branch spec already has.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the cell registry for non-FSM-keyed rule spaces belong in this epic or its own? | no | Milestone planning. Decided to build it; only placement is open, and it can spawn an epic without blocking the wiring. |
| Does `cancel` need treatment distinct from `promote`, given both emit the force trailer through the shared `transitionTrailers` helper? | yes | Milestone planning, by reading the shared path. Decides whether the wiring is one change or two. |
| Where does the coherence call sit in each verb — the verb body, or the shared plan-assembly seam that builds trailers? | yes | Milestone planning. A single seam is preferable if one exists, since per-verb calls are the shape that drifted. |
| Does adding the rule to the ack roster silence it too broadly, since an acknowledgement is keyed on SHA alone? | no | Accepted for now: an acknowledgement is a judgment about a commit, which is what the existing consumers assume. Revisit if someone needs to accept one rule while another still blocks. |

## Risks

| Risk | Impact | Mitigation |
| --- | --- | --- |
| The wiring makes a previously-succeeding command fail, breaking a consumer's automation | med | The ADR records the change; the CHANGELOG states it plainly. Automation legitimately forcing as a non-human actor was already blocked at push, so no working pipeline is broken — only the failure point moves earlier. |
| A verb's trailer set is assembled in more than one place, so a single coherence call misses a path | med | The bijection check is keyed on trailer construction, not on call sites, so an unrouted construction fails the gate. |

## Milestones

Not yet allocated; ids are assigned when `aiwfx-plan-milestones` runs. Candidate
deliverables, in execution order:

- The coherence wiring across the sovereign verbs, with the exhaustive
  cross-product test and the construction-to-guard bijection check.
- The ratification path: the ack roster entry, its coverage, and the real-repo
  test that the acknowledgement clears the finding.
- The surface corrections, including the folded-in finding-hint audit, and the
  sovereign dispatcher policy re-aimed at a code reference.
- The cell registry for rule spaces without an FSM coordinate — last, and the
  candidate for promotion to its own epic.

## References

- G-0534 — the sovereign policy's guard predicate, subsumed here.
- G-0333 — the Tier-1 / Tier-2 override boundary; its finding-hint audit is folded
  into this epic, the rest stays with the gap.
- G-0023 — delegated force, deliberately out of scope.
- [`docs/design/provenance-model.md`](../../../docs/design/provenance-model.md) —
  the trailer coherence rules this epic enforces.
- [`docs/design/legal-workflows-audit.md`](../../../docs/design/legal-workflows-audit.md)
  — the rule catalogue whose force entry names a chokepoint that was never built.
