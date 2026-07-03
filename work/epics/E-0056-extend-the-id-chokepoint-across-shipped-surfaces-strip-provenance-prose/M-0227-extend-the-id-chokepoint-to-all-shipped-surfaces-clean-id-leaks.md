---
id: M-0227
title: Extend the id chokepoint to all shipped surfaces; clean id leaks
status: in_progress
parent: E-0056
tdd: required
acs:
    - id: AC-1
      title: Broadened markdown scan fires on real ids; placeholder silent
      status: open
      tdd_phase: done
    - id: AC-2
      title: Statusline comment scan fires on real ids; shell code exempt
      status: open
      tdd_phase: red
    - id: AC-3
      title: Code, fenced, and link-destination carve-outs preserved
      status: open
      tdd_phase: red
    - id: AC-4
      title: Whole shipped tree green under the broadened check
      status: open
      tdd_phase: red
---
## Goal

The id chokepoint flags a real aiwf-internal id in any shipped surface — the
`description:` frontmatter, entity templates, role-agent cards, the guidance
fragment, and the statusline's comments — not just `SKILL.md` bodies. Every
existing id leak the broadened check would fire on is removed in the same
change, so the check is green and the leak class is mechanically closed.

## Approach

Broaden the scan in `internal/check/skill_body_id.go`:

- Include the `description:` field — parse it out of the frontmatter and scan it
  with the same masked-prose pass used on the body.
- Walk every materialized `*.md` under `embedded` / `embedded-rituals` (drop the
  `SKILL.md`-only filter), covering entity templates and role-agent cards.
- Add `internal/skills/embedded-guidance/` to the scanned roots.
- Add a comment-scoped scan for `internal/skills/embedded-statusline/*.sh` — the
  markdown `proseMask` does not apply to shell, so scan `#` comment text for
  strict id-shapes, leaving shell code exempt.

Keep code spans and link destinations exempt (unchanged carve-out). Then clean
the leaks the broadened check now fires on: rewrite the statusline comments to
drop the id/provenance tags, the `aiwfx-start-epic` description to drop the
`ADR-0023` / `E-03` references, and the `epic-spec.md` template's `E-0002`
example to a placeholder shape.

## Acceptance criteria

Sketch — formalized at start-milestone:

1. A firing fixture per newly-covered surface: a real id planted in a
   `description:`, a template, an agent card, the guidance fragment, and a
   statusline comment each produces an id-chokepoint finding; a canonical
   placeholder in the same position does not.
2. The code and link-destination exemptions are preserved: a real id inside a
   fenced example or an ADR doc-link destination produces no finding.
3. The full shipped tree is green under the broadened check — every existing
   real-id leak found by the G-0348 audit is cleaned.

### AC-1 — Broadened markdown scan fires on real ids; placeholder silent

### AC-2 — Statusline comment scan fires on real ids; shell code exempt

### AC-3 — Code, fenced, and link-destination carve-outs preserved

### AC-4 — Whole shipped tree green under the broadened check

