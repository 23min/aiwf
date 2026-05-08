# ai-workflow Architecture

> **⚠ Superseded — archived.** This document captures the framework's *original* ambition — an event-sourced kernel, hash-verified projections, monotonic IDs, RFC 8785 canonicalization. It is **not** the current design. The research arc in [`../research/`](../research/) walked most of this back; the current direction lives in [`../working-paper.md`](../working-paper.md), [`../research/KERNEL.md`](../research/KERNEL.md), and [`../research/06-poc-build-plan.md`](../research/06-poc-build-plan.md). This file is kept for lineage — the reasoning is useful and a future framework version may revisit parts of it — but new readers should start with the working paper, not here.

**Status:** historical reference. The sections below describe what ai-workflow was *first imagined* to be: the data model, the event-sourced runtime kernel, the boundary between AI assistants and the deterministic engine, and the principles by which the original design proposed to evaluate future changes. Treat the document as a starting point the project moved on from, not as the active design.

---

## 0. What ai-workflow is

ai-workflow is a markdown-native project-management framework for repositories where humans and AI assistants (Claude, Copilot, others) collaborate over long horizons. It exists to keep the structural state of a project — what's planned, what's in flight, what's decided — coherent across many human-readable surfaces while letting both humans and AI edit those surfaces freely.

A consumer repository checks out the framework (typically as a git submodule under `.ai/`) and runs the install script once. The script writes adapter surfaces (skills, instruction files, templates) into the standard locations the consumer's tools discover (`.claude/`, `.github/`, root `CLAUDE.md`, etc.). From that point, AI assistants in the consumer repo have access to the framework's vocabulary and tooling.

The framework manages a small, well-typed set of entities — **epics**, **milestones**, **decisions**, **ADRs**, **gaps**, **contracts** — each with a stable id, required structural fields, and a lifecycle. Entities live in markdown files under conventional paths. Their structural fields live in YAML frontmatter; their narrative content lives in the body.

The framework's central design commitment: a sharp division of labor between **the AI assistant** (which owns intent, content, and semantic coherence) and **the engine** (a deterministic Go binary that owns structural integrity, atomicity, and constraint enforcement). The engine never generates prose; the assistant never directly mutates structural state. The contract between them is a closed-set vocabulary of typed actions, validated against per-kind boundary contracts, persisted as events in an append-only log, and projected into a hash-verified read-model.

This document describes that architecture in detail.

---

## 1. Thesis

The framework keeps a small set of structured artifacts consistent across many denormalized representations. The structured artifacts form a **relational schema with referential-integrity constraints**, persisted as **markdown files**, accumulated as **events in an append-only log**, and projected into a **hash-verified read-model** for fast queries and drift detection. The kernel is event-sourced: every structural mutation is recorded as a fact before its effect is applied. The engine owns atomicity, the closed-set vocabularies, and the integrity checks. The AI assistant owns prose, intent, and the creative work of recovering from unexpected state. The boundary between them is a typed action envelope, validated against a machine-readable contract.

Everything else in this document follows from these commitments.

---

## 2. The data model

### 2.1 Entities and relationships

The framework manages six entity kinds:

| Entity | Primary key | Lives in | Foreign keys |
|---|---|---|---|
| Epic | `E-NN[a-z]?` | `work/epics/<id>-<slug>/epic.md` | optional parent epic |
| Milestone | `M-NN-<slug>` | `work/epics/<epic>/<milestone>.md` | parent epic; `depends_on` siblings |
| ADR | `ADR-NNNN` | `docs/decisions/NNNN-<slug>.md` | superseded ADRs; referenced epics; contract bundle paths |
| Decision | `D-YYYY-MM-DD-NNN` | `work/decisions.md` (table row) | ratifies epic/milestone/scope-drift |
| Gap | `G-YYYY-MM-DD-NNN` | `work/gaps.md` (table row) | originating milestone/epic |
| Contract bundle | (ADR id + paths) | ADR frontmatter + `docs/architecture/contracts/{schemas,fixtures,examples}/<topic>/` | reference implementation file:line; schema version |

Relationships are typed and machine-readable: `parent`, `depends_on`, `supersedes`, `ratifies`, `cites`. Each relationship resolves a foreign key from one entity to another. The schema is small enough to enumerate; closed enough to validate; rich enough to model real project structure.

### 2.2 Three coordinated representations

The same data is persisted in three coordinated forms. Only one of them is the source of truth.

**(1) Markdown specs — the source of truth.** Each entity lives in a markdown file with YAML frontmatter (structural fields) and a prose body (narrative content). Humans and AI edit these files freely with their normal tools. Git tracks every change.

**(2) The event log (`.ai-repo/events.jsonl`) — the durable kernel.** Every structural mutation is recorded as an append-only event before its effect is applied. The log is the chronological record of how the project's structural state evolved. It is reproducible from the markdown specs (re-scanning produces a fresh projection that should match), but the log is what the framework writes and reads at runtime.

