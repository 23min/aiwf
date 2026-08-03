---
id: G-0526
title: aiwf ships source-discipline rules as prose with no seam to enforce them
status: open
priority: medium
---
## What's missing

aiwf tells a consumer how to hold their own source code and gives them no way to
enforce any of it. The rules travel as prose — the always-on guidance fragment,
the `wf-*` ritual skills, the code-health rubric — and the enforcement stays
here, in `internal/policies/`, a package nothing outside itself imports and
nothing materializes into a consumer tree.

The clearest instance is already shipped. The rule "state the conclusion, not the
drafting history" reaches a consumer as guidance prose; its mechanical half,
`comment-history-attrition`, runs only against this repo's Go source. A consumer
receives the judgment and none of the check.

There is no seam that would close it. `aiwf check`'s rules are a fixed Go set
over the planning tree; hooks run `aiwf check`; contracts verify data shapes
against fixtures. None of them can express "every test package in this repo
declares its own environment", which is the class of rule `internal/policies/`
exists to hold.

## Why it matters

The framework's stated principle is that correctness must not depend on an LLM
remembering to invoke a skill. That holds for the planning tree, where the
pre-push hook and `aiwf check` are authoritative. It does not hold for anything
aiwf says about a consumer's source, where every rule it ships is advisory and
survives only if an assistant reads it and chooses to comply.

## What the resolution is not

Shipping the rules. The checks are per-codebase: a Go AST scan means nothing in a
TypeScript repo, and even the language-generic ones bake in assumptions about
this tree's layout. `docs/` as a non-implementation path is already the recorded
failure of that reflex.

Nor a shipped chokepoint corpus, whatever its contents. The classification in
[`growth.md`](../../docs/design/growth.md) finds most of this repo's chokepoints
are mandates, satisfied only by adding an artifact once per subject, with no
retirement path. Exporting that wholesale hands a consumer a cost that recurs
with their codebase and a decision they never made.

## Resolution shape

The portable thing, if any, is the seam and the classification — not the rules.
Three candidate answers, in ascending cost:

1. **Nothing.** Consumers bring their own enforcement; aiwf's contribution stays
   advisory by design, and the asymmetry above is accepted and written down. This
   is the current state, undecided rather than chosen.
2. **A ritual that teaches the shape.** The mandate/ban/uniqueness/exactness
   split, and the rule that a mandate lands with an owner and a retirement
   trigger, are stack-agnostic and already authored. Shipping them is a prose
   addition that costs once.
3. **A declared-command seam.** A consumer names their own checks in
   `aiwf.yaml`, aiwf runs them and reports findings in its own envelope. The
   precedent is contracts, which ship zero validators and let the user declare
   the binary — the same posture, applied to source rather than to schemas.

Option 3 is the only one that makes a consumer's rule mechanical, and it is also
the one that adds a surface aiwf then maintains forever. Whether that trade is
worth taking is the decision this gap opens; it should not be settled by
reflex in either direction.

Settling it should also determine whether the narrower question about
test-parallelism discipline is an instance of this one or genuinely separate.

## Prior threads

- G-0104 asks the same ship-or-BYO question for a single discipline and parks it
  until a second consumer asks.
- G-0445 records a shipped gate baking in a path convention true only here.
- D-0053 is the worked case of a mandate accepted with a named retirement
  trigger, which is the discipline any answer here has to satisfy.
