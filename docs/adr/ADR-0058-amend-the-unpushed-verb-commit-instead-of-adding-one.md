---
id: ADR-0058
title: Amend the unpushed verb commit instead of adding one
status: proposed
---
> **Date:** 2026-11-08 · **Decided by:** Peter Bruinsma

## Context

Every mutating verb makes its own commit (kernel commitment 7), so planning dominates history. Over the ten weeks the [fewer-planning-commits research](../research/fewer-planning-commits.md) replayed, 1,883 of the 2,603 commits reachable on `main` are planning commits (§1). Most of them come in *runs*, verb commits with nothing else between them. Amend turns a run into one commit: a verb adds its operation to the commit at HEAD when that commit is the operator's own unpushed verb commit. Replayed over the window, it leaves 608 planning commits where today's verbs made 1,806, both counted without the rituals' `ROADMAP.md`-only commits (§4.1).

The mechanism had to keep no state beyond git, rewrite no commit a verb did not make, and work without anyone remembering a step (§5). Amend does all three. The prototype ran it through ten git scenarios ([its report](../research/fewer-planning-commits/reports/03-amend-prototype.md)), and the [transactions model](../research/transactions/model/RESULTS-amend.md) checked it against the same conditions as the fold. Over the conditions added for amend, the rule below fails fewer of the model's cells than today's per-verb commits with either kind of pointer, or than the best fold design (§4.4).

**Alternatives considered:**

- *Fold before push.* It leaves fewer commits only if it runs before every branch cut and every merge of another line into a branch; run only at push or merge, it is level with amend. It re-creates the code commits after the first planning commit it folds, which breaks the SHAs written into the repository before the push: acknowledgments, gap closures and citations in entity bodies. A pre-push hook cannot run it, so it is a step a user can skip (§2, §4.3).
- *Batch, with verbs not committing.* The kernel's own guards refuse the second verb that touches a file, and the pending state needs a second record beside git (§4.2).
- *The same rule without the reflog clause.* A `git rebase` gives a pushed verb commit a SHA no ref contains, so every containment clause passes and the next verb amends the pushed commit. In the model, the operation then lands twice (RESULTS-amend, headline item 3).
- *A patch-id test in place of the reflog clause.* It closes the same hole, but costs about 3 s for every verb while a large branch is pushed and not merged, against about 12 ms for reading the reflog (§4.4).
- *A setting to turn amend off, or amend as opt-in.* The commit path for a HEAD that is not eligible stays either way, so an off switch is a small change if a case for one arises. Nobody has asked for one. An opt-in reaches only the repositories that know to ask, and this one would opt in at once.

**What carries several operations in one commit.** Measured 2026-11-08 with aiwf built from `d50379c2e` (an untracked directory outside the build marked it `+dirty`) and git 2.51.1 on Linux, in a scratch repository with a bare remote. Four verbs ran on a pushed base: an `add decision`, its promotes to `accepted` and then `superseded`, and an `add gap`. Their records were then written into one commit on that base in each format below, and today's aiwf was run over it.

| Format | Verb trailers git parses | `commit-msg` hook | `aiwf history` | `aiwf check` | Hooked push |
|---|---|---|---|---|---|
| Today's keys, one group per operation | 4 | accepted | One event per entity, with every actor, verb and target stacked in one cell; the gap's history also shows the decision's operations | 0 errors | accepted |
| One new key per operation | 0 | accepted | "no history" for both entities | 2 errors, `provenance-untrailered-entity-commit` | refused |
| Today's keys for the last operation only, the rest under a new key | 1 | accepted | "no history" for the decision; one event for the gap | 0 errors | accepted |

Git notes do not carry the record: a plain push sent none to the remote, and neither a cherry-pick nor a rebase carried a note to the new commit.

Only the first format keeps every operation visible to today's readers. It also reads every commit already in history as a group of one. A new key would need a second format beside today's for ever, or a break of every reader on every commit. An older release still misreads a commit in the first format without saying so, so whichever release writes one must follow a release that reads one.

