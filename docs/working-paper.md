# A small framework for AI-assisted project tracking in git: a working paper

> **Status:** working paper / synthesis. Not a manual; not a tutorial. A defended position derived from the research arc in [`research/`](research/).
> **Date:** April 2026.
> **Audience:** anyone evaluating whether and how to put structured project-tracking state into a git repository alongside code that AI assistants are helping to produce.

---

## Abstract

AI-assisted software development changes what it means to track work. Plans, decisions, and structural state must remain accessible to a stateless assistant across sessions while still co-evolving with the code they describe. A natural impulse is to import event-sourced architectures into the repository — an append-only event log, a derived projection, hash-verified consistency — but this layering fights git's branching model in ways that grow worse, not better, with scale. We argue that the right framework, at the scale most teams actually operate at, is structurally smaller than this impulse suggests. Markdown files in the repository are sufficient as the canonical state; `git log` is sufficient as the audit trail; the residual job is a small validating engine and a few verbs that produce well-shaped commits. The paper articulates the problem, surveys related work, presents the formal reason an event log resists git, describes the position we walked back to, defends a layered location-of-truth model with enforcement at chokepoints the LLM cannot skip, and uses a working PoC as evidence. We close with the open questions the position does not settle, including identity across repo forks, cross-host skill fidelity, and the formal treatment of branch-divergent assistant rules.

---

## 1. Introduction

The medium of software work is changing. AI assistants now plan, design, code, and decide alongside humans on tasks measured in days, weeks, or months. They do this from a stateless starting point each session, often on partial context, and without a consistent place to consult what was already settled. The artifacts of long-horizon work — epics, milestones, decisions, gaps, contracts — must be both human-editable and machine-readable, both individually authored and collectively coherent. Existing approaches each address some of this and miss the rest.

This paper describes the question we asked, the design space we explored, the position we settled on, and the working code we built to test it. The position is, briefly: **markdown files in the repository are the source of truth, git is the time machine, and the framework's residual job is a small validating engine plus a few verbs that produce well-shaped commits**. We explain why this position is structurally smaller than what we initially proposed, why it survives contact with git in a way the larger design does not, and where its boundaries lie.

The contribution is not a novel algorithm or a new data structure. It is an argument that the right shape for this class of system has been pulled, by collective intuition, toward more sophistication than it deserves, and that a simpler answer better serves the case where AI assistants are doing the heavy lifting on long-horizon software work.

---

## 2. The problem

We define the problem space concretely, by symptom. The following recur across the projects we have observed and have personally lived:

- **The AI re-plans from scratch each session** because it cannot find the current plan, or finds an out-of-date one.
- **Renaming or rescoping a milestone silently breaks references** elsewhere in the repository.
- **Switching branches changes the rules** the assistant believes it should follow, often without the human noticing.
- **Decisions get re-litigated** because no one — human or AI — knows whether something has already been settled.
- **Plans drift faster than they're recorded**, and structural state quietly desynchronizes from the code it claims to describe.

These symptoms generalize into a small number of needs an AI-aware planning framework must serve: recording planning state; expressing relationships among planning items; supporting their evolution under pressure; keeping history honest; validating consistency mechanically; generating human-readable views; coordinating AI behavior; and surviving parallel work by humans and assistants. The full list, with cross-cutting properties, appears in [`research/KERNEL.md`](research/KERNEL.md).

What does not appear in that list is equally important. *Maintaining a totally-ordered event log* is not a need; it is one possible mechanism for serving the "keep history honest" need. *Keeping a derived graph projection* is not a need; it is a possible optimization. *Hash-chaining* is not a need; it is one way to detect drift between two stores. The conflation of needs with mechanisms is, we will argue, a primary source of overdesign in this space.

---

## 3. Related work

The space adjacent to this problem has grown rapidly in the last two years. We summarize it in six clusters, distinguishing what has been formalized in peer-reviewed work from what is currently presented in product documentation and industry essays.

### 3.1 Spec-driven and AI-coding tooling

A wave of tools (GitHub's *Spec Kit*; AWS's *Kiro*; Tessl; Block's *Goose*; Sourcegraph's Amp; the Claude Code, Cursor, Continue, and Aider IDE-integrated assistants) treats the planning specification as a first-class repository artifact. They share a pattern: markdown-shaped specs, agent reads-and-updates conventions, and per-repository configuration. None has, to our knowledge, formally addressed the merge semantics of those artifacts under git branching; the field is too young.

