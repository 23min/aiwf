# aiwf is operating-model-agnostic — what the wider landscape implies

> **Status:** defended-position
> **Hypothesis:** aiwf is not a position relative to Spec-Driven Development specifically; it is the durable structural-state layer underneath whatever AI-assisted operating model a team runs — vibe coding, pair-programming, TDD-with-AI, issue-driven, plan-execute-review, memory-driven, harness/skill-driven, exploration-first, contract-first, formal methods, workflow-tool-driven, agentic-swarm. The arc's prior framing pinned the bet against SDD because SDD is the loudest discourse; the wider survey shows SDD is one cell in a roughly thirteen-cell landscape, and the framework's lane is *under* the operating model, not against any particular one.
> **Audience:** anyone who read [`10`](10-spec-based-as-waterfall.md) and inferred the framework is a position on SDD specifically; anyone running an operating model that isn't SDD-shaped and wondering whether the framework is for them.
> **Premise:** the three field surveys ([`surveys/understanding-spec-driven-development`](surveys/understanding-spec-driven-development.md), [`surveys/workflow-work-division-and-who-implements`](surveys/workflow-work-division-and-who-implements.md), [`surveys/ai-assisted-operating-models`](surveys/ai-assisted-operating-models.md)) mapped the field; this document closes the loop by stating where in that landscape aiwf sits and what that implies for scope, framing, and what to refuse.
> **Tags:** #thesis #aiwf #scope #positioning

---

## Abstract

The arc's earlier docs argued the framework's bet from first principles ([`07`](07-state-not-workflow.md), [`08`](08-the-pr-bottleneck.md)) and against the closest competing organizing shape ([`10`](10-spec-based-as-waterfall.md)). That positioning landed cleanly for readers who already had Spec-Driven Development in mind. The third survey, written after `10`, showed SDD is one named family among at least a dozen distinct AI-assisted operating models in active practice — vibe coding (Karpathy's term), pair-programming-with-Claude-Code, TDD-with-AI, issue-driven (Devin / Copilot coding agent / Codex cloud), plan-execute-review, memory-driven (CLAUDE.md / AGENTS.md / `.cursorrules`), harness/skill-driven (Anthropic Skills, MCP, ThoughtWorks Vol 34's "harness engineering"), exploration-first, contract-first/API-first, formal-methods-with-AI, workflow-tool-driven (Jira / Linear / GitHub Projects), and agentic-swarm. Most teams run two or three of these at once, often without naming any of them. Anchoring the framework's positioning to SDD is therefore narrower than the bet aiwf actually makes. This document widens the framing: aiwf is **operating-model-agnostic** — the durable structural-state layer underneath whatever operating model a team runs. The kernel's eight needs are about belief-state, not workflow shape; the framework composes with vibe coding (sparingly), pair-programming, issue-driven, harness-driven, and SDD all in the same way. What it refuses is the cell it explicitly does not fit (full delegation / agentic-swarm without ratification chokepoints) and the cell that wants to absorb it (workflow-tool-driven, where the tool's data model would replace the framework's). This is the same restraint [`11`](11-should-the-framework-model-the-code.md) applied to code-graph tools: *compose, don't absorb*. The framework's lane is neither a competitor to SDD nor a successor to it — it is a layer underneath any operating model that has durable structural decisions, regardless of what the operating model is called.

---

## 1. Why this widening is needed

The arc's positioning was built primarily against SDD because SDD is the loudest discourse in the field as of mid-2026. [`10`](10-spec-based-as-waterfall.md) carried that framing: "spec-based development is waterfall in disguise; continuous ratification is structurally different." The argument is sound for what it argues against, but it leaves an implicit reader: someone who sees the framework, asks "is this a Spec Kit competitor?", and reads `10` as "yes, against Spec Kit specifically."

That reader is reading the framework wrong. The bet is wider than the SDD critique. The third survey [`surveys/ai-assisted-operating-models`](surveys/ai-assisted-operating-models.md) makes the wideness visible: SDD covers one cell in a grid that has at least a dozen, and most teams running AI-assisted work in 2026 are not in the SDD cell. They are vibe coding, pair-programming, issue-driving, plan-execute-reviewing, memory-driving, or some combination — usually without naming any of these.

The framework's positioning has to be readable to those teams too, or the bet is narrower than its own substance. This document closes that gap.

It also retroactively scopes [`10`](10-spec-based-as-waterfall.md): *that* document's structural critique applies to the heavyweight rungs of SDD specifically (spec-first reaching for spec-anchored, with spec-as-source as the visible aspiration), and it always did, even though the abstract did not name the limit clearly until the survey was written. The tightening sits in `10`'s revised abstract; the wider positioning sits here.

