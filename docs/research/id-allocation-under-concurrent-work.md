# Id allocation under concurrent work: what each design guarantees

**Status:** research, 2026-10-25. It accompanies §6 of [Transactions and planning commits](collaborative-state-and-commit-granularity.md).

## Question

Several people or agents allocate aiwf ids on separate clones and branches, and a host merges their work. Which allocation designs keep ids unique, keep every reference pointing at the entity its author meant, and leave an id unchanged once others have seen it? Under which host settings does each work, and what does it cost?

Two independent tracks answered this:

- **A model.** An explicit-state model of every design explores every interleaving of clones, host and faults within small bounds.
- **Experiments.** Scripted runs against real Git, using aiwf built from `1e8e17e6c`.

They agree except in the places §6 lists. Where they disagree, the measurement corrects the model.

## Answer

1. **Pushing straight to the default branch, today's allocator keeps ids unique and unchanged,** apart from the same-file case in §4. A non-force push and the pre-push hook put the claims in order. What breaks is the repair: `aiwf reallocate` misdirects references on any tree that holds both colliding entities ([G-0745](../../work/gaps/G-0745-aiwf-reallocate-rewrites-references-meant-for-the-other-colliding-entity.md)).
2. **With merge requests, no design that renumbers an id after the branch is pushed keeps references correct.** A citation of the old id can sit on someone else's branch, out of reach of any rewrite. Once another entity takes that id, the citation names the wrong entity. References stay correct only when the id is fixed before the branch is pushed, by a ledger (B) or by construction (C), or when a provisional name becomes a number at a single point (D, E).
3. **Today's allocator with merge requests lands duplicates on the default branch** unless the host requires branches to be up to date before merging and runs `aiwf check` in a required pipeline. With that setting (design A), duplicates in different files cannot land. Ids still change after they are pushed, and three concurrent merge requests took 6 merge attempts and 6 to 9 reallocations.
4. **A duplicate that has landed cannot be repaired the way aiwf's hint suggests.** Renumbering one copy is refused by aiwf's hook, whether the push goes to the default branch or to a branch. Only renumbering both copies, in one particular order, or bypassing the hook gets through.
5. **A ledger branch (B) held every property in every mode and every fault explored,** including lost push replies and offline clones.
   - It works with today's allocator unchanged, as "allocate, claim, and reallocate if the claim is taken".
   - A clone that doesn't take part breaks it, so it needs a new check.
   - It is the "cross-branch monotonic ID counter" that CLAUDE.md lists among deliberate exclusions.
6. **Collision-free ids (C) need no network and no host feature.** They do need a change to the id grammar in kernel commitment 2, and today's parsers silently ignore the shapes they would use.
7. **One problem hits every design, and two more hit every design that renumbers:**
   - an abandoned branch that another branch cited leaves a dangling reference;
   - two entities with the same id, title and content merge into one file;
   - once clones prune a deleted branch, its ids can be handed out again.

## 1. What a design must keep

| | Property | Broken when |
|---|---|---|
| I1 | **Unique.** No two entities share an id on the default branch. | two merge requests land the same id |
| I2 | **Right target.** Every reference resolves to the entity its author meant. | a renumber carries a reference over to the other entity, or the cited entity is renumbered and its old id taken |
| I3 | **No reuse.** An id, once published, names one entity forever. | an abandoned branch's id is handed out again |
| I4 | **Stable.** An id does not change after it has been pushed. | the losing side of a collision is renumbered during review |
| I5 | **Progress.** Every collision ends resolved, or refused with a remedy a person can apply. | repairs chase each other, or a clone is stuck |

I2 and I4 separate the designs more sharply than uniqueness does.

To check I2, both tracks give every entity an identity of its own, apart from its id, and record with each reference which identity its author meant. The experiments write that into the reference (`G-0002 (means:alpha-gap)`) and resolve it through aiwf's own loader.

## 2. The designs