### 3.2 Memory and context systems for coding agents

The "memory bank" pattern (popularized in the Cline community), Cursor's project memory, Aider's repo-map, Continue's context store, and Anthropic's auto-memory in Claude Code each give the assistant a curated, persistent state to consult across sessions. They are not project-tracking systems per se but they share substrate concerns with one — what to write, how to read, how to keep it from rotting.

### 3.3 Per-repository rule files

Every major AI coding host now reads a per-repository rules file: `CLAUDE.md`, `.cursorrules` (and the newer `.cursor/rules/`), `CONVENTIONS.md` for Aider, `.github/copilot-instructions.md` for GitHub Copilot, `.continuerc.json` for Continue, `.windsurfrules` for Windsurf, and the proposed cross-tool `AGENTS.md` standard. All are git-tracked; all are branch-divergent; none has a formal merge story. Empirical evidence that small prompt changes shift agent behavior measurably is established in the in-context-learning literature [Lu et al., 2022], which underlines that the divergence is not benign.

### 3.4 External project management

Mature project-management tools (Linear, Jira, GitHub Projects, Asana, Shortcut, Height) solve multi-user editing, notifications, permissions, and rich querying — at the cost of keeping the planning state out of the AI's working set and decoupling it from the bisectability of code history. The trade-off is real and addressed in §7.

### 3.5 Mergeable structured state

Outside the AI-coding space, several lineages have studied how structured data can survive concurrent edits:

- **Conflict-free Replicated Data Types** [Shapiro et al., 2011], with the JSON-shaped Automerge implementation [Kleppmann & Beresford, 2017] being the most directly applicable to project-tracking state.
- **Patch theory**, via Pijul and the categorical-patches treatment of Mimram and Di Giusto [2013], offering a mathematically rigorous account of when merges are well-defined.
- **Operational Transformation** [Ellis & Gibbs, 1989], the model behind Google Docs and similar collaborative editors.
- **Versioned databases** — Dolt for SQL, TerminusDB for graph data, XTDB and Datomic for bitemporal queries, Irmin for git-like data structures — each demonstrating that branch-and-merge semantics can be lifted from text into structured domains.
- **The local-first software movement** [Kleppmann et al., 2019], which articulates the constraint set that this framework operates under: collaborative, offline-tolerant, multi-writer, no required server.
- **Edits-as-substrate work**, recently exemplified by *Denicek* [Petricek, 2025], representing programs as series of edits over a document — relevant to representation choice if not to location.

### 3.6 Ontology- and graph-based agent memory

A distinct lineage argues that AI agents need *structured* memory — a defined ontology plus a graph-based store that can represent typed entities, named relationships, hierarchical organization, and temporal dynamics. The reference is Yang et al. [2026], *Graph-based Agent Memory: Taxonomy, Techniques, and Applications*, surveying 200+ works and articulating a two-layer model: **knowledge memory** (the ontological scaffolding) and **experience memory** (instances filling that scaffolding). Production systems in this lineage include Graphiti for temporal-graph memory and a growing set of knowledge-graph-backed agent platforms.

This work is adjacent to ours, not the same. The Yang-et-al. lineage addresses *agent cognitive memory* — what an autonomous agent retains across sessions and tasks: facts learned, entities encountered, conclusions drawn. Our framework addresses *project artifact persistence* — epics, milestones, decisions, gaps, contracts that humans and assistants both consult because *the team* needs them as durable structure. The two share substrate concerns (typed relationships matter; flat similarity-search is insufficient) but they are not the same problem. We argue in §6 and §7 that for the scale we target, the typed-relationships need is met by markdown frontmatter validated by Go, without the cost of a graph database — and that conflating "structured" with "graph-DB-and-OWL" produces overdesigned solutions for project tracking. Where projects scale into long-horizon multi-agent territory, the Yang-et-al. apparatus becomes more appropriate; we treat that as a clean future graduation, not as a constraint on the PoC.

These are the bodies of work we considered before settling on the position in §6.

---

## 4. Why structured state in git resists totally-ordered event logs

