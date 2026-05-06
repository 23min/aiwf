# AI-Assisted Operating Models: A Field Survey

*A companion to "Understanding Spec-Driven Development" and "Workflow, Work Division, and Who Implements." Those pieces mapped one named practice. This one tries to map the wider landscape — the operating models people are actually running in 2026, most of which are not called Spec-Driven Development and many of which have no name at all.*

---

## Why this post exists

By April 2026, "Spec-Driven Development" is loud enough as a term that any survey of AI-assisted software practice tends to start there. The two prior posts in this small set followed that gravity: Understanding Spec-Driven Development mapped the term's fragmentation; Workflow, Work Division, and Who Implements expanded SDD's interpretations into operating models.

But SDD is one named family among several, and probably not the largest one in raw practitioner count. Many people running AI-assisted teams have never heard the term — or have heard it and don't recognize their own practice in any of its rungs. They are doing something else, and the something-else is the bulk of the field.

This post is the survey that doesn't take SDD as the center. It tries to lay out the operating models a practitioner can plausibly be running in early 2026, what each one assumes, and where the names came from where there are names. It's not exhaustive — the field is moving fast and category lines blur — but it covers the cells that mattered enough to develop their own discourse, tooling, or empirical record.

The structure:

1. The two axes that organize the rest.
2. Operating models, one section each.
3. Hybrids and how the categories blur.
4. What the survey doesn't settle.

I cite as I go. Bibliography at the end. Where I have not been able to find a primary source for a claim, I say so rather than reach.

---

## 1. The two axes

Most operating models sort on two independent questions, neither of which is "do you write a spec?"

**Axis A — Where the durable artifact lives.** What persists between sessions and across hand-offs?

- In a code-shaped artifact (the code itself, a test, a contract, a schema).
- In a prose-shaped artifact (a spec, an issue, an ADR, a memory file).
- In a model-shaped artifact (TLA+, Alloy, Lean, OpenAPI).
- In nothing durable at all — only the code in git survives.

**Axis B — How human judgment is applied.** When and how does the human ratify or correct the agent's work?

- Up-front, before agent execution (waterfall-shaped, big-spec-first).
- At gates (review checkpoints, between phases).
- Continuously (turn-by-turn pair-programming-shaped).
- After-the-fact (the agent runs to completion; the human reviews a PR).
- Almost never (full delegation; the agent merges its own work).

SDD lives at "durable spec artifact + judgment up-front-or-at-gates." Test-driven development lives at "durable test artifact + judgment continuously." Vibe coding lives at "no durable artifact + judgment after-the-fact, often shallow." Issue-driven development lives at "durable issue artifact + judgment after-the-fact." And so on.

The cells aren't strict — most teams hybridize — but they make the survey legible. Each section below names a cell or cluster.

---

## 2. The operating models

### 2.1 Vibe coding

Probably the largest cohort by raw count, almost certainly the smallest by amount written about it. **Andrej Karpathy named the practice on February 2, 2025**, in a now-much-cited X post: *"There's a new kind of coding I call 'vibe coding,' where you fully give in to the vibes, embrace exponentials, and forget that the code even exists"* [1]. The post described accepting whatever the agent produced — diffs, error fixes, "yolo" decisions — without close reading.

The term took off because it named something practitioners were already doing without admitting to. Open Cursor or Claude Code, type a request, accept the diff, run the tests (or skip them), commit, move on. There is no spec, no plan, no constitution, and often no review of what changed beyond a glance at the file tree.

Vibe coding sits at *no durable artifact + judgment after-the-fact, often shallow*. The artifact is the code, identical to the world before LLMs; what changed is that almost none of it was written by reading and almost none of it was reviewed by reading either.

Empirical visibility into vibe coding is poor — practitioners don't usually self-report and the term is partly a self-deprecation. Adjacent work suggests it is widespread among hobbyists and individual developers; less is known about its prevalence inside engineering teams. The Pearce et al. 2022 finding that ~40% of Copilot-completed programs in their test contained vulnerabilities, and the 2025 follow-up by Yan et al. finding LLM-generated vulnerability rates from 9.8% to 42.1% across categories, are the most-cited reasons to be cautious — but neither study isolated "vibe coding" as such [2, 3].

Vibe coding's defining feature is the absence of a durable specification of intent. That makes it the foil against which most other operating models in this survey define themselves.

### 2.2 Pair-programming-style collaboration

