# Id allocation under concurrent work: model-checking results

An explicit-state model checker (`model.py`, Python 3, 957 lines) that compares aiwf id-allocation designs. Every entity carries a hidden identity, independent of its id, and every reference records the identity its author meant, so a reference that resolves to the wrong entity is a checked violation. The model explores every interleaving of clones, host and faults within the bounds below, using breadth-first search, so each trace is a shortest one.

Run on 2026-10-25, Python 3.12.3, Linux, 16 cores. There were <!-- NRUNS --> runs, with <!-- NSTATES --> million states in total. <!-- NCOMPLETE --> runs completed their bound and <!-- TRUNC --> were cut off at a state cap (marked `*`, see §5).

## Headline

1. **With merge requests, only designs that never renumber a published id keep references correct.** Those are B (a ledger ref), C (collision-free ids), D (MR-number ids) and E (ids minted at integration).
   - With merge requests, every renumbering design (S0, S0b, A, A+) changes an id after it was pushed, whenever a branch is published before its collision is visible (I4).
   - Once another entity holds the old id, a citation of that id on a third branch resolves to the wrong entity (I2; `A+@mrutd+xcite[3c]__I2_stale.txt`). No rewrite rule can fix this, because the citation is on someone else's branch.
   - In direct-to-default-branch mode, nothing is published before it reaches main. There the renumbering designs keep I3 and I4, and A+ holds every invariant.
2. **Today's design (S0) depends on the host setting.**
   - In direct-to-default-branch mode, the non-force push plus the pre-push hook serialise allocation. Ids stay unique on main and never change after publication. The only defect is G-0745's misdirecting rewrite (I2, `S0@direct+base[2c]__I2.txt`).
   - With merge requests and no up-to-date rule, the host merges a branch whose pipeline passed against an older main, and a duplicate lands on main (`S0@mr+base[2c]__I1.txt`, 6 steps).
   - Two repairs of the same duplicate can then race each other indefinitely (I5 livelock).
3. **S0b's scoped rewrite fixes both cases G-0745 reports, but not a third.** After a pull, a clone holds both entities and cites the other one on its own branch (`S0b@direct+base[2c]__I2.txt`). A+ adds one guard: refuse a citation whose id is ambiguous in the clone's view. That closes the third case.
4. **The up-to-date rule makes the default branch a compare-and-swap ledger for uniqueness (design A).** I1 always holds, and I5 holds; the only refusals are under abandon, and each has a human remedy. But ids still change after push (I4), and one id is published for two entities (I3).
5. **B holds every invariant under every condition, including a lost push reply and an offline clone**, apart from the abandon problem that every design shares (item 7). Two conditions apply. Claims must be non-force pushes (probe P1). Reconciling a lost reply must look for the clone's own claim, not just the id (P2 and P3).
6. **D and E keep numeric ids stable but replace a published provisional name ("h" in the matrix).**
   - D does this only when the branch is pushed before its MR is opened.
   - E always does it, by design.
   - E also needs the mint to follow references written against a provisional name minted earlier (P12).
   - E with title-derived slugs repeats the misdirection hazard at the slug level (I2).
7. **Two problems hit every design.**
   - A citation can end up dangling on main (abandon column, ◐; T8). Its pipeline passed while the cited branch existed, since the cross-branch-pending warning does not block. The cited branch is then abandoned, and the host merges on the stale verdict.
   - Under every renumbering design with merge requests (S0, S0b, A, A+), abandoning a branch frees its ids for reuse once clones prune (`A@mrutd+abandon[2c]__I3gone.txt`, 5 steps). A citation of the freed id then resolves to the new owner (T6).

## 1. Matrix

Each cell lists **I1 I2 I3 I4 I5**:

- ✓ means the invariant holds over the explored bound.
- ✗ means violated.
- ◐ in the I1 column means a duplicate reaches main transiently, but no terminal state keeps it.
- ◐ in the I2 column means no misdirection, but some terminal state has a dangling reference on main.
- `g` in the I3 column means an id is reused after the earlier binding vanished from every ref.
- `h` in the I4 column means numeric ids are stable, but a published provisional name is replaced.
- R in the I5 column means some traces end refused, with a human remedy (an unresolved reference, or a git conflict).
- `*` means the run hit the state cap. Its violations are real, but a ✓ holds only for traces shorter than the depth listed in §5.
- In the probe table, `I1ev` and `I2ev` mean a terminal state still has a duplicate, or a dangling or misdirected reference, on main. `I2:<cause>` names how a reference was misdirected: `rewrite` (a renumber rewrote it), `stale` (its target was renumbered) or `taken` (its target is not on main and another entity holds the id). `I5livelock` means the renumber bound was hit.

