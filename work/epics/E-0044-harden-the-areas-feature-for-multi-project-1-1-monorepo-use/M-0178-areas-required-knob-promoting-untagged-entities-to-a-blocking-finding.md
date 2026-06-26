---
id: M-0178
title: areas.required knob promoting untagged entities to a blocking finding
status: in_progress
parent: E-0044
depends_on:
    - M-0183
tdd: required
acs:
    - id: AC-1
      title: areas.required parses as a bool; required with zero members is rejected
      status: met
      tdd_phase: done
    - id: AC-2
      title: area-required errors on every untagged root entity across all five kinds
      status: met
      tdd_phase: done
    - id: AC-3
      title: required off or absent leaves area-required inert (pre-knob parity)
      status: met
      tdd_phase: done
    - id: AC-4
      title: a milestone never fires area-required; an untagged epic reports once
      status: open
      tdd_phase: green
    - id: AC-5
      title: aiwf add refuses an untagged create when areas.required is true
      status: open
      tdd_phase: red
    - id: AC-6
      title: area-required ships SKILL.md discoverability and a set-area hint
      status: open
      tdd_phase: red
---
## Goal

Add an `areas.required: true` knob that makes an untagged entity illegal — a blocking `aiwf check` finding (`area-required`, error severity) — for the 1:1 monorepo where every entity belongs to exactly one project. Inert (byte-for-byte E-0043 behavior) when absent or false. Orthogonal to `area-unknown` (which polices *present ⇒ declared*); this polices *present at all*.

## Context

E-0043 deliberately never flags an absent `area` ("absence is its own partition"). That is right for the carved-into-sections case but wrong for the 1:1 monorepo, where untagged is genuinely unassigned. This milestone adds the opt-in strictness without disturbing the default.

The knob requires a remediation path, or it strands the operator: with no clean verb to tag an existing entity, flipping `required:true` on a tree with untagged entities would force a hand-edit that itself trips `provenance-untrailered-entity-commit`. That hole is now closed — **M-0183 (`aiwf set-area … [--clear]`) ships first** (this milestone `depends_on` it), so the `area-required` hint points operators at a real one-command fix.

## Acceptance criteria

Formalized at start-milestone as AC-1–AC-6 (frontmatter `acs[]`; full statements + pinning tests under the AC sections below). Summary:

- **AC-1** — `areas.required` parses as a bool (default false); `validate()` rejects `required:true` with zero members.
- **AC-2** — error-severity `area-required` fires for every untagged, non-archived **root** entity across **all five self-tagging kinds** (epic, gap, ADR, decision, contract), naming it + the declared set; a tagged entity and an archived untagged entity raise nothing; wired end-to-end so `aiwf check` exits non-zero.
- **AC-3** — with `required` absent/false (or no `areas` block), `AreaRequired` returns nil — byte-for-byte pre-knob (E-0043) behavior.
- **AC-4** — no double-report: a milestone (derived area) never fires `area-required`; an untagged epic carrying untagged milestones yields exactly one finding (the epic). Distinct code from `area-unknown`.
- **AC-5** — `aiwf add` refuses an untagged create of a self-tagging root kind when `required:true`, with a message pointing at `--area`; unchanged when the knob is off (fail-fast at creation, not only at push).
- **AC-6** — `area-required` ships discoverable: a row in the `aiwf-check` SKILL.md (the `PolicyFindingCodesAreDiscoverable` haystack) + a `hint.go` remediation line pointing at `aiwf set-area`.

## Constraints

