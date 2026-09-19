---
title: Structural sweep audit — dead paths, dropped data flow, and convergent duplication measured across the module
status: captured
date: 2026-09-17
---

# Structural sweep audit

A dated snapshot of what the `wf-structural-sweep` ritual found across the whole
module at `b10eb2676`: code nothing reaches, values produced and never consumed,
and the same job implemented more than once in textually different code.

Like [`entity-truth-audit.md`](entity-truth-audit.md) and
[`gap-truth-audit.md`](gap-truth-audit.md), this is an **inventory rather than a
proposal**, and it ages by construction: every entry is either fixed (and deleted)
or still true. The `date:` is what makes it honest. Entries are promoted to a gap
or an epic when someone picks them up; an entry that is fixed without a tracker is
deleted here in the same change.

The sweep's four lenses find different things, and the two mechanical ones found
nothing that G-0417, G-0472, G-0473 and G-0533 did not already record. Everything
below that is new came from the two lenses no tool runs — convergent duplication
read against the `wf-codebase-health` rubric, and producer→consumer data flow —
plus one reachability run the ritual does not currently prescribe.

## Scope and method

Whole module: every package under `internal/` and `cmd/`, production and test
code, at HEAD `b10eb2676` on the `epic/E-0084` branch, in the devcontainer
(Linux, go1.25.11, golangci-lint 2.12.2, `deadcode` from `golang.org/x/tools`).

Four lenses, in order:

1. **Reachability**, two runs. `deadcode -test` over `./...` with tests as roots,
   as the ritual prescribes. Then `deadcode` over `./cmd/aiwf ./cmd/stresstest`
   only — the binaries' main roots, no tests — which reports production code
   kept alive solely by tests. Both with `-tags stress,testpins`.
2. **Clone catalogue.** `dupl` at the configured threshold (100) with every path
   exclusion in `.golangci.yml` lifted and only `_test.go` excluded, so the
   grandfathered files are measured rather than skipped.
3. **Convergent duplication**, nine independent read-only reviewers, one per
   package cluster (check, verb, policies, core model, CLI layer, git
   infrastructure, stress harness, install and config, render and spec), each
   handed the list of already-tracked findings and told to grep `work/gaps/`
   and `work/decisions/` for ownership before reporting.
4. **Data flow**, the same reviewers, tracing exported fields, return values and
   emitted records to a consumer that reads them for a decision.

Every reviewer worked against the working tree, not the `aiwf` on PATH. Where a
claim could be measured with a command it was; those entries say **measured**.
Where a claim rests on reading two sites it says **derived**, and the command
that would settle it is named under *What no measurement here settles*. The repo
was not modified; HEAD did not move.

Triage before removal: every reachability hit and every duplicate was checked
against open gaps and accepted decisions. Owned items are listed once under
*Already tracked* and not repeated in the ledger.

## Headline

| | |
|---|---|
| Reachability hits, tests as roots | 2 (both owned, G-0417) |
| Production functions reachable only from tests | **41**, plus one whole package |
| Clone pairs at threshold 100, exclusions lifted | 7 (5 production, all owned, G-0472) |
| Defects, unowned | **4** (all derived) |
| Dead paths and dropped data flow, unowned | **17** |
| Convergent duplication, milestone-shaped, unowned | 7 |
| Convergent duplication, patch-shaped, unowned | 12 per-package bundles |
| Tracked records whose premise has changed | 5 |

Most of this predates the 2026-08-05 sweep. That pass ran the two mechanical
lenses and a light reasoning pass; this one is the first to run the data-flow
lens and the binary-roots reachability run, and the dates on the findings agree:
the last reader of `Tree.PlannedFiles` left in April, the branch helpers in `cli/authorize` predate their `gitops` equivalents
by a month or more. The instrument got
sharper; the tree did not degrade at the rate the count suggests.

## Cross-cutting patterns

**A seam lands and the extraction stops early.** The dominant shape is not drift
over time but an extraction commit that routed some call sites and left the rest
in the same file. `eachActiveMilestone` (`internal/check/acs.go:295`) has six
unrouted sites beside it; `standardTrailers` (`internal/verb/ac.go:463`) has five;
the hook-marker predicate is re-inlined eight times across `initrepo` and
`doctor`; the `human/` actor prefix test is spelled inline at thirteen sites
tree-wide with two named helpers nothing routes through. Nothing fails when an
extraction stops at N sites, so it does.

**Recognizers live in `entity`; constructors live everywhere.** `entity` owns
`PathKind`, `IsArchivedPath`, `ActiveFormOf`, `IsTerminal`, `KindFromID` and
`IsCompositeID`, and consumers re-derive the inverse of each: file layout is
constructed in four verb files, terminality is answered by six predicates, the id
grammar is mirrored in three packages with no pin to the source.

**The git layer keeps its runner private.** `gitops.run`/`output` are unexported,
so 52 production sites outside `gitops` build their own `exec git`; sixteen bypass
a `gitops` function that already exists, eleven copy the trailered-log reader, and
"does HEAD resolve" is implemented five times with four of them collapsing a
fault into "no commits". D-0045, which chose one such copy, was decided before
`gitops.HasHEAD` existed.

**Reachability with tests as roots hides a class.** Test-only production code —
the coherence rule specs D-0062 deliberately keeps, but also `scope.LoadScope`,
`verb.AddAC`, a family of `gitops` commit helpers and the whole `pluginstate`
package — is invisible to the run the ritual prescribes and visible only from the
binaries' roots.

**One condition, several dispositions.** An unparseable Go file is `continue`d in
about 38 policies, returned as an error in three, emitted as a Violation in one,
and declared unreachable in two.

## What would prevent it

- **A binary-roots reachability policy.** `deadcode ./cmd/aiwf ./cmd/stresstest`
  in CI with an allowlist naming the deliberate test seams (D-0062's coherence
  specs, ADR-0014 §4's `Target`, G-0195's `CanonicalTrailerKeys` mirror). It
  would have caught 41 functions and one package. Cost once; the allowlist is
  the retirement trigger.
