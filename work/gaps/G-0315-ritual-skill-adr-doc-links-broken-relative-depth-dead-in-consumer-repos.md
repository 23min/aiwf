---
id: G-0315
title: 'Ritual-skill ADR doc-links: broken relative depth + dead in consumer repos'
status: open
discovered_in: M-0195
---
## Problem

The `aiwfx-*` / `wf-*` ritual skills under `internal/skills/embedded-rituals/`
cite design/ADR docs via markdown doc-links — the carve-out G-0299 grants
("a markdown link to a design/ADR doc is the one carve-out"). Two defects make
those links not serve their stated "read more" purpose:

1. **Broken relative depth.** A ritual SKILL.md lives at
   `internal/skills/embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md`
   — seven directory levels below the repo root — but the doc-links use six
   `../` segments (`../../../../../../docs/adr/ADR-XXXX-*.md`), which resolves to
   `internal/docs/adr/...` and does not exist. Pre-existing across the ritual
   skills (verb skills under `internal/skills/embedded/<skill>/` are only four
   levels deep and resolve correctly with four `../`). Surfaced while restoring
   two ADR links during the M-0195 sweep; the wrong depth was already there.

2. **Dead in a consumer repo regardless.** Even with the depth fixed, the link
   targets `docs/adr/ADR-XXXX-*.md` exist only in *aiwf's own* repo. A consumer
   materializes the skill into `.claude/skills/<skill>/` and has no `docs/adr/`
   tree, so the "read more" link is dead for the very audience the shipped skill
   serves.

## Why it matters

This questions the soundness of G-0299's doc-link carve-out as written. The
carve-out's premise is that a doc-link gives the reader a path to the decision;
for a *consumer-facing* shipped skill, that path doesn't exist. The carve-out is
meaningful only for aiwf's own dogfooding (where `docs/` is present), not for
the consumer who receives the materialized skill — the same cross-tree-leakage
class G-0299 exists to eliminate, one level up.

## Options to weigh

- **Fix the depth, accept dogfooding-only value.** Correct the `../` counts so the
  links resolve in aiwf's repo; document that the carve-out is dogfooding-only and
  consumers see a dead link. Cheapest; leaves the consumer experience poor.
- **Drop ADR references from shipped skill bodies entirely.** Reword "see the
  branch-model ADR" without a link; provenance lives in CLAUDE.md / design docs /
  commit trailers, not the consumer-facing behavioral skill. Conflicts with the
  prior discoverability ACs (M-0104/AC-2 etc.) that *require* the ADR reference —
  those would need revisiting.
- **Materialize a docs stub / link to a public URL.** Ship a minimal pointer or an
  absolute upstream URL the consumer can actually follow. More machinery.

## Sequencing

Discovered-in M-0195 (the skill-body sweep). In-family with E-0048 (skill &
ritual content integrity). The depth bug is mechanical; the carve-out-soundness
question is a design decision that interacts with the prior discoverability ACs —
likely a recorded decision (an ADR) before any sweep of the links.
