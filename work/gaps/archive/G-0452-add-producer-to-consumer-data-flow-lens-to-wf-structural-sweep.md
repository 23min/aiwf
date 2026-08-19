---
id: G-0452
title: Add producer to consumer data-flow lens to wf-structural-sweep
status: addressed
priority: low
addressed_by_commit:
    - aef85f87c
---
## What's missing

`wf-structural-sweep` has three lenses — reachability (dead functions), textual clones, and reasoned convergent duplication. None traces **data flow**: a value produced at one point and its consumers elsewhere. That misses a class of structural defect the other three provably cannot catch — a value that is *computed and then dropped*, a stage whose output nothing consumes, or the same derived value computed in two places.

This class is real, not hypothetical: a single-session sweep found a field produced by one stage, threaded through an options struct, and discarded at the next stage (assigned, passed as an argument, then read into the blank identifier and never used). Reachability analysis did not flag it — the field is "used" in the call-graph sense. Clone detection did not. Only a reader reasoning about the flow caught it. A producer→consumer trace flags it structurally: a value enters the graph with no consuming edge.

## Why it matters

- The three existing lenses share a blind spot precisely here: they reason about *reachability* and *similarity*, never about *whether a produced value is consumed*. Dropped values, orphan stages, and duplicate derivations sit in that blind spot.
- Keeping it *inside* the sweep — one heavy pass run infrequently — is deliberate: the value of a whole-graph data-flow trace is seeing producers and consumers together, which a piecemeal check fragments. The sweep is already the "run it rarely, look at everything" ritual; this is the lens that most rewards that framing.

## Resolution shape

Add a fourth lens to `wf-structural-sweep`, phrased **stack- and domain-agnostic** like the existing three (the skill ships into consumer repos):

> **Lens 4 — Data flow (producer→consumer).** For each value the system produces — a field, a computed/derived result, a stage's output — trace where it is consumed. Flag: **produced-but-unconsumed** (a value assigned/passed but never read for a decision); **orphan stages** (output nothing downstream consumes); **duplicate derivations** (the same value computed in two places, one of which should be the source); and **data-dependency cycles** (a producer that depends on a later consumer's output). This is the lens no reachability or clone tool reaches — a "used" value can still flow nowhere.

Constraints on the lens body: generic wording, no this-repo pipeline names, no real ids/paths — shipped-surface clean. The edit lands alongside a referencing structural test per the skill-edit-structural-test-backstop.

**How this repo runs the lens (dev-doc / gap scope, not the shipped surface):** trace the `load → check → project → apply → commit` pipeline — is every projection consumed by `apply`; does every `Plan` Op a verb emits get handled; is every options field read. The dropped-field case above is the motivating instance.

**Scope boundary.** This is a *reasoned discovery lens*, not a mechanical check. A full mechanical model of the pipeline data flow would require SSA-level Go analysis and a persisted graph-projection — both deferred / out of PoC scope per design-decisions.md. A **targeted seam check** (a policy test asserting "every emitted `Plan` Op is handled by `apply`; every options field is read") is a possible *mechanical follow-on*, bounded and SSA-free, but secondary to the reasoned lens and tracked separately if the friction earns it.

## Where to fix

- `wf-structural-sweep`'s `SKILL.md` — add Lens 4 under "The three lenses" (retitle to "The four lenses"), generic wording; update the description and anti-patterns to match.
- A referencing structural test under `internal/policies/` covering the new lens section.

## Related

- G-0450 — the `wf-structural-sweep` skill this lens extends.
- G-0447 — the convergent-duplication tax; duplicate-derivation findings overlap the pipeline projections it names.
- `wf-codebase-health` — the rubric the sweep drives; the data-flow lens has no rubric principle today, a possible rubric addition if the lens proves its worth.
