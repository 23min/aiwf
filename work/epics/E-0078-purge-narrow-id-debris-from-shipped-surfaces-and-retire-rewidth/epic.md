---
id: E-0078
title: Purge narrow-id debris from shipped surfaces and retire rewidth
status: proposed
---

## Goal

Make the canonical placeholder form the only id shape aiwf's shipped surfaces
carry, behind a guard that keeps it that way, and delete the migration verb whose
work is finished.

## Context

ADR-0008 completed the *runtime* width migration: input tolerance, allocator
emit, the drift check, and `prior_ids` resolution are all correct and
load-bearing. It deliberately deferred the example and prose cleanup, and that
debris is what remains.

The cost is behavioral rather than cosmetic. 203 narrow-id sites across 28 files
under `internal/skills/embedded{,-rituals,-guidance}/` materialize into consumer
repos, where they model narrow width as aiwf's current shape to every assistant
that reads them — and where aiwf's own ids are meaningless anyway. The larger
half of that population is narrow *placeholders*, concentrated in the rituals
that drive planning, so an assistant following the epic ritual learns the wrong
id shape before it writes anything.

The guard largely exists. `skill-body-id` already scans the right corpus
whole-file, covers templates and role-agent cards, runs at error severity
pre-push, is inert in consumer repos, and already holds the inverted polarity
these surfaces need. All 203 sites survive it for two specific, closable
reasons.

Separately, every tree the width migration targeted has since been rewidthed. A
narrow id in an active tree is therefore a defect rather than an unfinished
migration, which collapses the drift check instead of complicating it and leaves
`rewidth` with no job.

G-0481 holds the full audit — creation paths, per-tier counts, the two guard
holes, and the retirement blast radius.

## Scope

### In scope

- The two `internal/check/skill_body_id.go` holes: a mask that does not blank
  code spans and fenced blocks, and the placeholder-width check the file already
  claims is policed elsewhere.
- The Tier A sweep — every narrow id and narrow placeholder in the three
  embedded trees converted to canonical placeholder form, the two real-entity
  citations removed, and the entity templates repaired.
- A width-shaped lint over `README.md` and `docs/workflows.md`, plus their sweep.
- Retiring `rewidth`: the verb, its CLI package, `padToCanonical`, its
  registration sites, the collapsed drift rule, and an ADR superseding the
  ADR-0008 clauses that specify the verb.

### Out of scope

- **The `import` mint hole** (`internal/verb/import.go`, the explicit-id entry
  for a new entity). Independent of everything here and ships as a standalone
  patch.
- **The repo-facing doc residue** — `docs/design/**`, `docs/overview.md`,
  `docs/architecture.md`. These cite entities that were genuinely real at narrow
  width, so their fix is widening to the real id, not placeholdering. Filed as
  its own gap during the Tier B milestone.
- **Test fixtures.** A large share of the narrow `id:` values under
  `internal/check/testdata/` *are* the narrow-read-tolerance suite.
- **Code comments**, and everything frozen by convention: `docs/archive/**`,
  `docs/research/**`, `docs/explorations/**`, `CHANGELOG.md`, ADR-0008 itself.
- **Narrow read tolerance.** Permanent, and strengthened rather than weakened by
  this epic — see Constraints.

## Constraints

- **Narrow read tolerance and the drift check's archive exclusion are
  permanent.** `rewidth` skips archive subtrees by design, so every migrated repo
  that archived before migrating still holds narrow ids under `<kind>/archive/`,
  and after retirement nothing can ever widen them. Read tolerance is what keeps
  live cross-references into those entities resolving; it is a standing property
  of the input space, not a legacy concession.
- **`proseMask` stays unchanged.** `body-prose-id` shares it and legitimately
  needs code constructs exempt. The new behavior is a distinct mask, not an edit
  to the shared one.
- **The guard lands at warning severity and flips to error only once its own
  worklist is empty.** No milestone leaves the tree in a state where the sweep is
  incomplete and the push is blocked.
- **Placeholder form is the canonical letter-N shape, never a canonical-width
  fictional id.** A fabricated `` `E-0001` `` is a real entity in most consumer
  trees, which is the collision the placeholder convention exists to prevent.
- **Three files keep their narrow ids**, each citing them as the subject of a
  rule rather than as an example of current shape: the `aiwf-check` skill's
  grammar table, and the two planning rituals that sanction narrow numerics as
  conversational shorthand for not-yet-allocated milestones.