A continuous turn-taking pattern: the human steers tactically, the agent produces, both adjust in real time. The "spec" is the rolling conversation; the durable artifact is the code. Closer to XP's pair programming than to any 2025-era named practice, except the pair is a human and an LLM.

This is what most experienced Cursor / Claude Code / GitHub Copilot Chat users actually do, and it is *not* well-named in the discourse. It is sometimes called "AI pair programming" but the term has been used in too many ways to be precise. Aider, an open-source CLI AI pair programmer that predates Copilot Chat, has described its model in roughly these terms since 2023 [4].

It sits at *minimal durable artifact + judgment continuously*. The session has structure — turns, decisions, accepted and rejected diffs — but very little of that structure persists past the session except as commits. Anthropic's December 2025 internal-usage study describes engineers doing exactly this: 53 interviews and 200,000 transcripts, with engineers reporting Claude Code in 59% of daily work and median self-reported productivity at +50% [5]. The study's qualitative finding — that work is shifting "from implementation to supervision, guidance and quality control" — fits this operating model precisely.

The MSR 2026 study of agent-authored pull requests found that 58% of human intervention effort on agent PRs was guidance-level (restricting agent behavior, enforcing project conventions), 21% decision-level, 17% direct code changes [6]. These ratios describe pair-programming-style collaboration as much as anything else: the human is mostly steering and judging, not producing.

### 2.3 Test-driven development with AI assistance

TDD predates LLMs by twenty-five years. **Kent Beck's *Test-Driven Development by Example* (Addison-Wesley, 2003)** institutionalized the discipline: write the failing test first, make it pass, refactor [7]. The composition with LLMs is natural — write the failing test (or have the agent draft one), ask the agent to make it pass, run the test, refactor.

The operating model is *durable test artifact + judgment continuously*. Tests are the spec, in the executable sense; the agent is the producer; the test runner is a deterministic gate.

Beck has been an active and skeptical voice on AI-assisted development through 2025–2026. His January 2026 LinkedIn post (surfaced by Martin Fowler) criticized SDD's "freeze the spec, generate the code" framing as a return to a discredited assumption that nothing useful is learned during implementation [8]. The implied alternative — feedback loops as the load-bearing structure — is essentially the TDD position, updated for LLMs.

Empirically, TDD-with-AI is one of the better-evidenced operating models, partly because the test runner provides ground truth that can be measured. METR's July 2025 RCT of 16 experienced OSS developers on 246 issues did not isolate TDD specifically, but its central finding (AI tools made developers 19% slower in their own mature codebases, despite expecting +24%) has been widely interpreted as a warning about workflows where the loop *isn't* tightly closed [9]. The authors note METR has since redesigned the study because so many of its participant pool now refuse to work without AI tools [10].

### 2.4 Issue-driven (background-agent) development

A unit-of-work-defined-by-the-issue model: the human (or PM, or stakeholder) writes the issue body, assigns it to an agent, walks away, and reviews a pull request when the agent returns. The "spec" is the issue description.

The model became visible in 2025 with several near-simultaneous launches:

- **Devin** (Cognition Labs), announced March 12, 2024 as "the first AI software engineer" [11].
- **GitHub Copilot coding agent**, launched in public preview in May 2025 and reaching general availability later that year, assignable to issues like a human collaborator [12].
- **OpenAI Codex cloud**, the agent product line built on top of OpenAI's models, available through the ChatGPT and developer surfaces from mid-2025 onward [13].

The operating model is *durable issue artifact + judgment after-the-fact*. The issue body has to be detailed enough to brief the agent for an autonomous run; the PR is the surface where human judgment re-enters. Swarmia's March 2026 five-level autonomy taxonomy places this at Level 3 ("background coding agent") [14], which the surveys pulled into earlier posts.

The MSR 2026 study cited above used the AIDev dataset of agent-authored PRs and found that intervention frequency on agent PRs (52.17%) was lower than on human-authored PRs (83.59%) but cost per intervention was higher — larger churn and longer review duration [6]. The paper's framing of the shift, "collaboration with coding agents is shifting developer work from implementation to supervision, guidance and quality control," is the issue-driven operating model's central trade-off.

### 2.5 Plan-execute-review

A pattern, not a tool. Have the agent produce a plan first (no `spec.md`, no constitution, no per-feature folder); review and edit the plan in the chat; then have the agent execute. The plan is ephemeral — discarded or buried in scrollback after the diff lands.

