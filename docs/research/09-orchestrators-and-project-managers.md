# Orchestrators and project managers

> **Status:** thesis-draft
> **Hypothesis:** Orchestration is a craft that includes the judgment subset of project-management work and adds technical fluency in LLM-produced artifacts; some PMs become orchestrators, some don't, and orchestrators come from many disciplines beyond PM.
> **Audience:** anyone who's heard "we need an orchestrator" and isn't sure whether that's a new job, a rebranded one, or marketing.
> **Premise:** [07](https://proliminal.net/theses/state-not-workflow/) and [08](https://proliminal.net/theses/the-pr-bottleneck/) repeatedly gestured at "the human applying judgment at the right ratification points"; this document asks who that is.
> **Tags:** #thesis #hitl #software-development #roles

---

## Abstract

The previous essay claimed LLM-amplified teams need humans applying judgment at the right ratification points. The obvious follow-up: who is that person? The current discourse names them an *orchestrator*, but the term is doing different work in different mouths and overlaps suspiciously with the PM role. This essay separates the concepts. An orchestrator's work is technical-flavored — deciding what to delegate to which LLM, designing the artifacts the LLMs read and write, inserting ratification chokepoints, managing context across sessions, integrating outputs, and applying real-time quality judgment. PM work splits into a judgment subset (scope, prioritization, trade-offs) that maps onto orchestration cleanly, and an administrative subset (status reports, ceremonies, schedule wrangling) that LLMs and shared state are eating. Some PMs become orchestrators; some don't. And orchestrators are not only ex-PMs — senior engineers, designers, tech leads also fill the role when they combine judgment with technical fluency. The throughline: **judgment scales; production doesn't need to**, and roles defined by judgment compound in value while roles defined by production lose differentiation. The framework's job is to give orchestrators (whatever they're called) better artifacts to ratify.

---

In my last post I claimed that LLM-amplified teams need humans applying judgment at the right ratification points. Several people asked the obvious follow-up: *who is that person?* The current discourse has a name for them — **orchestrator** — but the name is doing different work in different mouths, and the role overlaps suspiciously with one we already have. Project manager.

So: are orchestrators project managers? Can a PM step into an orchestration role? If not, why not?

I've been trying to answer this honestly and I think the answer is more interesting than either "yes, same thing, new word" or "no, it's a totally new role."

## What an orchestrator actually does

Strip the marketing. An orchestrator in an LLM-amplified team is doing roughly this:

- **Deciding what work to delegate to which LLM.** Not all LLMs are equally good at all things. Knowing when to use a deep-thinking model versus a fast one, when to use a code model versus a generalist, when to spin up a specialized agent versus continue in conversation — this is a real skill.
- **Designing the artifacts the LLMs read and write.** Specs, ADRs, contracts, milestone scopes. The orchestrator decides what structure the work fits into, because that structure is what gives downstream LLM sessions enough context to produce coherent output.
- **Inserting ratification chokepoints.** Where do humans need to say yes or no? Which transitions can be automated and which require judgment? The orchestrator places these gates.
- **Managing context across sessions.** What does this LLM need to know? What from the last session is relevant? What can be summarized away? Context budgets are real and the orchestrator manages them.
- **Integrating outputs.** One LLM's contract becomes another LLM's input. Outputs from parallel sessions need to converge on a coherent state. When they don't, the orchestrator notices and reconciles.
- **Real-time quality judgment.** Is this LLM output good enough to ratify? Does it solve the actual problem, not just the framed one? Should we revise the plan based on what just emerged? This is constant.

Notice the shape. Orchestration is *technical-flavored*. It requires understanding what the LLM is doing, not just that work is being done. It requires knowing the artifacts well enough to ratify them. It requires being close to the work — close enough to catch a misframed prompt, a missed edge case, a contract that won't compose with its neighbors.

The autonomy gradient this orchestration sits on has firmed up enough in 2026 to name. The Swarmia five-level taxonomy — assistive (inline suggestions), in-editor agent (Cursor agent mode, Claude Code interactive), background coding agent (headless, async, returns a PR), autonomous teammate (the agent picks tasks), agentic swarm (many parallel agents under planner/worker/judge hierarchies) — describes the levels reasonably well, even if the numbering is descriptive rather than load-bearing (citation in [`surveys/workflow-work-division-and-who-implements`](surveys/workflow-work-division-and-who-implements.md)). The orchestrator's job kicks in seriously at level 3 and dominates at levels 4–5. Anthropic's December 2025 internal study reports Claude Code chains an average of 21 independent tool calls per task without human turns — up from 9.8 six months earlier — while engineers describe their own future as "taking accountability for the work of 1, 5, or 100 Claudes." That phrasing names the orchestrator's role from the inside: the unit of accountability is the agent's session, and as session length grows, so does the leverage of the human directing it.

## What a project manager actually does

Now the same exercise, honestly, for traditional PM work. Setting aside the marketing here too:

- **Scope and prioritization** — what gets built, what gets cut, in what order.
- **Schedule and dependency management** — who's blocked on what, when do we hit the date.
- **Resource allocation** — who works on what.
- **Risk management** — what could go wrong, what's our mitigation.
- **Stakeholder communication** — keeping the people outside the team informed.
- **Status reporting and visibility** — making the work legible to leadership.
- **Process governance** — running ceremonies, enforcing methodology, keeping the team's process consistent.

This is a mixed bag. Some of it is genuine judgment (scope, prioritization, risk). Some is administrative scaffolding (status reports, ceremonies, calendar wrangling). Some is communication (stakeholders).

## Where they overlap

The judgment subset of PM work — *what gets built, in what order, with what trade-offs* — overlaps cleanly with orchestration. Both involve deciding what's worth doing now. Both involve sequencing. Both involve weighing competing concerns.

A PM who was doing this work — *actually weighing trade-offs, deciding scope, choosing what to defer* — has the right mental muscles for orchestration. The judgment is the same shape. They just have to apply it to LLM-amplified work instead of human-only work.

This is the case where the answer is "yes, same thing, mostly." A senior PM who was doing strategic work can become an orchestrator. The transition is real but not radical.

## Where they diverge

Now the harder part. Orchestration requires something PM work historically didn't: **understanding what the producer is doing well enough to ratify it.**

Pre-LLM, a PM didn't need to understand the engineer's code. The engineer produced; the PM tracked. The judgment was *did we ship it on time, does it match the requirements, did stakeholders sign off?* — judgments external to the work.

LLM-era orchestration is different. The LLM produces fast and produces a lot. The orchestrator's judgment has to be *internal to the work* — does this contract actually compose with the others, does this ADR's reasoning hold, does this milestone's acceptance criteria really capture what we mean by "done"? An orchestrator who can't read the artifacts can't ratify them. They can only ratify that the LLM said it was done, which is exactly the failure mode of full-autonomy systems.

This is where some PMs hit a wall. If a PM's value was in administrative scaffolding — running ceremonies, generating status reports, keeping the calendar — that work is being eaten by LLMs and shared state. Standups are increasingly redundant when everyone can see the work in real-time. Status reports write themselves from the artifact state. Ceremonies optimize for visibility that the artifacts already provide.

A PM whose role was mostly administrative will find that orchestration requires something they weren't doing before: technical fluency in the artifacts. Not "can write code" necessarily, but "can read a spec critically, can spot a misframed contract, can tell when an ADR's reasoning is thin." That's a real skill that takes time to develop.

## The honest answer

Some PMs become orchestrators. Some don't. It depends on what they were actually doing.

PMs whose value was in *product judgment plus technical understanding* — the strategic ones, the ones who could push back on engineering with good reasons, the ones who shaped scope based on real understanding of trade-offs — slot into orchestration naturally. The work is recognizable; only the medium changes.

PMs whose value was in *coordination and communication* — keeping the team aligned, managing stakeholders, running process — find that the LLM era reduces demand for the coordination work (the artifacts handle it) while *increasing* demand for judgment they may not have built. They can develop that judgment. It isn't free.

And — this is the part the discourse usually misses — orchestrators don't have to come from PM. They come from anywhere that produces the right combination of *judgment about what's worth doing* and *technical fluency in the artifacts*. Senior engineers who developed product sense. Designers who learned systems thinking. Tech leads who took on more strategic scope. PMs who got close to the work. The orchestrator role isn't owned by any traditional discipline. It's the role that emerges when LLM-amplified work needs human direction, and it's filled by whoever can do both halves of that job.

## What roles fade and what stays sharp

Pulling on the same thread, the broader role economy in LLM-amplified teams:

**Fading:**
- Roles whose value was in *production throughput* — junior engineers, junior designers, junior PMs, manual QA. The LLM produces faster. Specialization-as-throughput is what's eroding.
- Roles whose value was in *administrative coordination* — schedule wrangling, status reporting, ceremony-running. The artifacts handle this when they're structured right.
- Roles whose value was in *translating between specialists* — the bridge roles that existed because PMs didn't speak engineer and engineers didn't speak design. LLMs translate. The bridges thin.

**Strengthening:**
- Roles whose value is in *judgment under uncertainty* — what's worth building, what's good enough, what's the right trade-off. The orchestration cluster.
- **Specialty judgment domains** — legal, security, accessibility, compliance, regulatory. Anywhere being wrong has real cost and the LLM is not yet trustworthy alone. These survive as ratification roles; the human says yes or no, possibly with LLM-generated evidence.
- **Senior craft** — the people who can tell when LLM output is subtly wrong, who hold the team's taste, who know what good looks like in their domain. Not because they produce more than the LLM, but because they *judge* better than the LLM.

The throughline: **judgment scales; production doesn't need to.** Roles defined by judgment compound in value. Roles defined by production lose differentiation. The MSR 2026 study of agent-authored PRs measures this from the outside: 58% of human intervention effort on agent PRs is *guidance-level* — restricting agent behavior, enforcing project conventions — and only 17% is direct code changes (citation in [`surveys/workflow-work-division-and-who-implements`](surveys/workflow-work-division-and-who-implements.md)). The shape of the surviving work is precisely the orchestrator's: directing what the agent does, ratifying what it produces, holding the conventions the agent doesn't infer. "What roles fade and what stays sharp" is, increasingly, the same question as "where in the LLM-amplified workflow does the human still have to think."

## Are orchestrators project managers? The compressed answer

No, but they have overlapping ancestry. Orchestration is a craft that *includes* the judgment subset of PM work — scope, prioritization, sequencing, trade-offs — and *adds* technical fluency in the artifacts the LLMs are producing. PMs who already had both can step in. PMs who had only the judgment half can develop the technical half. PMs whose value was mostly administrative will find the role doesn't fit.

It's also worth saying: *orchestrators aren't only ex-PMs.* The role is filled equally by senior engineers who grew product sense, designers who grew systems thinking, tech leads who grew strategic scope. The discipline of origin matters less than the combination of judgment and technical fluency.

## Where this applies and where it doesn't

The orchestration framing fits teams where:

- LLMs are doing significant production work.
- The artifacts (specs, contracts, ADRs, plans) are structured enough to be ratified rather than re-read.
- The team has moved past the "PRs don't scale" pain into something more continuous.

It fits less well in teams where:

- The PM role is largely about external stakeholder management — sales coordination, customer communication, executive reporting. That work persists and isn't really orchestration.
- The work is genuinely workflow-shaped (regulated industries, formal handoff chains, ops/incident response). Project management in those contexts has structural reasons to exist that LLMs don't dissolve.
- The team isn't yet using LLMs heavily. The orchestrator question doesn't fire if production is still throughput-limited at humans.

I'm not arguing PMs are obsolete. I'm arguing the role splits in a particular way under LLM amplification — judgment work converges with orchestration; administrative work gets eaten by LLMs and shared state — and people end up on different sides depending on what they were actually doing.

## The closing thought

Orchestration is a craft, not a job title. It'll get a job title eventually because the industry can't help itself, and that title will be filled by people from many traditional roles. The interesting question isn't "what should we call this person?" — it's "what work are they doing, and how do we make that work easier?"

I think the answer to the second question is *give them better artifacts*. Structured planning state. State models that survive re-entry. Ratification chokepoints that don't require reading thousand-line diffs. The framework I'm building is one attempt at this. There will be others.

But the role that uses those tools well — the orchestrator, whatever we end up calling them — is going to look less like a project manager *managing* a team and more like a senior craftsperson *directing* a workshop. They'll be close to the artifacts. They'll have strong opinions. They'll say no often. They'll move work fast not by producing more but by deciding faster about what's worth keeping.

If that sounds like a senior PM you've worked with, you've worked with a good one. If it sounds like a senior engineer, same. The role is what emerges from doing both halves of the job. The org chart hasn't caught up yet.

---

*Next, probably: what the artifacts an orchestrator needs actually look like — making the artifact-as-judgment-surface concrete.*

---

## In this series

- Previous: [08 — The PR bottleneck](https://proliminal.net/theses/the-pr-bottleneck/)
- Next: [10 — Spec-based development is waterfall in disguise](https://proliminal.net/theses/spec-based-as-waterfall/) → [11 — Should the framework model the code?](https://proliminal.net/theses/should-the-framework-model-the-code/)
- Related: [07 — State, not workflow](https://proliminal.net/theses/state-not-workflow/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
