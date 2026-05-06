# Understanding Spec-Driven Development

*An attempt to map a term that has run ahead of its own definition.*

---

## Why this post exists

Spec-Driven Development (SDD) went from obscure phrase to industry chatter in roughly twelve months. GitHub's Spec Kit has 88,000 stars on its repository as of May 2026 [19]. AWS built an IDE (Kiro) around the idea [20]. Martin Fowler's site published a careful review [4]. The ThoughtWorks Technology Radar placed it in "Assess" in November 2025 [5]. And yet, if you ask five practitioners what SDD *is*, you'll get five different workflows, three different philosophies, and at least one person who quietly means "I wrote a detailed prompt."

The disagreement isn't a sign the topic is unserious. It's the actual shape of the topic. This post tries to do four things:

1. Trace where SDD came from (it did not appear in 2025).
2. Lay out a taxonomy of what people mean today.
3. Survey the tooling landscape against that taxonomy.
4. Examine why teams adopt SDD, why some abandon it, and what the strongest cases on both sides actually argue.

I cite as I go. A bibliography sits at the end, and every citation is to a primary source where one exists.

---

## A short history: SDD did not start with LLMs

The current conversation behaves as if "spec drives code" is a 2025 invention. It isn't. There are at least four traditions feeding into the modern usage, and each carries baggage worth knowing.

**Formal methods (1970s–present).** Z notation, the B method, Event-B, VDM, TLA+, and friends are mathematical notations for specifying system behavior, often paired with refinement-based proofs that an implementation satisfies the spec. These have a real but narrow track record: aerospace, rail signaling, medical devices. The B method, for example, was used in transport automation systems in Paris and São Paulo. The cost — expertise, tooling, ceremony — has kept formal methods out of mainstream practice for decades [14].

**Model-Driven Development / Model-Driven Architecture (1990s–2000s).** MDA proposed that you'd describe systems in UML or a domain-specific language, then transform models into running code via generators. Birgitta Böckeler — who has direct industry experience with MDD — observes that the models in MDD were essentially specs, just expressed in custom UML or textual DSLs rather than natural language, and teams built code generators to turn those into code. MDD never took off for business applications because it sat at an awkward abstraction level and created too much overhead and constraint [4]. France and Rumpe's 2007 ICSE roadmap paper *Model-Driven Development of Complex Software* argued that "full realizations of the MDE vision may not be possible in the near to medium-term primarily because of the wicked problems involved" [15]. This matters: SDD's strongest claim — "spec is source, code is artifact" — is the same claim MDD made, with LLMs replacing the deterministic code generator. The lessons from MDD's failure are directly applicable.

**Test-Driven and Behavior-Driven Development (1999–2010).** Kent Beck's *Test-Driven Development by Example* (Addison-Wesley, 2003) and Dan North's BDD work institutionalized "write the spec of behavior first, then make it pass." This is *spec-first* in the broad sense, but the spec is executable test code, not a markdown document [16]. Modern SDD advocates routinely invoke TDD and BDD as ancestors — and Beck himself has objected to that lineage, which we'll get to.

**Contract-first APIs (2010s).** OpenAPI/Swagger, Protobuf, AsyncAPI, JSON Schema. You write the contract; you generate clients, servers, and tests from it. This is genuine, working, deterministic spec-as-source — for the narrow domain of API surfaces. The narrowness is the point: it works because the abstraction is small and the generation rules are deterministic.

**The LLM moment (2024–2025).** The catalyst that pushed SDD into the spotlight was OpenAI engineer Sean Grove's keynote *The New Code* at the AI Engineer World's Fair on June 4–5, 2025 in San Francisco [1]. Grove argued that prompt engineering produces ephemeral artifacts — developers keep generated code and discard the prompts. In his words: *"we keep the generated code and we delete the prompt. And this feels like a little bit like you shred the source and then you very carefully version control the binary"* [1, transcript]. He used OpenAI's Model Spec — a markdown document defining intended model behaviors, open-sourced February 12, 2025 [17] — as an example of a living, versioned specification. GitHub's Spec Kit launched on September 2, 2025 [3]. Kiro and Tessl arrived around the same time. By November 5, 2025, the ThoughtWorks Technology Radar had placed SDD in "Assess" [5]. By April 2026, ThoughtWorks Radar Volume 34 had moved on to discuss specific tools (Spec Kit and OpenSpec) under broader headings about coding-agent harnesses [21, 22].

