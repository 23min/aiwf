> **Status:** reference
> **Audience:** anyone reading this series for the first time, or returning after time away.
> **Tags:** #aiwf #research #thesis

---

## Abstract

This series is the record of a project finding its problem after committing to a solution. The repository started with a confidently designed architecture — an event-sourced kernel, hash-verified projections, monotonic IDs, RFC 8785 canonicalization — captured in [`docs/architecture.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/architecture.md). What it lacked was an articulated problem the architecture was solving. The research arc that follows is the work of discovering the actual problem and walking back the architecture in light of what discovery revealed. By the end, the framework is substantially smaller than the original ambition, and the position is honest about what it claims and what it doesn't. This document is the map: it explains the trajectory, gives the reading order, summarizes the kernel of needs the framework exists to serve, and names what to expect from each entry.

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
- **[`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/)** — collapses the synthesis into the smallest framework that delivers the value: six entity kinds, stable IDs, a pre-push validator, structured-commit history, in roughly four sessions.
- **[`07-state-not-workflow`](https://proliminal.net/theses/state-not-workflow/)** — argues that workflow is one render of state, not the substrate; the framework is a state model with optional workflow renders, not a workflow engine.
- **[`08-the-pr-bottleneck`](https://proliminal.net/theses/the-pr-bottleneck/)** — the PR is where the throughput limit becomes visible; continuous ratification by humans at state transitions replaces batched post-hoc review; HITL strengthens rather than dissolves under LLM amplification.
- **[`09-orchestrators-and-project-managers`](https://proliminal.net/theses/orchestrators-and-project-managers/)** — analyzes the relation between orchestration as an emerging craft and traditional PM work; some PMs become orchestrators, some don't, and orchestrators come from many disciplines.
- **[`10-spec-based-as-waterfall`](https://proliminal.net/theses/spec-based-as-waterfall/)** — argues that spec-based development tools (Spec Kit, Kiro, Tessl) encode the same artifact-handoff dependency graph as waterfall, and that continuous ratification is structurally different because it ratifies state transitions on a single artifact rather than handoffs between artifacts.

Plus one reference document and one satellite note outside the linear chain:

- **[`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)** — the eight needs and cross-cutting properties any framework proposal must serve. The rubric the series uses to evaluate its own claims.
- **[more is much more](https://proliminal.net/notes/more-is-much-more/)** — short note on the LLM-era inversion of Mies van der Rohe's "less is more" credo: with abundant production, the right human response is fewer artifacts to ratify, not more output to consume.

## 5. How to read this

Three reasonable paths, depending on what you want.

**If you're new and want to follow the discovery,** read in order from `00` through `10`. Expect to disagree with `00`'s starting assumptions; the series disagrees with them too, which is partly the point. By `06` the position is settled; `07`–`10` extend it into adjacent questions.

**If you're oriented and want the current position,** read the [working paper](https://proliminal.net/theses/working-paper/) first, then [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md) (or the §2 summary above), then [`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/). The series itself becomes a citation network rather than required reading.

**If you came for a specific question** (Why not Automerge? Why not Linear? Why not give up on git?), use the trajectory map in §4 to find the document that answered it. Most questions land in `02`, `04`, or `05`.

## 6. What this series is not

A few things this series does not try to be, named to prevent disappointment:

- **Not a finished design.** The PoC is small and unproven. Real use is what would tell us if the position holds. Nothing here has been pressure-tested at scale.
- **Not normative for other projects.** The framework targets a specific shape of work — solo through small-team, weeks-to-months horizons, AI-amplified, willing to live in markdown and git. Teams outside that shape may find different positions correct.
- **Not a survey of the field.** Where the series cites other work (Automerge, Pijul, Bayou, MetaGPT, Spec Kit, Kiro, ACE), it does so to position this framework's bets, not to comprehensively review the literature.
- **Not stable yet.** Docs marked `thesis-draft` are explicitly works-in-progress. Docs marked `defended-position` have been pushed back on and held, but may still be revised as the PoC encounters real friction.

The series is, instead, **a record of the work of finding what to build**. It's slower than a finished design and more honest than a confident claim. That tradeoff is the point.

---

## In this series

- Next: [00 — Fighting git](https://proliminal.net/theses/fighting-git/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
- Synthesis: [working paper](https://proliminal.net/theses/working-paper/)
- Starting state (superseded): [docs/architecture.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/architecture.md)
