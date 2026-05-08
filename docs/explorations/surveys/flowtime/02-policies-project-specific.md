# Project-specific policies — only meaningful inside FlowTime

> Rules whose subject is a FlowTime concept (engine, model schema, NaN handling,
> flow authority, run provenance, port topology, deprecated surfaces). They would
> not transfer cleanly to another repo without translation. They are also the
> rules with the **most concrete enforcement** — schemas, validators, grep guards,
> integration tests — because they have something specific to bind to.

---

## A. The model schema and its surrounding contracts

This block is the "core kernel" of project-specific policy. Most items have rung-3
enforcement (schema validator) and rung-1 documentation (the schema doc).

### P-1. Model schema is canonical, single-source-of-truth post-substitution
- **Source:** `docs/schemas/README.md` ("One model, one schema, one validator").
- **Rung:** 3 (`ModelSchemaValidator` enforces; integration tests verify).
- **Notes:** Post-E-24 invariant. The C# source of truth is `ModelDto`; the structural contract is `model.schema.yaml`; the runtime validator is `ModelSchemaValidator`. Three-layer pin: type, schema, validator — matches the design-space §7 shape A + B + C model exactly.

### P-2. Model fields are camelCase throughout (including provenance)
- **Source:** `docs/schemas/README.md`; `CLAUDE.md` ("never reintroduce snake_case fields").
- **Rung:** 3 (schema-validated).

### P-3. `binMinutes` is a deprecated schema field; never reintroduce
- **Source:** `CLAUDE.md` ("current schema uses `{ bins, binSize, binUnit }`").
- **Rung:** 1 (prose) + 2 (validator rejects). Pre-existing drift in `tests/FlowTime.Tests/ApiIntegrationTests.cs:93` (caught by dead-code report) shows rung 2 is incomplete.

### P-4. Top-level model has no `window:`, `generator:`, `mode:`, `metadata:` keys
- **Source:** dead-code report ("Leaked-state schema fields absent").
- **Rung:** 1 (asserted by absence test, paraphrased) + grep evidence.
- **Notes:** Negative space — the schema is closed; these top-level keys would be drift. Rung-2-by-grep is in place.

### P-5. Provenance is camelCase and embedded in the unified model shape (E-24 unification)
- **Source:** `docs/schemas/README.md`; `model.schema.yaml`.
- **Rung:** 3.
- **Notes:** Pre-E-24 was snake_case + sidecar; deliberate semantic shift, no migration ("forward-only" — see P-25).

### P-6. `Template` (authoring-time, pre-substitution) is a separate contract
- **Source:** `docs/schemas/README.md`.
- **Rung:** 3 (`TemplateSchemaValidator`).
- **Notes:** Two contracts, not one. Worth lifting as a pattern: when a substitution boundary exists, it justifies two schemas.

### P-7. Sim emits `ModelDto` shape directly; Engine accepts the same shape
- **Source:** `docs/schemas/README.md`.
- **Rung:** 3 (cross-surface integration tests).

### P-8. Telemetry-run manifest provenance is *intentionally* snake_case (vs model-schema camelCase) — or it is drift
- **Source:** Dead-code report flag at `FileSeriesReaderTests.cs:132-133` ("Question for human: is the telemetry-run manifest schema deliberately snake_case while the model schema is camelCase, or has the manifest schema drifted out of step with E-24's camelCase convergence?").
- **Rung:** 0 (open question — needs a decision).
- **Notes:** This is exactly the kind of policy gap the framework should help surface. Today the dead-code-audit skill *flagged* it as needs-judgement; the framework has no place to *record the decision* once made.

### P-9. Schema validation errors return `400 Bad Request` with a JSON body naming the offending field
- **Source:** `docs/schemas/README.md` ("Error handling").
- **Rung:** 2 (integration tests).

### P-10. Array parameters declare element type via `arrayOf` (`double` default; `int` supported); length and per-element min/max enforced
- **Source:** `docs/schemas/README.md`.
- **Rung:** 3.

---

## B. Numeric / floating-point policy (the NaN policy)

This block is one of the cleanest examples of a fully-developed engineering policy
in the corpus. It has a name, a rationale, three tiers, a per-site enforcement
table, and a "how to add a new site" workflow.

### P-11. Three-tier NaN policy: Tier 1 returns 0.0 (no activity); Tier 2 returns null (metric unavailable); Tier 3 NaN sentinel (data not provided); exception for invalid PMF (programming error)
- **Source:** `docs/architecture/nan-policy.md`.
- **Rung:** 2 (`tests/FlowTime.Core.Tests/Safety/NaNPolicyTests.cs` per-tier coverage).
- **Notes:** Rare example of an explicitly-named, externally-justified, table-driven policy in production .NET code. **This is what a "policy entity" should look like in shape A.**

