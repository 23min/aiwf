---
title: Verb-layer cleanup — closing the redundancies and invariant gaps in the call-graph
status: captured
date: 2026-07-18
---

# Verb-layer cleanup — closing the redundancies and invariant gaps in the call-graph

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf
entity kind ([G-0311](../../work/gaps/G-0311-no-cross-cutting-initiative-tier-above-epic-for-multi-component-features.md)),
so this file lives under `docs/initiatives/` as an umbrella capture, following
the precedent of [`id-lifecycle.md`](id-lifecycle.md) and
[`agent-agnostic-execution-topology.md`](agent-agnostic-execution-topology.md).

This is not an ADR: none of the findings below require a new architectural
decision — the shared spine they should converge on already exists and is
already used correctly by most verbs. This is not a plan: it deliberately
avoids committing to epics or milestones. Its job is to hold the map and the
finding list still so a right-sized set of gaps (or one small epic) can be
scoped from a coherent picture, instead of each item being independently
rediscovered.

## Initiative statement

The mutating-verb layer (`internal/cli/<verb>` → `internal/verb` →
`internal/gitops`) is a deliberately enforced DAG: every mutating verb loads
the tree, validates a projection, and writes through exactly one sink
(`gitops.CommitVerbChange`), and a policy test
(`internal/policies/verbs_validate_then_write.go`) mechanically bans any verb
from reaching git or the filesystem another way. That guarantee holds today.

A call-graph trace across **every** verb package in the kernel (2026-07-18,
covering the full set: all entity-mutating verbs, the contract subsystem,
the read-only verbs, the setup/maintenance cluster, and the smaller
miscellaneous verbs) found the spine intact but surfaced a set of local
defects that never individually justified their own investigation: one verb
bypasses the shared id-allocation path entirely and hand-rolls its own
(reopening part of the cross-branch collision exposure that path was built
to close), a second silently swallows a class of git-read errors that its
own sibling verbs treat as fail-loud, several verb pairs hand-duplicate
structurally identical logic instead of sharing a helper the codebase
otherwise knows how to share, a "fail/envelope" triad is independently
reimplemented in three separate CLI packages, the contract subsystem runs
three different ad hoc validation-gate styles across its three mutating
verbs, doctor hardcodes marker strings that `initrepo` already exports
specifically so doctor doesn't have to, and the read-only verbs (`show`,
`status`, `render`, `check`) form an unenforced verb-to-verb dependency
chain rather than depending on a neutral shared library. None of these break
the DAG property, but each is a small crack in it, and they compound: a
change to `check`'s rule set, `FinishVerb`'s envelope, or a hook marker
string has to be manually mirrored into every verb that quietly diverged,
with nothing to catch a miss.

A follow-up adversarial-verification pass (one independent skeptic per
finding, instructed to refute rather than confirm) re-examined every finding
against fresh source reads. Most held; one — the original F1 — did not, and
its refutation is itself informative: the original claim assumed `check`'s
area rules were reachable from `projectionFindings`, but they require
git-history data (`touchedByEntity`) that no in-memory verb-time projection
has, so they can never fire there regardless of which verbs call it. See
"Verification pass" below for the full per-finding verdict and what changed
as a result.

## The call graph

### Mutating verbs — the enforced spine

Every mutating verb (`add`, `promote`, `edit-body`, `cancel`, `rename`,
`retitle`, `move`, `reallocate`, `authorize`, `acknowledge illegal|mistag`,
`milestone depends-on`, and — with the gap noted below — `set-area`/
`set-priority`) follows one template:

```text
cli/<verb>.Run
   -> cliutil.ResolveRoot / ResolveActor / AcquireRepoLock
   -> tree.Load / cliutil.LoadTreeWithTrunk
   -> verb.<Fn>(ctx, tree, ...)
        -> entity.ValidateTransition        (FSM gate -- Promote only)
        -> entity.AllocateID                (Add only)
        -> projectionFindings -> check.Run  (validate-then-write gate)
        -> verb.Apply
             -> gitops.CommitVerbChange -> gitops.CommitTree -> gitops.ReconcilePaths
   -> cliutil.DecorateAndFinish / FinishVerb
```

`verb.Apply` (`internal/verb/apply.go:62`) and `gitops.CommitVerbChange`
(`internal/gitops/verbcommit.go:38`) are the one and only writer, enforced by
`internal/policies/verbs_validate_then_write.go`'s AST scan for direct calls
to `gitops.Mv/Add/Commit/CommitAllowEmpty/CommitTree/CommitVerbChange/
ReconcilePaths` or raw filesystem writes from any exported `internal/verb`
function.

`archive` and `cancel` are structural exceptions inside this otherwise
uniform shape (each is a deliberate design choice, not a defect on its own —
see Findings):

```text
archive:  cli/archive.Run -> verb.Archive -> verb.Apply directly
          (hand-rolls its own JSON envelope instead of FinishVerb)

cancel:   cli/cancel.Run -> verb.Cancel   (defined in internal/verb/promote.go,
          not a cancel.go -- uses entity.CancelTarget/IsTerminal instead of
          entity.ValidateTransition, with its own cascade guards)
```

### Read-only verbs — an unenforced sub-DAG

`history`, `list`, `show`, `status`, `render`, and `check` don't route
through a neutral shared library; they import each other's `internal/cli`
packages directly for exported helpers:

```text
history        list           (leaves: git log/trailers; tree filter)
   |              |
   v              v
  show   ----->  status
   |              |
   +------+-------+
          v
        render

check -> contract.RunValidation
check -> history.ReadHistory      (tests_metrics.go)
update -> doctor.WriteHealth
```

No cycle exists (`history`/`list` import neither `show` nor `status` nor
`render`), so the acyclic property survives, but `render` depends on three
sibling verb packages for library-shaped logic (`show.ReadEntityBody`/
`LoadEntityScopeViews`, `status.BuildStatus`, `history.HistoryEvent`), and
`check` depends on two (`contract`, `history`) the same way. This works only
because nobody has yet added the edge that would close a cycle.

