# Liminara — Policy Corpus, Categorized

**Synthesis of:** `00-survey.md` (structural), `10-ai-scaffolding-policies.md` (338 candidates), `20-project-docs-policies.md` (287 candidates), `30-enforcement-mechanisms.md` (15 enforcers across 4 rungs).

**Total raw candidates extracted:** ~625 normative claims. After de-duplication across overlapping sources (CLAUDE.md mirrors `.ai/rules.md` mirrors `.claude/rules/ai-framework.md` because of the framework-sync mechanism), the unique-rule corpus is roughly 350–400.

**Frame:** the user-requested four buckets — *general engineering*, *project-specific*, *workflow / PM*, *rest* — plus a fifth that emerged in mining and is worth naming separately: *agent-behavior* (rules about how AI agents operate). Each bucket is sized below by hand-counted candidates from the raw lists.

---

## 1. The categorization at a glance

| Bucket | Count (~) | What's in it | Carries to other repos? |
|---|---|---|---|
| **A. General engineering** | ~140 | TDD, commits, formatting, linting, security, branch discipline, semver, reviewer checklist, ADR pattern. Applies to any codebase. | Yes — same shape, different commands. |
| **B. Project-specific (Liminara)** | ~180 | Truth model, doc-tree taxonomy, contract matrix, named CUE schemas, named directories, pack-manifest invariants, replay protocol, shim-ban with named-trigger exception. | No — these are this repo's invariants. The *shape* of the rules transfers; the *content* does not. |
| **C. Workflow / PM (aiwf domain)** | ~70 | Epic / milestone / ADR / decision / gap lifecycle, naming, two-surface decisions, tracking docs, roadmap as truth, sync-via-script. | Yes — this is exactly the aiwf framework's stuff. |
| **D. Agent-behavior** | ~90 | Q&A mode, session start, agent routing, skill invocation, subagent thoroughness, heartbeat, cost discipline, role responsibilities. | Mostly — these are framework defaults, project overrides exist. |
| **E. Rest** | ~25 | Operational guides (devcontainer lifecycle, Elixir/Python tooling), CHANGELOG, license, status-phase notes, README structure. Mostly description, not normative. | Mixed. |

> The split is rougher than the numbers suggest — many rules sit on a boundary. *"NEVER commit without explicit human approval"* is general-engineering in shape but lives in the agent-behavior layer because it binds the AI specifically. *"work/roadmap.md is the canonical sequencing source"* is a workflow rule by category but its phrasing ("the canonical X") is project-specific. The bucket assignment reflects the *dominant* axis; many rules belong to two.

---

## 2. Bucket A — General engineering (~140)

Rules that would apply to almost any software project, expressed in this stack's idiom.

### A.1 Test discipline (~25)

- **TDD by default for logic/API/data code** (red → green → refactor). Source: `tdd-conventions.md`, `.ai/rules.md`, skill `tdd-cycle`.
- **Branch coverage is a hard rule, not a target.** Every reachable conditional branch (if/else/switch/catch/?:/early-return) must have an explicit test. Audit happens *before* the commit-approval prompt, not after.
- **Tests are deterministic** — no randomness, no clock, no network. Independent — no shared mutable state. Edge cases covered. Names explain *what* is tested.
- **Antipatterns banned:** code before tests, tests that can't fail, skipping refactor, testing implementation details, execution-order dependencies.
- **Genuinely unreachable branches require a "Coverage notes" entry in the milestone spec** with the reason.
- **Test names follow framework convention.** Project's test framework is the runner; tests run with `mix test` (Elixir), `uv run pytest` (Python).
- **Property tests cover invariants** (DAG generation, termination, event integrity, completeness invariants — actively used in Elixir).
- **Golden fixtures verify cross-language hash equivalence** (Python SDK ↔ Elixir runtime).

### A.2 Commits & branches (~15)

