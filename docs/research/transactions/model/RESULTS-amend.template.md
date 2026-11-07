# Amending the unpushed tip: model-checking results

The [transactions model](RESULTS.md) (`model.py`), extended with a third way to get fewer planning commits: a verb **amends HEAD** instead of adding a commit when HEAD is eligible, and otherwise commits per-verb as today. Code commits are never rewritten, and there is no submit step. Everything [RESULTS.md](RESULTS.md) states about the model, its invariants (§2), host settings and conditions (§1) and assumptions (§6) holds here; this document adds the amend designs, the conditions that exercise them, and their results. Breadth-first search, so every trace is a shortest one.

Run <!-- WHEN -->, Python <!-- PYTHON -->, Linux, <!-- CPUS --> CPUs, <!-- JOBS --> workers: `run.py --amend`, which re-runs the original matrix as well. <!-- NRUNS --> distinct explorations are the ones this document adds. <!-- NTRUNC --> runs hit the state cap and <!-- NCYC --> state graphs have a cycle. Peak memory of one worker <!-- PEAK --> MB; wall time <!-- WALL --> s for the whole run.

## Headline

The comparisons below are between an amend design and the per-verb design with the same pointer and audit choice, so that what differs is the amend alone.

1. **In the original conditions amend changes nothing, because it almost never fires there.** A's `edit-body G` is run by `human/bob`, so the only consecutive verbs by one actor in the original scripts are `add adr, promote M in_progress` in `adr`, and amend is harmless there. No original-condition cell of any amend design differs from its per-verb counterpart (§2). Everything amend does is in the amend conditions.
2. **Three of the review's risks close: two by the corrected rule's extra clauses, one by the actor test both rules share.** The first three are failures of the naive rule in cells its per-verb counterpart keeps (§3):
   - **A branch cut from an unpushed commit** (`nr-fork`, T4d). A verb on the new branch amends the commit the branch was cut from, on that branch only; the `--no-ff` merge back lands the operation twice, and nothing reports it. This is the review's E2. The clause "no other local branch contains HEAD" closes it.
   - **An operation naming the SHA it replaces, in a trailer** (`nr-scope`, T6). `authorize M --end` amends the authorize commit, and its `aiwf-scope-ends` names the SHA the amend replaces: the scope never ends, and nothing reports it. The clause "names no SHA of HEAD" closes it, and so do operation-id pointers on their own.
   - **The same, in the bytes a verb writes** (`nr-byc`, T6). `promote G addressed --by-commit` naming the commit it amends leaves `addressed_by_commit` dangling, unreported. Only the SHA clause closes it: `addressed_by_commit` stays a SHA under ADR-0056.
   - **Switching actor under `authorize`** (`actor`). Neither rule amends across an actor switch, so `actor` reads exactly as per-verb. Without the actor test (probe AP1) the agent's verb amends the human's authorize commit, its `aiwf-authorized-by` names the replaced SHA, and the pipeline refuses the transaction for good (L✗); under the safe rule the SHA clause alone prevents that (AW1).
3. **Two failure classes remain under the corrected rule, in cells today's per-verb commits keep.**
   - **A push no remote-tracking ref records** (`untracked`, T4d, under the `merge` host setting and in direct mode). A verb amends a pushed commit; the pushed original survives, merged by the host or already on main, and the next merge or pull lands the operation twice, unreported. This is the hole the review predicted. Under the other non-squash hosts per-verb commits fail T4d in `untracked` too, by the `reuse` route (RESULTS.md), and amend reaches it in fewer steps.
   - **The operator's own rebase of a pushed verb commit** (`sa-after`, `sa-bob`, T4d, under merge-commit, semi-linear and fast-forward hosts). This one is not in the review. `git rebase` gives the pushed commit a new SHA, so no ref contains the copy, and both rules let the next verb amend it. The pushed original survives (the host merges the old head, or another operator builds on it and `git pull --rebase` brings it back), and the operation lands twice. Every clause of the corrected rule asks whether a ref contains HEAD, which is a test by SHA. The fold's W1 failed for this reason (RESULTS.md, headline item 1), and W2 closed it by judging the operation's identity instead.
