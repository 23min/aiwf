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
      status: open
    - id: AC-4
      title: reflog walk uses gitops.LocalBranchRefs; porcelain-only fns annotated
      status: open
    - id: AC-5
      title: doctor reads hook and guidance markers via initrepo; completeHookNames deduped
      status: open
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

`Cancel` (currently defined inside `internal/verb/promote.go`, gating
terminal transitions via `entity.CancelTarget`/`entity.IsTerminal` plus its
own cascade-guard error types) and `Promote`'s epic/milestone
terminal-promote guards independently implement the same "no terminal move
while a child is non-terminal" precondition, each side's comments already
acknowledging the mirroring. The two collapse onto one shared guard function
parameterized by target status, called from both `Cancel` and `Promote`;
`Cancel` moves into its own `internal/verb/cancel.go`.

### AC-4 — reflog walk uses gitops.LocalBranchRefs; porcelain-only fns annotated

`internal/check/reflog_walk.go` independently re-issues
`for-each-ref refs/heads/` instead of consuming `gitops.LocalBranchRefs`
(the isolation-escape-oracle's own divergence is a legitimate perf
optimization and stays as-is; `reflog_walk.go`'s is a plain duplicate).
`gitops.Commit`, `gitops.CommitAllowEmpty`, `gitops.Mv`, and `gitops.Add`
have no production callers — a comment at each definition marks them as
intentional test/porcelain-only APIs (the named "forbidden APIs" the
write-isolation policy checks against), removing the ambiguity for a future
reader.

### AC-5 — doctor reads hook and guidance markers via initrepo; completeHookNames deduped

`internal/cli/doctor/doctor.go` independently hardcodes the literal marker
strings (`"# aiwf:pre-push"`, `"# aiwf:pre-commit"`, `"# aiwf:commit-msg"`,
`"# aiwf:post-commit"`) instead of calling `initrepo`'s already-exported
`HookMarker()`/`PreCommitHookMarker()`/`CommitMsgHookMarker()`/
`PostCommitHookMarker()`, whose doc comments state they exist for exactly
this purpose. The same pattern repeats for the CLAUDE.md-import-marker
check (`initrepo.go`'s `guidanceMarkerLineIdx` vs `doctor/guidance.go`'s
`guidanceImportLinePresent`) and for `completeHookNames`, duplicated
verbatim between `internal/cli/initcmd/initcmd.go` and
`internal/cli/update/hooks.go`. `doctor.go` reads all four hook markers and
the guidance marker via `initrepo`'s exported functions instead of its own
copies, and `completeHookNames`'s duplicate collapses onto one shared
definition both `initcmd.go` and `update/hooks.go` call.

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

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
