---
id: G-0647
title: Wrap reconstructs gap-closure claims from prose; nothing records them at start
status: addressed
priority: high
addressed_by_commit:
    - 7ed5bca1b
---
## What's missing

Nothing records, when the work starts, which gaps it intends to close. The claim is reconstructed at wrap instead, out of free prose.

The asymmetry is visible inside one ritual. `aiwfx-start-milestone/SKILL.md` routes a *newly discovered* gap to the spec's `## Deferrals` section, and `templates/milestone-spec.md` carries that heading among its fourteen. Neither has a counterpart for a gap the milestone sets out to close. `aiwfx-wrap-epic/SKILL.md` precondition 6 then asks its reader to establish that "neither the epic's own spec nor any milestone's left a gap open that it explicitly claims to fix" — an open-ended search across every child spec's body, run after the work is finished.

Measured: of 183 open gaps, 0 carry `addressed_by` and 87 carry `discovered_in`. `discovered_in` records where a gap was found, not what fixes it, and `Tree.ReverseRefs` (`internal/tree/tree.go`) is built from frontmatter `ForwardRefs` rather than from prose. So no edge in the tree answers "which milestone claims to close this gap", and none can be derived from what is stored.

Measured: `aiwf show M-0269 --format json` returns thirteen body sections keyed by heading slug. Prose outside a named heading is not addressable that way, so the claim in its current form cannot be read by any command — only by a person re-reading the spec.

## Why it matters

The claim is made at the point of least context, and lands in the one shape nothing can query. Confirming that a `done` milestone closed what it said it would means re-reading every spec by hand, which is why the same failure recurred across dozens of epics before G-0431 was filed — and why G-0431's fix, which closes the gaps at wrap, still rests on that hand-search for its input.
