# State, not workflow — what software work becomes when LLMs participate in every role

> **Status:** defended-position
> **Hypothesis:** Workflow is one render of state — useful where genuine pipelines exist, misleading where it was scaffolding around throughput limits that LLMs are dissolving; the framework's right shape is a state model with optional workflow renders, not a workflow engine with a state cache.
> **Audience:** the user, after pushing back on a too-clean claim that "work doesn't flow anymore" and asking whether "work" and "flow" are themselves words that bias the design.
> **Premise:** software work in 2026 is dissolving the queue-based, station-bound, throughput-limited shape that workflow described well for several decades; what's left is *state*.
> **Tags:** #thesis #aiwf #workflow #state-model #software-development

---

## Abstract

The earlier docs ([06](https://proliminal.net/theses/poc-build-plan/) and prior) settled the framework's structural shape. This document examines a vocabulary trap that survived the synthesis: the words "work" and "flow" both bias the design toward a factory metaphor that LLM-amplified work is eroding. The doc steel-mans where workflow remains real (CI, regulatory chains, deploy pipelines — physical or legal facts that LLMs don't dissolve), then names the erosion: stages were a function of specialist throughput, and when LLMs collapse production time, queues evaporate and re-entry becomes routine. What's left is state — *what's currently true about the work* — which has answers regardless of process shape. The framework's right framing is a **state model with workflow as an optional render**, inverting the usual tool design that models workflow as primary and state as where-it-is-in-the-workflow. The doc engages with the Matrix line "there is no spoon" as a teaching device but explicitly resists it as a slogan: most workflow shape was scaffolding that dissolves; some pipelines are real and remain. The position has a failure mode (regulated industries, large enterprises, ops, sales) that the framework names honestly. State, not workflow, is the durable shape — but only for the team shapes the framework targets.

---

## 1. The question

Software work has been described as a workflow since long before software. Requirements analysis flows into design; design flows into architecture; architecture flows into implementation; implementation flows into testing; testing flows into deployment. The metaphor — work moves through stations, each station does its irreducible thing, the artifact accumulates value — is older than the assembly line and was inherited into knowledge work by analogy.

LLMs in every role are doing something to this picture. The question this document tries to settle: *what?* And in particular, is "workflow" still the right organizing concept for the framework `aiwf` is being built to support, or is it a vocabulary that smuggles in assumptions that no longer hold?

The user's framing makes the question sharper than the prior research docs put it: are even the words "work" and "flow" traps? The reasonable answer requires admitting that *some* of what looks like workflow erosion is real and *some* is overreach.

---

## 2. Steel-manning workflow

Before arguing against workflow as the dominant frame, name where it remains real and load-bearing.

### 2.1 Genuine pipelines that survive any role compression

A pull request flows through CI, then code review, then merge, then deploy. CI runs *before* merge for actual reasons: tests have to compile, lint has to pass, the integration has to hold. The order is not vestigial; it's a sequence of transformations that have to happen in that sequence because each step's output is the next step's input.

A regulated submission to the FDA flows through specific human reviewers in specific orders because the regulation requires it. No amount of LLM amplification changes the regulation. The same applies to SOX evidence chains, ISO 27001 audit trails, 21 CFR Part 11, GDPR data subject access requests, and every other compliance pipeline where the *order of human attestation* is a legal fact.

A deploy pipeline (build → test → stage → prod) is sequential because each environment is an empirical test of whether the previous environment's claims hold. Skipping a stage means losing the signal that stage produced.

These are not "stages" in the project-management sense. They are *physical or legal pipelines* whose ordering is structural. They will exist in 2030 and 2040 and remain workflow-shaped regardless of what LLMs do.

### 2.2 Industrial-revolution workflows have a real basis

The factory metaphor was not a category error when applied to physical work. A loaf of bread genuinely moves from miller to baker to oven to shelf because each station does irreducibly different things to the dough. The miller cannot do the baker's job without becoming a baker. The throughput of the line is genuinely bounded by the slowest station.

When this metaphor was extended to knowledge work in the late 20th century, it was already strained — the "material" was information, the "transformation" was thinking, and stations were specialist humans whose throughput was bounded by their attention rather than by physical equipment. But the metaphor wasn't *wrong*: specialists really were the bottleneck, handoffs really did exist, and "the artifact moves from the PM's tray to the architect's tray" really did describe what was happening, even when the trays were email inboxes.

So workflow is not an industrial-era illusion that knowledge workers should have abandoned. It was a reasonable model for a long time, because the conditions that produced it (specialist throughput as the bottleneck) were real.

### 2.3 The honest failure mode of "there is no workflow"

If `aiwf` declares "there is no workflow" as a slogan, it overreaches in three ways:

- It denies the genuine pipelines in §2.1.
- It implies that all workflow-shaped tools (Jira, ServiceNow, Camunda) are categorically wrong, which is a claim too strong to defend.
- It suggests the framework is for everyone, which it isn't — regulated industries, large enterprises, and any team whose work is genuinely pipeline-shaped will find a state model insufficient.

The position the framework needs is sharper: **the workflow that dominated knowledge work was a function of human specialist throughput, and LLMs erode that bottleneck, but the genuine pipelines remain.** State is the durable shape; workflow is a render that's accurate where pipelines are real and misleading where they were scaffolding around throughput.

---

## 3. The erosion — what LLMs actually do to stages

Stages in software work emerged because each stage required a different specialist with different tools producing a different artifact, and the time-cost of producing each artifact was significant. The PM took a week to write the PRD. The architect took two weeks to draw the diagrams. The designer took three weeks to mock the screens. The engineer took a month to implement. The tester took a week to find the bugs. Each stage had a queue because the producer was slow relative to the consumer.

LLMs collapse the production time. A solo person with LLMs can produce a passable PRD, a passable architecture, passable mockups, passable code, and passable tests in an afternoon. Not great work — but first-pass artifacts that are good enough to react to, edit, ratify, or reject.

What happens to "stages" when production is fast?

- **Queues thin sharply for the work LLMs handle well.** The PM's tray doesn't fill up the same way; the LLM produces drafts on demand. The architect's tray doesn't fill up the same way. The designer's tray doesn't fill up the same way. Some queues remain (priority decisions, attention, real-world dependencies, customer feedback cycles, decision latency). But the queues that defined stages — work-in-process backed up because a specialist couldn't produce fast enough — thin to the point that they stop being the bottleneck.
- **Specialist throughput stops being the bottleneck.** The bottleneck moves to *judgment*: which of these LLM-produced artifacts is actually good? Does this PRD reflect what users want? Does this architecture have the failure modes the LLM didn't see? Does this mockup match the brand voice? Judgment doesn't queue the same way production queued. Judgment is fast, can happen in parallel, and can re-enter prior decisions when new information arrives.
- **Re-entry becomes routine.** A session that started as "implement this milestone" routinely surfaces a requirements gap. A session that started as "design this screen" routinely changes the data model. Pre-LLM, this was painful because going back meant re-queuing through earlier stages. With LLMs, going back is cheap: the LLM can revise the PRD in minutes, not weeks, so the cost of re-entry approaches zero.
- **Specialization becomes about judgment, not production.** A "designer" in 2026 is not someone who can produce a wireframe (LLMs can produce wireframes). A designer is someone whose *taste* and *judgment* about wireframes is reliable. Same for architects, PMs, engineers. The role doesn't disappear; what defines the role shifts from "can produce X" to "can judge X reliably."

The erosion is specific: it's the *queue-based, throughput-limited, station-bound* version of workflow that's eroding. The pipelines that exist for non-throughput reasons (CI, compliance, deploy) are unaffected because their ordering is structural, not throughput-driven.

So the right claim is more nuanced than the slogan: **most of the workflow we inherited from pre-LLM team structure is the kind of scaffolding that dissolves under LLM amplification; some of it is real pipeline and remains.**

---

## 4. The reframe — state as primary, workflow as render

If the throughput-bound queues thin and re-entry becomes routine, the question "what stage is this in?" stops describing the work as well as it used to. The artifact isn't *in* a stage; it's in a state. The state describes what's currently true about the artifact, regardless of how it got there or who acted on it last.

State is what survives when the flow shape doesn't. The questions that have answers regardless of process shape:

- What is the current scope of this milestone?
- What's been ratified?
- What gaps are open?
- What contracts apply?
- What's blocked?
- What's been decided?
- What was superseded?

These questions don't care whether the team practices waterfall, dual-track agile, trunk-based development, or no methodology at all. They don't care whether the work moves through stations or happens in parallel. They don't care whether the PRD was written first or emerged from prototype iteration. They have answers; the framework's job is to keep those answers coherent.

A workflow-shaped tool answers "where is this?" — which station, which queue, which assignee. A state-shaped tool answers "what is true about this?" — which scope, which dependencies, which ratifications. The first question becomes meaningless when stations dissolve. The second remains meaningful regardless.

### 4.1 Workflow as a render

This does not mean workflow disappears as a *visualization*. A team that wants to see their work as a Kanban board with "Backlog / In Progress / In Review / Done" columns can do so by grouping statuses into columns. A team that wants a Gantt chart can compute one from `depends_on` edges and date frontmatter. A team that wants a CI pipeline can describe it as a sequence of state transitions on entities. All of these are *renders* of the underlying state, computed on demand.

The framework's stance: **state is canonical; workflow is one possible render among many.** Different teams render the state differently. None of those renders is the framework. None is privileged. The render is a courtesy for the team that finds it useful; the state is what's true.

This reverses the usual tool design. Jira, Linear, Asana, GitHub Projects all model workflow as primary (the board, the column, the assignee, the transition rule) and state as a property of where-it-is-in-the-workflow. The framework models state as primary and lets workflow be a derived view. The reversal is the design choice that survives the LLM-era erosion of queues.

### 4.2 What this preserves and what it discards

Preserved:
- Status enums per kind (`draft`, `in_progress`, `done`, `cancelled`, etc.) — describe artifact state, not pipeline position.
- FSM-legal transitions — describe invariants on the artifact (`done` doesn't reverse), not pipeline rules.
- Dependencies (`depends_on`) — describe what's structurally required, not what's queued behind what.
- Roles and ratification chokepoints — describe who has authority over which transitions, not who owns which stage.

Discarded:
- "Stage" as a vocabulary item. There is no `stage:` field in frontmatter, no `stages:` enum in contracts, no built-in `requirements → design → implementation` ordering.
- Handoff entities. A handoff is just a state transition with a role-typed actor; modeling it separately is double-bookkeeping.
- Workflow rules that encode *typical process order* (e.g., "design must complete before implementation can start"). This smuggles stages back in as cross-entity invariants. The framework should resist these; teams that need them can write them as project-specific checks, not as framework rules.

---

## 5. The vocabulary trap — "work" and "flow"

The user asked whether the words themselves are biasing the design. They are.

### 5.1 "Work"

"Work" in the industrial sense meant *labor applied to material to transform it*. The factory metaphor. In knowledge work, this was already strained — the material was information, the transformation was thinking, the boundary between "work" and "thinking" was always fuzzy.

With LLMs, the strain becomes a tear. The LLM does much of the *transformation* (drafting, generating, refactoring); the human does *judgment* about which transformations to keep. If "work" still means "labor applied to material," then most software work in 2026 is no longer work in that sense. It's curation, judgment, ratification.

The word still functions as shorthand and probably can't be replaced. But taken seriously, it's misleading: it suggests a producer-doing-labor frame when the actually-load-bearing activity is judge-applying-judgment.

### 5.2 "Flow"

"Flow" carries the assumption of *direction* — work moves from a source toward a sink, with intermediate transformations. This metaphor came from physical pipelines and was extended to information work by analogy.

Some flow remains real (CI, deploy, regulatory chains; see §2.1). Most of it doesn't, when examined. Decisions don't flow. They settle. New information unsettles them. They re-settle. There's no source and no sink. There's a *converging picture of what the team currently believes*. That's not flow; that's something more like *deliberation toward a converging state*.

Calling this "flow" biases the design toward shapes that fit pipelines better than they fit deliberation. Once you call something a workflow, you start adding stages, transitions, queues, assignments, due dates, escalations — all of which are correct for a pipeline and noise for a deliberation.

### 5.3 Better candidates

Three metaphors that fit better than "workflow" for the bulk of software work in 2026:

- **Cultivation.** Like gardening. You plant, prune, water, observe what grows, transplant. There are seasons but no assembly line. Things grow in parallel; you don't queue them. You revisit and revise. This metaphor handles re-entry, parallelism, and the absence of a fixed sink (the garden is never "done"; it's tended).
- **Deliberation.** Converging on a position through revision, with multiple participants, with re-entry. This is closer to how committees, courts, scientific communities, and editorial boards have always worked. The product is a *settled position*, not a *delivered artifact*.
- **State-keeping.** The framework's term. The work is *maintaining a coherent picture of what's currently true*. Updated continuously. Production of artifacts (specs, code, designs) is a means to this end, not the end itself.

None of these metaphors is the One True Picture. Each captures something workflow misses. The framework's positioning leans on *state-keeping* because it's the most directly implementable: a state model is a thing you can build; a deliberation engine is harder.

---

## 6. The Matrix reference, used carefully

> "Do not try and bend the spoon. That's impossible. Instead, only try to realize the truth: there is no spoon."

The reference is apt and worth engaging with.

In the film, the line means: *the constraint you think you're acting against doesn't exist; it's an artifact of your assumptions about the system you're in.* Neo can't bend the spoon as long as he believes the spoon is a real, separate object resisting him. Once he sees the spoon as part of a unified state he can edit, the resistance disappears.

For software work in the LLM era, the equivalent claim would be: *the workflow you think you're moving through doesn't exist; it's an artifact of your assumptions about how specialization and throughput shape labor.* Once you see the work as continuous deliberation over a shared state, the queues, handoffs, and stage boundaries dissolve — not because they were illusions all along, but because the constraints that produced them (specialist throughput bottlenecks) are dissolving.

This is *almost* right and importantly *not quite* right.

It's almost right because most of the workflow shape we inherited was scaffolding around throughput bottlenecks that no longer exist. When the bottleneck dissolves, the scaffolding becomes resistance to a constraint that isn't really there.

It's not quite right because some constraints *are* real spoons. CI is a real pipeline. FDA review is a real chain. You genuinely cannot deploy before you build. The Matrix line, taken as a slogan, risks misleading by suggesting all constraints are illusory. They aren't.

The careful version: **much of what we treat as workflow turns out, in the LLM era, to be a spoon that wasn't there. The state was always the truth; the flow was scaffolding around our throughput limits. Some pipelines are real and remain. Most aren't.**

The reference is fine as a teaching device in this document. It would be wrong as a framework tagline — slogans can't carry the qualification.

---

## 7. Implications for `aiwf`

The framework's existing design (per `01`–`06` and `KERNEL.md`) is mostly state-not-workflow already. The implications of taking this position seriously are mostly about *framing and what to refuse*, not about rebuilding the entity model.

### 7.1 Stays as-is

- Status enums per kind. Already pure state.
- FSM-legal transitions. Already invariants on the artifact, not pipeline rules.
- The six entity kinds (epic, milestone, ADR, gap, decision, contract). Already artifact-typed, not role-typed.
- Typed relationships (`parent`, `depends_on`, `supersedes`, `cites`, `discovered_in`, `addressed_by`, `relates_to`). Already statements about current state, not temporal order.
- One commit per mutation, history via `git log`. Provenance is orthogonal to state, as it should be.

### 7.2 Sharpens the framing

- The framework's pitch should lead with **"state model for the durable structural decisions of software work."** Not workflow tool. Not project-management tool. Not collaboration workspace. State model. This sharpens what it's for and removes ambiguity about whether it competes with Jira or Linear (it doesn't; they're workflow tools, this isn't).
- Skill content should describe operations as state transitions, not workflow steps. *"How to promote a milestone to `in_progress`"* not *"how to move a milestone from design to implementation."* Skills like `wf-promote`, `wf-add`, `wf-cancel` are already in this voice; keep them this way.
- Documentation that talks about the framework's value should foreground *"what is currently true about this work"* as the load-bearing question the framework answers.

### 7.3 What to refuse

- **No `phase:` or `stage:` field on entities.** Tempting for visualization. Bad for the model. If a team wants stage labels, they can compute them from status groupings.
- **No handoff entity.** A handoff is a state transition with a role-typed actor. Modeling it separately is double-bookkeeping.
- **No cross-entity workflow rules** that encode typical process order beyond what's already encoded as `depends_on` (which is a structural dependency, not a process rule). Resist "design must complete before implementation."
- **No assignee field as primary.** Assignment is a workflow concept. The actor on a state transition is recorded in the commit trailer; that's enough. If a team needs a "who's looking at this" surface, it's a render of recent activity, not a frontmatter field.
- **No SLAs, escalations, or time-bound transitions.** These are pipeline tools. The framework doesn't enforce them.

### 7.4 What workflow renders the framework can produce

For teams that want them:
- A Kanban board view, computed by grouping statuses into columns.
- A dependency graph, computed from `depends_on` edges.
- A burndown / cumulative flow diagram, computed from status changes over time in `git log`.
- A pipeline view (for CI / deploy / compliance), computed from a small per-team config that maps pipeline steps to status transitions.

Each of these is a renderer over the state, opt-in, computed on demand. None is a primary surface. None is privileged.

### 7.5 What this means for the build plan

The PoC plan in `06` does not need to change. Its entity model, status enums, and verbs are already state-shaped. The change is in *framing* (working paper, README, skill descriptions), not in code. Resist the urge to redesign code around this insight; the insight clarifies what's already being built.

The next layer (MCP server, web view, role-typed actors) should be designed with state-not-workflow as an explicit constraint. The MCP server exposes state queries and state mutations, not workflow operations. The web view shows current state, with workflow renders as opt-in views. The role concept is chokepoint authority over state transitions, not workflow position.

---

## 8. The honest failure mode

This position has a failure mode worth naming.

**The framework, framed as a state model, is wrong for teams whose work is genuinely workflow-shaped.** Specifically:

- **Regulated industries** where order of attestation is a legal fact. The framework can store the state these pipelines mutate, but the pipeline itself (review chain, audit trail, evidence collection) needs a workflow tool. Pretending the framework can replace that tool would be wrong.
- **Large enterprises** with formal handoff chains between specialist groups. When the bottleneck is genuinely throughput across organizational boundaries, workflow tools (Jira with custom workflows, ServiceNow, Camunda) describe the work better than state models.
- **Operations and incident response** where the sequence of actions is load-bearing (page → triage → mitigate → resolve → postmortem) and assignment matters in real-time. Runbook tools fit this; state models don't.
- **Sales and CRM-shaped work**. The pipeline is real; the stages are stations; the throughput matters. Salesforce was not a category error.

The framework's positioning, then, is for **teams where LLM amplification has already eroded the workflow shape**: solo developers, small teams doing trunk-based work, small product organizations, research-flavored engineering. These are not the only teams that exist. Pretending the framework is universal would be the same overreach as Spec Kit, Kiro, and ACE.

This is worth saying out loud in any framing document. The competitive position is not "workflow tools are obsolete." It is "for *these specific shapes of work*, state is the better organizing concept than workflow, and here's a small framework that takes that bet seriously."

---

## 9. What about "work" — the deeper substitution

If "work" is partly a trap, what should the framework actually be tracking?

Honest answer: **the converging picture of what the team currently believes about what they're building.**

This is not "work" in the labor-applied-to-material sense. It's closer to *belief state* — the team's collective answer to "what are we building, why, with what constraints, and what's been decided." LLM-amplified production happens around this belief state; humans ratify changes to it; the framework keeps the picture coherent.

If we wanted to be ruthlessly precise, the framework would be a **belief-state model** rather than a state model — but "belief state" is a term of art from AI/robotics that would confuse readers, and "state" is close enough. The framing in §6 ("state model for the durable structural decisions of software work") is the right level of precision: it points at belief state without using the technical term.

This substitution explains why workflow shape erodes: workflow is a model of *labor flow*, and the work-as-labor frame is dissolving. State is a model of *belief convergence*, and belief convergence is what's left when labor thins. Belief convergence is what the framework actually tracks — the thing that survives "work" being increasingly done by LLMs.

---

## 10. Open questions

The position does not settle several things; they're worth naming so future research can address them honestly.

1. **Does the state-not-workflow framing generalize outside software?** Probably partially. Knowledge work in general (research, writing, design, strategy) has the same throughput-erosion property. Domains with genuine pipelines (manufacturing, compliance, healthcare delivery) don't. Untested.
2. **At what team size does it break?** The framework targets solo through small-team. At what point does the state model become insufficient and a team needs explicit workflow tooling? Not measured.
3. **Does the chokepoint model survive multi-stakeholder ratification?** A state transition that requires three signoffs from three roles is workflow-shaped, even if it's framed as a chokepoint. The boundary between "chokepoint" and "approval workflow" needs sharpening when more than one ratifier is involved.
4. **How does the framework integrate with genuine pipelines (CI, deploy, compliance) that surround it?** State changes in the framework should trigger workflow tools downstream; workflow tools should report back state changes upstream. The integration shape is open.
5. **Is "belief state" the right precise term?** Adopting it would be honest but jargon-heavy. The current "state model" framing is approachable but slightly imprecise. Whether to migrate vocabulary later is open.
6. **Does the position survive empirical use?** The framework hasn't been used at scale yet. The state-not-workflow position might turn out to be defensible only at the scale and shape the PoC targets, and might need revision at the boundaries. Test by using.

---

## 11. The position in one paragraph

Software work in 2026 is dissolving the queue-based, station-bound, throughput-limited shape that "workflow" described well for several decades. LLMs collapse production time at every station, which evaporates queues, which dissolves stages, which makes "where is this in the pipeline?" stop being a meaningful question. What's left is *state* — what's currently true about the artifacts the team is converging on. The framework's bet is that state is the durable shape: the question "what is currently true about this milestone, this decision, this contract?" has an answer regardless of process shape, regardless of role specialization, regardless of how LLM-heavy the team's work is. Workflow remains real where genuine pipelines exist (CI, deploy, regulatory chains) and dissolves where it was scaffolding around throughput. The framework models state as canonical and workflow as one possible render among many. This is the position; "there is no workflow" is its slogan, with the standard caveat that slogans overreach and the careful version is in the body of this document.

> **A note on the framework's name.** "ai-workflow" predates this position. By the time the research arc landed on state-as-canonical and workflow-as-render, the name was already attached to the repo, the install script, the consumer adapters, and a body of writing. Renaming has real cost and is not the most pressing question; for now, the name is what it is, and the framework is a state model regardless of what it's called. Whether to rename is an open question, deferred.

---

## 12. References to the rest of the arc

- [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md) — the eight needs and cross-cutting properties; this document proposes adding "state primary, workflow optional" to the cross-cutting list.
- [`00-fighting-git`](https://proliminal.net/theses/fighting-git/) — established that the framework's job is small and substrate-respecting.
- [`01-git-native-planning`](https://proliminal.net/theses/git-native-planning/) — established markdown-canonical, git-as-time-machine.
- [`02-do-we-need-this`](https://proliminal.net/theses/do-we-need-this/) — questioned the premise; this document answers a different version of "what is the framework actually for?"
- [`03-discipline-where-the-llm-cant-skip-it`](https://proliminal.net/theses/discipline-where-the-llm-cant-skip-it/) — established CI as the chokepoint. State transitions via FSM are exactly the kind of mechanical guarantee that document called for.
- [`04-governance-provenance-and-the-pre-pr-tier`](https://proliminal.net/theses/governance-provenance-and-the-pre-pr-tier/) — established roles and ratification chokepoints. This document reframes those as authorities over state transitions, not positions in a workflow.
- [`05-where-state-lives`](https://proliminal.net/theses/where-state-lives/) — established the layer model. This document refines what *state* in "where state lives" means.
- [`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/) — the PoC. This document confirms the PoC's design is correct and proposes only framing changes.
- [working paper](https://proliminal.net/theses/working-paper/) — the defended position. This document feeds a framing-level revision: the framework is a state model first, with the explicit disclaimer that it's not a workflow tool.

---

## Appendix — The definition

The compressed statement of what the framework is, drafted in the conversation that produced this document and worth carrying forward:

> `aiwf` is a state model for the durable structural decisions of software work, designed for teams where LLMs participate in every role. It is not a workflow engine, a project-management tool, or a collaboration workspace. It is the place where the answer to *"what is currently true about this work"* lives, in a form that humans, LLMs, CI, and other tools can read and write through a small typed vocabulary.

This belongs in the working paper's abstract or §6 ("The position"), and at the top of the PoC's README. It is not a tagline (too long, too precise) but it is the right paragraph for any reader who needs to understand quickly what the framework is and what it isn't.

---

## In this series

- Previous: [06 — PoC build plan](https://proliminal.net/theses/poc-build-plan/)
- Next: [08 — The PR bottleneck](https://proliminal.net/theses/the-pr-bottleneck/)
- Forward: [10 — Spec-based development is waterfall in disguise](https://proliminal.net/theses/spec-based-as-waterfall/) — applies state-not-workflow to the spec-based methodology specifically; argues spec-based encodes workflow's artifact-handoff structure.
- Forward: [11 — Should the framework model the code?](https://proliminal.net/theses/should-the-framework-model-the-code/) — uses §4.1's state-vs-render distinction as the central lens.
- Synthesis: [working paper](https://proliminal.net/theses/working-paper/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
