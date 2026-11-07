# Independent review of commit-noise-options.md — findings to build on

Source: a fresh-context review on 2026-11-07. Its scripts are in
`scratchpad/cn/`, its experiments in `scratchpad/cnx/` (scratchpad =
/tmp/claude-1000/-workspaces-aiwf/eb2423dd-a702-4bd8-a308-1404db3d3991/scratchpad).
Window: `6b87e0f2..4343f2bd1` (oldest `origin/main` reflog entry to HEAD).

## Corrected amend eligibility (candidate 3)

A verb amends HEAD only when all hold:
- HEAD is a non-merge commit whose operations are all aiwf verb operations
  (not a ritual commit made with plain `git commit --trailer aiwf-verb: ...`;
  ritual verbs are listed in `internal/.../ritual_trailer_verbs.go`), by the
  same actor (and principal);
- no ref other than the current branch, and no other worktree's HEAD, contains
  HEAD, and no remote-tracking ref contains it;
- nothing the new operation writes (trailers or file bytes) names HEAD's SHA.

Known holes even then: a pushed commit may be reachable from no remote-tracking
ref (`--single-branch` clone, `git push <url>`, pruned tracking ref, lost push
reply). Then the next push is refused as non-fast-forward, and `git pull` would
land the operation twice silently.

## Measured risks (reviewer's experiments)

- E2: a verb on a branch cut from an unpushed main commit, amending that commit
  on the branch only, records the operation twice after the merge; `aiwf check`
  reports 0 errors and the hooked push succeeds; `aiwf history` lists it twice,
  the second as a garbled multi-operation row. The start-milestone ritual
  (promote + authorize on main, then cut the milestone branch) creates this shape.
- E3: in a `--single-branch` clone, pushes reach the remote but no
  `refs/remotes` ref contains HEAD.
- E1 (candidate 2): a second verb on a file with an uncommitted verb write is
  refused by `checkUncommittedConflict` (internal/verb/apply.go) and
  `guardClaimPaths` (claimguard.go:150).
- E5 (candidate 2): `git merge` refuses over pending changes; a new worktree
  does not see them.

## Argued risks

- An operation can name the SHA it would replace: authorize then `--end` or a
  terminal promote in one run (`aiwf-scope-ends` holds the authorize commit SHA,
  authorize.go:555; scopes matched by equality, scope.go:190) — the scope never
  ends; `aiwf-branch-sha` (cli/authorize/authorize.go:230, 389); `--by-commit`
  of a planning commit (promote.go:671; check.go:991 only checks non-empty).
- `readActorOpenerEntities` (internal/cli/cliutil/provenance.go:205,229) reads
  `aiwf-entity` into one field; a multi-operation commit breaks it.
- `SortedTrailers` (gitops/trailers.go:141) has no production caller although
  provenance.go:103 says trailers are re-sorted on emit; for a multi-operation
  commit, trailer order is what groups operations.
- Amend loses each operation's commit time unless the record carries it;
  `git reset HEAD~` undoes a run; an earlier verb's `commit_sha` becomes
  unreachable; neither research document models amend.
- Fold re-creates 448 code commits and 106 merges in the window, re-signed; a
  fold of commits a child branch was cut from lands operations twice.

## Counts (planning commits left of 1,883 non-merge planning commits)

- fold ≈348 (≈235 if ritual commits could fold), batch ≈374 (≈261),
  amend 616–677. 114 ritual-stamped non-merge commits; 77 touch only ROADMAP.md.
- No agent actor appears in the window (every aiwf-actor is human/peter).
