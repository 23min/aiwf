# Acceptance criteria and TDD plan

**Status:** proposal · **Audience:** PoC iteration I2 (continuation of [`poc-plan.md`](../archive/poc-plan-pre-migration.md) sessions 1–5 and [`contracts-plan.md`](contracts-plan.md) I1).

This document plans first-class acceptance criteria and opt-in TDD enforcement in aiwf, from first principles.

ACs in v1 (`../ai-workflow`) lived in a separate `tracking-doc.md` per milestone, with checkboxes and a Work Log. The convention worked but was not load-bearing — nothing prevented a milestone from being marked done while ACs sat unchecked, because the tracking doc was outside the kernel's view. v3 adopts the convention but inverts the layering: the AC list moves into the milestone doc itself (frontmatter + matching body sections), the kernel validates it, and the legacy tracking-doc template dies. TDD recording follows the same shape: opt-in per milestone, per-AC phase tracked in frontmatter, audit rule that "AC `met` requires phase `done`."

This factoring is the load-bearing call: it lets aiwf govern AC progress and TDD discipline mechanically while keeping ACs sub-scoped to milestones (no seventh entity kind, no global AC allocator).

For the design context that justifies this shape, see [`design-decisions.md`](../design/design-decisions.md) §"Acceptance criteria and TDD (added in I2)".

---

## 1. The model

ACs are structured sub-elements of a milestone, addressed by composite id `M-NNN/AC-N`. Two locations hold their state:

| Location | What it carries |
|---|---|
| Milestone frontmatter | `acs[]` (structured truth: id, title, status, tdd_phase). The kernel reads and validates this. |
| Milestone body | `### AC-N — <title>` heading per AC, with prose detail (description, examples, edge cases, references). The kernel cross-checks heading-vs-frontmatter. |

The body of the milestone doc also carries the merged-in tracking sections — Work Log, Decisions made during implementation, Validation, Deferrals, Reviewer notes — that v1 kept in a separate file. Same kernel-blind prose status as the spec sections; they're convention, not data.

Example:

```yaml
---
id: M-007
title: Engine warning surface
status: in_progress
parent: E-03
tdd: required
acs:
  - id: AC-1
    title: "Engine emits warning on bad input"
    status: open
    tdd_phase: red
  - id: AC-2
    title: "Pack receives canonical OpResult"
    status: met
    tdd_phase: done
---

## Goal
…

## Acceptance criteria

### AC-1 — Engine emits warning on bad input

When X occurs, the engine emits a `Warning` with `code = "bad-input"` and the offending value. (Prose, examples, refs to ADRs.)

### AC-2 — Pack receives canonical OpResult

…

## Work Log

### AC-2 — met · commit `abc1234` · tests 12/12

Brief prose if needed.

## Decisions made during implementation

- (none)

## Validation

- `go test ./...` — 234 pass, 0 fail.
```

---

## 2. ID convention

| Element | Format | Allocator |
|---|---|---|
| Milestone | `M-NNN` | global, `max+1` per current branch |
| AC | `AC-N` (sub-scoped under milestone) | per-milestone, `max+1` over the full `acs[]` (including cancelled entries) |
| Composite id | `M-NNN/AC-N` | derived |

The composite id is the only form used in cross-references, trailers, and `aiwf history`. The bare `AC-N` is meaningful only relative to its milestone.

**Sequencing under cancellation.** AC ids are position-based and stable: `acs[i].id == fmt.Sprintf("AC-%d", i+1)` for every index. Cancelled ACs stay in `acs[]` at their original position (status flip, not deletion); the allocator picks `max+1` over the full list including cancelled entries. After cancelling AC-0003 and adding a new one, the list reads `[AC-1, AC-2, AC-3 (cancelled), AC-4]`. References to a cancelled AC's composite id always resolve. This mirrors the milestone/epic id-stability model.

The id grammar in `internal/entity/entity.go`'s `idLeadingPattern` extends from `^(?:ADR|[EMGDC])-\d+` with a sibling pattern that recognizes `<entity-id>/AC-\d+` composites. The bare-id form remains the default; the composite form is recognized only where references and verbs accept it (i.e., where the field's `AllowedKinds` is open, or where the verb explicitly takes a composite id).

---

## 3. Status sets and transitions

### AC status

`open | met | deferred | cancelled`.

| From | Legal targets |
|---|---|
| `open` | `met`, `deferred`, `cancelled` |
| `met` | `deferred`, `cancelled` (scope-change after the fact) |
| `deferred` | (terminal) |
| `cancelled` | (terminal) |

