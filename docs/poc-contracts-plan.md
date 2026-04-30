# Contracts plan

**Status:** proposal · **Audience:** PoC contracts work (continuation of [`poc-plan.md`](poc-plan.md) sessions 1–5).

This document plans contract verification in aiwf from first principles.

A **contract** is a bounded surface in the consumer repo whose shape is enforced mechanically — typically a CUE schema, a JSON Schema, a `.proto`, an OpenAPI document, or any other file format with a deterministic validator. The framework's job around contracts is exactly three things:

1. Track that a contract exists (a registry record tying it to ADRs and a status).
2. Run the user's validator against fixtures on demand and pre-push, surfacing pass/fail as findings.
3. Catch silent breakage when a schema changes by re-running historical fixtures against the head schema.

The engine owns enforcement orchestration. The user owns validators: the binary (`cue`, `ajv`, `protoc`, …) is installed by the user, and its invocation shape is *declared* in `aiwf.yaml` — not compiled into the engine and not shipped as executable scripts in the consumer repo. Recipes for common languages ship as embedded markdown content, opt-in via `aiwf contract recipe install`, never selected as defaults.

This factoring is the load-bearing call: it lets aiwf stay small and language-agnostic while making contract verification a first-class part of the pre-push chokepoint.

---

## 1. The model

Two things that share the word "contract"; keep them separate:

| Term | What it is | Where it lives |
|---|---|---|
| **Contract entity** | A registry record — `id`, `status`, `linked_adrs`, `description`. Pure planning state. | `work/contracts/C-NNN-<slug>/contract.md` |
| **Contract binding** | The schema path, fixtures path, and validator name for a contract surface. Operational state. | `aiwf.yaml.contracts.entries[]` |

The link between them is the `id` string. Same id in both, two responsibilities, two locations.

A third internal term, **kind schema**, names the engine-internal frontmatter rules under `tools/internal/entity/`. It is unrelated to user-facing contracts and never appears in user docs or skill prose; the separate name keeps the vocabulary collision-free.

---

## 2. What aiwf owns vs. what the user owns

### aiwf owns

- The contract entity schema, its closed status set (`proposed → accepted → deprecated → retired`, plus terminal-cancel `rejected`), and one transition function.
- A complete verb surface for every mutation of `aiwf.yaml.contracts.*` — no hand-editing required by user or LLM. See §6.
- The `contracts:` block in `aiwf.yaml` (parser + structural validator + programmatic round-trip writer).
- The fixture-walking pass shape: `<fixtures>/<version>/{valid,invalid}/*` with the verify rule (valid must pass, invalid must fail) and the evolve rule (every historical valid fixture must pass HEAD).
- The substitution runner: takes a validator's `command` + `args` template, substitutes `{{schema}}`, `{{fixture}}`, `{{contract_id}}`, `{{version}}`, executes, captures stdout/stderr, interprets exit code.
- Pre-push integration: when `aiwf check` runs and `aiwf.yaml.contracts.entries[]` is non-empty, verify+evolve runs as part of the same envelope. Terminal-status contracts (`rejected`, `retired`) are skipped.
- Recipe materialization: embedded markdown content for known languages, installable via verbs; user-supplied custom validators installable via the same verbs.
- The `aiwf-contract` skill, embedded and materialized like other skills.
- Lifecycle commit trailers and `aiwf history C-NNN` integration via the existing trailer machinery, including binding mutations.

### The user owns

- The schema, in any language they choose.
- The fixtures, valid + invalid.
- The validator binary (`cue`, `ajv`, `protoc`, …) — installed via the user's tool-versions / devcontainer / package manager.
- The choice of which validator to run for which contract.
- The decision of whether to ship a contract bundle or merely register the entity (registry-only is supported).

The engine never ships a validator binary, never branches on language name, never embeds a hardcoded validator implementation.

---

## 3. ID convention

Existing kinds keep their prefixes (locked in `tools/internal/entity/entity.go`); add the contract row formally:

| Kind | Prefix | Example |
|---|---|---|
| epic | `E-` | `E-21` |
| milestone | `M-` | `M-PACK-A-01` |
| adr | `ADR-` | `ADR-OPSPEC-01` |
| decision | `D-` | `D-2026-04-20-025` |
| gap | `G-` | `G-0007` |
| **contract** | `C-` | `C-001` |

Lock in the project's design-decisions doc.

---

## 4. The contract entity — narrow it

Drop today's `format` + `artifact` fields and the `contract-artifact-exists` validator (the schema-copying-into-contract-dir model). Replace with:

| Field | Required | Notes |
|---|---|---|
| `id` | yes | `C-NNN`. Existing. |
| `title` | yes | Existing. |
| `status` | yes | One of `proposed / accepted / deprecated / retired / rejected`. |
| `linked_adrs` | no | List of ADR ids. The "why" of the contract. |
| `description` | no | Free prose. Body content; not validated. |

**Allowed transitions:**

```
proposed → accepted → deprecated → retired
        ↘ rejected ↙
```

- `proposed → accepted` — the proposing ADR landed.
- `accepted → deprecated` — phase-out begins.
- `deprecated → retired` — phase-out complete; archive.
- `proposed → rejected` (via `aiwf cancel`) — proposing ADR was rejected.
- `accepted → rejected` (via `aiwf cancel`) — accepted but never adopted.
- Terminal: `retired`, `rejected`. Pre-push verification skips contracts in either state.

The entity is a planning record only. Schemas, fixtures, and validators are not on the entity — they are in `aiwf.yaml`. This separation is the load-bearing simplification.

Migration: PoC has zero real contract entities, so this is a code change only. The `entity.go` struct loses `Format`/`Artifact`, gains `LinkedADRs`. The validator goes away. Tests update.

---

## 5. `aiwf.yaml` — the new `contracts:` block