**(3) The graph projection (`.ai-repo/graph.json`) — the derived read-model.** A normalized denormalization of structural fields, computed from the event log (and reconcilable with the markdown specs by re-scanning). The projection includes a SHA-256 hash of its canonical form, allowing drift detection in O(1) compared to byte-for-byte comparison. The projection file is gitignored by default; it is rebuildable.

The narrative projections (ROADMAP rows, CLAUDE.md sections, dependency graphs, audit reports) are denormalized human-readable views derived from the graph projection plus the spec bodies.

### 2.3 What's where, and why

| In frontmatter | In body | In events.jsonl | In graph.json | In projections |
|---|---|---|---|---|
| id, kind, status, parent, github_issue, dates | scope, motivation, design notes, acceptance criteria | every mutation: action, payload, actor, timestamp, hash | resolved structural fields with confidence + evidence | human-readable summaries |

Frontmatter is the membrane: it carries the typed structural fields the engine reads, plus narrative-adjacent fields (title, completed-date) that the projection needs to render summaries. The body is opaque to the engine; the engine never parses prose.

This split is load-bearing. It means the projection stays small (kilobytes), refactor-friendly (renaming a field updates one spec, not many surfaces), and machine-checkable (every value is a string from a closed set, not free-form prose). The cost: *content* drift — a narrative claim out of sync with the structural reality — needs a separate tool family (the doc-lint pipeline), with a different solution shape.

---

## 3. The event-sourced kernel

### 3.1 Why event-sourcing

The framework's most expensive failure modes are partial writes (graph + frontmatter half-applied), drift between surfaces (projection disagrees with specs), and inability to explain how the project got to its current state. An event-sourced kernel addresses all three:

- **Partial writes** become impossible because the event is recorded *before* the effect; if the effect fails, the trace shows what was attempted, and recovery is forward-only.
- **Drift** becomes detectable in O(1) by comparing the projection's stored hash against a recomputed hash from replay.
- **History** is first-class: every structural change is in the log with its actor, timestamp, and payload.

Event-sourcing is implemented as the smallest possible thing that buys these properties: an append-only JSONL file with a closed-set action vocabulary and a deterministic projection function.

### 3.2 The event envelope

Every event in `.ai-repo/events.jsonl` conforms to a closed schema:

```json
{
  "schema_version": "1.0",
  "ts": "2026-04-26T14:32:17Z",
  "run_id": "run-abc123",
  "correlation_id": "session-xyz789",
  "attempt_id": "attempt-001",
  "idempotency_key": "promote-E-19-active-to-complete-001",
  "actor": "claude/session-abc",
  "action": "promote",
  "payload": {
    "node": "E-19",
    "kind": "epic",
    "from_status": "active",
    "to_status": "complete"
  },
  "post_state_hash": "sha256:abc...",
  "patch_sha256": "sha256:def..."
}
```

Required fields:

| Field | Why |
|---|---|
| `schema_version` | Forward-compat for envelope changes |
| `ts` (RFC 3339 UTC) | Total ordering, replay reproducibility |
| `run_id` | Groups events from one engine invocation |
| `correlation_id` | Groups events from one user/agent intent across multiple engine calls |
| `attempt_id` | Distinguishes retries of the same logical action |
| `idempotency_key` | Detects duplicate events on replay; skipped without effect |
| `actor` | `claude/<session>`, `human/<git-ident>`, `ci/<run-id>` |
| `action` | One of the closed-set verbs (§5) |
| `payload` | Typed per action; validated against the action's JSON Schema |
| `post_state_hash` | SHA-256 of the canonicalized projection after this event |
| `patch_sha256` | Hash of the underlying patch, when applicable |

The payload schema for each action lives in `framework/schemas/actions/<action>.json` (JSON Schema 2020-12). The engine validates incoming actions against the schema before persisting.

### 3.3 Trace-first writes

The engine's transaction order for any structural mutation:

1. **Validate** — the proposed action against its JSON Schema, the boundary contract for the entity kind, and the cross-entity invariants given current projection state.
2. **Append event** — to `events.jsonl` under an `O_APPEND` write protected by a process-level flock. The event is recorded as a *fact about what is being attempted* before any irreversible effect.
3. **Apply effects** — write spec frontmatter changes; update the graph projection; regenerate any deterministic fenced sections (ROADMAP table rows, CLAUDE.md status sections).
4. **Append confirmation event** — `effect.applied` with the post-state hash. If any step in (3) fails, this event is *not* written, and the trace shows the attempt without confirmation.
5. **Return result** — to the caller. The result includes the event sequence number and the post-state hash, so the caller can verify or replay.

If a process crashes between step 2 and step 5, the recovery path is: re-run the verify command (§3.5). It detects the missing confirmation, re-applies idempotently if safe, or surfaces the inconsistency as a finding.

### 3.4 Closed-set action vocabulary

Initial vocabulary, deliberately small:

