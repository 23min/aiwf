# Id-allocation experiments: results

**Run:** 2026-10-25. aiwf built from `1e8e17e6c` (it reports `v0.42.0+dirty`). git 2.51.1, Linux devcontainer. Every remote is a bare repository reached over the local (file path) transport. Host behaviour is simulated by scripts, so no GitLab or GitHub behaviour was measured.
**Layout:** `scripts/` holds the scripts. They write full outputs to `out/` and disposable repositories, which each run recreates, to `runs/`, both under `$IDEXP_WORK` (default `${TMPDIR:-/tmp}/aiwf-idexp`) and neither committed; `out/` paths below are relative to it. The aiwf binary under test must be on `PATH`, since the installed hooks call it too.
**Seed** (`scripts/lib.sh` `setup`): an `aiwf init` repo holding E-0001, M-0001 (draft) and G-0001 on `main`, pushed to `host.git`. `mkclone <name> hooks` copies aiwf's four hooks into a clone, since clones carry none.
**Reference markers.** Every citation written by a script records what its author meant. In a body it reads `G-0002 (means:alpha-gap)`. For a frontmatter field, the referrer's title carries `(parent means alpha-epic)`. `scripts/verify.py` resolves each marked id through aiwf's own loader (`aiwf list --kind K --archived --format=json`) and compares the target's slug with what was meant. It prints `OK`, `WRONG` (a different entity), `AMBIGUOUS` (the id is duplicated) or `DANGLING`.

---

## 1. Designs

### S0: today's behaviour (baseline)

Scripts: `s0_visibility.sh`, `s0_merge.sh <noff|ff-rebase|semilinear> <hooks|nohooks>`, `s0_repair.sh <before-merge|after-merge|after-rebase>`, `s0_samepath.sh`, `s0_samepath_epic.sh`. In each run alpha and beta clone first. Each then allocates a gap and an epic on its own branch. Alpha adds a milestone under its epic and cites its gap from G-0001. Beta adds a gap `discovered_in` its epic and cites its gap from M-0001 (`scripts/actors.sh`).

**Allocation: what beta's `aiwf add gap` returns** (`out/s0-visibility.txt`). Expected per ADR-0025: alpha's id is skipped whenever it is visible in a local or remote-tracking ref.

| alpha's G-0002 | beta | beta got |
|---|---|---|
| committed, not pushed | none | G-0002 (collision) |
| pushed branch | no fetch | G-0002 (collision) |
| pushed branch | `git fetch` | G-0003 |
| pushed branch | `aiwf add --fetch` | G-0003 |
| pushed, then remote branch deleted | `git fetch --prune` | G-0002 (the id is reused) |
| same clone, sibling worktree | none | G-0003 |

**Detection, with hooks, after the collision exists** (`out/s0-repair-before-merge.txt`):

| moment | beta's hooked `git push` of its branch |
|---|---|
| D1: alpha's G-0002 only on `origin/feature/A`, beta has fetched it | passes (`2 findings (0 errors, 2 warnings)`) |
| D2: alpha merged to main, beta has not fetched | passes |
| D3: alpha merged, beta fetched | refused: `error ids-unique/trunk-collision: id "G-0002" is allocated on this branch (work/gaps/G-0002-beta-gap.md) and on trunk (...)` |

A read-side signal does exist before any merge (`out/s0-readside.txt`). A third clone that has fetched both branches gets `aiwf show G-0002` → `G-0002 · cross-branch collision · known on: refs/remotes/origin/feature/A, refs/remotes/origin/feature/B … declining to pick one`, and `aiwf list --kind gap` shows `G-0002  ⇄ collision`. `aiwf check` on that clone reports nothing for it (`2 findings (0 errors, 2 warnings)`).

