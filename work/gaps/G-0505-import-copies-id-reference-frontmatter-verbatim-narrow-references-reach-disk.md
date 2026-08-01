---
id: G-0505
title: import copies id-reference frontmatter verbatim; narrow references reach disk
status: open
---
## What's missing

`buildEntityFromEntry` (`internal/verb/import.go`) copies a manifest entry's
frontmatter map wholesale and resolves only two keys — `id` and, for a milestone,
`parent`. Every other field arrives on disk at whatever width the manifest
declared, including the fields that hold entity-id *references*:

- `depends_on`
- `superseded_by`
- `discovered_in`
- `addressed_by`

So a manifest declaring a milestone whose `depends_on` names `` `M-001` `` stores
that spelling next to a sibling recorded as `` `M-0001` ``, naming the same entity
by a width nothing on disk uses.

`parent` is resolved because it is load-bearing in ways the others are not: it
selects the directory the child is written into, and the guards that walk an
epic's children compare it literally. Both consequences were measured, and both
are closed. The remaining four are read only through canonicalizing lookups, so
this gap is about the stored spelling rather than about behavior.

`prior_ids` is the deliberate exception and must not be canonicalized: recording
the id an entity previously carried — narrow, because that is what it was — is
the field's entire purpose. Any fix decides the include-set explicitly rather
than sweeping every id-shaped string.

`addressed_by_commit` holds SHAs, not entity ids, and is out of scope.

## Why it matters

Nothing fails. Every consumer of the four remaining fields resolves through a
canonicalizing lookup, so the reference is found, `aiwf check` reports nothing,
and the divergence never surfaces as an error. That is what makes it durable: a
narrow id sits on disk indefinitely, reading as current shape to anyone — human
or model — who opens the file. That reading is the cost the narrow-id work exists
to remove; a reference field is simply a place it was never looked for.

The `parent` case is the caution worth carrying into any fix here: a field that
looks like a pure lookup key can turn out to be consumed structurally, and the
consequence surfaced only when an id it pointed at changed width. Confirm how
each of the four is actually read before deciding the field is inert.

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

Resolve the remaining id-reference fields as the manifest is materialized, with
`prior_ids` excluded by name rather than by accident. The include-set is the
decision worth recording: it is currently four fields, and a fifth added to the
entity schema later inherits the defect silently unless the set is derived from
something the schema itself carries.

Resolve rather than canonicalize, the way `parent` does. Canonicalizing the
declared spelling is wrong whenever the referent is an entity resident at legacy
width: it would point the reference at a width that entity does not carry. The
target is the id the referent actually holds.

Pin it with the same output-property shape the minting fix used — drive the
import and measure what lands on disk, since a static rule cannot recognize which
string literals are ids. The existing invariant test already walks every entity's
own id, its path, and its containment; extending it to the remaining reference
fields is the natural home.

Worth settling alongside the narrow-id policy work rather than in isolation, since
the question "which id-bearing fields carry canonical width, and which record
history" is the same question that governs read tolerance.

## Where to fix

- `internal/verb/import.go` — `buildEntityFromEntry`, where the frontmatter map is
  copied and only `id` and `parent` are resolved. `lookupEpicDir` in the same file
  is the worked example of resolving a reference to the referent's actual id.
- `internal/verb/canonical_width_invariant_test.go` — the invariant test the
  remaining fields extend, alongside its width and containment assertions.
- `internal/entity/entity.go` — the frontmatter struct that defines which fields
  hold references.

## Related

- G-0481 — the narrow-id audit; its creation-path table covers minting, which is
  closed, and does not cover the reference axis this gap opens.
