# Transactions and planning commits: model-checking results

An explicit-state model checker (`model.py`, Python 3, standard library) for the transaction fold proposed in [Transactions and planning commits](../../collaborative-state-and-commit-granularity.md) (§4, §9.2). Every entity value records the hidden identity of the operation that wrote it, and every operation's trailer group carries the operation's hidden identity, so "this operation landed", "this record survived" and "this record names the commit its author meant" are checked properties, not inferences. Breadth-first search, so every trace is a shortest one.

Run <!-- WHEN -->, Python 3.12.3, git 2.51.1, Linux, 16 CPUs, 12 workers. <!-- NROWS --> (host setting, condition, design) results came from <!-- NRUNS --> distinct explorations with <!-- NSTATES --> states in total. <!-- NTRUNC --> runs hit the state cap and <!-- NCYC --> state graphs have a cycle, so every run explored its whole bound and every terminal state was classified. Peak memory of one worker <!-- PEAK --> MB; wall time <!-- WALL --> s.

## Headline

1. **No combination of the six choices keeps every invariant in every condition, under any host setting. One combination fails the fewest cells.** That is `replay.append.record.opid.record`: replay the folded commits' net change; on re-submit, append a second planning commit; carry the transaction's identity in each planning commit's record; write authorization pointers as the authorizing operation's identity, not a SHA; and have the transition audit read the record's operations in order. It is among the best-ranked designs under every host setting (§3, ranking table), and it fails the fewest cells overall; the counts for it, for the default design `replay.replace.record.opid.record` and for today's per-verb commits are under the matrices in §3. Under merge-commit, semi-linear and fast-forward hosts, and in direct mode, it keeps every invariant in `base`, `mixed`, `crash`, `lost`, `nocheck`, `reuse`, `auth` and `adr`, and in direct mode also in `after`. Apart from one all-messages squash cell that `replace` keeps (§5, resub), what it still fails, every fold design fails. Each cause is either a move outside aiwf, a record that names a non-aiwf commit, or a fold of operations that were already pushed:
   - **`revert`.** A host revert lands entity changes with no trailers and moves M from `done` back to `draft` (T3, T7). `aiwf check` on main reports both after the revert lands; under the up-to-date rule, every later transaction's pre-push hook then fails on that commit until someone acknowledges it (L = `M`).
   - **`resolve`.** A conflict resolved by hand inside a merge commit can drop the resolver's own operation (T1), revert the other transaction's (T2), and move M `draft → done` (T7x). Nothing reports any of it, because both audits skip merge commits (A8, A9). This is R4, plus the T2 case §8 did not show.
   - **`cherry`.** A cherry-picked planning commit lands A's operations on main through B before A merges (T1). No `aiwf check` rule looks for it.
   - **`after`, `bob`.** A pushes a per-verb commit, then folds it into a planning commit. The older push survives somewhere, and the operation lands twice: once in the per-verb commit, once in the planning commit (T4d). In `after`, the host merges the older push first; in `bob`, a second operator builds on it and A's `git pull --rebase` brings it back. Today's per-verb commits do not do this: it is the one place folding is worse than today on merge-commit, semi-linear and fast-forward hosts. What-ifs W2 (`bob`) and W3 (`after`) close it: submit refuses to fold an operation whose record is already on the remote branch, judged by operation identity. Refusing once the commits are pushed, judged by SHA (W1), does not close it, because the operator's own rebase gives pushed commits new SHAs. Run across the whole matrix for both re-submit values (§3, "The refusal rule W2 across the matrix"), W2 removes the duplicate in `after` and `bob` on every non-squash host and adds no invariant failure, and no submit then makes the next push need force on a branch that did not need it. It adds a refusal instead: a transaction that pushed a verb before submitting cannot be submitted, and the model counts that refusal as actionable (§3, the rule table).
   - **`ack`, `abc`.** A record that names a commit which is not an aiwf operation breaks whenever that commit is re-created, and four things re-create it: the fold (the commit follows a planning commit), the operator's `git rebase`, a semi-linear or fast-forward host, and any squash. These records are `aiwf acknowledge illegal <sha>` of a hand edit and `addressed_by_commit` naming a code commit. Under `abc`, `addressed_by_commit` dangles on main and nothing reports it (T6). Under `ack`, the finding the acknowledgement covered comes back on the new SHA, and the pre-push hook refuses the transaction (L✗).
2. **The squash message setting decides more than any of the six choices.**
   - With a title-only squash or git's own `--squash` message, no design keeps the record (T4), the attestation (T3) or the audit (T7). That is §8 E4's result, reproduced; `aiwf check` on main reports T3 and T7 after landing, and nothing reports the lost record.
   - With every commit's message appended, the record survives only while the planning commit is the branch's last commit. In the model the operator may merge main after submitting, and then every design loses it (`squash_all@base@…__T4.txt`). The proposal states the condition (§5); the model shows an ordinary move that breaks it.
