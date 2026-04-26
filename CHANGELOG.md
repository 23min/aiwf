# Changelog

All notable changes to ai-workflow are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project loosely uses [Semantic Versioning](https://semver.org/) once releases start shipping.

Until v0.1.0 lands, every entry sits under `[Unreleased]`.

## [Unreleased]

### Added

- Work-tracking system for this repo (#3): `task` issue template requiring a build-plan row, scope, acceptance criteria, and principles-checklist risks; PR template requiring an issue citation, re-asserted acceptance bullets, and CHANGELOG confirmation; a `pr-conventions` CI workflow that mechanically enforces conventional-commit titles, issue citations in the PR body, and a touched `[Unreleased]` CHANGELOG entry (skip with the `internal-only` label). `CLAUDE.md` "Work tracking" section codifies the agreement and pins a concrete trigger for revisiting whether to dogfood the framework on its own development.
- Stage 2 PR 1: Go infrastructure scaffold. `go.mod`, `.golangci.yml`, `Makefile`, Go CI workflow, and a stub `aiwf` binary that emits the standard JSON envelope on `--help`, `--version`, and reports `NOT_YET_IMPLEMENTED` for any other verb. Establishes the envelope contract in tests from day one.

### Changed

- Migrated `.golangci.yml` to golangci-lint v2 schema; CI workflow now pins `golangci-lint-action@v7` with `version: v2.11.4`. Local `golangci-lint run` now matches CI on dev machines running Go 1.23+.
