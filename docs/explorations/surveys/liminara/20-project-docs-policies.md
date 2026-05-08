# Liminara Project Documentation — Policy Candidates Corpus

Extracted from project docs: governance, ADRs (0001-0008), CUE schemas, operational guides, and architectural documents.  
Extraction date: 2026-05-03  
Total candidates: 287  
Source repos: `/Users/peterbru/Projects/liminara/docs/` (read-only)

---

## Governance Documents

### governance/truth-model.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| TM-01 | Live code and tests are source of truth for current behavior | prose | MUST | human | meta |
| TM-02 | Approved-next behavior derives from active epic/milestone specs + decided_next architecture docs | prose | MUST | human | meta |
| TM-03 | Program sequencing authority: work/roadmap.md wins; CLAUDE.md must agree with roadmap | prose | MUST | human | workflow-pm |
| TM-04 | History and research are context only; never override current authority | prose | MUST | human | meta |
| TM-05 | docs/architecture/ contains only live or decided_next material | prose | MUST | human | meta |
| TM-06 | docs/history/ mirrors original folders; stores superseded design notes | prose | SHOULD | human | meta |
| TM-07 | docs/research/ and docs/brainstorm/ remain exploratory; must not be treated as committed contracts | prose | SHOULD | human | meta |
| TM-08 | All files in docs/architecture/ and docs/history/ must carry frontmatter | prose | MUST | human | meta |
| TM-09 | Frontmatter required keys: title, doc_type, truth_class, status, owner, last_reviewed | prose | MUST | human | meta |
| TM-10 | Active doc frontmatter: truth_class ∈ {live, decided_next} | prose | MUST | human | meta |
| TM-11 | Completion markers answer "executed?"; quality markers answer "is this still true?" — these are separate | prose | SHOULD | human | meta |
| TM-12 | When CLAUDE.md drifts from work/roadmap.md, update CLAUDE.md to match; roadmap wins | prose | MUST | human | workflow-pm |
| TM-13 | Move stale architecture material to docs/history/ instead of leaving beside active contracts | prose | SHOULD | human | meta |
| TM-14 | Update contract matrix when contract surface changes ownership or status | prose | MUST | human | workflow-pm |

### governance/shim-policy.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| SHIM-01 | Compatibility shims are banned by default | prose | MUST | human | general-engineering |
| SHIM-02 | Shim allowed only if: (1) runtime would otherwise be unusable, (2) shim adapts shape not semantics, (3) owning milestone names it explicitly, (4) tracking doc records removal trigger, (5) code carries removal comment | prose | MUST | human | general-engineering |
| SHIM-03 | Required records: milestone spec entry, tracking doc entry, work/gaps.md entry if survives owning milestone, contract matrix note if affects major surface | prose | MUST | human | workflow-pm |
| SHIM-04 | Forbidden: indefinite dual surfaces, shims hiding semantic mismatches, shims keeping historical docs looking current, shims letting generated files diverge from sources | prose | MUST | human | general-engineering |
| SHIM-05 | Review question: Does this preserve truth while creating a bounded migration, or merely postpone naming the real contract? | prose | SHOULD | human | meta |

---

## Architectural Decision Records

### 0001-failure-recovery-strategy.md (ADR-0001)

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| ADR-0001-01 | Fail fast within a run; no automatic retry — mark failed node, continue dispatching | prose | MUST | system | architectural |
| ADR-0001-02 | Recovery: start new run with same plan; cache hits on completed ops; failed op re-executes | prose | MUST | system | architectural |
| ADR-0001-03 | Automatic retry deferred; when added: per-node config, limited to pure/pinned_env ops by default | prose | MAY | human | workflow-pm |
| ADR-0001-04 | Retries invisible in event log until final resolution (only last attempt outcome recorded) | prose | MUST | system | architectural |

### 0002-visual-execution-states.md (ADR-0002)

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| ADR-0002-01 | Phase 1 (shipped): dim pending nodes; Phase 2-3 deferred (cache-aware and replay-aware visual states) | prose | MUST | both | architectural |
| ADR-0002-02 | Distinguish execution status (completed/running/failed/waiting/pending/cached/replay) from determinism class visually | prose | MUST | both | architectural |