4. **Amend needs the record audit, as the fold does.** In `twostep` (`promote M in_progress` then `promote M done`, nothing between) amend makes one commit, which the parent-diff audit reads as `draft → done`; the pre-push hook refuses it under both rules and every host setting (L✗). Reading the record fixes it. No other modelled script amends two status steps of one entity.
5. **Against the best fold design** (§2, "The corrected rule against the best fold design"), `amend.safe.opid.record` keeps what today's per-verb commits keep and loses what they lose. It keeps the original `after` and `bob` cells, which the fold fails, because amend does not fire there. It also keeps `nr-fork` under every non-squash host, and `nr-byc` where no host rebase re-creates the named commit, both of which the fold fails: the fold folds the commit the child branch was cut from, and folds away the commit `addressed_by_commit` names. It fails `untracked`, which the fold keeps, and it carries per-verb's T4a and its `reuse` and `cherry` duplicates. Over the amend conditions it fails more cells than either per-verb commits with the same pointers and audit or the best fold design (§2, failing cells).
6. **The new conditions also show three things about designs RESULTS.md covers.**
   - The model now ends a scope. With SHA pointers, today's per-verb commits fail T6 in `scope` under every non-squash host setting and in direct mode: the operator's or the host's rebase re-creates the authorize commit, and the end names nothing (G-0752). Nothing reports it. With operation-id pointers they keep it, except under an all-messages squash, which drops the authorize operation's record. ADR-0056 had argued this without a run.
   - Today's per-verb commits fail T4d in `fork`, under every non-squash host setting and in direct mode: a `git rebase` onto `origin/main` of a branch cut from an unpushed commit copies that commit, and the merge back lands the original and the copy.
   - The best fold design fails T4d in `fork` and `nr-fork`: it folds the commit the child branch was cut from, which the child still carries. The review argued this risk; the run shows it.

## 1. What was added

**Amend designs.** `submit=amend` with three dimensions; a design id is `amend.<rule>.[opid.]<audit>`.

| dimension | values | meaning in the model |
|---|---|---|
| rule | `naive`, `safe` | the eligibility test, below |
| ptr | `leave` (SHA, no `opid.` in the id), `opid` | pointers to aiwf operations written as SHAs, as today, or as the operation's identity, as [ADR-0056](../../../adr/ADR-0056-name-an-authorize-scope-by-a-random-id-not-its-commit-sha.md) writes `aiwf-authorized-by` and `aiwf-scope-ends`. Pointers to commits that are not aiwf operations stay SHAs under both, as under the fold |
| audit | `parent`, `record` | as for the fold: the transition audit compares a commit with its parent, or reads its record's operations in order |

`fold`, `resub` and `txid` do not apply: there is no submit and no planning commit, so no transaction identity and no duplicate-record check (the check needs every operation to carry an identity, and ADR-0056 gives one only to a scope).

**The eligibility test.** A verb amends HEAD when all of these hold, otherwise it commits per-verb:

- *naive:* HEAD is a commit an aiwf verb made (single-operation or already amended; not code, not a hand edit, not a fold's planning commit, not a merge); its git author is the operator running the verb, and every trailer group in it carries the new operation's `aiwf-actor`; and no remote-tracking ref contains it. The remote-tracking refs are `origin/main` as last fetched and `origin/<branch>` as last pushed or fetched; under `untracked` only `origin/main`.
- *safe:* naive, and no other local branch (or another worktree's HEAD; the model has one such ref, in `fork`) contains HEAD, and the new operation names no SHA of HEAD, in a trailer (`aiwf-authorized-by`, `aiwf-scope-ends`, `aiwf-force-for`) or in the bytes it writes (`addressed_by_commit`).

The review's rule also excludes ritual commits made with plain `git commit --trailer aiwf-verb: …` and compares the principal; the model has no ritual commits, and its agent operation's principal is the human operator, so neither clause is exercised.

**How an amended commit records several operations.** As the fold's planning commit does (A12): one message whose last paragraph holds one trailer group per operation, in operation order, each group carrying its operation's identity, verb, entity, actor, `aiwf-to` and pointers. The amend re-creates HEAD on HEAD's parent with HEAD's tree plus the new operation's change, keeps HEAD's subject, and appends the new group after HEAD's groups. Its kind stays `verb`, not the fold's `plan`, so `aiwf check`'s duplicate-record rule (a fold design under `txid=record`) does not apply to it. A rebase, cherry-pick or host rebase keeps the whole group list, as it keeps any message (G9, G14).

**Conditions that exercise amend.** In the original conditions, A's script has only one pair of consecutive verbs by one actor, in `adr`: `edit-body G` is run by `human/bob` (the E4 fixture's actor), so it never amends into, and is never amended by, A's own promotes. The amend conditions therefore change the script:

| condition | what it is |
|---|---|
| `sa-<c>` | condition `<c>` with A's `edit-body G` run by A's own actor, so `edit-body G, promote M done` is a run of two verbs by one actor. Everything else, including the environment moves, is `<c>`'s |
| `fork` | A: `promote M in_progress`; cut a branch from that unpushed commit and check it out; `edit-body G`; `code a1`; merge the branch back into the branch it was cut from with `--no-ff`, as the wrap rituals do. In direct mode the branch cut from is local main, which A may push while the other branch is checked out. This is the review's E2 (the start-milestone ritual's shape) |
| `reuse4` | `reuse` with the stage after A's fourth step: A pushes after `edit-body G`, the host may merge that, then A runs `promote M done` and merges again. As in `reuse`, before its first merge A takes in main only where it must |
| `untracked` | `reuse4`, and A's pushes leave no remote-tracking ref containing what they pushed: in MR mode the clone tracks only `main` (`git clone --single-branch`, the review's E3), in direct mode A pushes by URL, so `origin/main` moves only when A fetches. It stands for every way the review lists of a pushed commit no remote-tracking ref contains (single-branch clone, `git push <url>`, a pruned tracking ref, a lost push reply), applied to every push rather than one |
| `scope` | A: `code a1`, `authorize M`, `authorize M --end` (its `aiwf-scope-ends` names the authorize operation) |
| `actor` | A: `authorize M` (human), the agent's `promote M in_progress` (`aiwf-authorized-by`), the human's `promote M done`, which ends the scope (`aiwf-scope-ends`). Every adjacent pair switches actor |
| `byc` | A: `code a1`, `promote M in_progress`, `promote G addressed --by-commit <the promote's commit>` |
| `twostep` | A: `code a1`, `promote M in_progress`, `promote M done`: two status steps of one entity with nothing between |
| `nr-<c>` | `<c>` where neither operator ever runs `git rebase`; they take in main by merging it (the model's `norebase` flag, which probe P8 also uses). The operator's own rebase already breaks SHA pointers and copies unpushed commits under today's per-verb commits, which hides what amend adds; `nr-` shows amend's part alone |

`scope` and `actor` make the model end a scope, which [RESULTS.md](RESULTS.md) does not: T6 now covers `aiwf-scope-ends`. `aiwf check` has no rule for an `aiwf-scope-ends` that names no opening commit (the readers match it by equality, `internal/scope/scope.go:190`), so such a T6 failure is never reported.

The amend conditions run for the four per-verb designs, the eight amend designs and two fold designs (the best, `replay.append.record.opid.record`, and the default, `replay.replace.record.opid.record`); the original conditions run for every design, the amend designs included.

**Probes.** Two runs on `actor` change the amend test itself, each with a control run:

<!-- PROBES -->

## 2. Matrices

Cells list the invariants that fail, with RESULTS.md's marks: `—` all hold, `L✗` a liveness failure, `!` reported by `aiwf check` on main after the landing, `✗` not reported. Rows are the original conditions, then the amend conditions.

<!-- MATRICES -->

#### Failing cells

Counted as RESULTS.md counts them: one cell per (host setting, condition, invariant), and a safety cell counts as failing also where the run is not live. The two condition sets have different scripts, so their columns are not comparable with each other.

<!-- COUNTS -->

#### Each amend design against per-verb commits with the same pointers and audit

Every (host setting, condition) cell where an amend design differs from the per-verb design that shares its pointer and audit choice (`amend.<rule>.parent` against `perverb.parent`, `amend.<rule>.opid.record` against `perverb.opid.record`, and so on). A cell not listed is the same under both.

<!-- VSPERVERB -->

#### The corrected rule against the best fold design

`amend.safe.opid.record` against `replay.append.record.opid.record`, every cell where they differ.

<!-- VSFOLD -->

#### One choice at a time, around `amend.safe.opid.record`

As in RESULTS.md: each row changes one dimension of the reference design. `worse`: the reference keeps the invariant and the variant does not; `better`: the reverse; `vacuously`: the side that keeps a safety invariant is not live there. The trace is the failing side's shortest violation.

<!-- ONEATATIME -->

#### The `sa-` conditions

<!-- SACHECK -->

## 3. Failure classes amend adds, with their shortest traces

Each class is an invariant failing in a condition under some amend design while the per-verb design with the same pointers and audit keeps it there. The trace is the shortest over every host setting and amend design in the class; only its actions are shown, and `traces/` holds the full file (branches and main after each step).

<!-- CLASSES -->

## 4. Reading the results

**What each clause of the eligibility test buys.** Each row names the clause, the failure that dropping it lets through, and the run that shows it. The control is the run where the clause is in force.

| clause | without it | shown by |
|---|---|---|
| no remote-tracking ref contains HEAD (both rules) | a pushed commit is amended and its operation lands twice | `untracked` against `reuse4`: the same script with the push tracked, where both rules keep T4 under the `merge` host setting and in direct mode |
| HEAD's every group carries the new operation's actor (both rules) | the agent's verb amends the human's `authorize` commit and names it by the replaced SHA; under the naive rule the transaction is stuck | probe AP1, control `actor` |
| no other local branch contains HEAD (safe) | the commit a branch was cut from is amended on that branch only; the merge back lands it twice | `nr-fork`, naive against safe |
| the new operation names no SHA of HEAD, in a trailer (safe) | `aiwf-scope-ends` names the replaced SHA and the scope never ends; with operation-id pointers the clause is not needed for it | `nr-scope`, naive against safe; `amend.naive.opid.*` keeps it |
| the same, in the bytes the verb writes (safe) | `addressed_by_commit` names the replaced SHA; operation-id pointers do not cover it | `nr-byc`, naive against safe |
| HEAD is a non-merge verb commit (both rules) | not exercised as a separate clause; with it, no amend run breaks T5 (no code commit is re-created by an amend) or T1 (no amend drops an operation) | every amend matrix |

**Where per-verb commits fail too, amend fails sooner.** In `fork`, `scope`, `byc` and `untracked` the operator's own rebase, or a host's, already breaks today's per-verb commits, so the cells read the same. The naive rule gets there in fewer steps and with no rebase at all: in `fork` the duplicate lands in 6 or 7 steps against 10 or 12, and in `scope` and `byc` the pointer dangles in 4 or 5 steps against 8 to 10. The safe rule takes the per-verb route in those cells, except in `untracked`, where both rules take amend's shorter route. The `nr-` conditions are those cells with the rebase removed.

**What neither rule closes, and what might.** Both remaining classes are a pushed operation in a commit that no ref names any more: the push went untracked, or the operator's rebase gave it a new SHA. A test of ref containment cannot see either.
- *The rebase route* (`sa-after`, `sa-bob`) would close under a test of what is on the remote rather than which SHA: refuse to amend when HEAD's patch matches a commit on a remote-tracking ref (`git cherry`), or when an operation in HEAD is recorded on the remote branch (the fold's W2, which needs an identity in every operation's record). Neither is modelled; the run that would settle the patch test is a third rule value checking `delta(HEAD)` against the remote-tracking refs' commits, run with `python3 -I run.py --amend --only sa-after,sa-bob,untracked`, a few minutes.
- *The untracked route* has no local remedy that the model shows. A patch test does not help, since no tracking ref holds the pushed commit at all; candidates are fetching before every amend, which helps in direct mode only (the fetch shows the pushed commit on `origin/main`), and refusing to amend on a branch that has no remote-tracking ref. Neither is modelled.

**Where amend reads better than per-verb.** In `sa-cherry`, per-verb commits fail T4d and amend does not. That is not a protection: per-verb's duplicate is B cherry-picking A's lone `edit-body G` commit, and under amend that operation sits in one commit with `promote M done`, which does not apply on B's branch (M is still `draft` there). The cherry-pick that remains, of `promote M in_progress`, does not land twice under either: A's later `promote M done` conflicts with B's copy (argued from the traces, not a separate run).

**Squash hosts.** Apart from `twostep` under the parent audit, amend leaves every squash cell as per-verb commits have it. Under an all-messages squash the last commit's groups survive, and an amended last commit carries more of them, but the earlier commits' groups are still lost.

## 5. Assumptions added, and what the model cannot say about amend

**Assumptions** (RESULTS.md §6's numbering continues):

- **G16.** `git commit --amend`, like the amend modelled here, re-creates HEAD on HEAD's parent: a new SHA, the same parent. The old HEAD stays wherever another ref holds it. This is git's object model, not a measurement here; the review's E2 measured the consequence in a scratch repository (the old commit and the amended one both reached `main` after a merge).
- **G17.** In a `git clone --single-branch` clone, a push of another branch updates no `refs/remotes` ref. Measured by the review's E3 (its clone's fetch refspec covers `refs/heads/main` only, and its pushed `feature` branch has no remote-tracking ref).
- **A16.** The amend as modelled: an aiwf verb amends with the same compare-and-swap ref update a verb commit uses today, so an amend is atomic and has no crash fault of its own; the eligibility test reads the refs as they are locally, without fetching.
- **A17.** A ritual merge commit is untrailered in the model (the rituals' merge commits carry trailers, but both audits skip merge commits, A8 and A9).

**What the model cannot say about amend.**

- **Readers of a multi-operation commit.** Like the fold, the model assumes every reader reads every trailer group in order (A12). The review names readers that do not: `readActorOpenerEntities` (`internal/cli/cliutil/provenance.go`) reads `aiwf-entity` into one field, and `SortedTrailers` (`internal/gitops/trailers.go`) has no production caller, so nothing re-sorts a group's trailers. An amend whose run includes an `authorize` breaks the first; the model cannot see it.
- **Per-operation times.** Commit dates are abstracted away (G12), so the model cannot say that an amended commit keeps one committer date for every operation in it.
- **What a verb printed.** An earlier verb's `metadata.commit_sha` names a commit the next amend replaces. The model has no consumer of verb output.
- **`git reset HEAD~` and the reflog.** Undoing a run, and aiwf's reflog-reading check, are not modelled.
- **How many commits are saved.** The model checks invariants, not counts; the review's counts from history are the measure.
- **Detecting a duplicate.** No modelled `aiwf check` rule reports an operation recorded twice outside a fold's planning commits, so every T4d here is unreported. A rule that reports it needs an identity in every operation's record, which ADR-0056 does not give.
- **Rules the review states that are not exercised:** ritual commits, tags, and the principal clause (§1). `aiwf-branch-sha`, which an `authorize --branch` writes as that branch's tip SHA, is the same kind of pointer as `--by-commit` in `byc` (an operation naming HEAD's SHA) and is argued from it, not modelled.
- **Bounds.** RESULTS.md §8's bounds hold. An amended commit holds at most two operations in these scripts, and `fork` has one branch cut from one unpushed commit.

## 6. How to rerun

From this directory, standard library only:

```
python3 -I run.py --amend --jobs 12   # the original matrix and the amend designs, conditions and probes
python3 -I make_amend.py              # RESULTS-amend.md from RESULTS-amend.template.md, results.json and traces/
python3 -I make_tables.py             # RESULTS.md as before; it reads only the original designs and conditions
```

<!-- STATS -->
