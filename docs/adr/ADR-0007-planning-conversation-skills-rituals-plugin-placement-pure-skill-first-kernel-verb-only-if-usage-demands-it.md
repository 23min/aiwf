---
id: ADR-0007
title: 'Planning-conversation skills: rituals-plugin placement; pure-skill first, kernel verb only if usage demands it'
status: proposed
---

## Context

E-21 plans a planning-conversation skill (`aiwfx-whiteboard`) that synthesises the open-work landscape into a tiered view, a recommended sequence, and a Q&A-gated walk through pending decisions. Three placement-and-shape decisions surfaced during milestone planning on 2026-05-08 and were locked there:

1. **Where does it live** — kernel-embedded (under `internal/skills/embedded/aiwf-*`, shipped by the engine binary) or in the rituals plugin (under `aiwf-extensions/skills/aiwfx-*`, distributed via the Claude Code marketplace)?
2. **What shape ships first** — pure skill on top of existing read verbs (`aiwf status`, `aiwf list`, `aiwf show`, `aiwf history`) or a skill+verb pair where a new kernel verb produces the structured data the skill narrates?
3. **What is it called** — name candidates surfaced in conversation included `recommend-sequence`, `landscape`, `paths`, `focus`, `next`, `survey`, `synthesise-open-work`, `critical-path`. Each was evaluated against fit, clarity, and PM-jargon avoidance.

Each decision is principle-shaped — the answer applies beyond `aiwfx-whiteboard` to every future planning-conversation skill. ADR-0006 (*Skills policy: per-verb default; topical multi-verb when concept-shaped; no skill when --help suffices*) covers the **complementary** axis — granularity *within a topic*. This ADR covers **placement and tiering across the kernel/plugin boundary**, plus the name worked example so the rationale is reproducible. Together the two ADRs define how aiwf skills are organised across both axes.

Two CLAUDE.md kernel principles bear directly on the reasoning:

- **"Kernel functionality must be AI-discoverable."** Every verb, flag, JSON envelope field, body-section name, finding code, trailer key, and YAML field is reachable through `aiwf <verb> --help`, embedded skills, CLAUDE.md, or design docs — kernel-discoverable channels. This pulls toward kernel-embedded skills **for verbs**; it does not pull planning conversations into the kernel, because a planning conversation is not a kernel capability.
- **"The framework's correctness must not depend on the LLM's behavior."** Skills are advisory; the kernel verbs and check rules are authoritative. A planning conversation that lives in the rituals plugin remains advisory by construction; promoting it to a kernel verb purely because "it would be discoverable there" inflates the kernel surface without any new authoritative guarantee.

## Decision

### Placement — planning-conversation skills live in the rituals plugin, not kernel-embedded

Planning-conversation skills go in the rituals plugin (`aiwf-extensions/skills/aiwfx-*`). Kernel-embedded skills (`internal/skills/embedded/aiwf-*`) are reserved for verb wrappers — skills whose body documents a single kernel verb (or a topical group of verbs per ADR-0006).

The discriminator is *whether the skill primarily surfaces a kernel capability*. A verb wrapper does; a planning conversation does not. `aiwfx-whiteboard` invokes existing read verbs as data sources, but its substantive content is the synthesis rubric and the conversation gate — neither is part of the kernel verb surface.

The existing pattern this codifies:

| Skill | Layer | Why |
|---|---|---|
| `aiwf-status`, `aiwf-list`, `aiwf-history`, `aiwf-contract` | Kernel-embedded | Verb wrappers; surface a kernel capability for AI discovery. |
| `aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`, `aiwfx-record-decision`, `aiwfx-release` | Rituals plugin | Planning conversations and lifecycle rituals; orchestrate multiple verbs and conversation steps. |
| `aiwfx-whiteboard` (this ADR) | Rituals plugin | Planning conversation — synthesises tree state into a tiered landscape and Q&A-gated walk-through. Not a verb wrapper. |

### Tiering — pure-skill first; kernel verb only when usage demands it

When a synthesis function is first imagined, it ships as a pure skill on top of existing read verbs. A backing kernel verb is filed only when usage shows the skill is **re-deriving the same structured data on every invocation** — i.e., the skill body grovels through prose to reconstruct shape that should be a typed query result.

Trigger conditions for promotion to a skill+verb pair:

- The skill body repeats a non-trivial parsing or reduction every run because the underlying read verbs don't surface the shape directly.
- Two or more skills duplicate the same reduction logic, or the same operator workflow runs the synthesis often enough that re-derivation becomes a measurable cost.
- A reasonable kernel-side query would replace the reduction with a single call returning the same shape.

Until these conditions are observed, the pure-skill form is preferred. This matches the principle "skills are advisory; the verb surface is authoritative" — extending the verb surface speculatively, in advance of a re-derivation pattern, costs both the kernel review burden and the reversibility tax (every kernel verb owes *"what verb undoes this?"* per CLAUDE.md *Designing a new verb*).

The deferred follow-on for `aiwfx-whiteboard` is a `landscape`-style kernel verb (working title: `aiwf landscape`) that would return the tiered open-work structure as JSON for the skill to narrate. **It is not filed by E-21.** The trigger for filing it is the conditions above — concretely, repeated runs of `aiwfx-whiteboard` on real planning sessions where the operator can name the structured data the skill keeps re-deriving.

This rule closes E-21 success criterion #7 (*"An ADR or D-NNN captures the design choice between pure-skill (this epic) and skill+verb (the deferred follow-on), with the rationale for starting pure-skill."*).

