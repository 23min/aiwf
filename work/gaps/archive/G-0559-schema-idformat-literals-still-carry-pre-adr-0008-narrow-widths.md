---
id: G-0559
title: Schema IDFormat literals still carry pre-ADR-0008 narrow widths
status: addressed
priority: medium
addressed_by_commit:
    - b778bc9f32ad147df12bb83df3fa6c7c032175ce
---
## What's missing

`entity.Schema.IDFormat` still carries the per-kind widths ADR-0008 replaced.
The `schemas` table in `internal/entity/entity.go` declares `E-NN`, `M-NNN`,
`G-NNN`, `D-NNN`, `C-NNN` — the pre-canonicalization list. Only the ADR entry
is correct, and only because that kind was always four digits.

Three surfaces print it:

- `aiwf schema` text — the `id format:` line, once per kind.
- `aiwf schema --format=json` — the `id_format` field.
- The `frontmatter-shape` finding message, via `entity.ValidateID`'s
  `id %q does not match %s format`, reached from `internal/check` and again
  from `internal/manifest`.

The regexes are unaffected and should not change: `^E-\d{2,}$` and its siblings
are the permanent read tolerance ADR-0008 commits to. What is stale is the shape
the kernel *advertises*, which names the tolerance floor where it should name
the canonical emit width.

## Why it matters

The verb's stated audience is skill authors and scaffolding tooling — its own
doc comment says they can read the schema once and stop guessing at the
contract. An author who follows it writes a narrow id, which
`entity-id-narrow-width` then reports at error severity in the active tree.

The `frontmatter-shape` message is the worse of the two, because it is advice
given at the moment of failure. An id below the regex floor produces
`does not match E-NN format`, and an operator who complies lands on a width
that is itself an error-severity finding. The `aiwf-check` skill routes readers
here explicitly: a too-short `id:` is malformed rather than narrow, it says, and
`frontmatter-shape` reports it with the expected format.

## Why no rule reaches it

The three width rules are corpus-scoped to markdown: `doc-id-width` over the
configured documentation corpus, `skill-body-id` over the embedded skill and
guidance trees, `body-prose-id` over entity bodies. A Go string literal belongs
to none of them.

`TestPolicy_NarrowIDLiteralsAllowlisted` is the one policy that scans Go source,
and its grep is `"[EMGDC]-[0-9]{1,3}"` — narrow *numeric* literals. A
placeholder-form literal falls outside that grammar. The policy's own doc
comment states its scope using the placeholder spelling while the grammar
comment three lines below states numerics, so the uncovered case reads as
covered.

M-0083 removed this same per-kind enumeration from CLAUDE.md's commitment on
stable ids and pinned its absence with a test. The prose copy was swept; the
table the CLI prints was not.

## Resolution

`IDFormat` stays a single string and is derived, not stored: `entity.IDPrefix`
and `entity.CanonicalPad` already hold the two facts it assembles from, and
`IDPrefix`'s doc comment already directs consumers to call it rather than
re-hardcode a prefix — which the `schemas` table did. `SchemaForKind` and
`AllSchemas` fill the struct field, so the table cannot carry a stale copy.

Splitting into a canonical-emit shape and a separate accepted-input shape was
rejected. Both consumers want the canonical shape. `aiwf schema` publishes what
an author should write, and the `frontmatter-shape` message is only ever reached
by an id below the kind's digit floor — a narrow-but-legal id like `E-07` passes
the regex and is caught by `entity-id-narrow-width` instead, so the tolerated
grammar is never the thing the operator needs at that seam. Naming the tolerance
floor there is what built the trap. A second field would also be a hand-copy of
`idPatterns`, which is the failure this gap records, at a new site.

Contracts were considered as the oracle and do not fit: `aiwf contract verify`
runs a validator over stored fixtures, so it compares a schema against curated
examples and never sees live output. Deriving the value removes the drift it
would have been asked to detect.

## Where to fix

- `internal/entity/entity.go` — the six `IDFormat` fields in the `schemas`
  table, and `ValidateID`'s error message if the split lands.
- `internal/cli/schema/schema_test.go`,
  `internal/cli/integration/schema_cmd_test.go`,
  `internal/manifest/manifest_test.go` — three expectations pin the narrow
  spelling and move with the literals.
- `internal/policies/narrow_id_sweep_test.go` — its doc comment misstates the
  grammar it enforces; widening the grep to placeholder-form literals is the
  chokepoint that would have caught this.

## Related

- ADR-0008 — the canonical-width policy these literals predate.
- G-0538 — the same seam (Go literals reaching a consumer's terminal) on the
  identity axis rather than the width axis.
- G-0517 — narrow citations in the design docs; a documentation corpus, not
  source.
