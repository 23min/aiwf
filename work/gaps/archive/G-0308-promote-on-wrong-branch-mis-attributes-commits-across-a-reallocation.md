---
id: G-0308
title: promote-on-wrong-branch mis-attributes commits across a reallocation
status: addressed
addressed_by_commit:
    - ae05f212
---
## What's missing

`promote-on-wrong-branch` — and any history-walking check that keys on a
commit's `aiwf-entity:` trailer — is not reallocation-aware. When the check
walks `git log` and reads `aiwf-entity: X` on a promote commit, it attributes
that commit to whatever entity *currently* holds id `X`, even when `X` was
reallocated to a different id after the commit landed. The check should follow
the reallocation chain (an entity's frontmatter `prior_ids`, or the
`aiwf-prior-entity:` trailer on the reallocate commit) so a pre-reallocation
commit resolves to its *current* entity and is judged against that entity's
parent-epic branch.

## Why it matters

`aiwf reallocate` rewrites the entity file, its frontmatter id, and every
cross-reference — but it cannot rewrite the immutable `aiwf-entity:` trailer in
commits that predate the renumber (history rewrites are forbidden by the
kernel). So every history-walking check that matches on `aiwf-entity:`
mis-attributes pre-reallocation commits. This is not hypothetical: it produces a
standing false positive on a clean tree, which erodes trust in the finding set —
a genuine wrong-branch promote would hide among the false ones. It recurs
structurally with parallel-branch work, which is exactly the situation that
triggers a reallocation (two branches allocate the same id; one renumbers at
merge).

## Evidence (the observed false positive)

`aiwf check` reports `promote-on-wrong-branch` for commit `46dee977`
(`aiwf promote M-0195 draft -> in_progress`): "landed on epic/E-0044…, expected
epic/E-0048." But `46dee977` carries `aiwf-entity: M-0195` for the entity that
was reallocated to M-0208 (`58e3bb19 aiwf reallocate M-0195 -> M-0208`; M-0208's
frontmatter `prior_ids: [M-0195]`, parent E-0044, done, archived). That promote
was correctly on E-0044's branch. The *current* M-0195 is a different entity
(E-0048 skill-body, draft) that took the freed id. The check matched
`46dee977`'s trailer to the current M-0195 (parent E-0048) and flagged the
mismatch — a pure artifact of the renumber, not a real wrong-branch act.

## Fix direction

Resolve a commit's `aiwf-entity:` id to the current entity through the
reallocation chain before judging its branch: build the old-id → current-id map
from `prior_ids` (or the `aiwf-prior-entity:` reallocate trailers), map the
trailer id forward, then compare against the resolved entity's expected branch.
When the trailer id maps to nothing current (reallocated then cancelled, or the
current holder is archived), drop the finding rather than mis-attribute it. Add
a fixture with a reallocated entity whose pre-reallocation promote lives on the
old parent's branch, and assert no finding fires.
