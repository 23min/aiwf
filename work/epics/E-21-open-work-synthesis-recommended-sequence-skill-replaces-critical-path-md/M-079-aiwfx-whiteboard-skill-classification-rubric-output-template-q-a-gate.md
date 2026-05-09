---
id: M-079
title: 'aiwfx-whiteboard skill: classification rubric, output template, Q&A gate'
status: in_progress
parent: E-21
depends_on:
    - M-078
tdd: advisory
acs:
    - id: AC-1
      title: Skill scaffolded at aiwfx-whiteboard with frontmatter and SKILL.md
      status: met
      tdd_phase: done
    - id: AC-2
      title: Frontmatter description carries the natural-language query phrasings
      status: met
      tdd_phase: done
    - id: AC-3
      title: Body documents the tier-classification rubric for open-work landscape
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Body documents output template: landscape, sequence, first decision, pending'
      status: met
      tdd_phase: done
    - id: AC-5
      title: Body documents the Q&A gate flow per CLAUDE.md one-at-a-time convention
      status: met
      tdd_phase: done
    - id: AC-6
      title: 'Body documents anti-patterns: no operator override, no verb invention'
      status: met
      tdd_phase: red
    - id: AC-7
      title: M-074 skill-coverage policy or plugin equivalent accepts the skill
      status: met
    - id: AC-8
      title: Skill materialised by aiwf init / aiwf update via the rituals plugin
      status: met
---

# M-079 — aiwfx-whiteboard skill: classification rubric, output template, Q&A gate

## Goal

Ship the `aiwfx-whiteboard` skill — a planning-conversation ritual under `aiwf-extensions/skills/` that loads tree state via existing read verbs (`aiwf status`, `aiwf list`, `aiwf show`, `aiwf history`) and produces a tiered open-work landscape, a recommended sequence, a first-decision fork, and a Q&A-gated walk-through of pending decisions. After this milestone, an operator can ask *"what should I work on next?"* in any Claude Code session opened against this repo and receive structurally consistent direction synthesis without scrolling back through prior planning conversations.

## Context

`aiwf status` and `aiwf render roadmap` render *state* (what entities exist, what their statuses are). Neither renders *direction* (what to do next, in what order, with which dependencies foregrounded). E-21's operator-name-the-gap quote crystallises the missing affordance: *"It's very difficult to keep sequence in my head when there are so many ADRs, epics, milestones, gaps."* The synthesis pattern that produced `work/epics/critical-path.md` on 2026-05-08 is the template; this milestone graduates it into a reproducible skill body.

M-078 (preceding milestone) records the rationale for *where* this skill lives (rituals plugin) and *why* it stays pure-skill (no kernel verb until usage justifies). M-079 is the implementation — body content, classification rubric, output template, Q&A flow. M-080 (following milestone) validates the result against `critical-path.md` as a fixture and retires the holding doc.

The skill body is content, not code, so this milestone is `tdd: advisory`. The substantive testable surface is the structural-agreement check in M-080 — whether running the skill on the live tree produces output close to the existing critical-path.md. Section-presence drift-prevention tests are encouraged but not required at AC level.

## Acceptance criteria

### AC-1 — Skill scaffolded at aiwfx-whiteboard with frontmatter and SKILL.md

Skill lives at `aiwf-extensions/skills/aiwfx-whiteboard/SKILL.md` (path verified against the existing `aiwfx-plan-epic`/`aiwfx-plan-milestones` layout). Frontmatter declares `name: aiwfx-whiteboard` matching the directory; description is non-empty; the file is a single SKILL.md (no template subdirs are required for v1, since the synthesis output is templated in the body, not in side files).

### AC-2 — Frontmatter description carries the natural-language query phrasings

Description text contains, at minimum, these query phrasings (each in quotes or backticks so AI description-match retrievers can lift them as-is):
- *"what should I work on next?"*
- *"give me the landscape"*
- *"where should we focus?"*
- *"what's the critical path?"*
- *"synthesise the open work"*
- *"draw the whiteboard"* (or equivalent metaphor-anchored phrasing)

Total of at least five phrasings. The description is dense by design — Claude Code routes by description-match, and the skill's name (`whiteboard`) is metaphor-shaped not query-shaped, so the description does the routing work.

### AC-3 — Body documents the tier-classification rubric for open-work landscape

