# Fewer planning commits in future history: three candidate mechanisms

Goal (the maintainer's): future git history with far fewer aiwf planning commits,
for every reader (local `git log`, GitHub's commit list). Past history is not
touched. Criteria: future-proof, pure, most benefit for the cost; a kernel change
is also a chance to revisit and improve the code it touches.

Today every mutating verb makes exactly one commit (kernel commitment 7 in
CLAUDE.md), built by `internal/gitops/committree.go` `CommitTree` against a
throwaway index seeded from HEAD's tree, parent HEAD, `update-ref` with
compare-and-swap; `internal/gitops/verbcommit.go` `CommitVerbChange` wraps it.

## The three candidates

1. **Fold before push.** Verbs commit as today. An explicit step before pushing
   (on `main`) or before merging a branch rewrites the operator's own unpushed
   commits so the planning commits become one commit at the tip, carrying one
   trailer group per operation. Code commits after the first folded verb are
   re-created. Designed in `docs/research/collaborative-state-and-commit-granularity.md`
   §4.2 (read it), modelled in `docs/research/transactions/model/RESULTS.md`.
   A pre-push hook cannot fold: measured in a scratch repo, a pre-push hook that
   rewrote the branch changed the local branch but the push still sent the
   original commits. So the fold is an explicit step; the hook can only refuse.

2. **Batch at write time, no rewriting.** Verbs write their file changes but do
   not commit. Each operation's record (the trailers it would have written) is
   kept somewhere until the operator commits the batch as one commit. The
   2026-10-24 seed (`docs/research/collaborative-state-research-specification-2026-10-24.md`
   §5.1 "batched Git") is the origin; the newer research does not weigh it
   against the fold.

3. **Coalesce into the unpushed tip.** A verb amends HEAD instead of adding a
   commit when HEAD is a planning commit made by the same operator and not yet
   reachable from any remote-tracking ref; otherwise it makes a new commit as
   today. Code commits are never rewritten. One planning commit per run of
   consecutive verbs between code commits or pushes.

All three produce commits that carry several operations, so every reader of
history must learn a multi-operation commit (research §4.2 consequence 2 and
§10 questions 2–3: e.g. `fsm-history-consistent` compares each commit with its
parent).

## Measurements (window 2026-08-26 .. 2026-11-05, this clone's push log of origin/main)

- Full history reachable on `main` in the window: 2,603 commits, 1,883 carrying
  `aiwf-verb`. `main`'s first-parent line: 591 (484 planning); the rest came in
  through 70 merges onto `main`, from 115 branches counting milestone branches
  merged into epic branches (1,925 non-merge commits: 1,399 planning, 526 other).
- Planning commits that would remain:
  - candidates 1 and 2: one per push on `main` (84 pushes carried planning
    commits) plus about one per branch (up to 115): about 199;
  - candidate 3: one per run of consecutive planning commits: 91 on `main`'s
    line within pushes plus 332 on branches: 423.
- Candidate 2's exposure: code commits made on the same branch or in the same
  push after a planning verb and before the batch would commit, i.e. while
  planning changes would sit uncommitted: 431 on branches, 17 on `main`.
  The research measured the hazard (§8, E3): uncommitted planning files were
  swept into a code commit by `git add -A`, the commit hooks accepted it, and
  only `aiwf check` at push raised `provenance-untrailered-entity-commit`.
- Not measured: whether any branch was pushed before it merged (candidate 1
  leaves pushed planning commits unfolded; candidate 3 starts a new commit).
- One author made every commit in the window.

## Code facts

- Verbs read entity state from the working tree (`tree.Load`), so pending
  file changes are visible to the next verb under all three.
- Active authorization scopes are read from `git log` at verb time
  (`internal/cli/cliutil/provenance.go` `loadActiveScopesForActor`,
  `readActorOpenerEntities`), not from the working tree.
- The red-first gate (`internal/verb/promote_phase_gate.go`) inspects the dirty
  working tree and excludes `work/**` and `docs/**`.
- Production code that reads git history directly (excluding `internal/policies`
  and `internal/stresstest`): `internal/cli/status/{worktrees,status}.go`,
  `internal/gitops/{gitops,refs,revwalk,catfile}.go`,
  `internal/cli/cliutil/{scopes,provenance}.go`, `internal/cli/check/{provenance,isolation_escape_oracle}.go`,
  `internal/check/{reflog_walk,entity_body_section_dropped,post_cutoff,orphan_dag,instruction_fence,id_rename_untrailered,head_history,fsm_history_consistent,area_mistag,acks}.go`,
  `internal/entityview/{scopes,historyevent}.go`, `internal/cli/show/show.go`.
- ADR-0056 (accepted, not built) replaces the authorize-commit SHA in
  `aiwf-authorized-by` / `aiwf-scope-ends` with a random scope id carried by the
  opening commit, so a rewrite of that commit no longer breaks the pointers.
