---
id: M-0312
title: Re-point the skill-edit backstop from content reference to provenance
status: draft
parent: E-0087
tdd: required
acs:
    - id: AC-1
      title: An untrailered edit to a watched shipped surface is refused
      status: open
    - id: AC-2
      title: A trailered, entity-owned edit passes with no policy test naming its path
      status: open
    - id: AC-3
      title: A trailered edit naming an unresolvable entity is refused
      status: open
    - id: AC-4
      title: CLAUDE.md no longer states that a SKILL.md edit requires a structural test
      status: open
---
## Goal

Replace the skill-edit backstop's content-reference predicate with a provenance one, so
an edit to a watched shipped surface must ride a trailered, entity-owned commit instead
of being accompanied by a policy test that happens to name its path.

## Context

M-0196 shipped the current predicate under E-0048, closing G-0220. That gap's complaint
was mostly about provenance — nothing owned the edit, nothing recorded why it happened —
and the predicate that closed it asks a content question instead: does the edited path
appear as a string literal somewhere in the concatenated policy test sources. D-0071
settles the correction and records why the stronger content variants were rejected
rather than adopted.

This milestone lands before its sibling because the current predicate constrains how the
assertion corpus can be removed. While it stands, each watched skill's path must survive
as a literal somewhere in the policy sources, which limits deletion to whole functions
rather than whole files.

## Acceptance criteria

### AC-1 — An untrailered edit to a watched shipped surface is refused

A commit that adds or modifies a `SKILL.md` under the watched set, carrying no
`aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailers, produces a finding that names the
path and states which trailers are absent. This is the arm G-0220 recorded as silent:
the edit it describes was an ordinary `git commit` against a shipped surface, and
nothing fired.

Fixture: a synthetic commit touching a watched surface with a bare Conventional Commits
subject and no trailers. The policy fires.

### AC-2 — A trailered, entity-owned edit passes with no policy test naming its path

The same edit, riding a commit whose trailers are present and whose `aiwf-entity`
resolves to a real entity, produces no finding — with **zero** policy tests referencing
the edited path anywhere in `internal/policies`.

This is the inversion that retires the content mandate, and it fails today: the current
predicate refuses precisely this case. The fixture must assert silence under the
no-referencing-test condition specifically, since a fixture that happens to run in a tree
where some unrelated test names the path would pass for the wrong reason.

### AC-3 — A trailered edit naming an unresolvable entity is refused

Trailers present, but `aiwf-entity` names an id that resolves to nothing — a typo, a
fabricated id, an entity never created. The policy refuses.

Provenance that points nowhere is not provenance, and converging here would let any
edit satisfy the gate by inventing an id. This mirrors the kernel's own rule that a verb
resolves its arguments before asking whether the request is already satisfied.

Whether the resolved entity must additionally be non-terminal is E-0087's first open
question. It is settled on this milestone and recorded in its Design notes; this AC pins
resolution only.

### AC-4 — CLAUDE.md no longer states that a SKILL.md edit requires a structural test

No section of `CLAUDE.md` asserts that an embedded-rituals `SKILL.md` edit must land
alongside a referencing structural test under `internal/policies/`. The check is an
**absence** assertion over the named sections that carry the rule today — the
ritual-authoring section and the enforcement list.

An absence assertion is deliberate, and it is the reason this AC is not the pattern
D-0070 retires. It is a ban: it costs once, there is no wording to maintain as the prose
around it evolves, and it cannot be satisfied accidentally. It fires if the mandate is
ever re-added.

The positive half — that the surviving prose states the provenance rule correctly and
readably — is **not** asserted. That is held at review, per D-0070's disposition for
content correctness. Asserting it would mint exactly the rotting check this epic exists
to remove.

## Constraints

- The watched surface set does not change. This milestone alters what an edit must
  prove, not which edits are watched.
- The predicate must be decidable without judgment — trailers and entity resolution are
  mechanical; "is this edit well-documented" is not.
- The replacement ships with firing fixtures on both arms: an untrailered or unowned edit
  is refused, and a trailered, entity-owned edit passes with no accompanying test.
- Every surface describing the current mandate is updated in this milestone, not left to
  the sibling. A gate whose documentation still states the old rule is half-landed.

## Design notes

- D-0071 carries the decision, the growth-curve evidence, and the rejected alternatives
  (retire outright; narrow to the evidence-backed classes; strengthen to
  section-level reference).
- `provenance-untrailered-entity-commit` already enforces the analogous property over
  entity files. Reuse that shape rather than inventing a second notion of provenance.
- The policy remains an aiwf-repo invariant and stays inert in a consumer tree.

## Surfaces touched

- `internal/policies/skill_edit_structural_test_backstop.go`
- CLAUDE.md — the ritual-authoring and enforcement sections
- Any shipped guidance that restates the content mandate

## Out of scope

- Deleting the existing prose-assertion corpus. That is the sibling milestone's
  deliverable, and it is sequenced after this one.
- G-0504's separate complaint that `aiwf doctor` byte-checks only verb skills while
  ritual and guidance drift read as healthy.
- Extending provenance enforcement to shippable surfaces the current backstop does not
  already watch.

## Dependencies

- D-0071, accepted.
