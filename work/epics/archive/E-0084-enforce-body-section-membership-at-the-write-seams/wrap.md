# Epic wrap — E-0084

**Date:** 2026-09-17
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0084-enforce-body-section-membership-at-the-write-seams

## Milestones delivered

- M-0331 — Refuse at the push a body that reached a commit without passing a verb (merged 26d6dc980)
- M-0332 — Retire the prose that restates the section set (merged de38696f3)

The verb seam this epic planned as its first milestone was delivered before
either of these by M-0329, carried by E-0091 and delivering this epic's first
scope item early from a different epic; the epic's spec lists it struck through
rather than as a milestone of its own.

## Changelog entry

### Added — E-0084: the push refuses an entity body that dropped a required section

`aiwf check`, and so the pre-push hook, now refuses a push that leaves a required
section — `## Goal`, `## What's missing`, and the rest of each kind's set — out of
an entity's body, reporting `entity-body-section-dropped` at error severity. It
catches edits that never passed through `aiwf edit-body`, including a plain
`git commit` carrying aiwf trailers, and it judges everything the checked-out
branch adds over its upstream as one push: a section removed on a branch merged
in, or in a file a later commit renamed, archived or reallocated, is reported, and
one added and removed again within the push is not. A section already missing
where the branch left its base, or already missing on trunk, is never reported.
An entity the push creates must carry every required section unless
`aiwf import` or `aiwf add --force` created it; an import in per-entity commit
mode now stamps `aiwf-verb: import` on each commit, so `aiwf history` shows
those entities as imported rather than added, and the `aiwf status` digest,
which counts only `add` commits under "Gaps opened" and "ADRs created", no
longer counts them there. Restore
the heading with its content to clear the finding — for a gap, decision, ADR or
contract that is not terminal, an empty required section is itself an error — or
keep a removal with `aiwf acknowledge illegal <sha> --reason "..."`, adding
`--for-entity <id>` when the same commit is also reported by
`provenance-untrailered-entity-commit`. The check runs only when the branch has
an upstream or `--since <ref>` is passed, and the pre-push hook runs it on the
branch checked out where the push is made.

### Changed — E-0084: the `aiwf-add` skill routes to `aiwf template` instead of restating the section set

The `aiwf-add` skill no longer carries a per-kind table of required body
sections. Run `aiwf template <kind>` to see what a kind requires; `aiwf add`
and `aiwf edit-body` already refuse a body that omits one, and the refusal
names the missing headings. Consumers see the change on their next
`aiwf update`.

## Summary

The epic set out to make a body that omits a required section impossible to
write, so that "required" is a refusal rather than a name, and to delete the
prose that existed only because nothing enforced it. Both landed: the verb seam
under M-0329, the push seam under M-0331, and the two section tables under
M-0332, each replaced by a route to the declaration that owns the set. The push
seam is what closed G-0571, because a body can reach a commit without passing
any verb, and the wrap-milestone ritual's own plain `git commit` is that path.

Scope shifted in two places. The verb seam moved to E-0091 before this epic's
first milestone started, so the milestones delivered here are the push seam and
the deletion. And `aiwf import` stays outside the verb seams on its deprecation,
which no surface records (G-0667), so the epic's first success criterion — every
body-supplying verb refuses — is left open on purpose and closes when that gap
does.

## ADRs ratified

- ADR-0049 — a create must be complete; an edit or a push must not regress
  (supersedes ADR-0048)

## Decisions captured

- D-0090 — absent and empty required sections get separate codes
- D-0092 — the push seam asks non-regression, not completeness; superseded
  within the epic by ADR-0049, which follows an entity by path through the moves
  the push made and holds an entity the push creates to completeness unless
  `aiwf import` or a forced `aiwf add` wrote it

## Follow-ups carried forward

Named in the epic's References and still open:

- G-0667 — `aiwf import` is deprecated but every surface presents it as current;
  the first success criterion closes with it
- G-0530 — milestone specs mandate four sections that duplicate structured data;
  the adjacent, out-of-scope question

Deferred by M-0331, each an edge of the push gate's reach:

- G-0679 — a branch started from a local ref is never judged at its first push
- G-0684 — a path git quotes is invisible to the push gate
- G-0685 — the pre-push hook judges the checked-out branch, not the refs being pushed
- G-0686 — a move the push makes outside one add-and-delete commit reads as a create
- G-0687 — `acknowledge illegal --for-entity` refuses a merge commit

Deferred by M-0332:

- G-0674 — the `--principal` flag's help text cites an internal iteration label
- G-0675 — the section-set scan's corpus is hand-maintained and narrows in silence

## Doc findings

Clean. The scoped sweep covered the five narrative docs this epic touched —
ADR-0048, ADR-0049, `design-decisions.md`, `legal-workflows-audit.md` and
`legal-workflows-first-principles.md` — checking markdown link integrity
(relative targets and heading anchors), TODO-class markers, heading-hierarchy
sanity, and every backticked `aiwf` invocation on the lines the epic added
against the verbs `aiwf --help` lists. No broken links, no markers, no skipped
heading levels, and every invocation resolves.

## Handoff

Ready for the next epic. A required section is now a refusal at every seam a
body passes but one: `aiwf add` and `aiwf edit-body` at the write, the push for
whatever reached a commit another way. What a kind requires is read from one
declaration, and the two tables that restated it are gone.

Deliberately left open: `aiwf import`, outside the verb seams until its
deprecation is recorded one way or the other (G-0667); and the five edges of the
push gate's reach that M-0331 deferred — three ways for a body to go unjudged
(G-0679, G-0684, G-0685), one refusal over debt the pusher did not create
(G-0686), and one finding with no clearing form (G-0687). ADR-0043, ADR-0048,
D-0092 and G-0571 are terminal and still in the active tree, and this epic joins
them once it closes; the next archive sweep moves them all, and a wrap does not
run one.
