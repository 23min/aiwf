---
id: G-0445
title: diff-shape gate hardcodes docs/ exclusion, wrong for some consumer repos
status: open
discovered_in: M-0276
---
`isPlanningPath` in the red/green diff-shape gate (M-0276/AC-6) hardcodes two
excluded prefixes: `work/` and `docs/`. `work/` is correct for every aiwf
consumer — the entity tree and the verb's own frontmatter write live there. But
`docs/` is not an aiwf-managed path: in a consumer repo `docs/` may hold real
shippable implementation (a docs-site, MDX components, a generator). The gate
would silently exclude it — a false-**pass** (a `docs/` implementation change at
`--phase red` would not trip the gate), never a false-refuse.

Options: make the excluded set configurable, scope the exclusion to the entity
tree only (`work/`), or document the assumption explicitly. A kernel check that
ships to consumers should not bake in an aiwf-repo-specific path convention.

Surfaced by the M-0276 wrap design review.
