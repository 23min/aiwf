# Changelog

All notable changes to `aiwf` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project follows [Semantic Versioning](https://semver.org/).

Releases ship as git tags on `poc/aiwf-v3`. The Go module proxy
resolves them when a consumer runs `aiwf upgrade` or
`go install <pkg>@latest`. The branch is not planned to merge to
`main`.

When cutting a release, see [`CLAUDE.md`](CLAUDE.md) § *Go conventions §
Release process*. The tag-push CI check at
[`.github/workflows/changelog-check.yml`](.github/workflows/changelog-check.yml)
fails any pushed `v*` tag that does not have a matching `## [X.Y.Z]`
section in this file.

## [Unreleased]

### Added — E-14: Cobra and completion

Eight milestones (M-049 through M-055 plus M-061) replace the stdlib-`flag` dispatch with [`github.com/spf13/cobra`](https://github.com/spf13/cobra), ship `aiwf completion bash|zsh|fish|powershell`, and wire shell tab-completion for every value-taking flag and id positional. New CLAUDE.md principle: **CLI surfaces must be auto-completion-friendly** — mechanically enforced by a drift test that fails CI when a flag lands without completion wiring or an opt-out entry.

- **M-049 — Bootstrap Cobra dispatch + migrate `version`.** `cmd/aiwf/main.go` becomes a Cobra root command tree; `version` is the first natively-Cobra subcommand. Other verbs continue through a `newPassthroughCmd` adapter until their own milestone migrates them. Exit codes 0/1/2/3 are preserved across Cobra's `Execute` boundary via a typed `*exitError` shuttle.
- **M-050 — Read-only verbs migrated.** `check`, `history`, `doctor`, `schema`, `template`, `render` are now native Cobra commands. `render` keeps its dual surface (`render roadmap` subcommand vs `render --format=html` parent flag) so the public CLI shape doesn't break consumer scripts.
- **M-051 — Mutating verbs migrated.** `add` (with `add ac` as a Cobra subcommand sharing PersistentFlags), `promote`, `cancel`, `rename`, `edit-body`, `move`, `reallocate`, `import`. Single-commit-per-verb invariant preserved; trailer keys (`aiwf-verb` / `aiwf-entity` / `aiwf-actor` plus the I2.5 provenance set) byte-identical; repo-lock contract unchanged.
- **M-052 — Setup verbs migrated.** `init`, `update`, `upgrade`. Marker-based artifact regeneration (skills under `.claude/skills/aiwf-*`, hooks under `.git/hooks/`) goes through the same `initrepo.Init` and `initrepo.RefreshArtifacts` entry points as before, so installed hooks stay byte-identical.
- **M-053 — `aiwf completion <bash|zsh|fish|powershell>` + static completion.** Cobra's auto-generated completion verb emits sourceable shell scripts; the README install one-liner is `source <(aiwf completion zsh)`. Static `--format=text|json` completion across read-only verbs; kind enumeration on `aiwf add`, `aiwf schema`, `aiwf template`; per-kind status enumeration on `aiwf promote <id> <TAB>` derived from the id's prefix without loading the tree; closed-set `--phase` (red|green|refactor|done) and `--on-collision` (fail|skip|update); `add ac M-NNN` filtered to milestone ids.
- **M-054 — Dynamic id completion + drift-prevention test.** `--epic`, `--discovered-in`, `--relates-to`, `--linked-adr`, `--by`, `--superseded-by` enumerate live entity ids; positionals on `history`, `promote`, `cancel`, `rename`, `edit-body`, `move`, `reallocate` likewise. Failures (no `aiwf.yaml`, malformed tree) collapse to an empty list rather than spamming the shell. The drift test in `cmd/aiwf/completion_drift_test.go` walks the Cobra tree and asserts every value-taking flag has either a completion function bound or an entry in the curated opt-out list with a one-line rationale.
- **M-055 — Documentation pass.** Every migrated verb's `--help` now carries a Cobra `Examples:` block with concrete copy-pastable invocations. CLAUDE.md § Go conventions § CLI conventions names Cobra as the standard CLI library and points to the `runXCmd` helper / drift-test chokepoint pattern.
- **Follow-up cleanup commit — `show`, `status`, `whoami`, `authorize` migrated** (the four verbs the epic Scope didn't enumerate). Each gains native Cobra dispatch + completion wiring; the drift test's opt-out list shrinks accordingly.
- **M-061 — Contract family migration + help-recursion regression test.** `aiwf contract` becomes a native Cobra command tree: `verify`, `bind <C-id>`, `unbind <C-id>`, `recipes`, plus the `recipe` sub-tree (`show`, `install`, `remove`). `--validator` completes from declared validators in `aiwf.yaml`; `recipe show|install` from the embedded recipe set; `recipe remove` from declared validators; `bind`/`unbind` positionals from contract entity ids. Subprocess integration test exercises every subcommand against the migrated binary. Bundled into this milestone: a regression test pinning the SetHelpFunc inheritance fix (a `c.Help()` re-entry that recursed to stack overflow on `aiwf <subverb> --help` until M-053 — the test caught a still-live instance of the same bug on `render roadmap --help` during authoring), and this very `[Unreleased]` retrofill.

User-visible behavior delta: `--help` output for every verb now uses Cobra's standard rendering (Long description + Examples + Usage + Flags blocks), replacing the hand-rolled per-verb usage strings. Exit codes, JSON envelope shape, single-commit-per-verb, and trailer-key behavior are byte-identical to pre-migration.

Auto-completion install (one line in shell rc):

```bash
# zsh
source <(aiwf completion zsh)

# bash (requires bash-completion v2)
source <(aiwf completion bash)
```

After sourcing, `aiwf <TAB>` lists the verb catalog; `aiwf promote E-01 <TAB>` lists the kind's allowed statuses; `aiwf check --format=<TAB>` lists `text|json`; `aiwf add milestone --epic <TAB>` enumerates live epic ids from the planning tree.

## [0.6.0] — 2026-05-06

### Added — E-15: Reduce planning-verb commit cardinality

Five milestones cumulatively cut a planning session's commit count by ~75% and remove the last skill/check policy contradictions around body editing. Closes **G-051**, **G-052**, **G-053**, and **G-054**.

- **M-056 — `--body-file` flag on `aiwf add` for all six kinds.** Pass `aiwf add <kind> --title "..." --body-file <path>` (or `--body-file -` for stdin) to land body prose in the same atomic commit as the new frontmatter, replacing the per-kind default template. Eliminates the create-then-hand-edit pattern that today triggers `provenance-untrailered-entity-commit` warnings on the body-edit commit. The file must contain body content only — leading `---` (frontmatter delimiter) is refused so the create commit can't accidentally produce a double-frontmatter file.
- **M-057 — Batched `--title` on `aiwf add ac`.** `aiwf add ac M-NNN --title "..." --title "..." --title "..."` now creates N acceptance criteria in one atomic commit instead of one commit per AC. The commit emits one `aiwf-entity:` trailer per created composite id (allocation order), so `aiwf history M-NNN/AC-X` finds the batch commit for any AC in the batch. Whole-batch validation: if any title is empty or prosey, the entire batch aborts before disk work. `--tests` is rejected when N>1 (a single test-metrics value can't apply unambiguously to multiple ACs). Single-`--title` invocation continues to work unchanged — same subject shape, same single trailer.
- **M-058 — `aiwf edit-body` verb + skill reconciliation.** New verb `aiwf edit-body <id> --body-file <path>` (or `--body-file -` for stdin) replaces the markdown body of an existing entity in a single atomic trailered commit. Frontmatter is left untouched — that stays the domain of `promote` / `rename` / `cancel` / `reallocate`. Refuses leading `---` content and composite ids (AC body sub-section editing deferred). The aiwf-add skill no longer carves out plain-git body edits; every entity-file mutation now goes through a verb route, and `provenance-untrailered-entity-commit` only fires on accidental hand-edits as designed.
- **M-059 — Resolver-pointer flags on `aiwf promote`.** New `--by`, `--by-commit`, and `--superseded-by` flags write the matching frontmatter field atomically with the status change, so `aiwf promote G-NNN addressed --by M-007` (or `--by-commit <sha>`) and `aiwf promote ADR-NNNN superseded --superseded-by ADR-MMMM` no longer require a follow-up hand-edit to satisfy the `gap-resolved-has-resolver` and `adr-supersession-mutual` checks. Flags reject mismatched kind/target-status combinations and are mutex with `--audit-only` (which is empty-diff by definition).
- **M-060 — Bless-current-edits mode for `aiwf edit-body`.** Running `aiwf edit-body <id>` with no `--body-file` flag now commits whatever the user has edited in the working copy of the entity file, with the standard `edit-body` trailer set. Refuses cleanly when there is no diff, when the diff includes frontmatter changes (pointer at `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate`), or when the file has no HEAD version (pointer at `aiwf add`). The existing `--body-file <path>` (and stdin) mode from M-058 stays exactly the same — same trailers, same atomicity, same rules; only difference is what happens when the flag is absent. AC body sub-section edits work for free via bless mode on the parent milestone — no composite-id resolver needed.

## [0.5.2] — 2026-05-06

### Fixed
- **G50 — Pre-commit hook tolerant of gitignored `STATUS.md`.** The hook regenerates `STATUS.md` and stages it via `git add`; when the consumer has gitignored the file (legitimate — it's regenerated every commit), `git add` exited non-zero and `set -e` aborted the entire hook, in violation of the hook header's "tolerant by design — never blocks commits" promise. The aborted commit also commonly orphaned `.git/index.lock`, masquerading as an aiwf bug on subsequent verbs. Fix: append `2>/dev/null || true` to the `git add` invocation. Consumer migration: run `aiwf update` after upgrading. (`572bc96`)
- **G48 — `aiwf init`, `aiwf update`, and `aiwf doctor` honor `core.hooksPath`.** A consumer who has set git's `core.hooksPath` (a tracked-hooks pattern via husky/lefthook or a home-grown convention) previously got hooks installed at `.git/hooks/` regardless — git's hook lookup missed them and the validation chokepoint silently disappeared. New `gitops.HooksDir` helper resolves the effective hooks directory once; init, update, and doctor read through it. Reports (`StepResult.What`, doctor lines, migration messages) reflect the actual install path. Consumer migration: run `aiwf update` after upgrading. (`6432f0f`)

## [0.5.1] — 2026-05-06

### Added
- **G49 — `addressed_by_commit:` field on the gap kind.** Optional multi-string list of commit SHAs that resolved a gap. The `gap-resolved-has-resolver` rule now passes when *either* `addressed_by:` (entity refs) or `addressed_by_commit:` (commit SHAs) is non-empty. Models the real semantic: a gap can be closed by a specific commit, not just a milestone. (`de39e01`)

### Internal
- **G38 — Kernel now dogfoods aiwf against itself.** The kernel repo became an aiwf consumer: 47 legacy gaps imported as `G-NNN` entities, 13 epics + 48 milestones imported from the historical PoC plan, `gaps.md` and `poc-plan.md` archived under `docs/pocv3/archive/`. The hook reshape (drop `core.hooksPath`, slim the kernel's policy-lint hook to `.git/hooks/pre-commit.local` via the G45 chain) treats the kernel like any consumer with pre-existing hooks. End-to-end exercise of `aiwf init`, `aiwf import`, `aiwf add gap`, `aiwf promote`, `aiwf promote --audit-only`, and the pre-commit/pre-push hook chains against real consumer-shaped state. New gaps surfaced and filed via the framework: G48 (`aiwf init` should honor `core.hooksPath`), G49 (above, fixed in this release).

## [0.5.0] — 2026-05-05

### Added
- **G46 — Structured remediation on `go install` package-path failures.** When a release relocates the cmd package within the module (as v0.4.0 did), `aiwf upgrade` now detects the Go toolchain's "module found but does not contain package" stderr and prints a hint pointing at the CHANGELOG plus the manual `go install <new-path>@<target>` recovery command. The v0.3.x → v0.4.0 transition pain that surfaced this gap is the canonical example; v0.5.0+ binaries handle the next path-change gracefully. (`93a3e2b`)

### Changed (breaking)
- **G47 — `aiwf_version` field retired from `aiwf.yaml`.** The field was a set-once pin that produced chronic doctor noise without serving its intended purpose. Removed via the same pattern as the I2.5 legacy-actor-strip: loader becomes tolerant (no longer required), `aiwf init` no longer writes it, `aiwf update` strips it on every refresh, doctor's `pin:` row goes away. Pre-G47 yamls load fine and surface a one-line deprecation note until `aiwf update` cleans them. The running binary's version is now the authoritative answer (`aiwf version`); newer-release detection lives at `aiwf doctor --check-latest`. (`25bf5ea`)

## [0.4.0] — 2026-05-05

### Changed (breaking)
- **Repo reorg to Go-standard layout.** `tools/cmd/aiwf` → `cmd/aiwf`, `tools/internal/...` → `internal/...`, `tools/e2e` → `e2e`. Module path unchanged at `github.com/23min/ai-workflow-v2`. **Install path changed:** `go install github.com/23min/ai-workflow-v2/cmd/aiwf@latest` (previously `.../tools/cmd/aiwf@latest`). Existing tags resolve via the old path; new releases via the new path. `tools/CLAUDE.md` merged into root `CLAUDE.md` as a new "Go conventions" section so discovery semantics work post-reorg. (`a137132`)

### Added
- **G45 — Hook chaining via `.local` siblings.** aiwf-managed `pre-push` and `pre-commit` hooks now invoke `<hook-name>.local` (if present and executable) before running aiwf's own check. `aiwf init` auto-migrates a pre-existing non-marker hook to its `.local` sibling, preserving content byte-for-byte and exec bit. Non-executable `.local` fails loud (both at hook runtime and via `aiwf doctor`). New `ActionMigrated` step result; `HookConflict` now signals only the rare `.local`-already-exists collision. `aiwf doctor` reports chain shape per hook. Unblocks consumers with pre-existing hooks (husky, lefthook, hand-written) from `aiwf init` friction. (`49e7764`)

## [0.3.0] — 2026-05-05

### Added
- **G37** — Cross-branch id collisions are detectable and resolvable. Trunk-aware allocator + `prior_ids` lineage; `aiwf history` walks renamed entities through their full chain; `aiwf check` flags cross-tree collisions; `aiwf reallocate` handles the trunk-ancestry tiebreaker. (`271f514`, `b9d73d8`, `c5a98c1`, `a6e8067`, `685f288`)
- Three new policies + 14 backfilled subcode docs to broaden discoverability-lint coverage. (`2b094e3`)
- **G38** — Dogfooding gap filed: investigation of running aiwf against its own kernel repo. Open. (`dd25c06`)
- **G40** — `aiwf check` now reports `unexpected-tree-file` for stray files under `work/`. Tree-shape changes go through verbs; body-prose edits to existing entity files remain free-form. Configurable via `aiwf.yaml: tree.allow_paths` (glob exemptions) and `tree.strict: true` (promote warning to error). Files inside contract directories are auto-exempt. New design doc `docs/pocv3/design/tree-discipline.md`; rule folded into `aiwf-add` and `aiwf-check` skills (no new skill). (`bdd43c2`)
- **G41** — Tree-discipline now runs at pre-commit *and* pre-push. New `aiwf check --shape-only` flag runs only the tree-discipline rule (no trunk read, no provenance walk, no contract validation), wired into the aiwf-managed pre-commit hook. The LLM gets an in-loop signal when a stray file lands, regardless of which AI client it's using — git hooks are agent-agnostic. (`fb2e1e4`)
- **G42** — Pre-commit hook responsibilities decoupled. The tree-discipline gate now installs unconditionally when aiwf is adopted in the repo; `aiwf.yaml: status_md.auto_update` controls only whether the script body includes the STATUS.md regen step. Opting out of the regen no longer removes the gate. `aiwf doctor`'s pre-commit reporting updated with new "ok, gate-only" healthy state.

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
