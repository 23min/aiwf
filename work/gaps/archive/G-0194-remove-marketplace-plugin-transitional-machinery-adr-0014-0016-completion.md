---
id: G-0194
title: Remove marketplace-plugin transitional machinery (ADR-0014/0016 completion)
status: addressed
addressed_by_commit:
    - debf7f5f
---
## What's missing

ADR-0014 retired the marketplace distribution channel (E-0038, done). ADR-0016 retired the upstream authoring channel (G-0193, done). The GitHub repo is archived. But the kernel still carries transitional machinery that assumes adopters might have the old marketplace plugin installed:

- `internal/cli/doctor/` emits a `marketplace-rituals-overlap` finding when it detects the marketplace plugin alongside the materialized rituals.
- `internal/config/` carries the `recommended_plugins` surface (D-0016 already proposed its retirement).
- `internal/policies/marketplace_sunset_test.go` guards the sunset transition.
- `README.md` carries a "Migrating from the marketplace plugin?" section.
- `CLAUDE.md` §"Operator setup" describes the overlap detection and plugin-disable dance.
- Doctor integration tests exercise the overlap path.

There are zero known external adopters. The upstream repo is archived. The transitional period is over — this machinery is dead code and stale documentation.

## Why it matters

The kernel principle *"no plugin architectures for a single implementation"* applies to transitional guards too. Keeping the overlap-detection code running costs cognitive load (readers encountering it think the marketplace plugin is still a live concern) and test maintenance (the sunset test must keep passing for a scenario no one will hit again).

## Resolution shape

One wf-patch on `patch/g-0194-marketplace-retirement-completion`:

### Code removals

- Delete the `marketplace-rituals-overlap` finding emission in `internal/cli/doctor/` (the check function, its finding constant, and any helper exclusive to it).
- Delete `doctor.recommended_plugins` from `internal/config/` (the config field, its parse path, and any test fixture exercising it). Closes D-0016.
- Delete `internal/policies/marketplace_sunset_test.go`.
- Remove marketplace-overlap test cases from doctor integration tests.

### Doc edits

- `README.md`: remove the "Migrating from the marketplace plugin?" blockquote section (lines ~183-184).
- `CLAUDE.md` §"Operator setup": remove the paragraph about `marketplace-rituals-overlap`, the plugin-disable dance, and the per-invocation-consent exception note. Keep the section focused on `aiwf init` / `aiwf update` as the materialization path.

### Separate follow-up (not this gap)

The `plugins/` directory name under `internal/skills/embedded-rituals/plugins/` is a structural artifact of the upstream repo's layout. Renaming it to something neutral (e.g. `bundles/` or flattening the hierarchy) touches `go:embed` directives, `ListRituals()`, `listRitualFiles()`, all fs.WalkDir paths, and test assertions. That's a distinct refactor — file it separately if/when the naming causes friction.

### Validation

- `aiwf doctor` no longer emits `marketplace-rituals-overlap` (the finding code is gone).
- `go test ./...` passes after all removals.
- `golangci-lint run` clean.
- `aiwf check` clean.
- README and CLAUDE.md read coherently with no mention of a marketplace plugin the reader can no longer install.

## References

- [ADR-0014](../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) — retired the distribution channel.
- [ADR-0016](../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md) — retired the authoring channel.
- [G-0193](./G-0193-implement-adr-0016-retire-upstream-rituals-authoring-channel.md) — the implementing work that landed the upstream retirement.
- [D-0016](../../work/decisions/D-0016-retire-doctor-recommended-plugins-verify-materialized-rituals-de-dupe-guard.md) — proposed retirement of `doctor.recommended_plugins`.

Surfaced during the G-0193 wf-patch session (2026-05-31): the retirement was declared complete but the transitional machinery was still live.