**Reading the steps.** In the model, a commit holding `promote in_progress` and then `promote done` fails a check that compares the commit with its parent: the check reads `draft → done`, and the push is refused. A check that reads the commit's steps in order passes it (RESULTS-amend, headline item 4).

A step can also be claimed twice. After a push whose reply was lost, `git pull --rebase` leaves an amended commit whose message still claims the first operation while its diff holds only the second (§4.4; the prototype's scenario S05, case c2). If that claimed operation is a promote, a check reading every step reads a step to the status the entity already has. The FSM refuses such a step: `entity.ValidateTransition` returned "cannot transition" for `accepted → accepted` on a decision, and likewise for a gap, a milestone and an epic. The push would be refused over a commit whose diff is legal. That refusal is argued from the model's audit, which reads every step; no step audit exists in aiwf to run.

## Decision

- **A verb amends HEAD when HEAD is the operator's own unpushed verb commit**, which means all of the following hold; otherwise it commits as today:
  - HEAD is on a branch and has one parent.
  - Every operation HEAD records is a kernel verb's, not a ritual's, and has the new operation's actor and principal.
  - The top line of HEAD's reflog is a verb's and names HEAD. Every verb writes its ref update with a reflog message beginning `aiwf:`. `git commit`, `commit --amend`, `reset`, `rebase`, `merge`, `pull`, `cherry-pick`, `checkout` and `stash` each write a line of their own; a push or a fetch writes none (§4.4).
  - No ref other than HEAD's branch contains HEAD: no other branch, tag, stash or remote-tracking ref, and no other worktree's HEAD.
  - Nothing the new operation writes names HEAD's SHA.
- **An amend replaces HEAD with one commit.** It sits on HEAD's parent, holds HEAD's tree plus the operation's change, keeps HEAD's author and author date, and takes a new committer date. The ref update is a compare-and-swap against HEAD under the repository lock, as on today's commit path.
- **The commit records each operation in today's trailer keys:**
  - The subject is `aiwf: <n> operations (<verb> <entity>, …)`, naming the first four.
  - The body holds one section per operation, in order, headed `[i] <its subject>`. The operation's body is indented beneath its heading, so no body line can be read as the next heading.
  - The trailer paragraph holds one group per operation, in order. Each group starts at `aiwf-verb` and carries that operation's trailers, including its `aiwf-op-id` ([ADR-0057](ADR-0057-give-every-operation-an-identity-and-read-a-repeat-once.md)) and an `aiwf-op-time` in RFC 3339.
  - Every trailer in a group belongs to that group's operation, so an `aiwf-force` exempts its own operation's step and no other.
- **Every verb writes `aiwf-op-time` on its own commit too.** A commit with one operation is then a group of one, and an amend appends a record without changing an earlier one: the trailer paragraph HEAD had is a prefix of the amended commit's.
- **Readers read a commit as the operations it records.** One reader splits a commit into its operations. `aiwf history`, `aiwf show`, `aiwf status` and the render pass list operations, so an event is an operation. A commit without `aiwf-op-time` reads as a group of one, timed by its author date.
- **`aiwf check` reads a commit's status steps in order against the commit's diff.** It skips a step whose operation one of the commit's ancestors already records: the record that counts is the first in history order, as in ADR-0057.
- **Amend is written only where the rest is in place.** No release writes an amended commit unless it also writes ADR-0056's scope ids and ADR-0057's operation identities, and unless the release before it reads a commit with several operations.
- **Amend is on in every repository, with no setting.**
- **Commitment 7 gains one sentence:** "The commit may replace the operator's own unpushed verb commit, carrying that commit's operations forward." The verb still makes exactly one commit, so the promise stays atomic.

## Consequences

- **A run of verbs becomes one commit.** A run is ended by a code commit, a merge, a push, a branch cut from HEAD, a ritual's commit, a git command that moves HEAD, or an operation that names HEAD's SHA.
- **An operation can still be recorded twice**, and nothing refuses it. Each of these routes does it:
  - a push that leaves no remote-tracking ref, followed by a merging pull;
  - a clone made from the working repository;
  - a branch created while a verb amends.

  Readers read the repeat once and `aiwf check` warns of it (ADR-0057).
- **Two outcomes are not repeats:**
  - A push made while a verb amends lands the pre-amend commit, and the next push is rejected until the operator pulls.
  - After a lost push reply, a pull with rebase leaves a commit that claims an operation its diff lacks. The step audit skips that operation, and readers read it once.
- **Undoing is coarser.** `git reset HEAD~` undoes a run, not one verb.
- **`metadata.commit_sha` names a commit the next amend replaces.** It still names the commit that holds the operation when the verb exits, but a script that saves it holds a SHA that stops resolving. In this repository only a stress test reads the field.
- **Each verb pays for the containment test.** It took about 0.5 s per verb in this repository (§4.4). Its cost with a fresh commit-graph is not measured (§6).
- **An older release misreads an amended commit** (Context), and no finding says so. A clone that stays on an older release keeps misreading.
- **A commit made by `git commit` is never amended,** even one carrying a verb trailer, because the reflog clause refuses any HEAD a git command moved last.
- **Signing follows today's commit path:** `commit.gpgsign` signs the amended commit. Signing, `post-rewrite` and notes under amend are not measured (§6).
- **What changes:**
  - the commit path in `internal/gitops`: the eligibility test, the amend, and a reflog message on every verb's ref update;
  - a trailer key, `aiwf-op-time`, and the [provenance model](../design/provenance-model.md)'s trailer table;
  - one reader that splits a commit into operations, replacing every reader of aiwf trailers that takes one operation per commit;
  - on top of that reader:
    - the history readers, `aiwf status` and the render pass;
    - `fsm-history-consistent` and the other rules that read `aiwf-force`;
  - the docs that state commitment 7:
    - `CLAUDE.md` and [design-decisions.md](../design/design-decisions.md) §"One git commit per mutating verb" gain the sentence, as do the rule tables that restate it;
    - [workflows.md](../workflows.md)'s "a six-step sequence below is six commits" is reworded;
  - [design-lessons.md](../design/design-lessons.md)'s convention that an *event* is a trailer-bearing commit, which becomes an operation;
  - the tests that count one new commit per verb:
    - the prototype's run of `internal/gitops` and `internal/verb` found four;
    - the rest of the suite and the stress tests' commit counts were not run with it;
  - the CHANGELOG entry, which says history changes shape in every repository and a saved `commit_sha` can stop resolving.

## Validation

The decision holds while all of these do:

- the research's replay, run again on a window after release, shows planning commits falling as §4.1 predicts;
- every `operation-recorded-twice` warning traces to a route the Consequences name;
- no amend has rewritten a commit that no verb made.

Revisit it if any of these happens:

- a repeat arrives by another route;
- an amend changes a record it carried forward;
- the containment test grows slow enough that verbs feel it.

## References

- [ADR-0056](ADR-0056-name-an-authorize-scope-by-a-random-id-not-its-commit-sha.md): the scope id an amended `authorize` commit keeps, where a pointer to its SHA would stop resolving.
- [ADR-0057](ADR-0057-give-every-operation-an-identity-and-read-a-repeat-once.md): the operation identity, the reading of a repeat once, and the warning.
- [ADR-0022](ADR-0022-verb-commits-built-via-a-temp-index-commit-tree-primitive.md): the commit-tree path that the amend extends.
- [ADR-0036](ADR-0036-same-status-fsm-transitions-converge-to-noop-not-refusal.md): a request already satisfied makes no commit, and so amends nothing.
- The research this decision draws on:
  - [`docs/research/fewer-planning-commits.md`](../research/fewer-planning-commits.md): the counts (§1, §4.1), the rule and what remains (§4.4), what amend needs (§5) and what is not settled (§6);
  - [its evidence](../research/fewer-planning-commits/README.md): the prototype's report and scenarios, and the reflog probes;
  - [the model's amend results](../research/transactions/model/RESULTS-amend.md);
  - [`docs/research/collaborative-state-and-commit-granularity.md`](../research/collaborative-state-and-commit-granularity.md): today's readers on a commit with several operations (§8, E5) and the open questions on the record (§10, questions 2, 3 and 8).
