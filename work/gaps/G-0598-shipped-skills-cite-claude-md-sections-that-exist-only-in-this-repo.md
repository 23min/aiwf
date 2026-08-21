---
id: G-0598
title: Shipped skills cite CLAUDE.md sections that exist only in this repo
status: open
priority: high
---
## What's missing

Shipped skills delegate rules to named sections of this repo's `CLAUDE.md`. That file
is repo-development guidance and never ships: `aiwf init` / `aiwf update` materialize
the rituals into a consumer's `.claude/` and maintain a single marker-wrapped import
line in the consumer's root `CLAUDE.md`, and write nothing else into it. So every such
citation resolves here and dangles in every consumer repo.

Measured across the shipped surfaces on the date of filing, roughly twenty citations
in eleven skills name a section of this repo's `CLAUDE.md`: the Q&A and
working-with-the-user section, commit conventions, gate discipline and the
declared-sequence bright line, subagent worktree isolation, the id-shaped-label rule,
worktree placement, ADR authoring, the provenance model, and id-collision resolution.
One reference in the set is correct — the builder agent card points at the consumer's
own project rules generically, naming no section.

Two of them are mis-targeted even in this repo: the wrap-epic and wrap-milestone
rituals cite a *Provenance model* section of `CLAUDE.md`, but that content lives in a
design document, not in `CLAUDE.md` at all.

An existing rule already forbids this. The skills policy holds that a shipped surface
carries only imperative, consumer-scoped instruction and cites no filesystem path. A
pointer into this repo's development doc is both. Only the id half of that rule is
mechanized, by the `skill-body-id` check, and these citations drifted in behind the
unenforced half.

## Why it matters

A dangling pointer that a consumer's assistant follows and cannot resolve is the mild
case: the instruction degrades to whatever the skill says locally.

The severe case is that it resolves to the wrong thing. `CLAUDE.md` is a conventional
filename and *Working with the user* is a conventional heading, so a consumer may well
have a section by that name carrying their own unrelated content. The whiteboard
ritual then delegates its decision format, and the patch and release rituals their
gate discipline, to arbitrary prose that happens to share a heading. Gate discipline
is the rule protecting every irreversible act aiwf walks an operator through, and it
is among the delegated ones.

The failure is silent in both directions. Nothing checks it: the cross-document
citation walk that D-0070 preserves validates skill-to-skill section references only,
so a `CLAUDE.md` target is outside its reach. And because the citations resolve
correctly in this repo, dogfooding cannot surface the breakage either — the one
condition under which the defect appears is the one this repo never runs in.

## Resolution shape

Delete the citation clauses; keep the rules they introduce.

Every site was read before this was filed, and every one restates its rule inline
immediately after the citation. The patch ritual cites the section, then enumerates
all three of its gates in full. The wrap rituals cite commit conventions, then name
the three required trailer keys. The whiteboard ritual cites the Q&A format, then
spells it out in the same sentence. So deletion loses no content at any site, and the
two mis-targeted provenance citations resolve as a side effect rather than needing
their own fix.

Re-pointing the citations at the shipped guidance fragment was considered and
rejected. It covers only the subset whose content actually ships — commit conventions,
ADR authoring, the provenance model and id-collision resolution are repo-development
material that deliberately does not — and for the remainder it trades a visibly broken
pointer for a quietly fragile one, since the fragment's delivery rides an import that
is opt-out-able and that G-0523 already records as able to fail unobserved.

Deletion alone does not stop the pattern regrowing, since what produced it is a rule
whose relevant half nothing enforces. The companion gap filed alongside this one
covers that chokepoint. Sequencing matters between them: the chokepoint fires at error
severity pre-push over exactly these surfaces, so landing it first blocks every push
until this cleanup is done. This lands first, or both land together.
