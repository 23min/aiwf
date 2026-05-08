# Subagents design — research todo (scratch)

Working notes on holes / expansion points in the parallel-TDD-subagents design (`parallel-tdd-subagents.md` + `ADR-0003` + `ADR-0004`). Branch: `research/subagents-design`. Delete or fold into the design docs when settled.

Going through one at a time, Q&A format.

## Plan

What we did in this session was much broader than `parallel-tdd-subagents.md` (which is scoped to TDD-cycle subagents). We worked out a substrate that includes the agent registry, role-based concurrency, sub-scope provenance, declarative work-shapes (pipelines), forensic bundles, harvest-as-public-contract, and the strict-lane scope boundary. TDD subagents become *one consumer* of that substrate.

Two-session approach:

1. **This session:** produce a thorough synthesis document at `docs/pocv3/design/agent-orchestration.md`. Top-to-bottom design synthesis (motivation, principles, agent model, work-shape declarations, provenance/forensics, harvest, scope boundaries, deferred items, glossary). Authoritative source for everything decided.
2. **Future session:** fold the synthesis into canonical docs:
   - `CLAUDE.md` — strict-lane principle, sovereignty rule update.
   - `docs/pocv3/design/design-decisions.md` — forensics scope boundary.
   - `docs/adr/ADR-0003` — sovereignty rule (D6 6a), recursion-bounds note (D6 6d).
   - `docs/adr/ADR-0004` — terminality table cleanup (D6 6b).
   - `docs/pocv3/design/parallel-tdd-subagents.md` — restructure as a consumer of the substrate, or absorb entirely.

This scratch stays alongside the synthesis as the Q&A trail — forensic record of how each decision got made, branches considered, leans-vs-picks. Useful for future readers who want to know "what did they reject?". Move to `docs/pocv3/archive/` after the synthesis is committed if the conversation history lookup gets stale.

Synthesis doc name: `agent-orchestration.md` — neutral on the cycle/loop/workflow/pipeline terminology that's still open (D3).

---

