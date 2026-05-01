# Design lessons — mutation, identity, vocabulary

Three principles distilled from a design discussion about how aiwf mutates state, how it names what it mutates, and how it expresses those names in the surfaces humans and agents read. These are not new architecture; they are tightenings of rules already implicit in the v3 PoC, written down so they stay true as the surface grows.

The principles are deliberately framed in their own terms — not in terms of any specific substrate or external system. They stand on their own merits.

---

## 1. Identity is not location

**Principle.** Every piece of work has two distinct properties: **who it is** (its identity) and **where it currently lives** (its location, or its snapshot coordinate). These look interchangeable in small systems but diverge under any kind of reorganization — renames, moves, restructures, history rewrites. Systems that conflate them produce dangling references whenever location changes.

**The rule.** Anchor cross-references on identity, never on a snapshot coordinate. Identities are stable names (`E-001`, `M-002`, `D-014`); coordinates are content-derived fingerprints (commit hash, projection hash, sequence number) that change whenever the content changes. Names survive change; fingerprints don't.

**Coordinates are still useful** — as **time anchors in a query**. The query shape is always `(identity, time-coordinate)`, never just `(coordinate)`. So:

| Question | Query shape |
|---|---|
| What does E-001 look like now? | (E-001, HEAD) |
| What did E-001 look like before yesterday's hotfix? | (E-001, timestamp) |
| What changed in E-001 between two points? | (E-001, hash-A) vs. (E-001, hash-B) |
| What was the state of everything when commit abc landed? | (*, hash-abc) |
| Show E-001's full history | (E-001, ∅) |

The simplest mental check: if you removed the identity from the query and only kept the coordinate, would the query still mean what you wanted? If yes, the coordinate is correctly playing time-anchor. If no, the coordinate has slipped into doing identity's job.

**Status in v3.** Already followed. All trailers reference entities by ID (`aiwf-entity`, `aiwf-prior-entity`, `aiwf-prior-parent`); commit hashes appear only in display output (`--pretty=tformat:%H`) for human consumption. No internal logic anchors on a hash. Lookups go through `git log --grep "^aiwf-entity: <id>$"`.

**Watch-points.**

- When the projection-hash layer (RFC 8785 + SHA-256, `aiwf verify`) lands, no trailer or cross-reference may take a projection hash as input. Hashes are inputs to *verify-style* commands (compare-against-recorded-hash) and outputs to humans (display); they are not inputs to entity-to-entity links.
- When entities grow links to each other (a decision references a milestone, an ADR references a contract version), the link target is always an entity ID. Never a commit hash, never a projection hash, never a file path.

---

## 2. Atomicity is a unit, not a sequence

**Principle.** Every logical mutation has exactly one atomicity boundary, drawn around the operation as a whole. Either the whole operation lands or none of it does. There is no observable half-state. Rollback-on-failure is defensive engineering against a problem that shouldn't exist: the operation should never have been observable in the half-state in the first place.

**The rule.** All multi-step mutations go through `Apply`. No verb invents its own staging, its own commit, or its own rollback path. If a verb needs to write multiple files, it goes through `Apply`. Any rollback logic that exists is internal to `Apply`; outside the boundary, the operation is binary.

**The design question for new verbs** is not "how do I sequence these writes?" — it is "what's the single atomic operation this verb performs?" If you can't name that, the verb isn't ready to be implemented.

**Status in v3.** Mostly followed. G2 (atomic rollback on Apply failure) hardened the boundary recently. The discipline is in place; the rule is to keep it that way.

**Watch-points.**

- Any new verb that writes more than one file: review whether it's going through `Apply` or open-coding its own sequence.
- Any rollback logic that lives outside `Apply` is a smell. Either fix the abstraction or pull the logic inside the boundary.
- Hooks (pre-commit, pre-push) are advisory, not load-bearing. The engine's invariants must be enforced inside the verb, not at the hook boundary. A hook is a fast-fail courtesy for the user; the verb must remain correct without it.

---

## 6. Don't fight the substrate's vocabulary

**Principle.** Every framework sits on top of substrates the user already understands — the filesystem, the version control system, the language, the shell. Those substrates come with mental models the user has internalized, often over years. A framework that introduces new vocabulary competing with the substrate's vocabulary forces every collaborator to maintain two mental models for the same artifacts, and the cost of that doubling is paid on every interaction, forever.

