---
id: M-0288
title: Sweep shipped surfaces to canonical placeholders and enforce at error severity
status: draft
parent: E-0078
depends_on:
    - M-0287
tdd: required
acs:
    - id: AC-1
      title: No embedded surface carries a narrow id or placeholder outside the keep-list
      status: open
    - id: AC-2
      title: Shipped entity templates seed no id shape that body-prose-id rejects
      status: open
    - id: AC-3
      title: The keep-list files retain their teaching tokens unchanged
      status: open
    - id: AC-4
      title: The rule runs at error severity and this repo passes it
      status: open
---

## Goal

Convert every narrow id and narrow placeholder in the shipped surfaces to
canonical placeholder form, remove the two real-entity citations, repair the
entity templates, and close the milestone by turning the guard to error so the
debris cannot return.

## Context

The preceding milestone gives this one a mechanical worklist: run the rule, sweep
what it names, re-run. The population is 203 sites across 28 files — 51 narrow
numerics and 152 narrow placeholders — against 213 already-canonical placeholder
forms that show what the target looks like.

The sweep target is the canonical letter-N placeholder, uniformly. Canonicalizing
the digits would be the wrong fix twice over: it misses the 152 placeholders
entirely, and a fabricated canonical-width id is a *real* entity in most consumer
trees, which is the collision the placeholder convention exists to prevent.

## Acceptance criteria

### AC-1 — No embedded surface carries a narrow id or placeholder outside the keep-list

Every file under the three embedded trees is free of narrow ids and
below-canonical-width placeholders, except the three keep-list files.

This subsumes the two real-entity citations — a real milestone id in an
`aiwf-edit-body` fenced comment and a real epic id in an `aiwf-acknowledge`
`--reason` example. Both entities are `done` and archived, which is the rot the
skills policy predicts; the preceding milestone's rule catches them as real ids
rather than as width defects, and neither is fixed by widening.

Evidence: the rule from the preceding milestone, run over the real tree, reports
zero findings.

### AC-2 — Shipped entity templates seed no id shape that body-prose-id rejects

The shipped milestone, decision, and ADR templates carry narrow placeholders
today, and `body-prose-id` names exactly that shape as a leak it rejects in entity
bodies. The frontmatter and heading occurrences are inert, since `aiwf add` stamps
the real id over them; the prose-guidance lines are author-facing text that can
survive into a committed body, where the shipped check then fires on a consumer's
own entity.

Evidence: `ScanBodyProseID` run over each shipped template's body reports zero
findings — the shipped check applied to the shipped template, so the two surfaces
cannot contradict each other again.

### AC-3 — The keep-list files retain their teaching tokens unchanged

Two of the three keep-list files need surgical edits rather than exemption: the
planning rituals keep their narrow *numerics*, which teach the
conversational-shorthand rule, while their narrow *placeholders* must go. A
mechanical find-and-replace across the corpus would destroy the teaching cases.

Evidence: a positive assertion that each keep-list file still contains the
specific tokens its rule depends on, so a sweep that flattens them fails rather
than passing silently.

### AC-4 — The rule runs at error severity and this repo passes it

The severity flip is the last act of the milestone: it lands only once the sweep
is complete, so no intermediate state blocks a push on debris that has not been
cleared yet.

Evidence: an assertion on the emitted finding's severity, plus this repo's own
`aiwf check` at zero errors.

## Constraints

- **Placeholder form is the canonical letter-N shape, never a canonical-width
  fictional id.**
- **The severity flip lands last.** No commit in this milestone leaves the tree
  with an incomplete sweep and an error-severity rule.
- **The keep-list files are edited surgically**, never swept mechanically.
- **Worked-example transcripts lose distinct ids and keep their titles.** The
  fictional scenario in the status and list skills renders every milestone as the
  same placeholder; the titles carry the distinctions. A guard with no transcript
  carve-out is worth more here than the vividness, because the sweep is re-run by
  a rule rather than by eye.

## Design notes

- `PolicySkillEditStructuralTestBackstop` requires every modified
  `embedded-rituals/**/SKILL.md` to be referenced by some `internal/policies`
  test. Nine ritual skill files need edits in this sweep and all nine already
  carry a referencing test, so the backstop is satisfied without new test-writing
  work. The verb-skill tree is outside the backstop's scope, and role-agent cards
  and templates are not `SKILL.md` at all.

## Out of scope

- The repo-facing doc corpus.
- Test fixtures, whose narrow ids are largely the narrow-read-tolerance suite.
- Code comments.

## Dependencies

- M-0287 — supplies both the worklist and the assertion this milestone is
  measured against.

## References

- G-0481 — per-tier counts, the two real-entity citations, the template vector.
