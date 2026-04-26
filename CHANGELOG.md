# Changelog

All notable changes to ai-workflow are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project loosely uses [Semantic Versioning](https://semver.org/) once releases start shipping.

Until v0.1.0 lands, every entry sits under `[Unreleased]`.

## [Unreleased]

### Added

- Stage 2 PR 1: Go infrastructure scaffold. `go.mod`, `.golangci.yml`, `Makefile`, Go CI workflow, and a stub `aiwf` binary that emits the standard JSON envelope on `--help`, `--version`, and reports `NOT_YET_IMPLEMENTED` for any other verb. Establishes the envelope contract in tests from day one.