- **Bans for the two most-copied predicates.** No `strings.HasPrefix(x, "human/")`
  outside one helper; no `exec.Command("git", …)` outside `gitops` without an
  allowlist entry. The same shape as the atomic-write and logging chokepoints.
- **A review rule at the moment of extraction.** When a helper is introduced,
  every existing site routes through it in the same change, or the leftover is
  filed as a gap before the commit. This is where the six-site residues came
  from.
- **Sweep on the ritual's own triggers**, not per merge: after a burst of fast
  machine-authored change, before a large refactor, when a package feels heavy.
  A full pass costs nine parallel reviewers and about twelve minutes; it re-finds
  everything not yet disposed of, so it earns its cost only after the previous
  batch has been worked. A cheaper standing signal is the two mechanical runs —
  binary-roots `deadcode` and unfiltered `dupl` — as an advisory make target
  reporting counts against this document.

## Ready to act on

**Tracked records whose premise changed:**

- **G-0453** (7-vs-8 SHA width): now tree-wide — `entityview.ShortHash` 7,
  `internal/cli/render/resolver.go:770` 8, `internal/verb/acknowledgeillegal.go:114`
  8, `internal/policies/skill_edit_provenance_backstop.go:282` 7. Measured:
  milestone pages print eight characters, `aiwf show` prints seven.
- **D-0045**: chose a private `hasCommits` in `entityview` because the only
  alternative considered was importing `cliutil`. `gitops.HasHEAD` landed eleven
  days later in a package `entityview` already imports. The premise no longer
  holds; re-decide.
- **G-0417**: unchanged, but the same rung-pair change that made
  `PreflightBranchNotFoundError` dead also made `AuthorizeOptions.BranchExists`
  dead in the verb (`internal/verb/authorize.go:445` discards it; the CLI still
  spawns `git show-ref` to compute it). Neither G-0417 nor D-0018 names the field.
- **G-0590**: its body says the HTML path renders the ack reason; `HistoryRow.Reason`
  is produced at `internal/cli/render/resolver.go:664` and read by no template.
- **G-0472 / G-0473 / G-0533**: unchanged. All five production clone pairs still
  fire at threshold 100 with the exclusions lifted, and the two stale grandfather
  entries still match nothing.

## Findings ledger

Locations are `file:line` as read at `b10eb2676`. Size is the disposal shape a
reader would expect — patch or milestone — not a commitment.

### A — defects

**A3. Doctor's binary-staleness check hardcodes `origin/main`.** *Derived.*
`internal/cli/doctor/binary_staleness.go:58` uses `refs/remotes/origin/main`
where `cfg.AllocateTrunkRef()` is the source. A repository using a different
trunk can miss the stale-binary advisory or compare against the wrong ref.
Patch; overlaps E-0093's doctor changes.

**A5. Actor derivation reads `user.email` from different repos per verb.**
*Derived.* `internal/cli/cliutil/actor.go:51` runs `git config --get user.email`
with no `cmd.Dir` (`:44` discards `root` with `_ = root`), so every verb reads the
cwd's repo; `internal/initrepo/initrepo.go:668` sets `cmd.Dir = root`. Six
`git config --get` readers carry four exit-1 conventions
(`gitops.go:282`, `committree.go:158`, `cli/check/git_config.go:33`,
`isolation_escape_oracle.go:197`, `actor.go:51`, `initrepo.go:667`). Whether cwd ≠
root is reachable for verbs other than `doctor` was not traced. Patch plus a
decision on which repo is the identity source.

**A6. Two terminality predicates disagree with the FSM.** *Derived.*
`internal/cli/status/worktrees.go:555-568` `isTerminalStatus` lists
`StatusDeprecated`, which is not terminal in any kind (`transition.go:50`:
`deprecated → retired`), and ignores its `kind` argument;
`internal/verb/authorize.go:739-741` `isTerminalStatus` is
`len(AllowedTransitions)==0`, which is *true* for an unknown status where
`entity.IsTerminal` is false, so `authorize` refuses a junk-status entity as "at
terminal status" (`:359-361`) instead of the R1 "unrecognized" refusal. Unreachable
today because ritual branches yield only E/M/G drivers. See D1 for the full set.

### B — dead paths and dropped data flow

**B1. The scope FSM's legality check is test-only; production replays scopes
through a second, permissive walker.** *Verified.* `scope.LoadScope`,
`IsLegalScopeTransition`, `classifyTransition`, `indexTrailers`
(`internal/scope/scope.go:85-221`) have no production caller.
`cliutil.ReplayScopes` (`internal/cli/cliutil/scopes.go:62-107`) does the real
work — multi-scope, and it silently drops a pause with no active scope or a resume
with no paused one (`:82,87`) where `LoadScope` errors — and
`internal/verb/authorize.go:722` duplicates its `mostRecent`.
`docs/design/design-decisions.md`'s "one Go function for legal transitions" names,
for scopes, a function nothing consults at runtime. Milestone: pick one walker and
decide whether illegal history is an error or is ignored.

**B2. `internal/pluginstate` is linked into neither binary.** *Verified*
(`go list -deps`). ADR-0014 §5 replaced its use with `MaterializedRituals`; only
its own tests reach it. Patch: delete with tests.

**B3. `Tree.PlannedFiles` is written by four verbs and read by nothing.**
*Verified.* `internal/tree/tree.go:44-49` (field, whose doc says checks consult it),
`:247-255` (`HasPlannedFile`, zero callers); writers `internal/verb/common.go:76,93`,
`import.go:413`, `reallocate.go:579`. Thirteen check files read disk
unconditionally; the projection excludes `body-prose-id` via
`skipDuringProjection` (`common.go:267`) instead. Patch.