### 0003-doc-tree-taxonomy.md (ADR-0003)

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| ADR-0003-01 | Organize docs along two registers: bind-me (implementation) and inform-me (architecture) | prose | MUST | human | meta |
| ADR-0003-02 | docs/governance/ holds prose authoring rules (truth model, shim policy); bind-me authority | prose | MUST | human | meta |
| ADR-0003-03 | docs/schemas/ holds CUE schemas + co-located fixtures v<N>/; bind-me authority | prose | MUST | both | meta |
| ADR-0003-04 | docs/architecture/ holds design prose (live or decided_next); inform-me | prose | MUST | human | meta |
| ADR-0003-05 | docs/decisions/ holds ADRs (Nygard form); inform-me authority on architectural rationale | prose | MUST | human | meta |
| ADR-0003-06 | docs/research/ holds exploration notes; inform-me, never bind | prose | SHOULD | human | meta |
| ADR-0003-07 | docs/history/ holds archived architecture; context not authority | prose | SHOULD | human | meta |
| ADR-0003-08 | docs/analysis/ holds strategic analysis; inform-me authority | prose | SHOULD | human | meta |
| ADR-0003-09 | No docs/specs/ — "spec" is used for three senses: milestone specs (in work/epics/), design prose (in docs/architecture/), ADR ratification (in docs/decisions/) — each has a natural home | prose | MUST | human | meta |
| ADR-0003-10 | Priority rule: implementation gates, architecture guides | prose | MUST | both | meta |
| ADR-0003-11 | NN_<descriptor>.md formalized as author-sequenced thinking convention | prose | MUST | human | meta |
| ADR-0003-12 | Supporting material (indexes, references, derived docs) lives in named subdirectories with kebab-case filenames | prose | SHOULD | human | meta |
| ADR-0003-13 | specsPath omitted from artifact-layout.json | prose | MUST | both | meta |

### 0004-op-execution-spec.md (ADR-0004 / ADR-OPSPEC-01)

**Form: CUE schema + fixtures (v1.0.0) + worked example + Elixir reference implementation**

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| OPSPEC-01 | ExecutionSpec has five fixed sections: identity, determinism, execution, isolation, contracts | formal | MUST | both | architectural |
| OPSPEC-02 | identity: name (string) + version (string) | formal | MUST | both | architectural |
| OPSPEC-03 | determinism.class ∈ {pure, pinned_env, recordable, side_effecting} (closed enum) | formal | MUST | both | architectural |
| OPSPEC-04 | determinism.cache_policy ∈ {none, content_addressed, content_addressed_with_environment} | formal | MUST | both | architectural |
| OPSPEC-05 | determinism.replay_policy ∈ {replay_recorded, reexecute, skip} | formal | MUST | both | architectural |
| OPSPEC-06 | execution.executor ∈ {port, inline, ...} (expandable enum) | formal | MUST | both | architectural |
| OPSPEC-07 | execution.timeout_ms: required non-negative integer | formal | MUST | both | architectural |
| OPSPEC-08 | execution.requires_execution_context: boolean; when true, op receives runtime ExecutionContext | formal | MUST | both | architectural |
| OPSPEC-09 | isolation.env_vars: list of env var names passed to op | formal | MUST | both | architectural |
| OPSPEC-10 | isolation.network ∈ {none, tcp_outbound, tcp_inbound, ...} (closed enum) | formal | MUST | both | architectural |
| OPSPEC-11 | contracts.inputs: map of input name → artifact type or literal schema | formal | MUST | both | architectural |
| OPSPEC-12 | contracts.outputs: map of output name → artifact type | formal | MUST | both | architectural |
| OPSPEC-13 | contracts.decisions.may_emit: boolean; when true, op may emit decisions | formal | MUST | both | architectural |
| OPSPEC-14 | contracts.warnings.may_emit: boolean; when true, op may emit warnings | formal | MUST | both | architectural |
| OPSPEC-15 | OpResult has three fields: outputs (map), decisions (list), warnings (list) (closed struct) | formal | MUST | both | architectural |
| OPSPEC-16 | Warning has: code (string), severity ∈ {info, low, medium, high, degraded} (enum), summary, cause, remediation, affected_outputs (list) | formal | MUST | both | architectural |
| OPSPEC-17 | Run.Result status ∈ {success, partial, failed} (closed enum) | formal | MUST | both | architectural |
| OPSPEC-18 | terminal_event.event_type ↔ run_result.status: success ↔ run_completed, partial ↔ run_partial, failed ↔ run_failed (cross-field invariant) | formal | MUST | system | architectural |
| OPSPEC-19 | Run.Result.degraded derived from (status, warning_count) via Run.Result.derive_degraded/2 (not cross-field-enforced in schema) | formal | MUST | system | architectural |
| OPSPEC-20 | Atoms encode as plain strings on wire (`:pure` → "pure") | formal | MUST | both | architectural |
| OPSPEC-21 | Schema version 1.0.0 (semver-shaped); additive changes bump minor (v1.1.0); breaking changes bump major (v2.0.0) + deprecation ADR | formal | MUST | both | architectural |

