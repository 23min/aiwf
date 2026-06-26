---
id: M-0183
title: aiwf set-area verb to tag one entity to a declared area member
status: draft
parent: E-0044
tdd: required
acs:
    - id: AC-1
      title: set-area rewrites the target entity's area to the member in one commit
      status: open
      tdd_phase: red
    - id: AC-2
      title: the set-area commit carries trailers and aiwf history renders the retag
      status: open
      tdd_phase: red
    - id: AC-3
      title: set-area refuses unknown id, undeclared member, milestone target, and no-op
      status: open
      tdd_phase: red
    - id: AC-4
      title: set-area with the prior member reverses a previous set-area
      status: open
      tdd_phase: red
    - id: AC-5
      title: set-area ships tab-completion, --help, and skill-coverage
      status: open
      tdd_phase: red
---
## Goal

Add `aiwf set-area <id> <member>`: a verb that points a single entity at an existing declared area member, in one trailered commit. It is the guaranteed remediation for `areas.required` (M-0178) — when the knob flags an untagged entity, `set-area` is the one-command unblock — and a generally useful retag operation independent of the knob.

## Context

Today `--area` is creation-time only (`aiwf add --area`); no verb tags or retags an entity after creation. Hand-editing the `area:` frontmatter trips the `provenance-untrailered-entity-commit` audit, and `rename-area` (M-0177) renames a *member* across the whole tree — it cannot tag a single untagged entity. So an operator who enables `areas.required: true` on a tree with any untagged entity has no clean path to clear the resulting blocking finding. `set-area` closes that gap, mirroring the single-entity frontmatter-edit + trailer shape of `aiwf retitle`.

