---
id: G-072
title: milestone depends_on has six kernel read sites and zero writer verbs; populating it requires a hand-edit aiwf edit-body refuses, and neither aiwf-add nor aiwfx-plan-milestones tells the full story
status: open
discovered_in: E-20
---
## What's missing

The milestone `depends_on:` field has six read sites in the kernel and zero writer verbs:

| Concern | Location | Direction |
|---|---|---|
| Struct field on milestone | `internal/entity/entity.go:361` (`DependsOn []string`) | read |
| Schema declaration | `internal/entity/entity.go:457,460` (Optional Multi field, `AllowedKinds: KindMilestone`) | read |
| Forward-ref enumeration | `internal/entity/refs.go:38–39` (every `depends_on` entry is a `ForwardRef`) | read |
| Cycle detection | `internal/check/check.go:487–512` (`no-cycles/depends_on` finding) | read |
| Provenance scope walk | `internal/check/provenance.go:283` | read |
| Render | `cmd/aiwf/render_resolver.go:108` | read |
| Verbs that *write* the field | none | — |

The kernel reads it, validates references against the milestone set, validates the resulting DAG for cycles, walks it for provenance scope-resolution, and renders milestone graphs from it — but no verb under `cmd/aiwf/` produces a commit that sets the field. The only path to populate `depends_on:` today is hand-edit + ad-hoc commit.

That hand-edit collides with `aiwf edit-body`'s body-only contract. Adding `depends_on: [M-prev]` to a milestone's frontmatter and running `aiwf edit-body M-NNN` produces the deliberate refusal: *"frontmatter changed in the working copy — `aiwf edit-body` is body-only by design; use `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate` for structured-state edits."* None of the four named verbs sets `depends_on`. The operator's only options are (a) drop the field and live with prose-only sequencing, or (b) hand-craft a commit with manual `aiwf-verb:` / `aiwf-entity:` / `aiwf-actor:` trailers — which forges a verb path that doesn't exist and sets the precedent that frontmatter is editable as long as you remember the trailers.

Surfaced during E-20 planning: `aiwfx-plan-milestones` directs the operator to set `depends_on` on M-073/M-074, but committing those frontmatter edits required dropping the field and falling back to prose-form sequencing in the milestone spec's `## Dependencies` section and the epic's Milestones list. The 3-milestone linear chain in E-20 makes that fallback cheap, but a multi-milestone parallel-branch epic would lose machine-checkable edges (cycle detection, render arrows, provenance scope walks).

## Why it matters

The asymmetry is the wrong shape for two kernel principles:

1. **"Every mutating verb produces exactly one git commit"** implies the inverse — structured-state changes flow through verbs, not hand-edits-with-trailers. A field that's part of the schema and validated by `aiwf check` should also be writable through a verb. Today the discipline silently assumes hand-editing is the way.
2. **"Kernel functionality must be AI-discoverable"** is met partway. The kernel's `aiwf-add` skill (`internal/skills/embedded/aiwf-add/SKILL.md`) makes no mention of `depends_on` — correctly, because `aiwf add milestone` doesn't accept a `--depends-on` flag and doesn't write the field. The companion plugin skill `aiwfx-plan-milestones` (in `ai-workflow-rituals`) tells operators to "edit M-NNN's frontmatter" to add `depends_on`, with no mention of the chokepoint they'll hit on the next commit. The two skills are at different layers, neither covers the full story, and an AI walking the kernel's discoverability surface alone would conclude `depends_on` doesn't exist.

Three plausible fix shapes, increasing in scope:

- **Add `--depends-on` to `aiwf add milestone`.** Lands the field at allocation time; the same atomic commit carries the trailer. Cleanest for the planning workflow but doesn't help when a dependency is discovered after allocation. Update `aiwf-add` skill to mention the flag.
- **A dedicated `aiwf milestone depends-on M-NNN --on M-MMM` (or `--on M-MMM,M-PPP` for the multi case) plus its `--clear` inverse.** Covers the "discovered later" case. Produces one commit. Trailers identical to other mutating verbs. Update both `aiwf-add` and `aiwfx-plan-milestones` skills (the latter dropping the hand-edit instruction in favor of the verb).
- **Allow `depends_on` edits to ride along with `aiwf edit-body` under a narrow exception.** Cheapest; ugliest. Re-opens the body-only contract for one specific frontmatter field; sets a precedent that other "small" frontmatter fields might claim the same. Probably wrong but worth listing for completeness.

The first or second is the right shape. Pick whichever fits when the friction is paid for.

When the writer verb lands, two skills must be updated:

- `internal/skills/embedded/aiwf-add/SKILL.md` — gain a section (or table row) covering `depends_on` writes through whichever verb shape ships.
- `aiwfx-plan-milestones` SKILL.md (in the `ai-workflow-rituals` plugin) — replace step 6's "edit `M-NNN`'s frontmatter" with the verb invocation; tighten the reference so the planning workflow has a clean commit path end-to-end.

The skills coverage policy planned for E-20 / M-074 will catch only "every backticked `aiwf <verb>` mention resolves to a real verb" — not the inverse direction "every kernel field has a writer verb mentioned in some skill". A complementary check (kernel-field-has-writer or schema-field-has-skill-mention) is a separate kernel-discipline concern, worth filing if/when the writer verb design lands.

Discovered during E-20 milestone planning, captured before the M-073/M-074 frontmatter edits would have produced an `aiwf edit-body`-refused commit. Not in scope for E-20 because E-20's surface is the `list` verb, the skills split, and the skills coverage policy — not new milestone-mutation verbs.
