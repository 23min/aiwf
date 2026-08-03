---
id: G-0488
title: The loader's area normalization rides into the next verb's commit
status: wontfix
priority: low
discovered_in: M-0282
---
## What's missing

The tree loader normalizes as it reads, and the next write-verb commits that
normalization as though it were its own change.

`tree.Load` blanks `area` on every entity whose kind does not carry its own —
milestones derive theirs from the parent epic. The comment records the intent
plainly: `omitempty` drops the cleared key on the next write-verb, "auto-stripping
the invalid value". So a milestone whose committed frontmatter carries an `area:`
key loses it the next time any verb rewrites that file, in that verb's commit,
under that verb's trailer, with no event naming the change.

The trigger is narrow — a milestone can only acquire an `area:` key by hand-edit,
by import, or from a tree written before the field's scope was closed — but the
mechanism is not: the loader is free to normalize anything, and every serializing
verb re-serializes from the loaded model rather than editing the fields it owns.

## Why it matters

This is the same defect E-0075 is named for — a verb committing frontmatter it
does not own — reached by a different route, and one that epic's precondition
cannot see. There is no divergence between disk and HEAD when it happens: the
working copy is clean, the record is clean, and the strip is introduced *between
reading and writing*. A guard comparing disk against HEAD is looking at the wrong
pair.

That makes it worth recording for a reason beyond its own severity. E-0075's
root-cause framing is that verbs compare against a projection of disk while
believing they compare against the record. This case shows the framing is
incomplete: the deeper cause is that a verb rewrites a **whole file
re-serialized from an in-memory model** rather than editing the fields it owns,
and that model is lossy by design. HEAD-divergence is one way the rewrite goes
wrong; normalization is another, and the surgical-commit approach ADR-0038 defers
is the shape that would address both.

## Scope

The interaction between loader-side normalization and whole-file re-serialization
on the write path. The `area` strip is the instance in hand; the general question
is which normalizations the loader performs and whether any of them may reach a
commit unannounced.

Out of scope: whether blanking a milestone's `area` is correct — it is, the field
is not meaningful there. The question is who commits the correction, and under
what name.

## Resolution options

1. **Make the strip explicit.** A verb whose serialization would change a field it
   was not asked to change either refuses or reports. Consistent with E-0075's
   verdict, but needs a notion of which fields a verb owns — the per-verb
   ownership map that ADR-0038 defers as the bulk of the surgical-commit work.
2. **Stop normalizing at load; validate instead.** Leave the invalid value in the
   model and let a check rule report it, so the correction becomes a deliberate
   act with its own commit. Cleaner separation, and it matches this project's
   stance that errors are findings rather than silent repairs. Costs a migration
   for any tree currently relying on the auto-strip.
3. **Leave it, and record it as accepted.** Defensible on the narrowness of the
   trigger. Costs nothing now, and leaves a second unowned-write route open for
   whoever next asks why the guard did not catch something.

Option 2 is the lean on principle — a silent repair is exactly what this project
treats as a finding elsewhere — but it is the most invasive. Option 1 becomes
cheap once the ownership map exists, so the sequencing may decide this rather
than the merits.