### P-12. Tier 1 sites never produce NaN or Infinity; output always finite `double`
- **Source:** `nan-policy.md` ("Key invariant").
- **Rung:** 2 (test invariant per site).

### P-13. Tier 2 sites return `double?`; callers must handle `null` (omit metric or display "N/A")
- **Source:** `nan-policy.md`.
- **Rung:** 3 (type system).

### P-14. Tier 3 NaN values are created at ingestion boundaries (CSV cells, missing constraint data)
- **Source:** `nan-policy.md`.
- **Rung:** 2 (per-site test).

### P-15. `Pmf` constructed with all-zero probabilities throws `ArgumentException` (programming error)
- **Source:** `nan-policy.md`; `Pmf.cs`.
- **Rung:** 2 (test) + 3 (the throw is the gate).

### P-16. New division/modulus/ratio sites: determine tier, add guard immediately before the division, add test, update the policy doc
- **Source:** `nan-policy.md` ("Adding New Division Sites").
- **Rung:** 1 (workflow prose).
- **Notes:** This is a **process policy embedded in an engineering policy doc**. Worth lifting as its own thing — see [03-policies-workflow.md](03-policies-workflow.md).

---

## C. Flow-authority policy (E-25, in-flight)

Worth its own subsection because it is a live worked example of policy ratification.
The work itself is structured exactly the way the design-space doc would predict.

### P-17. Routing authority is declared at the producer's outgoing edges
- **Source:** `M-066-edge-flow-authority-decision.md`.
- **Rung:** Currently 0 (in proposal). Slated for ADR ratification + schema + compile + analyse enforcement (M-066/M-067/M-069).
- **Notes:** The team explicitly named the rungs in advance: schema rejection, compile-time fan-out detection, two new analyser warnings. This is rung-aware policy design.

### P-18. For static-share fan-outs (class 3), edge weights are normative
- **Source:** `M-066`.
- **Rung:** 0 (proposed). Slated for rung 3 (engine implementation).

### P-19. Class-1 dynamic routing requires `kind: router` nodes; class-2 capacity-aware allocation is not surfaced today
- **Source:** `M-066`.
- **Rung:** 0 (proposed).

### P-20. Consumer-side expr arithmetic must not encode peer-relative splits
- **Source:** `M-066` (the structural argument: such models are not telemetry-replayable, not invariant under peer addition, not robust to capacity-aware allocation).
- **Rung:** 0 → 3 (planned schema rule, compile-time detector, analyser warning).
- **Notes:** A rare *negative* policy with a strong "why" — three independent reasons it must be banned. The why is the durable part; the schema rule is the implementation.

### P-21. Exactly one routing authority per producer-fan-out point
- **Source:** `M-066` ("AC-6 — ADR names the schema/compile/analyse enforcement points").
- **Rung:** 0 → 2 (planned compile-time error if 0 or >1 routing authorities detected).

### P-22. `edge_flow_mismatch_incoming` / `_outgoing` warnings fire when expr-layer arrivals diverge from edge-weight apportionment
- **Source:** `InvariantAnalyzer.cs:323-335`; G-032; M-066 context.
- **Rung:** 2 (live; the warning fires).
- **Notes:** Pre-E-24 this didn't fire because the engine wasn't writing per-edge `flowVolume` series. Post-E-24 it fires correctly. The team treats the *new firing* as evidence of a real underlying inconsistency, not as a bug — exactly the design-space §11 "stale specs mislead" pattern, surfaced as policy.

---

## D. Run provenance and artifacts

### P-23. Engine is the single source of truth for all artifacts (models + runs + telemetry); Sim provides temporary storage for UI workflows
- **Source:** `docs/architecture/run-provenance.md`.
- **Rung:** 1 (architecture doc).
- **Notes:** Cross-surface architectural policy. No mechanical enforcement; relies on developer discipline + code review.

### P-24. UI orchestrates the workflow — Sim and Engine do NOT communicate directly
- **Source:** `run-provenance.md`.
- **Rung:** 1.
- **Notes:** Same as above. A test that asserts "no HTTP client to Sim's URL exists in Engine source" could move to rung 2 cheaply.

### P-25. Forward-only regeneration when a runtime boundary changes — prefer over compatibility readers
- **Source:** `CLAUDE.md` ("When a runtime boundary changes, prefer forward-only regeneration of runs, fixtures, and approved outputs over compatibility readers that recover missing facts").
- **Rung:** 1.
- **Notes:** Strong project-level migration policy. Comes up repeatedly in E-24 ("existing stored bundles from before m-E24-02 are obsolete — forward-only — no migration").

