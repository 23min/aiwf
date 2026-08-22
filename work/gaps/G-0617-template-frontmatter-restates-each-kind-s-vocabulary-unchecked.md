---
id: G-0617
title: Template frontmatter restates each kind's vocabulary, unchecked
status: addressed
priority: medium
addressed_by_commit:
    - db843cead
---
## What's missing

Every entity template under
`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/` opens with
a frontmatter block whose comments restate the kind's closed vocabulary — the
status set, and which fields are optional. All six carry one, in the shape
`status: open  # aiwf gap statuses: open | addressed | wontfix`.

That vocabulary already has an owner. `entity.schemas` declares each kind's
`AllowedStatuses` and `OptionalFields`, and `aiwf schema <kind>` prints them.
The comments are a hand-maintained second copy, and nothing links the two: a
grep for `AllowedStatuses` or `OptionalFields` across `internal/policies`
returns no template-related hit. Four policies read these templates — they
check that the frontmatter parses, that the required sections are present at
`## ` level, that a title heading is marked optional, and that the placeholder
id resolves to a kind. None reads the vocabulary the comments teach.

The block is also unusable in place. `validateUserBodyBytes`
(`internal/verb/common.go`) refuses body content beginning with a `---`
delimiter, on both the `aiwf add --body-file` and `aiwf edit-body --body-file`
routes, so a filled-in template must have its frontmatter deleted before the
verb will accept it.

## Why it matters

Add a status to a kind's FSM and the six templates go on teaching the old set,
in every consumer repo, with `aiwf check` clean and CI green. The copy is the one
nothing re-derives, and its readers are the least equipped to notice: a consumer
has no neighbouring entities to compare against, which is the reason the
templates ship at all.

The frontmatter is not inert weight, which is what makes the trade real rather
than obvious. The placeholder id inside it is how a template file is resolved to
its kind — filenames do not carry the kind, since two of the six are named
`epic-spec.md` and `milestone-spec.md` rather than for the kind they serve. Any
change that removes the block has to answer what replaces that mapping first.
