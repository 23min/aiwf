---
id: G-0409
title: 'link-destination rewrite doesn''t handle #fragment/?query suffixes'
status: addressed
discovered_in: M-0245
addressed_by_commit:
    - aa57a1b6
---
## What's missing

\`RewriteLinkDestinations\` (\`internal/verb/linkrewrite.go\`, M-0245) treats a
link destination's entire \`(...)\` contents as a bare path and resolves it via
\`path.Clean\`. A destination carrying a \`#fragment\` or \`?query\` suffix (e.g.
\`(docs/adr/ADR-0004-foo.md#uniform-archive)\`) never matches a moved entity's
\`From\` path, so anchored links are silently left unrewritten on move —
diverging from the primitive's own durability goal for that link shape.

## Why it matters

Anchored entity links are a real, common shape in this repo (design docs
linking to a specific ADR section, cross-references into a named heading).
E-0063's whole purpose is making entity links survive a move; an anchored
link rotting on archive/rename is exactly the failure the epic exists to
prevent, just for a link shape neither M-0245's spec nor the epic's Scope
section currently enumerates. Surfaced during M-0245's independent
code-quality review (both the code-quality and design-quality passes named
it independently); not a defect against M-0245's own ACs, but real scope
the wiring milestones (M-0246/M-0247/M-0248) should account for — either an
added AC before they wire this primitive into \`archive\`/\`rename\`/\`retitle\`,
or an explicit recorded decision to defer.