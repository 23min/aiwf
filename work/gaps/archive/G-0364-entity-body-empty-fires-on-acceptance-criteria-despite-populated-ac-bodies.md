---
id: G-0364
title: entity-body-empty fires on Acceptance criteria despite populated AC bodies
status: addressed
discovered_in: M-0231
addressed_by_commit:
    - 2eed14d0
---

## What's missing

`entity_body.go`'s `entityBodyEmpty` rule treats a milestone's `## Acceptance
criteria` section as non-empty when it contains `### AC-N` sub-headings — but
only within that section's own scanned content window, which `scanH2Sections`
bounds at the *next* `## ` heading. `aiwf add ac`'s `appendACHeading`
(`internal/verb/ac.go`) rewrites an existing `### AC-N` heading in place when
one already exists for that id, but otherwise always appends the new heading
at the absolute end of the body — never scoped to the Acceptance-criteria
section. `aiwf add milestone` itself scaffolds a bare two-section body
(`## Goal` / `## Acceptance criteria`, no gap between them), so this doesn't
fire there. The rich `templates/milestone-spec.md` (`## Constraints`, `##
Design notes`, `## Surfaces touched`, `## Out of scope`, `## Dependencies`,
`## References` between Acceptance-criteria and body-end) only lands via the
`aiwfx-plan-milestones` ritual pasting it in as static text, complete with
placeholder `### AC-1` / `### AC-2` headings already positioned inside the
section.

So the failure is narrower than "any milestone on the current template": the
placeholder AC-1/AC-2 headings get rewritten in place correctly when `aiwf
add ac` targets them, landing inside the scanned window with no bug. It
breaks for AC-3 onward (no placeholder left to rewrite), or for any AC whose
placeholder text was already replaced with real prose before `aiwf add ac`
ran for it — both land past `## Reviewer notes` at body-end, outside the
window, so `## Acceptance criteria`'s scanned content reduces to just the
stripped HTML comment. Reproduced live: a milestone on the rich template with
AC-1/AC-2 already de-placeholdered, both with genuine prose, still fires
`entity-body-empty` on `## Acceptance criteria`.

An older milestone (M-0066) doesn't hit this because its simpler template had
no sections between `## Acceptance criteria` and the wrap-side `##
Decisions made during implementation`, so its AC headings happened to land
inside the scanned window.

## Why it matters

The warning is un-clearable once a milestone accumulates an AC beyond the
template's two placeholders (or de-placeholders one early) — filling in
AC-heading prose does not clear it, since the check never looks where the
heading landed. A warning that can never be satisfied by doing the thing it
asks for trains operators to ignore it, which defeats its purpose as a real
content-completeness signal.

Recommended fix: have `aiwf add ac` insert new `### AC-N` headings inside the
Acceptance-criteria section (after the section heading, or after the last
existing `### AC-N` heading there) instead of at body-end — the actual root
cause, reusing the existing section-boundary scanner rather than a third
implementation. Two alternatives considered and ruled out:

- Teaching `entity_body.go` to scan for `### AC-N` headings anywhere in the
  body, unscoped, would make the check blind on every wrapped milestone —
  `acsBodyCoherence` (`internal/check/acs.go`) already had to work around
  this exact trap for its duplicate-heading check, since `## Work log`
  legitimately repeats `### AC-N — <outcome>` headings with the identical
  shape.
- Reordering the template so nothing sits between Acceptance-criteria and
  body-end doesn't address the root cause: `appendACHeading` appends at
  absolute end-of-body regardless of section order, so AC-3+ (or a
  de-placeholdered AC) would still land past whatever now sits at the end
  (Work log, Reviewer notes, etc.).
