# Epic wrap — E-0078

**Date:** 2026-08-03
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0078-purge-narrow-id-debris-from-shipped-surfaces-and-retire-rewidth
**Merge commit:** c17332bdf

## Milestones delivered

- M-0287 — Detect real ids in code spans and below-width placeholders in shipped surfaces (merged 5da71935c)
- M-0288 — Sweep shipped surfaces to canonical placeholders and enforce at error severity (merged 151fb0471)
- M-0289 — Lint and sweep narrow ids from README and the workflows guide (merged 1fa79916b)
- M-0290 — Retire the rewidth verb and fire the drift check on any narrow active id (merged cde20e842)

## Summary

Narrow-width ids were still being taught. They sat in the skills an assistant
reads while operating aiwf, in the entity templates it copies from, and in the
repo's own README and workflow guide — modelled as current shape long after the
kernel had settled on four digits. This epic closed the guard that was supposed
to catch them, swept every surface it covers, and then removed the migration
verb whose existence implied the transition was still under way.

The guard had two holes rather than one. It masked code constructs, so every
numeric id hiding in a command example or a fenced transcript passed unseen; and
it declined to check width at all, so a placeholder built from three N's read as
correct. Both are closed, and the corpus now includes the surfaces the rule
always claimed to cover.

Retiring the verb turned out to be forced rather than optional. Once a narrow id
in an active tree is an error, the verb's own `aiwf check` preflight refuses on
it — so the migrator would have declined to run on exactly the trees it existed
to convert. That also collapsed the drift rule rather than merely rewording it:
its uniform-versus-mixed classifier existed only to stay silent on a
pre-migration tree, and no such tree remains.

What the epic deliberately did not do is widen narrow ids in the design docs and
architecture notes, where the citations are of entities that were genuinely
narrow at the time. That residue has its own scope.

## ADRs ratified

- ADR-0039 — retire the rewidth verb; ADR-0008's migration clauses lapse

## Decisions captured

- D-0051 — extend skill-body-id with subcodes, not a sibling rule
- D-0052 — dissolve the shipped-surface keep-list instead of mechanizing it

## Follow-ups carried forward

- G-0514 — skill-body-id flags CLI metavariables and non-id acronyms
- G-0516 — comment-history-attrition misses stale comments outside the changed lines
- G-0517 — widen narrow id citations in the design docs, overview, and architecture
- G-0529 — CHANGELOG completeness rests on recall at epic wrap and is never checked
- G-0532 — entity-id-narrow-width reads only the filename, not frontmatter

G-0531 was filed and cancelled during M-0290's wrap: roughly forty comments
naming the retired verb, in files with no consumer reach and no decision in the
work. Tracking a mechanical chore costs a reader's attention the fix would not.

## Doc findings

Doc-lint over every markdown file the epic touched — 62 documents. Four
unresolved `aiwf` invocations, each a disposition rather than a defect:

- `docs/adr/ADR-0008-…md` and `docs/adr/ADR-0039-…md` name the retired verb.
  Deliberate: ADRs are dated decision records, superseded rather than rewritten,
  and ADR-0039 is the record of what lapsed. The retirement's own scan exempts
  `docs/adr/` for this reason.
- `docs/initiatives/adoption-maturity-advisor.md` proposes an `aiwf advise`
  verb. Forward-looking tier; a proposed verb is not drift.
- `ROADMAP.md` names the verb inside an archived epic's own goal prose. The file
  is generated, and the source is frozen by the archival convention.

No broken intra-repo links and no removed-feature docs in the epic's change-set.
The documentation TODO markers under `docs/initiatives/` describe TODO density
as a maturity signal; they are the subject, not a marker left behind.

## Handoff

The canonical-width policy is now enforced rather than migrated toward: the
allocator emits canonical width, no verb can produce anything narrower, and a
narrow id in an active tree is an error at the pre-push boundary. Narrow read
tolerance stays, and its justification changed — with nothing able to widen an
id in place, it is what keeps live cross-references into entities archived
before the convention resolving. That is a standing property of the input space
now, not a transitional courtesy, and ADR-0039 says so.

Left open deliberately: the frontmatter axis (G-0532), where a narrow `id:`
under a canonical filename is reported by nothing. Closing it means deciding how
the width rule interacts with `idPathConsistent`'s deliberate width-blindness —
a design call, not a correction. G-0517 carries the design-doc residue the epic
scoped out from the start.
