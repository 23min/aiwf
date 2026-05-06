> **Status:** reference
> **Audience:** anyone reading this series for the first time, or returning after time away.
> **Tags:** #aiwf #research #thesis

---

## Abstract

This series is the record of a project finding its problem after committing to a solution. The repository started with a confidently designed architecture — an event-sourced kernel, hash-verified projections, monotonic IDs, RFC 8785 canonicalization — captured in [`docs/architecture.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/architecture.md). What it lacked was an articulated problem the architecture was solving. The research arc that follows is the work of discovering the actual problem and walking back the architecture in light of what discovery revealed. By the end, the framework is substantially smaller than the original ambition, and the position is honest about what it claims and what it doesn't. The Background section below sketches how the project arrived at that starting state; this document then explains the trajectory, gives the reading order, summarizes the kernel of needs the framework exists to serve, and names what to expect from each entry.

---

## Background

This series is not the start of the work. During a recent customer engagement I implemented an internal observability platform for a complex distributed system, and decided at the outset that I would not write any code by hand — every line would be produced by an AI assistant under my direction. Over that engagement I formalized the habits I developed into what I came to call my first AI framework: an assistant-agnostic set of markdown files and shell scripts. When the assistant made mistakes, I fixed the framework — what would now be called the harness.

When the contract ended I set out to redesign the framework from first principles. That redesign was v2 — and, having a clear sense of which failure modes I most wanted to eliminate (the assistant cheating past steps, forgetting prior decisions, leaving messy merges through course corrections, ignoring decisions that *were* on record because it hadn't found them), I let the design grow. The v2 architecture absorbed everything I thought might help: a relational schema for the planning graph, integrity constraints, event sourcing, a hash-verified graph projection, state machines. It is recorded as [`docs/architecture.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/architecture.md). No code was ever written against it.

