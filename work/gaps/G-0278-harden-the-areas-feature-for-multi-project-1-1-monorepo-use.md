---
id: G-0278
title: Harden the areas feature for multi-project (1:1) monorepo use
status: open
discovered_in: E-0043
---
## Context

The area feature (`E-0043`, shipped in `v0.17.0`) groups entities by workstream and is currently label-only: a closed set declared in `aiwf.yaml` (`areas.members` + `areas.default`), assigned per root entity in frontmatter, surfaced via `--area` filters and grouped `status` / roadmap / HTML renders.

Its primary intended use is a **multi-project monorepo with a 1:1 project↔area mapping** — each area names one project directory. In that setting the area shadows a *physical* boundary, which is both the lowest-risk case and an opportunity: the kernel can regain an oracle (the project's paths) that the single-project-carved-into-sections case structurally lacks.

Keystone: **anchor each area to the path glob of the project it represents.** Once an area knows where its project lives, the checks aiwf "can't have" for a purely semantic boundary all become buildable.

## Tier 0 — close the silent-drop holes (cheap; no new config)

- **Partition totality / disjointness property test** on `internal/areagroup`: for any input, every item lands in exactly one output group (count-in == count-out, no dupes). Makes the view-layer drop failure mechanically impossible rather than hoped-for.
- **Referential integrity on the closed set**: removing or renaming a still-referenced area in `aiwf.yaml` currently orphans entities silently into the complement bucket. Make that a loud finding; make rename a verb (`aiwf rename-area`) that atomically rewrites every referencing entity.
- **`areas.required: true` knob**: in a 1:1 monorepo every entity belongs to exactly one project, so untagged is genuinely illegal. The knob promotes the untagged finding from advisory to error.

## Tier 1 — the oracle (the real monorepo hardening)

- **`paths:` per area member**: evolve `config.Areas` from a flat label list to label+location, e.g. `members: [{name: app-a, paths: ["projects/app-a/**"]}]`. The existing custom `Areas` unmarshaler can accept both the old string form and the object form (backward compatible). This is the keystone — everything in Tier 2 depends on it.
- **Bijection / coverage check**: every declared area's glob matches a real directory (no dead config); every project directory maps to exactly one area (no project nobody slotted). The reverse check — a project directory with no area — is monorepo-specific and catches a newly-added project that fell off the map.

## Tier 2 — exploit the oracle

- **Mistag detection**: aiwf already links entities to commits via the `aiwf-entity:` trailer. For a landed entity, gather its commits and check the touched files fall under its area's glob; an `app-a`-tagged entity whose diff only hit `projects/app-b/**` is a warning (with an acknowledge path, since some cross-cutting is legitimate). This is the check that actually catches the "filed wrong, flew under the radar" failure.
- **Auto-derive / suggest area from paths**: once paths exist, `aiwf add` / wrap can infer or default the area from touched paths, driving manual tags (and mistags) toward zero. Planned work has no diff yet, so derivation lands at implementation / wrap time or from an explicit target-path hint.

## Payoff

Tiers 1-2 flip the risk profile of `--area` filtering. In the single-project case a mislabel can make an entity disappear from a filtered view, so the filter must never be treated as authoritative. Once the label is path-verified and mandatory, the filter becomes trustworthy — `aiwf list --area app-a` is a reliable "all app-a work," promoting the filter from convenience to load-bearing.

## Constraints (deliberately do NOT do)

- **Keep `area` single-valued** — resist any pull toward a list-of-areas per entity; that reintroduces the cross-cutting fuzziness the 1:1 model escapes.
- **Keep grouping non-gating in default views** — mandatory + verified paths make opt-in *filtering* safe, but the unscoped `status` must still never hide anything.

## Notes

Discovered while reflecting on `E-0043` post-release. This is a coherent multi-phase effort whose Tier structure maps cleanly to milestones — a natural candidate to seed an epic rather than be burned down as a single gap.
