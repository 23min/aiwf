---
id: E-0044
title: Harden the areas feature for multi-project (1:1) monorepo use
status: proposed
---

# E-0044 — Harden the areas feature for multi-project (1:1) monorepo use

## Goal

Make `--area` filtering **trustworthy** for the multi-project monorepo — the area feature's primary intended use — by anchoring each area to the path glob of the project it represents. Once an area knows where its project lives, the kernel regains an oracle (the project's paths) it structurally lacks for a purely semantic boundary, and the checks aiwf "can't have" for a label-only tag all become buildable. The payoff: `aiwf list --area app-a` becomes a reliable "all app-a work," promoting the filter from convenience to load-bearing.

## Context

E-0043 shipped the `area` feature in `v0.17.0`: a closed, validated, **label-only** grouping tag — a member set declared in `aiwf.yaml` (`areas.members` + a display-only `areas.default`), assigned per root entity in frontmatter, surfaced via `--area` filters and grouped `status` / roadmap / HTML renders. It is deliberately non-gating: the unscoped `status` never hides anything.

Label-only is the right floor, but it leaves a structural hole. In the **1:1 project↔area monorepo** — each area names exactly one project directory — the area shadows a *physical* boundary. That is both the lowest-risk case and an opportunity the single-project-carved-into-semantic-sections case can't offer: the project's paths are an oracle the kernel can check against. A mislabel in the label-only world makes an entity silently disappear from a filtered view, so the filter can never be treated as authoritative. Anchor the label to a verified path and make it mandatory, and the filter becomes reliable.

[G-0278](../../gaps/G-0278-harden-the-areas-feature-for-multi-project-1-1-monorepo-use.md) records the full problem and a three-tier design, discovered while reflecting on E-0043 post-release. This epic implements that design.

**What E-0043 already shipped (do not re-do):** the `area-unknown` check ([`internal/check/area_unknown.go`](../../../internal/check/area_unknown.go), M-0172) already fires — at warning severity — for any entity whose `area` is present but not a declared member, *including* the orphan case where a member is removed from `aiwf.yaml` while entities still reference it. So the orphan is not silent at the **check** layer today; what is silent is the **grouping view** ([`internal/areagroup`](../../../internal/areagroup/areagroup.go)), which buckets orphans into the complement without complaint. This epic adds the atomic-rewrite verb and the strictness knob around that existing finding — it does not carry a milestone that re-implements `area-unknown`.

## Scope

The work is three tiers with a hard dependency spine: Tier 0 is independent label-only hardening; Tier 1 introduces the path oracle (the keystone); Tier 2 exploits the oracle and depends entirely on it.

### In scope

**Tier 0 — close the silent-drop holes (no new config):**

- A **partition totality / disjointness property test** on `internal/areagroup`: for any input, every item lands in exactly one output group (count-in == count-out, no dupes, no drops). Makes the view-layer drop failure mechanically impossible rather than hoped-for.
- A **`aiwf rename-area` verb**: renaming a still-referenced area in `aiwf.yaml` today orphans entities silently into the grouping complement. The verb atomically rewrites every referencing entity's frontmatter and the `aiwf.yaml` member in one commit, with proper trailers — the same discipline `aiwf reallocate` applies to ids.
- An **`areas.required: true` knob**: in a 1:1 monorepo every entity belongs to exactly one project, so untagged is genuinely illegal. The knob promotes the untagged condition from advisory to a blocking finding. Orthogonal to `area-unknown` (which polices *present ⇒ declared*); this polices *present at all*.

**Tier 1 — the oracle (the keystone):**

- **`paths:` per area member**: evolve `config.Areas` from a flat label list to label+location — `members: [{name: app-a, paths: ["projects/app-a/**"]}]`. The existing custom `Areas` unmarshaler accepts both the old string form and the new object form (backward compatible — zero migration for existing configs). Everything in Tier 2 depends on this.
- A **bijection / coverage check**: every declared area's glob matches a real directory (no dead config), and every project directory maps to exactly one area (no project nobody slotted). The reverse direction — a project directory with no area — is the monorepo-specific catch for a newly-added project that fell off the map.

**Tier 2 — exploit the oracle:**

