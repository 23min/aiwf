# Policies as a Primitive: Design-Space Exploration

> **Status:** exploration
> **Audience:** anyone considering whether the framework should treat "policy" as a first-class concept, or how it would relate to existing kinds (ADRs, decisions, contracts).
> **Hypothesis (tentative):** much of what teams write into `CLAUDE.md`, `.cursorrules`, ADRs, contracts, and ad-hoc lint rules is a single underlying thing — a *policy* — and the lifecycle, provenance, and enforcement story for that thing is more uniform than it currently looks.
> **Tags:** #aiwf #policies #governance #exploration

---

## What this is, and what it isn't

This is a design-space exploration, not a proposal and not a decision. The goal is to map the territory cleanly enough that a future targeted session can produce a *defended position* — likely as `docs/design/policy-model.md` — rather than re-litigating the basics.

Some examples below come from concrete findings surfaced by the **PoC on the `poc/aiwf-v3` branch** (the v3 implementation, which is where mechanical-validation work currently lives). Surfacing findings of exactly this kind — places where the framework's existing concepts strain, or where adjacent concerns deserve their own treatment — is one of the PoC's stated jobs. Where I cite a PoC finding, I name it as such; the reader on `main` does not need the PoC code in front of them to follow the argument.

**Scope note: policy vs. governance.** This doc treats *policy* as the primitive — the unit that carries the rule. *Governance* is the larger system within which policies have force: the meta-rules and roles for who has authority, how policies enter and leave the system, what happens when they conflict, and how the meta-rules themselves change. Lifecycle (§5) and provenance (§6) below sit at the policy↔governance boundary; they are governance facets viewed through the policy lens. A full treatment of governance — authority structures, conflict resolution, amendment of the policy system itself — deserves its own exploration when the appetite is there, likely as `docs/explorations/governance-design-space.md`. This doc is deliberately the smaller of the two.

The shape of the exploration:

1. What "policy" actually is, and what it isn't.
2. Five axes of variation that separate one policy from another.
3. Concrete categories of policy, with examples.
4. The enforcement spectrum — six rungs from "remember it" to "prove it."
5. Lifecycle: how policies are born, applied, contested, superseded, retired.
6. Provenance: who authored, ratified, waived, superseded.
7. Forms: what a policy artifact actually looks like.
8. Relationship to existing kernel concepts (ADRs, decisions, contracts, skills).
9. How far the contracts concept can stretch.
10. Citation chains and DAG-style policies.
11. Spec-Driven Development adjacency — what the survey teaches.
12. Cross-project portability — what makes this hard.
13. Open questions for the targeted design session.

---

## 1. What "policy" actually is

A working definition: a **policy** is a *normative claim about state or process, written down by someone with authority, intended to bind future actors* — humans and AI alike. Three words in that definition are load-bearing:

- **Normative.** It says what *should* be true, not what *is*. ("Validators are advisory" is descriptive; "validators must be advisory unless `strict_validators: true`" is normative.)
- **Authority.** Someone with standing to write it. Provenance starts here; "who wrote this rule, and who's allowed to override it" is a first-class question, not an afterthought.
- **Bind future actors.** The policy is meant to outlive the moment of its writing. Otherwise it's a comment, or a chat, or a one-off correction.

Useful distinctions from neighbors:

| Concept | Distinguishing trait |
|---|---|
| **Convention** | Bindingness is implicit ("we usually do X"). A policy makes it explicit. |
| **Style guide** | Subject is form, not behavior. A special case of policy with narrow scope. |
| **Constitution** | Highest-level policy; expected to be more stable, harder to amend. The word implies a hierarchy. |
| **Contract** | A policy whose enforcement is structural agreement between two parties. Bilateral; verifiable. |
| **Specification** | A policy whose subject is *behavior* of a system. Narrower in scope, often more precise. |
| **Skill** (LLM) | Advisory documentation for an LLM — descriptive, occasionally prescriptive. Not all skills are policies. |
| **Rule** | Synonym in practice. "Policy" wins because it implies an authoring/authority loop. |
| **Governance** | The system within which policies have force; deals with authority, conflict, and amendment of the policy system itself. Policy is the unit; governance is the system that operates on units. See the scope note above. |

This matters: you cannot enforce a thing whose bindingness is unclear, you cannot waive a thing whose authority isn't named, and you cannot supersede a thing whose lifecycle is undefined. The naming exercise is not pedantry — it is a precondition for being able to *operate* on the thing.

---

## 2. Five axes of variation

Every policy varies along these axes. Tooling and process choices fall out of where you sit on each:

1. **Subject** — what is being constrained: code, config, docs, process, identity, infrastructure.
2. **Bindingness** — advisory, warning, blocking. Cuts across all subjects.
3. **Mechanism** — how the rule is checked: human reading, LLM, regex, AST, schema, type system, runtime check, formal proof.
4. **Locus** — where it runs: in conversation (LLM-side), at write time (lint/format), at commit, at push, at CI, at runtime.
5. **Lifecycle stage** — proposed, accepted, in-effect, waived, superseded, retired.

Two policies with the same subject can differ wildly on the other four ("don't expose secrets" can be advisory prose, a regex pre-commit, or a CI-gating scanner). Two policies with the same bindingness can have wildly different cost-to-enforce. The axes are mostly independent.

---

## 3. Concrete categories with examples

A non-exhaustive map, with one-line examples:

| Category | Example |
|---|---|
| **Permission / restriction** | "Only humans may invoke `--force`." |
| **Process / workflow** | "A milestone may not be promoted to `done` while any acceptance criterion is `open`." |
| **Specification** | "The HTTP layer must accept payloads conforming to `schema/v1.json`." |
| **Style / convention** | "Go code is `gofumpt`-clean; no `//nolint` without a one-line rationale." |
| **Architectural** | "The kernel may not import the rituals package; the dependency arrow is one-way." |
| **Verification** | "Every kernel finding code must be asserted by string in at least one test." |
| **Documentation** | "Every verb must be reachable through `<verb> --help`, an embedded skill, or `CLAUDE.md`." |
| **Provenance / audit** | "Every entity-touching commit must carry the structured trailers." |
| **Citation / dependency** | "A research doc may cite older docs only; the citation graph is acyclic." |
| **Quality threshold** | "Test coverage on internal packages should stay ≥90%." |
| **Lifecycle / freshness** | "An ADR more than 90 days `proposed` triggers a review reminder." |
| **Capability gating** | "The `delete` verb is disabled in agent scopes unless explicitly authorized." |

Several of these have direct PoC analogues. The PoC's `provenance-*` finding family enforces audit policies; the `discoverability` policy enforces a documentation policy; the `findings_have_tests` policy enforces a verification policy. The categories framing is the umbrella that makes their kinship visible.

---

## 4. The enforcement spectrum — six rungs

Each rung is a distinct mechanism with distinct cost, latency, and reliability characteristics:

| Rung | Mechanism | Latency | Determinism | Cost to author | Failure mode |
|---|---|---|---|---|---|
| 0 | **LLM memory** (just remember) | Conversation | None | Zero | Forgotten next session |
| 1 | **Markdown reminder** (CLAUDE.md, skill) | Compose-time | LLM judgment | Low | Ignored under context pressure |
| 2 | **Pattern lint** (regex / AST / glob) | Pre-commit / pre-push / CI | High | Medium | Brittle to refactors |
| 3 | **Schema / type check** | Build-time | High | Medium-high | Locks in shape |
| 4 | **Runtime contract / assertion** | Runtime | High | High | Late detection; performance cost |
| 5 | **Formal proof** | Build-time (proof obligation) | Total | Very high | Vanishingly rare in practice |

Cross-cutting observations:

- **Most policy failures escape because the rung is too soft for the bindingness claimed.** A finding from the PoC retrospective on its upgrade-flow iteration (gaps `G27` / `G28` / `G29` in `docs/pocv3/gaps.md` on the PoC branch) is exactly this shape: rules that should have been rung 2 (pattern + integration test) sat at rung 1 (a reasonable expectation in the author's head). Bugs shipped. The fix in each case was to add the missing rung-2 enforcement *and* document the underlying policy at rung 1 so future authors know the *why*.
- **Rung 1 is necessary even when rungs 2–5 exist.** A regex check that says "wrong" without saying *why* puts the cost of explanation back on the author at every transgression. The skill / CLAUDE.md prose is the explanation; the regex is the gate.
- **Rungs are composable per policy.** A single policy can have both a markdown rationale (rung 1) and a CI check (rung 2). That is actually the healthy state — the prose explains, the check enforces. Most of the PoC's existing checks operate this way: a finding code, a hint message (rung 1 inside the gate), and a deterministic detector (rung 2).
- **Rung 0 is a smell, not a rung.** If a policy lives only in conversational memory, it is not a policy yet.

The Spec-Driven Development survey at [`docs/research/surveys/understanding-spec-driven-development.md`](../research/surveys/understanding-spec-driven-development.md) maps onto this spectrum: Marc Brooker's "free-form natural language → RFC 2119 / EARS → Lean / TLA+" formality ladder is exactly rungs 1 → 3 → 5 for the *specification* sub-category of policy. The lessons from formal-methods narrowness (rung 5 is real but rare) port directly to the other policy categories.

---

## 5. The lifecycle / loop

A useful policy has roughly the same lifecycle as an aiwf entity:

```
proposed → accepted → in-effect → [waived | superseded] → retired
                          ↑              ↓
                          └─── revised ──┘
```

Several events deserve to be first-class:

- **Ratification.** Someone with authority moves the policy from proposed to accepted. For most teams this is implicit; making it a verb makes it auditable.
- **Distribution.** The moment the policy propagates from one repo to another. Cross-project portability lives here.
- **Application.** Each time the policy is enforced (or not). The ratio of "applied" to "waived" is a health signal.
- **Waiver.** A scoped, reasoned, time-bounded "this case is exempt." Waivers should be expensive to issue, cheap to inspect, and never permanent. The PoC's `--force --reason "..."` is the in-the-moment shape; a *committed* waiver would be the durable one.
- **Contestation.** An actor proposes the policy is wrong. Becomes a new proposal that supersedes the old.
- **Supersession.** Like ADRs — a new policy explicitly replaces an old one with a directed edge.
- **Retirement.** The policy is no longer in effect. Distinct from supersession (which has a successor) and from cancellation (which says it was never adopted).

Two failure modes the loop has to defend against:

1. **Drift** — the policy and the system diverge silently. Mitigation: every policy should have a continuously-running enforcement (a check, a test, a gate) so divergence surfaces. A policy with no rung-2-or-above enforcement is on the path to drift by default.
2. **Calcification** — the policy outlives its rationale; everyone follows it without remembering why. Mitigation: every policy carries its **why**. The "Why:" section in feedback memories or ADRs is exactly this. When the why is dead, the policy retires.

---

## 6. Provenance — the rich part

The PoC's existing provenance model has the most to offer here. Every policy event is principal × agent × scope:

| Event | Principal | Agent | Trailer / record |
|---|---|---|---|
| Propose | human (or agent in scope) | same | `policy-verb: propose` |
| Ratify | human (always, ideally) | same | `policy-verb: ratify` |
| Apply (gate) | n/a | the gate itself | `policy-applied: <code>` (commit trailer when blocking) |
| Waive | human | human or agent-in-scope | `policy-waived: <code>` `policy-reason: "..."` |
| Supersede | human | same | `policy-verb: supersede` `policy-supersedes: <prior>` |
| Retire | human | same | `policy-verb: retire` |

What this buys:

- **Querying "who waived this and why"** becomes a `git log` filter on the trailer family — same chokepoint as the PoC's `aiwf history`.
- **Supersession is a DAG**, identical to ADRs. The "if A.superseded_by = B then B.supersedes ⊇ {A}" mutuality rule is the same audit. The infrastructure is already there.
- **Force is sovereign.** Same rule as the PoC enforces today: only humans waive. An agent operating in a scope cannot self-grant a waiver, even when authorized for the work.
- **Time-bound waivers** are a natural extension: `policy-waived-until: 2026-06-01`. Past that date, the policy reapplies and any commits that depended on the waiver should re-surface as findings.

The citation-chain idea fits cleanly here. A research-doc DAG ("doc A cites docs B, C") *is* a provenance graph. The non-circularity rule is the same as the existing `no-cycles` finding for milestone `depends_on` and ADR `supersedes` chains. The check kind generalizes; the subject just changes from "milestone DAG" to "doc DAG."

---

## 7. Forms — what a policy artifact looks like

Three viable shapes, with different tradeoffs:

**Shape A — Prose + frontmatter (the aiwf-entity shape).** YAML frontmatter for the kernel-readable bits (id, status, supersedes, severity, enforcement-mechanism), markdown body for the rationale and specifics. Same shape as ADRs and decisions:

```yaml
---
id: P-001
status: in-effect
severity: error
enforcement: [pre-commit-hook, ci-test]
supersedes: []
---
# Test the seam, not just the layer

Why: ...
How to apply: ...
```

Pro: human-readable, lives in the repo, version-controlled, plays with existing aiwf tooling. Con: prose enforcement still depends on the LLM honoring it; there is a leap from prose to mechanism.

**Shape B — Structured DSL.** YAML or a custom format that is machine-evaluable end-to-end. OPA/Rego is the canonical example; CUE is another:

```cue
policy "no-secrets-in-frontmatter": {
    applies_to: "entity"
    rule: not_match(frontmatter, /^(api[_-]?key|password|token)/)
    severity: "error"
}
```

Pro: deterministic, no prose-to-mechanism leap. Con: heavy upfront cost, narrows expressiveness, needs an evaluator. Most policies do not *want* this much rigidity.

**Shape C — Code-as-policy.** The policy is a function that takes the system state and returns findings. The PoC's `internal/policies/` package already does this for the meta-rules it enforces (one Go function per policy, each producing findings).

Pro: fully expressive; lives in the same language as the system; testable like any other code. Con: opaque to non-engineers; a non-trivial barrier to adding a policy.

The honest answer is **all three coexist**: shape A for the human-facing rationale and provenance metadata, shape C for the deterministic check, optionally shape B for a portable subset. Shape A is the *index*; shapes B/C are the *implementations*. A single policy entity carries pointers to its enforcement implementations.

This is how aiwf already works for contracts: the contract entity (shape A) carries pointers to a validator and a schema file (shape B / C, depending on the validator). The pattern generalizes.

---

## 8. Relationship to existing kernel concepts

This is where the overlap question gets real. The PoC currently has:

- **Skills** (`aiwf-*` materialized skills) — advisory, prose-only, LLM-side. Rung 1.
- **CLAUDE.md** — advisory, prose-only, project-level. Rung 1.
- **Contracts** — bilateral, structural, with verify-and-evolve passes. Rungs 2–4 depending on the validator.
- **ADRs** — architectural decisions; supersedes-chain; advisory unless wired to a check.
- **Decisions** — scope-shaping commitments; same shape as ADRs.
- **Findings / checks** — the enforcement side; rung 2 (some are rung 3 via schema validation).
- **Meta-policies package** — meta-rules about the kernel itself; rung 2 in shape C.

A policy umbrella over these would observe:

- ADRs are policies whose subject is *architecture*. They have a status set, a supersedes chain, a why, an authoring trail. They are already most of the way to being policies.
- Contracts are policies whose subject is *interface shape*. They have a verify pass.
- Decisions are policies whose subject is *scope*. They have a supersedes chain.
- The existing meta-policies package is policies whose subject is *the kernel itself*. It already calls itself "policies."
- Skills are not policies — they are advisory documentation. But they often *cite* policies, and a skill that asserts "you must do X" is straying into policy territory and should ideally be backed by a policy entity.

The umbrella might be: **one entity kind that absorbs ADR + Decision + the meta-policies into a single shape, with a `subject:` field**. Keep contracts separate (they have their own bilateral structure). The win is consolidation: one supersedes-chain, one waiver mechanism, one provenance flow.

The cost: it would be a kernel-level move. Existing ADRs and decisions would need to map onto the new shape. Worth it only if the umbrella earns its keep across multiple repos. The PoC's deliberate principle of "build it when the second case shows up" applies here.

---

## 9. How far the contracts concept stretches

Today contracts in the PoC mean "schema validation of a data shape with fixtures." The expansion path is wide:

| Extension | Validator | Difficulty | Value |
|---|---|---|---|
| **API contracts** (current) | `cue`, `ajv`, OpenAPI tools | Already done in PoC | High |
| **Behavioral contracts** (Eiffel-style pre/post) | language-specific assertion frameworks | Medium | Medium |
| **Performance contracts** (latency budget, memory ceiling) | benchmark runners | High | Niche |
| **Security postures** (no secrets, no `eval`) | secret scanners, AST linters | Low-medium | High |
| **Naming contracts** (URLs follow pattern X) | regex / AST | Low | Low-medium |
| **Tree-shape contracts** (this dir contains files matching X) | path globbing + count | Low | Low |
| **Doc-presence contracts** (every public func has a doc) | godoc / sphinx checks | Low | Medium |
| **Citation contracts** (this doc must cite that one) | markdown link parser | Low | Medium |
| **Provenance contracts** (every commit carries trailer X) | git-log scanner | Already done in PoC | High |

The pattern: contracts work whenever you can write down "valid example, invalid example" and run them through a validator. As the subject gets fuzzier (performance, behavior, naming intent), the validators get harder to write and the false-positive/negative rate climbs.

The honest line: **contracts work brilliantly for shapes that have a closed grammar and a reproducible runner.** They strain at *intent*. "The API is RESTful" is not a contract; "every endpoint accepts a request matching `schema/req.json`" is. Push contracts as far as you can write down a runner; stop pretending when you cannot.

---

## 10. Citation chains and DAG-style policies

The framework already enforces acyclicity for two relations: milestone `depends_on` and ADR `supersedes`/`superseded_by`. The check kind is general; only the subject changes.

For documentation citation chains specifically:

- A research doc (or any doc) declares `cites: [doc-X, doc-Y]` in its frontmatter.
- A policy says: "the citation graph is acyclic" — fails when a cycle would form.
- A second policy says: "newer docs may cite older docs only" — depends on a temporal ordering, which can come from frontmatter dates or from `git log` of file creation.
- A third policy says: "every cited doc must exist" — the standard refs-resolve story.

These all follow the same shape as the PoC's existing `refs-resolve` and `no-cycles` validators. The lift to add them is small *if* citation is modeled as a structured frontmatter field rather than embedded in markdown link text.

A subtler case worth naming: **citations as a provenance signal for the docs themselves**. If doc B was written in response to doc A, the relation is `responds-to: A`, not `cites: A` — and the policy is "a response doc may not pre-date its target." This is the same shape as the PoC's authorize-scope provenance (the scope must exist before the act). The pattern recurs.

---

## 11. Spec-Driven Development adjacency

Several lessons from the SDD survey port directly to policy:

- **Bindingness fragmentation matches taxonomy fragmentation.** Spec-as-prompt → policy as "vibes." Spec-first → policy as written intent. Spec-anchored → policy that lives across changes. Spec-as-source → policy that *generates* the system. Each rung has a different cost-of-drift. Same as policies.
- **Brooker's three formality tiers** (free-form prose / RFC 2119 EARS / Lean+TLA+) is the policy enforcement spectrum, restricted to the *specification* category. The "use the formality you actually need" message ports verbatim.
- **Beck's critique** — "encodes the assumption that you won't learn during implementation" — applies to policies that are too rigid to evolve. A policy that cannot be revised after it is authored is a frozen burden. The supersedes-chain is the antidote: policies are *iterable*, just with provenance.
- **Drift is the universal failure.** "Stale specs mislead agents that don't know any better." Replace "specs" with "policies" and the sentence is identical.
- **Brownfield breakage.** SDD tools work on greenfield, fail on brownfield. Same threat: a policy framework that requires re-stating the whole system before it can govern any of it is unusable in repos that already exist. The OpenSpec lesson — *deltas, not full rewrites* — applies to policy adoption: a new policy lands as a delta, not a "now restate every project against the new framework."

---

## 12. Cross-project portability — what makes this hard

The natural ask is: pull policies into any project. Three honest constraints:

1. **Rung-1 portability is trivial.** A markdown file copies. A CLAUDE.md fragment imports. The cost is "remembering to copy."
2. **Rung-2+ portability is hard.** Pre-commit hooks, CI checks, type/schema validators all have project-specific assumptions (language, build system, test framework). A "test the seam" hook for Go and a "test the seam" hook for Python are different code, even if the policy is the same.
3. **Provenance across projects is the deepest problem.** If a policy is ratified in project X and pulled into project Y, what does "ratified" mean in Y? Did Y's authors agree? Can Y waive? When X supersedes the policy, does Y auto-update or stay pinned?

Three viable shapes, none free of cost:

- **Submodule / vendored copy.** Policy lives in a central repo, projects pull it in. Pro: explicit version pinning. Con: every project pays update cost.
- **Plugin / package.** Like aiwf's rituals plugin shape (in the PoC's companion repo). Pro: install-once, follow updates. Con: requires a packaging story per ecosystem.
- **Registry + sync.** Policies have stable ids; projects subscribe; updates propagate. Pro: one source of truth. Con: heavy infrastructure; needs a server or a well-known git repo + tooling.