So: the term is new, the underlying ideas are forty years old, and the failure modes are well-documented in those older traditions.

---

## The taxonomy: what people actually mean

The cleanest published taxonomy comes from Birgitta Böckeler at Thoughtworks [4]. She identifies three implementation levels:

- **Spec-first**: a well thought-out spec is written first, then used in the AI-assisted workflow for the task at hand.
- **Spec-anchored**: the spec is kept after the task is complete and used for evolution and maintenance of the feature.
- **Spec-as-source**: the spec is the main source file over time; only the spec is edited by the human; the code is regenerated.

These are not competing philosophies — they are increasing levels of investment, each addressing a failure mode of the previous one [4]. Böckeler observes that all SDD definitions she has found are at least spec-first, but not all strive to be spec-anchored or spec-as-source, and the maintenance strategy is often left vague.

That taxonomy is good, but in practice the term has fragmented further. Here is a finer cut, with Böckeler's three levels embedded:

**Spec-as-prompt.** The spec is a detailed prompt. You write it, run the agent, then live in the code. The "spec" is scaffolding, often discarded. This is what most casual practitioners mean. Böckeler notes she has heard "spec" used basically as a synonym for "detailed prompt" [4]. This sits below Böckeler's spec-first level — it is spec-shaped, not spec-driven.

**Spec-first (Böckeler level 1).** A structured spec is written first and used in the workflow for one task. This is where most teams adopting Spec Kit or Kiro actually live, regardless of marketing.

**Spec-anchored (Böckeler level 2).** The spec persists alongside the code, gets versioned, and is updated as the system evolves. This is the aspirational target of most current tooling. Böckeler observes that Spec Kit aspires to this — calling specs "living, executable artifacts that evolve with the project" [3] — but the way Spec Kit creates a branch per spec suggests in practice it treats specs as living artifacts only for the lifetime of a change request, not the lifetime of a feature [4].

**Spec-as-source / spec-as-truth (Böckeler level 3).** The spec is the only artifact humans edit. Code is regenerated. Tessl is the only tool Böckeler reviews that explicitly aspires to this level, with code files marked "GENERATED FROM SPEC – DO NOT EDIT" [4]. This is also where the strongest claims about "specifications as the new code" land.

**Spec-as-contract.** Formal-methods adjacent. The spec is a verifiable property the code must satisfy. TLA+, Dafny, property-based testing. Predates LLMs entirely. Marc Brooker explicitly includes this in his framing: specifications can be free-form natural language, structured (RFC 2119, EARS), or pull in exact statements (Lean, TLA+) when needed [7].

The fragmentation matters because each rung has different failure modes, different tooling needs, and different claims to credibility. If you don't know which one you are practicing, you are paying the cost of the higher level and getting the benefit of the lower one.

---

## The tooling landscape

A non-exhaustive map of what exists as of mid-2026, sorted roughly by where each tool sits on the taxonomy:

**GitHub Spec Kit** (open source, MIT license, launched September 2, 2025) [3, 19]. CLI that scaffolds a workflow of *Constitution → Specify → Plan → Tasks → Implement*, working across Copilot, Claude Code, Gemini CLI, Cursor, and many others (the supported-agents list as of April 2026 includes 25+ agents) [19]. Its memory-bank concept is called a "constitution" — a set of immutable project principles that govern subsequent development. Each workflow step uses checklists in markdown files as a definition-of-done [4]. Sits at spec-first; aspires to spec-anchored.

**Kiro** (AWS, launched 2025) [20]. VS Code-based IDE. Workflow is Requirements → Design → Tasks, with each step represented by a single markdown document. Requirements use user-story format with GIVEN/WHEN/THEN acceptance criteria. Memory bank called "steering." Currently spec-first; spec-anchored is more aspirational than implemented [4].

**Tessl Framework** (private beta as of late 2025) [4]. The only major tool that explicitly aspires to spec-as-source — generated code is marked "GENERATED FROM SPEC – DO NOT EDIT" and there is currently a 1:1 mapping between spec and code file. Closest to the MDA dream, with LLMs as the code generator.

