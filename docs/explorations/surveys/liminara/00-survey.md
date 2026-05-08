# Liminara Repository — Structural Survey

**Last surveyed:** 2026-05-03  
**Repository:** `/Users/peterbru/Projects/liminara`  
**Project:** Liminara runtime for reproducible nondeterministic computation  
**Status:** Pre-alpha, single-maintainer, archived policies visible

---

## 1. Top-Level Layout

### Root Directories

| Directory | Purpose | Notes |
|-----------|---------|-------|
| `.ai/` | AI Framework v2 (submodule) | Framework definitions, rules, skills; source of truth for agent behavior |
| `.ai-repo/` | Project-specific AI overrides | Rules, skills, config, recipes specific to Liminara |
| `.claude/` | Claude Code adapter outputs | Generated from `.ai/` + `.ai-repo/`; local build outputs |
| `.codex/` | GitHub Copilot instructions | Single file: `instructions.md`; auto-generated from framework |
| `.devcontainer/` | Dev environment spec | Dockerfile, post-create.sh, devcontainer.json |
| `.github/` | GitHub platform layer | Copilot instructions (32KB); skills/ directory; CI workflows |
| `.vscode/` | VSCode settings | Minimal config; Python environment activation |
| `.claude-sessions-backup/` | Session history archive | Retained from previous session management |
| `docs/` | Architecture and reference docs | Multi-register: bind-me (governance, schemas) vs. inform-me (architecture, research, decisions) |
| `work/` | Project planning and tracking | Roadmap, epics, milestones, decisions, agent history, releases |
| `runtime/` | Elixir umbrella project | Core runtime, web UI, Radar pack, Python integration |
| `ex_a2ui/` | A2UI integration (submodule) | Elixir package for observation UI; version 0.6.0 |
| `dag-map/` | DAG visualization (submodule) | TypeScript/Node visualization tool |
| `proliminal.net/` | Web frontend (submodule) | TypeScript/Next.js documentation site |
| `admin-pack/` | Admin domain pack (submodule) | Composable pack module |
| `integrations/` | Language integration experiments | Python ops runner and integrations |
| `test_fixtures/` | Reusable test data | Golden fixtures for validation |
| `scripts/` | Utility scripts | CUE validation, markdown processing, test automation |

### Root-Level Files (Policy Sources)

- **CLAUDE.md** (562 lines, auto-generated): Master documentation for Claude + Copilot. Section headings indicate: Session Start, Hard Rules, Agent Routing, Framework Sources, Artifact Layout, Project-Specific Rules, Contract Design, Liminara Rules, TDD Conventions, Current Work.
- **.gitmodules** (16 lines): Five submodules: `dag-map`, `proliminal.net`, `ex_a2ui`, `.ai` (framework), `admin-pack`.
- **.gitignore** (966 bytes): Standard exclusions; no unusual policy gates.
- **README.md** (131 lines): Project overview, core concepts, status phase, layout guide, getting-started commands.
- **CHANGELOG.md** (7243 bytes): Release history with Markdown date entries.
- **LICENSE** (Apache 2.0, 11286 bytes): Standard open-source license.
- **metrics.json** (782 bytes): Apparent metrics snapshot; ephemeral.
- **agent_runtime_specs.zip** (51MB): Binary archive; out of scope.
- **erl_crash.dump** (4MB): Erlang crash artifact; ephemeral.

---

## 2. AI Scaffolding Inventory

### `.ai/` Directory (AI Framework v2 Submodule)

**Nature:** Repo-neutral framework; defines generic agent model, shared skills, templates, and enforcement rules. Sources generated artifacts into `.claude/` and `.github/`.

