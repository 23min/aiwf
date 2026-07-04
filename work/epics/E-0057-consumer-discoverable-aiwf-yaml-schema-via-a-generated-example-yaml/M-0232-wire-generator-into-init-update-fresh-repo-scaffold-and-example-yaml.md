---
id: M-0232
title: 'Wire generator into init/update: fresh-repo scaffold and example.yaml'
status: draft
parent: E-0057
depends_on:
    - M-0231
tdd: required
acs:
    - id: AC-1
      title: Fresh-repo init writes aiwf.yaml as the fully-commented schema scaffold
      status: open
      tdd_phase: red
---

# M-0232 — Wire generator into init/update: fresh-repo scaffold and example.yaml

## Goal

Wire M-0231's generator into `aiwf init` and `aiwf update` so a consumer gets a
discoverable schema reference in their own repo: a fully-commented `aiwf.yaml`
scaffolded on a fresh repo, and an always-fresh, gitignored `aiwf.example.yaml`
written and refreshed on every run — while an existing `aiwf.yaml` is never
touched.

## Context

M-0231 produces the struct-derived generator but nothing user-facing consumes it
yet. This milestone lands the discoverability payoff and closes E-0057's
user-visible success criteria. The design (E-0057, Option C) is settled: the
never-stale reference lives in a generated sibling the user never owns, so
`update` can regenerate it freely without ever rewriting the user's live config.

## Acceptance criteria

<!-- Authored just-in-time at aiwfx-start-milestone via `aiwf add ac M-0232
     --title "..."`. Intended acceptance shape (sketched, not frozen):
       - fresh repo (no aiwf.yaml): `aiwf init` writes aiwf.yaml as the fully-
         commented scaffold from the generator
       - existing aiwf.yaml: `aiwf init` / `aiwf update` leave it byte-unchanged
       - `aiwf init` / `aiwf update` write and refresh a gitignored
         aiwf.example.yaml documenting every block
       - aiwf.example.yaml is added to the managed .gitignore (marker-managed)
       - `aiwf init --help` states the re-run is idempotent and lists what is
         never overwritten (aiwf.yaml, .claude/settings.json, user git hooks) -->

## Constraints

- **Never rewrite an existing `aiwf.yaml`.** `init`/`update` may create it when
  absent and may write/refresh `aiwf.example.yaml`, but an existing user
  `aiwf.yaml` is byte-unchanged. Consistent with the no-settings-edits-without-
  consent posture (ADR-0015).
- **`aiwf.example.yaml` is a derived artifact** — gitignored, regenerated every
  run, never hand-edited. Matches the `STATUS.md` / `site/` / materialized
  `.claude/` convention.
- **Idempotent re-run.** Running `init`/`update` twice yields the same tree
  (only the derived artifacts refresh).

## Design notes

- **Option C, locked at epic planning.** Rejected alternative: a marker-managed
  reference block regenerated *inside* `aiwf.yaml` on every `update` (the
  ADR-0018 guidance-import pattern). Rejected in favor of never touching the
  user's config file post-creation; the generated sibling carries the reference.
- **Fresh-repo inline comments may age** (never refreshed post-`init`) — accepted
  by design; the always-fresh `aiwf.example.yaml` sibling is the authority. A
  static top-of-file pointer in the scaffold routes there.
- **`.gitignore` management** reuses aiwf's existing marker-managed approach for
  gitignored artifacts (mechanism confirmed at start).

## Surfaces touched

- the `init` / `update` verb implementations (`internal/verb/` or equivalent)
- `.gitignore` management for the generated artifact
- `aiwf init --help` text

## Out of scope

- The generator itself — M-0231.
- Strict-decode / rejecting typo'd keys — `G-0307`.
- Committing `aiwf.example.yaml` (it stays gitignored).

## Dependencies

- **M-0231** — the schema model + generator this milestone renders through
  `init`/`update`.

## References

- [`E-0057`](epic.md) — parent epic (Option C design)
- [`M-0231`](M-0231-struct-derived-aiwf-yaml-schema-model-and-commented-yaml-generator.md) — the generator consumed here
- ADR-0015 — settings/config edits require explicit consent (the posture extended to config files)
- ADR-0018 — the marker-managed in-file pattern deliberately not used here

### AC-1 — Fresh-repo init writes aiwf.yaml as the fully-commented schema scaffold

