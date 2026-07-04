---
id: G-0363
title: 'epic-spec template ships completed: field the strict decoder rejects'
status: open
---
## What's missing

The embedded epic-spec template ships a `completed:` frontmatter key that no
entity struct field accepts, so a consumer who fills in the template verbatim and
runs `aiwf check` hits a hard `load-error` instead of a clean tree.

- Template source: `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md:6`
  — `completed:               # optional: YYYY-MM-DD, filled at wrap`.
- The entity frontmatter is decoded with `KnownFields(true)` (strict) — see the
  forward-compat note at `internal/entity/entity.go:412` — so any unknown key is
  a fatal decode error, not a silently-ignored field.
- The struct exposes `id`, `title`, `status`, `depends_on`, `parent`, `area`,
  `tdd`, `acs`, `supersedes`, `superseded_by`, … but **no `completed`**.

Observed while authoring E-0057: pasting the template's frontmatter produced

```
load-error: decoding frontmatter: yaml: unmarshal errors:
  field completed not found in type entity.Entity
```

and `aiwf show` / `aiwf list` could not see the entity until the line was removed.

## Why it matters

A shipped scaffold that its own strict loader rejects is a broken onboarding
path: the template is materialized into every consumer repo by `aiwf update`, and
the natural first act — fill in the frontmatter — yields an error the user did
not cause. It also contradicts the kernel principle that a template is a
ready-to-use starting point.

Two candidate fixes (settle when the gap is worked, not here):

1. **Drop `completed:` from the template.** Simplest; matches today's struct.
   Loses the (currently non-existent) completion-date affordance the comment
   promises.
2. **Add a `completed` field to the entity struct** (e.g. `completed string
   yaml:"completed,omitempty"`, filled at wrap), making the template honest and
   giving epics a real completion date. Larger — touches the struct, the wrap
   ritual, and possibly render surfaces.

A template-vs-struct drift test (assert every frontmatter key in every embedded
template is an accepted entity field) would prevent recurrence and is the
load-bearing part of whichever fix lands.

## Scope

The chosen fix (drop the key or add the field) plus a drift test pinning every
embedded-template frontmatter key to an accepted entity field. Audit the other
embedded templates (milestone, ADR, decision, contract, gap) for the same class
of stray key while here.