| Path | Size | Purpose |
|------|------|---------|
| `rules.md` | 143 lines | Framework-level rules: commits (hard gate), code quality, security, git, scratch discipline, conflict resolution |
| `paths.md` | (not checked) | Artifact layout defaults for roadmap, epic, milestone paths, ADR locations, scratch allocation |
| `agents/` | 4 files | Definitions of builder, deployer, planner, reviewer roles (source; generated to `.claude/agents/`) |
| `skills/` | 21 files | Shared skills: architect, design-contract, doc-lint, doc-garden, dead-code-audit, patch, plan-epic, plan-milestones, quality-score, release, review-code, start-milestone, tdd-cycle, update-framework, verify-contracts, wrap-epic, wrap-milestone, workflow-audit, workflow-graph, devcontainer, draft-spec |
| `templates/` | (not checked) | Standard document templates: adr.md, epic-spec, milestone-spec, quality-scorecard, tracking-doc, app-legibility |
| `tools/` | Go sources + Makefile | `wf-graph` tool for roadmap validation, graph scanning, and GitHub sync |
| `CLAUDE.md` | Auto-generated | Framework documentation mirrored into root (overridable) |
| `GUIDE.md`, `ROADMAP.md`, `CHANGELOG.md` | Reference | Framework documentation |
| `.golangci.yml` | Go lint config | Appears to be shipped with framework |

### `.ai-repo/` Directory (Liminara Overrides)

**Nature:** Project-specific rules, recipes, config, and skill overlays. Overrides framework defaults via `.ai/sync.sh`.

| Path | Size | Purpose |
|------|------|---------|
| `rules/` | 3 files | Project rules: **contract-design.md** (163 lines, Liminara-specific reviewer gate), **liminara.md** (204 lines, five core concepts, tech stack, working rules, truth discipline, doc-tree boundaries, contract matrix, decision records, validation pipeline, commit/git/submodule/project structure rules), **tdd-conventions.md** (103 lines, test coverage guide, implementation rules, code review format, test framework conventions) |
| `skills/` | 4 files | Project skills: app-legibility.md, design-contract.md (overlay for upstream), devcontainer.md (overlay) |
| `config/` | 2 files | **artifact-layout.json** (Canonical artifact paths: roadmap, epic root, milestone template, tracking doc template, ADR path, scratch location, thresholds), **commit.json** (Commit configuration) |
| `bin/` | 3 executables | Installed by `sync.sh`: `wf-graph` (versions managed), `contract-verify`, `check-contract-bundles` |
| `recipes/` | 2 files | Dead-code audit recipes: dead-code-python.md, dead-code-elixir.md |
| `scratch/` | Multi-file | In-progress work: RFC docs, YAML configs for milestones, framework-sync tracking |
| `.framework-sync-sha` | Hash | Tracks last framework sync; used by `sync.sh` to detect updates |
| `README.md`, `statusline.sh` | Reference | Documentation and shell status output |

### `.claude/` Directory (Claude Code Generated Outputs)

**Nature:** Local build outputs. Regenerated by `bash .ai/sync.sh`; sources in `.ai/` + `.ai-repo/`. NOT to be hand-edited except `.claude/CLAUDE.md` Current Work section (which is preserved).

**Agents (4 files):**
- `agents/builder.md`
- `agents/deployer.md`
- `agents/planner.md`
- `agents/reviewer.md`

**Rules (2 files):**
- `rules/ai-framework.md` (Generated framework rules)

**Skills (29 SKILL.md files):**
- Framework skills (28): `wf-*` prefix for framework skills (tdd-cycle, patch, update-framework, etc.)
- Project skills (1): `app-legibility/SKILL.md`

**Also:**
- `settings.json`, `settings.local.json` (Claude Code configuration)
- `statusline.sh` (Terminal status display)

### `.codex/` Directory (GitHub Copilot)

**Single file:** `instructions.md` (auto-generated, read-only)  
Summarizes AI Framework v2, lists agents, skills, templates, key rules.

---

## 3. Engineering Config & Enforcement

### Build & Validation

