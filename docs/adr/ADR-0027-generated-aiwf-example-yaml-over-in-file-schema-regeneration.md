---
id: ADR-0027
title: Generated aiwf.example.yaml over in-file schema regeneration
status: accepted
---
# ADR-0027 — Generated aiwf.example.yaml over in-file schema regeneration

> **Date:** 2026-07-05 · **Decided by:** human/peter

## Status vocabulary (aiwf)

aiwf's ADR statuses are: `proposed | accepted | superseded | rejected`.

- `proposed` — written up, open for discussion or ratification.
- `accepted` — in force. Steady state.
- `superseded` — replaced by a later ADR. Set `superseded_by` on this one and `supersedes` on the new ADR. Never delete the file.
- `rejected` — proposed and explicitly turned down. Keep the file for the reasoning trail; do not re-use the number.

## Context

E-0057 needed to give every aiwf consumer a discoverable, always-fresh reference for the whole `aiwf.yaml` schema, generated from the same structs that decode the file. Two designs were on the table:

1. **Regenerate a marker-managed block inside `aiwf.yaml` itself**, on every `aiwf update` — the same pattern ADR-0018 already uses for the CLAUDE.md guidance import: a fenced, tool-owned block coexisting with user-authored content in the same file, self-healing on every run.
2. **Render the schema reference into a separate, generated sibling file** — `aiwf.example.yaml` — entirely tool-owned, regenerated wholesale every run, never touching the user's own `aiwf.yaml`.

ADR-0015 already establishes a "no edits to settings/config without explicit per-invocation consent" posture for aiwf-managed files. A marker-managed block regenerated inside `aiwf.yaml` means the tool unilaterally rewrites a portion of a file the user hand-edits and owns — including its ordering and any comments placed near the block. ADR-0018's pattern works safely for CLAUDE.md because the managed content there is a single import line, trivially isolated between two HTML comment markers in a markdown file. A full, fully-commented schema reference is much larger (every config block, each with its default and description) and far more failure-prone to inject and re-inject idempotently into a live YAML document without risking silent corruption of content the user placed near it.

## Decision

aiwf renders the full schema reference into a separate, generated, gitignored sibling file — `aiwf.example.yaml` — at the consumer repo root. `aiwf init` and `aiwf update` write and refresh it unconditionally on every run, from `config.GenerateExample()`. The consumer's own `aiwf.yaml` is created once, on a fresh repo, as a fully-commented scaffold — and is never rewritten by any later `init`/`update` run.

This extends ADR-0015's posture: the tool never edits the user's live config file after creation, for any purpose — not even a well-intentioned, self-healing block.

## Consequences

**Positive:**
- `aiwf.example.yaml` can be regenerated with zero risk of corrupting a user's real config — the file carries no user content of its own.
- The consumer's `aiwf.yaml` keeps its byte-for-byte editorial history under normal git diffs; nothing aiwf-owned ever touches it post-creation.
- The generated sibling can be freely gitignored, so adding a new schema field never produces merge noise.

**Negative / accepted trade-offs:**
- The fresh-repo inline comments in the scaffolded `aiwf.yaml` go stale over time (never refreshed post-`init`), since aiwf never touches that file again. Accepted by design — `aiwf.example.yaml` is the always-fresh authority, and a static top-of-file pointer in the scaffold routes there (M-0232's Design notes).
- A teammate who hasn't run `aiwf update` won't see the generated reference file until they do. Accepted — `update` is the documented setup step, and the file regenerates on first run.
- This narrows where the ADR-0018 marker-managed in-file pattern applies: it remains correct for markdown-shaped consumer surfaces (CLAUDE.md) where the managed content is a single, trivially-isolable line. The generated-sibling pattern is the correct choice for YAML-shaped config surfaces, where broader in-place regeneration risks touching user content near the managed block.

## References

- Related ADRs: `ADR-0015` (no settings/config edits without consent — extended here), `ADR-0018` (marker-managed in-file pattern — deliberately not used here for the YAML case)
- Linked epics or milestones: `E-0057`, `M-0232`
