---
id: M-0272
title: Extract the read-side helpers into a neutral entityview package
status: in_progress
parent: E-0069
depends_on:
    - M-0269
    - M-0270
    - M-0271
tdd: none
acs:
    - id: AC-1
      title: read-side helpers live in a neutral package free of CLI dependencies
      status: met
    - id: AC-2
      title: render, check, status import the neutral package, not sibling CLI packages
      status: open
---
## Goal

Give the read-only verbs a neutral shared library: extract the verified
Cobra-free read-side helpers out of `show`/`history` into a neutral package
that `render`, `check`, and `status` consume instead of importing sibling
`internal/cli` verb packages.

## Context

Finding F6 (deep-dived and quantified): ~638 lines / ~16 exported symbols of
the combined `show`/`history` surface are already pure — `scopes.go` wholesale,
`EventFromCommit`, `ReadEntityBody`, and the `HistoryEvent`/`ReadHistory`/
trailer-parsing half of `history.go`. The rest is genuinely Cobra-specific and
stays. Three consumers (`render`, `check`, `status`) currently reach into
sibling CLI packages for this logic; the acyclic property survives only because
nobody has yet added the closing edge.

## Acceptance criteria

### AC-1 — read-side helpers live in a neutral package free of CLI dependencies

`internal/entityview` holds the full F6-scoped surface: `HistoryEvent`,
`ReadHistory`/`ReadHistoryChain` (plus their `ShortHash`/`StripTrailers`/
`SplitMultiValueTrailer`/bare-milestone-id trailer-parsing helpers),
`EventFromCommit`, the scope predicates `HasOwnScope`/`HasAuthorizedBy`/
`HasScopeData`, `ScopeView`/`AssembleScopeViews`/`LookupCommitDateCached`/
`LastEventSHA`, and `ReadEntityBody`. `go list -deps` on the package confirms
its only `23min/aiwf` dependencies are `internal/entity`, `internal/gitops`,
`internal/scope`, and `internal/codes` — no `internal/cli/*`, no Cobra.

The line drawn: a helper moves only if it needs no `internal/cli/cliutil`
(or other CLI-package) call. Two git-shelling orchestrators stay behind in
their original CLI packages because they compose the moved primitives with
`cliutil`'s scope-FSM replay (`cliutil.LoadEntityScopes`,
`cliutil.AuthorizeOpeners`) — `show.LoadEntityScopeViews` (now a thin
wrapper calling `entityview.ReadHistory`/`HasOwnScope`/`HasAuthorizedBy`/
`AssembleScopeViews`/`LookupCommitDateCached`) and `history.ScopeMapFor`
(calls `entityview.HasScopeData`). `history`'s own text-rendering
(`RenderTo`, `RenderActor`, `RenderScopeChips`) also stays — Cobra-free
itself, but not part of F6's reuse surface (no render/check/status call
site ever needed it) and genuinely specific to `aiwf history`'s text
output. `entityview.ReadHistoryChain` needed the same empty-repo guard
`cliutil.HasCommits` provides; rather than import `cliutil` (which would
have dragged Cobra back in transitively via its own `completion.go`), a
private `hasCommits` duplicate lives in `entityview` — see D-0045.

Evidence: `go build ./internal/entityview/...` plus `go list -deps` (no
`internal/cli` in the closure); the full pre-existing history/show/render/
check/status test suite, migrated onto the new package boundary and passing
unchanged (`go test -race ./...`); `internal/policies`'
`TestPolicy_LayeringDirection`, extended to assign `internal/entityview`
tier 4 (the check/render/htmlrender band — a domain package consumed by the
CLI layer, itself consuming only entity/gitops/scope), which fails CI on
any future upward or untiered import.

### AC-2 — render, check, status import the neutral package, not sibling CLI packages

`internal/cli/render` (`singlepass.go`, `resolver.go`), `internal/cli/check`
(`tests_metrics.go`), and `internal/cli/status` (`status.go`) no longer
import `internal/cli/history` or `internal/cli/show` for the F6 surface —
every call site (`HistoryEvent`, `ReadHistory`, `EventFromCommit`,
`HasOwnScope`, `HasAuthorizedBy`, `ScopeView`, `AssembleScopeViews`,
`ReadEntityBody`, `ShortHash`, `StripTrailers`) now resolves against
`internal/entityview`. `render`'s `singlepass.go` keeps its existing
`cliutil` dependency (for `CommitTrailers`/`ReplayScopes`/`OpenersFrom`,
unrelated to F6); `history` stays imported only where a genuinely
CLI-specific symbol is still needed (`history.RenderTo` in `show.go` and
`resolver.go`'s call sites, `history.ScopeMapFor` nowhere in these three).
The closing edge F6 warned about (`render`/`check`/`status` → `show`/
`history`, the only thing keeping the dependency graph acyclic by omission)
is gone: those three packages' sole path to the read-side projection logic
is now the neutral leaf.

Evidence: same as AC-1 — `go build ./...` and `go vet ./...` clean repo-wide;
the full test suite (including `internal/cli/render`, `internal/cli/check`,
`internal/cli/status`, and `internal/cli/integration`) green under
`-race -parallel 8`; `TestPolicy_LayeringDirection` passing confirms no
package in the `internal/cli/*` tier reaches into `entityview` upward of its
assigned tier, and no residual edge into `internal/cli/history` /
`internal/cli/show` survives in `render`/`check`/`status`'s import graphs
(spot-checked via `go list -deps` before landing).

## Constraints

- Mechanical only: import-path changes on the verified surface, no algorithm
  changes, no API redesign.
- Runs last in the epic, after the sibling milestones are done and green.

## Design notes

- Package name decided here (epic spec lean: `internal/entityview`).

## Out of scope

- Extracting anything Cobra-bound; the ~70% that stays put stays put.
- New read-verb features.

## Dependencies

- The three sibling E-0069 milestones (bug fixes, housekeeping, FinishVerb) —
  declared via `depends_on`.

## References

- `docs/initiatives/verb-layer-cleanup.md` §F6 (scope table, line inventory).

---

## Work log

## Decisions made during implementation

- D-0045 — `entityview` carries its own private `hasCommits` guard rather
  than importing `cliutil.HasCommits`, to keep the package genuinely free
  of `internal/cli/*`.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