**OpenSpec** (open source) [22]. Change-management oriented. Uses a delta format (ADDED, MODIFIED, REMOVED) explicitly designed for brownfield work. Listed by ThoughtWorks Radar Volume 34 in April 2026 as worth assessing because, unlike Spec Kit and Kiro which are better suited to greenfield, OpenSpec's "focus on spec deltas rather than defining a complete specification upfront" makes it well-suited for existing systems [22].

**BMAD-METHOD** (open source). Multi-agent orchestration framework. Describes itself as "Breakthrough Method of Agile AI-Driven Development." Broader than just code — applies to creative writing, business strategy.

**.cursorrules / Claude.md / AGENTS.md.** The lightweight end. Project-level rules files that some practitioners now call "specs." Functionally: spec-as-prompt with persistence.

The pattern: a spectrum from "lightweight rules file" to "MDA-style spec-as-source," with the heaviest tools making the strongest claims and carrying the most overhead.

---

## Why teams fail with it (and walk away)

The critiques are concrete and consistent. I will group them.

### The bureaucratic-overhead problem

Böckeler tried Kiro on a small bug fix and found that the tool turned the bug into "4 'user stories' with a total of 16 acceptance criteria, including gems like 'User story: As a developer, I want the transformation function to handle edge cases gracefully, so that the system remains robust when new category formats are introduced.'" She tried Spec Kit on a 3–5 point story and concluded she could have implemented the feature with plain AI-assisted coding in the same time it took to run and review the spec-kit results, while feeling more in control [4].

François Zaninotto's "Waterfall Strikes Back" piece documents the same pattern. A real example using Spec Kit to add current-date display to a time-tracking app produced 8 markdown files and 1,300 lines of text [9]. He summarizes: SDD "produces too much text, especially in the design phase" and "developers spend most of their time reading long Markdown files, hunting for basic mistakes hidden in overly verbose, expert-sounding prose" [9].

### The brownfield problem

Zaninotto reports that "SDD shines when starting a new project from scratch, but as the application grows, the specs miss the point more often and slow development. For large existing codebases, SDD is mostly unusable" [9]. ThoughtWorks Radar Volume 34 implicitly confirms this by recommending OpenSpec for brownfield work specifically because "many SDD frameworks (e.g., GitHub Spec Kit) or Agentic Skills workflows (e.g., Superpowers) are better suited to greenfield projects than brownfield ones" [22]. This is the same pattern that limited MDD: the abstraction works on toy problems and falls apart on real ones [15].

### The spec drift problem

Stale design docs mislead the next engineer who reads them; stale specs mislead agents that don't know any better — and agents will execute a plan that no longer matches reality, confidently, without flagging anything is wrong. This is the *spec-anchored aspiration* failing in practice: tools claim living specs but most teams don't have the discipline to maintain them [10].

### The non-determinism problem

A compiler is a deterministic function from source to binary. LLM spec-execution is not. The same specification produces different implementations across different runs — varying architectural choices, data structures, and error handling. Böckeler observed this directly when generating code multiple times from the same Tessl spec, noting "I have seen the non-determinism in action" [4]. This cuts against the "spec is source, code is artifact" framing in a way enthusiasts often hand-wave.

The empirical record on LLM code quality reinforces this. Pearce et al.'s study at IEEE Symposium on Security and Privacy 2022 generated 1,689 programs across 89 scenarios using GitHub Copilot and found approximately 40% contained vulnerabilities [24]. A 2025 follow-up by Yan, Vaidya, Zhang, and Yao found "most LLMs generate vulnerable code at rates ranging from 9.8% to 42.1% across diverse vulnerabilities" [25]. The point isn't that this kills SDD — it's that you cannot wave away the gap between spec and generated code as if it were a compilation step.

### The Beck critique

The most prominent critique came from Kent Beck, originally posted to LinkedIn and surfaced by Martin Fowler in *Fragments* on January 8, 2026 [13]. Beck's words exactly:

