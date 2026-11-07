# Amend prototype — scenario record and risk register (2026-11-07)

Saved by the parent from the experimenting agent's returned record (the agent
could not write files). Scenarios: `scenarios/` (lib.sh, ops.py, s01..s10);
outputs: `runs/s01.out`..`runs/s10.out` with temp repos beside them;
`gotest-default.log`, `gotest-amend-on.log`, `bin/aiwf`.

## Environment

git 2.51.1; go1.25.12 linux/amd64; Linux 7.0.14-linuxkit devcontainer. Binary
`amend-proto/bin/aiwf` (sha256 prefix 08d249b2f404d4a1), version
`v0.42.1-0.20261105192620-4343f2bd1323+dirty`, built from the research worktree
at 4343f2bd1 plus the uncommitted prototype (diff hash 2bc752b6c459eb67). Each
scenario: fresh mktemp dir, own HOME, GIT_CONFIG_NOSYSTEM=1, own bare remote,
`aiwf init --no-prompt`, one pushed init commit; hooks resolve the prototype via
PATH.

## Prototype (uncommitted in the research worktree)

- `internal/gitops/committree.go` (+9): `CommitTree` calls `tryAmend` when
  `AIWF_EXPERIMENT_AMEND=1` (61-65); failure-injection hook between commit-tree
  and update-ref (149-151), inert unless its variables are set.
- `internal/gitops/amend_experiment.go` (new, 399 lines).
- Env: `AIWF_EXPERIMENT_AMEND=1`; `AIWF_EXPERIMENT_AMEND_RULE=naive` for
  comparison; `AIWF_EXPERIMENT_AMEND_DELAY`, `AIWF_EXPERIMENT_AMEND_FAIL`.
  Decisions traced to stderr (`aiwf-amend: amended …` / `new commit: <reason>`).
- Eligibility (`amendEligibility`, 171-264): HEAD on a branch with one parent;
  trailer paragraph starts with aiwf-verb, one aiwf-verb per group, none a ritual
  verb; every group's actor and principal equal the new op's; `for-each-ref
  --contains HEAD` returns no ref but the current branch (covers branches, tags,
  remote-tracking refs, refs/stash); no other worktree HEAD contains HEAD; HEAD's
  7-char prefix appears nowhere in the new op's subject, body, trailers or bytes.
- Amend: temp index from HEAD's tree; commit-tree with HEAD's parent and HEAD's
  author; `update-ref -m "aiwf amend: <subject>" HEAD <new> <old>` (CAS).
- Record format (amend_experiment.go:1-45): subject `aiwf: <n> operations (<verb>
  <entity>, …)`; body `[i] <op subject>` + op body per op; one trailer paragraph
  of groups, each starting at aiwf-verb and ending with `aiwf-op-time: <RFC3339>`.
  Defect: drops non-trailer lines of HEAD's last paragraph (cherry-pick marker).
- Tests: default env `go test -count=1 ./internal/gitops/... ./internal/verb/...`
  → ok (8.465s, 37.570s). With amend on: gitops ok; 4 verb tests fail, each
  asserting one commit per verb (TestAuthorize_Pause_HappyPath,
  TestAddACBatch_EmitsOneEntityTrailerPerAC, TestAddACBatch_SingleTitleUnchanged,
  TestMove_RoundTrip). Not run: make lint, internal/policies, go test ./... .

## Scenarios (summary; decisive outputs in runs/)

- S01 runs on main + push: new commit after push / after code commit; remote has
  3 aiwf commits for 9 operations; hooked pushes pass; check 0 errors.
- S02 E2 under corrected rule: every "another branch holds HEAD" case made a new
  commit; 12 operations, 12 distinct. (a) agent refused — authorize amended into
  the promote (R1); (b) agent works (authorize last in its commit).
  `isolation-escape-orphaned-ai-commit` warnings (R13). (c1) a clone of the
  working repo → duplicate add G-0002, 11 warnings 0 errors, push accepted (R4).
- S03 authorize alternation (control vs amend): start-milestone ritual → agent
  refused, no Scopes section; pause+resume amended → events 2 vs 4; authorize then
  --end / terminal cancel → corrected rule makes a new commit (names HEAD); naive
  rule → scope-ends names a dead SHA, agent acts after --end; a human retitle
  amended after authorize → agent refused (out-of-scope). Final: 0 errors, 31
  distinct ops, no duplicate.
- S04 git ops: second worktree / detached worktree / stash → new commit (amend
  resumes after removal/pop); reset --soft HEAD~ undoes the run (a plain commit of
  the leftover → provenance-untrailered error); reset --mixed → next verbs refuse;
  reset --hard → run lost, reflog recovers; pull --rebase fine; pull (merge) → new
  commit; cherry-pick -x amended with marker dropped (R17). Side note: in a linked
  worktree the repo lock falls back to <root>/.aiwf.lock (repolock_unix.go:101-116).
