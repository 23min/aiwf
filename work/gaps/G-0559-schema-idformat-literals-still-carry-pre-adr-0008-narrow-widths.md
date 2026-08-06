---
id: G-0559
title: Schema IDFormat literals still carry pre-ADR-0008 narrow widths
status: open
priority: medium
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

## Resolution shape

One design question settles first: whether `IDFormat` stays a single string
widened to canonical, or splits into a canonical-emit shape and a separate
accepted-input shape for the two consumers to draw from.

Whichever lands, `entity.IDPrefix` and `entity.CanonicalPad` already hold the
two facts the string is assembled from, and `IDPrefix`'s doc comment already
directs consumers to call it rather than re-hardcode a prefix — which the
`schemas` table does.

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