> The descriptions of Spec-Driven development that I have seen emphasize writing the whole specification before implementation. This encodes the (to me bizarre) assumption that you aren't going to learn anything during implementation that would change the specification. I've heard this story so many times told so many ways by well-meaning folks–if only we could get the specification "right", the rest of this would be easy. [13]

Fowler, agreeing, connects this to feedback as a core XP value: "When Kent defined Extreme Programming, he made *feedback* one of its four core values. It strikes me that the key to making the full use of AI in software development is how to use it to accelerate the feedback loops" [13]. The critique maps most directly to spec-as-source workflows where humans never touch code.

### The waterfall echo

Zaninotto puts the Beck critique more bluntly: SDD reminds him of the Waterfall model, which required massive documentation before coding so developers could simply translate specifications into code. He invokes Brooks' *No Silver Bullet* (IEEE Computer, April 1987) to argue that software development is fundamentally a non-deterministic process and planning doesn't eliminate uncertainty [9, 18].

### The context-blindness and double-review problems

Zaninotto: SDD agents "discover context via text search and file navigation. They often miss existing functions that need updates, so reviews by functional and technical experts are still required." And: "The technical specification already contains code. Developers must review this code before running it, and since there will still be bugs, they'll need to review the final implementation too. As a result, review time doubles" [9].

### The false-control problem

Even with elaborate files, templates, prompts, workflows, and checklists, agents frequently don't follow all the instructions. Böckeler reports: Spec Kit's research step generated good descriptions of existing code, but the agent then ignored that these were existing classes and regenerated them as duplicates [4]. Larger context windows don't guarantee AI properly picks up on everything in them.

---

## The defense

The strongest case for SDD comes from Marc Brooker, VP and Distinguished Engineer at AWS who has worked closely with the Kiro team [7]. His central argument is that critics are attacking a strawman.

Brooker's framing: SDD "isn't about pulling designs *up-front*, it's about pulling designs *up*. Making specifications explicit, versioned, living artifacts that the implementation of the software flows from, rather than static artifacts." He notes that "software specifications are complex, dynamically changing, internally conflicting, and invariably incomplete. In specification driven development, the specification is the thing being iterated on, rather than the implementation. The iteration cycle is the same as before, but potentially much quicker because of the accelerating effect of AI" [7].

He also addresses the formality question directly: specifications can be free-form natural language, structured (RFC 2119, EARS), or pull in exact mathematical statements (Lean, TLA+) when needed. The spec is the upstream source for most changes, keeping in sync with implementation by being the place changes start [7].

The Grove version is more philosophical. Grove argues that "code itself is actually a lossy projection from the specification. In the same way that if you were to take a compiled C binary and decompile it, you wouldn't get nice comments and well-named variables. You would have to work backwards" [1, transcript]. Even good code, on this view, doesn't embody all the intentions and values; you have to infer the ultimate goal. The spec is what you should have been versioning all along; you've just been versioning the build output.

Both arguments are strongest where Beck and Zaninotto are weakest in attacking — not "freeze the spec, generate the code" but "raise the level of abstraction at which you do iterative development." The disagreement may be less about SDD itself and more about whether current *tools* genuinely support that iterative version, or whether they re-encode waterfall under an iterative banner.

---

## What this leaves us with

Several things are simultaneously true:

1. **The underlying observation is real.** Coding agents perform measurably better when given structured context than when given vague prompts. Some form of spec-first thinking is now table stakes.

2. **The strongest version of SDD — spec-as-source — has the worst track record.** MDD made the same claim with deterministic generators and failed [15]. LLMs add capability but also non-determinism.

3. **Most practitioners are running spec-first workflows but borrowing credibility from spec-as-source rhetoric.** The ThoughtWorks Radar itself flagged this in its Volume 34 introduction, calling out "semantic diffusion: the rapid emergence of new terms for evolving practices, often before their meanings have stabilized," with "spec-driven development and harness engineering" as specific examples [21].

4. **Brownfield work breaks most current tools.** OpenSpec's delta-format is one of the few responses that takes this seriously [22].

5. **The waterfall critique is partially fair and partially a strawman.** It is fair against literal Big Design Up Front uses of SDD; it is a strawman against Brooker's "pulling up, not up-front" version. The honest answer is "it depends which workflow you actually run" [7, 9, 13].

