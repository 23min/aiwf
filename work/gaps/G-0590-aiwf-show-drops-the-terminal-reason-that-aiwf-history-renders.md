---
id: G-0590
title: aiwf show drops the terminal reason that aiwf history renders
status: open
priority: high
---
## What's missing

`aiwf show` is the verb the shipped `aiwf-show` skill calls the canonical
per-entity inspection view. For a terminal entity it prints status, acceptance
criteria, references and a list of history events — and drops the one field that
says why the entity died.

Measured: 110 of 119 entity-level `cancel` commits carry a `--reason` body. The
renderer for it already exists, at `internal/cli/history/history.go:135`, and
`aiwf history` prints it. The HTML render path carries the same field. Only
`show` omits it.

The gap is visible in one command pair. `aiwf history E-0058` prints a
four-hundred-word finding: four independent adversarial reviews, a force-push
path that silently re-paid the walk cost the design existed to eliminate, and
the simpler watermark design that replaced it. `aiwf show E-0058` prints the
title, `status: cancelled`, one reference and three event lines. The epic's body,
which `show` does surface through `--format=json`, still opens by asserting that
this epic is the root-cause fix.

## Why it matters

A cancelled entity's body and its cancellation reason are different kinds of
document, and the more useful one is the one `show` hides. 36 of 37 cancelled
milestones never reached `in_progress`, so their bodies are plans that were never
tested against anything. The reason is a judgment about reality, attributed to a
named human, written at the moment the evidence was in hand.

So a reader who opens a terminal entity through the canonical verb meets the
untested plan and not the finding — and the plan reads as live, because one word
of frontmatter is the only thing marking it otherwise. Dismissing dead material
without reading it is what forgetting means in practice, and the field that would
let a reader do that is already written, already parsed, and already rendered by
two other surfaces.

The cost is bounded by that: this is not new capture, and it asks nothing of
whoever writes the next reason. It routes a record the project already keeps to
the surface its own documentation points readers at.

Two questions belong to the work rather than to this gap: whether the reason
appears for every terminal transition or only for the declining ones, and how a
long reason renders without crowding out the rest of the view.
