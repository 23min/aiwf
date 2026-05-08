# Enforcement Mechanisms — Liminara

## Overview

Liminara implements a **multi-rung, polyglot enforcement stack** that covers schema validation (CUE), code quality (Elixir/Credo + Quokka, Python/Ruff), static type checking (Elixir/Dialyzer, TypeScript strict mode), runtime contract verification (cross-language hash fixtures, property-based tests, golden fixtures), and workflow/roadmap integrity (wf-graph validators). The dominant pattern is **schema-as-code with fixture libraries**: five CUE schemas plus 40+ valid/invalid fixtures per schema form the skeleton, with a pre-commit hook guarding against fixture-schema drift (M-CONTRACT-01/02). No security scanners (SAST) or runtime assertions are detected. Notably missing: Elixir does not use Dialyzer in the Credo config (configured in mix.exs but not wired to pre-commit); Python lacks comprehensive type hints (no mypy/pyright); the Markdown docs use custom Python reflowers rather than a standardized linter.

---

## By Mechanism

### Rung 2 — Pattern Lint / Regex / Glob

#### Credo (Elixir code quality)
- **Source:** `/Users/peterbru/Projects/liminara/runtime/.credo.exs`
- **What it enforces:** Elixir style consistency (line length ≤120 chars, consistent spacing, no trailing whitespace, proper naming conventions, line-ending consistency). Credo checks span consistency (exception names, operators, parentheses, tabs vs. spaces), readability (function names, module names, predicate names, line length, trailing blanks), design (FIXME/TODO tags, duplicate code disabled), refactoring (cyclomatic complexity, function arity, nesting depth), and warnings (dangerous operations like `IEx.pry` left in code, unused operations). Quokka plugin (enabled) offloads many style checks (alias order, pipe layout, doc strings, module directive order).
- **Mechanism:** Pattern/lint rules; inline regex constraints for specific checks.
- **Locus:** IDE (editor.formatOnSave via ElixirLS in .devcontainer), pre-commit (via `credo` binary when invoked), CI (not wired in current CI workflow, but available).
- **Blocking/Warning:** Credo defaults to advisory warnings; can be configured strict: true per config (configured in mix.exs at line 4). Non-blocking by default unless CI integrates it.

#### Quokka (Elixir formatting + advanced style)
- **Source:** `.formatter.exs` plugin list + `.credo.exs` comments indicating delegated checks.
- **What it enforces:** Elixir code formatting (imports, aliases, module layout, pipe conventions, string sigils, unnecessary parentheses, large numbers, block nesting). Quokka is a code formatter + linter that standardizes form before Credo runs.
- **Mechanism:** Regex + AST-aware pattern matching (Elixir macro DSL).
- **Locus:** Editor via ElixirLS (formatOnSave), mixed with credo runs.
- **Blocking/Warning:** Formatters are typically non-blocking (auto-fix on save); style violations block only if strict mode is enabled and CI gate is in place.

#### Ruff (Python linting + formatting)
- **Source:** `/Users/peterbru/Projects/liminara/runtime/python/pyproject.toml`
- **What it enforces:** Python code errors (E, F), imports (I), warnings (W). Line length 100 chars, Python 3.12+ target.
- **Mechanism:** Regex + static analysis patterns (pycodestyle, PyFlakes, isort).
- **Locus:** IDE (editor.formatOnSave + source.fixAll in .devcontainer), manual runs.
- **Blocking/Warning:** Advisory unless gated by CI.

#### Markdown hardwrap detection + reflow (custom Python scripts)
- **Source:** `/Users/peterbru/Projects/liminara/scripts/detect-hardwrap-md.py`, `/Users/peterbru/Projects/liminara/scripts/reflow-md.py`
- **What it enforces:** Markdown docs must use soft-wrap (one paragraph per line), not hard-wrap (line breaks within prose). Detection classifies files by median line length + wrap-evidence ratio; reflow collapses hard-wrapped paragraphs into single lines while preserving code blocks, tables, lists, blockquotes, headings. YAML frontmatter is preserved.
- **Mechanism:** Regex (line classification) + heuristic (median length + wrap-evidence %), string collapsing algorithm.
- **Locus:** Manual (invoked by developer), not pre-commit or CI.
- **Blocking/Warning:** Advisory; developer-initiated, not enforced.

