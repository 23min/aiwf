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

## Findings (warnings)

| Code | Meaning |
|---|---|
| `titles-nonempty` | Title is missing or whitespace-only. |
| `adr-supersession-mutual` | ADR A says it's superseded by B, but B does not list A in its `supersedes`. |
| `gap-resolved-has-resolver` | Gap is `addressed` but `addressed_by` is empty. |

## Don't

- Don't bypass the pre-push hook with `--no-verify` to "fix it later" — broken state on `main` is the thing this hook exists to prevent.
- Don't try to make findings disappear by deleting files; `aiwf cancel <id>` is the right way to retire an entity.
