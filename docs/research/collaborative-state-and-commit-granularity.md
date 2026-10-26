# Transactions and planning commits: collaborative planning with quiet history

> **Status:** thesis-draft
> **Date:** 2026-10-26
> **Question:** How can people and AI agents plan and execute in parallel, on Git hosts whose settings aiwf does not control, while ordinary branch history stays readable and a visitor to the default branch still sees the plan?
> **Working answer:** Treat each bounded piece of work as a *transaction* on one branch. Its planning verbs commit there as they do today; submitting the transaction folds them into one planning commit at the tip, carrying a record of every operation; the host's merge commits it. Ids need a serialization point, and which one is chosen by modelling and measurement, not by assertion. Everything beyond a small core is opt-in.
> **Audience:** maintainers deciding the next kernel change; anyone adopting aiwf on a host and settings it does not choose.
> **Premise:** builds on [`05-where-state-lives`](05-where-state-lives.md), [KERNEL.md](KERNEL.md), [D-0037](../../work/decisions/D-0037-defer-adr-0001-g-0281-and-emb-pending-a-measured-id-collision-trigger.md), [D-0054](../../work/decisions/D-0054-keep-the-reasoning-derive-the-facts-record-obligations-not-events.md), [D-0073](../../work/decisions/D-0073-planning-happens-on-main-implementation-work-happens-in-a-worktree.md), [D-0095](../../work/decisions/D-0095-rituals-enter-new-worktrees-with-cd-not-enterworktree.md) and the [id-lifecycle initiative](../initiatives/id-lifecycle.md); the measurements in §8 come from runs on 2026-10-25 and 2026-10-26.
> **Tags:** #aiwf #collaboration #git #transactions #ids #provenance #hosts

---

## Abstract

aiwf records planning state as Markdown in ordinary Git branches, and every mutating verb makes one commit. In this repository's history, 133 of the 200 commits `git log -200 736018f9` lists carry an `aiwf-verb` trailer, and 165 of the 200 first-parent commits do. On a branch a team shares, that is the history everyone reads. Some host settings make it worse: a title-only squash merge destroys aiwf's record, and a commit-message rule may refuse its commits outright. aiwf is also adopted all at once: its branch model, provenance audits and rituals assume aiwf owns the branch, and its ids are allocated by scanning what one machine can see.

This document proposes a smaller kernel with opt-in layers and two changes. First, a *transaction* — a feature, a chore, an epic being planned, an ADR being written — runs on one branch named the team's way, its planning verbs commit there as they do today, and submitting it folds those commits into one *planning commit* at the tip, carrying a per-operation record in its trailers. Second, ids get a deliberate serialization point; §6 compares the candidates.

Measurements against real aiwf verbs, a scripted submit and local emulations of host merge methods found the record surviving every merge method except a title-only squash; today's readers garbling a multi-operation commit silently; a design with a second planning worktree failing on five counts; and `aiwf reallocate` misdirecting references after a collision between machines. A [companion study of id allocation](id-allocation-under-concurrent-work.md) compares the id designs in a model and in experiments. A [model checker of the fold](transactions/model/RESULTS.md) compares its open choices under §5's merge methods and squash messages, each with and without the up-to-date rule, and §4.2 states the design it supports. Three defects that predate this proposal are filed: [G-0743](../../work/gaps/G-0743-the-repo-lock-lands-in-the-project-tree-in-a-linked-worktree.md), [G-0744](../../work/gaps/G-0744-the-untrailered-entity-audit-trusts-any-commit-carrying-an-aiwf-verb-trailer.md) and [G-0745](../../work/gaps/G-0745-aiwf-reallocate-rewrites-references-meant-for-the-other-colliding-entity.md). §9 sets out how a model checker and property tests together decide the open design questions before code is written.

## 1. Goals

1. **Adopt in part.** A project takes the core and the disciplines it wants. KERNEL.md already states this as a quality bar ("Modular and opt-in"); the current implementation does not deliver it, because several disciplines have no switch (§3).
2. **Quiet code history.** A branch shows its code commits and one planning commit, not one commit per verb.
3. **A visible plan.** A visitor to the default branch on any host sees the planning tree as Markdown, current as of the last merge. Real-time freshness is not required.
4. **Memory for LLMs.** An agent can find what is planned, what was decided and what happened, and archived entities drop out of the active view.
5. **Robust parallel work.** Multi-operation changes are atomic; ids do not collide, or collisions are stopped before they land; worktrees and machines do not overwrite each other's work.
6. **Coexist with external trackers.** A project may keep tickets in a tracker such as Jira as the source of truth, and do untracked work (chores, docs) beside it. This is a link from an entity to a ticket, not the tracker sync that `design-decisions.md` defers.
7. **Host-neutral.** No guarantee depends on one host's features; where a guarantee needs a host setting, aiwf says which, and detects its absence.

## 2. Requirements

| # | Requirement |
|---|---|
| R1 | aiwf is a core plus opt-in disciplines (§3). |
| R2 | Acceptance criteria are core. Whether `met` needs mechanical evidence follows the milestone's TDD policy. |
| R3 | Contracts stay, as the place to plug in verifiers whose output gives an agent feedback. |
| R4 | A transaction lands as its code commits plus one planning commit, not one commit per verb. |
| R5 | The default branch carries the visible plan, updated at each merge. Nothing syncs in real time. |
| R6 | Both integration modes are first-class and neither is preferred: commits directly to the default branch, and a protected default branch that changes only through a merge request. |
| R7 | Each operation's record — entity, operation, actor, principal, authority where one applies, and order — lives in commit metadata, not in a separate file. |
| R8 | Provenance keeps the principal and the operator. The operator is the actor that ran the verb (`aiwf-actor`); the git author records whose identity made the commit. |
| R9 | No guarantee relies on a feature only some hosts offer without aiwf naming the setting and detecting its absence. |
| R10 | No database backend. Dolt was considered and excluded: it adds a dependency and a schema migration, and planning state would stop being Markdown in the repository's branches. |