| Config File | Language | Enforcement | Path |
|------------|----------|------------|------|
| `mix.exs` | Elixir | Project definition, dependency management | `/Users/peterbru/Projects/liminara/runtime/mix.exs` |
| `mix.lock` | Elixir | Locked dependency versions | `/Users/peterbru/Projects/liminara/runtime/mix.lock` |
| `.formatter.exs` | Elixir | Code formatting rules | `/Users/peterbru/Projects/liminara/runtime/.formatter.exs` |
| `.credo.exs` | Elixir | Static analysis, code quality checks | `/Users/peterbru/Projects/liminara/runtime/.credo.exs` |
| `pyproject.toml` | Python | Project metadata, tool config (ruff, pytest) | `/Users/peterbru/Projects/liminara/runtime/python/pyproject.toml` |
| `uv.lock` | Python | Locked dependency versions via uv | `/Users/peterbru/Projects/liminara/runtime/python/uv.lock` |
| `tsconfig.json` | TypeScript | TS compiler options for proliminal.net | `/Users/peterbru/Projects/liminara/proliminal.net/tsconfig.json` |
| `package.json` | Node/TypeScript | Dependencies for dag-map | `/Users/peterbru/Projects/liminara/dag-map/package.json` |

### Schema & Contract Validation

| Schema File | Format | Purpose | Path |
|------------|--------|---------|------|
| `manifest/schema.cue` | CUE | Pack manifest validation (v1.0.0) | `/Users/peterbru/Projects/liminara/docs/schemas/manifest/schema.cue` |
| `plan/schema.cue` | CUE | Plan DAG schema | `/Users/peterbru/Projects/liminara/docs/schemas/plan/schema.cue` |
| `op-execution-spec/schema.cue` | CUE | Op execution specification | `/Users/peterbru/Projects/liminara/docs/schemas/op-execution-spec/schema.cue` |
| `wire-protocol/schema.cue` | CUE | Port wire protocol | `/Users/peterbru/Projects/liminara/docs/schemas/wire-protocol/schema.cue` |
| `replay-protocol/schema.cue` | CUE | Replay protocol definition | `/Users/peterbru/Projects/liminara/docs/schemas/replay-protocol/schema.cue` |

**Fixtures:** Each schema has `fixtures/v1.0.0/valid/` and `fixtures/v1.0.0/invalid/` directories with example YAML files for validation testing.

### CI/CD & Workflows

| File | Trigger | Enforcement | Path |
|------|---------|------------|------|
| `wf-graph-ci.yml` | PR/push to work/ or docs/decisions/ | Graph validation (wf-graph tool), roadmap alignment, GitHub issues sync | `/Users/peterbru/Projects/liminara/.github/workflows/wf-graph-ci.yml` |

**CI Enforcement:** 
- Scans work/graph.yaml for valid epic/milestone structure
- Validates against work/roadmap.md
- Syncs against GitHub issues (read-only check)
- Fails on error-severity findings; permits warnings
- Uploads findings.json, diff-roadmap.json, diff-github.json

### Linting & Code Quality Scripts

| Script | Language | Purpose | Path |
|--------|----------|---------|------|
| `cue-vet` | Bash | Validates CUE schemas against fixtures | `/Users/peterbru/Projects/liminara/scripts/cue-vet` |
| `pre-commit-cue` | Bash | Pre-commit hook for CUE validation | `/Users/peterbru/Projects/liminara/scripts/pre-commit-cue` |
| `install-cue-hook` | Bash | Installs pre-commit hook | `/Users/peterbru/Projects/liminara/scripts/install-cue-hook` |
| `detect-hardwrap-md.py` | Python | Markdown hardwrap detection | `/Users/peterbru/Projects/liminara/scripts/detect-hardwrap-md.py` |
| `reflow-md.py` | Python | Markdown reflow utility | `/Users/peterbru/Projects/liminara/scripts/reflow-md.py` |
| `generate_golden_fixtures.py` | Python | Generate test fixtures | `/Users/peterbru/Projects/liminara/scripts/generate_golden_fixtures.py` |

### Tool Pinning

| Tool | Version | Purpose | File |
|------|---------|---------|------|
| `cue` | 0.16.1 | Schema validation language | `.tool-versions` |