## 1. Edit-scope audit is LLM-behavior-dependent — needs a structural chokepoint
- Current design (`parallel-tdd-subagents.md:162`): "(a) subagent's system prompt restricts allowed paths and verbs; (b) parent audits the worktree diff post-cycle."
- Both are LLM-behavior-dependent — the very thing the design is supposed to fix.
- **Reframed (see #4 → role-based decision):** chokepoint is *role-determined*.
  - **Read-only / additive roles** (reviewer, planner-as-auditor, doc-gardener, test-additive): substrate denies out-of-surface writes. For Claude Code host: gitignored hook under `.claude/` denies Edit/Write tool calls outside the role's write surface. Enforcement is binary (allowed / denied), not negotiated.
  - **Builder role**: deviation-detection check rule (`scope-expanded`) compares the worktree diff against the cycle's *coarse* scope hint and emits a finding. The finding is *blocking* per the F-NNN AC-closure rule (`findings-block-met`), so surface is mechanical even if the prompt drifts. Not a hard merge block — an AC-closure block.
  - **Verb gate (L2)** applies regardless of role: subagents can't run `cancel`, `reallocate`, `authorize`, `--force`. Cheap, universal.
- Status: **resolved (frame); implementation details ride E-19**

## 2. Sub-agent delegation is reserved by G22 but the design assumes it works
- `provenance-model.md:93` reserves whether an authorize scope can spawn a sub-scope.
- `design-decisions.md:174` files sub-agent delegation under G22 / deferred.
- Design hand-waves provenance (`parallel-tdd-subagents.md:170-175`): both parent and subagent commit as `ai/claude`, flattening two levels of agency.
- **Resolution (D2):** hybrid — role-tagged actor for everything; sub-scope FSM only on builder-parallel dispatch. BYO-agent supported via capability/name split.
- Status: **resolved (frame); registry schema + scope-nesting details ride E-19**

## 3. Failure recovery is missing
- End-to-end flow assumes successful return + clean worktree + valid JSON.
- Unspecified: subagent runs out of context mid-cycle; commits code but never promotes; returns malformed JSON; claims pass but tests fail when re-run in parent; leaves dirty worktree.
- **Resolution (D4 + D5):** layered recovery — orchestrator-driven reap (primary) + kernel-side stale-cycle GC (safety net) + quarantine for forensics (preserves bundle); per-capability differentiation; sub-scope FSM extended to `ended-success | ended-failure | ended-discarded`; A2A-shaped cycle envelope (concepts borrowed, transport rejected); permanent provenance via trailers + ephemeral forensic bundles; harvestable export surface as public contract. **D5 clarifies scope boundary:** kernel guarantees metadata permanence + export tool; content delivery to durable storage is consumer responsibility (aspirational hook idea noted but not planned).
- Status: **resolved (frame); cycle envelope schema + reap verbs + export verb ride E-19**

## 4. Disjoint-fileset declaration mechanism is deferred but load-bearing
- `parallel-tdd-subagents.md:185` defers "parent's parallelization heuristic" to dogfooding.
- **Reframed:** declaring an exact fileset in advance is impossible — fileset is the *output* of the work, not an input. Most ACs in this repo are real engineering, not typo-fix territory.
- **Resolution: role-based concurrency policy.**
  - Lifetime always bounded (drift fix, universal). Concurrency is a per-role policy.
  - Read-only roles (reviewer, audit, lint): parallel by default. Substrate denies writes; no fileset problem.
  - Additive-write roles (test-additive, doc-gardener): parallel by default. Substrate restricts writes to bounded surface (new `*_test.go` files in declared package; `docs/**` partition). Each subagent gets a partition assignment from the parent.
  - Builder role: sequential default. Parallel only with explicit human dispatch + a *coarse* scope hint (package or directory level). Audit is `scope-expanded` deviation finding (see #1), not a strict whitelist.
  - Deployer: sequential by definition.
- **Where the policy lives:** kernel default per role; project override in `aiwf.yaml.subagents.roles[]` (or similar). New roles must declare a concurrency policy at definition time, or default to sequential.
- **Where the coarse-scope hint lives** (builder-parallel only): dispatch-time argument from the human (or the parent's heuristic when one ships). Form: glob/path list. Granularity: package or directory (not file).
- Status: **resolved (frame); aiwf.yaml schema + check rule code ride E-19**

## 5. The parent orchestrator is unnamed
- "Parent" appears 23 times in the doc; never assigned. Is it `aiwf` itself growing a `cycle` verb? A skill calling Claude Code's `Agent` tool? An external script?
- Implementation epics 3-5 size at "~1 milestone each"; orchestrator isn't on the list.
- Decisions ride on this: host-coupling, testability, the scope-FSM question from #2.
- **Resolution (D3):** strict-lane principle — aiwf primitives only, orchestration external. Driver-config picks LLM-skill or future deterministic sidekick (`aiwfdo`). Per-epic declarative work-shape declarations (terminology undecided: `cycle` / `loop` / `workflow` / `pipeline` / `orchestration`) live in epic body; reconciliation is the determinism mechanism.
- Status: **resolved (frame); LLM-skill driver implementation rides E-19; aiwfdo deferred**

## 6. Smaller seams
- `waived` requires `--force`; `invalid` doesn't, but is also human-only. Convention is inconsistent — either both need `--force` or the rule is "force only for accept-without-fix".
- ADR-0004's terminality table (`ADR-0004:30-35`) has inline parenthetical corrections — clean up before status moves out of `proposed`.
- "Soft check on missing fix link" — define "nearby" deterministically (same commit? same branch? N events back?).
- Findings-on-findings / recursive cycles — one-line statement either way (works via cross-refs / out of scope).
- **Resolution (D6):** see decisions log.
- Status: **resolved**

---

## Decisions log (filled as we go)

### D1 — Bounded-lifetime always; concurrency is role-based opt-in (closes #1, #4)

Two orthogonal axes; the original design tangled them:

- **Lifetime — always bounded.** One subagent per work unit, fresh context. This is the drift fix from M-066/AC-1 and is the load-bearing primitive. Universal.
- **Concurrency — role-determined.**
  - *Read-only roles* (reviewer / audit / lint): parallel by default. Substrate denies writes.
  - *Additive-write roles* (test-additive / doc-gardener): parallel by default. Substrate restricts writes to a bounded surface; parent assigns a partition.
  - *Builder role*: sequential default. Parallel only with human dispatch + coarse scope hint. Audit emits `scope-expanded` finding (blocks AC closure via `findings-block-met`); not a hard merge block.
  - *Deployer*: sequential by definition.

**Why:** declaring an exact fileset in advance is impossible — fileset is the *output* of the work, not an input. Read-only and additive-write roles avoid the problem by construction (substrate enforcement). Builder-parallel is opt-in territory with deviation detection rather than predictive enforcement.

**Aligns with:** existing `aiwf-extensions` agent vocabulary (builder/reviewer/planner/deployer). No new axis introduced.

**Implications for downstream items:**
- #2 (sub-agent provenance / G22): role-aware now — does each role open its own scope? Same scope as parent? See discussion next.
- #3 (failure recovery): per-role failure modes differ (read-only failures lose findings only; builder failures may leave dirty worktree).
- #5 (parent orchestrator): orchestrator now has a clear job — dispatch by role, enforce per-role concurrency policy from config, collect findings.

### D2 — Role-tagged actor + sub-scope on builder-parallel only; capability/name split for BYO-agent (closes #2)

Provenance and BYO-agent extensibility settled together.

**Provenance shape.**
- All subagents: actor string extends to `aiwf-actor: ai/<host>/<agent-name>` (e.g. `ai/claude/builder`, `ai/claude/security-auditor`). Existing `ai/claude` stays valid (parent-agent default; no role).
- Builder-capability + parallel + dispatched: opens a sub-scope inside the parent's existing authorize scope. Sub-scope reuses the existing scope FSM (`active | paused | ended`). One level of nesting maximum.
- Read-only / additive roles: flat actor with role tag, no sub-scope. Audit & verb-gate differentiation come from the actor tag alone.
- Sovereignty: dispatch is the human-sovereign act; sub-scope opening rides on it. No new sovereignty rule.

**No N-level agency.** Subagents cannot spawn subagents. PoC rule.

**Capability vs name split (BYO-agent answer).**
- **Capabilities** are kernel-pinned, closed-set: `read-only`, `additive-tests`, `additive-docs`, `full-builder`, `sequential-only`. Adding a capability is a kernel change (drift test in `internal/policies/`).
- **Agents** are user-named, open-set, declared in `aiwf.yaml.subagents.agents[]`. Each agent declares which capability it implements. Framework-shipped agents (builder/reviewer/planner/deployer) are pre-registered with their default capabilities. Users add their own (`security-auditor: { capability: read-only }`, `migration-runner: { capability: full-builder }`).
- Capability is what the substrate / verb gate / D1 concurrency policy reads. Name is what shows up in trailers and `aiwf history`.
- New custom agent: zero kernel change. New capability: kernel change.

**(d)'s sub-scope trigger keys on capability, not name.** A user-defined `migration-runner: { capability: full-builder }` gets the same sub-scope treatment as the framework-shipped builder when dispatched in parallel. Capability inheritance does the right thing automatically.

**Plugin-shipped agents.** Third-party plugins (`wf-rituals`, others) can declare agents the consumer should register. `aiwf doctor` recommends registering them — same pattern as the existing `recommended_plugins` check.

**Renames.** Agent renames in `aiwf.yaml` break `aiwf history` queries against the old name (history is name-keyed; pre-rename commits keep their old actor string). PoC discipline: don't rename agents already in use. Mirrors the existing slug/id discipline (id stable; slug drifts).

**Aligns with:** kernel's existing pattern for contracts (validators are user-installed by name, but contract semantics are kernel-pinned). Same shape.

### D3 — Strict-lane principle; declarative work-shape per epic; LLM driver first, sidekick deferred (closes #5)

**Strict-lane principle (new kernel rule, to be added to `CLAUDE.md` / `design-decisions.md`):**

> Aiwf provides primitives — agent/capability registry, audit-trailer schema, verb gate, check rules, reconciliation. Aiwf does **not** invoke LLMs, interpret pipelines, or own multi-step control flow. The orchestrator is external (a host-specific skill, a deterministic sibling binary, or an external script). New hosts get new orchestrators, not kernel changes.

This is the line that prevents aiwf from drifting into an orchestration framework. Every future "should aiwf do X?" question gets the same test: does X record state, validate state, or run flow? Recording and validating belong in aiwf; running flow doesn't.

**Driver config in `aiwf.yaml`:**

```yaml
orchestration:
  driver: llm           # | sidekick    (PoC: llm only; sidekick = future aiwfdo)
```

Consumer choice; switchable. Kernel reconciliation works regardless of driver.

**Per-epic declarative work-shape declaration.** Lives in the epic body as a fenced YAML block under a section heading (terminology open: `## Cycle` / `## Loop` / `## Workflow` / `## Pipeline` / `## Orchestration` — TBD). Layered schema:

| Layer | Owner | Drift policy |
|---|---|---|
| **Core** (kernel-owned, load-bearing for reconciliation) | aiwf | Closed-set; new fields require kernel change + drift test |
| **Hints** (orchestrator-owned, execution metadata) | each orchestrator | Free-form; aiwf ignores; orchestrator validates its own |

Concrete sketch (terminology placeholder; field names finalize at implementation time):

```yaml
- id: 1
  agent: planner          # core: must exist in registry
  per-epic: true          # core: cardinality
  scope-hint: docs/...    # core: drives scope-expanded check
  produces: design-doc    # hint: documentation/dashboard
- id: 2
  agent: builder-flagship
  per-ac: true
  depends-on: [1]
  scope-hint: internal/check/
  cycle-model-override: claude-opus-4-7   # hint: per-step model override
- id: 3
  agent: security-auditor       # BYO agent; no kernel change required
  per-epic: true
  depends-on: [2]
```

**Reconciliation rule.** A new check rule walks the trailer history of cycle commits for the epic, groups by `aiwf-pipeline-step:`, and compares against the declaration:

- `pipeline-step-missing` (per-epic step has no commits)
- `pipeline-step-missing-for-ac` (per-ac step missing for some AC)
- `pipeline-agent-mismatch` (cycle's `aiwf-actor` ≠ declared agent)
- `pipeline-model-mismatch` (cycle's `aiwf-cycle-model` ≠ pinned model)
- `pipeline-ordering-violation` (cycle ran before its declared dependency)
- `pipeline-step-extra` (cycle commit references step id not in declaration)
- `pipeline-references-unknown-agent` (informational; pipeline names an agent not in current registry)

All `pipeline-*` codes are kernel-pinned, finding-kind. Epic closure (`aiwfx-wrap-epic`) blocks on open `pipeline-*` findings.

**Determinism is post-hoc, not real-time.** The kernel doesn't prevent the agent from skipping a step; it mechanically detects deviations after the fact and gates closure on human triage. Same model as the rest of aiwf: pre-push hook, not real-time enforcement.

**BYO-agent validation rule (closes follow-up to D2).** Two-tier:
- **Static (informational):** `aiwf check` warns on `## <work-shape>` references to agents not in current registry. Helpful for typo-catching; non-blocking.
- **Runtime (blocking):** orchestrator refuses to dispatch a step if the named agent isn't in the registry, doesn't have its definition file at the host's expected path, or the model isn't available. Hard failure, actionable error.
- **Closed epics: never re-validated.** History is immutable; trailers reference whatever was registered at execution time. Renamed/removed agents don't retroactively invalidate closed epics. Mirrors the existing slug-rename rule (id stable, slug drifts; closed entities keep their old slug in history).

This keeps BYO agents *config + markdown only* — no kernel recompile, no kernel coupling beyond the closed-set capability enum.

**Cross-host pipelines.** Pipeline declarations are *data* — they can name agents from multiple hosts. Whether the configured orchestrator can execute them is a runtime concern. The skill driver runs in one host; aiwfdo could be cross-host (its capability) but PoC versions start single-host. No special handling in the kernel — same two-tier validation: static accepts; runtime fails clearly.

**Field-semantics drift.** If both orchestrators eventually implement a hint field (e.g. `partition`), they must mean the same thing. Documentation lives with the schema; CI test ensures behavior agreement when the second orchestrator arrives.

**Library factoring discipline.**

- Now: clean `internal/pipeline/` package in aiwf (parser + validator + reconciliation). Disciplined packaging within aiwf, not a public library.
- Now: schema spec written in CUE at `internal/pipeline/pipeline.cue`. Documentation + drift-prevention.
- Now: CI test runs the CUE schema through `cue vet` against fixture inputs and asserts the Go parser agrees. Drift between Go validator and CUE spec → CI failure.
- **CUE is dev/CI only — never a runtime dependency on consumers.** Kernel does not embed `cuelang.org/go/cue` library; does not require `cue` binary to be installed. Aligns with `design-decisions.md:125` ("aiwf never ships a `cue` or `ajv` binary"). The Go parser is what consumers actually run; CUE is the spec we hold ourselves to in CI.
- When (if) `aiwfdo` materializes: lift `internal/pipeline/` to `pkg/pipeline/` (public Go API), aiwfdo depends on it via the same `go.mod`. Same code, two binaries. No parser drift.

**Sequencing recommendation:**

1. **Now (D1+D2 implementation):** kernel registry, capability enum, trailer schema, sub-scope FSM, scope-expanded check rule, verb gate.
2. **Next:** `internal/pipeline/` parser + CUE spec + reconciliation check rule + finding codes + `aiwf check` informational warning. CI agreement test.
3. **Then:** LLM-driver skill (e.g. `aiwfx-run-<work-shape>`) consuming the schema. Validates dogfooding.
4. **Deferred:** `aiwfdo` design doc + sibling-binary epic. Filed only when LLM-driver dogfooding shows determinism friction worth the second tool. Lift `pipeline` package to `pkg/` at that point.

**Open: terminology.** The `## <work-shape>` section heading and related verbs/finding codes (`pipeline-step-missing`, `aiwf-pipeline-step:` trailer key, etc.) need a name. Candidates: `cycle` / `loop` / `workflow` / `pipeline` / `orchestration`. Pinned at implementation time; once pinned, drift-tested like other kernel-pinned vocabularies.

**Aligns with:**
- Kernel principle "framework correctness must not depend on LLM behavior" — reconciliation is mechanical, post-hoc.
- Layered location-of-truth — engine binary stays small; orchestrator lives external.
- Marker-managed framework artifacts — skill driver materializes via `aiwf init`/`aiwf update`.
- "Errors are findings, not parse failures" — pipeline declarations always load; deviations become findings.
- Existing closed-set discipline — capability enum, finding codes, trailer keys.

### D4 — Failure recovery, cycle envelope, provenance/forensics split, harvestable export (closes #3)

**Layered recovery model.**

- **Orchestrator-driven reap (primary).** Orchestrator catches subagent termination (success / timeout / error / substrate-deny tripped) and runs cleanup: closes the sub-scope, writes the cycle commit with full trailer set, persists the bundle.
- **Kernel-side stale-cycle GC (safety net).** A new doctor check + verb (`aiwf reap-stale-cycles`) walks open sub-scopes older than a threshold and closes them as `ended-failure` with a `cycle-orchestrator-died` finding. Catches the case where the orchestrator itself crashed.
- **Quarantine, never wipe.** Failed cycle bundle moves to `.aiwf/quarantine/<epic-id>/<milestone-id>/<ac-id>/<cycle-n>/` preserving the same hierarchy as `.aiwf/cycles/`. `aiwf reap-quarantine --older-than 90d` cleans up.

**Sub-scope FSM extension** (D2's sub-scope FSM gets richer end-states):

- `ended-success` — subagent returned cleanly, JSON valid, ready for merge.
- `ended-failure` — subagent failed (crash, timeout, malformed envelope, substrate-deny tripped). Bundle quarantined.
- `ended-discarded` — human-discarded mid-cycle.

**Per-capability differentiation:**

| Capability | Failure handling |
|---|---|
| `read-only` | No diff to preserve; substrate-deny becomes a `scope-leak` finding + `ended-failure`. Bundle wiped (only logs+envelope kept). |
| `additive-tests` / `additive-docs` | Partial writes within surface preserved in quarantine bundle. |
| `full-builder` | Quarantine bundle preserves diff + logs + prompt; `partial-state-in-worktree` finding pointing human at the bundle. |
| `sequential-only` (deployer) | Sub-scope failure has its own gravity; out of scope for E-19 (deployer doesn't get parallelized anyway). |

**Cycle envelope (A2A-shaped, custom transport).**

We borrow A2A concepts (agent card ≈ registry entry; task envelope ≈ cycle envelope; status enum literals; artifact concept ≈ bundle sidecars) but **don't adopt the protocol**. A2A is HTTP/JSON-RPC for running services; our subagents are spawned bounded-lifetime CLIs (`claude -p`, `codex --headless`) — wrong transport. If a future consumer wants to expose aiwf-orchestrated cycles via A2A endpoints, the wrapping layer is thin because the data shape is already aligned.

Envelope schema layered like the pipeline schema (kernel-owned core + orchestrator-owned hints), CUE spec at `internal/cycle/envelope.cue`, Go parser, CI agreement test. **CUE never ships at runtime.** Forgiving Go parser: unparseable envelopes fall back to `cycle-envelope-malformed` finding + raw blob preserved in bundle.

Embedded markdown via `*_md` fields (e.g. `summary_md`, `body_md` in findings). Extracted to sidecar files when persisting. Two wins: forensic completeness in one record; graceful parser degradation (markdown still readable if JSON fails to parse).

**No JSONL in-tree** (kernel rejects events.jsonl). JSONL is permitted as an *export output format* (D4 export verb below) — that's delivery, not storage.

**Forensic bundle layout.** Per cycle, mirroring the conceptual hierarchy:

```
.aiwf/cycles/
  <epic-id>/<milestone-id>/<ac-id>/cycle-<n>/
    envelope.json           # primary record (machine-readable, A2A-shaped)
    summary.md              # extracted from envelope.summary_md
    diff.patch              # full diff
    stdout.log / stderr.log # raw subagent output
    prompt.md               # the prompt sent to the subagent
    agent.yaml              # resolved registry entry at dispatch time
    findings/F-NNN-body.md  # one per finding's body
```

Bundle directory is gitignored by default (`aiwf init` adds the entry; `aiwf doctor` warns if missing). Quarantine mirrors the same hierarchy.

**Permanent provenance vs ephemeral forensics — two retention tiers, two substrates.**

| Tier | Where | Lifetime |
|---|---|---|
| **Permanent provenance** | Git trailers on cycle commits | Forever |
| **Ephemeral forensics** | `.aiwf/cycles/...` bundle, gitignored | Until reaped |

Trailers carry the long-term-interesting metadata; bundles carry the rich raw record. The split lets each have the right discipline. **No new event log file**; the kernel's events.jsonl prohibition holds.

**Permanent trailer surface (kernel-pinned, drift-tested):**

```
aiwf-cycle-id              aiwf-cycle-pipeline-step
aiwf-cycle-status          aiwf-cycle-scope-hint
aiwf-cycle-agent           aiwf-cycle-duration-ms
aiwf-cycle-model           aiwf-cycle-files-touched
aiwf-cycle-host            aiwf-cycle-lines-added
aiwf-cycle-prompt-hash     aiwf-cycle-lines-removed
aiwf-cycle-findings-count  aiwf-cycle-tests-added
aiwf-cycle-findings        (comma-separated F-NNN ids)
```

Each trailer ~50 bytes; full set ~700 bytes per cycle commit. Acceptable. Stat-field semantics specified precisely in `internal/cycle/` docs (e.g. "files-touched" = unique modified paths in the cycle's diff against parent commit).

**Prompt content strategy — hash-only now, CAS later.** Trailer carries `aiwf-cycle-prompt-hash: sha256:...`; full prompt lives in the forensic bundle. When bundle is reaped, prompt is lost from local. **Future upgrade path:** content-addressable store at `.aiwf/prompts/<hash>.md` (deduplicated, tracked by git, small because most prompts repeat). Hash trailer is the future-compat hook; CAS can ship later without schema breakage. **Don't build CAS speculatively** — the hash field is the hook.

**Provenance check rule.** `cycle-trailer-incomplete` flags cycle commits missing required provenance trailers. Catches orchestrator drift (skill forgot to stamp duration; sidekick wrote partial set on crash).

**Harvestable export surface (public contract).**

Aiwf produces structured data; aiwf doesn't analyze it. Anyone wanting to use the data — for training, audit warehouse, dashboards, recovery archive, cross-project meta-analysis, regulated compliance — pulls it via a stable export verb and ships wherever. Aiwf isn't involved in destinations.

**`aiwf export-cycles`** verb:

```
aiwf export-cycles --since 2026-01-01 --format jsonl > cycles.jsonl
aiwf export-cycles --epic E-19 --include-bundles --format tar > e19.tar
aiwf export-cycles --redact-prompts --since 2025-01-01 > sanitized.jsonl
```

Each record = trailer-derived metadata merged with envelope content; bundles optionally inlined/tarred. JSONL for warehouse-piping; tar for full-bundle archives.

**Schema stability is a public contract.** Trailer keys, envelope schema, and export record format are kernel-pinned, drift-tested, and treated as a versioned public API. Additive changes are routine; renames/removals are breaking-change territory with deprecation discipline. CUE specs from D3 and D4 are the canonical documentation external harvesters target.

**Retention adapts to the harvest setup.**
- No external harvester: local bundle is only forensic copy → keep longer (default 90d/365d success/failure).
- Daily harvest to permanent store: bundle is redundant → reap aggressively (default 7d/30d).

Configured via `aiwf.yaml.cycles.retention`; `aiwf doctor` warns if `harvest_to` is set but no harvest has happened recently.

**Speculative use cases enabled (not designed for):**
- Training data: bundles contain (prompt, response, outcome quality) tuples in the right shape for fine-tuning.
- Recovery: harvest archive contains enough to reconstruct planning state if local repo dies (parallel record, not a git replacement).
- Cross-project meta-analysis: shared harvest warehouse → org-level pattern learning.
- Compliance / regulated audit: harvest archive is the audit trail of record.

None of these justify building anything beyond the export verb. They justify the schema-stability discipline that makes the verb's output consumable.

**`aiwf cycle-stats` query verb (deferred).** Once trailers carry richness, an in-repo query verb computes "average duration of builder cycles using Opus", "findings-per-agent grouped by scope-hint", etc. Walks `git log`, parses trailers, aggregates. Implementation easy; deferred until dogfooding shows demand. Trailer schema enables it whenever it ships.

**Aligns with:**
- Strict-lane principle (D3) — aiwf produces data; analysis lives outside.
- "Errors are findings, not parse failures" — malformed envelopes become findings, never crashes.
- Existing closed-set discipline — trailer keys, finding codes, status enum.
- Kernel rejection of events.jsonl — preserved (in-tree storage uses git+bundles, not append logs).
- Layered location-of-truth — kernel produces; consumers (harvesters, dashboards, training pipelines) consume externally.

### D5 — Forensics scope boundary (clarifies D4)

**The kernel's forensics promise has an explicit boundary.**

Aiwf **guarantees**:
- **Permanent metadata** via git trailers on cycle commits — what happened, when, by whom, with what model, producing which findings, with what prompt-hash. Always in git, always in CI, queryable forever.
- **An export tool** (`aiwf export-cycles`) that emits bundle + trailer-derived data in a stable, drift-tested schema for downstream consumption.

Aiwf does **not guarantee**:
- **Content delivery to durable storage.** Bundle content (raw logs, full prompt text, narrative summaries) lives gitignored. Whether it reaches a permanent destination — backup, warehouse, training-data archive, compliance store — is the consumer's responsibility.
- **Bundle survival past local reap policy.** Default retention reaps bundles after a configurable period; reaped bundles are gone from the local machine.

**Why this is the right boundary for the PoC:**
- Aiwf's correctness doesn't depend on bundle persistence. Reconciliation, AC closure, FSM transitions all work from trailers alone.
- "Chain of custody is intact" is provable from trailers (agent + model + prompt-hash + outcome). "Reconstruct the exact LLM exchange" is the richer claim that depends on bundle survival — and is a consumer concern, not a kernel correctness property.
- Consumer-discipline is appropriate for *consumer-facing affordances*; the kernel principle ("framework correctness must not depend on LLM behavior") doesn't extend to "content delivery must not depend on consumer discipline" — that would over-stretch the rule.

**Documentation requirement.** The boundary above ships in `CLAUDE.md` / `design-decisions.md` so consumers don't assume aiwf delivers more than it does.

**Aspirational, not planned: pre-push harvest hook.** A hook similar in shape to the existing pre-push `aiwf check` hook could close the discipline gap structurally for consumers needing compliance-grade forensics: `aiwf.yaml.cycles.harvest_to: <destination>`; aiwf-managed marker hook calls `aiwf export-cycles --since last-push --upload-to <destination>` at push time. The pieces it would ride on (export verb, marker-hook system, config schema, redaction flags) are already in the design or already exist; the wiring would be small.

**This is an idea, not a plan.** No milestone, no epic, no roadmap commitment. Captured here so future readers know the upgrade path is shaped if a consumer ever surfaces concrete compliance needs. Until that consumer exists, the work is not on the schedule and should not appear in any planning artifact.

**Aligns with:**
- Strict-lane principle (D3) — aiwf doesn't analyze, store, or transport.
- Honest scoping — kernel promises only what it guarantees mechanically.
- YAGNI — don't build the hook; shape the path so building it is easy if needed.

### D6 — Smaller seams cleanup (closes #6)

**6a — `--force` on both `waived` and `invalid` (single sovereignty rule).**

Closing a finding to *any* terminal status without code change requires `--force` and `--reason`, regardless of which terminal status. Aligns with M-017's existing pattern for sovereign terminal acts. Single rule, no exceptions: "every finding-terminal transition by a human-actor without a code fix requires `--force --reason`."

The semantic-purity argument for distinguishing `waived` (sovereign override of a real concern) from `invalid` (declaration that the concern wasn't real) was considered and rejected for the PoC. Reading 1 wins on simplicity: one rule to remember, slight extra friction on `invalid` is acceptable because both transitions are equally consequential — both flip a finding to terminal without any code change.

**Action:** before `ADR-0003` moves out of `proposed`, update §"Status FSM":
- `waived` — unchanged (already requires `--force --reason`).
- `invalid` — change "Requires `--reason`" to "Requires `--force --reason`."

**6b — Clean up ADR-0004's terminality table (mechanical doc fix).**

`ADR-0004:30-35` carries inline parenthetical corrections from drafting. Replace with the clean table, verified against `internal/entity/transition.go:12-49`:

```
- epic:     done, cancelled
- milestone: done, cancelled
- ADR:      superseded, rejected
- gap:      addressed, wontfix
- decision: superseded, rejected
- contract: retired, rejected
- finding (proposed): resolved, waived, invalid
```

Source-of-truth note already exists at `ADR-0004:27` ("FSM definitions per kind in `internal/entity/transition.go` are the source of truth"); that's enough context for any reader surprised that `accepted` (ADR/decision) or `deprecated` (contract) aren't terminal. No additional prose needed.

**6c — Soft-check semantics: any-ancestor-to-merge-base.**

When a finding promotes to `resolved`, fire warning-only finding `finding-resolved-without-fix-link` if **no prior ancestor commit reachable from the resolve commit (excluding the resolve commit itself, walking back until merge-base with main) carries `aiwf-entity: <F-NNN>` in its trailers**.

- Scope: only `resolved` transitions fire the check. `waived` and `invalid` skip it (no fix expected/needed by definition).
- Excludes the resolve commit itself (which always carries the trailer because the verb mutates the finding).
- Walks branch ancestry; bounded by merge-base with main (cost is small for routine branches).
- Warning-only: surfaces the discipline gap without blocking. Promotion to blocking is a future judgment call after PoC dogfooding shows the rule's reliability.

Stricter alternatives rejected: same-commit (α) conflicts with fix-then-promote-later workflows; fixed N-window (β) is brittle to team cadence.

**6d — Findings on findings; recursive cycles.**

*Findings on findings:* just works via the existing data model. Findings produced while resolving another finding link via `linked_entities` (e.g., F-008's frontmatter contains `linked_entities: [F-007]`). AC closure check walks the linked-findings graph; new findings on the same AC block closure regardless of which cycle produced them. `aiwf history F-007` shows the F-008 cross-reference because both commits carry standard trailers.

*Recursive subagent spawning:* **forbidden** by D2's PoC rule (subagents cannot spawn subagents; N=2 max). If a cycle's findings need a follow-up cycle to fix, the orchestrator dispatches it on the next pass; subagents in flight cannot dispatch others. The recursion bound lives at the orchestrator level, where the human's policy controls it.

**Action:** add a one-paragraph note to `ADR-0003` (probably in §"Consequences" or a new §"Recursion bounds" section):

> Findings produced while resolving another finding are recorded normally via `linked_entities`. Recursion is bounded by the AC-closure block — open findings on this AC keep it open regardless of which cycle produced them — and by the PoC's N=2 subagent rule (D2): subagents cannot spawn subagents. If a cycle's findings warrant a follow-up cycle, the orchestrator dispatches it on the next pass.

Defensive cycle-detection in the `linked_entities` graph (e.g., F-007 → F-008 → F-007) was considered and rejected as YAGNI — link cycles aren't a real failure mode (semantically odd, but break nothing) and adding a check rule for a hypothetical case violates "ship what's used."

**Aligns with:**
- M-017's existing sovereignty pattern (force-flag for consequential terminal acts).
- Kernel "single rule, no exceptions" preference where the cost is only ergonomic.
- D2's PoC subagent-spawn bound.
- "Errors are findings, not parse failures" — soft check warns rather than blocks; humans triage.
- Existing `linked_entities` data model handles recursion without new mechanism.


