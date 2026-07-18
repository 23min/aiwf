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
defects that never individually justified their own investigation: **three**
verbs silently skip the validate-then-write gate the rest of the layer
treats as universal, one verb bypasses the shared id-allocation path
entirely and hand-rolls its own (reopening exactly the cross-branch
collision exposure that path was built to close), several verb pairs
hand-duplicate structurally identical logic instead of sharing a helper the
codebase otherwise knows how to share, a "fail/envelope" triad is
independently reimplemented in three separate CLI packages, the contract
subsystem runs three different ad hoc validation-gate styles across its
three mutating verbs, doctor hardcodes marker strings that `initrepo`
already exports specifically so doctor doesn't have to, and the read-only
verbs (`show`, `status`, `render`, `check`) form an unenforced verb-to-verb
dependency chain rather than depending on a neutral shared library. None of
these break the DAG property, but each is a small crack in it, and they
compound: a change to `check`'s rule set, `FinishVerb`'s envelope, or a hook
marker string has to be manually mirrored into every verb that quietly
diverged, with nothing to catch a miss.

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

### F1 — `SetArea`/`SetPriority`/`RenameArea` skip the universal validate-then-write gate

`internal/verb/verb.go`'s package doc states unconditionally that every verb
runs the projection check before writing. Confirmed by grepping every
`internal/verb/*.go` file for `projectionFindings(`: it's called from
`ac.go`, `add.go`, `editbody.go`, `import.go`, `milestone_depends_on.go`,
`move.go`, `promote.go`, `reallocate.go`, `rename.go`, and `retitle.go` — but
**not** from `setarea.go`, `setpriority.go`, or `renamearea.go`. All three
validate their new field value inline and go straight to `plan(&Plan{...})`.
`check` has dedicated `area-mistag`/`area-unknown`/`area-overlap` rules that
operate on exactly the fields these three verbs mutate (area membership and,
for rename-area, every entity's `area:` tag tree-wide), so this is a
violation of a documented invariant across three call sites, not a style
inconsistency. This is one of the two findings in this document that should
be treated as a bug rather than a cleanup preference (the other is F10).

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

### F3 — `acknowledge illegal` hand-rolls git plumbing instead of calling `gitops`

`internal/verb/acknowledgeillegal.go:191-213` (`shaAckable`) shells out
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

`internal/cli/archive/archive.go:220-234,279-288` calls `verb.Apply` directly
and hand-builds its own success/error envelope (`emitArchiveEnvelope`,
`failArchive`, `withCommitSHA`) instead of `cliutil.DecorateAndFinish`/
`FinishVerb`. `withCommitSHA` is a verbatim duplicate of
`internal/cli/cliutil/apply.go:100-109` — the doc comment at
`archive.go:239-242` already admits archive predates that shared helper's
adoption. This turns out not to be unique to archive: the *same* three-function
triad (`failX`/`emitXEnvelope`/`withCommitSHA`) is independently reimplemented
again in `internal/cli/rewidth/rewidth.go:214,232,250` and
`internal/cli/importcmd/importcmd.go:257,275,293` — each ~15 lines, each
admitted by its own comments as a mirror of the others (e.g.
`rewidth.go:244`: "Mirrors `cliutil.withCommitSHA` / archive's..."). All
three exist because `cliutil.FinishVerb` doesn't support dry-run or
multi-`Plan` output, which archive, rewidth, and import each need. A future
change to the shared outcome contract (a new exit code, a new envelope
field) now has to be manually mirrored into three places instead of one, or
it silently drifts in whichever copies get missed.

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

### F7 — minor duplications and near-dead API surface

- `for-each-ref refs/heads/` is independently re-issued in
  `internal/check/reflog_walk.go:138` and
  `internal/cli/check/isolation_escape_oracle.go:324` instead of both
  consuming `gitops.LocalBranchRefs` (`internal/gitops/refs.go:57`) — the
  isolation-escape-oracle variant has a legitimate perf reason to diverge
  (batches in `%(objectname)`), but `reflog_walk.go`'s is a plain duplicate.
- Two independently-maintained "closed set of archivable kinds" lists:
  `internal/verb/archive.go:392,403` (`isKnownKind`/`allKindNamesArchive`)
  and `internal/cli/archive/archive.go:120,293`
  (`archiveKindCompletions`/`validArchiveKind`) must be hand-kept in sync.
- `gitops.Commit`, `gitops.CommitAllowEmpty`, `gitops.Mv`, `gitops.Add`
  (`internal/gitops/gitops.go:69,74,86,97`) and `gitops.Init`
  (`internal/gitops/gitops.go:63`) have no production callers — only tests
  and the write-isolation policy's own ban-list reference the first four.
  Intentional (they're the named "forbidden APIs" the policy checks
  against), but as production API surface they read as dead ends; a comment
  at each definition marking them test/porcelain-only would remove the
  ambiguity for a future reader.

### F8 — `import` bypasses the shared id-allocation path entirely

`internal/verb/import.go:244-306` hand-rolls `idPrefix`, `formatID`,
`parseIDInt`, and `computeHighestPerKind` instead of calling
`entity.AllocateID` (`internal/entity/allocate.go:59`, the path `add.go:137`
uses correctly) and duplicates `entity.IDPrefix`
(`internal/entity/allocate.go:36`), which is explicitly documented as the
single source of truth for id-prefix formatting. Because import's
auto-id path never consults `tree.Tree.AllocationIDs()` (zero references in
`import.go`), it reintroduces exactly the cross-branch id-collision exposure
`entity.AllocateID` was built to close — an imported entity can silently
collide with an id allocated on another unpushed branch, the same failure
mode the id-lifecycle work already fixed for `add`. This is the second
finding in this document that should be treated as a bug: it's not just
duplicated code, it's a correctness regression relative to `add`'s
guarantees for anyone who imports rather than adds.

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
these skip validation outright the way F1's three verbs do — each runs
*something* — but there's no shared abstraction across the three, so each
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
might otherwise assume it does. Lower severity than F1/F8 — an observation
for the release-safety story, not a code defect to fix.

## Common sink nodes (for reference)

- **Tree load:** `tree.Load` / `cliutil.LoadTreeWithTrunk`
  (`internal/cli/cliutil/treeload.go:27`).
- **Frontmatter read/write:** `entity.Serialize` paired with `entity.Split`.
- **Id allocation:** `entity.AllocateID` (`internal/entity/allocate.go:59`),
  fed by `tree.Tree.AllocationIDs()` (`internal/tree/tree.go:205`) — the
  intended single allocation path for `Add`; `Import` bypasses it (F8).
- **Validation gate:** `projectionFindings` (`internal/verb/common.go:123`)
  → `check.Run`, diffed against the pre-mutation tree.
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
sequencing dependency between them beyond F1 and F8 being the
highest-priority fixes (they're correctness/invariant violations, not
preferences). None require a design decision; each is ordinary kernel work
behind an existing chokepoint (`projectionFindings`, `entity.AllocateID`,
`gitops`, `cliutil.FinishVerb`, `entity.ValidateTransition`).

**Bugs — fix first, independent of everything else:**

1. **F1** — route `SetArea`, `SetPriority`, and `RenameArea` through
   `projectionFindings` like every other structural verb.
2. **F8** — replace `import.go`'s hand-rolled `idPrefix`/`formatID`/
   `parseIDInt`/`computeHighestPerKind` with `entity.AllocateID`, restoring
   cross-branch collision protection for imported entities.

**Local cleanups — mechanical, low-risk, no design decision needed:**

3. **F2** — extract one shared path-rewrite helper
   (`computeEntityPathRewrite(e, replace func(name string) (string, error))`)
   and have both `rename.go` and `reallocate.go` call it.
4. **F3** — replace `acknowledgeillegal.go`'s hand-rolled `exec.Command` calls
   with `gitops.IsAncestor`/`gitops.CommitExists`.
5. **F4** — migrate `archive`, `rewidth`, and `import`'s CLI dispatchers onto
   a `cliutil.FinishVerb` extended to support dry-run/multi-`Plan` output,
   deleting all three duplicated `withCommitSHA`/envelope triads.
6. **F5** — collapse the cascade "no terminal move while a child is
   non-terminal" guard into one shared precondition function parameterized
   by target status, called from both `Cancel` and `Promote`; consider
   moving `Cancel` into its own `internal/verb/cancel.go`.
7. **F7** — fold `reflog_walk.go`'s ref listing onto `gitops.LocalBranchRefs`;
   unify the two archivable-kind lists behind one source; annotate the
   unreferenced `gitops` functions as test/porcelain-only.
8. **F9** — wire `doctor.go`'s hook-marker and guidance-marker detection onto
   `initrepo`'s already-exported marker functions instead of hardcoded
   string literals; collapse `completeHookNames`'s duplicate between
   `initcmd.go` and `update/hooks.go`.

**Judgment calls — need a decision, not just a patch:**

9. **F10** — decide whether the contract subsystem's three per-verb
   validation-gate styles should converge on one shared "scoped projection
   check" concept, or whether their divergence is justified by each verb's
   narrower blast radius (bind touches one binding; recipe install/remove
   touch config wiring, not entity content).
10. **F11** — decide whether `rewidth` should match `reallocate`'s broader
    sweep into archived entities, or whether the current active-tree-only
    scope is the right call (archived entities are terminal, so a stale
    reference there may not matter in practice).
11. **F12** — no code change implied; worth naming in the release-process
    docs that `aiwf upgrade` provides no automated rollback, so a bad tag
    requires a manual `go install` to recover.

**Largest item — its own milestone, not a patch:**

12. **F6** — extract `history`'s and `show`'s reusable, non-Cobra logic into
    neutral packages (e.g. `internal/history`, `internal/entityview`) that
    `render`/`check`/`status` depend on instead of on sibling `internal/cli`
    packages.

Two of these are filed as gaps — [G-0422](../../work/gaps/G-0422-no-presence-check-that-structural-verbs-call-projectionfindings.md)
and [G-0423](../../work/gaps/G-0423-no-clone-detection-linter-to-catch-duplicated-verb-layer-logic.md),
covering the two prevention mechanisms from the section below. The rest of
this document remains the map from which to file the balance: bundle
F2/F3/F5/F7/F9 as one small "verb-layer housekeeping" epic (F4's direct fix
is covered by G-0423's cleanup list); treat F10/F11/F12/F6 as separate
decisions/milestones each, since each carries its own judgment call or
blast radius.

## Why the existing guardrails missed these findings

The natural follow-up question: this repo has mutation testing
(`mutate-hunt`), a diff-scoped branch-coverage gate, and a substantial
`internal/policies/` AST-check suite — shouldn't one of those have caught
F1 or F8 before this audit found them? Mostly no, and the reason splits the
findings above into three genuinely different categories with three
different remedies.

**Sins of omission (F1, F8) are invisible to every test-execution-based
method, not just this repo's.** Mutation testing perturbs an existing
conditional, negates an existing comparison, or drops an existing
statement, then checks whether a test goes red — it has no operation for
"insert a call that was never written." The branch-coverage gate has the
same shape: it demands every *existing* line be exercised, which says
nothing about a line that doesn't exist. `SetArea`/`SetPriority`/
`RenameArea`'s inline validation is internally consistent and can be 100%
covered and 100% mutation-killed while still never calling
`projectionFindings` — the bug is an absence, not a mistake in present
logic, and no amount of exercising present logic can find an absence.

The AST policy that feels closest, `PolicyVerbsValidateThenWrite`, doesn't
help either: it's a **ban-list** (walks every exported verb function and
asserts a set of forbidden write calls is *absent*), which is the mirror
image of what F1 needs — a **presence** check asserting a required call
*is* present. Both shapes are equally buildable with the same AST-walking
technique; the repo has simply only built the ban-list version for this
particular gate. It has already built the presence-list version
elsewhere — `test_setup_presence.go` (every test package needs a
`TestMain`), `skill_coverage.go` (every verb needs a skill or an allowlist
entry), `firing_fixture_presence.go` (every policy needs a firing
fixture) — so this is an uncovered instance of an existing pattern, not a
new category of tooling. G-0422 tracks building it for `projectionFindings`.

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

## Risks and boundaries

**Risk: F6 scope creep.** Extracting `history`/`show`'s shared logic into
neutral packages touches import graphs across `render`, `check`, and
`status` simultaneously. Worth doing, but it's the one item here with real
blast radius — sequence it after the small, local fixes land and their
tests are green, not bundled with them.

**Risk: mechanical backstops don't cover most of these.** Only F1 and F8 sit
behind (or restore) a testable invariant — `check`'s `area-*` rules already
exist for F1; `entity.AllocateID`'s collision-avoidance behavior already has
test coverage for F8. Every other finding (F2-F7, F9-F12) is a readability/
maintainability/judgment-call item with no chokepoint that would fail CI if
it regressed or recurred — they rely on code review catching reintroduction,
the same as any other refactor.

**Risk: F11's scope-decision has a correctness edge either way.** Widening
rewidth to match reallocate's archive-inclusive sweep is a bigger behavior
change than it looks (it changes what `aiwf rewidth --apply` touches on a
tree with archived entities) — don't fold it into the F2-F9 housekeeping
pass; decide and scope it on its own.

## Desired future property

Every structural verb — not just most of them — runs through the same
validate-then-write gate, and every id-allocating path shares one
collision-avoidance implementation. A change to a shared contract (the
projection check, the commit-outcome envelope, a git-plumbing helper, a hook
marker string) only has to be made once to reach every verb that depends on
it, because no verb hand-rolls its own copy. The contract subsystem's
mutating verbs share one validation-gate concept instead of three
independent styles. The read-only verbs depend on neutral shared libraries
the same way the mutating verbs depend on `entity`/`gitops`/`check`, rather
than on each other's CLI packages.

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
guardrails missed these findings" section and gaps G-0422/G-0423.
