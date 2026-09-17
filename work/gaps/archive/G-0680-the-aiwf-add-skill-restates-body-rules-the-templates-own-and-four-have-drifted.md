---
id: G-0680
title: The aiwf-add skill restates body rules the templates own, and four have drifted
status: addressed
addressed_by_commit:
    - 66f09e010
---
## What's missing

`internal/skills/embedded/aiwf-add/SKILL.md` §"What to write per kind" states what
each kind's body sections should contain. Five of the six kind templates under
`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/` state theirs
too. D-0086 named that subsection the owner, but its question was scoped to an
acceptance criterion; for the other kinds nothing rules, so neither copy binds.
Nothing derives one from the other and no check compares them.

Expected: where a template and the owning subsection both describe a section, they
agree. Measured 2026-09-14 on `main` at `df0dd2c5f`:

```
$ sed -n '225,233p' internal/skills/embedded/aiwf-add/SKILL.md | grep -oE \
  '\*\*[A-Za-z /]+\.\*\*|one paragraph[^.;]*|one or two sentences|who consumes it'
**Epics.**
one paragraph, no longer than four sentences
**Milestones.**
**Gaps.**
one paragraph naming the symptom and the affected surface
one paragraph naming the operational impact
**ADRs / decisions.**
one or two sentences
**Contracts.**
who consumes it

$ grep -H -oE '1–2 sentences|One or two paragraphs|One short paragraph|Keep this short[^.]*' \
  internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/*.md \
  | sed 's|.*/templates/||' | sort
adr.md:One or two paragraphs
decision.md:One short paragraph
epic-spec.md:1–2 sentences
gap.md:Keep this short — with the one exception that a measurement is worth its space
milestone-spec.md:1–2 sentences
```

Epic `## Goal` is four sentences against 1–2.
ADR `## Decision` is one or two sentences against one or two paragraphs. Gap
`## What's missing` is one paragraph against a section that also requires a pasted
command and its output. Contract `## Purpose` is "who consumes it" against a
template naming both ends, which calls a one-sided record "a data format, not a
contract, and it does not need this record". Milestone states no length where its
template caps at 1–2 sentences. Only decision `## Decision` is compatible — one or
two sentences fits inside one short paragraph.

## Why it matters

An author following one surface writes a shape the other rejects, and nothing
reports it: a sweep of `internal/policies/` and `internal/check/` finds every
check pinned to `entity.RequiredSections` for section *names* and none comparing
either surface's section *content*. Both materialize into every consumer repo, so
the reader most exposed is one with no prior about which is current, and neither
says it is a copy.

The disagreement also widens on its own. Each template edit made without the
owning subsection in view adds one more, and the copy that goes wrong is the one
nothing re-reads.

## Related

- G-0665 fixed this one kind up — what an acceptance criterion body holds — and
  D-0086 records the ownership rule it settled. The remaining templates were not
  brought into line with it.
