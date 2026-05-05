---
name: jsonschema
command: ajv
args:
  - validate
  - -s
  - "{{schema}}"
  - -d
  - "{{fixture}}"
---

# JSON Schema recipe

[JSON Schema](https://json-schema.org) is the lingua franca for validating
JSON payloads. The recipe binds aiwf to [Ajv](https://ajv.js.org) via the
`ajv-cli` package — fast, mature, draft-2020-12 compatible.

It's a good fit when:

- The contract is over JSON wire shapes (REST request bodies, event
  envelopes, config files).
- You want broad ecosystem coverage — Ajv is the de-facto JSON Schema
  validator, and the schemas you write are portable to other languages.
- A Node.js runtime is available on the developer's machine and in CI.

## Validator block

The frontmatter at the top of this file is the validator block that
lands in `aiwf.yaml.contracts.validators.jsonschema` when you run
`aiwf contract recipe install jsonschema`. The substitution variables
({{schema}}, {{fixture}}, {{contract_id}}, {{version}}) are the only
inputs aiwf passes.

## Install the binary

```bash
npm install -g ajv-cli ajv-formats   # Ajv plus the formats plugin
```

Confirm the install:

```bash
ajv help
```

## Gotchas

- **Draft selection.** Ajv defaults to draft-2020-12. If your schema
  declares an older `$schema`, Ajv honors it; if it's missing, you may
  want to add one explicitly so behavior doesn't depend on the binary's
  default.
- **`format:` keywords need the formats plugin.** The recipe assumes
  `ajv-formats` is installed; otherwise `ajv validate` ignores
  `"format": "email"` etc.
- **`$ref` resolution is relative to the schema file.** If your schema
  splits across files, point `--schema` at the entrypoint and let Ajv
  follow refs.
- **Output is on stderr.** Ajv writes the human-readable error to
  stderr; aiwf captures both streams and surfaces them in the finding's
  `detail`.

## Worked example

A minimal schema:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["id", "kind"],
  "properties": {
    "id":   { "type": "string", "pattern": "^[A-Z]-\\d+$" },
    "kind": { "enum": ["create", "update", "delete"] }
  },
  "additionalProperties": false
}
```

A valid fixture:

```json
{ "id": "M-001", "kind": "create" }
```

An invalid fixture (extra field, fails `additionalProperties: false`):

```json
{ "id": "M-001", "kind": "create", "stray": true }
```

`ajv validate -s schema.json -d fixture.json` returns 0 for the valid
case and non-zero for the invalid one.
