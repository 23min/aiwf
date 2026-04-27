# The kernel — what the framework actually needs to do

> **Status:** reference. Promoted from §1 of `01-git-native-planning.md` and refined through subsequent research. Short by design.
> **Purpose:** when any architecture proposal, skill, verb, or research direction is on the table, walk this list. If the proposal does not serve one of these, it is not in scope. If a need on this list is not yet served, that gap should be named explicitly.
> **Discipline:** this document changes by deliberate edit, with reasoning recorded in commit messages or in a numbered research doc. It does not drift.

---

## The needs

The framework exists to do these things, and only these things:

1. **Record planning state** — what epics exist, what milestones, what's in flight, what's done, what's blocked, what was decided. Persistent, accessible to humans and AI from inside the repo.

2. **Express relationships** — milestones belong to epics; milestones depend on milestones; decisions ratify scope; gaps motivate new work; ADRs constrain code. Relationships are first-class and machine-checkable.

3. **Support evolution** — insert a milestone between two others; rewrite a milestone when a dependency changes; spawn a new epic when a discovered gap is too large; supersede a decision; reorder priorities. Plans are clay, not stone, until they are committed.

4. **Keep history honest** — when a thing changed, who changed it, why. Provenance is queryable and auditable but does not have to take a particular form (an event log, git history, structured commit trailers, or external systems can each carry the load, alone or in combination).

5. **Validate consistency** — references resolve, status transitions are legal, terminal states are not silently undone, cycles do not form, ids do not collide or get reused. Validation is mechanical, fast, deterministic, and does not require AI judgment.

6. **Generate human-readable views** — the ROADMAP, dependency diagrams, status reports, audit reports. Views are derived on demand from the canonical state; they are courtesies, not authority.

7. **Coordinate AI behavior** — skills, rules, contracts shape how AI assistants act on the project. Kept versioned with the work so behavior is reproducible per checkout.

8. **Survive parallel work** — multiple humans, multiple AI assistants, multiple branches, possibly weeks of divergent work. Merging is well-defined for the structural state; semantic conflicts are surfaced as findings, not silently resolved.

---

## Cross-cutting properties

These are not separate needs but quality bars every solution to the eight needs must meet:

- **Enforcement does not depend on the LLM choosing to enforce.** Skills are advisory; CI gates and validators are authoritative. (See `03-discipline-where-the-llm-cant-skip-it.md`.)
- **Referential stability is real.** An id like `E-19`, once allocated, always means the same entity, even after rename, move, or removal. Tombstones, not silent deletions.
- **Honest about meaning.** The framework guarantees referential and structural stability. It does not pretend to guarantee semantic meaning, which is a property of prose and human understanding.
- **Engine is invocable without an AI.** Every verb takes flags, reads stable input formats, emits a JSON envelope, exits with documented codes. Humans, CI, and other tools drive it directly.
- **Opt-in over prescriptive.** Different project shapes (solo↔team, short↔long, greenfield↔brownfield, unregulated↔regulated) need different subsets of the framework. The kernel is small; everything else is opt-in via `.ai-repo/`.
- **Soft in raw studio; AI-assisted strictness pre-push; mechanical strictness at the PR gate; sealed at main.** Iteration is unconstrained on a personal branch in early shaping. As work approaches readiness, framework verbs and validators tighten the loop pre-push, while the AI is still in conversation and local tools are fast. CI on the PR is the mechanical gate that does not depend on the LLM. Main is the sealed artifact. (See `04-governance-provenance-and-the-pre-pr-tier.md` §6.)
- **Modular and opt-in.** A small kernel everyone can use, plus modules each project enables based on its shape on the team-size, horizon, brownfield-depth, and regulation axes. There is no single default configuration; the kernel is always on, everything else composes per project via `.ai-repo/config/`. (See `04` §4.)
- **Governance and provenance are first-class UX, not side effects.** The renderers and queryable surfaces for who-did-what-and-why and what-can-change-here are core, not optional. Storage choices serve those surfaces; surfaces do not get derived as afterthoughts of storage. (See `04` §2.)

---

## What is *not* on the list

The following are implementation choices, not needs. Past architecture work has sometimes treated them as load-bearing; they are not:

- An append-only `events.jsonl` event log file.
- A separate `graph.json` projection with a hash chain.
- A monotonic ID counter coordinated across branches.
- A closed entity vocabulary fixed before use.
- Trace-first writes as a permanent ledger (vs. as a crash-recovery journal).
- A globally totally-ordered sequence of mutations.

Each of these may turn out to be *a* way to serve one of the needs. None is *the* way. Proposals adopting them owe an argument that they serve a need on the list better than the alternatives.

---

## How to use this document

When proposing change:

1. Identify which need(s) the proposal serves. If none, reject or rescope.
2. Identify which cross-cutting property the proposal might strain. If any, address it explicitly.
3. If the proposal introduces an item from the "not on the list" section, justify why this need can't be served otherwise.

When reviewing a proposal:

- The proposer should have done the above. If they have not, ask them to.
- Specifically: ask "which need is this serving, and is there a smaller way to serve it?"

When the kernel itself feels wrong:

- Open a numbered research document examining the case for change. Do not edit the kernel without recorded reasoning.
- The kernel should be the slowest-changing artifact in the framework's design.