## 2. The landscape, briefly

The three surveys carry the full mapping; this section names the cells without re-arguing them. From [`surveys/ai-assisted-operating-models`](surveys/ai-assisted-operating-models.md):

- **Vibe coding.** No durable artifact + after-the-fact judgment. Karpathy named it February 2, 2025; the term stuck because it described what people were already doing.
- **Pair-programming-with-LLM.** Minimal durable artifact + continuous turn-by-turn judgment. The unnamed modal practice for senior engineers using AI tools heavily.
- **TDD-with-AI.** Tests as durable artifact + continuous judgment, with the test runner as the deterministic gate. Predates LLMs by a quarter-century; composes well with them.
- **Issue-driven (background-agent).** Issue as durable artifact + after-the-fact PR review. Devin (March 2024), Copilot coding agent (May 2025), Codex cloud (2025) are the visible tools.
- **Plan-execute-review.** Ephemeral plan + judgment up-front-and-continuously. Folkloric — practiced widely, almost never named.
- **Memory-driven.** Long-running memory file (CLAUDE.md, AGENTS.md, `.cursorrules`) as durable artifact + continuous judgment with the memory as a strong prior. ThoughtWorks Vol 33 placed it in Adopt.
- **Skill-driven / harness-driven.** Executable skills as durable artifact + continuous judgment, with skills as procedural memory. Anthropic Skills (October 2025), MCP (November 2024), Cursor commands. ThoughtWorks Vol 34 named "harness engineering" alongside SDD as a parallel emerging practice.
- **Spec-Driven Development.** Spec as durable artifact + judgment up-front or at gates. Five rungs: spec-as-prompt, spec-first, spec-anchored, spec-as-source, spec-as-contract. Detailed in [`surveys/understanding-spec-driven-development`](surveys/understanding-spec-driven-development.md) and audited in [`10`](10-spec-based-as-waterfall.md).
- **Exploration-first / spike-driven.** Throwaway prototype + continuous judgment, with the right to restart. The artifact is meant to be discarded; the learning is the output.
- **Contract-first / API-first.** Contract as durable artifact + judgment up-front-or-at-gates, at the API surface only. OpenAPI, Protobuf, AsyncAPI, JSON Schema. Predates LLMs by a decade.
- **Formal-methods-with-AI.** Formal model (TLA+, Lean, Alloy, Dafny) as durable artifact + judgment up-front, with proof or model-checking as the gate. Small but real cohort; LLMs help mostly in the translation layer.
- **Workflow-tool-driven.** Tool's data model (Jira, Linear, GitHub Projects, ServiceNow) as durable structure + judgment at gates. The status quo for two decades; not eroded by AI, just amplified.
- **Agentic-swarm.** Inter-agent contracts as durable artifact + judgment at the swarm boundary. Mostly research-grade or vendor-internal as of mid-2026.

Most teams hybridize: a startup might run Spec Kit for new features (SDD), Cursor with `CLAUDE.md` day-to-day (memory + pair-programming), OpenAPI at service boundaries (contract-first), and Linear for ticket flow (workflow-tool-driven). Per the Anthropic December 2025 study cited in the third survey, task-level routing across operating models is the modal practice for senior engineers using AI tools heavily.

## 3. The kernel test, applied to the operating-model question

The kernel ([`KERNEL.md`](KERNEL.md)) is the rubric. Walk the operating-model question against it: *does any of the framework's eight needs require a specific operating model to be served?*

1. **Record planning state** — no. Planning state is what the team currently believes about epics, milestones, decisions, gaps, contracts. Vibe coders, pair-programmers, issue-drivers, harness-builders, and SDD practitioners can all have this state. None of them is required to.
2. **Express relationships** — no. A milestone belongs to an epic regardless of how the team works inside the milestone. A decision ratifies scope regardless of whether the scope was articulated as a Spec Kit `spec.md` or a Cursor chat.
3. **Support evolution** — no. Plans are clay regardless of operating model.
4. **Keep history honest** — no. Provenance via git log + structured trailers is operating-model-independent.
5. **Validate consistency** — no. References resolve, FSM transitions are legal, IDs do not collide — these are properties of the structural state, not of how the team produced it.
6. **Generate human-readable views** — no. Views derive from canonical state; the state's source operating model does not affect the render.
7. **Coordinate AI behavior** — *partially*. Skills and rules shape how AI acts; different operating models impose different conventions on which skills/rules matter. But the framework's job is to coordinate skills as content, not to prescribe the operating model.
8. **Survive parallel work** — no. Merging structural state is operating-model-independent.

