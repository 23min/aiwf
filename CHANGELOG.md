# Changelog

All notable changes to `aiwf` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project follows [Semantic Versioning](https://semver.org/).

Releases ship as git tags on `poc/aiwf-v3`. The Go module proxy
resolves them when a consumer runs `aiwf upgrade` or
`go install <pkg>@latest`. The branch is not planned to merge to
`main`.

When cutting a release, see [`tools/CLAUDE.md`](tools/CLAUDE.md) §
*Release process*. The tag-push CI check at
[`.github/workflows/changelog-check.yml`](.github/workflows/changelog-check.yml)
fails any pushed `v*` tag that does not have a matching `## [X.Y.Z]`
section in this file.

## [Unreleased]

### Added
- **G37** — Cross-branch id collisions are detectable and resolvable. Trunk-aware allocator + `prior_ids` lineage; `aiwf history` walks renamed entities through their full chain; `aiwf check` flags cross-tree collisions; `aiwf reallocate` handles the trunk-ancestry tiebreaker. (`271f514`, `b9d73d8`, `c5a98c1`, `a6e8067`, `685f288`)
- Three new policies + 14 backfilled subcode docs to broaden discoverability-lint coverage. (`2b094e3`)
- **G38** — Dogfooding gap filed: investigation of running aiwf against its own kernel repo. Open. (`dd25c06`)
- **G40** — `aiwf check` now reports `unexpected-tree-file` for stray files under `work/`. Tree-shape changes go through verbs; body-prose edits to existing entity files remain free-form. Configurable via `aiwf.yaml: tree.allow_paths` (glob exemptions) and `tree.strict: true` (promote warning to error). Files inside contract directories are auto-exempt. New design doc `docs/pocv3/design/tree-discipline.md`; rule folded into `aiwf-add` and `aiwf-check` skills (no new skill). (`bdd43c2`)
- **G41** — Tree-discipline now runs at pre-commit *and* pre-push. New `aiwf check --shape-only` flag runs only the tree-discipline rule (no trunk read, no provenance walk, no contract validation), wired into the aiwf-managed pre-commit hook. The LLM gets an in-loop signal when a stray file lands, regardless of which AI client it's using — git hooks are agent-agnostic.

### Changed
- `aiwf upgrade` prints a concrete recovery path (`$GOBIN`, `$GOPATH/bin`, or `$HOME/go/bin`) when post-install binary lookup fails, instead of a generic "run aiwf update manually" message. (`9a06c74`)

### Fixed
- **G39** — `aiwf upgrade` mis-parsed `go env` output when GOBIN was unset (the default Go install setup), failing immediately after `go install` succeeded. The post-install lookup now queries GOBIN and GOPATH in separate calls so there is no multi-line shape to mis-parse. (`9a06c74`)

## [0.2.3] — 2026-05-04

### Fixed
- **G35 / G36** — HTML render now generates pages for every entity kind (gap, ADR, decision, contract — previously 404'd) and renders entity-body markdown as HTML instead of escaped raw text. (`d1bf1e1`)

## [0.2.2] — 2026-05-04

### Fixed
- **G30** — `aiwf status` and `aiwf history` no longer pick up prose-mention false positives from `git log --grep`. (`7141f2a`)
- **G31 / G32 / G33** — Squash-merge defeating the trailer-survival contract; merge commits bypassing the untrailered-entity audit; `aiwf doctor --self-check` not exercising the manual-commit-recovery path. (`ad1175c`)
- **G34** — Mutating verbs no longer sweep pre-staged unrelated changes into their commit (uses a stash for isolation). (`890ab01`)

## [0.2.1] — 2026-05-03

### Fixed
- `aiwf check` provenance UX: per-entity findings, defined audit scope, `--since` flag (#5). (`c5b6ab7`)

## [0.2.0] — 2026-05-03

### Added
- **HTML render (Iteration I3)** — `aiwf render --format=html` produces a static site with full per-kind templates (epic, milestone, gap, ADR, decision, contract), a Linear-leaning palette, dark mode, content-hashed CSS for cache-busting, sidebar with logo and wordmark, left-side navigation panel, and a render report under `aiwf doctor`. Playwright browser tests cover the end-to-end render. (`6730c1a`, `e3977ad`, `cce0c21`, `9b88108`, `d9183d8`, `606bfab`)
- **`aiwf-tests` commit trailer** — opt-in TDD enforcement for milestones; new `acs-tdd-tests-missing` warning. (`d7fd072`, `77ccfb1`)
- JSON-completeness on `aiwf show`. (`7fd6524`)
- Testing rules in `tools/CLAUDE.md`: substring-vs-structural assertions, human verification, untested-branches policy, no-workaround rule. (`a30c509`)

### Changed
- `aiwf init` / `aiwf update` reconcile `.gitignore` for the html `out_dir`. (`056139d`)
- Release-prep polish: legacy actor stripped on update; `aiwf render --help`; aiwf-render skill; menu reordering; README clarification. (`406ac48`)
- Integrated `aiwf status` as a rendered HTML page. (`44ea40b`)

### Fixed
- **G27 / G28 / G29** — Test-the-seam, contract-driven, and spec-sourced testing gaps closed; binary-level integration tests retrofitted across version-related verbs. (`f810a86`)
- I3 audit follow-up: three render bugs + structural test additions. (`f71949f`)

## [0.1.1] — 2026-05-03

### Fixed
- `aiwf version` verb now reflects `version.Latest()` proxy resolution — the v0.1.0 seam regression that motivated G27/G28/G29. (`32672cd`)

## [0.1.0] — 2026-05-03

Initial PoC release. Six entity kinds; stable ids that survive rename, cancel, and collision; `aiwf check` as the validation chokepoint; marker-managed framework artifacts; structured commit trailers; principal × agent × scope provenance.

### Added
- **`aiwf upgrade` verb** — one-command flow with skew detection, install via `go install`, and re-exec into `aiwf update`. Includes `--check`, `--version=`, and friendly messages on missing `go` or proxy-disabled. Wired into `aiwf doctor --self-check`. (`3e2d7ff`, `d1c4b1c`, `6136754`, `efa59c2`)
- **`version` package** — `Current()`, `Compare()`, `Latest()` against the Go module proxy. (`62928e5`, `05dd773`)
- **G1 – G26** resolved across iterations I0–I2. See [`docs/pocv3/gaps.md`](docs/pocv3/gaps.md) for the full matrix.