**Host merges both MRs, no check** (`out/s0-{noff,ff-rebase,semilinear}-nohooks.txt`). Expected: the two G-0002 files have different paths, so git merges them without conflict. Observed: all three methods merged and pushed. Main then holds `G-0002-alpha-gap.md G-0002-beta-gap.md` and `E-0002-alpha-epic E-0002-beta-epic`, and `aiwf check` gives `7 findings (4 errors, 3 warnings)`. verify: `0 correct, 4 not` (all `AMBIGUOUS`). When a maintainer merges locally with the hooks installed (`out/s0-{noff,ff-rebase}-hooks.txt`), the push of main is refused (`error ids-unique: id "G-0002" is also used by …`, `[exit 1]`) and nothing lands.

**Repair, and whether references stay correct** (`out/s0-repair-*.txt`, `out/s0-noff-nohooks.txt`, `out/s0-repair-on-main.txt`):

| repair | verify after | push |
|---|---|---|
| beta runs `aiwf reallocate <path>` on its branch, *then* `git merge origin/main` | `4 correct, 0 not` | passes |
| beta runs `git merge origin/main`, *then* `aiwf reallocate G-0002` (the route the trunk-collision hint leads to) | `2 correct, 2 not`: `WRONG G-0001 body: G-0004 meant alpha-gap; resolves to beta-gap`, `WRONG M-0002 parent: E-0003 meant alpha-epic; resolves to beta-epic` | passes; the misdirection reaches main |
| beta runs `git rebase origin/main`, then reallocates | `2 correct, 2 not` (the same two) | passes |
| carol reallocates beta's copies on main | `2 correct, 2 not` | **refused by the hooked pre-push** (see defect 2) |

**Same path** (`out/s0-samepath.txt`, `out/s0-samepath-epic.txt`): both clones file a gap with the same title, so both files are `work/gaps/G-0002-login-fails.md`.

- With the same body as well, the two blobs are identical (`blob A: dd8962e… blob B: dd8962e…`). Beta's pre-push passes after fetching, the host merge is clean (`Merge made by the 'ort' strategy.`), and main holds a single G-0002 with `1 findings (0 errors, 1 warnings)`. verify flags both citations, alpha's and beta's, as `WRONG`, since both resolve to the one file. `aiwf history G-0002` lists two `add` commits, by alpha and by beta. The same happens with `aiwf add epic --title "Q3 planning"` and no `--body`, so it does not depend on the scripts supplying bodies.
- With different bodies, beta's pre-push still passes (the rule skips an id at the same path) and the host merge stops at `CONFLICT (add/add)`.