6. **There is a missing middle.** Lightweight, iterative, brownfield-aware spec workflows exist (OpenSpec, Cursor rules, AGENTS.md) but get less attention than the heavyweight tools whose marketing makes the boldest claims.

The most useful thing a practitioner can do is decide *which rung of the ladder* they are actually on, pay the cost of that rung honestly, and stop arguing as if there is one thing called SDD. The most useful thing a critic can do is target the specific rung — most published critiques attack spec-as-source, but most practiced SDD is spec-first or spec-as-prompt, where the critique mostly misses.

---

## References

All URLs verified as accessible on May 3, 2026.

### Primary sources — origin and definition

[1] Sean Grove, *The New Code — Specifications as the Fundamental Unit of AI-Era Programming*, AI Engineer World's Fair, San Francisco, June 4–5, 2025. Video: https://www.youtube.com/watch?v=8rABwKRsec4. (Quotes used in this post are from a third-party transcript at https://my.infocaptor.com/hub/summaries/ai-engineer/the-new-code-sean-grove-openai-8rABwKRsec4 — there is no official OpenAI transcript.)

[2] Deepak Babu Piskala, *Spec-Driven Development: From Code to Contract in the Age of AI Coding Assistants*, arXiv:2602.00180, January 30, 2026. Submitted to AIWare 2026. https://arxiv.org/abs/2602.00180

[3] Den Delimarsky, *Spec-driven development with AI: Get started with a new open source toolkit*, GitHub Blog, September 2, 2025. https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/

### Taxonomy and current-state analysis

[4] Birgitta Böckeler, *Understanding Spec-Driven-Development: Kiro, spec-kit, and Tessl*, martinfowler.com, October 15, 2025. https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html

[5] *Spec-driven development*, ThoughtWorks Technology Radar Volume 33, blip published November 5, 2025. https://www.thoughtworks.com/radar/techniques/spec-driven-development

[6] ThoughtWorks Technology Radar archive (for accessing volumes including Vol 33). https://www.thoughtworks.com/radar/archive

### Defense / advocacy

[7] Marc Brooker, *Spec Driven Development isn't Waterfall*, brooker.co.za, April 9, 2026. https://brooker.co.za/blog/2026/04/09/waterfall-vs-spec.html

[8] Marc Brooker, *On the success of 'natural language programming'*, brooker.co.za, December 16, 2025. https://brooker.co.za/blog/2025/12/16/natural-language.html

### Critique

[9] François Zaninotto, *Spec-Driven Development: The Waterfall Strikes Back*, Marmelab Blog, November 12, 2025. https://marmelab.com/blog/2025/11/12/spec-driven-development-waterfall-strikes-back.html

[10] Augment Code engineering, *What spec-driven development gets wrong*, February 20, 2026. https://www.augmentcode.com/blog/what-spec-driven-development-gets-wrong (vendor blog, included as a representative spec-drift critique)

[11] Isoform, *The Limits of Spec-Driven Development*, November 25, 2025. https://isoform.ai/blog/the-limits-of-spec-driven-development (vendor blog, included as a representative critique)

[12] Arcturus Labs, *Why Spec-Driven Development Breaks at Scale (And How to Fix It)*, October 17, 2025. http://arcturus-labs.com/blog/2025/10/17/why-spec-driven-development-breaks-at-scale-and-how-to-fix-it/

[13] Martin Fowler, *Fragments: January 8*, martinfowler.com, January 8, 2026 — containing Beck's verbatim LinkedIn post. https://martinfowler.com/fragments/2026-01-08.html. Original Beck post: https://www.linkedin.com/feed/update/urn:li:activity:7413956151144542208/

### Historical antecedents

[14] *Formal methods*, Wikipedia, accessed May 3, 2026. https://en.wikipedia.org/wiki/Formal_methods