- **Kind scope: all five self-tagging root kinds** (epic, gap, ADR, decision, contract). The knob is opt-in; a consumer enabling it asserts "every entity belongs to a project," and a cross-cutting record gets a declared shared area. Milestones/ACs derive their area from the parent epic and are never directly flagged (AC-4). Decided after a design discussion weighing least-surprise, code simplicity, and reversibility — recorded as a Decision below; the bool can later evolve to a per-kind list via the same dual-form unmarshal `members` uses (forward-compatible) if a consumer needs a subset.
- **Default-off, zero migration:** an existing tree with no `required` key validates and renders exactly as today. The field cannot exist in a pre-M-0178 config, so the `validate()` rejection of `required:true`+zero-members can only bite a new config.
- **Default views never hide.** `required` makes untagged a *check* finding; it does not make grouping gating. An unscoped `aiwf status` / roadmap still shows every entity.
- **Verbs mutate, checks gate.** The knob escalates a condition to a blocking finding; the fix is `aiwf set-area <id> <member>` (M-0183), a discoverable trailered loop — never a hand-edit.
- **Known limitation (severity asymmetry):** under `required:true`, an *untagged* entity is an error, but a *typo'd* (undeclared) area stays an `area-unknown` *warning* — the epic scopes escalating `area-unknown` out of scope. No-area is blocking; wrong-area is advisory. Recorded so it reads as a decision, not an oversight.

## Out of scope

- Path verification (Tier 1) — `required` only asserts area *presence*, not correctness against `paths:`.
- Reusing or escalating `area-unknown` — that finding stays *present ⇒ declared*; `area-required` is a separate rule and a separate code.

## Dependencies

- **M-0183 (`aiwf set-area`)** — the remediation verb the `area-required` hint points at. Must ship first (frontmatter `depends_on: [M-0183]`); it is `done`.

## Design notes

