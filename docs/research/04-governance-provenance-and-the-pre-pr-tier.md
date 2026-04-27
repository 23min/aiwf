# Governance, provenance, the project-shape spectrum, where CRDTs actually fit, and the pre-PR tier

> **Status:** research / synthesis. Not normative. Fifth in the series; reads on top of `00-fighting-git.md`, `01-git-native-planning.md`, `02-do-we-need-this.md`, `03-discipline-where-the-llm-cant-skip-it.md`, and `KERNEL.md`.
> **Audience:** the user, after they've named several gaps in the prior synthesis: governance/provenance UX is undertreated; one-size-fits-all is wrong; CRDTs may be the right tool for *part* of the solution; the "in-between" tier between studio and museum has poor tooling; and the Human-in-the-Loop may belong pre-PR rather than at PR-time.
> **Premise:** the framework should be opt-in along a spectrum of project shapes; CRDT primitives address a specific subset of needs precisely; governance and provenance need first-class UX, not side effects; and most reconciliation work belongs on the client side while the AI is still in conversation, not on the PR.

---

## 1. What this document is responding to

The user, in the prior turn, raised five distinct threads that the prior synthesis did not address well:

1. **Governance and provenance UX.** The prior docs treated provenance as a side-effect of whatever store happened to exist (events.jsonl in `00`, git log in `01`, structured trailers in `03`). Governance was barely named. Both deserve first-class treatment with a UX worth using.

2. **The 2026 landscape.** Many adjacent voices are wrestling with the same family of problems. We should know what they're saying so we don't reinvent and so we can position the framework against them.

3. **One-size-fits-all is overly ambitious.** Solo↔team, short↔long, greenfield↔brownfield, unregulated↔regulated all have legitimately different needs. The framework is opt-in by design; the design must take that seriously rather than implicitly target one project shape.

4. **CRDTs for part of the solution.** The user has an intuition that CRDT-end rigor has promise in a bounded subset. Engage with that seriously rather than waving toward Automerge generally.

5. **The pre-PR tier and HITL placement.** Current PR review tools are weak (essentially comments + conflict markers, host-dependent). But the most important reconciliation work happens *before* push, while the AI is still in conversation and local tools are available. The framework should make pre-PR tooling first-class, not treat the GitHub/GitLab/Azure DevOps PR UI as the canonical Workshop.

This document treats each thread, then proposes a concrete shape that integrates them.

---

## 2. Governance and provenance — first-class, not a side effect

### 2.1 What these words mean here

To avoid muddle, distinguish:

- **Provenance** — *who did what, when, why, and on the basis of what authority.* It answers "where did this come from?" both for entities (this milestone was created in PR #142 by Peter, citing ADR-0042) and for changes (this status transition happened in commit a4f8c21 by claude/session-abc, validated against the `milestone` FSM).
- **Governance** — *what is allowed to change, by whom, under what process, with what review.* It answers "is this change allowed?" (only repo maintainers can promote a milestone to `complete`; FSM transitions must be PR-reviewed; ADRs cannot be silently superseded; framework-version upgrades require an issue link).

Provenance is *historical* — looking back at what happened. Governance is *prospective* — controlling what can happen.

The two are linked: governance rules generate provenance records (e.g., "promotion required two reviewers" leaves a record naming the reviewers). Good UX for one informs the other.

### 2.2 Why both have been undertreated so far

Earlier docs assumed:

- Provenance = the event log (00) or `git log` + commit trailers (01, 03). These are storage answers, not UX answers.
- Governance = whatever CI checks for (03). This is enforcement, not policy declaration or audit.

Neither addresses the user's actual question: *when I, as a human, want to ask "why is this milestone scoped this way?" or "who decided to defer this gap?" or "can I even change this?", what do I do?* The current architecture's answer is "read the event log" or "read `git log`" — both of which are unappealing.

### 2.3 Provenance UX — what good looks like

A first-class provenance UX should make these queries trivial:

- **Per-entity timeline.** "Show me everything that happened to `M-005`": creation, scope edits, status changes, dependency edits, reviewer comments, related decisions, related ADRs, commits that closed it. Rendered chronologically with actors and rationales.
- **Per-decision provenance.** "Why is this ADR the way it is?": the discussion that preceded it (PR comments, linked issue), the decision itself, what it superseded, what cites it now.
- **Per-change explanation.** "This frontmatter field changed in this commit — show me the verb that changed it, the actor, the rationale, and any reviewers." If the change came in via `aiwf promote`, the structured commit trailer carries the verb and inputs. If it came in via direct edit, that fact is part of the provenance (and possibly itself a finding).
- **Cross-entity provenance.** "Show me every entity affected by ADR-0042" or "every milestone that cites G-2026-04-26-003."

The data to support these is already present in any of the storage models the prior docs proposed. The missing piece is the *renderer* — `aiwf history`, `aiwf trace`, `aiwf affected-by` — and the *commit-message discipline* that makes the underlying data legible.

This is a small body of work that would deliver disproportionate UX value. It belongs in the kernel of the framework, not in an optional module.

### 2.4 Governance UX — what good looks like

A first-class governance UX should make these surfaces clear:

- **What can I do here?** Given the current entity and the current actor, list the legal next operations. The verb `aiwf transitions M-005` should answer "what state changes are allowed; what would each require (e.g., reviewer approval, prerequisite milestone closure, ADR ratification)."
- **What's required for this PR to merge?** Given a PR's diff, surface every governance check that will run, with current pass/fail status. Today a PR shows "8 checks passing" — opaque. A good surface lists the *substantive* checks: "FSM transitions valid ✓ / no id collisions ✓ / two reviewers required ✗."
- **Who decided this rule?** Governance rules are themselves entities (ADRs, contracts, or framework principles). The UX should let any rule be inspected: "this transition requires two reviewers because of ADR-0042, ratified in PR #87."
- **Override path.** Governance must be overridable in genuine emergencies, with the override itself being a high-provenance event (named approver, named reason, follow-up issue). Currently, `--no-verify` and "merge without checks" are the override mechanisms — silent, ungoverned, untraceable.

The user's chief concern (`03`, §1) about skills being non-deterministic applies tenfold to governance: if "review" or "approval" lives only as a convention the LLM is supposed to follow, it isn't governance. Governance only exists where it's mechanically enforced and visibly surfaced.

### 2.5 Storage implications

For provenance to render well:

- **Per-mutation actor + rationale must be captured at write time.** The verb (or the commit trailer) records `actor`, `verb`, `inputs`, optionally `rationale`. Direct edits without this metadata are still allowed, but produce lower-fidelity provenance — and that fidelity loss is itself visible.
- **External-system links must be first-class.** "This was discussed in Linear ENG-1234" or "this came out of PR #87" should be machine-resolvable and renderable, not buried in prose.
- **Tombstones (from `03`) must carry full provenance.** Why the entity was removed, by whom, in which PR, what supersedes it.

For governance to enforce well:

- **Rules must be declarative and citeable.** A YAML contract that says "milestone promotion to `complete` requires two reviewers" is also the answer to "why does this need two reviewers?" — the rule and its justification colocate.
- **Enforcement must be at chokepoints (CI, branch protection, server-side hooks).** Per `03`. Skills that "remind" the LLM to ask for approval are not governance.

### 2.6 The minimal shape

Five additions to the framework deliver the bulk of provenance/governance UX:

1. `aiwf history <id>` — per-entity chronological timeline.
2. `aiwf trace <commit-or-pr>` — what entities/rules a change touched, with full lineage.
3. `aiwf transitions <id>` — what's allowed next, with the rule and the rule's source.
4. `aiwf affected-by <rule-or-decision>` — reverse lookup.
5. **Structured commit trailers** as the substrate (`aiwf-verb:`, `aiwf-actor:`, `aiwf-entity:`, `aiwf-rationale:`).

None require an event log. None require a graph projection. All work over the working tree + `git log`.

---

## 3. The 2026 landscape — what others are saying

Honest scope note: the AI-coding-agent end of this space is moving very fast. Most relevant primary sources are product docs, blog posts, and GitHub READMEs rather than peer-reviewed papers. I am confident about *what* exists; I am less confident about subtle differences between competing tools. Anyone acting on this list should verify the specific claim against current sources.

### 3.1 Spec-driven and PRD-as-code tools

Several tools have emerged in 2024–2026 trying to give AI agents (and humans) a structured plan to share:

- **GitHub Spec Kit** (`spec-kit`) — a methodology + CLI for "spec-driven development" with AI agents. Treats specs as the primary artifact; code is generated/maintained from them. Lives in the repo.
- **Kiro** (AWS) — agentic IDE that puts spec, design, and tasks into the repo as markdown alongside code. Explicitly multi-stage: spec → design → tasks → implementation, with the AI guided by the current stage.
- **Tessl** (Tessl Inc) — "spec-centric" coding workflow. Specs are first-class, code is verified against them.
- **Block's Goose** — agentic toolkit with planning surfaces; less spec-rigid but takes "plan in repo" seriously.
- **Sourcegraph Amp / Cody** — agent platforms whose latest iterations include in-repo plan persistence patterns.
- **Claude Code itself** — the framework being built here is one expression of the broader pattern; `CLAUDE.md`, `.claude/skills/`, agents and slash commands are Anthropic's hooks for the same problem.

What these share: they all put planning artifacts in the repo, alongside code, in markdown, and ask the agent to read+update them. None (that I know of) have solved the merge problem rigorously; most live with it and accept that branches diverge.

### 3.2 Memory and context for agents

Distinct from spec-driven planning — these are about session-spanning context for the agent:

- **Cursor's project memory** and **Continue.dev's context** — IDE-integrated memory that survives across sessions.
- **Aider's repo-map** — a compact representation of the codebase the agent uses for context.
- **Cline's "memory bank" pattern** — community-popular convention of a `memory-bank/` directory of curated markdown context files that the agent reads first.
- **Specstory** and **session-capture tools** — record AI-coding session transcripts as artifacts, sometimes reviewable, sometimes searchable.
- **Anthropic's auto-memory** (which is in use right now in this very conversation) — file-based, git-friendly, scoped to a project.

Pattern: convention + small format + lots of reads, few writes. Notable for *not* trying to be a database. The framework being built here could draw from this rather than competing with it.

### 3.3 Agent rule files and their divergence problem

Every major agent platform now has a per-repo rules file with the same fundamental design:

- `CLAUDE.md` (Claude Code)
- `.cursorrules` (Cursor) — and the newer `.cursor/rules/` directory format
- `CONVENTIONS.md` (Aider)
- `.github/copilot-instructions.md` (GitHub Copilot)
- `.continuerc.json` + context files (Continue.dev)
- `.windsurfrules` (Windsurf)
- `AGENTS.md` (a community proposal for a cross-tool standard)

All git-tracked. All branch-divergent. None with a formal merge story. The same problem in every tool. This is a genuine gap in the ecosystem; a thoughtful answer here would be a contribution.

### 3.4 Linear, GitHub Projects, Jira, and the "PM external" path

Mature project-management tools continue to add AI integrations:

- **Linear's MCP server and GitHub-native workflow.** Tickets ↔ commits ↔ branches are tightly coupled; AI can query.
- **GitHub Projects + GitHub Issues.** Entirely git-adjacent, with native PR linking.
- **Jira's recent agent integrations.** Atlassian's "Rovo" and similar.

These are the "PM external" path from `02`. They are mature, multi-user, governance-rich, and provenance-rich. The framework should treat integration with them as a real option for projects that already use them, not pretend they don't exist.

### 3.5 CRDTs and merge-aware collaborative state

Production-grade CRDT systems continue to mature:

- **Automerge 2/3** — JSON CRDT, the most-cited reference implementation.
- **Yjs** — text and structured CRDTs, widely used in collaborative editors.
- **Loro** — newer (2024) CRDT runtime, performance-focused.
- **Jazz** (jazz.tools) — local-first app framework with CRDT primitives.
- **Triplit** — collaborative database with relational sync.

These are not in the AI-coding space directly but are the *substrate* one would use if pursuing the Tier 2 / Lift direction from `00-fighting-git.md`.

### 3.6 Versioned databases (the "git for data" lineage)

- **Dolt** — git-for-SQL. Branch, merge, diff, blame, all on tables.
- **TerminusDB** — branched graph database.
- **XTDB** — bitemporal database with valid-time + transaction-time.
- **Datomic** — immutable history, point-in-time queries; not branchy in the git sense.

Useful as design lineage, especially for the user's "merging state machines, time machines, interleaving event logs, oracles" framing in the original conversation.

### 3.7 Methodology debates

Less tooling, more discourse:

- **"AHA" (Architecture Hypothesis Action)** and similar — methodology pushback against waterfall-flavored AI coding.
- **The "spec-first vs. code-first" debate** — whether AI should generate code from specs or evolve specs alongside code.
- **The "agent autonomy" spectrum** — full autonomy (Devin-style) vs. tightly-coupled HITL (Aider-style) vs. middle (Claude Code-style).

The framework being built is implicitly choosing tightly-coupled-HITL with strong process discipline. That should be named; many adjacent projects make different choices.

### 3.8 What the framework can borrow

- **From spec-driven tools** — the convention of staged artifacts (idea → spec → design → tasks → done) and the discipline of updating them as work progresses.
- **From memory systems** — convention-over-configuration, small markdown files, idempotent reads.
- **From rule-file tooling** — the recognition that agent behavior is itself a versioned artifact and merges of agent rules are an unsolved problem worth solving.
- **From PM tools** — first-class governance/provenance UX (notifications, mentions, approval flows). The framework's UX should aspire to this even though the substrate is files, not a server.
- **From CRDTs** — primitives where they fit; see §5.
- **From versioned databases** — the model that branch-and-merge can be applied to structured data, not just text.

### 3.9 What the framework can contribute back

The framework's distinctive concerns — referential stability through tombstones, rule-divergence at merge, HITL placement before push — are not well-served by any individual tool listed above. A clean articulation of these problems and a working answer would be a contribution to the broader space, not a derivative.

---

## 4. The project-shape spectrum — opt-in, not prescriptive

### 4.1 The four axes

Genuinely different shapes need genuinely different framework subsets:

| Axis | One end | Other end |
|---|---|---|
| Team size | Solo | Large team |
| Horizon | Days/weeks | Months/years |
| Brownfield depth | Greenfield | Legacy with deep history |
| Regulation | Unregulated personal | Regulated (HIPAA, SOX, ISO 27001, etc.) |

A single project sits at one point on each axis. A solo greenfield short-horizon unregulated project (e.g., a weekend hackathon) and a large-team brownfield long-horizon regulated project (e.g., a bank's core ledger rewrite) need *different framework features*.

### 4.2 What changes along each axis

**Team size.** Solo: provenance is for future-self; merge conflicts are rare; the LLM and human are tightly coupled. Large team: provenance is for coworkers and auditors; merges are constant; many AI sessions concurrent; governance is enforcement, not self-discipline.

**Horizon.** Short: plan is mostly held in head; framework is overhead. Long: plan must be persisted because no human remembers everything; framework is essential.

**Greenfield vs. brownfield.** Greenfield: framework can shape the project from day one; conventions are easy to adopt. Brownfield: framework must coexist with existing conventions, existing CI, existing PM tools; must not require migration of historical data.

**Regulation.** Unregulated: provenance is a nicety; deletions are fine; "good enough" is good enough. Regulated: provenance is mandatory and audit-grade; deletions are forbidden (or strictly governed); every change must be attributable; specific frameworks (e.g., 21 CFR Part 11 for FDA) impose specific requirements.

### 4.3 Implications

- **The kernel must be small enough that solo-greenfield-short-unregulated finds it useful.** If the smallest viable use is heavyweight, the framework excludes its most numerous potential users.
- **The framework must be modular enough that large-team-brownfield-long-regulated can compose what it needs without dragging in everything.** Modules should be opt-in, ideally enableable after the project is started.
- **There is no "default config."** There is a kernel that is always on; everything else is per-project enable/disable in `.ai-repo/config/`.
- **The framework must integrate with external tooling, not replace it.** Brownfield projects already have Jira, GitHub Projects, an internal ADR convention. The framework adapts; the project doesn't.

### 4.4 Module candidates

Drawing from the prior research and this spectrum, a likely module decomposition:

| Module | Core or opt-in | Best for |
|---|---|---|
| `ids` (allocator + tombstones + rename) | Core | Everyone |
| `verify` (CI checks) | Core | Everyone |
| `epics-milestones` | Opt-in (recommended) | Long-horizon, multi-milestone projects |
| `adr` | Opt-in (recommended) | Anything with architecture decisions worth recording |
| `roadmap` (rendered) | Opt-in | Anyone who wants a generated overview |
| `gh-sync` (GitHub Issues) | Opt-in | Brownfield + GitHub-native |
| `linear-sync` | Opt-in | Brownfield + Linear |
| `audit` (regulated-grade provenance) | Opt-in | Regulated industries |
| `hitl-prepush` (the pre-PR tier; see §6) | Opt-in (recommended) | AI-heavy workflows |
| `crdt-registry` (id allocator as CRDT; see §5) | Opt-in | Long-running multi-branch projects |

The kernel is `ids` + `verify`. Everything else composes.

---

## 5. Where CRDTs actually fit

The user's intuition is right: CRDTs have promise *in a bounded subset* of this framework's needs. Naming the subset precisely is more useful than waving toward "Automerge for everything."

### 5.1 Quick refresher

A CRDT (Conflict-free Replicated Data Type) is a data type whose operations are designed to commute, so concurrent replicas converge to the same state regardless of the order in which they observe each other's changes. There are two flavors:

- **State-based (CvRDT)** — replicas exchange full state and merge via a join operation.
- **Operation-based (CmRDT)** — replicas exchange operations; commutativity guarantees order-independence.

Common primitives: G-Set (grow-only set), 2P-Set (add + tombstone), G-Counter, PN-Counter, OR-Set (observed-remove), LWW-Register (last-writer-wins), Multi-Value Register, Sequence CRDTs (RGA, LSEQ, Yjs's structure), JSON CRDT (Automerge).

### 5.2 What in this framework is CRDT-shaped

Walking the kernel needs against CRDT primitives:

**The id allocator namespace** — naturally a **G-Set** (grow-only set of allocated ids) plus a **2P-Set** (add to live; move to tombstone; never resurrect). Merge by union. Two branches both allocating `E-019` shows up as a deterministic collision finding — no silent corruption. This is the cleanest fit; even if nothing else CRDT-flavored is adopted, the id registry should be modeled this way.

**Tombstones** — naturally a **G-Set** (once tombstoned, always tombstoned). Reuse-of-id is a violation that the merge step detects.

**The reference graph (depends_on, parent, supersedes, cites)** — naturally an **OR-Set** per entity per relation. Add-wins or remove-wins is a per-relation policy choice. Cycles after merge are a finding, not silently allowed.

**Status as a register over a partial order** — `draft < active < complete` is a partial order; status merges by **least-upper-bound**. Two branches both promoting `M-005` from `draft` to `active` and `complete` respectively: merge to `complete`. Two branches promoting in incompatible directions (e.g., `complete` and `cancelled`, both terminal): merge surfaces a conflict. By construction this prevents `complete → in_progress` regressions.

**Annotations / labels / tags** — pure G-Set or 2P-Set per entity. Trivial.

**The set of entities themselves** — G-Set (entity exists) plus per-entity tombstone. Same shape as ids.

### 5.3 What in this framework is NOT CRDT-shaped

Equally important — naming where CRDTs do *not* belong:

- **The prose body of an entity.** Sequence CRDTs (Yjs, Automerge text) exist, but for prose that humans write and edit deliberately, git's text merge is adequate and human-resolvable conflicts are *good* (they prompt review). A CRDT here would silently auto-merge prose in ways the human didn't intend.
- **The human-friendly id (`E-19`).** This must be sequential and short; the CRDT-friendly version (ULID, content hash) is hostile to the user's stated requirement.
- **Decisions and rationales.** Their meaning is human; their merge is human judgment. The framework can detect that they changed; it should not auto-resolve.
- **Cross-branch ordering of unrelated decisions.** "Did we decide A before B, or B before A?" is genuinely incomparable across branches. Any storage that pretends otherwise is lying.

### 5.4 The hybrid shape

A working answer looks like:

- **CRDT-modeled metadata layer** — id registry, tombstones, references, status, annotations. This is small, fits in one file or a small directory, can be encoded as plain JSON/YAML with CRDT semantics enforced by the engine on read (no need for a binary CRDT runtime if the structures are simple enough — the engine just applies the merge function deterministically when it sees concurrent edits).
- **Plain markdown for everything else** — prose bodies, ADR contents, narrative roadmap. Git-merged.
- **CI verifies the merge result is consistent.** If the metadata layer's merged state is invalid (cycle, illegal transition, etc.), CI surfaces a finding.

This is *not* "use Automerge for the planning state" — that would be heavy-handed. It is "use CRDT *primitives* where they precisely match the data shape, and stick with markdown elsewhere." The engine's merge logic for the metadata is small (a few hundred lines of Go, modeled on the standard CRDT primitives).

### 5.5 Why this is more promising than full CRDT-substrate

- **Keeps the on-disk format human-readable.** No binary files; no opaque substrate.
- **Keeps git as the time machine.** No parallel history mechanism.
- **Solves the parts that genuinely need it.** Id collisions, status merge, reference convergence — these are exactly where merge problems hurt.
- **Leaves human judgment for human concerns.** Prose conflicts still surface; humans still decide.
- **Composable.** The CRDT-registry can be its own module (see §4.4), opt-in for projects that need it.

### 5.6 The key reference

Anyone implementing this should read **Shapiro et al., "A Comprehensive Study of Convergent and Commutative Replicated Data Types"** (Inria 2011) for the primitive catalog, and **Kleppmann & Beresford 2017** for the JSON-shaped composition. Loro's documentation is also a clean modern introduction. Beyond that, the implementation is small enough that primary research is not needed — this is well-trodden ground.

---

## 6. The pre-PR tier — moving HITL where the AI is still in the loop

### 6.1 The user's observation, sharpened

PR review tools (GitHub, GitLab, Azure DevOps, Bitbucket) are weak. They offer:

- Inline comments threaded on diff hunks.
- Conflict markers in files (and a simple in-browser conflict resolver for trivial cases).
- Approval/rejection.
- Required checks (CI gates).
- @-mentions and notifications.

What they do *not* offer:

- An AI-in-the-loop conversation about the changes.
- Live re-running of validators with proposed fixes.
- Quick try-this-fix-and-see-if-it-passes loops.
- Semantic conflict resolution (only textual).
- Anything tailored to this framework's domain (referential integrity findings, FSM transition explanations, etc.).

Meanwhile, before the PR exists, on the user's machine, *all of those things are available*: the AI is in active conversation, local tools run instantly, the git history is mutable, and the human's attention is already engaged.

The user's question — "isn't half of the PR done on the client side, while still on the branch?" — is exactly right. Most reconciliation work *should* happen there. The PR should be a checkpoint that confirms work already done well, not the place where work first gets done.

### 6.2 Pre-PR tier: what it is

Re-tier the model from `03`:

| Sub-tier | Locus | Tools |
|---|---|---|
| Studio (raw) | Local branch, early iteration | AI + IDE + filesystem |
| Pre-PR Workshop | Local branch, preparing to push | AI + framework verbs + local validators + git tooling |
| PR Workshop | After push, GitHub/GitLab UI | CI + reviewers (human + AI) + comments |
| Museum | After merge to main | Branch protections, audit |

The Pre-PR Workshop is where the framework can do its best work, because:

- The AI has full context (ongoing conversation).
- Local tools run in milliseconds, not minutes.
- History is still mutable (rebase, squash, fixup).
- Failures cost a re-run, not a re-push and re-CI cycle.
- The human is engaged, not async.
- No webhook latency, no notification fatigue.

The PR Workshop is then narrower in scope: it confirms the Pre-PR work, gathers asynchronous reviewer input, gates merge. It is genuinely necessary (especially for multi-person teams) but it should not be where the framework expects most reconciliation to happen.

### 6.3 What pre-PR tooling looks like

A `aiwf prepush` (or `aiwf review`) verb that runs locally and produces a structured "PR-readiness report":

- All `aiwf check *` validators run; findings listed.
- Each finding has a "fix" suggestion the AI can apply.
- The AI can iterate: "fix all auto-fixable findings, then re-run and report."
- A summary suitable for the PR description is generated.
- A pre-push git hook ensures `aiwf prepush` was run successfully before push (with `--skip` requiring a reason).

A `aiwf preview-merge` verb that simulates the merge into the target branch and reports:

- File conflicts (textual).
- Semantic conflicts (per the CRDT layer in §5: id collisions, status conflicts, reference cycles).
- New referenced entities that don't exist on target.
- Suggested resolutions for each.

A `aiwf rebase --aware` wrapper that walks rebase steps with framework awareness — at each step, validate; if validation fails, prompt the user (or AI) to fix before continuing.

### 6.4 What changes in the PR Workshop

If pre-PR tooling does its job, the PR Workshop becomes:

- A *summary* (the report from `aiwf prepush`, posted as PR description).
- A *signature collection* (reviewer approvals).
- A *governance gate* (CI re-runs the same checks server-side; it can't be lied to).
- A *discussion forum* for the irreducibly human parts: "is this the right scope?", "should we defer?".

The diff view, comment threads, and approval UI — the parts current PR tools are good at — remain useful for the discussion forum. The reconciliation work is already done.

### 6.5 What the framework cannot fix about PR review tools

Some of the user's frustration is structural: PR review UI is host-dependent, and the framework cannot replace GitHub's UI. What the framework *can* do:

- Make the host's UI carry the framework's signal effectively (e.g., generate a PR description that includes the readiness report and the affected-entities list).
- Provide a `aiwf review-pr <pr-number>` CLI/skill that lets a human or AI review a PR locally with full framework awareness, rather than depending on GitHub's web UI.
- Offer a `aiwf comment <pr> <hunk> <text>` that posts framework-aware comments back to the host.
- Provide an MCP server or skill that lets agents (Claude, others) query and act on PRs via the host's API with framework semantics.

None of this displaces GitHub. It makes GitHub adequate for what it does, while moving the heavy work to where it belongs.

### 6.6 The HITL placement principle

State as a principle for the framework:

> **Place the human-in-the-loop step where the AI is also in the loop and local tooling is available — i.e., before push, on the branch — by default. Treat the PR as a confirmation gate plus an asynchronous discussion forum, not as the primary workshop.**

This is opt-in (per §4) — teams that want the PR to be the primary review locus can configure that. But the default biases toward the place where reconciliation is fast, contextual, and AI-assisted.

### 6.7 The "I don't want discipline; I just want it to work" path

The user noted: *"I suppose some people want to just automate everything and are not really disciplined and just care about getting things done regardless."* The framework should accommodate this honestly:

- Provide a `aiwf prepush --auto-fix --auto-commit` mode that applies all auto-fixable findings and commits, without prompting.
- Make the pre-push hook configurable: warn-only (disciplined-but-pragmatic) vs. block (disciplined-strict) vs. off (autonomy-first).
- Document the trade-offs visibly.

The framework's principle is not "everyone must be disciplined." It is "discipline should be *available* and *easy*; the absence of discipline should be a configured choice, not an accidental one."

---

## 7. How this updates the kernel

The kernel doc (`KERNEL.md`) was extracted before this synthesis. It does not need substantive change — the eight needs and cross-cutting properties already stand — but two cross-cutting properties should be sharpened or added:

**Sharpen** the existing "Soft in studio, strict at the gate" to: "Soft in raw studio; AI-assisted strictness pre-push; mechanical strictness at the PR gate; sealed at main." (The "gate" was previously implicitly the PR; now it's a chain of two.)

**Add** a property: "**Modular and opt-in.** A small kernel everyone can use, plus modules each project enables based on its shape on the team-size, horizon, brownfield-depth, and regulation axes."

**Add** a property: "**Governance and provenance are first-class UX, not side effects.** The renderers and queryable surfaces for who-did-what-and-why and what-can-change-here are core, not optional."

I'll not edit `KERNEL.md` from this document — that's for a deliberate edit with its own commit. Naming the proposed updates here so they're visible.

---

## 8. The integrated picture

Combining the prior research with this document's threads:

- **Storage**: markdown files for prose; small CRDT-modeled metadata layer (id registry, tombstones, status, references) for the parts that need merge-awareness; no separate event log; no separate graph projection.
- **Identity**: stable ids separated from display names; tombstones for removed entities; CRDT registry for collision-detection across branches.
- **Provenance**: structured commit trailers; renderers (`aiwf history`, `aiwf trace`, `aiwf affected-by`) over `git log` + tree state; first-class UX, not buried in storage.
- **Governance**: declarative rules in YAML contracts; mechanical enforcement at CI + branch protection; queryable surfaces (`aiwf transitions`); explicit override paths with provenance.
- **Validation**: stateless `aiwf check *` that runs over the working tree; same checks run pre-push and on CI.
- **Tier model**: Studio (raw branch) → Pre-PR Workshop (branch + AI + local validators) → PR Workshop (CI + async review) → Museum (main).
- **Modularity**: small kernel (`ids` + `verify`); everything else opt-in via `.ai-repo/config/modules.yaml`.
- **External integration**: optional sync modules for GitHub Issues, Linear, Jira; framework adapts to project's existing tools rather than demanding replacement.
- **AI-rule divergence**: acknowledged and treated as a versioned-artifact problem; merge of skill/rule changes is reviewed in the PR Workshop, with the framework able to detect and highlight rule changes specifically.

---

## 9. Open questions this document does not close

1. **What's the exact on-disk format for the CRDT metadata layer?** Plain JSON with CRDT semantics in the engine, or a Loro/Automerge document? Trade-off: human-readability vs. battle-tested merge.
2. **What's the right granularity for governance rules?** Per-kind in contract YAML is clear; cross-kind rules ("any milestone with status `complete` and a referenced ADR with status `superseded` triggers a finding") are murkier.
3. **How exactly does the framework integrate with external PM without becoming a sync-engine maintenance burden?** Each adapter is non-trivial; how many is the framework willing to maintain?
4. **Does the pre-PR tier need its own dedicated UI, or is it sufficient as a CLI + skill?** A small TUI for "walk through the readiness report" might add UX value beyond plain text.
5. **What's the migration story for projects mid-flight?** Greenfield is easy; a project with months of `.ai-repo/events.jsonl` already needs a way in.
6. **How does this interact with multi-repo work (monorepos with multiple sub-projects, or coordinated multi-repo product lines)?** Out of scope here, but eventually load-bearing.

These belong in subsequent research docs or in proposals.

---

## 10. Where this leaves us

Five research docs in (counting `KERNEL.md` separately as a reference, not a research narrative), the picture is:

- `00-fighting-git.md` — events.jsonl + hash chain fight git.
- `01-git-native-planning.md` — drop the event log and graph; markdown + git suffice.
- `02-do-we-need-this.md` — question whether a framework is needed at all.
- `03-discipline-where-the-llm-cant-skip-it.md` — if a framework is needed, its real value is enforcement at chokepoints.
- `04` (this document) — governance/provenance UX, project-shape modularity, CRDTs for the parts that need them, pre-PR tier as the right HITL placement.
- `KERNEL.md` — the eight needs and cross-cutting properties, against which all proposals should be evaluated.

A concrete shape is now visible: a small kernel (`ids` + `verify`) with a CRDT-modeled metadata layer; first-class provenance/governance UX renderers; pre-PR tooling that does most reconciliation while AI is in the loop; PR Workshop as confirmation gate plus async discussion; modular opt-in for everything beyond the kernel; honest accommodation of the "I just want it to work" path alongside the "I want full discipline" path.

The next step, when you're ready, is not more research — it is choosing one or two modules from this picture and writing concrete architecture proposals against them, then implementing the smallest viable version. The candidates with highest leverage are:

1. The kernel module: `ids` (allocator, tombstones, rename) + `verify` (the four-to-five `aiwf check *` commands) + the CI workflow that runs them.
2. The provenance renderers (`aiwf history`, `aiwf trace`).
3. The pre-PR tier verb (`aiwf prepush`).

Each is a few weeks of work. Each delivers immediate value. Each can be adopted independently. Each is consistent with the kernel and with the prior research's constraints. The framework, in its real shape, is starting to look very tractable.
