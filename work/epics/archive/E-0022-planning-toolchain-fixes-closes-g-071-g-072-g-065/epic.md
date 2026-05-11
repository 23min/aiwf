---
id: E-0022
title: Planning toolchain fixes (closes G-0071, G-0072, G-0065)
status: done
---

# E-0022 — Planning toolchain fixes (closes G-0071, G-0072, G-0065)

## Goal

Ship three Tier 1 kernel-discipline fixes together before E-0020 implementation begins. Each removes a recurring source of noise or workaround in the planning workflow: lifecycle-gate the `entity-body-empty` rule (G-0071), add a writer surface for milestone `depends_on` (G-0072), and add a `retitle` verb for entities and ACs (G-0065). After this epic, planning a multi-milestone epic produces a clean tree, milestones declare their DAG via verb, and titles can be corrected when scope shifts.

## Context

These three gaps surfaced during E-0020 planning (`work/epics/critical-path.md`'s Tier 1) and share one motivating observation: **the planning workflow's tooling has gaps that produce persistent noise or force prose-only workarounds.** Each gap is small individually; together they're the difference between "planning a multi-milestone epic produces a clean tree" and "every planning session starts with N+M warnings and prose-only sequencing."

Concretely, allocating M-0072..M-0074 for E-0020 produced 24 `entity-body-empty/ac` warnings (G-0071, case 1); the standing `ADR-0002` produces 3 more `entity-body-empty/adr` warnings (G-0071, case 2); the depends_on edges between M-0072→M-0073→M-0074 had to fall back to prose in the milestone spec bodies because no verb writes the field (G-0072); and during the same session, scope discussions revealed that titles drift relative to body — but `aiwf rename` only changes slugs (G-0065). All three frictions were paid in full during E-0020 planning.

The epic is **pre-E-0020** by design: shipping these three fixes first means E-0020's M-0072 starts on a clean baseline, the cross-milestone dependency edges are first-class via verb, and any scope adjustment during implementation can correct the titles cleanly. The cost of waiting is that every future multi-milestone planning session pays the same friction in full.

The epic also **complements E-0021** (synthesis skill, deferred) — E-0021's tier classification will be more deterministic when more dependency data is structured (G-0072 + future G-0073) and when the warning baseline is clean (G-0071). Not a blocker for E-0021; just compatible groundwork.

Out of scope versus G-0073 (the broader cross-kind generalisation): G-0072's narrow milestone-only writer is the immediate pain. G-0073 expands the schema to cross-kind blocking and adds status-aware FSM gating — substantially more work, separate implementation epic when the friction surfaces. M-B in this epic is designed so G-0073's eventual fix extends rather than replaces it (verb signature stays compatible).

## Scope

### In scope

- **Lifecycle-gate the `entity-body-empty` rule** (G-0071). Status gating in `internal/check/entity_body.go`: skip the rule when the entity (or, for ACs, the parent milestone) is in a non-active lifecycle state. Two predicates:
  - For terminal-status entities: skip when `entity.IsTerminal(kind, status)` returns true. Helper added by E-0020/M-0072; this epic *consumes* it.
  - For ACs: skip when the parent milestone's status is `draft`.
  - Closes the 24 AC warnings on E-0020's M-0072/M-0073/M-0074 plus the 3 standing ADR-0002 warnings, plus the same pattern recurring on every future planning session and every preserved terminal entity.
- **Writer surface for milestone `depends_on`** (G-0072). Two coupled additions:
  - `--depends-on` flag on `aiwf add milestone` (sets edges at allocation time; comma-separated list of milestone ids).
  - `aiwf milestone depends-on M-NNN --on M-MMM[,M-PPP] [--clear]` dedicated verb (sets edges after allocation; `--clear` removes all entries). Reuses the same underlying writer; the flag is sugar for the verb at creation time.
  - Updates `aiwf-add` (kernel) skill and `aiwfx-plan-milestones` (rituals) skill so both surfaces are AI-discoverable.
  - **Narrow scope:** milestone→milestone only (matches today's schema's `AllowedKinds: KindMilestone`). G-0073's cross-kind generalisation is **explicitly out of scope**. The verb design must be a clean subset of G-0073's eventual generalisation — see *Constraints*.
- **Retitle verb** (G-0065). `aiwf retitle <id> <new-title> [--reason ...]`. Single mutation of frontmatter `title:` only.
  - Composite-id support for AC titles: `aiwf retitle M-NNN/AC-N "<new-title>"` retitles the AC entry inside the parent milestone's `acs:` array.
  - Title only — no body changes, no slug renames (those go through `aiwf rename`). The two verbs are deliberately separate to keep mutations atomic and reasoning local.
- All three milestones produce one git commit each per kernel rule (every mutating verb produces exactly one commit).
- TDD: required for all three milestones — net-new verb logic, FSM rule changes, and predicate functions all want red/green/refactor cycles.

### Out of scope

- **G-0073's cross-kind `depends_on` generalisation** (schema relaxation across all kinds, per-kind `SatisfiesDependency(kind, status)` predicate, status-aware FSM gating that consumes `depends_on`, reverse query). G-0073 stays open as the broader design lens; that epic, when filed, lists G-0072's writer as a folded-in dependency.
- **Phase gating for G-0071** (ACs in `tdd: required` milestones, gated on `tdd_phase`). Status gating covers both Case 1 and Case 2 with one predicate; phase gating is more precise for Case 1 only and adds complexity. Defer until precision-need justifies.
- **A combined "rename + retitle" verb.** Some scope changes shift both slug and title; today they're separate operations. A combined verb would require careful atomicity (one commit, two mutations); not worth the complexity for the rare case.
- **Title history rendering.** Old titles aren't preserved on retitle — only the current frontmatter `title:` changes; history is implicit via `git log`. A "show me previous titles" feature is deferred.
- **AC body retitling.** The AC heading (`### AC-N — <title>`) under `## Acceptance criteria` will be regenerated to match the new title as part of the same atomic commit (consistency with frontmatter), but no separate "edit AC body" surface is added — `aiwf edit-body` covers that.
- **ADR-0001/0003/0004 ratification or implementation.** Independent decisions, separate epics.
- **E-0020 itself.** This epic precedes E-0020 implementation; M-0072 starts after E-0022 wraps.

## Constraints

- **KISS — each fix is the smallest viable change.** G-0071 fix is rule-edit-plus-helper-consumption (~10 lines in `entity_body.go` plus tests). G-0072 is one new verb plus one flag plus skill updates. G-0065 is one new verb with composite-id support (existing pattern from `promote`/`show`/`history`). None creep into broader scope.
- **TDD: required for all three milestones.** Each new predicate, verb, and rule-behaviour change is exercised by tests before the implementation lands. Per the kernel discipline for net-new verb logic.
- **"What verb undoes this?" gate honored.**
  - G-0071's fix is a rule-behaviour change (read-only); reversal is config or revert-via-rebuild, not a separate verb.
  - G-0072's writer reverses via re-invocation with different inputs (`--clear` empties the list; passing different ids replaces).
  - G-0065's retitle reverses via re-invocation (`aiwf retitle <id> <previous-title>`).
- **AI-discoverability — each new verb has `--help` + skill update.** G-0072 updates `aiwf-add` (kernel embedded skill) and `aiwfx-plan-milestones` (rituals plugin skill). G-0065 either gets a new `aiwf-retitle` skill or an addition to `aiwf-rename`'s skill that disambiguates the two verbs. This epic ships *before* E-0020/M-0074's skills-coverage policy, so review enforces during the epic; the policy guards going forward.
- **Forward-compatibility with G-0073 (cross-kind `depends_on`) is non-negotiable for M-B.** When G-0073's epic ships eventually, today's `aiwf milestone depends-on M-NNN --on M-MMM` should extend cleanly to `aiwf <kind> depends-on <id> --on <id>[,<id>]` with referent-kind validation per the per-kind schema. Verb signature today must be a clean subset of the generalised design tomorrow — no design decisions that close off the cross-kind path.
- **Closed-set completion wiring.** The new verbs (`milestone depends-on`, `retitle`) and their entity-id arguments wire through the existing Cobra completion infrastructure, satisfying `cmd/aiwf/completion_drift_test.go` per E-0014's chokepoint.
- **Composite-id parsing in `aiwf retitle`** reuses the existing pattern from `aiwf show <M-NNN/AC-N>`, `aiwf history <M-NNN/AC-N>`, `aiwf promote <M-NNN/AC-N>`. No new id-parsing code; consume the existing helper.

## Success criteria

- [ ] `aiwf check` warning count drops by 27 on the kernel repo's current tree: 24 `entity-body-empty/ac` (E-0020 milestones' freshly-allocated ACs) + 3 `entity-body-empty/adr` (ADR-0002). The `unexpected-tree-file` warning on `critical-path.md` persists; that's E-0021's job.
- [ ] `aiwf add milestone --epic E-NN --tdd <policy> --title "..." --depends-on M-PPP[,M-QQQ]` allocates a milestone and atomically writes the `depends_on:` frontmatter array in the same commit; `aiwf check`'s cycle detection sees the edge.
- [ ] `aiwf milestone depends-on M-NNN --on M-MMM[,M-PPP]` sets edges on an already-allocated milestone in one commit; `--clear` empties the list. Both invocations carry proper aiwf trailers; `aiwf history M-NNN` renders the change.
- [ ] `aiwf retitle E-NN "<new-title>"` (or any entity id) updates the frontmatter `title:` in one commit; `aiwf history E-NN` shows the retitle event with the previous and new titles visible from the diff.
- [ ] `aiwf retitle M-NNN/AC-N "<new-title>"` updates the AC's `title:` inside the parent milestone's `acs:` array AND regenerates the corresponding `### AC-N — <new-title>` heading in the body, atomically in one commit.
- [ ] The `aiwf-add` and `aiwfx-plan-milestones` skills mention the `depends_on` writer surface; an AI assistant invoking either skill can discover the verb path. The `aiwf-rename` skill (or a new `aiwf-retitle` skill) covers the title-vs-slug distinction.
- [ ] Closed-set completion for `--depends-on`, `aiwf milestone depends-on --on`, `aiwf retitle <id>` argument enumerations passes `cmd/aiwf/completion_drift_test.go`.
- [ ] G-0071, G-0072, G-0065 each promoted to terminal status via `aiwf promote`; closing commits cite this epic in their bodies.
- [ ] All three milestones close with `tdd_phase: done` on every AC; per-milestone branch-coverage audit per `wf-tdd-cycle`'s hard rule.