The trace for any ✗ is `traces/<design>@<mode>+<condition>[<bound>]__<invariant>.txt`, where the bound is `2c` or `3c`. The I2 traces are also split by cause into `I2_rewrite`, `I2_stale` and `I2_taken`. Terminal-state traces end in `-terminal.txt`. Every trace is a shortest trace: one step per line, with the full state after each step.

**Conditions:**

- **base**: 2 clones; clone A has 2 entities and B has 1; each clone may write 1 citation, in the commit that creates an entity.
- **offline**: as base, with clone B offline until it reconnects.
- **lost**: as base, with one lost push reply (applied or not applied) on any push, claim, MR open or mint.
- **xcite**: one cross-branch citation of an entity on another clone's unmerged branch. Runs at 2 clones with 1 entity each, and at 3 clones where B only cites while A and C allocate.
- **abandon**: xcite, plus one branch deletion (the MR is closed and fetches prune).

**Modes:**

- **direct**: no merge requests; non-force pushes to main.
- **mr**: protected main, merge requests, a pipeline running `aiwf check` on the merge result, no up-to-date rule.
- **mrutd**: as mr, plus the up-to-date rule.

<!-- TABLES -->

**Notes on the matrix:**

- S0, S0b and A **in direct mode** fail I2 by the unscoped rewrite or by the same-branch citation. Nothing is published before it reaches main, which is why I3 and I4 hold there.
- S0 and S0b **in mr** fail I1 because pipeline verdicts go stale. Every terminal state that keeps a duplicate on main also exhausted the renumber bound of 3 (checked by `check_dup_terminal.py`: 812 of 56,666 terminal states at the base bound, all with 3 renumbers). So the duplicate persists because of the livelock, not because no repair verb exists.
- A+ **in mrutd** fails I2 only when a third clone's citation goes stale or taken, or when an id is freed by an abandon. It fails I3 because two clones can publish one id before either sees the other. In direct mode A+ holds everything.
- **P6** (A without the pipeline) still holds I1, because under the up-to-date rule the client pre-push hook sees the merged tree. Breaking I1 takes both checks gone (P6b).
- **P8** (two mint triggers, no compare-and-swap, each minting every pending entity in a fixed order) holds I1. Main only grows, so the race produces identical renames or a git rename/rename conflict, never a duplicate. A trigger that mints only its own merge does produce duplicates without compare-and-swap (P8b) and none with it (P8c).

**Key traces**, all in `traces/`:

| id | file | what it shows |
|---|---|---|
| T1 | `S0@mrutd+base[2c]__I2.txt` | G-0745: reallocate after the pull rewrites B's reference onto A's renumbered gap |
| T2 | `S0b@direct+base[2c]__I2.txt` | scoped rewrite: A pulls, cites B's G-1 while its own G-1 is still there, then renumbers and carries the citation along |
| T3 | `S0@mr+base[2c]__I1.txt` | no up-to-date rule: two pipelines pass against an empty main and both merges land the same id |
| T4 | `S0@mr+base[2c]__I5livelock-terminal.txt` | B renumbers its gap on its branch, but the host merges B's earlier push (its pipeline ran before A merged); A repairs the duplicate on main; both repairs land and the duplicate moves to the next id, repeatedly |
| T5 | `A@mrutd+base[2c]__I4.txt` | A: B's pushed G-1 becomes G-2 (8 steps) |
| T6 | `A+@mrutd+abandon[2c]__I2_taken.txt` | A abandons; B's citation of A's G-1 then resolves to B's own new G-1 |
| T7 | `A+@mrutd+xcite[3c]__I2_stale.txt` | C's G-1 merges first, A's cited G-1 becomes G-2, and B's citation now names C's entity |
| T8 | `Blazy@mrutd+abandon[2c]__I2ev-terminal.txt` | every design: the citing branch merges, then the cited branch is abandoned, leaving a dangling reference |
| T9 | `D@mrutd+base[2c]__I4h.txt` | D: branch pushed before the MR opened; the handle is replaced by `!1.1` |
| T10 | `Eslug@mrutd+base[2c]__I2.txt` | E-slug: a late `~t1` reference meant for a1 resolves to b1, which reused the slug after a1 was minted |

## 2. Per design

Renumbers count renumber and slug-rename commits per explored terminal state. The share is taken over terminal states of the explored graph, which counts every interleaving once; it is not a rate.