The pragmatic middle: **a single repo of policy entities (shape A), each pointing at language-/tooling-agnostic specifications, with per-target enforcement implementations (shape C) shipped separately.** Project pulls in the policies, then opts in to the implementations that match its stack. Provenance lives in the central repo; projects waive locally.

This is roughly the OPA model (Rego centralized, evaluators distributed), the GitHub-rulesets model, and the ESLint-config model. All three have working precedent.

---

## 13. Open questions for the targeted design session

Holding these as questions, not assertions:

1. **Is "policy" really an umbrella, or just a synonym for "the existing things we have"?** If ADRs, decisions, contracts, and meta-rules already cover 90% of the territory, the umbrella may be premature consolidation. Honest test: what category of policy *cannot* fit into one of those four today? Quick guess: process / workflow policies and citation/dependency policies.
2. **Does enforcement always need to live in the same repo as the policy?** A central policies repo with distributed enforcers is one shape; a per-repo "policies + their checks together" is another. Different cost models.
3. **How tightly should policy supersession compose with code change?** A policy that changes shape mid-iteration can break commits-in-flight. ADRs solve this by allowing both old and new to be `accepted` until the system catches up; policies probably need the same grace.
4. **What is the smallest viable kernel for policy?** Could be: id + status + supersedes-chain + reason + an optional pointer to enforcement code. That is exactly the ADR shape today. So perhaps the answer is "extend ADRs" not "introduce a new kind."
5. **Where does AI judgment fit on the rung scale?** Rung 1 (a skill saying "check this") is the natural home. But "LLM-as-linter" is fundamentally a probabilistic check at any rung. Treating LLM-judgment as a deterministic gate is a category error; treating it as a hint is fine.
6. **Should policies ever block retroactively** — a new rule that some existing commits violated? The PoC's standing-check pattern (a warning that surfaces on every push for pre-existing-but-now-illegal state) shows the answer is yes, *as warnings*; promotion to error needs a deliberate decision. The pattern: lint forward, surface backward.
7. **What is the smallest portable unit?** A single policy file? A bundle? A versioned set? Tree-shake-able vs. all-or-nothing?
8. **When does the governance discussion need to happen — before or after we settle on the policy primitive?** Some questions about authority, conflict resolution, and amendment of the policy system itself can be deferred (they only bind once policies actually exist); others (who has standing to ratify) shape the policy primitive itself and may need to settle first. Worth deciding deliberately whether to start the governance exploration before, alongside, or after the policy work.