## Design decisions (locked at planning time)

These four design questions surfaced during epic scope confirmation and were resolved via Q&A before milestone planning. They constrain the milestone implementations and feed `aiwfx-plan-milestones` as locked inputs.

| Decision | Rationale |
|---|---|
| **`aiwf retitle` lives in a new `aiwf-retitle/SKILL.md`** (not an addition to `aiwf-rename`'s skill). | Title and slug are parallel mutations on different fields, not topically related. Same discoverability-priority lens that gave us the `aiwf-status` / `aiwf-list` split: focused descriptions outrank topical bundling for distinct query phrasings. `aiwf-rename`'s skill body adds a redirect to retitle for title changes (parallel to the redirect E-0020/M-0073 puts in `aiwf-status` for query-shaped prompts). |
| **`aiwf retitle` accepts `--reason`.** | Matches the convention from `aiwf promote`, `aiwf cancel`, `aiwf authorize`, `aiwf edit-body` — every soft-state-mutating verb accepts `--reason`. Title changes during scope refactors deserve a "why"; the reason becomes searchable history via `aiwf history`. Optional flag. |
| **Multiple `depends_on` entries are expressed via comma-separated lists** (`--on M-072,M-073` on the dedicated verb; `--depends-on M-072,M-073` on `aiwf add milestone`). | Matches `--linked-adr` and `--relates-to` precedent (id-list flags). Same parsing strategy works for both surfaces, uniform with the kernel's id-list pattern. The id-list semantic reads naturally as "the list of milestones I depend on." |
| **`--depends-on` referents are validated at allocation time** (the verb refuses if any id doesn't resolve to an existing milestone). | Matches `--epic`, `--linked-adr`, `--discovered-in` precedent — the kernel's habit is validate-at-allocation for entity-id flag values. Fast feedback on typos; tree never carries dangling refs even briefly. Cycle detection remains `aiwf check`'s job (different layer — DAG validity, not referent existence). |

No further open questions blocking milestone planning.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| G-0071's fix exempts entities that *should* have body prose (active milestones with empty load-bearing sections). | Medium | Status gating skips only `draft` (pre-impl) and terminal (post-impl); active states (`in_progress`, `accepted`, `proposed` for ADRs etc.) still fire the rule. The active-state population the rule was originally designed for is preserved. |
| M-B's narrow milestone-only design closes off G-0073's cross-kind path. | Medium | The constraint is named non-negotiable above. M-B's verb signature must be a clean subset of `aiwf <kind> depends-on <id> --on <id>` — review at planning enforces. If a design proposal would close off the path, it's rejected. |
| `aiwf retitle M-NNN/AC-N` requires regenerating the body's `### AC-N` heading; the regeneration could conflict with operator hand-edits to that heading. | Low | The heading is verb-managed today (`aiwf add ac` scaffolds it; the body-coherence check enforces the format). Operators are not expected to hand-edit AC headings — that's the same discipline that already governs `aiwf add ac`. The retitle verb regenerates per the same convention. |
| Cumulative scope of three milestones plus their tests inflates the epic beyond "Tier 1 toolchain fixes" framing. | Low | Each milestone's scope is locked at planning time; each is small individually; they bundle for shipping convenience and design rhyming, not because they're inseparable. If one creeps, it splits — the epic accepts a fourth milestone rather than letting any one bloat. |

## Milestones

<!-- Bulleted list, ordered by execution sequence. Status lives in each milestone's frontmatter. Milestone ids are global (M-NNN), not epic-scoped; allocated by aiwfx-plan-milestones. -->

- [M-0075](M-075-lifecycle-gate-entity-body-empty-rule-closes-g-071.md) — Lifecycle-gate `entity-body-empty` rule; closes G-0071 · `tdd: required` · depends on: —
- [M-0076](M-076-writer-surface-for-milestone-depends-on-closes-g-072.md) — Writer surface for milestone `depends_on` (`--depends-on` flag on `aiwf add milestone` + dedicated `aiwf milestone depends-on`); closes G-0072 · `tdd: required` · depends on: —
- [M-0077](M-077-aiwf-retitle-verb-for-entities-and-acs-closes-g-065.md) — `aiwf retitle <id|composite-id>` for entity and AC titles; closes G-0065 · `tdd: required` · depends on: —

(Internal milestone dependencies are loose: M-A is independent of M-B and M-C; M-B and M-C are independent of each other and could swap if planning surfaces a reason. The recommended order is M-A → M-B → M-C, with M-A first because it pays for itself immediately by clearing warnings.)

## ADRs produced (optional)

(None expected. The narrow design choices in this epic are recorded inline in the milestone specs — verb shapes, predicate signatures, skill placement — not durable enough for ADR shape.)

## Dependencies

- **No upstream blockers.** All three fixes target existing code paths; no proposed ADR needs to land first.
- **`entity.IsTerminal(kind, status)` helper** is added by E-0020/M-0072 per the E-0020 spec. M-A consumes it. **Important sequencing:** if E-0022 ships *before* E-0020/M-0072, this epic's M-A also adds `IsTerminal` (and E-0020/M-0072 then re-uses it). If E-0020/M-0072 ships first, M-A consumes the existing helper. The two consumers of the helper coordinate on the same import path; the first to ship adds it.
- **Compatible with E-0021** (synthesis skill, deferred). E-0021's tier classification is more deterministic with G-0072's structured DAG and G-0071's clean baseline; not a blocker.

## References

- [`work/epics/critical-path.md`](../critical-path.md) — Tier 1 list naming these three gaps as the recommended pre-E-0020 cleanup. This epic is the implementation artefact for that recommendation.
- G-0071 — entity-body-empty rule lifecycle-blind (covers Case 1 draft milestones and Case 2 terminal-status entities). Closed by M-A.
- G-0072 — milestone `depends_on` has six read sites and zero writer verbs. Closed by M-B.
- G-0065 — no `aiwf retitle` verb (entities or ACs). Closed by M-C.
- G-0073 — broader cross-kind `depends_on` generalisation (out of scope; subsumes G-0072's writer scope when its eventual epic ships).
- E-0020 — Add list verb. Consumes M-A's clean baseline; `entity.IsTerminal` helper coordinated with E-0020/M-0072 (whichever ships first adds it).
- E-0021 — Open-work synthesis skill (deferred). Compatible groundwork: more structured DAG data and clean baseline make the synthesis more deterministic.
- ADR-0004 (proposed) — names `entity.IsTerminal(kind, status)` by name. M-A's predicate use is forward-compatible with ADR-0004's archive convention when that ADR's implementation epic lands.
- `internal/check/entity_body.go` — the rule M-A modifies.
- `cmd/aiwf/add_*.go`, `cmd/aiwf/rename_cmd.go` — patterns M-B and M-C follow for verb shape and trailer handling.
- `cmd/aiwf/completion_drift_test.go` — chokepoint that all new flags and verbs traverse.