**The rule (asymmetric).**

- **Adding** new vocabulary for genuinely new concepts: fine, even encouraged. *Entity*, *event*, *projection*, *contract*, *gap*, *recipe* — these are aiwf-specific things the substrate doesn't have. Name them deliberately.
- **Replacing** existing substrate vocabulary with framework-private synonyms: refuse. A commit is a commit. A branch is a branch. A file is a file. A merge is a merge. If aiwf is tempted to introduce its own word for something the substrate already names, that's friction without gain.

**The failure mode this prevents.** A framework that owns its substrate's vocabulary in private. When the framework says "checkpoint" but means "commit," every conversation between agent and human, every doc, every error message, has to do a translation. The framework becomes the gatekeeper of meaning. Users feel it as "I have to think in framework terms now," and they're right.

**Status in v3.** Unverified. Probably mostly aligned, but no one has done the sweep.

**Required actions.**

1. **Sweep.** Walk aiwf's surface for places where it has its own name for something the substrate already names. The sweep needs to cover every surface where vocabulary leaks:
   - command-line help text and flag descriptions
   - error messages
   - skill prose and templates (skills are the loudest source — they are written for an AI audience and tend to over-explain)
   - generated commit messages and trailer keys
   - doc prose (`architecture.md`, `build-plan.md`, README, this directory)

2. **Sanitize on import.** When importing skills, templates, or recipes from elsewhere (which the build-plan anticipates), foreign vocabulary travels with them. The import path is where to catch private dialects before they spread.

**Watch-points.**

- New commands and new doc sections: ask "does this word name a concept the substrate already has?" If yes, use the substrate's word.
- Generated output: what humans see in `aiwf status`, error messages, and commit subjects is the loudest vocabulary surface. A single rename in CLI help has more reach than ten renames in internal package names.

---

## On reversal — absorbed into "immutability of done"

An earlier draft included a fourth principle, "Reversal is a verb": for every mutation, name what undoes it. On reflection this is a tightening of the existing **"immutability of done"** principle (in `architecture.md` and the root `CLAUDE.md`), not a separate architectural rule. The substance is captured as a verb-design checklist item:

> **When designing a new verb**, the design isn't done until you can answer "what verb undoes this?" If the answer is "another verb of the same kind that supersedes" (e.g., another `set-status`), good. If the answer is "an explicit terminal-state transition like cancellation," good. If the answer is "you can't, and that's deliberate — here's why" (e.g., `complete` is terminal; defects spawn a new entity via `hotfix`), good. If the answer is "we'll figure that out later," the verb isn't ready.

Most aiwf verbs reverse via the same verb with different inputs (state transitions are symmetric). Terminal states are special-cased via dedicated verbs (`hotfix`, `cancel`). What this principle prevents is a verb whose reversal story is "well, you'd have to manually edit the events file."

---

## Underlying coherence

These principles cohere around a single underlying idea: **systems that mutate state should be honest about what they're mutating, when, and how it can be undone — using the names everyone already knows.**

- Items 1 and 6 are about honesty in **naming**: identities are stable names, not coordinates; substrate concepts use substrate names, not private synonyms.
- Item 2 is about honesty in **operation**: a mutation either happened or didn't; there is no half-state to reason about.
- The absorbed reversal rule is about honesty in **closure**: every mutation has a named undo (often via another mutation), or an explicit "this is terminal."

Together they describe a system that is predictable to reason about, in either direction in time, regardless of who's reasoning about it — human, agent, or another tool.

---

## What this turns into, concretely

Three follow-on artifacts, in priority order:

1. **Vocabulary sweep** (principle 6). Walk the surfaces listed above; record findings and renames in a focused PR. This is the largest unknown — the other two principles are already followed; this one is unverified.
2. **Architecture-doc tightening** (all three principles). Add the three rules to `docs/architecture.md` (or its v3 equivalent) so they are part of the load-bearing engineering surface, not just a research note. Cite back to this document for the full discussion.
3. **Verb-design checklist update**. Add the "what reverses this verb?" question to whatever checklist gates new-verb design. One sentence in the right place.

None of this requires engine changes. The principles describe what the engine already does, plus a sweep to confirm one surface is consistent. The work is in writing down rules so future contributors don't have to rediscover them by accident.