### P-26. Provenance is optional and backward-compatible; no provenance.json if not provided
- **Source:** `run-provenance.md`.
- **Rung:** 2 (integration tests).

### P-27. `inputHash` captures deterministic fingerprint of template + parameters + telemetry bindings + RNG seed
- **Source:** `run-provenance.md`.
- **Rung:** 2 (test of hash determinism).
- **Notes:** The exact rung-2 enforcement the design-space doc identifies as ideal: hash-verifiable provenance.

### P-28. Provenance parameters stored as JSON strings (not typed numbers)
- **Source:** `run-provenance.md` ("Parameter Type Preservation").
- **Rung:** 2 (tests). Documented limitation; two failing tests are acknowledged.
- **Notes:** A *pragmatic-trade-off* policy — "we accept type loss because the alternative is hard." The doc names the alternative considered (raw YAML string) and rejects it. Provenance in the design-space sense.

### P-29. Header (`X-Model-Provenance`) takes precedence over embedded YAML provenance if both present
- **Source:** `run-provenance.md`.
- **Rung:** 2 + log warning if both present.

### P-30. Canonical run artifact layout: `run_<timestamp>/{model/, series/, run.json, manifest.json}`
- **Source:** `run-provenance.md` ("Canonical Artifact Layout").
- **Rung:** 2 (`RunArtifactWriter` enforces).
- **Notes:** Tree-shape policy with code as enforcer.

---

## E. Truth-discipline guards (project-specific specializations of general rules)

### P-31. When a milestone explicitly owns a bridge or cleanup seam, do not preserve the bridge helper past that milestone as a tolerated coexistence state
- **Source:** `CLAUDE.md` ("Truth Discipline > Guards").
- **Rung:** 1.
- **Notes:** Specializes G-34 (no-temporary-shims) to milestones. The grep guards in `work/guards/` (P-44+) are the rung-2 implementation of this.

### P-32. Do not reconstruct semantic or analytical identity in adapters or clients from `kind`, `logicalType`, file stems, or similar heuristics when compiled/runtime facts can own that truth
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** Strong, repo-specific guard against a recurring code smell. Rung-2 enforcement would need an LLM-as-linter pass.

### P-33. Do not keep both a bridge abstraction and its compiled replacement once the replacement milestone is active unless the spec explicitly allows a coexistence window
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### P-34. "API stability" does not mean "keep old functions around." When a function has no production callers after a refactor, delete it and its tests in the same change
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (prose). The dead-code-audit skill is the rung-2 surfacer; deletion is human-driven.

### P-35. Do not let adapter/UI projection become the only place where semantics exist
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

---

## F. Port topology and devcontainer (project-specific instances)

### P-36. Default ports: 8081 Engine API, 8090 Sim API, 5173 Svelte UI, 5219/7047 Blazor, 8091 Sim diagnostics, 5091 Engine dev profile
- **Source:** `CLAUDE.md` ("Build & Run").
- **Rung:** 1 (prose) + 2 (devcontainer config; per-spec `baseURL` overrides).

### P-37. Never blindly kill all processes on port 8081 — devcontainer port-forwarder listens there
- **Source:** `CLAUDE.md` ("Devcontainer Port Safety").
- **Rung:** 1 + 2 (the `kill-port-8081` task filters safely).
- **Notes:** Specific to FlowTime; G-42 is the generic version.

### P-38. To free port 8081, filter by process name: only kill `dotnet` processes
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

---

## G. Branch / version conventions (project-specific instances)

### P-39. Epic integration branches use `epic/E-{NN}-<slug>`; milestone branches `milestone/<milestone-id>`; feature branches `feature/<surface>-<milestone-id>/<desc>`
- **Source:** `CLAUDE.md` ("Branching & Versioning").
- **Rung:** 1 (convention) + 2 (any branch-name-pattern check could elevate).

### P-40. Single-surface quick changes can branch from `main` and PR back to `main` when no milestone integration branch is needed
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### P-41. Version format `<major>.<minor>.<patch>[-pre]`; milestone completions typically bump minor
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### P-42. Release notes in `docs/releases/` with milestone-based naming (e.g., `SIM-M2.7-v0.6.0.md`)
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

---

## H. Deletion-stays-deleted (the grep-guards pattern)

A whole pattern lives here: when a milestone deletes symbols, a shell script asserts
they stay deleted across `src/` and `tests/` (excluding `docs/` and `work/` so
historical surfaces can still reference the deleted concepts). The guards are
milestone-named (`m-E19-02-grep-guards.sh`, `m-E19-03-grep-guards.sh`,
`m-E19-04-grep-guards.sh`).