- **NEVER `git commit` or `git push` without explicit human approval.** "Continue" / "ok" / "looks good" do *not* count. (Mirrored across CLAUDE.md, .ai/rules.md, all four agent files. Most-replicated rule in the corpus.)
- **Conventional Commits format:** `feat:` `fix:` `chore:` `docs:` `test:` `refactor:`.
- **Branch discipline:** don't commit milestone work to `main`.
- **Stage-show-stop-wait:** the agent stages, shows summary, stops, waits. No autonomy past staging.
- **Never deploy without green tests on `main`.**
- **Semver on releases.**
- **Document rollback steps for infrastructure changes.**

### A.3 Code quality / language idiom (~30)

- **Per-language formatters and linters are the gate**, not advisory commentary. Elixir: Credo + Quokka + `mix format`. Python: Ruff (E, F, I, W; line length 100; py312). TypeScript: strict mode where configured. Go: golangci-lint (errcheck, govet, ineffassign, staticcheck, unused, gocritic, revive, gosec, bodyclose, unconvert, misspell).
- **Static type checking exists per language** but is *not* uniformly gated: Dialyzer configured, not pre-commit-wired; ty installed, not gated; no mypy.
- **No secrets in code, no PII, no `eval`.**
- **Reviewer checklist:** all ACs met; tests cover ACs; tests deterministic; build passes; no unrelated changes; naming follows conventions; error handling adequate; no secrets/PII; README/docs updated if public API changed.
- **Naming conventions per language** (snake_case Elixir/Python, camelCase TS).
- **Be specific in feedback** — reference files and lines.
- **Distinguish blocking issues from suggestions.**

### A.4 Documentation (general) (~10)

- **ADRs follow Michael Nygard 2011 pattern** (Context → Decision → Consequences + status vocabulary).
- **README updated when public API changes.**
- **Markdown docs are soft-wrap (one paragraph per line)**, not hard-wrap. (Custom Python detector + reflow scripts; advisory.)

### A.5 Security & defensive coding (~10)

- **No secrets, tokens, API keys in code.** (`gosec` enforces in Go; absent in Elixir/Python.)
- **No `eval`, no shelling out without justification.**
- **Defensive paths count as reachable** — if a guard ships, it gets a test.

### A.6 Process minima (~10)

- **Never deploy without health checks.**
- **Document why for every load-bearing decision** (ADR or decision-log).
- **Don't introduce abstractions until the third instance.**
- **Don't add error handling for impossible scenarios.**

### A.7 Stack-aware "general" (~40 mixed)

This is the messy middle: rules that would be general but become stack-specific in expression. Recipes (`recipes/dead-code-python.md`, `recipes/dead-code-elixir.md`) are the canonical example — same intent, different commands.

---

## 3. Bucket B — Project-specific (Liminara) (~180)

The largest bucket. Rules whose content is unique to this codebase. *Shape* often generalizes; *content* doesn't.

### B.1 The "truth model" (~15)

- **`work/roadmap.md` is the only current-sequencing and build-plan source.**
- **`.ai-repo/config/artifact-layout.json` is the canonical artifact-layout source**; generated surfaces *mirror* it, never redefine.
- **`docs/architecture/` contains only live or decided-next architecture.** Historical material moves to `docs/history/`.
- **`docs/history/` is context, not authority.**
- **Disputed current behavior:** live code, tests, and canonical persistence specs win.
- **Disputed approved-next behavior:** active epic/milestone spec + decided-next architecture docs win.
- **All files in `docs/architecture/` and `docs/history/` carry frontmatter** with required keys: `title, doc_type, truth_class, status, owner, last_reviewed`.
- **Active doc frontmatter:** `truth_class ∈ {live, decided_next}`.
- **CLAUDE.md drift from roadmap:** update CLAUDE.md, never the other way.

### B.2 Doc-tree taxonomy (ADR-0003) (~15)

