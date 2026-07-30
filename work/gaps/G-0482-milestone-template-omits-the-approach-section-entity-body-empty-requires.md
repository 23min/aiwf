---
id: G-0482
title: Milestone template omits the Approach section entity-body-empty requires
status: open
priority: low
discovered_in: E-0075
---
## What's missing

The `entity-body-empty` rule requires a milestone body to carry `Goal`,
`Approach` and `Acceptance criteria`. No shipped surface produces an `Approach`
section.

Measured: the milestone template ships `Goal`, `Context`, `Acceptance criteria`
and a dozen further sections, with zero occurrences of the word "Approach"
anywhere in it. The `aiwf add milestone` scaffold ships `Goal` and
`Acceptance criteria`. The string appears nowhere under the embedded rituals. Of
285 milestone files in this tree, 54 carry `## Approach`; none of the eight most
recent do.

The requirement is inert rather than violated. A section whose heading is missing
outright is not "empty" in this rule's sense — a stance its own doc comment
states deliberately — so the rule stays silent on every milestone that omits the
section entirely, which is the shape the template guarantees.

## Why it matters

This is the same class as G-0479 — a required-sections list disagreeing with the
template meant to satisfy it — with a different failure mode. G-0479 is a heading
at the wrong *level*; this is a section name that exists on no shipped surface at
all. Two instances across two different kinds makes the divergence structural
rather than a one-off, which is the argument for a detector rather than two
point fixes.

The substantive question underneath is unresolved either way. Either `Approach`
is load-bearing, in which case every milestone drafted from the shipped template
is missing a required section and nothing says so; or it is not, in which case
the rule names a section the project has stopped writing and the requirement
belongs on `Context`. Today the requirement is neither enforced nor withdrawn.

## Scope

Reconcile the surfaces that disagree for the milestone kind:
`requiredSectionsByKind`, the shipped milestone template, the `aiwf add
milestone` scaffold, and the body-key contract `aiwf show --format=json` exposes.

Out of scope: the epic-kind instance, tracked as G-0479; and any general detector
for surfaces drifting from the rules that read them, which E-0076 owns.

## Resolution options

1. **Require `Context`, drop `Approach`.** Matches the template, the scaffold and
   recent practice in one edit. The requirement starts doing real work the day it
   lands, because every milestone drafted from the template already has the
   section.
2. **Add `## Approach` to the template and the scaffold.** Keeps the rule's
   stated intent and treats the template as the thing that drifted. Costs a
   section on every new milestone, and does nothing for the existing files that
   omit it, since a missing heading still does not fire.
3. **Require neither — leave `Goal` and `Acceptance criteria`.** Honest about
   what is actually enforced today, at the cost of dropping a requirement nobody
   has argued against on its merits.

Option 1 is the lean. It is the only one where the requirement becomes live
immediately rather than aspirationally, and `Context` is what both the template
and current practice already produce. Option 2 is defensible only if someone can
say what `Approach` is for that `Context` is not — which is the question the
divergence has been deferring.