---

## 14. Lateral options and prior art

This section catalogues prior art and lateral options the design session should walk in with on the table. It is descriptive, not prescriptive — naming what exists and what each thing buys, so the session does not re-invent something the field already worked out.

### 14.1 The pre-LLM Policy-as-Code lineage

Policy-as-Code (PaC) is a real, well-developed field that predates the LLM moment by roughly a decade. Most LLM-era policy thinking is downstream of it.

| Tool | Vintage | Shape | What it does well | What it deliberately does not do |
|---|---|---|---|---|
| **OPA / Rego** | 2016, CNCF Graduated 2021 | Declarative DSL, evaluator-as-service, WASM compile target | K8s admission control, Terraform validation, microservice authz; opa test fixtures; bundle versioning | Imperative side; non-decision-point shapes |
| **Cedar** | AWS, 2023 | Constrained DSL with formal semantics, decidability guarantees, policy-analysis tools | Mechanical reasoning over policy interactions ("does P1 imply P2?"); deny-overrides explicit; small, analyzable | Anything outside authorization-shaped decisions |
| **HashiCorp Sentinel** | 2017 | Imperative DSL, integrated with Terraform/Vault Enterprise | Plan-time evaluation tightly bound to HashiCorp lifecycle | Use outside HashiCorp stack |
| **Kyverno** | 2019 | YAML-as-policy, K8s-native | "If you can't bring yourself to learn Rego" K8s admission | Languages with logic richer than YAML can express |
| **styra DAS / Permit.io / Oso** | 2020+ | Commercial PaC platforms over OPA / their own engines | Auth, RBAC, ABAC for products | Project-engineering policy outside auth |
| **XACML** | OASIS, 2003 | XML-as-policy, the original PaC | Demonstrated the *idea*; widely deployed in enterprise auth | Usable greenfield: pendulum swung to OPA largely because XACML was unusable |

The do's, don'ts, and gotchas this lineage produces are stable enough to take seriously:

- **Policies are data, not procedures.** Write them so they can be queried, indexed, diffed, and tested. The minute they become "code that happens to be in a `.rego` file," they lose half their value.
- **Evaluator-as-library beats evaluator-as-DSL-plus-server.** OPA's biggest design win was the WASM compile target; Cedar's parallel is its formal semantics. The lesson: pick a substrate with deterministic execution or formal analysis support, not just "another DSL we wrote."
- **Per-decision-point policies, not per-rule.** A policy answers a *question* ("can principal X do action Y on resource Z?"). The question-shape constrains the policy to be testable in isolation.
- **Conflict resolution is the killer.** Two policies that both apply to the same decision — which wins? OPA's default: deny-overrides. Cedar: explicit precedence rules, formally checked. Most home-rolled systems get this wrong.
- **Versioning and revocation are first-class.** Policy-in-flight (a workflow that started under v3, finishes under v4) is the most common bug.
- **Test fixtures matter as much as the language.** OPA's `opa test` and Cedar's policy-analysis tooling are what make the systems usable.

What this lineage does *not* address: any of the §3 territory that isn't access-control-shaped. OPA can't enforce *"every public function has a docstring"* without elaborate scaffolding. Cedar deliberately doesn't try. PaC was built for authorization questions; engineering-principle policies are a different shape.

### 14.2 The LLM-era literature, such as it is

There is no mature literature on policy frameworks specifically for LLM-assisted development. The adjacent bodies of work that touch on it:

- **Constitutional AI** (Bai et al., Anthropic, December 2022, arXiv:2212.08073) — *training* an LLM with a written constitution as the source of behavioral norms. Different problem (model alignment, not project policy), but the framing of "a written set of principles the model is bound by" is what later work draws from.
- **OpenAI Model Spec** (open-sourced February 12, 2025) — markdown document of intended model behaviors, versioned in git, used internally as ground truth for training and publicly as the spec. The closest published example of "markdown-as-policy-source" with versioning, public PR review, and behavior-test linkage. A vendor's policy framework about its model, not a framework for consumers' projects, but instructive.
- **ThoughtWorks "harness engineering"** (Vol 34, April 2026) — names *feedforward controls* (specs, skills, AGENTS.md) plus *feedback controls* (deterministic gates, mutation testing, type checkers, fuzzing). Closest mainstream-radar treatment of what a policy framework would be; taxonomy not design.
- **MCP** (Anthropic, November 2024) as the integration layer for what could become a policy-evaluation surface. Several MCP servers are emerging that look policy-shaped (security scanners, license checkers, conventional-commits validators), but none are *policy frameworks* per se — they are policy *evaluators* exposed through MCP.
- **"Constitution" features in AI tooling** — Spec Kit's *constitution*, Kiro's *steering*, Claude.md, AGENTS.md, .cursorrules. None are policy frameworks in the PaC sense; they are advisory text the agent is asked to follow, with no evaluator and no determinism guarantee. ThoughtWorks Vol 33 placed *curated shared instructions for software teams* in Adopt without claiming any of this is mechanical.
- **Empirical evidence consumers want it.** The MSR 2026 *Behind Agentic Pull Requests* study found 58% of human intervention effort on agent PRs is convention-enforcement. That is empirical demand for policy frameworks; the supply is missing.

The honest picture: **there is no published, mature LLM-era policy framework**. There are vendor-specific shapes, PaC tools that don't speak the LLM idiom, and emerging integration layers (MCP) that could carry one. The gap is exactly where this exploration sits.

### 14.3 Policy-as-Code, Code-as-Policy, Policy-as-Specification

Three positions in the literature, not synonyms:

- **Policy-as-Code (PaC).** Policy *is* code in a dedicated language (Rego, Cedar, Sentinel) or structured data (Kyverno YAML, OPA bundles). Defining commitment: *separation of policy from application*. Trade-off: testability, diffability, conflict-analysis tools, paid for with a separate substrate.
- **Code-as-Policy.** Policy *is* application code — functions in the application's own language that take input and return findings. The exploration's §7 shape C is exactly this. Defining commitment: *the policy lives where the system lives*. Trade-off: full expressiveness, opaque to non-engineers, policy-policy interaction (conflict, supersession) handled in app code.
- **Policy-as-Specification.** Policy is a *specification* in a formal or semi-formal language (TLA+, Alloy, Lean, or structured English like RFC 2119 / EARS) that humans author and a runner verifies. Closer to formal methods than to PaC. Brooker's "free-form prose → RFC 2119 / EARS → Lean / TLA+" formality ladder is exactly this axis.

For aiwf the salient design question is *which of these the policy entity points at*. The §7 recommendation (shape A as the index, shapes B/C as implementations) maps cleanly: the policy entity is **policy-as-data** (markdown body + frontmatter), pointing at one or more enforcement implementations that may be **policy-as-code** (a CUE file, a Rego module), **code-as-policy** (a Go function), or **policy-as-specification** (a TLA+ spec, an EARS-form acceptance test).

Same pattern as contracts already use today: a contract entity (data) points at validators (code or schema). The pattern generalizes cleanly to policies.

### 14.4 CUE specifically — how far it gets us

CUE (Marcel van Lohuizen, Google, successor to BCL/GCL) sits at an unusual point: more expressive than JSON Schema, less heavyweight than OPA, declarative throughout, and explicitly designed for *unifying configuration and constraint*. Worth a careful read for the framework's purposes.

What CUE buys cleanly:

- **Schema validation** identical to JSON Schema in expressive power but with lattice-based semantics that compose. Two CUE files unify; conflicts are detected mechanically. Killer feature for *layered* policies.
- **Constraint expression** beyond schema: regexes, value ranges, cross-field invariants (`#Foo: { a: int, b: int, a < b }`), mutual exclusion, presence-implies-presence. Most of the §3 territory that's about *shape of data* (frontmatter rules, naming patterns, citation invariants, dependency rules) is one CUE file each.
- **Validation as a step**, not a service. `cue vet` runs against any input; CI gates it; pre-commit hooks gate it. No evaluator-as-process.
- **A genuine type system** that catches errors before runtime.
- **Unification as composition.** Multiple consumers' policies merge by unification; conflicts surface as findings. Hand-rolled YAML policies don't compose; CUE does.

What CUE does *not* buy:

- **Anything imperative.** *"If the milestone is in scope X, the validator must run command Y"* is hard to express in CUE. CUE is great at "what must be true"; less good at "what must happen."
- **Interaction with foreign tools.** A policy whose enforcement is *"run `gofumpt --check` and pass"* doesn't live in CUE; CUE checks the policy *exists*; the runner is somewhere else.
- **Time and lifecycle.** Waiver expiration, supersession events, policy-applied audit records — these are state transitions, not constraints. CUE models the *snapshot*; the *history* lives elsewhere (git log, audit machinery).
- **Performance contracts of the dynamic kind.** *"p95 latency under 200ms"* is enforced by a benchmark runner. CUE can record the policy *and the threshold*; the runner is third-party.

