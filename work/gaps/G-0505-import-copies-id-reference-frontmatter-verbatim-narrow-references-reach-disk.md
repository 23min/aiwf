---
id: G-0505
title: import copies id-reference frontmatter verbatim; narrow references reach disk
status: open
---
## What's missing

`buildEntityFromEntry` (`internal/verb/import.go`) copies a manifest entry's
frontmatter map wholesale and overwrites exactly one key — `id`. Every other
field arrives on disk at whatever width the manifest declared, including the
fields that hold entity-id *references*:

- `parent`
- `depends_on`
- `superseded_by`
- `discovered_in`
- `addressed_by`

So a manifest declaring an epic as `` `E-11` `` with a milestone whose `parent`
reads `` `E-11` `` produces an epic stored at canonical width and a milestone
pointing at it by a width nothing on disk uses.

`prior_ids` is the deliberate exception and must not be canonicalized: recording
the id an entity previously carried — narrow, because that is what it was — is
the field's entire purpose. Any fix decides the include-set explicitly rather
than sweeping every id-shaped string.

`addressed_by_commit` holds SHAs, not entity ids, and is out of scope.

## Why it matters

Nothing fails. Every consumer of these fields resolves through a canonicalizing
lookup, so the reference is found, `aiwf check` reports nothing, and the
divergence never surfaces as an error. That is what makes it durable: a narrow id
sits on disk indefinitely, reading as current shape to anyone — human or model —
who opens the file. That reading is the cost the narrow-id work exists to remove;
a reference field is simply a place it was never looked for.

Distinct from the minting defect that G-0481's creation-path audit covers, which
is about allocating a *new* id for a *new* entity and is closed. This is the
reference axis: an id that already belongs to some entity, written down at a width
the writer chose. Different property, different fix, and this one carries a
carve-out the minting fix does not.

Import is the only route that reaches these fields from untrusted external input.
The structured verbs that write them — the dependency declarer, the gap closer,
the supersession recorder — validate their arguments against existing entities
first, so they cannot introduce a width no entity uses.

## Resolution shape

Canonicalize the id-reference fields as the manifest is materialized, with
`prior_ids` excluded by name rather than by accident. The include-set is the
decision worth recording: it is currently five fields, and a sixth added to the
entity schema later inherits the defect silently unless the set is derived from
something the schema itself carries.

Pin it with the same output-property shape the minting fix used — drive the
import and measure what lands on disk, since a static rule cannot recognize which
string literals are ids. The existing invariant test already walks every entity's
own id and path; extending it to reference fields is the natural home, and it is
where the assertion that found this defect was removed.

Worth settling alongside the narrow-id policy work rather than in isolation, since
the question "which id-bearing fields carry canonical width, and which record
history" is the same question that governs read tolerance.

## Where to fix

- `internal/verb/import.go` — `buildEntityFromEntry`, where the frontmatter map is
  copied and only `id` is overwritten.
- `internal/verb/canonical_width_invariant_test.go` — the invariant test whose
  reference-field assertion was removed when this gap was filed.
- `internal/entity/entity.go` — the frontmatter struct that defines which fields
  hold references.

## Related

- G-0481 — the narrow-id audit; its creation-path table covers minting, which is
  closed, and does not cover the reference axis this gap opens.
