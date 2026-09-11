# Epic wrap — E-0091

**Date:** 2026-09-11
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0091-hold-each-spec-section-to-what-it-uniquely-holds-and-check-what-ships

## Milestones delivered

- M-0326 — Add a Release note section and write the epic wrap's changelog from it (merged 1ecbe7661)
- M-0327 — See entity-trailered commits in history, at AC granularity, guarded on commit (merged c9a889d4c)
- M-0329 — An entity body that omits a required section is refused at the write (merged 092f72418)
- M-0330 — Check Unreleased against what shipped before the release tag (merged b3be35c77)

Four patches carried the epic's prose-shaped work, each closing a gap and none
carrying an acceptance criterion:

- G-0636 — every milestone-spec section rule gets one owner (merged 94a97fa86)
- G-0665 — what an acceptance criterion body holds gets an owner (merged db514ab28)
- G-0530 — `## Work log` retires, its unique fact derivable without it (merged c6ecab7d8)
- G-0613 — the changelog category set widens to Keep a Changelog's six (merged 913cf0fa8)

M-0328 was allocated and cancelled without being written; its goal and criteria
sections are empty. What it would have carried is held by D-0086 and the
`aiwf-add` skill's per-kind body table instead.

## Changelog entry

### Added — E-0091: four gates that hold a spec section, an entity body, and a release's notes to what they claim

A milestone spec now carries `## Release note` — the user-visible delta, written
at the milestone's own wrap by whoever did the work — and `aiwfx-wrap-epic`
composes the epic's changelog entry from those notes rather than reconstructing
it from milestone titles and merge SHAs. `aiwf check` reports a `done` milestone
whose note an author never wrote (`milestone-done-empty-release-note`, warning).
A companion check resolves the section names written in the ritual tree against
the headings the shipped templates actually carry, so a surface naming a section
no artefact has is reported.

`aiwf add` and `aiwf edit-body` now refuse a body missing a section its kind
requires, and the two ask different questions because a create and an edit sit
in different positions. `add` wants every declared section, for every kind, and
`--force --reason` still bypasses it — now stamping its trailer on epic and
milestone too, where it was previously inert. `edit-body` refuses only a write
that *drops* a section the committed body carries, in both bless and
`--body-file` mode, so an author editing one section is never refused over an
omission they did not introduce; a deliberate removal is recorded with
`aiwf acknowledge illegal`. Both refusals name the section they missed.

The `commit-msg` hook gained three refusals, each catching at composition what
was previously found later or not at all: a subject naming an `(M-NNNN/AC-N)`
scope whose `aiwf-entity` trailer names something else or nothing; an aiwf
trailer block git will not read, because a blank line leaves it out of the
message's final paragraph, making it invisible to `aiwf history` and carrying
any unrecognized `aiwf-verb` value straight past the check meant to refuse it;
and a staged edit to the shipped ritual tree whose message names no entity.
Alongside them, `aiwf history <id>` now lists a commit whose only aiwf trailer
names the entity — the implementation commits and shipped-surface edits it used
to discard — rendering `-` where a verb and actor would be.

A release tag now fails when its notes do not say what it ships. The
`changelog-check.yml` workflow gained a second job running `make
changelog-audit`, which compares every commit since the last release that
changed aiwf's embedded skill, ritual, template, agent-card, hook and guidance
content against what the release notes cite, and fails the tag on a shipped
change no entry names. A milestone's delta is cited by its parent epic, never by
its own id; Go source under those trees materializes the content rather than
being content a consumer receives, so it owes no entry. A second finding — a
shipped-surface commit carrying no `aiwf-entity` trailer — is reported without
failing, since with no entity named the audit cannot tell whether an entry
covers it. Run `make changelog-audit` before tagging rather than meeting it at
the push; `AIWF_CHANGELOG_BASE=<ref> make changelog-audit` audits a past range,
and unset, the audit does not run at all.

## Summary