**History after a correct repair** (`out/s0-repair-before-merge.txt`): `aiwf history G-0002` (alpha's gap) lists alpha's add, beta's add and beta's reallocate. `aiwf history G-0004` (beta's) lists the same three commits.

**Conclusion.** A collision is prevented only when the other id has been pushed and fetched, or sits in a local branch. Otherwise it is detected in one of three places: at the branch's pre-push, but only once the other id is on trunk *and* fetched; at a hooked maintainer's push of main; or by `aiwf show/list` on a third clone. A hosted merge lands it. References stay correct only when the renumbering runs on the branch while the tree holds the developer's own copy alone, before main is merged in or rebased onto. The ids of the renumbered side change after they have already been pushed.

### A: the default branch as the ledger

Scripts: `a_host.sh` holds the host rule. It merges only if `git merge-base --is-ancestor origin/main origin/<mr>` holds and `aiwf check` passes on the MR tip; it then runs `git merge --no-ff` and a non-force push of main. `a_run.sh <right|wrong> <seq|simul> <all|main>` drives three clones (alpha, beta, gamma). Each allocates a gap, an epic and a milestone, all three colliding (G-0002, E-0002 and M-0002 for everyone), and cites them with markers. Then `a_edge.sh` and `a_edge3.sh` cover the edges.

- *right*: fetch; reallocate each path that `aiwf check` reports as `trunk-collision` (at that point the tree holds only the developer's own copy); then merge `origin/main` in.
- *wrong*: fetch; merge `origin/main` in; reallocate each id reported as `ids-unique`.
- *seq*: each refused developer updates before the next MR is tried. *simul*: every refused developer of a round fetches first, then all of them update.

| policy / schedule / fetch | merge attempts for 3 MRs | `aiwf reallocate` runs | duplicate ids on main | verify on main |
|---|---|---|---|---|
| right / seq / all | 6 | 6 | 0 | 6 correct, 0 not |
| right / simul / all | 6 | 9 (gamma renumbers twice) | 0 | 6/6 |
| right / simul / main-only | 6 | 9 | 0 | 6/6 |
| right / seq / main-only | 6 | 9 | 0 | 6/6 |
| wrong / seq / all | 11 (capped at 8 rounds) | 6 | 0 | 2 correct, 2 not |

In the *wrong* run gamma never merges. Its second update hits `CONFLICT (content): Merge conflict in work/gaps/G-0001-seed-gap.md` and in alpha's milestone file, because beta's and gamma's reallocates had each rewritten alpha's referrers, differently. Main ends with alpha's two referrers pointing at beta's entities (`out/a-wrong-seq-all.txt`).

Edges:

- **The rule refuses a colliding branch that is up to date** (`out/a-edge3.txt`): `MR mr/beta (main merged in, no reallocate) -> REFUSED-check-failed: error ids-unique: id "G-0002";`.
- **A byte-identical same-path collision passes the rule** (`out/a-edge.txt` E1): `MR mr/beta -> MERGED`. Main ends with one `E-0002-q3-planning` and two `add` commits in its history.
- **Two host workers run the rule at once** (E2, 10 trials): one merge lands every time. The other worker reports `REFUSED-main-moved`, meaning its push of main was rejected. Observed: `merge commits on host main=1` in 10 of 10 trials.

**Conclusion.** Only one claim was measured: no collision with a *different path* can land, because a branch that contains main's tip carries both copies, and `ids-unique` refuses that tree. The rule does not stop a same-path, byte-identical collision. Every losing MR's ids change after its branch has been pushed. References stay correct only under the *right* order. With 3 concurrent MRs, the "contains the tip" rule took 6 merge attempts. Reallocations were 6 (each loser renumbered each of its 3 entities once) when the second loser could see the first loser's renumbered branch, and 9 when the losers updated from the same fetched state, or fetched main only (gamma renumbered each entity twice). The rule's correctness rests on the host serializing merges. Here that was a non-force push of main; a real host was not measured.

### B: a ledger ref

Scripts: `b_facts.sh` measures the git facts, `b_consume.sh` how aiwf consumes the ledger, `b_lib.sh` holds the protocol, then `b_race.sh`, `b_lostreply.sh` and `b_nonparticipant.sh`. The ledger is the ordinary branch `aiwf/ids`. Each claim is a commit built with plumbing (read-tree of the parent → update-index of one blob → write-tree → commit-tree) that adds `work/gaps/G-NNNN.md`, whose content holds `claimed-by:` and a nonce. The claim is pushed with a non-force push.

**Consumption with no aiwf code change** (`out/b-consume.txt`). Main holds G-0001 and the ledger claims G-0005.

| ledger form | `aiwf add gap` allocates |
|---|---|
| branch `aiwf/ids`, fetched by default to `refs/remotes/origin/aiwf/ids` | G-0006 |
| `refs/aiwf/ids` fetched to the local `refs/aiwf/ids` | G-0002 (not seen) |
| `refs/aiwf/ids` fetched to `refs/remotes/origin/aiwf-ids` by a configured refspec | G-0006 |
| the claimer, filing the gap it claimed | G-0006, not G-0005 |

`aiwf add` has no flag that takes an id. So **claim first, then add** cannot be consumed without a code change: the allocator skips the claimed id, the claimant included. **Allocate, then claim** works unchanged. `aiwf add` picks max+1 over a view that includes the ledger, the client claims that id, and if someone else holds it, the client runs `aiwf reallocate` (whose allocator now sees that claim) and claims again. All of this happens before the id is cited or the branch pushed.

There is a side effect on clones that do not have the entity yet. If the claim file's content differs from the entity's, a third clone gets `aiwf show G-0002` → `cross-branch collision … declining to pick one`, and a body citation gets a `body-prose-id` warning `has diverging content across refs/remotes/origin/aiwf/ids, refs/remotes/origin/feature/A`. With a claim file that copies the entity, `aiwf show` resolves. Once the entity is on main, the ledger causes no findings (`out/b-race-main-check.txt`: `1 findings (0 errors, 1 warnings)`, a milestone warning).

**Race** (`out/b-race.txt`). Three clones with hooks run the protocol at the same moment as background jobs, then the host merges all three branches with no check. Over 8 trials, every trial ended with ledger claims `G-0002.md G-0003.md G-0004.md`, main gaps `G-0001 G-0002 G-0003 G-0004`, `ids-unique errors on main: 0` and `verify: 3 correct, 0 not`. For all 24 filings, the id at first push equals the id on main (`shared->main: alpha:G-0003->G-0003 …`). Reallocations per trial were 0+1+2. The push rejections seen were `(reference already exists)` ×16, `(incorrect old value provided)` ×7 and `(fetch first)` ×1.

**Lost replies** (`out/b-lostreply.txt`):

| case | recovery | outcome |
|---|---|---|
| L1: push applied, client killed (`timeout 1` while the host's post-receive sleeps) | look for my nonce in the ledger | `client rc=1; host ledger now: a0bb1d9`, then `final id G-0002 after 2 claim attempts`; one claim on the ledger |
| L1 | naive: any holder counts as someone else | `final id G-0003`; the ledger holds G-0002 *and* G-0003, both with `nonce: alpha-1`, so G-0002 is burned |
| L2: push rejected, client believes it succeeded | naive | main holds `G-0002-alpha-gap.md G-0002-beta-gap.md`; `ids-unique errors on main: 2` |
| L2 | read back the ledger before using the id | `read-back: ledger holds G-0002 for 'claimed-by: alpha' -> reallocate and claim again`; unique, `2 correct, 0 not` |

**Non-participant** (`out/b-nonparticipant.txt`): a clone that runs plain `aiwf add` without fetching got G-0002 next to alpha's claimed G-0002. Its hooked push passed, and main ended with `ids-unique errors on main: 2`.

**Conclusion.** In this setup uniqueness held and no id changed after it was shared, under three conditions:

1. Every allocator follows allocate-then-claim.
2. The ledger is visible under `refs/heads/*` or `refs/remotes/*`.
3. The client confirms its claim by reading it back (by nonce) after any push outcome.

A non-participant breaks it. Each claim push from a hooked clone needs `--no-verify` (fact 17).

### C: collision-free ids (namespaced, random, wider)

Scripts: `c_shapes.sh`, `c_trailers.sh`, `c_widths.sh` (`out/c-shapes.txt`, `out/c-trailers.txt`, `out/c-widths.txt`). Each shape is a hand-made `work/gaps/<file>` committed through the hooks.

| id shape | loader / check | `aiwf show` and verbs | prose citation | `--relates-to` | next `aiwf add gap` |
|---|---|---|---|---|---|
| `G-pb-0012` | `unexpected-tree-file` (warning) | not found | `error body-prose-id/malformed-shape … "G-pb"` (the edit is refused) | `refs-resolve/unresolved` (refused) | G-0002 |
| `G-7f3k` | `unexpected-tree-file` | not found | `malformed-shape "G-7f3k"` (refused) | refused | G-0002 |
| `G-0012a` | `unexpected-tree-file` | not found | `malformed-shape` (refused) | refused | G-0002 |
| `G-00012` (5 digits) | accepted | show, retitle and promote all work | accepted | accepted | G-0013 |
| `G-123456789` | accepted | all work | accepted | accepted | **G-123456790** |
| `G-0012-pb-…` (namespace in the slug) | accepted as plain `G-0012` | all work | accepted | accepted | G-0013 |
| `G-1234.2` | `unexpected-tree-file` | not found | read as `G-1234`: `body-prose-id/unresolved: … unknown id "G-1234"` | refused | G-0002 |
| `G-1234-2` (`id: G-1234-2`) | `error id-path-consistent: frontmatter id "G-1234-2" does not match path-encoded id "G-1234"` | show and verbs work on `G-1234-2`; `aiwf list` lists it twice | — | accepted | G-1235 |

`G-0012` and `G-00012` coexist as two entities with no finding. A commit carrying `aiwf-entity: G-pb-0012` passes the commit-msg hook, and `aiwf history G-pb-0012` lists it. Because `unexpected-tree-file` is a warning by default, a namespaced file passes pre-commit and pre-push and lands invisible to every aiwf verb.

**What a kernel change would touch.** The id-shape regexes and numeric parsing are:

- `internal/entity/entity.go`: `idPatterns` :285–292, `compositeIDPattern` :317, the per-kind filename patterns :514–517, `idLeadingPattern` :659, `IDFromPath` :676 and `PathKind` :733.
- `internal/entity/allocate.go`: `AllocateID` :59 and `parseIDNumber` :88, where the max+1 is numeric.
- `internal/entity/canonicalize.go`: `Canonicalize` :32 and `IDGrepAlternation`.
- `internal/check/body_prose_id.go`: `idTokenPattern` :92, `strictBareIDPattern` :97, `strictCompositeIDPattern` :101.
- `internal/check/doc_id_slug.go`:56 and `internal/check/skill_body_id.go`:109.
- `internal/verb/reallocate.go`: `proseRewritePattern` :297 and `reallocatePaths` :452.
- `internal/trunk/trunk.go` `idsFromPaths`, which reads paths through `PathKind` and `IDFromPath`.
- `internal/entityview/historyevent.go`:330, `internal/roadmap/roadmap.go`:220, `internal/branchparse/branchparse.go`:30/53/99, `internal/cli/check/commit_msg.go`:150 and `internal/aiwfyaml/aiwfyaml.go`:63.
- Repo policies: `internal/policies/no_dangling_entity_refs.go`:25, `design_doc_anchors.go`:24 and `no_hardcoded_entity_paths.go`:19.

This list comes from grepping for the id regexes; it was not checked by building a changed kernel.

### D: merge-request numbers, and E: mint at integration (ADR-0001)

**D** (`scripts/d_mr.sh`, `out/d-mr.txt`; shapes in the C table):

- A plain number such as `G-0057` is accepted. A second entity of the same kind in one MR has no legal shape: `G-1234.2` is invisible, and `G-1234-2` raises the `id-path-consistent` error.
- `aiwf reallocate` takes no target id, so renumbering to the MR number has to be done by hand. That hand renumbering (`git mv` plus `sed`, plain commit) is refused at a hooked push: `error provenance-untrailered-entity-commit: commit 471064e touched G-0057 with no aiwf-verb: trailer`. Afterwards `aiwf history G-0057` gives `no history for G-0057`.
- Once G-0057 is on main, the sequential allocator jumps: bob's next `aiwf add gap` gives **G-0058**. Sequential allocation and MR-number ids therefore share one number space.

**E** (`scripts/e_mint.sh`, `out/e-mint.txt`). Protected main is simulated by a `pre-receive` hook that admits only push option `actor=merge-bot` or `actor=mint-bot`. The mint itself is a shell stand-in (`git mv` plus `sed`); aiwf has no `assign-pending`, so no real mint implementation was measured.

- Inbox files under `work/gaps/inbox/<slug>.md` give `unexpected-tree-file` (a warning), and branch pushes pass.
- In the window after the merge and before the mint, a clone of main sees `gap ids aiwf sees: G-0001`: the inbox entity has no id.
- `relates_to: <slug>` is refused (`refs-resolve/unresolved`).
- A mint push raced by a merge is rejected with `! [rejected] main -> main (fetch first)`. Re-minting on the new tip gave `G-0002-alpha-login-loop.md G-0003-beta-login-loop.md`.
- aiwf's commit-msg hook refuses the mint commit: `commit-msg refuses aiwf-verb trailer value(s): ["assign-pending"]`.
- Without `aiwf-principal:`, `aiwf check` on main reports `error provenance-trailer-incoherent: commit 914bce0: actor "framework/aiwf-mint" is non-human but aiwf-principal: is missing` (`scripts/e_principal.sh`, `out/e-principal.txt`). With ADR-0001's full trailer block (`aiwf-entity:`, `aiwf-mint:`, `aiwf-principal: human/merger`) check passes, and `aiwf history G-0002` lists the mint commit.

So a minting commit on a protected main needs:

1. a bot identity that the protection lets push to main;
2. a fetch, re-mint and retry loop for the non-fast-forward rejection;
3. `assign-pending` admitted to aiwf's verb set;
4. a human `aiwf-principal`.

---

## 2. Measured facts (assumptions a model would encode)

| # | Fact | Command → observed | Output |
|---|---|---|---|
| 1 | A non-force push is refused when the remote ref moved: `(fetch first)` if the client lacks the new tip, `(non-fast-forward)` if it has it | `git push origin <c>:refs/aiwf/ids` → `! [rejected] … (fetch first)`; after a fetch, `… (non-fast-forward)` | out/b-facts.txt F2, F3 |
| 2 | A second creation of the same new ref is refused; when the creations are concurrent the server says `(reference already exists)` | F1 → `(fetch first)`; races → `(reference already exists)` ×16 | b-facts F1, b-race |
| 3 | Two concurrent updates that both pass the client-side check: the server applies one and rejects the other | `! [remote rejected] … (incorrect old value provided)` ×7 | out/push-errors.log |
| 4 | `--force-with-lease=<ref>:<sha>` is an explicit compare-and-swap | stale sha → `(stale info)`; current sha → `7be6de9..b2eb521` | F4 |
| 5 | `receive.denyNonFastForwards=true` protects only `refs/heads/*`; a forced rewind of `refs/aiwf/ids` is accepted | `refs/aiwf/ids` → `+ 0c49917...744faf4 (forced update)`; `refs/heads/aiwf/ids` → `remote: error: denying non-fast-forward` | F5, F5b |
| 6 | A clone or default fetch does not bring `refs/aiwf/*`; a branch `aiwf/ids` arrives as `refs/remotes/origin/aiwf/ids` | fresh clone refs: `refs/heads/main refs/remotes/origin/HEAD refs/remotes/origin/aiwf/ids refs/remotes/origin/main` | F6 |
| 7 | `--atomic`: one rejected ref rejects every ref in the push | `! [rejected] feature/A -> feature/A (atomic push failed)`; `host has feature/A? no` | F8 |
| 8 | A client killed after receive-pack updated the ref sees failure although the ref moved | `client rc=1; host ledger now: a0bb1d9` | b-lostreply L1 |
| 9 | Two branches adding files with the same id but different slugs merge with no conflict under `--no-ff`, rebase plus fast-forward, and rebase plus `--no-ff` | all `[exit 0]` | out/s0-*-nohooks.txt |
| 10 | Same path: identical blobs merge into one file; different blobs give `CONFLICT (add/add)` | see S0 | out/s0-samepath.txt |
| 11 | The allocator takes max+1 over the working tree ∪ `refs/heads/*` ∪ `refs/remotes/*` ∪ trunk; an id is skipped only while it is visible; a pruned branch's id is reused | visibility table | out/s0-visibility.txt |
| 12 | The allocator reads any `refs/remotes/*` tree, including a ledger's, but not `refs/aiwf/*` | G-0006 / G-0002 / G-0006 | out/b-consume.txt |
| 13 | `aiwf add` cannot be given an id; one large id moves the counter | `G-123456789` → next `G-123456790`; `G-0057` → next `G-0058` | c-shapes, d-mr |
| 14 | `ids-unique` compares the working tree with `origin/main` only; the same id at the same path is silent whatever the content | beta's pre-push passes with a different-content same-path G-0002 on trunk | s0-samepath |
| 15 | The pre-push hook does not fetch; with a stale `origin/main` a trunk collision passes | D2 passes, D3 refused | s0-repair-* |
| 16 | A tree holding both copies is refused by the pre-push hook (`ids-unique`) | maintainer's local merge: `error: failed to push some refs` | s0-noff-hooks |
| 17 | The pre-push hook refuses any pushed ref whose commit is not HEAD | `aiwf pre-push: refusing to push d8104bd…: not the checked-out commit.`; `--no-verify` passes | F7 |
| 18 | The reallocate tiebreaker uses trunk ancestry: on a branch it renumbers the side not on trunk; on main, by id, it refuses | `id "G-0002" is ambiguous (all sides are already on refs/remotes/origin/main); pass a path instead` | s0-repair-on-main |
| 19 | `aiwf reallocate` rewrites every reference to the old id, including those meant for the entity that keeps it (G-0745) | the WRONG rows above | s0-repair-after-*, a-wrong |
| 20 | `aiwf show/list` flag an id absent locally that two refs hold with different blobs; `aiwf check` stays silent unless the id is cited (then a warning) | `cross-branch collision … declining to pick one` | out/s0-readside.txt, b-consume C6 |
| 21 | `aiwf history <id>` lists every commit whose `aiwf-entity:` names the id or one of its prior ids | see the S0 history paragraph | s0-repair-before-merge |
| 22 | The commit-msg hook refuses an unknown `aiwf-verb:` value but accepts any `aiwf-entity:` value | `["assign-pending"]` refused; `G-pb-0012` accepted | e-mint, c-trailers |
| 23 | An unrecognized id shape under `work/` is only an `unexpected-tree-file` warning by default, so pre-commit and pre-push pass | `4 findings (0 errors, 4 warnings)` `[exit 0]` | c-trailers |
| 24 | A plain commit touching an entity is a blocking `provenance-untrailered-entity-commit` error at pre-push once the branch has an upstream | d-mr | out/d-mr.txt |
| 25 | A non-human `aiwf-actor:` without `aiwf-principal:` is a blocking error | `error provenance-trailer-incoherent … aiwf-principal: is missing` `[exit 1]`; with a principal `0 errors` | out/e-principal.txt |
| 26 | Simulated host: a non-force push of main serializes concurrent merges | 10/10 trials, one merge each | out/a-edge.txt |

---

## 3. Defects in aiwf

1. **`aiwf reallocate` rewrites references meant for the entity that keeps the id.** Covered by **G-0745**. Reproduce with `s0_repair.sh after-merge`, `s0_repair.sh after-rebase`, or `s0_merge.sh noff nohooks` (the repair on main). When two losers each run the hint's route, their rewrites of the winner's referrers conflict textually, and the third MR's update stops at that conflict (`a_run.sh wrong seq all`); G-0745's body now records this.

2. **Renumbering one copy of a collision that has landed on main is refused by the pre-push hook.** Filed as G-0746. Repro: `s0_merge.sh noff nohooks`, then in a hooked clone of main:
   ```
   $ aiwf reallocate work/gaps/G-0002-beta-gap.md          # aiwf reallocate G-0002 -> G-0004, exit 0
   $ git push origin main
   error ids-unique/trunk-collision: id "G-0002" is allocated on this branch (work/gaps/G-0002-alpha-gap.md) and on trunk (work/gaps/G-0002-beta-gap.md) …
   $ aiwf reallocate work/gaps/G-0002-alpha-gap.md          # in a fresh hooked clone, the other side instead: refused by the verb, exit 1
   ```
   The rule fires because `origin/main` still holds the duplicate, and the rename exemption (`internal/check/check.go:387`) matches only a trunk path renamed to the branch's path for *the same* id. `s0_repair_branch.sh` shows the routes that do pass. Renumbering both copies, the second clone's first, passes the hooked push of its branch with `0 errors`. A `git push --no-verify` of the one-copy repair lands it, and `aiwf check` then gives `0 errors`. On a main checkout the same duplicate is also reported twice, as `ids-unique` and as `ids-unique/trunk-collision`; that is noise only.

3. **A same-id, same-path collision is never detected by aiwf.** Filed as G-0747. Repro: `s0_samepath_epic.sh`. Two clones each run `aiwf add epic --title "Q3 planning"`, and the host merges both. Main then holds one `E-0002-q3-planning/epic.md` standing for two entities, `aiwf check` gives `0 errors`, and `aiwf history E-0002` shows two `add` commits. With different content, the branch pre-push still passes and only git's add/add conflict catches it. Design A's host rule lets the identical case through (`a_edge.sh` E1). The cause is in `internal/check/check.go` `DisputedTrunkIDs`, which skips a trunk id at the same path whatever the content, and in git, which merges identical adds silently.

4. **After any reallocate, `aiwf history` mixes the two entities' commits.** Filed as G-0748. This happens even with the correct repair order. Repro: `s0_repair.sh before-merge`. `aiwf history G-0002` (alpha's) lists `human/beta add … "Beta gap"` and `human/beta reallocate`, and `aiwf history G-0004` (beta's) lists `human/alpha add … "Alpha gap"`. Archived **G-0037** (addressed) names this symptom ("reallocate splits the audit trail") and its resolution claims the prior-id chain gives one correct timeline; the measurement says otherwise for an id two entities held.

5. *Minor.* For a file whose frontmatter id differs from its path id (`G-1234-2.md`, already an `id-path-consistent` error), `aiwf list --kind gap` lists the entity twice: the second entry carries `"cross_branch_ref":"refs/heads/main"`, the checked-out branch itself. The body-prose warning likewise says `known only on refs/heads/main (not yet merged into this branch)` while on main. Repro: `c_shapes.sh` row mr-dash-sub. Filed as G-0750.

Not defects, but design interactions a model needs: facts 5, 15, 17, 23.

---

## 4. Not measured locally, and what would settle it

- **Real host merge behaviour.** Not measured: GitLab merge trains, "fast-forward only" and "pipelines must succeed"; GitHub's "require branches up to date" and its merge queue; whether a hosted merge is atomic against concurrent merges; squash merges. A test project on each host, with a scripted pair of MRs, would settle it. Design A's guarantee rests on this.
- **Pushing to custom ref namespaces on hosts.** Not measured: whether GitHub and GitLab accept `refs/aiwf/ids`, whether protected-branch rules cover it, and whether forced updates of it can be denied (locally, `receive.denyNonFastForwards` does not cover it). Settle with a test repository on each host.
- **Lost replies over HTTPS/SSH, or through proxies.** Only the local transport, with a killed client, was measured.
- **Contention at scale.** Three clients were measured, not tens. Retry counts and latency under many clients and a real network are unmeasured.
- **D: how MR numbers behave.** Unmeasured: whether a number exists before the branch is pushed; GitHub numbers being shared with issues; GitLab per-project iids; forks; reopened or closed MRs. These need a host. No aiwf verb renumbers to a chosen id, so a real D flow cannot run without new code.
- **E: a mint implementation.** No `aiwf assign-pending` exists, and ADR-0001's `post-merge`/`post-commit` hooks and CI workflow do not either. Whether hosted merges fire anything that could mint, and how a bot obtains bypass rights on a protected branch, need a host plus the new verb.
- **C: what a kernel change would actually require.** The file list comes from grep; building a kernel with a new id grammar and running its tests would settle it.
