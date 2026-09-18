---
id: G-0694
title: wf-patch step 6 runs its lenses serially and hand-rolls the mutation probe
status: open
---
## What's missing

`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md` step 6
names three review lenses — `wf-review-code`, `wf-rethink` where the change introduced
a design surface, `wf-vacuity` where it touched tested logic — as a sequence of
sentences, saying nothing about whether they may run at once. They share no state:
the same step's reviewer-dispatch contract already forbids the mutation probe from
touching the orchestrator's checkout, so the isolation that concurrent dispatch needs
is mandated there already.

The same step routes the test-sufficiency lens to `wf-vacuity` without carrying that
skill's own first instruction. `wf-vacuity` §"Defer to a real tool first" says to
prefer a project's mutation harness, and a diff-scoped one over a whole-tree one,
before reaching for the manual probe; this repo ships that harness as `make
mutate-diff` (`scripts/mutate-diff.sh`). A reader entering the lens through step 6
meets the four manual mutation shapes first.

Nothing here is measured: both are claims about what the skill text instructs, and
that file is what a reader should look at.

## Why it matters

Step 6 is the step a patch's author waits on. Lenses dispatched one after another
cost the sum of their wall-clock rather than the longest of them, and the manual
probe re-derives per patch what a shipped command already does.

That cost lands on the busiest surface in the repo: 25 `patch/` branches merged to
`main` in the three months to 2026-09-17, eleven of them changing fewer than 50 lines.
