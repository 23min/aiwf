---
id: G-0592
title: Entity templates carry no genre instruction, so specs accrue rationale
status: addressed
addressed_by_commit:
    - 5100b4f33
---
## What's missing

`templates/adr.md` tells its author what an ADR is for. It carries four blocks of
genre instruction, including *"Keep this section honest: if an alternative was
considered and rejected, name it and say why."*

`templates/epic-spec.md` and `templates/milestone-spec.md` carry none. Neither
says what a spec is for, and the epic template's `## Context` prompt asks *"What
exists today? Why is this work needed now? What prior epics does it build on?"* —
a request for justification with no bound on it.

So nothing tells an author that a spec states what will be built while a judgment,
a rejected alternative, or an argument for why an approach fails belongs in an ADR
or a decision record. Rationale accumulates in the spec instead, where a builder
reads it as requirement.

## Why it matters

The shipped guidance already says the reasoning is worth keeping — *"What no
check can carry — a judgment, a rejected alternative, why the obvious approach
fails — is worth its words."* That is right, and it is why a blanket "state, do
not justify" rule would be wrong. The missing instruction is not *whether* to
record reasoning but *which record holds it*.

Observed while drafting an epic against the milestone-preflight initiative: five
independent review passes returned findings concentrated in the draft's argued
passages rather than in its scope, and removing the argument changed what the
next pass found — from disputed reasoning to missing pieces. Each justification
was individually defensible and locally cheap; the accumulation was neither.

An argued spec also costs twice. It is read on every consultation, and its
argument is the part no check can verify — CLAUDE.md's shipped-surface rule puts
rationale in the class held at review, with no mechanical catch.

## Resolution shape

Give `epic-spec.md` and `milestone-spec.md` the genre line `adr.md` already has:
a spec states what will be built and what is excluded; a judgment or a rejected
alternative is linked, not reproduced. Bound the `## Context` prompt so it asks
for what exists rather than why the work is justified.

Templates are the authoring-moment surface and materialize to consumers, so the
instruction reaches the point where the choice is made. It does not reach an
`aiwf edit-body` on an existing entity; whether that gap needs closing is a
separate question this does not answer.

Of the two, only the first is mechanically pinnable. That both templates open
with a single commented preamble is structural and is pinned by
`TestSpecTemplates_OpenWithACommentedPreamble`. That the `## Context` prompts —
and the matching bullets in `aiwfx-plan-epic` and `aiwfx-plan-milestones`, which
carry the same prompt — stay free of a request for justification is content
correctness, held at review under D-0050 rather than pinned: an assertion over
the phrasing pins a reading rather than a rule.