| Action | Effect |
|---|---|
| `node.add` | Create a node (id allocated by engine), add parent edge |
| `node.promote` | Status transition (validated against constraints) |
| `node.pause` / `resume` | Lifecycle vocabulary |
| `node.block` / `unblock` | Lifecycle vocabulary |
| `node.cancel` | Terminal cancellation |
| `node.remove` | Remove node + cascading edges |
| `node.rename` | Rename id (cascades to all edges and references) |
| `edge.add` / `edge.remove` | Manage `depends_on`, `ratifies`, `supersedes` |
| `effect.applied` | Confirmation paired with a preceding mutation event |
| `agent.result` | Validated agent intention (proposes one or more node/edge actions) |
| `output.rejected` | An agent action rejected by validation; logged for empirical contract tuning |
| `verify.start` / `verify.ok` / `verify.fail` | Audit pipeline events |
| `run.init` / `run.end` | Run boundary markers |

This is a closed enum. Adding an action requires shipping a payload schema, the engine code that handles it, and the boundary-contract entries that authorize it per kind. Speculative actions are not added.

### 3.5 Hash-verified projection and replay

The graph projection (`graph.json`) is computed by a pure function from the event log:

```text
graph := project(events)
```

The projection is canonicalized using **JSON Canonicalization Scheme (RFC 8785)**: keys sorted lexicographically, no whitespace, UTF-8, deterministic number representation. The SHA-256 of the canonicalized form is stored on the projection itself as `projection_hash_sha256`.

The verify command:

```bash
aiwf verify
```

Replays events from origin (or from the last confirmed snapshot, if snapshotting is enabled), recomputes the projection, recomputes the hash, and compares against the stored hash. Three outcomes:

- **`ok`** — hashes match; the projection is consistent with the event log.
- **`mismatch`** — hashes differ; the projection has been mutated outside the engine, or the event log has been truncated. Surfaces as a finding with the divergence location.
- **`corrupted`** — the projection cannot be deserialized, or the event log fails schema validation. Hard finding; requires operator intervention.

Verify is also run as a CI gate and as the final step of every engine invocation. The cost is sub-second for project sizes the framework targets.

### 3.6 Immutability of done

Terminal states are terminal. The engine rejects transitions out of `complete` and `cancelled`. To correct a defect in completed work:

```bash
aiwf hotfix M-PACK-A-02 --reason "discovered missing acceptance criterion"
```

This creates a new entity (`M-PACK-A-02-hotfix-1`, kind `emergency_patch`, parent `M-PACK-A-02`) with its own lifecycle. The original stays `complete`; the history is honest about what happened.

This rule prevents the most common cause of structural drift: hand-editing frontmatter to flip a `complete` back to `active` because something was missed.

### 3.7 What is *not* in the event log

The event log records **structural mutations only**:

- Mutations of nodes and edges in the graph.
- Validation rejections (`output.rejected`).
- Run boundaries and verify outcomes.

The event log does **not** record:

- Audit findings from external scans (those are tool output, not mutations).
- Doc-lint findings.
- Body-prose edits to specs (the markdown is the source of truth for prose; git tracks it).
- File operations on `src/` or other non-structural files.

Keeping the log narrow is what makes the kernel small and trustworthy.

---

## 4. The LLM/engine boundary

### 4.1 Two actors, sharply divided

| | **AI assistant** | **Engine** |
|---|---|---|
| **Reads** | Markdown bodies, user intent, conversation history, the projection (via the engine), the event log (via the engine) | Frontmatter, edges, projection, event log, boundary contracts |
| **Knows** | What things mean; whether a spec is well-written; what the user wants; what's idiomatic | What's structurally allowed; what would happen if X transitioned; whether two surfaces agree |
| **Authors** | Markdown content; agent.result envelopes; commit messages; tracking-doc bodies; ADR rationale; CLAUDE.md narrative | Nothing it doesn't have to. JSON envelopes, finding lists, hashes |
| **Decides** | What to propose; how to phrase; which trade-off to surface | Whether a proposed action is *legal* given current state |
| **Guarantees** | Nothing on its own — interpretive, stochastic, can be wrong | Atomicity; constraint correctness; integrity; replay reproducibility |

The AI assistant is *interpretive*. It can be wrong, mealy-mouthed, or convinced of things it shouldn't be. The engine is *deterministic*. It refuses an action and explains in machine-readable terms.

### 4.2 The contract between them

The contract is a **typed action envelope** (`agent.result`) validated against a **per-kind boundary contract** (§6). The assistant constructs an envelope; the engine validates and either applies or rejects it.

This is not a patch YAML. The envelope is JSON with a closed-set `action` field and a payload schema selected by the action. The assistant calls the engine via typed function-calling (MCP, OpenAI tools, or CLI subcommands that map 1:1 to the action vocabulary). Patches exist as an *internal* representation the engine uses for batching and event payloads; they are not the assistant's authoring surface.

### 4.3 What the engine does

The engine's responsibilities — and only these:

1. **Atomic structural mutations.** Append event → apply effect to specs/projection → append confirmation. No other actor writes structural files.
2. **Closed-set validation.** Vocabulary, action schemas, status enums, kind enums.
3. **Cross-row constraint enforcement.** "Milestone cannot transition to active while a `depends_on` milestone is in `draft`." Enforced at write time.
4. **Knowledge queries.** "What's the legal next state for X?" "What's the branch-name convention for milestone Y?" "What's blocked by E-19?" "Which template applies to a `spec` task in this kind?"
5. **Projection management.** Build, hash, verify, expose.
6. **Event log management.** Append, query, replay.
7. **Audit checks.** Drift detection across surfaces (§8).

