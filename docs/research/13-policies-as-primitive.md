# Should aiwf adopt policy as a primitive?

> **Status:** thesis-draft
> **Hypothesis:** "Policy" — as the exploration uses it — is a real and substantial territory: principles about engineering, design, testing, security, performance, naming, documentation, audit, and governance, expressed as binding normative claims with provenance and a lifecycle. That territory has its own audience (anyone working on or in the project) distinct from ADR's audience (the system's architecture), decision's audience (scope), and contract's audience (interface shape). The principled answer is **yes — policy earns a place as a first-class kind**, even though several structural primitives line up with ADRs. *Audience and reading discipline*, not structural shape, is what makes a kind a kind. The exact form of the primitive is deliberately undecided here; the exploration in [`docs/explorations/policies-design-space.md`](../explorations/policies-design-space.md) is where the form-question converges. One open sub-question this thesis surfaces but does not settle: whether *project-engineering policies* (about the consumer's software) and *framework-internal policies* (about aiwf's own operation) want one kind or two.
> **Audience:** anyone reading the policies exploration and asking whether the umbrella deserves a kernel primitive, a rename of an existing kind, or no change at all.
> **Premise:** the kernel ([`KERNEL.md`](KERNEL.md)) is the rubric; the audit voice is [`02`](02-do-we-need-this.md); the should-this-be-absorbed posture is [`11`](11-should-the-framework-model-the-code.md); the operating-model-agnostic positioning is [`12`](12-operating-model-agnostic.md). The exploration doc is the source material; this thesis takes a position on the should-question and explicitly defers the form-question.
> **Tags:** #thesis #aiwf #scope #kernel #policy

---

## Abstract

The exploration in [`docs/explorations/policies-design-space.md`](../explorations/policies-design-space.md) maps a real territory: a *policy* is a normative claim about state or process, written down with authority, intended to bind future actors. The territory is broad and concrete — engineering principles ("test the seam, not just the layer"), security postures ("no secrets in frontmatter"), performance contracts ("p95 latency under 200ms"), naming and documentation rules ("every public function has a doc"), citation and provenance requirements ("research docs cite older docs only"), audit requirements ("every entity-touching commit carries trailer X"), capability gates ("the `delete` verb is disabled in agent scopes"), governance and lifecycle commitments ("ADRs over 90 days proposed trigger review"). The exploration's §3 category list and §9 contracts-stretch table together describe what kinds of artifacts the framework would be carrying. None of these is the right home in any of the existing kinds: ADRs cover *architectural* decisions about the system being built; decisions cover *scope* commitments; contracts cover *bilateral interface* shape with mechanical verification. Cramming process rules, engineering principles, security postures, or audit policies into ADR would break what readers expect from an ADR (Nygard's recognizable artifact about system structure) and would smuggle a foreign subject into a kind whose audience is different. The audit voice from [`02`](02-do-we-need-this.md) is *do not add a new kind if an existing one serves the need*; the structural primitives a policy would need (id, status, supersedes-chain, ratification, retire, provenance) do line up with ADRs, but **structural shape is not the test for kindness**. The test is *audience and reading discipline*. Policy has both, distinct from ADRs. Therefore: **policy earns a place as a seventh kernel kind**. The exact form of that primitive — entity body, frontmatter, lifecycle states, waiver shape, relation to skills, cross-project portability, and whether project-engineering and framework-internal policies share one kind or split into two — is deliberately undecided in this thesis; that question is the work of the exploration converging on a defended `docs/design/policy-model.md`. What this thesis fixes is only the **should**: yes, with kernel-level standing, with ADR staying purely architectural.

---

## 1. The exploration's central claim, restated

The exploration's working definition: *a policy is a normative claim about state or process, written down by someone with authority, intended to bind future actors — humans and AI alike.*

Three load-bearing words: *normative* (what should be true, not what is), *authority* (someone with standing to write it), *bind future actors* (the policy is meant to outlive the moment of writing).

The exploration's central question: should aiwf introduce **policy** as a unifying kernel primitive — a new entity kind with its own lifecycle, provenance, and verbs — or is the territory adequately served by extending one of the existing kinds (ADR, decision, contract)?