| design | requires | offline | stability (I4) | renumbers in explored traces | works when |
|---|---|---|---|---|---|
| **S0** (today) | nothing beyond git and the pre-push hook | allocates against a stale view; collisions surface at push or merge | holds in direct mode; violated with MRs, ids change after push | direct ≤2 (74% of terminals); mrutd ≤2 (79%); mr hits the bound of 3 (99%) | only uniqueness, and only in direct mode or with the up-to-date rule; references never safe across a renumber (I2 ✗ everywhere) |
| **S0b** | as S0, plus the ability to tell which branch introduced a reference line | as S0 | as S0 | as S0 (direct ≤2, 66%; mr hits 3) | as S0; fixes G-0745's cases, not the same-branch citation (T2) |
| **A** = S0b + up-to-date rule + required pipeline | host: required status check running `aiwf check` on the merge result, and "branch must be up to date" (or a merge train); in direct mode the push itself is the compare-and-swap, plus the hook | as S0 | violated in MR mode (T5); holds in direct mode | ≤2 (71% of terminals in mrutd) | I1 always; I5 holds, with refusals carrying a human remedy under abandon; I2 fails everywhere (T2); I3 and I4 fail with MRs |
| **A+** (refined: A + refuse ambiguous citations + check collisions against every visible ref before first publication) | as A | as A | violated with MRs: two clones that push before seeing each other still collide; holds in direct mode | ≤2 | direct mode: everything; with MRs: I1, I2 and I5 for collisions between two branches, but I2 fails for a citation on a third branch (T7) and after an abandon (T6) |
| **B eager / B lazy** | a shared ledger ref (e.g. `refs/aiwf/ids`) that every clone can push to without force; network at claim time; the ledger seeded with existing ids (not modelled) | the entity keeps a provisional handle until the claim, and the pre-push hook refuses unclaimed entities, so handles never leave the clone | holds in every run | 0 | always within the bound; eager and lazy did not differ on any invariant |
| **C namespace** | a unique namespace per clone (assignment is itself a coordination step: P10) | fully offline | holds | 0 | always, provided namespaces never repeat; ids are no longer short sequential numbers |
| **C random** | a token space large enough that collisions are negligible (P11 shows a small space failing I1) | fully offline | holds | 0 | always, under that probability assumption; long ids (ADR-0001 rejected them for this) |
| **D** (MR number in the id) | an MR workflow (not available in direct mode, nor for planning on main per D-0073); the host's unique MR numbers; network to open the MR; an MR openable before the entities are pushed | handles until the MR opens | numeric ids hold; a handle pushed before the MR opened is replaced (h) | 0 | always for ids; references to an old handle need the follow-renamed-handle step (P13) |
| **E unique handle** (ADR-0001) | a mint trigger with push rights to the protected main; minting every pending entity, or compare-and-swap on the mint push (P8b); the mint must also rewrite late handle references | fully offline on branches | numeric ids hold; the published handle is always replaced at integration (h, by design) | 0 in MR modes; in direct mode E is S0 (ADR-0001 mints immediately on trunk) | MR modes only; direct mode inherits S0's I2 |
| **E slug handle** | as E | as E | as E, plus slug renames after publication | ≤1 | I2 fails: slugs collide, get reused after minting, and slug renames rewrite references (T10) |

## 3. Assumptions the model encodes

Each assumption needs a real-git or real-host test before a decision rests on it. Each line gives the assumption, then the test that would confirm it.

**Git**

- **G1.** A non-force push is an atomic compare-and-swap. It succeeds only if the remote tip is an ancestor of the pushed commit, and concurrent pushes to one ref are serialised. Test: N clones race `git push` to one bare repository; exactly one wins and the rest get non-fast-forward.
- **G2.** After a lost push reply, the ref is either updated or not, and the next fetch shows which. Test: cut the connection after `receive-pack` applies an update, then fetch.
- **G3.** Merging follows aiwf entity files correctly:
  - a renumber is a file rename that merges cleanly against an untouched side;
  - two different renames of one file conflict (rename/rename);
  - two branches adding the same path conflict (add/add);
  - edits to different reference lines merge cleanly;
  - two different edits of one reference line conflict.

  Test: script each case with aiwf-shaped files.
- **G4.** Integration is by true merges, so "add commit is an ancestor of trunk" is meaningful. Squash or rebase merges, often a host's default, break the reallocate tiebreaker. Test: squash-merge on the host, then `git merge-base --is-ancestor <add-sha> origin/main`.
- **G5.** Fetch prunes deleted remote branches (`fetch.prune`). Without pruning, an abandoned id stays visible only to clones that fetched it before deletion, and never to a fresh clone. Test: delete a branch remotely, then fetch with and without prune.
- **G6.** Clones and CI fetch every branch, which cross-branch reference resolution relies on. ADR-0041 records that CI often does not (G-0536). Test: inspect the refs in CI's checkout.

