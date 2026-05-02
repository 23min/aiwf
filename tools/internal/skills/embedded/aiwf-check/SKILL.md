---
name: aiwf-check
description: Use when the user wants to validate the planning tree or asks why `aiwf check` reported a finding. Explains each finding code and the typical fix.
---

# aiwf-check

The `aiwf check` verb is a pure function from the working tree to a list of findings. It runs as a `pre-push` git hook; that hook is the chokepoint that turns the framework's guarantees into mechanical enforcement.

## When to use

- The user wants to know "is the tree clean?".
- A push was blocked by the pre-push hook and the user is asking what the finding means.
- A verb refused to write because the projection introduced a finding.

## What to run

```bash
aiwf check                # human-readable text
aiwf check --format=json  # JSON envelope for tooling
aiwf check --format=json --pretty
```

## Findings (errors)

| Code | Meaning | Typical fix |
|---|---|---|
| `ids-unique` | Two entities share an id. Almost always from a parallel-branch merge. | `aiwf reallocate <path>` on the loser. |
| `frontmatter-shape` | Required field missing or malformed. | Add the field; check the kind's id format. |
| `status-valid` | Status is not in the kind's allowed set. | Pick a status from the kind's set (see `aiwf-promote`). |
| `refs-resolve/unresolved` | A reference points at an id that does not exist. | Either the target was never created, or the id is mistyped. |
| `refs-resolve/wrong-kind` | A reference points at an entity of the wrong kind. | A milestone's `parent` must be an epic; an ADR's `supersedes` must be ADRs; etc. |
| `no-cycles` | A cycle in the milestone `depends_on` DAG or the ADR `supersedes` chain. | Remove a back-edge. |
| `contract-config` | A contract binding in `aiwf.yaml` references an id with no entity, a missing schema/fixtures path, or a contract entity has no binding. | Run `aiwf contract bind` / `aiwf add contract`, fix the path, or `aiwf contract unbind`. |
| `fixture-rejected` | A `valid/` fixture failed the schema. | Make the schema accept it, or move it to `invalid/`. |
| `fixture-accepted` | An `invalid/` fixture passed the schema. | Tighten the schema, or move to `valid/`. |
| `evolution-regression` | A historical `valid/` fixture fails the HEAD schema. | Revert the schema change, migrate the fixture, or rebind. |
| `validator-error` | Every valid fixture for a contract was rejected — the schema or validator invocation is likely broken. | Inspect the captured stderr and fix the schema or validator command. |
| `environment` | Validator binary not on PATH. | Install it (see the recipe's install instructions) or fix `command:` in `aiwf.yaml`. |

## Findings (warnings)

| Code | Meaning |
|---|---|
| `titles-nonempty` | Title is missing or whitespace-only. |
| `adr-supersession-mutual` | ADR A says it's superseded by B, but B does not list A in its `supersedes`. |
| `gap-resolved-has-resolver` | Gap is `addressed` but `addressed_by` is empty. |
| `provenance-untrailered-entity-commit` | A commit between `@{u}` and `HEAD` touched an entity file with no `aiwf-verb:` trailer (manual `git commit`). Repair with `aiwf <verb> --audit-only --reason "..."`. |

## Provenance findings (errors)

These fire on commit history, not tree state. Each names the offending commit's short SHA in its message.

| Code | Meaning | Typical fix |
|---|---|---|
| `provenance-trailer-incoherent` | A required-together pair is partial, or a mutually-exclusive pair are both present (e.g., `aiwf-on-behalf-of:` without `aiwf-authorized-by:`, `aiwf-actor: ai/...` without `aiwf-principal:`, `aiwf-actor: human/...` *with* `aiwf-principal:`). | Re-create the commit using the correct verb invocation; `--principal human/<id>` is required when the actor is non-human. |
| `provenance-force-non-human` | `aiwf-force:` present on a commit whose `aiwf-actor:` is not `human/...`. | `--force` is sovereign — only humans wield it. Have a human invoke the verb directly. |
| `provenance-actor-malformed` | `aiwf-actor:` does not match `<role>/<id>`. | `git config user.email` is malformed; fix it (see `aiwf doctor`). |
| `provenance-principal-non-human` | `aiwf-principal:` role is not `human/`. | Principal must be human/<id>; agents and bots cannot be principals. |
| `provenance-on-behalf-of-non-human` | `aiwf-on-behalf-of:` role is not `human/`. | Same as principal — rebuild from the originating authorize commit. |
| `provenance-authorized-by-malformed` | `aiwf-authorized-by:` is not 7–40 hex. | Copy the correct SHA from `aiwf history <scope-entity>`. |
| `provenance-authorization-missing` | The authorize SHA does not name an `aiwf-verb: authorize / aiwf-scope: opened` commit. | Typo or stale SHA after force-push; use the full SHA. |
| `provenance-authorization-out-of-scope` | The verb's target entity has no reference path to the scope-entity. | Either authorize the right entity or work on something the existing scope already reaches. |
| `provenance-authorization-ended` | The scope was already ended (terminal-promote / revoke). | Open a fresh scope with `aiwf authorize <id> --to <agent>`. |
| `provenance-no-active-scope` | An `ai/...` actor produced a commit with no `aiwf-on-behalf-of:`. | Open an authorization scope, or run the verb as the human directly. |
| `provenance-audit-only-non-human` | `aiwf-audit-only:` present on a non-human actor's commit. | Only humans may backfill audit trails. |

## Don't

- Don't bypass the pre-push hook with `--no-verify` to "fix it later" — broken state on `main` is the thing this hook exists to prevent.
- Don't try to make findings disappear by deleting files; `aiwf cancel <id>` is the right way to retire an entity.
- Don't try to "amend away" a `provenance-untrailered-entity-commit` warning — `aiwf <verb> --audit-only --reason "..."` is the first-class repair path and keeps history append-only.
