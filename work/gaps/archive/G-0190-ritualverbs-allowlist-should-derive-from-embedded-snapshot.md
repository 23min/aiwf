---
id: G-0190
title: ritualVerbs allowlist should derive from embedded snapshot
status: addressed
discovered_in: E-0038
addressed_by_commit:
    - ab75d376
---
## What's missing

`internal/check/trailer_verb_unknown.go` carries a hand-maintained `ritualVerbs`
map (`wrap-epic`, `wrap-milestone`) that the `trailer-verb-unknown` rule consults
alongside the Cobra command tree. The values were transcribed from the embedded
rituals at G-0180 fix time but have no mechanical link to the snapshot under
`internal/skills/embedded-rituals/`. If a future ritual stamps a new non-kernel
`aiwf-verb:` value (e.g. `start-epic`, `record-decision`), the allowlist must be
extended by hand — otherwise the kernel fires `trailer-verb-unknown` on every
commit the new ritual produces, exactly the same bug class G-0180 closed.

The comment at line 48 of the file already notes this: "ideally derive it from
the embedded snapshot — tracked alongside G-0180."

## Why it matters

The gap reopens the G-0180 bug class silently whenever a ritual author adds a
trailer verb and forgets to update the kernel allowlist. The kernel and the
rituals are authored by the same team today, but the coupling is invisible — no
test or build step fails if the two diverge. A drift test (akin to
`TestRituals_VendoredMatchesUpstream`) that extracts `aiwf-verb:` values from
the embedded skill markdown and asserts they appear in `ritualVerbs` would close
the loop mechanically.
