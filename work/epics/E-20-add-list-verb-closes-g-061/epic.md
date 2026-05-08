---
id: E-20
title: Add list verb (closes G-061)
status: active
---

# E-20 — Add list verb (closes G-061)

## Goal

Ship `aiwf list` as the AI's hot-path read primitive over the planning tree, route AI discovery to it via a split-skill design that demotes `aiwf status` to its real role (human-curated narrative), and lock the discoverability surface against drift via a kernel policy. Closes G-061, whose central observation — *"AI assistants are instructed to invoke a non-existent verb"* — remains true today on every materialized consumer repo.

## Context

Today's read surface has four verbs: `aiwf status` (curated snapshot), `aiwf show` (one entity), `aiwf history` (one entity's timeline), `aiwf check` (validation). None is a filter primitive. `aiwf status` looks like one but isn't — it deliberately omits done epics, accepted ADRs, addressed gaps, and cancelled work because its job is *forward-looking narrative for a human reader*. An AI asked "every milestone with `tdd: required` that isn't done" can't answer from status — even from `--format=json` status — because the data isn't there.

The verb's absence already shipped a defect:

- [`docs/pocv3/plans/contracts-plan.md`](../../../docs/pocv3/plans/contracts-plan.md) references `aiwf list contracts` five times (lines 209, 425, 489, 593, 708) as the canonical generic verb, including in the section explaining why contract-specific list/status/matrix verbs were not built.
- [`internal/skills/embedded/aiwf-contract/SKILL.md`](../../../internal/skills/embedded/aiwf-contract/SKILL.md) line 33 instructs AI assistants to use `aiwf list contracts` for the "list recipes / contracts" workflow.
- The verb does not exist. Every assistant consulting that skill is told to invoke a command that returns "unknown command" — the inverse of the kernel principle *"kernel functionality must be AI-discoverable"*.

The skills surface is also un-policed for verb coverage. `internal/policies/discoverability.go` and `internal/policies/config_fields_discoverable.go` enforce that finding codes and config fields appear in some AI-discoverable channel; nothing structurally guards that every verb has corresponding skill coverage, or that mentions of `aiwf <verb>` inside a skill body resolve to a real verb. G-061's closing paragraph called out exactly this follow-up: a `skill-references-unknown-verb`-style check.

Two proposed ADRs name the future surface this verb has to be forward-compatible with:

- [`docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md`](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — moves terminal-status entities into `work/<kind>/archive/` on the same atomic commit as the status flip. Cites `aiwf list` by name: *"shows active by default; `--include-archived` (or `--archived`) includes archived entities."*
- [`docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md`](../../../docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — adds `finding` (F-NNN) as the seventh kind. Findings are projected to be the highest-volume kind once cycle-time emission turns on.

This epic does not depend on either ADR landing. It commits to a default semantic and a flag (`--archived`) that lets `aiwf list` track ADR-0004's design verbatim once that ADR ships, and to a kind-discovery shape that picks up `finding` automatically once ADR-0003 ships. Both ADRs explicitly reference `aiwf list`; designing *with* them costs nothing relative to designing without them.

## Scope

### In scope

- New verb `aiwf list` with V1 flags `--kind`, `--status`, `--parent`, `--archived`, `--format=text|json`, `--pretty`. Default behavior: filter out terminal-status entities. Default sort: id ascending. Default text format: one row per entity (id, status, title, parent). `--format=json` emits the standard envelope with `result` as an array of `{id, kind, status, title, parent, path}` summaries. No-args invocation prints per-kind counts.
- Closed-set shell completion for `--kind` (off `entity.AllKinds`) and `--status` (kind-aware), wired through Cobra's `RegisterFlagCompletionFunc`. Drift-prevention test in `cmd/aiwf/completion_drift_test.go` is satisfied without an opt-out entry.
- New helper `entity.IsTerminal(kind, status)` in `internal/entity/transition.go` (named in ADR-0004 §Trigger; does not exist today). Pure function; closed-set per-kind switch.
- Refactor: extract a shared filter helper from `cmd/aiwf/status_cmd.go`'s kind/status filtering (`buildStatus`'s slice loops at `status_cmd.go:259–333`) so `aiwf list --kind gap --status open` and the *Open gaps* section of `aiwf status` cannot drift. Status keeps its narrative composition (AC progress, recent activity, health counts) on top of the shared helper.
- New embedded skill `internal/skills/embedded/aiwf-list/SKILL.md`. Frontmatter description densely populated with list-shaped natural-language phrasings. Body: filter recipes, output shape, JSON envelope, when-to-use-list-vs-status decision criteria.
- Tighten `internal/skills/embedded/aiwf-status/SKILL.md`: description narrowed to narrative-snapshot phrasings; body adds a *prefer `aiwf list` for tree queries — that is the hot path* redirect.
- New kernel policy `internal/policies/skill_coverage.go`, modeled on `config_fields_discoverable.go`. Asserts: every embedded skill has non-empty `name:` and `description:` frontmatter; skill `name:` matches its directory and the `aiwf-<topic>` convention; every top-level Cobra command is documented by some embedded skill or appears in an opt-out allowlist with a one-line rationale comment per entry; every backticked `aiwf <verb>` mention inside a skill body resolves to a real registered verb. Wired into `policies_test.go` via `runPolicy(t, PolicySkillCoverageMatchesVerbs)`. Subsumes G-061's `skill-references-unknown-verb` follow-up suggestion.
- New ADR capturing the judgment rule for skills: per-verb skill is the default for mutating verbs that carry decision logic; topical multi-verb skill (precedent: `aiwf-contract`) when users reach for the concept rather than the verb; no skill when `--help` plus tab-completion fully cover the surface; discoverability priority can justify splitting within an otherwise topical group (precedent: `aiwf-status` and `aiwf-list` after this epic).
- `CLAUDE.md` gains a *Skills policy* section pointing at the ADR (judgment) and the policy (mechanical companion). Length: ~10 lines, plus an entry in the *What's enforced and where* table.
- Documentation drift fix: every `aiwf list contracts` reference in `docs/pocv3/plans/contracts-plan.md` (5 occurrences) and `internal/skills/embedded/aiwf-contract/SKILL.md` line 33 updated to `aiwf list --kind contract`.

### Out of scope

- `aiwf list` flag axes beyond V1: `--actor`, `--since`, `--has-tdd`, `--ac-status`, `--has-findings`, `--format=md`. Each requires its own filter-shape decision and its own skill-recipe entry; defer until concrete friction earns the addition.
- A new `aiwf-show` embedded skill. The skills-coverage policy lists `show` in the opt-out allowlist with rationale `"deferred — see follow-up gap"`; the epic logs that follow-up gap so the absence is tracked, not papered over.
- Implementation of ADR-0003 (finding kind) and ADR-0004 (archive convention). The list verb is forward-compatible with both but does not depend on either.
- Closure of G-068 (discoverability policy misses dynamic finding subcodes). Different policy, different fix shape, kept out so this epic stays scoped.
- Migration of the skills coverage policy into a future `P-NNN` under the `aiwf-rituals` bundle. The policy-model.md opt-in module is not a dependency and is not a deliverable here. Migration becomes name-only when the module lands.
- Any change to verb shape for verbs other than `list`. The contracts-plan and contract-skill drift fix is documentation-only — it updates references to the new verb's shape, not the contract verb surface.

## Constraints

- **KISS / YAGNI on flag set.** Ship the V1 axes (`--kind`, `--status`, `--parent`, `--archived`) and stop. Future axes earn their place when a concrete query needs them.
- **Forward-compatibility with ADR-0003 and ADR-0004 is non-negotiable.** Default semantic *must* be "non-terminal-status entities" (the same predicate ADR-0004 uses to decide archive moves). The `--archived` flag *must* match ADR-0004's preferred name verbatim. Kind enumeration *must* read from `entity.AllKinds` (or equivalent) so adding `KindFinding` later extends list and its completion automatically. If any of these three would force a UX-breaking change when the proposed ADRs land, the design is wrong.
- **Closed-set completion wiring is enforced.** `--kind` and `--status` flag values bind to completion via `cmd.RegisterFlagCompletionFunc`; the existing drift test `cmd/aiwf/completion_drift_test.go` blocks merges without it. No opt-out additions.
- **Reversal: list is read-only.** No commit produced; no inverse needed. Matches `status`, `show`, `history`. Per the *Designing a new verb* section of `CLAUDE.md`, this answers "what verb undoes this?" cleanly.
- **Skills coverage policy follows existing precedent.** Same `Violation` shape, same `readDiscoverabilityChannels` haystack helper, same allowlist-with-rationale-comment pattern as `internal/policies/config_fields_discoverable.go`. No new framework primitives in `internal/policies/`; this is one new policy file plus a test entry.
- **Mechanical vs. judgment split is preserved.** The ADR captures the judgment rule (when per-verb, when topical, when none, when discoverability priority justifies splitting). The policy captures only the mechanically evaluable invariants. The two artifacts cross-reference each other; neither smuggles the other's role.
- **No verb-shape detour.** This epic does *not* litigate `aiwf list <kind>` (positional plural) vs. `aiwf list --kind <kind>` beyond recording the choice in the implementation milestone. The flag form is locked by epic-time decision; the contracts-plan and contract-skill updates apply that shape.

## Success criteria

<!-- Observable outcomes at epic close, not tests. -->

- [ ] `aiwf list` with no args prints per-kind counts of non-terminal entities; sample shape: `5 epics · 47 milestones · 12 ADRs · 14 gaps · 3 decisions · 1 contract`.
- [ ] `aiwf list --kind milestone --status done --parent E-13` returns the matching entities in id-ascending order; the equivalent `--format=json --pretty` invocation emits a valid envelope whose `result` is an array of summary objects, each carrying `{id, kind, status, title, parent, path}`.
- [ ] `aiwf list --archived` includes terminal-status entities; the same invocation without `--archived` excludes them. Pre-ADR-0004 the difference is a status filter; post-ADR-0004 the same flag also walks `archive/` subdirs without a list-side code change.
- [ ] `aiwf list --kind contract` returns every registered contract; the five `aiwf list contracts` references in `docs/pocv3/plans/contracts-plan.md` and the line in `internal/skills/embedded/aiwf-contract/SKILL.md` resolve to working invocations.
- [ ] The In-flight, Open-decisions, and Open-gaps slices of `aiwf status` come from the same shared filter helper as `aiwf list`. A targeted test asserts that "filter by `kind=gap, status=open` via list" and "the *Open gaps* slice produced by `buildStatus`" agree on the same fixture tree.
- [ ] `internal/skills/embedded/aiwf-list/SKILL.md` exists, materializes through `aiwf init` and `aiwf update`, and its `description` enumerates list-shaped natural-language query phrasings that an AI assistant would emit. Body covers filter recipes, output shape, and the when-to-use-list-vs-status decision criteria.
- [ ] `internal/skills/embedded/aiwf-status/SKILL.md` description no longer covers list phrasings; the body redirects to `aiwf list` for tree queries.
- [ ] `go test ./internal/policies/...` runs `TestPolicy_SkillCoverageMatchesVerbs` and passes on the current tree. CI fails any future PR that adds a top-level Cobra command without either skill coverage or an explicit allowlist entry; CI also fails when an embedded skill body references a verb (`aiwf <verb>` backticked) that is not registered.
- [ ] The skills-coverage policy's allowlist contains an entry for every verb that ships without a skill, each entry carrying a one-line rationale comment in source. `show` appears in the allowlist with rationale `"deferred — see follow-up gap"`; the follow-up gap is filed under `work/gaps/`.
- [ ] A new ADR captures the judgment rule named in *Constraints* above. The ADR is `proposed` at minimum by epic close; ratification is not a blocker for the epic's done state.
- [ ] `CLAUDE.md` carries a *Skills policy* section pointing at the ADR and `internal/policies/skill_coverage.go`. The *What's enforced and where* table lists the new policy in the blocking-via-CI-test row.
- [ ] G-061's status flips from `open` to its terminal value via `aiwf promote`; the closing commit's `aiwf-entity:` trailer references G-061 and the body cites this epic as the resolution.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does `aiwf list` accept a positional kind argument as syntactic sugar for `--kind`, e.g. `aiwf list milestones`? | No | Decided in M1 design review. Lean: no — the flag form is uniform with the rest of the surface; sugar costs us a special pluralization case per kind for marginal terseness. |
| Does the `--parent` flag accept any composite parent (`E-NN`, `M-NNN`) or only the canonical-parent shape per kind? | No | Decided in M1 design review. Lean: accept any id whose value is referenced as `parent:` by some entity; reject other shapes. |
| Does the skills-coverage policy run as a Go test under `internal/policies/` only, or also as a pre-push hook? | No | Decided in M3. Lean: Go test only, matching the existing precedent in `policies_test.go`. CI is the chokepoint. |
| `show` in the allowlist with rationale "deferred" — what's the follow-up gap's text? | No | Filed during M3. Captures: `show` is the per-entity inspection verb every AI reaches for; `--help` covers the surface mechanically but body-rendering branches and composite-id handling probably warrant a skill. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| `aiwf list` ships before ADR-0004 lands; the active-default behavior surprises users who expected "all entities". | Low | Document the default in `--help` and the skill body. The default matches what every existing read verb already does (status, show — both default to the active subset). The flag name (`--archived`) telegraphs the future. |
| The skills-coverage policy fires false positives on existing skills that mention verbs in non-canonical forms (e.g. backticked `aiwf list contracts` after the documentation update misses a substitution). | Low | The policy parses backticked `aiwf <verb>` mentions and resolves against the registered Cobra command set. The contracts-plan and contract-skill update is part of M1's exit; M3's policy lights up only after that update lands. CI failures during the transition are signal, not noise. |
| Adding the policy late in the epic surfaces existing skill drift the policy was not designed to fix. | Medium | M3 runs the new policy against the current tree before considering itself done. Any pre-existing drift surfaces as findings; each is either fixed in-epic (small fix), allowlisted with rationale, or filed as a follow-up gap. The policy does not ship until the tree passes it. |
| Refactor of `status_cmd.go`'s filter slices into a shared helper introduces regressions in the existing status output. | Medium | The shared helper is added with parity tests against the current status output before status is rewritten to call it. Status text and JSON output are golden-tested per the existing pattern; M1 exit requires both goldens unchanged on the current fixture tree. |

## Milestones

<!-- Bulleted list, ordered by execution sequence. Status lives in each milestone's frontmatter. Milestone ids are global (M-NNN), not epic-scoped; allocated by aiwfx-plan-milestones. -->

- [M-072](M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md) — `aiwf list` verb, status filter-helper refactor, contract-skill drift fix · `tdd: required` · depends on: —
- [M-073](M-073-aiwf-list-skill-aiwf-status-skill-tightening.md) — New `aiwf-list` embedded skill; tighten `aiwf-status` description and body · `tdd: advisory` · depends on: M-072
- [M-074](M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md) — `internal/policies/skill_coverage.go`, judgment ADR, `CLAUDE.md` *Skills policy* section, follow-up gap for `aiwf-show` skill, G-061 closure · `tdd: required` · depends on: M-073

## ADRs produced (optional)

- ADR-NNNN — Skills policy: per-verb default, topical multi-verb when concept-shaped, no skill when --help suffices, discoverability priority justifies splitting within a topical group

## References

- G-061 — *Generic `aiwf list <kind>` verb referenced as canonical in contracts plan and shipped contract skill, but never implemented* (`work/gaps/G-061-*.md`). Closed by this epic.
- ADR-0003 (proposed) — *Add finding (F-NNN) as a seventh entity kind* (`docs/adr/ADR-0003-*.md`). List is forward-compatible: `--kind finding` works automatically once `KindFinding` is added to `entity.AllKinds` and `tree.Load` walks `work/findings/`.
- ADR-0004 (proposed) — *Uniform archive convention for terminal-status entities* (`docs/adr/ADR-0004-*.md`). List V1 default semantic ("non-terminal entities") and the `--archived` flag are taken from ADR-0004 §"Display surfaces" verbatim.
- E-13 — *Status report*. Established `aiwf status` and the curated-narrative framing this epic clarifies.
- E-14 — *Cobra and completion*. Established the `cmd.RegisterFlagCompletionFunc` pattern and the `cmd/aiwf/completion_drift_test.go` chokepoint that V1's `--kind` and `--status` completion wires through.
- `internal/policies/discoverability.go` — `PolicyFindingCodesAreDiscoverable`. Precedent for the new skills-coverage policy's haystack and Violation shape.
- `internal/policies/config_fields_discoverable.go` — `PolicyConfigFieldsAreDiscoverable`. Precedent for the new skills-coverage policy's allowlist-with-rationale pattern.
- `cmd/aiwf/status_cmd.go:259–333` — the kind/status filter slices that M1's shared helper extracts.
- `internal/tree/tree.go:178` — `tree.Load`'s `filepath.WalkDir` walk. The list verb consumes its output; no new walking is added.
- `docs/pocv3/design/policy-model.md` — future opt-in policy module; named to record the migration story (the `internal/policies/skill_coverage.go` policy lifts to a `P-NNN` under `aiwf-rituals` when the module lands). Not a dependency.
- `CLAUDE.md` — kernel principles cited verbatim: *"kernel functionality must be AI-discoverable"*, *"CLI surfaces must be auto-completion-friendly"*, *"the framework's correctness must not depend on the LLM's behavior"*, *"every mutating verb produces exactly one git commit"* (list does not mutate; this principle is the reason no commit is required), the *Designing a new verb* "what verb undoes this?" gate.
