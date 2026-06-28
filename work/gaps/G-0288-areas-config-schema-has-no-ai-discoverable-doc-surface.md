---
id: G-0288
title: 'areas: config schema has no AI-discoverable doc surface'
status: open
discovered_in: M-0179
---
## Problem

No AI-discoverable surface documents the `aiwf.yaml: areas:` block schema. The fields
`members`, `default`, `required` (E-0043 / M-0178) and now `members[].paths` (E-0044 / M-0179)
are documented only in the `internal/config/config.go` struct doc comments — source-level, not
reachable via `aiwf <verb> --help`, an embedded skill, CLAUDE.md, or the design docs.

Per the kernel principle "Kernel functionality must be AI-discoverable," a user-facing config
field needs a channel an AI assistant or human routinely consults. Discovered in M-0179
(`wf-review-code`). This predates M-0179 — even the E-0043 fields lack a discoverable schema
doc, so M-0179 inherits the gap rather than opening it.

## Direction

Land a discoverable areas-config schema doc when `paths` gains observable behavior (M-0180),
or sooner as its own change. Candidate homes: an `aiwf-check` / areas skill section, a design
doc cross-referenced from CLAUDE.md, or `--help` text on a relevant verb. Cover the full block
(`members` / `default` / `required` / `paths`) so the whole schema is reachable, not just paths.