Net: none of the eight needs requires a specific operating model. The framework is operating-model-agnostic by construction; the prior arc framing was an accident of which discourse was loudest, not a property of the kernel.

The cross-cutting properties tell the same story: enforcement-not-LLM-dependent, referential stability, honest about meaning, engine-invocable-without-AI, soft/strict tier model, modular opt-in — none of these select for an operating model. *Modular opt-in* explicitly leaves operating-model decisions to the consumer.

## 4. Where aiwf composes (most cells) and where it refuses (two)

The framework's posture is **compose, don't absorb** — the same posture [`11`](11-should-the-framework-model-the-code.md) committed to with code-graph tools. Apply that lens to the operating-model landscape.

### 4.1 Composes cleanly

For most cells, aiwf and the operating model run in different layers and require no integration beyond convention:

- **Vibe coding.** Aiwf is over-equipped for pure vibe coding (the operating model assumes no durable belief-state). But many "vibe coders" in fact have implicit structural decisions (an epic in their head, an ADR they would write down if asked) — aiwf is a way to externalize those when they want to. Optional, not imposed.
- **Pair-programming-with-LLM.** Aiwf is a session-spanning memory of structural decisions; the pair-programming session reads from and writes back to that memory. Skill-shaped integration: *before structurally editing a contract, read the milestone scope; before promoting a milestone to `done`, ratify acceptance criteria.*
- **TDD-with-AI.** Aiwf doesn't replace the test suite — the tests are the operating model's spec. Aiwf records the milestones and contracts the tests are evidence for. The test runner stays authoritative for behavior; aiwf stays authoritative for the structural decisions about what behavior is wanted.
- **Issue-driven.** The issue body is a render of (part of) the milestone scope; the agent's PR is a contribution to the milestone's progress. Aiwf integrates by import (issue → milestone update) and export (milestone → issue body), not by replacing the issue tracker.
- **Plan-execute-review.** The plan is ephemeral; aiwf records the durable consequences (decisions, ADRs, gaps surfaced). Each session writes back to the structural state at its conclusion; the plan itself can be discarded.
- **Memory-driven.** Aiwf is itself partly memory-shaped (it makes structural decisions queryable across sessions). It composes with `CLAUDE.md` / `AGENTS.md` rather than replacing them: those files carry behavioral rules; aiwf carries structural state; the assistant reads both. Worth saying explicitly: aiwf is not a replacement for `AGENTS.md`-style instructions.
- **Harness/skill-driven.** Strongest natural composition. Aiwf's skills *are* harness components in the ThoughtWorks Vol 34 sense. Skill content invokes engine verbs; the framework's pre-PR validators are feedback controls. The two practices reinforce each other.
- **Spec-Driven Development.** Per [`10`](10-spec-based-as-waterfall.md): the framework is structurally different (state vs. handoff), but composes with SDD tools at the boundary. A Spec Kit `spec.md` for one milestone is a render of the milestone's scope at the moment of the change request. An OpenSpec delta is a render of "what's changing about this contract." A spec-as-contract OpenAPI document is the body of a contract entity. None of these are absorbed; all are integrated by import/export at the boundary.
- **Exploration-first.** The exploration is below the framework's lane (the throwaway code is meant to be thrown away). What aiwf records is the *learning* the exploration produced — a gap surfaced, a decision made, a milestone reframed. The exploration's own artifacts can be deleted; the structural consequence persists.
- **Contract-first / API-first.** Contracts are already a kind aiwf supports. An OpenAPI document is the body (or a `live_source` reference target) of a contract entity. Contract verification is mechanical; the framework's contract-verify hook composes with whatever validator the consumer chose.
- **Formal-methods-with-AI.** Same shape as contract-first at a different formality tier. A TLA+ spec is the authoritative body of a contract entity; the model-checker is the verification tool the framework's hook delegates to. Brooker's "free-form prose → RFC 2119 / EARS → Lean / TLA+" formality ladder maps onto the framework's `severity` and validator pointers, not the kernel.

### 4.2 Refuses, in two directions

The framework explicitly does not fit two cells, and naming this honestly is part of the positioning:

**Workflow-tool-driven (as primary substrate).** Jira, Linear, GitHub Projects, ServiceNow describe the work as a *workflow* — stations, columns, queues, assignees, transitions, SLAs. The framework's bet ([`07`](07-state-not-workflow.md)) is the inverse: state primary, workflow render. Teams whose primary substrate is a workflow tool can run aiwf alongside (state-keeping at the structural-decision layer; workflow-tool-keeping at the ticket layer), but they cannot make aiwf the workflow tool. The framework refuses to be one. This is by design and consistent with the kernel.