```yaml
# Existing aiwf.yaml fields stay as they are.
aiwf_version: 0.1.0
actor: human/peter

# New, optional, additive.
contracts:
  validators:
    cue:
      command: cue
      args: [vet, "{{schema}}", "{{fixture}}"]
    jsonschema:
      command: ajv
      args: [validate, -s, "{{schema}}", -d, "{{fixture}}"]

  entries:
    - id: C-001
      validator: cue
      schema:   docs/schemas/opspec/schema.cue
      fixtures: docs/schemas/opspec/fixtures
```

### Substitution variables

Exactly four. Documented once, never extended without a kernel-level decision.

| Variable | Resolves to |
|---|---|
| `{{schema}}` | Repo-relative path from `entries[].schema`. |
| `{{fixture}}` | Repo-relative path to the fixture file currently being checked. |
| `{{contract_id}}` | The contract's `C-NNN` id. |
| `{{version}}` | The fixture-tree version directory name (e.g. `v1`). |

### Exit-code semantics

| Validator exit code | aiwf interpretation |
|---|---|
| 0 | Fixture accepted by the schema. |
| Non-zero | Fixture rejected by the schema. |

stderr is captured and surfaced in the finding's `detail` field. stdout is captured for the same purpose. There is no structured-output parsing — the validator's exit code is the only judgment aiwf makes. (If a validator returns "rejected" for malformed-schema reasons rather than fixture-shape reasons, that's classified as `validator-error` based on whether the *same* invocation against any fixture produces consistent results; see §10.)

### Per-fixture invocation, by design

The validator is invoked **once per fixture file**. Even when the underlying tool supports batch mode (`cue vet schema.cue ./fixtures/...`, `ajv -s ... -d '*.json'`), aiwf does not use it. The reason: batch mode would require aiwf to parse the validator's stdout to attribute pass/fail to specific fixtures, which would lock the engine into per-validator output formats forever. Per-fixture invocation keeps the contract — exit code only — narrow and stable.

The cost is real: a hundred fixtures means a hundred subprocess spawns per pre-push run. Repos with very large fixture sets that hit pre-push friction gate verification in CI (running `aiwf contract verify --format=json` from a CI job) rather than at the pre-push hook.

### Programmatic mutation of `aiwf.yaml`

The engine writes to `aiwf.yaml` programmatically (verbs in §6 mutate the `contracts:` block). Round-trip is best-effort:

- Comments and formatting **outside** the `contracts:` block are preserved exactly.
- Within the `contracts:` block, formatting (indentation, intra-block blank lines) is normalized on first programmatic write. Subsequent writes are stable.
- Anchors and aliases inside the `contracts:` block are not supported; the writer fails loudly if it encounters any.

This is documented behavior, not an accident of the implementation. Users who want bespoke formatting inside `contracts:` should not get it; the verbs own that block.

### Working directory

The validator runs with the consumer repo root as cwd. Predictable, matches what `aiwf check` already does, lets `args` use repo-relative paths.

### Path resolution for `command`

`command` is resolved via `exec.LookPath`. It can be an absolute path, a PATH-relative binary name, or a repo-relative path (e.g. `./scripts/my-validator`). All three work; aiwf does not care which.

### Validation of the block at parse time

Every `entries[].validator` must reference a name in `validators`. Every `entries[].id` must match the `C-NNN` format. Every `entries[].schema` and `entries[].fixtures` path is checked for existence at verify time, not parse time (so `aiwf init` can succeed before files exist).

---

## 6. The verb surface

Every mutation of the contract surface — entity, binding, validator declaration — has a verb. Hand-editing `aiwf.yaml.contracts.*` is not a supported workflow; the LLM is instructed to use the verb in every case.

### Mutating verbs

| Verb | Effect | Trailer |
|---|---|---|
| `aiwf add contract --title "..." --linked-adr ADR-NNN [--validator <name> --schema <path> --fixtures <path>]` | Create entity. If all three binding flags present, also append to `aiwf.yaml.contracts.entries[]` in the same commit. | `aiwf-verb: add` |
| `aiwf contract bind <C-id> --validator <name> --schema <path> --fixtures <path>` | Add or replace the binding for an existing contract. Idempotent if args match exactly; errors on mismatch unless `--force`. | `aiwf-verb: bind` |
| `aiwf contract unbind <C-id>` | Remove the binding for a contract. Entity status unchanged. | `aiwf-verb: unbind` |
| `aiwf contract recipe install <name>` | Append the named embedded recipe's validator block to `aiwf.yaml.contracts.validators`. Idempotent if exact match exists; errors on name collision with different definition unless `--force`. | `aiwf-verb: recipe-install` + one `aiwf-entity:` trailer per binding currently referencing the validator name (so `aiwf history C-NNN` surfaces the change). |
| `aiwf contract recipe install --from <path>` | Same effect as above, but reads the validator block from a YAML file instead of the embedded recipe set. The path to a tool-driven custom-validator install. | Same trailers as above. |
| `aiwf contract recipe remove <name>` | Remove the named validator from `aiwf.yaml.contracts.validators`. Errors if any binding still references it (the user must `unbind` or rebind those contracts first). | `aiwf-verb: recipe-remove`. (No `aiwf-entity:` trailer because the verb errors out if any binding references it; once it succeeds, no contract is affected.) |
| `aiwf promote <C-id> <status> [--reason "..."]` | Walk the entity's status forward (`proposed → accepted → deprecated → retired`). Existing generic verb. | `aiwf-verb: promote` |
| `aiwf cancel <C-id> [--reason "..."]` | Move the entity to `rejected` from `proposed` or `accepted`. Existing generic verb. | `aiwf-verb: cancel` |

Status changes (`promote`, `cancel`) **never modify the binding**. The binding stays in `aiwf.yaml.contracts.entries[]` until removed via `unbind`. Pre-push verification automatically skips contracts whose entity status is `rejected` or `retired`.

### Read-only verbs

| Verb | Effect |
|---|---|
| `aiwf contract verify [<C-id>]` | Run verify+evolve. Both passes always run; no mode flags. `--format=json [--pretty]` for tooling. |
| `aiwf contract recipes` | List the embedded recipes plus the validators currently declared in `aiwf.yaml`. |
| `aiwf contract recipe show <name>` | Print the embedded recipe's markdown (the validator block, install instructions, gotchas, worked example). |
| `aiwf list contracts` | Generic verb. List contract entities with their statuses and binding state. |
| `aiwf show <C-id>` | Generic verb. Full record + lifecycle. |
| `aiwf history <C-id>` | Generic verb. Lifecycle events from git log, including `bind` / `unbind` / `recipe-install` / `recipe-remove` mutations relevant to the contract. |

### `aiwf contract verify` semantics

Both passes always run; the combined envelope is the only output shape.

Same exit codes as every other aiwf verb: `0` clean, `1` findings, `2` usage, `3` internal.

When `aiwf.yaml.contracts.entries[]` is empty or absent, the verb returns `status: skipped` with `metadata.reason: "no contracts configured"` and exits `0`. When invoked with a `<C-id>` whose entity status is terminal (`rejected` / `retired`), the verb returns `status: skipped` with `metadata.reason: "contract is <status>"` and exits `0`.

### Verb error semantics

Every mutating verb pre-validates its inputs against the projected state before any disk write. Common usage errors all exit `2` with a one-line `Error:` to stderr naming the specific cause and (where applicable) the verb that would resolve it. `aiwf.yaml` and the working tree are untouched on error. Specifically:

| Condition | Verb behavior |
|---|---|
| `aiwf contract bind <C-id>` and `<C-id>` has no entity in the tree | Exit `2`. `Error: no contract entity <C-id> found; create it first via 'aiwf add contract'.` |
| `aiwf contract bind ... --validator <name>` and `<name>` is not in `aiwf.yaml.contracts.validators` | Exit `2`. `Error: validator '<name>' not declared; install via 'aiwf contract recipe install <name>' or 'aiwf contract recipe install --from <path>'.` |
| `aiwf contract bind <C-id>` when an entry for `<C-id>` already exists with **different** values | Exit `2`. `Error: binding for <C-id> already exists with different values; pass --force to replace.` |
| `aiwf contract unbind <C-id>` when no entry exists | Exit `2`. `Error: no binding for <C-id> in aiwf.yaml.contracts.entries.` |
| `aiwf contract recipe install <name>` and `<name>` is not in the embedded recipe set | Exit `2`. `Error: no embedded recipe '<name>'; see 'aiwf contract recipes' for the shipped set, or use --from <path> for a custom validator.` |
| `aiwf contract recipe install --from <path>` and the file is missing, malformed YAML, missing required fields (`name`, `command`, `args`), or carries unknown fields | Exit `2`. `Error: <path>: <specific reason>.` Required fields: `name`, `command`, `args`. No other fields are accepted. |
| `aiwf contract recipe install <name>` when a validator with the same name but different definition already exists | Exit `2`. `Error: validator '<name>' already declared with different definition; pass --force to replace.` |
| `aiwf contract recipe remove <name>` when one or more bindings reference `<name>` | Exit `2`. `Error: validator '<name>' is referenced by bindings: C-001, C-007. Unbind or rebind those contracts first.` |
| `aiwf contract verify <C-id>` when `<C-id>` has no entity | Exit `2`. `Error: no contract entity <C-id> found.` |
| `aiwf add contract --validator <name>` and `<name>` is not declared | Exit `2`. Same message as `bind` above. The entity is **not** created — the verb is atomic across both files. |

The `--force` flag on `bind` and `recipe install` overrides only the exact-match-required-for-idempotency check. It never overrides shape validation (a malformed `--from` file is rejected even with `--force`).

Pre-push (`aiwf check`) does not invoke these mutating verbs and therefore does not produce these error classes — its concerns are the read-only `contract-config` and verification findings in §10.

---

## 7. Pre-push integration

`aiwf check` is the framework's chokepoint. Wire contract verification into it:

- If `aiwf.yaml.contracts.entries[]` is non-empty, `aiwf check` runs verify+evolve as one of its checks.
- Contracts whose entity status is `rejected` or `retired` are skipped — verification on a terminal-state contract would be misleading. The binding can stay in place (it's a planning concern when to remove it).
- Findings flow into the same envelope as frontmatter findings.
- The pre-push hook installs once via `aiwf init` (already shipped); no new hook to wire.

This is the structural piece without which the whole feature is advisory. The user does not need to remember to run `aiwf contract verify` — every push runs it.

There is no config knob to disable pre-push contract verification. `git push --no-verify` exists for one-off escapes; repos that find pre-push verification too slow gate verification in CI instead (running `aiwf contract verify --format=json` from a CI job). Disabling the chokepoint declaratively would erode the principle that makes the framework's guarantees real.

---

## 8. Fixture-tree convention

Hardcoded in the walker:

```
<entries[].fixtures>/<version>/valid/*<ext>      # must pass
<entries[].fixtures>/<version>/invalid/*<ext>    # must fail
```

Where `<version>` is any directory name (typically `v1`, `v2`, …), and `<ext>` is any file extension (the validator decides what it can read; aiwf does not filter by extension). Files outside this layout are silently ignored — natural enforcement, not a second engine.

The verify pass walks the *current* version (lexicographically highest directory name; document this). The evolve pass walks every version's `valid/` and runs them all against the HEAD schema.

`fixtures:` may point at an empty directory; the verb returns `status: skipped` for that contract with `metadata.reason: "no fixtures present"`. Useful when a contract is registered but its bundle is in flight.

---

## 9. Recipes — embedded content, verb-driven install

aiwf ships an embedded set of recipes, materialized like skills. Each recipe is a single markdown file containing:

- Prose explaining when the language is a good fit.
- The validator block (the exact `command` + `args` shape).
- Install instructions for the validator binary (`brew install cue`, `npm install -g ajv-cli`, etc.).
- Per-language gotchas (CUE's `vet` vs. `eval`, JSON Schema's draft selection, …).
- A worked example: schema + valid + invalid fixture.

The verbs in §6 are the surface: `aiwf contract recipes` (list), `aiwf contract recipe show <name>` (print), `aiwf contract recipe install <name>` (apply embedded), `aiwf contract recipe install --from <path>` (apply custom from a file), `aiwf contract recipe remove <name>` (uninstall).

### Initial recipe set

Two recipes ship in I1 — enough to demonstrate "adding a language is just a markdown file":

- `cue` — CUE schemas, `cue vet`.
- `jsonschema` — JSON Schema, `ajv validate`.

Recipes for Protobuf, OpenAPI, Pydantic, Avro land via PR when someone needs them. Each is a markdown file; no engine code change.

### Why ship recipes at all

Recipes are a courtesy, not a default. The engine never branches on recipe name. A user who picks a language we don't ship — Pydantic, custom validator, Cap'n Proto — writes a small YAML snippet describing the invocation and runs `aiwf contract recipe install --from <path>` to install it. No hand-editing of `aiwf.yaml`; the verb is always the path.

### Custom-validator file shape

The `--from <path>` flag reads a YAML file with this shape:

```yaml
name: pydantic
command: python
args: [-m, my_validator, "{{schema}}", "{{fixture}}"]
```

That's the entire contract — same fields the embedded recipes carry, no more. The verb validates the shape, applies the install, and the file can be deleted (or kept in the repo as documentation).

---

## 10. Findings

| Code | Severity | Meaning |
|---|---|---|
| `fixture-rejected` | error | `valid/` fixture failed the schema. Fix the fixture or loosen the schema. |
| `fixture-accepted` | error | `invalid/` fixture passed the schema. Tighten the schema. |
| `evolution-regression` | error | Historical valid fixture fails HEAD schema. Revert the schema change or migrate. |
| `validator-error` | error | Validator returned non-pass-non-fail (crashed, malformed schema, missing input). Surface stderr in `detail`. |
| `environment` | error | Validator binary not on PATH. One per contract, not one per fixture. |
| `contract-config` | error | `aiwf.yaml.contracts.entries[]` references missing schema, missing fixtures dir, unknown validator name, or id without a matching contract entity. Detected by `aiwf check`. |

`validator-error` is distinguished from `fixture-rejected` by consistency: if the same validator invocation crashes for *every* fixture under one contract, the engine emits one `validator-error` and skips the rest of that contract's fixtures. If it crashes for some and not others, each crash is a per-fixture `validator-error`.

There is no `--fix`. Every finding is a human decision.

---

## 11. The `aiwf-contract` skill

One skill, materialized into `.claude/skills/aiwf-contract/SKILL.md` like every other aiwf skill. Authoritative for the contract surface from the LLM's point of view.

The skill carries one cardinal rule — **never hand-edit `aiwf.yaml`** — and a verb-cheat-sheet table mapping every contract-related user request to the verb that performs it. Below the table, a decision tree enumerates every reasonable user ask: onboarding, authoring, verifying, debugging each finding, lifecycle moves, binding management, recipe management, custom validators, registry-only contracts, adopting an existing contract-verification setup, cancellation, pre-push behavior. Full draft below in §16.

---

## 12. Migration manifest extension

The current import manifest (`docs/poc-import-format.md`) carries entities only. Adopting aiwf in a repo that already has a contract-verification setup needs entities **and** the `aiwf.yaml.contracts:` config to land in one atomic operation. Extend the manifest:

```yaml
version: 1                              # unchanged; the new block is additive
entities: [...]                         # existing

contracts:                              # NEW — optional
  validators:
    cue:
      command: cue
      args: [vet, "{{schema}}", "{{fixture}}"]
  entries:
    - id: C-001                         # must match a `kind: contract` entity in `entities:`
      validator: cue                    # or in the existing tree
      schema:   docs/schemas/opspec/schema.cue
      fixtures: docs/schemas/opspec/fixtures
```

### Semantics

- Validation runs against the *projected* state (existing tree + new entities + patched `aiwf.yaml`). Findings abort before any disk write.
- Validator-block collisions: a validator name already in `aiwf.yaml` that's byte-equal after normalization is a no-op. A different definition with the same name follows `--on-collision` semantics (default `fail`, `skip` keeps existing, `update` overwrites).
- Entry collisions: an `entries[].id` already in `aiwf.yaml.contracts.entries[]` follows the same `--on-collision` rules.
- Atomicity preserved: entity writes, the `aiwf.yaml` patch, and the commit happen as one unit.
- Schemas and fixtures are *not* moved or copied. The producer places them; the manifest only references their paths. Both must exist on disk at import time, or the import errors with a `contract-config` finding.

### Commit trailers

Single-mode imports gain one new trailer when the `contracts:` block was applied:

```
aiwf-verb: import
aiwf-actor: human/peter
aiwf-touched: contracts
```

Per-entity-mode imports apply the `contracts:` block as one extra commit at the end of the batch with the same trailer. Config never spreads across N commits.

### Producer mapping for adopting repos

The migration follows aiwf's standard adoption shape (see `docs/poc-migrating-from-prior-systems.md`): the consumer repo writes a private projector that reads its existing contract setup and emits a manifest. aiwf consumes the manifest; it never reads the source.

A typical prior setup has, for each contract surface, four pieces of information: an identifier, the schema-language name, the schema's path, and the fixtures directory. The projector maps each row of that source data into the manifest:

| Source datum | Manifest target |
|---|---|
| identifier | `entities[].id` (allocate `C-NNN` if the source ids weren't `C-`-prefixed) and `contracts.entries[].id` |
| schema-language name | `contracts.entries[].validator` (and a corresponding `contracts.validators.<lang>` block if not already declared) |
| schema path | `contracts.entries[].schema` (verbatim) |
| fixtures path | `contracts.entries[].fixtures` (verbatim) |
| ADR file the contract was decided in | `entities[].frontmatter.linked_adrs[]` (the ADR itself becomes an `adr` entity if not already in the tree) |

If the source does not carry a status for the contract, the projector picks one: `accepted` when the linked ADR is in `accepted`/`approved`, else `proposed`. Producers can override per row.

Schemas and fixtures stay where they are; the manifest references their existing paths. The producer is responsible for placing them; aiwf does not move or copy schema files.

### Migration guide updates

Append a §"Migrating an existing contract-verification setup" section to `docs/poc-migrating-from-prior-systems.md` covering: recipe install precedes import; schemas and fixtures must already be on disk; the schema-evolution loop runs after the first import; iterate via `aiwf import --dry-run` until clean.

Full migration-manifest spec lands in `docs/poc-import-format.md` under a new §"Optional: contracts config" subsection.

---

## 13. Implementation increments

| Increment | Scope | Depends on |
|---|---|---|
| **I1 — Verify + evolve, full verb surface, in-engine.** | New `entity.Contract` field set (drop `format`/`artifact`, add `linked_adrs`; status set narrowed). Drop `contract-artifact-exists` validator. New `tools/internal/aiwfyaml/` package: yaml-Node-level reader/writer for the `contracts:` block, comment-preserving outside the block, normalizing within. New `tools/internal/contractverify/` package: substitution runner, fixture walker, verify pass, evolve pass. New CLI verbs: `aiwf contract verify`, `aiwf contract bind`, `aiwf contract unbind`, `aiwf contract recipes`, `aiwf contract recipe show`, `aiwf contract recipe install [<name>\|--from <path>]`, `aiwf contract recipe remove`. Extension of `aiwf add contract` with `--validator/--schema/--fixtures` flags. Pre-push integration in `aiwf check` (with terminal-status skip). New finding codes. Embedded recipes for CUE + JSON Schema. New `aiwf-contract` skill. | PoC sessions 1–5 complete (current branch baseline). |
| **I2 — Migration manifest extension.** | `contracts:` top-level block in import manifest parser; collision semantics; trailer; producer mapping documented. | I1. |
| **I3 — Recipe ecosystem.** | Add recipes for Protobuf, OpenAPI, Pydantic when a real consumer needs each. Pure markdown additions; no engine code. | I1. |
| **I4 — Optional: tighter milestone-wrap discipline.** | The `## Contract matrix changes` milestone-spec section, plan-time validation, wrap-time registry-presence check. Defer until a real consumer hits friction managing the registry by hand. | I1, plus a "milestone wrap" lifecycle moment that doesn't currently exist as a distinct check phase. |

I1 is the floor: without it, contracts are not enforced. I2 is the floor for adoption from a repo that already has a contract-verification setup. I3 is pure content (markdown additions; no engine code). I4 is deferred until real friction surfaces.

---

## 14. What this plan does *not* commit to

- No compiled-in validators. Adding a language is a YAML block plus an optional recipe markdown file, never an engine release.
- No `.aiwf/hooks.yaml` or executable hook scripts in the consumer repo. Validator declarations live in `aiwf.yaml`; the runner is in the engine.
- No `live_source` field on the entity. No drift detection on file paths. Contracts are verified by exercising fixtures, not by watching files.
- No `aiwf render contracts` / `CONTRACTS.md` auto-generation. A contract registry is browsable via `aiwf list contracts`. Hand-maintained matrix files in the consumer repo (e.g. `docs/architecture/indexes/contract-matrix.md`) remain repo content.
- No `aiwf contract list / status / matrix` verbs. Generic verbs (`aiwf list contracts`, `aiwf show C-NNN`, `aiwf history C-NNN`) cover these.
- No symbol-level live-source tracking. Out of scope for the engine.
- No `aiwf contract verify --mode <verify|evolve>`. Both passes always run.
- No filter flags (`--drifted`, `--untouched-since`, `--linked-adr`, `--verified-status`) until a real user asks.
- No vendored validator binaries. The user installs `cue` / `ajv` / etc. themselves.
- No structured parsing of validator stdout. Exit code is the judgment; stderr/stdout are surfaced verbatim in finding `detail`.
- No `skip_in_check` config knob (or any other declarative escape from pre-push contract verification). `git push --no-verify` is the per-push escape; CI is the move-the-gate option.
- No hand-editing of `aiwf.yaml.contracts.*` as a supported workflow. Every mutation has a verb. The skill's cardinal rule is that the LLM uses the verb, never the text editor.

If a future change reaches for any of the above, it is a kernel-level decision and gets surfaced explicitly.

---

## 15. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Engine accumulates validator-specific knowledge over time | Med | High (breaks the factoring) | Hard rule: `tools/internal/contractverify/` may not import a validator-specific package, may not branch on validator name, may not depend on a specific validator's binary being installed during tests. Tests run validators against real fixtures, but production code is validator-agnostic. |
| Substitution variable set creeps | Med | Med | Lock the four variables in `architecture.md`. Adding a fifth is a kernel-level decision. |
| Recipe markdown rots as engine evolves | Med | Low | Recipes ship embedded; CI builds the binary which embeds them; a recipe whose YAML block doesn't parse fails the build. |
| Validator binary missing → noisy findings | Low | Low | One `environment` finding per contract, not per fixture. Already specified. |
| User declares a validator that doesn't follow exit-code convention | Low | Med | Document the convention in the recipe; advise wrapping non-conforming validators in a 3-line shell script if needed. Don't try to autodetect. |
| Schemas and fixtures change paths after migration | Med | Low | `contract-config` finding fires on next `aiwf check`. User runs `aiwf contract bind <id> --force` to repair, or `aiwf contract unbind <id>` to remove. Mechanical, not silent. |
| YAML round-trip on `aiwf.yaml.contracts:` block surprises the user | Med | Low | The first programmatic write normalizes formatting *within* the `contracts:` block; everything outside is preserved exactly. Documented behavior (§5). Anchors/aliases inside the block are a hard error, not a silent corruption. |

---

## 16. The `aiwf-contract` skill — full draft

Materialized at `.claude/skills/aiwf-contract/SKILL.md`. Source at `tools/internal/skills/embedded/aiwf-contract/SKILL.md`.

````markdown
---
name: aiwf-contract
description: Use whenever the user mentions contracts — onboarding, authoring, verifying, evolving schemas, debugging findings, registering, deprecating, lifecycle, recipes, custom validators, or adopting an existing contract-verification setup. Authoritative for the contract verification surface.
---

# aiwf-contract

This skill is advisory; the binary is authoritative.

A **contract** in aiwf is a bounded surface in the consumer repo whose shape is enforced mechanically — typically a CUE schema, a JSON Schema, a `.proto`, or an OpenAPI document. aiwf provides:

1. A **registry record** (`contract` entity, `C-NNN`) tying the surface to ADRs and a status.
2. A **verification engine** (`aiwf contract verify`) that runs the user's chosen validator against fixtures and reports findings.
3. A **pre-push hook** (`aiwf check`) that runs verification automatically when contracts are configured.

aiwf ships **zero validators**. Validators are user-declared in `aiwf.yaml` via verbs, invoking binaries the user installed (`cue`, `ajv`, etc.). Recipes for common languages ship as embedded markdown — opt-in, not prescribed.

## The cardinal rule: never hand-edit `aiwf.yaml`

Every mutation of `aiwf.yaml.contracts.*` has a verb. The LLM uses the verb. Always.

| Want to … | Use this verb |
|---|---|
| Create a contract entity (and optionally bind it) | `aiwf add contract --title "..." --linked-adr ADR-NNN [--validator <name> --schema <path> --fixtures <path>]` |
| Add or replace a binding for an existing contract | `aiwf contract bind <C-id> --validator <name> --schema <path> --fixtures <path>` |
| Remove a binding | `aiwf contract unbind <C-id>` |
| Install a validator from a shipped recipe | `aiwf contract recipe install <name>` |
| Install a custom validator from a YAML file | `aiwf contract recipe install --from <path>` |
| Remove a validator declaration | `aiwf contract recipe remove <name>` |
| Move the entity's status forward | `aiwf promote <C-id> <status> --reason "..."` |
| Cancel a contract (rejected) | `aiwf cancel <C-id> --reason "..."` |
| Run verification | `aiwf contract verify [<C-id>]` |
| List recipes / contracts | `aiwf contract recipes`, `aiwf list contracts` |
| Inspect | `aiwf contract recipe show <name>`, `aiwf show <C-id>`, `aiwf history <C-id>` |

If the user asks for something not covered by a verb, the answer is **not** "edit `aiwf.yaml` directly." The answer is "we don't support that mutation; here is the closest verb."

## Decision tree — what is the user asking?

Match the user's request to the closest case below and follow the action.

### "I want to add contract verification" / "How do I start?"

Onboarding. Four steps, in order:

1. **Pick a validator language.** Run `aiwf contract recipes` to show what aiwf has prewritten. The user picks. Do not recommend a language; if the user already runs `cue` / `ajv` / etc. in their repo, prefer that.
2. **Install the recipe.** `aiwf contract recipe install <language>`. The verb appends the validator block to `aiwf.yaml.contracts.validators` and creates `docs/schemas/` if missing. Idempotent.
3. **Author the first contract bundle** — see "Author a new contract" below.
4. **Run `aiwf contract verify`.** If clean, push. The pre-push hook reruns verification on every push.

If the user's language has no shipped recipe, see "Language with no recipe."

### "I want to author a new contract" / "Design a contract for X"

A contract bundle has five pieces. All five are required to call the bundle complete. Walk the user through each:

1. **An ADR explaining what the contract is and why.** `aiwf add adr --title "..."`. The ADR body covers: who produces, who consumes, what failure modes a free-form interface would carry, what alternatives were considered.
2. **The authoritative schema** in the user's chosen language at `docs/schemas/<topic>/schema.<ext>`. The schema is the single source of truth. Generated types and runtime checks derive from it; never the other way around.
3. **At least one valid fixture** at `docs/schemas/<topic>/fixtures/v1/valid/<name>.<ext>`. Demonstrates the shape the schema accepts.
4. **At least one invalid fixture** at `docs/schemas/<topic>/fixtures/v1/invalid/<name>.<ext>`. Demonstrates a shape the schema rejects. **Invalid fixtures are not optional** — without them, the schema's permissiveness goes untested. "Schema accepted something we didn't intend" is the dominant contract bug class.
5. **A worked example** — one realistic, end-to-end scenario with concrete domain values. No `<placeholder>`, no `lorem ipsum`. Real names, real numbers, real dates. Lives at the path documented in the ADR. Proves a human can read the shape and tell what it means.

After authoring, register and bind the contract in one verb:

```bash
aiwf add contract \
  --title "Op execution spec" \
  --linked-adr ADR-OPSPEC-01 \
  --validator cue \
  --schema   docs/schemas/opspec/schema.cue \
  --fixtures docs/schemas/opspec/fixtures
```

That single verb creates the entity **and** the binding in one commit. Then:

```bash
aiwf contract verify C-NNN
```

and show the user the result.

If the user has not picked a validator yet, fall back to onboarding. If the user wants registry-only (no validator yet), omit the three binding flags — the entity is created without a binding; add it later via `aiwf contract bind`.

### "Verify the contracts" / "Run verification"

```bash
aiwf contract verify             # all contracts; both passes
aiwf contract verify C-NNN       # one contract
```

The verb runs **two passes** every time:

- **Verify pass:** every fixture under `valid/` must pass the validator; every fixture under `invalid/` must fail.
- **Evolve pass:** every historical valid fixture (across all `v<N>/` directories) must still pass the HEAD schema. Catches schema changes that silently break existing consumers.

There are no mode flags. Show the envelope. If clean, say so plainly. If findings, walk the user through them via the findings reference below.

Contracts whose entity status is `rejected` or `retired` are skipped automatically — that is correct, do not flag it as a bug.

### Finding: `fixture-rejected`

A `valid/` fixture was rejected. Two possibilities — let the user decide:

- **The fixture is wrong.** It declared a shape the schema (correctly) doesn't accept. Fix the fixture.
- **The schema is too strict.** Loosen the schema and document the change in the ADR if it's meaningful.

Never delete a fixture to silence a finding. If a fixture's domain is genuinely retired, that's an ADR-level decision; document it.

### Finding: `fixture-accepted`

An `invalid/` fixture passed the schema. The schema is too permissive. Tighten constraints. Highest-signal finding class — indicates the validator was silently approving shapes the team thinks are wrong.

### Finding: `evolution-regression`

A historical valid fixture (one that used to pass) no longer passes HEAD. The schema change broke an existing consumer. Two paths, never both:

1. **Revert the schema change** — if the regression was unintentional.
2. **Add a migration ADR + bump the schema major version** — if intentional. ADR documents what changed, why, and how consumers migrate. Often paired with a transformation step that brings old-version fixtures forward.

### Finding: `environment`

The validator binary is not on PATH. The fix lives in the user's tool-versions / devcontainer / install docs — outside aiwf's scope. Point at the recipe (`aiwf contract recipe show <language>`), which lists the install command.

### Finding: `validator-error`

Validator returned non-pass-non-fail (usually malformed schema or fixture). Stderr is in the finding's `detail` field; show it to the user and let them diagnose.

### Finding: `contract-config`

A binding in `aiwf.yaml.contracts.entries[]` is misconfigured. Common causes: schema path doesn't exist, fixtures directory doesn't exist, `validator:` references an unknown name, `id:` doesn't match any contract entity. The finding's `detail` names which.

The fix is **always a verb**: `aiwf contract bind <id> --validator <name> --schema <correct-path> --fixtures <correct-path> --force` to repair, or `aiwf contract unbind <id>` to remove. Do not edit `aiwf.yaml` to "fix" the path.

### "What contracts do I have?" / "List contracts"

```bash
aiwf list contracts
aiwf show C-NNN
aiwf history C-NNN
```

Generic verbs. Do not invent `aiwf contract list` or `aiwf contract status` — they don't exist by design.

### "Move C-NNN through its lifecycle"

The contract status set is closed:

```
proposed → accepted → deprecated → retired
        ↘ rejected ↙
```

```bash
aiwf promote C-NNN accepted   --reason "ADR-NNN approved 2026-04-30"
aiwf promote C-NNN deprecated --reason "phasing out for C-NNN+1"
aiwf promote C-NNN retired    --reason "no consumers remain"
aiwf cancel  C-NNN            --reason "rolled into ADR-NNN-rev2"
```

`--reason` is optional but appears in `aiwf history C-NNN`; encourage it for status moves.

**Status changes never modify the binding.** A `retired` contract still has its `entries[]` row in `aiwf.yaml`; pre-push verification simply skips it. To remove the binding, run `aiwf contract unbind C-NNN` separately. The two-step is intentional: status moves are forward-only (per the transitions diagram in the plan) and shouldn't destroy operational config that may still carry historical or audit value.

### "Deprecate a contract"

`aiwf promote C-NNN deprecated`. Deprecation is not retirement — verification still runs (the contract is not yet terminal). The signal is for human consumers ("don't write new code against this").

### "Stop verifying C-NNN" / "I don't want this contract checked anymore"

Two cases, two verbs:

- **The contract is finished its lifecycle:** `aiwf promote C-NNN retired`. Verification stops automatically (terminal-status skip). The binding stays in `aiwf.yaml` for historical reference.
- **The binding was wrong / outdated and should go away regardless of contract status:** `aiwf contract unbind C-NNN`. Removes the binding row only; entity status unchanged.

### "Language with no recipe" (Pydantic, Avro, custom validator, …)

The user creates a small YAML file describing the validator, then installs it via the verb:

```yaml
# pydantic-validator.yaml
name: pydantic
command: python
args: [-m, my_validator, "{{schema}}", "{{fixture}}"]
```

```bash
aiwf contract recipe install --from pydantic-validator.yaml
```

The verb validates the file shape and applies the install. The YAML file can be deleted after, or kept in the repo as documentation — aiwf doesn't track it.

Substitution variables: `{{schema}}`, `{{fixture}}`, `{{contract_id}}`, `{{version}}`. Exit code 0 = accept, non-zero = reject. Stderr surfaces in `validator-error` findings.

If the user wants to share their pattern, encourage upstreaming a recipe — but don't gate adoption on it.

### "Add a contract without a validator yet"

Allowed and supported. Run `aiwf add contract --title "..." --linked-adr ADR-NNN` **without** the `--validator / --schema / --fixtures` flags. The entity is created; no binding is added. The contract appears in `aiwf list contracts` as a registry record but has no verification target. When the user is ready to wire validation, run `aiwf contract bind <id> --validator ... --schema ... --fixtures ...`.

This is the right answer when:

- The schema language is undecided.
- The contract is a prose-only specification.
- The validator is not yet installed in the team's environment.

### "How do I add my own recipe to the engine?"

Recipes ship embedded in the binary. To upstream one, contribute a markdown file to `tools/internal/skills/embedded/aiwf-contract/recipes/<lang>.md` (PR). Local-only recipes are not supported by design — drift between repos defeats the recipe pattern. For per-repo custom validators, use `aiwf contract recipe install --from <path>` (above).

### "I already have a contract-verification setup — adopt me into aiwf"

Use `aiwf import <manifest>`. The migration manifest carries entities **and** the `aiwf.yaml.contracts:` config in one atomic operation. See `docs/poc-migrating-from-prior-systems.md` §"Migrating an existing contract-verification setup."

Producer mapping: for each existing contract surface, the projector emits one contract entity (with `linked_adrs` from the contract's ADR) + one entry in `contracts.entries[]` + a `contracts.validators.<lang>` block if not already declared. Schemas and fixtures stay where they are; the manifest references existing paths.

### "Cancel a contract entirely"

`aiwf cancel C-NNN` moves the entity from `proposed` or `accepted` to `rejected`. The contract is no longer verified (terminal-status skip). Schema files and fixtures are not deleted — cancellation is a status change. To remove the binding row from `aiwf.yaml.contracts.entries[]`, run `aiwf contract unbind C-NNN` separately.

If the user wants to delete schema files, that's a separate manual step; aiwf does not delete user files.

### "How does pre-push work for contracts?"

`aiwf check` runs as a pre-push hook (installed by `aiwf init`). When `aiwf.yaml.contracts.entries[]` is non-empty, `aiwf check` runs the verify+evolve passes for every contract whose entity status is **not** terminal. Findings block the push.

There is no config knob to disable pre-push contract verification. Repos with very large fixture sets that find pre-push too slow gate verification in CI instead (running `aiwf contract verify --format=json` from a CI job). Do not suggest a `skip_in_check` flag — it doesn't exist by design.

`git push --no-verify` is the per-push escape if the user genuinely needs to push past a finding once.

## Findings reference

| Code | Severity | Meaning |
|---|---|---|
| `fixture-rejected` | error | A `valid/` fixture failed the schema. |
| `fixture-accepted` | error | An `invalid/` fixture passed the schema. |
| `evolution-regression` | error | A historical valid fixture fails HEAD. |
| `validator-error` | error | Validator crashed or returned ambiguously. Stderr in `detail`. |
| `environment` | error | Validator binary not on PATH. One per contract. |
| `contract-config` | error | A binding in `aiwf.yaml.contracts.entries[]` is misconfigured. |

There is no `--fix`. Every finding is a human decision; aiwf reports, the user resolves.

## Don't

- **Don't hand-edit `aiwf.yaml`.** Every contract-related mutation has a verb; use it. If you reach for a text editor on `aiwf.yaml`, you are off the supported path.
- Don't recommend a default validator language. Recipes are an offering.
- Don't hand-edit `.git/hooks/pre-push`. `aiwf check` runs verification automatically.
- Don't add validator scripts under `scripts/`. Validators are declared via `aiwf contract recipe install`.
- Don't conflate the registry record (the `contract` entity) with the validator binding (`aiwf.yaml.contracts.entries[]`). Two responsibilities, two locations, linked by `id`. Different verbs for each.
- Don't suggest deleting fixtures to silence findings.
- Don't run `aiwf contract verify` against an empty `entries:` block expecting work — it returns "skipped" and that is correct.
- Don't invent new verb forms (`aiwf contract list`, `aiwf contract status`). The generic `aiwf list contracts` / `aiwf show C-NNN` are authoritative.
- Don't claim a `skip_in_check` config option exists. Pre-push verification is unconditional for non-terminal contracts.
````

---

## 17. References

- `docs/poc-design-decisions.md` — kernel commitments the contract surface must respect.
- `docs/poc-import-format.md` — to be extended with the `contracts:` block per §12.
- `docs/poc-migrating-from-prior-systems.md` — to be extended with §"Migrating an existing contract-verification setup" per §12.
- `tools/internal/entity/entity.go` — site of the entity narrowing (drop `Format`/`Artifact`, add `LinkedADRs`).
- `tools/internal/check/check.go` — site of the `contract-artifact-exists` removal and the new `contract-config` check.

---

## 18. Status

Updated as work lands. Granularity matches §13's increments, broken down into the discrete shippable steps inside I1.

| Step | Scope | State |
|---|---|---|
| I1.1 — Entity narrowing | `entity.Contract`: drop `Format`/`Artifact`, add `LinkedADRs`; narrow status set to `proposed → accepted → deprecated → retired` (+ `rejected`); update transition function; drop `contract-artifact-exists` validator; update fixtures and tests | ✅ done |
| I1.2 — `aiwfyaml` package | `tools/internal/aiwfyaml/`: yaml-Node-level reader/writer for the `contracts:` block, comment-preserving outside the block, normalizing within; anchors/aliases inside the block are a hard error | ✅ done |
| I1.3 — `contractverify` package | `tools/internal/contractverify/`: substitution runner (`{{schema}}`, `{{fixture}}`, `{{contract_id}}`, `{{version}}`), fixture walker (`<fixtures>/<version>/{valid,invalid}/*`), verify pass, evolve pass | ✅ done |
| I1.4 — `aiwf contract verify` verb | CLI integration of the verify+evolve runner; new `contract-config` finding (schema/fixtures path existence, entry id matches a contract entity); per-fixture finding codes (`fixture-rejected`, `fixture-accepted`, `evolution-regression`, `validator-error`, `environment`) shipped with the contractverify package in I1.3 | ✅ done |
| I1.5 — Bind/unbind verbs | `aiwf contract bind`, `aiwf contract unbind`; extend `aiwf add contract` with `--validator/--schema/--fixtures` flags; lifecycle commit trailers | ✅ done |
| I1.6 — Recipe verbs + embedded recipes | `aiwf contract recipes`, `recipe show`, `recipe install [<name>\|--from <path>]`, `recipe remove`; embed CUE + JSON Schema recipes via `embed.FS` | ✅ done |
| I1.7 — Pre-push integration | `aiwf check` runs verify+evolve when `aiwf.yaml.contracts.entries[]` is non-empty; terminal-status contracts (`rejected`, `retired`) skipped | ✅ done |
| I1.8 — `aiwf-contract` skill | Embed and materialize the skill at `.claude/skills/aiwf-contract/SKILL.md` (source draft is §16 of this doc) | ⏳ not started |
| I2 — Migration manifest extension | `contracts:` top-level block in import manifest parser; collision semantics; trailer; producer mapping documented | ⏳ deferred (after I1) |
| I3 — Recipe ecosystem | Recipes for Protobuf, OpenAPI, Pydantic added when a real consumer needs each | ⏳ deferred (real-friction trigger) |
| I4 — Milestone-wrap discipline | `## Contract matrix changes` section, plan-time validation, wrap-time registry-presence check | ⏳ deferred (real-friction trigger) |

State legend: ⏳ not started · 🚧 in progress · ✅ done · ⏸ paused.

Each I1 step lands as one or more commits with Conventional Commits subjects (`feat(aiwf): ...`, `chore(aiwf): ...`, `docs(poc): ...`). Mark a step ✅ only when its scope ships tested and `go test -race ./tools/...` + `golangci-lint run` are clean.