## 3. A layered aiwf

**Core, always on:** the six entity kinds (commitment 1) with their acceptance criteria; ids, allocation and `reallocate`; each kind's legal status transitions; references between entities; `show`, `list`, `status`, `history`, `schema`, `template`; archive; and a minimal `aiwf check` covering tree shape, id uniqueness, frontmatter validity and resolvable references.

**Disciplines, opt-in.** "Switch today" reads `internal/config/config.go` and `aiwf.yaml`'s parsed blocks at `1e8e17e6c`. Several existing keys raise a severity or make a field mandatory rather than turn a discipline on or off.

| Discipline | What it covers | Switch today |
|---|---|---|
| Contract verification | `contract verify/bind/recipes`, validators over fixtures | effectively yes: inert without bindings in `aiwf.yaml`'s `contracts:` block |
| TDD | `tdd` policy, `acs-tdd-audit`, the `aiwf-tests` trailer | per milestone (`--tdd` is required at `aiwf add milestone`) |
| Areas | `area_*` checks, `set-area`, mistag acknowledgements | by use: the checks act on area tags and `areas.members`; `areas.required` makes a tag mandatory |
| Authorization scopes | `authorize`, `isolation_escape` | by use only |
| Provenance audits | `provenance`, `fsm_history_consistent`, `trailer_verb_unknown`, `id_rename_untrailered`, `reflog_walk`, `cherry_picks`, `post_cutoff`, `acknowledge illegal` | none |
| Branch model | `promote_on_wrong_branch`, `cross_branch`, `head_history`, the verb-side branch guard | none |
| Prose and doc hygiene | `body_prose_id`, `doc_id_*`, `entity_body_section_dropped`, `milestone_release_note` | `docs.strict` raises severity |
| Priority | `set-priority` and its checks | by use only |
| Generated views | `ROADMAP.md`, `STATUS.md`, the HTML site | partly (`status_md.auto_update`, `html`) |
| Git hooks | pre-commit shape check, pre-push full check, post-commit status, commit-msg checks | `aiwf init --skip-hook` (a flag); `status_md.auto_update`; `provenance.refuse_coauthors` |
| Host artifacts | skills, rituals, agent cards, guidance packs | yes (`hosts`, `guidance`) |
| Worktree management | `aiwf worktree add` | by use; `worktree.dir` sets placement |
| Commit granularity | one commit per verb (kernel commitment 7) | none; §4 changes it |

Two observations bear on adoption in part:

- **The branch model refuses a team's branch names.** In the spike, `aiwf promote <milestone> in_progress` on a branch named `feature` was refused by the verb-side guard (`internal/verb/promote_branch_guard.go`): the activation "is expected on" the epic's own `epic/…` branch. A team whose branches follow its tracker's names cannot activate a milestone without `--force`. This is the ADR-0010 branch model; §11 lists what changing it touches.
- **Evidence for `met` is enforced in different places.** `acs-tdd-audit` enforces commitment 8's rule — an AC at `met` needs `tdd_phase: done` — only for milestones at `tdd: required` or `advisory`. Mechanical evidence for `met` is a review-time discipline ([D-0038](../../work/decisions/D-0038-ac-mechanical-evidence-reviewer-discipline-stands-no-evidence-flag.md)), and the shipped guidance (`internal/skills/embedded-guidance/aiwf-guidance.md`) requires it for every milestone, with an exception for criteria that claim an observation. R2 changes the guidance, not the kernel.

## 4. Transactions

### 4.1 What a transaction is

A **transaction** is one bounded piece of work — implementing a feature, a chore, planning an epic, writing an ADR, a decision or a document — on one branch in one worktree. Its code commits and its planning verbs both land on that branch, so the worktree always shows the transaction's own pending plan. It is identified by its branch and by the entity it is about, and needs no new entity kind.

| Database transaction | aiwf transaction |
|---|---|
| begin | the branch and its worktree are created |
| writes | code commits and planning verbs, each verb one commit as today |
| read your own writes | the worktree's `work/` holds the pending plan |
| prepare | submitting folds the planning commits into one planning commit at the tip |
| validate | `aiwf check` over the combined result |
| commit | the host merges the branch, landing code and planning together |
| rollback | abandoning the branch |

**What a transaction guarantees.** Its pending work is visible only on its own branch until it merges, unless someone cherry-picks one of its planning commits onto another branch: that branch's merge then lands those operations, and nothing reports it (§9.2). Code and planning land together in one merge, or not at all. Abandoning it leaves the default branch untouched. A transaction merged without being submitted lands its per-verb commits as today, and keeps what today keeps.

**What it does not.** The combined result is checked against the *current* default branch only where the host requires a merge request to be up to date and its pipeline to pass (§5); elsewhere, a check on the default branch after the merge is the backstop. Concurrent transactions are not serialized: two that change the same entity may conflict at merge, or may merge cleanly into a state neither checked. Nothing rolls a transaction back after it merges. Nothing is durable before it is pushed.

### 4.2 Submit: folding planning commits

Submitting rewrites the transaction's own branch so that its planning commits become one commit at the tip:

- **It replays, it does not copy.** The planning commit is the net change of the planning commits, applied by a three-way merge onto the branch as it stands. Copying `work/` from a snapshot instead reverts any planning another transaction merged meanwhile (§8, R1). The planning commit carries every file the verbs wrote, including `docs/adr/` and `aiwf.yaml`, not only `work/`. Commits before the first folded verb keep their SHAs; the code commits after it are re-created in order, and a merge of the default branch is re-made with its original second parent.
- **It folds only what has not been pushed.** An operation whose record is already on the transaction's remote branch, judged by the operation's identity, not by a commit's SHA, stays as the per-verb commit it is; submit folds the rest, and refuses if a pushed operation comes after an unpushed one. A pushed per-verb commit outlives a local fold: the host can merge the earlier push, or another operator builds on it and the next `git pull --rebase` brings it back, and an operation folded anyway is then recorded twice (§8, E7). The operator's own rebase gives pushed commits new SHAs, so a test by SHA misses it. What was pushed before submitting lands as per-verb commits and keeps today's weaknesses: in the model, a cherry-pick or a branch reused after its merge can land such a record twice under semi-linear, fast-forward or up-to-date hosts, and a host's rebase or squash can break a SHA it names. Refusing the whole submit instead, and merging the transaction as its per-verb commits, agrees with this rule on every invariant in the model and folds less.
- **It never makes a push need force.** In the model, no submit under that rule made the next push need force on a branch that did not need it. A push after the operator's own rebase of a pushed branch, which the up-to-date rule can require, still needs force, and a lease naming the SHA last pushed: a plain `--force` drops another operator's commit.
- **It refuses others' commits and mixed commits.** It refuses when the commits it would fold or re-create include anyone else's, judged by author. A commit touching both `work/` and code cannot be split without changing what its author committed, so it refuses that too.
- **It signs as the operator.** With signing configured, the folded commits are signed by whoever submits; aiwf's commit primitive must sign explicitly, since `git commit-tree` "does not transparently honor `commit.gpgsign`" ([ADR-0022](../adr/ADR-0022-verb-commits-built-via-a-temp-index-commit-tree-primitive.md)).
- **It records each operation.** The planning commit carries one trailer group per operation, in operation order. Each group carries the operation's own identity, and the planning commit carries the transaction's; the format is open (§10, question 2). Every verb's own commit carries its operation's identity as well, since the push rule and the authorization pointers both look an operation up by it.

![A branch before and after submit: three code commits interleaved with three per-verb planning commits become the same three code commits followed by one planning commit whose trailers carry one group per operation; submit folds only operations not yet pushed, signs as the operator, and refuses commits by others or commits that mix code and planning. The host then merges the branch into the default branch.](figures/transaction-fold.svg)