This is what many experienced Cursor and Claude Code users describe doing without any tool prescribing it. Claude Code's "plan mode" and Cursor's agent planning behaviors are tooling acknowledgements of the pattern. There is no canonical written description of plan-execute-review as a distinct practice; it is folkloric.

The operating model is *minimal durable artifact + judgment up-front-and-continuously*. The plan exists long enough to be reviewed but not long enough to rot; the durable artifact is the diff. Closer to spec-first than to vibe coding — but lighter than spec-first because there is no commitment to keep the plan.

This is one of the cells where the lack of a name is genuinely the story. A practice that millions of practitioners run does not have a stable label, and that absence shapes how the discourse misreads them as either "vibe coders" (too informal) or "SDD practitioners" (too formal).

### 2.6 Memory-driven / instruction-driven development

Persistent context, not per-change specs, shapes every session. The "spec" is the long-running rules file, read by every agent invocation, never (or rarely) edited per task.

The dominant artifacts:

- **`.cursorrules`** (Cursor), one of the earliest agent-targeted memory files, 2023–2024.
- **`CLAUDE.md`** (Anthropic Claude Code), released as part of Claude Code's launch in 2024 and continuously expanded since.
- **`AGENTS.md`**, a cross-tool convention promoted from 2025 onward; the agents.md site documents the convention as a vendor-neutral standard for agent instructions [15].

The ThoughtWorks Technology Radar Volume 33 (November 2025) placed *"curated shared instructions for software teams"* in **Adopt**, with the explicit recommendation that teams place these instruction files into the baseline repository used to scaffold new services [16]. The Radar's framing was direct: "relying on individual developers to write prompts from scratch is emerging as an anti-pattern."

The operating model is *durable memory-file artifact + judgment continuously, with the memory as a strong prior*. The memory file does not replace per-change human review; it makes per-change review faster by removing the need to re-establish context.

This is also the home of *constitutional* and *steering* approaches: Spec Kit's "constitution" (immutable project principles) and Kiro's "steering" files are tool-specific instances of the same pattern. The boundary between memory-driven and SDD blurs here — Spec Kit's constitution is memory-driven; its `spec.md` per change is spec-first. Most Spec Kit users are running both at once.

### 2.7 Skill-driven / harness-driven development

Procedures encoded as agent-callable skills, slash commands, subagents, or plugins. The "spec" of how to do recurring work — review a PR, run a security audit, ship a feature, generate a release note — lives as an executable skill, not as a per-change document.

Concrete artifacts:

- **Claude Code subagents and slash commands**, with the Skills feature released by Anthropic in October 2025 as a structured way to package procedures the agent can invoke [17].
- **Cursor commands and rules**, with similar shape.
- Custom **AGENTS.md / CLAUDE.md** sections that script multi-step procedures.
- **MCP (Model Context Protocol) servers** as the lower-level integration surface, released by Anthropic in November 2024 [18] and adopted broadly across vendors through 2025–2026.

The ThoughtWorks Technology Radar Volume 34 (April 2026) named *"harness engineering"* as a parallel emerging practice alongside SDD [19], with feedforward controls (skills, instructions, AGENTS.md) and feedback controls (deterministic gates, mutation testing, type checkers, fuzzing) framed together as "putting coding agents on a leash."

The operating model is *durable skill artifact + judgment continuously, with skills as the procedural memory*. It overlaps memory-driven (skills are a kind of memory) but is distinct in that skills are *executable* — invokable, parameterized, sometimes deterministic — where memory files are descriptive.

This is one of the cells the discourse is converging on as load-bearing. ThoughtWorks Vol 34's introduction explicitly named "harness engineering" alongside "spec-driven development" as a pair of new terms emerging together [19]. Whether the two stay distinct or one absorbs the other is open.

### 2.8 Spec-Driven Development

One named family, treated thoroughly elsewhere. Tool-led pattern: write a specification, agree on it, have an LLM execute it, optionally maintain the spec alongside the code. **GitHub Spec Kit** (open-source CLI launched September 2, 2025), **AWS Kiro** (VS Code-based IDE), **Tessl Framework** (private beta), and **OpenSpec** (delta-format brownfield-oriented) are the main implementations [20, 21, 22, 23]. Birgitta Böckeler's October 2025 review on martinfowler.com is the cleanest published taxonomy [24], and the ThoughtWorks Radar placed SDD in "Assess" in November 2025 [25].