A pragmatic read: **CUE could carry the static-shape part of the §3 territory cleanly** — engineering principles (the invariant ones), naming/documentation, citation/dependency, security postures (the static ones), and the framework-internal trailer / capability rules. Roughly half to two-thirds of the territory by honest count.

What CUE leaves uncovered: dynamic enforcement (performance, security scanners, mutation testing, runtime checks), lifecycle (waivers, supersession, retire), and the imperative side of process (gates that fire on state transitions). Those need other surfaces.

### 14.5 What the formal-methods cohort actually does

Formal methods is genuinely different from what most "policy frameworks" do. The cohort's core moves:

- **Specify behavior, not structure.** TLA+ describes a system's allowed *behaviors* (sequences of state transitions) and asks whether the implementation satisfies them. Refinement-based proofs check whether implementation behaviors are a subset of specified behaviors. The unit is *what could happen*, not *what must be true now*.
- **Model checking before proof.** Most formal-methods practice today is *model checking* — exhaustive exploration of small state spaces — not theorem proving. AWS uses TLA+ this way for distributed-systems work; Lamport, Newcombe et al. have published on this through the 2010s and 2020s. Formal methods scale when the model is small *enough*.
- **Refinement is the killer concept.** A high-level spec says *"this is a key-value store with linearizable consistency."* A lower-level spec says *"here's how we shard it."* Refinement is the proof that the lower-level spec produces only behaviors the higher-level spec allows. *This is what spec-as-source SDD wishes it had and doesn't.*
- **Counterexample-driven debugging.** When the model checker finds a violation, it produces a *counterexample trace* — a specific sequence of events that breaks the invariant. The trace is the bug report. Drastically better debugging surface than "tests failed somewhere."

What we could borrow without becoming formal-methods practitioners:

- **Invariants as policy expressions.** A subset of §3 is invariant-shaped: *"no entity-touching commit without trailers,"* *"the citation graph is acyclic,"* *"every promoted milestone has all acceptance criteria ratified."* These are state invariants; expressing them in checkable form (CUE, Go validators, EARS prose backed by a runner) is the formal-methods move. Aiwf already does this for some; generalizing the surface is small work.
- **Counterexample-style finding output.** Instead of *"validation failed,"* surface *"validation failed: milestone M-12 was promoted to `done` while AC-3 was still `open` (commit `abc123`)."* The PoC's existing finding format is partway here; making it counterexample-shaped where possible borrows the formal-methods discipline cheaply.
- **Refinement between layers.** A high-level policy says *"engineering principles must be enforceable."* A lower-level policy points at the specific enforcer. The relationship *is* the refinement. The framework can record this declaratively without a proof; the consistency check ("every high-level policy has a refining enforcer") is mechanical.
- **Small models over big ones.** Resist a single global model. Each policy is its own small thing; conflicts surface as findings. Cedar, OPA, and CUE all converge on this.

