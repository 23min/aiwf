---
id: M-0076
title: Writer surface for milestone depends_on (closes G-0072)
status: done
parent: E-0022
tdd: required
acs:
    - id: AC-1
      title: --depends-on flag on aiwf add milestone
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf milestone depends-on dedicated verb
      status: met
      tdd_phase: done
    - id: AC-3
      title: --clear flag empties the depends_on list
      status: met
      tdd_phase: done
    - id: AC-4
      title: Allocation-time referent validation refuses invalid ids
      status: met
      tdd_phase: done
    - id: AC-5
      title: Closed-set completion for new flags and verb
      status: met
      tdd_phase: done
    - id: AC-6
      title: aiwf-add skill updated; aiwfx-plan-milestones update documented
      status: met
      tdd_phase: done
    - id: AC-7
      title: Verb-level integration test drives the dispatcher
      status: met
      tdd_phase: done
---

# M-0076 — Writer surface for milestone depends_on (closes G-0072)

## Goal

Ship two writer surfaces for milestone `depends_on`: a `--depends-on` flag on `aiwf add milestone` for allocation-time edges, and a dedicated `aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ] [--clear]` verb for post-allocation edits. Both write the same frontmatter array atomically with proper aiwf trailers. Closes G-0072 — the kernel asymmetry where `depends_on` had six read sites and zero writers.

## Context

Milestone `depends_on` is structurally supported in the kernel — universal struct field, milestone-schema-validated, cycle-detected, render-consumed — but has no writer verb. Hand-edit + `aiwf edit-body` collides with the body-only contract. E-0020 planning paid this in full: M-0073 and M-0074 had their `depends_on` edges expressed in prose only. This milestone adds the two writer surfaces both shipping atomically: the flag covers the "I know the DAG when I allocate" case; the dedicated verb covers the "discovered later" case. Same underlying writer; the flag is sugar for the verb at creation time.

## Acceptance criteria

### AC-1 — --depends-on flag on aiwf add milestone

`aiwf add milestone --epic E-NN --tdd <policy> --title "..." --depends-on M-PPP[,M-QQQ]` accepts a comma-separated list of milestone ids and writes them to the new milestone's `depends_on:` frontmatter array atomically with the create commit. Absent flag produces no `depends_on` block (the YAML omitempty tag holds). The flag is milestone-only — passing it on any other kind (`aiwf add gap --depends-on M-001`) is a usage error. Wired through `verb.AddOptions.DependsOn` with cmd-side parsing via `splitCommaList`.

### AC-2 — aiwf milestone depends-on dedicated verb

`aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ]` is a top-level kind-prefixed verb that sets a milestone's `depends_on:` frontmatter array on an already-allocated milestone, in one commit with `aiwf-verb: milestone-depends-on` trailers. Replace-not-append semantics: a second invocation replaces the list rather than extending it. Forward-compatible with G-0073's eventual cross-kind generalisation (`aiwf <kind> depends-on <id> --on <ids>`) — the verb-name segment "milestone" is the kind, and the verb signature extends to other kinds without renaming this one. Implemented in `internal/verb/milestone_depends_on.go`; cmd in `cmd/aiwf/milestone_cmd.go`.

### AC-3 — --clear flag empties the depends_on list

`aiwf milestone depends-on M-NNN --clear` empties the target's `depends_on:` array (the YAML omitempty tag means the block disappears entirely from frontmatter). `--clear` and `--on` are mutually exclusive — passing both is a usage error caught at the cmd boundary. Bare invocation (neither `--on` nor `--clear`) is also a usage error so the verb can't no-op silently. Replace-with-trimmed-list covers single-element removal; `--remove-depends-on` is deferred per the spec's Out-of-scope.

### AC-4 — Allocation-time referent validation refuses invalid ids

Both `--depends-on` (on `aiwf add milestone`) and `--on` (on `aiwf milestone depends-on`) refuse before writing if any id doesn't resolve to an existing milestone. Errors name the specific unresolvable id so a comma-separated typo is fast to fix. Three failure modes covered: id not found, id of wrong kind (e.g. `E-01`), and partial-list with a mix of valid/invalid (whole call refuses, no partial writes). Cycle detection stays in `aiwf check` — referent existence is the writer's job, DAG validity is the check's job.

### AC-5 — Closed-set completion for new flags and verb

