# Changelog

All notable changes to `aiwf` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project follows [Semantic Versioning](https://semver.org/).

Releases ship as git tags on `main`. The Go module proxy resolves
them when a consumer runs `aiwf upgrade` or
`go install <pkg>@latest`.

When cutting a release, see [`CLAUDE.md`](CLAUDE.md) § *Go conventions §
Release process*. The tag-push CI check at
[`.github/workflows/changelog-check.yml`](.github/workflows/changelog-check.yml)
fails any pushed `v*` tag that does not have a matching `## [X.Y.Z]`
section in this file.

## [Unreleased]

### Changed — E-0027: Trailered merge commits from `aiwfx-wrap-epic` (closes G-0100)

The rituals plugin's `aiwfx-wrap-epic` skill now prescribes a *trailered* merge commit for the integration-target merge: `git merge --no-ff --no-commit <epic-branch>` followed by `git commit --trailer "aiwf-verb: wrap-epic" --trailer "aiwf-entity: E-NNNN" --trailer "aiwf-actor: human/<id>"`. Without `--no-commit`, git produces an untrailered merge commit and the kernel's existing `provenance-untrailered-entity-commit` finding fires once per entity file the merge touched (historical instances on E-0024 and E-0026 wrap commits remain as accepted artefacts; no history rewrite). The change is fixture-first per CLAUDE.md *Cross-repo plugin testing* — authoring at `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md`, structural drift-check tests in `internal/policies/aiwfx_wrap_epic_test.go`, copy-to-rituals-repo at wrap. No kernel rule changes; the chokepoint stays strict.

- **M-0090 — `aiwfx-wrap-epic` emits trailered merge commits; fixture + drift-check tests.** Fixture body rewrites step 5 of the wrap-epic workflow (and tightens step 8's wrap-artefact commit to carry the same trailers). Six AC tests pin: frontmatter shape, trailered-sequence substring in the merge-step section, structural section-scoping per CLAUDE.md *Substring assertions are not structural assertions*, cache-vs-fixture parity against the active install resolved from `installed_plugins.json`, post-wrap rituals-repo SHA recording, and kernel-rule unchanged. Rituals-repo copy committed at `3faae39`. This epic's own merge commit is the dogfood — the first trailered-merge wrap under the new ritual.

### Changed — E-0026: `aiwf check` per-code summary by default (closes G-0098)

Default text output of `aiwf check` collapses warnings to one line per finding-code: `<code> (warning) × N — <representative message>`. Errors continue to print per-instance — each error is per-instance-actionable. A new `--verbose` flag restores the full pre-epic per-instance shape byte-for-byte. The JSON envelope is unchanged modulo `metadata.root` (which is environmental); machines still receive every finding via `--format=json` regardless of `--verbose`. On the kernel tree the post-E-0023 / post-E-0024 advisory state (~176 near-identical `terminal-entity-not-archived` lines + the paired `archive-sweep-pending` aggregate) shrinks from a ~180-line scroll to a 5-line scannable summary. Sort order is count-desc with alphabetic tie-break (pinned so golden files don't drift). No check rules, severities, or finding codes changed.

- **M-0089 — Per-code text-render summary with `--verbose` fallback.** New `render.TextSummary` partitions findings: errors flow through the existing per-instance path, warnings group by `Code` into per-code buckets. `Text` was refactored to share a `renderPerInstance` helper so the verbose path stays byte-identical to the pre-epic behaviour by construction, not just by golden file. Sample message per code is the first finding's `Message` verbatim. Binary integration tests at `cmd/aiwf/check_summary_binary_test.go` (kernel-tree ≤10-line bound, byte-identity against captured baselines for verbose text, structural-equal modulo `metadata.root` for JSON, `--help` documentation of `--verbose`). Discovered the friction post-E-0024 when the advisory paired-finding shape became the new normal; this milestone collapses the noise at the render layer alone.

## [0.7.0] — 2026-05-10

### Changed (breaking) — Module path rename `github.com/23min/ai-workflow-v2` → `github.com/23min/aiwf` (closes G-0094)

The kernel's Go module path now matches its binary name. The GitHub repo is renamed from `23min/ai-workflow-v2` to `23min/aiwf`; the old repo is archived under the same owner and remains accessible as a historical record. Pre-rename tags (`v0.1.x` and earlier) continue to resolve at the old path until the Go module proxy expires them; post-rename releases ship at the new path only.

**Existing installs require manual reinstall.** `aiwf upgrade` from a binary built before this rename will keep querying the old proxy path and silently keep returning pre-rename tags. To migrate:

```bash
go install github.com/23min/aiwf/cmd/aiwf@latest
```

Clones of the repo: `git remote set-url origin git@github.com:23min/aiwf.git` (or HTTPS equivalent). GitHub's auto-redirect handles the URL change for already-cloned trees, but the new origin is the durable one.

### Changed — E-0023: Uniform 4-digit kernel ID width (closes G-0093)

Three milestones (M-0081, M-0082, M-0083) collapse the per-kind id-width policy (E-NN, M-NNN, ADR-NNNN, G-NNN, D-NNN, C-NNN) to a single canonical 4-digit form across every kernel id kind. Parsers tolerate narrower legacy widths on input so existing trees, branches, and commit trailers continue to validate without history rewrite; renderers and allocators always emit canonical width. Files **ADR-0008** (policy precedent for the entire epic; promoted from `proposed` to `accepted` at wrap). Closes **G-0093**.

- **M-0081 — Canonical 4-digit IDs in parser, renderer, and allocator.** Replaces the per-kind `canonicalPad` map with `const entity.CanonicalPad = 4`; consolidates the duplicate `canonicalPadFor` from `internal/verb/import.go` into the entity package. New `entity.Canonicalize(id)` left-pads any recognizable id to canonical width; `entity.IDGrepAlternation(id)` emits a regex matching both widths for `git log --grep` callers. Lookup-seam canonicalization threaded through `Tree.ByID` and friends; display surfaces (`aiwf list / status / show / history / render`) emit canonical ids regardless of on-disk filename. AC-5 mechanical chokepoint at `internal/policies/narrow_id_sweep_test.go` pins the test-fixture-sweep discipline.
- **M-0082 — `aiwf rewidth` verb + apply to this repo's tree.** Net-new top-level Cobra verb canonicalizes a consumer's narrow-width tree to 4-digit form: file renames per kind (active tree only; archives preserved per ADR-0004 forget-by-default) plus body-content reference rewrites (bare ids, composite ids, markdown links — code fences, inline backticks, URL fragments, archive paths excluded). Idempotent on canonical or empty trees; one commit per `--apply` with multi-entity-sweep trailer (`aiwf-verb: rewidth`, no `aiwf-entity:`). Applied to this repo's tree at wrap (200 file renames + 212 body rewrites in commit `f937288`).
- **M-0083 — Drift check, normative-doc amendments, skill content refresh.** New `entity-id-narrow-width` warning rule fires only on mixed-state active trees (some canonical alongside some narrow), staying silent on uniform-narrow, uniform-canonical, and empty trees — the on-demand-migration framing. ADR-0003 §"Id and storage" amended `F-NNN → F-NNNN` with cross-reference to ADR-0008; CLAUDE.md "What aiwf commits to" §2 collapses to a single uniform rule. Doc-tree narrow-id sweep over `docs/pocv3/design/` and `docs/pocv3/plans/` (~70 references) with allowlist for foreign-project / illustrative / historical mentions. Cross-repo skill refresh: kernel-embedded fixture at `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` matches the rituals plugin commit `808ad70bb368c7d687a207cc7b749e0b11529323`.

A follow-up `wf-patch` (`fix/rewidth-preflight-checks`) hardens M-0082's verb with a default-on preflight: `aiwf rewidth --apply` runs `aiwf check` first and refuses to proceed on error-severity findings (`--skip-checks` opts out); missing expected kind directories emit advisory warnings; all-missing bails with `exitUsage`. Discovered during epic dogfooding when the first `--apply` attempt revealed the absence of any built-in readiness check beyond dry-run.

### Changed — Trunk promotion (poc/aiwf-v3 → main)

The `poc/aiwf-v3` branch graduates to `main` on 2026-05-08. After this point, the engine, planning state, and design research all live in a single trunk. Future releases tag on `main` (next: v0.7.0); the `poc/aiwf-v3` branch is preserved as a frozen historical reference for the PoC's full git history.

The merge brings the design research previously isolated on `main` into the same trunk as the implementation:

- **Research arc** at `docs/research/` — `KERNEL.md`, `0-introduction.md`, and thirteen numbered docs (`00`–`13`) tracing the project from the original event-sourced ambition through to policies-as-primitive.
- **Working paper** at `docs/working-paper.md` — thesis-style synthesis; one-document entry point for visitors.
- **Explorations** at `docs/explorations/` — `01`–`05` on policies as a primitive, plus mining-corpus surveys at `surveys/{flowtime,liminara}/` and landscape surveys at `docs/research/surveys/`.
- **Pre-PoC design archive** at `docs/archive/` — original `architecture.md` and `build-plan.md` preserved for archaeology.
- **Repo conventions** carried from main: `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md`, `.github/workflows/pr-conventions.yml`. The trunk-based development model in `CLAUDE.md` is the primary workflow; PR ceremony applies to external contributions. `CONTRIBUTING.md` awaits a reconciliation pass.

### Added — E-21: Open-work synthesis: aiwfx-whiteboard skill replaces critical-path.md

Three milestones (M-078, M-079, M-080) graduate the open-work synthesis pattern from a one-off `critical-path.md` snapshot into the reproducible `aiwfx-whiteboard` skill. The skill ships in the `ai-workflow-rituals` plugin, materialises into the marketplace cache via `aiwf init` / `aiwf update`, and answers natural-language direction questions (*"what should I work on next?"*, *"draw the whiteboard"*, *"where should we focus?"*) by synthesising tree state into a tiered landscape, recommended sequence, first-decision fork, and Q&A-gated pending decisions. Files **ADR-0007** (placement, tiering, name rationale; status `proposed`). Adds two CLAUDE.md doctrines mid-flight: *AC promotion requires mechanical evidence* (born from M-078's "wrapped without tests" episode) and *Cross-repo plugin testing* (fixture-first authoring at `internal/policies/testdata/<skill>/SKILL.md`; deploy to rituals repo at wrap; drift-check test against marketplace cache). Retires `work/epics/critical-path.md` and the standing `unexpected-tree-file` warning it produced.

- **M-078 — Planning-conversation skills design ADR.** ADR-0007 captures placement (rituals plugin, not kernel), tiering (pure-skill first; kernel verb only when usage demands it), and name rationale (`aiwfx-whiteboard`; the deferred kernel-verb backing is `aiwf whiteboard`, not `aiwf landscape`).
- **M-079 — `aiwfx-whiteboard` skill: classification rubric, output template, Q&A gate.** Skill body covers input shape (which read verbs to call and in what order), tier classification rubric, output template, Q&A flow, and anti-patterns. Output ordering: action-shaped blocks (sequence / fork / pending) lead the rendered output; tiered landscape moves to last as supporting reference.
- **M-080 — Whiteboard skill fixture validation; retire `critical-path.md`; close E-21.** Fixture at `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` is the canonical authoring location; `TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent` re-applies the kernel skill-coverage invariants to the plugin skill (kernel policy walks `internal/skills/embedded/` only — see G-088), and `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` guards against fixture/cache drift.

Three follow-up gaps carry loose ends forward: **G-088** (kernel skill-coverage policy doesn't police plugin skills), **G-089** (whiteboard gitignored cache anti-pattern revision — *addressed* mid-epic), **G-090** (drift-check test has three branches not unit-tested; refactor lookup to take cache root as parameter for hermetic testing).

### Added — E-22: Planning toolchain fixes (closes G-071, G-072, G-065)

Three milestones (M-075, M-076, M-077) close the Tier-1 planning-toolchain frictions whose cumulative tax on every multi-milestone planning session justified bundling them. Closes **G-071**, **G-072**, and **G-065**.

- **M-075 — Lifecycle-gate the `entity-body-empty` rule (closes G-071).** The check now skips pre-implementation drafts and post-terminal artifacts: per-kind status gating exempts entities still being scoped (e.g. `draft` epics) and entities past their useful life (e.g. `done` milestones, `cancelled` gaps). Drops the kernel repo's warning baseline from 46 to ~1. Adds the shared `entity.IsTerminal(kind, status)` helper used by E-20/M-072 as well.
- **M-076 — Writer surface for milestone `depends_on` (closes G-072).** Closes the kernel asymmetry where `depends_on` had six read sites and zero writers. New `--depends-on` flag on `aiwf add milestone` lands the field at create time; new dedicated verb `aiwf milestone depends-on M-NNN --on M-AAA --on M-BBB` (with `--clear` to drop the field) edits it post-creation. Both produce trailered atomic commits with referent validation. Verb shape is forward-compatible with G-073's eventual cross-kind generalisation. **G-079** filed for `aiwfx-plan-milestones` plugin-skill documentation update upstream.
- **M-077 — `aiwf retitle` verb for entities and ACs (closes G-065).** New verb `aiwf retitle <id> --to "<new title>"` corrects frontmatter `title:` for top-level kinds and composite-id ACs in a single trailered commit, so scope refactors no longer leave `title:` permanently misleading. Ships with a dedicated `aiwf-retitle` embedded skill plus a redirect from `aiwf-rename` (slug rename remains the rename verb's job). README.md updated to add the two new verbs and refresh the materialized-skills count to twelve.

### Added — E-20: Add list verb (closes G-061)

Three milestones (M-072, M-073, M-074) ship `aiwf list` as the AI's hot-path read primitive over the planning tree, route AI discovery to it via a split-skill design that demotes `aiwf status` to its real role (human-curated narrative), and lock the discoverability surface against drift via a kernel policy. Closes **G-061** and **G-085**. Files **ADR-0006** (skills policy: per-verb default, topical multi-verb when concept-shaped, no skill when `--help` suffices, discoverability priority justifies splitting). Adds CLAUDE.md *Skills policy* section.

- **M-072 — `aiwf list` verb + status filter-helper refactor + contract-skill drift fix.** New verb `aiwf list` with V1 flags `--kind`, `--status`, `--parent`, `--archived`, `--format=text|json`, `--pretty`. Default behavior: filter out terminal-status entities (forward-compatible with ADR-0004's archive convention). Default sort: id ascending. No-args invocation prints per-kind counts. `--format=json` emits the standard envelope with `result` as an array of `{id, kind, status, title, parent, path}` summaries. Closed-set completion wired for `--kind` and `--status` (kind-aware). Shared filter helper `tree.FilterByKindStatuses` extracted from `cmd/aiwf/status_cmd.go` so list and the *Open gaps* / *In-flight* / *Open-decisions* slices of `aiwf status` cannot drift. New `entity.IsTerminal(kind, status)` helper. Seven `aiwf list contracts` references in `docs/pocv3/plans/contracts-plan.md` and the `aiwf-contract` skill swept to `aiwf list --kind contract`; `TestNoReintroducedDeadVerbForms_ContractsAndSkill` is the future-drift guard.
- **M-073 — `aiwf-list` skill + `aiwf-status` skill tightening.** New embedded skill `internal/skills/embedded/aiwf-list/SKILL.md` with description densely populated by list-shaped natural-language phrasings; body covers filter recipes, output shape, JSON envelope, and when-to-use-list-vs-status criteria. The `aiwf-status` skill body opens with a bold-paragraph redirect to `aiwf list` for tree queries; description tightened to narrative-snapshot phrasings only. Both skills materialize via `aiwf init` and `aiwf update`.
- **M-074 — Skill-coverage policy + judgment ADR + CLAUDE.md skills section.** New kernel policy `internal/policies/skill_coverage.go` asserts every embedded skill has non-empty `name:`/`description:` frontmatter, skill `name:` matches its directory and the `aiwf-<topic>` convention, every top-level Cobra command is documented by some embedded skill or appears in the allowlist with one-line rationale, and every backticked `aiwf <verb>` mention inside a skill body resolves to a real registered verb. Subsumes G-061's `skill-references-unknown-verb` follow-up suggestion. ADR-0006 captures the judgment rule; CLAUDE.md *Skills policy* section points at the ADR (judgment) and the policy (mechanical companion); the *What's enforced and where* table gains a row pinning the policy to its CI test chokepoint. **G-087** filed as the deferred `aiwf-show` skill follow-up referenced by the allowlist's `show` entry rationale.

### Added — E-18: Operator-side dogfooding completion (closes G-062, G-064)

Two milestones (M-070, M-071) close the operator-side gap that G-038 left partial: the kernel repo's design assumed `aiwf-extensions` and `wf-rituals` were present, but neither plugin was installed for this project's scope, and there was no kernel mechanism to detect that. Closes **G-062** and **G-064**.

- **M-070 — `aiwf doctor` warning for missing recommended plugins.** Config-driven `doctor.recommended_plugins` check warns once per declared-but-missing plugin against `~/.claude/plugins/installed_plugins.json`. Generic mechanism — any consumer can declare expected plugins in `aiwf.yaml` and get the warning when state doesn't match.
- **M-071 — Install ritual plugins in kernel repo + document operator setup.** This repo declares both plugins in `aiwf.yaml`, project-scope-installs them, and CLAUDE.md gains an *Operator setup* section documenting the install path — including the project-scope-vs-user-scope nuance discovered during the install (the CLI form `claude /plugin install <name>@<marketplace>` defaults to user scope; only the interactive `/plugin` menu offers a project-scope choice).

Two deferrals filed as gaps: **G-069** (`aiwf init`'s `printRitualsSuggestion` hardcodes the user-scope CLI form) and **G-070** (`aiwf doctor` has no `--format=json` envelope).

### Added — E-17: Entity body prose chokepoint (closes G-058)

Three milestones (M-066, M-067, M-068) make non-empty body prose a kernel-enforced property across every entity kind, not just ACs. Closes **G-058**. Captures **D-001** (asymmetric semantics: top-level sections count sub-headings as content; AC bodies require non-heading prose).

- **M-066 — `aiwf check` finding `entity-body-empty`.** New rule lights up empty load-bearing body sections per kind: warning by default, error under `aiwf.yaml: tdd.strict: true`. Per-kind body-section dispatch with the asymmetric semantics call codified in D-001 — top-level sections count sub-headings as content, AC bodies require leaf prose.
- **M-067 — `aiwf add ac --body-file` for in-verb body scaffolding.** `aiwf add ac M-NNN` now accepts `--body-file <path>` (positional pairing for batched ACs, stdin shorthand for single ACs); leading `---` is refused across both forms. AC body content lands in the same atomic create commit as the heading, completing the body-file surface across every `aiwf add` shape.
- **M-068 — `aiwf-add` skill names "fill in the body" as required next step.** The embedded skill cross-references M-066 and M-067 so operators reading the skill alone get the full non-empty-body picture — rule, verb, workflow — without grepping source.

Two follow-up gaps survive the wrap: **G-067** (wf-tdd-cycle is LLM-honor-system advisory under load) and **G-068** (discoverability policy misses dynamic finding subcodes).

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
- **Repo reorg to Go-standard layout.** `tools/cmd/aiwf` → `cmd/aiwf`, `tools/internal/...` → `internal/...`, `tools/e2e` → `e2e`. Module path unchanged at `github.com/23min/ai-workflow-v2` *(at the time of this release; subsequently renamed — see Unreleased)*. **Install path changed:** `go install github.com/23min/ai-workflow-v2/cmd/aiwf@latest` (previously `.../tools/cmd/aiwf@latest`). Existing tags resolve via the old path; new releases via the new path. `tools/CLAUDE.md` merged into root `CLAUDE.md` as a new "Go conventions" section so discovery semantics work post-reorg. (`a137132`)

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
- **G1 – G26** resolved across iterations I0–I2. See [`docs/pocv3/archive/gaps-pre-migration.md`](docs/pocv3/archive/gaps-pre-migration.md) for the full matrix.
