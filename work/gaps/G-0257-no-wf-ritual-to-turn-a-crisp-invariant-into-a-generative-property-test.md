---
id: G-0257
title: no wf-* ritual to turn a crisp invariant into a generative property test
status: open
---
## What's missing

The `wf-rituals` plugin has no ritual for turning a unit's **crisp invariant**
into a *generative* test — one that fabricates many inputs and asserts a property
holds across all of them, rather than checking a single hand-picked example.
`wf-tdd-cycle` produces example-based tests and audits branch coverage; neither
it nor `wf-review-code` helps an author recognise a conservation / round-trip /
idempotence / monotonicity invariant and express it as a property or metamorphic
test.

## Why it matters

This is the **only** stop-gap in the verification-ritual family with a real
*mechanical floor*. `wf-rethink` (G-0256) and the proposed `wf-vacuity` both
emit reports a human reads; their safety depends on LLM behaviour. A property
test, once authored, **runs in CI and fails loudly** — it does not depend on the
assistant remembering anything. It is a value-gate over the *sampled* value space:
it does not prove `∀` the way a verifier (loom-light's job) would, but
sampled-and-mechanical is a far stronger guarantee than unsampled-and-LLM-judged,
and it is the closest skill-level approximation of loom-light's correctness
value-gate. See `docs/pocv3/plans/loom-light-plan.md` for the property this
approximates.

It also fits existing house precedent: the repo already runs `Fuzz*` and property
tests (G44 — `internal/entity/transition_property_test.go` asserts the FSM closed-set
invariants exhaustively, the `Fuzz*` targets exercise parsers). The skill
systematises that habit instead of leaving it a one-off.

## What this is / is not

- **Scoped to the crisp-invariant subset** — parsers, ledgers, state machines,
  allocators, serializers, comparators, id allocators. Glue, IO, and UI are out;
  forcing a property onto them is the high-cost-low-yield trap (loom-light §1.5).
- **Samples, does not prove.** A property test checks the invariant over generated
  inputs, not over the whole space. The exhaustive-proof gap is exactly what a
  verifier closes; the skill must say so and not over-claim.
- **An authoring ritual, not a checking one.** It *produces* a gate; the proposed
  `wf-vacuity` *audits* assertions that already exist. The two are complementary.

## Fix shape

Ship `wf-property-test` as a `wf-rituals` skill at
`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-property-test/SKILL.md`,
in the `wf-*` house style (frontmatter `name:` + `description:`; `When to use`;
`Workflow`; `Anti-patterns`; `Constraints` with 🛑 callouts), covering:

1. **Identify the invariant-bearing unit** — the scope gate; skip glue/IO/UI.
2. **Name the property** in a canonical family — conservation, round-trip,
   idempotence, order-independence, monotonicity, invariant-preservation.
3. **Express it against the language's generator** — Go `testing/quick` / native
   `Fuzz*`, Hypothesis, fast-check, proptest — and let it shrink failures.
4. **Commit the test (and any seed corpus).** The artifact is the gate.
5. **State the honest limit** — sampling, not proof; reference `wf-rethink` and
   `wf-vacuity` for the design-quality and assertion-strength companions.

Authored framework-agnostic (ships to non-aiwf consumers too), composable by
`wf-tdd-cycle` at the RED step when the unit has a crisp invariant. No manifest or
test edits required (recursive `//go:embed` auto-discovers it).
