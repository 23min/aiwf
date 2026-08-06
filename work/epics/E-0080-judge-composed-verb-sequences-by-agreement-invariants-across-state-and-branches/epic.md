---
id: E-0080
title: Judge composed verb sequences by agreement invariants across state and branches
status: proposed
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
authored on one branch and judged on another. And its oracle is **monotonic** —
it asserts `aiwf check` never regresses, which by construction cannot catch a
finding carrying the wrong severity for its state.

G-0558 is the first measured instance of what that misses: `aiwf check` and
`aiwf check --fast` render opposite verdicts on the same bytes in the same
working copy, a composition no scenario walks. D-0063 is accepted and records
the direction this epic implements.

## Scope

### In scope

- The three agreement invariants D-0063 names: every read path renders the same
  verdict on the same bytes; a verdict is stable under refs the tree does not
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
  makes them safe to build while that policy is still in flight.
- Fixing G-0558 or G-0556. This epic builds the oracle that judges them; the
  fixes are their own patches.

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
- **The agreement invariants land red and must not merge red.** They fail
  against today's tree and turn green when G-0558's fix lands. Nothing from the
  first milestone merges to mainline before that fix does — the red-to-green
  flip across that merge is the evidence the oracle works, and it is what keeps
  the oracle independent of the fix it judges.

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
      agreement invariant, and it is green on mainline.
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
| The first milestone merges before G-0558 and lands red on mainline | med | Named as a constraint above and checked at that milestone's wrap gate |

## Milestones

<!-- Candidates, in execution order. Ids are allocated by aiwfx-plan-milestones;
     no id-shaped label is written here before the verb assigns one. -->

- Agreement-invariant oracle — the three properties from D-0063, red against
  today's tree · depends on: —
- Widened mutation space — seed acceptance criteria, set areas and priorities,
  edit bodies · depends on: the oracle milestone
- Two-branch-plus-merge scenario — deterministic, untagged, every push ·
  depends on: the oracle milestone
- Acceptance-criterion composition fuzz — the met-under-`tdd: required`
  invariant · depends on: the widened mutation space
- Catalog reclassification — named scenarios reserved for document-specified
  verdicts · depends on: —

## References

- G-0121 — the parent gap this epic closes.
- G-0564 — the remainder G-0121 leaves behind.
- D-0063 — widen the stress walker; keep its oracle invariant-shaped.
- G-0468 — the precedent for holding an oracle to a shape rather than a subject.
- G-0558, G-0556 — the measured instances the agreement invariants judge.
- G-0073, G-0168 — the kernel ceilings on how far the mutation space widens.
- E-0033, E-0062, E-0071 — the prior work this builds on.
- `CLAUDE.md` — the lane rules for tagged versus untagged scenarios.
