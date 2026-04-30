# The PR bottleneck is a process problem

> **Status:** thesis-draft
> **Hypothesis:** The PR bottleneck is a process problem, not a tool problem; continuous ratification by humans at state transitions replaces batched post-hoc review, and HITL strengthens rather than dissolves under LLM amplification.
> **Audience:** anyone feeling the weight of LLM-generated PRs and wondering if the answer is a better PR tool.
> **Premise:** [07](https://proliminal.net/theses/state-not-workflow/) is the technical version of this argument; this document is the essay version, aimed at a wider reader.
> **Tags:** #thesis #hitl #git #software-development

---

## Abstract

LLMs broke the production-time assumption that PR-based review was sized to. Pre-LLM, production was slow and review fit in the gaps; post-LLM, production is fast and review queues grow. The PR is where the throughput limit becomes visible, but it isn't the cause. This essay argues that the answer isn't a better PR tool but a different process: **continuous ratification** by humans at state transitions, distributed across the work, replaces batched post-hoc review of large diffs. The unit of human attention shifts from issue (production unit) to milestone (ratification unit). What we lose — the social functions of PRs (teaching, knowledge sharing, psychological safety) — is real and named honestly; the post doesn't pretend continuous ratification handles them. The corollary that follows from the argument is the inversion of the standard "AI replaces humans" frame: **HITL gets stronger, not weaker, when AI joins the team** — because every LLM session needs ratification, and ratification scales better than production.

---

I keep seeing the same complaint from teams that have started using LLMs seriously: too many PRs, too large, humans can't review them fast enough. The instinct is to fix the tool — better diff viewers, AI-assisted review, suggestion bots. I think the instinct is wrong. The PR isn't broken. The *process* is. The PR is just where the breakage becomes visible.

This is a draft of why I think so, and what I think replaces it.

---

## The symptom

LLMs don't care about what's right-sized for a human. An LLM can work tirelessly on a scope larger than a typical issue, faster than any human could, and produce a coherent diff that touches twenty files. If we let LLMs open PRs and route them through human review, we get exactly what teams are reporting: a queue of large PRs that humans can't keep up with.

The standard responses are predictable. Work longer hours. "Multitask" through reviews (which means rubber-stamping). Buy a better tool. Hire more reviewers — except nobody actually hires reviewers; the trend is the other way. Or quietly let the LLM merge its own work, which means dropping HITL entirely.

None of these fix the problem. They just decide which form of damage to absorb.

## The diagnosis

The PR is a bottleneck because *our entire process was sized to human production capacity, and LLMs broke that sizing*. Pre-LLM, a human took a week to write a feature, an hour to review it, a few minutes to merge. The ratios worked. The reviewer was idle most of the time relative to the producer. Production was slow; review fit in the gaps.

Post-LLM, production is fast. A producer-with-LLM ships in a day what used to take a week. The reviewer's time hasn't changed. The ratios invert. Suddenly there's no slack in the reviewer's day; every hour they look at PRs is an hour they're not producing themselves. The queue grows.

This isn't a bug in the PR tool. It's a property of the *process* — the assumption that production happens privately, then surfaces all at once for batched, post-hoc review. That assumption was reasonable when production was slow. It isn't anymore.

## What a PR actually is

A pull request, in git's etymology, is a request to *pull* a branch into a shared one. It's an integration boundary. Review is something that got attached to the integration boundary because the integration boundary was a convenient place to put it.

This matters because "kill the PR" sounds like "kill peer review," which it isn't. Peer review is valuable and stays. What dies is the *batched, post-hoc, all-at-once* version of integration that the PR encodes. The integration moment can be smaller, more continuous, more woven into the work — and review can come earlier, distributed across the work rather than concentrated at the end.

So the question is not whether to kill review. The question is whether *batched post-hoc review of large diffs* is the right unit of human judgment in a world where LLMs produce most of the diff.

## Two paths the industry is taking, both wrong

The first path is **full autonomy.** Multi-agent frameworks where roles are filled by LLMs, work flows between them, and humans only see the output. No HITL. The bet is that the LLMs are good enough to police themselves. They aren't, not yet, and probably not for the kinds of work where judgment under uncertainty matters most. Drift accumulates. Direction is lost. The system optimizes for what the LLMs can measure, which is rarely what the team actually wants.

The second path is **spec-then-execute.** Human and LLM collaborate on a spec; once the spec is agreed, the LLM executes it. This is better — judgment happens early, when the cost of changing direction is low. But it inherits the PR's worst feature: review is still batched and post-hoc, just moved upstream. You signed off on the spec; now you have to read what the LLM produced from it and verify it matches. That's still a large diff to review.

Both paths leave humans either too far out of the loop (path 1) or in the loop only at gates (path 2). The actual problem — that *batched review of large output doesn't fit a world of fast production* — is unaddressed.

## A third path: continuous ratification

The model I think actually works: humans and LLMs collaborate continuously across the work. The plan is shaped together, the LLM proposes, the human ratifies as work progresses, gaps are surfaced and decided in real-time. By the time a milestone is "done," there is no PR-shaped review event because *all the review has already happened*, distributed across the work.

The unit of human action shifts from "review a diff" to "ratify a state transition." Did this milestone reach `in_progress` legitimately? (Did we agree the scope? Yes.) Did this ADR get accepted? (Did we discuss the trade-off? Yes.) Did this gap get addressed? (Did we look at the resolution? Yes.) Each ratification is small, fast, and contextual. None of them require sitting in front of a thousand-line diff.

What this requires:

- **The unit of work has to be sized to the human's ratification capacity, not the LLM's production capacity.** A milestone — coherent scope, clear acceptance criteria, a few days of LLM work — fits in a human's head as something to ratify. An issue is too small (you'd ratify forty a day); a feature release is too large (you can't hold all the trade-offs at once). Milestone is the sweet spot.
- **State has to be visible and structured.** What's currently true about each milestone, ADR, decision, contract? The human needs to be able to see the state without re-reading the diff. This is what makes ratification fast.
- **Ratifications themselves have to be a durable record.** Git is the audit trail of edits. We also need an audit trail of *judgments* — who decided what, when, with what reasoning. This is the missing artifact in today's tools.

What you give up: the LLM's autonomy. The LLM can't just "go off and do it." It has to come back to the human for ratification at every state transition. This is a feature, not a bug. The LLM doing more without the human means the human ratifying more later in larger chunks, which is the bottleneck we're trying to escape.

## The unit of human attention shifts

A human can produce maybe one issue's worth of code per day. A human can *ratify* ten times that, if the LLM has done the production. So the right-sized unit for a human in 2026 is a milestone, not an issue. Milestones are larger, contain multiple commits, and align with what a human can hold in their head as a coherent change to ratify.

This is the inversion that makes continuous ratification work. The PR-era bottleneck was "human reviews each diff." The LLM-era bottleneck is "human ratifies each state transition." The second scales because it operates on smaller, more frequent, more contextual decisions instead of on large, infrequent, decontextualized ones.

## What we lose — the social cost

PRs aren't only mistake-catching. They serve other functions worth naming honestly:

- **Teaching.** Junior engineers learn from senior reviewers. The PR is a structured place where craft gets transmitted.
- **Knowledge sharing.** "You're touching that code? Here's what I learned about it last quarter." Cross-team awareness flows through PR threads.
- **Psychological safety.** The act of getting approval before merging is reassuring. You're not alone with the decision.

Continuous ratification between one human and an LLM doesn't replicate any of these. The LLM can simulate teaching ("here's why this pattern is preferred"), but it isn't a senior engineer transmitting taste. The LLM can surface relevant prior code, but it isn't a teammate sharing what they learned. The LLM can ratify, but its approval isn't the same as a peer's.

So *for solo work*, this loss is mostly theoretical — those functions weren't there to begin with. For team work, the loss is real and needs a different mechanism: pairing sessions, milestone-level reviews between humans (not diff-level), deliberate teaching surfaces. Continuous ratification handles the bottleneck-as-mistake-catching, but it doesn't handle the bottleneck-as-craft-transmission. That's a different problem and the post isn't going to solve it; just naming it honestly.

## Where this applies and where it doesn't

The continuous-ratification model works cleanly for:

- Solo developers using LLMs heavily.
- Small teams doing trunk-based product work.
- Greenfield and small-to-medium brownfield projects.
- Research-flavored engineering where direction shifts often.

It works less well for:

- **Regulated industries** where the order of attestation is a legal fact. The pipeline is real and isn't dissolving.
- **Large enterprises** with formal handoff chains between specialist groups, where workflow tools (Jira, ServiceNow, Camunda) describe the work better than state models.
- **Operations and incident response** where sequence of action is load-bearing.
- **Anywhere with hard compliance requirements** that prescribe a specific review chain.

I'm not arguing PRs are dead everywhere. I'm arguing that *for the kinds of work where PRs are felt as ceremony rather than as load-bearing structure*, continuous ratification is the better answer. The PR-as-bottleneck pain is the signal that you're in the first kind of work, not the second.

## The corollary

If LLMs handle production and humans do ratification, and ratification scales better than production, then **adding LLMs increases the leverage of each human reviewer rather than replacing them.**

This is the inversion that matters. The standard frame is "AI replaces jobs." The actual shape, when the bottleneck is judgment rather than production, is the opposite: the more LLMs are involved, the *more* human judgment matters, because every LLM session needs ratification. You don't need fewer humans. You need more humans applying judgment at the right places, with the right context, with the right tools to ratify quickly.

HITL doesn't dissolve when AI gets good. HITL gets *stronger*. The judgment chokepoints multiply. The leverage of each human's "yes" or "no" goes up, not down.

This is the corollary worth taking seriously: in the LLM era, **HITL > more humans, not fewer.** The teams that figure out how to put humans at the right ratification points, with the right cadence, with the right artifacts, will move faster than teams that either drown in PRs or hand the keys to the LLMs entirely.

The PR bottleneck is the symptom. Continuous ratification is the process. Humans, doing more judgment with less production, are the leverage.

---

*A future post: orchestration as a distinct role from project management — what changes about the work of "deciding what gets built" when LLMs do the building. Different problem, different post.*

---

## In this series

- Previous: [07 — State, not workflow](https://proliminal.net/theses/state-not-workflow/)
- Next: [09 — Orchestrators and project managers](https://proliminal.net/theses/orchestrators-and-project-managers/)
- Related: [03 — Discipline where the LLM can't skip it](https://proliminal.net/theses/discipline-where-the-llm-cant-skip-it/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
