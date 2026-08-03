---
id: M-0290
title: Retire the rewidth verb and fire the drift check on any narrow active id
status: in_progress
parent: E-0078
tdd: required
acs:
    - id: AC-1
      title: No rewidth command is registered and its verb and CLI packages are absent
      status: open
      tdd_phase: green
    - id: AC-2
      title: A narrow id in an active tree fires at error severity, mixed or not
      status: open
      tdd_phase: done
    - id: AC-3
      title: Archive entries never fire and archived-entity cross-references resolve
      status: open
    - id: AC-4
      title: No shipped or normative surface tells an operator to run the verb
      status: open
    - id: AC-5
      title: An ADR records which clauses the retirement supersedes
      status: open
---

## Goal

Delete the width-migration verb whose work is finished, and convert the drift
finding it existed to point at into a plain statement that a narrow id in an
active tree is a defect.

## Context

Every tree the migration targeted has been migrated. That single fact turns the
drift check inside out: its mixed-versus-uniform classifier exists only to stay
silent on a uniform-narrow tree, because that meant *pre-migration*, and no such
tree remains. The rule therefore collapses rather than needing the delicate
rewording a retirement would otherwise force, and the verb is left with no job.

This is a net deletion. What it adds is one ADR and one edit to the kernel's
commitments; what it removes is a verb package, a CLI package, a fourth
id-shape parser, and a branch of classification logic.

## Acceptance criteria

### AC-1 — No rewidth command is registered and its verb and CLI packages are absent

The command does not appear in the root command's children or its help listing,
and the verb package, the CLI package, and `padToCanonical` are gone.

The parser deleted here sits outside the three that the tracked id-parser
unification covers, all of which live in the entity package — so this removes a
fourth site an id-grammar change would have to touch, without changing that gap's
scope or creating a sequencing constraint against the epic that owns it.

Evidence: an assertion that no registered command carries the name, plus the
completion-drift test — which enumerates registered verbs — passing with it
absent.

### AC-2 — A narrow id in an active tree fires at error severity, mixed or not

A narrow-width id anywhere in the active tree produces an error-severity finding,
whether or not canonical ids sit alongside it. The uniform-narrow silence is
gone, because the state it modelled no longer occurs.

The remediation the finding names is undoing the hand-edit or file move that
produced it. No verb widens an id in place any more, and the reallocation verb is
not the answer — it assigns a different number rather than the same one at
canonical width.

Evidence: fixtures for a uniform-narrow tree and a mixed tree, both firing at
error, plus this repo's own tree passing.

### AC-3 — Archive entries never fire and archived-entity cross-references resolve

The archive exclusion is permanent, not incidental. The retired verb skipped
archive subtrees by design, so any repo that archived before migrating still
holds narrow ids there, and after this milestone nothing can ever widen them.

That makes narrow read tolerance load-bearing for live cross-references into
archived entities, not merely for history queries against old commit trailers.

Evidence: a fixture tree with narrow archived entries and canonical active ones
reporting zero findings; a cross-reference from an active entity to an archived
narrow one resolving through the loader; and a history query on a narrow id still
returning its timeline.

### AC-4 — No shipped or normative surface tells an operator to run the verb

The finding's remediation text, the check skill's mention, the root help listing,
and the kernel commitment that names the verb all stop referring to it.

Evidence: a structural assertion that the verb name appears in no shipped surface
and in no normative doc outside the archival tier and the changelog, both of which
are frozen by convention.

### AC-5 — An ADR records which clauses the retirement supersedes

The migration ADR's runtime claims — input tolerance, allocator emit, drift
detection, prior-id resolution — remain true and load-bearing. Only the clauses
specifying the verb lapse. The new ADR says which, so a reader of the original is
not left guessing how much of it still holds.

Evidence: a structural assertion that the ADR exists, is `accepted`, and names
the superseded clauses; and that the original carries the reciprocal link.

## Constraints

- **Narrow read tolerance is not touched.** Canonicalization, the grep
  alternation, prior-id resolution, and narrow commit trailers all stay. This
  milestone removes a write path and a migration tool, never a read path.
- **The archive exclusion stays**, permanently and for a stated reason.
- **The kernel commitment that names the verb is edited, not deleted** — the
  stable-ids commitment it sits inside remains true.

## Design notes

- The epic leaves open whether the new ADR supersedes the migration ADR wholly or
  clause-wise. Decided here: clause-wise. A whole supersession would imply the
  runtime migration is no longer authoritative, which is false and would strand
  four live properties with no record.

## Out of scope

- Any sweep of example ids. Independent of the shipped-surface and doc
  milestones, and blocked by neither.
- The narrow-read-tolerance code paths.

## Dependencies

- None. Independent of every other milestone in the epic.

## References

- G-0481 — the retirement blast radius and the permanence argument for read
  tolerance.
- ADR-0004 — the archive convention that makes archived narrow ids permanent.
