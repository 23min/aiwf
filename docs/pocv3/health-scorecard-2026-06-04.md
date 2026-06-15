# aiwf Codebase Health Scorecard

**Date:** 2026-06-04
**Branch:** epic/E-0030-branch-model-chokepoint
**Rubric:** "Principles for a healthy codebase" (A1–G3, 24 principles)

## Summary

aiwf is an exceptionally healthy codebase. Across 24 principles, 21 came back Strong, only 3 came back Weak (C3 Atomic writes, D2 Equivalence tests at seams, D3 Branch coverage on touched code, E1 Structured logs), and none are Missing. The kernel's stated commitment that "framework correctness must not depend on LLM behavior" is visibly load-bearing: a 124-file `internal/policies/` Go test package mechanizes most invariants the rubric cares about (verb-validate-then-write, FSM shape, principal-write guards, trailer-keys-via-constants, no-history-rewrites, no-retry-loops-on-git, no-timestamp-manipulation, race-parallel-cap, setup_test.go presence, finding-codes-have-tests/hints, hardcoded-entity-paths). The four Weak verdicts concentrate around enforcement gaps for properties the design already articulates but the chokepoints don't yet pin.

### Verdict counts
| Verdict   | Count |
| :-------- | ----: |
| Strong    | 20 |
| Weak      | 4 |
| Missing   | 0 |
| N/A       | 0 |

### At-a-glance table

| Code | Principle | Verdict |
| :--- | :-------- | :------ |
| A1 | High cohesion | Strong |
| A2 | Low coupling | Strong |
| A3 | Layered (no upward dependencies) | Strong |
| B1 | Typed interfaces | Strong |
| B2 | Schemas at boundaries | Strong |
| B3 | Pre/post conditions and invariants | Strong |
| C1 | Single source of truth | Strong |
| C2 | Idempotence | Strong |
| C3 | Atomic writes | Weak |
| C4 | Versioned schemas with migration paths | Strong |
| D1 | Behavior pinned, not implementation | Strong |
| D2 | Equivalence tests at seams | Weak |
| D3 | Branch coverage on touched code | Weak |
| D4 | Tests at the right altitude | Strong |
| E1 | Structured logs | Weak |
| E2 | Designed failure modes | Strong |
| E3 | Audit trail | Strong |
| E4 | Self-explaining errors | Strong |
| F1 | Names that don't lie | Strong |
| F2 | Comments only for non-obvious "why" | Strong |
| F3 | Decision records that survive turnover | Strong |
| G1 | Reproducible | Strong |
| G2 | Reversible | Strong |
| G3 | Observable in production | Strong |

## Priority actions

