# Agent orchestration — design synthesis

This is the substrate underneath specific agent-using features (TDD cycles, doc gardening, security audits, code review, …). It captures the agent registry, role-based concurrency, sub-scope provenance, declarative per-epic work-shapes, forensic bundles + permanent provenance, and the strict-lane scope boundary that keeps aiwf as a data + audit layer rather than an orchestration framework.

Companion artifacts:

- [`_scratch-subagents-research.md`](_scratch-subagents-research.md) — Q&A trail showing how each decision got made (branches considered, alternatives rejected). Forensic detail behind this doc.
- [`parallel-tdd-subagents.md`](parallel-tdd-subagents.md) — original (narrower) design for TDD-cycle subagents. After canonical updates, will be restructured as a consumer of this substrate or absorbed entirely.
- [`ADR-0003`](../../adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — finding (F-NNN) as 7th entity kind.
- [`ADR-0004`](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — uniform archive convention.
- [`ADR-0009`](../../adr/ADR-0009-orchestration-substrate-vs-driver-split.md) — ratifies the three load-bearing decisions in this design doc (substrate-vs-driver split per §6.1, trailer-only cycle events per §6.3, isolation as parent-side precondition per §6.2 / §7.7). Currently `proposed`; iterates as this doc's design is shaped.
- [`G-0099`](../../../work/gaps/G-0099-worktree-isolation-parent-side-precondition.md) — the concrete failure mode that surfaced the isolation-as-precondition rule (§6.2 step 3, §7.7).

---

## 1. Motivation

The proximate trigger is **M-0066/AC-0001**, where a long implementation session lost track of branch-coverage discipline mid-cycle. The TDD-cycle skill (`wf-tdd-cycle`) was advisory text — easy to drift through under the pressure of a long conversation. The retrospective finding: the framework's correctness can't depend on the LLM remembering rules over many turns. That's the kernel principle from `CLAUDE.md` ("framework correctness must not depend on LLM behavior") applied to the TDD cycle itself.

The proposed fix isn't a stricter skill prompt; it's a **structural** one: bound the cycle's lifetime to a subagent invocation. A subagent that starts fresh, sees only the task contract + the relevant code, and returns when done can't drift the way a long conversation can. The protocol *is* the lifetime of the agent.

That insight generalizes beyond TDD. *Any* unit of agent work — code review, security audit, doc gardening, refactor pass — benefits from the same bounded-lifetime model. So the TDD-cycle case is one consumer of a more general substrate. This document specifies that substrate.

The substrate has to answer four questions in a way that respects the existing kernel principles:

1. **How do we represent agents structurally** — so that "the framework knows about a security-auditor" is config + markdown, not a kernel recompile?
2. **How do we constrain what an agent can do** — substrate-level, not just by prompt obedience?
3. **How do we declare the work-shape of an epic** — so a multi-step pipeline is auditable and the kernel can mechanically reconcile intent vs. fact?
4. **How do we capture and retain forensics + provenance** — without turning aiwf into an event-log database or analysis platform?

The rest of this document answers each of those.

---

## 2. Principles (substrate-specific)

These extend the existing kernel principles in [`CLAUDE.md`](../../../CLAUDE.md) and [`design-decisions.md`](design-decisions.md). Each is load-bearing for what follows.

### 2.1 Strict lane

> Aiwf provides primitives — agent/capability registry, audit-trailer schema, verb gate, check rules, reconciliation. Aiwf does **not** invoke LLMs, interpret pipelines, or own multi-step control flow. The orchestrator is external (a host-specific skill, a deterministic sibling binary, or an external script). New hosts get new orchestrators, not kernel changes.

This is the line that prevents aiwf from drifting into an orchestration framework. Every "should aiwf do X?" question gets the same test: does X record state, validate state, or run flow? Recording and validating belong in aiwf; running flow doesn't.

### 2.2 Bounded lifetime is universal; concurrency is opt-in

Bounded subagent lifetime (the drift fix) and parallel concurrency (an optimization) are orthogonal axes. The original `parallel-tdd-subagents.md` tangled them. This substrate separates them:

- **Lifetime — always bounded.** One subagent per work unit, fresh context. Universal.
- **Concurrency — role-determined and opt-in.** Read-only and additive-write roles parallelize safely; full-builder roles run sequentially by default and only parallelize when the human dispatches with explicit coarse-scope hints.

### 2.3 Substrate enforcement, not prompt obedience

For any constraint that matters, the chokepoint must be below the LLM:

- **Read-only / additive-write roles**: substrate denies out-of-surface writes (e.g., host-level hooks blocking Edit/Write tool calls outside a declared partition).
- **Builder roles** (where the dynamic-fileset reality means substrate write-deny doesn't work): kernel-side **deviation-detection check rule** compares the worktree diff against the cycle's coarse scope hint and emits a finding. The finding is *blocking* via the F-NNN AC-closure rule. Mechanical surface even if the prompt drifts.
- **Verb gate**: subagents can't run `cancel`, `reallocate`, `authorize`, or `--force` regardless of role. Cheap, universal.

Critically: **declaring the exact fileset per cycle in advance is impossible** — fileset is the *output* of the work, not an input. Most ACs in this repo are real engineering, not typo-fix territory. So the substrate sidesteps fileset declaration for read-only/additive roles (substrate enforcement at coarse capability granularity) and accepts deviation-detection at coarse granularity for builder roles.

### 2.4 Closed-set kernel + open-set user-named

The kernel pins *categories of behavior* (closed-set, drift-tested in `internal/policies/`) and lets users freely *name and configure* specific instances. This pattern already exists for contracts (kernel pins recipe surface; users install validators by name) and slugs (id stable; slug user-controlled).

Applied to agents:

- **Capabilities** are kernel-pinned, closed-set: `read-only`, `additive-tests`, `additive-docs`, `full-builder`, `sequential-only`. Adding a capability requires a kernel change (drift test enforces).
- **Agent names** are user-declared in `aiwf.yaml.subagents.agents[]`. Each agent declares which capability it implements. Framework-shipped agents (builder/reviewer/planner/deployer) come pre-registered. Users add their own (`security-auditor`, `migration-runner`, …) as config + markdown — no kernel recompile.

### 2.5 Post-hoc reconciliation as determinism

Pipelines are declared as data on the epic. Aiwf does *not* enforce the pipeline in real time (that would re-introduce orchestration in the kernel). Instead, aiwf **mechanically reconciles** the trailer history of cycle commits against the declared pipeline at check-time / wrap-time, emits findings on deviation, and gates epic closure on triage.

This is a different definition of determinism than "the agent did exactly what we said." It's: "every deviation is mechanically visible, and closure requires human signoff on each one." It's enforceable in the only way a state-tracking framework honestly can enforce LLM behavior — by making every deviation visible and gating closure.

### 2.6 Permanent metadata + ephemeral content

Forensics and provenance have different retention concerns. The substrate splits them across two storage substrates:

- **Permanent provenance** — small kernel-pinned metadata fields stamped onto cycle commits as git trailers. Always in CI, immutable, queryable from any clone via `aiwf history` and `git log`. Survives any cleanup.
- **Ephemeral forensics** — rich raw record (full diff, raw stdout/stderr, full prompt, narrative summary) lives in a gitignored directory. Reaped after a configurable retention period.

Aiwf guarantees only the *metadata permanence*; bundle content delivery to durable storage is the consumer's responsibility (see §10).

---

## 3. Agent model

### 3.1 Registry

`aiwf.yaml.subagents.agents[]` declares the agents available in the project:

```yaml
subagents:
  agents:
    - name: builder              # framework-shipped
      capability: full-builder
      host: claude-code
      model: claude-opus-4-7

    - name: reviewer             # framework-shipped
      capability: read-only
      host: claude-code
      model: claude-sonnet-4-6

    - name: doc-gardener         # framework-shipped
      capability: additive-docs
      host: claude-code
      model: claude-haiku-4-5

    - name: security-auditor     # user-added (BYO)
      capability: read-only
      host: claude-code
      model: claude-sonnet-4-6

    - name: migration-runner     # user-added (BYO)
      capability: full-builder
      host: claude-code
      model: claude-opus-4-7
```

**Required fields:** `name` (unique within the registry), `capability` (one of the kernel-pinned closed-set values).

**Optional fields:** `host` (defaults to first installed host), `model` (defaults to host's preferred default).

**Validation:**

- Aiwf reads the registry at load time and validates names are unique and capabilities are kernel-pinned values. Drift here is a planning-tree finding.
- Beyond that, aiwf doesn't validate agent definition files exist, hosts are installed, or models are reachable. Those are *runtime* concerns for the orchestrator.

### 3.2 Capabilities (closed-set)

| Capability | Substrate enforcement | Default concurrency |
|---|---|---|
| `read-only` | Substrate denies all writes | Parallel |
| `additive-tests` | Writes restricted to new `*_test.go` files in declared package; no edits to existing files | Parallel with partition |
| `additive-docs` | Writes restricted to declared `docs/**` partition | Parallel with partition |
| `full-builder` | Code modification; no a-priori write restriction (deviation-finding instead) | Sequential default; parallel by explicit dispatch |
| `sequential-only` | No parallelism by definition (e.g., deployer) | N/A |

Capability is what the substrate / verb gate / concurrency policy reads. New capability values require a kernel change with drift test in `internal/policies/`.

### 3.3 BYO agent flow

Adding a user-defined agent has three layers, only one of which is kernel-side:

1. **Registry entry** (user, kernel-validated): add a row to `aiwf.yaml.subagents.agents[]`. Kernel validates uniqueness and capability membership.
2. **Agent definition file** (user, host-specific, kernel-oblivious): write the markdown/config the host expects. Claude Code: `.claude/agents/<name>.md` with system prompt and tool config. Other hosts: their own conventions.
3. **Pipeline reference** (user, kernel-validated): reference the agent name in an epic's `## Pipeline` declaration. Kernel validates the named agent exists in the current registry.

**Static validation:** `aiwf check` emits a non-blocking informational finding (`pipeline-references-unknown-agent`) if a future epic's pipeline names an agent missing from the registry. Helpful for typo-catching; non-blocking because the consumer might not have set up the agent yet.

**Runtime validation:** the orchestrator (skill or sidekick) refuses to dispatch a step if the named agent isn't in the registry, doesn't have its definition file at the host's expected path, or the model isn't available. Hard failure, actionable error.

**Closed epics: never re-validated.** History is immutable; trailers reference whatever was registered at execution time. Renamed/removed agents do not retroactively invalidate closed epics. This mirrors the existing slug-rename rule (id stable; slug drifts).

### 3.4 Multi-host

Hosts can be mixed in one registry. The orchestrator decides at run time whether it can execute:

- The skill driver typically runs in one host. If the pipeline references an agent for a different host, dispatch fails with a clear error.
- Future `aiwfdo` (sibling deterministic orchestrator, see §6.4) could dispatch heterogeneously by shelling to different host CLIs. Kernel doesn't care.

Cross-host pipelines are declarable as *data*; whether the configured orchestrator can execute them is a runtime concern with no special handling in the kernel.

---

## 4. Provenance and sovereignty

### 4.1 Role-tagged actor

Trailer schema extends to: `aiwf-actor: ai/<host>/<agent-name>`. Examples:

- Parent agent (no role context): `aiwf-actor: ai/claude` (existing, unchanged).
- Subagent: `aiwf-actor: ai/claude/builder`, `ai/claude/security-auditor`, etc.

Existing `ai/claude` stays valid for parent-agent contexts (no agent role). Role tag differentiates subagents in `aiwf history` queries, audit, and the verb gate.

### 4.2 Sub-scope FSM (builder-parallel only)

D2 deferred G22's general sub-scope question and adopted a narrow form: builder-capability + parallel + dispatched opens a sub-scope inside the parent's existing authorize scope. Read-only and additive-write roles use flat actor with role tag, no sub-scope.

Sub-scope reuses the existing scope FSM with extended end-states (introduced in §7):

- `active` → `paused` | `ended-success` | `ended-failure` | `ended-discarded`

One level of nesting maximum. The dispatch is the human-sovereign act; sub-scope opening rides on it. No new sovereignty rule.

### 4.3 No N-level agency

PoC rule: subagents cannot spawn subagents. N=2 max. If a cycle's findings need a follow-up cycle to fix, the orchestrator dispatches it on the next pass; subagents in flight cannot dispatch others. The recursion bound lives at the orchestrator level, where the human's policy controls it.

### 4.4 Verb gate

When `aiwf` is invoked inside a subagent context (signaled by env var or flag at spawn), the verb dispatcher refuses:

- `aiwf cancel` — terminal flips are sovereign.
- `aiwf reallocate` — id mutations are sovereign.
- `aiwf authorize` — agents cannot delegate further (combined with §4.3).
- `--force` on any verb — sovereignty stays human-only.

The env-var/flag is the *signal*; the substrate-level write deny (capability enforcement) is the actual safety. Belt + suspenders.

### 4.5 Sovereignty rule for finding-terminal transitions (D6 6a update)

Closing a finding to *any* terminal status without code change requires `--force` and `--reason`, regardless of which terminal status. Single rule:

- `aiwf promote F-NNN resolved [--reason "..."]` — the resolving commit references F-NNN via the standard trailer; soft check warns when no associated fix link is present (§4.6).
- `aiwf promote F-NNN waived --force --reason "..."` — sovereign accept.
- `aiwf promote F-NNN invalid --force --reason "..."` — sovereign reject (false positive).

This updates `ADR-0003` §"Status FSM" — the original ADR distinguished `waived` (force) from `invalid` (no force); D6 unified them.

### 4.6 Soft check on missing fix link (D6 6c)

When a finding promotes to `resolved`, fire warning-only finding `finding-resolved-without-fix-link` if **no prior ancestor commit reachable from the resolve commit (excluding the resolve commit itself, walking back until merge-base with main) carries `aiwf-entity: <F-NNN>` in its trailers**.

Scope: only `resolved` transitions. `waived` and `invalid` skip (no fix expected/needed by definition). Warning-only initially; promotion to blocking is a future judgment call after dogfooding.

---

## 5. Work-shape declarations (per-epic pipelines)

### 5.1 Where they live

Each epic's body may contain a fenced YAML block under `## <work-shape>` (terminology open: `## Cycle` / `## Loop` / `## Workflow` / `## Pipeline` / `## Orchestration` — pinned at implementation time, drift-tested thereafter). Throughout this document the placeholder is `## Pipeline`.

```markdown
## Pipeline

```yaml
- id: 1
  agent: planner
  per-epic: true
  scope-hint: docs/pocv3/design/
  produces: design-doc

- id: 2
  agent: builder-flagship
  per-ac: true
  depends-on: [1]
  scope-hint: internal/check/
  cycle-model-override: claude-opus-4-7

- id: 3
  agent: reviewer
  per-ac: true
  depends-on: [2]
  timeout: 10m

- id: 4
  agent: doc-gardener
  per-epic: true
  depends-on: [3]
  partition: [README.md, docs/pocv3/]

- id: 5
  agent: security-auditor      # BYO agent
  per-epic: true
  depends-on: [3]
```
```

### 5.2 Layered schema

| Layer | Owner | Drift policy |
|---|---|---|
| **Core** (load-bearing for reconciliation) | aiwf | Closed-set; new fields require kernel change + drift test |
| **Hints** (execution metadata) | each orchestrator | Free-form; aiwf ignores; orchestrator validates its own |

**Core fields (kernel-owned):**

| Field | Purpose |
|---|---|
| `id` | Stable per-step identifier; stamped into commits as `aiwf-pipeline-step: N` |
| `agent` | Must exist in the registry; capability is derived from there |
| `depends-on` | Sequencing constraint; reconciliation checks ordering by commit timestamp |
| `per-ac` / `per-epic` | Determines expected cycle-commit cardinality (one per AC vs. one for the whole epic) |
| `scope-hint` | Read by the deviation-detection check rule for builder-capability steps |

**Hint fields (orchestrator-owned, examples):** `produces`, `partition`, `timeout`, `cycle-model-override`, `prompt-prefix`, `retry-on`. Aiwf parses the YAML, validates kernel-owned fields, ignores the rest. Each orchestrator validates and uses its own hint set.

### 5.3 Reconciliation algorithm

A new check rule walks the trailer history of cycle commits for the epic, groups by `aiwf-pipeline-step:`, and compares against the declaration:

| Finding code | Fires when |
|---|---|
| `pipeline-step-missing` | Per-epic step has no cycle commits |
| `pipeline-step-missing-for-ac` | Per-ac step missing for some AC under the epic's milestones |
| `pipeline-agent-mismatch` | Cycle's `aiwf-actor` doesn't match the declared agent |
| `pipeline-model-mismatch` | Cycle's `aiwf-cycle-model` doesn't match the registry-pinned model |
| `pipeline-ordering-violation` | A cycle ran before its declared `depends-on` cycles |
| `pipeline-step-extra` | Cycle commit references a step id not in the declaration |
| `pipeline-references-unknown-agent` | Pipeline names an agent not in the current registry (informational only) |

All `pipeline-*` codes are kernel-pinned, finding-kind, and gate epic closure (the existing `findings-block-met` rule applied at epic level). Determinism is post-hoc, not real-time — the kernel doesn't prevent the agent from skipping a step; it mechanically detects deviations after the fact and gates closure on human triage.

### 5.4 Driver config

```yaml
# aiwf.yaml
orchestration:
  driver: llm           # | sidekick      (PoC: llm only; sidekick = future aiwfdo)
```

- **`llm` driver** — host-specific skill (e.g. `aiwfx-run-<work-shape>` under `.claude/skills/aiwf-*`) reads the schema and dispatches via the host's headless CLI / Agent tool. PoC default.
- **`sidekick` driver** — future deterministic Go binary (`aiwfdo`, "Deterministic Orchestrator"; see §6.4) reads the schema and dispatches mechanically. Deferred until LLM-driver dogfooding shows determinism friction worth solving.

The kernel's reconciliation works for both drivers; the choice is consumer-facing.

### 5.5 Schema evolution discipline

The kernel-owned core is small and stable. Hint fields are orchestrator-specific and evolve with each orchestrator's roadmap. Field-semantics drift across orchestrators is a real risk: if both the LLM driver and aiwfdo eventually implement `partition`, they must mean the same thing. Documentation lives at the schema definition site; CI test ensures behavior agreement when the second orchestrator arrives.

The schema spec is authored in CUE at `internal/pipeline/pipeline.cue` (canonical documentation + drift-prevention). The kernel's runtime parser is hand-written Go in `internal/pipeline/`. CI runs `cue vet` against fixtures and asserts the Go parser agrees with the CUE schema. **CUE never ships at runtime** — kernel doesn't embed `cuelang.org/go/cue`; consumers don't need `cue` installed. This aligns with the existing rule (`design-decisions.md:125`: "aiwf never ships a `cue` or `ajv` binary"): CUE is the spec we hold ourselves to in CI; the Go parser is what runs.

---

## 6. The orchestrator

### 6.1 Strict-lane consequences

Per §2.1, the orchestrator is *external to aiwf*. Aiwf's contribution to orchestration is:

- The capability/agent registry (data).
- The pipeline schema (data).
- Trailer schema for cycle events (audit primitives).
- Verb gate for subagent contexts.
- Reconciliation check rules.

Aiwf does **not** invoke LLMs, manage worktrees, parse subagent envelopes, or interpret pipelines. Those are orchestrator concerns.

### 6.2 LLM driver (skill-based, ships first)

Lives under `.claude/skills/aiwf-*` (or each host's equivalent). Marker-managed by `aiwf init` / `aiwf update`. Reads the pipeline declaration, **materialises subagent isolation as a parent-side precondition** (`git worktree add` followed by an observable presence check via `git worktree list`) before invoking the agent-dispatch tool with the worktree path as the working directory, collects JSON envelopes, calls aiwf primitives (`aiwf authorize`, `aiwf add finding`, `aiwf check`, etc.) to record events. Where the host's dispatch tool offers an isolation kwarg (e.g., Claude Code's `Agent` `isolation: "worktree"`), it is treated as a hint — not the load-bearing mechanism. See [ADR-0009](../../adr/ADR-0009-orchestration-substrate-vs-driver-split.md) §Decision 3, §7.7 below, and G-0099 for rationale.

**Sequence per cycle:**

1. Resolve agent from registry → capability → concurrency policy.
2. For builder-parallel: open sub-scope.
3. **Materialise isolation as precondition.** `git worktree add <path> <branch>`; verify presence via `git worktree list`; refuse to dispatch if the worktree did not materialise.
4. **Spawn subagent** with appropriate env signal + agent definition file; invoke the agent-dispatch tool with the worktree path as the working directory.
5. Wait for return; collect envelope; persist forensic bundle.
6. Walk envelope's `findings[]`; call `aiwf add finding` per finding.
7. **Reconcile isolation.** A kernel `aiwf check` rule (`isolation-escape`) verifies every commit carrying the cycle's `aiwf-cycle-id` is reachable from the cycle's `aiwf-cycle-worktree-branch` trailer; a mismatch fires the `isolation-escape` finding and forces `ended-failure`. The driver does not own this check — it lands in `aiwf check` so every driver inherits the same enforcement.
8. Close sub-scope with appropriate end-state.
9. Surface to human if findings block AC closure.

**Where the human invokes it.** Most likely an extension of `aiwfx-start-milestone`: when it reaches an AC ready for cycle dispatch, hand off to the cycle skill instead of inline TDD. Direct invocation (`/aiwfx-run-cycle <ac-id>`) is also useful for re-running a single AC.

### 6.3 Cycle event recording — no new verbs

Per the strict-lane discipline, cycle dispatch is *not* a new verb pair (`aiwf cycle-begin` / `aiwf cycle-end` was an early proposal — rejected). Instead, cycle structure is recorded as **trailers on existing verbs' commits**:

```
aiwf-verb: add
aiwf-entity: F-007
aiwf-actor: ai/claude/builder
aiwf-cycle-id: M-066/AC-1#cycle-1
aiwf-cycle-role: builder
aiwf-cycle-pipeline-step: 2
aiwf-cycle-scope-hint: internal/check
aiwf-cycle-model: claude-opus-4-7
```

`aiwf history` walks trailers; cycle reconstruction (timing, agent attribution, scope hints, findings tally) is a query on top of existing audit primitives. No new control surface in the kernel.

### 6.4 Sidekick driver (`aiwfdo`, deferred)

If LLM-driver dogfooding shows determinism friction worth solving, a sibling Go binary `aiwfdo` ("Deterministic Orchestrator") would consume the same pipeline schema and dispatch mechanically. Same `go.mod`, separate `cmd/aiwfdo/` directory; pipeline parser lifted from `internal/pipeline/` to `pkg/pipeline/` (public Go API); both binaries vendor it.

`aiwfdo` is **deferred, not planned**. The pipeline schema and reconciliation check work today regardless of driver; building the sidekick is YAGNI until a real consumer needs strict determinism over LLM judgment.

---

## 7. Cycle envelope and forensic bundle

### 7.1 Envelope shape (A2A-shaped, custom transport)

The cycle envelope is the JSON record returned by a subagent at the end of its lifetime. We borrow concepts from Google's A2A protocol (April 2025) — agent card ≈ registry entry; task envelope ≈ cycle envelope; status enum literals; artifact concept ≈ bundle sidecars — but **don't adopt the protocol**.

Rationale: A2A is HTTP/JSON-RPC for *running services* (agent endpoints, conversational task lifecycle, async streaming via SSE). Our subagents are *spawned bounded-lifetime CLIs* (`claude -p`, `codex --headless`) that exit when done. The transport mismatch is real; wrapping a CLI in an A2A HTTP service to satisfy the protocol would be ceremony with no payoff. If a future consumer wants to expose aiwf-orchestrated cycles via A2A endpoints, the wrapping layer is thin because the data shape is already aligned.

**Schema** (kernel core + orchestrator hints, layered like the pipeline schema):

```json
{
  "status": "success",
  "cycle_id": "M-066/AC-1#cycle-1",
  "agent": "builder-flagship",
  "model": "claude-opus-4-7",
  "summary_md": "## What I did\n\nImplemented the new check rule by...",
  "findings": [
    {"code": "branch-coverage-gap", "linked_acs": ["M-066/AC-1"], "body_md": "..."}
  ],
  "diff_path": "diff.patch",
  "stats": {"duration_ms": 192340, "files_touched": 4, "tests_added": 6, "lines_added": 187, "lines_removed": 23}
}
```

**Embedded markdown via `*_md` fields** (`summary_md`, `body_md`, etc.). Two wins: forensic completeness in one record; graceful parser degradation (markdown stays readable if JSON fails to parse). On persist, `*_md` fields are extracted to sidecar files for human consumption.

**Forgiving parser.** Unparseable envelopes don't crash the orchestrator; they fall back to `cycle-envelope-malformed` finding + raw blob preserved in bundle. Standard "errors are findings, not parse failures" pattern.

**No JSONL in-tree.** Kernel rejects events.jsonl as event log substrate (`CLAUDE.md` "What is *not* in the PoC"). JSONL is permitted as an *export output format* (§10) — that's delivery, not storage.

**Schema spec** in CUE at `internal/cycle/envelope.cue`; Go parser in `internal/cycle/`; CI agreement test. CUE never ships at runtime (same rule as the pipeline schema).

### 7.2 Forensic bundle layout

Each cycle gets a directory mirroring the conceptual hierarchy:

```
.aiwf/cycles/
  <epic-id>/<milestone-id>/<ac-id>/cycle-<n>/
    envelope.json           # primary record (machine-readable)
    summary.md              # extracted from envelope.summary_md
    diff.patch              # full diff
    stdout.log              # raw subagent stdout
    stderr.log              # raw subagent stderr
    prompt.md               # the prompt sent to the subagent
    agent.yaml              # resolved registry entry at dispatch time
    findings/
      F-007-body.md         # one file per finding's body
      F-008-body.md
```

`.aiwf/cycles/` is gitignored by default. `aiwf init` adds the entry; `aiwf doctor` warns if it's missing.

### 7.3 Quarantine on failure

Failed cycle bundle moves to `.aiwf/quarantine/<epic-id>/<milestone-id>/<ac-id>/cycle-<n>/` preserving the same hierarchy. Never wiped. Failed cycles are forensically interesting; quarantine names them ("look here") without polluting active state. `aiwf reap-quarantine --older-than 90d` provides cleanup.

### 7.4 Sub-scope FSM end-states (extends D2)

| End-state | Meaning | Bundle disposition |
|---|---|---|
| `ended-success` | Subagent returned cleanly; envelope valid; ready for merge | Bundle stays; reaped per retention policy |
| `ended-failure` | Subagent failed (crash, timeout, malformed envelope, substrate-deny tripped) | Bundle quarantined |
| `ended-discarded` | Human-discarded mid-cycle | Bundle stays; tagged as discarded |

### 7.5 Per-capability differentiation

| Capability | Failure handling |
|---|---|
| `read-only` | No diff to preserve (writes were denied); substrate-deny becomes a `scope-leak` finding + `ended-failure`. Bundle minimal — only logs + envelope. |
| `additive-tests` / `additive-docs` | Partial writes within surface preserved in quarantine bundle. |
| `full-builder` | Quarantine bundle preserves diff + logs + prompt; `partial-state-in-worktree` finding pointing the human at the bundle. |
| `sequential-only` (deployer) | Sub-scope failure has its own gravity; out of scope for the substrate; deployers don't parallelize anyway. |

### 7.6 Recovery: orchestrator reap + kernel GC safety net

- **Orchestrator-driven reap (primary).** Orchestrator catches subagent termination (success / timeout / error / substrate-deny tripped) and runs cleanup: closes the sub-scope, writes the cycle commit with full trailer set, persists or quarantines the bundle.
- **Kernel-side stale-cycle GC (safety net).** A doctor check + verb (`aiwf reap-stale-cycles`) walks open sub-scopes older than a threshold and closes them as `ended-failure` with a `cycle-orchestrator-died` finding. Catches the case where the orchestrator itself crashed.

### 7.7 Isolation as parent-side precondition (closes G-0099)

Subagent isolation is **materialised by the parent before dispatch** — not requested via an agent-dispatch tool kwarg. The precondition + reconciliation pattern matches the kernel principle that the framework's correctness must not depend on the LLM (or its harness) honoring a kwarg: materialisation has to be observable before the agent runs; misbehavior has to be detectable after.

**Precondition (§6.2 step 3).**

1. `git worktree add <path> <branch>` materialises the isolated workspace.
2. `git worktree list` is the observable presence check; if the expected path is absent, the parent refuses to dispatch.
3. The agent-dispatch tool is invoked with the worktree path as the working directory. Any host-specific isolation kwarg is a hint, not the mechanism.

**Reconciliation (§6.2 step 7).** Post-cycle, the parent verifies that every commit carrying the cycle's `aiwf-cycle-id` lives on the cycle's declared worktree branch and the diff is rooted inside the worktree path. A mismatch fires the `isolation-escape` finding and the cycle ends as `ended-failure` regardless of the subagent's envelope status. Pairs with the existing `scope-expanded` rule (§2.3, §8) as the post-cycle deviation-detection family.

**Failure mode caught.** A subagent whose `git worktree`-aware harness silently fell back to the live tree (the G-0099 session); a subagent that ran `cd ..` and committed in the parent checkout; an agent-dispatch tool that ignored the working-directory argument. All three produce commits whose branch / path don't match the cycle's declared worktree — all three fire `isolation-escape`.

**Check site (kernel-side).** The precondition is parent-side by definition (it runs before any agent is invoked). The post-cycle reconciliation is a **kernel `aiwf check` rule** (`isolation-escape`) that reads cycle trailers — specifically `aiwf-cycle-id` and `aiwf-cycle-worktree-branch` (§9 trailer surface) — and asserts every commit carrying that cycle-id is reachable from the declared worktree branch ref. The check is decidable from `git log` alone, so it composes with the existing pre-push hook and CI surface without requiring filesystem inspection. Driver-side enforcement is not the chokepoint; the kernel rule is — every driver (current Claude Code skill, hypothetical `aiwfdo` sidekick per §6.4, third-party drivers) inherits identical enforcement for free. See [ADR-0009](../../adr/ADR-0009-orchestration-substrate-vs-driver-split.md) §Decision 3 for rationale.

---

## 8. Scope enforcement summary

Combining §2.3 with the role table in §3.2:

| Layer | Mechanism | Applies to |
|---|---|---|
| **L1 — substrate write deny** | Host-level hooks (e.g., gitignored `.claude/` hook) deny Edit/Write tool calls outside the role's allowed surface | Read-only, additive-tests, additive-docs |
| **L2 — verb gate** | `aiwf` dispatcher refuses subagent-forbidden verbs (`cancel`, `reallocate`, `authorize`, `--force`) | All capabilities |
| **L3 — deviation-detection check rule** | Kernel-side `scope-expanded` check compares cycle diff against declared scope hint; emits blocking finding on deviation | Full-builder (where L1 doesn't apply) |
| **L4 — isolation reconciliation** | Kernel-side `isolation-escape` check rule asserts every commit carrying `aiwf-cycle-id` is reachable from the cycle's `aiwf-cycle-worktree-branch` trailer; mismatch fires `isolation-escape` (see §7.7) | All cycles dispatched against a worktree |

L1 is host-specific (Claude Code today; another host gets its own implementation). L2, L3, and L4 are kernel-side, host-agnostic. The combination is belt + suspenders: L1 catches misbehavior before commit; L3 and L4 catch it before AC closure.

---

## 9. Provenance via trailers (permanent record)

Every cycle commit carries a kernel-pinned trailer set. The full surface:

```
aiwf-verb               (existing)
aiwf-entity             (existing)
aiwf-actor              (existing, extended: ai/<host>/<agent-name>)
aiwf-cycle-id
aiwf-cycle-status       (ended-success | ended-failure | ended-discarded)
aiwf-cycle-role         (capability)
aiwf-cycle-agent        (registry name)
aiwf-cycle-model
aiwf-cycle-host
aiwf-cycle-pipeline-step
aiwf-cycle-worktree-branch  (git ref where cycle commits must live; consumed by isolation-escape rule)
aiwf-cycle-scope-hint
aiwf-cycle-prompt-hash  (sha256:... — content-addressable hook for future CAS)
aiwf-cycle-duration-ms
aiwf-cycle-files-touched
aiwf-cycle-lines-added
aiwf-cycle-lines-removed
aiwf-cycle-tests-added
aiwf-cycle-findings-count
aiwf-cycle-findings     (comma-separated F-NNN ids)
```

Each trailer ~50 bytes; full set ~700 bytes per cycle commit. All kernel-pinned, drift-tested in `internal/policies/trailer_keys.go`. Stat-field semantics specified precisely in `internal/cycle/` documentation (e.g., "files-touched" = unique modified paths in the cycle's diff against the parent commit).

**Provenance check rule** (`cycle-trailer-incomplete`) flags cycle commits missing required trailers. Catches orchestrator drift (skill forgot to stamp duration; sidekick wrote partial set on crash).

**Prompt content strategy.** Trailer carries `aiwf-cycle-prompt-hash` only; full prompt lives in the forensic bundle. When bundle is reaped, the prompt content is lost from local disk. **Future upgrade path:** content-addressable store at `.aiwf/prompts/<sha256>.md` (deduplicated, tracked by git, small because most prompts repeat across cycles). The hash trailer is the future-compatibility hook; CAS can ship later without schema breakage. Don't build CAS speculatively.

---

## 10. Forensics scope boundary + harvest

### 10.1 Aiwf's forensics promise

Aiwf **guarantees**:

- Permanent metadata via trailers (always in git, always in CI, queryable forever via `aiwf history`).
- An export tool (`aiwf export-cycles`) that emits bundle + trailer-derived data in a stable, drift-tested schema for downstream consumption.

Aiwf does **not guarantee**:

- Content delivery to durable storage. Bundle content (raw logs, full prompt text, narrative summaries) lives gitignored. Whether it reaches a permanent destination — backup, warehouse, training-data archive, compliance store — is the consumer's responsibility.
- Bundle survival past local reap policy.

**Why this is the right boundary.** Aiwf's correctness doesn't depend on bundle persistence. Reconciliation, AC closure, FSM transitions all work from trailers alone. "Chain of custody is intact" is provable from trailers (agent + model + prompt-hash + outcome). "Reconstruct the exact LLM exchange" is the richer claim that depends on bundle survival — and is a consumer concern, not a kernel correctness property.

### 10.2 Harvestable export surface

Aiwf produces structured data; aiwf doesn't analyze it. Anyone wanting to use the data for any purpose pulls it via a stable export verb and ships it wherever. Aiwf is not involved in destinations.

**`aiwf export-cycles` verb:**

```
aiwf export-cycles --since 2026-01-01 --format jsonl > cycles.jsonl
aiwf export-cycles --epic E-19 --include-bundles --format tar > e19.tar
aiwf export-cycles --redact-prompts --since 2025-01-01 > sanitized.jsonl
```

Each record = trailer-derived metadata merged with envelope content; bundles optionally inlined or tarred. JSONL output is the natural shape for piping into warehouses; tar for full-bundle archives.

**Schema stability is a public contract.** Trailer keys, envelope schema, and export record format are kernel-pinned, drift-tested, and treated as a versioned public API. Additive changes are routine; renames/removals are breaking-change territory with deprecation discipline.

### 10.3 Retention adapts to harvest setup

| Setup | Default retention |
|---|---|
| No external harvester | Bundle is only forensic copy → keep longer (90d / 365d for success / failure) |
| Daily harvest to permanent store | Bundle is redundant copy → reap aggressively (7d / 30d) |

Configured via `aiwf.yaml.cycles.retention`. `aiwf doctor` warns if `harvest_to` is set but no harvest has happened recently.

### 10.4 Aspirational pre-push harvest hook

A hook similar in shape to the existing pre-push `aiwf check` hook could close the discipline gap structurally for consumers needing compliance-grade forensics: `aiwf.yaml.cycles.harvest_to: <destination>`; aiwf-managed marker hook calls `aiwf export-cycles --since last-push --upload-to <destination>` at push time. The pieces it would ride on (export verb, marker-hook system, config schema, redaction flags) are already in the design or already exist; the wiring would be small.

**This is an idea, not a plan.** No milestone, no epic, no roadmap commitment. Captured here so future readers know the upgrade path is shaped if a consumer ever surfaces concrete compliance needs. Until that consumer exists, the work is not on the schedule and should not appear in any planning artifact.

### 10.5 Speculative use cases enabled (not designed for)

- **Training data:** bundles contain (prompt, response, outcome quality) tuples in the right shape for fine-tuning agents on successful patterns.
- **Recovery:** harvest archive contains enough to reconstruct planning state if local repo dies (parallel record, not a git replacement).
- **Cross-project meta-analysis:** shared harvest warehouse → org-level pattern learning.
- **Compliance / regulated audit:** harvest archive is the audit trail of record.

None of these justify building anything beyond the export verb. They justify the schema-stability discipline that makes the verb's output consumable.

### 10.6 `aiwf cycle-stats` query verb (deferred)

Once trailers carry richness, an in-repo query verb computes "average duration of builder cycles using Opus", "findings-per-agent grouped by scope-hint", etc. Walks `git log`, parses trailers, aggregates. Implementation is straightforward; deferred until dogfooding shows demand. Trailer schema enables it whenever it ships.

---

## 11. Findings on findings; recursion bounds (D6 6d)

**Findings on findings** just work via the existing data model. Findings produced while resolving another finding link via `linked_entities` (e.g., F-008's frontmatter contains `linked_entities: [F-007]`). AC closure check walks the linked-findings graph; new findings on the same AC block closure regardless of which cycle produced them. `aiwf history F-007` shows the F-008 cross-reference because both commits carry standard trailers.

**Recursive subagent spawning** is **forbidden** by the §4.3 rule (subagents cannot spawn subagents; N=2 max). If a cycle's findings need a follow-up cycle to fix, the orchestrator dispatches it on the next pass; subagents in flight cannot dispatch others. The recursion bound lives at the orchestrator level, where the human's policy controls it.

Defensive cycle-detection in the `linked_entities` graph (F-007 → F-008 → F-007) is YAGNI — link cycles aren't a real failure mode (semantically odd, but break nothing) and adding a check rule for a hypothetical case violates "ship what's used."

---

## 12. Implementation surface

The substrate decomposes into independently-shippable kernel changes. Each is small enough to be one or two milestones; together they form the substrate underneath specific consumer features (E-0019, future doc-gardening epics, security-audit epics).

### 12.1 Kernel changes

**Registry and capabilities:**

- `internal/agents/` — capability enum (closed-set), registry parser, validation.
- `aiwf.yaml` schema extension — `subagents.agents[]` entries.
- New finding code: `pipeline-references-unknown-agent` (informational).

**Sub-scope FSM extension:**

- `internal/entity/` (or wherever scope FSM lives) — extended end-states (`ended-success | ended-failure | ended-discarded`); one level of nesting allowed.
- New trailer keys: `aiwf-cycle-id`, `aiwf-cycle-status`, plus the rest of §9's surface.
- Drift test in `internal/policies/trailer_keys.go`.

**Verb gate for subagent context:**

- Env-var/flag detection in the verb dispatcher.
- Refusal logic for `cancel`, `reallocate`, `authorize`, `--force` when in subagent context.

**Pipeline schema:**

- `internal/pipeline/` — Go parser, validator, reconciliation algorithm.
- `internal/pipeline/pipeline.cue` — canonical schema spec.
- New finding codes: `pipeline-step-missing`, `pipeline-step-missing-for-ac`, `pipeline-agent-mismatch`, `pipeline-model-mismatch`, `pipeline-ordering-violation`, `pipeline-step-extra`.
- CI agreement test (Go ↔ CUE).

**Cycle envelope:**

- `internal/cycle/` — Go parser, validator.
- `internal/cycle/envelope.cue` — canonical schema spec.
- CI agreement test.
- New finding code: `cycle-envelope-malformed`.

**Forensic bundle + reap:**

- `aiwf reap-stale-cycles` verb.
- `aiwf reap-quarantine` verb.
- `aiwf doctor` checks: gitignore present, stale cycles, quarantine size.
- `aiwf.yaml.cycles.retention` schema.

**Provenance check rule:**

- `cycle-trailer-incomplete` — flags cycle commits missing required trailers.

**Sovereignty rule update:**

- `aiwf promote F-NNN invalid` requires `--force --reason` (D6 6a).

**Soft check on missing fix link:**

- `finding-resolved-without-fix-link` — warning-only finding (D6 6c).

**Export surface:**

- `aiwf export-cycles` verb with `--format`, `--include-bundles`, `--redact-prompts`, `--since`, `--epic` flags.
- Schema-stability discipline (drift tests treat export shape as a public contract).

### 12.2 Skill changes

- LLM-driver cycle skill (e.g. `aiwfx-run-<work-shape>`) consuming the pipeline schema and registry. Marker-managed via `aiwf init` / `aiwf update`.
- Extension of `aiwfx-start-milestone` to dispatch through the cycle skill instead of inline TDD.

### 12.3 Documentation

- `CLAUDE.md` — add the strict-lane principle (§2.1).
- `design-decisions.md` — add the forensics scope boundary (§10.1).
- `ADR-0003` — sovereignty rule update (D6 6a); recursion-bounds note (§11).
- `ADR-0004` — terminality table cleanup (D6 6b).
- `parallel-tdd-subagents.md` — restructure as a consumer of this substrate, or absorb entirely.

---

## 13. Sequencing

This synthesis is the design source; downstream work decomposes into landing the substrate before specific consumers (E-0019) are picked up. Suggested order:

1. **Synthesis lands** (this document).
2. **Canonical doc updates** (separate session): folds the synthesis into `CLAUDE.md`, `design-decisions.md`, `ADR-0003`, `ADR-0004`, `parallel-tdd-subagents.md`.
3. **Substrate epics filed and landed** (each ~1 milestone, can mostly run in parallel):
   - Agent registry + capability enum + verb gate.
   - Sub-scope FSM extension + role-tagged actor + cycle trailer keys.
   - Pipeline schema parser + reconciliation check + finding codes (CUE spec + Go parser + CI agreement test).
   - Cycle envelope schema + forensic bundle layout + reap verbs (CUE spec + Go parser + CI agreement test).
   - Export verb + schema-stability drift tests.
   - F-NNN entity kind (existing dependency from E-0019, ADR-0003).
   - Uniform archive convention (existing dependency from E-0019, ADR-0004).
   - Findings-gated AC closure (existing dependency from E-0019).
4. **E-0019 rewritten** to consume the substrate. Expected to shrink substantially — the TDD-cycle agent definition + cycle skill driver integration + dogfooding milestone become the core scope, while substrate decisions move out to step 3.
5. **E-0019 implementation.**

Steps 1-2 are documentation. Step 3 is the bulk of the implementation work and is where most engineering attention goes. Steps 4-5 are smaller because the substrate already exists.

---

## 14. What's deliberately not in this design

Items considered and deferred. Each can be added later when real friction shows up.

- **Aiwfdo (sidekick orchestrator).** Sibling Go binary that consumes the same pipeline schema and dispatches deterministically. Aspirational — the LLM driver suffices for PoC; the schema is shaped to make `aiwfdo` cheap to build later.
- **CAS for prompts.** `.aiwf/prompts/<sha256>.md` deduplicated content-addressable store, tracked by git. Hash trailer (§9) is the future-compatibility hook; full CAS waits until prompt-effectiveness analysis is a real consumer.
- **Pre-push harvest hook.** Closes the discipline gap structurally for compliance-grade consumers. Aspirational; not on the roadmap.
- **N>2 sub-scope nesting.** Subagents cannot spawn subagents in the PoC. Future extension is G22 territory; not needed for any current use case.
- **`aiwf cycle-stats` query verb.** In-repo aggregation over cycle trailers. Trailer schema enables it; implementation deferred until dogfooding shows demand.
- **Body-section validators for finding bodies** (analogous to M-0066's `entity-body-empty` for milestones). Findings will eventually require structured `## Resolution` / `## Waiver` sections on terminal promotion. Wait for the body-section pattern to settle on the existing kinds before generalizing.
- **`aiwf reframe F-007 --as-gap` verb.** Cross-references between F-NNN and G-NNN already work via `linked_entities`; a dedicated verb is convenience, not necessary.
- **Subagent observability beyond the JSON return.** When a subagent does the wrong thing, the parent currently sees the result, not the reasoning. Richer introspection (full subagent transcripts surfaced into `aiwf history`) is deferred until real friction shows up.
- **Heuristic auto-detection of independent ACs.** Builder-parallel dispatch relies on the human declaring disjoint coarse scopes. Inferring independence from AC body prose is a future optimization.
- **Cross-cycle findings.** A finding pertaining to no specific AC but to the milestone or epic as a whole. The data model supports this (empty `linked_acs` + non-empty `linked_entities`); no current producer emits them.
- **Cross-host pipeline execution.** The PoC's LLM driver runs in one host; cross-host execution is a future orchestrator capability, not a kernel concern.
- **Adopting A2A as actual transport.** Considered and rejected for the PoC; concepts borrowed, transport not. Re-evaluate if cross-vendor agent interop becomes a real concern.

---

## 15. Open / undecided

Items still open at the synthesis level. Each needs to be pinned at implementation time but doesn't block this document.

- **Terminology.** The `## <work-shape>` section heading and related verb / finding code names. Candidates: `cycle` / `loop` / `workflow` / `pipeline` / `orchestration`. Pinned at implementation time; once pinned, drift-tested like other kernel-pinned vocabularies.
- **Default retention windows.** `aiwf.yaml.cycles.retention` defaults (90/365 vs 7/30 with harvest). Picked at implementation time; tunable per consumer.
- **Cycle id format.** Composite (`M-066/AC-1#cycle-1`) is the working assumption; encoding may need refinement when filesystem-path safety is checked (slashes in directory names are awkward — likely the bundle path uses hierarchy `M-066/AC-1/cycle-1/` and the trailer carries the composite `M-066/AC-1#cycle-1` directly).
- **Stat-field semantics.** "Files-touched", "tests-added" need precise, testable definitions. Documented at the trailer keys site; specified before the cycle envelope schema lands.
- **Sub-scope id allocation scheme.** Whether sub-scope ids are independent (`S-NNN`?) or composite (`<parent-scope-id>.<n>`). Pinned at implementation time.

---

## 16. Glossary

| Term | Meaning |
|---|---|
| **Subagent** | An LLM invocation with bounded lifetime (one work unit, fresh context, returns when done) operating under a role-tagged actor. |
| **Capability** | Kernel-pinned closed-set value describing what an agent is allowed to do (`read-only`, `additive-tests`, `additive-docs`, `full-builder`, `sequential-only`). |
| **Agent** | User-named, registry-declared instance carrying a name, capability, host, optional model. Examples: `builder`, `reviewer`, `security-auditor`. |
| **Role** | Synonymous with agent name in trailer context (`aiwf-actor: ai/<host>/<role>`). |
| **Cycle** | One subagent invocation: lifetime from spawn to return, including envelope, bundle, and the commits it produces. (Terminology placeholder; see §15.) |
| **Pipeline** | Per-epic declarative work-shape: ordered steps, each naming an agent and depending on prior steps. Kernel-pinned core schema + orchestrator-owned hints. (Terminology placeholder; see §15.) |
| **Orchestrator** | External component that reads the pipeline and dispatches cycles. Either host-specific skill (LLM driver, ships first) or future deterministic sibling binary (`aiwfdo`, deferred). |
| **Sub-scope** | Authorize-scope nested inside the parent's scope, opened on builder-parallel dispatch. One level of nesting maximum. |
| **Forensic bundle** | Per-cycle directory under `.aiwf/cycles/...` containing envelope + sidecars + raw logs + prompt + resolved agent config. Gitignored, ephemeral. |
| **Quarantine** | Forensic bundle from a failed cycle, moved under `.aiwf/quarantine/...` preserving the same hierarchy. Never wiped automatically. |
| **Permanent provenance** | Kernel-pinned trailer set on cycle commits. Always in git, immutable, queryable forever. |
| **Reconciliation** | Mechanical comparison of trailer history against the declared pipeline; emits `pipeline-*` findings on deviation; gates epic closure. |
| **Strict lane** | The kernel rule that aiwf provides primitives only — not orchestration, LLM invocation, or analysis. |
| **L1 / L2 / L3** | The three layers of scope enforcement (substrate write deny / verb gate / deviation-detection check rule). See §8. |

---

## 17. References

- [`_scratch-subagents-research.md`](_scratch-subagents-research.md) — Q&A trail of how each decision was reached. Contains decisions D1-D6 with full rationale, rejected alternatives, and option enumerations.
- [`parallel-tdd-subagents.md`](parallel-tdd-subagents.md) — original (narrower) design for TDD-cycle subagents. To be restructured or absorbed during canonical doc updates.
- [`ADR-0001`](../../adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md) — proposed inbox/mint id allocation. F-NNN inherits whichever model the framework adopts.
- [`ADR-0003`](../../adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — finding (F-NNN) as 7th entity kind. Sovereignty rule update (§4.5) and recursion-bounds note (§11) pending canonical update.
- [`ADR-0004`](../../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — uniform archive convention. Terminality table cleanup pending canonical update.
- [`design-decisions.md`](design-decisions.md) — kernel principles. Forensics scope boundary (§10.1) pending canonical addition.
- [`provenance-model.md`](provenance-model.md) — principal × agent × scope. Sub-scope FSM extension (§4.2) layers on this model.
- [`tree-discipline.md`](tree-discipline.md) — existing tree-shape rules. Forensic bundle layout (§7.2) and quarantine (§7.3) add a sub-rule.
- [`CLAUDE.md`](../../../CLAUDE.md) — project instructions. Strict-lane principle (§2.1) pending canonical addition.
- [E-0019](../../../work/epics/E-0019-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md) — the first consumer of this substrate. Currently `proposed`; will be rewritten after substrate epics land to consume the substrate rather than re-invent pieces of it.
