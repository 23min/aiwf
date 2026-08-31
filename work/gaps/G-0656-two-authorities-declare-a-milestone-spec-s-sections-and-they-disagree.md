---
id: G-0656
title: Two authorities declare a milestone spec's sections and they disagree
status: open
---
## What's missing

Two surfaces declare what sections a milestone spec has, in two languages, and
they disagree. Nothing reconciles them and no check notices.

`internal/entity/required_sections.go` names two for the milestone kind, `Goal`
and `Acceptance criteria`. `templates/milestone-spec.md` — materialized into
every consumer repo by `aiwf init` / `aiwf update` — ships a much larger set,
and it is the one authors and rituals actually fill. Measured 2026-08-30: the
kernel declares 2, the template carries 17 `##` headings.

## Why it matters

The disagreement decides whether a milestone can close.

M-0326 added a check that reads the *template's* authority: it reports a
milestone reaching `done` with no release-note section. That rule was briefly
landed at error severity, which makes it a `promote` precondition, and a
milestone created through `aiwf add` could then not reach `done` at all — the
kernel demanded a section its own declaration does not list, `aiwf template
milestone` does not write it, and `--force` does not relax a projection finding.
The rule was reverted to warning, which sidesteps the collision without
resolving it.

Any future rule inherits the same trap from whichever authority it reads. One
keyed to the kernel set cannot see the sections the template ships and consumers
fill; one keyed to the template demands sections `aiwf add` never writes. The
choice is currently made per rule, by whichever file its author happened to
open.

## Direction

Name one authority and derive the other from it.

Which one is open. The kernel set has the mechanical consumers — it is what
`aiwf add` writes and what `entity-body-empty` reads — while the template set is
the vocabulary in actual use. Growing the kernel to match the template is the
blast radius G-0571 measures. Deriving the kernel from the template means kernel
code reading a materialized markdown file, which inverts the layering the repo
holds elsewhere. Neither is free, which is why this is filed rather than fixed
in passing.

## References

- G-0571 — nothing enforces that an entity body carries its kind's required
  sections. The enforcement half of the same problem, and it already carries an
  inherited obligation from M-0326.
- G-0636 — milestone-spec section rules are restated across five surfaces with
  no owner. The prose half.
- M-0326 — added the consumer that surfaced this, and reverted its severity
  rather than resolving it.