**B4. Forty-one production functions reachable only from tests.** *Verified*
(`deadcode` from the binaries' roots, test-support packages excluded).

| package | functions | disposition |
|---|---|---|
| `internal/scope` | `LoadScope`, `IsLegalScopeTransition`, `classifyTransition`, `indexTrailers` | B1 |
| `internal/gitops` | `Mv`, `Add`, `Commit`, `CommitAllowEmpty`, `HeadSubject`, `HeadBody`, `HeadTrailers`, `HasRemotes`, `CanonicalTrailerKeys`, `SortedTrailers` | `CanonicalTrailerKeys` is G-0195's mirror guard; the rest unowned |
| `internal/verb` | `AddAC` (the CLI calls `AddACBatch`), `AsCoherenceError`, `declaredCoherenceRules`, `declaredForcePredicatedRules`, `declaredCoherenceTrailerAxis`, `PreflightBranchNotFoundError.{Error,Code}` | coherence trio owned by D-0062; `Preflight*` by G-0417; `AddAC`, `AsCoherenceError` unowned |
| `internal/trunk` | `LocalRefIDs`, `RemoteRefIDs`, `LocalRefHits`, `RemoteRefHits`, `Result.IDStrings` | accessors with no production reader |
| `internal/skills` | `GuidanceBytes`, `StatuslineBytes`, `Materialize`, `MaterializeTo` | `MaterializeTo` is ADR-0014 §4's `Target` seam; the byte accessors are policy-test inputs |
| `internal/cli/cliutil` | `ReorderFlagsFirst`, `flagName`, `IsVerbGroup` | unowned |
| `internal/tree` | `LoadError.Error`, `LoadError.Unwrap` | `LoadError` is built four times and never used as an error |
| `internal/check` | `walkStatusChanges` | comment: "retained for the existing unit tests" |
| `internal/config` | `AcceptedKeys` | policy-test input |
| `internal/entity` | `SovereignActShapes` | policy-test input |
| `internal/cli/check` | `EnumerateRegisteredVerbs` | policy-test input |
| `internal/cli/status` | `renderAge` | unowned |
| `internal/version` | `Skew.String` | Stringer never invoked |

Patch: per-entry triage; delete the unowned, name the deliberate seams in an
allowlist the policy under *What would prevent it* reads.

**B6. The stress seed is logged as replayable and nothing can replay it.**
`internal/stresstest/repeat.go:12-15` claims replay; `cmd/stresstest/run.go:19-38`
has no `--seed` flag and `:171-172` always draws `rand.Int64()`; 15 of 16
constructors discard the seed (`registry.go:89-132`); only `verb_sequence.go:83`
consumes it. Patch: add `--seed`, or drop the claim and stop logging a decoy.

**B7. HTML view-model fields computed at real cost and read by no template.**
`internal/cli/render/resolver.go:81,226` `LastActivity` (a history lookup per
epic and per milestone), `:315` `MilestoneData.LinkedEntities` (a full
`ForwardRefs`+`ReferencedBy` walk per milestone; the template reads the
`LinkedDecisions` subset filtered at `:316-320`), `:122,148-150`
`AllFileName`/`IncludeArchived`/`ActiveFileName`, `:467,491,613,652-664,671-726`;
`internal/htmlrender/pagedata.go:17,409-415` declared and never produced. None of
the seven embedded templates references any of them. `*-all.html` is never emitted
(measured: zero files) while `pagedata.go:38-44` still documents the pair.
`IsCurrentGaps` (`pagedata.go:93`) is consumed by `_sidebar.tmpl:36-40` and produced
by nothing (measured: no `aria-current` on `gaps.html`); `htmlrender.go:188` passes
`includeArchived=true` unconditionally, so the kind index is titled "All gaps" over
a chip strip defaulting to Active. Patch.

**B8. Shipped no-op flags and blank-identifier keep-alives.** `aiwf render --scope`
and `--no-history` (`internal/cli/render/render.go:66-67,300-301`; help says
"reserved; not yet implemented"; carried by `help_banner_drift_test.go:92` and
`completion_drift_test.go:93`; the gap-truth audit already recorded `--scope E-0058`
rendering the whole site); `htmlrender.Options.Scope`/`Root` never read;
`internal/cli/cliutil/actor.go:44` `_ = root`; `internal/cli/add/add.go:440` `_ = k`.
CLAUDE.md bans the blank-identifier keep-alive. Patch.

**B9. `aiwf-prior-parent` is written by `move` and read by nothing.**
`internal/gitops/trailers.go:21`, `internal/verb/move.go:117`; absent from
`bulkTrailerKeys` (`revwalk.go:109-121`), from `HistoryEvent`, from
`ReadHistoryChain`'s format, and from every check rule. Patch, or a decision that
it is raw-git-only.

**B10. The `hooks:` block has two decoders and the typed one has no production
reader.** `config.Load` decodes non-strictly into `Config.Hooks`/`HookDecision`
(`internal/config/config.go:75-93,892-913`); every real reader re-parses via
`aiwfyaml.decodeHooks` (`internal/aiwfyaml/hooks.go:54-75`, `KnownFields(true)`).
They disagree on `hooks: {x: {enabled: true, typo: 1}}`: accepted at
`internal/cli/update/update.go:120`, exits internal at `:166`. `HookDecision` has
no production caller. Patch.

**B11. Branch spec cells are a test-name registry wearing a `Rule` struct.**
`Preconditions`/`Verb` on the fourteen hand-written cells
(`internal/workflows/spec/branch/rules.go:41,54,74,…`) are read by no test and use
subjects outside `EvaluatePredicate`'s vocabulary (`spec/evaluate.go:369-416`); the
112 generated cells (`rules_m0162_ac3.go`) carry `Outcome: Legal` regardless of
behaviour (`:32-37`: "fires isolation-escape" → Legal) and are consumed only by the
ID-level bijection (`internal/policies/m0162_ac4_bijection_test.go:89`);
`branch.AntiRules()` (`antirules.go:78-84`) returns an empty slice under a comment
claiming an aggregation no call site performs. Nothing drives `branch.Rules()`
against a verb. Milestone with a decision: derive real outcomes from each
scenario's expectation, or shrink to an id registry and stop calling them rules.

**B12. The cell generator hardcodes a dead worktree path.** *Measured.*
`scripts/m0162-build-ac3-cells.py:6` sets `root = Path("/workspaces/aiwf-M-0162")`;
the regeneration instruction at `rules_m0162_ac3.go:19-21` cannot be followed.
With the path patched in a scratch copy it reproduces the committed cell set
exactly; the header comment says 112 cells and the file holds 113. Patch.

**B13. `ensureSkills` dry-run is a separate implementation that undercounts.**
`internal/initrepo/initrepo.go:716-726` lists the verb skills (19 from the embed);
the real arm (`:727-742` → `skills.go:422-509`) writes 39 skills, 4 agents and 6
templates and surfaces unknown `agents:` keys, which dry-run hides. Derived from
the embed. Patch.

**B14. `SyncHookMaterialization` discards both hook-settings results.**
`internal/cli/cliutil/hooks.go:103,109` (`_, wireErr :=`, `_, unwireErr :=`);
`BackupPath`, `Wrote`, `WiredEvents`, `RemovedFromEvents`
(`internal/skills/hooks_settings.go:14-31,168-184`) have no reader outside tests,
so the `.bak` of `.claude/settings.json` is taken silently — unlike the statusline
path (`cliutil/statusline.go:120-132`), which prints it. Patch.

**B15. Two policies emit findings outside the `Violation` envelope.**
`internal/policies/no_dangling_entity_refs.go:45` and
`aiwf_promote_epic_active_audit.go:114` return `[]string`; their wrappers call
`t.Errorf` directly; neither carries a `Policy: "` literal, so the firing-fixture
inventory (`firing_fixture_presence.go:143-163`) and `PolicyViolationPolicyIDLiteral`
never see them. Patch.

**B16. Stress `Run` state read only by `stress`-tagged tests.**
`internal/stresstest/concurrent_milestone_race.go:94-99,541-551` spawns two extra
`aiwf show` subprocesses per attempt for values the production oracle
(`classifyMilestoneRaceOutcomes`, `:572`) never takes; `Collided()`
(`cross_worktree_id_race.go:71-76`) is unreachable from the runner; `wantRunIDs`
(`concurrent_writer_at_scale.go:61-66`) and `verbEnvelope.Result.Path`
(`verbenvelope.go:36`) likewise. Patch.

**B17. Small unconsumed values**, one bundle: `applyTx.ctx`
(`internal/verb/apply.go:124`); `AuthorizeOptions.BranchExists` (see G-0417 above);
`AllowResult.Reason` (`allow.go:177`, the consumer reads `Err`);
`gitops.Worktree.HeadSHA`, `repolock.Lock.path`, `gitops.errBlobReaderClosed`
(documented for `errors.Is`, unexported), `pathutil.ErrNotAbsolute`;
`contractconfig.Resolved.Entry`; `statusChange.Parent`/`walkError.Parent` in the
FSM walker; `gitEnv()` (`internal/gitops/gitops.go:512-518`) a constant-nil
producer that four `gitops` sites bypass anyway; `HasRef` ≡ `CommitExists`
(`refs.go:215,250`, byte-identical bodies).

### C — convergent duplication, milestone-shaped

**C1. `gitops` keeps its runner private, so the tree re-implements git plumbing.**
52 production `exec git` sites outside `gitops` in 24 files. Sixteen bypass an
existing `gitops` function: `internal/cli/authorize/authorize.go:171-235`
(`ritualLocalBranches` → `LocalBranchRefs`, `currentBranch` → `CurrentBranch`,
`branchExists` → `BranchExists`, `branchTipSHA` → `ResolveCommitSHA`),
`internal/cli/doctor/doctor.go:957` (`currentBranch`),
`internal/check/acks.go:196` (`resolveFullSHA` → `ResolveCommitSHA`, byte-identical
command), `reflog_walk.go:233` (`isAncestorViaGit` → `IsAncestor`),
`internal/cli/check/provenance.go:299` (→ `CommitExists`), `commit_msg.go:404,411`
(→ `HasRef("MERGE_HEAD")`, `StagedPaths`), `internal/cli/status/worktrees.go:1337`
(→ `DirtyPaths`), `doctor/selfcheck.go:417` (→ `Add`/`Commit`). Eleven copy the
`--pretty=tformat` + `\x1e`/`\x1f` trailered-log reader; twelve copy the
`exitErr.Stderr` wrap. Trailer key→value indexing exists four times
(`internal/check/provenance.go:861` ≡ `internal/scope/scope.go:215` ≡
`internal/cli/cliutil/scopes.go:126` ≡ `internal/verb/coherence.go:366`) plus six
inline scans. "Does HEAD resolve" exists five times; `gitops.HasHEAD`
(`gitops.go:229`) distinguishes a fault from an empty repo and the four copies
(`cliutil/gitstate.go:11`, `entityview/historyevent.go:82`,
`check/fsm_history_consistent.go:325`, `doctor.go:972`) collapse both to false.
Milestone: export the runner or a `ForEachCommit`, then route; absorbs G-0672's
seam and re-opens D-0045.

**C2. The parse loop is re-inlined in about 45 policies with four parse-error
dispositions.** 49 `WalkGoFiles(` callers and 51 `token.NewFileSet()` sites;
`continue` at `read_only.go:93-98` and about 37 more, `return err` at
`stress_lane_census.go:98`, a Violation at `test_setup_presence.go:92-99`,
`//coverage:ignore` at `violation_policy_id_literal.go:48`. The partial helper
`parseGoFilesInDir` (`skill_coverage.go:657`) serves none of the three
`os.ReadDir`+`ParseFile` sites. Milestone; absorbs G-0508.

**C3. Entity file layout is constructed in four verb files and the archive target
three ways.** `internal/verb/add.go:504-531`, `import.go:296-328,364`,
`linkrewrite.go:218-226`, `pathrewrite.go:69-95`; `archive.go:407-440` three
`archiveTargetFor*` where `Join(Dir(p), "archive", Base(p))` is the inverse of the
existing `entity.ActiveFormOf`; `computeArchiveMoves` epic and contract arms
(`:291-320`, `:341-356`) are the same twelve lines. `entity` owns every recognizer
and no constructor; `entity.go:4-5` "knows nothing about the filesystem" is already
false. Milestone.

**C4. No canonical id index on `Tree`; six consumers rebuild one and three key
raw.** `tree.ByID` (`internal/tree/tree.go:529-537`) is a linear scan with
`Canonicalize` per element, called per element inside `parentChainReaches`
(`:473-488`) and `ResolvedArea` (`:547-559`). Canonical indexes are rebuilt at
`internal/check/check.go:606-619`, `body_prose_id.go:212-225`,
`internal/contractcheck/contractcheck.go:48-51`, `internal/verb/import.go:94-97`;
**raw** at `check.go:345-359` (`idsUnique.seen`), `:934-939`
(`adrSupersessionMutual`) and `DisputedTrunkIDs`, so a narrow and a canonical
spelling of one number in the same tree never collide in `ids-unique`, against
"same-state is not same-spelling". Whether raw keying there is deliberate is
undocumented. Milestone.

**C5. `aiwf.yaml` is loaded more than twenty times per process with four failure
behaviours.** Fail-loud: `internal/cli/cliutil/treeload.go:32-35`,
`update/update.go:120-124`, `worktree/worktree.go:149-152`, `doctor/doctor.go:205-213`.
Silently default: `cli/check/check.go:192`, `treeload.go:141,156,171,184,198,219`
(six `Configured*` helpers), `resolvelogger.go:28,72`, `doctor.go:445,726`,
`doctor/guidance.go:27`, `worktree.go:114` (the same verb fails loud on the same
file at `:149`), `render/render.go:315,412`. `initrepo` re-parses five times per
run (`initrepo.go:590,605,751,1206,1224`) while every caller already holds `cfg`,
and two of those loads swallow every error into "no aiwf.yaml or unreadable".
`config.Load` classifies a parse failure as a non-`ErrNotFound` error
(`config.go:928`). Milestone: one load per
invocation and one failure classification.

**C6. `cli/render.Resolver` mirrors five unexported `htmlrender` helpers and three
walks; `defaultResolver` is test-only production code.**
`internal/cli/render/resolver.go:754` ≡ `internal/htmlrender/default_resolver.go:346`
(`acRollup`/`acMetTotal`), `:736` ≡ `paths.go:427`, `:745` ≡ `htmlrender.go:314`,
`:173`/`:192` ≡ `default_resolver.go:263/282`, `sidebarWithStatus` (`:346-381`) ≡
`sidebar` (`:96-133`), IndexData and EpicData walks; three plural→kind tables. Each
copy's comment says it mirrors the original because the original is unexported.
`defaultResolver` (357 lines) is reachable only through the nil-`Data` fallback
(`htmlrender.go:139-141`), which only `htmlrender`'s own tests take. G-0222
(wontfix) declined a conformance suite, not the duplication. Small milestone.

**C7. Ancestry is re-walked by subprocess while an in-memory `CommitDAG` exists in
the same invocation.** `internal/check/fsm_history_consistent.go:642` runs one
`git rev-list` per ack; `post_cutoff.go:61-87` runs `git rev-list cutoff..HEAD`;
`id_rename_untrailered.go:145-200` re-reads trailers that `head []HeadCommit`
already carries — against `BuildCommitDAG` (`internal/cli/check/provenance.go:90`)
and `orphan_dag.go:105` `isAncestor`. D-0030 forbids caching verdicts across
invocations, not deriving ancestry in-process. Milestone; byte-identity of the
resulting findings is argued, not measured.

### D — convergent duplication, patch-shaped

Each bundle names a helper that exists and the sites that re-implement it. The fix
is routing, not design.

**D1. Terminality and closed sets.** `entity.IsTerminal` (`transition.go:129`) is
canonical with 21 production callers; re-derived at `internal/cli/status/worktrees.go:555`,
`internal/verb/authorize.go:739`, `internal/verb/auditonly.go:263-267,276-290`
(a hand-copied per-kind table whose comment mandates a same-commit update nothing
enforces, in a file that already derives cancel terminals from `AllowedStatuses`
at `:133-141`), `internal/cli/cliutil/provenance.go:271` (`IsTerminalPromote`),
`internal/workflows/spec/evaluate.go:204`. `auditonly.go:292-310` re-lists
`entity.IsAllowedACStatus`/`IsAllowedTDDPhase`. A6 names the two that are wrong.

**D2. The `human/` actor predicate**, inline at `internal/verb/promote_sovereign_act.go:42`,
`allow.go:143`, `authorize.go:310`, `acknowledgeillegal.go:75`, `acknowledgemistag.go:47`,
`coherence.go:219,358`, `internal/cli/cliutil/provenance.go:69`,
`internal/cli/archive/archive.go:144`, `internal/cli/importcmd/importcmd.go:131`,
`internal/check/provenance.go:168,176,205,890` (`isHumanRoleID`),
`internal/gitops/trailers.go:232`. Two of the thirteen are named helpers nothing
else routes through.

**D3. The AC heading grammar.** Four readers — `internal/entity/body.go:198`
(`^### (AC-\d+)\b`, accepts `AC-1.5` and attributes its prose to `AC-1`),
`internal/verb/ac.go:587`, `internal/check/acs.go:668` (one space),
`internal/check/entity_body.go:58` (`\s+`) — and three writers (`ac.go:490,561,608`).
`###  AC-1` with two spaces is a heading for `entity-body-empty/ac` and not for
`acs-body-coherence` or `ParseACSections`. `entity_body.go:269` `scanACBodies`
re-implements `entity.ParseACSections` and bypasses `ACSectionIsEmpty`
(`body.go:262`), whose comment says it was exported so a verb-time gate and a
check-time rule "consult the same definition of empty without drifting".

**D4. Milestones under an epic, repeated scans and mixed ID matching.**
`internal/check/epic_terminal_children.go:52-64` (canonical),
`internal/workflows/spec/evaluate.go:219-232` (literal),
`internal/roadmap/roadmap.go:61-67,140-146` (raw key, then a rescan per epic in
the same render), `internal/cli/status/status.go:473-480` (canonical),
`internal/cli/render/resolver.go:504-514` (rescan per page).
`import.go:305-317` rewrites `parent:` to the resolved epic's stored ID. AC progress is
computed three ways (`default_resolver.go:346`, `resolver.go:754`, `status.go:149`).
One `Tree.ChildrenOf(id)` and one exported progress function.

**D5. Policies.** Function-body text extraction copied verbatim seven times
(`read_only.go:106-111`, `apply_callers_lock.go:65-70`, `authorized_by_via_allow.go:38-43`,
`principal_write_sites.go:41-46`, `verbs_validate_then_write.go:67-72`,
`integration_tests_assert_trailers.go:97-102`, `no_retry_loops_on_git.go:63-68`);
test-package discovery three times (`test_setup_presence.go:43-62`,
`git_test_env_harden.go:70-86`, `shipped_prose_assertion.go:126-152`) with two
roots and two skip lists; the `AIWF_COVERAGE_BASE` read and the empty/zero-SHA
guard five times (`branch_coverage_audit.go:55,86`, `comment_history_attrition.go:50,151`,
`test_executable_write.go:54,78`, `skill_edit_provenance_backstop.go:67,96`,
`changelog_completeness.go:370`) with the contract restated in three doc comments;
`walkMarkdown` (`design_doc_anchors.go:142-169`) fills `FileEntry.Path` with an
absolute path against `policies.go:45`'s contract, nothing reads it, and two policies
re-walk `.md` themselves (`discoverability.go:175`, `milestone_section_name_resolution.go:109`);
"check layer" defined three ways (`finding_codes_documented_in_skill.go:104-115`
five prefixes; `discoverability.go:94`, `findings_have_tests.go:122` two;
`finding_hints.go:224`, `finding_code_adoption.go:61` one); `relTo` bypassed by
five inline `filepath.Rel`+`ToSlash`; `findTopLevelVerbs` (`skill_coverage.go:559`)
production code with test-only callers. Also six `git` shell-outs each hand-build
`exec.Command` with their own error wrap (`branch_coverage_audit.go:334,377`,
`changelog_completeness.go:196,231,242`, `comment_history_attrition.go:79`);
sequence with C1.

**D6. Verb.** `standardTrailers` is bypassed at `add.go:206`, `rename.go:116`,
`import.go:430,446`. The `aiwf.yaml` write tail spelled five times with no
`planEntityWrite` twin (`contractbind.go:115-136,174-192`,
`contractrecipe.go:64-84,117-131`, `add.go:491-500`), three carrying the same
`coverage:ignore`; Rename and Retitle re-inline about twelve lines of rename
plumbing (`rename.go:86-109` vs `retitle.go:135-143,172-180`); the dir-shaped-kind
predicate `Kind == KindEpic || Kind == KindContract` inline at `rename.go:102,137`,
`retitle.go:174`, `pathrewrite.go:74`, `archive.go:206,238`;
`expectedActivationBranch` (`promote_branch_guard.go:127-152`) duplicates
`internal/cli/check/provenance.go:581-613` across the layer boundary with the
stated reason "verb cannot import cli" — which argues for `branchparse`, not two
copies; `auditOnlyTrailers` (`auditonly.go:218-230`) re-inlines
`transitionTrailers`; `joinIDs` (`contractrecipe.go:172`) and `runeListString`
(`add.go:354`) hand-roll `strings.Join`.

**D7. Check.** `eachActiveMilestone` (`acs.go:295`) bypassed at
`acs.go:55,211,244,368,568` and `milestone_release_note.go:63` after the seam
landed; `isArchivePath` (`entity_id_narrow_width.go:132`, any segment named `archive`) vs
`entity.IsArchivedPath` (ADR-0004 position); `matchesAnyGlob` (`area_mistag.go:199`)
vs `claimedByAnyArea` (`area_coverage.go:199`); three identical severity escalators
(`entity_body.go:79`, `area_unknown.go:42`, `doc_id_width.go:204`).

**D8. Install and config.** "Is this hook ours?" re-inlined as a substring match
at `internal/initrepo/initrepo.go:1357,1465,1546,1634` and
`internal/cli/doctor/doctor.go:495,612,697,748`, while the guidance marker is
line-anchored (`GuidanceMarkerLineIdx`, `:859-866`) and `initrepo.go:855-858`
claims the two follow the same pattern; `.gitignore` line-set membership four times
(`:1073-1080`, `:1246-1257`, `:859-866`, `skills/statusline.go:450-454`) and removal
twice (`stripExactLines` `:1177-1195` ≡ `stripHTMLOutDirLines` `:1262-1280`, the
first's comment saying the semantics "live in one place"); `sortedKeys`
(`aiwfyaml.go:638-649`) a hand-rolled insertion sort beside `sort.Strings`; two
`__AIWF_VERSION__` constants and renderers (`guidance.go:20,36`, `statusline.go:19,32`);
an `fs.FS` adapter (`recipe.go:197`, `os.go:11`) serving one `fs.ReadFile`;
`initcmd.gateAndPersistHookDecisions` (`initcmd.go:206-245`) and
`update.gateAndSyncHookDecisions` (`update/hooks.go:20-69`) near-identical twins.
`Config.Hosts` is parsed and documented with no reader — a recorded reservation
for multi-host, in tension with CLAUDE.md's out-of-scope list.

**D9. CLI.** The read-verb output scaffold hand-rolled nine times
(`contract/verify.go:41`, `show/show.go:76`, `template/template.go:57`,
`render/render.go:64`, `status/status.go:287`, `schema/schema.go:43`,
`list/list.go:124`, `history/history.go:48`, `check/check.go:56`) with three
validation spellings, a `--pretty` warning on three of nine, and a fourteen-copy
`render.JSON → Errorf → ExitInternal` tail each carrying the same
`//coverage:ignore`; `cliutil.AddFormatFlags` (`outputformat.go:77`) exists but
registers `--trace`, so read verbs bypass it. `milestone tdd` and `depends-on`
(`milestone.go:88-114,178-205`) skip `BeginVerbDiag`, and
`verb_scaffold_convergence.go:169-194` detects re-inlines but not omissions.
`status`, `render`, `template` and `schema` (`root.go:185-199`) never receive the
root-minted correlation id and fire no diagnostic event — `status` being the verb
the post-commit hook runs on every commit (the statusline script itself spawns no
verb). `worktree add` is a hybrid: dead
`--trace`, an envelope only for lock contention, `BeginReadVerbDiag` on a verb that
takes the lock. `init`, `update` and `worktree add` copy the artifact ledger
(`initcmd.go:126`, `update.go:135`, `worktree.go:229`) and write three remedies for
one hook-collision condition that already disagree on which verb to re-run.
`list.IsKnownKind` (`list.go:272`) ≡ `cliutil.ParseKind` (`verbhelpers.go:15`); the
kind-completion closure is byte-identical in `add.go:122`, `schema.go:46`,
`template.go:60`. Five `metadata` keys (`swept_count`, `imported_count`,
`shape_only`, `filtered_out`, `actual_area`) have no hit in `docs/`, skills or
CLAUDE.md.

**D10. Render.** `roadmap.go:166-179` `readEpicGoal` ≡ `entityview.ReadEntityBody`
and `:187-212` `extractSection` ≡ `entity.ParseBodySections(body)["goal"]` (the
HTML path already takes the helper route); two markdown escapers
(`roadmap.go:252` vs `status.go:1498`, different character sets, both escaping
titles into table cells); `status.go:1439-1445` hand-rolls two glyphs for STATUS.md
where `:1118` routes through `render.StatusGlyph` for four, so draft and cancelled
rows carry no marker and no golden pins either. `internal/cli/render/resolver.go:632`
joins root and path before `ReadEntityBody` joins again.

**D11. Stress harness.** `runAiwfJSON` (`verbenvelope.go:104`) bypassed at
`concurrent_id_allocation.go:70,107`, `concurrent_move.go:118,151`,
`cross_worktree_id_race.go:100,192`, `concurrent_milestone_race.go:181,527`,
`concurrent_writer_at_scale.go:133`, with four byte-identical `{execErr, out}`
structs; `errorCode` (`archive_during_active_scope.go:217`) ≡ `envelopeErrorCode`;
**two "build the aiwf binary under test" implementations** (`stresstest/binary.go:25`
vs `cliutil/testutil/proc.go:91`), the second ad-hoc-signing on darwin and the
first not, so `make stress` on a macOS host runs an unsigned binary — the G-0128
state; the commit-delta bracket copied five times (`concurrent_id_allocation.go:86`,
`concurrent_move.go:130`, `concurrent_milestone_race.go:500`, `verb_sequence.go:275`,
`disk_fault.go:65`); four finding-search loops, one keyed on the literal
`"promote-on-wrong-branch"` (`promote_on_wrong_branch_detection.go:144`) where
`check.CodePromoteOnWrongBranch.ID` exists and `verb_sequence.go:59` already uses
it; `cellcoverage/fixture.go:666,688` re-split composite ids by hand instead of
`entity.ParseCompositeID`; `nextTDDPhaseTowards` (`:606-621`) re-encodes the phase
FSM edges.

**D12. Consumer-side grammar mirrors with no pin.** `internal/check/body_prose_id.go:97-101`
(`strictBareIDPattern`, comment: "mirrors entity.idPatterns"),
`internal/entityview/historyevent.go:330`, `internal/aiwfyaml/aiwfyaml.go:90`
(`^C-\d{3,}$`) — vs `entity.KindFromID`/`IsCompositeID`/`ValidateID`; `aiwfyaml`
can import `entity` without a cycle. `internal/verb/import.go:270-279` re-creates
`Parse`'s `KnownFields` decoder. No test pins any mirror to its source.

## Scorecard

Repo-level, against the `wf-codebase-health` rubric. Strong means the principle
holds with a named chokepoint or a consistent pattern; Weak means it holds in
places and the ledger names where it does not.

| Principle | Verdict | Evidence |
|---|---|---|
| A1 cohesion | Strong | one rule, verb or scenario per file throughout; exceptions `initrepo.go` (five concerns in 1742 lines), `auditonly.go`'s own tables |
| A2 coupling | Weak | every verb imports the catch-all `cliutil` (G-0227); `cli/render` mirrors `htmlrender` because its helpers are unexported (C6); `changelog_completeness.go` reaches into a sibling for separators (G-0672) |
| A3 layering | Strong | `layering_direction.go` enforces the tier graph and no upward import was found — but it governs imports only, and `check` spawns git directly at fifteen sites (C1) |
| B1 typed interfaces | Strong | named structs at every boundary; exceptions: nine-to-eleven-positional `Run` signatures in `cli/check`, `status`, `initcmd`, `update`; four identical private structs in the stress harness |
| B2 schemas | Weak | `Parse` is `KnownFields(true)` but `hooks:` has a second non-strict decoder (B10); the raw-report JSONL schema lives only in its writer; five `metadata` keys undocumented (D9) |
| B3 invariants | Weak | `verb.go:38` "exactly one of Findings, Plan, NoOp" is false at `add.go:239`, `rename.go:114`, `retitle.go:191`; `FileEntry.Path` contract broken by `walkMarkdown` (D5); `refs.go:11` says `ErrRefNotFound` is wrapped by `HasRef`, which never wraps it |
| C1 single source | **Weak** | the sweep's dominant class: terminality ×6, id index ×6, path layout ×5, AC heading ×4, `aiwf.yaml` ×20+, HEAD probe ×5, trailer index ×4 |
| C2 idempotence | Strong | ADR-0036 NoOp guards chokepointed by `noOpClaimScopes`; every `ensure*` converges to Preserved |
| C3 atomic writes | Strong | `pathutil.AtomicWriteFile` plus its chokepoint; all twelve non-test `os.WriteFile` sites allowlisted with rationale; caveat: exemptions are whole-file and `os.CreateTemp` is outside the scanned set |
| C4 versioned schemas | Weak | legacy `actor:`/`aiwf_version:` tolerance is hand-rolled line stripping; `manifest.supportedVersion` is the only declared schema version |
| D1 behaviour pinned | Strong | integration through `cli.Execute`; a pure `classify*` per stress scenario; every policy has a firing fixture, the ledger down to one deliberate entry |
| D2 seam equivalence | Weak | `LoadScope` vs `ReplayScopes` have no equivalence test and disagree (B1); the two `hooks:` decoders unpinned (B10); JSONL composition preserves raw events without typed interpretation; Strong where it exists (`EventFromCommit`, the single-pass index) |
| D3 branch coverage | Strong | diff-scoped gate inside `make ci` |
| D4 altitude | Strong | subprocess at the seam, in-process fixtures |
| D5 findings become checks | Strong | the policies package is this principle; caveat B15 |
| E1 structured logs | Strong | slog via `BeginVerbDiag`, `forbidigo` fence; gaps: `status`, `render`, `template`, `schema`, `milestone` emit no event (D9) |
| E2 designed failures | Weak | one condition, four dispositions for an unparseable Go file (C2) and for a malformed `aiwf.yaml` (C5); four HEAD probes collapse a fault into "empty" (C1) |
| E3 audit trail | Strong | trailers on every plan; edge: `aiwf-prior-parent` write-only (B9) |
| E4 self-explaining errors | Strong | a remedy per path role in verb refusals; Weak in `cli`, where "not found" is reported four ways (G-0483) |
| F1 names | Weak | `isTerminalStatus` ≠ `IsTerminal` (A6); `skills.HooksDir` is `.claude/hooks` while `gitops.HooksDir` is `.git/hooks`; `--root` help "(default: cwd)" on a verb that walks up; `Outcome: Legal` on cells that fire (B11) |
| F2 comments | Weak | drafting-history residue (`fsm_history_consistent.go:190` "The pre-lift line was", `acks.go:12` "Lifted from", eight drop-narration blocks in `branch/rules.go`); comments asserting a parity that does not hold (`initrepo.go:855`, `reflog_walk.go:146`, `pagedata.go:38`) |
| F3 decision records | Strong | guards cite the ADR, decision or gap that pins them at the enforcement site |
| G1 reproducible | Strong | no clock in core; sorted iteration in render; Weak: the stress seed is a decoy (B6) |
| G2 reversible | Strong | every verb doc answers "what undoes this"; LIFO undo journal (D-0029); Weak: dry-run is a separate implementation in three `initrepo` steps (B13) |
| G3 observable | Weak | four read verbs invisible to the diagnostic log (D9) |
| H1 reuse | **Weak** | helpers exist and are bypassed at six, eight, nine and thirteen sites (D7, D8, D9, D2) |
| H2 no dead weight | **Weak** | 41 test-only production functions (B4), one unlinked package (B2), `PlannedFiles` (B3), a dozen unread view-model fields (B7), no-op flags (B8), `gitEnv()` (B17) |
| H3 additions carry | Weak | per-subject mandates with no retirement: `terminalStatusesForKind`'s same-commit note (D1), the `ackedSHAs` consumer roster (`acks.go:17-45`), per-scenario `*ExpectedWarnings` (D-0063, accepted), each new policy hand-wired three times (D-0025, accepted) |

## Already tracked

Owned by an open gap or an accepted decision and not repeated above:
G-0417 (dead `branch-not-found` code, retained per D-0018); G-0453, G-0454, G-0455
(the G-0447 remainder); G-0472, G-0473, G-0533 (the `dupl` families and catalogue;
E-0077's cancellation records that two of the four collapses are not worth doing);
G-0477 (dead guard in `isTopLevelActorLine`); G-0672 (the two git range scans);
G-0508 (the `internal/verb` Decl walk); G-0535 (three `cmd/aiwf`-scoped policies);
G-0545 (the coherence-guard seam policy); G-0456 (the two prelude arms); G-0563
(bare `tree.Load` vs `LoadTreeWithTrunk`); G-0169 (verbs with no `--format`);
G-0483 / D-0044 (code-less verb errors); G-0459, G-0460, G-0458 (event-shaped
verbs, repeat authorize, AC phase input); G-0684 (quoting in the section-dropped
walker); G-0692 (stale `check --fast` help); G-0666 (the 64 KB scanner ceiling);
G-0644 (the orphan walk verdict);
G-0157 (the per-worktree subprocess fan-out in `status`); G-0400 (verb coverage of
the stress catalogue); G-0555 / G-0645 (shared-binary temp dirs); G-0468, G-0491
(stress oracle shape, `ETXTBSY`); G-0222 (wontfix, the resolver conformance
matrix); G-0227 (the `cliutil` split); D-0045 (see *Ready to act on*); D-0062
(coherence specs test-consumed by design); D-0063 / M-0257 (per-scenario warning
baselines); D-0025 (the bespoke policy subsystem); D-0030 (no cross-invocation
verdict cache); ADR-0011 (the spec tables consumed by policies only); ADR-0014 §4
(the `Target` seam); ADR-0017 (opt-in diagnostics).

## What no measurement here settles

Each of these is derived from reading, with the command that would settle it:

- **A4** — `aiwf contract bind c-1 …` then `git log -1 --format=%(trailers)`;
  expect `aiwf-entity: c-1`.
- **A5** — from a cwd whose repo has a different `user.email` than `--root`'s,
  `aiwf whoami --root <root>` vs `aiwf init --root <root>`'s derived actor.
- **D3** — a milestone body with `###  AC-1 — x` (two spaces); `aiwf check`
  should report it as a heading under one rule and as missing under another.
- **C7** — that DAG-derived ancestry yields byte-identical findings to the
  per-ack `git rev-list`.
- **D11** — whether the unsigned stresstest-built binary crashes on a current
  macOS host; no darwin machine was available and CI is ubuntu-only.
- The Playwright suite under `e2e/playwright` (opt-in, in no workflow, last
  touched 2026-05-12) was not run.
- Reviewer censuses (about 45 parse loops, 52 git sites, 13 actor sites) are
  grep counts; a subset of each was read in full.
- `deadcode` under RTA under-reports methods on types that flow into interfaces:
  `tree.(*Tree).HasPlannedFile` has no caller anywhere and neither run lists it.
  The grep census is the supplement, not a replacement.

## Provenance

Swept 2026-09-17 against the working tree at `b10eb2676`, method as described
under *Scope and method*. No fixes were applied in the same pass, so every
citation reflects the tree as read. The August 2026 sweep that produced G-0472,
G-0473 and G-0533 is the previous snapshot; the finding classes it did not run
are named under *Headline*.