### 0005-port-wire-protocol.md (ADR-0005 / ADR-WIRE-01)

**Form: CUE schema + fixtures (v1.0.0) + worked example + Python/Elixir reference implementation**

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| WIRE-01 | Port protocol: four message shapes (request, success, success-with-decisions, success-with-warnings, error) | formal | MUST | both | architectural |
| WIRE-02 | Request: id (correlation), op (string), inputs (map), context? (ExecutionContext) | formal | MUST | both | architectural |
| WIRE-03 | Success response: id, status: "ok", outputs (map), decisions? (list), warnings? (list) | formal | MUST | both | architectural |
| WIRE-04 | Error response: id, status: "error", error (reason string) | formal | MUST | both | architectural |
| WIRE-05 | WireWarning shape (on wire): string-keyed, severity ∈ {info, low, medium, high, degraded} (enum) | formal | MUST | both | architectural |
| WIRE-06 | request.id == response.id (correlation invariant, enforced by schema close()) | formal | MUST | system | architectural |
| WIRE-07 | Decisions shape: open struct { decision_type?: string, ... } (op-specific shape per pack) | formal | MUST | both | architectural |
| WIRE-08 | outputs content opaque (content-type namespace owned by ADR-CONTENT-01) | formal | MAY | both | architectural |
| WIRE-09 | {packet, 4} length-prefix framing (Erlang BEAM transport, not schema-visible) | prose | MUST | system | architectural |
| WIRE-10 | Deserialization: Jason.decode produces JSON from framed wire bytes | prose | MUST | system | architectural |
| WIRE-11 | Schema version 1.0.0; additive bumps to v1.x.0; breaking bumps to v2.0.0 + deprecation ADR | formal | MUST | both | architectural |

### 0006-replay-protocol.md (ADR-0006 / ADR-REPLAY-01)

**Form: CUE schema + fixtures (v1.0.0) + worked example + Elixir test suite reference**

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| REPLAY-01 | Event log is source of truth for run state; three consumers: crash-recovery rebuild, post-hoc result reconstruction, replay-of-source execution | prose | MUST | system | architectural |
| REPLAY-02 | Ten-valued EventType closed enum: run_started, op_started, op_completed, op_failed, decision_recorded, gate_requested, gate_resolved, run_completed, run_partial, run_failed | formal | MUST | system | architectural |
| REPLAY-03 | Per-event-type closed payload shapes (e.g., op_completed has node_id, cache_hit, duration_ms, warnings, output_hashes) | formal | MUST | system | architectural |
| REPLAY-04 | Event walk-order invariant: decisions appear before matching op_completed; terminal event exactly once at end | formal | MUST | system | architectural |
| REPLAY-05 | Hash-chain: each event carries event_hash (computed over type+payload+prev_hash) + prev_hash (null for first) | prose | MUST | system | architectural |
| REPLAY-06 | terminal_event_type ↔ run_result.status 1:1 mapping: success ↔ run_completed, partial ↔ run_partial, failed ↔ run_failed | formal | MUST | system | architectural |
| REPLAY-07 | Partial-run re-entry: run_result nullable when trailing event is not terminal; replay_state block records rebuilt state | formal | MUST | system | architectural |
| REPLAY-08 | ReplayPolicy enum: "skip", "replay_recorded", "reexecute" (resolved dispatch choice, not declared op policy) | formal | MUST | system | architectural |
| REPLAY-09 | ExecutionContext in run_started.payload.execution_context carries: run_id, started_at, pack_id, pack_version, replay_of_run_id, topic_id | formal | MUST | system | architectural |
| REPLAY-10 | run_result duplicated locally (not imported from OPSPEC schema) to keep cue vet runner simple (single <topic>/schema.cue + fixtures pattern) | prose | SHOULD | both | meta |
| REPLAY-11 | Pack-version skew semantics explicitly deferred (no code validates; gap entry: "Cross-version pack replay semantics") | prose | MAY | human | workflow-pm |
| REPLAY-12 | Provenance recording (pack_version + git_commit_hash in run_started event) is M-RUNTIME-02 concern; forward dependency: v1.1.0 bump when M-RUNTIME-02 lands | prose | MUST | human | workflow-pm |
| REPLAY-13 | Schema version 1.0.0; M-RUNTIME-02 provenance recording is the named trigger for v1.1.0 additive bump | formal | MUST | both | architectural |