### Contract subsystem — same enforced spine, different write target

`bind`, `unbind`, and `recipe install|remove` are ordinary mutating verbs in
shape — they route through `verb.Apply` → `gitops.CommitVerbChange` exactly
like the entity-tree verbs — but their `Plan` writes `aiwf.yaml`'s
`contracts:` block, not an entity markdown file:

```text
cli/contract bind|unbind    -> verb.ContractBind / ContractUnbind
cli/contract recipe install|remove -> verb.RecipeInstall / RecipeRemove
        (all four) -> verb.Apply -> gitops.CommitVerbChange
                       (writes aiwf.yaml, not work/<kind>/*.md)

cli/contract verify (read-only) -> contract.RunValidation
        -> contractcheck.Run       (structural: does the binding point at
                                     something real?)
        -> contractverify.Run      (executes the user's external validator
                                     binary against fixture files)

cli/check -> contract.RunValidation   (the same entry point, reused)
```

`contract verify` is a genuinely separate, intentional validation pipeline —
it validates external data shapes (CUE/JSON-Schema documents) against
fixtures, not aiwf entity state, so composing it into `check.Run` by
slice-append at the `aiwf check` layer rather than folding it in is the
right shape, not a smell.

### Setup/maintenance cluster — outside the entity-tree DAG by design

`init`, `update`, `upgrade`, and `doctor` sit entirely outside
`PolicyVerbsValidateThenWrite`'s scope: they materialize gitignored,
derived framework artifacts (skills, hooks, statusline, health cache), never
touch the entity tree, and never create a git commit:

```text
cli/init    -> initrepo.Init -> ... -> initrepo.RefreshArtifacts
cli/update  -> initrepo.RefreshArtifacts (same pipeline init uses)
                -> doctor.WriteHealth (best-effort, failure-only-logs)
cli/upgrade -> version.Latest (informational skew check)
            -> exec("go install <pkg>@<target>")   (fetch+verify+place,
               entirely delegated -- aiwf implements no rollback of its own)
cli/doctor  -> read-only by default (AST-pinned); --write-health and
               --self-check are the only two opt-in side effects

(all writes route through pathutil.AtomicWriteFile -- no raw os.WriteFile
 in production code in this cluster)
```

## Findings

### F1 — `verb.go`'s package doc overclaims "every verb"; the actual scope is narrower and undocumented