### 4.4 What the AI assistant does

Everything else, specifically:

1. **Author markdown content.** Spec bodies, ADR rationale, tracking-doc prose, commit messages, CLAUDE.md narrative paragraphs.
2. **Run git commands directly.** Branch creation, stashing, rebasing, recovery from unexpected git state. The assistant is excellent at this; engine wrapping would lose flexibility.
3. **Edit `src/` and other non-structural files.** Code, configuration, tests. The engine has no opinion.
4. **Compose `agent.result` envelopes** for structural mutations. Calls engine verbs.
5. **Recover creatively** from engine-rejected actions: read the finding, adjust intent, retry differently.
6. **Orchestrate** compound flows by composing engine verbs and prose authoring.

### 4.5 The split for mechanical post-effects

When a structural mutation has post-effects (cutting a branch, scaffolding a tracking doc, regenerating fenced sections), they split along the same line:

- **Engine-side post-effects (deterministic transformations of structural state):** ROADMAP table-row regeneration, CLAUDE.md fenced-section regeneration, projection update, event append. Atomically with the mutation.
- **Assistant-side post-effects (free-form mechanical work):** branch creation (using engine-emitted naming convention), tracking-doc scaffolding (using engine-emitted template path), spec-body authoring, narrative paragraphs in CLAUDE.md.

The litmus test for which side an operation belongs on:

> Does the assistant doing this directly risk a partial-write or invariant violation it can't reliably handle?

If yes → engine. If no → assistant. Most candidate operations end up assistant-side, which is what keeps the engine small.

### 4.6 What the boundary forbids

These rules are non-negotiable:

- The engine never generates prose. No "intelligent commit messages," no "AI-suggested resolutions," no narrative summarization. The engine emits structured data.
- The assistant never writes the projection or the event log directly. Always via engine verbs.
- The assistant never invents IDs. ID allocation is the engine's job to prevent collisions across parallel sessions.
- The engine never depends on assistant judgment to validate state. If the engine cannot decide legality from schema + contracts + projection, the schema is missing a declaration; fix the schema, don't add cleverness.
- The engine is invocable without an AI assistant. Every binary takes flags, reads stable input formats, emits a JSON envelope. Humans, CI scripts, and other tools drive it directly.

---

## 5. The verb set

A relational system has roughly four verb classes: **DDL** (define schema), **DML** (mutate data), **DQL** (query data), and **DCL** (govern access). The framework's verbs map cleanly:

### 5.1 DDL

Schema is declared in the framework source, not at runtime. There are no runtime DDL verbs; new entity kinds, action types, or status enums require a framework version bump.

### 5.2 DML — write verbs

Each verb is a typed function call. The CLI form maps 1:1 to the action vocabulary in §3.4.

| Verb | Effect |
|---|---|
| `aiwf init` | Bootstrap `.ai-repo/` from existing markdown tree |
| `aiwf add <kind> [parent]` | Allocate id, create node, add parent edge |
| `aiwf promote <id> --to <state>` | Status transition with constraint validation |
| `aiwf pause / resume / block / unblock <id>` | Lifecycle vocabulary |
| `aiwf cancel <id>` | Terminal cancellation |
| `aiwf hotfix <id>` | Spawn a correction child for a closed entity |
| `aiwf remove <id>` | Remove node + cascading edges |
| `aiwf rename <id> <new-id>` | Rename with edge cascade |
| `aiwf split / merge` | Restructure entity boundaries |
| `aiwf apply --envelope <file>` | Apply a validated agent.result envelope |

### 5.3 DQL — read verbs

| Verb | Effect |
|---|---|
| `aiwf query [filter]` | Return matching nodes/edges as JSON |
| `aiwf transitions <id>` | List legal next states from the entity's current state |
| `aiwf blocked-by <id>` | Walk depends_on edges |
| `aiwf branch-name-for <id>` | Return the canonical branch name for an entity |
| `aiwf template-for <kind> <task>` | Return the template path for scaffolding |
| `aiwf history [--node <id>] [--since <when>]` | Filter the event log |
| `aiwf render` | Render dependency graph (mermaid, dot) |
| `aiwf report [--since]` | Structural snapshot for humans |
| `aiwf verify` | Replay + hash-compare; return `ok` / `mismatch` / `corrupted` |
| `aiwf audit` | Run all surface-drift checks; emit findings |

### 5.4 DCL

Governance is human review and git permissions. There are no runtime governance verbs.

### 5.5 Output discipline

Every verb emits a JSON envelope on stdout:

```json
{
  "tool": "aiwf",
  "version": "0.x.y",
  "status": "ok | findings | error",
  "findings": [...],
  "result": {...},
  "metadata": {...}
}
```

`--pretty` indents for human reading; the unindented default is what consumers parse. Logging goes to stderr via structured logging (`log/slog`). Exit codes: `0` ok, `1` findings, `2` usage error, `3` internal error.

---

## 6. Boundary contracts

