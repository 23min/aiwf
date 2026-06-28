---
id: G-0304
title: statusline epic HUD should show only the session's epic, not the in-flight list
status: open
---
## Problem

The epic HUD (`.claude/statusline.sh`) shows, on main / non-ritual branches, the
full in-flight epic list (capped at 3 + `+N` overflow) — the behavior M-0192
shipped to satisfy G-0188's "don't show blank on main" goal. In practice that
list is noise: the operator only wants the HUD to reflect the epic the current
session is working in. The full in-flight set is already available via `aiwf
status`.

## Direction

Narrow the epic HUD to the session's epic only:

- Ritual branch (`epic/E-*` → that epic; `milestone/M-*` → that epic + the
  milestone inline) — unchanged.
- main / non-ritual / patch branch → render no epic HUD segment at all (blank).

This supersedes the non-ritual in-flight-list behavior from M-0192 and the
anti-blank rationale of G-0188: blank-on-main is acceptable because `aiwf
status` covers the backlog. The implementation is mostly deleting the
non-ritual list / cap / overflow branch; flip the M-0192 AC-1 harness test from
"renders the in-flight list on main" to "renders no epic HUD on main."