---

## 4. Docs Surface

### Directory Structure

| Path | Contents | Register |
|------|----------|----------|
| `docs/architecture/` | Live/decided-next system design | Inform-me (guides work, does not gate it) |
| `docs/decisions/` | ADRs (8 files: NNNN-slug.md) | Inform-me; Nygard format |
| `docs/governance/` | Prose rules (truth-model.md, shim-policy.md) | Bind-me (gates authoring) |
| `docs/schemas/` | CUE schemas + fixtures (v1.0.0) | Bind-me (machine-validated) |
| `docs/analysis/` | Strategic and compliance analysis | Inform-me |
| `docs/research/` | Exploration and investigation (21 subdirs) | Inform-me |
| `docs/history/` | Archived architecture | Context only, not authority |
| `docs/domain_packs/` | Pack design and authoring guidance (15 dirs) | Inform-me |
| `docs/guides/` | Operational guides (4 files: devcontainer, elixir, python, pack design) | Bind-me (operational procedures) |
| `docs/design_language/` | Visual and conceptual language | Inform-me |
| `docs/brainstorm/` | Exploration notes | Inform-me |
| `docs/badges/` | Status/milestone badges | Ephemeral |
| `docs/public/` | Public-facing documentation | Publish-me |

### Master Index Files

- **docs/liminara.md** (67609 bytes): Comprehensive project reference; appears to be a synthesized guide.
- **docs/index.md** (55875 bytes): Entry point index for documentation tree.
- **docs/log.md** (9444 bytes): Activity log.

### ADR Inventory

| ID | Slug | Topic | Path |
|----|------|-------|------|
| 0001 | failure-recovery-strategy | Failure and recovery handling | `docs/decisions/0001-failure-recovery-strategy.md` |
| 0002 | visual-execution-states | State visualization | `docs/decisions/0002-visual-execution-states.md` |
| 0003 | doc-tree-taxonomy | Documentation register split (bind-me vs. inform-me) | `docs/decisions/0003-doc-tree-taxonomy.md` |
| 0004 | op-execution-spec | Op execution specification | `docs/decisions/0004-op-execution-spec.md` |
| 0005 | port-wire-protocol | Port communication protocol | `docs/decisions/0005-port-wire-protocol.md` |
| 0006 | replay-protocol | Replay protocol definition | `docs/decisions/0006-replay-protocol.md` |
| 0007 | pack-manifest | Pack manifest schema | `docs/decisions/0007-pack-manifest.md` |
| 0008 | pack-plan | Pack plan schema | `docs/decisions/0008-pack-plan.md` |

### Master Document Headings

**CLAUDE.md** (562 lines):
- Session Start | Hard Rules | Agent Routing | Framework Sources | Resolved Artifact Layout | Project-Specific Rules | Contract Design (4 assertions) | Liminara Rules (working rules, tech stack, doc-tree boundaries, etc.) | TDD Conventions | Current Work

**README.md** (131 lines):
- What It Is | Core Concepts | Current Status | Repository Layout | Getting Started | Domain Direction | Docs Map | Discussions | License

---

## 5. Tech Stack Summary

**Core Runtime:** Elixir/OTP umbrella project (`runtime/mix.exs`). Supervisor-based orchestration, ETS for hot metadata, JSONL event logs, filesystem artifact storage.

**Observability Layer:** ex_a2ui (Elixir package, v0.6.0) providing A2UI observation via Bandit + WebSock. Phoenix LiveView for primary web UI.

**Compute Plane:** 
- Python ops via Erlang `:port` interface
- uv for Python package management
- ruff for linting, ty for type-checking, pytest for tests

**Data Layer:** CUE schemas (v0.16.1) for contract definitions; YAML fixtures for validation.

**Web Frontend:** TypeScript + Next.js (proliminal.net submodule) for documentation site.

**Visualization:** TypeScript/Node (dag-map submodule) for DAG visualization.

**Admin:** Domain pack (admin-pack submodule) composable ops.