For the full taxonomy, history, and critique, see the companion post *Understanding Spec-Driven Development*. For workflow implications, see *Workflow, Work Division, and Who Implements*.

The operating model is *durable spec artifact + judgment up-front-or-at-gates*. The spec persists for the change (spec-first), for the feature (spec-anchored), or as the source of truth (spec-as-source). Spec-as-contract is a separate cell — see §2.10.

### 2.9 Exploration-first / spike-driven

Build a throwaway prototype to learn what to build; then either continue or restart. The artifact is *meant* to be discarded; the learning is the output. The pattern predates LLMs (XP "spikes," "tracer bullets," lean startup MVPs) and survives well into the AI era because LLMs make the throwaway radically cheaper.

There is no canonical 2025-era written treatment of LLM-amplified exploration as a distinct operating model. It surfaces in adjacent literature: Beck's January 2026 critique of SDD invokes essentially this argument — "you aren't going to learn anything during implementation that would change the specification" is bizarre because of course you are [8]. Marc Brooker's December 2025 piece on "the success of natural language programming" describes a similar shape — iteration on cheap implementations as the way useful abstractions surface [26].

The operating model is *no durable artifact (deliberately) + judgment continuously, with the right to restart from scratch*. The differentiator from vibe coding is intent: the exploration is meant to inform a redesign, not to ship.

### 2.10 Contract-first / API-first

Predates LLMs by a decade and composes well with them. Write the contract first (OpenAPI, Protobuf, AsyncAPI, JSON Schema); generate clients, servers, mocks, and tests from it; have humans or agents implement to the contract.

The operating model is *durable contract artifact + judgment up-front-or-at-gates, at the API surface only*. Inside the implementation, anything goes — vibe coding, pair-programming, TDD — but the contract is the gate. The narrowness is the point: the abstraction is small, the generation rules are deterministic, and the contract is testable mechanically.

This is the "spec-as-contract" rung of the SDD taxonomy [24] but in practice most contract-first practitioners would not describe themselves as doing SDD. They would say "we do API-first design," and they have been doing it since OpenAPI 2.0 (Swagger) in 2014.

### 2.11 Formal-methods-with-AI

A small but real cohort. **TLA+** (Lamport, 1999), **Alloy**, **Lean**, **Dafny** used as the source of truth, sometimes with LLMs translating between informal intent and formal artifact. Marc Brooker has written about this directly, situating formal methods inside a wider "specifications as the upstream artifact" frame and noting that formal statements (Lean, TLA+) compose with informal ones in the same workflow [27].

The operating model is *durable formal-model artifact + judgment up-front, with mechanical proof or model-checking as the gate*. Aerospace, distributed systems, and high-assurance domains have used this for decades; the AI addition is mostly in the translation layer.

### 2.12 Workflow-tool-driven

The tool defines the unit. **Jira**, **Linear**, **GitHub Projects**, **ServiceNow**, **Asana** organize work as stories, issues, tickets, tasks. Agents fill in the implementation. The "spec," such as it is, is the issue or story description; the durable structure is the tool's data model (status, sprint, assignee, dependency).

This is the operating model the previous twenty years of software methodology built tooling for, and it has not gone away. It persists most strongly in larger organizations, regulated industries, and any team where the workflow-shaped tooling is mandatory for non-AI reasons (compliance, audit, multi-team coordination).

The DORA 2025 report's "AI is an amplifier" finding [28, 29] applies here strongly: workflow-tool-driven teams whose underlying discipline is good get amplified positively, those whose underlying discipline is poor get amplified negatively. The operating model itself does not determine outcome.

### 2.13 Multi-agent / agentic-swarm development

The frontier. Multiple agents coordinating with minimal human supervision under planner / worker / judge hierarchies. Cursor's reported FastRender project — up to 2,000 parallel instances generating over a million lines of code — is the most-cited 2026 instance [14]. **BMAD-METHOD** is an open-source orchestration framework that organizes work across multiple agent roles for both code and non-code outputs [30]. Anthropic's December 2025 internal study notes engineers framing their future as "taking accountability for the work of 1, 5, or 100 Claudes" [5].

The operating model is *durable artifacts as inter-agent contracts + judgment at the swarm boundary*. The artifacts are necessary because the agents don't share session state; the human's judgment moves outward to the swarm's outputs rather than its individual moves.

This is mostly research-grade or vendor-internal as of mid-2026. The intersection with the rest of the landscape — does swarm work nest inside SDD, or replace it, or run alongside it? — is unsettled.

