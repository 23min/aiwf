# A small framework for AI-assisted project tracking in git: a working paper

> **Status:** defended-position
> **Hypothesis:** A small state model — markdown files as canonical, git as the time machine, validation at chokepoints, continuous ratification at state transitions — is the right shape for project tracking in software teams where LLMs participate in every role; workflow is one render among many, not the substrate.
> **Audience:** anyone evaluating the framework's current position, or considering whether the position generalizes to their team.
> **Premise:** synthesizes the position established across the [research arc](https://proliminal.net/theses/ai-workflow-research/) `00`–`09`.
> **Tags:** #thesis #aiwf #state-model #git #hitl #software-development

---

## Abstract

AI-assisted software development changes what it means to track work. Plans, decisions, and structural state must remain accessible to a stateless assistant across sessions while still co-evolving with the code they describe. The natural impulse is to import event-sourced architectures into the repository — an append-only event log, a derived projection, hash-verified consistency — but this layering fights git's branching model in ways that grow worse with scale. We argue, on evidence from a working PoC and from the trajectory of related tools, that the right shape at the scale most teams actually operate at is materially smaller. Markdown files in the repository are sufficient as the canonical state; `git log` is sufficient as the audit trail; the residual job is a small validating engine and a few verbs that produce well-shaped commits. To this technical position we add three claims that the LLM era forces: that **state is canonical and workflow is a render**, not the other way around; that **continuous ratification at state transitions** replaces batched post-hoc PR review and makes humans more leveraged, not less; and that the role economy is shifting from *production* to *judgment*, with orchestrators emerging as a craft drawn from many disciplines. The framework's competitive position is *not* universal — regulated industries, large enterprises with formal handoff chains, and operationally pipeline-shaped work need workflow tools, and the framework is honest about this. Where it applies, the framework is portable, vendor-neutral, and small. We close with what's not yet settled: identity across forks, cross-host skill fidelity, the formal treatment of branch-divergent assistant rules, and whether the position survives empirical use at scale.

---

## Who this is for

This is a working notebook of one developer thinking through what AI-assisted software work needs, and building the smallest plausible answer to find out. It is aimed at solo practitioners and small teams — where the author feels the friction directly and where the framework, if it ships, has the best chance of fitting. Adjacent readers are welcome: technical project managers and tech leads, spec-driven-development practitioners, tool-builders working on neighboring problems, and engineers earlier in their AI adoption who want a serious treatment of what changes when AI is structurally in the loop.

It is not aimed at large enterprise teams with mature workflow tooling, regulated work where process is the compliance artifact, or anyone looking for a finished product to adopt. The focus on solo and small-team work is a deliberate scoping of the experiment, not a claim about where the interesting problems live; the contribution is the thinking, done in public, with the PoC as a concrete attempt to act on it.

---

## 1. Introduction

The medium of software work is changing. AI assistants now plan, design, code, and decide alongside humans on tasks measured in days, weeks, or months. They do this from a stateless starting point each session, often on partial context, and without a consistent place to consult what was already settled. The artifacts of long-horizon work — epics, milestones, decisions, gaps, contracts — must be both human-editable and machine-readable, both individually authored and collectively coherent.

This paper is the synthesis of a project that found its problem after committing to a solution. The repository started with a confident architectural ambition — an event-sourced kernel, hash-verified projections, monotonic IDs, RFC 8785 canonicalization — captured in [`docs/archive/architecture.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/archive/architecture.md). The architecture was elaborate; what it lacked was an articulated problem the elaboration was solving. The [research arc](https://proliminal.net/theses/ai-workflow-research/) that followed is the work of discovering the problem — and walking the architecture back in light of what discovery revealed. The earlier [working paper](https://proliminal.net/theses/working-paper/) settled the technical position. This second working paper extends it to cover what the LLM era forces: workflow is dissolving, ratification is compounding, roles are shifting, and the framework's competitive position must name these explicitly.

The contribution is not a novel algorithm or a new data structure. It is an argument that the right shape for this class of system has been pulled, by collective intuition, toward more sophistication than it deserves; that a simpler answer better serves the case where AI assistants are doing the heavy lifting on long-horizon software work; and that the framework's value, properly framed, is not workflow management but state-keeping with humans ratifying at chokepoints the LLM cannot skip.

---

## 2. The problem

We define the problem space concretely, by symptom. The following recur across the projects we have observed and have personally lived:

- **The AI re-plans from scratch each session** because it cannot find the current plan, or finds an out-of-date one.
- **Renaming or rescoping a milestone silently breaks references** elsewhere in the repository.
- **Switching branches changes the rules** the assistant believes it should follow, often without the human noticing.
- **Decisions get re-litigated** because no one — human or AI — knows whether something has already been settled.
- **Plans drift faster than they're recorded**, and structural state quietly desynchronizes from the code it claims to describe.
- **PR queues grow** as LLM-amplified production outpaces human review; reviewers either rubber-stamp or work longer hours.

These symptoms generalize into a small number of needs an AI-aware planning framework must serve: recording planning state; expressing relationships among planning items; supporting their evolution under pressure; keeping history honest; validating consistency mechanically; generating human-readable views; coordinating AI behavior; and surviving parallel work by humans and assistants. The full list, with cross-cutting properties, appears in [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md). What the symptoms have in common is that they are all failures of the team's *converging belief state* to remain coherent across sessions, branches, and roles. The framework's job is to keep that belief state coherent.

What does *not* appear on the list of needs is equally important. *Maintaining a totally-ordered event log* is not a need; it is one possible mechanism. *Keeping a derived graph projection* is not a need; it is a possible optimization. *Hash-chaining* is not a need; it is one way to detect drift. The conflation of needs with mechanisms is, we will argue, a primary source of overdesign in this space.

---

## 3. Related work

The space adjacent to this problem has grown rapidly. We summarize it in seven clusters, distinguishing what has been formalized in peer-reviewed work from what is currently presented in product documentation and industry essays.

### 3.1 Spec-driven and AI-coding tooling

A wave of tools (GitHub's *Spec Kit*; AWS's *Kiro*; Tessl; Block's *Goose*; Sourcegraph's Amp; the Claude Code, Cursor, Continue, and Aider IDE-integrated assistants) treats the planning specification as a first-class repository artifact. They share a pattern: markdown-shaped specs, agent reads-and-updates conventions, and per-repository configuration. None has, to our knowledge, formally addressed the merge semantics of those artifacts under git branching; the field is too young.

### 3.2 Memory and context systems for coding agents

The "memory bank" pattern (popularized in the Cline community), Cursor's project memory, Aider's repo-map, Continue's context store, and Anthropic's auto-memory in Claude Code each give the assistant a curated, persistent state to consult across sessions. They are not project-tracking systems per se but they share substrate concerns with one — what to write, how to read, how to keep it from rotting.

### 3.3 Per-repository rule files

Every major AI coding host now reads a per-repository rules file: `CLAUDE.md`, `.cursorrules` (and the newer `.cursor/rules/`), `CONVENTIONS.md` for Aider, `.github/copilot-instructions.md` for GitHub Copilot, `.continuerc.json` for Continue, `.windsurfrules` for Windsurf, and the proposed cross-tool `AGENTS.md` standard. All are git-tracked; all are branch-divergent; none has a formal merge story. Empirical evidence that small prompt changes shift agent behavior measurably is established in the in-context-learning literature [Lu et al., 2022], which underlines that the divergence is not benign.

### 3.4 External project management

Mature project-management tools (Linear, Jira, GitHub Projects, Asana, Shortcut, Height) solve multi-user editing, notifications, permissions, and rich querying — at the cost of keeping the planning state out of the AI's working set and decoupling it from the bisectability of code history. The trade-off is real and addressed in §10.

### 3.5 Mergeable structured state

Outside the AI-coding space, several lineages have studied how structured data can survive concurrent edits:

- **Conflict-free Replicated Data Types** [Shapiro et al., 2011], with the JSON-shaped Automerge implementation [Kleppmann & Beresford, 2017] being the most directly applicable to project-tracking state.
- **Patch theory**, via Pijul and the categorical-patches treatment of Mimram and Di Giusto [2013], offering a mathematically rigorous account of when merges are well-defined.
- **Operational Transformation** [Ellis & Gibbs, 1989], the model behind Google Docs and similar collaborative editors.
- **Versioned databases** — Dolt for SQL, TerminusDB for graph data, XTDB and Datomic for bitemporal queries, Irmin for git-like data structures — each demonstrating that branch-and-merge semantics can be lifted from text into structured domains.
- **The local-first software movement** [Kleppmann et al., 2019], which articulates the constraint set the framework operates under: collaborative, offline-tolerant, multi-writer, no required server.
- **Edits-as-substrate work**, recently exemplified by *Denicek* [Petricek, 2025], representing programs as series of edits over a document — relevant to representation choice if not to location.

### 3.6 Ontology- and graph-based agent memory

A distinct lineage argues that AI agents need *structured* memory — a defined ontology plus a graph-based store that can represent typed entities, named relationships, hierarchical organization, and temporal dynamics. The reference is Yang et al. [2026], surveying 200+ works and articulating a two-layer model: knowledge memory (ontological scaffolding) and experience memory (instances). Production systems in this lineage include Graphiti for temporal-graph memory.

This lineage is adjacent to ours, not the same. It addresses *agent cognitive memory* — what an autonomous agent retains across sessions and tasks. We address *project artifact persistence* — epics, milestones, decisions, gaps, contracts that humans and assistants both consult because *the team* needs them as durable structure. The two share substrate concerns (typed relationships matter; flat similarity-search is insufficient) but they are not the same problem.

### 3.7 Multiplayer agentic workspaces

GitHub's research prototype *Ace* [GitHub Next, 2026] and adjacent visions — Maggie Appleton's "Zero Alignment" essay being the clearest articulation — argue that single-player coding agents create a coordination problem at the team level: "one developer, two dozen agents, zero alignment." Their answer is real-time multiplayer chat plus cloud microVMs plus shared agent access, with the *session* (chat + plan + commits) becoming the unit of work rather than the PR. We engage with this position in §6 and §10. Our framework takes the opposite bet: rather than collapsing collaboration into one hosted workspace, it treats markdown-in-git as a federation hub with role-shaped doors, with each role's tool of choice as a participant. ACE's diagnosis (multi-role alignment is the bottleneck) is correct; our prescription is portable, async-first, and vendor-neutral, where ACE's is hosted, real-time, and vendor-coupled.

---

## 4. Why structured state in git resists totally-ordered event logs

The most promising-looking import from the event-sourcing tradition is to keep an append-only `events.jsonl` file in the repository, alongside a derived projection (`graph.json`), with each event carrying a sequence number and a hash of the post-state. A reader can replay the events to reconstruct the projection and detect drift in O(1) by hash comparison. Within a single linear history, this is an excellent design. It does not survive git branching.

Two branches that have diverged from a common ancestor will, with high probability, both append events to `events.jsonl`. Git's textual three-way merge will, in the absence of conflicts on overlapping lines, concatenate the appends. The result is a syntactically valid JSONL file with three structural problems: sequence numbers collide, post-state hash chains break, and total ordering is destroyed. Identifier allocation suffers in parallel — a monotonic per-kind counter is structurally a multi-master replication primitive, and concurrent writers will allocate the same id without coordination.

These are not bugs that careful engineering can fix within the chosen representation. They are consequences of the encoding. Total ordering of writes is a property of *file content*; file content is what git merges; therefore total ordering cannot be a property of the merged file. Standard responses — custom git merge drivers (analogous to lockfile merges in npm, yarn, cargo), branch-aware identifiers, demoting the hash chain to be branch-local — are all viable, and individually and together costlier than the failure mode they address. They reproduce, in repository-tracked text, what git already provides at the file-tree level.

The CRDT and local-first literatures are, in our reading, the right foundation for this problem. Git already behaves like a per-file CRDT for line-structured text. Layering a non-CRDT abstraction (a totally-ordered hash-chained log) over a mergeable substrate is the structural error. Either the substrate is lifted (replace the log with an Automerge document), or the abstraction is lowered (drop the linearity claim and let git's per-file merge stand). The middle path — keep the linearity claim and mediate at merge time — survives but pays an ongoing cost. For the project shapes most actual users target, the cost is not justified. The abstraction should be lowered. The full argument is in [`00-fighting-git`](https://proliminal.net/theses/fighting-git/).

---

## 5. The walk-back

The repository's earlier design proposed an event-sourced kernel with hash-verified projections. The research arc walked it back. The relevant abandonments:

- **The append-only `events.jsonl`** is removed. Its job — recording mutations with actor, timestamp, and intent — is done by `git log` augmented with structured commit trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`).
- **The `graph.json` projection** is removed. Its read-model role is filled by markdown frontmatter, parsed on demand. Drift detection is replaced by validation: there is no second store to drift from.
- **RFC 8785 canonicalization and SHA-256 hashing of the projection** are removed. Without a second store, there is nothing to canonicalize against.
- **The monotonic ID counter** is replaced by a tree-scan allocator (pick `max(seen ids of this kind) + 1` at write time) plus an explicit collision-resolution verb. Branch-aware coordination is unnecessary at the target scale.
- **Trace-first writes as a permanent ledger** are replaced by trace-first writes within a single verb's execution: a journal file in `.ai-repo/journal/` (gitignored) records intent before file edits and is deleted on commit success. Crash recovery is local; no permanent ledger.
- **The closed entity vocabulary as YAML-defined boundary contracts** is replaced, for the PoC, by hardcoded Go structs for six kinds.

What survives is the principle that the engine is invocable without an AI assistant, the principle that closed-set vocabularies belong in code rather than in prose, and the principle that errors are findings rather than parse failures. These were never the costly part.

---

## 6. The position

We claim five load-bearing things. The first three are technical; the fourth is conceptual; the fifth is positioning.

**(1) Markdown files are the source of truth.** Each entity (epic, milestone, ADR, gap, decision, contract) lives in a markdown file with YAML frontmatter (typed structural fields) and a prose body (narrative content). The frontmatter is what the engine reads; the body is opaque to the engine. Files merge cleanly under git's per-file three-way merge in the common case; structurally meaningful conflicts surface as either YAML-conflict markers (which a human resolves) or post-merge findings reported by the validator.

**(2) Git is the time machine.** History is `git log`, filtered per entity by structured commit trailer. The engine provides a thin renderer (`aiwf history <id>`) that reads the log; it does not maintain a parallel ledger. Bisectability is git's bisectability. Audit is git's audit. Provenance ties to commits because commits are the units of mutation.

**(3) The framework's residual job is small.** Six things are needed: a tree loader that parses frontmatter into typed structs; a validator that reports findings on the loaded tree; a small set of verbs that perform multi-file mutations atomically (one git commit per verb, with structured trailers); identity ergonomics that survive rename and collision; a skill-materialization step that gives AI hosts the conventions in their native shape; and a pre-push git hook that runs the validator. The PoC implements this as a small Go binary — small enough to read end-to-end in an afternoon — built in a focused week or two.

**(4) The framework is a state model, not a workflow engine.** This claim is the load-bearing reframe `07` contributed. Workflow has been the dominant organizing concept for software work for several decades because human specialist throughput was the bottleneck and queue-based stations described the work accurately. LLMs collapse production time at every station for the work LLMs handle well, which thins the throughput-bound queues that defined stages, which makes "where is this in the pipeline?" stop describing the work as well as it used to. What's left is *state* — the converging picture of what the team currently believes about what they're building. State has answers regardless of process shape, regardless of role specialization, regardless of how LLM-heavy the team's work is. The framework models state as canonical and lets workflow be a derived view, opt-in, computed on demand by teams that find a Kanban board or Gantt chart useful. This inverts the usual tool design (Jira, Linear, Asana model workflow as primary), and the inversion is what makes the framework durable in the LLM era.

**(5) Continuous ratification at state transitions replaces batched post-hoc review.** This claim is what `08` contributed. The PR bottleneck most teams feel in 2026 is not a tool problem; it is a *process* problem caused by LLMs producing faster than humans can review batched diffs. The answer is not better review tooling but a different process: humans ratify state transitions throughout the work — small, contextual, fast acts of judgment — rather than reading thousand-line diffs at the end. The unit of human attention shifts from issue (a production unit, sized to human throughput) to milestone (a ratification unit, sized to what a human can hold in their head). Git remains the audit trail of edits; the framework adds an audit trail of judgments via structured commit trailers and the state-transition verbs that produce them. The closest competing organizing shape is spec-based development (Spec Kit, Kiro, Tessl); we argue at length in [`10-spec-based-as-waterfall`](https://proliminal.net/theses/spec-based-as-waterfall/) that spec-based encodes the same artifact-handoff dependency graph as waterfall and that continuous ratification is structurally different because it ratifies state transitions on a single converging artifact rather than handoffs between artifacts.

The compressed statement of what the framework is, useful for any reader who needs to understand quickly:

> `aiwf` is a state model for the durable structural decisions of software work, designed for teams where LLMs participate in every role. It is not a workflow engine, a project-management tool, or a collaboration workspace. It is the place where the answer to *"what is currently true about this work"* lives, in a form that humans, LLMs, CI, and other tools can read and write through a small typed vocabulary.

What this is not: a project-management replacement, a state-management abstraction layered on git, a knowledge graph, a multi-agent orchestrator, or a code-execution agent. It is a small, opinionated, deterministic shim between humans and AI assistants doing project-tracking work in markdown files inside a git repository.

### A note on the name

The framework is called `aiwf`, originally an acronym for *ai-workflow*. The name reflects what we set out to build — a workflow-shaped tool for AI-assisted teams — and the journey produced something different: a state model with workflow as one render among many. Claim (4) above argues, at length, that workflow is the wrong organizing concept for the kinds of work the framework targets. The name is now in tension with the position.

We have nonetheless kept the name. Software project names often diverge from their etymology — `git` did not stay "the stupid content tracker," `vim` is not meaningfully "Vi IMproved" anymore — and what matters is what the tool does, not what the letters once stood for. From this point on, `aiwf` is treated as an opaque four-letter name. Documentation, skill descriptions, and user-facing materials describe the framework as "a small state model for AI-assisted project tracking" rather than "an AI workflow framework." The four letters stay; the tagline shifts. The cost of renaming binaries, URLs, repository, and accumulated documentation does not pay for itself, given that any reader who follows the documentation will quickly learn what the framework actually is. We acknowledge the tension and move on.

---

## 7. The layered location-of-truth model

State in this system lives at six layers, each chosen to minimize the friction of its update pattern:

| Layer | Where | Why |
|---|---|---|
| Engine binary (`aiwf`) | Machine-installed via `go install` (or future package manager) | Standard tool distribution, same as `git` itself |
| Per-project policy (`aiwf.yaml`, `aiwf.lock`) | In the repository, git-tracked | Team-shared, CI-readable, travels with clone |
| Per-project planning state (`work/`, `docs/adr/`) | In the repository, git-tracked | Co-evolves with code; bisectable |
| Project-specific skills | In the repository, git-tracked | Team's own conventions evolve with the project |
| Framework + third-party skill content | Bundled with binary or registry-cached | Independent versioning |
| Per-developer config | `~/.config/aiwf/` | Personal; no team coupling |
| Materialized skill adapters + runtime cache | In the repository but gitignored | Composed from the layers above, regenerated only on explicit `aiwf init` / `aiwf update` |

The materialization invariant is load-bearing. The adapter set the AI host actually reads is regenerated on explicit user action, not implicitly on `git checkout`. This decouples assistant behavior from branch state in the common case, while still letting branches genuinely diverge in skill source when that is the work being done. It mirrors how npm packages, language toolchains, and IDE plugins behave: branch switches do not silently change the tooling. The full argument and evidence are in [`05-where-state-lives`](https://proliminal.net/theses/where-state-lives/).

---

## 8. The chokepoint argument and continuous ratification

The framework provides skills that document what the AI should do. Skills are advisory text; the AI may invoke them, may overlook them, or may operate in a mode that does not load them. Any guarantee that depends on the assistant remembering to follow a skill is not a guarantee — it is hope dressed up as policy. This is the most operationally consequential conclusion of the research arc, and it shapes everything else.

The framework's actual guarantees come from `aiwf check` running as a `pre-push` git hook and again on the PR in CI. The hook is what turns "the assistant is supposed to keep references valid" into "broken references cannot reach the remote." The validator is mechanical, deterministic, and fast; it operates without AI judgment; it produces findings any caller (human, AI, CI script) can act on. This is the chokepoint where the framework imposes itself on the work, and it is intentionally narrow: validation, not approval; reporting, not rewriting.

Sitting between the chokepoint and the human is **continuous ratification**. As work proceeds — a milestone is being shaped, a contract is being drafted, an ADR is being argued — the LLM proposes mutations, and the human ratifies them at state transitions. The framework's verbs produce one git commit per ratification, with a structured trailer recording who ratified, with what role, on what artifact. The commit log becomes the durable audit trail of judgments — distinct from, and complementary to, the audit trail of edits that git already provides.

This positioning resolves the soft/hard tension between exploratory planning ("plans are clay") and durable record ("main is a museum") that motivates much of the research arc. Studio behavior — branches, drafts, free iteration — is unconstrained. *Workshop* behavior — the moment between studio and museum — is where the validator runs and where most reconciliation work belongs, while the AI is still in conversation and local tools are available. *Museum* behavior — what is on the published main — is sealed by the chokepoint having done its job upstream. The framework does not impose museum semantics on the studio.

The corollary that follows is the inversion of the standard "AI replaces humans" frame: when LLMs produce and humans ratify, and ratification scales better than production, **HITL gets stronger, not weaker, in the LLM era**. The leverage of each human's "yes" or "no" goes up, not down. *Teams that figure out how to put humans at the right ratification points, with the right cadence, with the right artifacts, move faster than teams that either drown in PRs or hand the keys to the LLMs entirely.* The broader observation — that LLM-era abundance inverts Mies van der Rohe's "less is more" credo into something closer to "more output, fewer artifacts" — is sketched in the satellite note [more is much more](https://proliminal.net/notes/more-is-much-more/).

---

## 9. Roles in the LLM era

The role economy in LLM-amplified teams is shifting along an axis the discourse has not yet fully named. *Production* roles — those whose value was in producing artifacts at human throughput — lose differentiation as LLMs produce passable first-draft artifacts at every stage. *Judgment* roles — those whose value is in deciding what's worth keeping — compound in value because every LLM session needs ratification.

A new role emerges from this shift, currently called **orchestrator** in the discourse. Stripped of marketing, an orchestrator decides what work to delegate to which LLM, designs the artifacts the LLMs read and write (specs, ADRs, contracts, milestone scopes), inserts ratification chokepoints, manages context across sessions, integrates outputs, and applies real-time quality judgment. Notice the shape: orchestration is *technical-flavored* — it requires understanding what the LLM is doing well enough to ratify the output, not just tracking that work is being done.

This is where the orchestrator role partly overlaps with traditional project management and partly diverges from it. The judgment subset of PM work — scope, prioritization, sequencing, trade-offs — maps onto orchestration cleanly. PMs who already had this combined with technical fluency can step in. PMs whose value was in administrative coordination (status reports, ceremonies, schedule wrangling) find that the LLM era reduces demand for that work (the artifacts handle it) while *increasing* demand for judgment they may not have built. Orchestrators are equally drawn from senior engineers who developed product sense, designers who learned systems thinking, tech leads who took on more strategic scope. The role is filled by whoever can do both halves of the job; the discipline of origin matters less than the combination.

The framework's job, against this backdrop, is *to give orchestrators (whatever they're called) better artifacts to ratify*. Structured planning state. Stable identifiers. Clear ratification chokepoints. An audit trail of judgments. The PoC is a concrete attempt at this. The full role analysis is in [`09-orchestrators-and-project-managers`](https://proliminal.net/theses/orchestrators-and-project-managers/).

---

## 10. Where this applies and where it doesn't

The framework's competitive position is *not* universal. The honest scope statement, derived from the research arc:

The framework fits cleanly for:

- **Solo developers** using LLMs heavily, where the bottleneck is keeping multiple LLM sessions consistent across what used to be roles.
- **Small teams** doing trunk-based product work, where production is fast and the PR-as-batched-review ceremony has become friction.
- **Greenfield and small-to-medium brownfield projects** where the team can adopt new conventions without negotiating across organizational boundaries.
- **Research-flavored engineering** where direction shifts often and re-entry into prior decisions is routine.

It fits less well for:

- **Regulated industries** where the order of attestation is a legal fact (FDA submissions, SOX evidence chains, ISO 27001 audit trails, 21 CFR Part 11). The pipeline is real and isn't dissolving; the framework can store the state these pipelines mutate but cannot replace the workflow tool that orchestrates them.
- **Large enterprises with formal handoff chains** between specialist groups, where workflow tools (Jira with custom workflows, ServiceNow, Camunda) describe the work better than state models because the bottleneck is genuinely throughput across organizational boundaries.
- **Operations and incident response** where sequence of action is load-bearing (page → triage → mitigate → resolve → postmortem) and assignment matters in real-time.
- **Sales, CRM, and other genuinely pipeline-shaped work** where stages are stations and throughput matters. Salesforce was not a category error.

We are not arguing PRs are dead, workflow tools are obsolete, or PMs are unnecessary. We are arguing that *for the kinds of work where PRs are felt as ceremony, where workflow tools feel like overhead, and where LLMs have already eroded the queue-based shape*, a state model with continuous ratification is a better fit. The PR-as-bottleneck pain is the signal that you're in the first kind of work, not the second. Outside that zone, the framework's position is wrong, and we say so plainly.

---

## 11. The PoC as evidence

A position is only as good as the implementation that tests it. The PoC, on the `poc/aiwf-v3` branch, is a single Go binary `aiwf` with the following surface:

- Six entity kinds — epic, milestone, ADR, gap, decision, contract — each with a closed status set and a one-function transition rule.
- Stable identifiers (`E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`) preserved across rename, cancel, and collision; `aiwf reallocate` handles collisions deterministically.
- Mutating verbs (`add`, `promote`, `cancel`, `rename`, `reallocate`) each producing one git commit with a structured trailer.
- A read verb (`aiwf check`) that walks the working tree and reports findings (id collisions, unresolved references, illegal transitions, missing frontmatter, contract artifacts that don't exist on disk, cycles).
- A history verb (`aiwf history <id>`) reading `git log` filtered by entity trailer.
- Skill content embedded in the binary, materialized to `.claude/skills/wf-*` and gitignored.
- A pre-push hook installed by `aiwf init` that runs `aiwf check`.

The PoC is not a finished framework. It is an existence proof that the position can be implemented in a few sessions of focused work, the resulting tool delivers the guarantees we claim it does, and the surface remains small enough that an evaluator can read the whole thing in an afternoon. Where it grows from here is a question for real use to answer. The build plan is in [`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/).

---

## 12. Open questions

The position does not settle several questions; we name them so future work can address them honestly.

- **Identity across repository forks and transfers.** A repository's `git config remote.origin.url` is fragile under forks and ownership transfers; absolute paths are fragile under moves; in-repo identifier files require human-maintained coordination. The right choice for cross-repository project identity remains open.
- **Cross-host skill fidelity.** The framework materializes skill adapters per AI host; how much fidelity is preserved across Claude Code, Cursor, Copilot, Continue, and Aider is a function of those hosts' adapter formats and is not under the framework's control. A common skill-content format is desirable; whether it can be agreed across vendors is not certain.
- **Branch-divergent assistant rules.** Per-repository rule files diverge across branches in every major AI coding tool. We have proposed a materialization invariant that mitigates this; we have not formally treated the problem. The literature [Lu et al., 2022; Bai et al., 2022] suggests the divergence is not benign, but the formalization is open.
- **Multi-machine without a server.** A single developer working across machines, or a small team without shared infrastructure, currently relies on `git push`/`pull` for state synchronization. Local-first techniques (CRDT-merged state, Automerge-based stores) could complement this; the trade-offs at this scale are not yet measured.
- **Compliance and regulated industries.** The PoC's provenance is sufficient for engineering hygiene but has not been evaluated against specific regulatory frameworks. Whether the structured-commit-trailer model can carry that load, or whether regulated environments need a richer provenance store, is open.
- **Multi-role collaboration across non-engineering tools.** PMs, designers, and stakeholders generally do not edit markdown in IDEs. The federation-hub framing in §3.7 implies an MCP server, a web view, and role-specific entry points (email forms, Slack slash commands), so each role's tool reads and writes the same canonical artifacts. The shape is sketched but the implementation is open.
- **Sessions as a first-class concept.** A session — one human plus their LLM working for an hour, producing publishable output — is the natural unit between "the agent edits a file" and "a commit lands on main." Whether this needs to be a framework entity, or remains a runtime concept of an MCP server, is open.
- **A CRDT-modeled metadata layer.** A small subset of the framework's state (the identifier registry, the reference graph, status registers over partial orders) is naturally CRDT-shaped. The PoC does not implement this; it uses path-prefix collision detection and a manual reallocation verb. The case for a small CRDT layer remains, especially as concurrency increases.
- **Graduation to graph-substrate memory at long-horizon scale.** Yang et al. [2026] argue that long-horizon, multi-agent, regulated-domain settings need ontology-plus-graph memory architectures with explicit knowledge / experience / temporal layers. Our framework is sized below that threshold; whether it can graduate cleanly to such a substrate (e.g., emit triples derivable from frontmatter, sync with an external knowledge graph) or whether projects in that range need a different framework entirely is open.
- **Whether the position survives empirical use.** The framework hasn't been used at scale yet. The state-not-workflow position might turn out to be defensible only at the scale and shape the PoC targets, and might need revision at the boundaries. Test by using.

---

## 13. Conclusion

We set out to design an AI-aware project-tracking framework and ended up with one substantially smaller than we initially proposed. The walk-back was not a retreat from ambition; it was a realization that the ambition was layered on the wrong substrate. Git already provides much of what we wanted to build — total ordering within a branch, atomicity at the commit, attributable history, branching as first-class divergence — and the residual job for an AI-aware framework is correspondingly modest.

The LLM era forces three further reframes that the technical position alone could not anticipate. **State is canonical and workflow is a render** — workflow tools as a category are correct for genuine pipelines and increasingly mismatched for the kinds of software work LLMs are reshaping. **Continuous ratification replaces batched review** — humans say yes or no at state transitions throughout the work, rather than reading thousand-line diffs at the end, and HITL gets stronger rather than weaker as LLMs amplify production. **Roles shift from production to judgment** — orchestration is the craft that emerges, drawn from many disciplines, defined by the combination of strategic judgment with technical fluency in LLM-produced artifacts.

The framework's value, properly framed, is the smallest set of mechanical guarantees that lets a forgetful AI and a busy human not lose track of what is planned, decided, and done — and lets them ratify the answer continuously, in small acts of judgment, with a durable record of who decided what. At the scale most projects actually operate at, the smallest set is six entity kinds, stable identifiers, a pre-push validator, a structured-commit history reader, and skill materialization that does not change under the assistant's feet. Build that. Use it. Let real friction tell you what to add.

Where the framework is wrong — regulated industries, large enterprises, ops, sales, anywhere the pipeline is genuinely real and not throughput-scaffolding — we say so plainly. Where it is right, the position is portable, vendor-neutral, and small. That tradeoff is the bet.

---

## References

### Peer-reviewed and academic

- Bai, Y., et al. (2022). *Constitutional AI: Harmlessness from AI Feedback.* Anthropic. arXiv:2212.08073.
- Ellis, C. A., & Gibbs, S. J. (1989). *Concurrent Operational Transformation in Distributed Editors.* Proceedings of SIGMOD '89.
- Gilbert, S., & Lynch, N. (2002). *Brewer's Conjecture and the Feasibility of Consistent, Available, Partition-Tolerant Web Services.* ACM SIGACT News, 33(2).
- Kleppmann, M., & Beresford, A. R. (2017). *A Conflict-Free Replicated JSON Datatype.* IEEE Transactions on Parallel and Distributed Systems, 28(10), 2733–2746.
- Kleppmann, M., Wiggins, A., van Hardenberg, P., & McGranaghan, M. (2019). *Local-First Software: You Own Your Data, in Spite of the Cloud.* Onward! 2019.
- Lamport, L. (1978). *Time, Clocks, and the Ordering of Events in a Distributed System.* Communications of the ACM, 21(7), 558–565.
- Lu, Y., Bartolo, M., Moore, A., Riedel, S., & Stenetorp, P. (2022). *Fantastically Ordered Prompts and Where to Find Them: Overcoming Few-Shot Prompt Order Sensitivity.* Proceedings of ACL 2022.
- Mimram, S., & Di Giusto, C. (2013). *A Categorical Theory of Patches.* Electronic Notes in Theoretical Computer Science, 298.
- Petricek, T. (2025). *Denicek: Computational Substrate for Document-Oriented End-User Programming.* UIST '25.
- Shapiro, M., Preguiça, N., Baquero, C., & Zawirski, M. (2011). *Conflict-Free Replicated Data Types.* SSS 2011.
- Terry, D. B., et al. (1995). *Managing Update Conflicts in Bayou, a Weakly Connected Replicated Storage System.* SOSP '95.
- Yang, et al. (2026). *Graph-based Agent Memory: Taxonomy, Techniques, and Applications.*

### Industry essays and commentary

- Appleton, M. (2026). *One Developer, Two Dozen Agents, Zero Alignment.* https://maggieappleton.com/zero-alignment
- GitHub Next (2026). *Ace: a multiplayer agentic workspace.* Research preview.
- Kleppmann, M. (2017). *Designing Data-Intensive Applications.* O'Reilly.
- Nygard, M. (2011). *Documenting Architecture Decisions.* Industry essay; the seed of the ADR convention.

### Tools and products (cited by URL or name; not peer-reviewed)

- *Automerge.* https://automerge.org/ — JSON CRDT library and runtime.
- *Yjs.* https://yjs.dev/ — text and structured CRDTs, used in collaborative editors.
- *Loro.* https://loro.dev/ — newer CRDT runtime.
- *Pijul.* https://pijul.org/ — version control system based on patch theory.
- *Dolt.* https://www.dolthub.com/ — Git-style branching and merging for SQL databases.
- *TerminusDB.* — branched graph database.
- *Irmin.* https://irmin.org/ — Git-like distributed data store for OCaml/MirageOS.
- *Datomic, XTDB.* — bitemporal databases.
- *Graphiti.* — temporal-graph memory system for AI agents.
- *GitHub Spec Kit.* — methodology and CLI for spec-driven development with AI agents.
- *Kiro* (AWS), *Tessl, Block Goose, Sourcegraph Amp / Cody.* — agent platforms with repository-scoped context.
- *Anthropic Claude Code, Cursor, GitHub Copilot, Aider, Continue.dev, Cline.* — AI coding assistants whose per-repo configuration files exhibit the branch-divergence problem this framework engages with.

### Related research-arc documents

- [`0-introduction`](https://proliminal.net/theses/ai-workflow-research/) — the trajectory map and kernel summary.
- [`KERNEL.md`](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md) — the eight needs and cross-cutting properties.
- [`00-fighting-git`](https://proliminal.net/theses/fighting-git/) — the formal account of the git-event-log incompatibility.
- [`01-git-native-planning`](https://proliminal.net/theses/git-native-planning/) — the position that markdown plus git suffices.
- [`02-do-we-need-this`](https://proliminal.net/theses/do-we-need-this/) — the audit of whether a framework is needed at all.
- [`03-discipline-where-the-llm-cant-skip-it`](https://proliminal.net/theses/discipline-where-the-llm-cant-skip-it/) — the chokepoint argument.
- [`04-governance-provenance-and-the-pre-pr-tier`](https://proliminal.net/theses/governance-provenance-and-the-pre-pr-tier/) — modular opt-in, governance UX, where CRDTs fit, the pre-PR tier.
- [`05-where-state-lives`](https://proliminal.net/theses/where-state-lives/) — the layered location-of-truth model.
- [`06-poc-build-plan`](https://proliminal.net/theses/poc-build-plan/) — the concrete PoC plan.
- [`07-state-not-workflow`](https://proliminal.net/theses/state-not-workflow/) — state primary, workflow as render.
- [`08-the-pr-bottleneck`](https://proliminal.net/theses/the-pr-bottleneck/) — continuous ratification replaces batched review.
- [`09-orchestrators-and-project-managers`](https://proliminal.net/theses/orchestrators-and-project-managers/) — the orchestration craft and its relation to PM work.
- [`10-spec-based-as-waterfall`](https://proliminal.net/theses/spec-based-as-waterfall/) — argues that spec-based development encodes the same artifact-handoff structure as waterfall, and that continuous ratification is structurally different.
- [`11-should-the-framework-model-the-code`](https://proliminal.net/theses/should-the-framework-model-the-code/) — audits whether aiwf should absorb code-graph functionality (graphify, GitNexus); concludes the framework's lane is decisions about code, not code structure.
- [more is much more](https://proliminal.net/notes/more-is-much-more/) — satellite note: a riff on the LLM-era inversion of "more is more" — production is abundant, ratification is the leverage.

---

## In this series

- Index: [introduction](https://proliminal.net/theses/ai-workflow-research/)
- Build plan: [06 — PoC build plan](https://proliminal.net/theses/poc-build-plan/)
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
