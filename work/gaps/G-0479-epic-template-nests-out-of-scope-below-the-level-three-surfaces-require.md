---
id: G-0479
title: Epic template nests out-of-scope below the level three surfaces require
status: addressed
priority: low
addressed_by_commit:
    - 01b1cf758
---
## What's missing

The shipped epic template nests its out-of-scope section one heading level
below what three kernel surfaces name, so an epic drafted by following the
ritual as written satisfies none of them.

`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md`
places `### Out of scope` as a subsection of `## Scope`. Against that:

- `internal/check/entity_body.go` lists an epic's load-bearing top-level body
  sections as `Goal`, `Scope`, `Out of scope`, all at `##`.
- `aiwf template epic` — the scaffold `aiwf add` writes — emits `## Goal`,
  `## Scope`, `## Out of scope`, flat.
- `aiwf show --format=json` documents the epic body map as
  `goal/scope/out_of_scope`.

The `entity-body-empty` rule does not fire on the mismatch, because a section
whose heading is missing outright is not "empty" in that rule's sense — a
deliberate stance, stated in the rule's own doc comment. So the divergence is
silent on both sides: the check passes, and the JSON key simply never appears.

Measured: every epic in this tree drafted from the rich template — E-0070 and
E-0071 among them — carries no `out_of_scope` key in `aiwf show --format=json`.
Epics carrying the flat form do. The `aiwfx-plan-epic` ritual instructs the
operator to replace the scaffolded body with the rich template, so following the
ritual is what produces the divergence.

## Why it matters

Three surfaces state the same contract and a fourth quietly breaks it, which is
the shape where a reader stops looking. An operator who reads
`internal/check/entity_body.go` concludes `## Out of scope` is required and
enforced; it is named but unreachable for template-drafted epics. A consumer
reading `body.out_of_scope` from the JSON envelope gets nothing for exactly the
epics most likely to have a considered out-of-scope section, and no error says
why.

The cost today is small — no data is lost, the prose is present under a
different heading level. What makes it worth filing is that it is a concrete,
fully-measured instance of a class already tracked: a shipped surface that
drifts from the behaviour it describes, with reading as the only detector. It is
useful as a calibration target for whatever detector that work produces, because
the answer here is mechanically checkable in a way most prose drift is not.

## Scope

The heading level in the embedded template, and whichever surface is judged to
be the one that should move. `aiwf update` re-materializes the embedded copy
into consumers, so the fix travels without a migration step.

Already-drafted epics are a separate question from the template fix: their
bodies keep the nested form until edited, so a template change alone does not
make their `out_of_scope` key appear.

## Resolution options

1. **Flatten the template** to `## Out of scope`, matching the check rule, the
   `aiwf add` scaffold, and the JSON contract. Smallest change, and it moves the
   one surface that is out of step with the other three. Costs the visual
   pairing of in-scope and out-of-scope as siblings under `## Scope`.
2. **Teach the body parser to descend**, so a `### Out of scope` nested under
   `## Scope` populates `out_of_scope`. Preserves the template's structure, but
   makes the section-to-key mapping depth-sensitive for one kind, and leaves
   `aiwf add`'s scaffold and the template disagreeing about shape.
3. **Drop `Out of scope` from the required-sections list** and from the
   documented JSON body map, conceding that it is not load-bearing. Honest if
   true, but it is the section that records deliberate exclusions, which is
   exactly the content a reader most needs and cannot reconstruct.

Option 1 is the lean: one surface is out of step with three, and it is the one
with no dependants — nothing reads the template's heading levels except the
operator following the ritual.

Whichever is chosen, the four surfaces should end up agreeing, and a test should
assert that they do — otherwise the next template edit reopens this silently.