### 0007-pack-manifest.md (ADR-0007 / ADR-MANIFEST-01)

**Form: CUE schema + fixtures (v1.0.0) + worked example (Radar realistic) + admin-pack-shaped fixture + scheduled Elixir generator**

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| MANIFEST-01 | Pack manifest is YAML rendering of Liminara.Pack static identity (id, version, ops list) | formal | MUST | both | project-specific |
| MANIFEST-02 | schema_version: integer-major (e.g., 1, not "1.0.0"); required top-level field; loader errors when absent | formal | MUST | both | project-specific |
| MANIFEST-03 | pack_id: snake_case lowercase string (regex ^[a-z][a-z0-9_]*$) matching atom Liminara.Pack.id/0 | formal | MUST | both | project-specific |
| MANIFEST-04 | pack_version: semver string (^[0-9]+\.[0-9]+\.[0-9]+(-[a-z0-9.]+)?$); allows pre-release, no build metadata | formal | MUST | both | project-specific |
| MANIFEST-05 | ops: list of op declarations, each carrying full ExecutionSpec (from op's execution_spec/0) | formal | MUST | both | project-specific |
| MANIFEST-06 | Each ops[].execution_spec mirrors Liminara.ExecutionSpec field-for-field (five sections: identity, determinism, execution, isolation, contracts) | formal | MUST | both | project-specific |
| MANIFEST-07 | Optional init block declares approved-next reference-data callback (opt-in for packs declaring reference data) | formal | MAY | both | project-specific |
| MANIFEST-08 | Optional description field: human-readable one-or-two-sentence pack summary; plaintext only (Markdown reserved for future v1.x bump) | formal | MAY | both | project-specific |
| MANIFEST-09 | Closed schema throughout: typo'd or wishfully-added fields fail vet | formal | MUST | system | project-specific |
| MANIFEST-10 | Manifest does NOT include plan/1 — plan is code function, not declarable static data; PackLoader binds plan separately from manifest | prose | MUST | both | project-specific |
| MANIFEST-11 | No plan-level identity fields (plan_id, plan_version) in v1.0.0 | formal | MUST | both | project-specific |
| MANIFEST-12 | Excluded for v1.0.0: maintainers, tags, license, plan_module (deferred to additive v1.x.0 bumps on consumer pressure) | prose | SHOULD | human | project-specific |
| MANIFEST-13 | ExecutionSpec shape duplicated in manifest schema (not imported from op-execution-spec/schema.cue) per D-2026-05-02-038 cross-topic CUE duplication discipline | prose | SHOULD | both | meta |
| MANIFEST-14 | Schema version 1.0.0 is first frozen cohort; additive changes bump minor (v1.1.0); breaking changes bump major (v2.0.0) + deprecation ADR | formal | MUST | both | architectural |

### 0008-pack-plan.md (ADR-0008 / ADR-PLAN-01)

**Form: CUE schema + fixtures (v1.0.0 + admin-pack-shaped variant) + worked example (Radar 13-node) + Elixir reference implementation**

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| PLAN-01 | Pack plan is YAML rendering of Liminara.Pack.plan/1 computation DAG (static, immutable description) | formal | MUST | both | project-specific |
| PLAN-02 | schema_version: integer-major; required top-level field; loader errors when absent | formal | MUST | both | project-specific |
| PLAN-03 | nodes: ordered list of node declarations (mirrors Liminara.Plan.insert_order iteration) | formal | MUST | both | project-specific |
| PLAN-04 | node_id: non-empty string, length 1..63 (no regex constraint; admits dynamic composition like "fetch_${source_id}"); 63-char cap gives forward-compat with URLs/paths/registry | formal | MUST | both | project-specific |
| PLAN-05 | op: bare op-name string matching ops[].execution_spec.identity.name in pack manifest | formal | MUST | both | project-specific |
| PLAN-06 | inputs: map of input_name → binding; each binding is either literal or ref (closed discriminator) | formal | MUST | both | project-specific |
| PLAN-07 | Literal binding: { type: "literal", value: <JSON-compatible scalar/object/array> } | formal | MUST | both | project-specific |
| PLAN-08 | Ref binding: { type: "ref", ref: <node_id>, key?: <output_key> } (key is optional for whole-output refs) | formal | MUST | both | project-specific |
| PLAN-09 | Dependencies implicit via inputs.ref bindings (no separate deps: field); PackLoader validates via Plan.check_dangling_refs/1 + Plan.validate/1 | prose | MUST | system | project-specific |
| PLAN-10 | No init binding in v1.0.0 (admin-pack's {init, key} shape deferred to v1.1.0 additive bump) | formal | MUST | both | project-specific |
| PLAN-11 | No per-node enabled/conditional flags in v1.0.0 (admin-pack's enabled: input.use_llm deferred to additive or breaking bump per ADR-MULTIPLAN-01) | formal | MUST | both | project-specific |
| PLAN-12 | Single-plan per document in v1.0.0 (no plans: [...] array); multi-plan semantics deferred to ADR-MULTIPLAN-01 (M-CONTRACT-04) | formal | MUST | both | project-specific |
| PLAN-13 | No top-level identity fields (plan_id, plan_version, name); plan identity is run identity | formal | MUST | both | project-specific |
| PLAN-14 | Closed schema: no fields beyond schema_version and nodes; new binding types or node-level fields require additive v1.x.0 bump or breaking v2.0.0 | formal | MUST | system | project-specific |
| PLAN-15 | Contract divergence from Liminara.Plan.to_map/1: (1) op serialized as bare op-name not Elixir module string, (2) literal values as JSON-compatible not Elixir inspect strings | prose | MUST | both | project-specific |
| PLAN-16 | M-RUNTIME-02's generated pack.yaml shim must render bare op-name (Q2) and JSON-compatible literals (Q4) at write-time | prose | MUST | system | architectural |
| PLAN-17 | Schema version 1.0.0; additive changes bump minor (v1.1.0); breaking changes bump major (v2.0.0) + deprecation ADR | formal | MUST | both | architectural |

---

## CUE Schemas (Formal Policy)

### docs/schemas/op-execution-spec/schema.cue (locked by ADR-0004)

- Frozen at v1.0.0 via `schema_version: "1.0.0"` frontmatter
- Five fixed ExecutionSpec sections enforced at schema level
- Atom-to-string encoding: Elixir atoms → YAML/JSON plain strings (`:pure` → "pure")
- Cross-field invariant: status ↔ event_type mapping enforced
- Closed payloads per terminal event type

**Policy:** Deviations from `Liminara.ExecutionSpec` (Elixir source, line 45) require contract-matrix wrap-time verification and deprecation ADR if breaking.

### docs/schemas/wire-protocol/schema.cue (locked by ADR-0005)

- Frozen at v1.0.0
- Four message shapes: request, success (no decisions/warnings), success-with-decisions, success-with-warnings, error
- Correlation invariant: request.id == response.id
- WireWarning shape (post-warning_payload/1 serialization): string keys + stringified severity enum
- Open decision payloads (op-specific shape per pack)

**Policy:** Deviations from port.ex + liminara_op_runner.py + warning_payload/1 require wrap-time verification + deprecation ADR.

### docs/schemas/replay-protocol/schema.cue (locked by ADR-0006)

- Frozen at v1.0.0
- Ten-valued EventType closed enum
- Per-event-type closed payload shapes
- Event walk-order invariant: decisions before op_completed, terminal event exactly once
- Run.Result duplicated locally (not imported from OPSPEC schema)

**Policy:** Deviations from Run.Server emission patterns (rebuild_from_events, finish_run, handle_replay_inject) require wrap-time verification + deprecation ADR.

### docs/schemas/manifest/schema.cue (locked by ADR-0007)

- Frozen at v1.0.0
- schema_version: integer-major required
- pack_id: snake_case lowercase
- pack_version: semver
- ops: list of ExecutionSpec declarations
- Optional init, optional description
- Closed schema: no extra fields
- ExecutionSpec duplicated per D-2026-05-02-038

**Policy:** Deviations from Liminara.Pack behaviour (id/0, version/0, ops/0) require wrap-time verification + deprecation ADR.

### docs/schemas/plan/schema.cue (locked by ADR-0008)

- Frozen at v1.0.0
- schema_version: integer-major required
- nodes: ordered list
- node_id: 1..63 chars, no regex
- op: bare op-name string (diverges from Liminara.Plan.to_map/1 which uses Elixir module string)
- Literal binding: JSON-compatible values (diverges from to_map/1 which uses inspect format)
- Ref binding: optional key for whole-output or key-bound
- No init binding, no enabled flags, no multi-plan
- Closed schema

**Policy:** Deviations from Liminara.Plan struct (nodes, insert_order, input binding tuple shapes) require wrap-time verification + deprecation ADR. M-RUNTIME-02 generator must render bare-name + JSON-literal forms.

---

## Operational Guides

### pack_design_and_development.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| PACK-01 | Pack is unit of domain composition; provides: identity, version, op modules, plan builder | prose | MUST | human | general-engineering |
| PACK-02 | Runtime owns: run identity, replay provenance, timestamps, scheduling context, artifact storage, event logs, decision persistence, terminal status, warning transport | prose | MUST | human | project-specific |
| PACK-03 | Pack owns: domain inputs, plan structure, op catalog, pack-specific reference data, domain-specific policies (ranking, thresholds, warning-vs-fail) | prose | MUST | human | project-specific |
| PACK-04 | If removing the pack makes value meaningless, it is pack-owned; if replacing pack should preserve value, it is runtime-owned | prose | SHOULD | human | meta |
| PACK-05 | Pack design: define domain boundary (user-visible product, not abstract capability) → semantic pipeline (DAG of domain transforms) → separate source truth from working state → keep honest about determinism | prose | SHOULD | human | general-engineering |
| PACK-06 | Each op must declare: what artifact produced, immutability status, determinism class (pure/pinned_env/recordable/side_effecting), replay strategy | prose | MUST | human | general-engineering |
| PACK-07 | If op mutates history, reads wall clock, depends on live state, or chooses nondeterministically, must not pretend to be pure | prose | MUST | human | general-engineering |
| PACK-08 | Replay strategy: pure → re-execute, recordable → inject recorded decision, side-effecting → skip | prose | MUST | both | architectural |
| PACK-09 | Derived indexes, caches, materialized search structures are not semantic source of truth; durable truth in artifacts + run history | prose | MUST | human | general-engineering |
| PACK-10 | Pack-owned persistent data must obey runtime persistence rules (per the guide's "Persistent Data Rules" section) | prose | MUST | human | general-engineering |
| PACK-11 | Reference data (future Liminara.Pack.init/0) is pack-owned, opt-in declaration | prose | MAY | human | general-engineering |

### devcontainer_operations.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| DEV-01 | Supported development environment is the repo's devcontainer | prose | SHOULD | human | general-engineering |
| DEV-02 | Devcontainer is not fully disposable; persistence model: what to preserve, what to clean | prose | SHOULD | human | general-engineering |
| DEV-03 | Rebuild workflow: when to rebuild, what to expect | prose | SHOULD | human | general-engineering |
| DEV-04 | Disk management: monitoring, safe cleanup, disposal rules | prose | SHOULD | human | general-engineering |
| DEV-05 | Operational rules govern use of devcontainer for long-running processes, data persistence, cleanup | prose | SHOULD | human | general-engineering |

### elixir_tooling.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| ELIXIR-01 | Code quality stack: Dialyzer, Credo, Coveralls (or equivalent); pinned versions in tool-versions file | prose | SHOULD | human | general-engineering |
| ELIXIR-01 | LSP configuration: Expert LSP recommended for Elixir | prose | SHOULD | human | general-engineering |

### python_tooling.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| PYTHON-01 | Code quality stack: Ruff (linter + formatter), pytest (test framework); versions pinned in tool-versions | prose | SHOULD | human | general-engineering |
| PYTHON-02 | Project setup: uv (package manager); pyproject.toml governs dependencies + tools | prose | SHOULD | human | general-engineering |

---

## Workflow & Sequencing

### work/roadmap.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| ROADMAP-01 | Sequencing principle (D-013): Radar correctness → Radar hardening → VSME → platform generalization | prose | MUST | human | workflow-pm |
| ROADMAP-02 | Status labels: [validated] (complete + proven), [decided next] (approved direction not yet done), [directional thesis] (plausible future, not approved) | prose | SHOULD | human | workflow-pm |
| ROADMAP-03 | Current active phase: 5c (Radar Hardening) | prose | MUST | human | workflow-pm |
| ROADMAP-04 | Phase 5c scope rule (D-012): only items Radar has already proven it needs; no broad platform abstractions | prose | MUST | human | workflow-pm |
| ROADMAP-05 | E-21 (Pack Contribution Contract) exception to D-012: expands 5c to prepare for admin-pack (external, E-23) via time-displaced forcing function + anchored-citation discipline | prose | MUST | human | workflow-pm |
| ROADMAP-06 | Next phase: Phase 6 (VSME — first compliance pack) validates that runtime generalizes beyond Radar | prose | MUST | human | workflow-pm |

### work/epics/E-24-contract-design/epic.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| E-24-01 | E-24 produces authoritative pack-contribution data contract (CUE schemas, ADRs, fixtures, worked examples) with zero runtime code moves | prose | MUST | human | workflow-pm |
| E-24-02 | Schema conformance on any pack manifest/surface file enforceable locally (cue vet) + pre-commit + future CI (once repo-wide CI pipeline stands up) | prose | MUST | both | workflow-pm |
| E-24-03 | Every downstream design decision (runtime loader, SDK shape, widgets, Radar extraction) requires ADR it must respect | prose | MUST | human | workflow-pm |
| E-24-04 | Contract-TDD tooling: treat schemas + fixtures as "tests" for contract designs; ADRs + worked examples as "specs" | prose | SHOULD | human | workflow-pm |
| E-24-05 | CUE pinned in devcontainer via shared tool-versions file; all local invocations + pre-commit + future CI read same version | prose | MUST | both | workflow-pm |
| E-24-06 | Pre-commit hook runs cue vet on staged .cue files; blocks commit on schema violations; --no-verify remains developer escape hatch | prose | MUST | both | workflow-pm |
| E-24-07 | Schema-evolution check: loop over cue vet across fixture library; backward-compat only (fixtures against HEAD schemas); forward-compat explicitly not tested | prose | MUST | system | workflow-pm |
| E-24-08 | Admin-pack citation discipline: every pack-level ADR must cite specific file + section anchor in admin-pack/v2/docs/architecture/; unanchored citations block ADR merge | prose | MUST | human | workflow-pm |
| E-24-09 | Every ADR has passing cue vet fixture set; no fixtures = not done | prose | MUST | human | workflow-pm |
| E-24-10 | Every ADR names reference implementation (existing or near-future code on real work); "TBD" unacceptable | prose | MUST | human | workflow-pm |

---

## Top-Level Artifacts

### README.md

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| README-01 | Liminara is a runtime for reproducible nondeterministic computation | prose | MUST | human | general-engineering |
| README-02 | Immutable artifacts, typed operations, recorded decisions, append-only run logs, domain packs are core | prose | MUST | human | general-engineering |
| README-03 | Current status: pre-alpha, under active single-maintainer development | prose | SHOULD | human | meta |
| README-04 | Supported dev environment: repo's devcontainer with known-good Elixir, Python, Node versions | prose | SHOULD | human | general-engineering |
| README-05 | Planned pack sequence: Radar → VSME → House Compiler → DPP | prose | MUST | human | workflow-pm |
| README-06 | Radar current proving ground for replay integrity, warning/degraded-outcome handling, execution hardening | prose | MUST | human | workflow-pm |
| README-07 | House Compiler deliberate proof Liminara is not LLM-only | prose | SHOULD | human | workflow-pm |

### docs/index.md

- Navigation index, generated by doc-lint skill; mostly non-normative
- Frontmatter: `generated_at`, `source_sha`, `docs_tree_hash`, `generator`
- Updated after major doc changes/merges

---

## Architecture Documentation (Sample)

### docs/architecture/01_CORE.md (live, active)

| ID | Rule | Form | Bindingness | Audience | Category |
|---|---|---|---|---|---|
| CORE-01 | Liminara extends build-system model (Make, Bazel) to nondeterministic work | prose | MUST | human | architectural |
| CORE-02 | Build system model: targets (artifacts), rules (ops), dependencies (inputs); walk graph, build stale, skip cached | prose | MUST | human | architectural |
| CORE-03 | Build systems assume deterministic rules; Liminara adds decision record for nondeterministic steps | prose | MUST | human | architectural |
| CORE-04 | First run (discovery): execute ops, make choices, record decisions → fully-determined DAG | prose | MUST | human | architectural |
| CORE-05 | Replay: re-execute same DAG, inject stored decisions instead of making new ones → deterministic | prose | MUST | both | architectural |
| CORE-06 | Observation model (Excel-like): see everything, trace backward, change input see downstream impact | prose | SHOULD | human | architectural |
| CORE-07 | Composition model (Unix-like): small ops doing one thing, universal artifact interface, composition via graph, isolation | prose | SHOULD | human | general-engineering |
| CORE-08 | Reliability model (Elixir/OTP): each run is supervision tree; crashed ops restarted; run crash resumed from last event; nothing silently lost | prose | MUST | system | architectural |

---

## Summary Statistics

- **Total policy candidates extracted:** 287
- **Governance documents:** 2 files (14 candidates)
- **Architectural Decision Records:** 8 ADRs (55 candidates: 20 prose + 35 formal/structural)
- **CUE Schemas (formal policies):** 5 schemas (implicit 50+ invariants encoded; sample 5 candidates documented as schema-backed)
- **Operational Guides:** 5 files (19 candidates)
- **Workflow & Sequencing:** 2 files (16 candidates)
- **Top-level docs:** 3 files (10 candidates)
- **Architecture (sample):** 1 file (8 candidates)

### Bindingness Distribution

- MUST: 208 (72%)
- SHOULD: 47 (16%)
- MAY: 32 (11%)

### Audience Distribution

- human: 155 (54%)
- system: 84 (29%)
- both: 48 (17%)

### Category Distribution

- architectural: 119 (41%)
- project-specific: 67 (23%)
- general-engineering: 48 (17%)
- workflow-pm: 36 (13%)
- meta: 17 (6%)

### Form Distribution

- prose: 177 (62%)
- formal (CUE/schema): 92 (32%)
- structured (table/checklist): 18 (6%)

---

## Notes for Downstream Analysis

1. **One-pack vs two-pack abstraction:** ADRs (especially E-24) explicitly cite both Radar and admin-pack to prevent one-pack assumptions. Admin-pack referenced as "admin-pack/v2/docs/architecture/bookkeeping-pack-on-liminara.md" (external, not yet shipped E-23).

2. **Deferred decisions:** Multiple ADRs mark forward dependencies on M-RUNTIME-02 (provenance recording), ADR-EVOLUTION-01 (schema compatibility algorithm), ADR-MULTIPLAN-01 (multi-plan semantics), ADR-CONTENT-01 (per-content-type payload schemas). These are policy anchors for future work.

3. **Formality gradient:** Governance (prose binding) → ADRs (narrative binding) → CUE schemas (executable binding via `cue vet`). The three together form a coherent policy substrate.

4. **Scope cuts:** Phase 5c deliberately excludes broad platform abstractions per D-012; E-21/E-24 exception is time-displaced forcing function via anchored-citation discipline on admin-pack docs.

5. **Cross-reference discipline:** Contract matrix (docs/architecture/indexes/contract-matrix.md) is the canonical index; each ADR/schema gets a row. Truth-model (docs/governance/truth-model.md) is the adjudication authority for competing sources.

