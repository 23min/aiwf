# Should the framework model the code?

> **Status:** defended-position
> **Hypothesis:** Two adjacent tools (graphify, GitNexus) model the code as a knowledge graph; aiwf should not absorb that role. The framework's lane is the *decisions about the code*, not the code's structural understanding. The kernel does not authorize a code-graph; only one narrow consistency check (symbol-level reference resolution for fields like `live_source`) survives the rubric, and even that only when the contracts work demands it.
> **Audience:** anyone proposing to add code-aware verbs (impact analysis, blast-radius, structural search, multimodal corpus indexing) to aiwf, or wondering whether the existence of those tools means aiwf should grow toward them.
> **Premise:** the kernel is the rubric ([`KERNEL.md`](KERNEL.md)); the audit voice is [`02`](02-do-we-need-this.md); the state-not-render distinction is [`07`](07-state-not-workflow.md). This document applies all three to a temptation the prior arc did not face: the existence of two well-engineered code-graph tools whose features look adjacent enough to ours that absorption seems plausible.
> **Tags:** #thesis #aiwf #scope #state-model

---

## Abstract

Two open tools, [graphify](https://github.com/safishamsi/graphify) (~10k LOC Python) and [GitNexus](https://github.com/abhigyanpatwari/GitNexus) (~94k LOC TypeScript), turn a folder of code into a queryable knowledge graph and use that graph to give AI assistants better orientation, blast-radius analysis, and impact-before-edit awareness. Their existence raises a question the arc had not yet faced: should the framework grow a code-graph too? The unique-join argument is real — aiwf is the only tool that knows both the code and the decisions about it, so aiwf is the only tool that *could* compute "this symbol's callers + the contract that asserts about it + the ADR that decided it + the gap open against it." But applying [`KERNEL.md`](KERNEL.md)'s eight-needs rubric, [`02`](02-do-we-need-this.md)'s audiences distinction, and [`07`](07-state-not-workflow.md)'s state-vs-render boundary, the principled answer is no. A comprehensive code-graph fails the kernel: none of the eight needs requires it, the assistant audience does not need structured code-graph state to do its job, and a derived graph over code is not aiwf's state — it's a render over material the framework does not own. One narrow case survives: symbol-level reference resolution as a consistency check on fields aiwf does own (`live_source` in the contracts work). That is mechanical, deterministic, LLM-skip-proof, and lands as a small extension of the kernel's existing fifth need ("validate consistency"). Everything beyond that — orientation, impact, multimodal extraction, cross-repo contract auto-extraction — is *adjacent-tool work*, and the framework's posture should be **compose, don't absorb**: expose aiwf's state through stable surfaces so adjacent tools can join with it, but do not embed their roles. The arc's slogan: *aiwf records assertions about code; aiwf does not model code.*

---

## 1. The temptation

Two tools have lately been doing impressive things in this space.

**graphify** is a context-engineering tool for the AI assistant. It scans any folder — code, markdown, PDFs, images, videos — and emits a knowledge graph as files (`graph.json`, an interactive `graph.html`, and a one-page `GRAPH_REPORT.md` showing god nodes and surprising connections). Three-pass pipeline: deterministic tree-sitter AST extraction across 25 languages; local Whisper transcription for video/audio; LLM-driven extraction over prose, papers, images, transcripts. Edges tagged `EXTRACTED` / `INFERRED` / `AMBIGUOUS` with confidence scores. Always-on integration is a PreToolUse hook that surfaces the report before the assistant grep/globs. Headline claim: 71.5× fewer tokens per query on a mixed corpus. Distributed as a skill (`/graphify`) for many host AI tools.

**GitNexus** is a code-intelligence platform. CLI + MCP server + browser web UI + cross-repo orchestration. Storage in an embedded graph DB hit with Cypher. Twelve-phase indexing DAG (`scan → structure → markdown → cobol → parse → routes → tools → orm → crossFile → mro → communities → processes`). Hybrid BM25 + vector search. Auto-extracted execution flows ("processes"), HTTP/gRPC/topic contracts across repos, blast-radius analysis with risk levels. The CLAUDE.md it installs in consumer repos enforces behavioral patterns: *"MUST run impact analysis before editing any symbol; MUST run detect_changes before committing."* Polyform Noncommercial license, with a SaaS arm.

Both are good. Both solve a real problem — making assistants smarter about *the code as it currently is*. Their feature surface (impact, detect-changes, knowledge-graph-as-context, contract registries) overlaps verbally with things aiwf cares about (the contracts work, the assistant's behavior, structural understanding). The temptation to absorb is strong: aiwf already knows the decisions; if it also knew the code, it could compute joins neither of those tools can.

The rest of this document audits whether that temptation is principled or whether it's the same trap [`02`](02-do-we-need-this.md) caught the original architecture in.

---

## 2. What problem each tool actually solves

Stripped of marketing:

- **graphify** solves *orientation* for the assistant working over a corpus. The corpus may include code, but it may also be papers, screenshots, whiteboard photos, video transcripts. The graph is regenerated from the corpus on demand; it is not state in the system-of-record sense. The audience is the assistant inside the directory.

- **GitNexus** solves *structural awareness of the code* for the assistant editing the code. The graph is canonical state in *its* sense — persistent, indexed, queryable, the basis for impact analysis. Its audience is the assistant about to make a change.

- **aiwf** solves *the converging picture of what the team currently believes about what they're building* — the durable structural decisions, recorded in a small typed vocabulary, in a shape humans, LLMs, CI, and other tools can all read and write. (Per [`07`](07-state-not-workflow.md) §11 and the appendix definition.) Its audience is everyone consulting the team's belief state.

The three tools sit on different layers of the stack. graphify and GitNexus model **the code**. aiwf models **the decisions about the code**. They are not competitive; they are at different layers. A team can run aiwf and graphify alongside; a team can run aiwf and GitNexus alongside. The audit question is whether aiwf should *also* model the code itself.

---

## 3. The kernel test

[`KERNEL.md`](KERNEL.md) is the rubric. Every proposal earns its place by serving one of the eight needs, or it doesn't enter the framework. The eight, in compressed form:

1. Record planning state.
2. Express relationships.
3. Support evolution.
4. Keep history honest.
5. Validate consistency.
6. Generate human-readable views.
7. Coordinate AI behavior.
8. Survive parallel work.

Walk a comprehensive code-graph against each, asking *does this need a code-graph to be served?*

- **(1) Record planning state** — no. Planning state is epics, milestones, decisions, gaps, contracts. None of these *is* code structure; some *reference* code structure (a contract's `live_source`).
- **(2) Express relationships** — no. The relationships the framework expresses are *between aiwf entities* (this milestone belongs to that epic, this decision ratifies that scope). Code structure is relationships *within the code*, a different graph.
- **(3) Support evolution** — no. Evolution of *plans* doesn't need a code-graph. Evolution of *code* is git's job.
- **(4) Keep history honest** — no. Provenance for who-changed-what is `git log` plus structured trailers (per [`01`](01-git-native-planning.md) and [`03`](03-discipline-where-the-llm-cant-skip-it.md)). Provenance for code is again `git blame`, not aiwf.
- **(5) Validate consistency** — *partially*. Aiwf already validates that references between entities resolve, FSM transitions are legal, ids don't collide. If a contract field references a code symbol, the kernel's posture is that this reference must resolve mechanically. That is a narrow extension that may need AST awareness — but only enough to answer "does this symbol exist?" — not a full graph.
- **(6) Generate human-readable views** — no. Views are derived from canonical state ([`07`](07-state-not-workflow.md) §4.1). Code-graph renders are views over *code*, which the framework does not own as state.
- **(7) Coordinate AI behavior** — partially. Coordinating *which graphify or GitNexus output the assistant should consult* is a thing the framework's skills can do. *Producing* that output is not.
- **(8) Survive parallel work** — no. Parallel-work concerns are about merging structural state, not about making the code graph survive merges.

Net: only (5) admits any extension, and only the narrowest possible one. The kernel does not authorize a code-graph. A proposal to add one would need to either argue an existing need is being served better than alternatives (it isn't), or propose a kernel addition (and the kernel changes by deliberate edit, with reasoning recorded — not by feature accretion).

This is exactly the rubric the arc has been applying since [`02`](02-do-we-need-this.md). Every prior doc *narrowed* scope by walking proposals against the kernel. This doc continues the posture.

---

## 4. The audiences distinction

[`02`](02-do-we-need-this.md) §6 made a load-bearing observation that bears repeating verbatim because it directly settles this question:

> Structured state pays off for *programmatic* consumers (CI gates, dashboards, audit), not for AI assistants. Confusing these two audiences inflates the design.

And:

> The forgetful AI is fixed by giving it a habit, not by giving it a database.

The temptation to add a code-graph to aiwf is largely the same temptation [`02`](02-do-we-need-this.md) caught the original architecture in: *if we structure more state, the assistant will reason more reliably.* The arc concluded that structure pays off for the *programmatic consumer* that does not exist yet (CI gate verifying contracts, audit dashboard, regression detector) and not for the assistant that already reads prose well.

For graphify and GitNexus, the trade is different — they are themselves the programmatic consumer. graphify reads its graph to compute token-efficient orientation; GitNexus reads its graph to compute blast radius. They earn the structure they build because their products consume it. Aiwf would build structure for an assistant audience that doesn't need it (the assistant already has graphify or GitNexus or tree-sitter or its own context window) and a programmatic consumer that doesn't exist (the framework has no code-aware verb that would consume a code-graph beyond what we already cover at the kernel level).

The single exception is the contracts work's `live_source` field. There the *programmatic consumer* — `aiwf contract verify` checking that the reference resolves — is real, narrow, and on the kernel's existing list. That is the only audience the structure is built for. This is a much smaller commitment than "build a code-graph."

---

## 5. The state-vs-render lens

[`07`](07-state-not-workflow.md) §4.1 distinguishes **state (canonical) from renders (courtesies derived from state on demand).** The framework owns state. Renders are produced over the framework's state, opt-in, computed when wanted, not authoritative.

Apply this lens to a code-graph:

- Is a code-graph **state**? No. The code is state — files in a git repository, mutable through commits. A graph derived from those files is a render of the code, not new state.
- Is it a **render the framework could produce**? Renders are produced over the framework's state, not over arbitrary external content. The ROADMAP renders entity state; the dependency graph renders typed relationships between entities. A code-graph would render *code* — material the framework does not own and has no decisions about.
- Is **symbol-existence resolution** state? No, but it's a consistency check over a reference embedded in state. The reference (`live_source: file.go#Symbol`) is state — it's a field on a contract entity. The check is mechanical resolution of that reference against the codebase. Same shape as `aiwf check refs` resolving a milestone's `parent` field against an actual epic. *Cross-artifact reference resolution is already a thing the framework does;* extending it from "milestone → epic" to "contract → code symbol" is a small generalization, not a new category.

So the lens is clean: aiwf does not own *the code-graph*, it owns *the references its entities make into the code* and the verification that those references resolve. The verification machinery happens to need AST awareness, but the framework's posture is "I check the consistency of my references" — not "I model the code."

This mirrors the framework's posture toward git itself ([`01`](01-git-native-planning.md), [`05`](05-where-state-lives.md)): aiwf does not own commits, it relies on `git log` for provenance. It does not own ADR documents, it owns the entities that point at them. It does not own the build pipeline, it owns the contract entity that describes the surface the build pipeline must respect. The pattern throughout the arc is: *aiwf records assertions about adjacent material; aiwf does not absorb the adjacent material.* A code-graph would absorb. Symbol-existence resolution does not.

---

## 6. The unique-join temptation, taken seriously

The strongest counter-argument to this position is the unique-join argument. State it carefully:

> Aiwf is the only tool that knows both the team's decisions and the code those decisions are about. Therefore aiwf is the only tool that can compute "this symbol's callers AND the contract that asserts about it AND the ADR that decided it AND the gap open against it." That join is uniquely valuable; ceding it is ceding the framework's deepest differentiation.

Take the premise. The join *is* uniquely valuable. The question is not whether the join should exist but **where it should be computed**.

Two answers:

**(a) The framework computes the join.** Aiwf grows code-aware verbs (impact, detect-changes, blast-radius). The framework absorbs code-graph functionality so the join can be served as a single primitive. This is the expansive position.

**(b) The framework exposes its state for adjacent tools (or future modules) to join.** Aiwf's projections (entities, contracts, gaps, ADRs, the audit history) are exposed through stable surfaces — files, structured CLI output, eventually MCP resources. graphify and GitNexus already expose their code-graphs through analogous surfaces. The join is computed by composition, on demand, by whoever needs it. This is the compose-don't-absorb position.

Position (b) preserves every kernel quality bar. The framework stays small; it does not learn languages, store graphs, or run extractors. Position (a) violates the kernel's authorization (no need calls for it), violates [`02`](02-do-we-need-this.md)'s audiences caution (building structure for an audience that doesn't need it), and violates [`07`](07-state-not-workflow.md)'s state-vs-render boundary (treating derived view as state).

Position (b) has a real cost worth naming: the join is not a single primitive in any tool. The assistant has to learn to compose — *first ask aiwf about the contract; then ask the code-graph tool about the symbol; then assemble the picture.* That is more steps. It is also exactly the [`02`](02-do-we-need-this.md) prescription: *give the assistant a habit, not a database.* The habit is "consult both layers." The framework's job is to make its layer queryable through skills the assistant can invoke; the adjacent tool's layer is queryable through *its* surfaces. Composition is the work; absorption is the shortcut that breaks the kernel.

A future module — opt-in, per [`04`](04-governance-provenance-and-the-pre-pr-tier.md) §4 — could ship the join *as a render*, computed by composing aiwf's state with whichever code-graph tool the consumer uses. That stays inside the kernel because it is a render module, not a kernel addition. It is allowed; it is not required.

---

## 7. The narrow case that survives

What does survive a serious read of the kernel: **symbol-level reference resolution.**

The contracts work introduces a `live_source` field. At the file-path tier, `live_source: tools/internal/eventlog.go` is a reference that must resolve. That's served by `os.Stat`. No code-graph needed. But a contract whose authoritative implementation is *a particular function*, not a whole file, wants `live_source: tools/internal/eventlog.go#Append` — and that reference is interesting. Renaming the function while leaving the file intact is *exact drift the contract was designed to catch.* File-level resolution misses it; symbol-level resolution catches it.

This is a kernel-(5) extension. *"Validate consistency — references resolve."* Symbol-level references are a kind of reference. Making them resolve is exactly the kind of mechanical, deterministic, LLM-skip-proof check [`03`](03-discipline-where-the-llm-cant-skip-it.md) calls for. The implementation is bounded:

- AST extraction for the languages aiwf already supports (per the PoC's chosen scope), enough only to enumerate symbols by name in a file.
- A `aiwf check refs` extension that resolves `<path>#<symbol>` references the same way it currently resolves entity-id references.
- No graph artifact, no projection beyond a finding, no exposed traversal verbs, no impact analysis, no multimodal extraction.

This is a small, narrowly-scoped, kernel-justified consistency check. It is *not* a code-graph. It happens to need AST awareness, but the framework does not become code-graph software by gaining the ability to answer *"does this named symbol exist in this file?"*. The exposed surface is a single check; the internal implementation may use tree-sitter; nothing about the framework's posture toward the code changes.

This is the only case the audit admits.

---

## 8. The compose-don't-absorb posture, in practice

Aiwf's stance toward graphify, GitNexus, and any future tool in this space:

1. **Aiwf does not include their features.** No impact verb, no blast-radius, no embeddings, no community detection, no multimodal extraction, no cross-repo Contract Registry, no MCP-first integration, no graph DB, no web UI, no automatic CLAUDE.md mutation.

2. **Aiwf exposes its state cleanly.** The projection (entities, ADRs, gaps, contracts, validation findings) is queryable through the engine's CLI in JSON envelopes; via files (`CONTRACTS.md`, `ROADMAP.md`); eventually via MCP resources. Adjacent tools that want to join with aiwf's state read these surfaces.

3. **Aiwf's skills can recommend adjacent tools.** When a consumer wants impact analysis, the framework's skill content can teach the assistant how to invoke the consumer's chosen code-graph tool — *not* how to invoke an aiwf-internal one. That is the kernel-(7) "coordinate AI behavior" need, served by composition.

4. **Future render modules may compose.** A future opt-in module could compute the join (contract + symbol + ADR) by reading both layers and rendering a finding. That stays inside the kernel as a render. It is not in scope today.

5. **Aiwf tells consumers honestly that this is the design.** The README and pocv3 plan should name what aiwf does *not* do, and recommend adjacent tools by name where appropriate. The framework's strategic position is the lane it commits to, not the lane it pretends to.

This is the same shape as `aiwf` toward `git`: the framework exposes structured trailers and verbs over commits, but does not *own* commits — git owns commits, the framework records assertions about them. Code-graph tools own code structure; the framework records assertions about it.

---

## 9. What's worth borrowing without absorption

Three patterns in the adjacent tools are worth registering, none of which require absorbing the tool:

- **Edge confidence tagging.** graphify tags every edge `EXTRACTED` / `INFERRED` / `AMBIGUOUS`. The aiwf analogue, applied to *findings* rather than code edges, is honest discrimination between assertions, derived facts, and unresolved ambiguity. Already aligned with the kernel's "honest about meaning" quality bar; worth being explicit about it in finding-output formats.

- **PreToolUse hook as the always-on injection.** Both graphify and GitNexus install a hook that surfaces relevant context before the assistant searches. Aiwf can adopt the same shape for surfacing *its* state — current epic, open ADRs, the contract registry — before structural edits. This is kernel-(7) and does not require code-graph functionality.

- **Behavioral discipline written into skills.** GitNexus's *"MUST run impact analysis before editing any symbol"* is a behavior pattern, not a feature. Aiwf's analogue: *"before structurally editing a symbol named in a contract's `live_source`, run `aiwf show <C-id>` to read the drift guard."* The discipline is a skill; the skill needs only the framework's existing surface to work. No code-graph required.

These borrowings strengthen the framework's lane without crossing it.

---

## 10. The honest failure mode

This position has a failure mode worth naming, in keeping with [`07`](07-state-not-workflow.md) §8's discipline.

**Aiwf's compose-don't-absorb position is wrong for teams whose work demands a *single primitive that joins both layers*.** Specifically:

- **Highly regulated teams** where audit demands a single, mechanically-validated trace from "this regulation requires X" to "this code symbol satisfies X" with no composition step that could elide the link. For these teams, a unified primitive is load-bearing for compliance, and the absence of one is an audit risk.
- **Teams where the assistant must work without network or external tools.** If the consumer does not run graphify or GitNexus and only has aiwf, then the join cannot be computed by composition. The assistant lacks the second layer entirely.
- **Tooling consumers (CI gates, dashboards) where the join is the product.** A CI gate that wants to fail builds when "any symbol named in a contract `live_source` is missing in the working tree" can be written, but only if it knows how to look up symbols. If aiwf does not provide the lookup, the gate has to depend on whatever code-graph the consumer chose, which fragments the gate per consumer.

For these cases, the right answer is the future render module described in §6 — *opt-in*, *per-consumer*, joins against whichever code-graph tool the consumer runs. The kernel stays clean; the join becomes available where it is justified.

What is *wrong* in every case is to bake the code-graph into the framework. That ships every consumer the cost of every consumer's worst case, in service of a need most do not have. Aiwf's modular opt-in posture ([`04`](04-governance-provenance-and-the-pre-pr-tier.md) §4) is the right answer.

---

## 11. Implications for the contracts work and the pocv3 plan

The contracts post-PoC plan ([`docs/pocv3/contracts.md`](../pocv3/contracts.md)) currently lists symbol-level `live_source` as increment I4, depending on "the read-side reference-resolution lens." That dependency phrasing is consistent with this document's position: I4 is a small consistency-check extension, not a code-graph. The plan should not be reframed; only the language should be tightened — *"reference resolution"* not *"code graph"*, and the I4 scope explicitly bounded to symbol-existence checking, not traversal.

Anything beyond I4 in the code-aware direction (impact, detect-changes, multimodal extraction) is **out of scope for the framework**. If a consumer wants those, the recommended path is graphify or GitNexus alongside aiwf, with the framework's skills knowing how to compose with them. A future opt-in module, justified by real consumer need, may add the join as a render.

This sharpens what the framework commits to: contracts get teeth at the file level (drift detection), then at the symbol level (reference resolution), then through whatever validator the consumer wires into the verifier hook. The framework provides the slots, the consistency checks, and the events; the rest is composition.

---

## 12. The position in one paragraph

Two adjacent tools — graphify and GitNexus — model the code as a knowledge graph and use that graph to make AI assistants better at orientation, impact analysis, and contract awareness. Their existence is the strongest external pressure the arc has yet faced toward expanding aiwf's scope. The kernel ([`KERNEL.md`](KERNEL.md)) does not authorize the expansion: none of the eight needs requires code-graph state, the assistant audience does not need structured code-graph state to do its job ([`02`](02-do-we-need-this.md) §6), and a graph derived from code is not aiwf's state — it is a render over material the framework does not own ([`07`](07-state-not-workflow.md) §4.1). One narrow case survives — symbol-level reference resolution as a consistency check on `live_source` fields the framework owns. Everything beyond that is *adjacent-tool work*, and the framework's posture is **compose, don't absorb**: expose aiwf's state through stable surfaces, recommend adjacent tools where appropriate, leave the join to a future opt-in render module if consumer need ever justifies one. The lane: *aiwf records assertions about code; aiwf does not model code.* This is the same restraint the arc has practiced from [`00`](00-fighting-git.md) onward — every prior doc narrowed scope; this one continues the discipline against a temptation to expand.

---

## 13. Open questions

The position does not settle several things, named so future research can address them honestly:

1. **What's the smallest tree-sitter footprint that serves symbol-existence resolution?** I4 in the pocv3 plan is bounded but unwritten. A separate doc may need to specify the scope of AST-awareness aiwf accepts before it counts as code-graph creep.
2. **Does the future render module ever earn its keep?** The §10 failure modes are real. Whether any consumer hits them in practice will be the test. If yes, the module is justified; if no, the position holds without addition.
3. **How does aiwf's recommendation of adjacent tools stay honest as those tools evolve?** graphify and GitNexus have different licenses, different posture toward the assistant, different long-term commercial intent. The framework's skills should not silently couple consumers to either; recommendations should be neutral and per-consumer.
4. **Are there other "adjacent tool" categories the same audit applies to?** Code-graph is one. Test runners that expose pass/fail state, deploy systems that expose release state, telemetry systems that expose runtime state are others. The compose-don't-absorb posture probably generalizes; this doc only treats the code-graph case.

---

## 14. References

- [`KERNEL.md`](KERNEL.md) — the rubric. The eight needs and quality bars this document audits against.
- [`02-do-we-need-this`](02-do-we-need-this.md) — the audit voice. The audiences distinction (programmatic vs. assistant) and the over-engineering caution applied directly here.
- [`03-discipline-where-the-llm-cant-skip-it`](03-discipline-where-the-llm-cant-skip-it.md) — chokepoint principle. Symbol-existence resolution as a mechanical check survives because it does not depend on the LLM remembering to enforce.
- [`04-governance-provenance-and-the-pre-pr-tier`](04-governance-provenance-and-the-pre-pr-tier.md) §4 — modular opt-in. The future render module described in §6 lives at this layer if it ever ships.
- [`05-where-state-lives`](05-where-state-lives.md) — the layer model. The framework's posture toward adjacent state stores (git, ADR docs, build pipelines, code-graphs) is consistent across layers.
- [`07-state-not-workflow`](07-state-not-workflow.md) §4.1 — state-vs-render boundary. The cleanest principled lens on why a code-graph isn't aiwf's state.
- The pocv3 contracts plan ([`docs/pocv3/contracts.md`](../pocv3/contracts.md)) — the practical surface this document constrains. I4's symbol-level live_source is the only concrete code-aware extension authorized.

---

## In this series

- Previous: [`10 — Spec-based development is waterfall in disguise`](10-spec-based-as-waterfall.md)
- Synthesis: [working paper](../working-paper.md)
- Reference: [`KERNEL.md`](KERNEL.md)