It went too fast. Running the system in my head, the design ran into problems immediately. What followed is the series recorded here — a forced re-discovery of the problem I was actually trying to solve, and especially of the problems I was *not* going to solve. By around the sixth entry I was ready to draft a scoped-down architecture in a KISS / YAGNI / PoC spirit: no event log, no graph projection, no CRDTs. The audience became explicit too — v1 had been an internal tool; the new framework would be a public repository, with the constraints that implies. What was abandoned remains "v2"; the proof of concept that emerged from this research is v3, distilled in [`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/) and [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md).

The series, then, is research that came out of a misdirected architecture which failed on arrival, and which led to a much less ambitious but arguably more capable framework than the one originally imagined. The problem became better defined; the criteria and requirements were formalized in the kernel.

---

## 1. What you'll find here

If you came to this series cold and started at [`00-fighting-git`](https://proliminal.net/theses/fighting-git/), you'd reasonably wonder why a document opens by attacking a totally-ordered hash-chained event log without first explaining who proposed one. The answer is that one of us did, in the original architecture, before the problem was clear. The series begins *in medias res* because that is where the work began — already mid-design, already committed to mechanism, already needing to walk back.

Two things are useful to know before reading further:

- **`docs/architecture.md` is the starting state**, not the current position. It describes the framework as it was first imagined: ambitious, clever, structurally elaborate. The README marks it as superseded; the research arc is the record of why.
- **The current position is in the [working paper](https://proliminal.net/theses/working-paper/) and [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)**, with the [PoC build plan](https://proliminal.net/theses/poc-build-plan/) as the actionable distillation. If you only want the destination, read those three. The numbered docs are the journey. A condensed summary of the kernel is in §2 below.

The trajectory the series records, in one paragraph: the original architecture was a sophisticated solution layered onto git, and the more it was examined, the more it fought the substrate. As we examined *why* it fought, the actual problem the framework was meant to solve started to come into focus — referential stability, the forgetful AI, the LLM-as-non-deterministic-skill-invoker, the soft/hard tension between exploratory planning and durable record. Each of those discoveries shrunk the design surface. By the end, the framework's residual job is markdown files plus a small validating engine plus a few verbs — at the scale most teams operate at, that's enough.

## 2. The kernel — what the framework needs to do, in summary

The series uses [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md) as the rubric against which every proposal is evaluated. The full text is short but worth a glance; this section is the further-condensed version, useful for orientation.

The framework exists to do **eight things**, and only these:

1. **Record planning state** — what's planned, what's in flight, what's done, what's blocked, what was decided. Persistent and accessible to humans and AI from inside the repo.
2. **Express relationships** — milestones belong to epics, milestones depend on milestones, decisions ratify scope, gaps motivate new work, ADRs constrain code. Typed and machine-checkable.
3. **Support evolution** — insert a milestone between two others, rewrite a milestone when a dependency changes, supersede a decision, reorder priorities. Plans are clay until they are committed.
4. **Keep history honest** — when a thing changed, who changed it, why. Provenance is queryable and auditable.
5. **Validate consistency** — references resolve, status transitions are legal, terminal states are not silently undone, cycles do not form, ids do not collide. Mechanical, fast, deterministic, AI-judgment-free.
6. **Generate human-readable views** — ROADMAP, dependency diagrams, status reports, audit findings. Derived on demand from canonical state; courtesies, not authority.
7. **Coordinate AI behavior** — skills, rules, and contracts shape how AI assistants act on the project. Versioned with the work so behavior is reproducible per checkout.
8. **Survive parallel work** — multiple humans, multiple AI assistants, multiple branches, possibly weeks of divergent work. Merging is well-defined for structural state; semantic conflicts surface as findings.

Cutting across all eight, **a handful of quality bars** any solution to those needs must meet:

- **Enforcement does not depend on the LLM choosing to enforce.** Skills are advisory; CI gates and validators are authoritative.
- **Referential stability is real.** An id like `E-19`, once allocated, always means the same entity — even after rename, move, or removal. Tombstones, not silent deletions.
- **Honest about meaning.** The framework guarantees referential and structural stability. It does not pretend to guarantee that the *meaning* of an entity stays fixed — that's a property of human and AI understanding, not of data structures.
- **Engine is invocable without an AI.** Every verb takes flags, reads stable input formats, emits a JSON envelope, exits with documented codes. Humans, CI, and other tools drive it directly.
- **Soft in raw studio; AI-assisted strictness pre-push; mechanical strictness at the PR gate; sealed at main.** Iteration is unconstrained on a personal branch in early shaping; framework verbs and validators tighten the loop pre-push; CI on the PR is the mechanical gate that does not depend on the LLM; main is the sealed artifact.
- **Modular and opt-in.** A small kernel everyone can use, plus modules each project enables based on its shape on the team-size, horizon, brownfield-depth, and regulation axes.
- **Governance and provenance are first-class UX, not side effects.** The renderers and queryable surfaces for who-did-what-and-why and what-can-change-here are core, not optional.
- **Layered location-of-truth.** Engine binary lives external (machine-installed). Per-project policy and planning state live in the consumer repo. Materialized skill adapters live in the consumer repo but are gitignored. Each layer lives where its constraints are best served.

If a proposed change does not serve one of those eight needs, it is out of scope. If it strains one of those quality bars, the strain has to be addressed explicitly. The kernel is the slowest-changing artifact in the framework's design, by deliberate discipline.

## 3. The discipline this series tries to keep

A few rules govern how documents in this series are written and revised:

- **Each document is a step.** It builds on the prior ones, sometimes pushes back on them, sometimes obsoletes parts of them. Read in order if you're new; jump in via the working paper if you're oriented.
- **Problems and solutions co-evolve.** No document is allowed to assume the problem has already been articulated. If a doc proposes a solution, it has to say what need on `KERNEL.md`'s list it serves. If it proposes a need, it has to say why.
- **Walking back is fine.** Several documents explicitly retract claims earlier ones made. This is a feature of doing the work in public, not a bug. The trajectory *is* the argument.
- **Slogans are flagged as such.** Lines like "there is no workflow" or "kill PRs" appear in the series as teaching devices, not framework taglines. Where they overreach, the body of the document says so plainly.

## 4. The trajectory

A short map of the series. Read top to bottom on first encounter; cross-reference freely on return.

- **[`00-fighting-git`](https://proliminal.net/theses/fighting-git/)** — establishes that a totally-ordered hash-chained event log layered onto git fights git's branching model and cannot be made to survive merges by construction. Names the substrate problem the rest of the series will respond to.
- **[`01-git-native-planning`](https://proliminal.net/theses/git-native-planning/)** — proposes the substrate-respecting alternative: markdown files as canonical state, git as the time machine, framework as a small validating engine.
- **[`02-do-we-need-this`](https://proliminal.net/theses/do-we-need-this/)** — audits whether a custom framework is needed at all, given that ADRs plus a context document plus a habit solves 80% of the forgetful-AI problem with two weeks of work.
- **[`03-discipline-where-the-llm-cant-skip-it`](https://proliminal.net/theses/discipline-where-the-llm-cant-skip-it/)** — argues that the framework's correctness must rest on enforcement chokepoints (CI, pre-push hooks) the LLM cannot skip; skills are advisory ergonomics, not the discipline layer.
- **[`04-governance-provenance-and-the-pre-pr-tier`](https://proliminal.net/theses/governance-provenance-and-the-pre-pr-tier/)** — extends the synthesis to governance/provenance UX, modular opt-in by project shape, bounded CRDT primitives, and pre-PR HITL placement.
- **[`05-where-state-lives`](https://proliminal.net/theses/where-state-lives/)** — refines the location-of-truth question into six layers, with engine binary external, planning state in repo, materialized adapters gitignored and stable across branch switches.
- **[`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/)** — collapses the synthesis into the smallest framework that delivers the value: six entity kinds, stable IDs, a pre-push validator, structured-commit history, in a focused week or two of work.
- **[`07-state-not-workflow`](https://proliminal.net/theses/state-not-workflow/)** — argues that workflow is one render of state, not the substrate; the framework is a state model with optional workflow renders, not a workflow engine.
- **[`08-the-pr-bottleneck`](https://proliminal.net/theses/the-pr-bottleneck/)** — the PR is where the throughput limit becomes visible; continuous ratification by humans at state transitions replaces batched post-hoc review; HITL strengthens rather than dissolves under LLM amplification.
- **[`09-orchestrators-and-project-managers`](https://proliminal.net/theses/orchestrators-and-project-managers/)** — analyzes the relation between orchestration as an emerging craft and traditional PM work; some PMs become orchestrators, some don't, and orchestrators come from many disciplines.
- **[`10-spec-based-as-waterfall`](https://proliminal.net/theses/spec-based-as-waterfall/)** — argues that spec-based development tools (Spec Kit, Kiro, Tessl) encode the same artifact-handoff dependency graph as waterfall, and that continuous ratification is structurally different because it ratifies state transitions on a single artifact rather than handoffs between artifacts.
- **[`11-should-the-framework-model-the-code`](https://proliminal.net/theses/should-the-framework-model-the-code/)** — applies the audit voice (`02`), layer model (`05`), and state-vs-render boundary (`07`) to the question of whether the framework should absorb code-graph functionality (graphify, GitNexus). Concludes the framework's lane is decisions about code, not code structure; one narrow consistency check (symbol-level reference resolution for fields like `live_source`) survives.
- **[`12-operating-model-agnostic`](https://proliminal.net/theses/operating-model-agnostic/)** — closes the loop opened by the three field surveys: SDD is one cell in a roughly thirteen-cell landscape of AI-assisted operating models, and the framework's bet does not select for any of them. aiwf is the durable structural-decision layer underneath whatever operating model a team runs; composes with most cells through stable surfaces; refuses two cells where its bet structurally does not fit (workflow-tool-driven as primary substrate, agentic-swarm without ratification chokepoints).
- **[`13-policies-as-primitive`](https://proliminal.net/theses/policies-as-primitive/)** — applies the audit voice to the policies-design-space exploration. Names the territory concretely first (engineering principles, security postures, performance budgets, naming and documentation rules, citation and provenance requirements, capability gates, governance commitments — the breadth of the exploration's §3 categories and §9 stretch table) and surfaces a sub-class distinction that matters for the form-question: *project-engineering policies* (what must be true of the consumer's software) vs. *framework-internal policies* (how aiwf itself operates). On audience and reading discipline — not structural shape — policy is distinct from ADR (architecture), decision (scope), and contract (interface). ADR must stay pure; policy earns a place as a first-class kind. Form (one kind or two for the sub-classes, body shape, lifecycle states, waiver shape, enforcement-pointer shape, portability) is deliberately deferred to the targeted design session the exploration converges on.

Plus one reference document and several satellites outside the linear chain:

- **[`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)** — the eight needs and cross-cutting properties any framework proposal must serve. The rubric the series uses to evaluate its own claims.
- **[more is much more](https://proliminal.net/notes/more-is-much-more/)** — short note on the LLM-era inversion of Mies van der Rohe's "less is more" credo: with abundant production, the right human response is fewer artifacts to ratify, not more output to consume.
- **[`surveys/understanding-spec-driven-development`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/surveys/understanding-spec-driven-development.md)** — a field survey of Spec-Driven Development: history, taxonomy (five rungs of the ladder), tooling landscape, and the published critiques and defenses. Grounds the structural argument in [`10`](https://proliminal.net/theses/spec-based-as-waterfall/) with primary-source citation; written as an explainer rather than a thesis.
- **[`surveys/workflow-work-division-and-who-implements`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/surveys/workflow-work-division-and-who-implements.md)** — companion survey on what each SDD interpretation implies for lifecycle, work division, and the autonomy spectrum (assistive → swarm). Pulls in the empirical record on AI-assisted productivity (METR, Anthropic, DORA, MSR 2026) that grounds claims in [`07`](https://proliminal.net/theses/state-not-workflow/), [`08`](https://proliminal.net/theses/the-pr-bottleneck/), and [`09`](https://proliminal.net/theses/orchestrators-and-project-managers/).

## 5. How to read this

Three reasonable paths, depending on what you want.

**If you're new and want to follow the discovery,** read in order from `00` through `13`. Expect to disagree with `00`'s starting assumptions; the series disagrees with them too, which is partly the point. By `06` the position is settled; `07`–`13` extend it into adjacent questions.

**If you're oriented and want the current position,** read the [working paper](https://proliminal.net/theses/working-paper/) first, then [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md) (or the §2 summary above), then [`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/). The series itself becomes a citation network rather than required reading.

**If you came for a specific question** (Why not Automerge? Why not Linear? Why not give up on git?), use the trajectory map in §4 to find the document that answered it. Most questions land in `02`, `04`, or `05`.

## 6. What this series is not

A few things this series does not try to be, named to prevent disappointment:

- **Not a finished design.** The PoC is small and unproven. Real use is what would tell us if the position holds. Nothing here has been pressure-tested at scale.
- **Not normative for other projects.** The framework targets a specific shape of work — solo through small-team, weeks-to-months horizons, AI-amplified, willing to live in markdown and git. The focus on solo and small-team work is a deliberate scoping of the experiment, not a claim about where the interesting problems live; teams outside that shape may find different positions correct. Adjacent readers — technical project managers and tech leads, spec-driven-development practitioners, tool-builders, engineers earlier in their AI adoption — are welcome regardless.
- **Not a sales pitch.** The author works through these problems in solo and small-team settings because that is where personal experimentation is tractable. The PoC may ship, may pivot, or may end up endorsing a different framework that solves these problems better. The contribution is the thinking, done in public, with the PoC as a concrete attempt to act on it.
- **Not a survey of the field.** Where the series cites other work (Automerge, Pijul, Bayou, MetaGPT, Spec Kit, Kiro, ACE), it does so to position this framework's bets, not to comprehensively review the literature.
- **Not stable yet.** Docs marked `thesis-draft` are explicitly works-in-progress. Docs marked `defended-position` have been pushed back on and held, but may still be revised as the PoC encounters real friction.

The series is, instead, **a record of the work of finding what to build**. It's slower than a finished design and more honest than a confident claim. That tradeoff is the point.

---

## In this series

- Next: [00 — Fighting git](https://proliminal.net/theses/fighting-git/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
- Synthesis: [working paper](https://proliminal.net/theses/working-paper/)
- Starting state (superseded): [docs/architecture.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/architecture.md)