`--depends-on` (on `aiwf add`) and `--on` (on `aiwf milestone depends-on`) both register `completeEntityIDFlag(KindMilestone)` so shell completion proposes milestone ids only. The positional milestone-id arg on `aiwf milestone depends-on` registers `completeEntityIDArg(KindMilestone, 0)` for the same. The drift-prevention chokepoint test in `cmd/aiwf/completion_drift_test.go` verifies wiring exists; `cmd/aiwf/milestone_depends_on_completion_test.go` adds named M-0076-specific assertions.

### AC-6 — aiwf-add skill updated; aiwfx-plan-milestones update documented

`internal/skills/embedded/aiwf-add/SKILL.md` gains a "Milestone `depends_on`: declare DAG edges via verb (M-0076)" section describing both writer surfaces, the replace-not-append semantic, the `--clear`/`--on` mutex, and the don't-hand-edit guidance. The skill's frontmatter description is broadened to mention dependency declaration so AI assistants asking "how do I declare a milestone's dependencies?" route into this skill. `aiwfx-plan-milestones` lives in the `ai-workflow-rituals` plugin (separate repo) — its update is documented in this milestone's Deferrals so the change is filed upstream when the plugin is next touched.

### AC-7 — Verb-level integration test drives the dispatcher

Per CLAUDE.md "Test the seam, not just the layer": `TestMilestoneDependsOn_DispatcherSeam_AddFlag` and `TestMilestoneDependsOn_DispatcherSeam_Verb` drive the cmd → verb → projection → apply → git path end-to-end via `run([]string{...})`, then assert both the on-disk milestone frontmatter shape AND that `aiwf history M-NNN` finds the trailered create/depends-on commit (proving the verb's trailer chain reached git). A regression where, say, the cmd flag is read but never copied into AddOptions, would slip past unit tests but trip these.

## Constraints

- **Comma-separated lists** for both `--depends-on M-PPP,M-QQQ` and `--on M-PPP,M-QQQ` per the epic's locked decision. Matches `--linked-adr` and `--relates-to` precedent.
- **Allocation-time referent validation** per the epic's locked decision: any id passed to `--depends-on` or `--on` must resolve to an existing milestone before the writer commits. Matches `--epic` / `--linked-adr` / `--discovered-in` precedent.
- **Narrow milestone-only scope.** Schema's `AllowedKinds: []Kind{KindMilestone}` for `depends_on` referents is unchanged. Cross-kind generalisation is G-0073's territory; this milestone explicitly does not touch it.
- **Cycle detection stays in `aiwf check`** — referent-existence is the verb's job; DAG validity is the check's job. Different layers; don't duplicate.
- **Replace-not-append semantics.** `aiwf milestone depends-on M-NNN --on M-PPP,M-QQQ` REPLACES the milestone's `depends_on` list with `[M-PPP, M-QQQ]`. To add a single dependency to an existing list, the operator passes the full updated list. Append-style would be a separate flag (deferred).
- **`--clear` is exclusive with `--on`.** Either clear the list, or set it to specific ids; not both.
- **One commit per invocation** per kernel rule. Trailers: `aiwf-verb: milestone-depends-on` (or similar — name decided in implementation), `aiwf-entity: M-NNN`, `aiwf-actor: <derived>`.
- **TDD-required.** Each AC drives a red→green→refactor cycle. AC-7 is the seam test (per CLAUDE.md *Test the seam, not just the layer*).
- **Forward-compatibility with G-0073 is non-negotiable.** Verb signature must be a clean subset of the future `aiwf <kind> depends-on <id> --on <id>` cross-kind verb. Specifically: the verb-name segment "milestone" is the *kind*, and the future generalisation extends to other kinds without renaming this verb. Document this in *Design notes* below.

## Design notes

- **Flag style:** `--depends-on` on `aiwf add milestone` matches the naming of the YAML field. Comma-separated parsing reuses the existing `parseCommaSeparatedIDs` helper (or the equivalent — naming decided in implementation).
- **Dedicated verb shape:** `aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ] [--clear]`. The verb-name segment "milestone" makes it a kind-prefixed verb. Reads naturally: *"milestone depends-on"* describes the operation (the milestone declares its dependencies). Forward-extends to `aiwf epic depends-on E-NN --on ADR-NNN,M-PPP` when G-0073's cross-kind generalisation lands.
- **Validation flow:** parse args → resolve `--epic` (for add) / target milestone (for verb) → resolve each `--depends-on` or `--on` id against the loaded tree → if any unresolvable, refuse before writing. Errors include the specific unresolvable id.
- **Cycle detection:** the writer doesn't pre-check for cycles. A new cycle introduced by the writer surfaces at the next `aiwf check` (which is the pre-push hook on most repos). This matches the layered design: writers ensure referent existence; check validates DAG global properties.
- **Replace semantics rationale:** the simpler primitive. Append-style invites confusion when the user wants to "set the list to exactly these"; replace is unambiguous. Append can be added later as `--on-add M-PPP[,M-QQQ]` if friction earns it.
- **`--clear` mutual exclusion:** if both `--clear` and `--on` are passed, refuse with a usage error. Don't silently choose one.
- **Skill updates:**
  - `internal/skills/embedded/aiwf-add/SKILL.md` adds a section on `--depends-on` for milestone allocation. Frontmatter description gains phrasings like *"add a milestone with dependencies"*.
  - `aiwfx-plan-milestones` skill (in the `ai-workflow-rituals` plugin, NOT in this repo): step 6 currently says *"edit M-NNN's frontmatter"*. The change replaces that with the verb invocation. Since the plugin is external, M-0076 captures the change as a documented update — actual file edit happens via PR to the plugin repo. AC-6 acceptance is met when (a) `aiwf-add` skill is updated in this repo AND (b) the plugin update is filed (issue/PR/note in work log) so it lands.

## Surfaces touched

- `cmd/aiwf/add_*.go` — extend `aiwf add milestone` with `--depends-on`.
- `cmd/aiwf/milestone_cmd.go` (new) — top-level `milestone` cobra command + `depends-on` subcommand.
- `cmd/aiwf/completion_drift_test.go` — completion wiring assertion.
- `internal/skills/embedded/aiwf-add/SKILL.md` — describe `--depends-on`.
- `aiwfx-plan-milestones` plugin skill — separate repo; documented change captured in work log.

## Out of scope

- Cross-kind `depends_on` (G-0073's broader generalisation).
- Reverse query (`aiwf list --depended-on-by M-NNN` lives in E-0020's `aiwf list` once that ships, or as a follow-up to G-0073).
- Append-style flag (`--add-depends-on M-PPP`) — deferred until friction earns it.
- A `--remove-depends-on` flag for surgical removal — same; replace-with-trimmed-list covers the case.
- Status-aware FSM gating that consumes `depends_on` (G-0073's territory).

## Dependencies

- No prior milestones in E-0022.
- Existing `aiwf add milestone` and Cobra completion infrastructure.

## Coverage notes

- `internal/verb/add.go:validateDependsOnReferents` — 100%. Branch audit: kind != milestone (returns nil); empty list (returns nil); unresolvable id (returns error); wrong-kind referent (returns error); all-valid (returns nil). Each branch has a dedicated test in `cmd/aiwf/add_milestone_depends_on_test.go`.
- `internal/verb/add.go:applyAddOpts` (milestone arm) — 100%. The new `len(opts.DependsOn) > 0` guard is exercised by both the "with --depends-on" tests and the "without" baseline.
- `internal/verb/milestone_depends_on.go:MilestoneDependsOn` — every conditional branch covered: composite-id rejection (`TestMilestoneDependsOn_CompositeIDRejected`), clear+on mutex, neither set, unknown target, wrong-kind target, self-loop (`TestMilestoneDependsOn_SelfDependencyRejected`), unknown referent, wrong-kind referent, clear-true arm, set-arm, replace semantics, multiple deps. Statement coverage 80% (the 20% remaining is defensive IO error paths in `readBody` / `entity.Serialize` that can't fire on a well-formed tree without filesystem-level corruption).
- `cmd/aiwf/milestone_cmd.go:newMilestoneCmd` and `newMilestoneDependsOnCmd` — 100% (constructed by every test that walks the cmd tree, including the policy/drift tests).
- `cmd/aiwf/milestone_cmd.go:runMilestoneDependsOnCmd` — 74%. The uncovered region is the lock/loadTree error arms which require filesystem failure or concurrent-locked state to exercise; the project convention treats those as defensive (matching the parallel arms in `runEditBodyCmd`, `runRenameCmd`, etc.).

## References

- E-0022 epic spec (parent).
- G-0072 — names the writer-verb gap (six read sites, zero writers).
- G-0073 — broader cross-kind generalisation (out of scope; M-0076's design forward-extends).
- E-0020 — pays the prose-only-sequencing cost in milestone planning today.
- Existing patterns: `--linked-adr` parsing in `cmd/aiwf/add_*.go`; `--relates-to` in decision allocation; `--epic` validation flow in milestone allocation.

---

## Work log

### AC-1 — --depends-on flag on aiwf add milestone

Added `DependsOn []string` to `internal/verb/AddOptions`, validated milestone-only via `validateAddOptsForKind`, applied to entity in `applyAddOpts`. Cmd-side parsing via `splitCommaList` matches `--linked-adr` / `--relates-to` precedent. Wired completion via `completeEntityIDFlag(KindMilestone)`. `addCreationRefs` extended so the I2.5 allow-rule sees the new outbound refs. Tests: 7 cases in `cmd/aiwf/add_milestone_depends_on_test.go` covering single id, multi-id, absence, non-milestone-kind rejection, unknown referent, wrong-kind referent, partial-list.

### AC-2 — aiwf milestone depends-on dedicated verb

New top-level verb `aiwf milestone` with `depends-on` subcommand at `cmd/aiwf/milestone_cmd.go`; verb logic at `internal/verb/milestone_depends_on.go`. One commit per invocation with `aiwf-verb: milestone-depends-on` trailers. Replace-not-append semantics. Forward-compatible with G-0073 (the `milestone` segment is the kind; cross-kind generalisation extends without renaming this verb). Tests: 10 cases in `cmd/aiwf/milestone_depends_on_test.go` covering set-single/multi/replace, unknown/wrong-kind target, unknown/wrong-kind referent, plus the explicit branch-coverage tests for composite-id and self-loop rejection.

### AC-3 — --clear flag empties the depends_on list

`--clear` boolean flag on `aiwf milestone depends-on` empties `depends_on:` (the YAML omitempty tag means the block disappears). Mutex with `--on` enforced at the cmd boundary; bare invocation (neither flag) is also a usage error so the verb can't no-op silently. Lint pass forced renaming the local variable from `clear` to `clearList` per `gocritic`'s `builtinShadow` (Go 1.21+ has `clear()`). Tests in the same file as AC-2 (`TestMilestoneDependsOn_Clear`, `_ClearAndOnMutex`, `_NoFlagIsUsage`).

### AC-4 — Allocation-time referent validation refuses invalid ids

`validateDependsOnReferents` runs before `id := entity.AllocateID(...)` so a refused call leaves no partial trace. The verb-side equivalent is inline at the top of `MilestoneDependsOn`. Three failure modes covered: id not found, id of wrong kind, partial-valid list (the whole call refuses; no partial writes). Self-loop guard (`--on M-NNN` where M-NNN is the target) is a nice-to-have caught at the same layer.

### AC-5 — Closed-set completion for new flags and verb

`--depends-on` (on `aiwf add`) and `--on` (on `aiwf milestone depends-on`) both register `completeEntityIDFlag(KindMilestone)`. The positional milestone-id arg uses `completeEntityIDArg(KindMilestone, 0)`. Generic `TestPolicy_FlagsHaveCompletion` and `TestPolicy_PositionalsHaveCompletion` already cover absence-of-wiring; explicit M-0076-named assertions in `cmd/aiwf/milestone_depends_on_completion_test.go` pin the specific surfaces so a future refactor that drops the wiring fails with a named message.

### AC-6 — aiwf-add skill updated; aiwfx-plan-milestones update documented

`internal/skills/embedded/aiwf-add/SKILL.md`: frontmatter description broadened to mention dependency declaration; milestone row in the kinds table extended with `--depends-on`; new "Milestone `depends_on`: declare DAG edges via verb (M-0076)" section describing both writer surfaces, replace-not-append, the `--clear`/`--on` mutex, and the don't-hand-edit guidance. The `aiwfx-plan-milestones` plugin skill lives in `ai-workflow-rituals` (separate repo); its update is captured in Deferrals as G-0079 so the change is filed upstream.

### AC-7 — Verb-level integration test drives the dispatcher

`TestMilestoneDependsOn_DispatcherSeam_AddFlag` and `_DispatcherSeam_Verb` drive `run([]string{...})` end-to-end through cmd → verb → projection → apply → git, then assert (a) on-disk frontmatter shape AND (b) `aiwf history M-NNN` finds the trailered commit (proving the trailer chain reached git). A regression where the cmd flag is read but never copied into AddOptions slips past unit tests but trips here, per CLAUDE.md's "Test the seam, not just the layer" rule.

## Decisions made during implementation

- (none — all decisions pre-locked above)

## Validation

- `go test -race ./...` — green. All packages pass.
- `go build -o /tmp/aiwf ./cmd/aiwf` — green.
- `golangci-lint run ./cmd/aiwf/ ./internal/verb/ ./internal/skills/` — 0 issues (after the `clear` → `clearList` rename to satisfy `gocritic`'s `builtinShadow`).
- `aiwf check` — 0 errors, 2 unrelated warnings (`provenance-untrailered-scope-undefined` because the milestone branch has no upstream yet; `unexpected-tree-file` on `work/epics/critical-path.md` is E-0021's scope).
- `aiwf show M-076` — every AC at `met` + `tdd_phase: done`; status `in_progress` (promoted to `done` by this wrap).
- Coverage: see Coverage notes — branch audit clean across both new functions and the modified `entityBodyEmpty` consumers.
- Real-tree dogfood: `aiwf add milestone --depends-on …` and `aiwf milestone depends-on … --on …` both round-trip through `aiwf history` with the right trailers. The seam tests pin this end-to-end.

## Deferrals

- [G-0079](../../gaps/G-079-aiwfx-plan-milestones-plugin-skill-needs-depends-on-documentation-m-076-added-the-verb-but-the-plugin-lives-in-ai-workflow-rituals-upstream.md) — `aiwfx-plan-milestones` skill update lives in the `ai-workflow-rituals` plugin (separate repo). Per the spec's AC-6 acceptance bar, this milestone closes when (a) `aiwf-add` is updated in this repo (done) AND (b) the plugin update is filed (G-0079 captures the upstream PR).

## Reviewer notes

- **Replace-not-append is the simpler primitive.** A second invocation of `aiwf milestone depends-on M-NNN --on M-XXX` replaces the list — it does not extend. To add a single dep to an existing list, the operator passes the full updated list. Append-style (`--add-depends-on M-XXX`) is deferred until friction earns it; the spec's Out-of-scope and the verb's body comment both pin this. Reviewers should resist a "make it append by default" reflex — that closes off the unambiguous-replace path which is the easier primitive to reason about.
- **Verb name segment "milestone" is the *kind*, not stutter.** `aiwf milestone depends-on M-NNN --on M-PPP` reads as "milestone-X declares dependency on milestone-Y" with the leading `milestone` indicating which kind owns the verb. When G-0073's cross-kind generalisation lands, `aiwf <kind> depends-on <id> --on <ids>` extends the same shape to other kinds without renaming this verb. Reviewers worried about "redundant 'milestone' in `aiwf milestone depends-on M-NNN`" — that's the kind segment, and it stays load-bearing under the future generalisation.
- **Forward-compatibility with G-0073 is non-negotiable per the epic spec.** The verb signature (`<kind>` segment, `--on <ids>` list flag) is a clean subset of the cross-kind future. The narrow milestone-only schema (`AllowedKinds: []Kind{KindMilestone}`) is intentionally unchanged here; G-0073 expands it when the cross-kind friction surfaces.
- **`clear` → `clearList` rename was a lint fix, not a design change.** Go 1.21+ has a builtin `clear()`. `gocritic`'s `builtinShadow` rule flags any local named `clear`. Renaming was the cheapest fix; the verb and flag are still spelled `--clear` on the user-facing surface.
- **Cycle detection deliberately stays in `aiwf check`.** The writer's job is referent existence; the check's job is DAG validity. Different concerns, different chokepoints. A cycle introduced by this verb surfaces at the next pre-push hook rather than being pre-checked at write time. This matches the layered design and keeps the writer cheap; reviewers should not push for pre-write cycle detection without a concrete friction case.
- **`addCreationRefs` extended** so the I2.5 allow-rule's reachability check sees the new outbound `depends_on` refs at allocation time. Without this, a non-human actor scoped to (say) M-0001 could add a milestone with `--depends-on M-001` even if the scope didn't reach the new entity's parent. Standard pattern; no surprises.
- **No ADR/D-NNN produced.** Every locked design choice (skill placement, comma-separated lists, allocation-time validation, replace-not-append, `--clear` mutex) was pre-locked in the epic spec. Nothing surfaced mid-implementation that warranted a separate decision artifact.
- **`.gitignore` change rides along** because the kernel's marker-managed-skills convention (`.claude/skills/aiwf-*` is gitignored per CLAUDE.md) was missing from the consumer repo's gitignore. Two-line addition; harmless, on-topic for keeping the consumer-repo state clean. Not a separate patch.
