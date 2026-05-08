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
      status: open
      tdd_phase: red
    - id: AC-6
      title: aiwf-add skill updated; aiwfx-plan-milestones update documented
      status: open
      tdd_phase: red
    - id: AC-7
      title: Verb-level integration test drives the dispatcher
      status: open
      tdd_phase: red
---

# M-076 — Writer surface for milestone depends_on (closes G-072)

## Goal

Ship two writer surfaces for milestone `depends_on`: a `--depends-on` flag on `aiwf add milestone` for allocation-time edges, and a dedicated `aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ] [--clear]` verb for post-allocation edits. Both write the same frontmatter array atomically with proper aiwf trailers. Closes G-072 — the kernel asymmetry where `depends_on` had six read sites and zero writers.

## Context

Milestone `depends_on` is structurally supported in the kernel — universal struct field, milestone-schema-validated, cycle-detected, render-consumed — but has no writer verb. Hand-edit + `aiwf edit-body` collides with the body-only contract. E-20 planning paid this in full: M-073 and M-074 had their `depends_on` edges expressed in prose only. This milestone adds the two writer surfaces both shipping atomically: the flag covers the "I know the DAG when I allocate" case; the dedicated verb covers the "discovered later" case. Same underlying writer; the flag is sugar for the verb at creation time.

## Acceptance criteria

### AC-1 — --depends-on flag on aiwf add milestone

### AC-2 — aiwf milestone depends-on dedicated verb

### AC-3 — --clear flag empties the depends_on list

### AC-4 — Allocation-time referent validation refuses invalid ids

### AC-5 — Closed-set completion for new flags and verb

### AC-6 — aiwf-add skill updated; aiwfx-plan-milestones update documented

### AC-7 — Verb-level integration test drives the dispatcher

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
