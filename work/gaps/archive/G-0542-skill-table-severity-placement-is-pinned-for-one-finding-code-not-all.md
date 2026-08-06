---
id: G-0542
title: Skill-table severity placement is pinned for one finding code, not all
status: addressed
addressed_by_commit:
    - 8b6b30061
---
## What's missing

The `aiwf-check` skill documents every finding code in one of two tables,
headed *errors* and *warnings*. Which table a code sits in is how a consumer
answers "will this block my push?" — the placement is the signal, not the
prose in the cell.

`TestPolicy_SkillBodyIDRowMatchesEmittedSeverity` pins that placement against
the severity the rule actually emits. It does so for exactly one finding code,
`skill-body-id`, named as a literal in the test. Its own comment states the
reasoning in general terms — an operator reads the errors table as "this blocks
my push" — and that reasoning holds for every documented code.

So the property is asserted for one row and the rest of the table is free to
drift. It has. A sweep of the table found four rows whose placement disagrees
with the severity the rule emits:

- `area-required` — emits error, documented under warnings.
- `provenance-untrailered-entity-commit` — emits error, documented under
  warnings.
- `milestone-done-zero-acs` — emits warning, documented under errors.
- `milestone-draft-incomplete-acs` — emits warning, documented under errors.

The first two are unambiguous: a flat error severity with no configuration
knob. The last two are less certain — the sweep was a regex heuristic, and
several rules here escalate severity from a knob in `aiwf.yaml`, which a regex
cannot evaluate.

A fifth row, `entity-id-narrow-width`, carried the same defect and was moved
into the errors table alongside the fix for G-0532. That change was already
correcting the identical mislabel in the legal-workflows audit, and leaving the
shipped surface wrong while fixing the internal one would have been worse than
leaving both.

## Why it matters

The table is a shipped surface. It materializes into every consumer repo, and
it is what an operator reads to decide whether a finding stops their push today
or is something to schedule. A row in the wrong table is not cosmetic there: it
either promises a block that will not come, or threatens one that will not
either.

Fixing the four rows without widening the check leaves the table exactly as
free to drift as it was — the next rule added, or re-severitied, lands wherever
its author puts it and nothing notices. The row moves are the symptom; the
one-code scope of the check is the defect.

## Direction

Widen `TestPolicy_SkillBodyIDRowMatchesEmittedSeverity` from its single literal
to every code the skill documents, deriving emitted severity from the rule
rather than from a second hand-maintained list. The row moves then fall out as
the change that makes it pass.

One design question has to be answered first, and it is why this is a gap
rather than a sweep: what the check should demand of a rule whose severity is
configuration-dependent. Several rules emit warning by default and error under
an `aiwf.yaml` strictness knob. Three answers are available — file them by
their default severity, file them by their strict severity, or give the skill a
third category for knob-escalated findings — and the choice decides both the
check's shape and how much of the table moves.

## Provenance

Found 2026-08-04 during the independent review of the G-0532 patch, by a
reviewer sweeping the skill table to judge whether that patch's own row move
was in scope. The condition predates that patch.