This thesis takes a position on the should-question. It deliberately does *not* take a position on the form-question — what fields, what status set, how waivers look, whether shape A / B / C from the exploration's §7 wins. That converges in the targeted design session the exploration's §13 names as the next step.

## 2. What "policy" actually covers — name the territory before testing it

A common error in scoping debates is to argue about a kind without first stating concretely what artifacts the kind would carry. This thesis names the artifacts before testing the should-question, drawing directly from the exploration's §3 *categories with examples* and §9 *contracts-stretch* tables.

The territory, with concrete examples (paraphrased and grouped from the exploration; not re-listed exhaustively):

**Engineering principles and methodology.**
- *Test the seam, not just the layer.*
- *No `//nolint` without a one-line rationale.*
- *Go code is `gofumpt`-clean.*
- *The kernel package may not import the rituals package; the dependency arrow is one-way.*

**Security and safety postures.**
- *No secrets in frontmatter.*
- *No `eval` or arbitrary-code-execution in security-sensitive paths.*
- *Inputs at trust boundaries are validated by schema.*

**Performance and operational contracts.**
- *p95 latency under 200ms on the public API.*
- *Memory ceiling per worker is 512 MiB.*
- *Test coverage on internal packages stays ≥90%.*

**Naming, documentation, and discoverability.**
- *URLs follow pattern `/<resource>/<id>`.*
- *Every public function has a docstring.*
- *Every verb is reachable through `<verb> --help` or an embedded skill.*

**Citation, dependency, and provenance.**
- *Research docs may cite older docs only; the citation graph is acyclic.*
- *Every entity-touching commit carries the structured trailers.*
- *A response doc may not pre-date the doc it responds to.*

**Verification and quality thresholds.**
- *Every kernel finding code is asserted by string in at least one test.*
- *Mutation testing must pass at ≥70% on critical packages.*
- *Fuzz tests run nightly on the parsing layer.*

**Process, lifecycle, and freshness.**
- *A milestone may not be promoted to `done` while any acceptance criterion is `open`.*
- *An ADR more than 90 days `proposed` triggers a review reminder.*
- *No force-merge to main.*

**Capability gating.**
- *The `delete` verb is disabled in agent scopes unless explicitly authorized.*
- *Only humans may invoke `--force`.*

**Governance and authority.**
- *Only humans waive policies; agents cannot self-grant exemptions.*
- *Time-bounded waivers expire automatically and re-surface findings.*

This list is the concrete shape of the territory. Some of these clearly look like contracts (the performance and security ones, especially); some look like project conventions (Go-style, naming); some look like meta-rules about aiwf itself (capability gating, audit trailers); some look like governance about *the policy system itself* (only humans waive). They share one structural feature — *a normative claim about how this project works, with authority, intended to bind future actors* — and they share an audience — *people working on or in the project, including the AI* — but the subject matter ranges widely.

That range is what makes the should-question non-trivial. The next two sections audit it.

## 3. Two sub-classes the territory contains

Looking at the territory in §2, two distinguishable sub-classes emerge. The thesis names them so the rest of the audit does not silently slide between them; the form-question of whether they share one kind or split into two is deferred.

**Project-engineering policy.** Policies about *the software the team is building* — engineering principles, design rules, testing discipline, security postures, performance budgets, naming, doc-presence, security scanners' verdicts. Read by anyone editing or reviewing the project's code; cited by skills that govern AI-assisted code work. Examples: *test the seam*, *no secrets*, *p95 latency ≤ 200ms*, *every public func has a doc*. The audience is *project engineers*; the question they bring is *what must be true of the system I am editing?*

**Framework-internal policy.** Policies about *how aiwf itself operates inside this consumer's repo* — capability gates on aiwf verbs, audit trailers, governance rules over waivers, lifecycle freshness rules over framework entities themselves. Read by framework maintainers and operators; cited by aiwf's own skills and meta-policies package. Examples: *agents may not invoke `--force`*, *every entity-touching commit carries trailers*, *only humans waive*, *kernel may not import rituals*. The audience is *framework operators and maintainers*; the question they bring is *what must be true of how aiwf is operated here?*