| | Design | What orders the claims |
|---|---|---|
| S0 | Today. The next number after the highest id in every ref the clone can see ([ADR-0025](../adr/ADR-0025-allocator-s-cross-branch-view-spans-all-refs-fed-to-allocation-only.md)). `aiwf check` detects a collision, and `aiwf reallocate` renumbers one side and rewrites every reference to the old id. | the push to the default branch in direct mode; nothing with merge requests |
| S0b | S0, with a reallocate that rewrites only the references its own branch introduced. | as S0 |
| A | S0b, plus the host's "branch must be up to date" rule (or a merge train) and a required pipeline that runs `aiwf check` on the merge result. | the host's merges |
| A+ | A, plus two guards. One refuses a citation whose id is ambiguous in the clone's view. The other checks a new id against every visible ref before its first push. | as A |
| B | A ledger branch, `aiwf/ids`. A clone claims an id with a non-force push that adds one file. *Eager* claims at `aiwf add`; *lazy* claims before the first push. | Git's compare-and-swap on the ledger |
| C | Ids unique by construction, from a namespace per clone or at random. | nothing; none is needed |
| D | The merge request's number in the id. | the host's numbering |
| E | [ADR-0001](../adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md): provisional entities in an inbox, numbered by a mint on the default branch after the merge. | the mint's push |

## 3. Results by host setting

Each cell names the properties that failed in the model, and a dash means all five held within its bounds. In the table:

- **h:** the numeric id is stable, but a provisional name that was already pushed gets replaced.
- **refused:** some runs end refused, with a remedy a person can apply.

Each cell combines four conditions, where they apply: the base case, one clone offline, one lost push reply, and one cross-branch citation (with two clones and with three). §4 adds the conditions that every design shares.

| Design | Direct to the default branch | Merge requests, no up-to-date rule | Merge requests with the up-to-date rule |
|---|---|---|---|
| S0 (today) | I2 | I1, I2, I3, I4, I5 | I2, I3, I4 |
| S0b | I2 | I1, I2, I3, I4, I5 | I2, I3, I4 |
| A | I2 | not applicable: A is that rule | I2, I3, I4 |
| A+ | — | not applicable | I3, I4; I2 when a third clone cites |
| B, eager or lazy | — | — | — |
| C, namespace or random | — | — | — |
| D | not applicable | h | h |
| E with unique handles | I2: in direct mode E is S0 | h | h |
| E with title-slug handles | I2 | I2, h, refused | I2, h, refused |

Some failures need explaining:

- **S0b and A fail I2 even in direct mode.** A clone pulls, cites the other clone's entity while its own entity with the same id is still in its tree, and then renumbers its own, carrying the citation along. A+'s first guard closes this case.
- **A+ fails I2 when a third clone cites.** C's entity merges first, A's entity is renumbered, and B's citation of A's old id now names C's entity. No rewrite rule can fix it, because the citation is on B's branch.

What the experiments add, design by design:

- **S0.**
  - When the other clone's id had been pushed and fetched, or sat on a branch in the same clone, the allocator skipped it. Otherwise both clones got the same id.
  - A host merge without a check landed the duplicate under every merge method tried: `--no-ff`, rebase then fast-forward, and rebase then `--no-ff`.
  - A maintainer who merged locally with the hooks installed had the push to `main` refused.
  - References stayed correct only when the losing branch renumbered before merging the default branch in. Reallocating after the merge or rebase that the trunk-collision hint leads to misdirected two of four references, and the push passed.
- **A.**
  - Three concurrent merge requests each collided on a gap, an epic and a milestone, and no duplicate in a different file landed. That took 6 merge attempts.
  - It took 6 reallocations when each losing clone could see the previous one's renumbered branch, and 9 when they updated from the same fetch.
  - When the losers followed the hint's order instead (merge the default branch in, then reallocate), two of them rewrote the winner's referrers differently. The third merge request then hit a content conflict and never merged.
- **B.**
  - In 8 races of three hooked clones, every filing kept the id it was pushed with, and no duplicate reached the default branch.
  - After a lost reply, reading the ledger back for the clone's own nonce recovered the claim. The naive recoveries burned an id, or landed a duplicate.
  - A clone running plain `aiwf add` without claiming landed a duplicate.
- **C.**
  - Namespaced and random shapes such as `G-pb-0012` and `G-7f3k` load as nothing. The only finding is an `unexpected-tree-file` warning, so the file passes every hook and is invisible to every verb.
  - Wider numbers load, and one large id moves the counter: after `G-123456789` the next `aiwf add gap` gives `G-123456790`.
- **D.**
  - A second entity of the same kind in one merge request has no legal shape. `G-1234.2` is invisible, and `G-1234-2` raises the `id-path-consistent` error.
  - No verb renumbers to a chosen id, and the pre-push hook refuses a renumber done by hand as an untrailered entity commit.
  - **The MR-number ids share one number space with ordinary allocation.** After `G-0057` lands from merge request 57, `aiwf add gap` hands out `G-0058`, the number merge request 58 would use.
