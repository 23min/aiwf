# Workflow, Work Division, and Who Implements

*A companion to "Understanding Spec-Driven Development." If that piece argued the term has fragmented, this one argues that each fragment implies a different way of running a project — and the differences matter more than the SDD discourse usually admits.*

---

## What this post is about

The previous post argued that "Spec-Driven Development" covers at least five distinct workflows hiding under one banner: spec-as-prompt, spec-first, spec-anchored, spec-as-source, and spec-as-contract. That distinction was about *what the spec is*. This one is about everything around it — how projects are planned, how work gets sliced, where documentation lives, and who actually writes the code.

These questions are usually treated separately from SDD. I think that's a mistake. The interpretation of SDD a team picks is not just a doc-management choice; it implies a particular shape of lifecycle, a particular way of dividing tasks, and a particular answer to "who implements." Most teams don't realize they're committing to all three when they pick a tool.

The structure here:

1. The five SDD interpretations, expanded into workflow archetypes.
2. Lifecycle and where the documentation lives.
3. Work division — what an "issue" or "story" means when an agent does the implementation.
4. Who implements — the autonomy spectrum from autocomplete to swarm.
5. What the empirical record actually says (and doesn't).
6. Practical implications.

I cite as I go. Bibliography at the end.

---

## The five SDD interpretations as workflow archetypes

Recap from the previous post, with the workflow implication for each made explicit:

**Spec-as-prompt.** The "spec" is a detailed prompt. Workflow: traditional dev cycle + careful prompt-writing. The code is authoritative. Documentation is whatever Confluence/README/Notion you already had, plus maybe a `.cursorrules` or `AGENTS.md`. Work is divided as it always was — by tickets in Jira/Linear/GitHub Issues. Implementation is human-with-assistant: a developer with Cursor, Copilot, or Claude Code in their editor.

**Spec-first (Böckeler level 1).** A structured spec is written first, used to drive one task, then the spec is discarded or left to rot. Workflow: explicit specify-plan-tasks-implement loop, but only for the duration of the change. This is what GitHub Spec Kit and Kiro encode out of the box [1]. Documentation forks: the spec lives in the repo for that change (often in a feature branch), while the broader codebase docs continue elsewhere. Work is divided by "specs," not stories — and a spec usually decomposes into 5–20 numbered tasks. Implementation is human-supervised agent: the developer reviews each generated task.

**Spec-anchored (Böckeler level 2).** The spec persists alongside code and gets versioned with it. Workflow: same specify-plan-tasks-implement, plus an explicit *spec maintenance* step. The harder claim — and the one most current tooling fails at. Documentation centralizes: the spec replaces design docs and ADRs, or sits beside them. Work division becomes spec-deltas (OpenSpec's ADDED/MODIFIED/REMOVED format is the explicit version) [3]. Implementation is still human-supervised agent, but the human's job is now also to *update the spec when reality bites back* — a step almost no team enforces.

**Spec-as-source / spec-as-truth (Böckeler level 3).** The spec is the only artifact humans edit. Workflow: humans edit specs; code is regenerated. Tessl explicitly aspires to this [1]. Documentation collapses: the spec *is* the documentation. Work division stops being about "implementation tasks" and becomes about "spec changes." Implementation is fully delegated to agents — but the human has to trust the regeneration.

**Spec-as-contract.** Formal-methods adjacent. The spec is a verifiable property the code must satisfy. Workflow: contract-first (OpenAPI, Protobuf, AsyncAPI), test-driven, or property-based. Documentation is the contract. Work division is by API surface or property. Implementation can be human, agent, or both — the contract is the gate, not the source.

The thing to notice: these are not just five flavors of "writing a spec." They're five distinct operating models. Each one implies different answers to where docs live, how work is sliced, and who does what.

---

## Where the documentation actually lives

In the world before LLMs, documentation had a roughly stable layout: README at the project root, ADRs in `/docs/adr`, design docs in Confluence or Notion, runbooks somewhere SRE could find them, comments inline in the code. Stale docs were a known problem but a tolerable one — humans could re-read code and infer intent.

The agent era breaks this in two ways. First, agents read all of it indiscriminately, so contradictions between the README and the design doc become bugs. Second, several new documentation types have appeared — `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, Spec Kit's "constitution," Kiro's "steering" files — that target agents specifically. The ThoughtWorks Technology Radar Volume 33 (November 2025) listed "curated shared instructions for software teams" as something to *adopt*, noting that "relying on individual developers to write prompts from scratch is emerging as an anti-pattern" and recommending that teams place these instruction files into the baseline repository used to scaffold new services [10].

Mapping documentation onto the SDD archetypes:

- **Spec-as-prompt:** old docs + an instructions file (`AGENTS.md`, etc.). Authoritative source: code.
- **Spec-first:** old docs + per-feature spec folder (e.g. `specs/001-foo/spec.md, plan.md, tasks.md`). Authoritative source: code, but spec is the briefing for that change.
- **Spec-anchored:** old docs partially absorbed into living specs. Authoritative source: spec for behavior, code for behavior-not-yet-in-the-spec. Conflict resolution becomes a real workflow concern.
- **Spec-as-source:** spec replaces most other documentation. Authoritative source: spec.
- **Spec-as-contract:** contract is its own document, deliberately narrow. Old docs continue alongside.

The boring but important observation: most teams haven't decided which of these they're committing to, so they end up with several at once — `AGENTS.md` *plus* per-feature specs *plus* old Confluence design docs *plus* inline code comments — with no rule for what's authoritative when they conflict. Agents read all of them and average over them. This is a documentation problem nobody named yet.

---

## Work division: what is a "task" when an agent does the work?

In Agile-as-practiced, the unit of work is a story, sized in story points (or t-shirts, or hours, depending on the team). A story has a description, acceptance criteria, and a person assigned to implement it. Story points are a *human-effort* estimate. The whole machinery — sprint planning, velocity, capacity — is built on that.

Agents break the unit. A 5-point story for a senior developer might be 10 minutes of agent time plus 30 minutes of review, or 4 hours of agent time plus a full day of review when the agent goes off the rails. The variance is high and the predictors are different. Familiarity with the code matters less for the agent than for the human; the size of the relevant context window matters more.

What the SDD interpretations do to work division:

**Spec-as-prompt** preserves stories more or less unchanged, just changes what the developer does inside one. Story points still mean something, roughly. This is also where productivity research has produced the most surprising number: METR's July 2025 randomized controlled trial of 16 experienced open-source developers on 246 issues in their own mature repositories found that allowing AI tools made developers 19% *slower*, despite developers estimating beforehand that AI would save them 24% [4]. The effect was strongest exactly where stories looked most "normal" — high developer familiarity, mature codebase, well-understood task. METR has since announced a redesign of the study because of selection effects — many of their developer pool now refuse to work without AI even when paid $50/hour [5] — but the original finding stands as the strongest evidence that "story" as a unit doesn't translate cleanly to AI-augmented work.

**Spec-first** changes the unit. Spec Kit's workflow generates a `tasks.md` with explicit numbered, dependency-ordered tasks marked for parallel execution where possible (`[P]` markers) [1]. A "spec" replaces a story; tasks within the spec replace subtasks. Spec Kit's tutorial explicitly suggests that some teams will write tasks fine-grained enough to be issued one-per-prompt to the agent, then move on to the next [1]. This is a different unit of accounting than a story point — it's closer to "atomic agent action."

**Spec-anchored** exposes a problem. If the spec evolves with the code, work-division has to track both: the change to the spec *and* the code change that implements it. OpenSpec's delta format (ADDED/MODIFIED/REMOVED) is one of the few honest attempts to make spec-deltas a first-class work unit [3]. Most other tools handle this badly — Spec Kit, for instance, creates a branch per spec, which Böckeler points out makes the spec a per-change-request artifact, not a per-feature one [1].

**Spec-as-source** flips the unit entirely. The work item is a spec change. Implementation tasks don't exist as separate things — they're regenerated. This is the most theoretically clean and the least empirically tested. Tessl is the main example, still in private beta as of the previous post [1].

**Spec-as-contract** is the easiest case: the contract change *is* the work item. OpenAPI changes, Protobuf changes, schema changes — all of these have been first-class work units for a decade.

A separate empirical signal cuts across all of these. Faros AI's telemetry on 22,000+ developers found that AI increases pull-request size by ~50% — and DORA's 2025 State of AI-Assisted Software Development report (nearly 5,000 respondents) found that "working in small batches amplifies AI's positive effects" while large-batch work amplifies the downsides [6, 7]. The implication: whatever your work unit is called, agents push it *larger* by default, and the teams getting positive ROI are the ones actively fighting that drift.

---

## Who implements: the autonomy spectrum

The 2026 conversation has converged on a roughly-agreed spectrum, though different sources name it differently. The cleanest published version is Swarmia's five-level taxonomy from March 2026, which I'll use as a reference [9]:

**Level 1 — Assistive.** Inline suggestions in a single file. GitHub Copilot circa 2021. The human writes; the AI predicts. Every keystroke is human-judged.

**Level 2 — In-editor agent.** The agent reads multiple files, plans changes, executes them, but stays inside the human's editing session. Cursor's agent mode, Claude Code in interactive mode. The human reviews each step, can intervene, and decides when to commit.

**Level 3 — Background coding agent.** The agent runs asynchronously outside the editor — Claude Code's headless mode, GitHub Copilot's coding agent (assigned via issue), OpenAI Codex cloud. The human assigns a task, walks away, and reviews a pull request when the agent returns. Anthropic's December 2025 internal-usage study reports that Claude Code now chains together an average of 21 independent tool calls per task without human intervention, up from 9.8 six months earlier — a 116% increase — while the average number of human turns per task dropped 33% [2].

**Level 4 — Autonomous teammate.** The agent picks tasks on its own — flaky-test repair, dependency updates, doc drift. Uber's FlakyGuard is the most-cited documented example [9]. GitHub Agentic Workflows shipped this in technical preview in February 2026 [9]. The human's job is to review the agent's choices, not just its work.

**Level 5 — Agentic swarm.** Multiple agents working together with minimal human supervision. Cursor's FastRender project reportedly used a Planner/Worker/Judge hierarchy with up to 2,000 parallel instances generating over a million lines of code [9]. Most teams are not here, and probably shouldn't be.

Mapping this spectrum onto the SDD archetypes is where the picture gets sharp:

- **Spec-as-prompt** lives at Levels 1–2.
- **Spec-first** typically lives at Levels 2–3. The spec exists precisely to enable Level 3 — give the agent enough context to run autonomously for a while.
- **Spec-anchored** is the natural home of Level 3 and bridges to Level 4. The spec is what makes "the agent picks the next task" tractable, because there's a durable description of what should be true that the agent can compare reality against.
- **Spec-as-source** is essentially a Level 4 design — the spec drives regeneration, the human edits the spec rather than triaging tickets.
- **Spec-as-contract** is orthogonal: a contract enables higher levels of autonomy *for the slice of the system the contract covers*, regardless of how the rest of the system is built.

Two empirical findings about Level 3+ work that don't get cited together but should:

A peer-reviewed empirical study at MSR 2026 ("Behind Agentic Pull Requests") compared agent-authored pull requests (APRs) against human-authored ones (HPRs) using the AIDev dataset. Human intervention occurred less frequently in APRs (52.17%) than in HPRs (83.59%) — but when it did occur, it required *higher* review effort: larger code churn and longer review duration. The taxonomy of intervention types found that 58% of human effort on APRs was "guidance-level" (restricting what the agent does, enforcing project conventions), 21% "decision-level," 17% direct code changes, and 4% operational [8]. The authors' summary: "collaboration with coding agents is shifting developer work from implementation to supervision, guidance and quality control" [8].

Anthropic's own engineers, in the December 2025 study, expressed exactly this shift in their own words. One described work as "70%+ to being a code reviewer/reviser rather than a net-new code writer," and another framed their future role as "taking accountability for the work of 1, 5, or 100 Claudes" [2]. More than half said they could "fully delegate" only 0–20% of their work; the rest required active supervision and validation [2].

---

## What the empirical record actually says

Pulling the threads together. There are now four bodies of evidence worth weighing:

1. **METR's July 2025 RCT (16 OSS developers, 246 issues).** Surprising negative result: AI tools made developers 19% *slower* on issues in their own familiar mature codebases [4]. This is the strongest counter-narrative finding in the literature. Caveats: small N, narrow population (experienced OSS maintainers, not enterprise teams), early-2025 tools (Cursor + Claude 3.5/3.7 Sonnet). METR has since redesigned the study because so many participants now refuse to work without AI [5].

2. **Anthropic's December 2025 internal study (132 engineers + 53 interviews + 200,000 Claude Code transcripts).** Median self-reported productivity: +50%, up from +20% twelve months earlier. Claude in 59% of daily work, up from 28%. Twenty-seven percent of Claude-assisted work was "qualitatively new" — work that wouldn't have been done without it. Most engineers said they could "fully delegate" only 0–20% of their work [2]. Caveats acknowledged in the report: convenience sampling, social-desirability bias, employees of an AI lab, self-reported productivity. Anthropic explicitly cites METR and notes that "the factors METR identified as contributing to lower productivity than expected... closely correspond to the types of tasks our employees said they don't delegate to Claude" [2].

3. **DORA's 2025 State of AI-Assisted Software Development (~5,000 respondents).** Central finding, repeated by every commentator: "AI is an amplifier" — it magnifies existing organizational strengths and weaknesses rather than creating performance on its own. AI adoption correlated with both higher throughput *and* higher instability. Platform quality, value-stream management, and small-batch discipline emerged as the differentiators between teams getting positive ROI and teams making things worse [6, 7].

4. **MSR 2026 mining-challenge paper on agent-authored PRs.** Lower intervention frequency on APRs, but higher cost per intervention, with the bulk of human effort going to guidance and convention-enforcement rather than direct coding [8].

The contradictions are real and instructive. METR says AI slows experienced developers in familiar code; Anthropic says experienced developers report dramatic speedups; DORA says both are right depending on the surrounding system; MSR's PR study says even when the speedup is real, the work has shifted in kind. Anyone telling you "AI makes engineers X% more productive" is selecting one of these and ignoring the others.

The honest synthesis is: AI assistance changes the *shape* of work as much as the speed. Some of that shape change is captured by the SDD interpretation a team adopts (or fails to adopt deliberately). Some of it lives in workflow disciplines — small batches, fast feedback loops, platform quality — that predate AI but become more load-bearing in its presence.

---

## Practical implications

If the workflow/work-division/who-implements decisions are downstream of the SDD interpretation, here are the tradeoffs each archetype implies — stated honestly, not optimistically:

**Spec-as-prompt is realistic and cheap, and probably what most teams should start with.** It assumes nothing about your tooling, doesn't require new docs, doesn't change work division much. The risk is you stagnate at Level 1–2 autonomy and never invest in the disciplines that would let you go higher. METR's slowdown finding is *most* relevant to this archetype.

**Spec-first works well for greenfield features in moderately complex codebases.** It enables Level 3 autonomy for that feature. The risk is the spec rots after the feature ships, you accumulate spec-folders nobody reads, and the next change ignores them. Almost every Spec Kit / Kiro user lives here whether they planned to or not [1].

**Spec-anchored is where most of the value lives, and where most teams fail.** Living specs require discipline that existing tooling doesn't enforce. OpenSpec's delta-format is one of the few tools that takes this seriously [3]; the rest aspire to spec-anchored and deliver spec-first. The honest version requires explicit "update the spec when reality changes" as a workflow step, owned by someone, gated in CI.

**Spec-as-source is research-grade.** Tessl is the cleanest published implementation; it's still in beta. The conceptual problems (non-determinism in regeneration, brownfield import, edit-vs-regenerate ambiguity) are real and unsolved. Don't bet the company on this in 2026.

**Spec-as-contract is the safest of the lot.** It works, it's been working for a decade, and it now composes well with agents. If you have an API or a schema, write the contract first — but recognize that this only solves a slice of your system, not the whole thing.

A few cross-cutting recommendations that emerge from the empirical record rather than from any one SDD interpretation:

- **Pick your autonomy level explicitly per task type, not per team.** The Anthropic study found engineers delegating differently based on task: low-stakes/easily-verifiable/repetitive went to the agent; design and high-stakes work stayed human [2]. This is task-level routing, not team-level.

- **Resist the PR-bloat drift.** AI default behavior is to produce larger PRs. DORA's data is clear: small batches amplify AI's positive effects [6]. Whatever your work unit, you probably need to fight to keep it smaller than the agent wants to make it.

- **Invest in the harness before the autonomy.** ThoughtWorks Volume 34 framed this as "putting coding agents on a leash" — feedforward controls (specs, skills, AGENTS.md) plus feedback controls (deterministic gates, mutation testing, type checkers, fuzzing) [11]. DORA's amplifier finding says the same thing in different words: the platform you put the AI on top of determines whether the AI helps or hurts.

- **Treat documentation hierarchy as a workflow concern, not an afterthought.** Decide what's authoritative when sources conflict. Decide which audience (humans, agents, both) each document targets. Decide who's responsible for each document being current. Almost no one is doing this deliberately, and the cost of not doing it compounds as agents read more of your stack.

---

## What this leaves us with

Three claims, in decreasing order of confidence:

1. **The SDD interpretation a team picks implies a workflow shape, a work-division unit, and an autonomy level.** This is structural, not optional. Teams that pick a tool without understanding which interpretation it encodes are committing to all three by default.

2. **The empirical record on AI-assisted productivity is genuinely mixed, and the contradictions track the SDD interpretation question.** METR's slowdown finding is most damaging to spec-as-prompt; Anthropic's speedup finding is consistent with spec-first/spec-anchored at the senior-engineer level; DORA's amplifier finding says both are right depending on platform maturity. Workflow discipline is doing more of the work than tooling.

3. **The biggest open question is what happens to the work itself.** The MSR PR study and the Anthropic interviews converge on a real shift: less implementation, more supervision, more guidance, more convention-enforcement. The role of "engineer who writes code" is becoming the role of "engineer who reviews code an agent wrote in response to a spec the engineer wrote." Whether that's a good or bad change is genuinely uncertain, and the early-2026 honest answer is "we don't know yet."

The previous post argued that SDD's biggest problem is semantic diffusion — five workflows hiding under one term. This post argues the same diffusion runs all the way through workflow, work division, and implementation. Picking your SDD interpretation explicitly is also picking how your team plans, slices, and ships. Doing all three by accident produces the very dysfunction the DORA report warns about: faster, not better.

---

## References

All URLs verified accessible May 3, 2026.

[1] Birgitta Böckeler, *Understanding Spec-Driven-Development: Kiro, spec-kit, and Tessl*, martinfowler.com, October 15, 2025. https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html

[2] Saffron Huang et al., *How AI Is Transforming Work at Anthropic*, Anthropic, December 2, 2025. https://www.anthropic.com/research/how-ai-is-transforming-work-at-anthropic. (Survey of 132 engineers, 53 interviews, 200,000 Claude Code transcripts. All quoted statistics — including the 21.2 / 9.8 tool-call comparison, the 33% drop in human turns, the 50% reported productivity gain, the 27% "new work" figure, and the "70%+ code reviewer/reviser" quote — are from this primary source.)

[3] *OpenSpec*, ThoughtWorks Technology Radar Volume 34, April 2026. https://www.thoughtworks.com/radar/tools/openspec

[4] Joel Becker, Nate Rush, Beth Barnes, David Rein, *Measuring the Impact of Early-2025 AI on Experienced Open-Source Developer Productivity*, METR, July 10, 2025. Blog: https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/. Paper: https://arxiv.org/abs/2507.09089. (16 developers, 246 tasks, 95% CI [-40%, -2%].)

[5] METR, *We are Changing our Developer Productivity Experiment Design*, February 24, 2026. https://metr.org/blog/2026-02-24-uplift-update/

[6] *Announcing the 2025 DORA Report*, Google Cloud Blog, September 23, 2025. https://cloud.google.com/blog/products/ai-machine-learning/announcing-the-2025-dora-report. Full report: https://dora.dev/dora-report-2025/. (~5,000 respondents.)

[7] Naomi Lurie, *DORA Report 2025 Key Takeaways: AI Impact on Dev Metrics*, Faros AI, September 25, 2025. https://www.faros.ai/blog/key-takeaways-from-the-dora-report-2025. (Cited for the +50% PR-size telemetry finding from Faros's 2026 dataset; Faros is a vendor and the underlying telemetry is proprietary, so flag accordingly.)

[8] *Behind Agentic Pull Requests: An Empirical Study on Developer Interventions in AI Agent-Authored Pull Requests*, MSR 2026 Mining Challenge poster, April 13, 2026. https://2026.msrconf.org/details/msr-2026-mining-challenge/26/Behind-Agentic-Pull-Requests-An-Empirical-Study-on-Developer-Interventions-in-AI-Age. (Used the AIDev dataset; statistics quoted are from the abstract.)

[9] Swarmia five-level autonomy taxonomy (March 2026), as summarized in *How AI Coding Agents Evolved from Autocomplete into Autonomous Pull Request Machines*, SoftwareSeni, April 2026. https://www.softwareseni.com/how-ai-coding-agents-evolved-from-autocomplete-into-autonomous-pull-request-machines/. (Secondary summary; the original Swarmia taxonomy is the underlying primary source.)

[10] *Curated shared instructions for software teams*, ThoughtWorks Technology Radar Volume 33 (Adopt), November 2025. https://www.thoughtworks.com/radar/techniques

[11] *Volume 34* introduction, ThoughtWorks Technology Radar, April 2026 PDF. https://www.thoughtworks.com/content/dam/thoughtworks/documents/radar/2026/04/tr_technology_radar_vol_34_en.pdf

### Background and tooling references (carried over from the prior post)

[12] Marc Brooker, *Spec Driven Development isn't Waterfall*, brooker.co.za, April 9, 2026. https://brooker.co.za/blog/2026/04/09/waterfall-vs-spec.html

[13] François Zaninotto, *Spec-Driven Development: The Waterfall Strikes Back*, Marmelab Blog, November 12, 2025. https://marmelab.com/blog/2025/11/12/spec-driven-development-waterfall-strikes-back.html

[14] GitHub Spec Kit repository. https://github.com/github/spec-kit

[15] Kiro (AWS). https://kiro.dev/

---

## Note on sources

Citation [9] is a secondary summary — the original Swarmia taxonomy (March 2026) is the primary source, but I have not been able to verify the original directly and am citing the SoftwareSeni summary as the accessible representation. Reader should treat the level-numbering as descriptive shorthand, not a load-bearing taxonomy.

Citation [7] cites Faros AI's proprietary telemetry. The +50% PR-size finding is from a vendor with a commercial interest in the answer; treat as suggestive rather than authoritative. The DORA report [6] is the primary source for the underlying claim that small-batch discipline matters.

The METR study [4] and the Anthropic study [2] reach contrasting conclusions about productivity. Both are open about their methodological limitations. I have presented both rather than picking one. The honest reading is that the answer depends heavily on context — codebase familiarity, task type, platform maturity — and that any single-number productivity claim is suspect.
