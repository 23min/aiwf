# Discipline where the LLM can't skip it — chokepoints, referential stability, and the PR tier

> **Status:** defended-position
> **Hypothesis:** Skills are advisory; the framework's correctness must rest on enforcement chokepoints (CI, pre-push hooks) the LLM cannot skip; referential stability via stable IDs and tombstones plus a PR tier of mechanical checks is what delivers the value the user actually needs from in-repo planning.
> **Audience:** the user, after naming what's actually load-bearing in their experience.
> **Premise:** the user has real benefit from in-repo planning beyond ADRs; the "semantic stability" they care about is referential (names like `E-19` surviving rename/insert/delete/move); their chief operational concern is that skills are advisory and the LLM may silently skip reconciliation.
> **Tags:** #thesis #hitl #aiwf #software-development

---

## Abstract

[02](https://proliminal.net/theses/do-we-need-this/) recommended Shape A and asked the user to live with it; they pushed back. They have specific benefit from in-repo Epics/Milestones beyond ADRs; the "semantic stability" they want is referential, not prose-meaning; and their main concern is that skills are non-deterministic — the LLM may not invoke them, and things get missed. This document takes those corrections seriously and asks: what is the smallest set of mechanical guarantees that delivers (a) usable in-repo planning, (b) referential stability for ids like `E-19`, and (c) discipline that does not depend on the LLM remembering to do its job? The answer: stable ids separated from display names, tombstones for removed entities, structured commit trailers as the audit substrate, and a small set of `aiwf check *` validators running pre-push and on the PR. The "Workshop" tier — the open PR — is the missing middle between studio-as-clay and museum-as-sealed; it is where mechanical strictness lives. Skills become ergonomic accelerators, not the discipline layer. Build `aiwf check ids`, `aiwf check refs`, `aiwf check transitions`, plus `aiwf rename` and `aiwf remove` first; everything else can wait.

---

## 1. What the user actually said matters

Three corrections to the prior research docs, in the user's words:

1. *"I have actually had great benefit from having a roadmap with future epics, and a docs/ directory with architecture documents. So some of this content that would otherwise only exist in Jira or in an architect's head, when it is in the repo, gives great benefit."*

2. *"I have benefited from the Epics/Milestone system… With semantic stability, I mean, names like `E-19` need to be stable. I'm not so much referring to prose."*

3. *"My chief concern with skills is that they are not always used and the LLM doesn't behave in a deterministic way, so the planning and merging and reconciliation procedures are messy and things get missed, unbeknownst to the user. Yes, main is the museum, but it can't be a museum of broken things. Somewhere we need discipline. Somewhere between the studio and the museum."*

These three corrections, taken together, completely reshape the design problem. The previous research docs were chasing the wrong problems:

- `fighting-git.md` answered "how do we make events.jsonl survive merges" — but the user does not particularly care about events.jsonl; they care about referential stability.
- `git-native-planning.md` answered "how do we eliminate state stores so git handles everything" — but the user actively values having a structured planning model in the repo, not just commits and prose.
- `do-we-need-this.md` answered "do you need a framework at all" — but the user has confirmed: yes, more than ADRs, with discipline, but not the over-engineered current architecture.

This document attempts to answer the *actual* question implied by those three corrections: **what is the smallest set of mechanical guarantees that delivers (a) usable in-repo planning, (b) referential stability for ids like `E-19`, and (c) discipline that does not depend on the LLM remembering to do its job?**

---

## 2. The chokepoint argument

The user's third correction is the most important. Let me state it as a principle:

> **Enforcement that runs only when the LLM remembers to run it is not enforcement. It is hope dressed up as policy.**

Skills are documentation directed at an LLM. The LLM may:
- Read them and follow them. (The intended case.)
- Read them and forget mid-task because the conversation drifted. (Common.)
- Not read them because the trigger conditions in the skill description didn't match the user's phrasing. (Common.)
- Read them, decide they don't apply, and proceed without invoking them. (Common.)
- Be in fast/auto mode where skills aren't loaded reliably. (Happens.)

In every one of those cases, anything the skill was supposed to do **doesn't happen**, and the user often has no way to tell that it didn't. The result is exactly what the user described: "the planning and merging and reconciliation procedures are messy and things get missed, unbeknownst to the user."

The fix is not "write better skills" or "write more imperative skills" — that just lowers the failure rate marginally. The fix is **putting the actual enforcement somewhere the LLM cannot skip it.**

There are three loci where enforcement can live, ranked by reliability:

| Locus | Reliability | When it fires | Bypassable? |
|---|---|---|---|
| CI on the PR | Very high | At PR open / push | Only by an authorised admin merging without checks |
| Pre-commit / pre-push git hooks | High | At local `git commit` / `git push` | Yes, with `--no-verify` |
| `aiwf` verbs that the LLM is supposed to invoke | Low | Only if invoked | Trivially — the LLM just edits files directly |
| Skills that document what to do | Lowest | Only if loaded and followed | Skills are advisory text; not enforcement at all |

**The framework's correctness story benefits from resting on the top of this table, not the bottom.** Skills are useful — they speed the LLM up, give it the right verbs to reach for — but they are an unreliable layer to lean on for *guarantees*. CI on the PR is where the guarantees actually hold.

This reframes "trace-first writes," "the assistant never writes the projection directly," "the assistant never invents IDs," and similar architectural commitments. As written in `CLAUDE.md`, those commitments are *requests to the assistant*. They will be obeyed sometimes and not obeyed other times. **They become real guarantees only when CI checks them on every PR.**

---

## 3. What CI can actually check (and therefore what the framework should make checkable)

This is the design lever. The framework's structural choices should be made to maximize what a fast, deterministic CI check can verify. Below is a list of properties CI can actually enforce, ordered roughly by leverage:

1. **Referential integrity.** Every cited id (`E-19`, `M-PACK-A-02`, `ADR-0042`) resolves to a real entity in the tree. Catches: rename without redirect, deletion without cleanup, typos.
2. **No id reuse.** No two entities ever share an id. Catches: branch-collision survivors that didn't get renamed.
3. **No id resurrection.** An id that has appeared on `main` and been removed cannot be reused for a new entity. (Use a tombstone file or a registry; see §4.)
4. **Frontmatter conformance.** Every entity's frontmatter validates against its kind's schema.
5. **Status transitions are FSM-legal.** From `git diff main...HEAD`, the implied transition (`status: draft` → `status: in_progress`) is allowed by the FSM for that kind.
6. **No editing of terminal-state entities' structural fields.** Catches: accidental un-completion of completed work.
7. **Dependency graph is acyclic.** No `depends_on` cycle.
8. **ROADMAP / CHANGELOG / similar rendered docs are in sync** with the structural state. Either generated and checked-in (with CI verifying regeneration is a no-op) or generated on demand and not checked-in.
9. **Branch carries the milestone it claims to carry.** If branch is named `milestone/M-005-...`, the PR contains commits that touch `M-005`.
10. **PR template fields are filled.** Closes-link, acceptance-criteria checklist, principles-checklist conformance.

Items 1–7 are the **structural correctness layer**. Items 8–10 are the **discipline layer**. The framework's design point is to run both on every PR and block merge on failure (with override-with-reason for emergencies).

What CI **cannot** check:
- Whether the prose body of a milestone accurately describes the work.
- Whether the chosen scope is sensible.
- Whether a decision is wise.
- Whether the AI followed its skills.

The first three are inherent (they are human judgment). The fourth is interesting: CI can check the *outcome* of skills (was the right thing done?) but not the *process* (did the LLM invoke the skill?). And outcome-checking is sufficient — if the right thing was done, it doesn't matter whether a skill was invoked or the human did it manually or the LLM happened to do it without the skill.

**Design implication: don't try to verify skill invocation. Verify outcomes. Make outcomes mechanically checkable.**

---

## 4. Referential stability done right

The user's redefinition: "names like `E-19` need to be stable." Let me unpack what this actually requires and how to deliver it without an event log.

### 4.1 What "stable" means in practice

The user's pain points (rename, insert, delete, move) translate to specific operations the framework must handle:

- **Rename**: the entity formerly known as "Pricing extraction" is now "Pricing service extraction." The id `E-19` should remain `E-19`.
- **Insert**: a new milestone is added between `M-005` and `M-006`. Existing ids should not shift.
- **Delete**: a milestone is removed. References to its id elsewhere (in CHANGELOG, in commit messages, in other milestones' `depends_on`) should be detected.
- **Move**: a milestone is reassigned from epic `E-19` to epic `E-23`. Its id should remain stable; references should still resolve.

Notice that each of these is a **mechanical** problem. None require AI judgment. All can be checked by CI.

### 4.2 The mechanism: ids are immutable; everything else is mutable

Two-level naming:

- **Stable id**: `E-19`, `M-005`, `ADR-0042`. Once allocated, never changes. Never reused. The id is the entity's primary key.
- **Display name** (slug, title): mutable. Lives in frontmatter. Renames change this without touching the id.

The path on disk should *contain* the id but *should not depend on the slug for identity*. Two patterns work:

- `work/epics/E-019/epic.md` — id is the directory; slug doesn't appear in the path. Pro: rename is just a frontmatter edit. Con: directory listings are less readable.
- `work/epics/E-019-pricing-extraction/epic.md` — id-prefixed slugged directory. Pro: readable. Con: rename is a `git mv`, which complicates id-based lookup. Mitigation: a tiny `aiwf rename` verb that does the `git mv` and updates references in one commit.

The id is what code, CI, and other entities reference. The slug is what humans read.

### 4.3 Deletion without breaking references

This is where the framework adds real value. When an entity is deleted:

- A **tombstone file** replaces the entity at the same id: `work/epics/E-019/TOMBSTONE.md` with frontmatter `status: removed`, `removed_at: <date>`, `removed_in: <commit-or-pr>`, optional `superseded_by: E-023`.
- The tombstone keeps the id alive: anything referencing `E-19` still resolves (to a tombstone, not a live entity). CI can distinguish "referenced and present" from "referenced and tombstoned."
- The tombstone keeps the id reserved: future allocations do not reuse it.

Tombstones are git-tracked. They merge cleanly (separate file per dead entity). They give the AI a way to explain "what happened to E-19" without an event log: just read the tombstone.

### 4.4 Move (reparenting)

`M-005` was under `E-019`; now it belongs under `E-023`. Two viable representations:

- **Path move**: `git mv work/epics/E-019/M-005-foo.md work/epics/E-023/M-005-foo.md`. Update `parent:` in frontmatter. Update `epic.md` references. CI verifies the consistency.
- **Path stable, parent in frontmatter**: keep the file where it is (or in a flat `work/milestones/`), let the `parent:` frontmatter field do the assignment. Renderers reconstruct the hierarchy on demand.

The second option is more git-friendly (no `git mv` to confuse blame and merge). Reconsider the directory hierarchy as a *rendering* concern rather than a *storage* concern.

### 4.5 The id allocator without coordination

Branch-aware allocation that avoids collisions and never reuses tombstoned ids:

- The allocator scans `work/` (live entities + tombstones) and picks the next id higher than any seen.
- On a branch, allocations may collide with main's parallel allocations (both pick `E-019`).
- At merge time, a collision-detection check (CI or merge-time) detects the dual-allocation. Resolution: rename one. The renamed id appears as a tombstone redirect (`work/epics/E-019-renamed/TOMBSTONE.md` with `renamed_to: E-021`).
- Suffix scheme (`E-019a`) is also viable, especially when the two collisions are genuinely sibling concepts.

The key invariant: **ids never silently conflict; the resolution is a normal small commit that CI can verify.**

### 4.6 The CI checks that make this real

For referential stability, CI runs (on every PR, blocking):

1. **`aiwf check ids`** — every entity has a unique id; no id reused; no id reused that appears in a tombstone on `main`.
2. **`aiwf check refs`** — every reference (`depends_on`, `parent`, `supersedes`, `cites`, plus `E-NNN` patterns in any markdown) resolves to a live entity or a tombstone with explicit acknowledgment.
3. **`aiwf check tombstones`** — every removed entity has a tombstone; no entity went missing without one.
4. **`aiwf check renames`** — if frontmatter `slug` changed, the id is unchanged; if path changed, it was via a recognized rename (git mv detected).

These four checks deliver the referential stability the user is asking for. They are mechanical, fast, and not silently bypassable by the LLM because they run on the PR, not in the LLM's session.

---

## 5. The PR tier — the missing middle between studio and museum

The user said: *"Somewhere we need discipline. Somewhere between the studio and the museum."* This is the **PR tier**, and the framework should treat it as a first-class concept.

### 5.1 Three tiers, not two

| Tier | Locus | Stance | Enforcement |
|---|---|---|---|
| Studio | Local branch, pre-PR | Soft, malleable, exploratory | Local hooks (advisory), `aiwf` verbs (helpful) |
| Workshop | PR (open, awaiting review) | Stricter, structured, reviewed | CI checks, blocking on failures |
| Museum | `main` after merge | Hard, append-only-in-meaning, citeable | Merge protections, branch protections |

A pattern these earlier docs and the original architecture share is conflating Studio with Workshop. The user's clay-and-garden metaphor describes Studio. The user's "museum can't be broken" describes Museum. The *enforcement*, on the analysis here, lives in Workshop.

### 5.2 What happens at each tier

**Studio** (working branch, before PR):

- Mutate freely. Squash, rebase, rewrite history.
- `aiwf` verbs are convenient but not required.
- Local pre-commit hook may run a fast subset of checks as a courtesy. It does not block — it warns.
- The LLM can break things temporarily; that's part of iteration.

**Workshop** (PR open):

- All `aiwf check *` checks run on every push to the PR branch.
- Failures block merge.
- The PR description includes a re-assertion of acceptance criteria and a CHANGELOG entry (already required by `pr-conventions.yml`).
- Reviewers (human + AI) inspect and approve.
- This is where "things don't get missed unbeknownst to the user" — the user sees the CI status before they merge.

**Museum** (after merge to main):

- Branch protections enforce that PRs went through the Workshop tier.
- Tombstones, not deletions, when entities are removed.
- Renames carry redirects.
- The CHANGELOG and ROADMAP are kept in sync with structural state by CI.

### 5.3 Why this scoping resolves the soft/hard tension

Soft metaphors apply to Studio. Hard requirements apply to Museum. Workshop is where the conversion happens — where clay is inspected, possibly fired in part, possibly sent back to the Studio for more shaping.

This means the framework can be *kind* in Studio (no append-only burden, no immutable-on-completion friction, no skill-invocation requirement) while being *strict* in Workshop and Museum. The discipline lives at the gate, not at every keystroke.

---

## 6. The minimal structured planning model

Given the user's confirmation that they want more than ADRs in the repo, but the prior research argues against the architecture's full ambition, what is the right middle?

### 6.1 Keep

- **Epics and milestones as first-class entities** (the user has confirmed value).
- **ADRs** for decisions.
- **A roadmap** as a curated reading of the current and near-future epics/milestones (rendered or hand-maintained — both work).
- **Architecture documents** in `docs/` (already serves the user well; keep as-is, just ensure references to entity ids are CI-checkable).
- **Stable ids with tombstones** for referential stability.
- **The principle that the engine is invocable without an AI** — this is what makes CI-as-enforcement possible.

### 6.2 Drop (or strongly defer)

- The append-only `events.jsonl`. Replace with: per-entity history derived from `git log` + structured commit trailers, plus tombstone files for explicit removals.
- The `graph.json` projection with hash chain. Replace with: on-demand rendering by `aiwf render`, validated by `aiwf verify` against the working tree.
- The closed entity vocabulary as a hard schema. Start with `epic`, `milestone`, `adr`, `decision`, `gap` and let kind addition be a normal evolution; do not lock the schema down before use confirms it.
- Trace-first writes as a permanent ledger. Replace with: in-flight journal file (gitignored) for crash recovery, deleted on commit.
- The propagation-preview as a complex engine feature. Start with `aiwf check refs` reporting affected references on rename/move; add interactive preview later if it's clearly missed.

### 6.3 Add (the parts that actually matter)

- **A small set of CI checks** (`aiwf check ids`, `aiwf check refs`, `aiwf check tombstones`, `aiwf check renames`, `aiwf check transitions`) that run blocking on every PR.
- **Tombstone files** as the explicit, git-friendly way to remove entities without losing referential integrity.
- **Stable id ↔ display name separation** so renames are cheap and references survive.
- **Structured commit trailers** (`aiwf-verb:`, `aiwf-entity:`) so `git log` can be queried as a per-entity history without a separate event log.
- **A `post-merge` git hook** that runs `aiwf check *` and surfaces findings — so the user never silently ends up with broken state.

### 6.4 Skill content (advisory, ergonomic, not enforcement)

Skills are still valuable — they speed the LLM up and give it the right vocabulary. They just stop being load-bearing for correctness:

- A skill for "how to add a new milestone" that walks the LLM through the right verbs.
- A skill for "how to handle a discovered gap mid-implementation."
- A skill for "how to merge a planning branch into main."

When the LLM follows these, the work is fast and clean. When the LLM doesn't follow them, **CI catches the gap before merge.** The user is no longer dependent on the LLM's reliability.

---

## 7. Skill non-determinism, addressed directly

Skills will continue to be sometimes-followed and sometimes-not. The framework's response should be at three layers:

### 7.1 Make skill triggers as broad and reliable as possible

The skill description (the `description:` frontmatter that the harness uses to decide whether to load a skill) is the single point of failure. Most skill misses are because the description didn't match the user's phrasing. Mitigations:

- Write descriptions broadly (cover synonyms, common phrasings).
- Include explicit "TRIGGER when:" and "SKIP when:" sections in the description (this pattern is already used by the `claude-api` skill in this very repo's installed skills).
- Use auto-load patterns where the harness supports them (some platforms allow always-loaded "context" skills for specific repos).

This reduces the miss rate but does not eliminate it.

### 7.2 Build verbs that are obvious to reach for

If the LLM's natural impulse to "edit the milestone file" can be redirected to "use `aiwf promote`," the verb gets used. This requires:

- Verbs that are *easier* than the manual alternative, not harder.
- Verbs that produce better commits (structured trailers, sensible messages) than the LLM would write by hand.
- Documentation in `CLAUDE.md` or top-level skill that introduces the verbs early in any planning session.

This further reduces the miss rate.

### 7.3 Catch what slips through with CI

Whatever the prior two layers miss, CI catches. The user pushes to a PR. CI runs. Findings appear. The user (or the LLM) addresses them before merge.

This is the layer that converts "the LLM might miss it" into "missed work cannot quietly reach main." Without it, skills and verbs are useful suggestions but not guarantees.

---

## 8. What this implies for the existing architecture

Practical consequences of taking this synthesis seriously:

1. **`docs/archive/architecture.md` should be split.** A core "what the framework actually guarantees" section (small, mechanical, CI-enforced) and a "current direction of exploration" section (the rest). Today the document presents the full ambition as if all of it is committed.

2. **`CLAUDE.md`'s "Architectural commitments" benefit from being re-stated as enforcement layers.** Each commitment can answer: where is this enforced? Skill (advisory)? Verb (helpful but bypassable)? CI (mechanical, blocking)? Today the commitments read as obligations on the assistant; reframing them as guarantees the framework provides via specific mechanisms makes the enforcement story legible.

3. **The build plan should be reordered.** CI checks (`aiwf check *`) and the id/tombstone/rename mechanics should land *before* any of the event log or projection work. They are the load-bearing parts. The rest is optional optimization.

4. **The first new code to write is `aiwf check`**, with `ids`, `refs`, `tombstones`, `renames`, `transitions` subcommands. Each is a small, testable, deterministic function from the working tree to a list of findings. Each is independently useful. Together they constitute the enforcement layer.

5. **The events.jsonl and graph.json work in `docs/archive/build-plan.md` Stage 2 is a candidate for pausing.** Not deletion — the design might still fit a later, larger-scoped version — but explicit deferral until the CI-checks layer has been used long enough to show what the event log would actually be needed for.

---

## 9. Answering the user's specific worries one more time

**"My chief concern with skills is that they are not always used."**
→ Stop relying on them for correctness. Move correctness to CI. Skills become ergonomic accelerators, not the discipline layer.

**"Things get missed, unbeknownst to the user."**
→ CI runs on every PR. The user sees the status. Findings are visible in the PR's checks tab. Nothing slips silently into Museum.

**"Names like `E-19` need to be stable."**
→ Stable ids separated from display names; tombstones for removed entities; ids never reused; CI checks for collisions and broken references. Mechanical guarantee.

**"Main is the museum, but it can't be a museum of broken things."**
→ Branch protection requires PRs through the Workshop tier; CI in the Workshop tier blocks merge on findings; nothing reaches Museum without passing.

**"Somewhere between studio and museum we need discipline."**
→ The Workshop tier (the open PR). That is where discipline lives. Studio stays soft so iteration is fast; Workshop is strict so what crosses into Museum is sound.

**"I have benefited from the Epics/Milestone system."**
→ Keep it. Just rebuild it on referentially-stable ids + tombstones + CI checks, instead of events.jsonl + hash chain.

**"I have benefited from having a roadmap and architecture documents in the repo."**
→ Keep them. The framework should make their cross-references mechanically verifiable (CI check that `docs/archive/architecture.md` mentions of `E-19` resolve), not replace them.

---

## 10. Recommended next step

Build, in order, a single PR each:

1. **`aiwf check ids`** — scan `work/`, find every entity, report duplicates and reused tombstoned ids. Write the tombstone format spec at the same time.
2. **`aiwf check refs`** — scan `work/`, `docs/`, `ROADMAP.md`, `CHANGELOG.md` for entity-id references; report unresolved.
3. **`aiwf check transitions`** — given `git diff main...HEAD`, infer status changes and validate against the FSM (closed-set, declared in YAML).
4. **CI workflow** that runs all three on every PR, blocking on findings.
5. **`aiwf rename`** — the verb that makes referential stability ergonomic (frontmatter slug change, optional path move via `git mv`, preserves id, updates references).
6. **`aiwf remove`** — the verb that creates a tombstone instead of deleting.

This is roughly 4–6 weeks of focused work. After it lands, the user has:

- Mechanical guarantees of referential stability.
- A discipline tier that the LLM cannot bypass.
- A path forward for adding richer features only when their absence stings.

And critically: the user can stop worrying about whether the LLM "remembered" to do something. CI runs whether anyone remembered or not.

---

## 11. The deepest reframe

The user's frustration with skills is, on this reading, one of the most important signals in the research series. It says: *this framework's correctness should not depend on the LLM's behavior.* That principle, taken seriously, shapes most of the rest of the design:

- Storage choices are made for what CI can check, not for what reads well in prose.
- Verb design is about producing outcomes CI can verify, not about ritual the LLM should follow.
- Skills are convenience, not control.
- The Workshop tier (PR + CI) is the load-bearing discipline layer.
- The Museum tier (main) is what the discipline layer protects.
- The Studio tier (branch) gets to be soft because the Workshop catches what slips.

Once that principle is internalized, much of the prior architecture's complexity reads as scaffolding for a guarantee model larger than the situation requires. A small, well-scoped CI check layer plus referentially-stable ids plus tombstones plausibly covers the user's actual ask at this scale, and the LLM's reliability stops being a load-bearing assumption.

---

## Appendix — How this fits with the prior three research docs

- **`fighting-git.md`** identified that events.jsonl + hash chain fight git. This doc obviates that fight by not maintaining events.jsonl in the first place.
- **`git-native-planning.md`** proposed dropping the event log and graph entirely and letting markdown + git be the truth. This doc adopts that move and adds the missing piece: the CI-based enforcement layer.
- **`do-we-need-this.md`** asked whether a framework was needed at all. This doc answers: yes, but a much smaller one — basically a CI check suite + tombstones + a few verbs, plus skills that are openly advisory.

The three docs together constitute a journey from "the architecture as designed" to "what the user actually needs." This doc is the synthesis: the user's confirmed values (in-repo planning, referential stability, discipline) plus the prior docs' constraints (no fighting git, enforcement at chokepoints) yields a concrete, buildable shape that is dramatically smaller than the original architecture and dramatically more reliable than skills-alone.

---

## In this series

- Previous: [02 — Do we even need this?](https://proliminal.net/theses/do-we-need-this/)
- Next: [04 — Governance, provenance, and the pre-PR tier](https://proliminal.net/theses/governance-provenance-and-the-pre-pr-tier/)
- Related: [08 — The PR bottleneck](https://proliminal.net/theses/the-pr-bottleneck/) — extends the chokepoint argument
- Reference: [KERNEL.md](https://github.com/23min/ai-workflow-v2/blob/main/docs/research/KERNEL.md)
