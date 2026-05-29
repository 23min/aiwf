---
id: D-0016
title: Retire doctor.recommended_plugins; verify materialized rituals + de-dupe guard
status: proposed
---
## Decision

Retire the `doctor.recommended_plugins` config surface entirely as part of the
marketplace sunset (ADR-0014 §5): remove the `Config.Doctor` struct + field,
its validation and pre-shape check, the `aiwf doctor` recommend-warning
(`recommended-plugin-not-installed`), and the init-time `/plugin marketplace
add` nudge (`printRitualsSuggestion`). `aiwf doctor` instead **verifies the
materialized ritual artifacts** under `.claude/` and emits a **de-dupe guard**
when an enabled marketplace plugin overlaps with materialized rituals.

## Context

ADR-0014 §5 phase (b) says "flip `aiwf doctor` from recommending the plugin to
verifying the materialized artifacts, and drop `doctor.recommended_plugins`
from the default `aiwf.yaml`." Once the embedded path ships (M-0149/M-0150) and
the agent-target seam lands (M-0151), the marketplace channel is redundant and
Claude-only, so the recommend-warning has nothing left to recommend.

## Why full removal (not deprecate-in-place)

- `aiwf.yaml` is decoded with lax `yaml.Unmarshal` (no `KnownFields`), so an
  existing consumer that still declares `doctor.recommended_plugins` simply has
  the key **ignored** — removing the Go field breaks no one.
- Flipping doctor (AC-1) already forces removal of the M-070 recommend-warning
  tests and the two self-check steps regardless; keeping an unread config field
  would leave dead, untested config surface — a YAGNI/"no vestigial knobs"
  smell.
- A pinned vendor SHA (M-0148) is stronger provenance than a marketplace-install
  recommendation, so nothing of value is lost.

## Consequences

- Consumers upgrading past this point: `aiwf update` removes the
  `recommended_plugins` block guidance from docs; any lingering key in their
  yaml is inert (lax decode). `aiwf doctor` now reports rituals materialization
  status and a disable-the-plugin nudge on overlap.
- `loadEnabledPlugins` is retained — the de-dupe guard reuses it.

## References

- **ADR-0014** §5 (marketplace retirement, de-dupe guard).
- **M-0152** — the implementing milestone.
- **G-0179** — the full-local-CI-gate gap filed alongside.