[15] Robert France and Bernhard Rumpe, *Model-Driven Development of Complex Software: A Research Roadmap*, in *2007 Future of Software Engineering* (FOSE '07), IEEE, May 2007, pp. 37–54. DOI: 10.1109/FOSE.2007.14. https://dl.acm.org/doi/10.1109/FOSE.2007.14

[16] Kent Beck, *Test Driven Development: By Example*, Addison-Wesley, 2003. ISBN 978-0321146533.

[17] OpenAI, *OpenAI Model Spec* (open-sourced February 12, 2025). Repository: https://github.com/openai/model_spec. Current rendered version: https://model-spec.openai.com/

[18] Frederick P. Brooks, Jr., *No Silver Bullet — Essence and Accidents of Software Engineering*, IEEE Computer, Vol. 20, No. 4, April 1987, pp. 10–19. (Originally published in *Information Processing 86*, IFIP Tenth World Computing Conference, 1986.) Open tech-report version: https://www.cs.unc.edu/techreports/86-020.pdf

### Tooling references

[19] GitHub Spec Kit repository. https://github.com/github/spec-kit (88,000 stars, 7,600 forks as of May 3, 2026; latest release v0.7.0 dated April 14, 2026)

[20] Kiro (AWS). https://kiro.dev/

[21] *Volume 34* introduction (with the "semantic diffusion" callout), ThoughtWorks Technology Radar, April 2026 PDF. https://www.thoughtworks.com/content/dam/thoughtworks/documents/radar/2026/04/tr_technology_radar_vol_34_en.pdf

[22] *OpenSpec*, ThoughtWorks Technology Radar Volume 34, April 2026. https://www.thoughtworks.com/radar/tools/openspec

[23] *GitHub Spec Kit*, ThoughtWorks Technology Radar Volume 34 (Languages and Frameworks), April 2026. https://www.thoughtworks.com/radar/languages-and-frameworks/github-spec-kit

### Empirical context (LLM code quality)

[24] Hammond Pearce, Baleegh Ahmad, Benjamin Tan, Brendan Dolan-Gavitt, Ramesh Karri, *Asleep at the Keyboard? Assessing the Security of GitHub Copilot's Code Contributions*, 43rd IEEE Symposium on Security and Privacy (SP 2022), San Francisco, May 23, 2022, pp. 754–768. DOI: 10.1109/SP46214.2022.9833571. arXiv preprint: https://arxiv.org/abs/2108.09293. (1,689 programs across 89 scenarios; ~40% contained vulnerabilities.)

[25] Hao Yan, Swapneel Suhas Vaidya, Xiaokuan Zhang, Ziyu Yao, *Guiding AI to Fix Its Own Flaws: An Empirical Study on LLM-Driven Secure Code Generation*, arXiv:2506.23034, June 28, 2025. https://arxiv.org/abs/2506.23034. (Vulnerability rates from 9.8% to 42.1% across diverse vulnerabilities.)

---

## Note on sources

Citations [10], [11], and [12] are vendor blogs whose authors have a commercial stake in promoting alternatives to the tools being critiqued. They are included because they document common practitioner failure modes, not as neutral evidence. Their factual claims (specifically about agents marking work done without doing it, file-count bloat, and brownfield breakage) are corroborated by the more independent sources [4] and [9].

Citation [1] is a YouTube video. The transcript I quote from is third-party (InfoCaptor, generated automatically); there is no official OpenAI transcript. Quotes have been spot-checked against the video timestamps.

Citation [13] reproduces Beck's verbatim LinkedIn post via Fowler's site. The LinkedIn post is the primary source; Fowler's framing is secondary commentary. Both links are provided.

Citations [16] and [18] are pre-internet primary sources without convenient direct online versions. The IEEE Xplore/ACM/Amazon links would require a paid subscription; the UNC tech report PDF [18] is the closest open primary version of the Brooks paper. The Beck book is in print; no open source.

Two claims that appeared in earlier secondary sources have been **dropped from this version because I could not verify them against primary literature**: a "110,000 surviving AI-introduced issues" figure from an Augment Code marketing post, and a "SonarQube analysis of five LLMs" claim attributed to an arXiv August 2025 paper that I could not locate. Neither claim was load-bearing for any argument made above.

---

*This post deliberately presents both the strongest case for SDD and the strongest case against it. Where I have taken a position, it is that the term has fragmented to the point where blanket statements ("SDD works" / "SDD is waterfall") tell you more about which sub-version the speaker has in mind than about the practice itself. The honest version of any SDD discussion starts by naming which rung of the ladder is in play.*