- **E.**
  - The mint needs a bot identity that may push to the protected branch, and it must retry after a non-fast-forward rejection.
  - The commit-msg hook refuses `assign-pending` as an unknown verb, and `aiwf check` refuses a bot actor without an `aiwf-principal` trailer.
  - Between the merge and the mint, the default branch holds entities that have no id, and a reference by provisional name is refused.

## 4. What hits every design, or every design that renumbers

- **An abandoned, cited branch leaves a dangling reference.** A branch cites an entity on another branch that isn't merged yet. aiwf reports that as a cross-branch-pending warning, not an error, so the citing branch merges. Then the cited branch is abandoned. Every design ends with a reference to nothing on the default branch. This comes from the model and was not measured.
- **Pruning frees ids.** This affects S0, S0b, A and A+. Alpha pushed `G-0002` on a branch, beta fetched it, alpha deleted the branch, and beta ran `git fetch --prune` and then `aiwf add gap`. Beta got `G-0002`. The id comes back only while no higher id of its kind is visible.
  - ADR-0025 accepts that "an id consumed by a remote branch that is later abandoned is burned permanently". That holds while the abandoned branch stays on the remote and in clones' remote-tracking refs, and stops holding once it is deleted and pruned.
  - A citation written in the meantime, say in a merge request comment, a Jira ticket or another branch, then names the new entity.
  - B keeps its claims on the ledger, so an abandoned id there is a hole and is never reused.
- **Same id, same file.** Two clones that allocate the same id for entities with the same title produce the same path. This affects S0, S0b, A and A+, every design that lets two clones pick the same id.
  - With identical content, Git merges the two into one file without a conflict. Two `aiwf add epic --title "Q3 planning"` runs ended as one epic on the default branch, with two `add` commits in `aiwf history` and 0 errors from `aiwf check`.
  - With different content, Git stops at an add/add conflict.
  - aiwf's pre-push passes in both cases, because `DisputedTrunkIDs` (`internal/check/check.go:288`) skips an id at the same path. A's host rule lets the identical case through.

## 5. What each design requires and costs

| Design | Requires | Code change | Offline | Costs | Who renumbers after a host merge |
|---|---|---|---|---|---|
| S0 | nothing | none | allocates against a stale view | duplicates land when merge requests are used | nobody, by renumbering one copy through a hooked push (§6, item 2) |
| A | the host's up-to-date rule or a merge train; a required pipeline running `aiwf check` with the default branch fetched ([G-0536](../../work/gaps/G-0536-aiwf-check-has-no-ci-backstop-the-primary-oracle-is-per-clone-opt-in.md) is that missing pipeline) | a scoped reallocate (G-0745); A+'s two guards | as S0 | ids change during review; merge attempts and rebases grow with concurrent merge requests; citations on other branches go stale | the losing merge request's author, on the branch, before merging the default branch in |
| B | a ledger branch that only moves by non-force pushes, best protected by a host rule against force pushes; network at claim time; every claim confirmed by reading back the clone's own nonce; every clone taking part | none for allocate, claim and reallocate; a check that refuses an entity without a matching claim. Claim pushes need `--no-verify` today, because the pre-push hook refuses a ref that isn't checked out | eager: no allocation offline; lazy: the claim waits until before the first push | one push per allocation; holes from abandoned claims; reverses an exclusion in CLAUDE.md | nobody: claimed ids do not collide |
| C | a namespace per clone, itself assigned without repeats, or a random token space large enough | the id grammar, kernel commitment 2: the regexes and numeric parsing a grep for the id patterns finds, unconfirmed by a build | fully | ids are no longer short sequential numbers; ADR-0001 rejected flat hash ids for that reason | nobody |
| D | a merge-request workflow and the host's numbering | a shape for a second entity per request; a verb that renumbers to a chosen id; a separate number space from ordinary allocation | until the merge request opens | cannot serve planning pushed straight to the default branch ([D-0073](../../work/decisions/D-0073-planning-happens-on-main-implementation-work-happens-in-a-worktree.md)) | the host, by numbering |
| E | a bot that may push to the protected branch; a retry loop; a human principal on every mint | the mint verb, inbox loading, rewriting late references | fully, on branches | id-less entities on the default branch until the mint; provisional names always replaced; ADR-0001 is proposed and deferred by [D-0037](../../work/decisions/D-0037-defer-adr-0001-g-0281-and-emb-pending-a-measured-id-collision-trigger.md) | a bot, after the merge |

