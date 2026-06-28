---
id: G-0304
title: statusline epic HUD should show only the session's epic, not the in-flight list
status: addressed
addressed_by_commit:
    - dc9be365
---
## Problem

The epic HUD (`.claude/statusline.sh`) showed, on main / non-ritual branches,
the full in-flight epic list (capped at 3 + `+N` overflow) — M-0192's behavior
(G-0188 anti-blank). In practice that list is noise: the operator only wants the
HUD to reflect the entity the current session is working in. The full set is
already available via `aiwf status`.

## Direction

Narrow the HUD to the session's entity, and extend it from epics to gaps:

- Ritual branch (`epic/E-*` → that epic; `milestone/M-*` → that epic with the
  milestone inline) — unchanged.
- Patch branch (`patch/G-NNNN-*`) → the gap the wf-patch is fixing, with its
  status glyph and color — the same session-entity treatment epics get.
- main / non-ritual / gap-less `patch/<slug>` → no HUD segment (blank); the
  backlog lives in `aiwf status`.

Adopt `patch/G-NNNN-<slug>` as the wf-patch branch convention (collapsing the
former fix/patch/chore prefixes) so the HUD can read the gap from the branch
name; a gap-less patch stays `patch/<slug>`. Supersedes the M-0192 AC-1 /
G-0188 non-ritual-list behavior.
