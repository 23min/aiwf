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
| Run verification | `aiwf contract verify` |
| List recipes / contracts | `aiwf contract recipes`, `aiwf list --kind contract` |
| Inspect | `aiwf contract recipe show <name>`, `aiwf show <C-id>`, `aiwf history <C-id>` |

If the user asks for something not covered by a verb, the answer is **not** "edit `aiwf.yaml` directly." The answer is "we don't support that mutation; here is the closest verb."

## Decision tree — what is the user asking?

Match the user's request to the closest case below and follow the action.

### "I want to add contract verification" / "How do I start?"

Onboarding. Four steps, in order:

1. **Pick a validator language.** Run `aiwf contract recipes` to show what aiwf has prewritten. The user picks. Do not recommend a language; if the user already runs `cue` / `ajv` / etc. in their repo, prefer that.
2. **Install the recipe.** `aiwf contract recipe install <language>`. The verb appends the validator block to `aiwf.yaml.contracts.validators` and creates the schemas directory if missing. Idempotent.
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
aiwf contract verify
```

and show the user the result.

If the user has not picked a validator yet, fall back to onboarding. If the user wants registry-only (no validator yet), omit the three binding flags — the entity is created without a binding; add it later via `aiwf contract bind`.

### "Verify the contracts" / "Run verification"

```bash
aiwf contract verify
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

Validator returned non-pass-non-fail (usually malformed schema or fixture, or every valid fixture got rejected — a strong signal the schema itself is broken). Stderr is in the finding's `detail` field; show it to the user and let them diagnose.

### Finding: `contract-config`

A binding in `aiwf.yaml.contracts.entries[]` is misconfigured. Common causes: schema path doesn't exist, fixtures directory doesn't exist, `validator:` references an unknown name, `id:` doesn't match any contract entity. The finding's `detail` names which.

The fix is **always a verb**: `aiwf contract bind <id> --validator <name> --schema <correct-path> --fixtures <correct-path> --force` to repair, or `aiwf contract unbind <id>` to remove. Do not edit `aiwf.yaml` to "fix" the path.

### "What contracts do I have?" / "List contracts"

```bash
aiwf contract recipes        # validators side
aiwf show C-NNN              # one entity in detail
aiwf history C-NNN           # lifecycle
```

Generic verbs cover the entity side. Do not invent `aiwf contract list` or `aiwf contract status` — they don't exist by design.

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

**Status changes never modify the binding.** A `retired` contract still has its `entries[]` row in `aiwf.yaml`; pre-push verification simply skips it. To remove the binding, run `aiwf contract unbind C-NNN` separately. The two-step is intentional: status moves are forward-only and shouldn't destroy operational config that may still carry historical or audit value.

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

Allowed and supported. Run `aiwf add contract --title "..." --linked-adr ADR-NNN` **without** the `--validator / --schema / --fixtures` flags. The entity is created; no binding is added. The contract appears as a registry record but has no verification target. When the user is ready to wire validation, run `aiwf contract bind <id> --validator ... --schema ... --fixtures ...`.

This is the right answer when:

- The schema language is undecided.
- The contract is a prose-only specification.
- The validator is not yet installed in the team's environment.

### "How do I add my own recipe to the engine?"

Recipes ship embedded in the binary. To upstream one, contribute a markdown file to `tools/internal/recipe/embedded/<lang>.md` (PR). Local-only recipes are not supported by design — drift between repos defeats the recipe pattern. For per-repo custom validators, use `aiwf contract recipe install --from <path>` (above).

### "I already have a contract-verification setup — adopt me into aiwf"

Use `aiwf import <manifest>`. The migration manifest is planned to carry entities **and** the `aiwf.yaml.contracts:` config in one atomic operation. (Manifest extension lands in I2 of the contracts plan.)

For the engine's current state, do migration in two phases: first land the contract entities via `aiwf import`, then run `aiwf contract recipe install` and `aiwf contract bind` for each binding.

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
| `validator-error` | error | Validator crashed or every valid fixture was rejected. Stderr in `detail`. |
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
- Don't run `aiwf contract verify` against an empty `entries:` block expecting work — it returns nothing, which is correct.
- Don't invent new verb forms (`aiwf contract list`, `aiwf contract status`). The generic `aiwf show C-NNN` and `aiwf history C-NNN` are authoritative.
- Don't claim a `skip_in_check` config option exists. Pre-push verification is unconditional for non-terminal contracts.