The ledger must be a branch. Locally, `receive.denyNonFastForwards` refused a forced rewind of `refs/heads/aiwf/ids` but accepted one of `refs/aiwf/ids`. The allocator sees a ledger only under `refs/heads/*` or `refs/remotes/*`, and a default fetch brings a branch but not `refs/aiwf/*`.

**Fit with transactions.** Submit runs before a transaction's first push, so it is a natural claim point for B's lazy variant. If a claim is refused, the transaction reallocates on its branch while its tree holds only its own copy, which the experiments showed keeps references correct. If the default branch has been merged into the transaction and holds the entity that took the id, the tree holds both, and the reallocate has to be scoped (S0b). None of this was run.

## 6. Where the model and the measurements disagree

1. **Identical files.** The model assumes that two branches adding the same path always conflict, but Git merges identical additions silently. So the model's result that A always keeps ids unique is wrong for two entities with the same id, title and content. §4 has the measurement.
2. **Repairing a landed duplicate.** The model recognises a renumber by the entity's identity, so a repair passes its checks, and two repairs of one duplicate can race forever. aiwf exempts a disputed trunk id only when its trunk path was renamed to the branch's path for that same id (`internal/check/check.go:387`). A renumber moves the other copy to a new id, so the copy that kept the id is still reported as colliding. Renumbering one copy, the second clone's gap and epic, was refused at a hooked push.
   - **On the default branch:** the push of `main` was refused.
   - **On a repair branch:** the push failed with `error ids-unique/trunk-collision: id "G-0002" is allocated on this branch (work/gaps/G-0002-alpha-gap.md) and on trunk (work/gaps/G-0002-beta-gap.md) for different entities`.

   Renumbering both copies passed, provided the second clone's copy went first; the verb itself refused the other order. Otherwise the repair needs `--no-verify`. The model's race needs one of those routes, and it was not reproduced in aiwf.
3. **D's number space.** The model gives MR-number ids a space of their own. aiwf's allocator counts them as ordinary ids (§3, D).

## 7. The id-lifecycle initiative's protocol

The comparison is with §"Protocol specification" of the [id-lifecycle initiative](../initiatives/id-lifecycle.md).

**Confirmed.**

- A confirmation gated by compare-and-swap gives uniqueness at every instant. B held I1 everywhere, and a force-pushed claim broke it in 8 steps.
- Re-deriving ground truth after a lost reply is necessary, and it has to identify the claimant, not just the id. Trusting the lost reply, or finding the id claimed and taking the claim as one's own, both landed duplicates.
- The batch confirm needs its compare-and-swap guard whenever two mint triggers can mint different subsets.

**Wrong.**

- `ReferentialIntegrityUnderRenumber` ("no `refs` set anywhere contains `old`") is the property that the misdirecting rewrite satisfies. `RenumberOnCollision` as written ("replace `old` with `new` everywhere it appears") is that rewrite. The property needed is I2, and stating it takes the meant identity of each reference, which a set of ids cannot express.
- The instantiation table marks E-0052 "Eventual" and leaves out the host setting. With the up-to-date rule and a pipeline, or with direct pushes and the hook, S0 is unique on the default branch at every instant.
- `Convergence` overclaims when two repairs can be in flight for one duplicate (§6, item 2).
- `NoPrematureReference` does not prevent misdirection. A citation can be ambiguous when written, and a citation of a pushed entity that isn't merged yet goes stale when that entity is renumbered.

**Missing.**

- There are no stability (I4) or reuse (I3) properties.
- Host merge semantics are absent: stale pipeline verdicts, the up-to-date rule, and hosted merges that skip local hooks.
- E's window between merge and mint, and its late references.
- Abandoned branches.

## 8. Defects these measurements found

1. [G-0745](../../work/gaps/G-0745-aiwf-reallocate-rewrites-references-meant-for-the-other-colliding-entity.md): `aiwf reallocate` rewrites references meant for the entity that keeps the id. When several branches lose, their rewrites of the winner's referrers also conflict.
2. [G-0746](../../work/gaps/G-0746-renumbering-one-copy-of-a-duplicate-id-on-the-trunk-is-refused-at-pre-push.md): renumbering one copy of a landed duplicate is refused at pre-push (§6, item 2).
3. [G-0747](../../work/gaps/G-0747-two-same-id-entities-at-the-same-path-merge-into-one-file-without-any-finding.md): a same-id, same-path collision is never reported (§4). `DisputedTrunkIDs` skips an id at the same path, and Git merges identical additions.
4. [G-0748](../../work/gaps/G-0748-aiwf-history-lists-the-other-entity-s-commits-after-a-collision-is-reallocated.md): after a reallocate, `aiwf history` lists the other entity's commits for both entities, even when the repair was done in the right order. Archived [G-0037](../../work/gaps/archive/G-0037-cross-branch-id-collisions-split-the-audit-trail-allocator-is-local-tree-only.md) named this symptom and is marked addressed.
5. [G-0749](../../work/gaps/G-0749-a-deleted-and-pruned-branch-s-ids-are-allocated-again.md): a deleted and pruned branch's ids are allocated again (§4).
6. [G-0750](../../work/gaps/G-0750-aiwf-list-reports-the-checked-out-branch-as-a-cross-branch-ref.md): `aiwf list` reports the checked-out branch as a cross-branch ref, and lists an entity twice when its frontmatter id and path id differ.

