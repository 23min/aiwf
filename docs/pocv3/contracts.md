# Contracts: post-PoC plan

**Status:** proposal · **Audience:** post-PoC framework work · **Predecessor:** the PoC `contract` entity (kept, extended, never replaced).

This document plans the move from "contracts as a recordable entity" to "contracts as bounded surfaces the engine helps repos enforce." It does not propose engine code that knows about CUE, OpenAPI, JSON Schema, or any other validator. It proposes the *slots* and *hooks* that let consumer repos run their chosen validator with the engine's coordination.

The reframe is grounded in two observations:

1. The framework's predecessor proved out a working pattern in real consumer use: rich contract discipline (validator schemas, fixtures, a hand-maintained matrix index, plan-time milestone sections, wrap-time row checks, drift guards) implemented entirely as **repo-local content using framework primitives**. The framework knew nothing about specific validators; it provided the rules/skills mechanism, structured milestone specs, and wrap hooks. Repos filled those slots.
2. That same predecessor work has an open framework issue calling for plan-time template support and wrap-time doc-lint enforcement of contract-matrix discipline. That's the gap the PoC currently leaves and that the post-PoC plan should close.

The plan below closes that gap, scoped so the engine never grows a validator dependency.

---

## 1. The model: entity vs. surface

Two senses of "contract" exist; they must stay verbally distinct.

| Term | What it is | Where it lives |
|---|---|---|
| **Contract entity** | A tracking record — id, status, lifecycle events, pointers, optional verifier metadata | aiwf engine (`events.jsonl` + projection) |
| **Contract surface** | The schema, fixtures, validator binary, drift-guard prose | The consumer repo, in whatever shape they choose |

The entity *describes* a surface; it does not *contain* one. This mirrors the existing `adr` entity: an ADR record points at an ADR markdown file in `docs/decisions/`; the entity is the engine's slot, the document is the repo's content.

A third term is reserved for engine internals: **boundary contracts** are the YAML files in `framework/modules/<kind>/contracts/*.yaml` that declare per-kind validation rules. They are unrelated to user-facing contract entities and must always be called out by their full name to avoid the collision.

---

## 2. What aiwf owns vs. what the repo owns

### aiwf owns

- The contract entity schema (closed-set fields, validated by its boundary contract).
- Lifecycle events: `contract.proposed`, `contract.ratified`, `contract.evolved`, `contract.deprecated`, `contract.retired`, `contract.verified`, `contract.drift_detected`.
- The contract registry projection — a derived index. Auto-rendered from events on demand.
- One new verb: `aiwf contract verify [<id>]`.
- Filter flags on the generic `list` verb (`--drifted`, `--verified-status`, `--linked-adr`, `--untouched-since`).
- Structured `## Contract matrix changes` milestone-spec section (closed-set keys: `added`, `updated`, `retired`).
- A wrap-hook surface for repo-supplied verifiers — documented input event, documented output finding shape, exit-code semantics. The engine invokes; it does not validate.
- Live-source path drift detection (file-level in this plan; symbol-level once the read-side reference-resolution lens lands).

### The repo owns

- The schema (CUE, JSON Schema, OpenAPI, prose, anything).
- The fixtures.
- The validator binary and how it's installed.
- The verifier hook script (`scripts/verify-contracts.sh` or equivalent).
- The drift-guard rationale prose.
- The choice of *what* counts as a contract surface in their architecture.

This factoring matches the predecessor pattern: the framework provides primitives; the repo provides discipline. The only thing this plan adds is **mechanical enforcement of the existing primitives** that the predecessor lacked.

---

## 3. ID convention

Existing kinds use established prefixes. Don't change them:

| Kind | Prefix | Example |
|---|---|---|
| epic | `E-` | `E-21` |
| milestone | `M-` | `M-PACK-A-01` |
| adr | `ADR-` | `ADR-OPSPEC-01` |
| decision | `D-` | `D-2026-04-20-025` |
| gap | `G-` | `G-0007` |
| **contract** | `C-` | `C-0042` |