Body contains a *Tier classification rubric* section that names each tier explicitly, gives the leverage-on-future-work criterion that places an item in that tier, and gives examples drawn from `critical-path.md`'s actual tier placements. At minimum, five tiers (Tier 1 = compounding fixes, Tier 2 = architecturally foundational, Tier 3 = workflow rituals, Tier 4 = operational debris, Tier 5 = defer). Rubric is reproducible (criterion-based) but acknowledges LLM judgement at the margins (the placement of a borderline item is allowed to vary; the criteria themselves do not).

### AC-4 — Body documents output template: landscape, sequence, first decision, pending

Body contains an *Output template* section specifying the sections the skill emits: (a) tiered landscape table — one row per open item with kind, cost-estimate, what-it-unblocks columns; (b) recommended sequence prose with numbered ordering and explicit "before/after/parallel" framing; (c) first-decision fork — concrete options A/B/C with pros/cons/lean; (d) pending-decisions list — open Q&A items the operator may walk through. Template structure is fixed across runs; content is judgement-driven.

### AC-5 — Body documents the Q&A gate flow per CLAUDE.md one-at-a-time convention

Body contains a *Q&A gate* section. The gate fires after the recommendation is rendered. Gate text: *"Walk through the pending decisions one at a time, or is the recommendation enough?"*. When the operator opts in, the skill walks one decision at a time per CLAUDE.md *Working with the user* §Q&A format — context, options with pros/cons, lean, numbered options, wait for choice. When the operator declines, the skill exits cleanly with a one-line summary.

### AC-6 — Body documents anti-patterns: no operator override, no verb invention

Body contains an *Anti-patterns* section. Lists at minimum: (1) the skill does not replace the operator's judgement — it surfaces and gates; (2) the skill does not invent verbs that don't exist on the kernel surface (per E-21 epic constraint *"no verb invention"*); (3) the skill does not persist its output to a file (on-demand re-derivation is the contract; persisted artefacts go stale within hours); (4) scope is locked to direction-synthesis — adjacent functions ("should I refactor X?", "is this design good?") prompt "should this be its own skill?" not silent extension.

### AC-7 — M-074 skill-coverage policy or plugin equivalent accepts the skill

Run M-074's `internal/policies/skill_coverage.go` policy (or whichever scope-extension covers plugin-side skills) against the new skill and verify zero violations. If M-074's policy is kernel-only and does not yet cover plugin skills, this AC is satisfied by adding a one-line note in the milestone work log explaining the gap and (optionally) filing a follow-up gap rather than expanding M-074's scope from this milestone.

### AC-8 — Skill materialised by aiwf init / aiwf update via the rituals plugin

After installing/updating the rituals plugin in the consumer repo, `aiwf doctor` reports the skill as present (or whatever the equivalent verification surface is for plugin-installed skills). The plugin's marketplace metadata or registration list (whatever points consumers at the new skill) is updated as needed. This AC verifies the distribution path, not just the source file.

## Constraints

- **Pure-skill, no kernel verb.** Per M-078's ADR. No new kernel code; no new verbs. The skill calls only existing read verbs: `aiwf status`, `aiwf list`, `aiwf show`, `aiwf history`. If a verb the skill body would benefit from doesn't exist, file a follow-up gap rather than encoding a hand-edit workaround.
- **Read-only over the planning tree.** No mutations. The Q&A walk-through can suggest mutations to the operator (*"want me to file a gap for that?"*) but does not perform them as a side effect of the skill.
- **One-at-a-time Q&A is non-negotiable.** Per CLAUDE.md *Working with the user* §Q&A format. Batched-question rendering breaks the user's documented preference and the epic's stated mitigation against authoritative-but-brittle output.
- **No persisted artefact.** The skill's output goes to the conversation, not to disk. `critical-path.md` is being deleted in M-080 precisely to remove the persisted-artefact anti-pattern; this milestone must not reintroduce it.
- **Output template is consistent across runs.** Same tree state → structurally identical output. Tier *contents* and *lean* may vary with LLM judgement; tier set, section order, and column headers do not.
- **Description-match routing assumed.** Frontmatter description densely covers query phrasings (AC-2). Skill must work as the destination of natural-language queries even when the user does not type the skill name.

## Design notes

- Skill scaffold layout (refine at start-milestone): single `SKILL.md` under `aiwf-extensions/skills/aiwfx-whiteboard/`. No `templates/` subdirectory in v1 — output template is documented inline in the body, in the *Output template* section. If iteration shows the template benefits from being a separate file (e.g., for reuse across the deferred verb-backed v2), the split lands in a follow-up milestone.
- Frontmatter shape (refine at authorship):
  ```yaml
  name: aiwfx-whiteboard
  description: |
    Use to answer direction-synthesis questions like "what should I work on next?",
    "give me the landscape", "where should we focus?", "synthesise the open work",
    "draw the whiteboard", "what's the critical path?". Loads tree state via
    aiwf status / aiwf list / aiwf show / aiwf history; produces a tiered open-work
    landscape, a recommended sequence, a first-decision fork, and an optional Q&A
    gate over pending decisions. Read-only; no commit.
  ---
  ```