**Integration:** Python SDK in `integrations/python/`; golden fixtures in `test_fixtures/`.

**Languages:** Elixir (primary), Python (secondary compute), TypeScript (visualization/web), CUE (schemas).

---

## 6. Submodules & Nested Projects

### Submodule Inventory

Five submodules in `.gitmodules`:

| Name | URL | Purpose | Status |
|------|-----|---------|--------|
| `dag-map` | github.com/23min/DAG-map | DAG visualization tool | Active; TypeScript |
| `proliminal.net` | github.com/23min/proliminal.net | Web frontend / documentation | Active; TypeScript/Next.js |
| `ex_a2ui` | github.com/23min/ex_a2ui | A2UI observation integration (v0.6.0) | Active; Elixir |
| `.ai` | github.com/23min/ai-workflow | AI Framework v2 | Framework submodule; source of truth |
| `admin-pack` | github.com/23min/admin-pack | Administrative domain pack | Active; Elixir |

### Nested Projects (Not Submodules)

| Path | Type | Purpose | Language |
|------|------|---------|----------|
| `runtime/` | Umbrella | Core Liminara runtime, web UI, Radar pack | Elixir |
| `runtime/python/` | Python project | Op runner, Python ops | Python |
| `integrations/python/` | Integration | Python SDK and experiments | Python |
| `test_fixtures/` | Fixtures | Golden data for validation | YAML |
| `scripts/` | Utilities | CUE validation, markdown processing | Bash, Python |

**Cluster:** runtime + integrations + ex_a2ui form the execution tier; dag-map + proliminal.net form the UI tier; admin-pack is a composable ops module; .ai is the instruction framework.

---

## 7. Workflow & PM Layer

### Planning & Tracking Structure

| Path | Purpose | Structure |
|------|---------|-----------|
| `work/roadmap.md` | High-level sequencing and build plan | **Single source of truth** for current and approved-next work |
| `work/epics/` | Active and completed epics | Subdirs: E-{NN}[letter]-{slug}/ with epic.md spec files |
| `work/milestones/` | Milestone tracking and specs | Structured per epic; tracking docs |
| `work/done/` | Completed work archive | Organized by epic |
| `work/releases/` | Release artifacts | Release history |
| `work/decisions.md` | Shared decision log | Append-only log across all agents; indexed by [D-NNN] |
| `work/agent-history/` | Per-agent learnings | Role-specific history files (read-only by role) |
| `work/gaps.md` | Deferred work and blockers | Items not yet scheduled |
| `work/graph.yaml` | Roadmap graph (machine-generated) | Validated by `wf-graph-ci` workflow |
| `work/decisions/` | Subdirectory | Appears to be backup or legacy |

### Epic Naming Convention

Pattern: `E-{NN}[optional-letter]-{slug}`

Examples from directory listing:
- E-01-data-model-spec (completed)
- E-02-python-compliance-sdk (completed)
- ... E-10-port-executor (completed)
- E-11b-radar-serendipity (active/recent)
- E-12-op-sandbox (active)
- E-21-pack-contribution-contract (active)
- E-24-contract-design (active)
- E-25-runtime-pack-infrastructure (active)
- E-26-pack-dx (active)
- E-27-radar-extraction-and-migration (active)

### Milestone Naming Convention

Pattern: `M-<TRACK>-<NN>` (defined in `.ai-repo/config/artifact-layout.json`)

### Agent History

Per-agent knowledge accumulation in `work/agent-history/<role>.md`:
- builder.md
- deployer.md
- planner.md
- reviewer.md

### Artifact Layout Configuration

**Canonical source:** `.ai-repo/config/artifact-layout.json`

Resolved paths:
- `roadmapPath`: `work/roadmap.md`
- `epicRootPath`: `work/epics/`
- `epicSpecFileName`: `epic.md`
- `milestoneSpecPathTemplate`: `work/epics/<epic>/<milestone-id>-<slug>.md`
- `trackingDocPathTemplate`: `work/milestones/tracking/<epic>/../<milestone-id>-tracking.md`
- `completedEpicPathTemplate`: `work/done/<epic>/`
- `adrPath`: `docs/decisions/`
- `adrTemplatePath`: `.ai/templates/adr.md`
- `scratchPath`: `.ai-repo/scratch/`