`ADR-` is intentionally three letters because the industry convention is universally recognized; the asymmetry buys recognition that strict one-letter symmetry would lose. `C-` for contract follows the one-letter pattern for new kinds.

Lock this in `architecture.md` so it doesn't drift.

---

## 4. Capabilities (not tiers)

Earlier drafts of this proposal used a four-tier model (0 through 3). That introduced a vocabulary the user has to learn before they can do anything. Replace it with **capabilities**: independent, opt-in, no required order beyond *track* being first.

| Capability | What you add | What it gives | Required for adoption? |
|---|---|---|---|
| **Track** | the entity record | A registry of named contracts, linked to ADRs | Yes — the floor |
| **Drift detection** | a `live_source` path on the entity | Engine watches the path across commits; emits a finding when it disappears or moves | No |
| **Matrix discipline** | a `## Contract matrix changes` section in milestone specs | Plan-time validation that the section is well-formed; wrap-time check that the registry reflects the declared changes | No |
| **Mechanical verification** | a `verifier` block on the entity + a registered hook | `aiwf contract verify` runs the hook; results recorded as events | No |

A repo can adopt **any subset**. The capabilities don't require each other, and skipping one doesn't degrade the others. There is no `tier` field on the entity — the engine derives capability state from which fields are populated.

The recipes below are the unit of choice the user presents to an LLM: "follow recipe X for contract C-0042." The capability model is explanatory framing for the docs.

---

## 5. Verbs

