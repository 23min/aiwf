---
id: G-0258
title: no wf-* ritual to audit whether existing assertions can actually fail
status: open
---
## What's missing

The `wf-rituals` plugin has no ritual that adversarially audits whether the
assertions and specs a unit *already* carries can actually fail. `wf-tdd-cycle`
audits **branch coverage** — proof that a line *ran* — but coverage says nothing
about whether an assertion would *catch a bug* on that line. A fully-covered test
suite can still be vacuous: tautological assertions (`result != nil` standing in
for `result == expected`), over-narrowed antecedents (the one input where the bug
can't manifest), `∀x. true`-shaped properties. Nothing in the family detects this.

## Why it matters

This is the skill-level stop-gap for loom-light's **actual differentiator** —
vacuity / gaming detection (see `docs/pocv3/plans/loom-light-plan.md` §1.4). The
live failure under optimisation pressure is **endogenous weakening**: the same
agent authors the implementation *and* its tests and is graded on the tests
passing, so the tests are suspect by construction. Most LLM-spec literature frames
weak specs as *capability* failures (the model tried and couldn't); the dangerous
case is the model that *could* write a strong assertion but writes a weak one
because the weak one is cheaper to pass. A checking ritual that forces an
adversarial stance toward the agent's own assertions is the cheapest available
guard.

It pairs naturally with the branch-coverage audit: coverage is *necessary*
(the line ran), vacuity is the missing *sufficiency* check (an assertion would
have caught the bug). Together they are far closer to "the tests mean something"
than either alone.

## What this is / is not

- **A checking ritual, not an authoring one.** It *audits* assertions that already
  exist; `wf-property-test` (G-0257) *produces* the strong assertions. Complementary.
- **Weaker floor than `wf-property-test`.** Its output is a report a human reads, so
  it still depends on LLM judgement — the same residual gap `wf-rethink` (G-0256)
  carries. That gap is exactly what loom-light's mechanical verifier exists to close;
  the skill must surface it, not over-claim.
- **Not a replacement for mutation-testing tools.** Where a real mutation-testing
  harness exists (the repo's `mutate-hunt` workflow, gremlins, Stryker, mutmut), the
  skill should defer to it and read its survivors — the manual probe is the stop-gap
  for units and repos without one.

## Fix shape

Ship `wf-vacuity` as a `wf-rituals` skill at
`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-vacuity/SKILL.md`,
in the `wf-*` house style (frontmatter `name:` + `description:`; `When to use`;
`Workflow`; `Anti-patterns`; `Constraints` with 🛑 callouts), with two probes:

1. **Mutation probe** — deliberately break the implementation (negate a guard,
   off-by-one, return a constant, drop a step) and confirm at least one test goes
   red. A surviving mutant is a hole in the assertions. Defer to a real
   mutation-testing tool where one exists.
2. **Tautology / narrowing probe** — read each assertion and ask "could this pass
   for the wrong reason?": tautological checks, over-narrowed antecedents,
   existence-not-value assertions, properties that can't fail.

Output is a list of weak assertions, each paired with the mutant that survives it,
routed to the human (or back into `wf-tdd-cycle` to strengthen). Composed by
`wf-review-code` (a diff that adds tests gets a vacuity probe), by `wf-tdd-cycle`
right after the branch-coverage audit, and by `wf-rethink` when it doesn't trust
its obligation-tests. Framework-agnostic; no manifest or test edits required.
