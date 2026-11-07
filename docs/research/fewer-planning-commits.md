# Fewer planning commits: amend, fold and batch, measured

> **Status:** thesis-draft
> **Date:** 2026-11-07
> **Question:** Which mechanism gives future git history far fewer aiwf planning commits, for every reader, at the least cost and risk: folding planning commits before a push, batching operations so verbs do not commit, or amending the operator's own unpushed planning commit?
> **Working answer:** Amend, on a foundation every write-side option needs: a scope id that survives a rewrite ([ADR-0056](../adr/ADR-0056-name-an-authorize-scope-by-a-random-id-not-its-commit-sha.md)), one reader for commits that carry several operations, and an identity per operation with a check that refuses one recorded twice. Pair it with ritual changes that keep planning steps together. Batch fails against the kernel's own guards; the fold removes the most commits only when it also runs before every branch cut, and breaks the commit references this repository relies on.
> **Audience:** the maintainer deciding the next kernel change; whoever implements it.
> **Premise:** builds on [collaborative-state-and-commit-granularity.md](collaborative-state-and-commit-granularity.md) (the fold, §4.2) and its [transactions model](transactions/model/RESULTS.md). The evidence behind every number here is in [fewer-planning-commits/](fewer-planning-commits/README.md).
> **Tags:** #aiwf #git #history #commits #amend #fold #provenance

---

## Abstract

Every aiwf verb makes one commit (kernel commitment 7), so planning dominates history. Over ten weeks of this repository (2026-08-26 to 2026-11-05), 1,883 of 2,603 commits on `main` are planning commits; 1,806 if the rituals stop committing `ROADMAP.md`. The window was replayed under each mechanism. Amend was also prototyped and model-checked, the fold was simulated in scratch repositories and model-checked, and batch was tested against the kernel's guards and git's own refusals.

- **Fold before push** leaves 271 planning commits, but only if it runs before every branch cut and every merge of another line into a branch; run only at push or merge it leaves 607–653, level with amend. It re-creates the code commits after the first planning commit it folds, which breaks references written into the repository before the push: in the window, 13 of 14 acknowledgments, 4 gap closures, 12 SHAs in commit messages and 226 SHA citations in entity bodies. A pre-push hook cannot do it, so it is a step a user can skip.
- **Batch, with verbs not committing,** is refused by the kernel's own guards on the second verb that touches a file, needs a second record of pending state, and must commit at every merge and branch cut, which leaves it no better than the fold.
- **Amend the unpushed planning tip** never rewrites a commit made outside a verb, except one hand-made with a verb trailer, and works without anyone remembering a step. It leaves 608 planning commits, 528 with two ritual changes, and 455 at best with a third. Its remaining failures end in an operation recorded twice that no check detects, or in a rejected push. A duplicate check closes the first kind; the same idea applied by patch-id finds 16 operations this repository already records twice.

A *run* below is a sequence of verb commits with nothing else between them; amend turns each run into one commit.

## 1. The problem, measured

The window is `6b87e0f24..4343f2bd1`: the oldest entry of this clone's `origin/main` reflog to HEAD.

