# Amending the unpushed tip: model-checking results

The [transactions model](RESULTS.md) (`model.py`), extended with a third way to get fewer planning commits: a verb **amends HEAD** instead of adding a commit when HEAD is eligible, and otherwise commits per-verb as today. Code commits are never rewritten, and there is no submit step. Everything [RESULTS.md](RESULTS.md) states about the model, its invariants (§2), host settings and conditions (§1) and assumptions (§6) holds here; this document adds the amend designs, the conditions that exercise them, and their results. Breadth-first search, so every trace is a shortest one.

Run 2026-11-08 12:05 UTC, Python 3.12.3, Linux, 16 CPUs, 12 workers: `run.py --amend`, which re-runs the original matrix as well. 2434 distinct explorations are the ones this document adds. 0 runs hit the state cap and 0 state graphs have a cycle. Peak memory of one worker 214 MB; wall time 709.7 s for the whole run.

## Headline

The comparisons below are between an amend design and the per-verb design with the same pointer and audit choice, so that what differs is the amend alone.

1. **In the original conditions amend changes nothing, because it almost never fires there.** A's `edit-body G` is run by `human/bob`, so the only consecutive verbs by one actor in the original scripts are `add adr, promote M in_progress` in `adr`, and amend is harmless there. No original-condition cell of any amend design differs from its per-verb counterpart (§2). Everything amend does is in the amend conditions.
2. **Three of the review's risks close: two by the corrected rule's extra clauses, one by the actor test every rule shares.** The first three are failures of the naive rule in cells its per-verb counterpart keeps (§3):
   - **A branch cut from an unpushed commit** (`nr-fork`, T4d). A verb on the new branch amends the commit the branch was cut from, on that branch only; the `--no-ff` merge back lands the operation twice, and nothing reports it. This is the review's E2. The clause "no other local branch contains HEAD" closes it.
   - **An operation naming the SHA it replaces, in a trailer** (`nr-scope`, T6). `authorize M --end` amends the authorize commit, and its `aiwf-scope-ends` names the SHA the amend replaces: the scope never ends, and nothing reports it. The clause "names no SHA of HEAD" closes it, and so do operation-id pointers on their own.
   - **The same, in the bytes a verb writes** (`nr-byc`, T6). `promote G addressed --by-commit` naming the commit it amends leaves `addressed_by_commit` dangling, unreported. Only the SHA clause closes it: `addressed_by_commit` stays a SHA under ADR-0056.
   - **Switching actor under `authorize`** (`actor`). No rule amends across an actor switch, so `actor` reads exactly as per-verb. Without the actor test (probe AP1) the agent's verb amends the human's authorize commit, its `aiwf-authorized-by` names the replaced SHA, and the pipeline refuses the transaction for good (L✗); under the safe rule the SHA clause alone prevents that (AW1).
3. **One failure class remains under the reflog rule, in cells today's per-verb commits keep. The corrected rule leaves a second, which the reflog clause closes.**
   - **A push no remote-tracking ref records** (`untracked`, T4d, under the `merge` host setting and in direct mode). A verb amends a pushed commit; the pushed original survives, merged by the host or already on main, and the next merge or pull lands the operation twice, unreported. This is the hole the review predicted. Under the other non-squash hosts per-verb commits fail T4d in `untracked` too, by the `reuse` route (RESULTS.md), and amend reaches it in fewer steps.
   - **The operator's own rebase of a pushed verb commit** (`sa-after`, `sa-bob`, T4d, under merge-commit, semi-linear and fast-forward hosts), under the naive and corrected rules. This one is not in the review. `git rebase` gives the pushed commit a new SHA, so no ref contains the copy, and both rules let the next verb amend it. The pushed original survives (the host merges the old head, or another operator builds on it and `git pull --rebase` brings it back), and the operation lands twice. Every clause of the corrected rule asks whether a ref contains HEAD, which is a test by SHA. The fold's W1 failed for this reason (RESULTS.md, headline item 1), and W2 closed it by judging the operation's identity instead. The reflog rule closes it by asking what last moved HEAD: the rebase did, so the next verb commits anew. No reflog design fails these cells, and the reflog clause makes no cell worse (§2, one choice at a time).
4. **Amend needs the record audit, as the fold does.** In `twostep` (`promote M in_progress` then `promote M done`, nothing between) amend makes one commit, which the parent-diff audit reads as `draft → done`; the pre-push hook refuses it under every rule and host setting (L✗). Reading the record fixes it. No other modelled script amends two status steps of one entity.
5. **Against the best fold design** (§2, "The reflog rule against the best fold design"), `amend.reflog.opid.record` keeps what today's per-verb commits keep and loses what they lose. It keeps the `after` and `bob` cells, original and `sa-`, which the fold fails: in the original ones amend does not fire, and in the `sa-` ones the reflog clause stops it. It also keeps `nr-fork` under every non-squash host, and `nr-byc` where no host rebase re-creates the named commit, both of which the fold fails: the fold folds the commit the child branch was cut from, and folds away the commit `addressed_by_commit` names. It fails `untracked`, which the fold keeps, and it carries per-verb's T4a and its `reuse` and `cherry` duplicates. Over the amend conditions it fails fewer cells than either per-verb commits with the same pointers and audit or the best fold design (§2, failing cells); the corrected rule fails more than both.
6. **The new conditions also show three things about designs RESULTS.md covers.**
   - The model now ends a scope. With SHA pointers, today's per-verb commits fail T6 in `scope` under every non-squash host setting and in direct mode: the operator's or the host's rebase re-creates the authorize commit, and the end names nothing (G-0752). Nothing reports it. With operation-id pointers they keep it, except under an all-messages squash, which drops the authorize operation's record. ADR-0056 had argued this without a run.
   - Today's per-verb commits fail T4d in `fork`, under every non-squash host setting and in direct mode: a `git rebase` onto `origin/main` of a branch cut from an unpushed commit copies that commit, and the merge back lands the original and the copy.
   - The best fold design fails T4d in `fork` and `nr-fork`: it folds the commit the child branch was cut from, which the child still carries. The review argued this risk; the run shows it.

## 1. What was added

**Amend designs.** `submit=amend` with three dimensions; a design id is `amend.<rule>.[opid.]<audit>`.

| dimension | values | meaning in the model |
|---|---|---|
| rule | `naive`, `safe`, `reflog` | the eligibility test, below |
| ptr | `leave` (SHA, no `opid.` in the id), `opid` | pointers to aiwf operations written as SHAs, as today, or as the operation's identity, as [ADR-0056](../../../adr/ADR-0056-name-an-authorize-scope-by-a-random-id-not-its-commit-sha.md) writes `aiwf-authorized-by` and `aiwf-scope-ends`. Pointers to commits that are not aiwf operations stay SHAs under both, as under the fold |
| audit | `parent`, `record` | as for the fold: the transition audit compares a commit with its parent, or reads its record's operations in order |