#### golangci-lint (Go tools)
- **Source:** `/Users/peterbru/Projects/liminara/.ai/.golangci.yml`
- **What it enforces:** Go code in .ai/tools/: errcheck (error handling), govet (vet checks), ineffassign (unused assignments), staticcheck, unused (dead code), gocritic (style), revive (style + naming), gosec (security — with exceptions for deliberate G301/G304/G306 cases), bodyclose (HTTP response closure), unconvert (redundant conversions), misspell (typos). 5-minute timeout.
- **Mechanism:** Multiple specialized linters + AST analysis.
- **Locus:** Manual (make -C tools lint) or CI (not observed in current workflows).
- **Blocking/Warning:** Manual gate; not pre-commit or CI-enforced in observed config.

---

### Rung 3 — Schema / Type Check

#### CUE Schema Validation (Five topics)
- **Source:** `/Users/peterbru/Projects/liminara/docs/schemas/{topic}/schema.cue` for each topic:
  1. **op-execution-spec** — ExecutionSpec (identity, determinism, execution, isolation, contracts), OpResult (outputs, decisions, warnings), Warning (severity taxonomy locked at: info/low/medium/high/degraded), RunResult (status, aggregation fields), Terminal events (run_completed / run_partial / run_failed).
  2. **wire-protocol** — Port wire protocol (Liminara.Executor.Port ↔ Python ops over stdio). Request (id, op, inputs, optional context), Response (success with outputs/decisions/warnings or error with message).
  3. **plan** — Pack computation plan (DAG of op invocations). schema_version (integer), nodes (id, op, inputs), InputBinding (literal value or ref to another node's output). Forbids dangling refs and cycles at runtime (Plan.from_map/1), not schema-level.
  4. **manifest** — Pack metadata (pack_id, pack_version, ops list, optional description/init block). Op declarations include full ExecutionSpec per op. pack_id matches regex `^[a-z]([a-z0-9]+(_[a-z0-9]+)*)?$` with 63-char limit (DNS-label).
  5. **replay-protocol** — Event-log stream (append-only JSONL, event_type discriminator + payload). 10 event types (run_started, op_started, op_completed, op_failed, decision_recorded, gate_requested, gate_resolved, run_completed, run_partial, run_failed). Includes hash chain (event_hash, prev_hash), ExecutionContext rider, replay-inject inputs, terminal status derivation.

- **Invariants enforced by schema (Rung 3 only; cross-field runtime checks deferred to Rung 4):**
  - All required fields present and correctly typed (string, int, bool, object, array).
  - Enum values locked to actual usage set (determinism: pure|pinned_env|recordable|side_effecting, executor: inline|task|port, network: none|tcp_outbound, severity: info|low|medium|high|degraded).
  - Optional fields marked with `?:` for nilable Elixir fields.
  - `close()` blocks stray top-level or in-struct keys.
  - Regex constraints: pack_id, plan node_id (max 63 chars), pack_version (semver shape).
  - Cross-field constraints (CUE `if` guards): terminal_event.event_type must match run_result.status; request.id == response.id.

- **Fixture library layout (per M-CONTRACT-01 AC6):**
  ```
  docs/schemas/<topic>/schema.cue
  docs/schemas/<topic>/fixtures/v<N>/valid/<name>.yaml
  docs/schemas/<topic>/fixtures/v<N>/invalid/<name>.yaml
  ```
  Total: 5 topics × ~8 valid fixtures each + invalid regressions = 40+ fixture files.

- **Mechanism:** CUE language constraints (type, enum, regex, `close()`, conditional guards).
- **Locus:** Pre-commit via `scripts/pre-commit-cue`, developer manual runs via `scripts/cue-vet`, CI (not currently wired).
- **Blocking/Warning:** Blocking pre-commit; invalid fixture on valid/ path fails; valid fixture passing invalid/ is a regression (schema-evolution loop regression).

#### CUE Validation Scripts
- **`scripts/cue-vet`** (no-arg mode): Schema-evolution loop — walks `docs/schemas/*/fixtures/v*/` and validates every valid fixture must pass, every invalid fixture must fail. Exit code 1 on any mismatch.
- **`scripts/pre-commit-cue`:** Invoked by `.git/hooks/pre-commit` (installed via `scripts/install-cue-hook`). Vets each staged .cue file individually; if any fixture is staged, runs the schema-evolution loop. Blocks commit on failure.
- **`scripts/install-cue-hook`:** Idempotent installer for `.git/hooks/pre-commit`. Detects foreign hooks and refuses to overwrite.

#### Dialyzer (Elixir static type checker)
- **Source:** `runtime/mix.exs` line 11–13: `dialyzer: [plt_add_apps: [:mix, :ex_unit]]`
- **What it enforces:** Elixir runtime type errors (type mismatches, unmatched return values, unreachable code, arity mismatches). Dialyzer builds a PLT (persistent lookup table) and analyzes the beam files.
- **Mechanism:** Static type inference + data-flow analysis.
- **Locus:** Manual (mix dialyzer) or IDE, not pre-commit or CI-observed.
- **Blocking/Warning:** Advisory (manual invocation); not gated in observed CI.

#### TypeScript strict mode (dag-map only)
- **Source:** dag-map does not have a tsconfig.json; proliminal.net (submodule, out of scope) has one.
- **What it enforces:** Not actively configured in the primary liminara dag-map artifact.
- **Mechanism:** TypeScript compiler flags (if tsconfig existed, would check: noImplicitAny, strictNullChecks, strictFunctionTypes, etc.).
- **Locus:** Not enforced.
- **Blocking/Warning:** N/A.

---

### Rung 4 — Runtime / Test-based Contract Verification

#### Property-based tests (ExUnit + ExUnitProperties)
- **Source:** `/Users/peterbru/Projects/liminara/runtime/apps/liminara_core/test/liminara/property_test.exs`
- **What it enforces:**
  - DAG generator validity: all generated plans are valid (Plan.validate/1 succeeds).
  - Termination invariant: every plan terminates within 5 seconds.
  - Event integrity invariant: every completed run has a valid hash chain (Event.Store.verify/1).
  - Completeness invariant: every started node has a terminal event (op_completed or op_failed). For successful runs, all nodes must be started.
- **Mechanism:** Property generators (Liminara.Generators.dag_plan()) + assertion checks. Runs 50–100 iterations per property.
- **Locus:** Test suite (ExUnit); runs on `mix test` or via CI.
- **Blocking/Warning:** Test failure blocks the build.

#### Golden fixtures (cross-language hash verification)
- **Source:** `/Users/peterbru/Projects/liminara/scripts/generate_golden_fixtures.py` generates fixtures in `test_fixtures/golden_run/`
- **Test:** `/Users/peterbru/Projects/liminara/runtime/apps/liminara_core/test/liminara/golden_fixtures_test.exs`
- **What it enforces:** Hash chain integrity (event_hash computed from event_type, payload, prev_hash, timestamp matches recorded event_hash). Run seal (seal.run_seal == final_event.event_hash). Decision hash validity. Tampered events (payload modified) fail hash verification.
- **Mechanism:** Golden fixtures generated by Python SDK (canonical_json, hash_bytes, hash_event functions) + Elixir test recomputes hashes and asserts equality. Contract: both runtimes must produce identical hashes for the same canonical JSON payload.
- **Locus:** Test suite; runs on mix test.
- **Blocking/Warning:** Test failure blocks the build.

#### Python op test conventions
- **Source:** `/Users/peterbru/Projects/liminara/runtime/python/tests/test_*.py` (8 test files: op_runner, radar_cluster, radar_embed_dedup, radar_fetch, radar_llm_dedup, radar_normalize, radar_rank, radar_summarize).
- **What it enforces:** Op correctness (input → output transformations), LLM dedup logic, clustering, summarization behavior. Tests use pytest + assertions.
- **Mechanism:** Unit tests with fixtures + assertions.
- **Locus:** Test suite; runs on pytest (pyproject.toml `test` optional-dependency).
- **Blocking/Warning:** Failure blocks the build if CI gates on pytest.

#### Elixir app-level tests
- **Source:** `runtime/apps/*/test/liminara/` subdirs (canonical, hash, plan, op, execution_contract, runtime_contract, golden_fixtures, replay, etc.).
- **What it enforces:** Core contracts (ExecutionSpec shape, Canonical JSON encoding, Plan DAG validation, Op dispatch, Run.Server event emission, Replay walker consistency, Execution context riders). Tests verify cross-module contract adherence.
- **Mechanism:** ExUnit assertions + property tests.
- **Locus:** Test suite; runs on mix test.
- **Blocking/Warning:** Failure blocks build.

#### wf-graph validators (workflow/roadmap integrity)
- **Source:** `.github/workflows/wf-graph-ci.yml` invokes wf-graph from `/Users/peterbru/Projects/liminara/.ai-repo/bin/wf-graph` (compiled Go binary from `.ai/tools/cmd/wf-graph/main.go`).
- **What it validates:**
  - `wf-graph scan`: Walks repo surfaces (work/, docs/decisions, .ai/) and emits raw JSON per surface.
  - `wf-graph validate`: Emits findings (error-severity and warn-severity) on cycles, status/location drift, dangling refs, ghosts. Non-zero exit on error-severity findings.
  - `wf-graph diff-roadmap`: Compares ROADMAP.md dep-graph section + per-row statuses against graph.yaml.
  - `wf-graph diff-github`: Compares `github_issue:` fields against gh issue state (opt-in).
- **Mechanism:** Graph walk + constraint checking (topological sort for cycles, field presence, edge referential integrity, status consistency).
- **Locus:** CI (pull_request and push to main, gated on work/** / docs/decisions/** / work/graph.yaml changes). GitHub PR annotations.
- **Blocking/Warning:** error-severity findings block the CI job (exit 1); warn-severity flows through as annotations.

#### contract-verify (design-contract schema validation for frameworks)
- **Source:** `/Users/peterbru/Projects/liminara/.ai-repo/bin/contract-verify` (compiled Go binary from `.ai/tools/cmd/contract-verify/main.go`).
- **What it validates:** Schema bundles (CUE or language-specific schemas) with valid/invalid fixture libraries. Verifies every valid fixture passes, every invalid fixture fails, and historical fixtures still validate (schema-evolution loop). Used by framework consumers to gate contract-backed design documents.
- **Mechanism:** Language-agnostic fixture runner (invokes CUE, JSON Schema, TypeScript type-checker, etc. per configured validator).
- **Locus:** Manual invocation (CI gating available but not observed in liminara.yml); likely used by design-contract workflows in the parent framework.
- **Blocking/Warning:** Failure blocks if gated by consumer CI.

---

## By Domain

### Elixir code
- **Rung 2:** Credo (line length, naming, whitespace, FIXME/TODO, complexity, arity). Quokka (formatting, alias order, module layout).
- **Rung 3:** Dialyzer (type checking, configured but not pre-commit gated).
- **Rung 4:** ExUnit + property tests (DAG generation, termination, event integrity, completeness invariants). App-level contract tests (canonical encoding, plan validation, execution semantics, replay consistency). Test count: 100+ test cases across liminara_core, liminara_radar, liminara_observation, liminara_web.

### Python code
- **Rung 2:** Ruff (E, F, I, W checks; line length 100 chars; py312 target).
- **Rung 3:** No mypy/pyright; zero type hints observed.
- **Rung 4:** pytest (8 op test modules; unit tests with fixtures and assertions).

### TypeScript / JavaScript (dag-map)
- **Rung 2:** None configured (no ESLint, Prettier).
- **Rung 3:** No strict tsconfig.
- **Rung 4:** Node test runner (`scripts/test` in package.json: node --test test/unit/*.test.mjs). Playwright for visual tests.

### Go code (.ai/tools/)
- **Rung 2:** golangci-lint (errcheck, govet, staticcheck, gosec, revive, gocritic, etc.). Not pre-commit gated in observed config.
- **Rung 3:** Go type system (enforced by compiler).
- **Rung 4:** Not observed in liminara codebase.

### CUE schemas
- **Rung 3:** CUE language constraints (type, enum, regex, close(), if guards).
- **Rung 4:** Fixture library (40+ valid/invalid YAML + schema-evolution loop via cue vet). Pre-commit hook enforces fixture-schema consistency.

### Markdown docs
- **Rung 2:** Custom hardwrap detection + reflow (scripts/detect-hardwrap-md.py, scripts/reflow-md.py). Not enforced pre-commit; advisory only.
- **Rung 3:** None.
- **Rung 4:** None.

### Roadmap / workflow graph (work/graph.yaml, ROADMAP.md, docs/decisions/)
- **Rung 2:** None.
- **Rung 3:** None.
- **Rung 4:** wf-graph validators (scan, validate, diff-roadmap, diff-github). CI-gated; blocks on error-severity findings.

### Commits
- **Rung 1 prose:** CLAUDE.md (commit message conventions, inferred from docs).
- **Rung 2-4:** No automated commit message linting observed.

### Submodules (.ai/, proliminal.net/)
- **Rung 2-3:** .ai/ has golangci-lint. proliminal.net (Next.js submodule) not analyzed (out of scope per spec).
- **Rung 4:** Not observed.

---

## By Locus

### Pre-commit hooks
- **scripts/pre-commit-cue:** Schema validation (CUE) + fixture-schema consistency. Installs via `scripts/install-cue-hook`.
- **Status:** Defined, idempotent installer present, but .git/hooks/pre-commit not installed by default (developer must run `scripts/install-cue-hook`).

### Push
- None wired.

### CI (GitHub Actions)
- **wf-graph-ci.yml:** wf-graph validate, diff-roadmap, diff-github. Triggers on PR and push to main (gated on work/**, docs/decisions/**, work/graph.yaml). Blocks on error-severity findings.
- **Status:** Active, blocking.

### Build / test (mix, pytest, node)
- **Elixir:** `mix test` (ExUnit, property tests, golden fixtures). `mix credo` (available, not gated).
- **Python:** `pytest` (op tests; gating not observed).
- **TypeScript:** `node --test` (unit tests); `node test/flow-visual.spec.mjs` (visual tests with Playwright).

### IDE / editor
- **ElixirLS:** formatOnSave via .formatter.exs + Quokka. Credo linting available.
- **Python:** Ruff formatOnSave + source.fixAll.
- **Configured in:** `.devcontainer/devcontainer.json` (VS Code extensions + settings).

### Manual / developer-invoked
- **scripts/cue-vet:** Manual schema validation run.
- **scripts/detect-hardwrap-md.py, scripts/reflow-md.py:** Manual markdown linting.
- **make -C tools lint:** Go linting.
- **mix dialyzer:** Elixir type checking.

---

## Gaps

### Unenforced rules (Rung 1 prose only — no mechanical check):

1. **Commit message conventions:** No pre-commit hook enforces commit message format (e.g., "feat:", "fix:", subject length ≤72 chars, blank line before body). Likely documented in CLAUDE.md or README but not enforced.

2. **Cross-app consistency (Elixir):** No enforcer verifies that all Elixir apps use identical formatter/credo configs. Manual sync required.

3. **Python type hints:** No mypy/pyright enforces type annotations. "Duck typing by convention" is relied upon; no static type checking.

4. **TypeScript (dag-map):** No tsconfig, no eslint, no prettier. Defaults to Node.js native test runner with no strictness.

5. **Markdown taxonomy/structure:** No enforcer validates doc-tree taxonomy (e.g., "all ADRs must be in docs/architecture/", "ROADMAP sections must match epic registry"). Likely governed by prose conventions in docs/governance/ or .ai-repo/rules/.

6. **Dependency/import cycles:** Elixir Credo does not check for circular dependencies in the umbrella app structure. Runtime catches some, but no upfront validation.

7. **Security (SAST):** No security scanner (Semgrep, Trivy, etc.) gates commits or CI.

8. **Integration tests:** No observed cross-module or cross-language integration test suite (e.g., Elixir runtime ↔ Python ops end-to-end, Elixir ↔ TypeScript DAG visualization).

9. **Documentation completeness:** No enforcer requires that new code is accompanied by docs (e.g., "every new op must have an ADR", "every new module must have doc comments"). Credo's ModuleDoc check is delegated to Quokka (status: enabled) but not integrated into a pre-commit gate.

10. **Submodule versioning:** .ai/ (framework) and proliminal.net (web UI) are submodules; no enforcer ensures they are pinned to specific commits or checked for updates in CI.

11. **API stability / contract breaking:** No schema versioning enforcer (ADR-EVOLUTION-01 is approved but not yet landed). When MANIFEST or PLAN schema_version bumps, no automated check ensures backward-compatibility story (e.g., v1.0 → v1.1.0 is additive, → v2.0.0 is breaking and requires migration ADR).

12. **Fixture-fixture consistency:** No cross-fixture validator (e.g., "if test_fixtures/golden_run/events.jsonl exists, then test_fixtures/golden_run/seal.json and decisions/ must also exist and be consistent"). Relies on generation script (`generate_golden_fixtures.py`) idempotency.

13. **Markdown line length:** Soft-wrap convention (one paragraph per line) is checked by custom scripts, but no pre-commit gate enforces it. Developers must manually run detect-hardwrap-md.py.

14. **Language-specific test naming:** Python tests follow test_*.py convention; Elixir uses *_test.exs. No enforcer validates naming (would require directory walker + regex per language).

15. **Code coverage:** No code-coverage threshold enforcer (e.g., "tests must cover >80% of lines"). Would require CI gate on mix test --cover or pytest --cov.

---

## Enforcement Maturity Summary

| **Rung** | **Count** | **Tech** | **Locus** | **Blocking** | **Gaps** |
|----------|-----------|----------|----------|-------------|---------|
| **Rung 2** | 6 tools | Credo, Quokka, Ruff, golangci-lint, custom MD scripts | IDE, manual, CI (none) | Advisory (IDE format on save) | Go linting not pre-commit gated; Markdown not CI gated |
| **Rung 3** | 3 checkers | CUE schemas (5 topics), Dialyzer, TS strict (unconfigured) | Pre-commit (CUE), manual (Dialyzer) | Blocking pre-commit (CUE) | Dialyzer not pre-commit; TypeScript not configured |
| **Rung 4** | 5 test suites | ExUnit properties + golden fixtures, pytest, node tests, wf-graph, contract-verify | Test suite + CI | Blocking build/CI | No integration tests; no SAST; no security; fixture generation not schema-gated |
| **Rung 1 (prose)** | 0 enforcers | Commit conventions, doc taxonomy, API versioning, import cycles | CLAUDE.md, ADRs, governance docs | None | Completely unenforced |

**Dominant pattern:** **Schema-as-code** (CUE) with **fixture-library validation** and **pre-commit + test gates**. Strengths: type-safe contracts, property-based stress tests, event hash integrity, workflow graph consistency. Weaknesses: Dialyzer and type hints opt-in (not enforced), Markdown and commit conventions prose-only, no security scanning, no integration test bridge between Elixir/Python/TypeScript components.

---

## Configuration Files Reference

### Core enforcement configs:
- `.devcontainer/devcontainer.json` — IDE extensions + formatOnSave settings.
- `.devcontainer/Dockerfile` — CUE 0.16.1 installation, Elixir 1.18.4, Python 3.12, Node.js 22, uv, GitHub CLI, Claude Code CLI.
- `runtime/.formatter.exs` — Quokka plugin, input paths (lib/, test/, config/).
- `runtime/.credo.exs` — Elixir linting: consistency, readability, refactoring, warnings (25 checks enabled, 20+ delegated to Quokka).
- `runtime/mix.exs` — Deps: credo, dialyxir, quokka, mix_unused, ex_doc.
- `runtime/python/pyproject.toml` — Ruff target=py312, line-length=100, selects E/F/I/W.
- `.ai/.golangci.yml` — Go linting (14 linters, security exceptions).
- `.tool-versions` — CUE 0.16.1 (pinned).
- `.github/workflows/wf-graph-ci.yml` — CI workflow: scan, validate, diff-roadmap, diff-github; blocks on error-severity.

### Scripts:
- `scripts/cue-vet` — Single entry point for schema validation (no-arg mode = schema-evolution loop).
- `scripts/pre-commit-cue` — Pre-commit hook logic (stages .cue files + fixtures).
- `scripts/install-cue-hook` — Idempotent installer for .git/hooks/pre-commit.
- `scripts/detect-hardwrap-md.py` — Classify Markdown wrapping.
- `scripts/reflow-md.py` — Collapse hard-wrapped prose.
- `scripts/generate_golden_fixtures.py` — Generate hash-chain fixtures (cross-language canary).

### Test organization:
- `runtime/apps/liminara_core/test/` — 30+ test modules (properties, golden fixtures, canonical, hash, plan, op, execution contract, replay).
- `runtime/python/tests/` — 8 test modules (op runner, Radar ops: cluster, embed_dedup, fetch, llm_dedup, normalize, rank, summarize).
- `test_fixtures/golden_run/` — Generated by generate_golden_fixtures.py; contains events.jsonl, seal.json, decision records, artifacts (content-addressable blobs).

### Schemas:
- `docs/schemas/op-execution-spec/schema.cue` — ExecutionSpec, OpResult, Warning, RunResult, Terminal events.
- `docs/schemas/wire-protocol/schema.cue` — Port protocol (request/response).
- `docs/schemas/plan/schema.cue` — Pack plan (DAG of op invocations).
- `docs/schemas/manifest/schema.cue` — Pack identity + op declarations.
- `docs/schemas/replay-protocol/schema.cue` — Event log stream + replay semantics.
- Each schema has `fixtures/v1.0.0/valid/*.yaml` and `invalid/*.yaml`.