| | commits | planning commits |
|---|---:|---:|
| Everything reachable on `main` (what GitHub's commit list shows) | 2,603 | 1,883 |
| `main`'s first-parent line | 591 | 484 |
| Brought in by merges from 115 branches | 2,012 | 1,399 |

Planning commits are non-merge commits carrying `aiwf-verb`. 114 of them are made by the rituals with plain `git commit --trailer`, and 77 of those touch only `ROADMAP.md`. Nothing on a critical path reads `ROADMAP.md`; the counts marked "without ROADMAP" assume the rituals stop committing it.

## 2. Candidates

- **Fold.** Verbs commit as today. A step before pushing, or before merging a branch, rewrites the operator's unpushed commits so their planning commits become one commit at the tip, carrying one trailer group per operation. Code commits after the first folded planning commit are re-created. This is the earlier research's `submit`; a "pre-push compacter" is the same mechanism.
- **Batch.** Verbs write their files but do not commit; each operation's record is kept until one commit lands for the batch.
- **Amend.** A verb amends HEAD instead of adding a commit when HEAD is the same operator's planning commit and nothing else holds it; otherwise it commits as today.
- **Side ref** (a fold variant). Verbs commit to a private ref while the branch receives only code commits; one planning commit lands at the tip at submit.
- **`aiwf batch`** (one call that runs several operations and makes one commit). Its batches are a subset of amend's runs, so it removes no more than amend does; it composes with amend rather than replacing it.

Git cannot remove commits from the middle of a line without re-creating every later commit. A step before the push can therefore compact all planning into one commit (the fold), compact each run in place (re-creating every commit after the first run, for amend's yield: worse than either), or compact only the run at the tip (no re-creation, but only the last run). Amend is the third done continuously: each run is compacted while it is still the tip, before a code commit seals it.

## 3. Method

Each piece of work is recorded in [fewer-planning-commits/reports/](fewer-planning-commits/reports/) with its commands and observations:

1. An independent review of the candidates against the code; its corrected amend rule is the one the later work uses.
2. A replay of the window under each mechanism's rules (`history/`).
3. An amend prototype behind an environment variable, with ten real-git scenarios (`amend-prototype/`).
4. Fold and side-ref experiments in scratch repositories (`fold-cost/`).
5. The transactions model extended with eight amend designs and run over its full matrix ([RESULTS-amend.md](transactions/model/RESULTS-amend.md)); its existing results were re-derived unchanged.

All ran on 2026-11-07 with git 2.51.1, Go 1.25.12 and Python 3.12.3 in a Linux devcontainer, against aiwf built from `4343f2bd1`.

## 4. Results

### 4.1 Planning commits left

| | as history is (of 1,883) | without ROADMAP (of 1,806) |
|---|---:|---:|
| Fold, before every branch cut and every merge of another line | 348 | 271 |
| Fold, also folding the rituals' commits | 234 | 234 |
| Fold, only at push or merge | 684–730 | 607–653 |
| Batch | 374 | 297 |
| Amend | 727 | 608 |
| Amend, evidence cites the code commit rather than the planning tip | — | 567 |
| Amend, rituals commit through verbs | — | 574 |
| Amend, both | — | 528 |
| Amend, both, with spec edits that follow a code commit moved to the next run | — | 455 at best |

A fold that runs only at push or merge cannot fold past a commit a branch was cut from without landing operations twice (4.3), so every fork freezes what precedes it; the range depends on whether a fork counts only once a child commit shows it existed. The last amend row moves every movable run and so is the fewest commits that change can leave: 73 of the 92 runs consisting only of `edit-body` after a code commit could join the next run on their branch.

### 4.2 Batch

- The second verb on a file with a pending write is refused by `checkUncommittedConflict` (`internal/verb/apply.go`), and promotes and retitles by `guardClaimPaths` (`internal/verb/claimguard.go`): both compare the disk with HEAD. Lifting that means comparing with HEAD plus the pending records, a second record of what each path should hold.
- A merge that touches a pending file is refused, `rebase` and `pull --rebase` are refused outright, and a new worktree does not see pending state (measured in the side-ref experiment, which leaves files in the same state). So a batch has to commit before a merge that touches its files, before a rebase, and before a branch is cut into another worktree; the replay assumes it commits at every merge and branch cut.
- Active scopes are read from `git log` (`internal/cli/cliutil/provenance.go`), so an `authorize` would have to commit at once.
- Where the pending records would live is unsolved (argued, not prototyped): under `.git/`, `stash`, `checkout`, `reset` and `clean` change the files without the records; in the tree, they are the event log CLAUDE.md excludes; and a file and its record are not written atomically.
- In the window, 437 code commits (420 on branches, 17 on `main`) were made while a verb's write would have been pending, each a chance for the sweep the earlier research measured (§8, E3): `git add -A` committed pending planning files with a code commit, and the hooks accepted it.

### 4.3 Fold

Measured in scratch repositories and over the window:

- **References.** A fold re-creates commits that records already name. Over the window it would have broken records verbs write (13 of 14 `aiwf acknowledge illegal` records, 4 of 56 gap closures by `--by-commit`) and SHAs the operator writes (12 of 18 in commit messages, 226 citations in entity bodies). 7 of the 8 `.gitleaksignore` fingerprints added on one milestone branch name a commit a fold would re-create; whether the pre-push scan then flags them again was not run. In the experiment, `aiwf show` and a routine hooked push reported none of the broken references; `aiwf check --since` reported one class. A fold that also rewrites SHAs inside the branch made those references resolve again, but cannot reach short SHAs, tags, notes or other clones.
- **Forks.** 78 of the window's 114 branch cuts were made from a commit not yet pushed, 73 of them from a planning commit at the tip; 46 of those are the epic and milestone start promotes. A branch, worktree or stash made before a fold and merged after it landed operations and code commits twice, and nothing reported it.
- **Signing.** `commit-tree` ignores `commit.gpgsign`: a fold that passes `-S` kept every re-created commit signed, and one that does not left all seven unsigned, with no warning (`fold-cost/out/signing.txt`).
- **Intermediate states.** Every re-created commit holds a tree that never existed, without the planning changes moved to the tip, and gave a different `aiwf check` result from its original.
- **A pre-push hook cannot fold.** A hook that rewrote the branch changed the local branch while the push sent the original commits (`fold-cost/out/prepush-cannot-fold.txt`). The fold is a step a user runs, and a plain `git push` skips it.
- **The side ref** keeps code commits intact on its clean path, but brings back batch's hazards almost one for one: `git add -A` swept pending planning into a code commit, merges touching a pending file were refused, a new worktree did not see pending state, aiwf's guards refused a second verb, and one verb committed a pending entity straight onto the branch.

### 4.4 Amend

The rule that holds: a verb amends HEAD only when HEAD is a non-merge commit whose operations are all verb operations (not a ritual's plain commit) by the same actor and principal; no other branch, tag, stash, remote-tracking ref or worktree HEAD contains it; and nothing the new operation writes names HEAD's SHA. The rule first considered, "not on any remote-tracking ref", lands an operation twice when a branch is cut from an unpushed planning commit, which the start-milestone ritual does by design: it promotes on the epic branch and then cuts the milestone branch from that commit.

- **What holds.** No amend in the model or the prototype dropped an operation, and none rewrote a commit made without aiwf trailers. Under the rule above, the branch-cut and start-milestone shapes produced no duplicate, and an operation naming the tip's SHA got a new commit. The repo lock serialized concurrent verbs, and the compare-and-swap on `update-ref` refused a `git commit` made while a verb was amending. A crash mid-amend left the same state as one on today's commit path.
- **What remains.** Each of these ends in an operation recorded twice that `aiwf check` and the pre-push hook accept: a push that leaves no remote-tracking ref (a push to a URL, a lost push reply, a deleted tracking ref) followed by a merging pull; the operator's own rebase of a pushed verb commit, which gives it a SHA no ref contains (found by the model); a clone made from the working repository; a branch created while a verb amends. Every clause of the rule tests ref containment, which is a test by SHA. Two outcomes are not duplicates: a push made while a verb amends lands the pre-amend commit and the next push is rejected, and a pull with rebase after a lost push reply left a commit whose message claims an operation its diff lacks.
- **Costs that stay.** `git reset HEAD~` undoes a run of verbs, not one. An earlier verb's printed `commit_sha` becomes unreachable. A hand-made commit carrying a kernel verb trailer and code was amended by the next verb. The containment test took about 0.5 s per verb in this repository.
- **Against the model's invariants.** The model checks each design against each (condition, invariant) pair, a *cell*. Over the conditions added for amend, `amend.safe.opid.record` fails 445 cells: fewer than today's per-verb commits with SHA pointers (530), more than per-verb commits with operation-id pointers (437) or the best fold (435). It keeps cells the fold fails (forks, a pointer naming its own commit) and fails one the fold keeps (the untracked push).
- **Today's readers misread an amended commit:** an `authorize` amended into a promote leaves the agent refused; a pause and resume collapse to one event; `promote active` then `done` in one commit is read as an illegal jump and the push refused; an unrelated `aiwf-force` in the same commit then silences that error; the TDD ladder collapses to one history event; `STATUS.md` drops an entity. A reader that splits a commit into its operations would fix most of these (argued; none was built): the TDD ladder only partly, since `git log` still shows one commit time, and the forced case only if `aiwf-force` applies to one operation rather than the commit. The fold needs the same reader.

### 4.5 What splits amend's runs

Without ROADMAP, 571 runs start (`history/out/ordering_analysis.txt`): 290 because HEAD is a code commit, 108 after a merge, 71 at a branch cut, 40 because the operation names HEAD's SHA, 34 at a push, 13 after a ritual commit, 12 at a cherry-pick and 3 for other reasons, counting each run by the first reason the replay records.

- Of the 290, 108 runs are only `edit-body`, mostly on milestone specs; wrap-milestone asks for the spec's evidence before review and again after each corrective commit. 72 contain a TDD phase step, which the work orders: red comes before the code and green after it.
- After a merge, 48 runs are gap closures made right after the merge that fixed the gap (28 after a patch merge, 13 after a milestone wrap, 7 other), and 27 open with the merged milestone's promote to done, 14 of which also start the next milestone.
- The branch cuts follow the start ritual's rule that sovereign acts land on the epic branch before the milestone branch is cut.
- The 40 self-citations are mostly evidence that names the planning tip a check ran at.

### 4.6 What history looks like

The last three pushes to `main` and milestone M-0349's branch, 57 commits, become 17 under amend and 13 under the fold (`history/out/render.txt`). `plan` is a verb commit, `code` any other commit, `rit` a ritual's plain commit, `*` a commit the fold re-creates with a new SHA.

On `main` the two differ only in the last push, where the fold moves the code commit before the planning commit:

```
amend                                         fold
plan  aiwf: 13 operations (set-priority …)    plan  aiwf: 13 operations (set-priority …)
merge docs(research): land …                  merge docs(research): land …
plan  aiwf edit-body ADR-0056                 plan  aiwf edit-body ADR-0056
plan  aiwf: 18 operations (edit-body …)      *code  docs(initiatives): drop the claim …
code  docs(initiatives): drop the claim …     plan  aiwf: 18 operations (edit-body …)
```

On the branch, amend keeps a planning commit between code commits; the fold moves all planning to the end and re-creates every commit after the first planning commit:

```
amend                                         fold
code  chore(guidance): freeze …               code  chore(guidance): freeze …
plan  aiwf: 3 operations (edit-body …)       *code  test(policies): hold M-0349's inventory …
code  test(policies): hold M-0349's …        *code  test(policies): derive each host's …
code  test(policies): derive each host's …   *code  test(policies): require direct cites …
plan  aiwf: 8 operations (edit-body, AC met…)*code  test(policies): pin the inventory test's …
code  test(policies): require direct cites … *code  test(policies): pin the empty-home rule …
plan  aiwf: 2 operations (edit-body …)       *rit   docs(plan): record M-0349's review …
code  test(policies): pin the inventory …     plan  aiwf: 16 operations (edit-body …)
plan  aiwf: 2 operations (edit-body …)
code  test(policies): pin the empty-home …
plan  aiwf edit-body M-0335
rit   docs(plan): record M-0349's review …
```

## 5. Conclusion

Amend meets the criteria the mechanism was chosen by. It keeps no state beyond git, derives every decision from git, rewrites no commit a verb did not make (one hand-made with a verb trailer aside), and needs nothing remembered: a user or an LLM that only runs verbs gets it. It leaves more planning commits than the best fold (528, or 455 at best, against 234 with the same ritual changes), and the difference is the planning commits left between code commits on branches: spec edits after a code commit and TDD steps around the implementation. The fold reaches its number only by also running before every branch cut and every merge of another line; run at push alone it is level with amend. It stays possible later on the same foundation, for hosts that squash.

What amend needs before it ships, and any write-side mechanism needs:

1. **A scope id that survives a rewrite.** [ADR-0056](../adr/ADR-0056-name-an-authorize-scope-by-a-random-id-not-its-commit-sha.md) names each scope by a random id carried by its opening commit, so amending or folding that commit cannot strand the scope. The model also shows today's per-verb commits failing with SHA pointers when a scope ends (G-0752).
2. **One reader for commits that carry several operations,** replacing the readers that each take one operation per commit.
3. **An identity per operation, and a check that refuses one recorded twice.** This is the alternative ADR-0056 rejected ("Give every verb's commit an identity"), on the ground that "with one commit per verb nothing but the scope readers would read such an identity". Once a commit carries several operations, the duplicate check reads it, so a decision for amend revisits that alternative. Neither the check nor the identity was built or modelled here. It would close amend's duplicate cases and the fold's fork duplicates; the 16 operations already recorded twice in this repository carry no identity, and a patch-id comparison finds them (`history/out/ordering_analysis.txt`). Every push that carried them passed.

And ritual changes: rituals commit through verbs rather than plain `git commit --trailer`; evidence cites the code commit that was checked rather than the planning tip; `ROADMAP.md` is rendered on demand rather than committed by rituals. A fourth, moving spec edits that follow a code commit into the next run, saves at most 73 more commits, at the cost of a spec that lags the code between runs, which wrap-milestone's review currently relies on not happening.

## 6. Not settled

- Whether a patch-id test (`git cherry` against remote-tracking refs) closes the rebase hole. The model needs a third eligibility rule written first, then `run.py --amend --only sa-after,sa-bob,untracked`, a few minutes.
- What closes the untracked-push hole other than the duplicate check.
- Whether any branch in the window was pushed before its merge; that needs the host's push history. Where it happened, the branch yields above are overstated for every mechanism.
- The containment test's cost with a fresh commit-graph (`git commit-graph write --reachable`, then time it again).
- How `aiwf render --format=html` shows a multi-operation commit; signing, `post-rewrite` and notes under amend; Windows.

## 7. Found along the way

- 16 operations are already recorded twice in this repository's history, among them the two `aiwf add decision D-0083` commits, and nothing reported them.
- `internal/cli/cliutil/provenance.go:103` says trailers are re-sorted on emit by `gitops.SortedTrailers`, which has no production caller.
- `readActorOpenerEntities` (`internal/cli/cliutil/provenance.go`) reads `aiwf-entity` into one field, so it misreads any commit with several operations.
- The `aiwf-force` exemption in `fsm-history-consistent` applies to a whole commit, not to one operation in it.
- G-0697's body cites `5179769a5`, which HEAD does not reach.

## Provenance

Measured 2026-11-07 on branch `patch/research-commit-noise` from `main` at `4343f2bd1`. The scripts, their outputs and the reports are in [fewer-planning-commits/](fewer-planning-commits/README.md); the model's extension and results are in [transactions/model/](transactions/model/RESULTS-amend.md). The amend prototype is kept as a patch, [amend-experiment.patch](fewer-planning-commits/amend-prototype/amend-experiment.patch), not as code. A second independent review checked a draft's numbers and attributions against those sources; the numbers here are re-derived from the scripts in `history/` after its findings.
