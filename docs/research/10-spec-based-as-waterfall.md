# Spec-based development is waterfall in disguise — and continuous ratification is the alternative

> **Status:** thesis-draft
> **Hypothesis:** Spec-based development as currently practiced encodes the same artifact-handoff dependency graph as waterfall — sequential phases, gated transitions, downstream re-flow on upstream change — and iteration does not dissolve the trap; continuous ratification is structurally different because it ratifies state transitions on a single artifact rather than handoffs between artifacts.
> **Audience:** anyone evaluating spec-based development tools (Spec Kit, Kiro, Tessl, similar), or anyone who has heard "spec-based = good, waterfall = bad" and wants the analysis sharper.
> **Premise:** [07](https://proliminal.net/theses/state-not-workflow/) argued workflow erodes under LLM amplification; [08](https://proliminal.net/theses/the-pr-bottleneck/) argued continuous ratification replaces batched review. This document examines what spec-based development is structurally, why it inherits waterfall's defining property, and where it remains right anyway. The companion survey [`surveys/understanding-spec-driven-development`](surveys/understanding-spec-driven-development.md) supplies the field-level grounding (taxonomy, history, tooling, citations); this document takes the framework's structural position.
> **Tags:** #thesis #software-development #aiwf #workflow

---

## Abstract

Spec-based development — write a spec, agree on it, then have an LLM execute it — is having a moment. Tools like GitHub Spec Kit, AWS Kiro, and Tessl present it as the responsible alternative to "vibe coding" and as the discipline that makes AI-assisted development trustworthy. But "spec-based development" is not one thing; the field has fragmented into at least five rungs of a ladder (spec-as-prompt, spec-first, spec-anchored, spec-as-source, spec-as-contract), and each rung has different stakes. This document argues that the structure of spec-based development *as the heavyweight tools encode it* — primarily spec-first reaching for spec-anchored, with spec-as-source as the visible aspiration — is *structurally identical* to waterfall: a sequence of phased artifacts (spec → plan → tasks → code) with gated handoffs and downstream re-flow when upstream changes. Iteration doesn't escape the trap; iterative waterfall is still waterfall. The same shape was tried forty years ago as Model-Driven Development, with deterministic generators where today's tools use LLMs; MDD failed at the same boundary. The framework's alternative — continuous ratification at state transitions on a single converging artifact — is genuinely different because nothing handoffs to anything; the same artifact accumulates judgments over time. This is not "spec-based is wrong"; spec-based remains right where waterfall always was right (regulated, multi-team, contractual), and spec-as-contract remains a working narrow case. It is "spec-based-the-organizing-shape is the wrong default for the work types LLMs are reshaping," and continuous ratification is what fits that reshape.

---

## 1. The claim

**Spec-based development as currently practiced by the heavyweight tools is waterfall in disguise.** Strip the marketing and the dependency graph is the same:

```
spec → plan → tasks → code → tests → review → merge
```

Each phase produces an artifact. Each subsequent phase consumes its predecessor's output. Each transition is gated — usually by human approval — and changing an upstream artifact requires re-flowing all downstream artifacts. This is *the* defining structural property of waterfall: artifacts in sequence, dependencies one-way, change ripples downstream.

The marketing argues spec-based is *not* waterfall because (a) iteration is built in, and (b) the LLM does the slow parts so the cycle is fast. We'll address both. Neither dissolves the structure; they just make it cheaper.

A pre-emptive scoping note. "Spec-based development" is not one thing — the term covers at least five distinct workflows (see §3 and the companion survey). The structural-equivalence claim above lands hardest on **spec-first reaching for spec-anchored** (the Spec Kit / Kiro pattern) and on **spec-as-source** (Tessl). It applies less to **spec-as-prompt** (which is just a detailed prompt and inherits whatever shape the surrounding workflow has) and not at all to **spec-as-contract** (which is a different shape — a verifiable property at a narrow boundary, predating LLMs by a decade and still working). The argument that follows uses "spec-based" to mean the heavyweight rungs unless the text says otherwise.

## 2. Steel-manning waterfall

Waterfall's reputation is worse than its substance. The Royce paper usually cited as inventing waterfall ("Managing the Development of Large Software Systems," 1970) actually proposed iteration — Royce's diagrams include feedback loops between phases. The "pure" waterfall everyone derides is largely a strawman; what real organizations did, even in the 1970s, was iterative-waterfall.

What waterfall got right:

- **Artifact handoffs are real.** A designer needs a requirement to design against; an engineer needs a design to implement; a tester needs an implementation to test. These dependencies aren't invented; they reflect the structure of the work.
- **Gates catch problems early.** Reviewing a requirements document before three months of engineering is cheaper than reviewing the engineering after.
- **Specialization is efficient when specialists are scarce.** A specialist designer producing for many engineers is faster than every engineer doing their own design.

What waterfall got wrong:

- **The cost of revision was too high to bear.** Re-flowing a phase took weeks; teams resisted late changes; products shipped against obsolete requirements.
- **Hand-off latency dominated.** Waiting for the next phase to receive your output, then waiting for them to consume it, was where projects spent most of their calendar time.
- **Specialists became bottlenecks.** The whole pipeline ran at the slowest specialist's pace.

The Agile movement of the 2000s addressed the third failure mode (specialist throughput) by cross-functional teams, the second (handoff latency) by short cycles, and the first (revision cost) by iterating in small increments. Agile didn't dissolve the artifact-handoff structure; it reduced the cost of each handoff and increased their frequency.

## 3. Five rungs of the ladder, and which one we're arguing about

The companion survey [`surveys/understanding-spec-driven-development`](surveys/understanding-spec-driven-development.md) lays out the taxonomy in detail; the short version is enough to sharpen the claim here.

- **Spec-as-prompt.** The spec is a detailed prompt, often discarded after use. Spec-shaped, not spec-driven. Whatever shape the surrounding workflow has, this rung inherits.
- **Spec-first.** A structured spec is written first and used to drive one task. The spec usually doesn't survive the change. This is what most Spec Kit and Kiro users actually practice, regardless of marketing.
- **Spec-anchored.** The spec persists alongside code and gets versioned with it as the system evolves. The aspirational target of most heavyweight tools; rarely achieved in practice.
- **Spec-as-source.** The spec is the only artifact humans edit; code is regenerated. Tessl is the cleanest published implementation. The strongest claim, the worst track record.
- **Spec-as-contract.** A verifiable property at a narrow boundary — OpenAPI, Protobuf, TLA+. Predates LLMs entirely; works because the abstraction is small and generation is deterministic.

The structural-equivalence claim applies to the heavyweight middle of the ladder (spec-first → spec-anchored → spec-as-source). These are the rungs where the artifact-handoff dependency graph is most visible, and where the "iteration makes it not-waterfall" defense is most often made. Spec-as-prompt is too informal to argue about as a shape; spec-as-contract is genuinely a different shape and earns the exemption.

A historical parallel makes the structural critique sharper. **Model-Driven Development / Model-Driven Architecture** (1990s–2000s) proposed exactly the same arrangement: describe systems in UML or a textual DSL, then transform models into running code via deterministic generators. Birgitta Böckeler, who has direct industry experience with MDD, observes that "the models in MDD were essentially specs, just expressed in custom UML or textual DSLs rather than natural language" and that teams built generators to produce the code. France and Rumpe's 2007 ICSE roadmap paper concluded that "full realizations of the MDE vision may not be possible in the near to medium-term primarily because of the wicked problems involved." MDD never crossed into mainstream business development.

This matters because the claim "spec is source, code is artifact" is not new; it is the claim MDD made, with LLMs replacing the deterministic code generator. MDD failed at the abstraction boundary — the cost-of-revision problem was worse, not better, because revising the model required re-running the generator and re-validating the output. LLMs do not improve this; they make it worse along one axis, because LLM regeneration is non-deterministic. The same spec produces different code on different runs. Böckeler reports observing this directly with Tessl. The lessons from MDD's failure are directly applicable to spec-as-source today, and the heavyweight tools' silence on this history is itself a tell.

## 4. Steel-manning spec-based development

Spec-based development's pitch is genuinely worth taking seriously:

- **Specs are checkable.** A human can read a spec faster than they can read the code that implements it. Reviewing specs catches misunderstandings before LLMs spend cycles producing wrong things.
- **Specs document intent.** When the LLM produces something off-spec, you have a reference for "what did we ask for?" that's separate from "what did we get?"
- **Specs reduce LLM context burden.** Instead of inferring what to build from a vague request plus codebase context, the LLM gets a structured statement of what to build.
- **Specs make change explicit.** Modifying the spec is a deliberate act; the change is visible; the implementation can be re-run.

The strongest version of the steel-man comes from Marc Brooker, who has worked closely with the Kiro team. Brooker argues that critics are attacking a strawman: SDD "isn't about pulling designs *up-front*, it's about pulling designs *up*. Making specifications explicit, versioned, living artifacts that the implementation of the software flows from, rather than static artifacts." On this framing, the spec is what is iterated on — the implementation flows from each iteration — and the cycle is the same as agile, just at a different level of abstraction.

This is the version of SDD worth disagreeing with carefully, because the structural critique that follows has to land on Brooker's framing too, not just on the strawman version. Anyone arguing "specs are bad" is overreaching. The question isn't whether specs help; it's whether *spec-based-as-the-organizing-shape* fits the work, and whether Brooker's "pulling up not up-front" version genuinely escapes the artifact-handoff structure or merely makes the handoffs cheaper.

## 5. Why iterative waterfall doesn't escape the trap

The standard defense: "but the spec can change, and we re-execute." Two responses.

**First: iteration is not the same as continuous integration of judgment.** When the spec changes, you have a discrete event (spec v2), and the LLM re-runs to produce code v2. The change is visible, but it's also *isolated* — a snapshot at a moment, separated from the prior snapshot by the boundary of the spec edit. This is the same shape as iterative waterfall: phases iterate, but they remain phases with hand-offs at their edges. Brooker's "pulling up" framing reduces the cost of the iteration; it does not dissolve the boundary.

Continuous ratification is structurally different. There's no "spec snapshot" and "code snapshot" — there's a single artifact (a milestone, an ADR) accumulating ratifications over time. Each ratification is a small commit on the same artifact. The artifact converges; phases never separate.

**Second: re-execution is not free, even with LLMs.** Spec-based tools talk about cycle time as if revision is cost-free. It isn't, even when the LLM is fast:

- Re-executing produces a *different* implementation, which means the human's prior review of the previous implementation is invalidated. You're back to reviewing a thousand-line diff. The non-determinism noted in §3 is exactly this problem — and it is worse than MDD's was, not better.
- Partial work in progress at the moment of spec change has to be discarded or merged manually. The LLM doesn't know what you'd already accepted.
- If the spec changes during implementation, the LLM has to either pause (delaying), continue with stale spec (producing wrong work), or re-plan (wasting prior work). Each option has cost.

Iterative waterfall reduced revision cost compared to pure waterfall, but it never eliminated the fundamental issue that *change at any phase ripples through downstream phases*. Spec-based development inherits this property. LLM speed makes the ripple cheap, not free, and the cheaper-than-free claim is what most spec-based marketing implicitly relies on.

## 6. The Agile counter-argument

The obvious objection: "Agile already solved this. Spec-based is just Agile with LLMs."

Agile got closer to dissolving the artifact-handoff shape than waterfall did, but it didn't fully escape. The artifacts changed names — user stories instead of requirements, tasks instead of design, code instead of code, automated tests instead of test plans — but they remained sequential and gated. A user story in a sprint board is still consumed by a developer, who produces code, which is consumed by reviewers, who approve before merge. The handoffs are smaller, faster, and tighter, but they're still there.

XP, lean, and trunk-based development each took further steps. XP's pair programming dissolved the design/implementation handoff (two specialists collaborating in real-time). Lean's just-in-time decisions reduced the scope of upstream artifacts (don't write the spec until you need it). Trunk-based development reduced merge friction by integrating early and often. None of them dissolved the *artifact* shape; they just reduced its cost.

Continuous ratification is the next step. Instead of artifacts in sequence with handoffs, there's *one artifact per concern* (one milestone, one ADR, one contract) with judgments accumulating on it over time. The PM doesn't hand a spec to the architect; the PM and the architect both edit the same milestone, in the same file, with their commits attributed to their roles. The engineer doesn't receive a finished design; the engineer reads the same milestone, contributes commits to its scope, and ratifies its acceptance criteria as part of starting work. The "phases" are not stations the artifact moves through; they are different ratifications applied to the same artifact, often by different people, with the framework recording who ratified what.

This is genuinely different from spec-based, and from Agile, and from XP, and from any of the methodologies that retain artifact-handoff as a primitive.

## 7. Why continuous ratification is structurally different

The structural difference matters because it changes which problems disappear and which ones remain.

**Problems that disappear with continuous ratification:**

- *Re-flow when upstream changes.* There's no upstream artifact to flow from. The same artifact is being edited; revisions don't ripple, they accumulate.
- *Stale phase output.* Output is never finalized as a "phase deliverable" that other phases depend on. The current state of the artifact *is* the current state.
- *Hand-off latency.* No hand-off. Multiple roles edit the same file (or the same MCP-mediated state) with their contributions visible immediately.
- *Spec-vs-implementation drift.* The spec and the implementation are not separate artifacts in this shape — the milestone's spec body, its acceptance criteria, and its commit history are one accumulating thing.

**Problems that remain:**

- *Genuine sequencing.* Some work *does* depend on prior work. Code can't be tested before it's written. Deployment can't happen before merge. Continuous ratification doesn't dissolve real dependencies; it just doesn't *invent* dependencies the work doesn't need.
- *Specialist judgment.* The architect's ratification of a milestone's structural soundness is different from the PM's ratification of its scope; both are needed; they don't substitute. Specialization persists as judgment, even when production becomes role-agnostic.
- *Coordination cost.* Multiple humans editing the same artifact need some discipline to avoid stepping on each other. The framework's pre-PR session model handles this, but the cost isn't zero.
- *Audit trail clarity.* When five people ratify different aspects of a milestone over a week, the audit trail needs to be greppable. Structured commit trailers help; the framework's job is to make this work.

The disappearance of artifact-handoff problems is what makes continuous ratification fit LLM-amplified work. The retained problems are real but bounded.

## 8. When spec-based development is right anyway

This document is *not* arguing spec-based development is universally wrong. It's arguing it's wrong as the *default* for LLM-amplified product work. There are genuine cases where spec-based is the right shape:

- **Regulated industries** where a signed-off spec is a legal artifact. The FDA requires specifications; the FAA requires specifications; medical device development requires specifications. The spec isn't just a planning artifact; it's a compliance artifact. Iteration on a regulated spec involves specific change-control processes that *are* waterfall-shaped because the regulators require them to be.
- **Contractual work** where a customer pays for delivery of a specified thing. The spec is the contract; modifying it requires negotiation. Continuous ratification by the implementing team isn't appropriate when the customer has signing authority over scope.
- **Multi-team integration** where a spec is the interface between teams that don't share infrastructure or process. An API specification is the handshake between producer and consumer teams; both ratify the spec, not the implementation. This is genuinely waterfall-shaped at the team boundary, even if each team uses continuous ratification internally.
- **Bidding and estimation** where the spec is the basis for committing to a budget or timeline. You can't commit without first specifying.
- **Spec-as-contract at narrow boundaries** — the rung the structural critique exempts. OpenAPI, Protobuf, schema-first ETL. The abstraction is small, the generation is deterministic, the artifact is genuinely different in kind from a "spec → plan → tasks → code" pipeline.

The framework should integrate with spec-based tools where teams need them — exporting milestones as specs for compliance, importing specs as milestones for implementation — without trying to replace them in their proper domain.

## 9. The honest summary

Spec-based development as the heavyweight tools encode it — spec-first reaching for spec-anchored, with spec-as-source as the aspiration — is structurally waterfall: the artifact-handoff dependency graph is identical, and iteration reduces revision cost without dissolving the structure. The claim is not new; Model-Driven Development tried it forty years ago with deterministic generators and failed at the same boundary, and LLM non-determinism makes the failure mode worse along one axis, not better. The strongest version of the steel-man — Brooker's "pulling designs up, not up-front" — reduces the cost of each iteration but does not eliminate the artifact boundary that the structural critique attacks. This isn't a damning critique of specs as such; waterfall is right for some work, spec-as-contract is right for narrow boundaries, and so on. But it *is* a critique of spec-based-as-the-organizing-shape for LLM-amplified product work, where the artifact-handoff structure is itself the friction LLMs make most painful.

Continuous ratification is structurally different because it ratifies *state transitions on a single artifact*, not handoffs between artifacts. The same milestone accumulates commits from multiple roles; the same ADR gathers ratifications over time; the same contract evolves with comments from PM, architect, and implementer interleaved on the same lines. Phases dissolve into roles applying judgment in parallel on the same converging picture.

The framework's bet, restated: for the work LLMs are reshaping, the artifact is the unit, ratification is the act, state is the record, and handoffs are an artifact of throughput-bounded specialization that LLMs are dissolving. Spec-based development is the closest competitor to this position; it remains right where waterfall was always right; it is the wrong default for the work types LLMs amplify most. Naming which rung of the ladder is in play — spec-as-prompt, spec-first, spec-anchored, spec-as-source, spec-as-contract — is the honest start of any SDD discussion, including this one.

---

## In this series

- Previous: [09 — Orchestrators and project managers](https://proliminal.net/theses/orchestrators-and-project-managers/)
- Next: [11 — Should the framework model the code?](https://proliminal.net/theses/should-the-framework-model-the-code/)
- Related: [07 — State, not workflow](https://proliminal.net/theses/state-not-workflow/), [08 — The PR bottleneck](https://proliminal.net/theses/the-pr-bottleneck/)
- Companion survey: [`surveys/understanding-spec-driven-development`](surveys/understanding-spec-driven-development.md) (taxonomy, history, tooling, full citations)
- Synthesis: [working paper](https://proliminal.net/theses/working-paper/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