### 6.1 Per-kind contracts as data

Each entity kind has a machine-readable contract under `framework/contracts/<kind>.yaml`:

```yaml
kind: milestone
schema_version: "1.0"
required_frontmatter:
  - id
  - kind
  - status
  - parent
  - title
status_enum:
  - draft
  - active
  - blocked
  - complete
  - cancelled
permitted_actions:
  - node.add
  - node.promote
  - node.pause
  - node.resume
  - node.block
  - node.unblock
  - node.cancel
  - node.remove
  - node.rename
  - edge.add
  - edge.remove
prohibitions:
  - direct_write_to_projection
  - direct_write_to_event_log
  - status_to_complete_with_open_blockers
  - status_change_from_terminal_state
transition_constraints:
  draft_to_active:
    require: "all depends_on edges target nodes in terminal state"
  active_to_complete:
    require: "all acceptance_criteria checked"
output_schemas:
  node.add: schemas/actions/node-add.json
  node.promote: schemas/actions/node-promote.json
  # ...
templates:
  spec: framework/templates/milestone-spec.md
  tracking_doc: framework/templates/milestone-tracking-doc.md
naming:
  branch: "milestone/{id}-{slug}"
```

### 6.2 What contracts replace

Contracts consolidate what was previously diffuse:

- The closed-set status enum is no longer hardcoded in Go alone; it's declared in YAML and the Go enum is generated from it (or validated against it at startup).
- "Required frontmatter fields" is no longer scattered across audit checks and skill prose; it's declared once.
- "Which actions are permitted on this kind" is no longer implicit; it's explicit and validated.
- Branch naming, template paths, and other conventions are queryable via `aiwf branch-name-for` instead of duplicated in each lifecycle skill.

### 6.3 What contracts don't do

Contracts are **data**, not a meta-language. They declare facts about what is allowed; they do not contain conditionals, expressions, or logic. The engine reads the YAML, applies the constraints in compiled Go, and reports findings on violation. Adding new constraint *types* (new YAML keys the engine recognizes) is a framework-version change. Adding new constraint *values* (new transitions, new prohibitions for existing types) is a content edit.

This is the right tradeoff: YAML for the data that varies per kind, Go for the logic that interprets it. Avoid the temptation to grow the YAML into a Turing-complete constraint language.

---

## 7. Constraints and lifecycle

### 7.1 Closed-set vocabularies

The framework enforces closed sets at multiple layers:

- **Entity kinds:** `epic`, `milestone`, `adr`, `decision`, `gap`, `contract` (plus consumer extensions, declared via additional YAML files).
- **Status enums:** per-kind, declared in the boundary contract.
- **Action vocabulary:** per §3.4.
- **Edge types:** `parent`, `depends_on`, `supersedes`, `ratifies`, `cites`.
- **Confidence levels:** `high`, `medium`, `low`, `none`.
- **Severity levels:** `low`, `medium`, `high`, `critical`.

Closed-set means: ship only the values with current call sites. Speculative future values are not added until a real consumer needs them.

### 7.2 Transitions and constraints

Status transitions are validated as cross-row constraints. Examples:

| Transition | Constraint |
|---|---|
| `milestone:draft → active` | All `depends_on` blockers in terminal state |
| `milestone:active → complete` | All acceptance-criteria boxes checked |
| `epic:planning → active` | At least one milestone exists in `draft` |
| `epic:active → complete` | All child milestones in terminal state |
| `epic:* → cancelled` | Operator confirmation (terminal, no recovery) |
| `adr:proposed → accepted` | Required frontmatter fields present |
| `adr:* → superseded` | A superseding ADR exists and references this one's id |
| `* → terminal_state → *` | Forbidden (immutability of done; use `hotfix` instead) |

Constraints are implemented in compiled Go in `tools/internal/wfgraph/validate/`. The boundary contract declares *which* constraints apply to which transitions; the Go code implements *how* they're checked. New constraint logic requires a framework version bump.

### 7.3 Effects and propagation

Effects are mutations the engine performs as part of a successful transition:

- Atomic projection + frontmatter write.
- Regeneration of any fenced sections (ROADMAP table row, CLAUDE.md status section) that reference the entity.
- Event log append (one mutation event + one confirmation event).

After a transition, the engine emits a **propagation preview**: which downstream entities just had a constraint become satisfied? This is *visibility, not automation*. The engine reports "M-PACK-A-01 was blocked-by E-19 (now satisfied; eligible to start)"; it does not automatically transition M-PACK-A-01. Humans decide when work starts; the engine only enforces when it can't start yet.

### 7.4 The transitioning problem, solved by trace-first

Trace-first writes (§3.3) are the framework's answer to partial-failure recovery. The event-before-effect ordering means:

- Crash between event and effect: the trace shows the attempt; verify detects the missing confirmation; the operator (or `aiwf finalize`) re-applies idempotently.
- Crash between effect and confirmation: the effect is in place but unconfirmed; verify detects the inconsistency and emits a finding; the operator confirms or rolls forward.
- Crash after confirmation: clean state.

In all cases, recovery is forward-only. The framework never rolls back a confirmed transition; defects are corrected by `hotfix` (§3.6).

