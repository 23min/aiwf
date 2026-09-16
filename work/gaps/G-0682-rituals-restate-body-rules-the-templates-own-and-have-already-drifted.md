---
id: G-0682
title: Rituals restate body rules the templates own, and have already drifted
status: open
---
## What's missing

D-0093 assigns a kind's body-content rules to that kind's template. Four shipped
rituals state their own beside it, a bullet or a row per section of the entity
being scaffolded:

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md:40-48`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md:40-47`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-record-decision/SKILL.md:59-75`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-record-gap/SKILL.md:22-29`, `:42-54` and `:58-67`

A fix lands in those files; the rules they restate live beside the sections they
govern under `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/`.
Not every ritual statement about a section is one of these:
`milestone-spec.md:102-113` hands `## Closes`, `## Decisions made during
implementation`, `## Deferrals`, `## Release note`, `## Validation` and
`## Reviewer notes` to `aiwfx-start-milestone` and `aiwfx-wrap-milestone`, and says
so — those rituals own what they state.

Expected: where a ritual and its template both describe a section, the ritual
carries what the template carries. Measured 2026-09-14 on branch
`patch/G-0680-template-body-rules` at `66f09e010`:

```
$ T=internal/skills/embedded-rituals/plugins/aiwf-extensions/templates
$ S=internal/skills/embedded-rituals/plugins/aiwf-extensions/skills

$ grep -n 'per major piece\|why not yet\|named removal trigger' $T/epic-spec.md
28:- <Feature or capability — one bullet per major piece of work, often a milestone>
32:- <Explicitly excluded item — usually the most-tempting adjacent work, with a one-line "why not yet">
36:- <Technical invariant, non-negotiable rule, shim-policy exception with a named removal trigger>
$ sed -n '43,44p' $S/aiwfx-plan-epic/SKILL.md
   - **Scope** and **Out of scope** — sibling top-level sections, both populated.
   - **Constraints** — invariants, banned shortcuts, shim policies.

$ sed -n '34,37p' $T/milestone-spec.md
<!-- 2–3 sentences: what exists before this milestone, what must be in place, what
     changed to make it possible now. Prior milestones, blocking dependencies
     resolved, decisions landed. Not a re-telling of the epic, and not an argument
     for the work. -->
$ sed -n '42p' $S/aiwfx-plan-milestones/SKILL.md
   - **Context** — what exists before; what must be in place; what changed to make it possible now.

$ sed -n '46p' $T/adr.md; sed -n '35p' $T/decision.md
State the decision in plain terms. One or two paragraphs. Imperative voice ("we use X for Y" rather than "it was decided that…"). If there are sub-decisions, bullet them. If the decision is phased, say so.
What was decided. Imperative voice. One short paragraph.
$ sed -n '63p;73p' $S/aiwfx-record-decision/SKILL.md
- **Decision** — what's decided, in plain imperative voice.
- **Decision** — what's decided.
```

Every rule on the left is absent from the bullet on the right: the removal trigger a
shim-policy constraint must name, the bound on an epic's scope bullets, the "why not
yet" beside an exclusion, the cap on a milestone's `## Context` and its ban on
re-telling the epic, the length of an ADR's or a decision's `## Decision`, and the
instruction to bullet sub-decisions and to say when a decision is phased. That is a
sample of the bullets on three of the four surfaces, not a sweep of them. The drift
is omission rather than contradiction — nothing read states a rule its template
denies.

It widens through ordinary work. The first two omissions above date from this gap's
own parent commit, which put both rules into `epic-spec.md` and left
`aiwfx-plan-epic:43` untouched; `f952d05fb` had done the same to `gap.md` alone
earlier the same day. Nothing reported either.

`aiwfx-record-gap` is the one that says it is a copy — `:22-24` sends the filer to
`gap.md` as "where those rules are maintained" and `:26-29` states only what it adds
— and the one that carries more than its template rather than less, at `:63-67`,
which tells a filer what to do when there is nothing to run.

## Why it matters

Each of the four has the author fill the template itself — copied, or pasted over
the body `aiwf add` wrote — in the same step that states the ritual's own list. So
the fuller rule reaches the working file and is then deleted by the act of filling
the section it sits in, whether that is a placeholder (`epic-spec.md:28`), a comment
(`milestone-spec.md:34-37`) or the prose itself (`adr.md:46`). What the author is
left working to is the ritual's checklist, which is the terser copy, so a constraint
the template would have imposed is simply not there: an epic constraint with no
removal trigger, an ADR with sub-decisions not bulleted, a phased decision that
never says so. Nothing compares the two copies on content —
`internal/policies/milestone_section_ownership.go` compares which ritual *owns* a
milestone section, and records that the other half, "whether a rule's prose is
stated twice", is unpinnable under D-0070 and carried at review — so each template
edit made without the four rituals in view is another omission nobody sees.

## Related

- G-0680 removed the same shape from the `aiwf-add` skill, and D-0093 settled the
  ownership it now routes to. Neither reached the rituals.