### P-43. Every deleted symbol from a deletion-milestone stays deleted in `src/` and `tests/`
- **Source:** `work/guards/m-E19-0{2,3,4}-grep-guards.sh`.
- **Rung:** 2 (`rg`-based shell guards; documented "exits 0 if every guard passes, 1 otherwise").
- **Notes:** Beautiful textbook example of rung-2 enforcement coupled to a milestone identifier. **This is the policy shape the framework should formalize.**

### P-44. When a guard's pattern would be too broad (matches legitimate uses), drop the guard rather than allowlist (which would hide real regressions)
- **Source:** `m-E19-02-grep-guards.sh` AC2 comment block (literally explains why one guard was dropped).
- **Rung:** 1.
- **Notes:** Rare meta-rule about how to *design* guards. Worth lifting.

### P-45. Per-guard allowlist for known-good matches in specific files
- **Source:** `m-E19-02-grep-guards.sh` (`allowed_paths` array).
- **Rung:** 2 + 1 (the comment naming the rationale).

### P-46. Fail fast if `rg` is missing on PATH (otherwise the script silently passes everything)
- **Source:** `m-E19-02-grep-guards.sh` (literally says "Without this check, `rg ... || true` would silently return empty output for every guard and falsely report PASS — which would make the whole script a no-op").
- **Rung:** 2 + 1 (incident-derived).
- **Notes:** **Exemplar of why "tool-failure-is-a-finding" matters as a meta-policy.** A check that silently passes is worse than no check.

### P-47. Each guard scope: `src/` and `tests/` only — exclude `docs/`, `work/`, archive locations
- **Source:** `m-E19-02-grep-guards.sh` header comment.
- **Rung:** 1 (rationale: "historical/documentation surfaces can and should still reference the deleted concepts — only runtime code and tests must stay clean").
- **Notes:** A *scoping* policy about a *guard policy*. Recursive policies are a real shape.

### P-48. Deferred-scope exceptions are documented inline (e.g., AC6 `/v1/run` deletion deferred per `D-2026-04-08-029`)
- **Source:** `m-E19-02-grep-guards.sh` header comment.
- **Rung:** 1 (the comment) + 2 (the deferred deletion is tracked elsewhere).

---

## I. Existence-by-absence policies

### P-49. Solution-wide absence of `binMinutes` schema field references is a target state
- **Source:** dead-code report; CLAUDE.md.
- **Rung:** 2 (grep evidence in the dead-code report).

### P-50. Solution-wide absence of `SimModelArtifact`, `SimNode`, `SimOutput`, `SimProvenance`, `SimTraffic`, `SimArrival`, `SimArrivalPattern` (E-24 cleanup)
- **Source:** dead-code report.
- **Rung:** 2 (evidence) but no committed grep guard yet — would benefit from one.

### P-51. Solution-wide absence of `Template Legacy*` aliases (E-24 cleanup)
- **Source:** dead-code report.
- **Rung:** 2 (evidence).

---

## Cross-cut observations on the project-specific bucket

1. **The schema + validator + tests stack is the rung-3 backbone.** Anything that can be expressed as a closed grammar (model schema, template schema, manifest schema) gets the strongest enforcement. This matches the design-space §9 observation that contracts work brilliantly for closed grammars and strain at intent.
2. **The NaN policy is the gold-standard "policy entity" in this corpus.** It has a name, three explicit tiers, a per-site enforcement table, a how-to-add-a-new-site workflow, and tests per tier. If the framework's policy primitive needs a single referent to design *for*, it is this doc.
3. **The flow-authority work shows the framework's gap.** The team is *manually* coordinating: prose ADR + schema rule + compile-time detector + analyser warning + repo-wide doc sweep. Each of the four enforcement layers is tracked in a separate AC; they could trivially diverge. A policy primitive that bound them together as one entity with multiple `enforces[]` runners would directly address this.
4. **Forward-only migration is a strong, repeated FlowTime stance.** P-25 + P-50 + P-51 + the E-24 narrative all reinforce this. The framework should have a "migration mode" verb: `policy supersede --forward-only` vs `--with-coexistence-window`.
5. **The grep-guards directory is unrecognized prior art for the framework.** It is exactly the rung-2 enforcement the framework would want for "deletion-stays-deleted" policies. Today, FlowTime has hand-rolled bash scripts; the framework could provide a verb (`aiwf check guards` or similar) that generates and runs them from a declarative spec.
6. **There are open policy decisions sitting in the dead-code report (P-8) with no ratification path.** "Is the snake_case telemetry-manifest deliberate?" is a question that should turn into an ADR or a decision; today it is buried in a generated report and will rot.
