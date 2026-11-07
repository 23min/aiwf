# Model checker: amend designs (2026-11-07)

Saved by the parent from the modelling agent's returned record (the agent could
not write files). Results: research worktree
`docs/research/transactions/model/RESULTS-amend.md`; runs and traces:
`model-amend/run/`; baseline: `model-amend/orig/`.

## Changes (uncommitted, research worktree, docs/research/transactions/model/)

- `model.py`: eight amend designs `amend.<rule>.[opid.]<audit>` (rule naive|safe;
  SHA or operation-id pointers; audit parent|record). naive = HEAD a non-merge verb
  commit, same git author and aiwf-actor, no remote-tracking ref contains it; safe
  = naive + no other local branch contains HEAD + the new op names HEAD's SHA in
  neither a trailer nor written bytes. New steps: same-actor `edit-body G`,
  `authorize --end`, scope-ending `promote M done`, `--by-commit` naming a verb's
  own commit. New conditions: `sa-*`, `fork`, `reuse4`, `untracked`, `scope`,
  `actor`, `byc`, `twostep`, `nr-*` controls (operators never rebase). Fork support.
  Amended commit records operations as the fold's planning commit does (one group
  per op, in order, each with identity, actor, pointers); kind stays `verb`.
- `run.py`: `--amend` adds the designs, conditions and probes AP1, AW1; without it
  the original matrix runs.
- `make_tables.py`: reads only original designs/conditions/probes so RESULTS.md
  regenerates identically.
- New: `make_amend.py`, `RESULTS-amend.template.md`, `RESULTS-amend.md`.
- RESULTS.md, RESULTS.template.md, fidelity.py, rule_w2.py, gitchecks/ unchanged.

## Regression

Unmodified-copy pipeline reproduces RESULTS.md (timings aside). `run.py --amend
--jobs 12`: 740 s, 1,689 new explorations, 0 capped, 0 cyclic, peak 211 MB.
18,446 baseline rows: 0 missing, 0 different; 12,878 new rows. fidelity.json
byte-identical; 1,536 rule_w2 rows identical. Python 3.12.3, Linux, 16 CPUs.

## Results

Original conditions: amend changes no cell (fires only in `adr`, harmlessly).
Cells where amend fails and its per-verb counterpart holds:
- naive: nr-fork T4d, nr-byc T6, nr-scope T6 (SHA pointers), untracked T4d,
  sa-after/sa-bob T4d;
- safe: untracked T4d, sa-after/sa-bob T4d;
- either rule with the `parent` audit: twostep L✗ on every host setting.
`amend.safe.opid.record` vs best fold: keeps the original after/bob cells, nr-fork
and nr-byc that the fold fails; fails `untracked`, which the fold keeps; shares
per-verb's weaknesses (no transaction tie under ff/direct, reuse and cherry
duplicates). Failing cells over the amend conditions: 445 vs 437
(perverb.opid.record) vs 435 (best fold).

## Counterexample classes amend adds

1. Branch cut from an unpushed commit (nr-fork, naive, 6 steps) — E2.
2. Op names the SHA it replaces in a trailer (nr-scope, naive + SHA pointers, 4
   steps): scope never ends.
3. Same in file bytes (nr-byc, naive, 4 steps): dangling addressed_by_commit;
   ADR-0056 does not cover it.
4. Push leaving no tracking ref (untracked, both rules, 8–9 steps): duplicate.
5. New: the operator's own rebase of a pushed verb commit (sa-after/sa-bob, both
   rules, merge/semi/ff hosts, 15–16 steps): the rebased commit has a new SHA no
   ref contains, the next verb amends it, the old push survives, the op lands
   twice. Every clause of both rules is a SHA/ref-containment test (the fold's W1
   rule failed the same way).
6. twostep (parent audit, 7 steps): in_progress then done read as draft → done;
   pre-push refuses forever. The `record` audit fixes it.
7. Actor switches: neither rule amends across one; probe AP1 (actor test removed)
   gets stuck on provenance-authorization-missing; AW1 shows the SHA clause alone
   holds.

## Claims

Verified: baseline unchanged; every row and class above as model output; sa-*
conditions change nothing for non-amend designs (840 cells); no amend run
rewrites a code commit (T5) or drops an operation.
Argued: model amend matches `git commit --amend`; no check rule for a scope-ends
naming no opener (scope.go:190); aiwf-branch-sha same class as byc (not modelled);
sa-cherry T4a vs T4d is not a protection.
Not settled: whether a patch-id test (`git cherry` against remote-tracking refs)
closes the rebase hole (`run.py --amend --only sa-after,sa-bob,untracked`, minutes);
what closes `untracked`; reader effects the model assumes away; commit counts.

## Also found

Scope ends (G-0752): today's per-verb commits fail T6 in `scope` with SHA
pointers; operation-id pointers fix it except under an all-messages squash.
Per-verb fails T4d in `fork` when the cut branch is rebased onto origin/main.
Best fold fails T4d in fork/nr-fork and T6 in byc/nr-byc.