- **Two registers: bind-me (gates work) and inform-me (informs work).**
- **Bind-me directories:** `docs/governance/` (prose authoring rules), `docs/schemas/` (CUE + fixtures).
- **Inform-me directories:** `docs/architecture/`, `docs/decisions/`, `docs/research/`, `docs/history/`, `docs/analysis/`, `docs/brainstorm/`.
- **Priority rule: implementation gates, architecture guides.**
- **No `docs/specs/` directory** — "spec" has three senses (milestone specs, design prose, ADRs); each has a natural home.
- **No single `contracts/` subtree** — contract components live in separate directories.
- **Author-sequenced thinking convention:** files prefixed `NN_<descriptor>.md` are top-tier in author sequence.
- **Supporting material lives in named subdirectories with kebab-case filenames.**

### B.3 Contract matrix discipline (~20)

- **`docs/architecture/indexes/contract-matrix.md`** is the live ownership/status index for every first-class contract surface.
- **Plan-time:** any milestone that creates / modifies / retires a contract surface MUST include `## Contract matrix changes` in its spec.
- **Wrap-time:** before wrapping, the reviewer verifies declared rows are present in the matrix with correct live-source paths. Row absence blocks wrap.
- **Rename / move / delete:** the same PR updates the matrix row.
- **Pack-level ADRs cite admin-pack** with file + section anchor.
- **Two-pack forcing rule:** ADRs that involve a pack cite both Radar (primary, in-tree) and admin-pack (secondary, in submodule). Prevents one-pack abstraction.
- **Reference-implementation citations shape contracts** — the contract is what the reference impl actually does, not what the prose claims.

### B.4 Contract bundle discipline (skill `design-contract`) (~16)

- **The bundle IS the contract:** ADR + schema + valid fixtures + invalid fixtures + worked example + reference implementation, reviewed together as one PR.
- **Invalid fixtures are not optional** — schema's permissiveness goes untested without them.
- **TBD reference impl = wish, not contract.** Must be existing (`file:line`) or scheduled (named milestone).
- **Schema evolution is first-class:** every committed historical fixture validates against the *current HEAD* schema. Fixture failure = consumer-breakage signal.
- **Worked example uses concrete domain values** — no placeholders, no lorem ipsum.
- **Invalid fixtures exercise real failure modes**, not syntactic errors.

### B.5 The five CUE-defined contracts (~30)

These are the *canonical* policies of the system, encoded as machine-checkable schemas + fixtures. Each has its own ADR.

| Schema | What it pins | Notable invariants |
|---|---|---|
| `op-execution-spec` | ExecutionSpec, OpResult, Warning, RunResult, terminal events | Determinism enum (`pure / pinned_env / recordable / side_effecting`); executor enum; severity enum (info/low/medium/high/degraded); terminal event ↔ run status cross-field guard. |
| `wire-protocol` | Port wire (Elixir Executor.Port ↔ Python ops over stdio) | request.id == response.id; success XOR error shape. |
| `plan` | Pack computation plan (DAG) | `schema_version` integer; nodes have id, op, inputs; InputBinding is literal *or* ref-to-prior-node; bare op-names; JSON-compat literals; single plan per document. Cycles + dangling refs caught at runtime, not in CUE. |
| `manifest` | Pack metadata | `schema_version` integer; `pack_id` DNS-label regex `^[a-z]([a-z0-9]+(_[a-z0-9]+)*)?$` ≤63 chars; closed schema (`close()`); `pack_version` semver. |
| `replay-protocol` | Append-only event-log JSONL | 10 event types (run_started, op_started, op_completed, op_failed, decision_recorded, gate_requested, gate_resolved, run_completed, run_partial, run_failed); event_hash + prev_hash chain; replay-injected inputs ride the event payload. |

All five are versioned `v1.0.0` with co-located valid + invalid fixtures. `cue vet` runs them; `scripts/cue-vet` (no-arg) does the schema-evolution sweep.

### B.6 Compatibility shims banned by default (~5)