These two sub-classes share the exploration's working definition (normative, authority, bind future actors), share most structural primitives (id, status, supersedes-chain, provenance, waiver mechanics), and share lifecycle moments (proposed → accepted → in-effect → waived/superseded → retired). They differ in audience, in cadence (project-engineering policies churn with the system; framework-internal policies are typically more stable), and in *who has standing to author and ratify them* (project teams vs. framework owners or framework adopters with delegated authority).

The form-question this raises — *one kind with a `subject:` field that distinguishes the two? Two sibling kinds with overlapping primitives? One kind with two distinct lifecycle profiles?* — is real and worth flagging, but it is a form decision and belongs in the design session, not this thesis.

What the thesis *does* fix: the territory in §2 is the territory the framework would be taking on if it adopts policy as a kind. Both sub-classes are in scope.

## 4. Why the structural-shape test is wrong

A first-pass audit of the should-question would walk the structural primitives a policy would need against what ADRs already provide.

- id, status, supersedes / superseded_by, scope and rationale, provenance — all present.
- The policy lifecycle (proposed → accepted → in-effect → waived/superseded → retired) is structurally close to ADR's (proposed → accepted → superseded → cancelled).
- The supersession DAG is identical in shape.
- The principal × agent × scope provenance model is identical.

If the test is "does the structural shape line up?", the answer would be yes, and the audit would conclude *extend ADR, do not add a kind*.

This audit is wrong. It conflates structural shape with kindness, and structural shape is not what makes a kind a kind.

The right test is **audience and reading discipline**. [`02`](02-do-we-need-this.md) §6 made this exact observation in a different register: *structured state pays off for programmatic consumers, not for AI assistants. Confusing these two audiences inflates the design.* Audiences differ in who reads the artifact, when, with what expectations, and what they need from it. Two artifacts can have identical structural primitives and still be distinct kinds because the readers' contracts with them differ.

Two further considerations push the same way:

- **The framework's commitment to honest naming.** The kernel's posture (per [`07`](07-state-not-workflow.md) §7.3): refuse asymmetric kinds that smuggle one subject into another. Widening ADR to carry engineering principles, security postures, performance contracts, citation rules, capability gates, audit policies, or governance rules would be exactly that smuggling — the kind's *name* would still be "ADR," but its contents would no longer be recognizable as architectural decisions about the system.
- **Industry recognizability of ADR.** ADR as an artifact is well-established outside aiwf — Nygard's 2011 framing, the *Architecture Decision Records* community, Joel Parker Henderson's templates, ThoughtWorks Radar adoption tracked since 2016. Readers walk into an `adr/` folder with a learned expectation: structural decisions about *the system being built* — module boundaries, technology choices, data model trade-offs, integration patterns. *"Test the seam"* and *"p95 latency ≤ 200ms"* and *"only humans may invoke `--force`"* do not match that expectation. Polluting it would cost more than it saves.

Net: structural primitives lining up is necessary but not sufficient. ADR's audience is people thinking about the system's architecture; policy's audience (in either sub-class) is people thinking about how the project works, with the AI in the loop, with what's allowed and what's gated, with what must be true of the code and of the operation. Same shape, different reading discipline, different kind.

## 5. The kernel test, redone with the territory in mind

Walk a policy primitive against [`KERNEL.md`](KERNEL.md)'s eight needs, asking *does the framework currently serve this need for the territory in §2, and well?*