3. **Each open choice is decided by a trace.** Section 5 has the detail.
   - **fold: replay.** Copy reverts the other transaction's planning: §8 R1, reproduced from the model's own rules in 12 steps, in every host setting, and not reported by `aiwf check`.
   - **audit: read the record.** The commit-to-parent audit refuses every transaction that starts and finishes a milestone. The pre-push hook reads its planning commit as `draft → done`, which is illegal, and the transaction can never be pushed (L✗ in every host setting).
   - **SHA pointers: operation identity, for authorizations.**
     - `leave` drops the `authorize` commit's SHA in the fold: R2, reproduced.
     - `rewrite` cannot repair that pointer, because it would have to name the planning commit that contains it, and a commit cannot carry its own SHA (G15).
     - `refuse` lands the transaction per-verb. A semi-linear or fast-forward host's rebase then re-creates the empty `authorize` commit under a new SHA (G14), and an all-messages squash leaves it off main; both leave `aiwf-authorized-by` dangling. Under every host, the operator's own `git rebase` does the same before the merge, and the pipeline refuses (L✗).
     - Only `opid` keeps `aiwf-authorized-by` resolving. It covers pointers to aiwf operations only; no choice keeps the two pointer kinds of `ack` and `abc`.
   - **Transaction identity: in the record.** With the branch name as the only identity, copies of planning commits that are already on main land a second time (T4d). This happens under semi-linear, fast-forward and up-to-date hosts when the branch keeps going after a merge (`reuse`), and after a cherry-pick. With the identity in the record, `aiwf check` refuses the duplicate before the merge; probe P10 shows that the check, not the identity alone, is what holds T4.
   - **Re-submit: close either way.** Without W2 the two differ only under squash, and append is ahead.
     - After a squash that dropped a record, `replace` re-folds an operation whose transition is already on main, and the audit reads `in_progress → in_progress` (L✗ in `after`).
     - `append` leaves two planning commits, and an all-messages squash keeps only the last one's record (T7 in `reuse`, `squash/all`).
     - Elsewhere they behave the same. `append` needs no force push to re-submit verbs run after a pushed planning commit, as long as those verbs are not yet pushed themselves.
     - With W2 the order reverses. `replace` then refuses to re-fold a planning commit once it is pushed, so its re-submit never makes a push need force, and it keeps the `after` cell above. The two differ only where §3's rule table says, and there `append` is the one that fails.
