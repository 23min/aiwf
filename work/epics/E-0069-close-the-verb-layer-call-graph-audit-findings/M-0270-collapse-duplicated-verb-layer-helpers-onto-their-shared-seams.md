---
id: M-0270
title: Collapse duplicated verb-layer helpers onto their shared seams
status: in_progress
parent: E-0069
tdd: none
acs:
    - id: AC-1
      title: rename and reallocate share one path-rewrite helper with both tail behaviors
      status: met
    - id: AC-2
      title: acknowledge illegal uses gitops's existence helper, not exec
      status: met
    - id: AC-3
      title: Cancel and Promote share one cascade guard; Cancel moves to cancel.go
      status: met
    - id: AC-4
      title: reflog walk uses gitops.LocalBranchRefs; porcelain-only fns annotated
      status: met
    - id: AC-5
      title: doctor reads hook and guidance markers via initrepo; completeHookNames deduped
      status: met
    - id: AC-6
      title: release docs state aiwf upgrade has no automated rollback
      status: open
---
## Goal

Collapse the audit's mechanical duplications onto the shared seams the codebase
already owns, so each duplicated helper exists exactly once.

## Context

Findings F2/F3/F5/F7/F9/F12 of `docs/initiatives/verb-layer-cleanup.md` — all
verified, none requiring a design decision. Each item is a local fold onto an
existing exported helper (`gitops.CommitExists`, `gitops.LocalBranchRefs`,
`initrepo`'s marker functions) or an extraction both call sites already comment
they should share.

## Acceptance criteria

### AC-1 — rename and reallocate share one path-rewrite helper with both tail behaviors

`internal/verb/rename.go`'s `renamePaths`/`substituteSlug` and
`internal/verb/reallocate.go`'s `reallocatePaths`/`substituteID` collapse onto
one shared helper (`internal/verb/pathrewrite.go`), parameterized on which
half (id-prefix or slug) the replacement substitutes and on the verified
semantic fork in the "no second hyphen" fallback: rename appends the new
slug (a slug-less id gains one), reallocate discards and replaces (a
slug-less name has nothing to preserve). The kind-switch/path-join shape
around the substitution — genuinely identical between the two callers — is
itself shared via `rewriteEntityName`, so `renamePaths`/`reallocatePaths`
are now thin callers passing their own substitution closure.

### AC-2 — acknowledge illegal uses gitops's existence helper, not exec

`internal/verb/acknowledgeillegal.go`'s `shaAckable` shells out directly
(`exec.Command("git", "merge-base", "--is-ancestor", ...)` and
`git rev-parse --verify <sha>^{commit}`) instead of calling the already
public `gitops.CommitExists` — the same function `Promote`'s own
`--by-commit` path uses for its existence check. `shaAckable` routes
through `gitops.CommitExists` alone instead of its own `exec.Command`
calls.

Existence, not HEAD-reachability, is the actual acceptance criterion:
reachability implies existence for any SHA git can compute ancestry
against, so a reachable-from-HEAD check can never accept a SHA
`gitops.CommitExists` would refuse, nor refuse one it would accept — it can
only add a second git subprocess call and its own exit-code edge case
(`git merge-base --is-ancestor` exits 128, not 1, for a SHA resolving to
no object at all) with no behavioral payoff. The G-0236 orphan-fallback
case is exactly why existence is the right criterion: its offending SHAs
are by construction unreachable from HEAD, so a reachability gate would
wrongly refuse the very SHAs that case needs acked.

### AC-3 — Cancel and Promote share one cascade guard; Cancel moves to cancel.go

`Cancel` (previously defined inside `internal/verb/promote.go`, gating
terminal transitions via `entity.CancelTarget`/`entity.IsTerminal`) and
`Promote`'s epic/milestone terminal-promote guards independently implemented
the same "no terminal move while a child is non-terminal" precondition, each
side's comments already acknowledging the mirroring. `Cancel` now lives in
its own `internal/verb/cancel.go`; both it and `Promote` call two shared
guards in `internal/verb/cancel_guards.go` —
`epicChildrenCascadeGuard(t, e, newStatus, buildErr)` and
`milestoneACsCascadeGuard(e, newStatus, buildErr)` — parameterized by the
target status (`Cancel` passes its always-terminal `entity.CancelTarget`
result; `Promote` passes its own requested `newStatus`) and by a
caller-supplied `buildErr` closure. The closure is what lets one guard body
serve both callers without collapsing their four distinct, tested error
types (`Epic{Cancel,Promote}NonTerminalChildrenError`,
`Milestone{Cancel,Promote}NonTerminalACsError`) — those carry D-0011's
closed legality-code set and message text; unifying them was a real option
but out of this AC's mechanical scope (no behavior/contract change), so the
closure is the seam that collapses the check without touching the
error surface.

### AC-4 — reflog walk uses gitops.LocalBranchRefs; porcelain-only fns annotated

`internal/check/reflog_walk.go` independently re-issued
`for-each-ref refs/heads/` instead of consuming `gitops.LocalBranchRefs`
(the isolation-escape-oracle's own divergence is a legitimate perf
optimization and stays as-is; `reflog_walk.go`'s was a plain duplicate).
`listRitualHeads` now calls `gitops.LocalBranchRefs` (which returns full
refnames, e.g. `refs/heads/epic/E-0069-...`) and strips the `refs/heads/`
prefix itself to reconstruct the short-name shape the ritual-branch filter
needs — behavior-identical to the old `%(refname:short)` format verb for
every ref under `refs/heads/`, since git's own `:short` semantics for a
plain branch ref is exactly "strip the `refs/heads/` prefix," with no
further compression the way `refs/remotes/<remote>/*` gets.

`gitops.Commit`, `gitops.CommitAllowEmpty`, `gitops.Mv`, and `gitops.Add`
have no production callers — a comment at each definition marks them as
intentional test/porcelain-only APIs (the named "forbidden APIs" the
write-isolation policy checks against), removing the ambiguity for a future
reader. `CommitAllowEmpty`'s doc comment previously claimed it was the path
`aiwf authorize` and `--audit-only` commits ride; that was stale — both
actually route through `gitops.CommitVerbChange`/`CommitTree`'s plumbing
path, which produces an empty-diff commit without needing this porcelain
wrapper. Corrected in place.

### AC-5 — doctor reads hook and guidance markers via initrepo; completeHookNames deduped

`internal/cli/doctor/doctor.go` independently hardcoded the literal marker
strings (`"# aiwf:pre-push"`, `"# aiwf:pre-commit"`, `"# aiwf:commit-msg"`,
`"# aiwf:post-commit"`) instead of calling `initrepo`'s already-exported
`HookMarker()`/`PreCommitHookMarker()`/`CommitMsgHookMarker()`/
`PostCommitHookMarker()`, whose doc comments state they exist for exactly
this purpose. The same pattern repeated for the CLAUDE.md-import-marker
check (`initrepo.go`'s `guidanceMarkerLineIdx` vs `doctor/guidance.go`'s
`guidanceImportLinePresent`) and for `completeHookNames`, duplicated
verbatim between `internal/cli/initcmd/initcmd.go` and
`internal/cli/update/hooks.go`. `doctor.go` now reads all four hook markers
via `initrepo`'s exported functions; `guidanceMarkerLineIdx` is exported as
`GuidanceMarkerLineIdx` and `doctor/guidance.go`'s copy deleted;
`completeHookNames`'s duplicate collapses onto
`cliutil.CompleteHookNames`, called by both `initcmd.go` and
`update/hooks.go` (their byte-identical duplicate tests consolidated into
one, in `cliutil`).

A mutation probe on the marker-drift risk (change a marker constant,
check whether detection and hook-writing silently drift together) found
a stronger safety net than expected: `internal/initrepo`'s byte-golden
hook-script fixture test independently pins the correct marker text, so
a constant typo is caught even though detection and writing now share
the same source. No mechanical backstop, however, would catch a future
fifth hook marker added to `initrepo` without a corresponding `doctor`
detection site wired up — the same "sin of omission" class F8 needed a
presence-check policy for; out of this AC's mechanical scope.

### AC-6 — release docs state aiwf upgrade has no automated rollback

`aiwf upgrade` delegates the entire fetch/verify/place sequence to a single
`exec.Command("go", "install", ...)` call — a reasonable minimalist design,
not an oversight, but one that ships no aiwf-level backup-old-binary step,
so a broken newly-installed binary has no automated rollback path (the
operator would have to manually `go install <pkg>@<older-tag>`). The
release-process documentation states this property explicitly, so a
"cut a release" / "aiwf upgrade" conversation doesn't wrongly assume
rollback exists.

## Constraints

- Pure refactors: no behavior change; existing tests stay green and each fold
  lands with a referencing test or rides one that pins the seam.
- The `dupl` tripwire (G-0423) stays green without new baseline entries.

## Design notes

- F2's shared path-rewrite helper parameterizes the "no second hyphen" branch —
  rename appends the new slug, reallocate discards and replaces; the verified
  semantic fork must survive the merge.
- F5 moves `Cancel` into its own `internal/verb/cancel.go` alongside the shared
  cascade guard.

## Out of scope

- The FinishVerb/envelope triad (its own milestone).
- The contract-gate and rewidth-sweep judgment calls (decision entities, not
  builds).

## Dependencies

- None — parallel-safe with the bug-fix milestone.

## References

- `docs/initiatives/verb-layer-cleanup.md` §F2/§F3/§F5/§F7/§F9/§F12; G-0423.

---

## Work log

### AC-1 — shared path-rewrite helper

renamePaths/reallocatePaths now both call substituteNamePart +
rewriteEntityName (internal/verb/pathrewrite.go); dupl baseline
exclusion removed · commit 0437f95 · tests 11/11 new (7 table cases +
4 subtests), full suite green

### AC-2 — acknowledge illegal uses gitops's existence helper

shaAckable routes through gitops.CommitExists alone (initial
gitops.IsAncestor + gitops.CommitExists version simplified after a
wf-vacuity mutation probe showed the ancestry check redundant and
retitled the AC accordingly); AC-4's out-of-history test updated to
match · commit 2d59566a · full acknowledgeillegal suite green (11
tests)

### AC-3 — shared cascade guard, Cancel moved to cancel.go

epicChildrenCascadeGuard/milestoneACsCascadeGuard added to
cancel_guards.go; Promote's two inline cascade checks and Cancel's
switch-based one now call them; Cancel moved out of promote.go into
new cancel.go · commit 74656ea5 · full existing Cancel/Promote test
suite green (100 tests), wf-vacuity mutation probe 3/4 caught (1
benign survivor: the epic kind-check is redundant with
nonTerminalEpicChildren's data shape but kept for readability/
future-proofing, unlike AC-2's ancestry check it costs nothing)

### AC-4 — reflog walk on gitops.LocalBranchRefs; porcelain-only fns annotated

listRitualHeads routes through gitops.LocalBranchRefs +
strings.TrimPrefix; Commit/CommitAllowEmpty/Mv/Add annotated
test/porcelain-only, CommitAllowEmpty's stale comment corrected ·
commit 7e5a6e6e · full reflog/orphan/gitops suite green, wf-vacuity
mutation probe 3/3 real mutations caught

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