The most promising-looking import from the event-sourcing tradition is to keep an append-only `events.jsonl` file in the repository, alongside a derived projection (`graph.json`), with each event carrying a sequence number and a hash of the post-state. A reader can replay the events to reconstruct the projection and detect drift in O(1) by hash comparison. Within a single linear history, this is an excellent design.

It does not survive git branching. The argument is mechanical, not aesthetic.

Two branches that have diverged from a common ancestor will, with high probability, both append events to `events.jsonl`. Git's textual three-way merge will, in the absence of conflicts on overlapping lines, concatenate the appends. The result is a syntactically valid JSONL file with three structural problems:

1. **Sequence numbers collide.** Each side allocated `seq=42, 43, 44, …` from the merge base.
2. **Post-state hash chains break.** Each event recorded the canonical hash of the projection *as observed at the time of writing*. After merge, replaying the concatenated log produces, at every step past the merge base, a projection whose hash matches neither the recorded hash on one side nor the other.
3. **Total ordering is destroyed.** The post-merge order is whatever the textual merger produced — no longer a meaningful causal order.

Identifier allocation suffers in parallel. A monotonic per-kind counter is structurally a multi-master replication primitive; concurrent writers will, with no coordination, allocate the same identifier, and the merge will produce two distinct entities with the same id and possibly the same path.

These are not bugs that careful engineering can fix within the chosen representation. They are consequences of the encoding. Total ordering of writes is a property of *file content*; file content is what git merges; therefore total ordering cannot be a property of the merged file. The standard responses — custom git merge drivers (analogous to lockfile merges in npm, yarn, cargo), branch-aware identifiers, demoting the hash chain to be branch-local with explicit reconciliation events — are all viable. They are also, individually and together, costlier than the failure mode they address. They reproduce, in repository-tracked text, what git already provides for files at the file-tree level.