What we should *not* try to borrow: actual proofs (Lean, Coq), the full TLA+ surface (most of §3 isn't behavior-shaped), the refinement-mapping tooling (overkill). The principle of *invariants + counterexamples + refinement-as-record* is what generalizes; the heavyweight formal apparatus does not.

### 14.6 Lateral options the §7 forms list did not enumerate

A few options the design session should hold open:

**Option α — CUE as the body language for static-shape policies, with code-as-policy escape hatch.**
- Policy entity is markdown frontmatter + body, where the body is CUE for static-shape policies and Go for the rest.
- `aiwf policy verify` runs `cue vet` for CUE-bodied policies, `go test` for code-bodied ones.
- Composition (multiple policies on one subject) is CUE unification for the static parts; explicit conflict-resolution declared for the imperative parts.
- Supersession, lifecycle, provenance live in the existing trailer machinery.
- Buys most of §3 cheaply, no new substrate.

**Option β — RFC 2119 / EARS form for the human-facing layer, runner-pointer for enforcement.**
- Policies written in structured English (RFC 2119: SHALL, SHOULD, MAY; or EARS: Event-driven, Ubiquitous, Unwanted-behavior, State-driven, Optional). Aerospace and automotive use this; *human-readable* and *machine-parseable enough* for tooling.
- Each requirement points at a runner: a CUE file, a Go validator, a CI command, a benchmark, a mutation-testing target.
- Framework's lane: the requirement entity, its provenance, its lifecycle, the runner pointer. Enforcement is delegated.
- Low-friction onramp for the cohort already using EARS / RFC 2119 (regulated industries, high-assurance).

**Option γ — Cedar-shaped capability policies + CUE-shaped invariants.**
- Capability gates (the framework-internal sub-class — *who may invoke `--force`*, *delete in agent scopes*) are a small Cedar policy bundle. Cedar's formal semantics let us *prove* properties like "no agent can self-grant a waiver."
- Engineering invariants (project-engineering sub-class) are CUE files referenced by policy entities.
- Two languages, but each used for what it does well; framework binds them at the policy-entity layer.
- Concrete instance of the §3 sub-class question expressed as two-bodies.

**Option δ — Property-based testing as the runner for as-much-as-possible.**
- Hypothesis (Python), QuickCheck-style (Go's `gopter`, Haskell's QuickCheck), proptest. Express engineering principles as properties: *"for any commit C, if C touches an entity, C carries trailers."* Generate inputs; check the property; counterexamples become findings.
- Borrows formal-methods counterexample discipline at low cost.
- Already adopted in some of §3 (fuzz testing of parsers); generalizes upward.
- Not a substitute for static checks (CUE is faster and more precise where it applies); a clean fit for the dynamic-invariant territory.

**Option ε — Treat policy as a specialization of contract.**
- Contracts already have body + validator pointer + lifecycle. *Add a discriminator field on contract* (`subject: interface | engineering-principle | security-posture | ...`) and let contract carry the policy load.
- Worth holding open as the *minimum-kernel-change* option. Contracts-as-bilateral-verified is closer to the engineering-policy reading than ADR-as-architecture is, so the audience pollution may be smaller.
- Risk: same as ADR widening, smaller. Has to be tested concretely against §3 examples.

### 14.7 What this might mean for the design session

A few moves the design session could make that §13's open questions did not fully spell out:

1. **Pick a body-language first; the entity surface follows.** If CUE carries two-thirds of the territory, the entity surface is "policy with a CUE body OR a runner-pointer" — a much smaller schema than "policy with a free-form prose body and an unspecified enforcement layer."
2. **Adopt RFC 2119 / EARS for the prose layer immediately.** Costs nothing, mechanically parseable, what every regulated industry uses. The framework can adopt it as the prose-form even before the rest of the form-question settles.
3. **Borrow Cedar's policy-conflict-analysis posture without adopting Cedar itself.** Two policies that govern the same subject must declare their precedence, or the framework surfaces a finding. One paragraph rule, not a feature.
4. **Borrow OPA's policy-bundle versioning.** A policy applies in versions; in-flight work is pinned to the version under which it started. Waivers granted under v3 must keep working when v4 lands, until v4's grace period expires.
5. **Borrow formal-methods counterexample reporting.** Every finding includes the specific trace that produced it. The reader doesn't reverse-engineer the failure.
6. **Don't pretend CUE is "just JSON Schema."** Its lattice semantics are the unification engine the framework needs for layered policies. If adopted, adopt it for that property, not because it's familiar.
7. **Hold Option ε open.** If §3's territory walks out as mostly contract-shaped, widening contract may be cheaper than a seventh kind. The thesis's *should* (`13`) does not preclude this — it says *policy earns kernel standing*; whether that standing is a new kind or a discriminator on contract is part of form. Both honor ADR purity and audience separation.

### 14.8 Reading list, in order of payoff

If grounding the design session in real prior art:

- **OPA documentation, especially the *test* section.** ~2 hours. Lessons about per-decision-point policies and conflict resolution are immediately applicable.
- **Cedar's formal-semantics paper** (Cutler et al., USENIX Security 2024). ~3 hours. The decidability discussion is what makes Cedar distinct from OPA; the design choices read as a reaction to OPA's expressiveness costs.
- **CUE's introduction and the *Lattice* explanation** in the language spec. ~2 hours. The lattice concept is what most CUE adopters miss; once it clicks, the framework's policy composition story writes itself.
- **The OpenAI Model Spec repository** (`github.com/openai/model_spec`). ~1 hour. Closest published example of "markdown-as-policy-source" with versioning, public PR review, behavior-test linkage.
- **EARS guide** (Mavin et al., *Easy Approach to Requirements Syntax*, originally Rolls-Royce 2009). ~1 hour. The prose-form move that costs nothing and pays a lot.
- **Lamport's TLA+ video course or *Specifying Systems*.** Heavier — not necessary unless taking refinement seriously.

### 14.9 What this section deliberately leaves to §13 and the design session

This catalogue does not pick winners. It is the menu the design session walks in with, so the conversation starts at "which combination, and why" rather than "what even exists." Specifically:

- Whether the policy entity is one kind or two (the §3 sub-class question) is unaffected by this catalogue; both options (one body-language for both sub-classes, or two with different bodies as in Option γ) are compatible with everything here.
- Whether the framework adopts CUE, Rego, Cedar, EARS, or some combination is form work for the design session.
- Whether existing aiwf machinery (contracts, the meta-policies package) absorbs into the new shape or stays alongside is downstream of the form decision.

The catalogue's point is: **the design session does not have to invent any of this from scratch**. The PaC field has worked out conflict resolution, versioning, test-fixture discipline, and decidability trade-offs. CUE has worked out lattice-based composition. Formal methods has worked out invariants and counterexamples. EARS has worked out the prose-form layer. The work in front of the framework is *selection and composition*, not invention.

---

## A frame to hold while iterating

Treat policy as **the umbrella that already-existing aiwf entities partially cover** — ADRs, decisions, contracts, the meta-policies package — and ask whether the umbrella earns its keep. The strongest argument for naming the umbrella is **provenance and lifecycle uniformity**: one supersedes-chain, one waiver mechanism, one principal × agent × scope shape across all of these. The strongest argument *against* is YAGNI: if no real policy exists today that does not fit one of the existing kinds, the umbrella is premature.

The strongest argument for cross-project portability is **rung-1 is already easy and worth doing now**: a curated CLAUDE.md fragment library copied into every repo would deliver a lot of the value at almost zero cost. Anything beyond that needs to clear a real bar of "what is this enforcing that I could not enforce repo-locally."

The strongest single test for whether a candidate "policy framework" earns its keep is the SDD lesson, ported: **does it raise the level of abstraction at which iteration happens, or does it freeze the abstraction in place?** A policy framework that lets policies evolve cheaply, with full provenance, passes. A framework that ossifies them fails — same way SDD-as-source fails when it cannot iterate.

---

*This document is intentionally exploratory. The next step, when the time is right, is a targeted session that picks a position on each of the open questions above and produces a defended `docs/design/policy-model.md`. That session should also revisit whether the umbrella concept survives contact with concrete cases — it may turn out the existing four kinds are enough, and the work is to sharpen them rather than introduce a new one.*