- **Banned by default.** Allowed only if (1) runtime would otherwise be unusable, (2) shim adapts shape not semantics, (3) owning milestone names it explicitly, (4) tracking doc records the *removal trigger*, (5) code carries a removal comment.
- **Forbidden:** indefinite dual surfaces; shims hiding semantic mismatches; shims keeping historical docs looking current; shims letting generated files diverge from sources.
- **Reviewer prompt:** *"Does this preserve truth while creating a bounded migration, or merely postpone naming the real contract?"*

### B.7 Per-language validation pipelines (~10)

- **Elixir:** `cd runtime && mix test apps/liminara_core/test`. Plus `mix format --check-formatted`, `mix credo`. Dialyzer configured but not gated.
- **Python:** `cd runtime/python && uv run pytest && uv run ruff check . && uv run ruff format --check .`. ty installed but not gated.
- **CUE:** `bash scripts/cue-vet` (validates all schemas against all fixtures).
- **Submodules:** `git submodule update --init --recursive` after clone or pull.

### B.8 Tech-stack-as-policy (~30)

A surprising body of rules that *encode* the tech stack as an invariant: the runtime is Elixir/OTP umbrella; the Python ops run via Erlang `:port`; CUE is the schema language at v0.16.1; uv is the Python package manager; observability rides ex_a2ui. Many of these are descriptive, but the prose binds the AI not to deviate.

### B.9 Phases & status (~10)

- Phase 5c is active; Radar hardening in progress; VSME (first compliance pack) is next.
- D-012 / D-013 / etc. — specific decisions logged with named scopes.
- Forward-deferred work: ADR-EVOLUTION-01 (schema compatibility), ADR-MULTIPLAN-01 (multi-plan), M-RUNTIME-02 (provenance).

### B.10 Other project-named rules (~30)

Naming conventions for everything; specific paths; specific tools; specific submodule URLs.

---

## 4. Bucket C — Workflow / PM (aiwf-domain) (~70)

Rules that the framework provides — ones aiwf would want to be the source of for any consumer repo.

### C.1 Entity lifecycle (~15)

- **Epic** (`E-NN[letter]-slug`) → milestones inside.
- **Milestone** (`M-TRACK-NN`) → ACs, tracking doc, status states.
- **ADR** (`NNNN-slug.md` in `docs/decisions/`) → Nygard form.
- **Decision** (`[D-NNN]` in `work/decisions.md`) → append-only shared log.
- **Gap** (`work/gaps.md`) → discovered work not yet scheduled.

### C.2 Two-surface decisions (~5)

- **`work/decisions.md`:** lightweight, append-only, day-to-day calls. Indexed `[D-NNN]`.
- **`docs/decisions/NNNN-slug.md`:** heavy ADRs for architectural commitments.
- **Both live, no conflict** — each serves its surface.

### C.3 Roadmap & graph (~10)

- **`work/roadmap.md`:** single source of truth for sequencing.
- **`work/graph.yaml`:** machine-generated graph; CI-validated by `wf-graph`.
- **Roadmap and graph stay in sync.** CI fails on error-severity findings, permits warnings.

### C.4 Tracking docs (~10)

- **One tracking doc per milestone** (`work/milestones/tracking/<epic>/<milestone-id>-tracking.md`).
- **Acceptance criteria checked off as TDD cycles complete.**
- **Decisions and deviations recorded in tracking doc as encountered.**
- **Wrap-milestone produces a release summary** (`work/releases/<milestone-id>-release.md`).

### C.5 Per-role agent history (~5)

- **`work/agent-history/<role>.md`** holds accumulated learnings; read-only by other roles, write-only by the owning role.
- **Heartbeat pattern for long-running agents:** ISO-8601-timestamped progress markers so parent doesn't think the subagent is stuck.

### C.6 Sync mechanism (~10)

- **Edit `.ai-repo/`, run `./.ai/sync.sh`** to update generated AI surfaces. Don't hand-edit `.claude/`, `.codex/`, `.github/skills/` (except CLAUDE.md Current Work, which is preserved).
- **Generated surfaces mirror config**, never redefine values.
- **Refresh context on config change** — when rules change mid-session, full re-read.

