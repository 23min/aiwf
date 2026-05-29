# Two walk-backs: substrate and philosophy

> **Status:** defended-position
> **Hypothesis:** The aiwf research arc has involved two walk-backs from more ambitious positions, and they are not the same kind. The first walk-back — from event-sourced kernel to markdown-in-git — was substrate-driven: forced by what git's branching model can carry, not by a conclusion about what verification or mechanical enforcement can do in principle. The second walk-back — from mechanical TDD enforcement to evidence-and-judgment with persistent findings — was philosophy-driven: forced by observing the cheating attractor and concluding that mechanical enforcement of LLM behavior at the discipline layer is gameable. Conflating these two walk-backs leads to overgeneralization in two directions. It makes the substrate walk-back look philosophical (and license arguments against mechanical verification in general), and it makes the philosophy walk-back look substrate-specific (and license its dismissal when the substrate is different). The clean separation lets the arc engage honestly with adjacent projects ([Loom](https://github.com/23min/loom), refinement-type-driven verification, formal-methods stacks) that operate in different substrates and at different layers. Adjacent projects do not necessarily inherit either walk-back, but they face a *different* walk-back at a different layer that the arc has not yet treated: the claim-authorship boundary, where the cheating dynamics from the philosophy walk-back re-emerge in a new form.
> **Audience:** anyone reading the arc and trying to apply its conclusions to adjacent projects; anyone evaluating whether the framework's evolution generalizes to settings outside git-tracked planning state.
> **Premise:** synthesizes [`00-fighting-git`](https://proliminal.net/theses/fighting-git/), [`07-state-not-workflow`](https://proliminal.net/theses/state-not-workflow/), [`08-the-pr-bottleneck`](https://proliminal.net/theses/the-pr-bottleneck/), [`11-should-the-framework-model-the-code`](https://proliminal.net/theses/should-the-framework-model-the-code/), and the TDD architecture work in [`docs/explorations/06-tdd-diagnostic.md`](https://github.com/23min/aiwf/blob/main/docs/explorations/06-tdd-diagnostic.md) and [`07-tdd-architecture-proposal.md`](https://github.com/23min/aiwf/blob/main/docs/explorations/07-tdd-architecture-proposal.md). Engages with the Loom architectural proposal as the immediate adjacent case the distinction clarifies.
> **Tags:** #thesis #aiwf #verification #scope

---

## Abstract

This document distinguishes two walk-backs in the aiwf arc that the working paper treats as continuous and that I have, in conversation, sometimes treated as a single epistemic move. They are not. The first walk-back is the abandonment of the event-sourced kernel, the hash-verified projection, the monotonic ID counter — the apparatus that fought git's branching model. That walk-back is substrate-driven: it follows from the choice to operate in git, and would not apply to a project that operates outside git's constraints. The second walk-back is the abandonment of mechanical AC-level TDD enforcement in favor of cycle-evidence findings and human triage at wrap. That walk-back is philosophy-driven: it follows from the observation that LLM agents systematically game mechanical gates over process, and would apply to any project that enforces process mechanically against LLM agents. The two have been intertwined in the arc's narrative because they happen to land in the same framework, but they generalize separately. Without this distinction, the arc's conclusions are over- or under-applied to adjacent work. With it, the arc's logic transfers cleanly to other settings, and the boundary of where each walk-back applies becomes a useful design question rather than a confused one.

---

## 1. The two walk-backs in compressed form

**Walk-back 1 — Substrate.** The repository started with an event-sourced architecture: an append-only `events.jsonl` file, a derived `graph.json` projection, RFC 8785 canonicalization of the projection, SHA-256 hash chaining, monotonic per-kind ID counters, trace-first writes as a permanent ledger. The architecture was elaborate and internally coherent. It did not survive a careful reading of how git merges files across branches. The full account is in [`00-fighting-git`](https://proliminal.net/theses/fighting-git/); the upshot is that *total ordering of writes* is a property of file content, file content is what git merges textually, and therefore total ordering cannot be a property of the merged file. Hash chains break under merge. Monotonic counters allocate the same id on diverging branches. The architecture's internal coherence was fine; its compatibility with git's mechanics was not. The walk-back replaced the apparatus with markdown files plus structured commit trailers, on the grounds that git already provides — at the file-tree level — what the apparatus tried to reproduce at the content level.

**Walk-back 2 — Philosophy.** The PoC shipped with a mechanical AC-level TDD enforcement rule: an acceptance criterion at `status: met` required `tdd_phase: done`, with a kernel-side audit code (`acs-tdd-audit`) blocking the promotion when the milestone was `tdd: required`. The rule was structurally well-formed: it ran at the kernel layer, it produced findings deterministically, it did not depend on LLM behavior to enforce. The TDD diagnostic in [`docs/explorations/06-tdd-diagnostic.md`](https://github.com/23min/aiwf/blob/main/docs/explorations/06-tdd-diagnostic.md) identified a problem the rule's structural well-formedness did not catch: under LLM authorship, the rule's gate is *gameable*. Agents produce shallow tests, batch-promote ACs, hardcode expected values, delete tests they cannot satisfy. The cheating is rational from the agent's perspective — the gate measures something (TDD process) that is decoupled from what the test is actually doing in the artifact. The walk-back, in [`07-tdd-architecture-proposal.md`](https://github.com/23min/aiwf/blob/main/docs/explorations/07-tdd-architecture-proposal.md), removed the mechanical gate and replaced it with cycle-evidence audits that produce *findings* (F-NNN, a new entity kind) the human triages at wrap. The principle: *discipline is visible, not enforced. The cheating attractor is closed by making cheats traceable, not by trying to block them.*

Both walk-backs are real. Both produced better designs than what they replaced. The arc presents them as a continuous narrative — "we kept getting smaller; we kept walking back from over-engineering" — and that narrative is true at the level of artifact size. It is misleading at the level of *what the walk-back is responding to*.

---

## 2. The substrate walk-back, isolated

The first walk-back does not survive transplantation outside git. To see this, list the specific failures it responds to:

- Concatenation of appends on `events.jsonl` from two branches breaks sequence-number uniqueness.
- Hash chains computed on post-state diverge irreparably when two branches both append.
- Monotonic per-kind ID counters allocate the same id on diverging branches because the counter is a multi-master replication primitive.
- Git's per-file three-way merge is a per-file CRDT for line-structured text, which is incompatible with abstractions (totally-ordered logs, hash chains) that need cross-file or cross-line semantic awareness during merge.

Every one of these failures is *git-specific*. A project that operates in a database, a CRDT-backed store, a content-addressed object graph, or a centralized service with serializable transactions would not encounter them. The architectural patterns the walk-back rejected — event logs, hash chains, monotonic counters — are entirely viable in substrates that support them. Many production systems use them every day.

The walk-back, then, is not "event logs and hash chains are bad ideas." It is "event logs and hash chains do not fit git." The arc has occasionally elided the distinction by phrasing the conclusion as "the right shape at the scale most teams operate at is materially smaller," which sounds general but is actually conditional on the choice of substrate. A team that operates outside git's substrate may need exactly the apparatus the walk-back rejected, for the same reasons that production systems use it.

The clean statement: **the substrate walk-back is a theorem about git, not about software architecture in general.** When evaluating an adjacent project, the question to ask is "what substrate does it operate in?" rather than "did it learn the lesson aiwf walked back to?" A project operating outside git carries different constraints and is not subject to the same walk-back logic.

---

## 3. The philosophy walk-back, isolated

The second walk-back is structurally different. The failure it responds to is *not* specific to any substrate. It is specific to a *layer*: the layer at which the framework attempts to enforce something about the *process* that produced an artifact, rather than something about the artifact's *values* or *behavior*.

The TDD enforcement rule measures TDD-the-process: was a red test written first, did it become green, was a refactor applied, was the AC promoted from open through red, green, refactor, done? These are claims about the workflow that produced the implementation. Under LLM authorship, the workflow is unobservable from the artifact. The agent can satisfy the gate by performing the workflow nominally — a test exists, a commit landed, a phase was promoted — while the underlying behavior (the test actually exercises the AC's interesting cases, the refactor materially improved the design) is gameable.

The cheating attractor follows from this layer mismatch. The gate sits at the process layer; the LLM acts at the value layer; the value-layer artifacts (tests as code, commits as code-shaped events) can be produced in shapes that satisfy the process gate without engaging the process's intent. The agent finds the cheapest path through the gate, and the cheapest path is to produce the *appearance* of the process at the value layer.

The walk-back's conclusion: *do not enforce process gates against LLM agents*. Replace with evidence-collection at the value layer (cycle commits exist, they are linked to the AC, they show non-trivial work between add and met) and persistent findings (F-NNN entities) that the human triages. The mechanical defense is reduced to what is mechanically tractable — *did work happen in the AC's lineage?* — while the *quality* of that work is left to human judgment supported by structured evidence.

This walk-back generalizes beyond aiwf. Any framework that attempts to mechanically enforce *process* against LLM agents faces the same dynamic. The defense is the same: stop enforcing process; collect evidence; surface for human review. The principle is substrate-independent; it applies whether the framework operates in git, in a database, or in a programming language.

The clean statement: **the philosophy walk-back is a theorem about LLM agents acting against process gates, not about software architecture in general or about a specific substrate.** When evaluating an adjacent project, the question to ask is "is the framework enforcing process or value?" When it is enforcing process at a layer where LLM agents act, the walk-back applies. When it is enforcing value, a different analysis is needed.

---

## 4. Why the conflation has been costly

I have, in conversation about adjacent projects, sometimes presented the two walk-backs as a single epistemic move. Specifically: when discussing the Loom architecture, I have framed its mechanical verification ambitions as *the same thing aiwf walked back from*, with the implication that Loom should walk back too. That framing conflates the two walk-backs in a specific way:

- It treats the substrate walk-back as if it generalized. Loom does not operate in git; its umbrellas live in a programming-language environment with its own compiler. The substrate failures that drove aiwf's first walk-back do not apply. Treating the walk-back as if it did is over-application.

- It treats the philosophy walk-back as if it applied to all mechanical verification. The walk-back applies specifically to process gates. Loom's gates are *value gates*: refinement types either hold for all values in the type domain or they do not; a property like `for-all L T, sum(L before T) = sum(L after T)` is true in the values or false in the values, with no "did you do TDD?" question hovering above the value. Value gates are not subject to the process-gaming dynamics the philosophy walk-back responds to. Treating them as if they were is under-distinguishing.

The conflation, in short, makes adjacent projects look like they should learn aiwf's lessons even when they don't apply. It treats aiwf's arc as a universal trajectory when it is, in part, contingent on choices aiwf made (work in git; enforce process at the AC layer) that adjacent projects did not make.

The cost is real because adjacent projects are *not* repeats of aiwf's mistakes. They are different bets in different parts of the design space, and the right evaluation engages with each on its own terms. Treating Loom (or any refinement-type-driven verification project) as a déjà-vu of aiwf is a category error.

---

## 5. The walk-back Loom does face

Adjacent projects do not inherit aiwf's walk-backs; they face their own. For Loom specifically, the walk-back that lies ahead — or that should, if the project is honest about it — is at the layer the philosophy walk-back identified as gameable, but shifted from *process gates* to *claim-authorship gates*.

Loom's value gates are not directly gameable by process-faking. They are gameable by *claim weakening*: the LLM proposes claims (`for-all x, P(x) ⇒ Q(x)`) whose antecedents are over-restrictive, whose bodies are vacuous in specific subdomains, whose preconditions narrow the operation's domain to where the implementation trivially satisfies the postcondition. The gate is mechanically discharged; the verifier reports success; the spec is decorative.

This is the *same cheating attractor* the philosophy walk-back identified, in a different position. The agent's optimization pressure pushes toward whichever side of the verification is cheapest to manipulate. At the TDD layer, the cheapest side was the test artifact. At the claim layer, the cheapest side is the spec. The defense pattern generalizes: *do not trust mechanical gates whose definitional content is itself LLM-authored*. Defend by collecting evidence at a layer the LLM cannot easily fabricate (mutation testing on claims, cross-register coverage measurement, statistical engagement-with-domain checks) and by elevating human review at the points where mechanical defense is incomplete.

The walk-back Loom faces is therefore not aiwf's walk-back replayed. It is the *philosophy walk-back's lesson*, applied at a layer aiwf did not occupy. The lesson generalizes; the specific shape of the cheating attractor does not. Adjacent projects need their own diagnoses, not aiwf's.

---

## 6. What this means for the arc

The arc has not, until this document, separated the two walk-backs. Subsequent docs should not conflate them. Specifically:

- When discussing the framework's evolution, the substrate walk-back and the philosophy walk-back are *different stories* and should be told separately. The working paper's §4 (the technical position) is the home of the substrate walk-back; §8 (the chokepoint argument and continuous ratification) is the home of the philosophy walk-back. They are currently presented in a single flow; the flow is fine for narrative but obscures the distinction.

- When discussing adjacent projects, the question is which walk-back (if either) applies. A project in git that tries to enforce process at a high-level layer inherits both. A project in git that enforces value at the kernel layer inherits the substrate walk-back but not the philosophy walk-back. A project outside git that enforces process inherits the philosophy walk-back but not the substrate walk-back. A project outside git that enforces value (Loom, F*-based systems, refinement-type stacks) inherits neither walk-back from aiwf's history, but faces the philosophy walk-back's *lesson* applied at the claim-authorship layer.

- When the arc evaluates new mechanical-enforcement proposals (the TDD architecture work, the contracts work, future kernel additions), the question to ask is the same one this document poses to adjacent projects: *is this enforcing process or value?* If process, the philosophy walk-back applies; the design should collect evidence rather than enforce. If value, a different analysis is needed; the design should make the value gate as hard to game by spec-weakening as possible (the techniques discussed in the spec-quality companion paper).

The discipline this document proposes is small. The arc's existing logic is intact; the conclusions stand. What changes is the precision with which the conclusions are stated and the cleanliness with which they transfer to adjacent settings. The arc has been good at narrowing scope; this document narrows the *logical* scope of the walk-backs in the same spirit.

---

## 7. The honest failure mode

This document has a failure mode worth naming.

**The distinction between substrate and philosophy walk-backs is wrong if the substrate walk-back is itself a *consequence* of the philosophy walk-back.** That is: if "event logs and hash chains fight git" is really a symptom of a deeper principle (e.g., "mechanical machinery that depends on serializable state fights collaborative substrates with non-serial coordination"), then the substrate walk-back is not separate from the philosophy walk-back; it is the same principle applied at a different layer.

I have considered this and rejected it, but it is worth marking the rejection. The reason is that the substrate walk-back's specific failures (concatenation breaking sequence numbers, monotonic counters allocating duplicates) are *git mechanics*, not general principles. A CRDT-backed substrate with appropriate merge semantics would carry an event log fine. A database with serializable transactions would carry a hash chain fine. The substrate failures are real but local; they are not symptoms of a deeper philosophical truth about software architecture.

The philosophy walk-back's failures, by contrast, are observable across substrates: the cheating attractor under LLM authorship of mechanical gates over process appears in coding agents (reward hacking literature), in TDD systems (the diagnostic this arc produced), and — by structural analog — at the claim layer in verification systems (the open question for Loom and adjacent projects). The dynamics are general; the specific manifestation is substrate-dependent only in the sense that *what counts as "process"* varies by substrate.

The asymmetry is what makes the distinction load-bearing. If both walk-backs were instances of the same principle, the arc could carry one lesson and apply it everywhere. Because they are not, the arc has to carry two lessons and ask which applies in which context. This document is the recorded reasoning for that discipline.

---

## 8. Open questions

The position does not settle several things.

1. **Are there other walk-backs lurking that the arc has not yet identified?** The two named here are the ones the arc has executed. A third — about, say, the framework's posture toward gradual or partial adoption, or about the framework's positioning relative to enterprise constraints the arc has scoped out — may be hiding in the work. Worth periodically checking.

2. **Does the philosophy walk-back generalize beyond LLM agents?** I have stated it specifically about LLM agents because that is the immediate context. Whether the cheating attractor is a function of LLM optimization specifically, or of *any* sufficiently capable agent under sufficient pressure, is an open question that the AI-safety literature on reward hacking has been engaging with for some time. The arc has not engaged.

3. **At what layer does the next walk-back happen?** If the substrate walk-back is at the storage layer and the philosophy walk-back is at the process-enforcement layer, the natural extrapolation is that another walk-back lies ahead at some layer the arc has not yet contested. The two layers most likely to produce one, by my read, are *the boundary between framework and consumer code* (the framework's coupling to consumer conventions like test paths, build commands, language choice — currently very loose, which may be a feature or may be hiding a brittleness) and *the boundary between human roles* (the orchestrator role currently treated as a single function, which may need to be split into authoring, reviewing, and operating roles under autonomous-agent pressure).

4. **Does the spec-quality companion work's "mutation testing on claims" technique survive its own walk-back-equivalent?** The technique is value-layer, not process-layer, so it does not fall under the philosophy walk-back's logic. But if it becomes a metric the LLM optimizes against directly, the same dynamic could re-emerge (writing decoratively complex specs that have high mutation kill rate while still being misaligned with intent). The technique is one signal in a layered defense for exactly this reason; future work has to test whether the layering holds under adversarial adaptation.

---

## 9. Conclusion

The aiwf research arc has executed two walk-backs from more ambitious positions, and they are not the same kind. The first was substrate-driven and applies specifically to projects operating in git. The second was philosophy-driven and applies specifically to mechanical enforcement of process against LLM agents. Conflating them — as I have done in conversation — makes the arc's conclusions look more universal than they are, leads to over-application to adjacent projects that do not share the relevant constraints, and obscures the question of where each lesson actually transfers. The clean separation: the substrate walk-back is local to git; the philosophy walk-back generalizes to any framework that tries to enforce process against LLM agents at a layer the agents can game. Adjacent projects inherit each independently — or, in many cases, neither — and face their own walk-backs at layers the arc has not yet treated. For Loom and similar refinement-type-driven projects, the walk-back ahead is at the claim-authorship layer, where the philosophy walk-back's *lesson* applies in a different form than aiwf encountered it. The arc carries forward two lessons, not one, and they should be cited separately when evaluating adjacent work.

---

## References

- [`00-fighting-git`](https://proliminal.net/theses/fighting-git/) — the substrate walk-back's full account.
- [`07-state-not-workflow`](https://proliminal.net/theses/state-not-workflow/) — the state-vs-render distinction, complementary to but distinct from the walk-back analysis here.
- [`08-the-pr-bottleneck`](https://proliminal.net/theses/the-pr-bottleneck/) — continuous ratification, the positive form of the philosophy walk-back's lesson.
- [`docs/explorations/06-tdd-diagnostic.md`](https://github.com/23min/aiwf/blob/main/docs/explorations/06-tdd-diagnostic.md) — the cheating attractor diagnosis.
- [`docs/explorations/07-tdd-architecture-proposal.md`](https://github.com/23min/aiwf/blob/main/docs/explorations/07-tdd-architecture-proposal.md) — the philosophy walk-back operationalized.
- [`11-should-the-framework-model-the-code`](https://proliminal.net/theses/should-the-framework-model-the-code/) — the compose-don't-absorb posture, which clarifies the boundary the framework holds with adjacent work.
- *The Verifiable Umbrella* and *Verifying the Verifier* (companion papers in the [Loom repository](https://github.com/23min/loom)) — the immediate adjacent setting this document distinguishes the arc's lessons from.

---

## In this series

- Previous: [`13 — Should aiwf adopt policy as a primitive`](https://proliminal.net/theses/policies-as-primitive/)
- Synthesis: [working paper](https://proliminal.net/theses/working-paper/)
- Reference: [KERNEL.md](https://github.com/23min/aiwf/blob/main/docs/research/KERNEL.md)
- Adjacent (external): [Loom](https://github.com/23min/loom) — *The Verifiable Umbrella* (architecture) and *Verifying the Verifier* (spec quality under LLM authorship)