The CRDT and local-first literatures are, in our reading, the right foundation for this problem. Git already behaves like a per-file CRDT for line-structured text. Layering a non-CRDT abstraction (a totally-ordered hash-chained log) over a mergeable substrate is the structural error. Either the substrate is lifted (replace `events.jsonl` with an Automerge document), or the abstraction is lowered (drop the linearity claim and let git's per-file merge stand). The middle path — keep the linearity claim, mediate at merge time — survives but pays an ongoing cost.

We observe that for the project shapes most actual users target (single developer or small team, weeks-to-months horizons, ordinary git workflows), the cost is not justified. The abstraction should be lowered.

---

## 5. The walk-back

The repository's earlier design [`architecture.md`] proposed an event-sourced kernel with hash-verified projections. We argued ourselves into it; we argued ourselves back out of it. The relevant abandonments, in order of severity:

- **The append-only `events.jsonl` file** is removed. Its job — recording mutations with actor, timestamp, and intent — is done by `git log` augmented with structured commit trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`).
- **The `graph.json` projection** is removed. Its read-model role is filled by markdown frontmatter, parsed on demand. Drift detection is replaced by validation: there is no second store to drift from.
- **RFC 8785 canonicalization and SHA-256 hashing of the projection** are removed. Without a second store, there is nothing to canonicalize against.
- **The monotonic ID counter** is replaced by a tree-scan allocator (pick `max(seen ids of this kind) + 1` at write time) plus an explicit collision-resolution verb (`aiwf reallocate`). Branch-aware coordination is unnecessary at the target scale.
- **Trace-first writes as a permanent ledger** are replaced by trace-first writes within a single verb's execution: a journal file in `.ai-repo/journal/` (gitignored) records intent before file edits and is deleted on commit success. Crash recovery is local; no permanent ledger.
- **The closed entity vocabulary as YAML-defined boundary contracts** is replaced, for the PoC, by hardcoded Go structs for six kinds. YAML-driven kinds remain a clean future extension when consumers actually need to customize the vocabulary.

What survives is the principle that the engine is invocable without an AI assistant, the principle that closed-set vocabularies belong in code rather than in prose, and the principle that errors are findings rather than parse failures. These were never the costly part.

---

## 6. The position

We claim three load-bearing things.

**Markdown files are the source of truth.** Each entity (epic, milestone, ADR, gap, decision, contract) lives in a markdown file with YAML frontmatter (typed structural fields) and a prose body (narrative content). The frontmatter is what the engine reads; the body is opaque to the engine. Files merge cleanly under git's per-file three-way merge in the common case; structurally meaningful conflicts surface as either YAML-conflict markers (which a human resolves) or post-merge findings reported by the validator.

**Git is the time machine.** History is `git log`, filtered per entity by structured commit trailer. The engine provides a thin renderer (`aiwf history <id>`) that reads the log; it does not maintain a parallel ledger. Bisectability is git's bisectability. Audit is git's audit. Provenance ties to commits because commits are the units of mutation.

**The framework's residual job is small.** Six things are needed: a tree loader that parses frontmatter into typed structs; a validator that reports findings on the loaded tree (referential integrity, status legality, no cycles, frontmatter well-formedness); a small set of verbs that perform multi-file mutations atomically (one git commit per verb, with structured trailers); identity ergonomics that survive rename and collision (slug-in-path, `aiwf rename`, `aiwf reallocate`); a skill-materialization step that gives AI hosts the conventions in their native shape; and a pre-push git hook that runs the validator. The PoC implements this in roughly 2,500 lines of Go.

What this is not: a project-management replacement, a state-management abstraction layered on git, a knowledge graph, a multi-agent orchestrator, or a code-execution agent. It is also explicitly not an *agent cognitive memory system* in the sense of Yang et al. [2026]: we track project artifacts that humans and assistants both consult, not the assistant's own learned recall across sessions. Those are different problems with different scales and different substrates; an agent that needs cognitive memory should compose our framework with a memory system designed for that purpose, not expect ours to serve both roles. The framework is a small, opinionated, deterministic shim between humans and AI assistants doing project-tracking work in markdown files inside a git repository.

---

## 7. The layered location-of-truth model and the chokepoint argument

State in this system lives at six layers, each chosen to minimize the friction of its update pattern.

| Layer | Where it lives | Why |
|---|---|---|
| Engine binary (`aiwf`) | Machine-installed via `go install` (or future package manager) | Standard tool distribution; same as `git` itself |
| Per-project policy (`aiwf.yaml`, `aiwf.lock`) | In the repository, git-tracked | Team-shared, CI-readable, travels with clone |
| Per-project planning state (`work/`, `docs/adr/`) | In the repository, git-tracked | Co-evolves with code; bisectable |
| Project-specific skills | In the repository, git-tracked | Team's own conventions evolve with the project |
| Framework + third-party skill content | Bundled with binary or registry-cached | Independent versioning |
| Per-developer config | `~/.config/aiwf/` | Personal; no team coupling |
| Materialized skill adapters + runtime cache | In the repository but gitignored | Composed from the layers above, regenerated only on explicit `aiwf init` / `aiwf update` |

The materialization invariant is load-bearing. The adapter set the AI host actually reads is regenerated on explicit user action, not implicitly on `git checkout`. This decouples assistant behavior from branch state in the common case, while still letting branches genuinely diverge in skill source when that is the work being done. It mirrors how npm packages, language toolchains, and IDE plugins behave: branch switches do not silently change the tooling.

Within this layout, *enforcement* is the second load-bearing claim. The framework provides skills that document what the AI should do. Skills are advisory text; the AI may invoke them, may overlook them, or may operate in a mode that does not load them. Any guarantee that depends on the assistant remembering to follow a skill is not a guarantee — it is hope dressed up as policy.

The framework's actual guarantees come from `aiwf check` running as a `pre-push` git hook. The hook is what turns "the assistant is supposed to keep references valid" into "broken references cannot reach the remote." The validator is mechanical, deterministic, and fast; it operates without AI judgment; it produces findings that any caller (human, AI, CI script) can act on. This is the chokepoint where the framework imposes itself on the work, and it is intentionally narrow: validation, not approval; reporting, not rewriting; advisory below the chokepoint, authoritative at it.

This positioning also addresses the soft/hard tension between exploratory planning ("plans are clay") and durable record ("main is a museum"). Studio behavior — branches, drafts, free iteration — is unconstrained. Workshop behavior — the moment between studio and museum — is where the validator runs, where the AI is still in the loop locally, and where most reconciliation work belongs. Museum behavior — what is on the published main — is sealed by the chokepoint having done its job upstream. The framework does not impose museum semantics on the studio.

---

## 8. The PoC as evidence

A position is only as good as the implementation that tests it. The PoC, on the `poc/aiwf-v3` branch, is a single Go binary `aiwf` with the following surface:

- Six entity kinds — epic, milestone, ADR, gap, decision, contract — each with a closed status set and a one-function transition rule.
- Stable identifiers (`E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`) preserved across rename, cancel, and collision; `aiwf reallocate` handles collisions deterministically.
- Mutating verbs (`add`, `promote`, `cancel`, `rename`, `reallocate`) each producing one git commit with a structured trailer.
- A read verb (`aiwf check`) that walks the working tree and reports findings (id collisions, unresolved references, illegal transitions, missing frontmatter, contract artifacts that don't exist on disk, cycles).
- A history verb (`aiwf history <id>`) reading `git log` filtered by entity trailer.
- Skill content embedded in the binary, materialized to `.claude/skills/wf-*` and gitignored.
- A pre-push hook installed by `aiwf init` that runs `aiwf check`.

The PoC is not a finished framework. It is an existence proof: the position can be implemented in a few sessions of focused work, the resulting tool delivers the guarantees we claim it does, and the surface remains small enough that an evaluator can read the whole thing in an afternoon. Where it grows from here is a question for real use to answer.

---

## 9. Open questions

The position does not settle several questions; we name them so future work can address them honestly.

- **Identity across repository forks and transfers.** A repository's `git config remote.origin.url` is fragile under forks and ownership transfers; absolute paths are fragile under moves; in-repo identifier files require human-maintained coordination. The right choice for cross-repository project identity remains open.
- **Cross-host skill fidelity.** The framework materializes skill adapters per AI host; how much fidelity is preserved across Claude Code, Cursor, Copilot, Continue, and Aider is a function of those hosts' adapter formats and is not under the framework's control. A common skill-content format is desirable; whether it can be agreed across vendors is not certain.
- **Branch-divergent assistant rules.** Per-repository rule files diverge across branches in every major AI coding tool. We have proposed a materialization invariant that mitigates this; we have not formally treated the problem. The literature [Lu et al., 2022; Bai et al., 2022] suggests the divergence is not benign, but the formalization is open.
- **Multi-machine without a server.** A single developer working across machines, or a small team without shared infrastructure, currently relies on `git push`/`pull` for state synchronization. Local-first techniques (CRDT-merged state, Automerge-based stores) could complement this; the trade-offs at this scale are not yet measured.
- **Compliance and regulated industries.** The PoC's provenance is sufficient for engineering hygiene but has not been evaluated against specific regulatory frameworks (21 CFR Part 11, SOX evidence, ISO 27001 audit trails). Whether the structured-commit-trailer model can carry that load, or whether regulated environments need a richer provenance store, is open.
- **The boundary between discipline and automation.** The framework draws the line at "validate; do not auto-fix." For a single-developer or small-team setting, this is the right default; for higher-volume settings, automation of the validator's findings (auto-fix safe ones, surface the rest) becomes more attractive. Where exactly that line should move is open.
- **A CRDT-modeled metadata layer.** We argued that a small subset of the framework's state (the identifier registry, the reference graph, status registers over partial orders) is naturally CRDT-shaped. The PoC does not implement this; it uses path-prefix collision detection and a manual reallocation verb. The case for a small CRDT layer remains, especially as concurrency increases.
- **Graduation to graph-substrate memory at long-horizon scale.** Yang et al. [2026] argue that long-horizon, multi-agent, regulated-domain settings need ontology-plus-graph memory architectures with explicit knowledge / experience / temporal layers. Our framework is sized below that threshold; whether it can graduate cleanly to such a substrate (e.g., emit triples derivable from frontmatter, sync with an external knowledge graph) or whether projects in that range need a different framework entirely is open. The on-disk format (markdown frontmatter with closed-set vocabularies and structured commit trailers) is simple enough that mechanical extraction into a graph store is plausible; whether that extraction preserves enough fidelity to be useful is not yet measured.

---

## 10. Conclusion

We set out to design an AI-aware project-tracking framework and ended up with one substantially smaller than we initially proposed. The walk-back was not a retreat from ambition; it was a realization that the ambition was layered on the wrong substrate. Git already provides much of what we wanted to build — total ordering within a branch, atomicity at the commit, attributable history, branching as first-class divergence — and the residual job for an AI-aware framework is correspondingly modest. The framework's value is the smallest set of mechanical guarantees that lets a forgetful AI and a busy human not lose track of what is planned, decided, and done. The smallest set, at the scale most projects actually operate at, is six entity kinds, stable identifiers, a pre-push validator, a structured-commit history reader, and skill materialization that does not change under the assistant's feet. Build that. Use it. Let real friction tell you what to add.

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
- Terry, D. B., Theimer, M. M., Petersen, K., Spreitzer, A. J., Demers, A. J., & Welch, B. B. (1995). *Managing Update Conflicts in Bayou, a Weakly Connected Replicated Storage System.* SOSP '95.
- Yang, et al. (2026). *Graph-based Agent Memory: Taxonomy, Techniques, and Applications.* Survey covering 200+ works on memory systems for LLM agents, including the knowledge-memory / experience-memory distinction and a taxonomy of knowledge graphs, hierarchical graphs, temporal graphs, and hypergraphs as memory substrates.

### Industry essays and books

- Kleppmann, M. (2017). *Designing Data-Intensive Applications.* O'Reilly.
- Nygard, M. (2011). *Documenting Architecture Decisions.* (Industry essay; the seed of the ADR convention.)

### Tools and products (cited by URL or name; not peer-reviewed)

- *Automerge.* https://automerge.org/ — JSON CRDT library and runtime.
- *Yjs.* https://yjs.dev/ — text and structured CRDTs, used in many collaborative editors.
- *Loro.* https://loro.dev/ — newer CRDT runtime focused on performance.
- *Pijul.* https://pijul.org/ — version control system based on patch theory.
- *Dolt.* https://www.dolthub.com/ — Git-style branching and merging for SQL databases.
- *TerminusDB.* — branched graph database.
- *Irmin.* https://irmin.org/ — Git-like distributed data store for OCaml/MirageOS.
- *Datomic.* — bitemporal database with point-in-time queries.
- *XTDB.* — bitemporal database emphasizing valid-time and transaction-time.
- *Graphiti.* — temporal-graph memory system for AI agents, cited by Yang et al. [2026] as the canonical example of graph-based memory with distinct creation and expiration timestamps.
- *GitHub Spec Kit (`spec-kit`).* — methodology and CLI for spec-driven development with AI agents.
- *Kiro* (AWS). — agentic IDE with in-repo spec/design/tasks.
- *Tessl.* — spec-centric coding workflow.
- *Block Goose.* — agentic toolkit with planning surfaces.
- *Sourcegraph Amp / Cody.* — agent platforms with repository-scoped context.
- *Anthropic Claude Code.* — CLI-integrated coding assistant; the host whose skill format the PoC targets first.
- *Cursor.* — IDE-integrated assistant; rule files at `.cursorrules` and `.cursor/rules/`.
- *GitHub Copilot.* — IDE integration with `.github/copilot-instructions.md`.
- *Aider.* — CLI assistant with `CONVENTIONS.md` per repository.
- *Continue.dev.* — IDE integration with `.continuerc.json` and per-repo context.
- *Cline.* — open-source coding assistant; community-popular "memory bank" pattern.

### Related research-arc documents in this repository

- [`research/KERNEL.md`](research/KERNEL.md) — the eight needs and cross-cutting properties.
- [`research/00-fighting-git.md`](research/00-fighting-git.md) — the formal account of the git-event-log incompatibility.
- [`research/01-git-native-planning.md`](research/01-git-native-planning.md) — the position that markdown plus git suffices.
- [`research/02-do-we-need-this.md`](research/02-do-we-need-this.md) — the audit of whether a framework is needed at all.
- [`research/03-discipline-where-the-llm-cant-skip-it.md`](research/03-discipline-where-the-llm-cant-skip-it.md) — the chokepoint argument.
- [`research/04-governance-provenance-and-the-pre-pr-tier.md`](research/04-governance-provenance-and-the-pre-pr-tier.md) — modular opt-in, governance UX, where CRDTs fit, the pre-PR tier.
- [`research/05-where-state-lives.md`](research/05-where-state-lives.md) — the layered location-of-truth model.
- [`research/06-poc-build-plan.md`](research/06-poc-build-plan.md) — the concrete PoC plan.
