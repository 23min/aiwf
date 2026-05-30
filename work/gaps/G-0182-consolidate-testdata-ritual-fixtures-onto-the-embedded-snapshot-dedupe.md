---
id: G-0182
title: Consolidate testdata ritual fixtures onto the embedded snapshot (dedupe)
status: open
discovered_in: E-0038
---
## What

The `internal/policies/testdata/<skill>/SKILL.md` fixtures are **separate
copies** of ritual skill content that also lives, post-E-0038, in the
canonical embedded snapshot at
`internal/skills/embedded-rituals/plugins/*/skills/<skill>/SKILL.md`. The two
can drift: the per-AC *content-assertion* tests under `internal/policies/`
read the fixtures, while `aiwf init`/`update` ships the embedded snapshot.

## Evidence

This duplication produced the M-0152 loose end (resolved in the retire-
cache-comparison-tests commit): the wrap-epic fixture carried a `Record
learnings` section that the pinned snapshot had dropped, and the marketplace-
era drift test fired on the divergence. The cache-comparison tests are gone,
but the fixtures still duplicate the embedded snapshot and can silently drift
from what actually ships.

## Desired resolution

Point the per-AC content-assertion tests at the **embedded snapshot** (the
canonical, drift-checked-against-upstream copy — `skills.ListRituals()` /
`ListRitualAgents()` / `ListRitualTemplates()` already expose its bytes) rather
than at a separate `testdata/` fixture, and delete the duplicated fixtures.
This makes the content tests assert against exactly what ships, and collapses
the drift class to the single `TestRituals_VendoredMatchesUpstream` guard.

Care points:
- Some content tests scope assertions to a named markdown section (CLAUDE.md
  § "Substring assertions are not structural assertions"); preserve that
  section-scoping when repointing at the embedded bytes.
- Keep AC6's structural merge-step check and other non-content assertions.

## References

- **E-0038** / **M-0148** (`TestRituals_VendoredMatchesUpstream`) / **M-0152**.
- CLAUDE.md § "Cross-repo plugin testing" (updated to the embed model).
- The retire-cache-comparison-tests commit that surfaced this.