### C.7 Session start checklist (~10)

- **Read `work/decisions.md`, `work/agent-history/<role>.md`, `work/gaps.md`, CLAUDE.md Current Work.**
- **Identify-agent-first** — read the agent file at session start; adopt its role.

### C.8 Verb-set (~10)

The skill names imply a verb-set: `plan-epic`, `plan-milestones`, `start-milestone`, `tdd-cycle`, `review-code`, `wrap-milestone`, `wrap-epic`, `release`. Each binds a sequence of activities.

---

## 5. Bucket D — Agent-behavior (~90)

Rules about *how AI agents operate*. This is its own bucket because it's neither the codebase, the project, nor the workflow per se — it's the operating discipline of the AI itself. (User said "rest"; this is what surfaced as distinct enough to name.)

### D.1 Q&A mode (~5)

- **When user says "Q&A":** structured decision-making. Context paragraph → pros/cons per option → lean → numbered options (usually 3, lean marked). One question at a time. Post-decision: execute as picked, no follow-up flourish.

### D.2 Agent routing (~10)

- **Intent → agent.** builder (build/implement/fix), planner (plan/design/scope), reviewer (review/validate), deployer (release/deploy).
- **Don't cross roles.** Planner doesn't run `start-milestone`. Reviewer doesn't write code.

### D.3 Subagent discipline (~20)

- **`Explore` at `quick` thoroughness by default.** Escalate only when `quick` leaves a real gap.
- **`general-purpose` with `model: "sonnet"`** for research / lookup / WebFetch / WebSearch.
- **Keep parent agents on parent (Opus) model** — architectural and review judgments carry irreducible load.
- **Heartbeat pattern** for long-running subagents.

### D.4 Skill invocation (~15)

- **Skills are lazy-loaded on invocation**, not at session start.
- **Skill names in `wf-` prefix are framework**; bare names are project.
- **Skills compose** — `wrap-milestone` calls into review patterns; `start-milestone` calls into TDD setup.

### D.5 Cost discipline (~10)

- **Don't use Opus for what Sonnet can do.**
- **Don't spawn subagents for trivial lookups** — use Read / Grep directly.
- **Parallelize independent calls** in one message.

### D.6 Per-role responsibilities (~30)

The four agent files (builder, planner, reviewer, deployer) each define focus, key skills, inputs needed, outputs produced, handoff phrasing. About 7-12 rules per agent.

---

## 6. Bucket E — Rest (~25)

What didn't fit elsewhere.

- **Operational guides** (`docs/guides/devcontainer_operations.md`, `elixir_tooling.md`, `python_tooling.md`, `pack_design_and_development.md`) — how to operate the dev environment. Mostly procedural, not normative.
- **CHANGELOG, LICENSE** — present, conventional, not policy-shaped.
- **`metrics.json`** — ephemeral.
- **Status-phase narrative** in README.
- **VSCode / devcontainer settings** — environmental, not project rules.

---

## 7. Cross-cuts: the same corpus, viewed differently

### 7.1 By enforcement rung (after `30-enforcement-mechanisms.md`)