![A sequence across four lanes: the agent, the transaction's branch, the host and the default branch. The agent runs aiwf verbs and code commits on the branch, submits to fold the planning commits into one, pushes and opens a merge request. Another merge lands first; where the host requires the branch to be up to date, CI re-runs aiwf check against the moved base before the host merges code and planning together. If the check fails, the agent revises on the branch; how a second submit works is open (§4.2, item 4).](figures/transaction-sequence.svg)

The fold changes SHAs and puts several operations in one commit. The [model](transactions/model/RESULTS.md) settles how the design handles three of the consequences, and leaves the fourth open (§9.2):

1. **Records that name commit SHAs.** `aiwf-authorized-by` and `aiwf-scope-ends` name an `authorize` operation. Here they carry that operation's identity instead of a SHA, and the provenance check resolves them by searching the records. The model checks `aiwf-authorized-by`; `aiwf-scope-ends` names the same kind of operation. Rewriting the SHA during the fold cannot work: the `authorize` step is folded into the planning commit the pointer would have to name, and no commit can carry its own SHA. Refusing to fold a transaction that holds such a pointer lands it per-verb, where the next rebase breaks the pointer anyway, as it does in today's aiwf (§8, E6). `acknowledge illegal <sha>` of a hand edit and `addressed_by_commit` of a code commit name commits that are not aiwf operations. They break whenever that commit is re-created, by the fold, the operator's rebase, a semi-linear or fast-forward host, or a squash, and no candidate fixes them.
2. **Transitions collapsed into one commit.** The transition audit reads the record's operations in order. Comparing each commit with its parent, as `fsm-history-consistent` does today, reads a milestone started and finished in one transaction as `draft → done`, and the pre-push hook refuses that transaction every time.
3. **Copies of planning commits.** A branch that keeps going after it merged, or a cherry-pick, can carry a copy of a planning commit already on the default branch under another SHA, so ancestry does not show that the two are one commit; in the model that happens under semi-linear, fast-forward and up-to-date hosts. `aiwf check` refuses an operation recorded in two planning commits, and submit refuses to fold records already on the default branch until the branch contains it; the identities help only because the check reads them. The check runs on the branch before the merge, so without the up-to-date rule a cherry-picked copy whose pipeline passed before its original merged still lands twice.
4. **Submitting again, which stays open.** A second submit either re-folds the existing planning commit with the verbs run since (replace) or adds a second planning commit (append). Under the push rule above, a pushed planning commit stays as it is, so either way verbs run after it fold into a second planning commit; the two differ only before the planning commit is pushed, where replace keeps one planning commit and append makes two. In the model they behave identically in every cell. R4's single planning commit therefore holds while nothing is pushed before the last submit, and, under append, only for a transaction submitted once.

### 4.3 Rejected: a separate planning branch in a second worktree

The planning verbs could instead commit to a companion branch, named aiwf's way, in a second worktree, with submit copying its result onto the transaction's branch. Review and measurement rejected it on five counts:

- **An agent cannot reach the second worktree under Claude Code.** [D-0095](../../work/decisions/D-0095-rituals-enter-new-worktrees-with-cd-not-enterworktree.md) records that, by Claude Code 2.1.233, a session that called `EnterWorktree` "is refused `cd` and `git -C` into any other checkout", and rejects routing through `--root` because it "can change with any Claude Code release".
- **Copying the planning branch's tree reverts other transactions' planning** (§8, R1).
- **Authorization records point at commits that never land** (§8, R2).
- **One epic cannot have two transactions in flight,** because git refuses to check out the epic branch in two worktrees and milestone activation must happen on exactly that branch (§8, R3).
- **The transaction's own worktree reads stale planning** (§8, E2), which works against goal 4.

### 4.4 What earlier designs collapse into

- A *work unit* and a *workflow transaction* are one concept, the transaction.
- A *published snapshot* is the default branch after a transaction merges.
- A *shared Git-ref live store* is not needed for planning state, which travels inside the merge request; a shared ref survives only as a candidate id ledger (§6).

## 5. Hosts and their settings

GitHub, GitLab, Azure DevOps and a bare Git remote share the Git protocol. What differs, and what threatens aiwf's goals, is configuration. aiwf needs to know a project's settings, and to refuse or warn where a setting defeats a guarantee rather than let the record degrade silently.

| Setting | Values | Effect on aiwf | Measured |
|---|---|---|---|
| Merge method | merge commit, semi-linear, fast-forward, squash | Merge commit, semi-linear and fast-forward keep every trailer; rebasing changes code SHAs, and the SHA of every commit a record names (§4.2). | locally emulated (§8, E4) |
| Squash message | title only; title and description; every commit's message; host templates | Title only keeps no record. A message that appends each commit's message in order keeps only its last paragraph's trailers, so the record survives only while the planning commit is the branch's last commit. Merging the default branch in after submitting moves it off the end, and in the model every design then loses the record; requiring the planning commit at the tip when the host squashes would close that, which is not modelled. `git merge --squash`'s own default message indents each message, so git parses no trailers from it, as [D-0080](../../work/decisions/D-0080-carry-gap-closure-claims-in-a-start-time-body-section-not-a-commit-trailer.md) also measured. A co-author paragraph a host appends after the trailer block hides it. | locally emulated; host templates not measured |
| Squash policy | off, allowed, encouraged, required | "Required" with a title-only message makes R7 unachievable. | not measured |
| Up to date before merge | required or not; merged-results pipelines; merge queues or trains | Decides whether `aiwf check` sees the tree that will land, and so whether a collision can be stopped before it lands (§6). | not measured |
| Commit-message rules | e.g. a ticket key in every subject | May refuse aiwf's commits at push. | not measured |
| Branch naming | team or tracker conventions | Collides with aiwf's branch model (§3). | aiwf side measured |
| Protected default branch | on or off | Decides which R6 mode applies, and whether the default branch can take an id-minting commit (§6). | n/a |
| Required signatures | on or off | A fold must sign as the operator (§4.2). | partly (§8, E1) |
| Protection of other branches | force pushes refused on a named branch | Matters only for a ledger branch (§6). A ledger outside `refs/heads` would not work: a default fetch does not bring it, and the allocator does not see it. | not measured |

The chokepoint is weaker than kernel commitment 3 reads. This repository's CI runs no `aiwf check` over its own planning tree ([G-0536](../../work/gaps/G-0536-aiwf-check-has-no-ci-backstop-the-primary-oracle-is-per-clone-opt-in.md)); hooks are installed per clone; and on a branch's first push, with no upstream set, the provenance audit is skipped ([G-0539](../../work/gaps/G-0539-a-plain-commit-staging-entity-files-passes-pre-commit-and-hides-until-push.md), [G-0679](../../work/gaps/G-0679-a-branch-started-from-a-local-ref-is-never-judged-at-its-first-push.md)). A host-merged default branch is judged by nothing unless a CI job runs `aiwf check` there.

## 6. Ids: choosing a serialization point

Handing out the next number so that nobody else can take it needs something that orders the claims. aiwf has none in team mode today: it allocates by scanning the refs one machine can see ([ADR-0025](../adr/ADR-0025-allocator-s-cross-branch-view-spans-all-refs-fed-to-allocation-only.md)), and [D-0073](../../work/decisions/D-0073-planning-happens-on-main-implementation-work-happens-in-a-worktree.md) keeps planning on `main` because "a freshly-allocated id that exists only on a branch is invisible to every other session and clone". With a protected default branch, planning cannot go to `main` directly, so a transaction allocates on its branch.

What was measured (§8, I1–I3):

- **On one machine there is no collision.** The allocator saw the other branch through local and remote-tracking refs and handed out the next number.
- **Across machines that have not seen each other's branches, both allocate the same id,** and a host that merges without re-checking lands the collision on the default branch.
- **The repair misdirects references.** Whenever `aiwf reallocate` runs on a tree that holds both colliding entities — the default branch after both merged, or a branch after the default branch is merged in, as the trunk-collision hint suggests — it rewrites every reference to the old id, including those that meant the entity keeping it. Frontmatter moves too: a milestone ended up under the other epic. `aiwf check` and the pre-push hook pass the result ([G-0745](../../work/gaps/G-0745-aiwf-reallocate-rewrites-references-meant-for-the-other-colliding-entity.md)). Reallocating on the branch before merging the default branch in gives the right result.
- **Holes in a sequence are harmless.** This repository's gaps run to G-0744 with three numbers absent, two left behind by reallocations and one that never appears in history, and nothing reads a hole as an error. Reuse is not harmless, because a stale citation would then name a different entity, and today's allocator can hand out an abandoned branch's id again once clones prune that branch ([id allocation](id-allocation-under-concurrent-work.md), §4).

Three designs order the claims, and a fourth borrows the host's:

| Design | Where claims are ordered | Conditions | Costs |
|---|---|---|---|
| **A. The default branch is the ledger** | the host's merges, with "up to date before merge" and a passing pipeline | that host setting; a CI job that runs `aiwf check`; a `reallocate` that rewrites only the references its own branch introduced | ids change after the branch is pushed, and a citation of the old id on another branch then names the wrong entity; merge attempts grow with concurrent merge requests |
| **B. A ledger branch** | Git's compare-and-swap on a non-force push to a shared branch, for example `aiwf/ids` | every clone claims its ids before its first push and confirms the claim by reading it back; a check that refuses an unclaimed id; the network at claim time | one push per allocation; abandoned claims leave holes |
| **C. Collision-free ids** | nowhere: unique by construction, as Git's own SHAs are | a new allocator; a namespace in the id also needs a change to the id grammar of commitment 2 | ids are longer, or prefixed, and harder to say aloud |
| **D. The merge request's number** | the host, which numbers every merge request | the host's API; a merge request must exist first | breaks R6 and R9; shares one number space with ordinary allocation; a second entity of a kind per request has no legal shape |

[ADR-0001](../adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md)'s mint at integration is a variant of A that assigns the id in a commit on the default branch, which a protected branch admits only through a bot. A tracker such as Jira is the ledger for tracked work; an epic or milestone can carry its ticket key, while ADRs, decisions and gaps still need aiwf ids. The [id-lifecycle initiative](../initiatives/id-lifecycle.md) specifies a confirmation protocol "at the level of detail a TLA+ module needs".

The [id-allocation study](id-allocation-under-concurrent-work.md) checks these designs, today's allocator and ADR-0001's mint against five properties: unique, the right target for every reference, no reuse, stable once pushed, and progress. It finds that with merge requests only the designs that fix an id before its branch is pushed, or replace a provisional name at one point, keep references right. Of those, B is the only one that held every property in both integration modes without changing the id shape. The study's §10 sets out the choice, and §7 sets out what it changes in the initiative's protocol.

## 7. Provenance: who, by what operation, on whose authority

The [provenance model](../design/provenance-model.md) names the principal, accountable and always human, and the operator, `aiwf-actor`, "who ran the verb", which may be an LLM. The git author and committer record whose identity made the commit, which is declared, not proven: `user.email` is whatever the machine says, and signed commits strengthen it.

A planning commit has one git author and one date. Each operation keeps its own `aiwf-actor` in its trailer group, so the operator survives; in the spike an `edit-body` run by a second person survived the fold as `aiwf-actor: human/bob`. Each operation's time does not survive, while KERNEL.md's need 4 reads "when a thing changed, who changed it, why"; R7 may need to carry it.

The planning commit makes one check load-bearing that is weak today. [G-0744](../../work/gaps/G-0744-the-untrailered-entity-audit-trusts-any-commit-carrying-an-aiwf-verb-trailer.md) records that the untrailered-entity audit skips any commit carrying an `aiwf-verb` trailer before it looks at what the commit touched; a planning commit whose trailers name fewer entities than it changes passes the same way (§8, E5). Merge commits are skipped as well, so a conflict resolution in a merge commit is unattested (§8, R4).

## 8. Measurements, 2026-10-25 and 2026-10-26

**Method.** Disposable repositories created by `aiwf init`; per-verb commits made by real aiwf verbs and hooks from a binary built at `1e8e17e6c` (built with untracked local files, so it reports `vcs.modified=true`); git 2.51.1 on Linux. The submit step and its planning commit were built by a script, not by an aiwf verb. Host merge methods are emulated with local git; no host was involved. The fixture transaction in E1, E2 and E4 interleaves three code commits with four operations: add an AC, start the milestone, edit a gap's body (run by a second person), mark the AC met. R1–R4 were demonstrated by an independent reviewer; R1 and R2 were re-run. E6 and E7 ran on 2026-10-26: E6 with aiwf built at `905cec139` (`main@v0.41.0-86-g905cec139`, also built with untracked local files, so `vcs.modified=true`), E7 with plain git and trailers written by hand.

**E1 — fold per-verb commits into one planning commit.** Final tree identical; code commits' patch-ids identical; every code commit got a new SHA, the one before the first planning commit included, because the script cherry-picked each onto the base. The fold §4.2 describes keeps the commits before the first folded verb ([model](transactions/model/RESULTS.md), §4). With the folder's signing turned off they came out unsigned; with it on, they are re-signed by the folder.

**E2 — planning in a second worktree, then submit.** Code commit SHAs unchanged; tree hashes identical to E1's; `aiwf status --worktrees` showed the milestone `[in_progress]` in the planning worktree while `aiwf show` in the transaction's worktree said `draft`.

**E3 — pending planning files copied into the code worktree.** `git add -A` swept them into a code commit, and the commit hooks accepted it. `aiwf check --since base` raised `provenance-untrailered-entity-commit`; a first push with no upstream went through, the audit skipped. With the files unstaged, `git rebase main` refused under default settings.

**E4 — merging into a default branch that moved meanwhile.** Commits added to the default branch:

| Method | Per-verb commits (today) | Planning commit |
|---|---:|---:|
| merge commit | 8 | 5 |
| semi-linear | 8 | 5 |
| fast-forward | 7 | 4 |
| squash, title only | 1 | 1 |
| squash, every message appended | 1 | 1 |

Every method kept all four `aiwf-verb` values except title-only squash, which kept none, and the appended squash, which kept one for the per-verb shape and four for the planning-commit shape. The planning-commit shape's clean `promote-on-wrong-branch` result depends on the order of its operations: that rule reads only the last value of a key.

**E5 — readers on a multi-operation commit.** Repeating today's trailer keys per operation: git parses every group and the commit-msg hook accepts them, but `aiwf history` shows one event per touched entity with every actor, verb and target joined into one cell, and `aiwf check` stays clean. A new `aiwf-op:` key: the commit counts as untrailered (two errors and a warning). A planning commit whose trailers name only the last of four operations: `aiwf check` reports nothing. A reviewer's search found readers of aiwf trailers in 27 non-test files. They handle a repeated key four ways — keep the last value, keep the first, join the values, or read all of them — and only the last way can read a multi-operation commit.

**E6 — rebasing a branch that carries an `authorize` commit, in today's aiwf.** On a branch: `aiwf authorize E-0001 --to ai/claude`, a code commit, the agent's `aiwf promote M-0001 in_progress --actor ai/claude --principal human/peter`, which recorded `aiwf-authorized-by` with the `authorize` commit's SHA, and `aiwf promote G-0001 addressed --by-commit` naming the code commit; the hooked push passed. After another clone moved the default branch, `git rebase origin/main` re-created the `authorize` commit under a new SHA. `aiwf check` then reported `error provenance-authorization-missing: … does not resolve to an authorize/opened commit`, and the hooked push was refused. The gap's `addressed_by_commit` named a commit the rebased branch no longer contains, and `aiwf check` reported nothing about it ([G-0396](../../work/gaps/G-0396-addressed-by-commit-sha-is-merge-fragile-derive-closure-from-git-history.md) tracks that).

**E7 — folding operations that were already pushed.** A branch carried a code commit, a per-verb commit, a code commit, and was pushed; a second per-verb commit followed. A fold kept the first code commit, re-created the second, and wrote one planning commit carrying both operations' groups. Another clone added a commit to the pushed branch and pushed it. `git pull --rebase` then skipped the re-created code commit as already upstream and replayed the planning commit on top, so the branch recorded the first operation twice: in its per-verb commit and in the planning commit.

**R1 — copying the planning tree reverts other planning.** Another transaction merged a gap; the transaction's branch was brought up to date and submitted by copying `work/` from its planning branch; the planning commit deleted that gap, and after the merge `aiwf show` reported it "not found" while `aiwf check --since base` reported no findings.

**R2 — authorization records point at commits that never land.** An agent's operation inside an `authorize` scope opened on the planning branch recorded `aiwf-authorized-by: <sha>` of that branch's `authorize` commit; the check on the transaction's branch failed with `provenance-authorization-missing`.

**R3 — one epic, one checkout.** Git refuses to check out the epic branch in a second worktree, and milestone activation refuses on any other branch.

**R4 — merge resolutions and reverts.** A conflict resolution in a merge commit that also moved a milestone from `draft` to `done` passed with no errors; a host revert of a merge produced untrailered entity changes and errors.

**I1–I3 — ids.** In one repository, a second branch allocated G-0003 after the first allocated G-0002. In two clones made before either allocated, both allocated G-0002. After both branches merged with `--no-ff`, `aiwf check` on the default branch raised `ids-unique` errors, and `aiwf reallocate` renamed one and rewrote both citations to the new id. With hooks installed, the second clone's pre-push check caught the collision; after merging the default branch into its branch and reallocating, the first clone's citation again named the renumbered gap, and the push passed. Reallocating before merging the default branch in left both citations right.

**Not covered:** behaviour of real hosts; a crash part-way through a submit; signing the planning commit itself; the direct-to-default-branch mode. The [model](transactions/model/RESULTS.md) covers a crash during submit and the direct mode, without measurement. For ids, the [id-allocation study](id-allocation-under-concurrent-work.md) covers the direct mode in its model.

## 9. Deciding by model and measurement

### 9.1 Two tools, each covering the other's blind spot

| | Model checking | Property tests |
|---|---|---|
| Runs | an abstract model of the protocol | the real code |
| Varies | sequences of actions by several actors, with faults, exhaustively within a bound | inputs to one unit, sampled or exhaustive when small |
| Finds | design flaws: races, orderings, a missing rule | implementation bugs: a unit breaking its stated property |
| Needs code | no | yes |
| Blind spot | only as faithful as the model | one unit at a time, rarely interleavings |

Used together, each covers the other:

- the model finds a design flaw and gives a short trace;
- the trace becomes a scenario test against the binary;
- property tests (the `wf-property-test` skill) check that each unit keeps the properties the model assumed — for submit, that the tree is right, that the record equals the operations in order, that code patch-ids are unchanged and that no planning change is unattested;
- tests against real git pin the facts the model encodes;
- once both exist, action sequences generated from the model are replayed against the binary as a differential oracle, which [`oracles.md`](../design/oracles.md) records the repository does not have.

The model is a tool for designing the next version first. Every flaw in §8's R-series was a design flaw a small model would have shown. Whether it stays as a permanent oracle depends on what keeping it in step with the protocol costs.

### 9.2 The model

![A state machine: open, then prepared after submit folds the planning commits, then in review once pushed with a merge request open, then merged when the host merges. In review loops on itself when the base moves and aiwf check re-runs, and returns to open when the check fails or changes are requested. Open, prepared and in review can all be abandoned by deleting the branch, which leaves the default branch untouched. A note says the environment can act in every state, and that the direct-to-default-branch mode is not drawn because it is undecided.](figures/transaction-states.svg)

The phase in the figure is one dimension of a transaction's state. The full state is the product of these, alongside the default branch's tree and history with each commit's *git-parsed* trailers:

| Dimension | Values |
|---|---|
| Phase | open, prepared, in review, merged, abandoned |
| Submission | not submitted; submitted; verbs run after submit; submitted twice |
| Branch against the default branch | up to date; behind; conflicting |
| Push state | not pushed; pushed; pushed and then rewritten; a pushed copy diverged on another machine |
| Host setting | merge method × squash message × up-to-date rule (§5) |
| Last `aiwf check` on the combined result | passed; failed; not run |
| Paths written | `work/` only; also `docs/adr/` or `aiwf.yaml` |
| Record fields that name SHAs | none; authorization; acknowledgement; `addressed_by_commit` |
| Operators | one; several |
| Authorization scope | none; open; paused; ended |
| Ids | none allocated; provisional; claimed under design A, B or C; colliding |
| Dependencies on other transactions | none; cites an entity another unmerged transaction allocated; shares a scope |

**Actions:** a verb; a code commit; submit; push; another transaction merging first; a rebase onto, or a merge of, the default branch; the host's merge by each method; a revert; a cherry-pick; a conflict resolution in a merge commit; an id claim under each design. The environment acts too: a hand edit, `git add -A`, a crash, a lost push reply, a host merging without re-checking.

**Safety invariants:**

- A merged transaction contributes all of its operations or none, and nothing it submitted is left out.
- The planning commit changes exactly the planning commits' net change, replayed onto the branch.
- Every entity change on the default branch is attested by its commit's trailers or is one the attesting verb may make; a merge commit's own content is attested or explicitly excluded.
- The operation record survives the merge.
- Code commits keep their SHAs unless they follow a folded verb, the host rebases, or the operator rebases.
- References resolve on the default branch after every merge, to the entity their author meant.
- Ids are unique on the default branch.

**Liveness, conditional:** a legal transaction eventually merges or is refused with an actionable reason, provided the host is reachable, contention is bounded, and no unacknowledged finding sits on the default branch. Under the up-to-date rule, one bad landing, such as a host revert, fails every later transaction's check until someone runs `aiwf acknowledge illegal` on it.

Three limits shape the model:

1. **Its assumptions are measurements.** "Git parses trailers only from a message's last paragraph" and "a title-only squash keeps no trailers" are facts about git and hosts. The model encodes them; tests against real git pin them.
2. **The moves nobody intends carry the value.** [G-0744](../../work/gaps/G-0744-the-untrailered-entity-audit-trusts-any-commit-carrying-an-aiwf-verb-trailer.md) came from an ordinary git move outside aiwf, found by enumerating four orderings by hand; arbitrary git commands are environment actions.
3. **One invariant needs a decision first.** "A change the attesting verb may make" needs a record of which entities each verb may touch, and no record holds that today.

**Results.** [The model](transactions/model/RESULTS.md) checks these invariants, except that it takes ids as unique and checks only the references that name commits, within the bounds of §9.3, step 1. It covers the merge-commit, semi-linear and fast-forward methods and three squash messages, each with and without the up-to-date rule, and the direct mode; of the state and actions above it leaves out ids, abandonment and dependencies between transactions. It supports the design §4.2 states. No combination of the choices keeps every invariant. What no design keeps, today's per-verb commits included:

- **A host revert** lands untrailered entity changes and moves a milestone `done → draft`. `aiwf check` reports both after the revert lands, and the liveness condition above then bites.
- **A conflict resolved by hand in a merge commit** can drop the resolver's own operation, revert the other transaction's, and move a milestone `draft → done`. Under merge-commit, semi-linear and fast-forward hosts nothing reports it, because both audits skip merge commits.
- **A cherry-picked planning commit** lands its operations through another transaction's merge, unreported.
- **Records that name commits which are not aiwf operations** (§4.2, item 1).
- **Title-only and git-default squash** lose the record unreported; `aiwf check` reports the unattested change and the illegal-looking step only after the merge.

Against today's per-verb commits, the design with §4.2's push rule keeps every invariant they keep except under an all-messages squash of a transaction whose planning commit was cherry-picked, where the record lands twice; and it keeps the authorization pointer, which today's per-verb commits lose to any rebase (§8, E6), everywhere except under an all-messages squash, where they lose it too. The model is the fold as §4.2 describes it, not code: submit, the record audit, operation identities and the duplicate check do not exist in aiwf.

### 9.3 Order of work

1. **Model the candidates before ratifying.** A small model — 2 transactions, up to 3 operations each, 1 base move, 1 re-submit, one fault per trace — compares the open choices of §4.2 under the host settings of §5 it can express; §9.2 summarises its [results](transactions/model/RESULTS.md). Ids were modelled and measured on their own ([id allocation](id-allocation-under-concurrent-work.md)), so this model takes ids as unique and stable.
2. **Measure what the model assumes.** Host settings on GitHub, GitLab and Azure DevOps; whether hosts accept a ledger branch and protect it from force pushes; how often concurrent allocation happens in a team. Both models list what they assume of git and the host, and the [fold model's list](transactions/model/RESULTS.md) (§6) marks the assumptions it shares with the id model, so one round of host measurements covers both.
3. **Ratify.** Decisions on the questions in §10, recorded as ADRs.
4. **Reconcile with the legal-workflow spec.** Its `branch` package holds 130 cells (counted 2026-10-25), all but one citing [ADR-0010](../adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s branch model, including a matrix of branch kinds against branch kinds. [ADR-0011](../adr/ADR-0011-legal-workflow-spec-methodology.md) scopes its methodology to entity rules and leaves branch choreography to the layer the `branch` package holds; a transaction layer belongs beside it. Each existing cell carries over or becomes an explicit decision.
5. **Name the corner cases** the model's counterexamples expose, each a cell with paired tests.
6. **Implement step by step,** each step useful alone and leaving aiwf correct, the spec cell before the code, and readers before writers: aiwf reads, or refuses loudly, a multi-operation commit before anything writes one.

Both models are small explorers hand-written in Python's standard library; whether [verifier-kit](https://github.com/23min/verifier-kit) could host them is unchecked.

## 10. Open questions

1. **Host settings.** For each host aiwf supports, which settings exist, which combinations aiwf can honour, and how aiwf learns them: declared in `aiwf.yaml`, detected, or both.
2. **The multi-operation record.** The trailer format, with each group carrying its operation's identity and the planning commit the transaction's (§4.2), and every reader of §8, E5 taught to read it or refuse it, including the transition walker reading the steps in order.
3. **What "event" means.** `design-lessons.md` §6 records that in aiwf prose an *event* is a trailer-bearing commit; a planning commit carries several operations.
4. **Which entities each verb may touch.** Needed by the attestation invariant and by any fix to G-0744. [G-0480](../../work/gaps/archive/G-0480-no-check-rule-detects-frontmatter-laundering-already-in-history.md) was declined because the field-level equivalent does not exist and because G-0475 was taken to cover the one route that silences a blocking rule.
5. **Ids.** Design A, B, C or D (§6). The [id-allocation study](id-allocation-under-concurrent-work.md), §10, sets out the choice.
6. **Records that name SHAs** after a fold. The model settles `aiwf-authorized-by`, and by the same argument `aiwf-scope-ends`: both carry the `authorize` operation's identity (§4.2, item 1); `acknowledge illegal <sha>` of a hand edit and `addressed_by_commit` stay open.
7. **Submitting again:** whether a re-submit before pushing keeps one planning commit or adds a second (§4.2, item 4). Transaction identity is settled: in the record, with a check that refuses a duplicate.
8. **Each operation's time** in the record (§7).
9. **Two operators in one transaction,** beyond an aiwf-asserted trailer.
10. **The direct-to-default-branch mode.** Per-verb commits as today, or a fold before each push. In the model a fold before each push works with §4.2's choices, and today's per-verb commits lose the authorization pointer there too.
11. **Merge commits and reverts.** Attesting a conflict resolution; a revert path. No design in the model attests either (§9.2).
12. **The tracker link.** How an entity points at a ticket, and how untracked work is represented.
13. **Contracts as verifiers.** Whether "plug in a verifier that gives the agent feedback" fits the contract surface as it stands, which validates fixtures against a schema.

## 11. Kernel commitments and decisions this touches

Each is a kernel-level decision, not a refactor:

- **Commitment 7, one commit per mutating verb, or none.** A transaction lands as one planning commit, or as more commits when it pushes before its last submit or re-submits under append (§4.2, item 4); the NoOp clause stands.
- **Commitment 4, `aiwf history` reads `git log`.** Still true once the readers understand a multi-operation commit.
- **Commitment 3, `aiwf check` at pre-push.** It needs a CI step that judges the default branch after a host merge.
- **Commitment 9, provenance.** Scopes are addressed by the SHA of their `authorize` commit, which a fold or a rebase changes (§8, E6). §4.2 addresses them by the operation's identity instead, which changes the shape the [provenance model](../design/provenance-model.md) gives `aiwf-authorized-by` and `aiwf-scope-ends`.
- **Commitment 2, ids.** Design A renumbers ids after they are pushed. Design C changes the allocator, and, with a namespace in the id, the id grammar.
- **CLAUDE.md's exclusion of "a cross-branch monotonic ID counter".** Design B is one.
- **[D-0073](../../work/decisions/D-0073-planning-happens-on-main-implementation-work-happens-in-a-worktree.md), planning on `main`.** Protected mode moves planning onto a branch, the exposure D-0073 names; §6 is the answer it needs.
- **[ADR-0010](../adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md), the branch model.** A transaction's branch is named by the team.
- **The module system.** `design-decisions.md` defers it until "the framework grows past ~20 verbs"; R1's opt-in disciplines reopen that item.

## 12. Deliberately not proposed

- A new entity kind for the transaction or the snapshot.
- A database, daemon or service.
- A second worktree per transaction.
- A snapshot export distinct from the default branch.
- Nested transactions.

## Relation to prior research, decisions and initiatives

- [`05-where-state-lives`](05-where-state-lives.md) concluded that planning state belongs in the repository; transactions keep it there.
- [`07-state-not-workflow`](07-state-not-workflow.md) frames aiwf as a state model with workflow as an optional render; a transaction adds no stage model, only a boundary over a branch and its entities.
- [D-0054](../../work/decisions/D-0054-keep-the-reasoning-derive-the-facts-record-obligations-not-events.md) holds that "a completed act does not [need an entity]: the commit holds it, the trailers make it queryable"; with a planning commit, one commit holds several acts.
- [D-0037](../../work/decisions/D-0037-defer-adr-0001-g-0281-and-emb-pending-a-measured-id-collision-trigger.md) deferred ADR-0001 and [G-0281](../../work/gaps/G-0281-opt-in-gaps-inbox-file-gaps-via-plumbing-onto-a-never-checked-out-ref.md)'s coordination ref on a reallocation rate measured in this repository's history; design B in §6 is that ref applied to every kind, under a regime that history did not contain.
- The [agent-agnostic execution topology](../initiatives/agent-agnostic-execution-topology.md) initiative holds that aiwf "does not prescribe or own the user's workflow"; one worktree per transaction keeps to that.
- The [adoption-maturity advisor](../initiatives/adoption-maturity-advisor.md) initiative holds that aiwf "never mandates adoption"; the opt-in layers of §3 are what it would advise on.
- The [controlled-forgetting](../initiatives/controlled-forgetting.md) initiative offers current sources by default and history only "through an explicit, labeled recall step", which is goal 4's archive-to-forget.
- The [id-lifecycle](../initiatives/id-lifecycle.md) initiative reconciles the id-minting designs and specifies the confirmation protocol §6 starts from.
- Open gaps this design depends on or would close: [G-0536](../../work/gaps/G-0536-aiwf-check-has-no-ci-backstop-the-primary-oracle-is-per-clone-opt-in.md), [G-0539](../../work/gaps/G-0539-a-plain-commit-staging-entity-files-passes-pre-commit-and-hides-until-push.md), [G-0602](../../work/gaps/G-0602-a-conflict-resolved-inside-a-shipped-skill-escapes-the-provenance-gate.md), [G-0657](../../work/gaps/G-0657-aiwf-history-drops-commits-whose-trailer-block-is-split-from-co-authored-by.md), [G-0679](../../work/gaps/G-0679-a-branch-started-from-a-local-ref-is-never-judged-at-its-first-push.md), [G-0698](../../work/gaps/G-0698-planning-rituals-still-require-a-ritual-branch-contrary-to-accepted-workflow.md).

## Provenance

This document replaces an exploratory research package dated 2026-10-24 that compared three storage backends (batched Git, a shared Git-ref store, a Dolt store) behind a common oracle, and from which the description of verifier-kit comes. It records a maintainer discussion on 2026-10-24 and 2026-10-25, a spike, and four independent reviews of an earlier draft (facts, design, scope, consistency with aiwf's records), whose demonstrations are §8's R-series. The scripts behind §8 are not committed. The model checker under [`transactions/model/`](transactions/model/RESULTS.md) was written by a subagent from a brief and run on 2026-10-26. Its fidelity replay and two matrix groups were re-run in a separate copy, the whole matrix was re-run from the committed copy and matched the first run in every cell, and `rule_w2.py` was added to run the push rules of §4.2 across the matrix. E6 and E7 were measured the same day. A second subagent with no part in the writing checked the claims against the model's output and re-ran E6 and E7. The three figures draw the design as this document states it; each was rendered in light and dark in headless Chrome 147 and inspected.
