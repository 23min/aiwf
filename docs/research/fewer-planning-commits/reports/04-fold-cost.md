# Fold cost and side-ref alternative — experiment record (2026-11-07)

Saved by the parent from the experimenting agent's returned record (the agent
could not write files). Scripts: `scripts/`; runs: `runs/`; transcripts: `out/`.

## Environment

git 2.51.1, OpenSSH 9.6p1, go 1.25.12, Python 3.12.3, Linux 7.0.14-linuxkit.
aiwf built from `4343f2bd1` (main) via `git archive HEAD`, `-buildvcs=false`
(`bin/aiwf`). Each run: fresh dir, bare `remote.git`, clone with `aiwf init` +
hooks, isolated HOME, `GIT_CONFIG_NOSYSTEM=1`, no ssh-agent.

## Experiment 1 — re-creating code commits

Fixture (`setup1.sh`): main pushed with E-0001, M-0001 (AC-1), G-0001; unpushed
`epic/E-0001-spike-epic`: C1 code; V1 promote M-0001 in_progress; C2 code; V2
authorize M-0001 --to ai/claude; C3 "fix: c3, follow-up to <C2 sha>"; V3 promote
M-0001/AC-1 met as ai/claude; V4 promote G-0001 addressed --by-commit <C2>; H hand
edit of E-0001 body (plain commit); A acknowledge illegal <H>; C4 code; M merge of
origin/main; C5 code. `AGENT=0` variant: no V2, V3 as human.

Fold (`fold.sh`, plumbing): pre-first-verb commits keep SHAs; later non-verb
commits re-created with `merge-tree --write-tree --merge-base` + `commit-tree`
(merges re-made with original second parent); verb changes applied in order as
one tip commit with repeated trailer groups; CAS update-ref. No-agent run: 4
planning commits folded, 6 commits re-created, final tree identical, patch-ids
identical, author dates kept.

1.1 SHA references: after the fold `addressed_by_commit`, `aiwf-force-for` and
the SHA in C3's message are no longer contained in HEAD; in a `--no-local` clone
of the pushed fold both SHAs fail `cat-file`; `aiwf show G-0001` neither shows nor
flags the dangling `addressed_by_commit`.

1.2 Checks (no-agent): before fold 0 errors (full and --since). After: full 0
errors (audit skipped, provenance-untrailered-scope-undefined); --since 1 error
provenance-untrailered-entity-commit on re-created H. First hooked push (-u)
passed; after `--no-ff` merge to main the hooked push passed (untrailered audit
walks first-parent, skips ordinary merges). Controls: upstream=origin/main →
refused; unfolded branch merged → 0 errors. Agent fixture adds
provenance-authorization-missing and two provenance-trailer-incoherent errors
(from putting several operations in one commit, not from re-creation); its push
is refused.

1.3 Signing (ssh ed25519, commit.gpgsign=true): fold with -S → all verify;
without -S → 7 new commits unsigned, exit 0, no warning (commit-tree ignores
commit.gpgsign). Key absent with -S → fold fails cleanly, branch unchanged; key
absent without -S → fold succeeds unsigned. `aiwf promote` with key absent fails
exit 3.

1.4 Forks (`forks.sh`): branch at C3, second worktree at V4 (code commit + add
gap), stash on pre-fold tip; after the fold, merging them records
M-0001 in_progress, AC-1 met, G-0001 addressed and the acknowledgment twice;
5 code commits duplicated by patch-id of 21; `aiwf check` reports nothing; hooked
push 0 errors; `aiwf history M-0001` lists the promote twice. `git stash apply`
clean.

1.5 Intermediate states (`inter.sh`): tests pass on all; re-created commits show
M-0001 draft / G-0001 open where originals show in_progress / addressed; --since
check on re-created H, C4, M, C5 carries the untrailered error.

1.6 Misc (`misc.sh`): tags stay on old commits (`describe` fails; v0.1 reaches 4
per-verb commits); hooked tag push refused (general rule); notes lost even with
notes.rewriteRef; post-rewrite never fires; bisect ranges differ; `aiwf history`
shows one fold row with time = fold time.

1.7 A plain-commit hand edit of text a folded verb created → fold conflicts and
refuses; §4.2's mixed-commit refusal does not cover it.

1.8 A rewriting fold (`fold-rw.sh`) mapping old SHAs to new in messages, trailers
and planning files → --since 0 errors, references resolve; short SHAs in
subjects stay stale; final tree differs by the rewritten line.

## Experiment 2 — planning commits on a side ref

Simulation (`sideverb.sh`, `submit.sh`) around the unmodified binary. Clean path
works: code SHAs kept, one submit commit, --since 0 errors, push passes.
Hazards observed:
- `git status`: one line per pending op (skip-worktree hides it, and hides diffs).
- `git add -A && commit`: pending files swept into a code commit, hooks accept;
  --since 2× untrailered; submit then makes an empty commit carrying both ops'
  trailers; first push passes.
- `commit -a`: sweeps tracked pending files, not untracked → split state.
- merge/pull --rebase/rebase refused when touching pending files.
- stash: reads revert; next side verb refused by claim guard; pop restores.
- checkout: pending carried to another branch, which then reports it; second chain.
- worktree add: new worktree does not see pending state.
- unmodified verbs: promote/retitle refused (guardClaimPaths), add ac refused
  (checkUncommittedConflict), edit-body refused; `edit-body G-0002 --body-file`
  committed the pending new entity onto the branch → submit add/add conflict;
  `add gap` allocated G-0003 seeing the untracked G-0002.
- readers: history lacks pending ops while status shows them.
- reset --hard / clean -fd: lose working files; side ref recovers them.
- push between verbs: code only; hook checked a tree whose planning was not pushed.
- conflict with main resolved in the working tree is not in the side ref →
  submit conflicts again (open).
- `git branch -m`: side ref orphaned.
skip-worktree closes status noise and sweeps but dead-ends merge/checkout/rebase
against a changed pending file ("No local changes to save"), needs re-setting
after every verb, and does not affect aiwf's HEAD-comparing guards.

## Claims

Verified: everything above as recorded. Argued: a fold can rewrite references
inside the branch but not outside it (tags, notes, clones, CI, trackers), not
short SHAs, not pointers to a folded planning commit; re-signing attests the
folder; fork duplicates need an operation identity a check reads; closing the
side-ref hazards touches every HEAD-comparing guard, every git-log reader,
checkout/rename/worktree handling, a recovery verb and submit-time conflicts.
Not settled: host acceptance of re-signed commits; locked agents/hardware keys;
folding second-parent (milestone-into-epic) per-verb commits; agent-fixture
intermediate states; time on long branches; the design's refusals and op
identity (not implemented); a real side-ref aiwf.