- Body section order (refine at authorship): *What it does* → *When to use* → *Inputs* (which verbs to call, in what order) → *Tier classification rubric* (AC-3) → *Output template* (AC-4) → *Q&A gate* (AC-5) → *Anti-patterns* (AC-6) → *Examples* (one walkthrough drawn from critical-path.md's content) → *References*.
- Inputs section names the read verbs: `aiwf status` for the in-flight summary, `aiwf list <kind> --status <s>` for per-kind enumeration (assumes E-20/M-072 has shipped), `aiwf show <id>` for detail when a referenced item warrants surfacing, `aiwf history <id>` for context on items whose recent activity matters.
- Examples section is one full walk-through using the current planning tree's actual content. The walk-through serves three purposes: (a) demonstrates the output template, (b) seeds the M-080 fixture validation, (c) gives the LLM a worked example to anchor on across runs.
- Q&A gate text (refine at authorship): *"Walk through the pending decisions one at a time, or is the recommendation enough?"* with explicit options 1/2/3 (Q&A walk / one-line summary / further follow-up the operator names).

## Surfaces touched

- `aiwf-extensions/skills/aiwfx-whiteboard/SKILL.md` (new — primary deliverable)
- Possibly `aiwf-extensions/marketplace.json` or equivalent registration surface (verify path at start-milestone)
- No kernel changes (`internal/`, `cmd/aiwf/` untouched)
- No CLAUDE.md edit (M-078's ADR is filed without re-editing CLAUDE.md; this milestone follows the same discipline)

## Out of scope

- Fixture validation against `critical-path.md` — happens in M-080.
- Deletion of `work/epics/critical-path.md` — happens in M-080.
- A `landscape` kernel verb — deferred per M-078's ADR.
- Constraint-aware re-prioritisation (*"we only have 2 days; what changes?"*) — future iteration of the skill.
- Incorporating external signals (calendar, deadlines) — explicitly out of scope per E-21.
- Migrating other planning rituals into the whiteboard metaphor — those are their own skills, unaffected by this milestone.
- Promoting M-078's ADR to `accepted` — separate decision after epic wrap.

## Dependencies

- **M-078** — design ADR must exist (status `proposed` or later) so this milestone's body can cite the placement and tiering rationale by ADR id.
- **E-20 / M-073** — `aiwf-list` skill should exist by now so this skill's *Inputs* section can reference `aiwf list <kind>` without that being a dangling reference. If M-073 hasn't shipped, the *Inputs* section uses the read verbs available at the time and notes the upgrade.
- **E-20 / M-074** — skill-coverage policy must exist so AC-7 has something to run against. If the policy is kernel-scope only, AC-7 is satisfied with a noted gap rather than scope-creeping into this milestone.
- **`aiwf-extensions` rituals plugin** — must be installed in the consumer repo for AC-8 to verify materialisation. The CLAUDE.md *Operator setup* section already requires it.

## Coverage notes

- (filled at wrap)

## References

- E-21 epic spec — scope, constraints, success criteria.
- M-078 — sibling milestone; the design ADR this milestone's skill body cites.
- M-080 — successor milestone; consumes this skill's output as a fixture.
- `work/epics/critical-path.md` — content the skill body's *Examples* section draws from; deleted in M-080.
- `aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`, `aiwfx-plan-milestones/SKILL.md`, `aiwfx-start-milestone/SKILL.md` — sibling planning rituals; conventions for skill body shape and frontmatter style.
- `internal/skills/embedded/aiwf-status/SKILL.md` — kernel-side sibling. Same job-shape (a one-screen synthesis); different layer (state, not direction).
- CLAUDE.md *Working with the user* §Q&A format — the convention AC-5 honours.
- CLAUDE.md *Engineering principles* §"Kernel functionality must be AI-discoverable" — primary authority for AC-2's dense-description requirement.

---

## Work log

(filled during implementation)

## Decisions made during implementation

- (none — all decisions are pre-locked above)

## Validation

(pasted at wrap)

## Deferrals

- (filled if any surface)

## Reviewer notes

- (filled at wrap)