---

## 3. Hybrids and where the categories blur

The cells above are illegible if read as exclusive. Most teams run two or three at once.

- A team using Spec Kit (SDD, spec-first) for new features, Claude Code with a `CLAUDE.md` (memory-driven) day-to-day, and OpenAPI specs (contract-first) at service boundaries is running three operating models simultaneously, each at a different layer.
- A solo developer who does plan-execute-review (§2.5) on weekday mornings and vibe codes (§2.1) on Saturday afternoons is running two, by mood.
- A startup that issue-drives bug fixes through Copilot's coding agent (§2.4) and pair-programs new features in Cursor (§2.2) is running two, by task type.

The Anthropic December 2025 study has the cleanest empirical statement of this: engineers reported being able to "fully delegate" only 0–20% of their work, with the remainder requiring active supervision; they routed differently *per task* — low-stakes / easily-verifiable / repetitive went to background agents, design and high-stakes work stayed pair-programmed [5]. Task-level routing across operating models is the modal practice for senior engineers using AI tools heavily.

Three blurred boundaries are worth naming:

**Memory-driven and skill-driven blur** because skills are a kind of memory and memories can be procedural. ThoughtWorks Vol 34's "harness engineering" frame includes both [19], which is probably correct.

**Plan-execute-review and spec-first blur** because a "plan" is a transient spec. The line is whether the artifact is meant to persist after the change ships. Most practitioners do not draw this line cleanly.

**Workflow-tool-driven and issue-driven blur** because issue-driven development uses workflow-tool primitives (the issue, the assignee, the PR). The difference is whether the agent acts on the issue (issue-driven) or the human acts and reports against it (workflow-tool-driven). With the rise of Copilot's coding agent and similar, the boundary is moving rapidly.

---

## 4. What the empirical record actually shows about which models work

Three bodies of evidence cut across all the operating models above:

- **METR's July 2025 RCT** found AI tools made experienced OSS developers 19% *slower* on issues in their familiar mature codebases, with a 95% CI of [-40%, -2%] [9]. The slowdown was measured under conditions closest to vibe coding and pair-programming on familiar code; it does not generalize cleanly to issue-driven or spec-first work.
- **Anthropic's December 2025 internal study** of 132 engineers reported a median +50% productivity gain with Claude Code, up from +20% twelve months earlier; 27% of Claude-assisted work was "qualitatively new" — work that wouldn't have been done without it [5]. The study explicitly cites METR and notes that "the factors METR identified as contributing to lower productivity than expected... closely correspond to the types of tasks our employees said they don't delegate to Claude."
- **DORA's 2025 State of AI-Assisted Software Development**, with ~5,000 respondents, framed AI as an *amplifier*: it magnifies existing organizational strengths and weaknesses rather than creating performance on its own [28, 29]. Platform quality, value-stream management, and small-batch discipline emerged as the differentiators between teams getting positive ROI and teams making things worse. **Faros AI's 2026 telemetry on 22,000+ developers** reported AI increases pull-request size by ~50%, and DORA found small batches amplify positive effects while large batches amplify the downsides [29].

The honest synthesis: no single operating model dominates the evidence, the answer depends heavily on context (codebase familiarity, task type, platform maturity, team discipline), and any single-number productivity claim about "AI-assisted development" is suspect because it is averaging over a dozen different operating models with different cost structures.

The practical implication, repeated by every commentator who has read both METR and Anthropic, is that *task-level routing across operating models* outperforms team-level commitment to one model. Anthropic's study is the strongest evidence for this — the engineers reporting +50% gains were the ones routing aggressively per task [5].

---

## 5. What this leaves us with

Several things are simultaneously true:

1. **The landscape is wider than any single named practice.** Spec-Driven Development, harness engineering, vibe coding, TDD-with-AI, issue-driven, pair-programming, memory-driven, skill-driven, exploration-first, contract-first, formal-methods, workflow-tool-driven, agentic-swarm — at least a dozen distinct cells, with most teams running two or three at once.

2. **Most teams have not consciously chosen which cells they occupy.** They are running whatever their tools encourage, whatever they read about most recently, and whatever fit their preceding habits. ThoughtWorks Vol 34's introduction called this out as "semantic diffusion: the rapid emergence of new terms for evolving practices, often before their meanings have stabilized" [19], naming SDD and harness engineering specifically. The same diffusion runs through the whole landscape.