### Governance Rules

**From `.ai-repo/rules/liminara.md`:**

#### Truth Discipline (Policy Authority)
- `work/roadmap.md` is **only current sequencing and build-plan source**
- `.ai-repo/config/artifact-layout.json` is **canonical artifact layout source**; generated surfaces must mirror it
- `docs/architecture/` contains **only live or decided-next architecture**; historical → `docs/history/`
- `docs/history/` is context, not authority
- If current behavior is disputed: live code, tests, canonical specs win
- If approved next-state: active epic/milestone spec + decided-next architecture docs win

#### Compatibility & Shims
- Compatibility shims **banned by default**
- Any exception needs **named removal trigger** in milestone spec and tracking doc

#### AI Instruction Changes
- To change AI behavior: edit `.ai-repo/` and run `./.ai/sync.sh`
- Do NOT hand-edit generated instruction files except `.claude/CLAUDE.md` Current Work section (preserved)

#### Doc-Tree Boundaries (Bind-me vs. Inform-me)

**Bind-me (gates authoring):**
- `docs/governance/`: Prose rules on truth model, shim policy, schema evolution
- `docs/schemas/`: CUE schemas + fixtures (machine-validated via `cue vet`)

**Inform-me (guides work, no gate):**
- `docs/architecture/`: Live/decided-next design
- `docs/decisions/`: ADRs (Nygard form)
- `docs/research/`: Exploration notes
- `docs/history/`: Archived architecture (context only)
- `docs/analysis/`: Strategic analysis

#### Contract Matrix Discipline
- Pack-level ADRs must cite admin-pack with file + section anchor
- Contract-matrix rows verified at epic wrap
- Radar-primary / admin-pack-secondary structure required
- Reference-implementation citations shape contracts

#### Decision Records (Two Surfaces, One Policy)
- `work/decisions.md`: Shared decision log (append-only, indexed [D-NNN])
- `docs/decisions/`: ADRs (architecture, formal; Nygard format NNNN-slug.md)
- **Both are live**; no conflict — each serves its surface

#### Validation Pipeline (Per Language)
- **Elixir:** `cd runtime && mix test apps/liminara_core/test`
- **Python:** `cd runtime/python && uv run pytest` + `uv run ruff check .` + `uv run ruff format --check .`
- **CUE:** `bash scripts/cue-vet` (validates schemas against fixtures)

