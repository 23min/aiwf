---
id: D-0063
title: Widen the stress walker; keep its oracle invariant-shaped
status: accepted
---
## Question

`cmd/stresstest` exists for workflow legality — that group is what the harness
is for, and it is the composition coverage G-0121 asks for. The catalog has
drifted away from it: five scenarios judge workflow legality, eleven judge
concurrency or fault injection.

A defect surfaced that the harness cannot catch by construction. `aiwf check`
and `aiwf check --fast` render opposite verdicts on the same bytes in the same
working copy (G-0558). No scenario composes a sequence that crosses a branch
boundary, and none exercises reference resolution at all.

Covering that class means either more named scenarios or one walker with a wider
state space. The choice is not obvious, and the state space is not the hard
part — the oracle is.

## Decision

**Widen the walker's mutation space, keep the oracle invariant-shaped, and
reserve named scenarios for specifications.**

Widen what a sequence may mutate: seed acceptance criteria, set areas and
priorities, declare edges, edit bodies, cut and merge branches. Judge the result
with properties that hold in every reachable state rather than with an expected
end state computed by the harness. Write a named scenario when the subject is a
verdict some document specifies, not when it is a state some sequence reaches.

## Reasoning

The mutation space is mechanical to widen. The oracle splits into two jobs that
scale very differently, and conflating them is what makes "widen it" sound
either trivial or impossible.

**An exact-verdict oracle needs a model.** Asserting "after this sequence the
tree is in state X and check says Y" requires something that knows the right
answer independently. For statuses that model already exists — the FSM in
`entity.ValidateTransition` — which is why the `verb-sequence` walker works at
all. For references it would mean reimplementing ADR-0030's tier rules inside
the harness: a second implementation of the kernel, drifting against the first,
and wrong in a way no test catches because it *is* the test.

**An invariant oracle needs no model.** It asserts properties that must hold in
every state without knowing what the state should be, so it costs nothing per
new axis of mutation. That asymmetry is the whole decision: widening pays off
precisely to the extent the oracle is invariant-shaped.

The invariants that reach this defect are agreement invariants, and none of them
needs to know the correct verdict:

- every read path renders the same verdict on the same bytes
- a verdict is stable under refs the tree does not need — same bytes, ref-less
  clone and full checkout, same answer
- a sequence's verdict does not depend on which branch ran which step

**Rejected: widen the state space and keep the current oracle.** `verb-sequence`
asserts that `aiwf check` never regresses. Monotonicity cannot catch a finding
carrying the wrong severity for its state, which is the entirety of G-0558 — the
finding is present in both verdicts and differs only in whether it blocks. A
wider state space under a monotonic oracle buys reachability without judgment.

**Rejected: more named scenarios as the primary route.** They do not compose:
each carries its own fixture, its own repo setup, its own oracle scaffolding,
and the catalog's five-to-eleven split shows that new scenarios accrete on the
concurrency side by default. But they are not worthless either, which is why
they are kept for specifications: as a state space widens, a random walk becomes
less likely to generate any particular composition, so a verdict that a document
pins deliberately wants a test that reaches it deliberately.

The precedent for holding an oracle to a shape rather than a subject is G-0468:
an oracle asserting how many actors get through, or how fast, measures the
runner rather than aiwf, and is refused at review. This decision applies the
same discipline on a different axis — an oracle asserting only that findings did
not increase measures monotonicity rather than correctness.

## Consequences

- A two-branch-plus-merge scenario is deterministic — no concurrency, no timing —
  so by the lane rules it lands untagged and runs on **every push**. That is the
  correct placement and it is real added cost per push, accepted deliberately
  here rather than discovered later.
- The agreement invariants are policy-independent. They hold under every option
  G-0556 is choosing between, so they can be built before that decision and
  before G-0558's fix — going red on today's tree, and turning green when
  G-0558 lands.
- `verb-sequence` seeds no acceptance criteria, so the AC-under-`tdd: required`
  invariant G-0121 names by hand is structurally unreachable today. Widening the
  mutation space is what unblocks it.
- A widened walker that declares edges will meet the milestone-only restriction
  on `depends_on` (G-0073) immediately, and has no verb for the set-at-create
  reference fields (G-0168). Both are live constraints on how far the mutation
  space can widen without kernel work.