**Agentic-swarm without ratification chokepoints.** The framework's HITL bet ([`08`](08-the-pr-bottleneck.md)) is that human ratification at structural transitions is what scales when production goes parallel. A Level-5 swarm (Cursor's reported FastRender, planner-worker-judge hierarchies, two-thousand-instance parallel runs) that strips ratification is the cell where the framework's bet dissolves. Aiwf can support swarms that retain human ratification at milestone-level chokepoints (the human ratifies the milestone's promotion; agents do everything below); it does not support swarms that ratify their own milestones.

These refusals are not "everywhere else is wrong." They are "the framework is wrong for these two cells, and naming that protects the framework's value where it does fit."

## 5. The audiences distinction, restated

[`02`](02-do-we-need-this.md) §6 made a load-bearing observation: *structured state pays off for programmatic consumers (CI gates, dashboards, audit), not for AI assistants. Confusing these two audiences inflates the design.*

The operating-model question is the same observation from a different direction. Every operating model produces *some* state in *some* form; what differs is whether the state is structured for programmatic consumption or only for the assistant. Vibe coding produces state only in code (programmatic, but not structured for planning). Memory-driven produces state in instruction files (structured for the assistant, not for CI). Workflow-tool-driven produces state in the tool (structured for both, but the tool's schema, not the framework's).

aiwf produces *structural-decision state in a small typed vocabulary*, designed for both audiences (assistant *and* CI/audit). That is its lane. The lane is independent of the operating model that *produced* the decisions, because the lane is about how the decisions are *recorded and validated*, not how they were arrived at.

This is why operating-model-agnosticism is principled, not lazy. The framework's value proposition lives at the structural-decision layer; that layer exists in any operating model that has durable structural decisions, which is most of them.

## 6. What the framework's framing should say (and currently doesn't quite say)

The arc's current artifacts position the framework against SDD directly:

- [`10`](10-spec-based-as-waterfall.md) is structured against SDD.
- [`07`](07-state-not-workflow.md) §5.3 names workflow-tool-driven as the failure mode.
- [`KERNEL.md`](KERNEL.md) is operating-model-silent (which is correct but does not affirmatively position the framework).
- The working paper and READMEs lean implicitly on the SDD framing because that is what the loud discourse asks about.

The widening this document proposes is one paragraph in the working paper / README and one section in `0-introduction`, not a redesign. The paragraph reads roughly:

> aiwf is the layer of *durable structural decisions* underneath whatever AI-assisted operating model you run — vibe coding, pair-programming, TDD-with-AI, issue-driven, plan-execute-review, memory-driven, harness/skill-driven, spec-driven, exploration-first, contract-first, formal-methods, agentic-swarm-with-chokepoints. The kernel does not select for an operating model; the framework composes with whichever you run, integrates with whichever tools that operating model brings, and refuses the two cells where its bet does not fit (workflow-tool-driven as primary substrate, swarms without human ratification). Whatever shape your sessions take, the question *what does the team currently believe is true about this work?* has an answer; aiwf is the place where that answer lives.

This is the operating-model-agnostic positioning, in one paragraph. It is not a tagline; it is the right paragraph for a reader trying to place the framework against the landscape they actually inhabit.

## 7. What this changes about the rest of the arc

Mostly it sharpens, doesn't redirect.

- [`10`](10-spec-based-as-waterfall.md) — already updated in the recent refactor to scope the structural critique to the heavyweight rungs of SDD specifically. This document confirms that scoping and adds the wider context.
- [`07`](07-state-not-workflow.md) — the failure-mode section already names workflow-tool-driven; this document generalizes from "workflow-tool-driven is the failure mode" to "workflow-tool-driven and unattended swarm are the two failure modes; everywhere else composes."
- [`08`](08-the-pr-bottleneck.md) — the continuous-ratification argument is operating-model-independent; this document confirms it.
- [`09`](09-orchestrators-and-project-managers.md) — orchestrators run *across* operating models, routing per task; this document gives that observation a name.
- [`11`](11-should-the-framework-model-the-code.md) — the compose-don't-absorb posture this document applies to operating models is the same posture `11` applied to code-graph tools. The two documents share a method: walk the kernel against an adjacent territory; refuse absorption; expose stable surfaces for composition.
- [`KERNEL.md`](KERNEL.md) — the kernel does not need to change. It is already operating-model-silent. This document affirmatively names the silence as principled and backs it by the wider survey.

## 8. The honest failure mode

This position has a failure mode worth naming.

**Operating-model-agnosticism as framing is wrong if a future operating model genuinely demands kernel changes.** Specifically:

- **A future agentic-swarm pattern that retains ratification but at a layer aiwf does not support.** If swarms develop a stable pattern where the ratification chokepoint is at the *task* layer rather than the milestone layer, the kernel's milestone-as-ratification-unit ([`08`](08-the-pr-bottleneck.md)) might be too coarse. The framework would have to add a finer-grained ratification entity, which would be a kernel change. This document does not address that; it would be future work if and when the pattern stabilizes.
- **A future operating model that genuinely changes what counts as a "durable structural decision."** All thirteen cells the survey covers share the assumption that decisions about software (epics, milestones, ADRs) are recognizably software-shaped. A radical shift in that — work that does not produce code as the primary artifact, or work where the artifact is a model rather than a system — could move the boundary. Out of scope here; flagged.
- **Cross-operating-model interoperability as a real consumer ask.** A team that imports specs from Spec Kit, exports issues to Linear, and runs Cursor day-to-day may want aiwf to coordinate across all three. The framework's current posture (stable surfaces, opt-in render modules) is the right shape for this, but the actual integrations have to be built. The third survey's "task-level routing" finding suggests this need will be widespread; the framework's response is to make integration cheap, not to absorb the integrated tools.

These failure modes are not arguments against the operating-model-agnostic positioning. They are real boundaries the position has to be honest about.

## 9. The position in one paragraph

The arc's earlier framing positioned the framework against Spec-Driven Development because SDD was the loudest discourse when the arc was written. The third survey [`surveys/ai-assisted-operating-models`](surveys/ai-assisted-operating-models.md) showed SDD is one cell in a roughly thirteen-cell landscape of AI-assisted operating models, most of which have no canonical name and many of which are practiced unknowingly. The framework's bet — durable structural-decision state, continuous ratification at state transitions, soft-to-strict tier discipline — does not select for any operating model; the kernel's eight needs are about belief-state, not workflow shape. aiwf is therefore **operating-model-agnostic**: the layer of durable structural decisions underneath whatever operating model a team runs, composing with most of them through stable surfaces and refusing only the two cells where its bet structurally does not fit (workflow-tool-driven as primary substrate, agentic-swarm without human ratification). The lane: *aiwf records the durable structural decisions about software work; aiwf does not prescribe how those decisions are produced.* This is the same restraint [`11`](11-should-the-framework-model-the-code.md) applied to code-graph tools, generalized one level up.

## 10. Open questions

1. **Does the operating-model landscape stabilize, or keep splintering?** The third survey named thirteen cells; the next radar might name fifteen. The framework's posture handles new cells gracefully (compose, don't absorb), but the framing prose has to stay current. Maintenance question, not design question.
2. **Do the two refused cells (workflow-tool-driven primary, unattended swarm) deserve framework integrations anyway?** A team primarily on Linear may want a one-way export from aiwf to Linear; a swarm with ratification chokepoints at milestone boundaries may want first-class swarm integration. Both are conceivable opt-in modules ([`04`](04-governance-provenance-and-the-pre-pr-tier.md) §4) without changing the kernel. Open question whether either earns its keep.
3. **How does the "task-level routing across operating models" insight ([`09`](09-orchestrators-and-project-managers.md), this document §2) shape skill content?** Skills could carry advice about which operating model is appropriate for which task type — *milestones get level-3 background work; ADRs stay pair-programmed; contracts stay human-edited.* Skill content, not engine work; cheap to add; real value. Worth a future skill-level decision.
4. **Should the framework's marketing (such as it is) ever lead with SDD specifically?** The honest answer is no — leading with SDD narrows the audience. The framing in §6 is operating-model-agnostic by design. But there will be readers for whom SDD is the entry point; how to handle them without re-anchoring the bet to SDD is a copy-writing question more than a design question.

---

## In this series

- Previous: [`11 — Should the framework model the code?`](11-should-the-framework-model-the-code.md)
- Next: [`13 — Should aiwf adopt policy as a primitive?`](13-policies-as-primitive.md)
- Related: [`07 — State, not workflow`](07-state-not-workflow.md), [`08 — The PR bottleneck`](08-the-pr-bottleneck.md), [`10 — Spec-based development is waterfall in disguise`](10-spec-based-as-waterfall.md)
- Surveys: [`surveys/understanding-spec-driven-development`](surveys/understanding-spec-driven-development.md), [`surveys/workflow-work-division-and-who-implements`](surveys/workflow-work-division-and-who-implements.md), [`surveys/ai-assisted-operating-models`](surveys/ai-assisted-operating-models.md)
- Synthesis: [working paper](../working-paper.md)
- Reference: [`KERNEL.md`](KERNEL.md)
