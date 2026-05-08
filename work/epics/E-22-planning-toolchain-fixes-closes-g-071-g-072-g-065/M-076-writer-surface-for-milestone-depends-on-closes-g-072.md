---
id: M-076
title: Writer surface for milestone depends_on (closes G-072)
status: in_progress
parent: E-22
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

# M-076 — Writer surface for milestone depends_on (closes G-072)

## Goal

Ship two writer surfaces for milestone `depends_on`: a `--depends-on` flag on `aiwf add milestone` for allocation-time edges, and a dedicated `aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ] [--clear]` verb for post-allocation edits. Both write the same frontmatter array atomically with proper aiwf trailers. Closes G-072 — the kernel asymmetry where `depends_on` had six read sites and zero writers.

## Context

Milestone `depends_on` is structurally supported in the kernel — universal struct field, milestone-schema-validated, cycle-detected, render-consumed — but has no writer verb. Hand-edit + `aiwf edit-body` collides with the body-only contract. E-20 planning paid this in full: M-073 and M-074 had their `depends_on` edges expressed in prose only. This milestone adds the two writer surfaces both shipping atomically: the flag covers the "I know the DAG when I allocate" case; the dedicated verb covers the "discovered later" case. Same underlying writer; the flag is sugar for the verb at creation time.

## Acceptance criteria

### AC-1 — --depends-on flag on aiwf add milestone

`aiwf add milestone --epic E-NN --tdd <policy> --title "..." --depends-on M-PPP[,M-QQQ]` accepts a comma-separated list of milestone ids and writes them to the new milestone's `depends_on:` frontmatter array atomically with the create commit. Absent flag produces no `depends_on` block (the YAML omitempty tag holds). The flag is milestone-only — passing it on any other kind (`aiwf add gap --depends-on M-001`) is a usage error. Wired through `verb.AddOptions.DependsOn` with cmd-side parsing via `splitCommaList`.

### AC-2 — aiwf milestone depends-on dedicated verb

`aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ]` is a top-level kind-prefixed verb that sets a milestone's `depends_on:` frontmatter array on an already-allocated milestone, in one commit with `aiwf-verb: milestone-depends-on` trailers. Replace-not-append semantics: a second invocation replaces the list rather than extending it. Forward-compatible with G-073's eventual cross-kind generalisation (`aiwf <kind> depends-on <id> --on <ids>`) — the verb-name segment "milestone" is the kind, and the verb signature extends to other kinds without renaming this one. Implemented in `internal/verb/milestone_depends_on.go`; cmd in `cmd/aiwf/milestone_cmd.go`.

### AC-3 — --clear flag empties the depends_on list

`aiwf milestone depends-on M-NNN --clear` empties the target's `depends_on:` array (the YAML omitempty tag means the block disappears entirely from frontmatter). `--clear` and `--on` are mutually exclusive — passing both is a usage error caught at the cmd boundary. Bare invocation (neither `--on` nor `--clear`) is also a usage error so the verb can't no-op silently. Replace-with-trimmed-list covers single-element removal; `--remove-depends-on` is deferred per the spec's Out-of-scope.

### AC-4 — Allocation-time referent validation refuses invalid ids

Both `--depends-on` (on `aiwf add milestone`) and `--on` (on `aiwf milestone depends-on`) refuse before writing if any id doesn't resolve to an existing milestone. Errors name the specific unresolvable id so a comma-separated typo is fast to fix. Three failure modes covered: id not found, id of wrong kind (e.g. `E-01`), and partial-list with a mix of valid/invalid (whole call refuses, no partial writes). Cycle detection stays in `aiwf check` — referent existence is the writer's job, DAG validity is the check's job.

### AC-5 — Closed-set completion for new flags and verb

`--depends-on` (on `aiwf add`) and `--on` (on `aiwf milestone depends-on`) both register `completeEntityIDFlag(KindMilestone)` so shell completion proposes milestone ids only. The positional milestone-id arg on `aiwf milestone depends-on` registers `completeEntityIDArg(KindMilestone, 0)` for the same. The drift-prevention chokepoint test in `cmd/aiwf/completion_drift_test.go` verifies wiring exists; `cmd/aiwf/milestone_depends_on_completion_test.go` adds named M-076-specific assertions.

### AC-6 — aiwf-add skill updated; aiwfx-plan-milestones update documented

`internal/skills/embedded/aiwf-add/SKILL.md` gains a "Milestone `depends_on`: declare DAG edges via verb (M-076)" section describing both writer surfaces, the replace-not-append semantic, the `--clear`/`--on` mutex, and the don't-hand-edit guidance. The skill's frontmatter description is broadened to mention dependency declaration so AI assistants asking "how do I declare a milestone's dependencies?" route into this skill. `aiwfx-plan-milestones` lives in the `ai-workflow-rituals` plugin (separate repo) — its update is documented in this milestone's Deferrals so the change is filed upstream when the plugin is next touched.

### AC-7 — Verb-level integration test drives the dispatcher