3. **The empirical record is too thin to rank operating models against each other.** It is rich enough to say that platform discipline, batch size, and task-level routing matter more than the choice of operating model itself. METR vs. Anthropic vs. DORA do not contradict each other; they triangulate that the surrounding system matters more than the named practice.

4. **Several of the cells have no canonical name.** Plan-execute-review, exploration-first-with-LLMs, and pair-programming-with-Claude-Code are widely practiced and barely named. The discourse focuses on the cells that have tooling vendors selling them.

5. **The categories are unstable.** Memory-driven and skill-driven are converging into "harness engineering"; issue-driven and workflow-tool-driven are converging as Copilot-style agents move into the issue surface; agentic-swarm and spec-as-source share aspirations and may or may not stay distinct.

A practitioner reading this survey can do worse than to ask, of their own work: *what is durable, when do I judge, and which cells am I actually running at the same time?* The honest answers usually surprise.

---

## References

URLs verified accessible May 3, 2026 unless otherwise noted.

[1] Andrej Karpathy, X (formerly Twitter), February 2, 2025. https://x.com/karpathy/status/1886192184808149383 (the post that named "vibe coding"). Discussed in many subsequent secondary sources; the original is short and worth reading.

[2] Hammond Pearce, Baleegh Ahmad, Benjamin Tan, Brendan Dolan-Gavitt, Ramesh Karri, *Asleep at the Keyboard? Assessing the Security of GitHub Copilot's Code Contributions*, 43rd IEEE Symposium on Security and Privacy (SP 2022), pp. 754–768. arXiv preprint: https://arxiv.org/abs/2108.09293.

[3] Hao Yan, Swapneel Suhas Vaidya, Xiaokuan Zhang, Ziyu Yao, *Guiding AI to Fix Its Own Flaws: An Empirical Study on LLM-Driven Secure Code Generation*, arXiv:2506.23034, June 28, 2025. https://arxiv.org/abs/2506.23034.

[4] Aider — AI pair programming in your terminal. Open-source project, GitHub: https://github.com/Aider-AI/aider. Active since 2023; documentation describes the pair-programming model directly.

[5] Saffron Huang et al., *How AI Is Transforming Work at Anthropic*, Anthropic, December 2, 2025. https://www.anthropic.com/research/how-ai-is-transforming-work-at-anthropic. Survey of 132 engineers, 53 interviews, 200,000 Claude Code transcripts.

[6] *Behind Agentic Pull Requests: An Empirical Study on Developer Interventions in AI Agent-Authored Pull Requests*, MSR 2026 Mining Challenge poster, April 13, 2026. https://2026.msrconf.org/details/msr-2026-mining-challenge/26/Behind-Agentic-Pull-Requests-An-Empirical-Study-on-Developer-Interventions-in-AI-Age. Used the AIDev dataset.

[7] Kent Beck, *Test Driven Development: By Example*, Addison-Wesley, 2003. ISBN 978-0321146533.

[8] Martin Fowler, *Fragments: January 8*, martinfowler.com, January 8, 2026 — containing Kent Beck's verbatim LinkedIn post critiquing SDD. https://martinfowler.com/fragments/2026-01-08.html. Original Beck post: https://www.linkedin.com/feed/update/urn:li:activity:7413956151144542208/

[9] Joel Becker, Nate Rush, Beth Barnes, David Rein, *Measuring the Impact of Early-2025 AI on Experienced Open-Source Developer Productivity*, METR, July 10, 2025. https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/. Paper: https://arxiv.org/abs/2507.09089.

[10] METR, *We are Changing our Developer Productivity Experiment Design*, February 24, 2026. https://metr.org/blog/2026-02-24-uplift-update/

[11] Cognition Labs, *Introducing Devin, the first AI software engineer*, March 12, 2024. https://cognition.ai/blog/introducing-devin

[12] GitHub, *Coding agent for GitHub Copilot* (public preview May 2025; subsequent GA). https://github.blog/2025-05-19-github-copilot-meet-the-new-coding-agent/ — title and date as published; I have not re-verified the exact URL after the GitHub blog migration in early 2026, treat as a redirected pointer if the canonical URL has moved.

[13] OpenAI Codex (cloud-based agent product line, 2025–). https://openai.com/index/introducing-codex/ — original announcement page; product naming has continued to evolve and several Codex iterations exist; cited for the operating-model cell, not for any specific feature.