**Revised after adversarial verification** — the original framing of this
finding ("`SetArea`/`SetPriority`/`RenameArea` skip the universal
validate-then-write gate," treated as a bug) does not survive scrutiny and
has been replaced below. Two things the original claim got wrong:

1. **The omission isn't unique to these three.** `archive.go`, `rewidth.go`,
   `authorize.go`, `acknowledgeillegal.go`, `acknowledgemistag.go`,
   `linkrewrite.go`, and `contractrecipe.go` also never call
   `projectionFindings` — so "unlike every other structural mutator" was
   false on its face.
2. **The specific rules cited couldn't have fired anyway.** `check.Run`
   (`internal/check/check.go:109-158`, what `projectionFindings` wraps)
   does not include `area-mistag`/`area-unknown`/`area-overlap` at all —
   those are composed only at the CLI layer
   (`internal/cli/check/check.go:245,266,295`), fed a `touchedByEntity` map
   derived from scanning git commit history, data no verb has inside an
   in-memory pre-write projection. Calling `projectionFindings` from
   `SetArea`/`SetPriority`/`RenameArea` would not have made these rules fire;
   the check that actually gates them is the pre-push hook's full
   `aiwf check` run, by design — `rewidth.go` states the identical rationale
   explicitly for its own case, and `area-mistag` is documented as
   warning-only, never escalating.

`internal/verb/verb.go`'s package doc does say, unconditionally, that every
verb runs the projection check before writing — but a cited design doc
(`docs/pocv3/design/design-decisions.md:251`) already scopes the guarantee
to a named "current set" that excludes exactly the verbs above. The doc
comment's absolute wording is the real, narrower defect: it overclaims a
scope wider than the codebase's actual, consistent, deliberate design, and
that overclaim is precisely what led this audit to misdiagnose a correct
omission as an invariant violation on its first pass. The corrected finding
is a documentation-accuracy gap, not a correctness bug — see G-0422's
revised scope in "Scoped cleanup targets."

### F2 — duplicated path-substitution logic: rename vs. reallocate

`internal/verb/rename.go:139-182` (`renamePaths`/`substituteSlug`) and
`internal/verb/reallocate.go:435-471` (`reallocatePaths`/`substituteID`) are
structurally identical: same kind-switch (epic/contract get a directory
rename, everything else a file rename), same "split on the second hyphen"
slug parsing. `reallocate.go:460` already comments that it is "same shape as
substituteSlug." The same file correctly shares `plannedDestinations`/
`renameEntityMoves` across `rename.go`, `reallocate.go`, and `retitle.go` —
so the project already knows how to fold this kind of helper together; these
two functions are the ones that were never collapsed.

**Verified nuance:** the two functions diverge in the "no second hyphen"
branch, and it's a real semantic difference, not just cosmetic —
`substituteSlug` **appends** the new slug, preserving the old id prefix
(`renamePaths`'s job is "keep the id, change the slug"), while `substituteID`
**discards** the name entirely and returns just the new id (`reallocatePaths`'s
job is "keep the slug, change the id"). A shared helper needs that branch
parameterized, not merged naively — the fix in "Scoped cleanup targets" is
adjusted to reflect this.

### F3 — `acknowledge illegal` hand-rolls git plumbing instead of calling `gitops`

`internal/verb/acknowledgeillegal.go:188-213` (`shaAckable`) shells out
directly (`exec.Command("git", "merge-base", "--is-ancestor", ...)` and
`git rev-parse --verify <sha>^{commit}`) instead of calling the already
public `gitops.IsAncestor` (`internal/gitops/refs.go:164`) and
`gitops.CommitExists` (`internal/gitops/refs.go:233`) — exactly the two
functions `Promote`'s own `--by-commit` path
(`internal/verb/promote.go:505-525`) uses correctly for the same check. This
is a read, not a write, so it doesn't threaten the write-isolation guarantee,
but it's the one mutating verb that talks to git directly instead of through
the seam the rest of the kernel treats as sole owner of git access.

### F4 — the fail/envelope/`withCommitSHA` triad is independently reimplemented three times

`internal/cli/archive/archive.go` calls `verb.Apply` directly and
hand-builds its own success/error envelope (`emitArchiveEnvelope`,
`failArchive`, `withCommitSHA`, the helpers spanning roughly lines 243-291)
instead of `cliutil.DecorateAndFinish`/`FinishVerb`. `withCommitSHA` is a
verbatim duplicate of `internal/cli/cliutil/apply.go:100-109` — archive's
own doc comment already admits it predates that shared helper's adoption.
This turns out not to be unique to archive: the *same* three-function triad
(`failX`/`emitXEnvelope`/`withCommitSHA`) is independently reimplemented
again in `internal/cli/rewidth/rewidth.go:214,232,250` and
`internal/cli/importcmd/importcmd.go:257,275,293` — each ~15 lines, each
admitted by its own comments as a mirror of the others (e.g.
`rewidth.go:244-245`: "Mirrors `cliutil.withCommitSHA` / archive's identical
helper"). **Verified:** this is a real gap in `FinishVerb`'s contract, not
mere copy-paste laziness — `cliutil.FinishVerb` (`apply.go:32-76`)
unconditionally calls `verb.Apply` with no dry-run branch and assumes a
single `*verb.Plan`; import genuinely needs multi-`Plan` handling (its loop
applies `res.Plans` and tracks `lastSHA`), and archive/rewidth both need a
pre-apply dry-run branch. A future change to the shared outcome contract (a
new exit code, a new envelope field) now has to be manually mirrored into
three places instead of one, or it silently drifts in whichever copies get
missed.

### F5 — `Cancel`'s parallel legality codepath and duplicated cascade guard

`Cancel` is defined inside `internal/verb/promote.go` (no `cancel.go` file
exists) and gates terminal transitions via `entity.CancelTarget`/
`entity.IsTerminal` plus its own cascade-guard error types
(`EpicCancelNonTerminalChildrenError`, `MilestoneCancelNonTerminalACsError`)
rather than `entity.ValidateTransition`, which `Promote` uses. Separately,
`Promote`'s own epic/milestone terminal-promote guards
(`internal/verb/promote.go:147-189`) comment that they are "mirroring
Cancel's own guard" — the same precondition (no terminal move while a child
is non-terminal) is implemented twice, once per verb, with comments on both
sides acknowledging the duplication rather than a single guard parameterized
by target status.

### F6 — read-verb helpers live in `internal/cli/<verb>` instead of a neutral package

`history.ReadHistory`/`HistoryEvent`/`EventFromCommit` and
`show.ReadEntityBody`/`LoadEntityScopeViews`/`AssembleScopeViews` are
library-shaped logic that happens to live inside another verb's CLI package.
`render` pulls from three sibling verb packages this way; `check` from two.
Functionally safe today (no cycle), but it means the CLI layer is doing
double duty as command host and shared library, there is no single place to
look for "the shared read-side helpers" when adding a new read verb, and
`show`'s view-building logic can't be unit-tested or reused without pulling
in its Cobra-wiring-adjacent package.

**Deep dive (this is the one area the first pass only traced at the
"which functions get called" level; a follow-up read every line of the
actual call chain).** Three corrections to the original framing:

- **`render`'s live dependency on `show` is narrower than it looked.**
  Grepping `internal/cli/render/*.go` production code for `show\.` turns up
  exactly `show.ScopeView` (type), `show.AssembleScopeViews`
  (`singlepass.go:164`), and `show.ReadEntityBody` (`resolver.go:643`).
  `show.LoadEntityScopeViews` is referenced only in comments and the test
  oracle — render rebuilds its gating logic locally instead of calling it
  (see the duplication note below).
- **`status` is a third consumer of `history`'s model half**, not just
  `render` and `check` — `internal/cli/status/status.go:21,47,713-768`
  reuses `history.HistoryEvent`/`ShortHash`/`StripTrailers` inside its own
  `git log` parser. That's a stronger signal for extraction than the
  two-consumer picture the first pass had.
- **Not everything in `show`/`history` is extraction-worthy, and forcing all
  of it out would be over-reach.** `scopes.go` (205 lines: `ScopeView`,
  `LoadEntityScopeViews`, `AssembleScopeViews`, `LookupCommitDateCached`,
  `LastEventSHA`) and `history`'s `EventFromCommit` (85 lines) are already
  fully Cobra-free and pure — an accident of the M-0116 per-verb-package
  migration, not a deliberate design choice (confirmed via `git log`: the
  file moved verbatim from `cmd/aiwf/show_scopes.go`). `ReadEntityBody`
  (~28 lines) and roughly half of `history.go` (the `HistoryEvent`
  struct/`ReadHistory`/`ReadHistoryChain`/`ShortHash`/trailer-parsing slice,
  ~285 lines) are separable with the same low effort. The bulk of both
  files — `NewCmd`/`Run`/JSON-envelope shaping/`--area` filtering/text
  rendering — is genuinely Cobra-specific and should stay put. Realistic
  library surface: ~638 lines / ~16 exported symbols out of ~2071 combined,
  about 30%. Estimated difficulty: low, well under a day — mechanical
  import-path swaps, no algorithm changes, confirmed by tracing every
  production and test call site.

**Two real, previously-undiscovered bugs surfaced by reading this code
end-to-end** are now their own findings, F13 and F14, below — the deep dive
paid for itself independent of the architecture question.

### F7 — minor duplications and near-dead API surface

- `for-each-ref refs/heads/` is independently re-issued in
  `internal/check/reflog_walk.go:138` and
  `internal/cli/check/isolation_escape_oracle.go:324` instead of both
  consuming `gitops.LocalBranchRefs` (`internal/gitops/refs.go:57`) — the
  isolation-escape-oracle variant has a legitimate perf reason to diverge
  (batches in `%(objectname)`), but `reflog_walk.go`'s is a plain duplicate.
- **Corrected after verification:** `internal/verb/archive.go:392,403`
  (`isKnownKind`/`allKindNamesArchive`) and
  `internal/cli/archive/archive.go:120,293`
  (`archiveKindCompletions`/`validArchiveKind`) are *not* the "must be
  hand-kept in sync" risk originally claimed — they're deliberately
  different sets. `isKnownKind`/`allKindNamesArchive` iterate all six kinds
  from `entity.AllKinds()`; `archiveKindCompletions` is a five-kind literal
  that excludes milestone on purpose, per ADR-0004. The one-element
  divergence is intentional design, not accidental drift — no action
  needed here.
- `gitops.Commit`, `gitops.CommitAllowEmpty`, `gitops.Mv`, `gitops.Add`
  (`internal/gitops/gitops.go:69,74,86,97`) have no production callers —
  only tests and the write-isolation policy's own ban-list reference them.
  **Corrected:** `gitops.Init` (`internal/gitops/gitops.go:63`) does *not*
  belong on this list — it has a genuine, if narrow, production caller via
  `internal/cli/doctor/selfcheck.go:115`, reached from the real, wired
  `aiwf doctor --self-check` flag. The other four remain intentional
  test/porcelain-only APIs (the named "forbidden APIs" the write-isolation
  policy checks against); a comment at each definition marking them as such
  would remove the ambiguity for a future reader.

### F8 — `import` bypasses the shared id-allocation path entirely

`internal/verb/import.go:244-306` hand-rolls `idPrefix`, `formatID`,
`parseIDInt`, and `computeHighestPerKind` instead of calling
`entity.AllocateID` (`internal/entity/allocate.go:59`, the path `add.go:137`
uses correctly) and duplicates `entity.IDPrefix`
(`internal/entity/allocate.go:36`), which is explicitly documented as the
single source of truth for id-prefix formatting. Both files' own comments
admit the mirroring (import.go:265-266: "parseIDInt is the package-local
mirror of entity.parseIDNumber... unexported so we recreate it here";
import.go:301-303: "Mirrors entity.AllocateID's formatting"). Because
import's auto-id path never consults `tree.Tree.AllocationIDs()` (zero
references in `import.go`), and `importcmd.go` has no `--fetch` flag the
way `add.go:208-212` does, `id: auto` (the documented first-class idiom for
greenfield entities, not a rare fallback) never sees ids from sibling local
branches or a teammate's pushed-but-unmerged remote branch.

**Verified nuance — the exposure is real but narrower than "silently
collide" suggests.** A collision against **trunk** ids is actually caught:
`Import` does call `projectionFindings` (`import.go:211`), which runs
`idsUnique` against the projected tree, and `idsUnique` does read
`t.TrunkIDs`. The genuinely unmitigated vector is collision against
`LocalRefIDs`/`RemoteRefIDs` (other local branches, other pushed branches),
which `idsUnique` deliberately excludes by design (a separate, accepted
scope boundary, E-0052) — that collision surfaces only later, at merge
time, via `aiwf reallocate`. This is exactly the pre-`AllocateID` exposure
the id-lifecycle work fixed for `add`, now reintroduced for `import`; one
migration doc (`docs/pocv3/migration/import-format.md:133`) even overclaims
"`auto` allocation never collides with existing ids by construction,"
reinforcing that this is a genuine, unaddressed gap rather than a
documented, accepted limitation. This remains the clearest bug-tier finding
in this document (F1's original bug framing did not survive verification;
see above).

### F9 — setup-verb cluster: exported marker helpers doctor doesn't call

`internal/initrepo/initrepo.go:1412-1422,1579` exports `HookMarker()`,
`PreCommitHookMarker()`, `CommitMsgHookMarker()`, and `PostCommitHookMarker()`
whose doc comments state, verbatim, that they exist "for `aiwf doctor` to
identify a marker-managed hook versus a user-written one." But
`internal/cli/doctor/doctor.go:488,605,690,741` independently hardcodes the
literal strings `"# aiwf:pre-push"`, `"# aiwf:pre-commit"`,
`"# aiwf:commit-msg"`, `"# aiwf:post-commit"` instead — `doctor` doesn't even
import `internal/initrepo`. Consequently `PostCommitHookMarker()` and
`CommitMsgHookMarker()` have zero callers anywhere outside `initrepo`'s own
tests: exported API with a documented consumer that was never wired up. If a
marker string ever changes in `initrepo.go`, `doctor.go`'s copies silently
drift out of sync with nothing but a partial integration test (which checks
only 2 of the 4 markers, and checks the hook file rather than doctor's
detection logic) standing in the way. The same pattern repeats at smaller
scale: `internal/initrepo/initrepo.go:850` (`guidanceMarkerLineIdx`) and
`internal/cli/doctor/guidance.go:46` (`guidanceImportLinePresent`)
independently implement the same CLAUDE.md-import-marker check, and
`completeHookNames` is duplicated verbatim between
`internal/cli/initcmd/initcmd.go:85-87` and
`internal/cli/update/hooks.go:17-19` (the update copy's own comment admits
it mirrors initcmd's). By contrast, the registry-hook drift check does this
correctly: both `doctor.go:400` and `cliutil.SyncHookMaterialization` share
one `skills.HookDrift` helper — proof the codebase already knows the right
shape, just didn't apply it to the git-hook and guidance markers.

### F10 — contract subsystem: three mutating verbs, three different validation-gate styles

`ContractBind` runs `contractCheckForBinding`
(`internal/verb/contractbind.go:140-149`) — a bespoke, narrower proxy for
`projectionFindings` that filters `contractcheck.Run`'s post-mutation
findings down to the one bound entity's id, rather than a true before/after
diff. `RecipeInstall` runs no config-correspondence gate at all
(`internal/verb/contractrecipe.go:29-73`), only idempotency/`--force`
checks. `RecipeRemove` runs a third, different shape: a manual referential-
integrity scan (`bindingsReferencing`, `contractrecipe.go:90-92`). None of
these skip validation outright — each runs *something* — but there's no
shared abstraction across the three, so each
was designed independently rather than converging on one gate-with-scope
concept the way entity-tree verbs converge on `projectionFindings`.

### F11 — rewidth's archive-subtree exclusion is inconsistent with reallocate's sweep

`rewidth`'s reference-rewrite walk explicitly and consistently excludes
`<kind>/archive/...` at every level (`internal/verb/rewidth.go:284,311,364,
598,723` all skip on `name == "archive"`), documented as deliberate
(`rewidth.go:28-29,47`: "Active-tree only... Links targeting
`<kind>/archive/...` are excluded by design"). `reallocate`'s equivalent
sweep (`rewriteReferences`/`findProseMentions`,
`internal/verb/reallocate.go:483,605`) walks `t.Entities`, which `tree.Load`
populates with archived entities too, so reallocate rewrites references
inside archived files while rewidth does not. Both are id-mutating verbs
that need to keep cross-references correct; today they give different
guarantees. Rewidth's choice is documented, not accidental, but it means an
archived entity's stale frontmatter/prose reference to a narrow-width id
that gets widened elsewhere in the same run is left uncorrected — worth an
explicit decision on whether that's acceptable (archived entities are
terminal, so a stale reference there may genuinely not matter) or whether
rewidth should match reallocate's broader sweep.

### F12 — `upgrade` has no aiwf-level rollback for a bad binary swap

`aiwf upgrade` delegates the entire fetch/verify/place sequence to a single
`exec.Command("go", "install", ...)` call (`internal/cli/upgrade/upgrade.go:296,302`)
— aiwf itself implements no backup-old-binary-first step, so if the newly
installed binary is broken, there is no automated rollback; the operator
would have to manually `go install <pkg>@<older-tag>`. This is a reasonable
minimalist design (not reinventing a binary installer) rather than an
oversight, but it's worth naming explicitly as a property `aiwf upgrade`
does **not** provide, since "cut a release" / "aiwf upgrade" conversations
might otherwise assume it does. Lower severity than F8/F13 — an observation
for the release-safety story, not a code defect to fix.

### F13 — `show` silently swallows history/scope-read errors that its own siblings treat as fail-loud

Surfaced by the F6 deep dive, not the original pass. `BuildShowView`
(`internal/cli/show/show.go:386-392`) and `BuildCompositeShowView`
(`show.go:557-563`) both do the equivalent of:

```go
events, err := history.ReadHistory(ctx, root, id)
if err == nil { view.History = limitEvents(events, historyLimit) }
if scopes, err := LoadEntityScopeViews(ctx, root, id); err == nil { view.Scopes = scopes }
```

On a `git log` failure (corrupt repo, environment fault), both fields
silently stay `nil`, `Run` still returns `cliutil.ExitOK`, and the JSON
envelope's `omitempty` tags make "couldn't read history" indistinguishable
from "this entity legitimately has no history." This is inconsistent with
how the rest of the kernel treats the identical failure class: `render`
(`render.go:318-329`) fails loud on the equivalent walk error with an
explicit comment reasoning that degrading silently would blank a whole
report section, which is "strictly worse than the old per-entity
best-effort... a corrupt/partial repo should stop the render, not emit a
misleadingly-empty site" — and the sibling `aiwf history` verb
(`internal/cli/history/history.go:119-130`) reaches the same conclusion for
the same error. `show` never got this fix; no test in
`internal/cli/show/*_test.go` exercises either error branch, confirming
it's an overlooked gap, not a documented, accepted risk (unlike the
`//coverage:ignore` annotations elsewhere in `scopes.go` that document *why*
a branch is hard to hit while still handling the error). This is a real
correctness bug, not an architectural preference.

### F14 — cross-timezone scope events can sort out of true chronological order

Also surfaced by the F6 deep dive. `AssembleScopeViews`
(`internal/cli/show/scopes.go:169-171`) sorts `ScopeView`s by lexical string
comparison of each event's `Opened` timestamp, sourced from git's `%aI`
format (author-local ISO-8601 with that author's UTC offset preserved,
`LookupCommitDateCached`, `scopes.go:183`, and the equivalent in `render`'s
own index). Because `%aI` preserves each commit's author-local offset
rather than normalizing to UTC, two commits from authors in different
timezones can sort lexically out of true chronological order — e.g. an
event timestamped `...T23:00:00-07:00` sorts before one timestamped
`...T05:00:00+00:00` even though the former happened later in real time.
Cosmetic in impact (it only affects row ordering in a scope table), but
it's a genuine correctness bug relative to the "chronological" framing the
feature promises, and both `show` and `render` inherit it identically since
they share the one sort call — fixing it once fixes both.

## Verification pass

Every finding from the original two audit passes was re-examined by an
independent skeptic agent instructed to refute it, not confirm it. Verdicts:

| Finding | Verdict | What changed |
|---|---|---|
| F1 | **Refuted** (as originally framed) | Rewritten from "3 verbs violate an invariant" to "the package doc overclaims scope"; see F1 and G-0422 |
| F2 | Confirmed, with revision | The "no second hyphen" branch is a real semantic fork, not identical — fix must be parameterized |
| F3 | Confirmed | Minor line-range correction only (188-213) |
| F4 | Confirmed, with revision | Archive.go line citation corrected; the "why" (FinishVerb's dry-run/multi-Plan gap) verified as real, not just historical accident |
| F5 | Confirmed | No changes |
| F6 | Confirmed, deepened | Extraction scope narrowed and quantified via a full read of the actual internals; surfaced F13 and F14 as a byproduct |
| F7 (ref-listing) | Confirmed | No changes |
| F7 (archivable-kind lists) | **Refuted** | The two lists intentionally differ by one kind (ADR-0004) — not a sync-drift risk |
| F7 (dead `gitops` fns) | Confirmed, with revision | `gitops.Init` has a real production caller via `doctor --self-check`; dropped from the dead-ends list |
| F8 | Confirmed, with revision | Trunk-collision is actually caught by `idsUnique`; the real gap is narrower (local/remote-ref collisions only) but still real |
| F9 | Confirmed | All four sub-claims verified verbatim, no changes |
| F10 | Confirmed | No changes |
| F11 | Confirmed | `tree.Load`'s inclusion of archived entities independently verified via the loader's own path-classification code |
| F12 | Confirmed | No changes |
| F13, F14 | New | Surfaced by the F6 deep dive, not the original pass |

The one substantive reversal (F1) is discussed in its own section above and
in "Why the existing guardrails missed these findings."

## Common sink nodes (for reference)

- **Tree load:** `tree.Load` / `cliutil.LoadTreeWithTrunk`
  (`internal/cli/cliutil/treeload.go:27`).
- **Frontmatter read/write:** `entity.Serialize` paired with `entity.Split`.
- **Id allocation:** `entity.AllocateID` (`internal/entity/allocate.go:59`),
  fed by `tree.Tree.AllocationIDs()` (`internal/tree/tree.go:205`) — the
  intended single allocation path for `Add`; `Import` bypasses it (F8).
- **Validation gate:** `projectionFindings` (`internal/verb/common.go:123`)
  → `check.Run`, diffed against the pre-mutation tree. **Scope, verified:**
  `check.Run` itself only covers rules computable from in-memory tree state
  — it does not include the CLI-composed, git-history-dependent rules
  (e.g. `area-mistag`/`area-unknown`/`area-overlap`, which need a
  `touchedByEntity` map built by scanning commit history). Those rules are
  only ever reachable via the pre-push hook's full `aiwf check`, by design —
  not a gap in `projectionFindings` (see F1).
- **FSM legality:** `entity.ValidateTransition`
  (`internal/entity/transition.go:79`) — sole production caller is `Promote`.
- **Authorization/reachability:** `verb.Allow` (`internal/verb/allow.go:138`),
  invoked once from `gateAndDecorate`
  (`internal/cli/cliutil/provenance.go:61,80`).
- **Apply/commit:** `verb.Apply` (`internal/verb/apply.go:62`) →
  `gitops.CommitVerbChange` (`internal/gitops/verbcommit.go:38`).
- **Outcome plumbing:** `cliutil.DecorateAndFinish` →
  `cliutil.FinishVerb` (`internal/cli/cliutil/apply.go:32`) — bypassed by
  archive, rewidth, and import (F4).
- **Contract-external validation entry point:** `contract.RunValidation`
  (`internal/cli/contract/contract.go:57`), shared correctly by both
  `contract verify` and `check.Run`.
- **Framework-artifact writes (outside the entity-tree DAG entirely):**
  `pathutil.AtomicWriteFile` (`internal/pathutil/atomicwrite.go:29`) — the
  sole write chokepoint for everything `init`/`update`/`upgrade`/`doctor`
  touch under `.claude/` and `.git/hooks/`; well-enforced by
  `internal/policies/atomic_write_chokepoint.go`, no gap found here.

## Scoped cleanup targets

Each finding above is independently fixable and independently testable — no
sequencing dependency between them beyond F8 and F13 being the
highest-priority fixes (they're correctness regressions, not preferences).
None require a design decision; each is ordinary kernel work behind an
existing chokepoint (`projectionFindings`, `entity.AllocateID`, `gitops`,
`cliutil.FinishVerb`, `entity.ValidateTransition`).

**Bugs — fix first, independent of everything else:**

1. **F8** — replace `import.go`'s hand-rolled `idPrefix`/`formatID`/
   `parseIDInt`/`computeHighestPerKind` with `entity.AllocateID`, closing the
   local-ref/remote-ref collision gap for imported entities (the trunk-side
   collision case is already caught by `idsUnique`).
2. **F13** — make `show`'s history/scope-read failures fail loud
   (`cliutil.ExitInternal`), matching `render`'s and `aiwf history`'s
   existing precedent for the identical error class; add the missing error-
   branch test.

**Small, low-risk fix — worth doing alongside the bugs above:**

3. **F14** — normalize scope-event timestamps before sorting in
   `AssembleScopeViews` (parse and compare as `time.Time`, or normalize to
   UTC before the lexical sort) so cross-timezone events sort in true
   chronological order.

**Local cleanups — mechanical, low-risk, no design decision needed:**

4. **F2** — extract one shared path-rewrite helper, parameterized on the
   "no second hyphen" behavior (append-slug for rename, discard-and-replace
   for reallocate — the two are not identical, see the verified nuance
   above), and have both `rename.go` and `reallocate.go` call it.
5. **F3** — replace `acknowledgeillegal.go`'s hand-rolled `exec.Command` calls
   with `gitops.IsAncestor`/`gitops.CommitExists`.
6. **F4** — migrate `archive`, `rewidth`, and `import`'s CLI dispatchers onto
   a `cliutil.FinishVerb` extended to support dry-run/multi-`Plan` output,
   deleting all three duplicated `withCommitSHA`/envelope triads.
7. **F5** — collapse the cascade "no terminal move while a child is
   non-terminal" guard into one shared precondition function parameterized
   by target status, called from both `Cancel` and `Promote`; consider
   moving `Cancel` into its own `internal/verb/cancel.go`.
8. **F7** — fold `reflog_walk.go`'s ref listing onto `gitops.LocalBranchRefs`;
   annotate the four genuinely-unreferenced `gitops` functions
   (`Commit`/`CommitAllowEmpty`/`Mv`/`Add` — not `Init`, which has a real
   caller) as test/porcelain-only. The archivable-kind-list item from the
   original pass is dropped — verified as intentional divergence, not drift.
9. **F9** — wire `doctor.go`'s hook-marker and guidance-marker detection onto
   `initrepo`'s already-exported marker functions instead of hardcoded
   string literals; collapse `completeHookNames`'s duplicate between
   `initcmd.go` and `update/hooks.go`.

**Documentation-accuracy fix — not a code change:**

10. **F1** — correct `internal/verb/verb.go`'s package doc to state the
    actual, narrower scope (which verbs run `projectionFindings` and why the
    others legitimately don't), matching the design doc's already-scoped
    "current set." Optionally, encode that set as an explicit, reviewed
    allowlist a policy can check against — see G-0422's revised scope below.

**Judgment calls — need a decision, not just a patch:**

11. **F10** — decide whether the contract subsystem's three per-verb
    validation-gate styles should converge on one shared "scoped projection
    check" concept, or whether their divergence is justified by each verb's
    narrower blast radius (bind touches one binding; recipe install/remove
    touch config wiring, not entity content).
12. **F11** — decide whether `rewidth` should match `reallocate`'s broader
    sweep into archived entities, or whether the current active-tree-only
    scope is the right call (archived entities are terminal, so a stale
    reference there may not matter in practice).
13. **F12** — no code change implied; worth naming in the release-process
    docs that `aiwf upgrade` provides no automated rollback, so a bad tag
    requires a manual `go install` to recover.

**Largest item — its own milestone, not a patch:**

14. **F6** — extract the already-pure pieces only (`scopes.go` and
    `history`'s `EventFromCommit` wholesale; `ReadEntityBody` and roughly
    half of `history.go` need splitting out of their Cobra-bound siblings)
    into a neutral package (e.g. `internal/entityview`) that
    `render`/`check`/`status` depend on instead of on sibling `internal/cli`
    packages. Verified scope: ~638 lines / ~16 exported symbols out of
    ~2071 combined (~30%); the rest is genuinely Cobra-specific and stays.
    Estimated difficulty: low, well under a day — mechanical import-path
    changes, no algorithm changes.

Two of these are filed as gaps — `G-0422`
(revised in scope after F1's refutation — now tracks documenting/enforcing
the actual, narrower `projectionFindings` scope rather than requiring it
universally) and `G-0423`
(one example softened after verification, the rest confirmed), covering the
two prevention mechanisms from the section below. The rest of this document
remains the map from which to file the balance: bundle F2/F3/F5/F7/F9 as one
small "verb-layer housekeeping" epic (F4's direct fix is covered by
G-0423's cleanup list); file F8/F13/F14 as bug gaps; treat F10/F11/F12/F6 as
separate decisions/milestones each, since each carries its own judgment call
or blast radius.

## Why the existing guardrails missed these findings

The natural follow-up question: this repo has mutation testing
(`mutate-hunt`), a diff-scoped branch-coverage gate, and a substantial
`internal/policies/` AST-check suite — shouldn't one of those have caught
F8 or F13 before this audit found them? Mostly no, and the reason splits the
findings above into four genuinely different categories with four
different remedies.

**Sins of omission (F8) are invisible to every test-execution-based method,
not just this repo's.** Mutation testing perturbs an existing conditional,
negates an existing comparison, or drops an existing statement, then checks
whether a test goes red — it has no operation for "insert a call that was
never written." The branch-coverage gate has the same shape: it demands
every *existing* line be exercised, which says nothing about a line that
doesn't exist. `import.go`'s hand-rolled id math is internally consistent
and can be well-tested while still never calling `entity.AllocateID` — the
bug is an absence, not a mistake in present logic, and no amount of
exercising present logic can find an absence. The AST policy that would
help here doesn't exist yet: `PolicyVerbsValidateThenWrite` is a **ban-list**
(walks every exported verb function and asserts a set of forbidden write
calls is *absent*); F8 needs the mirror-image **presence** check (assert a
required call *is* present) — a shape the repo has already built
elsewhere (`test_setup_presence.go`, `skill_coverage.go`,
`firing_fixture_presence.go`), just not for id allocation.

**Sins of stale coverage (F13) are a related but distinct mechanism worth
naming separately.** `show`'s silently-swallowed error branch (`if err ==
nil { ... }`, with no `else`) is *not* invisible to branch coverage in
principle — an uncovered branch is exactly the shape the coverage gate is
built to catch. It slipped through because the gate is **diff-scoped**: it
only forces coverage on lines changed since the base ref. This code
predates the gate (or was last touched before the rule existed), so it was
never required to prove its error branch was exercised, and nothing
retroactively re-checks it. A diff-scoped gate is the right tradeoff for an
actively-changing codebase — running full non-diff coverage on every commit
would be prohibitively noisy — but it means pre-existing, never-touched-since
code can carry an untested branch indefinitely, found only by someone
reading it end-to-end, which is exactly how F13 surfaced (the deep-dive
verification pass on F6, not a coverage report).

**Sins of duplication (F2, F4, F7, F9) are invisible to correctness tooling
by design.** Each duplicated copy can be independently correct and
independently well-tested — no unit test, property test, or mutation score
compares one function's body against another's. That requires a clone
detector, a different tool category entirely. `.golangci.yml`'s enabled
linter set has no such check (`gocritic`'s duplicate detection is
statement-local within one function, not cross-file), which is why the same
`failX`/`emitXEnvelope`/`withCommitSHA` triad could be reimplemented three
times (F4) without any lint ever firing. G-0423 tracks enabling a clone
detector as the mechanical backstop for this class.

**Sins of divergent design (F10, F11) aren't mechanically catchable at
all**, and shouldn't be forced to be — whether the contract subsystem's
three validation-gate styles should converge, or whether `rewidth` should
match `reallocate`'s archive-inclusive sweep, are judgment calls with a
real "maybe not" on each side. The only mechanism that surfaces this class
is a periodic cross-cutting trace like the one that produced this document,
not a permanent gate.

**A fifth category this section didn't originally have: sins of the audit
itself.** The original F1 claimed a mechanical-tooling gap (no presence
check for `projectionFindings`) that turned out not to exist, because its
factual premise (that `check.Run`'s area rules were reachable from
`projectionFindings`) was wrong. No tool caught that error either — it took
a dedicated adversarial-verification pass, reading `check.Run`'s actual rule
composition rather than trusting the first pass's characterization of it.
The lesson generalizes: an audit's own findings need the same skepticism
applied to the code it audits, and a single trace (however careful) is not
self-verifying. That's the concrete case for treating "verify the findings"
as its own pass, not an optional afterthought — see G-0422's revised scope,
which now documents the corrected understanding rather than the original
(wrong) diagnosis.

## Risks and boundaries

**Risk: F6 scope creep.** Extracting `history`/`show`'s shared logic into
neutral packages touches import graphs across `render`, `check`, and
`status` simultaneously. Worth doing, but it's the one item here with real
blast radius — sequence it after the small, local fixes land and their
tests are green, not bundled with them.

**Risk: mechanical backstops don't cover most of these.** Only F8 sits
behind a testable invariant — `entity.AllocateID`'s collision-avoidance
behavior already has test coverage, so fixing F8 to call it inherits that
coverage directly. F13 and F14 will each need a new, purpose-written test
for the error branch and the timezone-sort case respectively — neither
currently has one, which is precisely why they went unnoticed. Every other
finding (F1-F7, F9-F12) is a readability/maintainability/judgment-call/
documentation item with no chokepoint that would fail CI if it regressed or
recurred — they rely on code review catching reintroduction, the same as
any other refactor.

**Risk: F11's scope-decision has a correctness edge either way.** Widening
rewidth to match reallocate's archive-inclusive sweep is a bigger behavior
change than it looks (it changes what `aiwf rewidth --apply` touches on a
tree with archived entities) — don't fold it into the F2-F9 housekeeping
pass; decide and scope it on its own.

## Future option — a full multi-agent workflow sweep

This document's two verification rounds (an initial call-graph trace, then
one adversarial skeptic per finding plus a deep dive on F6) were each run as
a bounded set of `Explore` subagents — proportionate to the task, not the
largest tool available. A structured multi-agent `Workflow` run (this
environment's tool for deterministic multi-stage agent orchestration) could
go further in two specific ways this pass deliberately didn't:

1. **Multi-voter adversarial confirmation.** Every finding here was checked
   by exactly one skeptic. A workflow can run each finding past 3-5
   independent skeptics in parallel and require a majority "not refuted"
   verdict before a finding survives — this round's F1 reversal shows a
   single skeptic can still be wrong in either direction (it correctly
   refuted the original F1, but a single check is still a single point of
   failure); a quorum is more robust than one pass, at the cost of several
   times the agent calls per finding.
2. **A systematic sweep of the sink packages themselves.** This whole
   document audits how verbs *call into* `internal/entity`, `internal/tree`,
   `internal/gitops`, and `internal/check` — it does not audit those four
   packages' own internals for undiscovered issues the same way the F6 deep
   dive did for `show`/`render`. A workflow could run the same
   "trace shallow, then deep-dive the parts that look load-bearing" pattern
   used here for the verb layer, applied instead to the sink packages —
   `internal/entity`'s FSM tables, `internal/tree`'s loader, `internal/gitops`'s
   ref/commit primitives, `internal/check`'s rule composition (the exact
   layer whose actual behavior refuted F1) — with a completeness-critic
   final stage asking "what part of these four packages did no finder even
   look at."

Neither is needed to act on the findings already in this document — they're
independently fixable regardless. The reason to reach for the larger sweep
later is scale (auditing four foundational packages instead of ~35 verb
files) and confidence (quorum voting instead of single-skeptic verification),
not a gap in what's already here. Worth doing when there's a specific reason
to trust the sink packages less than this pass currently does — e.g. before
a release that leans hard on one of them, or if a future finding traces back
to a sink-package bug rather than a verb-layer one.

## Desired future property

The scope of which verbs run `projectionFindings` — and, more importantly,
*why* the others legitimately don't — is written down and enforceable, not
implicit in an overclaiming package doc that an audit can misread the way
this one initially did. Every id-allocating path shares one
collision-avoidance implementation, with no hand-rolled second copy. A
change to a shared contract (the commit-outcome envelope, a git-plumbing
helper, a hook marker string) only has to be made once to reach every verb
that depends on it, because no verb hand-rolls its own copy. `show` and its
siblings treat a git-read failure the same way everywhere — fail loud,
never silently blank a section — instead of one verb quietly diverging from
the others' precedent. The contract subsystem's mutating verbs share one
validation-gate concept instead of three independent styles. The read-only
verbs depend on neutral shared libraries the same way the mutating verbs
depend on `entity`/`gitops`/`check`, rather than on each other's CLI
packages.

## Provenance

Emerged from a 2026-07-18 conversation that asked for a data-flow analysis
of aiwf's most important verbs — call graphs, DAG-shape sanity check, dead
ends, redundancies. The first pass covered the entity-mutating verbs, the
core read-only verbs, and `worktree add`/`authorize`/`acknowledge` via one
`Explore` agent. A follow-up pass, prompted by the observation that the
first pass covered only a subset of the verb surface, added three more
`Explore` agents in parallel to cover the contract subsystem (`bind`/
`unbind`/`verify`/`recipes`), the setup/maintenance cluster (`init`/
`update`/`upgrade`/`doctor`), and the remaining miscellaneous verbs
(`import`/`rewidth`/`template`/`whoami`/`rename-area`) — the full verb
surface. Every finding cited above was independently spot-checked against
the actual source (grep/read confirmation of the specific file:line claims)
before being written up here. A third pass asked why none of the findings
were caught by existing tests or policies, producing the "Why the existing
guardrails missed these findings" section and gaps G-0422/G-0423. A fourth
pass, requested explicitly for higher confidence and more depth, ran one
independent adversarial skeptic per finding (instructed to refute, not
confirm) plus a dedicated deep dive into `show`/`render`'s internals (the
one area the earlier passes had only traced, not fully read) — producing
the "Verification pass" table, the F1 reversal, findings F13/F14, and the
"Future option" section scoping what a larger multi-agent workflow sweep
would additionally give.