---

## 8. The audit pipeline

### 8.1 What the audit checks

The audit is a deterministic surface-drift detector. It runs `aiwf audit` and emits findings in four groups:

| Group | What it checks |
|---|---|
| **Entity integrity** | Per-entity invariants — required fields present, vocabulary in closed set, status reachable from history |
| **Cross-entity integrity** | Foreign-key resolution — every cited id resolves; relationships consistent on both sides |
| **Projection consistency** | Denormalized views match their source — ROADMAP rows match graph; fenced sections match canonical state; numeric claims match counts |
| **External system sync** | Framework state matches git/external state — branch naming consistent with milestone ids; CHANGELOG entries match closed milestones; release tags exist for shipped milestones |

Each group emits structured findings:

```json
{
  "code": "ROADMAP_ROW_DRIFT",
  "severity": "medium",
  "node": "M-PACK-A-02",
  "surfaces": {
    "graph": "active",
    "roadmap": "draft"
  },
  "message": "ROADMAP shows status 'draft' but graph projection shows 'active'",
  "fix_hint": "regenerate ROADMAP fenced section: aiwf render --target roadmap"
}
```

### 8.2 Findings vs. failures

The audit never refuses to load. Inconsistent state is a finding to surface, not a parse failure that hides the problem. This applies to every read path in the framework: the engine loads what's there, validates what it can, and emits structured findings for everything that's wrong.

### 8.3 When the audit runs

- On every `aiwf` invocation as a fast pre-flight (entity integrity only).
- As CI gate before merge (full pipeline, gating on severity).
- On `aiwf audit` for explicit operator inspection.
- After every successful structural mutation as a verification step.

---

## 9. The skills layer

### 9.1 What skills are

Skills are markdown files under `framework/skills/` that AI assistants invoke. Each skill is a tightly-scoped checklist of operations, typically:

1. Query engine state (`aiwf query`, `aiwf transitions`).
2. Compose an `agent.result` envelope from the user's intent.
3. Submit to the engine (`aiwf apply`).
4. Author any prose that the engine handed off (tracking-doc body, narrative paragraph).
5. Run any non-structural file operations the engine doesn't own (git, code edits).

### 9.2 Skill design rules

- **Small.** Each skill is single-purpose; 50-150 lines is the target. Compound flows are achieved by composition (skill A invokes skill B), not by monolithic skills.
- **Lazy-loaded.** The host gets an *index* of available skills (name + 1-line description) at startup. Skill bodies load only when invoked. This is the largest single LLM-cost lever in the framework.
- **Composable.** A lifecycle flow like "wrap a milestone" is composed of: `wf-validate-transition`, `wf-apply-mutation`, `wf-scaffold`, `wf-narrate`, `wf-gate`. Each is invocable independently.
- **Engine-thin.** Skills delegate structural work to engine verbs; their prose covers intent, content, and dialogue, not patch composition.
- **Errors-as-findings aware.** A skill that gets findings back from the engine reads them, decides what to do, and either retries with a different envelope or surfaces the issue to the operator.

### 9.3 Skill categories

| Category | Examples |
|---|---|
| **Query** | `wf-status`, `wf-blocked-by`, `wf-history` |
| **Lifecycle** | `wf-add-epic`, `wf-promote`, `wf-pause`, `wf-wrap-milestone`, `wf-wrap-epic` |
| **Authoring** | `wf-draft-spec`, `wf-draft-adr`, `wf-narrate` |
| **Audit** | `wf-audit`, `wf-verify`, `wf-doc-lint` |
| **Recovery** | `wf-finalize`, `wf-hotfix`, `wf-reconcile` |

---

## 10. Modules and pluggability

### 10.1 Why modules

Not every consumer needs every capability. A small CLI project may want only epic + milestone tracking; a complex platform may want ADRs, contract bundles, doc-lint, and GitHub-Issues sync. Bundling everything as one monolithic install bloats the skill index, the adapter surface, and the LLM context cost for projects that don't use it.

The framework ships as a **core kernel** plus **opt-in modules**. Each module is a self-contained directory of skills, contracts, schemas, templates, and adapter rules that the consumer enables explicitly. Disabling a module removes its surface area without touching entity data.

### 10.2 Core vs optional

**Core (always present, not a module):**

- The event-sourced kernel (§3): events.jsonl, projection, verify.
- The verb engine (§5): the universal verbs that operate on any node/edge.
- The audit framework (§8): the structured-finding pipeline.
- The skill-index machinery (§9.2).
- The closed-set vocabularies that aren't kind-specific (Confidence, Severity, edge types).

**Required modules** (the framework is not useful without these; new consumers get them by default and would have to actively opt out):

- `epic` — epic kind + lifecycle.
- `milestone` — milestone kind + lifecycle.

**Optional modules** (default-on for new consumers; suggested by detection):

- `adr` — ADR tracking (suggested when `docs/decisions/` exists).
- `roadmap` — ROADMAP.md fenced-section regeneration.
- `narrative` — CLAUDE.md fenced-section maintenance.
- `decisions` — lightweight decisions table.
- `gaps` — gaps tracking.