Generic verbs (already in the engine's design — contracts come along for free):

```
aiwf list contracts [--filter ...]   # registry view
aiwf show C-0042                      # full record + lifecycle
aiwf history C-0042                   # events filtered to this entity
aiwf render contracts                 # auto-derived contract matrix → CONTRACTS.md
```

One new contract-specific verb:

```
aiwf contract verify [<C-id>]         # dispatches to the registered hook(s);
                                      # without an id, runs every contract that has one
```

That's it. Status, filtering, and matrix rendering all flow through generic verbs. Filter flags do the slicing:

```
aiwf list contracts --drifted               # live_source missing or stale
aiwf list contracts --verified-status fail  # last verify failed
aiwf list contracts --linked-adr ADR-0042   # everything that ADR created
aiwf list contracts --untouched-since 30d   # candidates for review
```

Don't add `aiwf contract status`, `aiwf contract list`, or `aiwf contract matrix`. They duplicate generic verbs and break symmetry across kinds.

---

## 6. Projection columns

`aiwf list contracts` shows these by default:

| Column | Source |
|---|---|
| `id` | entity |
| `status` | latest lifecycle event |
| `live_source` | entity field |
| `live_source_exists` | computed at projection time (file check) |
| `last_verified` | latest `contract.verified` event timestamp, or `—` |
| `last_verify_result` | `pass` / `fail` / `stale` / `unverified` |
| `linked_adrs` | count |
| `drift` | `true` if `live_source_exists=false` or last_verified > N days ago |

The matrix render (`aiwf render contracts`) uses these same columns in markdown table form. This is structured deterministic output — not prose generation, no principle violation.

---

## 7. Recipes

Each recipe is self-contained. A user (or an LLM) can pick one and follow it without reading the others. Each recipe ends with a copy-pasteable LLM prompt.

### Recipe A — Track a contract

**When to use:** You want a registry entry for a bounded surface in your system. No automated enforcement yet — just an authoritative record.

**Steps:**
1. Author an ADR explaining what the contract is and why (`docs/decisions/NNNN-<slug>.md`).
2. Create the contract entity via the engine's mutation verb. Required fields: `id`, `status: proposed`, `linked_adrs`, short `description` prose.
3. Ratify it (`status: ratified`) when the ADR is approved.

**LLM prompt:**
```
I want to register a new contract for <surface description>.
The ADR is at docs/decisions/NNNN-<slug>.md.
Please:
1. Allocate the next C-NNNN id via the engine.
2. Create the contract entity in status=proposed, linked to that ADR.
3. Show me `aiwf show C-NNNN` so I can verify.
Do not wire validation yet.
```

### Recipe B — Add drift detection

**When to use:** The contract has a clear authoritative file in the repo (a schema file, a `.proto`, a markdown spec) and you want the engine to notice when that file moves or disappears.

**Steps:**
1. Identify the live-source path — the single file that is the authoritative shape of this contract.
2. Update the contract entity to set `live_source: <path>`.
3. Confirm: `aiwf list contracts --drifted` returns nothing if the path exists.

**LLM prompt:**
```
For contract C-NNNN, the authoritative source is <path>.
Please:
1. Set live_source on the contract entity to that path.
2. Run `aiwf list contracts --drifted` and show me the output.
3. If the contract appears as drifted, explain why before fixing.
```

### Recipe C — Add matrix discipline to milestones

**When to use:** Contracts are being added/modified/retired by ongoing work and you want the registry to stay current automatically.

**Steps:**
1. In every milestone spec that touches a contract, add:
   ```markdown
   ## Contract matrix changes
   - added: [C-0042, C-0043]
   - updated: []
   - retired: []
   ```
   (or `none` if the milestone touches no contracts).
2. The engine validates the section structure at plan time.
3. At wrap time, the engine verifies the contract registry actually reflects the declared changes.
4. Drift in either direction (declared but not done; done but not declared) blocks wrap.

**LLM prompt:**
```
Milestone <M-id> currently has no `## Contract matrix changes` section.
This milestone touches contract(s) <list>.
Please:
1. Add the section with the correct added/updated/retired bullets.
2. Verify the milestone spec validates with `aiwf validate <M-id>`.
3. If you propose changes I haven't named, explain why before adding them.
```

### Recipe D — Wire a verifier hook (CUE)

**When to use:** The contract has a CUE schema and fixtures, and you want `cue vet` to run automatically as part of contract verification.

**Prerequisite:** CUE is installed in the repo's dev environment with a pinned version. Out of scope for aiwf — pin a version in a shared tool-versions file at the repo root, install it from the same file in the devcontainer, and document the choice in an ADR.

**Steps:**
1. Author the schema at `docs/schemas/<topic>/schema.cue` and fixtures at `docs/schemas/<topic>/fixtures/v<N>/*.yaml`.
2. Author a hook script at `scripts/verify-contract-cue.sh` that reads `$AIWF_CONTRACT_ID` and `$AIWF_LIVE_SOURCE` from env and runs `cue vet` against the registered fixture set. The hook outputs findings as JSON on stdout in the engine's documented finding shape, exit code 0 on pass, non-zero on fail.
3. Register the hook in `.aiwf/hooks.yaml`:
   ```yaml
   contract_verifiers:
     cue:
       command: scripts/verify-contract-cue.sh
   ```
4. Update the contract entity to set `verifier.kind: cue`, `verifier.schema: docs/schemas/<topic>/schema.cue`, `verifier.fixtures: docs/schemas/<topic>/fixtures/`.
5. Run `aiwf contract verify C-NNNN`. The engine invokes the hook, records a `contract.verified` event with the result.

**LLM prompt:**
```
For contract C-NNNN, please wire CUE-based mechanical verification.
The schema lives at <path>. Fixtures live at <path>.

Follow Recipe D in docs/pocv3/contracts.md:
1. Confirm the schema and fixtures exist; if not, stop and tell me.
2. Author scripts/verify-contract-cue.sh per the recipe (idempotent — don't overwrite if it exists).
3. Register the hook in .aiwf/hooks.yaml under contract_verifiers.cue.
4. Update C-NNNN's verifier block.
5. Run `aiwf contract verify C-NNNN` and show me the result.
6. If verification fails, explain the cue vet output before proposing fixes.
```

### Recipe E — Wire a verifier hook (JSON Schema / Ajv)

**When to use:** Same as Recipe D, but the contract is expressed as JSON Schema and you want `ajv` to validate fixtures against it.

**Steps:** Same shape as Recipe D, swapping CUE for Ajv:
1. Schema at `schemas/<topic>.schema.json`, fixtures at `schemas/<topic>/fixtures/*.json`.
2. Hook script `scripts/verify-contract-jsonschema.sh` runs `npx ajv validate` per fixture.
3. Register in `.aiwf/hooks.yaml` under `contract_verifiers.jsonschema`.
4. Set `verifier.kind: jsonschema` on the entity.
5. `aiwf contract verify C-NNNN`.

**LLM prompt:** Adapt Recipe D's prompt, substituting JSON Schema / Ajv for CUE.

### Recipe F — Prose drift guard (no automated validator)

**When to use:** The contract is real, the surface exists, but no validator is appropriate (or the team doesn't want the tooling overhead). Drift detection on the live-source path + a written drift guard is enough.

**Steps:**
1. Set `live_source` per Recipe B.
2. Set `drift_guard` on the entity to a one-paragraph rule about how the contract must not be undermined (e.g. "the only writer to this file is `eventlog.Append`; any new caller is a violation requiring a new ADR").
3. The reviewer-agent skill reads `drift_guard` during review and flags PRs that appear to violate it.

**LLM prompt:**
```
Contract C-NNNN has no automated validator and won't get one.
The drift guard is: <one-paragraph rule>.
Please:
1. Set drift_guard on the entity to that text.
2. Confirm with `aiwf show C-NNNN`.
```

---

## 8. The `contract` skill

Recipes are content; the skill is the orchestrator. Ship a single skill that bundles the above and lets an LLM pick the right one from a high-level user request.

**Skill location:** `framework/modules/contracts/skills/contract.md`

**Skill shape (sketch):**

```markdown
---
name: contract
trigger: /contract
description: |
  Guide a user through creating, evolving, or verifying contract entities.
  Picks the right recipe based on what the user wants and the current state
  of the contract.
---

## When invoked with no arguments
Run `aiwf list contracts` and ask the user what they want to do:
- Track a new contract (Recipe A)
- Add drift detection to an existing one (Recipe B)
- Add matrix discipline (Recipe C)
- Wire mechanical verification (Recipe D / E)
- Add a prose drift guard (Recipe F)
- Verify an existing contract (`aiwf contract verify`)

## When invoked with a contract id
Run `aiwf show <id>` first. Use the projection state to suggest the next
useful recipe — e.g. if `live_source` is empty, propose Recipe B; if
`verifier` is empty but a schema file exists at a conventional path,
propose Recipe D.

## Always
- Read the recipe end-to-end before executing it.
- If a step requires repo-specific knowledge you don't have (which
  validator, where schemas live), ask the user before guessing.
- Never edit the entity directly; always go through `aiwf` mutation
  verbs.
- Show `aiwf show <id>` after every change so the user sees the new state.
```

This is the LLM-affordance the user asked for: one skill, deterministic recipes, copy-pasteable prompts inside.

---

## 9. Implementation increments

Each increment is independently shippable and independently useful. They have a natural order but no hard dependencies between increments beyond what's noted.

| Increment | Scope | Depends on |
|---|---|---|
| **I1 — Capabilities Track + Drift** | Contract entity gets `live_source` field. List columns (`live_source_exists`, `drift`). `--drifted` filter. Recipes A, B, F + `contract` skill (initial form). C- prefix locked in `architecture.md`. | PoC `contract` entity exists |
| **I2 — Matrix discipline** | Structured `## Contract matrix changes` milestone-spec section. Plan-time validator. Wrap-time registry-presence check. `aiwf render contracts` for matrix output. Recipe C. | I1 |
| **I3 — Verifier hook surface** | `.aiwf/hooks.yaml` with `contract_verifiers` section. `aiwf contract verify [<id>]` verb. `verifier` block on entity. `contract.verified` events. Documented hook input/output contract. Recipes D and E. | I1 |
| **I4 — Symbol-level live source** | Once the read-side reference-resolution lens lands, `live_source` accepts symbol UIDs (e.g. `tools/internal/eventlog.go#Append`). Drift detection extends to symbol renames. | I1 + read-side reference-resolution |
| **I5 — Schema-evolution check** (optional) | Engine convenience verb that walks fixture libraries and runs the registered verifier per historical fixture against HEAD schemas. Pure orchestration; no validator embedded. | I3 |

I1 is the smallest useful step and unblocks predecessor-style discipline immediately. I3 is the increment that earns the "mechanical enforcement" claim.

---

## 10. Migration from PoC

There is no data migration. Strategy:

- The PoC `contract` entity stays exactly as it is.
- New optional fields (`live_source`, `verifier`, `drift_guard`, `schema_version`) are added to the entity's boundary contract over the increments.
- Existing PoC entities continue to work; absent fields just mean the corresponding capability isn't enabled for that contract.
- New events (`contract.verified`, `contract.drift_detected`) are additive; replay-from-events on a PoC-era log produces the same projection plus empty optional columns.
- No PoC consumer needs to do anything to keep working. Adopting a new capability is opt-in and per-contract.

The cherry-pick path from `main` to the post-PoC branch: this document, plus any PoC-era changes to the contract entity that don't conflict with the post-PoC increments. The post-PoC branch implements the increments; the PoC stays focused on its current scope.

---

## 11. Open questions

These don't block I1 but should be settled before I2 / I3.

| Question | Notes |
|---|---|
| Is the field named `verifier` or `validator`? | "Verifier" matches the verb (`aiwf contract verify`); "validator" matches industry vocabulary (CUE *validates*). Pick one and commit. Lean: `verifier`. |
| Hook config location: `.aiwf/hooks.yaml` vs. inside an engine config file? | Lean: `.aiwf/hooks.yaml` — repo-local, version-controllable, doesn't entangle with engine internals. |
| Does `aiwf contract verify` cover drift detection or is it a separate verb? | Lean: cover both. One verb, one envelope, "verify the contract is upheld in all dimensions we track." |
| What's the finding shape for hook output? | Defer to I3 design. Should match the existing engine-finding shape so contracts integrate with `aiwf list gaps`. |
| Should `live_source` accept multiple paths? | Lean: yes — list of paths. Some surfaces are multi-file. |
| Does the engine ship reference hook scripts (`verify-contract-cue.sh`, `verify-contract-jsonschema.sh`) or just document them? | Lean: ship them as templates that consumers copy into their repo, in the recipes folder. Engine doesn't depend on them. |

---

## 12. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Scope creep — engine starts knowing about validators | Med | High (breaks the principle factoring) | Hard rule in this doc + PR review: validator code never lands in the engine; it lives in repo hook scripts and recipe templates only |
| Recipe documentation rotting as engine evolves | High | Med | Each engine change touching contract surfaces must include a recipe-doc audit in its pre-PR checklist |
| LLMs picking the wrong recipe from vague user requests | Med | Low | The `contract` skill ALWAYS shows current state via `aiwf show` first and asks before guessing |
| Repos confused by entity-vs-surface distinction | Med | Med | Architecture doc + skill prose use the words consistently; "boundary contract" is always called by full name |
| Symbol-level live_source design slips because read-side slips | Low | Low | Path-level drift is fine for I1–I3; I4 is genuinely optional |

---

## 13. References

- The PoC `contract` entity in this repo — the foundation this plan extends.
- `framework/modules/<kind>/contracts/*.yaml` — the engine's *boundary contracts*, not user-facing contracts; collision-prone, always call by full name.
- The research arc under `docs/research/` — particularly `KERNEL.md` (the eight needs the framework serves), `02-do-we-need-this.md` (the audiences distinction), and `07-state-not-workflow.md` (state-as-canonical, render-as-courtesy). The contracts plan honors these: contract entities are state, the registry is a render, validators are repo-local content the engine composes with.
- This conversation — design discussion preceding this doc, summarized faithfully above.