[14] Swarmia five-level autonomy taxonomy (March 2026), as summarized in *How AI Coding Agents Evolved from Autocomplete into Autonomous Pull Request Machines*, SoftwareSeni, April 2026. https://www.softwareseni.com/how-ai-coding-agents-evolved-from-autocomplete-into-autonomous-pull-request-machines/. Secondary summary; the original Swarmia taxonomy is the underlying primary source.

[15] *AGENTS.md — A simple, open format for guiding coding agents*. https://agents.md/. Vendor-neutral convention documentation.

[16] *Curated shared instructions for software teams*, ThoughtWorks Technology Radar Volume 33 (Adopt), November 2025. https://www.thoughtworks.com/radar/techniques

[17] Anthropic, *Claude Skills* (released October 2025, formal launch announcement). https://www.anthropic.com/news/skills — citation for the announcement; documentation lives at https://docs.claude.com/en/docs/agents-and-tools/agent-skills/overview.

[18] Anthropic, *Introducing the Model Context Protocol*, November 25, 2024. https://www.anthropic.com/news/model-context-protocol. Specification: https://modelcontextprotocol.io/.

[19] *Volume 34* introduction (with the "semantic diffusion" callout naming SDD and harness engineering), ThoughtWorks Technology Radar, April 2026 PDF. https://www.thoughtworks.com/content/dam/thoughtworks/documents/radar/2026/04/tr_technology_radar_vol_34_en.pdf

[20] Den Delimarsky, *Spec-driven development with AI: Get started with a new open source toolkit*, GitHub Blog, September 2, 2025. https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/

[21] Kiro (AWS). https://kiro.dev/

[22] Tessl Framework — discussed in [24]; private beta as of late 2025.

[23] *OpenSpec*, ThoughtWorks Technology Radar Volume 34, April 2026. https://www.thoughtworks.com/radar/tools/openspec

[24] Birgitta Böckeler, *Understanding Spec-Driven-Development: Kiro, spec-kit, and Tessl*, martinfowler.com, October 15, 2025. https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html

[25] *Spec-driven development*, ThoughtWorks Technology Radar Volume 33, blip published November 5, 2025. https://www.thoughtworks.com/radar/techniques/spec-driven-development

[26] Marc Brooker, *On the success of 'natural language programming'*, brooker.co.za, December 16, 2025. https://brooker.co.za/blog/2025/12/16/natural-language.html

[27] Marc Brooker, *Spec Driven Development isn't Waterfall*, brooker.co.za, April 9, 2026. https://brooker.co.za/blog/2026/04/09/waterfall-vs-spec.html. The piece situates formal methods (Lean, TLA+) inside a wider specification-driven frame.

[28] *Announcing the 2025 DORA Report*, Google Cloud Blog, September 23, 2025. https://cloud.google.com/blog/products/ai-machine-learning/announcing-the-2025-dora-report. Full report: https://dora.dev/dora-report-2025/. ~5,000 respondents.

[29] Naomi Lurie, *DORA Report 2025 Key Takeaways: AI Impact on Dev Metrics*, Faros AI, September 25, 2025. https://www.faros.ai/blog/key-takeaways-from-the-dora-report-2025. Carries the +50% PR-size telemetry finding from Faros's 2026 dataset (vendor; treat as suggestive rather than authoritative for that specific number).

[30] BMAD-METHOD (open-source agentic orchestration framework). Repository: https://github.com/bmadcode/BMAD-METHOD.

---

## Note on sources

Several of the operating models discussed here (plan-execute-review, exploration-first-with-LLMs, pair-programming-with-Claude-Code) do not have canonical published descriptions; they are practitioner folklore. Where I have not cited a primary source for a description, the description is synthesized from observed practice and from adjacent published work, and should be read as descriptive rather than authoritative.

Citations [12] and [13] point to vendor announcement pages whose URLs have been moving through 2025–2026 as the products evolved. Where the link does not resolve, the operating-model cell described is the durable claim; the specific product instance is illustrative.

The Karpathy "vibe coding" post [1] is a single-tweet primary source; the term's definition has been elaborated in many secondary discussions since, but the original is the canonical reference for the term's coinage.

The METR study [9, 10] and the Anthropic study [5] reach contrasting conclusions about productivity. Both are open about their methodological limitations. The DORA report [28, 29] is presented as the synthesis that resolves the apparent contradiction by naming context as the determining variable. Readers should weigh all three rather than pick one.