1. Wire a diff-coverage gate into CI (e.g. `go-test-coverage` or `diff-cover --fail-under=80 --compare-branch=origin/main coverage.out`) and add an `internal/policies/branch_coverage_audit_test.go` that asserts every conditional branch in changed code is either covered or annotated `//coverage:ignore <reason>` — converts the `wf-tdd-cycle` skill's "HARD RULE" from honor-system into a mechanical chokepoint and closes the open exposure tracked at G-0067. [D3]
2. Promote the temp+rename pattern from `internal/aiwfyaml/aiwfyaml.go:187-197` and `internal/skills/skills.go:496-512` into a shared `pathutil.AtomicWriteFile(path, data, perm)` helper (write→`f.Sync()`→`os.Rename`), and route the central writer at `internal/verb/apply.go:140`, the three config rewrites at `internal/config/config.go:340/397/463`, the settings-json overwrite at `internal/skills/settings.go:95`, and the htmlrender per-file emit through it; coordinate the SKILL.md-loop + manifest pair via a staging dir + single `os.Rename` swap. [C3]
3. Add shared conformance suites at the two unsigned-cheque interface seams: a `PageDataResolver` matrix test that drives `defaultResolver` and the production `cli/render.Resolver` against the same fixture tree asserting equivalent `IndexData`/`EpicData`/`MilestoneData`/`EntityData` shapes, and a `BranchOracle` matrix that runs identical scenarios through both `fakeOracle` and `gitBranchOracle`; optionally add a cue↔jsonschema recipe equivalence test (skippable under `-short`) for the "same schema + valid/invalid fixture → same pass/fail" contract. [D2]
4. Either implement the CLAUDE.md `log/slog` prescription (introduce a single `internal/logger` wrapping `log/slog` with a JSON stderr handler; route the interpolated stderr lines in `statusline.go`, `root.go`, `move.go`, `cancel.go`, `upgrade.go` through `logger.Info("statusline.wrote", "path", res.Path)`-shaped calls; ban bare `fmt.Fprint*(os.Stderr, ...)` outside `cmd/aiwf/main.go` and `outputformat.go`'s human-text branch via `forbidigo`) OR amend CLAUDE.md to ratify the actual envelope-only practice and remove the aspirational logger line so design intent and implementation stop disagreeing. [E1]
5. Split `internal/cli/cliutil/` along its own package-doc fault lines into ~4 focused packages (identity, output, gitstate, flagsupport) and adopt an `Options`-struct pattern for the 8–10-positional-param `cli/<verb>/Run(...)` adapters in `list`, `cancel`, `authorize`, `milestone` — the deeper `internal/verb/` layer already uses `PromoteOptions`/`ContractBindOptions`; the discipline is one level deep and the cliutil grab-bag is the only real cohesion smell in the kernel. [A1]

## Detailed findings

## A. Module boundaries

### A1. High cohesion

- **Verdict:** Strong
- **Summary:** Each of the ~30 `internal/` packages has a one-sentence purpose stated in its package doc, and those purposes hold up to inspection. `cmd/aiwf/main.go` is 22 lines and defers everything to `internal/cli`. Large files are large because their single concern is genuinely complex (e.g. `internal/verb/rewidth.go` 946 lines for one width-sweep). The two real smells are bounded to the Cobra-adapter ring: `internal/cli/cliutil/` is an acknowledged 18-file grab-bag (its own package doc names the multi-concern bundling), and several `cli/<verb>/Run(...)` adapters carry 8–10 positional params where the underlying `internal/verb/` layer already uses `Options` structs.
- **Evidence:**
  - supports: `internal/entity/entity.go:1-7` doc explicitly disclaims fs/git/validation knowledge
  - supports: `internal/check/check.go:1-12` "errors are findings, not parse failures" single concern
  - supports: `cmd/aiwf/main.go:19-21` entry-only, 22 lines, defers all concerns
  - supports: `internal/verb/rewidth.go:60-938` 22 functions serving one concern (width-sweep)
  - refutes: `internal/cli/cliutil/exit.go:1-10` package doc acknowledges multi-concern bundling
  - refutes: `internal/cli/list/list.go:Run` 8 positional flag params (pattern repeats in cancel/authorize/milestone)
- **Smells found:**
  - util/cliutil grab-bag package mixing many unrelated concerns (identity, locking, output, completion, platform, statusline)
  - long positional param lists (8–10 params) in several `cli/<verb>/Run(...)` adapters where the deeper `internal/verb/` layer already uses Options structs
- **Recommended moves:**
  - Split `internal/cli/cliutil/` along its own package-doc fault lines into ~4 focused packages (identity, output, gitstate, flagsupport)
  - Adopt an Options-struct pattern at the `cli/<verb>/Run` boundary mirroring `internal/verb`'s PromoteOptions/ContractBindOptions
  - Consider extracting `cli/render/resolver.go` (785 lines, 24 methods on one Resolver type) into per-page sub-files
- **Adversarial note:** Tried to break Strong by hunting for cross-package private-state access, package-level mutable state, fs/git leaks into pure layers, and additional and-also functions. Kernel layers (entity, tree, check, verb, render, gitops, scope) are genuinely cohesive; the two acknowledged smells stay local to the Cobra-adapter ring.

### A2. Low coupling

- **Verdict:** Strong
- **Summary:** The 30-package `internal/` tree is a clean acyclic DAG with monotonic layering: leaf primitives (entity, codes, pathutil, aiwfyaml, version, branchparse — each with 0–1 internal deps) underpin core services (gitops, tree, scope, render — 1–2 deps), policy/business logic (check, verb, workflows/spec — 4–12 deps), and CLI orchestration (cli/cliutil — 14 deps). `go build ./...` is clean: no import cycles. Module seams use named interfaces (`htmlrender.PageDataResolver`, `check.BranchOracle`). No "utils" junk drawer; the most-imported helper `internal/cli/cliutil` is the CLI-orchestration hub never imported by lower layers. Package-level vars are all immutable lookup tables — no mutable global state for tests to mutate. Two narrow documented exceptions (`doctor.Dispatcher` to break a cycle; `reexecUpdate` as a test seam) are scoped and intentional.
- **Evidence:**
  - supports: `internal/codes/codes.go:1-9` "leaf: imports nothing from the module"
  - supports: `internal/htmlrender/htmlrender.go:24-60` defines `PageDataResolver` interface so renderer stays free of git/history walking
  - supports: `internal/check/isolation_escape.go` `BranchOracle` interface — minimal surface
  - supports: `go build ./...` builds cleanly, 249-edge graph is acyclic
  - refutes: `internal/cli/cliutil/` 30+ importers, 14 deps — coordination hub, mitigated by sibling-only usage
  - refutes: `internal/verb/` 12 internal deps (justified — kernel orchestration)
- **Smells found:** none
- **Recommended moves:**
  - If cliutil grows further, split it by concern so consumers don't pull the 14-dep hub for one helper
  - Codify the layering doctrine as an `internal/policies/` import-direction test
- **Adversarial note:** DFS over the 249-edge internal import graph found zero cycles. Verified foundation packages (entity, gitops, tree, check, verb, render) do not import cliutil. The two mutable package vars are documented exceptions, not the broad coupling pattern the principle warns about. Interfaces at seams (PageDataResolver, BranchOracle) are textbook narrow contracts.

### A3. Layered (no upward dependencies)

- **Verdict:** Strong
- **Summary:** Textbook layered architecture. The 22-line `cmd/aiwf/main.go` defers to `internal/cli.Execute`. Dependency arrows point uniformly downward: cmd → internal/cli → internal/verb → internal/check/render/htmlrender/initrepo → internal/tree/scope/trunk/contractcheck → internal/entity/gitops/aiwfyaml/config → internal/codes/pathutil. `go list -deps ./cmd/aiwf` produces a clean topological ordering with no cycles. Zero imports of `internal/cli` from anywhere outside `internal/cli/`. The one wrinkle: `internal/cellcoverage` (test fixture helper consumed only by `*_test.go` under `internal/policies/`) imports both `internal/verb` and `internal/cli/cliutil`, but this is excluded from the production binary closure.
- **Evidence:**
  - supports: `cmd/aiwf/main.go:11-21` 22-line entry-only main
  - supports: `internal/entity/` imports only `internal/codes`
  - supports: `internal/gitops/` zero internal imports
  - supports: `go list -deps ./cmd/aiwf` produces clean topological ordering
  - refutes: `internal/cellcoverage/authorized_scope.go` sideways dependency, test-only
  - refutes: `internal/cli/cliutil/scopes.go:1-13` `LoadEntityScopes` is a domain-shaped helper located under CLI layer (latent inversion risk)
- **Smells found:**
  - Domain-shaped helper located under CLI layer (`internal/cli/cliutil/scopes.go`'s `LoadEntityScopes`) — not yet a violation, but a latent inversion
- **Recommended moves:**
  - Relocate `internal/cli/cliutil/scopes.go` to a lower-level package such as `internal/scope/history.go`
  - Add an `internal/policies/` layering-invariant test that asserts the allowed direction mechanically
- **Adversarial note:** Adversarially grepped for upward imports from every domain package into verb/cli/cmd — zero hits. `go list -deps ./cmd/aiwf` produces a clean topological ordering with no cycles. The Strong verdict survives.

## B. Contracts

### B1. Typed interfaces

- **Verdict:** Strong
- **Summary:** Every module-boundary input and output is a named struct or named-type alias; closed sets are exposed via named enum-like types and exported constants; errors carry typed structured codes via the `entity.Coded` behavioral interface. No `interface{}`/`any` at module boundaries except a small set of JSON-envelope passthroughs by design opaque. Verb inputs use `Options` structs (AddOptions, PromoteOptions, AuthorizeOptions, ImportOptions, ContractBindOptions, AllowInput); verb outputs go through a uniform `verb.Result` discriminated record with a tagged `OpType` enum. The measurable softness: per-kind entity statuses (StatusActive, StatusDone, …) are plain untyped string constants rather than a `type Status string`, and a few other closed-set fields (Finding.Code, manifest.Entry.Kind, workflows/spec Predicate fields, OutputFormat.Format) are bare string. Compensated by closed-set const lists, runtime checks, and `PolicyEnumLiteralAdoption` CI policy.
- **Evidence:**
  - supports: `internal/entity/entity.go:19-29` Kind is named string type with six closed const values
  - supports: `internal/codes/codes.go:14-44` Class is a named int type, Code is a typed struct
  - supports: `internal/verb/verb.go:30-78` Result is a discriminated record, OpType is a named int enum
  - supports: `internal/gitops/trailers.go:15-49` Trailer keys are named string constants
  - refutes: `internal/entity/entity.go:42-65` Status const values are untyped string constants, not `type Status string`
  - refutes: `internal/check/check.go:71` Finding.Code is `string`, not a `codes.Code` or named string type
  - refutes: `internal/manifest/manifest.go` Entry.Kind and CommitSpec.Mode are bare string
- **Smells found:**
  - status passed as bare string at module boundaries despite an internal closed-set constant list (no `type Status string`)
  - Finding.Code typed as string rather than the named codes.Code descriptor
- **Recommended moves:**
  - Promote per-kind status to `type Status string`; retype the constants and propagate through ValidateTransition, IsTerminal, the transitions map, and verb signatures
  - Tighten `check.Finding.Code` to a named `FindingCode` type
  - Extend the `codes.Code` typed-descriptor pattern to every kernel finding code
- **Adversarial note:** Hunted for any-typed signatures, map[string]any abuse, positional tuples, and magic strings. All compensated by closed-set constants, runtime vocabulary checks, and PolicyEnumLiteralAdoption CI policy. Strong holds.

### B2. Schemas at boundaries

- **Verdict:** Strong
- **Summary:** aiwf treats schemas at boundaries as a first-class concern. The kernel declares each cross-process shape exactly once and validates on read: the per-kind frontmatter contract lives in a single `schemas` table in `internal/entity/entity.go:445` (drives `aiwf schema`, `refsResolve`, and the `entity.Schema` Go type — drift pinned by `TestSchemaMatchesForwardRefs`). Frontmatter YAML is decoded with `yaml.KnownFields(true)` at every read site. The JSON envelope is declared once in `internal/render/render.go:50` and conformance is asserted across every JSON-emitting verb by `TestEnvelopeSchemaConformance_AllJSONVerbs`. Commit trailers are declared once in `internal/gitops/trailers.go:15` with closed-set scope-event enum and per-key write-time `ValidateTrailer`. The `manifest.Manifest` struct carries an explicit integer `Version` field checked against `supportedVersion = 1` — the textbook "version field nothing checks" anti-pattern is explicitly avoided. `TestTestMetrics_StrictParseRoundTrip` pins writer→reader equivalence.
- **Evidence:**
  - supports: `internal/entity/entity.go:445` single `schemas` table for per-kind frontmatter contracts
  - supports: `internal/check/check_test.go:721` `TestSchemaMatchesForwardRefs` drift-prevention test
  - supports: `internal/cli/integration/envelope_schema_test.go:79` writer-vs-reader envelope conformance
  - supports: `internal/manifest/manifest.go:42` explicit Version field with supportedVersion check
  - refutes: `internal/manifest/manifest.go` outer Manifest.Parse uses bare yaml.Unmarshal — unknown top-level fields silently dropped
  - refutes: `internal/config/config.go` aiwf.yaml decode uses bare yaml.Unmarshal at top level (intentional for legacy field capture)
- **Smells found:** none
- **Recommended moves:**
  - Add a structural test in `internal/policies/` that pins the JSON envelope's required key set against `Envelope` struct field tags
  - Formalise the `raw*` decoder shim pattern (recipe.go:137, aiwfyaml.go:275) as a documented convention in CLAUDE.md
  - Consider a separate envelope schema-version field for forward compat
- **Adversarial note:** Adversarially probed the YAML/JSON decode sites, version fields, JSONL boundaries, cross-language struct duplication, and Envelope.Result typing. The single-source-of-truth plus drift-prevention tests hold up; the manifest/config top-level tolerance is documented design.

### B3. Pre/post conditions and invariants

- **Verdict:** Strong
- **Summary:** aiwf treats function contracts as a first-class engineering concern. Public functions in core packages carry detailed godoc that names inputs, return-value semantics for each branch, edge cases, and error conditions. Invariants are explicitly labeled (`Invariant:`, `Atomicity:`, `Idempotent`) and pinned by tests, both in the package itself (FSM property tests) and in the meta-policy layer (`internal/policies/fsm_invariants.go`). Defensive code is annotated with `//coverage:ignore <reason>`. The contract-style godoc shape is uniform across 423 exported functions.
- **Evidence:**
  - supports: `internal/entity/transition.go:73-94` ValidateTransition documents three distinct return cases
  - supports: `internal/verb/allow.go:92-104` AllowResult carries explicit "Invariant: every Allowed==false return sets Err" comment
  - supports: `internal/verb/apply.go:25-47` Atomicity contract documented and pinned by tests
  - supports: `internal/policies/fsm_invariants.go:10-38` PolicyFSMInvariants encodes four FSM-shape invariants as CI-blocking tests
  - supports: `internal/entity/transition_property_test.go:36-87` Property tests enforce FSM invariants exhaustively
  - refutes: `internal/gitops/gitops.go:73` Add has a one-sentence doc (acceptable for a 3-line wrapper)
- **Smells found:** none
- **Recommended moves:**
  - Add 1-line contracts to remaining tiny helpers like `dedupePaths`, `pluralize`
  - Lift the recurring "shares memory with the package-level constant" phrase into a documented convention in CLAUDE.md
- **Adversarial note:** Surveyed all 423 exported functions: zero exported functions lacking godoc, only 5 exported methods (all stdlib error-interface plumbing) without docs, only 1 short-doc instance. 70 coverage:ignore directives all carry rationales. The Strong verdict survives adversarial probing.

## C. Data discipline

### C1. Single source of truth

- **Verdict:** Strong
- **Summary:** SSOT is a load-bearing design principle, not an aspiration. The repo names a canonical store for each fact: markdown frontmatter for entity state, git trailers for history (no events.jsonl), the hardcoded `transitions` map and `schemas` table for FSM/refs, and derived facts (STATUS.md, ROADMAP.md, rendered HTML) explicitly computed via pure functions. The codebase actively polices parallel-source-of-truth as anti-pattern: M-0118 deduped captureStdout; M-072/AC-6 made `aiwf list` and `aiwf status` share `tree.FilterByKindStatuses`; M-0150 made `cliutil.AnnotationRegisteredVerbs` the single verb registry. Property tests assert the two FSM tables (AllowedStatuses + transitions) agree. Real but minor C1 smells: `idPrefix` switch duplicated in `internal/verb/import.go`, hardcoded entity-directory paths in three files, and the two-table FSM definition acknowledged and mitigated by property tests.
- **Evidence:**
  - supports: `internal/entity/transition.go:13-15` "markdown is the source of truth"
  - supports: `internal/entity/entity.go:77-83, 417-444` AllowedStatuses delegates to single schemas table
  - supports: `internal/workflows/spec/antirules.go:53-54` "NO event log file, no graph projection file"
  - supports: `internal/cli/cliutil/annotations.go:1-21` AnnotationRegisteredVerbs as single source for verb registry
  - refutes: `internal/verb/import.go:243-277` `idPrefix` switch duplicates `entity.IDPrefix()` (acknowledged in code as a mirror)
  - refutes: `internal/entity/transition_property_test.go:17-22` two parallel sources (AllowedStatuses + transitions) — drift policed by tests rather than design
- **Smells found:**
  - historical regression (already fixed): `aiwf version` printed a package-global "dev" while `version.Current()` returned the buildinfo value
  - joint two-table FSM definition (AllowedStatuses table + transitions map) — acknowledged and policed by property tests
- **Recommended moves:**
  - Consolidate the FSM's joint definition: derive AllowedStatuses from transitions[k] keys/values
  - Add a kernel-level finding/policy test for cache-mutation APIs not co-located with documented invalidation
  - Catalogue "derived artifacts" explicitly somewhere AI-discoverable
- **Adversarial note:** Hunted aggressively for parallel-state patterns. Found two real C1 smells (idPrefix duplication, hardcoded entity-directory paths) plus the already-acknowledged two-FSM-tables pattern. None severe enough to overturn Strong.

### C2. Idempotence

- **Verdict:** Strong
- **Summary:** aiwf's mutating-verb architecture treats idempotence as a first-class contract. `verb.Result` has an explicit `NoOp` discriminator with a `NoOpMessage`. Sweeping verbs (archive, rewidth, contract bind, contract recipe install) and artifact-management verbs (init/update, statusline scaffold) distinguish "already converged" from "needs change" and return NoOp in the converged case. Idempotence is asserted by named tests (TestRewidth_AlreadyCanonical_NoOp, TestArchive_NoOpResultOnConvergedTree, TestContractBind_IdempotentExactMatch, TestM0155_AC4_ProjectScopeWritesGitignoreAndRelativeSnippet, the `SlugifyDetailed` fuzz). Key-per-operation discipline: `entity.Canonicalize` normalises narrow vs canonical id forms; `appendPriorID` is documented idempotent; `dedupePaths` removes duplicates. `aiwf check` is naturally idempotent with stable-sorted output. Minor weakness: `aiwf rename`, `aiwf retitle`, `aiwf promote`, `aiwf cancel` return errors rather than NoOp when called with current-state inputs — user-facing UX shape, not state-corruption hazard. `acknowledge-illegal` is also non-idempotent in commit count.
- **Evidence:**
  - supports: `internal/verb/verb.go:30-35` Result.NoOp discriminator
  - supports: `internal/verb/archive.go:28` "Idempotent. An already-swept tree returns a NoOp Result"
  - supports: `internal/verb/contractbind.go:29-31` "idempotent against an exact match"
  - supports: `internal/policies/m0155_statusline_scaffold_test.go:131-142` no double-append idempotence enforced
  - refutes: `internal/verb/acknowledgeillegal.go:42-73` no NoOp path — re-runs append duplicate empty audit commits
  - refutes: `internal/verb/promote.go:197-205, rename.go:65-66, retitle.go:62, move.go:47` surface no-change cases as errors rather than NoOp
  - refutes: `internal/verb/add.go:91-313` by-design non-idempotent (allocates a fresh id each call)
- **Smells found:**
  - Re-running creates duplicates (limited: `aiwf acknowledge-illegal` appends a duplicate empty audit commit per invocation against the same SHA)
- **Recommended moves:**
  - Unify "no change needed" UX across verbs: NoOp + descriptive message instead of Go error on same-state inputs
  - Add a dedup guard to `aiwf acknowledge-illegal` mirroring `contract bind`/`contract recipe install`
  - Promote `verb.Result.NoOp` invariant to a kernel policy test in `internal/policies/`
- **Adversarial note:** Probed every mutating verb. The cases not flagged (add, authorize-open, editbody-explicit) are either by-design additive or state-convergent with non-NoOp UX. None violate the principle's core: state converges. Strong holds.

### C3. Atomic writes

- **Verdict:** Weak
- **Summary:** aiwf achieves transactional atomicity at the verb level (one git commit per mutation, defer-rollback in `internal/verb/apply.go` that restores worktree+index to pre-Apply state on any error including panic), and serializes concurrent verbs via flock-based repolock. Disk-level atomicity, however, is uneven. Two surfaces use the canonical temp+rename pattern: `aiwfyaml.Doc.Write` and `skills.writeManifest`. Every other write site — including the kernel's central writer at `internal/verb/apply.go:140`, every config edit, every HTML render at `internal/htmlrender/htmlrender.go:339`, every initrepo materialization, the settings.json write, statusline writes, and render-roadmap writes — calls `os.WriteFile` directly with no temp+rename and no fsync. A repo-wide grep for `f.Sync()`/fsync in production code returns zero hits.
- **Evidence:**
  - supports: `internal/verb/apply.go:48-162` single-commit-per-verb chokepoint with defer-rollback
  - supports: `internal/repolock/repolock_unix.go:1-50` POSIX advisory flock serializes mutating verbs
  - supports: `internal/aiwfyaml/aiwfyaml.go:187-197` canonical temp+rename pattern (missing fsync)
  - supports: `internal/skills/skills.go:496-512` temp+rename for ownership manifest (missing fsync)
  - refutes: `internal/verb/apply.go:140` central write uses bare `os.WriteFile` — no temp+rename, no fsync
  - refutes: `internal/skills/skills.go:355-365` SKILL.md materializations use direct os.WriteFile in a loop with no overall atomicity
  - refutes: `internal/config/config.go:340,397,463` three config-mutating paths use bare os.WriteFile
  - refutes: `internal/htmlrender/htmlrender.go:321-334` executeToFile uses os.Create + ExecuteTemplate + Close — no fsync, no temp+rename
  - refutes: zero fsync calls in production code anywhere in the repo
- **Smells found:**
  - two writes that must agree but one can fail (skills SKILL.md loop + manifest, config in-place rewrites, settings.json + .bak)
  - in-place overwrites of structured config files with no temp+rename (config.go x3, settings.go)
  - absent fsync everywhere — even the two temp+rename surfaces skip fsync-before-rename
- **Recommended moves:**
  - Promote the temp+rename pattern into a shared `pathutil.AtomicWriteFile(path, data, perm)` helper (write path+".tmp", `f.Sync()`, then `os.Rename`) and route central writers through it
  - Coordinate the two-writes-must-agree pair in `internal/skills/skills.go` via a staging dir + single `os.Rename` swap
  - Wrap `internal/htmlrender/htmlrender.go`'s whole-site render in a temp output dir + final `os.Rename` swap

### C4. Versioned schemas with migration paths

- **Verdict:** Strong
- **Summary:** aiwf has explicit, distributed migration paths for every observed schema change, paired with strict on-input validation and forward-compat tolerance for legacy data. ADR-0008's canonical id-width migration: the `aiwf rewidth` verb is idempotent, dry-run-by-default, ships in the binary, produces a single tracked commit, paired with parser tolerance (`entity.Canonicalize`) and a drift-detection check rule. The import manifest carries explicit `version: 1` with strict rejection of unsupported versions. Two deprecated `aiwf.yaml` fields (`actor:`, `aiwf_version:`) are read as `LegacyActor`/`LegacyAiwfVersion`, surface in `aiwf doctor`, and are stripped idempotently on `aiwf update`. Renames preserve history via `prior_ids` lineage. Every YAML decode site (entity, contracts block, recipes, import re-parse) uses `KnownFields(true)`. Releases are semver-tagged with mandatory CHANGELOG entries enforced by CI.
- **Evidence:**
  - supports: `internal/manifest/manifest.go:35-127` supportedVersion=1, validate rejects mismatches
  - supports: `docs/adr/ADR-0008-canonicalize-kernel-ids-to-4-digits.md:44-80` migration verb codified
  - supports: `internal/entity/canonicalize.go:10-74` forward-compat tolerance — narrow legacy widths accepted on read
  - supports: `internal/config/config.go:64-75, 303-412` Legacy* capture pattern for deprecated fields
  - supports: `internal/cli/upgrade/upgrade.go:23-83` real binary-upgrade verb with version pinning
  - refutes: `internal/recipe/recipe.go:136-160` embedded recipes have no version field
  - refutes: `internal/config/config.go:275` aiwf.yaml decoder uses bare yaml.Unmarshal at top level (intentional but open-ended for future drift)
  - refutes: `internal/manifest/manifest.go:74` manifest outer decode uses yaml.Unmarshal without KnownFields(true)
  - refutes: `docs/pocv3/design/design-decisions.md:230,241` minor documentation drift on aiwf_version
- **Smells found:** none
- **Recommended moves:**
  - Add a `version:` field to recipe frontmatter so future shape changes have a versioned knob
  - Declare a manifest v2 migration path before merging any v2 bump
  - Consider an `aiwf.yaml` top-level `schema_version:` (or formalize that the binary version IS the schema version)
- **Adversarial note:** Attempted to break the verdict by hunting for silent unknown-field drops. Found real C4-smell counter-examples for the consumer config, but the principle is about migration paths for changes that happen, and the demonstrated paths (rewidth, Legacy* capture+strip, prior_ids, KnownFields-strict, version-pinned manifest, semver tags) are genuine and load-bearing.

## D. Tests

### D1. Behavior pinned, not implementation

- **Verdict:** Strong
- **Summary:** Tests across the kernel pin observable behavior: on-disk file shape after a verb runs, frontmatter values, commit subjects, structured commit trailers parsed back from `git log`, exit codes, golden text/JSON output, and findings emitted by `check.Run` against fixture trees. The integration suite drives verbs through `cli.Execute` or a built binary, then inspects the real git tree — no mocks of the dispatcher, verb body, or loader. Almost no `Mock`/`Stub`/`Fake` types exist; the lone `fakeBlobReader` is a deliberate process-boundary fake. CLAUDE.md codifies the principle explicitly ("Substring assertions are not structural assertions"; "test the seam, not just the layer"). Minor weakness: `internal/htmlrender/htmlrender_test.go` relies on `strings.Contains` for HTML anchor/href presence — CLAUDE.md flags this exact pattern as a known weakness.
- **Evidence:**
  - supports: `internal/check/fixtures_test.go:20-99` runs check.Run against synthetic testdata trees
  - supports: `internal/cli/integration/single_commit_invariant_test.go:14-110` pins "exactly one git commit per verb" behaviorally
  - supports: `internal/cli/integration/binary_integration_test.go:39-100` binary-level subprocess tests
  - supports: grep for `gomock|mockery|testify/mock` across internal/ returns essentially zero results
  - refutes: `internal/htmlrender/htmlrender_test.go:61-181` strings.Contains for href/id assertions (known weakness)
  - refutes: `internal/cli/integration/doctor_cmd_test.go` 44 strings.Contains assertions on CLI human-readable output
- **Smells found:**
  - Some htmlrender tests use substring-style `strings.Contains` assertions on rendered HTML rather than DOM-structural parsing
- **Recommended moves:**
  - Adopt `golang.org/x/net/html` in `internal/htmlrender/htmlrender_test.go` so anchor/id assertions land inside named sections
  - Promote the structural-section discipline to an `internal/policies/` meta-test
  - Add a small TestUtil wrapping `html.Parse` + a `findInside(node, pred)` helper
- **Adversarial note:** Hunted aggressively: 88K test LOC and one process-boundary fakeBlobReader is the only test double. Substring assertions exist (~1025 total) but are dominated by CLI free-text checks where they are appropriate; the load-bearing kernel paths assert observable outcomes.

### D2. Equivalence tests at seams

- **Verdict:** Weak
- **Summary:** The codebase has several interchangeable-implementation seams: PageDataResolver (defaultResolver vs Resolver), BranchOracle (fakeOracle vs gitBranchOracle), blobReader (fakeBlobReader vs *gitops.BlobReader), and the cue/jsonschema recipes. Equivalence testing is uneven: strong examples exist (`TestEnvelopeSchemaConformance_AllJSONVerbs` parameterizes a schema-validator across every JSON-emitting verb; `TestTree_ByID_AcceptsBothWidths` drives the same lookup over narrow+canonical widths; `TestKindFSM_StateSetAgreement` exhaustively cross-checks the two FSM tables). But the most load-bearing resolver seam (PageDataResolver) has no shared conformance test, BranchOracle has no parameterized test running the same scenarios through both implementations, and cue/jsonschema recipes share no equivalence test.
- **Evidence:**
  - supports: `internal/cli/integration/envelope_schema_test.go:73-209` `TestEnvelopeSchemaConformance_AllJSONVerbs` parameterized conformance suite
  - supports: `internal/tree/tree_test.go:524-558` `TestTree_ByID_AcceptsBothWidths` matrix test
  - supports: `internal/entity/transition_property_test.go:42-67` `TestKindFSM_StateSetAgreement` exhaustive equivalence
  - supports: `internal/skills/materialize_target_test.go:35-58` `TestMaterialize_DefaultsToClaude` equivalence test
  - refutes: `internal/htmlrender/htmlrender.go:70` PageDataResolver has two implementations with no shared conformance test
  - refutes: `internal/cli/render/render_test.go:9-18` production Resolver has only smoke-shape test
  - refutes: `internal/htmlrender/htmlrender_test.go:210-271` bodyAwareResolver test workaround mimics cli/render's resolver wiring
  - refutes: `internal/check/isolation_escape.go:49-51` BranchOracle has no parameterized test through both implementations
  - refutes: `internal/recipe/embedded/` cue + jsonschema recipes share no equivalence test
- **Smells found:**
  - Reader/writer drift potential (defaultResolver vs Resolver): no shared conformance suite
  - Two implementations of the same protocol with no shared test (cue/jsonschema recipes)
- **Recommended moves:**
  - Add a shared PageDataResolver conformance suite driving both implementations against the same fixture tree
  - Add a BranchOracle conformance suite running identical scenarios through fakeOracle and gitBranchOracle
  - Add a cue/jsonschema recipe equivalence test asserting same pass/fail decisions on identical inputs

### D3. Branch coverage on touched code

- **Verdict:** Weak
- **Summary:** aiwf has a clearly articulated branch-coverage discipline — `wf-tdd-cycle` calls it a "HARD RULE" and CLAUDE.md reiterates it. The discipline is broadly practiced: 21+ files carry `//coverage:ignore <reason>` markers, milestone work logs cite branch-coverage audits, and CI publishes a `-coverprofile` artifact. However, the principle's specific call — "a coverage floor on lines/branches changed in this PR" — is not mechanically enforced. CI runs `go test -coverprofile` then `go tool cover -func | tail -n 1` which prints the overall percentage only, with no diff/PR-scoped gate and no failure threshold. Go's `-cover` is statement coverage, not branch coverage; no `gocov`, `diff-cover`, or codecov is wired up. G-0067 is an open gap calling this out: "An untested defensive branch passes the AC's promote because `aiwf check` doesn't see coverage." CLAUDE.md:427 says outright: "failing checks for low coverage are advisory at this stage."
- **Evidence:**
  - supports: `internal/skills/embedded-rituals/.../wf-tdd-cycle/SKILL.md:83` HARD RULE for diff-scoped branch-coverage audit
  - supports: `internal/skills/embedded-rituals/.../wf-review-code/SKILL.md:46` review-code rule 5 enforces branch coverage at review time
  - supports: `CLAUDE.md:344-353` "Test untested code paths before declaring code paths done"
  - supports: `internal/cli/rewidth/rewidth.go:89-176` nine `//coverage:ignore` markers each with rationale
  - refutes: `.github/workflows/go.yml:64-70` CI prints `tail -n 1` of coverage with no threshold check or failure gate
  - refutes: `Makefile:73-75` coverage target emits coverage.out and prints overall summary only
  - refutes: `CLAUDE.md:427` "failing checks for low coverage are advisory at this stage"
  - refutes: `work/gaps/G-0067-wf-tdd-cycle-llm-honor-system-advisory.md:10,28` open gap on honor-system advisory
  - refutes: no diff-coverage tool wired into any workflow
  - refutes: `internal/policies/` has no per-AC test-presence-per-branch or diff-coverage policy
- **Smells found:**
  - No coverage report consumed as a gate
  - Statement coverage, not branch coverage
  - Diff-scoped enforcement depends entirely on LLM following the wf-tdd-cycle ritual; G-0067 documents this as a known open exposure
- **Recommended moves:**
  - Wire a diff-coverage gate into the test workflow (`diff-cover --fail-under=80 --compare-branch=origin/main coverage.out`)
  - Add an `internal/policies/branch_coverage_audit_test.go` asserting every conditional branch in changed code is covered or annotated `//coverage:ignore`
  - Bump the existing Makefile coverage target to fail on overall coverage drop relative to a baseline

### D4. Tests at the right altitude

- **Verdict:** Strong
- **Summary:** aiwf's test suite picks altitude per scenario with explicit doctrine and clear stratification. Pure-function unit tests live next to the code (entity FSM, markdown rendering, version parsing, slugification, trailer parsing). Module-boundary integration tests drive real loaders + real git plumbing in tempdirs. Binary-level subprocess tests build `cmd/aiwf` into a tmpfile and exec it. Browser e2e lives at `e2e/playwright/tests/render.spec.ts` (55 tests). Mocking is essentially absent: only one file references "mock" and that reference is the comment "No mocks, no synthetic update-refs." The two `httptest` uses both mock at the true process boundary. "Test the seam, not just the layer" doctrine is codified in CLAUDE.md and referenced inline.
- **Evidence:**
  - supports: `internal/entity/transition_test.go:8-40` pure unit tests for `ValidateTransition`
  - supports: `internal/check/fixtures_test.go:18-50` module-boundary integration via `tree.Load + check.Run`
  - supports: `internal/cli/integration/binary_integration_test.go:38-60` binary subprocess tests for buildinfo/ldflags seam
  - supports: `internal/cli/integration/integration_g37_test.go:14-30` multi-process e2e with real bare origin + clones
  - supports: `e2e/playwright/playwright.config.ts:1-30` browser-level e2e (55 tests asserting per-tab content)
  - supports: `internal/cli/integration/upgrade_cmd_test.go:48-52` `httptest.NewServer` mocks proxy at process boundary
  - supports: `CLAUDE.md:292` explicit doctrine "Test the seam, not just the layer"
- **Smells found:** none
- **Recommended moves:**
  - Document the altitude taxonomy explicitly in CLAUDE.md §Testing
  - Consider widening Playwright e2e beyond the single `render.spec.ts` to limit tail risk from a shared-fixture break
- **Adversarial note:** Greps for mock/stub/fake/monkey-patch/sqlmock turn up nothing material. The only `httptest` uses mock at the true process boundary. Altitudes are clearly stratified.

## E. Errors, logs, audit trail

### E1. Structured logs

- **Verdict:** Weak
- **Summary:** aiwf is a CLI kernel with no long-running process or log file. The structured-output analog is the JSON envelope (render.Envelope: tool/version/status/findings/result/metadata) — rigorously typed, schema-tested, uniformly used. CLAUDE.md prescribes `log/slog` to stderr at INFO and says `fmt.Fprintln` to stderr is not a substitute, but that prescription is entirely unimplemented: zero `log/slog` imports anywhere, every diagnostic emission is `fmt.Fprintf` with string interpolation. Tests capture stdout/stderr bytes, not structured log events. Weak rather than Strong because the stated logger discipline is aspirational only; Weak rather than Missing because the JSON envelope is a substantive structured-output discipline that exists at every verb's exit.
- **Evidence:**
  - supports: `CLAUDE.md:463` prescribes log/slog to stderr at INFO
  - supports: `internal/render/render.go:46-58` Envelope struct with closed-set Status
  - supports: `internal/cli/cliutil/outputformat.go:47-98` single chokepoint for error/findings/success emission
  - supports: `internal/cli/integration/envelope_schema_test.go` envelope shape is schema-pinned in CI
  - refutes: grep -rn 'log/slog' across the entire Go source returns zero hits
  - refutes: `internal/cli/cliutil/statusline.go:36-93` 12 fmt.Fprintf/Printf with %v/%s interpolation
  - refutes: `internal/cli/root.go:106,124,151` interpolated stderr prints for error handling
  - refutes: `internal/render/render.go:26` metadata.correlation_id slot reserved but never populated by any verb
  - refutes: `.golangci.yml:33-43` forbidigo bans panic/os.Exit but not fmt.Print* — no mechanical chokepoint against bare prints
- **Smells found:**
  - Log messages with string interpolation (fmt.Fprintf/Printf with %v/%s across statusline.go, root.go, move.go, cancel.go, upgrade.go)
  - Grep-only debuggability for diagnostic emissions
  - Adding diagnostic output today means adding more fmt.Print* calls
- **Recommended moves:**
  - Either implement the CLAUDE.md prescription (introduce a single `internal/logger` wrapping `log/slog` with a JSON stderr handler) OR amend CLAUDE.md to ratify the actual envelope-only practice
  - Wire the documented `metadata.correlation_id` slot into actual verb metadata or remove it from the envelope contract
  - If logger discipline is adopted, add a forbidigo rule banning bare `fmt.Fprint*(os.Stderr, ...)` outside main.go and outputformat.go

### E2. Designed failure modes

- **Verdict:** Strong
- **Summary:** aiwf treats failure modes as first-class design surface. The kernel principle "errors are findings, not parse failures" is mechanized: `aiwf check` loads inconsistent state and returns findings instead of fatal-erroring. Every load-bearing failure path is documented in a comment block above the code that handles it (`internal/verb/apply.go:25-92` spells out the all-or-nothing rollback contract; `classifyGitError` documents lock-contention with lsof-based hint and the "kernel never silently retries" rule). Each documented mode has named tests. Concurrent access is policed via a typed file-lock with typed `ErrBusy` sentinel and bounded poll deadline. Network failure modes for the module proxy are enumerated and tested. Sentinel errors distinguish expected-not-found from unexpected-failure. The no-retry posture is mechanized by `internal/policies/no_retry_loops_on_git.go`. errcheck/errorlint/forbidigo are CI-blocking. 72 `//coverage:ignore` annotations each carry a rationale.
- **Evidence:**
  - supports: `internal/verb/apply.go:25-47` Apply's doc enumerates failure modes covered (atomicity, conflict-vs-stash split, panic-triggers-rollback)
  - supports: `internal/verb/apply.go:229-267` classifyGitError documents lock-contention surface with lsof hint
  - supports: `internal/policies/no_retry_loops_on_git.go:10-87` AST-level meta-policy banning git retry loops
  - supports: `internal/repolock/repolock_unix.go:1-114` typed `ErrBusy` sentinel, bounded deadline poll
  - supports: `internal/version/version.go:357-385` per-call context.WithTimeout, bounded LimitReader on error bodies
  - supports: `internal/check/check.go:1-12` "Run never returns an error" — failure-surface contract
  - supports: `internal/policies/findings_have_tests.go:7-25` meta-policy: every finding code referenced by a test
  - refutes: `internal/skills/settings.go:124` best-effort json.Unmarshal swallows error (acceptable; pretty remains nil)
  - refutes: `internal/cli/cliutil/lock.go:44` returned release closure ignores `lock.Release()` error (defensible deferred cleanup)
  - refutes: `internal/verb/apply.go` no specific ENOSPC/disk-full path — disk failures via generic write-wrap that triggers rollback
- **Smells found:** none
- **Recommended moves:**
  - Consider a synthetic-fault test harness to convert `//coverage:ignore requires concurrent FS mutation` branches into executed tests
  - Name disk-full / ENOSPC explicitly in the verb-failure documentation
- **Adversarial note:** Hunted for swallowed errors, naked retry loops, unbounded for-loops, panics in production, context-less network calls. Found only deliberate, documented patterns; the meta-policies forbid the principal smells mechanically. Strong holds.

### E3. Audit trail

- **Verdict:** Strong
- **Summary:** Perhaps the strongest principle in the kernel. Every state change rides through `internal/verb/apply.go` which guarantees "exactly one git commit per mutating verb" with structured trailers (aiwf-verb, aiwf-entity, aiwf-actor, plus the I2.5 provenance set: aiwf-principal/on-behalf-of/authorized-by/scope/scope-ends/reason/audit-only/force-for). Trailer keys are centralized constants and enforced as the only entry path by `internal/policies/trailer_keys.go`. The log IS the audit data: `aiwf history <id>` is just a `git log` filter — no separate event-log file. Actor identity is runtime-derived from `git config user.email`, shapes are validated, and a coherence/principal-agent-scope model is enforced both at write time and audit time. Manual git commits that escape the verb path are caught by `provenance-untrailered-entity-commit` (a Finding at pre-push). Documented gaps (G-0218 around operator-typed merge messages bypassing the closed-set verb registry; G-0220 about ritual SKILL.md changes lacking structural pins) indicate the team is actively hardening edges.
- **Evidence:**
  - supports: `internal/verb/apply.go:29-30` "every mutating verb produces exactly one git commit"
  - supports: `internal/gitops/trailers.go:15-49` centralized trailer key constants
  - supports: `internal/gitops/trailers.go:161-198` ValidateTrailer enforces shape constraints at write time
  - supports: `internal/policies/trailer_keys.go:9-63` CI-blocking meta-policy that no production file outside gitops contains trailer string literals
  - supports: `internal/policies/principal_write_sites.go:10-79` guards human/ prefix on principal write sites
  - supports: `internal/cli/history/history.go:233-282` `aiwf history` is a git log filter — no parallel storage
  - supports: `internal/check/provenance.go:399-499` `RunUntrailedAudit` emits findings per untrailered entity commit
  - supports: `internal/verb/coherence.go:73-156` full principal/agent/scope coherence rules
  - refutes: `work/gaps/G-0218-...md:9-18` operator-typed merge commits with fabricated aiwf-verb trailers pass pre-commit silently
  - refutes: `internal/policies/trailer_keys.go` PolicyTrailerKeysViaConstants regex `"([^"\\]*)"` appears structurally broken — runs against the live repo report zero violations despite `internal/cli/render/render.go:277-278` containing literal `"aiwf-verb"`/`"aiwf-actor"`
  - refutes: `internal/cli/render/render.go:280` render-roadmap invokes `gitops.Commit` directly with hand-built trailers (bypasses verb.Apply)
  - refutes: `internal/check/provenance.go:498` `CodeProvenanceUntrailedEntityCommit` emits at SeverityWarning — pre-push exits 0 even when manual entity edits land untrailered
  - refutes: `internal/cli/integration/trailer_shape_test.go` runtime trailer-shape test doesn't cover render-roadmap, archive, acknowledge-illegal, audit-only, rewidth, milestone-depends-on
  - refutes: `internal/cli/cliutil/actor.go` actor attribution is operator-controlled, not cryptographically attested
- **Smells found:** none
- **Recommended moves:**
  - Land G-0218: add a commit-msg git hook rejecting fabricated `aiwf-verb:` values at composition time
  - Tighten `trailer-verb-unknown` from warning to error once historical fabricated-trailer cleanup is complete
  - Consider an "audit-trail completeness" summary verb reporting open `provenance-untrailered-entity-commit` findings across the trunk window
- **Adversarial note:** Probed chokepoint claims and found the trailer-keys-via-constants policy regex is structurally broken (zero violations despite obvious literal-string references in render.go) and three audit-coverage findings emit at warning severity rather than blocking. Render-roadmap bypasses verb.Apply with hand-built trailers. The core architecture still satisfies the principle but several chokepoint claims are softer than first-review framed them.

### E4. Self-explaining errors

- **Verdict:** Strong
- **Summary:** aiwf has unusually disciplined, self-explaining error handling. Errors almost universally carry context, preserve causes via `%w` wrapping, and many surface concrete remediation hints inline. Mechanically enforced: `errorlint` is enabled so `%w` and `errors.Is/As` are required by CI; `forbidigo` blocks `panic`/`os.Exit` in library code with a single sanctioned re-panic site. Of 517 error constructions in non-test internal code, 268 use `%w` wrap-with-cause and 239 use the canonical `"<context>: %w"` shape. Typed coded errors carry machine-readable codes shared with the check-rule layer; check findings carry a structured `Hint` field populated from a centralized `hintTable`; verb errors routinely include the exact remediation command (e.g. lock-contention error includes lsof-derived holder hint).
- **Evidence:**
  - supports: `internal/verb/apply.go:245-267` classifyGitError wraps with detailed multi-line context including lsof-derived lock-holder hint
  - supports: `internal/verb/apply.go:219-226` checkStagedConflict returns multi-line error with literal recovery command
  - supports: `internal/verb/promote.go:370-374` Promote errors include FSM coordinate, why move was refused, exact flag to satisfy the rule
  - supports: `internal/entity/transition.go:96-119` FSMTransitionError is typed, implements Coded, lists allowed-set inline
  - supports: `internal/check/check.go:70-80, internal/check/hint.go:10-60` Finding.Hint populated from centralized hintTable; TestPolicyFindingCodesHaveHints enforces every code has a hint
  - supports: `.golangci.yml:9,33-43` errorlint enabled (enforces %w); forbidigo blocks panic/os.Exit in library code
  - refutes: `internal/cli/status/worktrees.go` 60+ bare `return err` — all propagate writer errors (appropriate I/O idiom)
  - refutes: `internal/skills/skills.go:312,317,321,384,450` bare `return err` propagating pre-wrapped callees (double-wrapping noise avoided)
- **Smells found:** none
- **Recommended moves:**
  - Replace internal-symbol references ("see acTransitions", "see tddPhaseTransitions") with allowed-set inline
  - Extend the typed Coded-error pattern to remaining verb-error sites
  - Add one sentence of remediation to a few short flag-validation errors
- **Adversarial note:** Hunted for cause-loss patterns, single-word errors, swallowed errors, panic sites outside the documented one. Zero cause-loss instances; single documented panic at apply.go:69. The 81 bare `return err` sites are all legitimate propagation.

## F. Reasoning aids

### F1. Names that don't lie

- **Verdict:** Strong
- **Summary:** Function names in aiwf read as sentences and match behavior with very high fidelity. Short, intent-revealing identifiers (Go convention: no `Get` prefix). Two of the most common "name lies" smells are mechanically blocked by Go test policies: `PolicyVerbsValidateThenWrite` forbids exported `internal/verb/` functions from calling write primitives, and `PolicyPrincipalWriteSitesGuardHuman` forbids principal-trailer write sites that don't include the human guard. No `util/helper/manager/do-stuff` names. `Validate*` family is consistently pure; `Is*` family is consistently boolean predicates; `Load*` family returns `(*T, []LoadError, error)` triples that honestly admit partial failure.
- **Evidence:**
  - supports: `internal/policies/verbs_validate_then_write.go:10-39` mechanically forbids exported verb funcs from calling write primitives
  - supports: `internal/policies/principal_write_sites.go:10-50` asserts every principal-trailer-writing function carries a `human/` guard
  - supports: `internal/entity/serialize.go:135-171` ValidateTitle/ValidateSlug are pure
  - supports: `internal/verb/apply.go:48-162` Apply is named accurately — applies a *Plan to the worktree
  - supports: `internal/check/check.go:227-256` rule funcs read as sentences (idsUnique, casePaths, frontmatterShape, refsResolve)
  - refutes: `internal/cli/doctor/doctor.go:111` `label(s string) string` is a short generic name (contextually clear within package doctor)
  - refutes: `internal/check/archive_rules.go:155` `ApplyArchiveSweepThreshold` mutates findings — name's "Apply" accurately signals mutation
- **Smells found:** none
- **Recommended moves:**
  - Optional micro-polish: rename `label(s string)` in `internal/cli/doctor/doctor.go:111` to `padLabel` or `reportLabel`
  - Extend `PolicyVerbsValidateThenWrite` pattern to assert `Validate*`/`Is*`/`Check*` functions across `internal/*` never call gitops/os write primitives
- **Adversarial note:** Adversarially searched for canonical lying-name smells: Get-prefixed mutators (none), util/helper/manager/doStuff shapes (none in production), Validate*/Check*/Is*/Has*/Parse*/Load*/Resolve*/Build* families (all consistently pure as named), verb-named functions that mutate (mechanically blocked).

### F2. Comments only for non-obvious "why"

- **Verdict:** Strong
- **Summary:** The aiwf codebase exhibits exemplary "why"-comment discipline. Across ~47k lines of `internal/*` code, comments overwhelmingly explain hidden constraints, design rationale, cross-package invariants, and historical context — almost never restate what the code does. Three patterns dominate: (1) Comments routinely cite the design-record they implement (679 references to G-/D-/ADR-/M-/E- entity ids inside `internal/verb/*.go` alone, 431 leading-line entity references across `internal/`), tying surprising choices back to written decisions. (2) Every `//coverage:ignore` and `//nolint` carries a one-line rationale; no bare suppressions. (3) Doc-comments on exported identifiers explain "why" alongside "what". Zero TODO/FIXME/XXX markers in non-test code. Short/restating comment patterns are entirely absent.
- **Evidence:**
  - supports: `internal/verb/apply.go:15-47` package-doc block is pure "why" (moves-before-writes ordering, atomicity guarantee, G34 isolation contract)
  - supports: `internal/verb/apply.go:113-115` inline comment cites G-0170 and explains the non-obvious why behind capturing both endpoints
  - supports: `internal/cli/upgrade/upgrade.go:177-191` proxyStaleHint's 18-line docblock explains external CDN propagation behaviour
  - supports: `internal/entity/transition.go:13-15` "deliberately one-directional FSM — there is no demote"
  - supports: grep across non-test code for TODO/FIXME/XXX/HACK returns only two hits in internal/check/entity_body.go (both validation tokens, not real markers)
  - supports: all //nolint:gosec directives carry inline rationale
  - refutes: `internal/verb/import.go:206-215` "Step 5: project." labels could look like restating "what" — turn out to map 1:1 to numbered pipeline in docstring
  - refutes: `internal/verb/rename.go:69` "Update the entity's path so checks see the projected location" is borderline but rationale clause carries the comment
- **Smells found:** none
- **Recommended moves:** none if Strong
- **Adversarial note:** Grep-hunted for restating-comment smells; spot-checked suspicious hits and found every short or imperative-looking comment was either a continuation line of a larger "why" block or a signpost into a documented pipeline. Strong verdict holds.

### F3. Decision records that survive turnover

- **Verdict:** Strong
- **Summary:** aiwf treats decision records as a first-class, mechanically-verified asset. 14 active ADRs in `docs/adr/` plus 2 in `docs/adr/archive/` (one rejected, one test-only), 17 active `D-NNNN` project-scoped decisions plus archived superseded ones. Decision records follow a consistent shape, are linked from Go source via inline comments (`// per D-0006's three-edge reachability model`, `// Per ADR-0008, the kernel emits...`), and are linked structurally via a `RuleSource{Decision: "D-0006"}` field on every legal-workflow rule. A CI-enforced policy test (`TestM0123_AC6_RuleDecisionSourcesResolve`) fails the build if any cited decision id does not resolve via the loader. CLAUDE.md explicitly forbids deletion and forbids gate language in ADRs to keep decision separate from scheduling. Supersession is canonical (47 supersede references across the corpus); archived ADRs stay reachable.
- **Evidence:**
  - supports: 14 ratified ADRs in `docs/adr/` with uniform Context/Decision/Consequences body
  - supports: 17 active D-NNNN decisions plus archive/ with superseded ones (D-0009 → D-0010)
  - supports: `work/decisions/D-0010-...md:10-46` explicit "What D-0009 got wrong" with empirical evidence
  - supports: `docs/adr/archive/ADR-0005-...md:90-114` rejected ADR retains full rationale + cross-refs
  - supports: `CLAUDE.md:38-50` ADR-authoring rules (decision is decision; no gate language; no bespoke per-ADR test pins)
  - supports: `internal/workflows/spec/rules.go:62-99` every legal-workflow rule carries `Sources: RuleSource{Decision: "D-0006"}`
  - supports: `internal/policies/m0123_ac6_decision_resolves_test.go:27-49` CI-enforced policy test for resolvable decision ids
  - refutes: `internal/cli/upgrade/upgrade.go` single "workaround" comment in production is fully rationalized
  - refutes: `docs/adr/ADR-0003-...md` minor consistency wart — header carries two `status:` lines (one in frontmatter, one embedded prose)
- **Smells found:** none
- **Recommended moves:**
  - Add a CI policy asserting every ADR/decision referenced in a Go-source comment exists in the tree
  - Document the lighter D-NNNN vs heavier ADR axis explicitly in CLAUDE.md
- **Adversarial note:** Hunted for F3 smells (no FIXME/TODO/"ask <person>"/"see Slack" in ADRs or decisions; no undocumented workarounds in production Go; all proposed/rejected ADRs retain full rationale). The supersede chain D-0009 → D-0010 even contains empirical evidence (44 false-positive count) for why D-0009 was overturned.

## G. Operational properties

### G1. Reproducible

- **Verdict:** Strong
- **Summary:** aiwf treats reproducibility as a load-bearing property: business logic (entity FSM, verb apply, check rules, gitops, render) contains zero calls to `time.Now`, `math/rand`, `crypto/rand`, or `os.Getenv`. Wall-clock and env reads are pushed to deliberate edges (status header date, worktree "X ago" labels, repo-lock timeout, HTTP version-proxy lookup, render elapsed_ms metadata). Non-determinism that can't be eliminated (map iteration, FS order) is explicitly converted to sorted/first-seen-order iteration with comments. Determinism is mechanically pinned by tests: `TestRender_DeterministicAcrossInvocations` demands byte-equal HTML across runs; `TestRenderStatus_Goldens` locks status to checked-in goldens. A dedicated repo policy forbids `GIT_AUTHOR_DATE`/`GIT_COMMITTER_DATE` references in non-test source. `go.sum` is committed; CI pins toolchain `go-version: 1.25`.
- **Evidence:**
  - supports: `internal/htmlrender/htmlrender.go:14-18` package doc enumerates determinism rules
  - supports: `internal/htmlrender/htmlrender_test.go:94-122` TestRender_DeterministicAcrossInvocations renders twice and demands byte-equality
  - supports: `internal/policies/no_timestamp_manipulation.go:16-37` forbids GIT_AUTHOR_DATE/GIT_COMMITTER_DATE in non-test source
  - supports: grep for time.Now in `internal/verb`, `internal/entity`, `internal/check`, `internal/gitops` returns zero hits in non-test files
  - supports: grep for math/rand|crypto/rand|rand. across non-test production code returns zero hits
  - supports: `go.mod`, `go.sum` committed; CI workflows pin go-version: 1.25
  - refutes: `internal/cli/status/status.go:326` BuildStatus stamps `Date: time.Now().UTC()` (display metadata only)
  - refutes: `internal/htmlrender/htmlrender.go:114,194` Render() captures time.Now() for ElapsedMs (emitted only in JSON envelope, not in HTML files)
  - refutes: `internal/repolock/repolock_unix.go:68,78` time.Now() drives lock-acquisition deadline (proper edge usage)
- **Smells found:** none
- **Recommended moves:**
  - Inject the clock into `status.BuildStatus` (now time.Time parameter or Clock interface)
  - Add a no-time-now policy under `internal/policies/` (mirroring `no_timestamp_manipulation.go`) for `internal/verb`, `internal/entity`, `internal/check`, `internal/gitops`, `internal/htmlrender`
- **Adversarial note:** Aggressive grep for time.Now/rand/os.Getenv across internal/{check,entity,verb,gitops,contractverify,htmlrender,roadmap,render} confirmed zero hits in core packages; checked template files for map ranges (they index by key, not range); confirmed Go 1.16+ os.ReadDir returns sorted entries. The four time.Now sites are all at acknowledged display/edge layers and the determinism tests pin the actual output bytes.

### G2. Reversible

- **Verdict:** Strong
- **Summary:** aiwf's mutation model is built around reversibility as a first-class design constraint. The architecture has no file-deletion primitive at all — `internal/verb/verb.go:60-77` defines only `OpWrite` and `OpMove`, so "delete" is impossible to express; "removal" is `aiwf cancel`/`promote` to a terminal status (soft-delete via FSM), and physical relocation is `aiwf archive` which moves files into `archive/` rather than unlinking them. Bulk-sweep verbs (archive, rewidth) are dry-run by default with `--apply` flipping the same Plan into execution. Every mutating verb takes a per-repo lock and goes through `verb.Apply` with a full transactional rollback. Confirmations are sized to blast radius: `--reason` is mandatory for `--force` and `--audit-only`, sovereign-act-shape transitions require a `human/` actor, settings.json edits require interactive `[y/N]` consent per ADR-0015. The "what verb undoes this?" rule is a hard prerequisite in CLAUDE.md.
- **Evidence:**
  - supports: `internal/verb/verb.go:60-77` FileOp has only OpWrite and OpMove — no OpDelete primitive
  - supports: `internal/verb/apply.go:46-92` full transaction: stash, capture pre-Apply state, defer rollback
  - supports: `internal/verb/apply.go:422-464` rollback restores pre-Apply state (not HEAD) — closes G-0170
  - supports: `internal/cli/archive/archive.go:42-99` dry-run-by-default with `--apply` opt-in gate
  - supports: `internal/verb/promote_sovereign_act.go:30-38` sovereign-act-shape transitions require `human/` actor
  - supports: `docs/adr/ADR-0004-...md:59-87` archive explicitly one-way: "You don't, deliberately"
  - supports: `CLAUDE.md:519-534` "Designing a new verb" gates every new verb on "what verb undoes this?"
  - refutes: `internal/skills/skills.go:342,460` Materialize uses `os.RemoveAll` on `.claude/skills` (scoped to gitignored marker-managed paths, mitigated by ownership manifest)
  - refutes: `internal/cli/render/render.go:267-283` render-roadmap bypasses verb.Apply (writes file + git add + commit directly)
  - refutes: `internal/initrepo/initrepo.go:1197` hook removal via os.Remove — but only after verifying the file carries the aiwf marker
- **Smells found:** none
- **Recommended moves:**
  - Extend dry-run/--apply pattern to `aiwf reallocate` and `aiwf rename` (wide-blast-radius cross-reference rewrites)
  - Document the absence of OpDelete as an explicit invariant in `internal/verb/verb.go`'s Plan/FileOp comment
  - Add a per-call pathutil.Inside guard for the `os.RemoveAll` calls in `internal/skills/skills.go`
- **Adversarial note:** Hunted for OpDelete primitives, force-push/reset-hard calls, hook bypasses, and unguarded destructive operations. PolicyVerbsValidateThenWrite is an AST-level chokepoint forbidding os.Remove/os.RemoveAll/os.WriteFile/os.Create in any exported verb function body; PolicyNoHistoryRewrites blocks --force pushes/rebases/--amend/filter-branch; PolicyNoSignatureBypass blocks --no-verify spellings. Strong verdict holds.

### G3. Observable in production

- **Verdict:** Strong
- **Summary:** aiwf is essentially designed as an observability-first system: every mutation is one git commit carrying a structured trailer block (aiwf-verb/entity/actor plus the full provenance set). "What happened, why, with what inputs" is recoverable via `aiwf history <id>` (a git-log filter — no parallel event log to drift), the HTML render's per-milestone Provenance tab, and the structured JSON envelope. Findings carry code/subcode/severity/message/path/line/entity_id/hint so a `check` failure self-locates. Sovereign acts require a `human/` actor + non-empty reason, validated at write time. Recovery paths (`--audit-only --reason`, `acknowledge-illegal --reason`, lock-contention diagnostics naming the holding PID via lsof) leave their own trailered record. Inline comments throughout reference the specific gap/decision/ADR that drove each branch.
- **Evidence:**
  - supports: `internal/gitops/trailers.go:15-49` 17 trailer constants define a comprehensive provenance vocabulary
  - supports: `internal/gitops/trailers.go:161-198` ValidateTrailer enforces human/ role on principal+on-behalf-of at write time
  - supports: `internal/cli/history/history.go:76-94` `aiwf history` resolves prior_ids lineage transparently
  - supports: `internal/render/render.go:1-69` JSON envelope (tool/version/status/findings/result/error/metadata) with typed EnvelopeError.Code
  - supports: `internal/check/check.go:70-80` Finding carries Code/Severity/Message/Path/Line/EntityID/Subcode/Hint
  - supports: `internal/verb/apply.go:245-327` classifyGitError detects index.lock contention and shells out to lsof to name the holding PID
  - supports: `internal/htmlrender/pagedata.go:267-310` HistoryRow surfaces the full provenance shape in rendered HTML
  - supports: `internal/cli/doctor/doctor.go:33-65` `aiwf doctor` is the local-diagnostic verb
  - refutes: `internal/` no production code calls log/slog despite CLAUDE.md naming it as the logging surface
  - refutes: `internal/render/render.go:26-49` JSON envelope documents `correlation_id` slot but no caller populates it
  - refutes: `internal/cli/` only read-only verbs populate the JSON envelope's Metadata map; mutating verbs emit no per-invocation metadata into the envelope
- **Smells found:** none
- **Recommended moves:**
  - Thread a `correlation_id` through Cobra root and into `render.Envelope.Metadata`
  - If runtime telemetry of long-running verbs becomes a concern, add a debug-only `--trace` flag emitting per-phase timings via log/slog
  - Promote the trailer-keys table to a one-page reader doc or `aiwf trailers --help`-style verb
- **Adversarial note:** Grepped for swallowed errors, missing trailer emissions, silent retries, and observability gaps. The `_ =` patterns are all benign (output stream Close, RegisterFlagCompletionFunc which panics only on dev misuse). Findings carry full structured shape. Recovery paths are themselves trailered. `aiwf history` reads git log directly. The minor correlation_id and slog gaps are real but immaterial for a one-commit-per-invocation CLI where the commit IS the observability record.

## Methodology

This scorecard was produced by an orchestrated review: one orient agent surveyed the repo to produce a layout/package/test-layout map; 24 scorer agents (one per principle) read the orient output plus targeted source files and returned a verdict, evidence, smells, and recommended moves; an adversarial verification pass ran on every Strong verdict to hunt for falsifying evidence; a synthesis pass merged the original and adversarial findings into the final verdicts. CLAUDE.md was treated as aspirational documentation — verdicts came from actual code, tests, and CI configuration, not from design intent. Today's date: 2026-06-04.

---

## Follow-ups

The audit surfaced ~43 atomic recommended moves across the 24 principles. They've been categorized as: (a) **nine cluster-gaps** to file as aiwf entities, (b) **micro-polishes** captured here directly, and (c) **already covered** by gaps filed at audit time. The cluster-gaps are the durable record — once filed they are queryable via `aiwf list --kind gap` and resolve in `aiwf history` for the next audit cycle. The micro-polishes sit here because each is a one-line change that doesn't justify a gap of its own.

### Already covered (gaps filed alongside the audit)

| Entity | Title | Source verdict |
|:--|:--|:--|
| G-0067 (augmented 2026-06-04) | wf-tdd-cycle is LLM-honor-system advisory; no mechanical RED-first guard | D3 |
| G-0221 | Disk-level atomic writes: no central temp+fsync+rename helper | C3 |
| G-0222 | No shared conformance suites at unsigned-cheque interface seams | D2 |
| ADR-0017 + G-0223 | Opt-in slog diagnostic logging, default off, XDG state-home file route | E1 |

### Cluster-gaps filed

Filed as aiwf entities on 2026-06-05; bodies available via `aiwf show <id>`.

| ID | Title | Source sections | Items |
|:--|:--|:--|--:|
| G-0227 | Layering & cohesion refactor: cliutil split + Options-struct adoption + policy | A1, A2, A3 | 5 |
| G-0228 | Type-system tightening: typed Status, FindingCode, codes.Code coverage | B1 | 5 |
| G-0229 | FSM consolidation + schema-versioning hygiene at config/manifest/recipe | C1, C4 | 4 |
| G-0230 | Verb UX uniformity: NoOp on same-state + dry-run on wide-blast verbs | C2, G2 | 3 |
| G-0231 | Audit-trail hardening: trailer-regex fix, render-roadmap routing, severity bump | E3 | 4 |
| G-0232 | Envelope enrichment: correlation_id wiring + mutating-verb metadata | G3, B2 | 4 |
| G-0233 | Test-shape upgrades: DOM-structural htmlrender, fault harness, e2e widening | D1, D4, E2 | 4 |
| G-0234 | Error-message polish: allowed-set inline, typed Coded coverage, remediation | E4, E2 | 4 |
| G-0235 | CLAUDE.md conventions sweep + guardrail policy tests (cited-ids, no-time-now) | B2, B3, C1, D4, F1, F3, G1, G2 | 10 |

**Total: 9 cluster-gaps, 43 atomic work items.**

The two largest concentrations:

- **`internal/policies/` additions** — 8 new policy tests across G-0227 (layering), G-0230 (NoOp invariant), G-0231 (positive-control trailer-regex test), G-0232 (envelope structural), G-0233 (DOM-structural), G-0235 (cited-ids, no-time-now-in-core, Validate-doesn't-write, cache-invalidation-documented). Plus one **policy bug fix** in G-0231 (`PolicyTrailerKeysViaConstants` regex is structurally broken — runs report zero violations against an obvious literal-string site).
- **CLAUDE.md additions** — six conventions practiced in code but not written down (raw-decoder shim, shares-memory phrase, altitude taxonomy, D-NNNN vs ADR axis, OpDelete-absence invariant, derived-artifacts catalogue). All folded into G-0235.

### Code-area hotspots

Files / packages that appear in three or more findings deserve attention as units rather than as line-by-line patches.

| Area | Findings | Pattern |
|:--|:--|:--|
| `internal/cli/cliutil/` | A1, A2, A3 | Acknowledged grab-bag; split into identity / output / gitstate / flagsupport |
| `internal/policies/` | A2, A3, B2, C1, C2, D1, F1, F3, G1 | Eight new policy tests + one broken regex |
| `internal/htmlrender/` | A1, C3, D1, D2 | Resolver split, atomic site, DOM-structural tests, conformance suite |
| `internal/verb/` | C2, C3, F1, G2 | NoOp uniformity, atomic writes, validate-then-write extension, dry-run pattern |
| `internal/entity/` | B1, C1 | `type Status string`, FSM single-source consolidation |
| `internal/skills/` | C3, G2 | Atomic writes (SKILL.md loop + manifest pair); `pathutil.Inside` guard for `os.RemoveAll` |
| `internal/config/` | C3, C4 | Atomic config rewrites; `schema_version` knob |
| `internal/cli/render/render.go` | E3 | `render-roadmap` bypasses `verb.Apply` and hand-builds trailers (single function, multiple findings) |
| `internal/render/render.go` | G3 | `correlation_id` slot dead code until something populates it |
| CLAUDE.md | B2, B3, C1, D4, F3, G2 | Six separate "document the convention" asks |

### Micro-polishes (captured here; no gap)

One-line or one-file changes too small to justify a gap of their own. Picked up in whatever next-hygiene-pass touches the surrounding file.

- **F1** — Rename `label(s string)` in `internal/cli/doctor/doctor.go:111` to `padLabel` or `reportLabel`.
- **B3** — Add a one-line doc to `dedupePaths` and `pluralize`.
- **C4** — Fix the minor `aiwf_version` drift in `docs/pocv3/design/design-decisions.md:230,241`.
- **F3** — Fix `docs/adr/ADR-0003-...md` carrying two `status:` lines (one frontmatter, one prose).

### Items deliberately not folded in

- **`internal/cli/integration/doctor_cmd_test.go`'s 44 `strings.Contains` assertions on CLI human-readable output** (D1 refuting evidence). These are appropriate substring assertions over free-text CLI output, not a smell — the rubric's "substring is not structural" rule applies to HTML/JSON, not to plain text.
- **Optional `--trace` debug flag** (G3 move 2). Deferred — depends on ADR-0017 logging surface landing first; will be a natural addition under G-0223 rather than a separate gap.

### How this section ages

This follow-ups section is the snapshot — the durable record is the gaps. When a cluster-gap reaches a terminal status, the standard archive sweep moves it; the entry here stays as historical context. When a future audit runs, the comparison point is "which of these 9 cluster-gaps closed, and what new things turned up?"