The epic set out to make a milestone spec carry only what no other record holds,
and to make a shipped change reach the changelog by check rather than by recall.
Both landed. `## Work log` — the spec's largest section, with no downstream
reader — retired once `aiwf history` could answer the one fact it uniquely held,
the link from an acceptance criterion to its implementing commit; `## Release
note` replaced it with a section that has a named consumer. Each spec section
now has one owner, chosen by where the rule binds, which settled the
`## Validation` timing contradiction as a side effect.

Scope shifted in one place worth naming. M-0328 was allocated for what an
acceptance criterion holds, then cancelled unwritten: the answer turned out to
belong with the per-kind body table in the `aiwf-add` skill rather than with the
spec's section rules, so it shipped as a patch under D-0086 instead of as a
milestone. Section enforcement also stops short of the tree — a write-time
refusal reads only the bytes it is writing, so the entities already missing a
section are untouched, and closing that is E-0084's push seam.

## ADRs ratified

- ADR-0048 — a create must be complete; an edit must not regress

## Decisions captured

- D-0085 — a milestone-spec section is owned by the surface where it is first written
- D-0086 — an acceptance criterion's content rule is owned by the `aiwf-add` skill
- D-0087 — the changelog audit blocks an uncited entity and reports an untrailered commit
- D-0088 — a changelog entry uses one of Keep a Changelog's six categories

## Follow-ups carried forward

Three gaps this epic names in *References* stay open, each recording in its own
body what landed and what remains:

- G-0530 — `## Work log` is retired; `## Dependencies`, `## Surfaces touched`
  and `## References` remain, each blocked on a question this epic did not answer
- G-0571 — the write half is closed; the bodies already committed without a
  section are E-0084's push seam
- G-0657 — the commit-time route is shut; the landed population of unparseable
  trailer blocks is historical record and is not rewritten

Opened during the epic and still open:

- G-0656 — no rule records whether it keys to the kernel's scaffold set or the template's
- G-0658 — nothing pins that root reaches the commit-msg guard that reads the index
- G-0661 — no check caps an in-function comment at the length D-0084 sets
- G-0662 — `wf-patch`'s review asks none of the five shape questions the wrap asks
- G-0663 — the confirmation round after a review finding has no chokepoint
- G-0664 — `shipped-prose-assertion` misses an assertion whose shipped path is built inline
- G-0666 — a body line over 64 KB makes a full section report as empty
- G-0667 — `aiwf import` is deprecated but every surface presents it as current
- G-0669 — the two embedded-tree ban policies duplicate one walk and one rule shape
- G-0670 — shipped skills name internal iteration labels and a repo filesystem path
- G-0671 — the changelog audit does not see a delta in a named kernel surface
- G-0672 — two policies duplicate the git range scan over a commit range

Named in a milestone's deferrals and open from before:

- G-0602 — a conflict resolved inside a shipped skill escapes the provenance gate
- G-0659 — a review finding is recorded twice and the spec copy is the one that drifts

## Doc findings

Clean. The scoped sweep covered the six narrative docs this epic touched —
`CLAUDE.md`, ADR-0043, ADR-0048, `design-decisions.md`, `legal-workflows-audit.md`
and `legal-workflows-first-principles.md` — checking markdown link integrity
(relative targets and heading anchors, resolved against the linked file) and
heading-hierarchy sanity. No broken links, no skipped heading levels. The two
`TODO` matches are prose about TODOs in shipped code, not markers.

## Handoff

Ready for the next epic. The section set is settled and enforced at the write,
the changelog is checked at the tag, and `aiwf history` answers the AC-to-commit
link that made retiring `## Work log` safe.

Deliberately left open: the tree-side half of section enforcement, which is
E-0084's push seam and needs a baseline ledger for the entities already carrying
the debt; the named-surface half of the changelog audit (G-0671), which is the
half that would catch a delta in a finding code, a verb, an `aiwf.yaml` key or
an exit code, and an epic that shipped only compiled behaviour; and the
remaining three duplicating spec sections, each of which wants its own
measurement of what a reader loses.

Two clean-up threads surfaced and were recorded rather than taken: duplicated
git-range scanning across policies (G-0669, G-0672), and shipped skills naming
things a consumer cannot resolve (G-0670).