### Name — `aiwfx-whiteboard`, with rejected alternatives

The name is a worked example demonstrating the placement and tiering rules in action: rituals-plugin placement → `aiwfx-` prefix → name candidate evaluated against fit, clarity, and PM-jargon avoidance.

**Selected: `aiwfx-whiteboard`.** The metaphor's fit-rationale:

- **Ephemerality.** A whiteboard is wiped between sessions; the synthesis output similarly does not persist (no on-disk artefact — see M-079's no-persisted-artefact constraint, and M-080's deletion of `critical-path.md`).
- **Surfacing-not-deciding.** Standing at a whiteboard, the operator decides; the board surfaces shape but does not pick. This matches the Q&A-gated structure (*"Walk through the pending decisions one at a time, or is the recommendation enough?"*) — the skill renders a recommended sequence but the operator chooses whether to walk through.
- **Operator-at-the-board.** The skill's framing is collaborative — the operator and the AI sketch the landscape together, the AI offers a lean, the operator decides. Names like `next` or `recommend` would centre the AI as authoritative; `whiteboard` keeps the human at the board.

Rejected alternatives, one-line rationale per:

- **`recommend-sequence`** — PM-jargon-shaped; same critique that retired `critical-path` as a name. Implies a single right answer; the synthesis is conversational, not authoritative.
- **`landscape`** — too geographic, not action-oriented; reads as a noun-only output, not an invitation to converse. Reserved for the deferred kernel verb (whose job *is* the landscape data).
- **`paths`** — vague; many things in a planning system have "paths" (file paths, dependency paths, code paths). Not specific to direction synthesis.
- **`focus`** — too narrow; "focus" implies prioritisation already done. The skill's job is producing the input to a focus decision, not naming the focus itself.
- **`next`** — query-shaped, captures only the prompt (*"what's next?"*) and not the synthesis act. Loses the surfacing-and-Q&A frame.
- **`survey`** — academic; reads as data-gathering rather than synthesis. Misses the conversational and ephemeral tone.
- **`synthesise-open-work`** — too literal; functionally accurate but misses the metaphor and the ephemeral framing. Hyphenated multi-word names also clash with the plugin's `aiwfx-<short>` convention.
- **`critical-path`** — rejected at the source: the holding doc that motivated this skill was already named `critical-path.md`, and that name was understood as misleading PM jargon (the planning tree is not a CPM network and the synthesis surfaces *recommendations*, not deterministic critical paths).

The rejected set is preserved here so a future planner proposing a similar synthesis skill sees the candidate evaluation pre-walked.

## Consequences

- **All future planning-conversation skills land in the rituals plugin.** The kernel surface stays bounded by verbs and verb-wrapper skills. A new planning-conversation skill that wants kernel placement must justify itself against this ADR — "it would be more discoverable" is not by itself sufficient, because the kernel-discoverability principle pulls only on *kernel capabilities*.
- **The deferred `landscape` verb is on the open-work radar but not filed.** It is owned by *future* operator usage, not by E-21. If/when filed, it goes through CLAUDE.md *Designing a new verb*: *"what verb undoes this?"* must be answered (a read verb is undone by re-running it with different inputs — an easy answer, but not a free one).
- **Pure-skill-first applies beyond `aiwfx-whiteboard`.** Any future synthesis skill (e.g., a *"what's blocking what"* cross-kind dependency mapper) starts as pure-skill on top of existing read verbs. Promotion to skill+verb is justified by observed re-derivation, not by speculative ergonomics.
- **The name worked example is the discoverable artefact for future name choices.** When a future planner proposes a name like `aiwfx-survey` or `aiwfx-landscape`, the rejected-alternatives section above is the precedent — the rationale (*"survey is academic"*, *"landscape is reserved for the deferred verb"*) is captured here, not re-litigated.
- **Status remains `proposed` through E-21 wrap.** Promotion to `accepted` is a separate decision after the epic closes. If M-079's implementation surfaces a constraint that revises the placement or tiering reasoning, this ADR is edit-bodied before any status change.

## References

- **ADR-0006** — *Skills policy: per-verb default; topical multi-verb when concept-shaped; no skill when --help suffices* — sibling ADR. Covers granularity *within a topic*; this ADR is its peer covering placement and tiering *across kernel/plugin*. Complementary, not overlapping.
- **E-21** — *Open-work synthesis: aiwfx-whiteboard skill replaces critical-path.md* — the epic that motivated the three decisions recorded here.
- **M-079** — *aiwfx-whiteboard skill: classification rubric, output template, Q&A gate* — the implementation milestone that consumes this ADR's decisions and cites it by id.
- **CLAUDE.md** *Engineering principles* §*"Kernel functionality must be AI-discoverable"* — primary authority for the placement reasoning. Pulls toward kernel-embedded skills *for verbs*; does not pull planning conversations into the kernel.
- **CLAUDE.md** *Engineering principles* §*"The framework's correctness must not depend on the LLM's behavior"* — secondary authority for the pure-skill-first rule. Skills are advisory; speculative kernel verbs do not add authoritative guarantees and increase the kernel surface for no observable benefit.
- **CLAUDE.md** *Designing a new verb* — gates any kernel verb filing on *"what verb undoes this?"*; informs the deferred-`landscape` posture above.
- **`work/epics/critical-path.md`** — the holding doc whose name is part of the rejected-alternatives reasoning; deleted in M-080.
