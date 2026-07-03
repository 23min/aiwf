---
id: G-0350
title: Roadmap regen commits but the wrap rituals label it no-gate
status: open
discovered_in: M-0223
---
## Problem

The `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`, and `aiwfx-release` rituals treat the
final roadmap regeneration as ungated housekeeping — the wrap-milestone body says to
run `aiwf render roadmap --write` "once more if the merge introduced state (a local
change, no gate needed)." But `aiwf render roadmap --write` is a **mutating verb that
emits its own commit**, trailered `aiwf-verb: render-roadmap` (its `--help` reads
"write ROADMAP.md and commit"). So a step the ritual labels "no gate needed" silently
produces a commit — the same gate-discipline drift E-0049's context already indicts for
the ungated promote / merge / branch-delete steps.

Surfaced during the M-0223 wrap: the operator, following the ritual, described the
roadmap regen as "no-gate housekeeping" inside the milestone's declared-sequence gate,
then it produced a commit (`aiwf render roadmap`, `aiwf-verb: render-roadmap`). This is
documented behaviour — not a verb defect — but a ritual/verb coherence gap in exactly
the property aiwf sells: honest, per-action gates.

## Root cause

`aiwf render roadmap --write` couples two concerns — it (a) regenerates ROADMAP.md and
(b) commits it as one atomic commit, isolating that commit from pending work via a
`git stash` dance. That dance is also why the verb **refuses to run on a dirty tree**:
mid-wrap, with the implementation staged, it aborted with "git stash can't recreate the
old path over the untracked file; commit or unstage your pending changes, then re-run
the verb." So the regen can only run once the tree is clean, which pins it to a specific
point in the wrap sequence and makes it a commit the ritual must account for.

## Guiding constraint — zero workflow friction

The roadmap must add **no friction** to the workflow: no extra gate, no extra step the
operator has to think about. The fix should position the regeneration optimally so it is
streamlined *away*, not gated in. The milestone drafter picks the mechanism; this gap
does not prescribe it. Directions to weigh:

- **Auto-regenerate** on the mutations that change roadmap state (a post-commit /
  post-mutation hook, the way `STATUS.md` is already regenerated) so it never appears in
  a ritual at all. ROADMAP.md is committed (unlike gitignored `STATUS.md`), so the hook
  would have to fold the update into the triggering commit or amend cleanly.
- **Fold** the ROADMAP.md refresh into an already-gated wrap commit (e.g. the merge or
  promote-done commit) so no separate commit — and thus no separate gate — ever exists.
- **Decouple** the verb: `--write` writes the file only; committing is the caller's
  (already-gated) concern. This also dissolves the stash dance and lets the verb run on
  a dirty tree.

Whichever direction: the operator should never reason about a roadmap gate, and the
ritual bodies (`aiwfx-wrap-milestone` / `aiwfx-wrap-epic` / `aiwfx-release`) must stop
describing a committing step as "no gate needed."

## Scope note

The general declared-sequence gate mechanism lives in foundation epic E-0050 (its
M-0203 was cancelled here and moved there). This gap is the concrete instance — the
roadmap-regen step's ritual framing plus the verb's generate/commit coupling — and is
tracked under E-0049 ("remaining start/wrap ritual fixes").
