---
id: E-0080
title: Judge composed verb sequences by agreement invariants across state and branches
status: active
---
## Goal

Make workflow-level legality mechanically checkable past a single axis of state
and a single branch, so a sequence of individually-legal verbs that leaves the
tree in a state nobody intended is caught by the harness rather than by a
downstream consumer.

Addresses G-0121. Its two non-mechanizable remainders move to G-0564.

## Context

The kernel pins per-entity legality tightly — six per-kind FSMs, the AC and
TDD-phase FSMs, the cross-cutting rules in `internal/check`, and the policy
tests in `internal/policies`. What is pinned far more loosely is what happens
across a *sequence* of individually-legal moves.

Prior work got most of the way there. E-0033 built `internal/workflows/spec`
with a closed predicate vocabulary and real positive and negative drivers;
M-0130 added the `fsm-history-consistent` tree invariant; E-0062 built the
`cmd/stresstest` verb-sequence walker, which composes real multi-step verb
chains against a correctness oracle; E-0071 added `milestone tdd` policy flips
to it.

A measurement on 2026-08-06 found the walker narrower than that summary implies,
in three specific ways. It moves **one axis of state** — `status`, the six FSMs
— so the reachable space is statuses, not references, areas, priorities,
`depends_on` edges, or bodies. It stays on **one branch**, so no reference is
ever evaluated in a context different from the one that authored it; every
two-branch scenario in the catalog is about contention, not about a reference
authored on one branch and judged on another. And its oracle reads **one
surface** — it runs `aiwf check` alone and judges the findings against a curated
baseline, flagging any error-severity finding and any warning outside it. That
is an absolute allowlist applied to each state on its own terms, so it does
judge what a wider walk reaches; what it cannot do is compare two read paths, so
a disagreement between `aiwf check` and `aiwf check --fast` on the same bytes is
invisible to it by construction.

G-0558 is the first measured instance of what that misses: `aiwf check` and
`aiwf check --fast` render opposite verdicts on the same bytes in the same
working copy, a composition no scenario walks. D-0063 is accepted and records
the direction this epic implements.

Both known instances are now repaired on mainline — G-0558 by giving a ref-less
surface the `unresolved-unverified` subcode to declare it did not build a tier,
G-0556 by splitting a cross-branch hit on whether its branch is published. That
changes what this epic is for, and sharpens it. It is not here to catch those
two; it is here so the next one is caught by the harness rather than by a
reader. It also fixes the shape of the agreement property: with a legitimate way
to decline a judgment in the vocabulary, agreement cannot mean identical finding
sets.

## Scope

### In scope

- The three agreement invariants D-0063 names: no two read paths contradict each
  other on the same bytes; a verdict is stable under refs the tree does not
  need, so a ref-less clone and a full checkout agree; a sequence's verdict does
  not depend on which branch ran which step.
- Widening the walker's mutation space beyond `status`: seed acceptance
  criteria, set areas and priorities, edit bodies.
- A two-branch-plus-merge scenario. It is deterministic — no concurrency, no
  timing — so by the lane rules it lands untagged and runs on every push.
- The acceptance-criterion composition invariant G-0121 states by hand: no AC is
  `met` under a `tdd: required` milestone whose `tdd_phase` is not `done`, after
  any legal verb sequence.
- Reclassifying the scenario catalog against D-0063's rule that a named scenario
  is reserved for a verdict some document specifies.

### Out of scope

- The declarative enumeration of blessed workflows, and multi-host choreography.
  Both move to G-0564; neither has a settled design question behind it.
- Cross-kind `depends_on` edges (G-0073) and mutation verbs for the
  set-at-create reference fields (G-0168). D-0063 names both as ceilings on how
  far the mutation space can widen. They are constraints here, not deliverables.
- Any change to the reference-resolution tier policy of ADR-0030 or ADR-0041.
  The agreement invariants are policy-independent by construction, which is what
  let them survive G-0556's classification change arriving between this epic's
  planning and its implementation.
- Repairing any disagreement the invariants find. G-0558 and G-0556 fixed the
  two known instances; a new one is filed as its own gap, not fixed here.

## Constraints

- **The oracle stays invariant-shaped.** No exact-verdict oracle, and in
  particular no reimplementation of the tier rules inside the harness — that
  would be a second implementation of the kernel, drifting against the first,
  and wrong in a way no test catches because it *is* the test. This is D-0063's
  central rejection and the reason widening pays off at all.
