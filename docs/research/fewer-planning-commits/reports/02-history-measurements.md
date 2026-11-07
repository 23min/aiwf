# History measurements for amend, fold and batch (2026-11-07)

Saved by the parent from the measuring agent's returned record (the agent could
not write files). Scripts and outputs: this directory (`extract.py`, `base.py`,
`amend.py`, `foldbatch.py`, `refs_extract.py`, `refs_analysis.py`, `cuts.py`,
`syncmerges.py`, `sanity.py`, `explore1.py`, `out/*.txt`, `*.json`).

Window `6b87e0f24..4343f2bd1` (2026-08-26 .. 2026-11-05). Linux devcontainer,
git 2.51.1, Python 3.12.3 (-I), go1.25.12. /workspaces/aiwf untouched.

## Headline

| | amend | fold F1 | fold F2 | batch |
|---|---:|---:|---:|---:|
| (a) as is, of 1,883 planning commits | 727 | 348 | 684–730 | 374 |
| (b) without the 77 ROADMAP-only ritual commits, of 1,806 | 608 | 271 | 607–653 | 297 |

F1 = the review's fold (a fork closes the current group, i.e. fold before every
cut and sync merge). F2 = fold only at push or merge; a fork freezes everything
before it. Batch = commit at every push, merge, fork and line end. Ritual commits
stay one each in all models (F1 with ritual commits foldable: 234).

## Definitions

- Commit classes: kernel (non-merge, non-ritual aiwf-verb) 1,769; ritual 114;
  code 563; merge 157. Verb sets from the code (`check.EnumerateRegisteredVerbs`,
  `skills.RitualTrailerVerbs()` via a scratch test overlay): 47 verbs, rituals
  plan-epic, plan-milestones, wrap-epic, wrap-milestone.
- Every commit `aiwf-actor: human/peter`; no principal; no commit has two verbs.
- 91 commits touch only ROADMAP.md: 77 ritual-stamped, 14 unstamped (counted as
  code).
- Pushed: from the `origin/main` reflog (118 entries, 117 fast-forward
  transitions); entry 2 is a fetch, treated as a boundary. A branch commit counts
  as pushed only once its merge reached origin/main.
- A real case of "pushed but no tracking ref": verb `fce20e31b` ran on top of a
  merge that reached the remote before this clone's tracking ref saw it (HEAD was
  a merge, so no candidate would have differed).
- 4 fast-forward merges onto main (a100c7c64 and three of epic/E-0092, carrying
  5 post-wrap verbs).

## Amend under the corrected rule (`amend.py`)

(a) 727 (613 run starts + 114 ritual; main 174, branches 553); 729 if unknown
cut times block. (b) 608 (571 + 37; main 143, branches 465); 610; 598 if the 14
unstamped ROADMAP-only commits also go. Without the ref check: 673 (a) / 553 (b);
without the SHA check: 686; neither: 629.

What ends a run (primary reason, case a; case b in brackets):
code commit as HEAD 290 (main 18, branches 272) [290]; merge 108 [108]; ritual
commit 63 [13]; push boundary 39 [46]; first commit on a branch cut from the tip
45 [46]; branch cut from HEAD 8 [8]; HEAD merged into another branch 4 [4]; op
names HEAD's SHA 41 [41] (mostly evidence citations, e.g. 0671289fa cites its
parent f63fe1da1); cherry-pick 14 [14]; actor switch 0; window start 1.
Total run starts 613 [571].

## SHA references a fold would break (`refs_analysis.py`)

| record | records | target unpushed when written | F1 breaks | F2 | amend |
|---|---:|---:|---:|---:|---:|
| addressed_by_commit (--by-commit) | 56 | 56 | 4 | 3 | 0 |
| acknowledge illegal (aiwf-force-for) | 14 | 13 | 13 | 13 | 0 |
| aiwf-branch-sha / authorized-by / scope-ends | 0 | – | – | – | – |
| SHAs in commit messages | 18 | 13 | 12 | 11 | 0 |
| SHAs cited in entity bodies under work/ | 469 | 336 | 226 | 216 | 0 |
| SHAs cited in docs | – | 20 | 14 | – | 0 |

No provenance trailers in the window; the last `aiwf authorize` was d21ed6828
(2026-07-21). Amend breaks 0 because its names-HEAD rule makes a new commit
instead (44 verbs). G-0697 (2502ca3dd) already cites 5179769a5, unreachable from
HEAD today. 7 of 8 `.gitleaksignore` fingerprints added on the M-0359 branch name
commits a fold would re-create (re-flag argued, not run).

## Branches cut from an unpushed commit (`cuts.py`)

114 cuts: 78 verified unpushed at cut (44 from branch lines, 34 from main), 35
possibly, 1 pushed. Of the 78, 75 based on a planning commit, 73 verified at the
tip. 38 milestone branches cut from `promote M-NNNN draft -> in_progress` at the
epic tip; 10 epic branches cut from `promote E-NNNN proposed -> active` on main.
Sync merges (`syncmerges.py`): 42 merges pull another line into a branch; in 28
the pulled commit was unpushed. The ref check turned 54 (a) / 55 (b) amends into
new commits (that each would have duplicated an operation is argued from E2).

## Not mentioned elsewhere

16 operations are already recorded twice in the window (14 verb commits
cherry-picked onto main with originals also merged; cab4fe195 and 6685f42bd
reappear on the M-0343 branch) and the pushes passed. Rewrites already in the
window: one amend of a docs commit on main (a3993e720), 12 rebased commits, one
reset that discarded verb 4083464be (redone as 400b77b79). A since-removed test
(`internal/policies/e0092_inventory_test.go`) read four pinned SHAs.

## Not settled

Whether any branch was pushed before its merge (needs the host's push history;
branch yields overstated wherever it happened); cut times of 35 "possible" cuts
and 9 unknown children; who pushed 2eacb2638; whether the 12 rebased verb commits
ran on a different HEAD. F1 merge re-creation count (33) not reconciled with the
review's 106.
