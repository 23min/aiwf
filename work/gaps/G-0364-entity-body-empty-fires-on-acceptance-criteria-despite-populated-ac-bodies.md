---
id: G-0364
title: entity-body-empty fires on Acceptance criteria despite populated AC bodies
status: open
discovered_in: M-0231
---
## What's missing

\`entity_body.go\`'s \`entityBodyEmpty\` rule treats a milestone's \`## Acceptance
criteria\` section as non-empty when it contains \`### AC-N\` sub-headings — but
only within that section's own scanned content window, which \`scanH2Sections\`
bounds at the *next* \`## \` heading. The current \`templates/milestone-spec.md\`
places \`## Constraints\`, \`## Design notes\`, \`## Surfaces touched\`, \`## Out of
scope\`, \`## Dependencies\`, and \`## References\` between \`## Acceptance
criteria\` and where \`aiwf add ac\`'s \`appendACHeading\` actually appends new
\`### AC-N\` headings (always at the absolute end of the body, per its own doc
comment). So on any milestone using the current full template, \`##
Acceptance criteria\`'s scanned window is only ever the HTML comment
(stripped to nothing) — the AC headings live elsewhere, disconnected. The
\`entity-body-empty\` warning fires regardless of whether the AC headings (or
their prose) are populated.

An older milestone (M-0066) doesn't hit this because its simpler template had
no sections between \`## Acceptance criteria\` and the wrap-side \`##
Decisions made during implementation\`, so its AC headings happened to land
inside the scanned window.

## Why it matters

The warning is permanently un-clearable on any milestone using the current
template — filling in AC-heading prose (the template's own instruction) does
not clear it, since the check never looks there. A warning that can never be
satisfied by doing the thing it asks for trains operators to ignore it,
which defeats its purpose as a real content-completeness signal. Fix
candidates: teach \`entity_body.go\` to scan for \`### AC-N\` headings
anywhere in the body (not bounded by the next \`## \`), or move \`## Constraints\`
etc. before \`## Acceptance criteria\` in the template, or have \`aiwf add ac\`
insert immediately after the \`## Acceptance criteria\` heading instead of at
body-end.