**Optional modules** (default-off; opt-in only):

- `contracts` — contract bundles + per-PR contract verification.
- `github-sync` — GitHub Issues mirror + drift detection.
- `release` — versioning + release tagging + tag-existence pre-flight.
- `doc-lint` — narrative drift detection (does the prose match the structural state?).

### 10.3 Module structure

Every module is a directory under `framework/modules/<name>/`:

```text
framework/modules/<name>/
├── MODULE.yaml          # metadata: name, description, version, requires, provides, default
├── README.md            # human-readable doc
├── skills/              # markdown skills the module exports (lazy-loaded)
├── contracts/           # boundary contracts for any new entity kinds
├── schemas/             # JSON schemas for any new actions
├── templates/           # templates for scaffolding
└── adapters/            # rules-injection text per host (claude, copilot)
```

`MODULE.yaml` example:

```yaml
name: adr
description: Architecture Decision Records (ADRs) tracking
version: 1
default: true
requires: [core]
provides:
  kinds: [adr]
  actions: [adr.add, adr.accept, adr.supersede, adr.deprecate]
  skills: [draft-adr, supersede-adr, list-adrs]
  templates: [adr-spec]
detection:
  - path: docs/decisions/
    suggest_enable: true
```

### 10.4 Consumer-side configuration

The consumer's `.ai-repo/config/modules.yaml` enumerates enabled modules:

```yaml
modules:
  - name: epic
  - name: milestone
  - name: adr
  - name: roadmap
  - name: narrative
  - name: decisions
  - name: gaps
```

The installer (`aiwf init`) writes this file based on detection + user confirmation. Subsequent management is via `aiwf module enable / disable / list / info`. Adapter regeneration runs automatically when the file changes.

### 10.5 Module interaction rules

To keep modules composable without a full plugin API:

- **Modules can declare new entity kinds** (with their own boundary contracts and templates).
- **Modules can declare new actions** (with payload schemas, validated by the engine).
- **Modules cannot mutate core data structures** at runtime — they extend the schema by declaration.
- **Module skills can call other modules' verbs** via the engine; they cannot call other modules' skills directly.
- **Cross-module dependencies are explicit** in `requires:` and validated at install time.
- **Modules version with the framework**, not separately. There is one release tag.

### 10.6 What modules don't do

Deliberately out of scope, per the framework's KISS/YAGNI commitments:

- No plugin marketplace.
- No remote module loading.
- No third-party module API (modules ship in the framework repo only).
- No dynamic dependency resolution.
- No per-module versioning.

If any of these become genuinely needed, revisit. Not before.

### 10.7 Onboarding flow

`aiwf init` is the single entry point. It:

1. Detects existing structure (looks for the `detection.path` of every module).
2. Suggests modules based on detection + the default-on/default-off split.
3. Shows a preview of what will be created or modified.
4. Confirms with the user.
5. Writes `.ai-repo/config/modules.yaml` and `.ai-repo/events.jsonl` (with a `run.init` event).
6. Generates the consumer's adapter surfaces (`.claude/skills/`, `.github/skills/`, `CLAUDE.md` framework section).
7. Runs `aiwf verify` to confirm the install is consistent.

The installer never overwrites consumer-owned files — it appends fenced sections to `CLAUDE.md` and `README.md` with clear delimiters. Users can opt out of any module mid-flow without halting.

---

## 11. Cost discipline

LLM context cost is the framework's largest unaccounted-for expense at scale. The architecture treats it as a first-class constraint:

- **Engine emits structured data, not prose.** A status report is JSON; a renderer (in the skill or in a downstream tool) shapes it for humans. Tokens used per query are minimized.
- **Skills load on demand.** No skill body is in context until invoked. The skill *index* is small.
- **Closed schemas reduce composition cost.** The assistant doesn't free-form a YAML envelope; it fills typed fields enforced by JSON Schema.
- **The projection is queryable in pieces.** `aiwf query --kind epic --status active` returns just the matching nodes, not the whole graph.
- **The "purified view" pattern.** When an assistant needs context for a decision, it requests a derived view (current state + relevant facts), not the raw projection or event log. This pattern is what keeps long-running sessions from drowning in irrelevant context.
- **Engine reports are findings, not narratives.** The audit emits structured findings; the assistant or a skill turns them into prose for the user. The engine never tries to be a writer.

These choices cumulatively halve typical session token cost compared to a naive design where skill bodies, full projections, and prose summaries are all eagerly loaded.

---

## 12. The principles checklist

When evaluating any proposed change to the framework:

1. **Does it move structural truth toward the engine, or away from it?** Toward is good.
2. **Does it move content toward the assistant, or away from it?** Toward is good.
3. **Does it require assistant judgment to validate structure?** If yes, the schema is missing a declaration; fix the schema, don't add cleverness.
4. **Does it require engine support for content?** If yes, the engine is sliding toward generating prose. Stop.
5. **Does it preserve trace-first ordering?** Every mutation must record an event before applying its effect.
6. **Does it preserve the projection's hash-verifiability?** If a change makes the projection non-canonical or non-replayable, it's wrong.
7. **Does it preserve atomicity?** A change that can leave structure in a half-mutated state needs a recovery story.
8. **Does it close a verb-set gap or open a new one?** Symmetric verb sets are easier to reason about than asymmetric ones.
9. **Does it auto-do something a human should decide?** If yes, replace the auto-do with a propagation preview.
10. **Could a non-AI consumer (a CI script, a different agent, a human at the CLI) drive this with the same outcome?** If no, the engine has grown too dependent on a specific assistant.

