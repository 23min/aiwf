---
id: E-0084
title: Enforce body-section membership at the write seams
status: proposed
---
## Goal

Make a body that omits a section its kind requires impossible to write, so "required"
stops being a name and starts being a refusal — and delete the prose that exists only
because nothing enforced it.

## Scope

- A scan over the bytes a verb is about to write, wired into every body-supplying
  verb, refusing at error severity for every kind.
- A gate on the push, riding the commit range the provenance audit already resolves,
  scoped to entities whose body content this push changed.
- Deleting the prose that states the section set once a refusal carries it. The
  enforcement is what makes the deletion safe; landing it without the deletion leaves
  the second copy in place and spends the epic for nothing.
- Closing G-0571.

## Out of scope

- The existing violations. 217 gap bodies and 27 decision bodies omit a required
  section; enforcing across the tree would raise 119 findings over 60 live entities,
  118 at error severity. Both seams read only bytes being written, so these are never
  in scope — that is the mechanism, not a grandfather clause. Paying the debt is a
  migration and needs its own evidence.
- Emptiness. Whether a section that is present carries content is ADR-0042's subject
  and E-0083's work: enforced at the readiness transition, not at a write.
- Whether the milestone template's four structured-data sections should exist at all
  (G-0530). That asks whether a section is worth carrying; this epic asks only that a
  declared one is present.
- Changing what any kind's required set contains. The set is E-0081's answer and this
  epic consumes it.

## Context

`entity.RequiredSections` has been the single definition of each kind's body sections
since E-0081, and every surface stating the set now derives from it or is tested
against it. Nothing enforces it. `RequiredSections`' own doc comment says so outright:
*"'Required' names what the scaffold writes, not a guarantee anything verifies."*

`entity-body-empty` reports a section that is present and empty; one absent outright is
skipped by design, and the `aiwf add` born-complete gate consults the same helper, so
it inherits the blind spot. The result is perverse rather than merely incomplete:
handed a body whose required section is present and empty, the gate refuses and says
`aiwf check` will block until it is filled — and an operator can satisfy that refusal
by deleting the heading instead of filling it, after which nothing says anything. The
stricter body is the one that is harder to land.

Measured: a body carrying invented top-level headings is accepted at creation for every
kind and reported by nothing afterwards. The rate declines but has not stopped — seven
of the last sixty-eight gaps, most recently G-0543.

ADR-0043 decides where enforcement lives and why it is forward-only by construction
rather than by policy. This epic implements it.

## Constraints

- Neither rule joins `check.Run`. `aiwf check`'s tree-wide output does not change, and
  no existing entity gains a finding.
- The push gate is scoped to entities whose *body content* changed in the range, never
  to entities merely touched. A status promote, a retitle, or an archive sweep on a
  non-conforming entity must stay unaffected; a touched-path scope would block ordinary
  work on debt it did not create.
- A violation is one thing: a required section absent as a top-level `## ` heading.
  Sections beyond the set are legal and never flagged. Order is not enforced.
- The scan reads `RequiredSections` and nothing else. The prose templates are not an
  input, so a template edit cannot change what is enforced.
- No workflow that is available today may become unavailable. An author who does not
  yet know a section's content keeps the heading and leaves it empty.
- Coordinate the finding code with E-0083 before either lands. Both add a body rule;
  ADR-0043 leaves open whether one code serves both properties or each needs its own,
  and that is cheaper to settle once than to reconcile twice.

## Success criteria

- [ ] Every body-supplying verb refuses a body omitting a required section, and a test
      per verb fails if the call is removed.
- [ ] A body reaching a commit without passing any verb is refused at the push, proven
      against the path that does this today rather than a synthetic one.
- [ ] `aiwf check` on this tree reports the same findings before and after the epic.
- [ ] A status promote, a retitle, and an archive sweep each succeed against an entity
      whose body omits a required section.
- [ ] Every passage listed in *Prose retired* is deleted, and a test that would have
      failed had it been merely corrected instead.
- [ ] G-0571 is `addressed`.

## Prose retired

Each states the section set, and each is deletable only once a refusal carries it:

- the `aiwf-add` skill's per-kind body-section table
- the body-sections table in `docs/design/design-decisions.md`
- `RequiredSections`' own "not a guarantee anything verifies" caveat, which the
  enforcement makes false

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| One finding code for membership and emptiness, or one each? | no | Settled with E-0083 before either epic's first milestone lands; ADR-0043 names it. |
| Does the push gate need its own severity, or does it inherit? | no | Settled in the milestone that builds it, against what the provenance audit already does. |
| Should the wrap-milestone ritual's plain-`git commit` write of the milestone spec route through `aiwf edit-body`? | no | Its own change, on its own merits — the push gate covers the hole either way. |

## Risks

- The push gate rides the provenance audit's range, which is skipped when no upstream
  is configured and no `--since` is passed. The gate inherits that. CI-on-push is the
  backstop; the epic should not pretend otherwise.
- A future body-writing verb that does not call the scan is unenforced at the verb
  seam. The push seam still covers it, so the failure degrades to late feedback rather
  than none.

## Milestones

- the verb seam: the scan, wired into every body-supplying verb, with a per-verb test
- the push seam: the gate on the provenance range, scoped to body-changed entities
- the deletion: retire the prose the enforcement makes redundant, and close G-0571

## References

- ADR-0043 — the decision this epic implements
- ADR-0042 — the adjacent decision, emptiness at the readiness transition
- E-0083 — the epic implementing ADR-0042; the finding-code question is shared
- G-0571 — the hole this closes, and the source of the 119-finding measurement
- E-0081 — gave the section set one owner and deliberately excluded enforcement
- G-0530 — the adjacent, out-of-scope question of section membership