- S05 pushed-but-untracked: single-branch clone affects only branches outside the
  fetch refspec; push to URL / lost reply / deleted tracking ref → amend, push
  rejected; then merge-pull → silent duplicate, check ok, push accepted;
  rebase-pull → commit claims 2 ops while its diff holds one.
- S06 crash/concurrency: injected failure → HEAD unchanged, status clean; kill -9
  leaves the same state as on the new-commit path (not amend-specific); lock
  serializes; git commit in the window → CAS refuses, verb rolls back; git branch
  in the window → silent duplicate; git push in the window → remote gets pre-amend
  commit, next push rejected.
- S07: NoOp unchanged; failed validation writes nothing; add+reallocate+
  set-priority in one commit, check 0 errors, history garbled; collision
  reallocate makes a new commit; cancel+archive one commit, 0 errors; STATUS.md
  drops G-0007 from "Gaps opened", table row splits.
- S08 readers: (a) promote active then done in one run → fsm-history-consistent
  illegal-transition proposed→done, push refused (control 0); (b) adding a forced
  op to the same run → 0 errors for all five ops, E-0002's error gone, push
  accepted; (c) TDD ladder → 1 history event at red's time, JSON verb
  "promote\npromote\npromote\npromote".
- S09 rule edges: --by-commit HEAD → corrected rule new commit; naive amends,
  dangling, 0 errors; a printed commit_sha becomes unreachable after the next
  amend; a hand-made `git commit --trailer aiwf-verb: promote` with code is
  accepted by commit-msg and then amended by the next verb (R16); ritual commit →
  new commit.
- S10 E2 naive → duplicate promote, 0 errors, push accepted; corrected → none.
- Cost: `git for-each-ref --contains HEAD` 463-641 ms in the research worktree
  (13,119 commits, 65 refs, commit-graph of 2026-09-29); `git worktree list`
  15-17 ms.

## Risk register

"Reader" = today's readers taught to split a commit into per-operation groups.

| id | risk | obs/arg | what fails; reported? | corrected rule prevents | remains with reader |
|---|---|---|---|---|---|
| R1 | authorize inside a multi-op commit invisible to the allow rule (provenance.go:207,232; scopes.go:68,129) | obs | agent refused at next verb; nothing reports at authorize/push | no | no |
| R2 | pause+resume in one commit keeps only the last event | obs | events 2 vs 4; silent | no | no |
| R3 | an op names the SHA it would replace (scope-ends, --by-commit HEAD, aiwf-branch-sha) | obs (naive) | scope never ends, agent acts after --end, dangling refs; silent | yes | yes |
| R4 | another clone holds the unpushed commit | obs | silent duplicate; push accepted | no | yes |
| R5 | pushed commit no tracking ref contains | obs | push rejected; then silent duplicate or a commit claiming an op its diff lacks | no | yes |
| R6 | race between eligibility check and update-ref | obs (3s injected) | silent duplicate or rejected push | no | yes |
| R7 | two transitions of one entity in one commit read as one jump | obs | false-positive error; push refused | no | no |
| R8 | aiwf-force exemption is per commit | obs | an unrelated forced op suppressed others' errors; silent | no | unless force scoped per group |
| R9 | readers print newline-joined values (history, show, status, JSON, STATUS.md) | obs | garbled output, JSON contract changes; silent | no | no |
| R10 | status digest classifies by c.Verb == "add" (status.go:1017) | obs | entities missing from "Gaps opened"; silent | no | no |
| R11 | show hides the scope (show/scopes.go:48, scopeguard.go:63) | obs | no Scopes section; silent | no | no |
| R12 | per-op times collapse; TDD ladder one event | obs | evidence collapses; silent | no | partly |
| R13 | amended agent commit reported isolation-escape-orphaned-ai-commit (reflog_walk.go:297) | obs | persistent false warning | no | yes |
| R14 | reset HEAD~ undoes a whole run | obs | soft: provenance error; mixed: verbs refuse; hard: lost, reflog recovers | no | yes |
| R15 | earlier printed commit_sha unreachable (commitment 7) | obs | dead references; --by-commit refuses | no | yes |
| R16 | hand-made commit with a kernel-verb trailer and code gets amended | obs | code commit rewritten; silent | no | yes |
| R17 | format drops foreign trailer-paragraph lines | obs (prototype defect) | cherry-pick marker lost | n/a | until fixed |
| R18 | crash mid-amend | obs | same as new-commit path | n/a | n/a |
| R19 | concurrency | obs | no failure found | n/a | n/a |
| R20 | for-each-ref --contains cost | obs | ~0.5 s per verb | n/a | yes |
| R21 | E2 | obs | naive: silent duplicate | yes | yes (rule stays) |
| R22 | not tested: gpg, post-rewrite/notes, render html, Windows | — | not settled | — | — |

R4, R5 and R6 end in a silent duplicate operation that `aiwf check` and the
pre-push hook accept; no current check detects an operation recorded twice.

## Not settled

Frequency of R4/R5/R6 in practice; real width of R6's window; --contains cost
with a fresh commit-graph; render html of multi-op commits; signing,
post-rewrite/notes, Windows.
