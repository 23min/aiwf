---
name: aiwf-retitle
description: Use when the user wants to fix or change an entity's title — "the title doesn't match anymore", "fix the title", "retitle to reflect new scope", "correct the title", "rename the title", or change an AC's title inside its parent milestone. Runs `aiwf retitle` so the frontmatter mutation rides through a verb route with proper trailers, instead of a hand-edit that triggers a `provenance-untrailered-entity-commit` warning.
---

# aiwf-retitle

The `aiwf retitle` verb updates the frontmatter `title:` of an existing entity (any of the six top-level kinds) or AC (composite id), in one atomic commit. Title only — no body changes (for top-level entities), no slug renames. Closes G-065 — the asymmetry where `aiwf rename` exists for slugs but no verb exists for titles.

## When to use

Scope refactors that change an entity's intent leave the frontmatter `title:` stale. The slug can be corrected via `aiwf rename`, but the title — which is what humans read in `aiwf status`, `aiwf show`, `aiwf history`, and the roadmap — stays misleading. `aiwf edit-body` can't fix it (frontmatter is off-limits to that verb). Plain `git commit` against the entity file triggers `provenance-untrailered-entity-commit` on the next `aiwf check`. `aiwf retitle` is the verb-route answer.

Triggers:
- *"the title doesn't match anymore"*
- *"fix the title"*
- *"retitle to reflect new scope"*
- *"correct the title"*
- *"rename the title"* — this phrasing overlaps with `aiwf-rename`. Title and slug are different fields; `aiwf-rename` does slugs only. If the user means the title (the prose label), this is the right verb. If they mean the slug (the path component), use `aiwf-rename`.

## What to run

```bash
# Top-level entity (any of the six kinds: epic, milestone, adr, gap, decision, contract)
aiwf retitle <id> "<new-title>" [--reason "..."]

# AC inside a milestone (composite id)
aiwf retitle M-NNN/AC-N "<new-title>" [--reason "..."]
```

Two positional arguments matching `aiwf rename`'s shape: id (or `M-NNN/AC-N`), new-title. The optional `--reason` flag lands in the commit body and surfaces in `aiwf history`, matching the pattern from `aiwf promote`/`cancel`/`authorize`/`edit-body`.

## What aiwf does

1. Looks up the entity (or AC) by id.
2. For top-level entities: rewrites the frontmatter `title:` field, leaves the body untouched.
3. For composite ids: rewrites the AC's `title` inside the parent milestone's `acs[]` AND regenerates the matching `### AC-N — <new-title>` body heading. Both happen in one atomic file write.
4. Validates the projected tree before touching disk; if a finding would be introduced, aborts with no changes.
5. Creates one commit with `aiwf-verb: retitle`, `aiwf-entity: <id>` (or `<id>/AC-N` for composite ids), `aiwf-actor: <actor>` trailers.

The slug, body prose (for top-level entities), and id are all unchanged. To change those, use a different verb: `aiwf rename` for slug; `aiwf edit-body` for body prose; `aiwf reallocate` for id.

## Validation

- Empty new title (after trimming whitespace) is rejected with a usage error.
- Same-as-current title is rejected — there's no diff to commit.
- Unknown entity id is rejected.
- Unknown AC id (e.g. `M-001/AC-99` when the milestone has fewer ACs) is rejected.

## Don't

- Don't hand-edit frontmatter to "skip the verb" — `aiwf history` won't show the retitle and the next `aiwf check` will surface `provenance-untrailered-entity-commit`.
- Don't use `aiwf retitle` for slug changes — that's `aiwf rename`. The two verbs stay separate by design (single-mutation rule keeps reasoning local).
- Don't expect the body prose under `## Goal`, `## Scope`, etc. to track the new title. That's `aiwf edit-body`'s job.
