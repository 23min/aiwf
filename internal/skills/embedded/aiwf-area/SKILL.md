---
name: aiwf-area
description: Use when choosing which `area` to tag an entity with, or to understand how aiwf's area feature works — the closed member set, the optional `paths:` oracle, the `areas.required` knob, and the area checks (mistag, dead-glob, overlap, area-unknown). Areas are aiwf's per-workstream / per-project grouping axis: you write code anywhere, but once you touch aiwf an entity belongs to a declared area (exactly one under `areas.required`), and `aiwf check` enforces that the tag matches where the work landed. Covers picking the area at `aiwf add` (explicit `--area`, or `--path-hint` derivation), fixing it with `aiwf set-area`, renaming a member with `aiwf rename-area`, and acknowledging legitimate cross-cutting with `aiwf acknowledge mistag`.
---

# aiwf-area

An **area** is aiwf's optional grouping tag for the workstream — most often one project directory in a monorepo (the "1:1 project↔area" case). It is declared once in `aiwf.yaml` and assigned per root entity (epic, ADR, gap, decision, contract) in frontmatter; a milestone inherits its area from its parent epic. Areas drive `--area` filters and grouped `status` / roadmap / HTML views.

```yaml
# aiwf.yaml
areas:
  members:
    - {name: app-a, paths: ["projects/app-a/**"]}
    - {name: billing, paths: ["projects/billing/**"]}
  required: true        # optional: every root entity must be tagged
```

The `paths:` glob is the **oracle**: it tells aiwf where an area's code lives, so the kernel can check the tag against reality instead of trusting a bare label. `paths:` is optional — without it an area is a label-only tag and the path-backed checks stay inert.

## The mental model

Hold two facts at once:

- **You operate everywhere.** Code work is not fenced by area — you read and edit any file in the repo regardless of how areas are drawn. Areas tag *planning entities*; they never restrict where you type.
- **aiwf treats the area as both a guide and a constraint.** When you create or tag an entity, the area is a **guide** — "which workstream does this belong to?" — and aiwf can derive it for you from a path. Once set, it is a **constraint**: the value set is *closed* (only declared `areas.members`, plus the reserved `global` sentinel, are valid), the `areas.required` knob can make tagging *mandatory*, and the **mistag** check verifies — at pre-push — that the entity's commits actually landed under its area's `paths:`. A tag that disagrees with where the work landed is a finding, not a silent mislabel.

So: pick the right area (aiwf helps you), and aiwf holds you to it. The guarantee is mechanical — `aiwf check` at pre-push, not this skill.

## Picking an area at `aiwf add`

Order of precedence, highest first:

1. **Explicit `--area <member>`** always wins. It is validated against `aiwf.yaml: areas.members` (plus `global`); an undeclared value is rejected at write time.
2. **`--path-hint <repo-relative-path>`** (when `--area` is omitted) derives the area from the `paths:` oracle: if the path falls under exactly one declared area's globs, that area is filled in. Reach for this when you know the file you're about to touch but not the area name — pass the path and let the kernel match it, rather than guessing the label. An absolute path under the repo root, or one with `./` / `..` segments, is normalized before matching.
3. For a **gap**, `--discovered-in <id>` derives the area from the source entity when nothing above set one.

If `--path-hint` is **ambiguous** (matches several areas) or matches none, aiwf sets no area and prints a suggestion — it never guesses. If both `--area` and `--path-hint` are given and they disagree, `--area` wins and aiwf notes the disagreement (the cheapest possible mistag check, at creation time).

```bash
# You know the file, not the area name — let the kernel map it:
aiwf add gap --title "Login throttle off-by-one" --path-hint projects/app-a/auth/login.go
# → derives area: app-a

# Or name it explicitly:
aiwf add epic --title "Billing rework" --area billing
```

## The area lifecycle

1. **Tag at creation** — `aiwf add … --area <m>` or `--path-hint <path>` (above).
2. **Fix or assign later** — `aiwf set-area <id> <member>` retags one entity; this is the remediation path when a tag is wrong or missing.
3. **Verified at push** — the mistag check (`area-mistag`) flags an entity whose commits landed entirely under a *foreign* area's `paths:`. This is the mechanical guarantee that the tag matches reality; see the `aiwf-check` skill for the finding and its fix.
4. **Acknowledge genuine cross-cutting** — when work legitimately spans areas, `aiwf acknowledge mistag <id> --reason "…"` records a sovereign, reasoned exemption (see the `aiwf-acknowledge` skill). Don't acknowledge a simple mis-tag — `aiwf set-area` it instead.
5. **Rename a member** — `aiwf rename-area <old> <new>` rewrites the `aiwf.yaml` member and every referencing entity in one commit. Never hand-edit a member name in `aiwf.yaml`; that orphans the entities still pointing at the old label.

## When areas are `required`

With `areas.required: true` (the 1:1 monorepo case — every entity belongs to exactly one project):

- `aiwf add` **refuses** to create an untagged root entity — tag it, or pass `--path-hint` so the kernel derives one.
- `aiwf check` raises a **blocking** `area-required` finding for any untagged root entity.
- For work that genuinely belongs to no single project, tag it `global` — the reserved cross-cutting sentinel. It is always a valid value and is excluded from the mistag check.

## Related

- **`aiwf-add`** — the full `--area` / `--path-hint` write path.
- **`aiwf-check`** — the area findings (`area-mistag`, `area-dead-glob`, `area-overlap`, `area-unknown`, `area-required`) and how to clear each.
- **`aiwf-acknowledge`** — `aiwf acknowledge mistag` for legitimate cross-cutting work.
- **E-0043 / E-0044** — the epics that shipped, then path-hardened, the area feature.