- **Mistag detection**: for a landed entity, gather its commits via the `aiwf-entity:` trailer and check that the touched files fall under its area's glob. An `app-a`-tagged entity whose diff only hit `projects/app-b/**` is flagged — with an acknowledge path, since some cross-cutting is legitimate. This is the check that actually catches the "filed against the wrong area, flew under the radar" failure.
- **Auto-derive / suggest area from paths**: once paths exist, `aiwf add` / wrap can infer or default the area from touched (or hinted) paths, driving manual tags — and mistags — toward zero.

### Out of scope

- **Multi-valued areas / a list-of-areas per entity.** `area` stays single-valued; that is exactly the cross-cutting fuzziness the 1:1 model escapes. (Carried forward from E-0043's out-of-scope.)
- **Gating the default views.** Mandatory + path-verified areas make opt-in *filtering* safe; the unscoped `status` / roadmap must still never hide anything. Verification raises filter trust, not view gating.
- **Per-area id namespacing** (`app-a/E-0001`): rejected in E-0043, still rejected — it breaks id stability (commitment #2).
- **A directory axis for entities.** Area `paths:` describe where the *consumer's project code* lives, for the oracle — they do not move aiwf's own kind-partitioned entity layout. The loader and the ADR-0004 archive convention are untouched.
- **Generalizing the oracle to the semantic-section case.** A single project carved into semantic areas has no path boundary to anchor to; this epic hardens the 1:1 path-backed case only and leaves the label-only path intact for the rest.

## Constraints

- **Backward-compatible config, zero migration.** A config using the existing `members: [app-a, app-b]` string form must parse and behave exactly as today. The object form is additive; `paths:` is optional even within it. The custom `Areas.UnmarshalYAML` absorbs both, per the spec-sourced-input discipline (test both forms, plus the mixed form).
- **`area` stays single-valued and closed-set.** No pull toward lists or a second grouping axis.
- **Default views never hide.** Path verification and `areas.required` make *filtering* trustworthy; they do not make grouping gating. An unscoped render shows every entity regardless of area health.
- **Single source of truth for the member set + its paths** is `aiwf.yaml: areas`. The field and the oracle validate against it and nothing else. No parallel registry.
- **Mistag is a warning with an acknowledge path, never silently gating.** Legitimate cross-cutting work exists; the check flags the suspicious case and offers a sovereign-traced acknowledgement, mirroring `aiwf acknowledge-illegal`.
- **Every new surface ships discoverable and tested.** Each verb, finding, knob, and config field lands with its `--help`, skill/skill-mention coverage, completion wiring, and the mechanical chokepoint that pins its claim — no half-finished implementations.

## Success criteria

<!-- Observable outcomes at epic close, not tests. -->

- [ ] An `aiwf.yaml: areas` block can declare a `paths:` glob per member using the object form, and a config using the legacy string-only form parses and behaves unchanged.
- [ ] `aiwf check` flags a declared area whose glob matches no directory (dead config) and a project directory that maps to no area (unslotted project), in the 1:1 monorepo fixture.
- [ ] `aiwf rename-area <old> <new>` renames the member in `aiwf.yaml` and rewrites every referencing entity in one trailered commit; no entity is left orphaned into the complement.
- [ ] With `areas.required: true`, an untagged entity raises a blocking finding; with the knob absent or false, behavior is exactly as E-0043 shipped.
- [ ] A landed entity whose commits touch only another area's paths surfaces a mistag finding, and an operator can acknowledge a legitimate cross-cutting case with a named, reasoned act.
- [ ] `aiwf add` (or wrap) can derive or suggest an entity's area from a path hint, so an operator can tag correctly without typing the area name.
- [ ] In a path-verified, `areas.required` monorepo, `aiwf list --area <name>` is a reliable "all work for that project" — no entity is missing due to a silent mislabel.
- [ ] Every milestone listed in *Milestones* below is `done`; G-0278 is promoted to `addressed` and archived under this epic's wrap.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Default-on auto-derive (silently set `area` from a path hint) vs suggest-only (warn/hint, human confirms)? | no | Resolved at the auto-derive milestone. Lean: derive at `aiwf add` when a single unambiguous path hint is given; otherwise suggest, never silently overwrite an explicit `--area`. Keep the human in the loop for ambiguous diffs. |
| Does the `paths:` flat-list → object schema evolution warrant an ADR? | no | Harvested at wrap via `aiwfx-record-decision`. Lean yes: it is a backward-compat schema contract with a forward-compat window, exactly the shape prior ADRs captured. |
| Glob semantics + dependency: which matcher (`**` doublestar) backs `paths:`, and does it add a dependency? | no | Resolved at the `paths:` keystone milestone; each new dep carries a one-line justification per Go conventions. Prefer an already-vendored matcher if one fits. |
| Bijection severity: is "a project directory with no area" a warning always, or an error only under `areas.required`? | no | Resolved at the bijection-check milestone. Lean: warning by default, escalating to blocking under `areas.required`, consistent with the Tier-0 knob. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The dual-form `Areas` unmarshaler silently mis-parses one form, breaking existing configs. | high | Spec-sourced test pass over string-only, object-only, and mixed forms before the keystone milestone closes; backward-compat is a named success criterion. |
| Mistag false positives on legitimate cross-cutting work train operators to ignore the finding. | medium | Warning severity + an acknowledge path; never gating. The check exists to surface the *suspicious* case, not to police every multi-area diff. |
| Tier 2's oracle-dependent value under-delivers once `paths:` lands, making the later milestones not worth their cost. | low | The tier spine lets the epic wrap early after Tier 0/1 if Tier 2 stops earning its keep — committed scope is full three tiers, but the sequence is decoupled. |

## Milestones

<!-- Candidate decomposition, refined and id-allocated by aiwfx-plan-milestones. Ordered by
     execution sequence; the path oracle is the foundation the Tier-2 work depends on. The
     two Tier-0 config/verb items may merge during decomposition. -->

Tier 0 (label-only hardening — independent of the oracle, mutually parallelizable):

- Partition totality / disjointness property test on `internal/areagroup`. *(No deps.)*
- `aiwf rename-area` verb — atomic cross-entity + `aiwf.yaml` rewrite in one trailered commit. *(No deps.)*
- `areas.required` knob — promote the untagged condition from advisory to blocking. *(No deps.)*

Tier 1 (the oracle — the keystone):

- `paths:` per-area config evolution with a backward-compatible dual-form `Areas` unmarshaler. *(No deps — foundation for all of Tier 1–2.)*
- Bijection / coverage check — declared glob ⇒ real directory, and project directory ⇒ exactly one area. *(Depends on the `paths:` keystone.)*

Tier 2 (exploit the oracle — depends entirely on the keystone):

- Mistag detection — landed entity's touched files vs its area glob, with an acknowledge path. *(Depends on the `paths:` keystone.)*
- Auto-derive / suggest area from a path hint at `aiwf add` / wrap. *(Depends on the `paths:` keystone.)*

## ADRs produced (optional)

<!-- Candidate; ratified or written during the epic, harvested at wrap. -->

- A decision on the `paths:` schema evolution (flat-list → object form, backward-compat + forward-compat window) — see *Open questions*.

## Supersedes

- [G-0278](../../gaps/G-0278-harden-the-areas-feature-for-multi-project-1-1-monorepo-use.md) — the three-tier seed this epic implements; promoted to `addressed` at wrap.

## References

- [E-0043](../archive/E-0043-optional-area-tag-for-grouping-entities-by-workstream/epic.md) — the label-only area feature this epic hardens (shipped `v0.17.0`).
- [`internal/areagroup/areagroup.go`](../../../internal/areagroup/areagroup.go) — the `Partition` helper the Tier-0 property test pins.
- [`internal/check/area_unknown.go`](../../../internal/check/area_unknown.go) — the already-shipped `area-unknown` finding the Tier-0 knob escalates around (not re-implemented).
- [`internal/config/config.go`](../../../internal/config/config.go) — the `Areas` type and custom unmarshaler the `paths:` evolution extends.
- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — archive convention left untouched (`paths:` describe consumer project code, not aiwf's entity layout).
- Design commitment #2 (stable flat ids) in `CLAUDE.md` — the constraint per-area id namespacing would violate.
