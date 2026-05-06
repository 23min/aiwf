---
id: G-053
title: No verb-flag populates resolver-pointer fields on status transitions
status: open
discovered_in: E-14
---

## Problem

Several status transitions in the kernel require a pointer field to be set as part of the transition: `gap.addressed` requires `addressed_by` (entity id) or `addressed_by_commit` (sha); `adr.superseded` requires `superseded_by`; similar pattern likely exists or will exist for other transitions that name a successor or resolver. The corresponding check rules (`gap-resolved-has-resolver`, etc.) correctly demand these fields are populated. But no verb route accepts the necessary argument: `aiwf promote G-NNN addressed` has no `--by <id>` or `--by-commit <sha>` flag, so the only way to satisfy the check is to hand-edit frontmatter — which the skill text forbids and which itself triggers `provenance-untrailered-entity-commit`.

## Behavior-distorting evidence

Users default to `wontfix` in place of the semantically-correct `addressed` because `wontfix` doesn't require a resolver pointer and so doesn't force them through the broken path. The verb route's incompleteness is visibly distorting user behavior away from accurate state — a gap that was genuinely addressed gets recorded as wontfix, just to dodge the friction. That's the strongest signal this is a load-bearing gap rather than a stylistic concern: it's actively degrading data quality.

## Root cause

Same pattern as G-052 (body edits), one level up: the kernel expresses a structured-state invariant (a transition needs a pointer) but the verb route doesn't carry the argument needed to satisfy it. The user is stuck choosing between hand-editing (rule violation) or picking a worse status (data-quality loss).

## Direction

Extend the relevant mutating verbs to accept the resolver-pointer flag(s) that the corresponding check rule requires:

- `aiwf promote <gap-id> addressed --by <M-NNN | E-NN | sha>` populates `addressed_by` (entity id) or `addressed_by_commit` (sha) depending on argument shape.
- `aiwf promote <adr-id> superseded --superseded-by <ADR-NNNN>` populates `superseded_by`.
- Generalize as new pointer-requiring transitions are introduced — flag wiring should be a small, repeatable pattern, not a per-transition special case.

The verb checks the projected frontmatter against the same check rule (`gap-resolved-has-resolver`, etc.) before committing — same atomicity and validation model as today.

## Relationship to G-051 / G-052 / E-15

Same theme: kernel demands behavior the verb routes can't deliver, forcing users into rule-violating workarounds. Worth solving alongside the body-edit and batching work in E-15 — the implementation pattern is the same shape (extend verb signatures with the arguments they should always have had).
