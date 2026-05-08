# Changelog

All notable changes to ai-workflow are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project loosely uses [Semantic Versioning](https://semver.org/) once releases start shipping.

Until v0.1.0 lands, every entry sits under `[Unreleased]`.

## [Unreleased]

### Added

- `docs/archive/` directory holding pre-PoC design documents (`architecture.md`, `build-plan.md`) with a README explaining what they are. New readers are pointed at `docs/working-paper.md` for the active direction.
- `docs/explorations/surveys/` collecting policy-corpus mining outputs (FlowTime, Liminara) graduated from `.scratch/`; feeds the policies-as-primitive exploration with concrete corpora rather than synthetic examples.
- `docs/explorations/05-policy-model-design.md` — design proposal for *policy* as an aiwf primitive: an opt-in seventh entity kind whose subject is the rules under which work happens, with a uniform engine-evaluator commitment that distinguishes it from ADRs, contracts, and decisions.
- Design research arc under `docs/research/`: `KERNEL.md` (the eight needs the framework should serve plus cross-cutting properties any solution must respect) and seven numbered documents `00`–`06` working through how a totally-ordered event log fights git, whether the framework should reinvent state management or let git be the time machine, where discipline must live so it does not depend on the LLM, where governance and provenance UX belong, where state lives across layers (binary / config / planning / skills / cache), and a concrete PoC build plan that survives all of the above.
- Work-tracking system for this repo (#3): `task` issue template requiring a build-plan row, scope, acceptance criteria, and principles-checklist risks; PR template requiring an issue citation, re-asserted acceptance bullets, and CHANGELOG confirmation; a `pr-conventions` CI workflow that mechanically enforces conventional-commit titles, issue citations in the PR body, and a touched `[Unreleased]` CHANGELOG entry (skip with the `internal-only` label). `CLAUDE.md` "Work tracking" section codifies the agreement and pins a concrete trigger for revisiting whether to dogfood the framework on its own development.
- Stage 2 PR 1: Go infrastructure scaffold. `go.mod`, `.golangci.yml`, `Makefile`, Go CI workflow, and a stub `aiwf` binary that emits the standard JSON envelope on `--help`, `--version`, and reports `NOT_YET_IMPLEMENTED` for any other verb. Establishes the envelope contract in tests from day one.

### Changed

- `README.md` rewritten to frame this repository as design research plus an experimental PoC, rather than the earlier provenance-first framework pitch. Visitors are pointed at `docs/research/` (start with `KERNEL.md`) for the current direction and at the `poc/aiwf-v3` branch for the active implementation.
- `docs/architecture.md`, `docs/build-plan.md`, and `ROADMAP.md` carry a banner noting they predate the research arc and the event-sourced design they describe was walked back. The documents are preserved because the reasoning remains useful.
- `docs/architecture.md` and `docs/build-plan.md` subsequently moved to `docs/archive/`; banners updated for the new relative paths. `ROADMAP.md` stays in place. Part of the `poc/aiwf-v3`-to-trunk promotion (see `PROMOTION-PLAN.md`, Step 4).

### Changed

- Migrated `.golangci.yml` to golangci-lint v2 schema; CI workflow now pins `golangci-lint-action@v7` with `version: v2.11.4`. Local `golangci-lint run` now matches CI on dev machines running Go 1.23+.

### Removed

- `tools/cmd/aiwf/` stub (124-line skeleton + 113-line test) and `tools/CLAUDE.md` (Go conventions doc). Both were placeholders from before the PoC moved to an idiomatic Go layout (`cmd/`, `internal/`) with a real implementation under the `poc/aiwf-v3` branch. The PoC's `CLAUDE.md` covers Go conventions for the trunk going forward.