**Host**

- **H1.** The host serialises merges into the default branch. Test: concurrent merge API calls.
- **H2.** The up-to-date rule refuses a merge unless the default branch is an ancestor of the MR head. Test: GitHub "require branches to be up to date", GitLab fast-forward merges or merge trains.
- **H3.** The pipeline checks the merge result as of when it ran, and is not re-run when main moves. This stale verdict is what lets duplicates and dangling references land. Test: open two MRs, merge one, then check whether the other's status is re-run.
- **H4.** A pipeline verdict is tied to the head commit; a new push re-runs it.
- **H5 (D).** MR numbers are unique and never reused, and an MR can be opened before the branch carries the entities. Test: open an MR on a branch identical to main. If the host refuses one, every D entity is published under a handle first. I believe GitHub refuses a pull request with no diff, but I did not check it.
- **H6 (E).** The mint job may push to the protected main, and its push is a non-force push. Test: branch protection with a bypass actor.

**aiwf**

- **A1.** Allocation is max+1 over the current ids in the working tree, local branches and remote-tracking refs (ADR-0025). Prior ids are not counted.
- **A2.** `ids-unique` reports two kinds of finding:
  - a duplicate within one tree;
  - a trunk collision: the same id at a different path on trunk.

  The trunk collision fires even when the branch's own entity is also on trunk, because `DisputedTrunkIDs` compares each trunk entry against the branch's entity with that id. The model reproduces this. A branch that keeps working after its merge can therefore be told to renumber an entity that is already on trunk. Rename recognition is assumed exact (ADR-0031). Test: merge a branch with a stale pipeline verdict so that main carries a duplicate, then run `aiwf check` on the other, already-merged branch.
- **A3.** `refs-resolve` checks the current id only. A hit on a remote-tracking ref is a warning, not an error (ADR-0041).
- **A4.** Reallocate breaks ties by trunk ancestry; when both or neither entity is in trunk, a human picks, which the model explores as a nondeterministic choice. The new id comes from the allocator's view.
  - S0 rewrites every reference in the tree, matching `rewriteReferences`.
  - S0b rewrites the reference lines introduced on the renumbered entity's branch. This assumes that branch is recoverable from history.
- **A5.** The pre-push hook is installed, runs against the last-fetched trunk, and does not fetch.
- **A6.** Each clone works on one branch. A citation is written in the commit that creates an entity, plus at most one cross-branch citation. A clone whose branch is fully merged acts only as the designated repairer of a duplicate on main.
- **A7 (E-slug).** `aiwf add` refuses a slug whose inbox path exists in the clone's own tree, but accepts one already carried by a minted entity. ADR-0001 checks slug uniqueness only in `aiwf check`, where it is a warning on a branch.

**Model abstractions**

- There is one entity kind. Id sequences are per kind and independent, so this loses nothing.
- At most 3 renumbers are allowed per trace; hitting the bound is reported as a livelock.
- B's reconcile, D's MR query and E's minting each run as one atomic step. E's mint is split into fetch and compare-and-swap push only when 2 mint triggers race.

## 4. The id-lifecycle initiative's protocol

The comparison is against `docs/initiatives/id-lifecycle.md`, §"Protocol specification".

**Confirmed**