- **No oracle measures the runner.** G-0468's precedent: an assertion about how
  many actors get through, or how fast, measures the machine rather than aiwf
  and is refused at review.
- **Lane choice is a cost decision, not a correctness one.** Per CLAUDE.md, a
  scenario goes behind `//go:build stress` when it drives real concurrently-
  scheduled subprocesses and would slow every push, and stays untagged
  otherwise. The two-branch scenario is untagged, and its per-push cost is
  accepted here deliberately rather than discovered later.
- **Agreement is non-contradiction, not identity.** A surface that reports
  `unresolved-unverified` is declining to judge, not disagreeing; two surfaces
  contradict only when both make substantive claims that cannot both hold. An
  identity-shaped property fires on correct trees — G-0558 gave surfaces a way
  to decline, and G-0556 made the full check's severity differ from a ref-less
  one on the same subject — so identity is refused at review alongside an
  exact-verdict oracle.
- **A green run is not evidence.** Both known instances are repaired on
  mainline, so every invariant here must be demonstrated failing against a
  constructed violation. An invariant observed only passing is indistinguishable
  from one that cannot fire.

## Success criteria

- [ ] Every agreement invariant listed under *In scope* is asserted by the
      harness, and each has been observed failing against a tree that violates
      it — not merely passing against one that does not.
- [ ] The walker composes sequences mutating every axis listed under *In scope*,
      or the axis is recorded as blocked against G-0073 or G-0168 rather than
      worked around.
- [ ] The acceptance-criterion invariant holds under composition, and the walker
      can reach a state that would violate it, so the assertion is not vacuous.
- [ ] `aiwf check` and `aiwf check --fast` are covered by the read-path
      agreement invariant; it is green on mainline, and it stays green across
      the `unresolved-unverified` and `cross-branch-local-only` classifications
      rather than reporting them as a disagreement.
- [ ] Every scenario in the catalog carries an explicit classification under
      D-0063's named-scenario rule.
- [ ] G-0121 is `addressed`, with its remainder carried by G-0564.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| How far does the mutation space widen before G-0073 and G-0168 stop it? | no | Measured during the widening milestone; each blocked axis is recorded against the owning gap rather than worked around |
| Is the catalog reclassification a retitle, a retirement, or a move? | for the last milestone only | Decided from the census that milestone produces; a retirement earns its own decision entity |
| Does the two-branch scenario's per-push cost stay acceptable as the catalog grows? | no | Measure it when the scenario lands and record the number; the lane rule decides per scenario |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The oracle drifts toward exact-verdict assertions as coverage pressure grows | high | D-0063 rejects it explicitly; an exact-verdict assertion is a review refusal, same as a runner-measuring one |
| The two-branch scenario's wall time later gets it tagged, hiding the class again | med | Cost accepted deliberately in D-0063; measure and record it rather than discovering it under pressure |
| Every invariant passes from the day it lands, so the suite reports coverage it does not have | high | Both known instances are already fixed on mainline; each invariant carries a constructed-violation test, and a green-only invariant is a review refusal |

## Milestones

- `M-0300` — the invariant-oracle seam plus the two properties that judge a
  single tree · depends on: —
- `M-0301` — widen what a sequence may mutate beyond `status` · depends on:
  `M-0300`
- `M-0302` — compose across a branch boundary; assert verdict independence ·
  depends on: `M-0300`
- `M-0303` — assert the acceptance-criterion composition invariant · depends on:
  `M-0301`
- `M-0304` — classify every catalog scenario against the named-scenario rule ·
  depends on: —

`M-0301` and `M-0302` are parallel once `M-0300` lands; `M-0304` is parallel
throughout.

The third agreement invariant D-0063 names — a sequence's verdict does not
depend on which branch ran which step — ships in `M-0302` rather than `M-0300`.
It is unreachable until a sequence can cross a branch boundary, so asserting it
earlier would be a property nothing generates a condition for.

## References

- G-0121 — the parent gap this epic closes.
- G-0564 — the remainder G-0121 leaves behind.
- D-0063 — widen the stress walker; keep its oracle invariant-shaped.
- G-0468 — the precedent for holding an oracle to a shape rather than a subject.
- G-0558, G-0556 — the measured instances the agreement invariants judge.
- G-0073, G-0168 — the kernel ceilings on how far the mutation space widens.
- E-0033, E-0062, E-0071 — the prior work this builds on.
- `CLAUDE.md` — the lane rules for tagged versus untagged scenarios.
