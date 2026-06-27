---
id: G-0293
title: Promote tdd_phase live, not in a burst at milestone wrap
status: open
prior_ids:
    - G-0292
discovered_in: M-0189
---
## Problem

The `tdd_phase` ladder (`red → green → refactor → done`) on a `tdd: required`
milestone's ACs carries signal **only when promoted contemporaneously**. The value
is temporal: `aiwf history M-NNNN/AC-N` is supposed to show `red` (failing test
written) with a timestamp *before* `green` (code makes it pass). That gap in time is
the evidence the test existed and failed first.

When all transitions are stamped in a burst at milestone wrap — all carrying the same
timestamp — there is zero evidence the test came first. The trail is indistinguishable
from "wrote the code, wrote the test after, back-stamped the ladder." It records the
shape of TDD while proving none of the substance. The ladder becomes ceremony.

## Root cause

The `aiwfx-start-milestone` ritual's "defer all commits to wrap" guidance (step 8) is
correct for `tdd: none` (no phases to lose) but, when carried into a `tdd: required`
milestone, collapses the phase promotes into a wrap-time burst. Phase promotes should
fire **live during the `wf-tdd-cycle`**, at the moment each transition actually
happens — `aiwf promote M-NNNN/AC-N --phase green` the instant the AC goes green.

## Deeper issue (honor-system vs. mechanical)

Even *live* phase promotes are not a mechanical guarantee. The kernel's
`acs-tdd-audit` only enforces "`met` requires `tdd_phase: done`" — it never checks that
`red` preceded `green` by a real interval, nor that the test actually failed at `red`.
So the ladder's meaningfulness rests on operator honesty, which bumps against the
load-bearing rule *"framework correctness must not depend on LLM behavior."* The actual
mechanical TDD floor in this repo is the diff-scoped coverage gate (G-0067): every
changed line must be tested or the merge fails. The phase ladder is the soft narrative
layer on top.

## Candidate directions (none committed)

- **Ritual fix** — make `aiwfx-start-milestone` explicit that under `tdd: required`,
  phase promotes fire live during the cycle and must never be bursted at wrap; possibly
  commit code incrementally per AC so the phases track real implementation commits.
- **Kernel nudge** — a `check` finding that warns when an AC's `red`/`green`/`done`
  transitions all land in one commit (or within a tiny timestamp window), making the
  honor-system partly mechanical.
- **Honest demotion** — accept that the ladder is advisory narrative (the coverage gate
  is the hard guarantee) and stop treating the phase timeline as evidence; reserve
  `tdd: required` for milestones where live phase tracking is actually intended.

## Discovered in

M-0189 (`worktree.dir` config knob). The ACs were seeded `red`, the implementation
done with genuine test-first discipline (each test written first and observed red, then
green), but the phase promotes were deferred — so the ladder would have been stamped at
wrap. M-0189 was closed with the ladder stamped plus a Work-log note that the phases are
retroactive (the test suite + 100% diff coverage are the real evidence). This gap
captures the systemic fix so the next `tdd: required` milestone does it right.