- **Check:** new `internal/check/area_required.go` — `CodeAreaRequired = "area-required"`, error severity. Mirror `area_unknown.go`'s guards (inert when `len(declared)==0`; skip archived) with the load-bearing twist: **skip `entity.KindMilestone`** (its `e.Area` is blanked at load, `tree.go:240`, and derived from the parent epic), so an untagged epic fires exactly once. No finding at all when `required` is false — the knob is the gate, not a warning→error bump.
- **Config:** add `Required bool` (`yaml:"required,omitempty"`) to `Areas` + the raw struct in `UnmarshalYAML`; `validate()` rejects `required:true`+zero-members, mirroring the existing `default`-needs-members rule.
- **CLI seam:** `internal/cli/check/check.go` already loads `cfg` (~line 156); read `cfg.Areas.Required` alongside `cfg.Areas.Members` and compose `check.AreaRequired(tr, areaMembers, areaRequired)` next to the `AreaUnknown` call — no new `cliutil` helper.
- **Add-time refusal:** `aiwf add` (the verb) refuses an untagged create of a self-tagging root kind under `required:true`. Milestones (derived) and a gap whose `--discovered-in` derives an area are unaffected (they aren't untagged). The add verb already reads configured areas for `--area` validation; it gains the `required` flag from the same source.
- **Coverage:** keep the defensive `len(declared)==0` guard in `AreaRequired` but drive `AreaRequired(tr, nil, true)` directly in AC-3's test so the branch is covered (the `validate()` rejection makes it unreachable through `config.Load`).

## References

- `internal/check/area_unknown.go` — the sibling *present ⇒ declared* finding this sits beside (not modified); also the model for the guards and the milestone-skip rationale.
- `internal/config/config.go` — the `Areas` schema + `validate()` the knob extends.
- `internal/cli/check/check.go` (~155–182) — the CLI seam composing config-dependent tree rules.
- `internal/tree/tree.go:240` / `ResolvedArea` — why milestones are skipped (blanked/derived area).
- M-0183 (`aiwf set-area`) — the remediation verb the hint points at.
- `internal/policies/skill_coverage.go` / `discoverability.go` — the discoverability chokepoint AC-6 satisfies.

### AC-1 — areas.required parses as a bool; required with zero members is rejected

**Property.** `aiwf.yaml: areas.required` decodes as a bool, default false when absent. `config.Load` rejects `required: true` with an empty `members` set (an unsatisfiable "every entity must be a member of the empty set"), mirroring the existing `default`-needs-members rejection.

**Mechanical assertion.** `TestConfig_AreasRequired_ParsesAndValidates` (`internal/config/config_test.go`) — table cases: absent→false; `required: true` with members→ok; `required: false` with no members→ok; `required: true` with zero members→error naming the field. Vacuity: dropping the new validate guard reddens the last case.

### AC-2 — area-required errors on every untagged root entity across all five kinds

**Property.** With `required:true` and a declared block, `AreaRequired` emits an **error**-severity `area-required` finding for every non-archived entity of a self-tagging root kind (epic, gap, ADR, decision, contract) whose `area` is empty — message names the entity + the declared set. A tagged entity and an archived untagged entity emit nothing. `aiwf check` surfaces the finding and exits non-zero.

**Mechanical assertion.** `TestAreaRequired_FiresForAllRootKinds` (`internal/check/area_required_test.go`) — a fixture tree with one untagged entity of each of the five kinds (+ a tagged one + an archived untagged one) asserts exactly five findings, each error-severity, each naming its entity, none for the tagged/archived. Integration `TestCheck_AreaRequiredExitsNonZero` (`internal/cli/integration/`) drives `aiwf check` on a `required:true` fixture and asserts a non-zero exit + the code. Vacuity: a "skip ADR/decision/contract" mutation reddens the per-kind count.

### AC-3 — required off or absent leaves area-required inert (pre-knob parity)

**Property.** With `required` false, absent, or no `areas` block, `AreaRequired` returns nil — byte-for-byte the pre-knob (E-0043) behavior; no untagged entity is flagged.

**Mechanical assertion.** `TestAreaRequired_InertWhenOff` (`internal/check/area_required_test.go`) — asserts nil findings for: `required:false` with untagged entities; `required` absent; and a direct `AreaRequired(tr, nil, true)` call (no declared members) covering the defensive empty-declared guard. Vacuity: making the rule fire when `required` is false reddens the first case.

### AC-4 — a milestone never fires area-required; an untagged epic reports once

**Property.** A milestone (area derived from its parent epic, blanked at load) never produces an `area-required` finding. An untagged epic carrying untagged milestones yields exactly one finding — the epic. `area-required` is a distinct code from `area-unknown`.

**Mechanical assertion.** `TestAreaRequired_NoDoubleReport` (`internal/check/area_required_test.go`) — a fixture with an untagged epic + two untagged milestones under it asserts exactly one `area-required` finding (the epic), zero for the milestones, and that `CodeAreaRequired != CodeAreaUnknown`. Vacuity: removing the `KindMilestone` skip reddens the count (it would jump to three).

### AC-5 — aiwf add refuses an untagged create when areas.required is true

**Property.** Under `required:true`, `aiwf add <epic|gap|adr|decision|contract>` with no `--area` (and no derivation) refuses with a clear message pointing at `--area` — failing fast at creation rather than at the next push. A milestone (derived) and a gap whose `--discovered-in` supplies an area are unaffected. With `required` off, `aiwf add` is unchanged.

**Mechanical assertion.** `TestAdd_RefusesUntaggedWhenRequired` (`internal/cli/integration/`) — under `required:true`: an untagged `add epic` refuses (no entity written, message names `--area`); `add epic --area <member>` succeeds; a milestone add succeeds untagged. Under `required:false`: an untagged `add epic` succeeds (parity). Vacuity: dropping the refusal lets the untagged-add case write an entity, reddening the test.

### AC-6 — area-required ships SKILL.md discoverability and a set-area hint

**Property.** `area-required` appears verbatim in the `aiwf-check` SKILL.md (the `PolicyFindingCodesAreDiscoverable` haystack) and carries a `hint.go` remediation line pointing operators at `aiwf set-area <id> <member>`.

**Mechanical assertion.** `TestPolicy_FindingCodesAreDiscoverable` (existing, `internal/policies/`) fails CI if `area-required` lacks a SKILL.md row — adding the row is the pin. `TestHint_AreaRequired` (`internal/check/hint_test.go`, or the existing hint-presence policy) asserts the `area-required` hint exists and mentions `set-area`. Vacuity: the discoverability policy reddens if the SKILL.md row is removed.
