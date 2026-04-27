# Changelog

All notable changes to ai-workflow are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project loosely uses [Semantic Versioning](https://semver.org/) once releases start shipping.

Until v0.1.0 lands, every entry sits under `[Unreleased]`.

## [Unreleased]

### Added

- Design research arc under `docs/research/`: `KERNEL.md` (the eight needs the framework should serve plus cross-cutting properties any solution must respect) and seven numbered documents `00`–`06` working through how a totally-ordered event log fights git, whether the framework should reinvent state management or let git be the time machine, where discipline must live so it does not depend on the LLM, where governance and provenance UX belong, where state lives across layers (binary / config / planning / skills / cache), and a concrete PoC build plan that survives all of the above.
- Work-tracking system for this repo (#3): `task` issue template requiring a build-plan row, scope, acceptance criteria, and principles-checklist risks; PR template requiring an issue citation, re-asserted acceptance bullets, and CHANGELOG confirmation; a `pr-conventions` CI workflow that mechanically enforces conventional-commit titles, issue citations in the PR body, and a touched `[Unreleased]` CHANGELOG entry (skip with the `internal-only` label). `CLAUDE.md` "Work tracking" section codifies the agreement and pins a concrete trigger for revisiting whether to dogfood the framework on its own development.
- Stage 2 PR 1: Go infrastructure scaffold. `go.mod`, `.golangci.yml`, `Makefile`, Go CI workflow, and a stub `aiwf` binary that emits the standard JSON envelope on `--help`, `--version`, and reports `NOT_YET_IMPLEMENTED` for any other verb. Establishes the envelope contract in tests from day one.

### Changed

- `README.md` rewritten to frame this repository as design research plus an experimental PoC, rather than the earlier provenance-first framework pitch. Visitors are pointed at `docs/research/` (start with `KERNEL.md`) for the current direction and at the `poc/aiwf-v3` branch for the active implementation.
- `docs/architecture.md`, `docs/build-plan.md`, and `ROADMAP.md` carry a banner noting they predate the research arc and the event-sourced design they describe was walked back. The documents are preserved because the reasoning remains useful.

### Changed

- Migrated `.golangci.yml` to golangci-lint v2 schema; CI workflow now pins `golangci-lint-action@v7` with `version: v2.11.4`. Local `golangci-lint run` now matches CI on dev machines running Go 1.23+.