1. **Record planning state** — *partially*. ADRs record architectural state; decisions record scope state; contracts record interface state. *Engineering-principle state*, *security-posture state*, *performance-budget state*, *audit-rule state*, *capability-gate state* are recorded today in scattered prose (skills, `CLAUDE.md`, ad-hoc rules, lint configs, Go meta-policies) without a queryable, supersedable, provenanced home. Policy as a kind closes this gap.
2. **Express relationships** — *partially*. Policies relate to entities they govern (a security policy applies to a contract; a citation policy applies to research docs), to other policies (supersession, contestation), and to skill content (a skill cites a policy). The first relation has no current shape; the others are inconsistent.
3. **Support evolution** — *partially*. Policies need to evolve via supersession with full provenance, including time-bounded waivers, contestation, and retirement. Today the framework has supersession for ADRs; the richer waiver story (the exploration's §5) has no current home.
4. **Keep history honest** — *yes, this is one of the strongest cases*. Provenance for policy events (propose, ratify, apply, waive, supersede, retire) is rich, structured, and uniform across the territory. Doing it inside ADR would mean ADRs grow waiver verbs, application events, and a richer status set that pollute the architecture-decision reading.
5. **Validate consistency** — *yes, this is the second strongest case*. Acyclicity in citation policies, "every applied policy must exist," waiver-expiration validation, "every 'you must' skill claim cites a policy," "every contract has at least one verifying validator referenced by a policy" — all are mechanical checks that benefit from a typed entity to check against.
6. **Generate human-readable views** — *partially*. A policy report (what's in effect, what's waived, what's superseded, with cited skills and findings) is a render. Renders need canonical state; canonical state needs a home.
7. **Coordinate AI behavior** — *yes, this is the third strongest case and the most operating-model-relevant*. [`12`](12-operating-model-agnostic.md) argued the framework is operating-model-agnostic; that posture means the framework cannot prescribe an operating model, but it can let *consumers* record the engineering principles, gates, and norms of whichever operating model they pick as policies the AI is bound by. Skills are the ergonomic surface; policies are the durable backing. Without a policy primitive, "you must" claims in skills float free.
8. **Survive parallel work** — neutral. Policies merge the same way ADRs do.

Net: needs (1), (4), (5), (7) are served partially or not at all today, with the load currently scattered across ADRs, skills, `CLAUDE.md`, lint configs, the meta-policies package, and ad-hoc convention. A policy primitive consolidates the load on its own audience, leaving the other kinds clean.

This is the inverse of the conclusion a structural-shape audit would reach, and it is the right one because the test was about audience, not shape, and the territory is concrete enough (per §2) that the audience is concrete too.

## 6. What policy is *not* — the boundary against ADR, decision, contract

Policy as a kind earns its place only if its boundary against the existing kinds is clean. State the boundaries:

- **ADR** — *binding decisions about the system's architecture: module boundaries, technology choices, data model trade-offs, integration patterns, the structural shape of the thing being built.* Read by anyone editing the system; the question they bring is *how is this system architected and why?*
- **Decision** — *binding commitments about scope: what is in, what is out, what is deferred, what is the team's stance on a scope question.* Read by anyone planning; the question they bring is *what have we committed to do or not do?*
- **Contract** — *bilateral interface shape with mechanical verification: API surfaces, schemas, behavioral assertions verifiable by a runner.* Read by anyone writing on either side of the interface; the question they bring is *what shape must this surface have?*
- **Policy** — *binding rules about how this project works: engineering principles, design and testing discipline, security and safety postures, performance and operational thresholds, naming and documentation rules, citation and provenance requirements, capability gates, governance commitments, waiver mechanics.* Read by anyone working on or in the project — including the AI — at moments when the question is *what must be true here, what's allowed, what's gated?*

The clean test: an artifact belongs to the kind whose audience-question it answers.
- *"Why is the storage layer eventually consistent?"* — ADR.
- *"Are we shipping multi-tenant in v1?"* — decision.
- *"What request shape does `/api/foo` accept?"* — contract.
- *"Test the seam, not just the layer."* — policy.
- *"No secrets in frontmatter."* — policy.
- *"p95 latency must stay under 200ms."* — policy.
- *"Every public function must have a docstring."* — policy.
- *"Research docs may cite older docs only."* — policy.
- *"The `delete` verb is disabled in agent scopes."* — policy.
- *"Only humans waive."* — policy.

Each test cleanly assigns to one kind. None benefits from being smuggled into another. The boundary holds across both project-engineering and framework-internal sub-classes.

A subtler boundary worth naming: **policy versus contract**. A contract is *a bilateral, mechanically-verified interface shape* — there is a producer side and a consumer side, and a runner that verifies the surface. Some of the territory in §2 (security postures, performance budgets) borders on contract because it has a runner. The distinguishing line: a contract describes *what an interface must accept or produce*; a policy describes *what the project must be true about*. *"This endpoint accepts requests matching schema X"* is a contract; *"every endpoint at trust boundary X validates inputs by schema"* is a policy whose enforcement may be contracts on each endpoint. The exploration's §9 stretch table makes this distinction implicit; this thesis makes it explicit.

## 7. The operating-model-agnostic angle

[`12`](12-operating-model-agnostic.md) argued the framework's bet does not select for an operating model. That argument *strengthens* the case for a policy primitive, not weakens it.

If the framework is operating-model-agnostic, the consumer is the one who picks the operating model and chooses the engineering principles, security postures, performance budgets, governance rules, and audit requirements that go with it. Each set is a body of policies — durable, supersedable, with provenance, read by humans and the AI alike.

Today consumers do this in `CLAUDE.md`, in `AGENTS.md`, in `.cursorrules`, in scattered skill prose, in lint configs, in CI scripts. The framework's response to "how does the consumer record their project policies?" is silence at the kernel level, prose-only at the skill level, and a fragmented enforcement story across third-party tools. That is a gap the operating-model-agnostic positioning makes more visible, not less.

A policy primitive lets the framework be agnostic about *which* operating model the consumer runs while giving them a typed home for *recording* the project policies whichever they pick. It is the missing surface that makes operating-model-agnosticism actually work.

## 8. The compose-don't-absorb posture, applied honestly

[`11`](11-should-the-framework-model-the-code.md) committed to *compose, don't absorb* with code-graph tools. The posture pulls in different directions for the two sub-classes named in §3.

**Project-engineering policies compose with adjacent enforcers.** Linters, type checkers, security scanners, performance benchmarks, mutation testers, fuzzers, doc-presence checkers all already exist as third-party tools. The framework should *not* absorb their roles. A policy entity for "no secrets in frontmatter" points at a secret scanner the consumer wires into the verifier hook; a policy for "p95 latency ≤ 200ms" points at the consumer's benchmark runner. The framework owns the *policy entity, its provenance, its lifecycle, and its citation by skills*; the framework does *not* own the secret scanner or the benchmark runner. This is exactly the `11` posture: expose stable surfaces, let adjacent tools be authoritative for what they are authoritative for.

**Framework-internal policies have no adjacent owner.** "Only humans waive," "every entity-touching commit carries trailers," "the kernel may not import rituals" — these are about aiwf's own operation. There is no third-party tool that owns this territory because the territory is the framework itself. Internal-policy enforcement lives in aiwf's verifiers and meta-policies package; the policy entity is the durable, supersedable record of the rule the verifier checks.

Both sub-classes earn a kernel slot; the form they take inside that slot may differ in how they reference enforcement. This is the kind of detail the form-question has to settle.

This is *not* contradicting `11`. It is applying the same test honestly: when adjacent tools own the territory, compose; when no adjacent tool does and the territory is squarely the framework's lane, the kernel grows. For project-engineering policies the adjacent tools are *enforcers*, not *owners of the policy itself* — the policy entity belongs to aiwf even when its enforcement does not.

## 9. What stays out of scope in this thesis

The thesis fixes the should. It does not fix the form. Several real questions remain open and belong in the exploration converging on a defended `docs/design/policy-model.md`:

- **One kind or two** for project-engineering vs. framework-internal policies. §3 named the sub-classes and flagged the question; the exploration has to answer it. Trade-off is between schema simplicity (one kind, possibly with a `subject:` or `audience:` field) and audience clarity (two kinds with deliberately distinct names).
- **Form (shape A / B / C from the exploration's §7).** Prose + frontmatter, structured DSL, code-as-policy, or a hybrid. The exploration's recommendation of "all three coexist, with shape A as the index and shapes B/C as implementations" is plausible; a defended position needs to settle which combination ships first.
- **Status set.** *proposed → accepted → in-effect → waived/superseded → retired* is one shape; ADR-style with *waived* and *retired* added is another; collapsing *accepted* and *in-effect* is a third. Pick deliberately.
- **Waiver shape.** Trailers on commits vs. sub-entity on policies. Different implications for retroactive enforcement.
- **Relation to skills.** The "skills cite policies" discipline is unambiguous as a working norm; whether it is enforced (a CI gate detects "you must" prose without a citation) or merely recommended is open.
- **Subject taxonomy.** §2's groupings (engineering principle / security posture / performance / naming / citation / verification / process / capability / governance) are descriptive, not yet structural. Whether they become a closed enum, an open string, or stay informal is a design choice.
- **Enforcement-pointer shape.** A project-engineering policy points at one or more enforcers (a validator, a CI command, a runner). The form of that pointer (path, command, plugin id, MCP capability) is a design choice.
- **Verbs.** Existing aiwf verbs (add, promote, supersede, cancel) cover ADR-shape work. Policy may need additions (apply, waive, retire, contest) or may collapse onto existing verbs with subject typing. Open.
- **Cross-project portability.** The exploration's §12 identifies this as the deepest problem. The thesis stays silent; the exploration's three shapes (submodule, package, registry+sync) are real and any of them works downstream of the kernel kind landing.
- **Kernel evolution discipline.** Adding a seventh kind is a kernel-level change. The kernel changes by deliberate edit, with reasoning recorded. This thesis is the recorded reasoning for the *should*; the formal kernel edit follows the form-decision.

These are real open questions and the exploration is the right place for them to converge. This thesis does not preempt that work.

## 10. The honest failure mode

This position has a failure mode worth naming.

**Adopting policy as a kind is wrong if the territory turns out to be irreducibly heterogeneous.** Specifically:

- **If "policy" splits into more than two sub-classes** with distinct audiences and reading discipline, then unifying them under one kind would repeat the mistake this thesis warns against (smuggling distinct subjects into one kind to save a kernel slot). The §3 split into project-engineering and framework-internal is the simplest plausible structure; the design session has to test whether further splitting is needed (e.g. *governance* policies separate from *engineering* policies, or *audit* policies separate from both).
- **If most artifacts in §2 in practice belong to existing kinds** — i.e., on closer inspection, most "performance contracts" are contracts, most "engineering principles" are skill content, most "framework-internal rules" are ADRs about kernel structure — then the new kind would carry too few real artifacts to earn its keep. The right answer would be to sharpen the existing kinds and skill content discipline, without a new kind. The §2 territory is broad enough that this seems unlikely, but the design session has to test it concretely.
- **If the form question turns out to demand a non-aiwf shape** for substantial parts of the territory — i.e., a config layer, a feature-flag service, an OPA-shaped evaluator owns part of the territory — then "compose, don't absorb" applies for that part and the kernel grows only for what is left. Worth holding open until the form work happens.

For each failure mode, the principled answer is the same: the design session converges on the form, the form work either confirms the should or surfaces evidence to revisit it. This thesis is *should, pending form*; the should is conditional on the form work landing somewhere coherent.

## 11. What this implies, narrowly

The thesis is small. Its implications are also small, because it deliberately defers form:

1. **Plan the kernel for a seventh kind.** No code change today. Reserve the slot in the entity-vocabulary memory and any documentation that lists the kinds, with a footnote that says "policy is a planned addition, form pending."
2. **Stop trying to widen ADR's audience description.** ADR is architectural decisions, period. Earlier prose drafts (including in `13`'s prior version) that proposed widening to "binding decisions about how this project works" should be reverted. ADR stays pure.
3. **Run the targeted design session the exploration's §13 describes.** The session takes a position on each open question in §9 of this thesis (one kind or two, form, status set, waiver shape, enforcement-pointer shape, etc.) and produces `docs/design/policy-model.md`. That doc — not this one — is what authorizes the kernel edit.
4. **Hold the existing partial-shapes in place until the design lands.** Skill prose with "you must" claims continues to live in skills; the meta-policies package continues to be Go code; `CLAUDE.md` continues to carry advisory norms; lint configs continue to live where they live. None of these gets retroactively migrated until the policy form is settled.

Everything substantive happens in the design session. This thesis is the standing it walks into that session with.

## 12. Open questions

1. **Does the §2 territory survive a concrete-cases audit?** Pick five real artifacts the framework would record (e.g. *test the seam*, *no secrets in frontmatter*, *p95 latency ≤ 200ms*, *the `delete` verb is disabled in agent scopes*, *every entity-touching commit carries trailers*). Test each: does it cleanly belong to ADR, decision, contract, or policy? The §6 boundary tests suggest the assignment is clean; the design session can confirm with a wider sample.
2. **One kind or two?** The §3 sub-classes (project-engineering, framework-internal) want either one kind with a discriminator field or two sibling kinds. The right answer depends on how often skills, validators, and renders need to discriminate between them in practice.
3. **What is the smallest viable policy primitive?** The exploration's §13.4 asks this. Answer is form work.
4. **Does the kernel changing from six to seven kinds (or eight, if the §3 split lands as two kinds) shake any prior commitments?** The framework entity-vocabulary memory's "6 kinds, no `story` or `task`, by deliberate choice" stays true to its intent (no work-decomposition kinds); the new count becomes canonical and the memory updates to reflect it.
5. **What is the relationship between policy and the configuration layer?** A policy says "no force-merge to main"; the git-hosting platform's branch-protection setting *enforces* it. Policy is the source of truth and the audit record; the configuration is a derived enforcement. Whether the framework supports declaring this composition (policy → enforcement-target metadata) is an open form question.
6. **Does adopting policy retroactively re-classify any existing PoC artifacts?** Likely yes — the meta-policies package is policy-shaped, by name. Whether existing ADRs in the project also re-classify or stay where they are is a per-document call the design session can codify.

---

## 13. References

- [`KERNEL.md`](KERNEL.md) — the rubric. Walked in §5, with the audience-not-structure correction.
- [`02-do-we-need-this`](02-do-we-need-this.md) §6 — the audiences distinction. The structural-shape audit fails the audiences test; this thesis applies the test correctly.
- [`07-state-not-workflow`](07-state-not-workflow.md) §7.3 — refuse asymmetric kinds that smuggle subjects. Argues against widening ADR; argues for a policy kind on its own audience.
- [`11-should-the-framework-model-the-code`](11-should-the-framework-model-the-code.md) — the compose-don't-absorb method. Applied honestly here: project-engineering policies compose with adjacent enforcers (linters, scanners, benchmarks); framework-internal policies have no adjacent owner; both belong in the kernel because the *policy entity itself* is aiwf's lane in either case.
- [`12-operating-model-agnostic`](12-operating-model-agnostic.md) — the operating-model-agnostic positioning *strengthens* the case for a policy primitive: consumers need a typed home for recording the engineering principles and norms of whichever operating model they pick.
- [`docs/explorations/policies-design-space.md`](../explorations/policies-design-space.md) — the source material. §3 (categories) and §9 (contracts-stretch) describe the territory; §5–§7 sketch lifecycle, provenance, and form options. The thesis confirms the should; the form-question stays in the exploration's lane.
- The framework entity-vocabulary memory (six kinds, no `story` or `task` by deliberate choice) — the consistency this thesis preserves: the new kind(s) are *not* work-decomposition kinds; the memory's intent (no `story`/`task`) is intact.
- Michael Nygard, *Documenting Architecture Decisions* (2011) — the canonical ADR framing. Cited as evidence that ADR is an externally-recognizable artifact whose audience expectation must be preserved, not absorbed into a wider role.

---

## In this series

- Previous: [`12 — aiwf is operating-model-agnostic`](12-operating-model-agnostic.md)
- Related: [`02 — Do we need this`](02-do-we-need-this.md), [`07 — State, not workflow`](07-state-not-workflow.md), [`11 — Should the framework model the code?`](11-should-the-framework-model-the-code.md)
- Source exploration: [`docs/explorations/policies-design-space.md`](../explorations/policies-design-space.md) — where the form-question converges.
- Synthesis: [working paper](../working-paper.md)
- Reference: [`KERNEL.md`](KERNEL.md)