## 9. Not settled

Each item needs a real host, or a build, to settle:

| Question | What would settle it |
|---|---|
| Do GitLab's merge trains and "fast-forward merge" setting, and GitHub's "require branches to be up to date" and merge queue, enforce what design A assumes? | a test project on each host, with a scripted pair of merge requests |
| Does a host re-run a pipeline when the target branch moves? | open two merge requests, merge one, and watch the other's status |
| What do squash merges do? They would break the reallocate tiebreaker, which goes by ancestry on the default branch, and they drop trailers (§5 of the main document). | squash-merge on the host, then `git merge-base --is-ancestor <add-sha> origin/main` |
| Does each host accept a ledger branch and protect it from force pushes? | branch protection on `aiwf/ids` on each host |
| Can a merge request be opened before its branch differs from the target? D depends on it. | try it on each host |
| Can a bot push to a protected branch, as E needs? | branch protection with a bypass actor |
| Does reading back after a lost reply work over HTTPS and SSH? | cut the connection after the server applies the update |
| How do retries and latency grow with tens of clients? | a load run against a hosted repository |
| Does CI fetch every branch, as cross-branch reference resolution assumes? | inspect the refs in a pipeline checkout |
| What does design C's grammar change touch? | build a kernel with the new grammar and run its tests |

## 10. Recommendation

The choice follows from two of the main document's open choices: which integration modes must work, and whether an id may change after its branch is pushed.

- **Direct to the default branch only:** today's allocator suffices once reallocate is scoped and the ambiguous-citation guard exists (A+), and the same-path case is detected.
- **Merge requests, with ids allowed to change during review:** A is the cheapest. It needs a host setting and a pipeline, and the team accepts that citations on other branches go stale. Under R9 aiwf would also have to name the up-to-date setting and detect when it is off.
- **Both modes, with ids fixed once pushed:** B or C.
  - B keeps today's id shape and works with today's allocator. It needs a ledger branch, a check that every entity has a claim, network before the first push, and the reversal of an explicit exclusion.
  - C needs no host and no network, but changes the id grammar and makes ids long or prefixed.
- **D and E** carry more than they give here:
  - D cannot serve planning pushed straight to the default branch, and its ids collide with ordinary allocation.
  - E needs a bot with bypass rights and a new verb, and leaves id-less entities on the default branch.

The requirements in the main document ask for both modes with neither preferred (R6), and for no guarantee resting on a host setting aiwf cannot detect (R9). Kernel commitment 2 asks for stable ids. By those requirements B, with lazy claims at submit, is the candidate. It is the only design that kept every property in both modes without changing the id shape.

## Provenance

- **Run:** 2026-10-25, on Linux in a devcontainer, with git 2.51.1, aiwf built from `1e8e17e6c` (it reports `v0.42.0+dirty`), and Python 3.12.3.
- **Experiments:** every remote was a bare repository on the local transport, and host behaviour was simulated by scripts. No GitLab or GitHub behaviour was measured.
- **Model:** a breadth-first explicit-state checker over 186 runs and 160 million states, with bounds of at most 3 clones, 3 entities, 2 citations, 1 fault, 1 abandoned branch and 3 renumbers.
  - 12 three-clone runs hit the state cap; their violations are real, and their passes cover only shorter traces.
  - It did not model rebases, squash merges, several clones on one branch, or more than one entity kind.
- **Code:** the model and its results are in [`id-allocation/model/`](id-allocation/model/RESULTS.md), and the experiment scripts and their results in [`id-allocation/experiments/`](id-allocation/experiments/RESULTS.md). Each results file says how to run it. Traces, outputs and the disposable repositories are written outside the tree and not committed.