- **CAS-gated confirmation gives uniqueness at every instant.** B holds I1 in every run, and the force-push probe P1 breaks it in 8 steps.
- **Stage 4, "re-derive ground truth… against the exact object this operator built", is necessary**, and it must identify the claimant, not just the id. P3 reconciles by id: the clone finds "G-1 claimed" (B's claim) and keeps it, and the result is a duplicate in 9 steps. P2 trusts the lost reply. P4, which never reconciles, is stuck (I5).
- **The batch confirm (E) keeps its compare-and-swap guard, as the protocol says it must.** The guard is necessary when two mint triggers can mint different subsets: P8b fails and P8c holds. When every trigger mints all pending entities in a fixed order, a missing guard yields a git rename/rename conflict rather than a duplicate (P8, which rests on assumption G3).
- **The instantaneous/eventual uniqueness split is real.** S0 provides only eventual uniqueness, and only when no livelock occurs, on a host without the up-to-date rule.

**Wrong**

- **`ReferentialIntegrityUnderRenumber` ("no `refs` set anywhere contains `old`") is the property the misdirecting rewrite satisfies.** `RenumberOnCollision` as specified ("replace `old` with `new` everywhere it appears… in every session") is the unscoped rewrite. It fails I2 (T1, and probe P5), as G-0745 found. The property needed is "every reference resolves to the identity its author meant", and stating it needs the hidden meant-identity per reference that this model adds. Recording a reference as an id, as `refs` does, cannot express misdirection.
- **The instantiation table's "E-0052: Eventual" leaves out the host setting.** With the up-to-date rule and a pipeline, or with direct non-force pushes and the hook, S0 is unique on main at every instant: the default branch is itself the compare-and-swap ref. Without the rule, it is not even reliably eventual (livelock).
- **`Convergence` "provided every collision is resolvable by the automatic tiebreaker" overclaims.** In T4 each repair is locally correct, and the tiebreaker is applied where it can decide. Still, two repairs of one duplicate both land: B's, made on its branch through the trunk-collision path, and A's, made on main. That recreates the duplicate at the next id, without bound. Convergence needs at most one repair in flight per duplicate, which no mechanism in S0 provides.
- **`NoPrematureReference` (reference only what is in `materialized[s]`) does not prevent misdirection.** A reference is resolved in the reader's tree, not the writer's:
  - a citation can be born ambiguous (T2's same-branch citation, or a cited id also bound locally);
  - a reference to a published but not yet merged entity goes stale when that entity is renumbered (T7).

**Incomplete**

- **There is no stability property**, although stable ids are the kernel's promise. I4 separates the designs more sharply than uniqueness does.
- **There is no reuse property.** ADR-0025's "burned permanently" holds only while the remote branch exists and is fetched. After deletion and pruning, S0, S0b, A and A+ reuse the id (`g` in I3, T6). In S0 there is no `confirmed` set that "never loses an element", only visibility.
- **Host merge semantics are absent:** stale pipeline verdicts, the up-to-date rule, and hosted merges that bypass local hooks. These decide S0's and A's results. A host merge without the up-to-date rule is a batch confirm without the compare-and-swap guard.
- **E's window between the host merge and the mint is not modelled by the protocol.** During it, main carries provisional entities.
- **E's late references are not covered.** A branch may cite a handle that another merge has already minted. ADR-0001 rewrites references only at mint time, and needs the mint record at later mints (P12 leaves the reference dangling).
- **The slug collision domain**, which the initiative's scoping note sets aside, shows the same misdirection hazard as numeric ids (E-slug, I2 ✗).
- **Abandoned branches create dangling references under every design** (T8). Nothing in the protocol removes a reference or an entity.
- **`ResyncBurden`** was not measured by this model.

## 5. How to run, and limits

Run from this directory. The outputs `results.json`, `traces/` and `tables.md` are written here, ignored by git and not committed, so the trace files named above exist only after a run:

```
python3 -I run.py --jobs 8 --max-states 5000000 --time-limit 1200   # whole matrix and probes: 9,368 CPU-seconds, about 20 min with 8 workers; about 27 GB of RAM in use at peak
python3 -I run.py --only "S0@mrutd" --jobs 4                         # a subset by run-id substring
python3 -I make_tables.py                                            # results.json to tables.md
python3 -I check_dup_terminal.py "$PWD"                              # the I1-vs-livelock check in §1
```

`run.py` writes `results.json` and `traces/`; `tables.md` holds the matrix, the probes and per-run statistics (states, completeness, BFS depth, renumbers).

**Limits**

- **Small bounds.** At most 3 clones, 3 entities in total, 2 citations, 1 fault, 1 abandon, 3 renumbers. Bugs that need more actors or longer chains, such as G-0274's batch reallocate, are out of reach.
- **Truncated runs.** <!-- TRUNC --> runs hit the state cap, all of them three-clone xcite or abandon runs: <!-- TRUNCLIST -->.

  Their two-clone counterparts completed. A ✓ in those cells holds only for traces shorter than the depth in brackets, all of which were explored.
- **Not modelled.** Rebases, squash merges (G4), several clones sharing a branch or an MR, linked worktrees sharing local refs, more than one entity kind, the content of entity bodies other than id references, and the cost metrics (`ResyncBurden`, push latency). Every run that built the state graph was acyclic. The one run re-run without the graph to save memory, `D@mrutd+xcite[3c]` at 18 million states, has no cycle check.
- **Simplified behaviour.** The model's "human" picks the reallocate target nondeterministically; real humans might notice misdirection, which the model never does. Refusals with a human remedy (R) are counted as acceptable outcomes for I5.
