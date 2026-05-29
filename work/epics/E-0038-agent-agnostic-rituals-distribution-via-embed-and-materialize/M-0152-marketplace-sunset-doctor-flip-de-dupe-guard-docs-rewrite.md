---
id: M-0152
title: 'Marketplace sunset: doctor flip, de-dupe guard, docs rewrite'
status: done
parent: E-0038
depends_on:
    - M-0150
    - M-0151
tdd: required
acs:
    - id: AC-1
      title: aiwf doctor verifies materialized artifacts instead of recommending plugin
      status: met
      tdd_phase: done
    - id: AC-2
      title: De-dupe guard detects enabled plugin and instructs disable, no settings edit
      status: met
      tdd_phase: done
    - id: AC-3
      title: Operator-setup docs rewritten; recommended_plugins dropped from default yaml
      status: met
      tdd_phase: done
    - id: AC-4
      title: Live-repo install smoke after de-dupe guard, human-verified
      status: met
      tdd_phase: done
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

(Relocated from M-0150/AC-4; sequenced here so the live smoke runs against the de-dupe guard.)

## Work log

### AC-1 — doctor verifies materialized rituals
Replaced the `appendRecommendedPluginsReport` recommend-warning with `appendMaterializedRitualsReport`, backed by a new `skills.MaterializedRituals(root, target)` status helper. Doctor now emits a `rituals:` ok line (or a soft "N not materialized — run `aiwf update`" warning). · tests: `TestMaterializedRituals_*` (skills), `TestDoctorReport_RitualsMaterialized_OK`, `TestDoctorReport_RitualsMissing_WarnsSoft`, `TestAppendMaterializedRitualsReport_EmptyRoot`

### AC-2 — de-dupe guard
Added `appendMarketplaceOverlapReport`: when a rituals marketplace plugin is enabled in `.claude/settings.json` AND rituals are materialized, doctor warns `marketplace-rituals-overlap` and instructs disable — never editing settings.json. · tests: `TestDoctorReport_MarketplaceOverlap_WarnsNoSettingsEdit`, `TestDoctorReport_NoOverlap_WhenPluginDisabled`, `TestDoctorReport_NoOverlap_WhenNotMaterialized`, `TestAppendMarketplaceOverlapReport_MalformedSettings`

### AC-3 — docs rewritten; recommended_plugins retired
Removed the `Config.Doctor` struct/field + `pluginEntryPattern` + `preCheckTypedShape`; deleted the obsolete init-time marketplace nudge (`rituals.go`); rewrote CLAUDE.md § "Operator setup" and README § 2 to the one-command `aiwf init`/`update` flow; dropped the `doctor:` block from this repo's `aiwf.yaml`. Per D-0016. · tests: `TestLoad_LegacyDoctorBlockIgnored` (back-compat lax decode), `TestM0152_OperatorSetupDocRewritten` (structural, scoped to the named section), `TestM0152_DefaultYamlDropsRecommendedPlugins`

### AC-4 — live-repo install smoke
`aiwf update` in this repo materialized 21 ritual artifacts (13 skills + 4 agents + 4 templates) → `aiwf doctor` reports `rituals: ok` and the de-dupe guard fires against the real `settings.json`, naming both enabled plugins and instructing disable without modifying the file. The interactive plugin-disable is the guard's standing operator instruction (requires the `/plugin` menu), not an action aiwf performs. · backed by the self-check `doctor de-dupe guard` step + the AC-2 integration tests.

The per-phase red→green→done→met timeline is authoritative in `aiwf history M-0152/AC-<N>`.

## Decisions made during implementation

- **D-0016** — retire the `doctor.recommended_plugins` config surface entirely (struct field, validation, recommend-warning, init nudge). Lax `yaml.Unmarshal` means old consumer yamls that still declare the key load cleanly (key ignored), so no consumer breaks.

## Validation

- `golangci-lint run ./...` — **0 issues** (whole branch); the run surfaced 5 findings mid-flight (gocritic `appendAssign`/`stringXbytes`, gofumpt), all fixed before wrap.
- `go vet ./...` clean; `go build ./...` clean; `go test ./...` green except `TestFSMHistoryConsistent_PerfBudget` (a heavy-parallel git-tree-contention flake in `internal/check`, unrelated; passes 2/2 in isolation).
- `aiwf doctor --self-check` passed (30 steps), including the new `doctor verifies rituals materialized` and `doctor de-dupe guard` steps.
- **Live smoke** (this repo): `aiwf update` → 21 artifacts materialized, `.gitignore` reconciled; `aiwf doctor` → `rituals: ok` + `marketplace-rituals-overlap` naming both plugins, `settings.json` byte-unchanged.

## Deferrals

- None. Non-Claude target *writers* remain out of scope (epic-level), unblocked by M-0151's seam.

## Reviewer notes

- **AC-4 is human-gated by nature.** The install smoke + guard detection are verified live and by tests; the actual plugin-disable is the guard's *instruction* (it requires the interactive `/plugin` menu and changes the running session's active skills), so it is an operator-standing action, not an aiwf step. Promoted met on the install-smoke + guard-detection evidence.
- **`Config.Validate()` retained empty.** No cross-field rules remain after the recommended_plugins removal; kept as the validation entry point (called by Load/Write) for future rules.
- **Latent golangci-lint findings keep appearing per milestone** (M-0151 had 9, M-0152 had 5) because CI has never run on this unpushed branch — exactly the class G-0179 tracks. Each was caught by running the full gate at wrap.
- **`.gitignore` change committed.** Running `aiwf update` in this repo for the live smoke reconciled `.gitignore` with the ritual patterns (M-0149/M-0150) for the first time — a legitimate dogfooding change, bundled into the wrap.