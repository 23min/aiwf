---
id: G-0219
title: 'aiwfx-wrap-milestone SKILL.md asymmetric: missing wrap-milestone trailer step'
status: open
discovered_in: M-0160
---

## What's missing

The `aiwfx-wrap-epic` SKILL.md prescribes a multi-step merge ritual at the equivalent of step 5: stage with `git merge --no-ff --no-commit`, then explicitly emit the commit with three required trailers (`aiwf-verb: wrap-epic`, `aiwf-entity: E-NNNN`, `aiwf-actor: human/<id>`). The corresponding `aiwfx-wrap-milestone` SKILL.md step 11 ("After merge") originally was a single bullet:

> "If the project uses an epic-integration branch, merge the milestone branch into the epic branch (`--no-ff` to preserve the milestone shape)."

No trailer prescription. No `--no-commit` step. No symmetry with `aiwfx-wrap-epic`'s pattern.

The kernel recognizes both `wrap-epic` and `wrap-milestone` as ritualVerbs (sourced from the embedded ritual snapshot per G-0190), so `aiwf-verb: wrap-milestone` IS a legitimate trailer value for the milestone-to-epic merge commit — the skill just never said so. The asymmetry left a hole: an operator (human or LLM) following the milestone-wrap skill had no prescription, so they either skipped the trailer entirely (untrailered merge commit; the kernel's `provenance-untrailered-entity-commit` rule may fire) or hand-derived something (and in M-0160's case, hand-derived `aiwf-verb: merge` — a category-confused fabrication, neither a Cobra verb nor an allowlisted ritual value).

## Why it matters

The asymmetry isn't theoretical — it produced the bug. During M-0160 wrap, the operator (me) drafted `git merge --no-ff` commit messages for both M-0159 (yesterday, commit `e1dc6dc6`) and M-0160 (today, commit `734dca4b`) that carried `aiwf-verb: merge`. The kernel's `trailer-verb-unknown` rule flagged both at pre-push (warning severity); the post-hoc fix path was `aiwf acknowledge-illegal` for each SHA (commits `11713db4`, `e8f1e9a1`).

**Repeated operator error within one wrap cycle is the canonical signal that a chokepoint is missing**, per CLAUDE.md's standard hint ("if you see this happen more than once"). The chokepoint here is the skill prescription itself — operators follow skill instructions; if the skill doesn't prescribe the right trailer, operators do whatever feels reasonable, and "feels reasonable" is unbounded.

Per CLAUDE.md "Kernel functionality must be AI-discoverable": every operational surface must be reachable through channels an AI assistant routinely consults. The aiwfx-wrap-milestone skill is one such surface. Missing the trailer-prescription step there is exactly the AI-discoverability hole the principle forbids.

## Proposed fix shape

Update `aiwfx-wrap-milestone` SKILL.md step 11 to mirror `aiwfx-wrap-epic`'s shape:

```bash
git checkout epic/E-NNNN-<slug>
git merge --no-ff --no-commit milestone/M-NNNN-<slug>
git commit -m "chore(milestone): wrap M-NNNN — <milestone title>" \
  --trailer "aiwf-verb: wrap-milestone" \
  --trailer "aiwf-entity: M-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

Plus an anti-pattern note explaining why `aiwf-verb: merge` is wrong (and pointing at G-0218's chokepoint proposal).

## Status

**Addressed at commit `5cf007f5`.** The SKILL.md edit landed during the same wrap cycle that surfaced this gap. The fix follows the aiwfx-wrap-epic shape literally.

**Companion concern**: this gap was filed AFTER the fix landed (out-of-discipline). The fix should have followed the gap, not preceded it; the AC pin asserting the new content is structurally present (per `internal/policies/`) is tracked under [G-0220](G-0220-ritual-skill-md-edits-without-structural-ac-pins-no-mechanical-backstop.md).

## Test surface

A structural test under `internal/policies/aiwfx_wrap_milestone_test.go` (the parallel of `aiwfx_wrap_epic_test.go`) asserts:
- The merge-step section exists (located by heading-hierarchy walk, not flat substring)
- The merge-step section contains the staged-merge step (`git merge --no-ff --no-commit`)
- The merge-step section contains all three required trailer flags (`--trailer "aiwf-verb: wrap-milestone"`, `--trailer "aiwf-entity: M-NNNN"`, `--trailer "aiwf-actor: human/`)
- The staged-merge appears BEFORE the trailered commit (ordering check)

Mirrors `TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck` line-for-line, just scoped to the wrap-milestone skill.

## Closing this gap

When the structural test lands and a milestone's AC pins it:
- Promote G-0219 to `addressed` with `--by M-NNNN` (the milestone whose AC owns the test).
- The `addressed_by_commit: [5cf007f5]` link records the skill-edit commit.

## Discovered in

M-0160 — surfaced during epic-merge prep when the post-push `aiwf check` flagged `trailer-verb-unknown` warnings on both the M-0159 and M-0160 merge commits, both fabricating `aiwf-verb: merge`. The diagnosis pulled apart "merge isn't an aiwf concept" (correct) from "wrap-milestone IS a recognized ritual value the skill should prescribe" (the missed-prescription that produced the fabrication). The skill asymmetry IS the operator-discoverability gap.
