# More is much more

> **Status:** note
> **Audience:** anyone watching the volume of LLM output increase and wondering whether the right human response is to produce more or to ratify better.
> **Tags:** #note #hitl #software-development

---

## Abstract

Mies van der Rohe's "less is more" was a credo of restraint — strip ornament, expose structure, let materials speak. Robert Venturi countered "less is a bore." With LLMs producing artifacts at near-zero cost, the natural first instinct is "now we can have more." On reflection, that instinct understates the shift: more isn't just more. *It's much more* — multiplicatively more, because LLM output compounds across sessions, parallelism, and revision. The interesting question isn't whether to embrace abundance or restraint; it's *what kind* of more we're getting and what the human's response should be. This note argues that the LLM era flips Mies's credo without endorsing Venturi's: we get much more output but should produce *fewer, more coherent* artifacts to ratify. More output, fewer artifacts. The leverage is in the ratification, not the production.

---

## The original

Ludwig Mies van der Rohe popularized "less is more" in mid-20th-century architecture. The credo wasn't really about quantity; it was about restraint as a discipline. Strip ornament. Expose structure. Let one well-chosen material speak louder than ten competing ones. The Barcelona Pavilion, the Farnsworth House, the Seagram Building — all argue the same point in different vocabularies.

Robert Venturi's *Complexity and Contradiction in Architecture* (1966) was the canonical counter: "less is a bore." Venturi argued that historical reference, ornamental complexity, and contradiction were not failures of discipline but features of architectures that respond to real human contexts. Modernism's reductivism, in his read, produced buildings that were rigorous and lifeless.

The dialectic has run for sixty years. It's a real argument with no clean winner; both sides describe genuine tradeoffs. What's interesting is what happens when production cost approaches zero.

## What LLMs do to the equation

Production used to be expensive. Designing a building, writing a specification, drafting a wireframe, producing code — each took specialist time, which was scarce, which made restraint (or at least selectivity) load-bearing. You couldn't produce ten variants because you couldn't afford ten variants. "Less is more" was easier to honor when "more" was hard.

LLMs invert this. Variants are cheap. Drafts are cheap. Iterations are cheap. *Multiple parallel explorations* are cheap. A solo person with LLMs can produce, in an afternoon, output that would have taken a small team a week. The instinct that follows is "now we can have more."

But "more" understates what's happening. Three things are compounding at once:

1. **Each session produces more.** An LLM session produces several screens of output where a human session might have produced one.
2. **Sessions multiply.** You can run many sessions in parallel — different framings, different LLMs, different contexts — without conflict.
3. **Each session can revise itself many times before publishing.** The LLM can produce a draft, critique it, rewrite, critique again, all within the session.

These multiply, not add. Output volume in 2026 isn't 10× what it was; it's something more like 100× to 1000× per unit of human attention. *More is much more.*

## So what's the human response?

The cheap answer is "embrace the abundance." Generate ten variants of every design. Run five LLMs on every problem. Let the volume compete in the marketplace of ideas. This is the Venturi position translated to LLMs: complexity is fine, contradiction is fine, the more the merrier.

The cheap answer is wrong. Or, more precisely: it's right for *exploration* and wrong for *commitment*. Producing many variants during exploration is genuinely useful — LLMs are excellent rapid-prototyping engines. But the artifacts that *commit* — the milestone the team is going to build, the ADR that constrains future code, the contract that defines an interface — should not be ten variants. They should be one converged thing, ratified by humans, with the variants discarded or filed as alternatives in the rationale.

This is where the inversion happens. The LLM era produces *more output*. The right human response is *fewer artifacts to ratify*. Not because abundance is bad, but because ratification is the bottleneck, and ratification scales by *coherence*, not by *count*. A human can ratify one well-shaped milestone in a few minutes. A human cannot meaningfully ratify ten variants of the same milestone in any amount of time — they all blur into "one of these is probably fine."

So the credo for the LLM era isn't "less is more" (Mies) and isn't "less is a bore" (Venturi). It's something closer to:

> **More output, fewer artifacts.**

LLMs produce abundantly. Humans ratify selectively. The leverage is in the ratification.

## Why this matters for tools

Tools that are designed for human-rate production fail under LLM-rate production. PR review is the canonical example: when the LLM produces a 1000-line diff in twenty minutes, the human reviewer either rubber-stamps or works longer hours. Neither is a good answer. The tool's *unit of human attention* — the diff — is wrong-sized for the new economy.

The framework's bet, expressed in [continuous ratification](https://proliminal.net/theses/the-pr-bottleneck/), is that the right unit for human attention is the *milestone* (or the ADR, the contract — the durable structural decision), not the diff. Multiple LLM sessions converge on a milestone over time; the human ratifies state transitions on it; the underlying production volume can be 100× without the ratification cost going up correspondingly. *More output, fewer artifacts.*

Tools that respect this shift are tools that:
- Make artifacts *converge* rather than *accumulate*. One milestone, ratified over time, beats ten draft milestones.
- Surface state, not volume. The question "what's currently true?" beats "what's been produced?"
- Make ratification fast and contextual. The human says yes or no on a small, well-shaped artifact, not on an avalanche of output.
- Discard exploration cleanly. The variants the LLM generated to find the right answer don't need to live in the audit trail forever.

Tools that don't respect this shift drown teams in output and produce the rubber-stamp dynamic that's currently the default.

## The deeper observation

There's a pattern across periods when production costs collapse. The printing press made books abundant; the response was libraries, indexes, and editors — institutions for *selecting* among the abundance. Photography made images abundant; the response was magazines, galleries, and curators. Recording made music abundant; the response was radio programmers, A&R, playlist editors. In each case, the cultural institutions that emerged were not about *producing more* but about *selecting among the abundant*.

LLMs are doing this to knowledge work. The institutions are still emerging — orchestrators, ratification chokepoints, state-keeping artifacts, the role of the editor at every stage. Mies's "less is more" was a 20th-century answer to scarcity-of-production. *More output, fewer artifacts* is a 21st-century answer to abundance-of-production.

Both are versions of the same instinct: the human's leverage is in selection and judgment, not in volume of output. The credo just changes form when production economics change.

## The corollary

If the human's leverage is in selection, then the institutions that grow up around LLM-amplified work will be selection-shaped, not production-shaped. The framework is one such institution — small, opinionated, deterministic, designed to make ratification fast on artifacts that converge rather than multiply. Other institutions will emerge: editorial roles in software, taste-makers for AI-generated design, curators of architectural pattern libraries, certifiers of generated content for regulated contexts. The work isn't disappearing; it's relocating from production to ratification.

The corollary, restated from [the PR bottleneck post](https://proliminal.net/theses/the-pr-bottleneck/): **HITL gets stronger, not weaker, when production costs collapse.** Every selection is leveraged. Every ratification is amplified. The human at the chokepoint moves more work than the human at the production station ever could — because the production is being done by the machine, and the *judgment* is what's left for humans to do, and judgment is what scales.

More output. Fewer artifacts. More leverage per yes-or-no.

That's the era we're in.

---

## Related

- [The PR bottleneck is a process problem](https://proliminal.net/theses/the-pr-bottleneck/) — the technical version of the same observation, focused on ratification at state transitions.
- [State, not workflow](https://proliminal.net/theses/state-not-workflow/) — why state-keeping is the durable shape when production-shaped workflow erodes.
- [The aiwf working paper](https://proliminal.net/theses/working-paper/) — the framework that takes this bet seriously.