| Rung | What enforces it | Approx % of corpus | Notes |
|---|---|---|---|
| **0 (forgotten next session)** | nothing | 0% | None visible — the existence of CLAUDE.md kills this rung. |
| **1 (markdown / prose)** | CLAUDE.md, .ai/rules.md, .ai-repo/rules/*, ADR prose, governance/*, skills | ~75% | The dominant rung. *Most rules in this repo are prose-only.* |
| **2 (pattern lint)** | Credo, Quokka, Ruff, golangci-lint, markdown reflow scripts | ~10% | Only language-formatting + style. |
| **3 (schema / type)** | CUE × 5, Dialyzer, ty, TS strict | ~10% | The CUE schemas are the strongest mechanical layer. |
| **4 (runtime / test)** | ExUnit, pytest, golden fixtures (cross-language hash), property tests, wf-graph CI | ~5% | Tests-as-policy: the property tests *are* invariant assertions. |
| **5 (formal proof)** | none | 0% | As expected. |

The salient observation from the parent exploration §4 holds: **the rung mismatch is real here.** Many MUST rules sit at rung 1 only. Examples:
- *"Compatibility shims are banned by default"* — rung 1 only. No grep gate.
- *"Doc-tree taxonomy: bind-me vs inform-me"* — rung 1 only. The directory layout *encodes* it but no check refuses a misfiled doc.
- *"Contract-matrix rows verified at wrap"* — rung 1 only. The reviewer is supposed to check.
- *"Live source paths in matrix kept current"* — rung 1 only. No automated verify-paths-resolve check.
- *"NEVER commit without explicit human approval"* — rung 1 only. Mirrored five times in prose, but nothing in `.git/hooks` enforces it (the AI's own discipline is the gate).
- *"Branch coverage audit before commit"* — rung 1 only. No coverage tool wired in.

The CUE layer is the bright spot. The five named contract surfaces are rung 3 with rung 4 fixture libraries — exactly the shape `policy-substrates-and-execution.md` §4 prescribes (CUE for static-shape policies + runners for dynamic ones).

### 7.2 By substrate fit (per the substrate doc §4)

If we tried to express the corpus in the candidate top-layer schema, what would each policy point at?

| Substrate | Count fit | Examples |
|---|---|---|
| **CUE body** (static-shape) | ~30 (and the 5 existing) | All schema-shaped rules. Frontmatter required-keys check fits naturally. Pack-id regex fits. The contract-matrix row schema would fit. |
| **EARS prose + runner** | ~80 | Most "MUST/SHOULD" rules with a clear trigger and observable evidence. *"When a milestone is wrapped, the system MUST verify all contract-matrix rows resolve to existing files."* — event-driven EARS, runner = path resolver. |
| **Code-as-policy (Go func / Elixir mod / Python script)** | ~40 | wf-graph validators. cue-vet schema-evolution loop. "The roadmap graph must be acyclic" is already this. |
| **Rego (capability-shaped)** | ~5 | *"Only humans may invoke `--force`"* class. Approval gates. |
| **TLA+ / formal** | 0 | None. |
| **Pure prose, no runner** | ~150 | Truth-model rules, doc-tree philosophy, agent-routing prose, Q&A protocol. Fundamentally judgment-driven. |
| **Doesn't fit (descriptive)** | ~50 | Tech-stack descriptions, status-phase narrative, operational guides. |

**Observation:** the bulk of the corpus is *prose with no runner*. That matches the parent doc's claim: rung 1 is necessary even when rungs 2–5 exist, and many policies *want* to live there. But the substrate doc's claim — that EARS+runner is a cheap upgrade for SHOULD/MUST prose — also lands: about 80 rules in this corpus could trivially become EARS-form with a small runner attached.

### 7.3 By bindingness gradient (RFC 2119)

Aggregated across both mining files (the corpus uses MUST / SHOULD / MAY casually rather than as RFC 2119 keywords):

- **MUST:** ~38% — the hard rules
- **SHOULD:** ~52% — the conventions  
- **MAY:** ~8%
- **MUST NOT:** ~2%

**The MUST-heavy distribution is suspicious.** Per the parent doc §4, *"most policy failures escape because the rung is too soft for the bindingness claimed."* This corpus has a lot of MUST-at-rung-1 rules. Some of them have to drop to SHOULD honestly (because rung-1 SHOULD is what the policy *can deliver*), or rise to rung 2+ enforcement (and *then* keep their MUST).

---

## 8. What this teaches about the policy primitive

Pulling the four buckets and the cross-cuts together, here's what the corpus suggests about the design questions in `policies-design-space.md` §13.

### 8.1 The umbrella earns its keep — for some categories more than others

- **Bucket B (project-specific) is where the policy primitive shines.** Of the 180 rules, ~30 are already mechanized as CUE; another ~80 are EARS-shaped and could be mechanized cheaply. The shared frame (provenance, lifecycle, supersedes, waivers) is exactly what's missing. Today these rules are scattered across `.ai-repo/rules/`, `docs/governance/`, ADR prose, and skill bodies. A policy entity with a ratified status, a why, and a runner pointer would consolidate them.
- **Bucket A (general engineering) doesn't need an umbrella as much.** The existing tools already enforce most of these. The gap is consistency: *"all repos using aiwf get the same TDD discipline"* is the cross-project portability question, not a per-repo policy question.
- **Bucket C (workflow/PM) is what aiwf already does** — these rules *are* the framework. They don't need to be policies; they're verbs / state-machines.
- **Bucket D (agent-behavior) is interesting.** These are skill-bodied rules today (lazy-loaded into context). Whether they want to be "policies" or stay as "skills" is a real design question. The parent doc's note — *"skills are not policies; they are advisory documentation, but a skill that asserts 'you must do X' is straying into policy territory"* — applies here: a lot of skill bodies in `.ai/skills/` *are* asserting MUSTs. The honest split is: skills carry *how to do the thing*, policies carry *that the thing must be done*.

### 8.2 Two distinct "policy populations" — a bifurcation worth naming

What this corpus suggests is the umbrella isn't one shape — it's two:

1. **Project-shape policies** (Bucket B): describe *the thing being built*. Truth model, doc-tree, contract matrix, schema invariants. Often one-per-project, often supersedable as the project evolves. Mostly authored by humans, ratified by humans, applied to AI + humans both. **CUE-bodied or EARS-bodied with runner.**
2. **Engineering-discipline policies** (Bucket A + parts of D): describe *how building is done*. TDD, commits, branch coverage, formatting. Often shared across repos, often stable across years. Fundamentally code-as-policy or test-as-policy. **Code-bodied or runner-pointer.**

The substrate doc's **Option α (CUE for static-shape, code-as-policy escape hatch)** maps cleanly onto this: project-shape policies fit CUE; engineering-discipline policies fit code-as-policy. One umbrella, two body shapes, with a discriminator.

### 8.3 The "skills as policies" question

The user asked: *"based on the type of things we are doing and the technology stack, we try to establish the skills we need which could be policies."*

Walking the corpus with that lens, the skills/policies that fall out of *this* tech stack are:

| Stack element | Skill / policy it suggests | Substrate |
|---|---|---|
| Elixir (umbrella, OTP) | TDD-cycle (with branch coverage audit), Credo+Quokka discipline, Dialyzer gating (currently missing) | code-as-policy + EARS prose |
| Python ops via `:port` | Cross-language fixture parity (golden hashes), Ruff gating, ty gating | code-as-policy |
| CUE schemas | Schema-evolution loop (every fixture validates against HEAD), invalid-fixture coverage, bundle-as-contract discipline | CUE itself + code-as-policy runner |
| Submodules (5) | Submodule update on clone/pull, version-pin discipline, two-pack rule | EARS prose |
| `work/` planning layer | Roadmap-canonical, two-surface decisions, agent-history-per-role | code-as-policy (wf-graph already does roadmap) |
| `docs/` register split | Bind-me vs inform-me, frontmatter required-keys, `truth_class` enum | CUE (frontmatter) + grep-for-misfiled-files |
| Multi-agent operation | Agent routing, Q&A protocol, heartbeat, model selection | prose only — judgment-driven |
| Conventional commits + approval gate | Commit-message format check, approval-trailer check | code-as-policy + capability gate (Rego-shaped) |
| ADR + decision log | ADR-supersedes acyclic, decision-id monotonic, decision/ADR cross-link | code-as-policy (already exists in PoC) |
| Contract-matrix | Row-paths-resolve-to-files, plan-time declaration, wrap-time check | code-as-policy + EARS-shaped trigger |

Of these, the *highest-leverage policies the framework could ship* — based on this one repo — would be:

1. **Bundle-as-contract** (already a skill `design-contract`, but no enforcer) — verify ADR + schema + fixtures + worked example + ref impl exist together; reject TBD ref impl.
2. **Schema-evolution loop** (script exists; framework could generalize). Every committed fixture validates against current HEAD schema.
3. **Frontmatter required-keys** (rung 1 today; trivial CUE schema).
4. **Contract-matrix row-paths-resolve** (rung 1 today; trivial code check).
5. **Compatibility-shim with named-trigger** (rung 1 today; could be a grep-for-removal-comment + tracking-doc cross-ref).
6. **Two-pack-citation rule** (rung 1 today; could be a markdown-link parser over ADRs).
7. **Branch coverage audit** (rung 1 today; needs language-specific runner — Elixir cover, Python coverage.py).

Each of these is ~one runner and ~one EARS sentence. Several already have *partial* implementations the framework could generalize.

### 8.4 What's notably absent

Three things the corpus does *not* contain that the policy literature would expect:

1. **Performance contracts.** No latency, memory, throughput rules. The system handles soft-real-time observability but no SLO is policy-encoded.
2. **Security postures.** `gosec` runs in Go; nothing comparable in Elixir/Python. No secret-scanner. No SAST.
3. **Capability-gating beyond the commit-approval rule.** No Rego-shaped *"only X principal may Y action on Z resource"* rules. The closest is the human-approval gate, which is a single recurring rule.

The absence is informative: **a single repo's policy population is dominated by structural and discipline rules**, with capability-shaped rules thin on the ground. Cedar-shaped policies are real but small in volume; CUE-shaped + EARS-shaped policies are the bulk. The substrate doc's Option α (CUE + code-as-policy escape hatch, skip Cedar) reads as right-sized for this kind of repo.

---

## 9. The numbers, summarized

- **Raw extraction:** 625 candidates across three mining passes (338 AI scaffolding, 287 project docs, plus the enforcement-layer mechanisms, which catalogued *enforcers* not *rules*).
- **Unique rules after de-dup:** ~350-400.
- **Bucket sizes:** ~140 general / ~180 project-specific / ~70 workflow / ~90 agent-behavior / ~25 rest.
- **Currently mechanized (rung 2+):** ~25% of unique rules. The CUE layer carries the heaviest load.
- **EARS-eligible (could be mechanized cheaply):** another ~25%.
- **Pure prose / judgment-driven:** ~50%. Many of these can't be mechanized — they're judgment, and that's fine.

The picture: **Liminara is rich enough to populate a policy substrate experiment.** The CUE backbone is real, the prose layer is dense, and the gap between MUST-claimed and rung-1-enforced is exactly the gap the policies-design-space doc names as the typical failure mode. A PoC that picks ~10 of the policies above and renders them through the Option α (CUE + code-as-policy + EARS) shape would have a defensible body of evidence for the kernel-level decision.

---

## 10. What's in the scratch directory

```
.scratch/liminara/
├── 00-survey.md                    structural survey (477 lines)
├── 10-ai-scaffolding-policies.md   338 candidates from CLAUDE.md, .ai/, .ai-repo/, agents, skills
├── 20-project-docs-policies.md     287 candidates from governance, ADRs, CUE schemas, guides, roadmap
├── 30-enforcement-mechanisms.md    15 enforcers across 4 rungs, with gaps
├── 40-categorized.md               this file — the four-bucket categorization + cross-cuts
└── README.md                       top-level summary (separate, for fast read)
```

Sanitization note: every file under `.scratch/liminara/` cites `liminara` by name and references its directory layout. If anything from this corpus lands in a public PoC artifact, the sanitization pass should rename `liminara` → `<consumer>`, generalize `Radar` / `admin-pack` / VSME, and replace named CUE schemas with one or two representative example schemas. The shape transfers; the content does not.