`deferred` and `cancelled` differ in intent: `deferred` = "still want this, not now" (link to the receiving milestone in body); `cancelled` = "no longer in scope" (link to a `D-NNN` decision in body).

### TDD phase

`red | green | refactor | done`. Linear:

| From | Legal targets |
|---|---|
| `red` | `green` |
| `green` | `refactor`, `done` |
| `refactor` | `done` |
| `done` | (terminal for that AC's cycle) |

`refactor` is optional — `green` may go directly to `done`. The linearity prevents a green-without-red claim.

### Cross-rule (the audit hook)

When milestone `tdd: required`, AC `status: met` requires `tdd_phase: done`. Enforced by the `acs-tdd-audit` finding (error). When `tdd: advisory`, the audit emits a warning instead. When `tdd: none` (the default), `tdd_phase` is ignored.

### Milestone-done implies AC progress

A milestone may not transition to `done` while any AC has `status: open`. (`deferred` and `cancelled` are acceptable terminal AC states for a done milestone, with the body explanation as the documentation.) Surfaces as the `milestone-done-incomplete-acs` finding (error). The check runs on every `aiwf check` pass — not just on verb projection — so a milestone that became `done` via `--force --reason` while ACs were still open will keep surfacing the inconsistency until the ACs reach a terminal state. The finding message lists the open AC ids. `--force --reason` overrides the verb-time refusal like any other transition; the standing check still reports.

---

## 4. The `--force --reason` escape

Any verb that performs a status transition (`promote`, `cancel`, phase updates) accepts:

- `--force` — relax the transition-legality check.
- `--reason "<text>"` — required when `--force` is set; non-empty after trim.

The reason lands as a `aiwf-force: <reason>` trailer alongside the standard trailers. `aiwf history` surfaces forced transitions distinctly so they're auditable.

`--force` does not relax other checks: id format, closed-set membership, reference resolution, body coherence — all still run on the projection.

---

## 5. Trailer schema extensions

Existing trailer set:

```
aiwf-verb: promote
aiwf-entity: M-001
aiwf-actor: human/peter
```

Add two trailers:

| Trailer | When | Value |
|---|---|---|
| `aiwf-to:` | every promote event (milestone, AC, AC tdd_phase) | target status or phase, e.g., `met`, `green`, `done` |
| `aiwf-force:` | only when the transition was forced | the reason string |

AC events use the same `promote` verb; the entity is the composite id:

```
aiwf-verb: promote
aiwf-entity: M-007/AC-1
aiwf-to: green
aiwf-actor: ai/claude
```

`aiwf history M-007/AC-1` matches by composite id; `aiwf history M-007` shows milestone-level events plus all of its ACs (the path-prefix match is anchored on the literal `/` boundary so `M-007/` cannot prefix-match `M-070/`).

**Rollout: forward-only writer, no inference, no backfill.** Once Step 5 lands, every new `promote` commit (milestone and AC) carries `aiwf-to:`. Pre-I2 commits stay as they are; `aiwf history` renders the target-state column as a dash (`-`) for trailer-less rows. The reader does not parse the commit subject to infer a target — the schema boundary is honest about being a boundary. No history rewrite.

---

## 6. References to ACs

Only **open-target** reference fields accept composite ids:

| Field | Owner kind | Targets |
|---|---|---|
| `addressed_by` | gap | any kind, including `M-NNN/AC-N` |
| `relates_to` | decision | any kind, including `M-NNN/AC-N` |

Closed-target fields are unchanged: `milestone.parent → epic`, `milestone.depends_on → milestone`, `adr.supersedes → adr`, `adr.superseded_by → adr`, `gap.discovered_in → milestone | epic`, `contract.linked_adrs → adr`. ACs do not become valid targets for those.

The `refs-resolve` check learns to resolve composite ids: split on `/`, look up the milestone, then look up the AC by id within `acs[]`. Findings distinguish `unresolved-milestone` (parent missing) from `unresolved-ac` (parent exists, AC id absent).

---

## 7. What `aiwf check` enforces

New findings, all scoped to the milestone:

| Code | Severity | Trigger |
|---|---|---|
| `acs-shape` | error | `acs[]` item has invalid `id` (must match `^AC-\d+$` and equal sequence position+1, including cancelled entries), `status` outside `{open, met, deferred, cancelled}`, or `tdd_phase` outside `{red, green, refactor, done}` |
| `acs-body-coherence` | warning | frontmatter AC has no matching `### AC-<N>` heading in body, or body has heading with no matching frontmatter AC. **Pairs by id only**, not by title text — body title is prose and remains kernel-blind, consistent with the design's "prose is not parsed" principle. |
| `acs-tdd-audit` | error (or warning if `tdd: advisory`) | milestone `tdd: required` and an AC has `status: met` with `tdd_phase != done` |
| `acs-transition` | error (verb-projection only) | the projected change moves an AC to a status or phase the closed-set FSM does not allow, and `--force` was not supplied |
| `milestone-done-incomplete-acs` | error | milestone `status: done` and at least one AC has `status: open`. Runs on every `aiwf check` pass; message lists the open AC ids. |

Existing findings are extended where they touch the new fields:

- `refs-resolve` — accepts composite-id targets in `gap.addressed_by` and `decision.relates_to`; surfaces `unresolved-ac` distinct from `unresolved-milestone`.
- `status-valid` — runs against every AC's `status` and `tdd_phase` in addition to the entity's own status.
- `frontmatter-shape` — `acs[]` items must carry `id`, `title`, `status`. `tdd_phase` is required when milestone `tdd: required`.

---

## 8. Verb surface

Minimal additions; reuse existing verbs where possible.

| Verb | Change |
|---|---|
| `aiwf add` | New target: `aiwf add ac <milestone-id> --title "..."` allocates the next `AC-N` (max+1 over the full `acs[]`), scaffolds the body section, commits with standard trailers. When the parent milestone is `tdd: required`, seeds `tdd_phase: red`; otherwise leaves `tdd_phase` absent. |
| `aiwf promote` | Accepts composite ids: `aiwf promote M-007/AC-1 met`. New `--phase <red\|green\|refactor\|done>` flag for TDD phase transitions (mutex with positional state). Adds `aiwf-to:` trailer to all promote commits (milestone and AC). |
| `aiwf cancel` | Accepts composite ids: `aiwf cancel M-007/AC-1` flips AC status to `cancelled`. The entry stays in `acs[]` at its original position. |
| `aiwf rename` | Accepts composite ids: `aiwf rename M-007/AC-1 "<new-title>"` updates `acs[].title` in the parent milestone's frontmatter and rewrites the matching `### AC-N — <title>` body heading. One commit, `aiwf-verb: rename`, `aiwf-entity: M-007/AC-1`. The bare-id form (`aiwf rename M-007 <new-slug>`) keeps the existing path-rename behavior; the verb dispatches on composite-vs-bare. |
| `aiwf history` | Accepts composite ids: `aiwf history M-007/AC-1` filters by `aiwf-entity: M-007/AC-1`. The bare milestone id `aiwf history M-007` shows milestone events plus all AC events (path-prefix match anchored on `/`). |
| `aiwf show` | New verb (see §9). |
| `aiwf check` | New finding codes (above). |

All transition verbs accept `--force --reason "<text>"`.

No `aiwf ac` verb namespace. ACs are addressed as composite ids; the existing verbs handle them.

---

## 9. `aiwf show`

A new read-only verb that aggregates per-entity state. For a milestone:

```
$ aiwf show M-007
M-007 · Engine warning surface · status: in_progress · tdd: required
  parent: E-03
  depends_on: [M-006]

  ACs:
    AC-1 [open]   · phase: red       · "Engine emits warning on bad input"
    AC-2 [met]    · phase: done      · "Pack receives canonical OpResult"

  Recent history (10):
    2026-04-30  promote  M-007/AC-2 → met       (ai/claude)
    2026-04-30  promote  M-007/AC-2 → done      (ai/claude, phase)
    2026-04-29  promote  M-007/AC-2 → refactor  (ai/claude, phase)
    …

  Findings:
    (none)
```

`--format=json` emits the structured envelope. The verb is pure aggregation over the existing data sources (frontmatter, `git log`, `aiwf check`); no new state.

For composite ids: `aiwf show M-007/AC-1` renders just that AC plus its history.

---

## 10. Rituals plugin updates

The plugin shrinks to match the kernel's new responsibilities.

| Skill / template | Change |
|---|---|
| `wf-tdd-cycle` | Stays. Drops "update tracking doc" (tracking is now in the milestone doc, kernel-validated). Adds: drive `aiwf promote M-NNN/AC-N --phase <p>` at each red/green/refactor transition; the trailers are written by the kernel verb. |
| `aiwfx-start-milestone` | Stays. Replaces "create tracking doc" with "scaffold AC body sections in the milestone doc" (no-op when the merged template already pre-renders them). Absorbs "as ACs progress, append Work Log entries" guidance from the deleted `aiwfx-track`. |
| `aiwfx-wrap-milestone` | Stays. Adds: "verify Work Log entries cover every AC marked `met`"; "promote milestone to `done` only after all ACs are terminal." Calls into `aiwf check` for the audit. |
| `aiwfx-track` | **Removed.** Its convention-pointer role is obsolete (tracking is a kernel section now); its workflow guidance moves into start/wrap. |
| `templates/tracking-doc.md` | **Removed.** Sections merged into `templates/milestone-spec.md`. |
| `templates/milestone-spec.md` | Expanded with the merged sections: `## Acceptance criteria` (with `### AC-N — <title>` placeholders), `## Work Log`, `## Decisions made during implementation`, `## Validation`, `## Deferrals`, `## Reviewer notes`. |

The `rituals-plugin-plan.md` status table gets an "I2 alignment" row noting the shrink.

---

## 11. Build plan

### Step 1 — milestone schema additions

- [ ] In `internal/entity/entity.go`: add `AcceptanceCriterion` struct (`ID string`, `Title string`, `Status string`, `TddPhase string` — all with `omitempty`) and `Acs []AcceptanceCriterion` + `TDD string` fields on `Entity`. YAML tags: `acs`, `tdd` and the inner `id`, `title`, `status`, `tdd_phase`. Empty string is the absent sentinel — `omitempty` drops it on round-trip; closed-set membership rules out `""` as a legal value, so the sentinel is unambiguous.
- [ ] Update milestone schema row in `schemas` map: `OptionalFields` += `acs`, `tdd`. Absent `acs:` parses as `nil` slice (treated as `[]`); absent `tdd:` parses as empty string (treated as `none`).
- [ ] Closed sets:
  - [ ] `acAllowedStatuses = []string{"open", "met", "deferred", "cancelled"}` and `IsAllowedACStatus(s string) bool`.
  - [ ] `tddPhases = []string{"red", "green", "refactor", "done"}` and `IsAllowedTDDPhase(p string) bool`.
  - [ ] `tddPolicies = []string{"required", "advisory", "none"}` and `IsAllowedTDDPolicy(p string) bool`. Absent `tdd:` defaults to `none`.
- [ ] Unit tests for shape, closed-set membership, and absent-field defaults (no `acs:` → `[]`; no `tdd:` → `none`).

### Step 2 — composite id grammar

- [ ] Extend the entity package with a sibling pattern recognizing `M-NNN/AC-N` composites.
- [ ] Add `ParseCompositeID(s string) (parent string, sub string, ok bool)` and `IsCompositeID(s string) bool` helpers.
- [ ] `KindFromID` returns the parent kind for composites; add `SubKindFromID` returning the sub-kind label (`"ac"`).
- [ ] Unit tests covering bare ids, composite ids, malformed inputs.

### Step 3 — transitions

- [ ] AC status FSM in `internal/entity/transition.go` (or sibling): `IsLegalACTransition(from, to string) bool`.
- [ ] TDD phase FSM: `IsLegalTDDPhaseTransition(from, to string) bool`.
- [ ] Milestone-done precondition: `MilestoneCanGoDone(m Entity) (bool, []string)` returning unmet AC ids.
- [ ] Unit tests for every legal/illegal pair plus the milestone-done check.

### Step 4 — `--force --reason`

- [ ] Add `--force` and `--reason` flags to every transition verb (`promote`, `cancel`, `add`'s implicit transition).
- [ ] Validation: `--reason` required when `--force` is set; non-empty after trim.
- [ ] Trailer writer: emit `aiwf-force: <reason>` when forced.
- [ ] Integration test: forced transition skipping FSM but still failing on coherence.

### Step 5 — trailer extensions

- [ ] Update the trailer writer in `internal/gitops/` to emit `aiwf-to:` on every `promote` event (milestone and AC).
- [ ] Update `aiwf history` to render `aiwf-to:` and `aiwf-force:` when present.
- [ ] Backwards compat: pre-I2 commits without `aiwf-to:` continue to render (the field is just absent).
- [ ] Unit tests for trailer parsing and rendering.

### Step 6 — `aiwf check` rules

- [ ] `acs-shape` (error) — id matches `^AC-\d+$` and equals position+1 (cancelled entries count toward position); status in closed set; `tdd_phase` in closed set when present.
- [ ] `acs-body-coherence` (warning) — pairwise check between frontmatter `acs[]` ids and body `### AC-<N>` headings. **Pairs by id only**, not by title text.
- [ ] `acs-tdd-audit` — error when `tdd: required`; warning when `advisory`; skipped when `none`.
- [ ] `acs-transition` (verb-projection only) — checked by the `Apply` orchestrator before commit.
- [ ] `milestone-done-incomplete-acs` (error) — runs on every `aiwf check` pass; fires when a milestone has `status: done` and at least one AC has `status: open`. Message lists the open AC ids. The verb-time transition refusal is the same rule projected; this finding additionally surfaces the inconsistency on the standing tree (e.g. after a `--force` push or hand-edit).
- [ ] Update `refs-resolve` to accept composite ids in open-target fields; distinguish `unresolved-milestone` from `unresolved-ac`.
- [ ] Update `status-valid` and `frontmatter-shape` to walk `acs[]`.
- [ ] Body parsing: minimal heading walker (no full markdown parser). Regex: `^### AC-(\d+)(?:\s*[—\-:]\s*(.+))?$` — accepts em-dash, hyphen, colon, or id-only forms. The capture groups feed both the coherence pairing (group 1, the id number) and `aiwf show` (group 2, the title text when present).
- [ ] Unit + integration tests.
- [ ] **Backwards-compat fixture coverage:** the existing `internal/check/testdata/clean/` fixtures (which carry no `acs:` or `tdd:` keys) must continue to produce zero findings under the new validators. This is the load-bearing assertion that absent fields default cleanly.
- [ ] **Positive-path fixture coverage:** add one or two new milestone fixtures under `testdata/clean/` exercising `tdd: required` with a well-formed `acs[]` and matching body headings; `aiwf check` must produce zero findings on them.

### Step 7 — verbs

- [ ] `aiwf add ac <milestone-id> --title "..."` — allocate next `AC-N` (max+1 over the full `acs[]` including cancelled), append to `acs[]`, scaffold `### AC-<N> — <title>` heading in body, commit. When the parent milestone is `tdd: required`, seed `tdd_phase: red`; otherwise leave absent.
- [ ] `aiwf promote <id> <state>` accepts composite ids. New flag: `--phase <p>` for TDD phase changes (mutex with positional state).
- [ ] `aiwf cancel <id>` accepts composite ids; the entry stays in `acs[]` at its original position.
- [ ] `aiwf rename <id> <new>` accepts composite ids: `aiwf rename M-NNN/AC-N "<new-title>"` updates `acs[].title` in the parent milestone's frontmatter and rewrites the matching `### AC-<N> — <title>` body heading. One commit, `aiwf-verb: rename`, `aiwf-entity: M-NNN/AC-N`. The bare-id form keeps existing path-rename behavior; verb dispatches on composite-vs-bare.
- [ ] `aiwf history <id>` accepts composite ids; bare milestone id matches its ACs by path prefix anchored on `/`.
- [ ] `aiwf show <id>` (new) — aggregate frontmatter + history + check findings; supports composite ids.
- [ ] Integration tests against fixture trees.

### Step 8 — STATUS.md and views

- [ ] Update `aiwf status --format=md` to render AC progress per milestone (count by status, plus a per-milestone breakdown).
- [ ] Update the embedded mermaid roadmap to optionally annotate milestones with AC progress (e.g., `M-007 (3/5)`).
- [ ] STATUS.md regeneration on the pre-commit hook (already in place from `update-broaden-plan.md`) picks up the new fields automatically.

### Step 9 — rituals plugin updates (separate repo)

**Sequencing.** Author both PRs (kernel I2 in `ai-workflow-v2`, rituals shrink in `ai-workflow-rituals`) before landing either. Land kernel first; land rituals immediately after in the same session. The kernel does not warn about stale skill names — if a still-installed `aiwfx-track` writes a `tracking-doc.md` after kernel I2, the kernel just ignores the file (it's not under `work/`'s validated tree). No allow-list, no deprecation list, no `aiwf doctor` warning about deleted skills.

- [ ] In `ai-workflow-rituals`:
  - [ ] Delete `skills/aiwfx-track/`.
  - [ ] Delete `templates/tracking-doc.md`.
  - [ ] Expand `templates/milestone-spec.md` with the merged sections.
  - [ ] Update `skills/wf-tdd-cycle/SKILL.md` to call `aiwf promote --phase` and drop tracking-doc edits.
  - [ ] Update `skills/aiwfx-start-milestone/SKILL.md` and `skills/aiwfx-wrap-milestone/SKILL.md` per §10.
- [ ] Update `rituals-plugin-plan.md` status table with the shrink.
- [ ] End-to-end test in a sandbox consumer: `aiwf add epic` → `aiwf add milestone` → `aiwf add ac` → `aiwfx-start-milestone` → `wf-tdd-cycle` → `aiwfx-wrap-milestone` → `aiwf promote M-NN done`.

### Step 10 — design doc and embedded skill update

- [ ] aiwf core's embedded skills (`internal/skills/embedded/`) updated where any of them mention milestones (`aiwf-add`, `aiwf-status`, `aiwf-promote`, `aiwf-history`).
- [ ] Root `CLAUDE.md` "What the PoC commits to" section gets a one-line mention of "ACs as namespaced sub-elements of milestones; TDD opt-in per milestone."

### Step 11 — reverse-reference index on `aiwf show` (precondition for I2.5 + I3)

The HTML render planned in [`governance-html-plan.md`](governance-html-plan.md) needs to know, for each entity, *which other entities reference it* (the inversion of the existing forward-ref graph). The same index is consumed by the I2.5 provenance model ([`provenance-model-plan.md`](provenance-model-plan.md)) for the scope reachability check (whether a verb's target entity reaches the scope-entity via the reference graph). And it benefits `aiwf check` audits ("ADR is unreferenced", "this gap claims `addressed_by: M-007` but M-0007 doesn't link back") that aren't render-specific. This is why it lands in I2 rather than I3 or I2.5 — it's a shared dependency.

- [ ] In `internal/check/check.go`: extract the forward-ref collection in `collectRefs()` so it can be inverted without re-walking. (Today it produces findings; the data is already collected.)
- [ ] In `internal/tree/`: build a reverse-ref index at tree-load time — an in-memory map `id → []referrer` covering frontmatter ref fields (per the schema table in `design-decisions.md`) plus composite-id mentions in open-target fields. Costs one O(N) walk; no new on-disk state.
- [ ] In `cmd/aiwf/show_cmd.go`: extend `ShowView` with a `referenced_by []string` field; populate from the reverse-ref index. JSON envelope includes the field even when empty (zero-value `[]`).
- [ ] `aiwf show --help` lists `referenced_by` alongside the existing fields. Embedded `aiwf-show` skill (or equivalent) updated to mention it. The AI-discoverability rule from `CLAUDE.md` requires this — a future AI assistant must be able to learn the field exists without grepping source.
- [ ] Tests: golden JSON files per kind covering the inversion (every entity that names a target appears in the target's `referenced_by`); composite-id targets resolve correctly (a gap with `addressed_by: M-007/AC-1` shows up in the AC's `referenced_by` *and* in the milestone's `referenced_by` via prefix); cycles don't loop (forward-ref already filters; the inversion follows).

---

## 12. What is NOT in scope

| Feature | Why not |
|---|---|
| AC as 7th entity kind | Composite ids `M-NNN/AC-N` give first-class addressability without inverting the composition relationship. |
| Per-AC commit allocation fields (`red_commit:`, `green_commit:`) | Trailers via `aiwf history` recover this. YAGNI. |
| Forcing a milestone to have ≥1 AC | ACs remain optional. |
| Forcing TDD entry conditions ("can't go in_progress without all ACs in red") | Audit-only at the kernel. The rituals plugin's `wf-tdd-cycle` drives the flow. |
| Per-AC global ids (`AC-1234`) | Sub-scoping is the point. |
| AC tombstone beyond status-cancel | `cancelled` is the tombstone. |
| Multi-host adapters for the new templates | Single-host (`claude-code`) for the PoC, per existing scope decisions. |
| FSM-as-YAML for the new closed sets | Hardcoded in Go, like the existing six. |

If real friction shows up later, revisit. YAGNI.

---

## Status

| Step | State | Owner |
|---|---|---|
| 1 — schema additions | proposed | core |
| 2 — composite id grammar | proposed | core |
| 3 — transitions | proposed | core |
| 4 — `--force --reason` | proposed | core |
| 5 — trailer extensions | proposed | core |
| 6 — `aiwf check` rules | proposed | core |
| 7 — verbs | proposed | core |
| 8 — STATUS.md views | proposed | core |
| 9 — rituals plugin shrink | proposed | rituals repo |
| 10 — design doc and embedded skill update | proposed | core |
| 11 — reverse-ref index on `aiwf show` (I3 render precondition) | proposed | core |