#### Commit Convention
- Conventional Commits format: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`
- **NEVER commit or push without explicit human approval** — "continue" / "ok" do not count (hard rule from framework)
- Branch coverage mandatory: every reachable conditional branch needs a test before declaring done
- Line-by-line audit required before commit-approval prompt

#### Git Workflow
- **TDD by default** for logic, API, data code: red → green → refactor
- **Branch discipline:** Do NOT commit milestone work to `main`
- Update `CLAUDE.md` Current Work after starting/wrapping milestone
- Submodules: Five registered; updated via `git submodule update --init --recursive`

### Status Phases

From README:
- Phase 5c currently active
- Completed: Data model, Python SDK validation, Elixir walking skeleton, OTP runtime, observation layer, Radar pack, execution-truth rewrite
- In progress: Radar hardening (warnings, degraded outcomes, pack contribution contract, op sandbox)
- Next: VSME (first compliance pack)
- Downstream: Platform generalization (persistence, scheduling, dynamic DAGs, container executor)

### Q&A Mode (AI Decision Protocol)

When user says "Q&A", switch to structured decision-making:
1. Context paragraph (constraint, trade-off, existing decisions)
2. Pros and cons per option (honest about cost, risk, reversibility)
3. Lean (one sentence; if weak, say so)
4. Numbered options (usually 3; lean marked)

Post-decision: execute exactly as picked, no follow-up flourish. One question at a time.

---

## Appendix: Rules Sources (Categorization Preview)

This survey identifies the following rule sources for later categorization:

### **Hard Gate / Enforcement**
- `.ai/rules.md` (commits, code quality, security, scratch discipline)
- `.ai-repo/rules/liminara.md` (truth discipline, doc-tree boundaries, contract matrix, decision records, validation pipeline, git workflow, submodules, project structure)
- `.ai-repo/rules/contract-design.md` (reviewer gate: pack-level ADRs, contract-matrix rows, radar-primary structure, reference-implementation citations)
- `.ai-repo/rules/tdd-conventions.md` (test coverage guide, implementation rules, code review format, test framework conventions)
- `docs/governance/truth-model.md` (execution truth foundations)
- `docs/governance/shim-policy.md` (compatibility shim removal triggers)

### **Agent & Workflow**
- `.claude/agents/builder.md`, `.claude/agents/deployer.md`, `.claude/agents/planner.md`, `.claude/agents/reviewer.md` (role definitions)
- `.claude/skills/` (29 skill SKILL.md files; framework + project-specific)
- `.ai/skills/` (21 skill definitions; source)
- `.ai-repo/skills/` (4 project skill overlays)
- CLAUDE.md (Agent Routing table, Session Start checklist, Q&A mode protocol)

### **Data Validation & Schema**
- `docs/schemas/manifest/schema.cue` (Pack manifest v1.0.0)
- `docs/schemas/plan/schema.cue` (Plan DAG)
- `docs/schemas/op-execution-spec/schema.cue` (Op execution)
- `docs/schemas/wire-protocol/schema.cue` (Port protocol)
- `docs/schemas/replay-protocol/schema.cue` (Replay protocol)
- `scripts/cue-vet` (CUE schema validation enforcement)
- `.github/workflows/wf-graph-ci.yml` (Graph structure validation)

### **Planning & Sequencing**
- `work/roadmap.md` (canonical sequencing source)
- `work/graph.yaml` (machine-generated roadmap graph)
- `work/epics/` (epic specs)
- `work/milestones/` (milestone specs and tracking)
- `.ai-repo/config/artifact-layout.json` (canonical artifact paths)

### **Operational Procedures**
- `docs/guides/pack_design_and_development.md` (pack authoring rules, ownership, persistent-data)
- `docs/guides/devcontainer_operations.md` (dev lifecycle, persistence, cleanup, rebuild)
- `docs/guides/elixir_tooling.md`, `docs/guides/python_tooling.md` (language-specific operational rules)
- `runtime/.formatter.exs` (Elixir formatting)
- `runtime/.credo.exs` (Elixir static analysis)
- `runtime/python/pyproject.toml` (Python tool config: ruff, pytest, ty)

### **Tech Stack & Integration**
- `runtime/mix.exs` (Elixir project definition)
- `runtime/python/pyproject.toml` (Python project definition)
- `.devcontainer/devcontainer.json` (Known-good environment spec)
- `.tool-versions` (Tool pinning: cue 0.16.1)

### **Project Status & Context**
- `README.md` (Status phase, current sequencing, docs map)
- `work/decisions.md` (Append-only decision log)
- `docs/decisions/` (8 ADRs; architecture decisions)
- `docs/history/`, `docs/research/`, `docs/analysis/` (Context, not authority)

---

**End of Survey**

Recommendations for next steps:
1. **Rules extraction** (categorization exercise): Group rules by category (general engineering, project-specific, workflow/PM, rest).
2. **Enforcement audit**: Map which rules are machine-enforced (CI, formatters, schema validators) vs. prose-gated (reviews, ADRs, governance docs).
3. **Policy narrative**: Build a minimal policy document that synthesizes bind-me rules into a single reference.
4. **Conflict detection**: Cross-check rule sources for contradictions or overlaps (e.g., agent rules in CLAUDE.md vs. .claude/agents/).

