---
id: M-0290
title: Retire the rewidth verb and fire the drift check on any narrow active id
status: in_progress
parent: E-0078
tdd: required
acs:
    - id: AC-1
      title: No rewidth command is registered and its verb and CLI packages are absent
      status: met
      tdd_phase: done
    - id: AC-2
      title: A narrow id in an active tree fires at error severity, mixed or not
      status: met
      tdd_phase: done
    - id: AC-3
      title: Archive entries never fire and archived-entity cross-references resolve
      status: met
      tdd_phase: done
    - id: AC-4
      title: No shipped or normative surface tells an operator to run the verb
      status: met
      tdd_phase: done
    - id: AC-5
      title: An ADR records which clauses the retirement supersedes
      status: met
      tdd_phase: done
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

## Work log

### AC-1 — No rewidth command is registered and its verb and CLI packages are absent
Verb package, CLI package, `padToCanonical`, `widenEntityID` and the three
registration sites deleted; absence pinned across the root command's children,
the registered-verbs annotation, and the help banner · commit db307fdc7

### AC-2 — A narrow id in an active tree fires at error severity, mixed or not
Classifier collapsed; any narrow filename id in the active tree fires at error,
remediation naming no verb · commits db307fdc7, 8c75749c6

### AC-3 — Archive entries never fire and archived-entity cross-references resolve
Aggregate zero-findings pin over a narrow-archive/canonical-active tree, plus an
active→archived-narrow reference resolving through the loader and the reference
check · commit 997e4d2f2

### AC-4 — No shipped or normative surface tells an operator to run the verb
Structural scan over four shipped-surface roots plus the Normative tier, skipping
nested `archive/` · commit 997e4d2f2

### AC-5 — An ADR records which clauses the retirement supersedes
ADR-0039 accepted, enumerating lapsed and surviving clauses; ADR-0008 carries the
notice in its preamble · commits 2905b8910, 8c75749c6

## Decisions made during implementation

- **ADR-0039** — retire the verb and supersede ADR-0008 clause-wise rather than
  wholly. Recorded as an ADR because it changes what a published decision means.
- **Supersession is prose, not frontmatter.** `aiwf promote ADR-0008 superseded
  --superseded-by` would flip the original's status and assert the whole
  canonical-width policy had lapsed, which is false — four of its runtime claims
  are live. The kernel has no clause-wise supersession, so the back-link is a
  preamble notice and the original stays `accepted`.
- **AC-1 and AC-2 land in one commit.** They are not independently green: the
  verb ran an `aiwf check` preflight that refuses on error-severity findings, so
  an error-severity narrow-id rule makes it refuse on the very trees it existed
  to convert. Splitting produces a commit whose tree is red.

## Validation

- `make ci` — exit 0 (race suite, diff-scoped coverage audit, firing-fixture
  meta-gate, `aiwf doctor --self-check` 29/29).
- `make lint` — 0 issues.
- `aiwf check` — 0 errors.
- Binary smoke — `aiwf rewidth` returns `unknown command`; the help banner names
  it nowhere.
- Net delta: 66 files, +924 / −3483.

## Deferrals

- **G-0532** — `entity-id-narrow-width` reads only the filename, so a narrow
  frontmatter id under a canonical filename is reported by nothing. Closing it
  requires deciding how it interacts with `idPathConsistent`'s deliberate
  width-blindness, which is a design call rather than a correction.

## Reviewer notes

Three independent fresh-context reviewers ran at wrap: two code-quality slices
and one design pass over `entityIDNarrowWidth`. The design pass returned KEEP on
the collapse. Both code slices returned request-changes; a deciding pass over the
full change-set returned four further blocking findings. All are fixed.

The findings worth carrying forward, because they say something about how this
milestone was built rather than about one line:

- **The rule's registration in `check.Run` had no pin.** Replacing that line with
  `_ = entityIDNarrowWidth` left the whole suite green while the shipped binary
  stopped gating narrow ids — on the one rule this milestone promoted from
  advisory to blocking. Every positive assertion drove the unexported function;
  the two tests that drove `check.Run` both assert zero findings. Now pinned by a
  seam test through `tree.Load` + `check.Run`, verified to kill that mutant.
- **The first sweep tracked what the tests enforced, not what the deletion
  invalidated.** The policy ledgers that were swept are exactly those with
  staleness detection; the four missed are exactly those without.
- **Two archive fixtures passed for the wrong reason.** They used `G-1`, which
  `entity.IDFromPath` rejects below the gap grammar's three-digit floor, so they
  never reached the archive guard they were written to exercise. A mutation probe
  found this; coverage did not.
- **A `//coverage:ignore` asserted an unreachable branch that is reachable.**
  `PathKind` classifies on `^<prefix>-\d+` while `IDFromPath` applies the kind's
  grammar floor, so a sub-floor id loads and lands there. Now covered rather than
  excused.

Declined, recorded here so a later reviewer meets a decision rather than a blank:

- **AC-4's text exempts "the archival tier and the changelog"; the test also
  exempts `docs/adr/`.** The carve-out is right — ADRs are dated decision records,
  superseded rather than rewritten, and ADR-0039 is the mechanism that records
  what lapsed. The AC's wording is what lags, and it is left as written rather
  than retitled to match what was built.
- **`plugin.json` manifests are not scanned** by the shipped-surface check, which
  filters to `.md` and `.sh`. Measured clean; they carry metadata, not prose.
- **ADR-0008 cites a `canonicalPadFor` symbol that no longer exists**, in a
  section ADR-0039 lists as standing. Pre-existing, and left alone rather than
  editing an accepted ADR's body beyond its supersession notice.
- **`R-AUDIT-0121` is tombstoned while `R-RULE-112` was deleted outright.** The
  asymmetry is real; only R-AUDIT has a contiguity check, so only it required a
  tombstone.
- **`entityRootPrefixes` is a hardcoded literal** and nothing fails if
  `entity.PathKind` gains a directory it lacks. Verified identical to the deleted
  derivation, set and order. The coupling predates this milestone.
- **G-0531 was filed and cancelled** — roughly forty comments naming the retired
  verb, in files with no consumer reach and no decision in the work. Tracking a
  mechanical chore costs a reader's attention the fix would not have.
