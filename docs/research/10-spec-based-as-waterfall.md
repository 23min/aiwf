# Spec-based development is waterfall in disguise — and continuous ratification is the alternative

> **Status:** thesis-draft
> **Hypothesis:** Spec-based development as currently practiced encodes the same artifact-handoff dependency graph as waterfall — sequential phases, gated transitions, downstream re-flow on upstream change — and iteration does not dissolve the trap; continuous ratification is structurally different because it ratifies state transitions on a single artifact rather than handoffs between artifacts.
> **Audience:** anyone evaluating spec-based development tools (Spec Kit, Kiro, Tessl, similar), or anyone who has heard "spec-based = good, waterfall = bad" and wants the analysis sharper.
> **Premise:** [07](https://proliminal.net/theses/state-not-workflow/) argued workflow erodes under LLM amplification; [08](https://proliminal.net/theses/the-pr-bottleneck/) argued continuous ratification replaces batched review. This document examines what spec-based development is structurally, why it inherits waterfall's defining property, and where it remains right anyway.
> **Tags:** #thesis #software-development #aiwf #workflow

---

## Abstract

Spec-based development — write a spec, agree on it, then have an LLM execute it — is having a moment. Tools like GitHub Spec Kit, AWS Kiro, and Tessl present it as the responsible alternative to "vibe coding" and as the discipline that makes AI-assisted development trustworthy. This document argues that the structure of spec-based development is *structurally identical* to waterfall: a sequence of phased artifacts (spec → plan → tasks → code) with gated handoffs and downstream re-flow when upstream changes. Iteration doesn't escape the trap; iterative waterfall is still waterfall. The framework's bet — continuous ratification at state transitions on a single converging artifact — is genuinely different because nothing handoffs to anything; the same artifact accumulates judgments over time. This is not "spec-based is wrong"; spec-based remains right where waterfall always was right (regulated, multi-team, contractual). It is "spec-based is the wrong default for the work types LLMs are reshaping," and continuous ratification is what fits that reshape.

---

## 1. The claim

**Spec-based development as currently practiced is waterfall in disguise.** Strip the marketing and the dependency graph is the same:

```
spec → plan → tasks → code → tests → review → merge
```

Each phase produces an artifact. Each subsequent phase consumes its predecessor's output. Each transition is gated — usually by human approval — and changing an upstream artifact requires re-flowing all downstream artifacts. This is *the* defining structural property of waterfall: artifacts in sequence, dependencies one-way, change ripples downstream.

The marketing argues spec-based is *not* waterfall because (a) iteration is built in, and (b) the LLM does the slow parts so the cycle is fast. We'll address both. Neither dissolves the structure; they just make it cheaper.

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

## 3. Steel-manning spec-based development

Spec-based development's pitch is genuinely worth taking seriously:

- **Specs are checkable.** A human can read a spec faster than they can read the code that implements it. Reviewing specs catches misunderstandings before LLMs spend cycles producing wrong things.
- **Specs document intent.** When the LLM produces something off-spec, you have a reference for "what did we ask for?" that's separate from "what did we get?"
- **Specs reduce LLM context burden.** Instead of inferring what to build from a vague request plus codebase context, the LLM gets a structured statement of what to build.
- **Specs make change explicit.** Modifying the spec is a deliberate act; the change is visible; the implementation can be re-run.

These are real benefits. Anyone arguing "specs are bad" is overreaching — for many use cases (especially regulated, contractual, or multi-team) specs are exactly right. The question isn't whether specs help. It's whether *spec-based-as-the-organizing-shape* fits the work.

## 4. Why iterative waterfall doesn't escape the trap

The standard defense: "but the spec can change, and we re-execute." Two responses.

**First: iteration is not the same as continuous integration of judgment.** When the spec changes, you have a discrete event (spec v2), and the LLM re-runs to produce code v2. The change is visible, but it's also *isolated* — a snapshot at a moment, separated from the prior snapshot by the boundary of the spec edit. This is the same shape as iterative waterfall: phases iterate, but they remain phases with hand-offs at their edges.

Continuous ratification is structurally different. There's no "spec snapshot" and "code snapshot" — there's a single artifact (a milestone, an ADR) accumulating ratifications over time. Each ratification is a small commit on the same artifact. The artifact converges; phases never separate.

**Second: re-execution is not free, even with LLMs.** Spec-based tools talk about cycle time as if revision is cost-free. It isn't, even when the LLM is fast:

- Re-executing produces a *different* implementation, which means the human's prior review of the previous implementation is invalidated. You're back to reviewing a thousand-line diff.
- Partial work in progress at the moment of spec change has to be discarded or merged manually. The LLM doesn't know what you'd already accepted.
- If the spec changes during implementation, the LLM has to either pause (delaying), continue with stale spec (producing wrong work), or re-plan (wasting prior work). Each option has cost.

Iterative waterfall reduced revision cost compared to pure waterfall, but it never eliminated the fundamental issue that *change at any phase ripples through downstream phases*. Spec-based development inherits this property. LLM speed makes the ripple cheap, not free, and the cheaper-than-free claim is what most spec-based marketing implicitly relies on.

## 5. The Agile counter-argument

The obvious objection: "Agile already solved this. Spec-based is just Agile with LLMs."

Agile got closer to dissolving the artifact-handoff shape than waterfall did, but it didn't fully escape. The artifacts changed names — user stories instead of requirements, tasks instead of design, code instead of code, automated tests instead of test plans — but they remained sequential and gated. A user story in a sprint board is still consumed by a developer, who produces code, which is consumed by reviewers, who approve before merge. The handoffs are smaller, faster, and tighter, but they're still there.

XP, lean, and trunk-based development each took further steps. XP's pair programming dissolved the design/implementation handoff (two specialists collaborating in real-time). Lean's just-in-time decisions reduced the scope of upstream artifacts (don't write the spec until you need it). Trunk-based development reduced merge friction by integrating early and often. None of them dissolved the *artifact* shape; they just reduced its cost.

Continuous ratification is the next step. Instead of artifacts in sequence with handoffs, there's *one artifact per concern* (one milestone, one ADR, one contract) with judgments accumulating on it over time. The PM doesn't hand a spec to the architect; the PM and the architect both edit the same milestone, in the same file, with their commits attributed to their roles. The engineer doesn't receive a finished design; the engineer reads the same milestone, contributes commits to its scope, and ratifies its acceptance criteria as part of starting work. The "phases" are not stations the artifact moves through; they are different ratifications applied to the same artifact, often by different people, with the framework recording who ratified what.

This is genuinely different from spec-based, and from Agile, and from XP, and from any of the methodologies that retain artifact-handoff as a primitive.

## 6. Why continuous ratification is structurally different

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

## 7. When spec-based development is right anyway

This document is *not* arguing spec-based development is universally wrong. It's arguing it's wrong as the *default* for LLM-amplified product work. There are genuine cases where spec-based is the right shape:

- **Regulated industries** where a signed-off spec is a legal artifact. The FDA requires specifications; the FAA requires specifications; medical device development requires specifications. The spec isn't just a planning artifact; it's a compliance artifact. Iteration on a regulated spec involves specific change-control processes that *are* waterfall-shaped because the regulators require them to be.
- **Contractual work** where a customer pays for delivery of a specified thing. The spec is the contract; modifying it requires negotiation. Continuous ratification by the implementing team isn't appropriate when the customer has signing authority over scope.
- **Multi-team integration** where a spec is the interface between teams that don't share infrastructure or process. An API specification is the handshake between producer and consumer teams; both ratify the spec, not the implementation. This is genuinely waterfall-shaped at the team boundary, even if each team uses continuous ratification internally.
- **Bidding and estimation** where the spec is the basis for committing to a budget or timeline. You can't commit without first specifying.

The framework should integrate with spec-based tools where teams need them — exporting milestones as specs for compliance, importing specs as milestones for implementation — without trying to replace them in their proper domain.

## 8. The honest summary

Spec-based development is structurally waterfall — the artifact-handoff dependency graph is identical, and iteration reduces revision cost without dissolving the structure. This isn't a damning critique; waterfall is right for some work, and so is spec-based. But it *is* a critique of spec-based as the default organizing shape for LLM-amplified product work, where the artifact-handoff structure is itself the friction LLMs make most painful.

Continuous ratification is structurally different because it ratifies *state transitions on a single artifact*, not handoffs between artifacts. The same milestone accumulates commits from multiple roles; the same ADR gathers ratifications over time; the same contract evolves with comments from PM, architect, and implementer interleaved on the same lines. Phases dissolve into roles applying judgment in parallel on the same converging picture.

The framework's bet, restated: for the work LLMs are reshaping, the artifact is the unit, ratification is the act, state is the record, and handoffs are an artifact of throughput-bounded specialization that LLMs are dissolving. Spec-based development is the closest competitor to this position; it remains right where waterfall was always right; it is the wrong default for the work types LLMs amplify most.

---

## In this series

- Previous: [09 — Orchestrators and project managers](https://proliminal.net/theses/orchestrators-and-project-managers/)
- Related: [07 — State, not workflow](https://proliminal.net/theses/state-not-workflow/), [08 — The PR bottleneck](https://proliminal.net/theses/the-pr-bottleneck/)
- Synthesis: [working paper](https://proliminal.net/theses/working-paper/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
