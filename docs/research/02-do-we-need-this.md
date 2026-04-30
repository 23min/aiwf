# Do we even need this framework? — questioning the premise

> **Status:** defended-position
> **Hypothesis:** A custom AI-aware planning framework is overdesigned for most teams; ADRs plus a curated context document plus a habit (Shape A) solves 80% of the forgetful-AI problem with two weeks of work, and the right discipline is to build that first and let real friction surface what — if anything — needs to grow.
> **Audience:** the person about to commit months of engineering to this codebase. Read before that.
> **Premise:** the original architecture is taken as given and audited against whether its ambition is justified by the problem actually being solved.
> **Tags:** #thesis #aiwf #research

---

## Abstract

[00](https://proliminal.net/theses/fighting-git/) and [01](https://proliminal.net/theses/git-native-planning/) take "we are building a framework" as given and ask how to build it well. This document does not. It audits the premise — does the framework, in its current ambition, deserve to exist? — and finds the premise wobblier than the prior docs assumed. AI assistants do not need structured state to reason about prose-bearing planning artifacts; "semantic determinism" is not achievable for prose; the framework cannot pretend to provide it. What's actually needed is a *discipline* (ADRs for decisions, a curated context document, a habit of reading and updating both) plus a tiny linter that checks well-formedness — roughly 200 lines of code, two weeks of work. The doc proposes three alternative shapes (Shape A: convention-and-skill, Shape B: hybrid with external PM, Shape C: the full git-native framework) and recommends building Shape A first, using it for three months, and letting real use surface what's actually missing. The full architecture is deferred until the requirement is proven, not before.

---

## 1. What the user actually asked

Reproduced because the framing matters:

> AI Assistants like context, things that are planned, decided, committed, etc. All that is otherwise in git issues or in the user's head needs to be accessible, and when changes are made, we need some kind of consistency forward, so that things don't lose their meaning over time, semantic stability.
> Do we even need an ai workflow framework? Is there such a thing as "semantic determinism"? Do we need to teach the AI how to forget?
> Using the clay modeling metaphor, clay has some history but it can be completely rewritten… I am trying to use "soft" metaphors together with determinism and hard requirements. Is it an abstraction layer or a fundamental incompatible way of thinking?
> Should we keep the project management completely out of the repo? But then how can the AI assist?

There are six distinct questions in there. I'll answer each, then propose three alternative shapes the project could take.

---

## 2. Did the original premise even hold?

The original premise, as it appears in `docs/architecture.md` and as I implicitly reinforced in `git-native-planning.md`, is approximately:

> *AI assistants in long-horizon collaborative work need a structured, machine-validatable, git-tracked representation of project state, with strong consistency guarantees and a closed vocabulary, so that they can reason coherently about what's planned/decided/in-flight without losing fidelity over time.*

This premise has three load-bearing claims, each of which deserves audit:

**Claim A — AI assistants need *structured* state, not just prose.**
Mostly false in 2026. Modern frontier models reason about prose with high fidelity, and most "state" the assistant needs is qualitative: what was decided, what's the current goal, what was tried and rejected. A well-written `DECISIONS.md` of 200 lines beats a normalized graph of 50 entities with 5 edge types for almost every actual question the assistant gets asked. Structured state pays off for *programmatic* consumers (CI gates, dashboards, audit), not for AI assistants. Confusing these two audiences inflates the design.

**Claim B — State must be *git-tracked* to be useful to the assistant.**
Partly true, partly tautological. The assistant is invoked from a checkout; whatever is in the checkout is trivially accessible. But "in the checkout" does not require "managed by a custom framework with its own database." A markdown file is in the checkout. A submodule is in the checkout. A `.env`-style pointer to an external tool is in the checkout. The framework's design conflates "I need this accessible from the checkout" with "I need to model this in my own data store inside the checkout."

**Claim C — Consistency-forward and "semantic stability" are achievable engineering goals.**
**Mostly false as stated.** This deserves its own section; see §3.

If A and C are wobbly and B is weaker than it looks, the original premise has been over-engineered. The question is not "how do we build the framework correctly" but "what is the smallest thing that actually helps?"

---

## 3. Is "semantic determinism" a real thing?

No, not for the things this framework deals with. Let me be precise about why.

There are several kinds of stability/determinism a system might offer:

- **Byte-for-byte determinism** — the same input produces the same output. Achievable, useful, narrow.
- **Referential stability** — an identifier (e.g., `E-19`) always points to the same entity over time. Achievable. The framework already aims for this.
- **Structural consistency** — the relationships between entities remain valid (no dangling references, no cycles). Achievable. `aiwf verify` does this.
- **Schema stability** — the shape of the data does not change unexpectedly. Achievable via versioned schemas.
- **Semantic stability / determinism** — *the meaning of an entity does not drift*. **Not achievable for prose-bearing artifacts.**

The body of an epic spec is prose. Prose drifts. The phrase "the payments rewrite" means something different in March than it meant in January, even if the title and id are unchanged. The framework's hash-verified projection is computed *over the structural fields only*. It says nothing about the prose. So the framework's "determinism" is the determinism of a small structural skeleton, not of the meaning of the work.

This matters because the user's intuition that "things shouldn't lose meaning over time" is doing a lot of work in the architecture, but the architecture cannot deliver on it. The framework can deliver:

- "If you cite `E-19`, that id will always resolve to *some* spec."
- "If you query the dependency graph, the structure will be valid."
- "If you ran `verify` and it was clean, the byte-for-byte projection is reproducible from the events."

It cannot deliver:

- "The project's understanding of what `E-19` is will not change as the team learns."
- "What the team meant by 'in_progress' six months ago is what you'll mean by it today."
- "An ADR's reasoning, read out of the context in which it was written, will mean what it meant then."

These are properties of human (and AI) understanding, not of data structures. Pretending otherwise builds a cathedral on sand.

**Practical implication:** the framework should be honest that it provides *referential and structural* stability, not *semantic* stability. Stop claiming the latter. Adjust verbs and findings accordingly.

---

## 4. Do we need to teach the AI to forget?

Yes, and the current architecture actively prevents this.

`CLAUDE.md` declares "**Immutability of done.** Terminal-state entities (`complete`, `cancelled`) never reverse." This is good for audit. It is **bad for planning ergonomics**.

When a milestone is "complete" but you later realize the diagnosis was wrong, or the scope was misnamed, or the rationale needs rewriting *for the benefit of future readers including the AI*, the framework's stance is "spawn a new entity via `aiwf hotfix`." This produces an audit trail that a forensic auditor would love and that a present-day AI assistant trying to plan tomorrow's work finds *actively confusing*. The AI now sees three entities with overlapping scope and must reconstruct which one is "current" by reading prose.

**Plans need pruning.** Gardens need pruning. Clay gets reabsorbed into the lump. The fact that git can `rebase -i` and rewrite branch history is not a bug; it is a feature that matches the medium. The framework's append-only stance is at war with this.

A more honest model would say:
- **On a feature branch, history is mutable.** Squash, rebase, rewrite freely. The plan is being shaped.
- **On `main`, history is durable but *not* immutable in meaning.** A landed entity can be marked superseded by a later commit; the old commit is still in `git log` for audit, but the *current view* shows only the present meaning.
- **"Forgetting"** is not deletion. It is "this should not appear in the assistant's working set when it asks 'what is the current plan.'" That is a query-time filter, not a data-modification operation.

This reframes the immutability requirement: the *git history* is immutable on main; the *current planning state* is allowed to evolve and prune. Today's framework conflates these.

---

## 5. Soft metaphors meet hard requirements: abstraction or incompatibility?

**Tension, not incompatibility.** The right design names the tension and scopes each half:

| Phase | Medium | Determinism | Audience |
|---|---|---|---|
| Exploration | Clay / branch | Mutable, rewritable | Present human + AI |
| Convergence | Garden / PR | Reviewed, structured | Team |
| Commitment | Stone / main | Append-only audit | Future humans, CI, compliance |

The original architecture applies "stone" semantics uniformly. Every event on every branch goes into the append-only log. Every status transition is final the moment it happens. That makes the studio (where most actual work happens) painful to use, in the name of properties (audit, reproducibility) that mostly only matter at the museum.

A scoped design would make the framework *progressively more strict* as work moves toward main:

- On a personal branch: minimal validation, maximally tolerant, easy to undo, no append-only ledger. The AI helps you iterate. Mistakes are cheap.
- In a PR: validation tightens, references must resolve, structural invariants enforced. The reviewer (human or CI) can see what changes.
- On main: full audit, immutable history (in the git sense), structured commit trailers, downstream rendering. This is the published artifact.

Today's framework demands museum behavior in the studio. That is the felt incompatibility.

---

## 6. Should we keep PM out of the repo entirely?

This is the question with the highest leverage. Let me steelman both sides.

### 6.1 The case for PM out of the repo

**External PM tools (Linear, Jira, Shortcut, Height, Notion, GitHub Projects) already solve every "structured planning" problem the framework is reinventing.** They have:

- Multi-user editing without merge conflicts (because no merging — server is single source of truth).
- Permissions, notifications, integrations, dashboards.
- Mature query languages.
- APIs for AI assistants to consume.
- No git fight, ever.

If the framework's job is "give the AI structured planning context," the path of least resistance is: **let the AI query Linear's API.** The repo carries no planning state. The merge problem evaporates because there is nothing to merge.

What you lose:
- **Bisectability of the plan.** You cannot `git checkout HEAD~50` and see what the plan looked like fifty commits ago. (External tools have history, but it's not co-versioned with the code.)
- **Offline access.** AI working without network can't see the plan.
- **Co-evolution of plan and code.** A PR cannot say "this commit lands milestone M-5" in a way that the framework can verify.
- **Single-source-of-truth for the AI.** It must learn to consult two systems.

What you gain:
- Most of this document.

### 6.2 The case for PM in the repo

**Co-location matters.** The strongest argument for PM-in-repo is the AI's working set. When the assistant is implementing a milestone, it benefits enormously from being able to read the milestone spec, the parent epic, the related ADRs, and the implementing code in the same context window without needing API credentials, network, or impedance-matching tools. The cognitive overhead of "go fetch this from Linear" is nontrivial — and the AI will sometimes skip the fetch, leading to drift.

Co-located PM also enables PR-level invariants: "this PR closes M-5" can be verified by CI; "this PR cites a real ADR" is greppable. External-tool PR conventions (`Closes ENG-1234`) exist, but they're inert text that nothing actually checks against the source of truth.

What you gain from PM-in-repo:
- AI working set is one filesystem.
- PR/CI can enforce plan↔code invariants.
- Bisectable history of plans.
- Works offline, in restricted environments, in ephemeral CI containers.

What you pay:
- The merge problem (`fighting-git.md`).
- A custom framework to maintain.
- Synchronization with external PM if the team uses one anyway.

### 6.3 The middle: crystallized decisions in repo, fluid PM external

**The most defensible position is a hybrid where the repo carries only what the code needs to know about, and external tools carry the rest.**

What lives in the repo:
- **ADRs** (`docs/decisions/`) — decisions that constrain code. Stable, low churn. ADR conventions are mature; just adopt MADR or similar.
- **A thin per-area `CONTEXT.md`** — what this directory is, what we're doing here, what the AI should know. Hand-curated. Updated as part of work. Maybe 50-200 lines per area.
- **A short `ROADMAP.md`** — the next 1-3 milestones, written as prose, manually maintained. Not a database; a reading.
- **Pointers** to the external PM tool: "for the full backlog see Linear project ENG; for sprint planning see…"

What lives external:
- Backlog grooming, sprint planning, task assignment, status updates, comments, notifications, multi-user collaboration, prioritization debates.

What the framework provides (in this hybrid):
- A skill/rule file (`.claude/skills/wf-context/`) that teaches the AI how to use ADRs, `CONTEXT.md`, `ROADMAP.md`, and how to fetch from external PM when needed.
- A small validator: ADRs are well-formed, references resolve, ROADMAP cites real ADRs.
- That's it.

**This is approximately 300 lines of skill content and ~500 lines of Go (mostly markdown parsing). No event log. No graph. No merge driver. No `aiwf` binary in the original ambitious shape.**

This is the boring, mature, well-trodden answer. It is also probably the right answer for most teams.

---

## 7. The forgetful and confused PM problem

The user observed that AI is "like a forgetful and confused PM." This is true and it is the actual problem the framework should solve. Let me reframe what the AI needs:

1. **A reliable place to look for "what we decided"** — so it doesn't re-litigate. ADRs.
2. **A reliable place to look for "what we're doing now"** — so it doesn't wander. `ROADMAP.md` or external PM with a clear pointer.
3. **A reliable place to record "what I just did and why"** — so the next session of itself (or the user) can pick up. Commit messages, PR descriptions, optionally a short `JOURNAL.md`.
4. **A way to detect "this referenced thing no longer exists or means something different"** — so it doesn't quote stale rationale. Lightweight reference validation.
5. **A convention for marking "this used to be true; it isn't anymore"** — so superseded decisions don't anchor future thinking. Standard ADR practice (`Status: Superseded by ADR-NNNN`).

That list does not require an event log, a graph, FSMs, hash-verified projections, contract YAMLs, or a closed entity vocabulary. It requires a *convention* (ADR + a couple of well-known files) and a *skill* (the AI knows where to look and what to update).

**The forgetful AI is fixed by giving it a habit, not by giving it a database.**

---

## 8. Three alternative shapes for the project

### 8.1 Shape A — Convention-and-skill, no engine ("the boring answer")

The framework is just:
- An ADR convention (adopt MADR; no custom format).
- A `CONTEXT.md` convention (50-200 lines per area, hand-curated).
- A `ROADMAP.md` convention (prose, manually maintained, optionally generated from external PM).
- A skill bundle (`.claude/skills/wf-*`) that teaches the AI how to read and update these.
- A 200-line linter (`aiwf lint`) that checks: ADRs are well-formed, citations resolve, no orphans.

No event log. No graph. No merge problem. No identity allocator. No FSMs. No closed vocabulary.

**Effort to build:** ~2 weeks for one person.
**What you lose:** programmatic validation of complex invariants; auto-rendered ROADMAP tables; structured cross-entity queries.
**What you gain:** a usable thing in two weeks, with no maintenance burden, that solves the forgetful-PM problem for 80% of cases.
**Best fit:** small teams, solo developers, projects where PM lives external.

### 8.2 Shape B — Hybrid: in-repo decisions + external PM (the middle)

Same as Shape A, plus:
- A `aiwf sync` verb that pulls a snapshot of "current sprint" from external PM into a `.cache/sprint.md` (gitignored or committed-as-snapshot).
- A skill that knows when to consult the external PM vs. the in-repo ADRs.
- Optional: a slim `aiwf gh` adapter for GitHub Issues, the most common case.

**Effort to build:** ~4-6 weeks.
**What you lose:** anyone using a PM tool the adapter doesn't speak.
**What you gain:** the AI gets the best of both worlds; co-located decisions for code context, external tool for fluid backlog.
**Best fit:** teams that already use Linear/Jira/GitHub Projects and don't want to leave it.

### 8.3 Shape C — The git-native framework as in `git-native-planning.md`

For teams that genuinely want everything in the repo and are willing to pay for the custom framework. Most of the existing architecture's ambition, but rebuilt on the model in `git-native-planning.md` (markdown files as sole canonical state; git as the time machine).

**Effort to build:** ~3-6 months for one person to land safely.
**What you lose:** simplicity; you maintain a framework forever.
**What you gain:** complete in-repo planning, bisectable plans, no external dependency, programmatic validation of arbitrary invariants.
**Best fit:** the project this framework is being built *for*, if "having a framework" is itself a goal (e.g., dogfooding for downstream consumers).

### 8.4 What the original architecture targets

The original architecture is approximately Shape C with the additional commitments of an event-sourced kernel and hash-verified projection. As `fighting-git.md` showed, those additions are what fight git. **Shape C without the event log and hash chain is achievable; the additions cost more than they're worth for the audience that wants Shape C.**

---

## 9. The honest recommendation

If I were advising the user as a friend rather than executing on a brief:

**Build Shape A first. Use it for three months. See what's actually missing.**

Most likely outcomes after three months:

- **80% probability:** Shape A is enough. The framework as currently scoped is a pleasant intellectual exercise but not what was actually needed. Ship Shape A as the framework. Retire the rest.
- **15% probability:** Shape A is missing one or two specific capabilities (e.g., "I genuinely need bisectable structured plans for compliance"). Build *just those* on top of Shape A. Stay small.
- **5% probability:** the user has a use case (multi-team, regulated industry, multi-month rollouts coordinated via the repo) that genuinely needs Shape C's full ambition. Build it then, with three months of real-use evidence to ground the design decisions.

The risk of building Shape C first and finding out you needed Shape A is enormous (months of wasted engineering, a framework that nobody uses including you). The risk of building Shape A first and finding out you needed Shape C is small (you have to add things, but the ADRs and CONTEXT files you wrote in Shape A are still valid in Shape C).

**This is the YAGNI principle in `CLAUDE.md` applied to the framework itself.** The framework's own engineering principles say "no speculative interfaces, no plugin architectures for a single implementation, add the second case when it shows up." The full architecture currently violates these against itself — it builds for a scope that has not yet been demonstrated to be needed.

---

## 10. On the soft/hard tension specifically

The user's instinct that there's a tension between soft metaphors (clay, gardening) and hard requirements (determinism, auditability) is correct. The resolution is:

- **Soft metaphors describe the *process*: how humans and AI shape the work.** Iteration, pruning, revision, occasional total restart. These are properties of how planning *happens*.
- **Hard requirements describe the *artifact*: what the published project looks like.** Reproducible builds, audit-able decisions, citeable references. These are properties of what planning *produces*.

The framework should impose hard requirements on the *artifact* (what's on `main`, what shipped, what's been reviewed) and respect soft metaphors in the *process* (what's on a branch, what's being explored). When it tries to impose hard requirements on the process — append-only event log on every branch, immutable terminal states the moment a status changes — it fights the medium and loses the goodwill of the people doing the work.

A clay sculpture, once fired, is durable and citeable. While being shaped, it is not. The kiln is the boundary. In software, the kiln is `main` (or perhaps the PR review). The framework's mistake is treating every keystroke as already-fired.

---

## 11. So — do you need a framework?

**You need a discipline. You probably do not need a framework in the original architecture's ambition.**

The discipline:
- ADRs for decisions.
- A short curated context document per area.
- A short curated roadmap.
- A habit (in skills/rules) that teaches the AI to read these first and update them as part of completing work.
- An external PM tool if the team is bigger than one or two people.

The framework wraps that discipline with:
- Validation of conventions (a few hundred lines of code).
- Skills for AI assistants (a few hundred lines of markdown).

That is it. Everything else in the current architecture is solving problems you do not yet have evidence you need to solve. The right next step is to prove you need them by living without them — not to build them speculatively and discover later that the simpler version would have been enough.

---

## 12. What this means for the existing work

This document should not be read as "throw out everything." The current architecture documents are valuable as **deferred design** — when (and if) Shape C becomes necessary, the work in `architecture.md`, `fighting-git.md`, and `git-native-planning.md` is the technical foundation for getting there.

But it should be read as **caution against committing months of engineering before the requirement is proven.** Concretely:

- Do not implement `events.jsonl` or `graph.json` in their current form. The case for their existence is not yet made.
- Do not implement the full verb set. Implement the smallest set that demonstrably helps.
- Do not finalize the entity vocabulary (`epic`, `milestone`, `adr`, `decision`, `gap`, `contract`) as the model. Start with ADRs and a context file. Add entities only when they prove useful.
- Do not build the install script, the adapter generator, the boundary contract loader, or the audit pipeline yet. None of these have shown they are load-bearing.
- **Do** keep writing skills. Skills are the cheapest, highest-leverage component. They can be developed against Shape A and reused if you ever escalate to Shape B or C.
- **Do** keep `docs/architecture.md` as the long-term north star, but mark it explicitly as "aspirational design for if/when scope demands it."

---

## 13. Closing thought

The user wrote: "I am trying to use 'soft' metaphors together with determinism and hard requirements."

The deepest answer is: **you don't have to choose, but you do have to scope.** Soft for the process, hard for the artifact. Soft on branches, hard on main. Soft for the AI to draft, hard for the human to ratify. The framework that respects this scoping is much smaller than the one currently designed — and much more likely to actually get used.

The project doesn't need a framework that imposes its model on every corner of the work. It needs a discipline that gives the AI a place to look, a place to write, and a habit of doing both. Build that. See what's missing. Build the next thing only when you have stubbed your toe on its absence.

---

## Appendix — A reading list for "the boring answer"

If Shape A or B is the right answer, the relevant prior art is:

- **MADR** (Markdown Any Decision Records) — github.com/adr/madr — mature ADR template.
- **adr-tools** (Nat Pryce) — shell scripts for managing ADRs. ~500 lines total. Battle-tested.
- **The original ADR essay** (Michael Nygard, 2011) — *Documenting Architecture Decisions.*
- **Continuous Documentation / docs-as-code** literature — Chris Ward, Anne Gentle. Argues prose-in-repo is the right medium for decisions.
- **The Cynefin framework** (Snowden) — distinguishes complicated (analyzable, deterministic) from complex (emergent, requires probing). Most of planning is complex; the framework treats it as complicated. This is the conceptual mismatch.
- **Donella Meadows, *Thinking in Systems*** — for why "stocks" (state) and "flows" (events) are different and the framework's mistake is sometimes confusing them.
- **For PM-out-of-repo conventions:** any of Linear's, GitHub's, or Shortcut's docs on linking commits/PRs to issues. Mature and sufficient for most teams.

The honest summary: the wheel exists, in several mature shapes. The reason to build a custom framework is if you have a specific, demonstrated need that the existing wheels can't carry. That case has not yet been made for this project.

---

## In this series

- Previous: [01 — Git-native planning](https://proliminal.net/theses/git-native-planning/)
- Next: [03 — Discipline where the LLM can't skip it](https://proliminal.net/theses/discipline-where-the-llm-cant-skip-it/)
- Forward: [11 — Should the framework model the code?](https://proliminal.net/theses/should-the-framework-model-the-code/) — same audit voice applied to a later temptation (the code-graph question).
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
