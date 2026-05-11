---
id: G-0116
title: aiwfx-start-epic creates worktree before promote/authorize on trunk-based repos
status: open
discovered_in: E-0029
---
## What's missing

`aiwfx-start-epic` (rituals-plugin skill at `plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md`) sequences its 10 steps as: preflight (1–4) → **worktree placement (5)** → **branch shape (6)** → delegation prompt (7) → **sovereign promote (8)** → **optional authorize (9)** → hand-off (10). When the operator picks a worktree or a non-`main` branch at steps 5–6, the sovereign `aiwf promote E-NN active` and `aiwf authorize E-NN --to ai/<id>` commits at steps 8–9 land on the new branch, not on trunk.

For trunk-based projects — like the aiwf kernel repo itself, per `CLAUDE.md` "trunk-based development on `main`: commit directly, no PR ceremony" — the consequence is that the kernel-state transitions (epic becomes `active`, scope becomes `active`) are invisible from trunk until the epic-wrap merge. Anyone running `aiwf status` from `main` in another shell sees the epic still as `proposed` and the scope absent, even while the agent is delegated and working on it from the feature branch. The two surfaces disagree about the kernel's truth-of-state.

The skill's own step 6 commentary acknowledges this is unresolved — *"G-0059 frames the open question of which branch-model convention aiwf should bless ... and the answer has not landed yet. Until G-0059 resolves, the skill surfaces the choice rather than presuming."* — but the chosen default ordering still embeds the PR-style assumption that all commits travel with the feature branch, including the metadata commits.

## Why it matters

Kernel state transitions are not implementation work. `aiwf promote E-NN active` and `aiwf authorize E-NN --to ai/<id>` are metadata commits that record *what is now true about the project* — and the project's truth-of-state lives on trunk for repos that work trunk-based. Putting these commits on a feature branch creates a temporal split: the branch's view of the project state is ahead of trunk's, the agent operates within a scope that doesn't exist from another shell's perspective, and `aiwf status` reports different states depending on which working tree the operator is in. Recovery is awkward (rebase the metadata commits onto trunk separately from the work commits, or accept the lag until merge).

For PR-style projects the current ordering is correct — metadata and work land together on the feature branch, the merge commit at wrap exposes the whole sequence atomically to trunk. The friction here is that the skill picks one branch-model convention by default and the choice is invisible to the operator until the commits land in the "wrong" place.

Recommended fix (advisory until G-0059 resolves): reorder steps so promote + authorize fire on the operator's current branch *before* the worktree/branch creation when the operator's policy is trunk-based; or surface the trunk-vs-PR choice as its own Q&A step earlier in the flow and let it gate the order. A lighter alternative: keep the current ordering but document explicitly in step 5/6 that "selecting a non-`main` branch means the promote + authorize commits land on that branch, not on trunk — for trunk-based projects, choose option 1 (stay on `main`) and create the worktree after activation."

Filed as a cross-repo observation: the skill lives in `ai-workflow-rituals`, this gap lives in the kernel's planning tree because that's where the dogfooding friction surfaced. Either the rituals repo absorbs the fix or G-0059's resolution here forces the skill's hand.

Discovered while activating E-0029 in this repo.