This is the inverse-blast-radius sibling of `rename-area`: `rename-area` changes the *vocabulary* (a member's name, carrying every referrer atomically); `set-area` changes one entity's *membership* against a fixed vocabulary.

## Acceptance criteria

Formalized at start-milestone as AC-1–AC-5 (frontmatter `acs[]`; full statements and pinning tests under the AC sections below). Summary:

- **AC-1** — `aiwf set-area <id> <member>` rewrites the entity's `area:` frontmatter to `<member>` in one commit, in **both** the untagged→tagged (remediation) and tagged→retagged (move-between-projects) directions; other entities untouched.
- **AC-2** — the commit carries `aiwf-verb: set-area` + `aiwf-entity:` + `aiwf-actor:`; `aiwf history <id>` renders the retag.
- **AC-3** — refuses an unknown id, an undeclared `<member>` (naming the declared set), a milestone or composite/AC-id target (area derives from the parent epic — message names the epic and the remediation command), an absent `areas` block, and a no-op — clear error, no write.
- **AC-4** — the **retag** direction reverses via the same verb (`set-area <id> <prior-member>`); the untagged→tagged direction is deliberately one-way (see Constraints).
- **AC-5** — tab-completion offers entity ids at `<id>` and declared members at `<member>`; `--help` and skill-coverage (allowlist) ship with it.

## Constraints

- Atomic: the single entity rewrite lands or nothing does — one commit, abort-before-commit on any validation failure.
- Single source of truth: `<member>` must already be declared in `aiwf.yaml: areas.members`; the verb never invents a member (that is `rename-area`'s and config's job). With no `areas` block declared, every member is undeclared, so the verb refuses.
- **"What undoes this?"** — for the **retag** direction (tagged→tagged), the same verb with the prior member. For the **tag** direction (untagged→tagged — the remediation case), the answer is the sanctioned *"you can't, and that's deliberate"*: undoing a tag means clearing `area` back to empty, which is out of scope (untagging has no use case and `required:true` forbids the empty state). A wrong *initial* tag is corrected forward by setting the right member, not by untagging.
- Provenance: a single target entity makes the verb authorized-AI-eligible — routed through the scope-gated finish with `VerbAct` and the entity as the target, so a scoped `ai/<id>` agent whose scope reaches the entity may run it (the inverse of `rename-area`'s human-only empty-target posture). Pinned by a positive regression test (scoped AI *allowed*). Tier-0 limit: until the `paths:` oracle lands (M-0179/M-0181), nothing verifies the agent tagged the *correct* area — only that it is *a* declared member; the wrong-area case is the Tier-2 mistag check's job.

## Out of scope

- Untagging (clearing `area` back to empty) — no use case yet; `required:true` forbids the empty state and a mis-tag is fixed by setting the correct member. Add later if a real need appears (YAGNI). This is what makes the tag direction one-way (Constraints).
- Renaming a member or mutating `aiwf.yaml` — that is `rename-area` (M-0177).
- Setting an area on a milestone or acceptance criterion — they derive from the parent epic; the verb refuses and points at the epic.

## Dependencies

- None. Independent Tier-0; sequenced before M-0178 (the `areas.required` knob depends on this verb as its remediation path).

## Design notes

- Mirror `aiwf retitle`'s single-entity frontmatter-edit + trailer-stamp shape (one write for the target entity, `aiwf-verb: set-area`).
- Reuse the declared-member validation the `area-unknown` check and `rename-area` already apply.
- **Completion needs a new composed `ValidArgsFunction`** — neither `CompleteAreaArg` (pos-0 area) nor `CompleteEntityIDArg` (pos-0 id) composes two positions. With `ExactArgs(2)` the completion-drift test requires a non-nil function, so dispatch on `len(args)`: position 0 → entity ids, position 1 → declared members. ~10 lines combining the two existing sources.

## References

- `aiwf rename-area` (M-0177) — the vocabulary-rename sibling; this is the membership-change counterpart.
- `aiwf retitle` — the precedent for a single-entity frontmatter edit + trailer stamp (and the composite-id refusal shape).
- `internal/check/area_unknown.go` — the declared-member validation reused for `<member>`.
- M-0178 — the `areas.required` knob whose remediation path this verb is.
- ADR-0006 — skills policy (allowlist / "--help suffices" case).

### AC-1 — set-area rewrites the target entity's area to the member in one commit

**Property.** `aiwf set-area <id> <member>` rewrites the `area:` frontmatter of the single entity `<id>` to `<member>` in ONE git commit, in both directions: untagged→tagged (the `areas.required` remediation case, `area: ""` → `member`) AND tagged→retagged (a move between projects, `area: old` → `member`). No other entity's frontmatter changes.

**Mechanical assertion.** `TestSetArea_AC1_RewritesUntaggedAndRetag` (`internal/cli/integration/setarea_test.go`) runs two cases in a fixture with ≥2 declared members: (a) an untagged root entity → asserts its frontmatter now carries `<member>`, exactly one new commit, every other entity byte-identical; (b) an already-tagged entity → asserts the retag, sibling entities untouched. Verb-level `TestSetArea_RewritesSingleEntity` pins the Plan shape (exactly one `OpWrite` for the target). Vacuity: a "rewrite the first untagged entity found" mutation reddens case (a); a "skip already-tagged entities" mutation reddens case (b).

### AC-2 — the set-area commit carries trailers and aiwf history renders the retag

**Property.** The single commit carries `aiwf-verb: set-area`, `aiwf-entity: <canonical id>`, and `aiwf-actor:` — and `aiwf history <id>` renders the set-area row. The `aiwf-verb` trailer suppresses the `provenance-untrailered-entity-commit` audit that a hand-edit would trip (the whole point of the verb).

**Mechanical assertion.** `TestSetArea_AC2_TrailersAndHistory` (integration) asserts the exact trailer set on `HEAD` and that `aiwf history <id>` shows the `set-area` row; a companion assertion confirms `aiwf check` reports no `provenance-untrailered-entity-commit` for the commit. `aiwf-verb: set-area` is auto-recognized by `trailer-verb-unknown` via the Cobra registration.

### AC-3 — set-area refuses unknown id, undeclared member, milestone target, and no-op

**Property.** The verb refuses, writing nothing (no frontmatter change, no commit), when: `<id>` resolves to nothing (unknown); `<member>` is not a declared member (or no `areas` block is declared) — error names the declared set; `<id>` is a milestone or composite/AC id (area derives from the parent epic) — error names the parent epic and gives the remediation command (e.g. `M-0183 derives its area from parent epic E-0044; run: aiwf set-area E-0044 <member>`); and the no-op (`<id>` already tagged `<member>`). All validation precedes any write.

**Mechanical assertion.** `TestSetArea_AC3_Refusals` (integration) asserts each refusal path leaves the tree byte-identical and the commit count unchanged. Verb-level `TestSetArea_ValidationRefusals` exhausts the cases and `TestSetArea_MilestoneTargetNamesEpic` pins the milestone/composite message shape (names the parent epic + the remediation command). Vacuity: inverting the declared-member check, or dropping the milestone guard, reddens the corresponding case.

### AC-4 — set-area with the prior member reverses a previous set-area

**Property.** For the **retag** direction, the verb is its own inverse: after `set-area E-0001 infra` (from `platform`), `set-area E-0001 platform` restores the prior tag and the rest of the tree. The untagged→tagged direction is deliberately NOT reversible by this verb (untagging is out of scope — Constraints).

**Mechanical assertion.** `TestSetArea_AC4_RetagReverses` (integration) retags forward then back and asserts the final frontmatter + tree equal the pre-retag state (byte-identical via deterministic `entity.Serialize`). The test scopes itself to the retag direction explicitly; a comment cites the one-way tag-direction decision so the absence of an untag-reversal test reads as intentional, not an omission.

### AC-5 — set-area ships tab-completion, --help, and skill-coverage

**Property.** `<id>` tab-completes to entity ids and `<member>` (position 1) to the declared `areas.members`; `aiwf set-area --help` ships; and the verb satisfies the skill-coverage chokepoint via a `skillCoverageAllowlist` entry (ADR-0006 "--help suffices"). The provenance posture (authorized-AI-eligible) is pinned by `TestSetArea_AuthorizedAIWithinScope` (a scoped `ai/<id>` actor whose scope reaches the target is *allowed* — the positive inverse of `TestRenameArea_AuthorizedAIRefused`).

**Mechanical assertion.** `TestSetArea_AC5_Discoverability` (integration) asserts the composed `ValidArgsFunction` returns entity ids at position 0 and declared members at position 1, and that the allowlist entry is present; the `skill_coverage` and completion-drift policy tests fail CI if the verb lacks coverage or a non-nil completion function. `TestNewCmd_SmokeShape` pins the command shape + `--help`. `TestSetArea_AuthorizedAIWithinScope` pins the AI-eligible posture.