`fold`, `resub` and `txid` do not apply: there is no submit and no planning commit, so no transaction identity and no duplicate-record check (the check needs every operation to carry an identity, and ADR-0056 gives one only to a scope).

**The eligibility test.** A verb amends HEAD when all of these hold, otherwise it commits per-verb:

- *naive:* HEAD is a commit an aiwf verb made (single-operation or already amended; not code, not a hand edit, not a fold's planning commit, not a merge); its git author is the operator running the verb, and every trailer group in it carries the new operation's `aiwf-actor`; and no remote-tracking ref contains it. The remote-tracking refs are `origin/main` as last fetched and `origin/<branch>` as last pushed or fetched; under `untracked` only `origin/main`.
- *safe:* naive, and no other local branch (or another worktree's HEAD; the model has one such ref, in `fork`) contains HEAD, and the new operation names no SHA of HEAD, in a trailer (`aiwf-authorized-by`, `aiwf-scope-ends`, `aiwf-force-for`) or in the bytes it writes (`addressed_by_commit`).
- *reflog:* safe, and an aiwf verb was the last to move HEAD: the top line of HEAD's reflog is the verb's. A rebase, pull, merge, cherry-pick, checkout or `git commit` writes its own line there, so after any of them the next verb commits anew. The model records, per operator, whether the step that last moved HEAD was a verb; a push or a fetch does not move HEAD.

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

The amend conditions run for the four per-verb designs, every amend design and two fold designs (the best, `replay.append.record.opid.record`, and the default, `replay.replace.record.opid.record`); the original conditions run for every design, the amend designs included.

**Probes.** Two runs on `actor` change the amend test itself, each with a control run:

| probe | invariant | what is changed | run | probe run fails | control run fails | trace |
|---|---|---|---|---|---|---|
| AP1 | L | amend.naive without the actor test: the agent's verb may amend the human's authorize commit, naming it by the SHA the amend replaces (both runs: operators merge main, never rebase) | `merge@actor` | L | none | `AP1-probe_merge@actor@amend.naive.record__L-stuck.txt` |
| AW1 | L | what-if, not a mutation: amend.safe without the actor test, so only its SHA clause stands between the two (both runs: operators merge main, never rebase) | `merge@actor` | none | none | — |

## 2. Matrices

Cells list the invariants that fail, with RESULTS.md's marks: `—` all hold, `L✗` a liveness failure, `!` reported by `aiwf check` on main after the landing, `✗` not reported. Rows are the original conditions, then the amend conditions.

#### The review's naive rule with today's SHA pointers and parent-diff audit (`amend.naive.parent`)

| condition | `merge` | `merge+utd` | `semi` | `semi+utd` | `ff` | `ff+utd` | `squash/title` | `squash/title+utd` | `squash/all` | `squash/all+utd` | `squash/git` | `squash/git+utd` | `direct` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| base | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| after | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| bob | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| mixed | — | — | — | — | — | — | T3! T4 | T3! T4 | — | — | T3! T4 | T3! T4 | — |
| crash | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| lost | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| revert | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T4a T7! | T3! T4a T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4a T7! |
| cherry | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T3! T4 T7! | T1 T3! T4 T7! | T1 T3 T4 T7! | T1 T3 T4 T7! | T1 T3! T4 T7! | T1 T3! T4 T7! | n/a |
| resolve | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T4a T7x | T1 T2 T3x T4a T7x | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3x T4a T7x |
| nocheck | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| reuse | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| auth | L✗ | L✗ | T6! L✗ | L✗ | T4a T6! L✗ | T4a L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 T6 L✗ | T3! T4 T6! L✗ | T3! T4 L✗ | T3! T4 L✗ | T4a L✗ |
| ack | L✗ | L✗ | T3! T6 L✗ | L✗ | T3! T4a T6 L✗ | T4a L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T4a L✗ |
| abc | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a T6 |
| adr | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T3 T4 | T3 T4 | T3! T4 | T3! T4 | T4a |
| sa-base | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-after | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-bob | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| sa-mixed | — | — | — | — | — | — | T3! T4 | T3! T4 | — | — | T3! T4 | T3! T4 | — |
| sa-crash | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-lost | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-revert | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T4a T7! | T3! T4a T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4a T7! |
| sa-cherry | T1 T4a | T1 T4a | T1 T4a | T1 T4a | T1 T4a | T1 T4a | T1 T3! T4 T7! | T1 T3! T4 T7! | T1 T3 T4 T7! | T1 T3 T4 T7! | T1 T3! T4 T7! | T1 T3! T4 T7! | n/a |
| sa-resolve | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T4a T7x | T1 T2 T3x T4a T7x | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3x T4a T7x |
| sa-nocheck | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-reuse | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| fork | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4d |
| reuse4 | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| untracked | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4d |
| scope | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a T6 |
| actor | L✗ | L✗ | T6 L✗ | L✗ | T4a T6 L✗ | T4a L✗ | T3! T4 T7! L✗ | T3! T4 T7! L✗ | T3! T4 T6 T7! L✗ | T3! T4 T6 T7! L✗ | T3! T4 T7! L✗ | T3! T4 T7! L✗ | T4a L✗ |
| byc | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a T6 |
| twostep | L✗ | L✗ | L✗ | L✗ | T4a L✗ | T4a L✗ | T3! T4 T7! L✗ | T3! T4 T7! L✗ | T4 T7! L✗ | T4 T7! L✗ | T3! T4 T7! L✗ | T3! T4 T7! L✗ | T4a L✗ |
| nr-fork | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4d |
| nr-scope | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a T6 |
| nr-byc | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a T6 |

#### The corrected rule with operation-id pointers and the record audit (`amend.reflog.opid.record`)

| condition | `merge` | `merge+utd` | `semi` | `semi+utd` | `ff` | `ff+utd` | `squash/title` | `squash/title+utd` | `squash/all` | `squash/all+utd` | `squash/git` | `squash/git+utd` | `direct` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| base | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| after | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| bob | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| mixed | — | — | — | — | — | — | T3! T4 | T3! T4 | — | — | T3! T4 | T3! T4 | — |
| crash | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| lost | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| revert | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T4a T7! | T3! T4a T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4a T7! |
| cherry | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T3! T4 T7! | T1 T3! T4 T7! | T1 T3 T4 T7! | T1 T3 T4 T7! | T1 T3! T4 T7! | T1 T3! T4 T7! | n/a |
| resolve | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T4a T7x | T1 T2 T3x T4a T7x | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3x T4a T7x |
| nocheck | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| reuse | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| auth | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T3! T4 T6 | T3! T4 T6! | T3! T4 | T3! T4 | T4a |
| ack | L✗ | L✗ | T3! T6 L✗ | L✗ | T3! T4a T6 L✗ | T4a L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T3! T4 L✗ | T4a L✗ |
| abc | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a T6 |
| adr | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T3 T4 | T3 T4 | T3! T4 | T3! T4 | T4a |
| sa-base | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-after | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-bob | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| sa-mixed | — | — | — | — | — | — | T3! T4 | T3! T4 | — | — | T3! T4 | T3! T4 | — |
| sa-crash | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-lost | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-revert | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T4a T7! | T3! T4a T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4a T7! |
| sa-cherry | T1 T4a | T1 T4a | T1 T4a | T1 T4a | T1 T4a | T1 T4a | T1 T3! T4 T7! | T1 T3! T4 T7! | T1 T3 T4 T7! | T1 T3 T4 T7! | T1 T3! T4 T7! | T1 T3! T4 T7! | n/a |
| sa-resolve | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T4a T7x | T1 T2 T3x T4a T7x | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3x T4a T7x |
| sa-nocheck | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-reuse | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| fork | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4d |
| reuse4 | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| untracked | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4d |
| scope | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a |
| actor | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T6 T7! | T3! T4 T6 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| byc | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a T6 |
| twostep | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| nr-fork | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4a |
| nr-scope | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a |
| nr-byc | — | — | T6 | — | T4a T6 | T4a | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a |

#### Today: per-verb commits, parent-diff audit (`perverb.parent`), amend conditions

| condition | `merge` | `merge+utd` | `semi` | `semi+utd` | `ff` | `ff+utd` | `squash/title` | `squash/title+utd` | `squash/all` | `squash/all+utd` | `squash/git` | `squash/git+utd` | `direct` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| sa-base | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-after | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-bob | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| sa-mixed | — | — | — | — | — | — | T3! T4 | T3! T4 | — | — | T3! T4 | T3! T4 | — |
| sa-crash | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-lost | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-revert | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T4a T7! | T3! T4a T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4a T7! |
| sa-cherry | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T3! T4 T7! | T1 T3! T4 T7! | T1 T3 T4 T7! | T1 T3 T4 T7! | T1 T3! T4 T7! | T1 T3! T4 T7! | n/a |
| sa-resolve | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T4a T7x | T1 T2 T3x T4a T7x | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3x T4a T7x |
| sa-nocheck | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-reuse | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| fork | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4d |
| reuse4 | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| untracked | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| scope | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a T6 |
| actor | L✗ | L✗ | T6 L✗ | L✗ | T4a T6 L✗ | T4a L✗ | T3! T4 T7! L✗ | T3! T4 T7! L✗ | T3! T4 T6 T7! L✗ | T3! T4 T6 T7! L✗ | T3! T4 T7! L✗ | T3! T4 T7! L✗ | T4a L✗ |
| byc | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a T6 |
| twostep | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| nr-fork | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4a |
| nr-scope | — | — | T6 | — | T4a T6 | T4a | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a |
| nr-byc | — | — | T6 | — | T4a T6 | T4a | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a |

#### Per-verb commits with operation-id pointers and the record audit (`perverb.opid.record`), amend conditions

| condition | `merge` | `merge+utd` | `semi` | `semi+utd` | `ff` | `ff+utd` | `squash/title` | `squash/title+utd` | `squash/all` | `squash/all+utd` | `squash/git` | `squash/git+utd` | `direct` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| sa-base | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-after | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-bob | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| sa-mixed | — | — | — | — | — | — | T3! T4 | T3! T4 | — | — | T3! T4 | T3! T4 | — |
| sa-crash | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-lost | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-revert | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T4a T7! | T3! T4a T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4a T7! |
| sa-cherry | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T4d | T1 T3! T4 T7! | T1 T3! T4 T7! | T1 T3 T4 T7! | T1 T3 T4 T7! | T1 T3! T4 T7! | T1 T3! T4 T7! | n/a |
| sa-resolve | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T4a T7x | T1 T2 T3x T4a T7x | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3x T4a T7x |
| sa-nocheck | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| sa-reuse | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| fork | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4d |
| reuse4 | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| untracked | — | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| scope | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a |
| actor | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T6 T7! | T3! T4 T6 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| byc | T6 | T6 | T6 | T6 | T4a T6 | T4a T6 | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a T6 |
| twostep | — | — | — | — | T4a | T4a | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T4a |
| nr-fork | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T3! T4 | T4a |
| nr-scope | — | — | — | — | T4a | T4a | T3! T4 | T3! T4 | T4 T6 | T3! T4 T6 | T3! T4 | T3! T4 | T4a |
| nr-byc | — | — | T6 | — | T4a T6 | T4a | T3! T4 T6 | T3! T4 T6 | T3 T4 T6 | T3 T4 T6 | T3! T4 T6 | T3! T4 T6 | T4a |

#### The best fold design (`replay.append.record.opid.record`), amend conditions

| condition | `merge` | `merge+utd` | `semi` | `semi+utd` | `ff` | `ff+utd` | `squash/title` | `squash/title+utd` | `squash/all` | `squash/all+utd` | `squash/git` | `squash/git+utd` | `direct` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| sa-base | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| sa-after | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| sa-bob | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 T7! | T3! T4 T7! | T3 T4 T7! | T3 T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| sa-mixed | — | — | — | — | — | — | T3! T4 | T3! T4 | — | — | T3! T4 | T3! T4 | — |
| sa-crash | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| sa-lost | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| sa-revert | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T7! |
| sa-cherry | T1 T4d | T1 | T1 | T1 | T1 | T1 | T1 T3! T4 T7! | T1 T3! T4 T7! | T1 T3 T4 T4d T7! | T1 T3 T4 T4d T7! | T1 T3! T4 T7! | T1 T3! T4 T7! | n/a |
| sa-resolve | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3x T7x | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3! T4 T7! | T1 T2 T3x T7x |
| sa-nocheck | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| sa-reuse | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | n/a |
| fork | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 | T3! T4 | T3 T4 | T3 T4 | T3! T4 | T3! T4 | T4d |
| reuse4 | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| untracked | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| scope | — | — | — | — | — | — | T3! T4 | T3! T4 | T4 | T3! T4 | T3! T4 | T3! T4 | — |
| actor | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| byc | T6 | T6 | T6 | T6 | T6 | T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T6 |
| twostep | — | — | — | — | — | — | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | T3! T4 T7! | — |
| nr-fork | T4d | T4d | T4d | T4d | T4d | T4d | T3! T4 | T3! T4 | T3 T4 | T3 T4 | T3! T4 | T3! T4 | T4d |
| nr-scope | — | — | — | — | — | — | T3! T4 | T3! T4 | T4 | T3! T4 | T3! T4 | T3! T4 | — |
| nr-byc | T6 | T6 | T6 | T6 | T6 | T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T3! T4 T6 | T6 |

#### Failing cells

Counted as RESULTS.md counts them: one cell per (host setting, condition, invariant), and a safety cell counts as failing also where the run is not live. The two condition sets have different scripts, so their columns are not comparable with each other.

| design | failing cells, original conditions | failing cells, amend conditions |
|---|---|---|
| `amend.naive.parent` | 496 | 641 |
| `amend.naive.record` | 496 | 555 |
| `amend.naive.opid.parent` | 406 | 543 |
| `amend.naive.opid.record` | 406 | 457 |
| `amend.safe.parent` | 496 | 624 |
| `amend.safe.record` | 496 | 538 |
| `amend.safe.opid.parent` | 406 | 531 |
| `amend.safe.opid.record` | 406 | 445 |
| `amend.reflog.parent` | 496 | 612 |
| `amend.reflog.record` | 496 | 526 |
| `amend.reflog.opid.parent` | 406 | 519 |
| `amend.reflog.opid.record` | 406 | 433 |
| `perverb.parent` | 496 | 530 |
| `perverb.record` | 496 | 530 |
| `perverb.opid.parent` | 406 | 437 |
| `perverb.opid.record` | 406 | 437 |
| `replay.append.record.opid.record` | 406 | 435 |
| `replay.replace.record.opid.record` | 435 | 543 |

#### Each amend design against per-verb commits with the same pointers and audit

Every (host setting, condition) cell where an amend design differs from the per-verb design that shares its pointer and audit choice (`amend.<rule>.parent` against `perverb.parent`, `amend.<rule>.opid.record` against `perverb.opid.record`, and so on). A cell not listed is the same under both.

| amend design | condition | amend fails | per-verb fails | host settings |
|---|---|---|---|---|
| `amend.naive.parent` | sa-after | T4d | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.parent` | sa-after | T4d | T4a (`perverb.parent`) | `ff`, `ff+utd` |
| `amend.naive.parent` | sa-bob | T4d | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.parent` | sa-bob | T4d | T4a (`perverb.parent`) | `ff`, `ff+utd` |
| `amend.naive.parent` | sa-cherry | T1 T4a | T1 T4d (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.naive.parent` | untracked | T4d | — (`perverb.parent`) | `merge` |
| `amend.naive.parent` | untracked | T4d | T4a (`perverb.parent`) | `direct` |
| `amend.naive.parent` | twostep | L✗ | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.parent` | twostep | T4a L✗ | T4a (`perverb.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.naive.parent` | twostep | T3! T4 T7! L✗ | T3! T4 T7! (`perverb.parent`) | `squash/title`, `squash/title+utd`, `squash/git`, `squash/git+utd` |
| `amend.naive.parent` | twostep | T4 T7! L✗ | T3! T4 T7! (`perverb.parent`) | `squash/all`, `squash/all+utd` |
| `amend.naive.parent` | nr-fork | T4d | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.parent` | nr-fork | T4d | T4a (`perverb.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.naive.parent` | nr-scope | T6 | — (`perverb.parent`) | `merge`, `merge+utd`, `semi+utd` |
| `amend.naive.parent` | nr-scope | T4a T6 | T4a (`perverb.parent`) | `ff+utd`, `direct` |
| `amend.naive.parent` | nr-byc | T6 | — (`perverb.parent`) | `merge`, `merge+utd`, `semi+utd` |
| `amend.naive.parent` | nr-byc | T4a T6 | T4a (`perverb.parent`) | `ff+utd`, `direct` |
| `amend.naive.record` | sa-after | T4d | — (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.record` | sa-after | T4d | T4a (`perverb.record`) | `ff`, `ff+utd` |
| `amend.naive.record` | sa-bob | T4d | — (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.record` | sa-bob | T4d | T4a (`perverb.record`) | `ff`, `ff+utd` |
| `amend.naive.record` | sa-cherry | T1 T4a | T1 T4d (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.naive.record` | untracked | T4d | — (`perverb.record`) | `merge` |
| `amend.naive.record` | untracked | T4d | T4a (`perverb.record`) | `direct` |
| `amend.naive.record` | nr-fork | T4d | — (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.record` | nr-fork | T4d | T4a (`perverb.record`) | `ff`, `ff+utd`, `direct` |
| `amend.naive.record` | nr-scope | T6 | — (`perverb.record`) | `merge`, `merge+utd`, `semi+utd` |
| `amend.naive.record` | nr-scope | T4a T6 | T4a (`perverb.record`) | `ff+utd`, `direct` |
| `amend.naive.record` | nr-byc | T6 | — (`perverb.record`) | `merge`, `merge+utd`, `semi+utd` |
| `amend.naive.record` | nr-byc | T4a T6 | T4a (`perverb.record`) | `ff+utd`, `direct` |
| `amend.naive.opid.parent` | sa-after | T4d | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.opid.parent` | sa-after | T4d | T4a (`perverb.opid.parent`) | `ff`, `ff+utd` |
| `amend.naive.opid.parent` | sa-bob | T4d | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.opid.parent` | sa-bob | T4d | T4a (`perverb.opid.parent`) | `ff`, `ff+utd` |
| `amend.naive.opid.parent` | sa-cherry | T1 T4a | T1 T4d (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.naive.opid.parent` | untracked | T4d | — (`perverb.opid.parent`) | `merge` |
| `amend.naive.opid.parent` | untracked | T4d | T4a (`perverb.opid.parent`) | `direct` |
| `amend.naive.opid.parent` | twostep | L✗ | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.opid.parent` | twostep | T4a L✗ | T4a (`perverb.opid.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.naive.opid.parent` | twostep | T3! T4 T7! L✗ | T3! T4 T7! (`perverb.opid.parent`) | `squash/title`, `squash/title+utd`, `squash/git`, `squash/git+utd` |
| `amend.naive.opid.parent` | twostep | T4 T7! L✗ | T3! T4 T7! (`perverb.opid.parent`) | `squash/all`, `squash/all+utd` |
| `amend.naive.opid.parent` | nr-fork | T4d | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.opid.parent` | nr-fork | T4d | T4a (`perverb.opid.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.naive.opid.parent` | nr-byc | T6 | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi+utd` |
| `amend.naive.opid.parent` | nr-byc | T4a T6 | T4a (`perverb.opid.parent`) | `ff+utd`, `direct` |
| `amend.naive.opid.record` | sa-after | T4d | — (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.opid.record` | sa-after | T4d | T4a (`perverb.opid.record`) | `ff`, `ff+utd` |
| `amend.naive.opid.record` | sa-bob | T4d | — (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.opid.record` | sa-bob | T4d | T4a (`perverb.opid.record`) | `ff`, `ff+utd` |
| `amend.naive.opid.record` | sa-cherry | T1 T4a | T1 T4d (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.naive.opid.record` | untracked | T4d | — (`perverb.opid.record`) | `merge` |
| `amend.naive.opid.record` | untracked | T4d | T4a (`perverb.opid.record`) | `direct` |
| `amend.naive.opid.record` | nr-fork | T4d | — (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.naive.opid.record` | nr-fork | T4d | T4a (`perverb.opid.record`) | `ff`, `ff+utd`, `direct` |
| `amend.naive.opid.record` | nr-byc | T6 | — (`perverb.opid.record`) | `merge`, `merge+utd`, `semi+utd` |
| `amend.naive.opid.record` | nr-byc | T4a T6 | T4a (`perverb.opid.record`) | `ff+utd`, `direct` |
| `amend.safe.parent` | sa-after | T4d | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.parent` | sa-after | T4d | T4a (`perverb.parent`) | `ff`, `ff+utd` |
| `amend.safe.parent` | sa-bob | T4d | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.parent` | sa-bob | T4d | T4a (`perverb.parent`) | `ff`, `ff+utd` |
| `amend.safe.parent` | sa-cherry | T1 T4a | T1 T4d (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.safe.parent` | untracked | T4d | — (`perverb.parent`) | `merge` |
| `amend.safe.parent` | untracked | T4d | T4a (`perverb.parent`) | `direct` |
| `amend.safe.parent` | twostep | L✗ | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.parent` | twostep | T4a L✗ | T4a (`perverb.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.safe.parent` | twostep | T3! T4 T7! L✗ | T3! T4 T7! (`perverb.parent`) | `squash/title`, `squash/title+utd`, `squash/git`, `squash/git+utd` |
| `amend.safe.parent` | twostep | T4 T7! L✗ | T3! T4 T7! (`perverb.parent`) | `squash/all`, `squash/all+utd` |
| `amend.safe.record` | sa-after | T4d | — (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.record` | sa-after | T4d | T4a (`perverb.record`) | `ff`, `ff+utd` |
| `amend.safe.record` | sa-bob | T4d | — (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.record` | sa-bob | T4d | T4a (`perverb.record`) | `ff`, `ff+utd` |
| `amend.safe.record` | sa-cherry | T1 T4a | T1 T4d (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.safe.record` | untracked | T4d | — (`perverb.record`) | `merge` |
| `amend.safe.record` | untracked | T4d | T4a (`perverb.record`) | `direct` |
| `amend.safe.opid.parent` | sa-after | T4d | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.opid.parent` | sa-after | T4d | T4a (`perverb.opid.parent`) | `ff`, `ff+utd` |
| `amend.safe.opid.parent` | sa-bob | T4d | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.opid.parent` | sa-bob | T4d | T4a (`perverb.opid.parent`) | `ff`, `ff+utd` |
| `amend.safe.opid.parent` | sa-cherry | T1 T4a | T1 T4d (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.safe.opid.parent` | untracked | T4d | — (`perverb.opid.parent`) | `merge` |
| `amend.safe.opid.parent` | untracked | T4d | T4a (`perverb.opid.parent`) | `direct` |
| `amend.safe.opid.parent` | twostep | L✗ | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.opid.parent` | twostep | T4a L✗ | T4a (`perverb.opid.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.safe.opid.parent` | twostep | T3! T4 T7! L✗ | T3! T4 T7! (`perverb.opid.parent`) | `squash/title`, `squash/title+utd`, `squash/git`, `squash/git+utd` |
| `amend.safe.opid.parent` | twostep | T4 T7! L✗ | T3! T4 T7! (`perverb.opid.parent`) | `squash/all`, `squash/all+utd` |
| `amend.safe.opid.record` | sa-after | T4d | — (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.opid.record` | sa-after | T4d | T4a (`perverb.opid.record`) | `ff`, `ff+utd` |
| `amend.safe.opid.record` | sa-bob | T4d | — (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.safe.opid.record` | sa-bob | T4d | T4a (`perverb.opid.record`) | `ff`, `ff+utd` |
| `amend.safe.opid.record` | sa-cherry | T1 T4a | T1 T4d (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.safe.opid.record` | untracked | T4d | — (`perverb.opid.record`) | `merge` |
| `amend.safe.opid.record` | untracked | T4d | T4a (`perverb.opid.record`) | `direct` |
| `amend.reflog.parent` | sa-cherry | T1 T4a | T1 T4d (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.reflog.parent` | untracked | T4d | — (`perverb.parent`) | `merge` |
| `amend.reflog.parent` | untracked | T4d | T4a (`perverb.parent`) | `direct` |
| `amend.reflog.parent` | twostep | L✗ | — (`perverb.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.reflog.parent` | twostep | T4a L✗ | T4a (`perverb.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.reflog.parent` | twostep | T3! T4 T7! L✗ | T3! T4 T7! (`perverb.parent`) | `squash/title`, `squash/title+utd`, `squash/git`, `squash/git+utd` |
| `amend.reflog.parent` | twostep | T4 T7! L✗ | T3! T4 T7! (`perverb.parent`) | `squash/all`, `squash/all+utd` |
| `amend.reflog.record` | sa-cherry | T1 T4a | T1 T4d (`perverb.record`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.reflog.record` | untracked | T4d | — (`perverb.record`) | `merge` |
| `amend.reflog.record` | untracked | T4d | T4a (`perverb.record`) | `direct` |
| `amend.reflog.opid.parent` | sa-cherry | T1 T4a | T1 T4d (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.reflog.opid.parent` | untracked | T4d | — (`perverb.opid.parent`) | `merge` |
| `amend.reflog.opid.parent` | untracked | T4d | T4a (`perverb.opid.parent`) | `direct` |
| `amend.reflog.opid.parent` | twostep | L✗ | — (`perverb.opid.parent`) | `merge`, `merge+utd`, `semi`, `semi+utd` |
| `amend.reflog.opid.parent` | twostep | T4a L✗ | T4a (`perverb.opid.parent`) | `ff`, `ff+utd`, `direct` |
| `amend.reflog.opid.parent` | twostep | T3! T4 T7! L✗ | T3! T4 T7! (`perverb.opid.parent`) | `squash/title`, `squash/title+utd`, `squash/git`, `squash/git+utd` |
| `amend.reflog.opid.parent` | twostep | T4 T7! L✗ | T3! T4 T7! (`perverb.opid.parent`) | `squash/all`, `squash/all+utd` |
| `amend.reflog.opid.record` | sa-cherry | T1 T4a | T1 T4d (`perverb.opid.record`) | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| `amend.reflog.opid.record` | untracked | T4d | — (`perverb.opid.record`) | `merge` |
| `amend.reflog.opid.record` | untracked | T4d | T4a (`perverb.opid.record`) | `direct` |

#### The reflog rule against the best fold design

`amend.reflog.opid.record` against `replay.append.record.opid.record`, every cell where they differ.

| condition | `amend.reflog.opid.record` fails | `replay.append.record.opid.record` fails | host settings |
|---|---|---|---|
| base | T4a | — | `ff`, `ff+utd`, `direct` |
| after | — | T4d | `merge`, `merge+utd`, `semi`, `semi+utd` |
| after | T4a | T4d | `ff`, `ff+utd` |
| after | T4a | — | `direct` |
| bob | — | T4d | `merge`, `merge+utd`, `semi`, `semi+utd` |
| bob | T4a | T4d | `ff`, `ff+utd` |
| crash | T4a | — | `ff`, `ff+utd`, `direct` |
| lost | T4a | — | `ff`, `ff+utd`, `direct` |
| revert | T3! T4a T7! | T3! T7! | `ff`, `ff+utd`, `direct` |
| cherry | T1 T4d | T1 | `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| cherry | T1 T3 T4 T7! | T1 T3 T4 T4d T7! | `squash/all`, `squash/all+utd` |
| resolve | T1 T2 T3x T4a T7x | T1 T2 T3x T7x | `ff`, `ff+utd`, `direct` |
| nocheck | T4a | — | `ff`, `ff+utd`, `direct` |
| reuse | T4d | — | `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| reuse | T3 T4 T7! | T3! T4 T7! | `squash/all`, `squash/all+utd` |
| auth | T4a | — | `ff`, `ff+utd`, `direct` |
| auth | T3! T4 T6 | T3! T4 | `squash/all` |
| auth | T3! T4 T6! | T3! T4 | `squash/all+utd` |
| ack | T3! T6 L✗ | L✗ | `semi` |
| ack | T3! T4a T6 L✗ | L✗ | `ff` |
| ack | T4a L✗ | L✗ | `ff+utd`, `direct` |
| ack | T3! T4 L✗ | L✗ | `squash/all`, `squash/all+utd` |
| abc | T4a T6 | T6 | `ff`, `ff+utd`, `direct` |
| abc | T3 T4 T6 | T3! T4 T6 | `squash/all`, `squash/all+utd` |
| adr | T4a | — | `ff`, `ff+utd`, `direct` |
| adr | T3 T4 | T3! T4 | `squash/all`, `squash/all+utd` |
| sa-base | T4a | — | `ff`, `ff+utd`, `direct` |
| sa-after | — | T4d | `merge`, `merge+utd`, `semi`, `semi+utd` |
| sa-after | T4a | T4d | `ff`, `ff+utd` |
| sa-after | T4a | — | `direct` |
| sa-bob | — | T4d | `merge`, `merge+utd`, `semi`, `semi+utd` |
| sa-bob | T4a | T4d | `ff`, `ff+utd` |
| sa-crash | T4a | — | `ff`, `ff+utd`, `direct` |
| sa-lost | T4a | — | `ff`, `ff+utd`, `direct` |
| sa-revert | T3! T4a T7! | T3! T7! | `ff`, `ff+utd`, `direct` |
| sa-cherry | T1 T4a | T1 T4d | `merge` |
| sa-cherry | T1 T4a | T1 | `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| sa-cherry | T1 T3 T4 T7! | T1 T3 T4 T4d T7! | `squash/all`, `squash/all+utd` |
| sa-resolve | T1 T2 T3x T4a T7x | T1 T2 T3x T7x | `ff`, `ff+utd`, `direct` |
| sa-nocheck | T4a | — | `ff`, `ff+utd`, `direct` |
| sa-reuse | T4d | — | `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| sa-reuse | T3 T4 T7! | T3! T4 T7! | `squash/all`, `squash/all+utd` |
| fork | T3! T4 | T3 T4 | `squash/all`, `squash/all+utd` |
| reuse4 | T4d | — | `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` |
| reuse4 | T3 T4 T7! | T3! T4 T7! | `squash/all`, `squash/all+utd` |
| reuse4 | T4a | — | `direct` |
| untracked | T4d | — | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd`, `direct` |
| untracked | T3 T4 T7! | T3! T4 T7! | `squash/all`, `squash/all+utd` |
| scope | T4a | — | `ff`, `ff+utd`, `direct` |
| scope | T4 T6 | T4 | `squash/all` |
| scope | T3! T4 T6 | T3! T4 | `squash/all+utd` |
| actor | T4a | — | `ff`, `ff+utd`, `direct` |
| actor | T3! T4 T6 T7! | T3! T4 T7! | `squash/all`, `squash/all+utd` |
| byc | T4a T6 | T6 | `ff`, `ff+utd`, `direct` |
| byc | T3 T4 T6 | T3! T4 T6 | `squash/all`, `squash/all+utd` |
| twostep | T4a | — | `ff`, `ff+utd`, `direct` |
| nr-fork | — | T4d | `merge`, `merge+utd`, `semi`, `semi+utd` |
| nr-fork | T4a | T4d | `ff`, `ff+utd`, `direct` |
| nr-fork | T3! T4 | T3 T4 | `squash/all`, `squash/all+utd` |
| nr-scope | T4a | — | `ff`, `ff+utd`, `direct` |
| nr-scope | T4 T6 | T4 | `squash/all` |
| nr-scope | T3! T4 T6 | T3! T4 | `squash/all+utd` |
| nr-byc | — | T6 | `merge`, `merge+utd`, `semi+utd` |
| nr-byc | T4a T6 | T6 | `ff` |
| nr-byc | T4a | T6 | `ff+utd`, `direct` |
| nr-byc | T3 T4 T6 | T3! T4 T6 | `squash/all`, `squash/all+utd` |

#### One choice at a time, around `amend.reflog.opid.record`

As in RESULTS.md: each row changes one dimension of the reference design. `worse`: the reference keeps the invariant and the variant does not; `better`: the reverse; `vacuously`: the side that keeps a safety invariant is not live there. The trace is the failing side's shortest violation.

| dimension | variant | invariant | worse or better | conditions | host settings | trace |
|---|---|---|---|---|---|---|
| amend | naive | T4 | worse | sa-after, sa-bob | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` | `merge@sa-after@amend.naive.parent__T4d.txt` |
| amend | naive | T4 | worse | nr-fork | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd`, `direct` | `merge@nr-fork@amend.naive.parent__T4d.txt` |
| amend | naive | T6 | worse | nr-byc | `merge`, `merge+utd`, `semi+utd`, `ff+utd`, `direct` | `merge@nr-byc@amend.naive.parent__T6.txt` |
| amend | safe | T4 | worse | sa-after, sa-bob | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd` | `merge@sa-after@amend.safe.parent__T4d.txt` |
| ptr | leave | T6 | worse | scope | `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd`, `direct` | `merge@scope@amend.reflog.parent__T6.txt` |
| ptr | leave | T6 | worse | auth, actor, nr-scope | `semi`, `ff` | `semi@auth@amend.reflog.parent__T6.txt` |
| ptr | leave | L | worse | auth, actor | all | `merge@auth@amend.reflog.parent__L-stuck.txt` |
| audit | parent | T3 | better, vacuously | twostep | `squash/all`, `squash/all+utd` | `squash_all@twostep@amend.reflog.record__T3.txt` |
| audit | parent | L | worse | twostep | all | `merge@twostep@amend.reflog.parent__L-stuck.txt` |

#### The `sa-` conditions

For the per-verb and fold designs the amend conditions run, every `sa-` cell equals its original condition's cell, state count included, in all 840 cells compared.

## 3. Failure classes amend adds, with their shortest traces

Each class is an invariant failing in a condition under some amend design while the per-verb design with the same pointers and audit keeps it there. The trace is the shortest over every host setting and amend design in the class; only its actions are shown, and `traces/` holds the full file (branches and main after each step).

**L in `twostep`**, where the per-verb design with the same pointers and audit keeps it. Amend designs: `amend.naive.parent`, `amend.naive.opid.parent`, `amend.safe.parent`, `amend.safe.opid.parent`, `amend.reflog.parent`, `amend.reflog.opid.parent`. Host settings: `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd`, `squash/title`, `squash/title+utd`, `squash/all`, `squash/all+utd`, `squash/git`, `squash/git+utd`, `direct`.
Shortest: `direct@twostep@amend.naive.parent__L-stuck.txt` (7 steps; `direct`, `amend.naive.parent`):

```
violation: terminal, stuck: [('A', ['pre-push hook: fsm-history-consistent on the branch'])]
0. initial state: main holds M (draft) and G (open)
1. A: code a1
2. A: op A_ip
3. A: op A_done [amends HEAD]
4. B: code b1
5. B: op B_add
6. B: push main
7. A: rebase onto main
```

**T4d in `nr-fork`**, where the per-verb design with the same pointers and audit keeps it. Amend designs: `amend.naive.parent`, `amend.naive.record`, `amend.naive.opid.parent`, `amend.naive.opid.record`. Host settings: `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd`, `direct`.
Shortest: `direct@nr-fork@amend.naive.parent__T4d.txt` (6 steps; `direct`, `amend.naive.parent`):

```
violation: A_ip has 2 records on main
0. initial state: main holds M (draft) and G (open)
1. A: op A_ip
2. A: cut branch m from its unpushed tip and check it out
3. A: op A_gbp [amends HEAD]
4. A: code a1
5. A: merge branch m into the branch it was cut from (--no-ff)
6. A: push main   <-- the violating step
```

**T4d in `sa-after`**, where the per-verb design with the same pointers and audit keeps it. Amend designs: `amend.naive.parent`, `amend.naive.record`, `amend.naive.opid.parent`, `amend.naive.opid.record`, `amend.safe.parent`, `amend.safe.record`, `amend.safe.opid.parent`, `amend.safe.opid.record`. Host settings: `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd`.
Shortest: `merge@sa-after@amend.naive.parent__T4d.txt` (15 steps; `merge`, `amend.naive.parent`):

```
violation: A_gbp has 2 records on main
0. initial state: main holds M (draft) and G (open)
1. A: code a1
2. A: op A_ip
3. A: code a2
4. A: op A_gbp
5. A: push (first; MR opened)
6. B: code b1
7. B: op B_add
8. B: push (first; MR opened)
9. host: merge B's MR by merge
10. A: rebase onto main
11. A: op A_done [amends HEAD]
12. host: merge A's MR by merge
13. A: rebase onto main
14. A: push
15. host: merge A's MR by merge   <-- the violating step
```

**T4d in `sa-bob`**, where the per-verb design with the same pointers and audit keeps it. Amend designs: `amend.naive.parent`, `amend.naive.record`, `amend.naive.opid.parent`, `amend.naive.opid.record`, `amend.safe.parent`, `amend.safe.record`, `amend.safe.opid.parent`, `amend.safe.opid.record`. Host settings: `merge`, `merge+utd`, `semi`, `semi+utd`, `ff`, `ff+utd`.
Shortest: `merge@sa-bob@amend.naive.parent__T4d.txt` (16 steps; `merge`, `amend.naive.parent`):

```
violation: A_gbp has 2 records on main
0. initial state: main holds M (draft) and G (open)
1. A: code a1
2. A: op A_ip
3. A: code a2
4. A: op A_gbp
5. A: push (first; MR opened)
6. B: code b1
7. B: op B_add
8. B: push (first; MR opened)
9. host: merge B's MR by merge
10. A: rebase onto main
11. A: op A_done [amends HEAD]
12. bob: fetch A's branch, edit-body M, push
13. A: fetch
14. A: pull --rebase its own remote branch
15. A: push
16. host: merge A's MR by merge   <-- the violating step
```

**T4d in `untracked`**, where the per-verb design with the same pointers and audit keeps it. Amend designs: `amend.naive.parent`, `amend.naive.record`, `amend.naive.opid.parent`, `amend.naive.opid.record`, `amend.safe.parent`, `amend.safe.record`, `amend.safe.opid.parent`, `amend.safe.opid.record`, `amend.reflog.parent`, `amend.reflog.record`, `amend.reflog.opid.parent`, `amend.reflog.opid.record`. Host settings: `merge`, `direct`.
Shortest: `direct@untracked@amend.naive.parent__T4d.txt` (8 steps; `direct`, `amend.naive.parent`):

```
violation: A_gbp has 2 records on main
0. initial state: main holds M (draft) and G (open)
1. A: code a1
2. A: op A_ip
3. A: code a2
4. A: op A_gbp
5. A: push main
6. A: op A_done [amends HEAD]
7. A: merge main into the branch
8. A: push main   <-- the violating step
```

**T6 in `nr-byc`**, where the per-verb design with the same pointers and audit keeps it. Amend designs: `amend.naive.parent`, `amend.naive.record`, `amend.naive.opid.parent`, `amend.naive.opid.record`. Host settings: `merge`, `merge+utd`, `semi+utd`, `ff+utd`, `direct`.
Shortest: `direct@nr-byc@amend.naive.parent__T6.txt` (4 steps; `direct`, `amend.naive.parent`):

```
violation: addressed_by_commit names verb:A_ip#3, not reachable as A_ip from main
0. initial state: main holds M (draft) and G (open)
1. A: code a1
2. A: op A_ip
3. A: op A_gaddrv [amends HEAD]
4. A: push main   <-- the violating step
```

**T6 in `nr-scope`**, where the per-verb design with the same pointers and audit keeps it. Amend designs: `amend.naive.parent`, `amend.naive.record`. Host settings: `merge`, `merge+utd`, `semi+utd`, `ff+utd`, `direct`.
Shortest: `direct@nr-scope@amend.naive.parent__T6.txt` (4 steps; `direct`, `amend.naive.parent`):

```
violation: scope-ends in A_end's record names verb:A_auth#3, not reachable as A_auth from main
0. initial state: main holds M (draft) and G (open)
1. A: code a1
2. A: op A_auth
3. A: op A_end [amends HEAD]
4. A: push main   <-- the violating step
```

## 4. Reading the results

**What each clause of the eligibility test buys.** Each row names the clause, the failure that dropping it lets through, and the run that shows it. The control is the run where the clause is in force.

| clause | without it | shown by |
|---|---|---|
| no remote-tracking ref contains HEAD (every rule) | a pushed commit is amended and its operation lands twice | `untracked` against `reuse4`: the same script with the push tracked, where every rule keeps T4 under the `merge` host setting and in direct mode |
| HEAD's every group carries the new operation's actor (every rule) | the agent's verb amends the human's `authorize` commit and names it by the replaced SHA; under the naive rule the transaction is stuck | probe AP1, control `actor` |
| no other local branch contains HEAD (safe, reflog) | the commit a branch was cut from is amended on that branch only; the merge back lands it twice | `nr-fork`, naive against safe |
| the new operation names no SHA of HEAD, in a trailer (safe, reflog) | `aiwf-scope-ends` names the replaced SHA and the scope never ends; with operation-id pointers the clause is not needed for it | `nr-scope`, naive against safe; `amend.naive.opid.*` keeps it |
| the same, in the bytes the verb writes (safe, reflog) | `addressed_by_commit` names the replaced SHA; operation-id pointers do not cover it | `nr-byc`, naive against safe |
| an aiwf verb moved HEAD last (reflog) | a pushed verb commit the operator rebased is amended; the pushed original survives and the operation lands twice | `sa-after`, `sa-bob`, safe against reflog |
| HEAD is a non-merge verb commit (every rule) | not exercised as a separate clause; with it, no amend run breaks T5 (no code commit is re-created by an amend) or T1 (no amend drops an operation) | every amend matrix |

**Where per-verb commits fail too, amend fails sooner.** In `fork`, `scope`, `byc` and `untracked` the operator's own rebase, or a host's, already breaks today's per-verb commits, so the cells read the same. The naive rule gets there in fewer steps and with no rebase at all: in `fork` the duplicate lands in 6 or 7 steps against 10 or 12, and in `scope` and `byc` the pointer dangles in 4 or 5 steps against 8 to 10. The safe and reflog rules take the per-verb route in those cells, except in `untracked`, where every rule takes amend's shorter route. The `nr-` conditions are those cells with the rebase removed.

**What the reflog rule closes, and what stays open.** Both classes the corrected rule leaves are a pushed operation in a commit that no ref names any more: the push went untracked, or the operator's rebase gave it a new SHA. A test of ref containment cannot see either.
- *The rebase route* (`sa-after`, `sa-bob`) closes under the reflog rule. Two other tests would close it, and neither is modelled. One refuses to amend when HEAD's patch matches a commit on a remote-tracking ref (`git cherry`); it computes a patch-id for every commit the remote has and HEAD lacks, which took seconds per verb for one large pushed branch where the reflog read took milliseconds ([patchid-cost.txt](../../fewer-planning-commits/reflog-rule/out/patchid-cost.txt)). The other refuses when an operation in HEAD is recorded on the remote branch (the fold's W2), which needs an identity in every operation's record.
- *The untracked route* has no local remedy that the model shows. Neither a patch test nor the reflog helps: no tracking ref holds the pushed commit, and a push writes no line on HEAD's reflog. Candidates are fetching before every amend, which helps in direct mode only (the fetch shows the pushed commit on `origin/main`), and refusing to amend on a branch that has no remote-tracking ref. Neither is modelled.

**Where amend reads better than per-verb.** In `sa-cherry`, per-verb commits fail T4d and amend does not. That is not a protection: per-verb's duplicate is B cherry-picking A's lone `edit-body G` commit, and under amend that operation sits in one commit with `promote M done`, which does not apply on B's branch (M is still `draft` there). The cherry-pick that remains, of `promote M in_progress`, does not land twice under either: A's later `promote M done` conflicts with B's copy (argued from the traces, not a separate run).

**Squash hosts.** Apart from `twostep` under the parent audit, amend leaves every squash cell as per-verb commits have it. Under an all-messages squash the last commit's groups survive, and an amended last commit carries more of them, but the earlier commits' groups are still lost.

## 5. Assumptions added, and what the model cannot say about amend

**Assumptions** (RESULTS.md §6's numbering continues):

- **G16.** `git commit --amend`, like the amend modelled here, re-creates HEAD on HEAD's parent: a new SHA, the same parent. The old HEAD stays wherever another ref holds it. This is git's object model, not a measurement here; the review's E2 measured the consequence in a scratch repository (the old commit and the amended one both reached `main` after a merge).
- **G17.** In a `git clone --single-branch` clone, a push of another branch updates no `refs/remotes` ref. Measured by the review's E3 (its clone's fetch refspec covers `refs/heads/main` only, and its pushed `feature` branch has no remote-tracking ref).
- **A16.** The amend as modelled: an aiwf verb amends with the same compare-and-swap ref update a verb commit uses today, so an amend is atomic and has no crash fault of its own; the eligibility test reads the refs as they are locally, without fetching.
- **A17.** A ritual merge commit is untrailered in the model (the rituals' merge commits carry trailers, but both audits skip merge commits, A8 and A9).
- **G18.** Every git command that moves HEAD writes a line to HEAD's reflog that names the command, and a push or fetch writes none there; a ref update with `-m` writes that message. Measured for the commands the model's operators run, and for stash, reset and `commit --amend`, in a scratch repository with git 2.51.1 ([reflog-probe.txt](../../fewer-planning-commits/reflog-rule/out/reflog-probe.txt)).
- **A18.** Under the reflog rule, an aiwf verb's ref update writes a reflog message that marks it as aiwf's. Today's writes none (`internal/gitops/committree.go`, the `update-ref` after `commit-tree`), so the rule needs that one argument added.

**What the model cannot say about amend.**

- **Readers of a multi-operation commit.** Like the fold, the model assumes every reader reads every trailer group in order (A12). The review names readers that do not: `readActorOpenerEntities` (`internal/cli/cliutil/provenance.go`) reads `aiwf-entity` into one field, and `SortedTrailers` (`internal/gitops/trailers.go`) has no production caller, so nothing re-sorts a group's trailers. An amend whose run includes an `authorize` breaks the first; the model cannot see it.
- **Per-operation times.** Commit dates are abstracted away (G12), so the model cannot say that an amended commit keeps one committer date for every operation in it.
- **What a verb printed.** An earlier verb's `metadata.commit_sha` names a commit the next amend replaces. The model has no consumer of verb output.
- **`git reset HEAD~` and the reflog.** Undoing a run, and aiwf's reflog-reading check, are not modelled. Of the reflog the model keeps only whether a verb moved HEAD last, for the reflog rule.
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

2434 distinct explorations (amend designs in every group, every design in the amend conditions, and the amend probes) produced the 7,164 (host setting, condition, design) results this document adds; 7,303,050 states in total; 0 runs hit the state cap; 0 state graphs had a cycle. The same run re-ran the original matrix and probes (2145 explorations), whose results are RESULTS.md's. Largest runs:

| run | states | BFS depth | complete | secs |
|---|---|---|---|---|
| `merge+utd@sa-bob@replay.replace.record.opid.record` | 189,480 | 28 | yes | 38.47 |
| `semi+utd@sa-bob@replay.replace.record.opid.record` | 189,480 | 28 | yes | 38.49 |
| `merge+utd@sa-bob@replay.append.record.opid.record` | 164,633 | 25 | yes | 33.79 |
| `semi+utd@sa-bob@replay.append.record.opid.record` | 164,633 | 25 | yes | 32.1 |
| `merge@sa-bob@replay.append.record.opid.record` | 149,989 | 25 | yes | 27.85 |
| `semi@sa-bob@replay.append.record.opid.record` | 144,666 | 25 | yes | 28.18 |
| `merge+utd@sa-after@replay.replace.record.opid.record` | 144,531 | 22 | yes | 25.61 |
| `semi+utd@sa-after@replay.replace.record.opid.record` | 144,531 | 22 | yes | 25.6 |