- **Opposite polarities stay opposite.** `body-prose-id` rejects the letter-N
  placeholder in entity bodies; `skill-body-id` requires it in shipped surfaces.
  Both are currently correct and neither moves.

## Success criteria

- [ ] No surface under `internal/skills/embedded{,-rituals,-guidance}/` carries a
      narrow id or a below-canonical-width placeholder, except the three files
      named in the Constraints allowlist — and a gate fails when one returns.
- [ ] The width claim `skill_body_id.go` makes about placeholder normalization is
      true: a letter-N placeholder below canonical width fails a gate.
- [ ] A real entity id written inside a code span or a fenced block in a shipped
      surface fails a gate, where today it passes.
- [ ] The shipped entity templates seed no id shape that `body-prose-id` rejects.
- [ ] `README.md` and `docs/workflows.md` carry no narrow id, and a gate fails
      when one returns.
- [ ] No `rewidth` verb exists, and no shipped or normative surface tells an
      operator to run one.
- [ ] A narrow id anywhere in an active tree produces an error-severity finding,
      whether or not the tree also contains canonical ids.
- [ ] A cross-reference into an archived narrow-id entity still resolves, and
      `aiwf history` on a narrow id still returns its timeline.
- [ ] G-0481 is promoted to `addressed`, and the deferred doc-residue gap exists
      with its own scope.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the placeholder-width check extend `skill-body-id` or ship as a sibling rule with its own finding code? | no | Decided in the first milestone. Same corpus and same polarity argue for extending; a distinct remediation message argues for a sibling. |
| Does the repo-facing doc lint fire from `aiwf check` or from `internal/policies`? | no | Decided in the doc-lint milestone. The check tier fires pre-push and catches in-context; the policy tier is a CI-only backstop. The corpus is repo-only either way, so both are inert for consumers. |
| Does the new ADR supersede ADR-0008 wholly or clause-wise? | no | Decided in the retirement milestone. ADR-0008's runtime claims stay true and load-bearing; only the clauses specifying the verb lapse. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Flipping the guard to error strands an unswept site and blocks pushes. | med | Warning-first ordering is the mitigation: the sweep works from the rule's own worklist rather than from a grep, and the flip to error is the last act of that milestone. |
| A tree still unmigrated at upgrade time meets an error-severity drift finding with no verb able to fix it. | low | Every known tree is migrated. The release carrying the deletion names the prior version as the migration path in `CHANGELOG.md`. |
| The sweep's scale invites mechanical find-and-replace across the allowlisted teaching cases. | low | The three allowlisted files are named in Constraints, and the guard must pass over them unchanged — a sweep that damages them fails the gate it was run against. |

## Milestones

In execution order, though only one edge is real: the sweep needs the guard to
exist, because the guard's output is the worklist it sweeps. The other two are
independent of everything and of each other, so their position here is
convenience rather than sequence.

- **M-0287** — close the two guard holes: a scan that does not blank code
  constructs, and the placeholder-width check the rule already claims is policed
  elsewhere. Lands at warning severity. `tdd: required`.
- **M-0288** — sweep the shipped surfaces to canonical placeholder form, remove
  the two real-entity citations, repair the entity templates, and flip the rule
  to error as the milestone's last act. Depends on M-0287. `tdd: required`.
- **M-0289** — the width-shaped lint over the two workflow-teaching docs and its
  sweep, plus filing the deferred doc-residue gap. Independent. `tdd: required`.
- **M-0290** — retire the migration verb: the deletions, the collapsed drift rule
  at error severity, the superseding ADR, and the kernel-commitment edit.
  Independent. `tdd: required`.

## References

- G-0481 — the audit this epic implements: creation paths, per-tier counts, the
  two guard holes, and the retirement blast radius.
- ADR-0008 — the width migration whose runtime half is complete and whose
  verb-specifying clauses this epic supersedes.
- ADR-0004 — the archive convention that makes archived narrow ids permanent.
- E-0076 — shares the first milestone's pattern (a documented rule with no
  detector) across three unrelated instances.
- G-0454, owned by E-0077 — unifies three id-shape parsers in `internal/entity`.
  Distinct from the fourth parser this epic deletes with the verb, so there is no
  sequencing constraint between the two epics.
