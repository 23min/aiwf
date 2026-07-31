---
id: G-0463
title: 'edit-body --body-file is not body-only: frontmatter drift rides the commit'
status: addressed
priority: medium
discovered_in: M-0281
addressed_by_commit:
    - 51d04c88
---
## What's missing

`aiwf edit-body <id> --body-file <path>` is documented as body-only, and bless
mode enforces that with an explicit frontmatter comparison. Explicit mode has no
such guard. It re-serializes the entity as the loader read it, which means any
frontmatter difference already present in the working copy is carried into the
commit alongside the body change.

Measured: with `area: kernel` added by hand to a working-copy entity's
frontmatter and a body byte-identical to the committed one, `edit-body
--body-file` committed the frontmatter change (`1 file changed, 1 insertion(+)`)
under an `aiwf-verb: edit-body` trailer.

## Why it matters

The structured-state verbs exist so that changes to `status`, `title`, `parent`,
`area` and the rest are auditable as what they are. `edit-body`'s own doc points
operators at `aiwf promote` / `rename` / `cancel` / `reallocate` for exactly this
reason, and bless mode refuses a frontmatter diff with that message.

Explicit mode therefore offers a route that lands a structural change wearing a
body-edit trailer. `aiwf history` shows a body edit; the provenance rules see a
properly trailered commit; nothing reports that a field moved. The exposure is
small — it needs a working copy already carrying frontmatter drift — but the
whole point of routing edits through verbs is that the commit says what happened.

## Scope

Pre-existing, and independent of the same-state convergence work that surfaced
it. The convergence guard added to explicit mode neither worsens nor improves
this: when frontmatter differs, the serialized result differs from HEAD, so the
verb declines to converge and behaves exactly as it did before.

## Options

1. **Mirror bless mode's guard** — compare the working copy's frontmatter against
   HEAD's and refuse when they differ, pointing at the structured-state verbs.
   Symmetric with the mode that already does it, and the refusal message exists.
2. **Serialize from HEAD's frontmatter** rather than the loader's, so explicit
   mode writes the supplied body onto the committed frontmatter and cannot carry
   drift. Silently correct rather than instructive, and it would quietly discard
   an edit the operator may have meant to make elsewhere.

Option 1 is the lean: refusing tells the operator something true about their
tree, and matching bless mode keeps one rule for one verb.
