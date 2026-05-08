---
name: cue
command: cue
args:
  - vet
  - "{{schema}}"
  - "{{fixture}}"
---

# CUE recipe

[CUE](https://cuelang.org) is a constraint-based schema language with strong
type inference and a single-binary CLI. It's a good fit when:

- The schema benefits from constraints richer than JSON Schema (e.g.
  cross-field invariants, computed defaults).
- A single static binary that can be installed via `brew install cue` or
  the official release tarballs is acceptable (no language runtime).
- Fixtures can live as JSON or YAML and `cue vet` against the schema.

## Validator block

The frontmatter at the top of this file is exactly the validator block
that lands in `aiwf.yaml.contracts.validators.cue` when you run
`aiwf contract recipe install cue`. The four substitution variables —
`{{schema}}`, `{{fixture}}`, `{{contract_id}}`, `{{version}}` — are the
only inputs aiwf passes; pick whichever your invocation needs.

## Install the binary

```bash
brew install cue                 # macOS
go install cuelang.org/go/cmd/cue@latest   # Go ≥ 1.21 toolchain
```

Confirm the install:

```bash
cue version
```

## Gotchas

- **`cue vet` vs. `cue eval`** — `vet` is the right verb here; it returns
  exit 0 on accept, non-zero on reject. `eval` resolves and prints the
  unified value, which is not what aiwf needs.
- **Schema and fixture types must align.** `cue vet schema.cue fixture.json`
  unifies the fixture *into* the schema; if the schema declares a closed
  struct (`#Foo: close({...})`), unknown fields in the fixture fail.
- **Comments don't carry semantic weight** — CUE's `//` comments are
  stripped before unification. Use definitions and constraints for
  structure.

## Worked example

A minimal schema:

```cue
// schema.cue
package opspec

#Op: {
    name:   string
    kind:   "create" | "update" | "delete"
    target: string
}
```

A valid fixture:

```json
{ "name": "noop", "kind": "create", "target": "/tmp/x" }
```

An invalid fixture (kind not in the allowed set):

```json
{ "name": "noop", "kind": "patch", "target": "/tmp/x" }
```

`cue vet schema.cue fixture.json` returns 0 for the valid case and
non-zero for the invalid one.
