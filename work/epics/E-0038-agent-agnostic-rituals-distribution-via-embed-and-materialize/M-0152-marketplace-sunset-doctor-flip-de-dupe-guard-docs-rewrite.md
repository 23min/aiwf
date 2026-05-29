---
id: M-0152
title: 'Marketplace sunset: doctor flip, de-dupe guard, docs rewrite'
status: draft
parent: E-0038
depends_on:
    - M-0150
    - M-0151
tdd: required
acs:
    - id: AC-1
      title: aiwf doctor verifies materialized artifacts instead of recommending plugin
      status: open
      tdd_phase: red
    - id: AC-2
      title: De-dupe guard detects enabled plugin and instructs disable, no settings edit
      status: open
      tdd_phase: red
    - id: AC-3
      title: Operator-setup docs rewritten; recommended_plugins dropped from default yaml
      status: open
      tdd_phase: red
    - id: AC-4
      title: Live-repo install smoke after de-dupe guard, human-verified
      status: open
      tdd_phase: red
---
## Goal

Retire the Claude marketplace channel: flip `aiwf doctor` from recommending the plugin to verifying materialized artifacts, drop `doctor.recommended_plugins` from the default `aiwf.yaml`, add a de-dupe guard that detects an enabled marketplace plugin and instructs the operator to disable it, and rewrite the operator-setup docs to the one-command flow.

## Context

Once M2–M4 make the embedded path stable, the marketplace is redundant and Claude-only. This milestone sunsets it and resolves the overlap hazard — a consumer with *both* the plugin enabled and the materialized artifacts present would expose duplicate skill `name:` values. It also updates or archives `rituals-plugin-plan.md` and rewrites CLAUDE.md § "Operator setup".

## Acceptance criteria

## Constraints

- Do **not** silently mutate the user's `.claude/settings.json` — detect-and-instruct only (ADR-0014 §5).
- Phased: the embedded path must be shipped and stable (M2–M4) before sunset.
- Doc-shaped ACs use structural assertions on the named markdown section, not flat substring greps (CLAUDE.md § "Substring assertions are not structural assertions").

## Design notes

- ADR-0014 §5 (marketplace retirement; de-dupe guard). The guard reuses the `enabledPlugins` read that `aiwf doctor` already performs against `.claude/settings.json`.

## Surfaces touched

- `internal/cli/doctor/`, the default `aiwf.yaml` seeding in `internal/initrepo/`, CLAUDE.md § "Operator setup", `docs/pocv3/plans/rituals-plugin-plan.md`.

## Out of scope

- Hard-removing the upstream rituals repo or its marketplace listing — that is the upstream's call. This milestone retires aiwf's *reliance* on the marketplace.

## Dependencies

- M3 and M4 — the embedded path complete (skills + agents + templates) and the agent-target seam in place.

## References

- **ADR-0014** (§5), **G-0177**, **E-0038**, **`docs/pocv3/plans/rituals-plugin-plan.md`**.

### AC-1 — aiwf doctor verifies materialized artifacts instead of recommending plugin

### AC-2 — De-dupe guard detects enabled plugin and instructs disable, no settings edit

### AC-3 — Operator-setup docs rewritten; recommended_plugins dropped from default yaml

### AC-4 — Live-repo install smoke after de-dupe guard, human-verified