Per CLAUDE.md "Test the seam, not just the layer": `TestMilestoneDependsOn_DispatcherSeam_AddFlag` and `TestMilestoneDependsOn_DispatcherSeam_Verb` drive the cmd → verb → projection → apply → git path end-to-end via `run([]string{...})`, then assert both the on-disk milestone frontmatter shape AND that `aiwf history M-NNN` finds the trailered create/depends-on commit (proving the verb's trailer chain reached git). A regression where, say, the cmd flag is read but never copied into AddOptions, would slip past unit tests but trip these.

## Constraints

- **Comma-separated lists** for both `--depends-on M-PPP,M-QQQ` and `--on M-PPP,M-QQQ` per the epic's locked decision. Matches `--linked-adr` and `--relates-to` precedent.
- **Allocation-time referent validation** per the epic's locked decision: any id passed to `--depends-on` or `--on` must resolve to an existing milestone before the writer commits. Matches `--epic` / `--linked-adr` / `--discovered-in` precedent.
- **Narrow milestone-only scope.** Schema's `AllowedKinds: []Kind{KindMilestone}` for `depends_on` referents is unchanged. Cross-kind generalisation is G-073's territory; this milestone explicitly does not touch it.
- **Cycle detection stays in `aiwf check`** — referent-existence is the verb's job; DAG validity is the check's job. Different layers; don't duplicate.
- **Replace-not-append semantics.** `aiwf milestone depends-on M-NNN --on M-PPP,M-QQQ` REPLACES the milestone's `depends_on` list with `[M-PPP, M-QQQ]`. To add a single dependency to an existing list, the operator passes the full updated list. Append-style would be a separate flag (deferred).
- **`--clear` is exclusive with `--on`.** Either clear the list, or set it to specific ids; not both.
- **One commit per invocation** per kernel rule. Trailers: `aiwf-verb: milestone-depends-on` (or similar — name decided in implementation), `aiwf-entity: M-NNN`, `aiwf-actor: <derived>`.
- **TDD-required.** Each AC drives a red→green→refactor cycle. AC-7 is the seam test (per CLAUDE.md *Test the seam, not just the layer*).
- **Forward-compatibility with G-073 is non-negotiable.** Verb signature must be a clean subset of the future `aiwf <kind> depends-on <id> --on <id>` cross-kind verb. Specifically: the verb-name segment "milestone" is the *kind*, and the future generalisation extends to other kinds without renaming this verb. Document this in *Design notes* below.

## Design notes

- **Flag style:** `--depends-on` on `aiwf add milestone` matches the naming of the YAML field. Comma-separated parsing reuses the existing `parseCommaSeparatedIDs` helper (or the equivalent — naming decided in implementation).
- **Dedicated verb shape:** `aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ] [--clear]`. The verb-name segment "milestone" makes it a kind-prefixed verb. Reads naturally: *"milestone depends-on"* describes the operation (the milestone declares its dependencies). Forward-extends to `aiwf epic depends-on E-NN --on ADR-NNN,M-PPP` when G-073's cross-kind generalisation lands.
- **Validation flow:** parse args → resolve `--epic` (for add) / target milestone (for verb) → resolve each `--depends-on` or `--on` id against the loaded tree → if any unresolvable, refuse before writing. Errors include the specific unresolvable id.
- **Cycle detection:** the writer doesn't pre-check for cycles. A new cycle introduced by the writer surfaces at the next `aiwf check` (which is the pre-push hook on most repos). This matches the layered design: writers ensure referent existence; check validates DAG global properties.
- **Replace semantics rationale:** the simpler primitive. Append-style invites confusion when the user wants to "set the list to exactly these"; replace is unambiguous. Append can be added later as `--on-add M-PPP[,M-QQQ]` if friction earns it.
- **`--clear` mutual exclusion:** if both `--clear` and `--on` are passed, refuse with a usage error. Don't silently choose one.
- **Skill updates:**
  - `internal/skills/embedded/aiwf-add/SKILL.md` adds a section on `--depends-on` for milestone allocation. Frontmatter description gains phrasings like *"add a milestone with dependencies"*.
  - `aiwfx-plan-milestones` skill (in the `ai-workflow-rituals` plugin, NOT in this repo): step 6 currently says *"edit M-NNN's frontmatter"*. The change replaces that with the verb invocation. Since the plugin is external, M-076 captures the change as a documented update — actual file edit happens via PR to the plugin repo. AC-6 acceptance is met when (a) `aiwf-add` skill is updated in this repo AND (b) the plugin update is filed (issue/PR/note in work log) so it lands.

## Surfaces touched

- `cmd/aiwf/add_*.go` — extend `aiwf add milestone` with `--depends-on`.
- `cmd/aiwf/milestone_cmd.go` (new) — top-level `milestone` cobra command + `depends-on` subcommand.
- `cmd/aiwf/completion_drift_test.go` — completion wiring assertion.
- `internal/skills/embedded/aiwf-add/SKILL.md` — describe `--depends-on`.
- `aiwfx-plan-milestones` plugin skill — separate repo; documented change captured in work log.

## Out of scope

- Cross-kind `depends_on` (G-073's broader generalisation).
- Reverse query (`aiwf list --depended-on-by M-NNN` lives in E-20's `aiwf list` once that ships, or as a follow-up to G-073).
- Append-style flag (`--add-depends-on M-PPP`) — deferred until friction earns it.
- A `--remove-depends-on` flag for surgical removal — same; replace-with-trimmed-list covers the case.
- Status-aware FSM gating that consumes `depends_on` (G-073's territory).

## Dependencies

- No prior milestones in E-22.
- Existing `aiwf add milestone` and Cobra completion infrastructure.

## Coverage notes

- (filled at wrap)

## References

- E-22 epic spec (parent).
- G-072 — names the writer-verb gap (six read sites, zero writers).
- G-073 — broader cross-kind generalisation (out of scope; M-076's design forward-extends).
- E-20 — pays the prose-only-sequencing cost in milestone planning today.
- Existing patterns: `--linked-adr` parsing in `cmd/aiwf/add_*.go`; `--relates-to` in decision allocation; `--epic` validation flow in milestone allocation.

---

## Work log

(filled during implementation)

## Decisions made during implementation

- (none — all decisions pre-locked above)

## Validation

(pasted at wrap)

## Deferrals

- (none)

## Reviewer notes

- (filled at wrap)
