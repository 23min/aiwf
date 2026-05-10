---
id: G-0065
title: 'No aiwf retitle verb: scope refactors that change an entity''s or AC''s intent leave frontmatter title fields permanently misleading; only slug rename is supported'
status: addressed
addressed_by:
    - M-0077
---

## What's missing

The kernel has no verb to mutate an entity's `title` field or an AC's `title` field. Existing verbs:

- `aiwf rename <id> <new-slug>` — slug only (file/dir name); title untouched.
- `aiwf edit-body <id>` — markdown body only; frontmatter untouched.
- No `aiwf retitle`, no `--title` flag on `rename`, no documented hand-edit path.

The kernel's own discipline forbids hand-editing frontmatter (per CLAUDE.md and the aiwf-add skill: *"The frontmatter itself is structured state; leave it to verbs"*). For `status:` this is enforced by trailer-coherence audits; for `title:` there is no audit, but the principle stands — and there is no verb to do it cleanly.

The closest non-hand-edit workaround is **cancel-and-replace**:

- For an AC: `aiwf cancel M-NNN/AC-N` then `aiwf add ac M-NNN --title "<new>"` allocates a new AC at the next position. The cancelled AC remains in `acs[]` (position-stable), so the milestone's frontmatter accumulates dead entries. Six AC retitles produce six cancellations + six adds = twelve commits.
- For a top-level entity: there is no parallel pattern at all. `aiwf cancel <id>` flips the entity to a terminal status; you can then `aiwf add <kind> --title "<new>"` to create a new entity, but the **id changes**, references in other entities now point to the cancelled entity, and `aiwf reallocate` doesn't fix that (reallocate is for collisions, not for ID continuity across renames).

In practice, the friction shows up as soon as a milestone's scope is refactored. **The triggering instance** for this gap was the M-0066 rescope (2026-05-07): G-0063's sub-decision #4 generalized the `acs-body-empty` rule to `entity-body-empty`, but the milestone's `title` and AC titles still say "acs-body-empty" / "ACs with non-empty body prose..." with no clean way to update them. The rescope had to ship a "## Rescope note" body section that explicitly tells readers the frontmatter titles are stale.

### Suggested shape

A new verb `aiwf retitle <id> <new-title>` that:

- Edits the `title:` field in the entity's frontmatter (or the matching `acs[].title` for composite ids).
- Validates: the new title must be non-empty and trimmed; the entity must exist; the frontmatter parses cleanly.
- Commits with `aiwf-verb: retitle`, `aiwf-entity: <id>` (composite for ACs), `aiwf-actor: <actor>` trailers. A new trailer `aiwf-old-title: "<old>"` would let `aiwf history <id>` render title changes legibly.
- One commit per invocation (kernel atomicity rule).
- Optionally accepts `--reason "..."` to land in the commit body for audit narratives like the M-0066 rescope.

### Open questions

- **Should retitle preserve the slug or update it?** Slug derives from title via `Slugify()` at creation. If retitle leaves the slug stale, the file/dir path doesn't reflect the new title. If retitle auto-renames the slug, the audit is two events (title + slug) bundled into one verb — which mirrors the existing `add contract --validator ... --schema ...` precedent for adjacent atomic writes. Lean: opt-in via `--also-rename-slug` flag, default no (preserve filesystem stability). The operator can run `aiwf rename` separately if they want.
- **Should title changes be reportable in `aiwf history`?** Currently history renders `aiwf-verb` and `aiwf-to:` — for retitle, the natural surfacing is "<id>: <old-title> → <new-title>" via the new trailers above. Implementation should add a renderer arm.
- **Provenance:** retitle is a frontmatter mutation, so the standard principal × agent × scope rules apply (no special sovereignty needed).

## Why it matters

- **Refactors leave audit trails permanently misleading.** Anyone running `aiwf show M-066` after the 2026-05-07 rescope sees a title that no longer reflects the milestone's scope, even though `aiwf history M-066` records the rescope. This is the inverse of the kernel's "prose is not parsed; titles are structured state" stance — the structured state can drift further from reality than the prose.
- **The cancel-and-replace workaround is unworkable for top-level entities.** AC retitling is bookkeeping noise (twelve commits for six retitles), but at least the ids stay stable. For an epic or milestone whose scope shifts, the workaround changes the id, breaking references throughout the planning tree. There is no kernel-supported path to *just change the title*.
- **G-0063's sub-decision #4 is currently impossible to enforce truthfully.** When `entity-body-empty` lands as a kernel-enforced finding, the operator who rescopes a milestone (like M-0066) cannot update its title to match. The milestone passes the body-empty check (its body is non-empty) but the title field is stale, and there's no rule that catches stale titles. That's a quality gap a `aiwf retitle` verb would close.
- **AI-discoverability:** when a future AI assistant reads the rescoped M-0066, the Goal section says one thing and the title says another. The skill ecosystem cannot reconcile that without a kernel-supported correction path.

### Predecessor / related references

- Triggering instance: M-0066 rescope (2026-05-07), recorded in that milestone's body under "## Rescope note (per G-0063)".
- Sibling: G-0063 (open) — sub-decision #4 governs the rescope; its in-flight work is what surfaced this gap.
- Sibling: G-0058 (open) — AC body chokepoint, becomes generalized via the M-0066 rescope; uncovered the missing retitle path.
- Adjacent: G-0061 (open) — `aiwf list <kind>` verb missing. Similar shape ("framework documents/assumes a verb that doesn't exist"); could be addressed under the same epic that adds verb-surface improvements.

