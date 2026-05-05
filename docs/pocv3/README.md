# `pocv3/` — load-bearing PoC documentation

This directory holds **all** documentation for the `poc/aiwf-v3` branch. The branch is intentionally isolated from `main`: research documents and the earlier architecture design have been removed at the `docs/` top level so they do not pollute Claude's working context. They remain on `main` for visitors who want to follow the design trajectory.

Co-locating PoC docs under `pocv3/` (rather than at `docs/` directly) keeps `docs/` clean for whatever lives there on `main`, so a future merge needs to merge tree contents rather than file-by-file. New PoC documentation lands here.

## Sitemap — what to read first

**For visitors:** start at [`overview.md`](overview.md) (what aiwf is, what it isn't), then [`architecture.md`](architecture.md) (the foundational reference: layers, data flow, boundaries, load-bearing principles), then [`workflows.md`](workflows.md) (worked walk-throughs).

**For engine contributors:** [`gaps.md`](gaps.md) (open + resolved defects, check the matrix before starting work), then [`design/design-decisions.md`](design/design-decisions.md) (kernel commitments any change must preserve), then [`design/design-lessons.md`](design/design-lessons.md) (the three architectural principles distilled from the design arc).

**For skill authors / AI scaffolders writing skills that touch aiwf state:** [`skill-author-guide.md`](skill-author-guide.md). Pair with `aiwf schema [kind]` and `aiwf template [kind]` at the CLI.

**For people picking up where the build left off:** [`plans/poc-plan.md`](plans/poc-plan.md) is the index — it links Sessions 1–5 and the iteration plans on top: [`contracts-plan.md`](plans/contracts-plan.md) (I1), [`acs-and-tdd-plan.md`](plans/acs-and-tdd-plan.md) (I2), [`provenance-model-plan.md`](plans/provenance-model-plan.md) (I2.5), [`upgrade-flow-plan.md`](plans/upgrade-flow-plan.md) (release/upgrade), [`update-broaden-plan.md`](plans/update-broaden-plan.md) and [`rituals-plugin-plan.md`](plans/rituals-plugin-plan.md) for cross-cutting work, plus the queued [`governance-html-plan.md`](plans/governance-html-plan.md) (I3) and [`status-report-plan.md`](plans/status-report-plan.md).

**For consumers migrating from a prior planning system:** [`migration/from-prior-systems.md`](migration/from-prior-systems.md) and [`migration/import-format.md`](migration/import-format.md).

## Layout

```
docs/pocv3/
  README.md                           this file (sitemap)
  architecture.md                     foundational reference (system shape, data flow, boundaries)
  overview.md                         1-2 page intro: what aiwf is, what it isn't
  workflows.md                        end-user workflow cookbook
  skill-author-guide.md               contract for skill scaffolders
  gaps.md                             open + resolved gaps; high-touch

  design/
    design-decisions.md               the kernel commitments + non-goals (load-bearing reference)
    design-lessons.md                 the three principles + sweep findings
    provenance-model.md               I2.5 principal × agent × scope model (full spec)

  plans/
    poc-plan.md                       Sessions 1-5 index + iteration cross-links
    contracts-plan.md                 I1 — contracts
    acs-and-tdd-plan.md               I2 — acceptance criteria + TDD
    provenance-model-plan.md          I2.5 — provenance model (principal × agent × scope)
    upgrade-flow-plan.md              release tagging, aiwf upgrade verb, doctor skew rows
    update-broaden-plan.md            broaden aiwf update to the upgrade-pipeline shape
    rituals-plugin-plan.md            rituals plugin extraction plan
    governance-html-plan.md           I3 — governance HTML render (queued)
    status-report-plan.md             markdown status renderer with mermaid (queued)

  migration/
    from-prior-systems.md             generic migration guide (two-stage producer-side)
    import-format.md                  the import manifest format spec
```

## What does *not* live here

- Skill source — embedded under `internal/skills/embedded/<skill-name>/SKILL.md`.
- Engineering rules — root `CLAUDE.md` and `CLAUDE.md`.
- Test fixtures — under `internal/check/testdata/`.

## Charter notes

- A document is **load-bearing** if a verb's behavior, a check's contract, or a skill's recipe references it. Load-bearing docs may not be deleted without a corresponding code change; renaming requires the cross-reference sweep that produced this layout.
- A document at the top level of `pocv3/` should be one a contributor is likely to read in a typical work session. Material that's only consulted on first contact (the migration guides) lives in subdirs.
- The categories (`design/`, `plans/`, `migration/`) are intentionally few. Resist the urge to add `notes/` or `wip/`; brainstorming material that hasn't earned a permanent home does not belong checked in.
