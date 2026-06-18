---
id: G-0256
title: no wf-* ritual for re-evaluating a design unit against pinned obligations
status: open
---
## What's missing

The `wf-rituals` plugin ships four engineering rituals — `wf-tdd-cycle`,
`wf-review-code`, `wf-patch`, `wf-doc-lint` — but none of them addresses
*design-quality drift*: the way an LLM coding agent, behaving as a greedy
optimizer, converges a unit on a **local optimum**. Each individual edit is
locally reasonable, so the result is *correct*, yet globally misshapen — carrying
incidental complexity, single-caller abstractions, and defensive layers that
exist only because of the order in which things were built. `wf-review-code`
reviews a diff for correctness and convention; it does not reconstruct a unit
from intent to separate path-dependent residue from load-bearing structure.

The naive form of the fix — "rewrite it from scratch and keep it if it feels
cleaner" — is worse than nothing: the judgment of "cleaner" comes from the same
greedy optimizer that produced the local optimum, so a from-scratch rewrite is
often just a *differently*-local optimum, shipped with a fresh regression. The
missing ritual is a disciplined version with an **obligation gate**: pin what the
unit owes (behavior, public interface, invariants, tests) *before* looking at the
current structure, and permit a rewrite only if it provably preserves every one.

## Why it matters

The problem this attacks is real in downstream repos and is the implementation-
layer twin of the failure mode the (aspirational, undecided) **loom-light**
proposal targets — see `docs/pocv3/plans/loom-light-plan.md`. loom-light supplies
a mechanical *value-gate over code correctness*; it does not yet exist and its
home (bundled vs standalone) is unpinned. A ritual skill is the cheapest
available stop-gap for the adjacent *design-quality* problem, and aiwf already has
the delivery mechanism: `wf-*` rituals materialize into every consumer repo on
`aiwf init` / `aiwf update` from one embedded source of truth (ADR-0014 /
ADR-0016), so shipping it here is how downstream repos get it without copy-paste
drift.

## What this is not

`wf-rethink` is a **design-quality / anti-accretion ritual, not a correctness
value-gate.** Passing a rethink says nothing about whether the code satisfies a
stated claim — that is the orthogonal property a verifier (loom-light's job)
checks. The skill must be framed honestly as the former and must not borrow the
"hand-rolled value gate" framing, which oversells it and miscategorizes it.

It is also the first `wf-*` ritual that is **pure model-judgment with no
mechanical floor** (`wf-tdd-cycle` has the branch-coverage audit; `wf-review-code`
has the `file:line` discipline). Its obligation gate is self-graded, and on
under-tested code the obligation list is enumerated from the same code being
judged. That residual gap is exactly what loom-light exists to close; the skill
must surface it rather than paper over it.

## Fix shape

Ship `wf-rethink` as a fifth `wf-rituals` skill at
`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-rethink/SKILL.md`,
in the `wf-*` house style (frontmatter `name:` + `description:`; `When to use`;
`Workflow`; `Anti-patterns`; `Constraints` with 🛑 callouts), with these
load-bearing design choices:

1. **Pin obligations before looking at the current structure** — the ordering is
   the whole point; obligations back-fitted to the existing code are worthless.
2. **The rewrite is gated behind explicit human approval** — verdict → present
   the concrete win → *wait for go* → implement. The drafted form auto-applies on
   a `rewrite` verdict, which collapses decision and action into one step and
   violates the per-mutation gate discipline every other ritual honors. 🛑
3. **Obligation-incompleteness defaults to keep** — when obligations cannot be
   enumerated from tests, types, or written invariants (i.e. they are inferred
   from the same code being judged), the correct verdict is `keep`, stated as a
   🛑 constraint, not a footnote.
4. **Default to keep; a rethink that changes nothing is a successful audit** —
   the counter-pressure against the agent's bias toward churn.
5. **One bounded unit only** — never rethink the whole codebase at once.

No manifest or test edits are required (recursive `//go:embed embedded-rituals`
auto-discovers it; the `skill_coverage` policy is scoped to `aiwfx-*` and does not
gate `wf-*` skills; `body-prose-id` does not scan skill bodies).