4. **Direct mode (§10, open question 10): folding before each push works** with the same choices. The best design keeps every invariant there except the shared failures (`revert`, `resolve`, `ack`, `abc`). Today's per-verb commits fail `auth` in direct mode as well.
5. **Three of these failures are in today's aiwf, without any fold.** All three appear in the per-verb rows (`perverb.parent`), under every host.
   - A `git rebase` of a branch that carries an `authorize` commit re-creates that commit under a new SHA (G14). Every later `aiwf-authorized-by` on the branch then names a commit that no longer exists, and the pipeline refuses with `provenance-authorization-missing` (L✗ in `auth`).
   - `addressed_by_commit` naming a branch commit dangles after a rebase, a semi-linear or fast-forward merge, or a squash, and `aiwf check` does not re-verify it (A11).
   - `acknowledge illegal` of a branch commit is lost the same way.
   The first two are measured against aiwf ([the proposal](../../collaborative-state-and-commit-granularity.md)'s §8, E6): after `git rebase origin/main`, `aiwf check` reports `provenance-authorization-missing` and the hooked push is refused, and it reports nothing about the gap whose `addressed_by_commit` the rebased branch no longer contains (G-0396 tracks the latter). The third is an inference from measured git facts and a reading of aiwf's code.

**Checks of the model itself.**

- **Fidelity.** R1, R2, R4 (the resolution and the revert), E3 and E5's probe-drop are reproduced. All ten E4 cells match the spike's counts, and E1's tree, patch-ids and record order match (§4).
- **Probes.** Eleven mutation probes cover T1 to T7, T4's duplicate rule and L, and each fails its invariant while its control run keeps it. Three what-if runs (W1 to W3) test candidate rules for folding operations that were already pushed, and `rule_w2.py` runs W2 across the whole matrix (§3).
- **Bounds.** Every run explored its whole bound: none hit the state cap and no state graph has a cycle.

## 1. What was modelled

**Two transactions.** A (the one under study) and B (the other transaction, which may merge first or second: the one base move). Each runs a fixed script of code commits and aiwf verbs; every interleaving of the two operators, the host and the environment is explored. A's base script is `code a1, promote M in_progress, code a2, edit-body G, promote M done` — three operations, one of them a milestone started and finished inside the transaction. B's is `code b1, add gap NB`.

**Design choices** (the dimensions of §4.2 and §10):

| dimension | values | meaning in the model |
|---|---|---|
| submit | `none` (today: per-verb commits), `fold` | whether submit folds the planning commits |
| fold | `replay`, `copy` | replay: the folded commits' net change applied three-way onto the branch as it stands; copy: every planning path taken from the tree of the last folded commit |
| resub | `replace`, `append` | a second submit re-folds the existing planning commit too (one planning commit, force push when pushed), or folds only verbs run after it (a second planning commit) |
| txid | `branch`, `record` | transaction identity: the branch name only, or carried in each planning commit's record (then `aiwf check` flags an operation recorded in two planning commits, and submit refuses to fold records already on main until the branch contains main) |
| ptr | `leave`, `rewrite`, `opid`, `refuse` | records naming a SHA: left as they are; rewritten during the fold with an old→new map; written by verbs as the target operation's identity instead of a SHA (aiwf operations only); or the fold is refused and the transaction lands per-verb |
| audit | `parent`, `record` | how the transition audit reads a commit: commit-to-parent diff, or the record's ordered operations |

A design id is `fold.resub.txid.ptr.audit`, for example `replay.replace.record.opid.record`; `perverb.<audit>` is today's shape. The proposal's text fixes `replay` and says a stable identity is needed; `replay.replace.record.opid.record` is called the default design below.

**Host settings:** merge method `merge` (merge commit), `semi` (semi-linear), `ff` (fast-forward after the host rebases), `squash/title`, `squash/all` (every commit's message appended), `squash/git` (`git merge --squash`'s own message), each with and without `+utd` (branch must be up to date). `direct` is the direct-to-default-branch mode: non-force pushes to main with the pre-push hook; under the fold design, submit runs before each push.

**Conditions** (environment actions; at most one fault per trace):

| condition | what it adds |
|---|---|
| base | nothing beyond the interleavings: B may merge first, A may rebase onto or merge main at any time |
| after | A may submit and push at any point (verbs run after submit, pushes before it); a branch may merge with verbs after its planning commit; A may merge twice (an early push, then what followed) |
| bob | a second operator fetches A's pushed branch, runs `edit-body M`, and pushes (non-force) |
| mixed | A's second code commit is a `git add -A` that also carries a hand edit of M's body |
| crash | submit may crash; the model's submit updates the branch ref once |
| lost | the reply to a force push may be lost (applied or not) |
| revert | the host may revert a merged transaction, landing the revert directly on main |
| cherry | B's operator may cherry-pick one of A's planning commits from A's pushed branch |
| resolve | B edits the same gap body as A; whichever transaction merges main second hits the conflict and resolves it by hand: ours, theirs, both, or both plus moving M to done (R4) |
| nocheck | the host merges whatever the pipeline said |
| reuse | A merges after its first two steps, keeps working on the same branch, and merges again |
| auth | A: `authorize M`, `code a1`, an agent's `promote M in_progress` carrying `aiwf-authorized-by` |
| ack | A: `promote M`, a hand edit `h1` of G's body, `acknowledge illegal h1`, `code a1` |
| abc | A: `promote M`, `code a1`, `promote G addressed --by-commit a1` |
| adr | A: `code a1`, `add adr` (writes `docs/adr/`), `promote M in_progress` |

## 2. Invariants, stated precisely

Each is checked on every transition that changes the default branch, or on the rewrite of a transaction's branch where stated.

- **T1 all or nothing.** (a) No rewrite of a transaction's branch (submit, force push, crash) drops one of its own operations that is not already on main; (b) a host merge lands every operation present in the merged head; (c) no operation of a transaction that has not merged is on main. "Operation present" means its write is in the lineage of a planning path. Empty-commit operations (`authorize`, `acknowledge`) contribute only their record and are covered by T4.
- **T2 net change.** (a) The planning commit's own diff equals the folded commits' net change applied onto the branch (pointer rewrites included under `ptr=rewrite`); (b) no rewrite or resolution on a branch removes another transaction's operation that the new tip descends from; (c) no operation that ever landed on main leaves it, unless its transaction was reverted. `m` marks a removal by a hand-made conflict resolution.
- **T3 attested.** Every non-merge commit that reaches main and changes an entity file names that entity in its git-parsed trailers, or is exempted by an `acknowledge illegal` naming its SHA. Assumption A13 stands in for "a change the attesting verb may make": a verb changes only the file of the entity its `aiwf-entity` names. `x` marks a merge commit whose own content (a value in neither parent) no trailer attests; today's merge-commit carve-outs exclude it wholesale.
- **T4 record survives.** Every operation present in a merged head has exactly one git-parsed trailer group on main: `✗` none, `d` two or more. `a` (reported, not counted as a failure, and shown in a cell only where T4 otherwise holds) means the transaction's records sit in several commits and nothing on main ties them to one transaction: no merge commit naming the branch reaches them, and under `txid=branch` no record names the transaction.
- **T5 code SHAs.** A code commit that reaches main as its own commit has the SHA it was born with, or one produced by a permitted rewrite: submit re-creating a commit that follows a folded planning commit, a host rebase, or the operator's own `git rebase`.
- **T6 pointers resolve.** Every SHA-naming record on main (`aiwf-authorized-by` and `aiwf-force-for` in parsed trailers, `addressed_by_commit` in frontmatter) names a commit reachable from main whose hidden identity is the one its author meant; an operation-identity pointer names an operation recorded on main.
- **T7 legal history.** Every status change in a non-merge commit reaching main, read the way the chosen audit reads it, is legal in order (acknowledged commits excepted). `x` marks an illegal status change carried by a merge commit's own content, which the audit skips (D-0010).
- **L, conditional liveness.** Every terminal state leaves each *legal* transaction merged (its operations on main), or refused for a reason a person can act on: a git conflict, a mixed commit, another author's commit, a duplicate record, a finding on the transaction's own branch when the transaction is not legal (`R`), or a finding on a commit already on main, whose remedy is `aiwf acknowledge illegal` (`M`). Anything else is `✗` (stuck). A refusal caused only by the model's bounds is not counted. The transaction whose branch carries the fault is not legal: A in `mixed`, B in `cherry`, and in `resolve` whichever transaction resolved the conflict by hand.

In every table, `✗` means violated and `aiwf check` on main after the landing reports nothing; `!` means every violating landing found is reported by `aiwf check` on main afterwards (the pipeline had passed it, or no pipeline ran); `*` marks a run that hit the state cap.

## 3. Matrix

Cells list the invariants that fail; `—` means all hold. `L✗` is a liveness failure. Mode and host settings are the columns.

<!-- TABLES -->

#### The refusal rule W2 across the matrix

`rule_w2.py` runs the design `replay.<resub>.record.opid.record`, for both re-submit values, in every host setting and condition, with and without W2: submit refuses to fold an operation whose record is already on the transaction's remote branch, judged by operation identity.

<!-- RULE -->

## 4. Fidelity: the model against §8's measurements

**E4, host merge methods.** The spike's fixture (three code commits, four operations, the default branch moved meanwhile) replayed through the model's own actions (`fidelity.py`), counting what A's merge adds to main:

<!-- E4 -->

**E1, the fold itself** (model, planning shape, merge commit): <!-- E1 -->.

<!-- FIDELITY -->

## 5. Per choice

The tables in §3 ("One choice at a time") vary one dimension of a reference design and list every cell that changes. This section reads them. A trace named `X__inv.txt` sits in `traces/`. It is named after the first design explored that agrees with the variant on every dimension the run read, so its id can differ from the variant's in dimensions that did not matter there.

**fold: replay or copy.**

- *Replay requires:* the net change of the folded commits, path by path, applied three-ways onto the branch as it stands. It also requires a rebuild that keeps every commit whose parent did not change and re-merges a merge of main with its original second parent (A14). When the branch changed a path the planning commits also changed, submit refuses with a conflict, which is actionable.
- *Copy costs:* it takes every planning path from the last folded commit's tree. If the branch took in main after that commit, the planning commit deletes or reverts what main brought, and `aiwf check` reports nothing because the planning commit carries an `aiwf-verb` trailer (A8, G-0744).
- Shortest case, `merge@base@copy.append.branch.leave.record__T2.txt` (12 steps): B adds gap NB and merges; A merges main into its branch; A submits; the planning commit deletes NB. Under the up-to-date rule copy also leaves A stuck, because the copied tree can move M backwards and the hook refuses (`merge+utd@base@copy…__L-stuck.txt`).
- Replay was never worse in any cell.

**resub: replace or append.**

- *Replace requires:* a lease-checked force push once the planning commit is pushed. The lease must name the SHA last pushed: `--force-with-lease` with no SHA, after a fetch, overwrote the other operator's commit in G11, and probe P2 (a plain `--force`) drops bob's operation (T1). It also requires refusing to rewrite other people's commits (A14). It keeps one planning commit at the tip, so an all-messages squash keeps the whole record (`squash_all@reuse@replay.append…__T7.txt` is the append failure that replace avoids).
- *Append requires:* nothing extra once the planning commit is pushed, because it folds only verbs run after it. It costs a second planning commit on the branch (against R4's "one planning commit"), and under an all-messages squash only the last planning commit's record survives.
- The one cell where replace fails is `squash_title@after@replay.replace…__L-stuck.txt`. A pushes a per-verb `promote M in_progress`, submits locally, and the host squash-merges the older push. A re-submits, and replace re-folds the `in_progress` step into a planning commit whose parent already has M `in_progress`, so the audit reads `in_progress → in_progress`.
- Outside squash hosts the two never differ.
- *With W2*, `replace` refuses once its planning commit is pushed, because the re-fold would include operations recorded on the remote branch; verbs run after that cannot be folded. No submit then makes the next push need force on a branch that did not need it, under either value (§3, the rule table).

**txid: branch name or record.**

- *Record requires:* a transaction identity in each planning commit's record, an `aiwf check` rule that refuses an operation recorded in two planning commits, and a submit that refuses to fold records already on main until the branch contains main (A15).
- *Branch costs:* after a semi-linear or fast-forward host rewrote the commits, or after a cherry-pick, nothing can tell a copy of a planning commit from the original, and both land (`merge+utd@reuse@replay.append.branch…__T4d.txt`, 19 steps; `merge+utd@cherry@replay.append.branch…__T4d.txt`).
- Under plain merge commits the identity changes nothing in `reuse` and `cherry`: the original planning commit keeps its SHA on main, so the copy is recognised by ancestry.
- Around the default design (`replace`), the branch identity also fails L in `after` under every non-squash host (`merge@after@replay.replace.branch.leave.record__L-stuck.txt`). After the host merged A's older push, the re-fold carries `A_ip`'s record onto a parent where M is already `in_progress`, and the audit refuses. With the identity in the record, submit sees `A_ip` already recorded on main and drops it.
- Probe P10 removes the duplicate check and keeps the identity; T4d fails. The identity is useful only to a reader that checks it.

**ptr: leave, rewrite, opid or refuse.**

- `opid` *requires:* every verb that names an aiwf operation writes that operation's identity (a new trailer key) instead of a SHA, and the provenance check resolves it by searching records rather than commits. It covers `aiwf-authorized-by` and `aiwf-scope-ends`, which name `authorize` operations. It does not cover `acknowledge illegal <sha>` of a hand edit or `addressed_by_commit` of a code commit, because neither target is an aiwf operation.
- `leave` *costs:* the fold drops the `authorize` commit, and the pointer names nothing (`merge@auth@replay.append.branch.leave.parent__L-stuck.txt`; the pre-push hook and the pipeline refuse with `provenance-authorization-missing`, R2).
- `rewrite` *costs:* the same for any pointer into the planning commit itself (G15). It repairs a pointer to a code commit the fold re-created, until a semi-linear or fast-forward host or a squash re-creates it again (`semi@ack@replay.append.branch.rewrite.parent__T6.txt`).
- `refuse` *costs:* the transaction lands per-verb (no quiet history), and per-verb commits are exposed to every later rewrite. A semi-linear host's rebase re-creates the empty `authorize` commit under a new SHA (G14), and `aiwf-authorized-by` dangles on main (`semi@auth@replay.append.branch.refuse.parent__T6.txt`, 10 steps).
- Under `ack` and `abc` no value keeps both T6 and L, because the operator's own `git rebase` already re-creates the named commit. Where `leave` or `opid` keep T6 under `ack`, they do so vacuously: the hook refuses the transaction first. Probe P8 shows that `refuse` holds `abc` only where nothing else rewrites the fixing commit.

**audit: commit-to-parent or record.**

- *Record requires:* every reader of trailers taught to read all groups in order (A12; today's readers do not, §8 E5). It also requires the record to survive the merge. Where it does not (title-only or git-default squash), both audits read the squash commit as one `draft → done` step (T7, reported after landing).
- *Parent costs:* it refuses every transaction that starts and finishes a milestone (`merge@base@replay.append.branch.leave.parent__L-stuck.txt`, 12 steps).
- The cells where parent looks better are all vacuous: it keeps T3, T4 or T7 there only because the transaction never lands.
- Probe P11 writes the groups out of order: the record audit reads `done` before `in_progress` and refuses.

## 6. Assumptions the model encodes

Each line: the assumption, then its source (a measurement, a code reading, or none), then the test that would settle it if it is unmeasured. **[id]** marks an assumption the id-allocation model (`docs/research/id-allocation/model/RESULTS.md`) also depends on (its §3 numbering is kept where it applies), so one round of host measurements covers both.

**Git**

- **G1 [id].** A non-force push is an atomic compare-and-swap. Not measured here; used for direct-mode pushes and for bob's push. Test: the id model's G1 race.
- **G2 [id].** After a lost push reply the ref is either updated or not, and a fetch shows which. Not measured. Test: cut the connection after `receive-pack` applies an update.
- **G3 [id].** A status line and a body paragraph of one entity file merge cleanly; two different values of one line, or two edits of one body line, conflict. Measured (`gitchecks/OUTPUT.txt`, G3a–c). The model treats every frontmatter line and every body as an independent merge unit; adjacent frontmatter lines changed on both sides would conflict in git, which the model does not represent (inference: git conflicts on adjacent hunks).
- **G7.** Git parses trailers only from a message's last paragraph; repeated keys in one paragraph all parse; a paragraph after the trailer block (for example a co-author line) hides it. Measured (`gitchecks/OUTPUT.txt`, G7).
- **G8.** `git merge --squash`'s default message indents each commit message, and git parses no trailers from it. Measured (G8), as D-0080 also found.
- **G9.** A cherry-pick, with or without `-x`, keeps the trailer block parseable and gives the commit a new SHA. Measured (G9).
- **G10.** A revert's message carries no trailers. Measured (G10).
- **G11.** A force push with `--force-with-lease=<ref>:<sha>` naming the SHA last pushed is refused ("stale info") when someone else pushed meanwhile; retrying it after a lost reply whose push had applied reports "Everything up-to-date" and exits 0; `--force-with-lease` with no SHA, after a fetch, overwrites the other operator's commit. Measured (G11a, G11b).
- **G12.** `git rebase` keeps a commit's SHA when its parent is unchanged; a re-created commit gets a new SHA. Measured (G12). The model is content-addressed and abstracts dates, so it cannot express a host that re-creates commits that need no rebase.
- **G13.** `git rebase` drops a commit whose patch is already upstream. Measured (G13). Both of this run's examples were copies of commits already squashed into main.
- **G14.** `git rebase` keeps a commit that started empty, under a new SHA, with its trailers. Measured (G14), using an `aiwf authorize`-shaped empty commit.
- **G15.** A commit's SHA covers its message, so no commit can carry its own SHA. This is git's object format, not a measurement. It is why `ptr=rewrite` cannot fix a pointer to an operation folded into the same planning commit.

**Host**

- **H1 [id].** The host serialises merges into the default branch. Not measured.
- **H2 [id].** The up-to-date rule refuses a merge unless the default branch is an ancestor of the branch head. Not measured.
- **H3 [id].** A pipeline checks the merge result as of when it ran and is not re-run when the default branch moves. Not measured.
- **H4 [id].** A pipeline verdict is tied to the head commit; a push re-runs it. Not measured.
- **H7.** Merge methods behave as §8 E4 emulated them locally: `merge` makes a `--no-ff` merge commit; `semi` rebases a branch that is behind, then makes a merge commit, and updates the source branch to the rebased commits; `ff` rebases a branch that is behind, then fast-forwards. A branch that is up to date is not rebased. Locally emulated only (E4), and the model reproduces E4's counts. Test: merge one branch by each method on GitHub, GitLab and Azure DevOps, and compare SHAs before and after, including on an up-to-date branch (GitHub's "rebase and merge" is believed to re-create commits even then; not checked).
- **H8.** A squash commit has one parent. Its message is the title only; or the title followed by every branch commit's message, oldest first (E4's `git log --reverse --format=%B` emulation); or G8's indented message. Real hosts' templates are not measured. Test: squash-merge a branch whose last commit carries a trailer block on each host, under each template setting, and run `git log --format=%(trailers)`.
- **H9.** The host's merge, squash and revert commits are made at merge time; nothing checks them before they land. Inference from H3 and H4.
- **H10.** A host revert lands one revert commit directly on the default branch. R4 demonstrated the revert's content; how it reaches the default branch (a direct commit, or a merge request whose pipeline runs `aiwf check`) is not measured.
- **H11.** A host merges a merge request whose commits add nothing to the tree, such as copies of commits already on main. Not measured. Test: open a merge request from a branch carrying only copies of merged commits.

**aiwf**

- **A5 [id], refined.** The pre-push hook is installed and runs `aiwf check` on the pushed tip; its provenance rules cover the commits since the upstream and are skipped on a branch's first push with no upstream (G-0539, G-0679); the FSM walk covers the whole history. Source: §5 and the gaps; the FSM scope is read from `internal/check/fsm_history_consistent.go`, not run.
- **A8.** The untrailered-entity audit skips any commit with an `aiwf-verb` trailer (G-0744) and every merge commit whose subject is not GitHub-squash-shaped (G-0231 carve-out). Source: `internal/check/provenance.go:513-534`, read; E5's probe-drop and R4 measured both effects.
- **A9.** `fsm-history-consistent` compares each non-merge commit's status with its parent's, skips merge commits (D-0010), and exempts a commit whose SHA an `aiwf-force-for` trailer names. Source: `internal/check/fsm_history_consistent.go:321-375`, read. The `record` audit value is the proposal's candidate, not existing code.
- **A10.** `provenance-authorization-missing` fires when `aiwf-authorized-by` does not resolve to an authorize commit in the walked history. Source: `internal/check/provenance.go:293-300`, read; R2 measured it.
- **A11.** `aiwf check` does not re-verify `addressed_by_commit`; only `aiwf promote --by-commit` checks that the SHA is reachable from HEAD, at promote time. Source: `internal/verb/promote.go:630-661` and `internal/check/check.go:979-995`, read. Measured: after a rebase, `aiwf check` reported nothing for a gap whose `addressed_by_commit` named a commit the rebased branch no longer contains (the proposal's §8, E6; G-0396).
- **A12.** The record is today's trailer keys repeated, one group per operation, in one paragraph (E5 format f1: git parses every group and the commit-msg hook accepts it), and every reader is taught to read all groups in order (open question 2). Today's readers do not (E5).
- **A13.** Each modelled verb changes only the file of the entity its `aiwf-entity` names (an AC's verbs change its milestone's file). This stands in for the missing record of which entities each verb may touch (§9.2, limit 3; open question 4). Verbs that touch more (reallocate, rename's reference rewrites, archive, the wrap rituals) are outside the model.
- **A14.** Submit as modelled. It fetches the default branch first. It treats a commit as planning when it carries an `aiwf-verb` trailer. It refuses a commit that mixes planning and code, and refuses when the commits it would fold or rewrite include anyone else's. It keeps every commit whose parent is unchanged, re-creates the others in order, and re-merges merge commits with their original second parent (as `git rebase --rebase-merges` does). It writes the record in operation order. It updates the branch ref once. The proposal states the refusals; the rest are implementation choices the model fixes.
- **A15.** The operator asks for a merge only once the pushed branch is submitted (except in `after`). Under `ptr=refuse` a refused fold lands per-verb. Under `txid=record` the identity is in every planning commit, `aiwf check` refuses an operation recorded in two planning commits, and submit refuses to fold records already on main until the branch contains main (then it drops them). Under `ptr=opid`, verbs write an aiwf operation's identity in `aiwf-authorized-by`; pointers to commits that are not aiwf operations (a hand edit's acknowledgement, a code commit's `addressed_by_commit`) remain SHAs.

## 7. Findings for the proposal

Each item states what the model shows and the trace or table that shows it; the bracket names the section of [the proposal](../../collaborative-state-and-commit-granularity.md) that carries it.

**Confirmed**

- **Replay, not copy.** Copying `work/` from a snapshot reverts planning another transaction merged meanwhile: copy fails T2 in every host setting (§5, fold). [§4.2]
- **The fold carries every path the verbs wrote,** not only `work/`. Probe P3 drops `docs/adr/` from the fold, and the `add adr` operation never lands (T1). [§4.2]
- **A rewrite of a pushed branch needs a lease naming the SHA last pushed.** P2 (plain `--force`) overwrites bob's operation (T1), and G11 measured the lease semantics. With W2, no submit makes a push need force; a push after the operator's own rebase of a pushed branch still does (§3, the rule table). [§4.2]
- **Collapsed transitions.** The commit-to-parent audit refuses a milestone started and finished in one transaction; reading the record fixes it (§5, audit). [§4.2]
- **Records naming SHAs break when the fold changes SHAs.** R2 is reproduced (`ptr=leave`). [§4.2]
- **A stable identity is needed once branch names vanish.** With the branch name only, duplicates land under semi-linear, fast-forward and up-to-date hosts (§5, txid). [§4.2]
- **Squash messages.** Title only keeps no record; an all-messages squash keeps only the last paragraph's trailers; `git merge --squash`'s default message indents each message, so git parses none (G7, G8; E4 replay). [§5]
- **Merge commit, semi-linear and fast-forward keep every trailer; rebasing changes code SHAs.** The E4 replay reproduces it (§4). [§5]
- **A planning commit whose trailers name fewer entities than it changes passes `aiwf check`.** Probe P5: T3 fails and `aiwf check` on main reports nothing. [§7]

**Findings the proposal states**

- **The spike's E1 re-created every code commit.** Its script cherry-picked each code commit onto `base`, so `feat: c1`, which precedes the first planning commit, changed SHA too (00e2045 → e43bf11 in the spike's output). The model's fold keeps `c1` (E1 replay, §4), and probe P7, which re-creates every commit as the script did, breaks T5. [§8, E1]
- **Folding operations that were already pushed records them twice** whenever the older push survives: the host merges it first (`after`), or another operator builds on it (`bob`). Refusing once the commits are pushed, judged by SHA, is not enough, because the operator's own `git rebase` gives pushed commits new SHAs (W1 still fails). Judging it by operation identity closes both routes (W2, W3, and §3's rule table). [§4.2]
- **The refusal of other people's commits applies to the commits the fold would fold or re-create,** identified by author (merge commits exempt), not by having been pushed: a test limited to the pushed SHAs lets bob's commit be folded into A's planning commit after A's own rebase. The model applies the rule as A14 states it. [§4.2]
- **A cherry-pick moves pending work.** A cherry-pick of A's planning commit onto B lands A's operations on main through B's merge (T1), under every design, and nothing reports it (`merge@cherry@replay.append.record…__T1.txt`). [§4.1]
- **The operator's own `git rebase` and `git pull --rebase` change code SHAs too.** The model counts both as permitted (T5); a pointer to such a commit still breaks (T6). [§9.2]
- **Rewriting SHA records during the fold with an old→new map** cannot fix a pointer to an operation folded into the same planning commit, which is the `authorize` case: a commit cannot name its own SHA (G15). [§4.2]
- **A transaction merged without being submitted keeps what today keeps, and no more.** Under an all-messages squash, the per-verb shape keeps only its last group (E4 replay, `perverb`). [§4.1]
- **A revert path.** A host revert lands untrailered entity changes and an illegal `done → draft`; `aiwf check` reports both after landing. Under the up-to-date rule every later transaction's pre-push FSM walk then fails on that commit (L = `M`) until someone runs `aiwf acknowledge illegal` on it. The same holds for any title-only squash that lands an illegal-looking step. [§9.2]
- **The merge-commit carve-outs hide more than R4 showed.** A hand resolution that keeps one side reverts the other transaction's planning (T2), silently, as well as moving a milestone `draft → done` (T7x). [§9.2]
- **The all-messages squash needs the planning commit at the tip when the host squashes.** The model lets the operator merge main after submitting, and every design then loses the record. Requiring the planning commit at the tip (re-submit after a merge of main) would close the route; that is inference, not modelled. [§5]
- **Copies of merged commits.** A branch that keeps going after its merge, or a cherry-pick, re-lands planning commits whose originals the host rewrote. The identity in the record matters only if `aiwf check` refuses the duplicate (P10). [§4.2]
- **Records naming non-aiwf commits.** `acknowledge illegal <sha>` of a hand edit and `addressed_by_commit` of a code commit have no fold-proof form. Every rewrite of the named commit breaks them, including the operator's own rebase, and `aiwf check` does not re-verify `addressed_by_commit` (A11). Only `aiwf-authorized-by` and `aiwf-scope-ends`, which name aiwf operations, have a fix among the candidates. [§4.2]
- **"A change the attesting verb may make."** The model assumed A13: a verb changes only its named entity's file. Under that assumption the planning commit's record must name every entity it changes, and probe P5 shows that `aiwf check` does not enforce it (G-0744). [§7; §9.2, limit 3]
- **Liveness under findings on main.** One bad landing blocks every later push under the up-to-date rule, so liveness needs a third condition: no unacknowledged finding on main. [§9.2]

## 8. How to rerun, and limits

From this directory, standard library only:

```
python3 -I run.py --jobs 12          # whole matrix, probes; writes results.json and traces/
python3 -I run.py --only "merge@base,ff@reuse" --jobs 4   # a subset by host@condition substring
python3 -I fidelity.py               # E1/E4 replay; writes fidelity.json
python3 -I rule_w2.py --jobs 12      # W2 across the matrix; writes rule_w2.json
python3 -I make_tables.py            # tables.md and RESULTS.md from the template and the three JSON files
bash gitchecks/run_all.sh            # the real-git checks; writes gitchecks/OUTPUT.txt
```

Each trace file named in this document is `traces/<host>@<condition>@<design>__<invariant>.txt`, one step per line with both branches and main after each step. A file ending `-silent.txt` is the shortest violation that `aiwf check` on main does not report.

<!-- STATS -->

**Limits.**

- **Bounds.** Two transactions; at most three operations each (four in the E4 replay); one base move (B's merge); at most two submits per transaction (one re-submit); one fault per trace; two merges for A only in `reuse` and `after`. B never pushes before its script is complete and takes in main only where its merge needs it.
- **One milestone, one gap and one new entity per transaction.** Several entity kinds, references between entities, ids, archive and areas are not modelled. Ids are unique and stable by assumption.
- **The fold is the model's own implementation (A14).** A real submit may differ in ways that matter: for example, the spike's E1 script re-created every code commit, which probe P7 shows breaks T5.
- **The `record` audit, `opid` pointers, the duplicate-record check and submit itself do not exist in aiwf.** The model states the behaviour a candidate would have; it checks the design, not code.
- **Hosts are emulated** as in §8, with the assumptions H7–H11.
- **T4's reporting and the duplicate check.** Every T4 violation is recorded as unreported, although the model's own check would report a duplicate on main after the merge. The duplicate check counts planning commits only, so a per-verb commit and a planning commit recording one operation are not flagged; W2 is what keeps that case off main.
- **W2's view of the remote.** W2 reads the remote branch as last pushed or fetched. A lost reply could leave that stale, but lost replies are modelled only for force pushes; in direct mode W2 has no effect.
- **Not modelled:** signing; each operation's time; a co-author paragraph appended by a host (G-0657); a host that rewrites commits that need no rebase; merge queues and merged-results pipelines; abandoning a branch; three or more transactions.