The framework's central commitment is the boundary in §4. The boundary is what makes everything else stable. Keeping it sharp is the work.

---

## 13. What this architecture does *not* try to be

To prevent scope creep, an explicit list of non-goals:

- **Not a code-execution agent.** The framework does not run an LLM, dispatch tasks, or orchestrate agent loops. It manages the *artifacts* an agent produces, not the agent.
- **Not a replacement for git.** Branches, commits, and merges are the assistant's job using normal git tooling. The framework records *which* milestone a branch corresponds to, not the contents of the branch.
- **Not a build system.** Constraint validation runs in milliseconds against the projection; it does not invoke compilers or test runners. Those happen in CI, with the framework's audit as a gating signal.
- **Not a knowledge graph for arbitrary facts.** The graph stores six entity kinds and five edge types. It is not a general-purpose triple store.
- **Not a narrative drift detector.** Whether the prose in a CLAUDE.md narrative *means* what the structural state says is an open semantic problem. The framework checks reference integrity (every cited id resolves); it does not check claim integrity (every claim is true).
- **Not a multi-agent orchestrator.** Multiple AI assistants can use the framework concurrently; the event log preserves total ordering. But the framework does not assign work to agents or coordinate their plans.

Each of these non-goals exists to keep the framework's surface small. Many adjacent systems exist to fill these roles; the framework integrates with them but does not replace them.

---

## Appendix A — File layout

A consumer repository using ai-workflow has this layout:

```text
<repo-root>/
├── .ai/                           # framework submodule (read-only from consumer)
│   ├── skills/
│   ├── templates/
│   ├── contracts/                 # boundary contract YAMLs
│   ├── schemas/                   # JSON schemas for actions
│   ├── tools/                     # Go binaries (or built into PATH)
│   ├── sync.sh
│   └── ...
├── .ai-repo/                      # framework runtime state (consumer-local)
│   ├── events.jsonl               # append-only event log (canonical)
│   ├── graph.json                 # derived projection (gitignored or committed)
│   ├── snapshots/                 # optional: periodic projection snapshots
│   ├── config/
│   │   ├── artifact-layout.json
│   │   └── ...
│   └── cache/
├── .claude/                       # generated adapters for Claude
│   ├── skills/wf-*/
│   └── rules/ai-framework.md
├── .github/                       # generated adapters for Copilot
│   ├── skills/<name>/
│   └── copilot-instructions.md
├── work/
│   ├── epics/
│   │   └── <id>-<slug>/
│   │       ├── epic.md
│   │       └── M-NN-<slug>.md
│   ├── decisions.md
│   └── gaps.md
├── docs/
│   ├── decisions/                 # ADRs
│   └── architecture/
│       └── contracts/             # contract bundles
├── ROADMAP.md
├── CLAUDE.md
├── CHANGELOG.md
└── README.md
```

The `.ai-repo/` directory holds the consumer's framework state. The `events.jsonl` is the durable kernel; everything else is derivable from it plus the markdown specs.

## Appendix B — Glossary

| Term | Meaning |
|---|---|
| **Entity** | A first-class artifact the framework tracks (epic, milestone, ADR, decision, gap, contract). |
| **Schema** | The typed definition of what fields entities have and what relationships are legal. |
| **Source of truth** | The markdown spec files. Reproducible inputs to the projection. |
| **Event log** | `.ai-repo/events.jsonl`. Append-only chronological record of structural mutations. The runtime kernel. |
| **Projection** | `.ai-repo/graph.json`. Derived read-model with hash for drift detection. Rebuildable from events. |
| **Action** | One of the closed-set verbs in §3.4 the engine accepts. |
| **agent.result** | The validated JSON envelope an AI assistant submits to propose a mutation. |
| **Boundary contract** | The per-kind YAML declaring permitted actions, schemas, prohibitions, and conventions. |
| **Constraint** | A condition on a transition or relationship the engine enforces at write time. |
| **Effect** | A mutation the engine performs as part of a successful action (projection write, frontmatter update, fenced-section regeneration). |
| **Propagation** | Downstream entities whose constraints become satisfied by an upstream transition. Visibility, not automation. |
| **Trace-first write** | The pattern of recording the event before applying the effect. |
| **Hash-verified projection** | A projection whose canonical form's SHA-256 is stored alongside it, enabling O(1) drift detection. |
| **Immutability of done** | Terminal states never reverse; corrections spawn new entities. |
| **Errors as findings** | Inconsistent state surfaces as structured findings, not parse failures. |
| **The boundary** | The division of labor between the AI assistant (intent + content) and the engine (structure + atomicity). |
