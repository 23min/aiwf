---
id: G-0588
title: The import verb ships with no consumer and its spec is archived
status: open
---
## What's missing

`aiwf import` ships. It is a live top-level verb — `aiwf import <manifest>`
bulk-creates entities from a YAML or JSON manifest in one atomic commit — and
its documented worked example still validates against the current binary,
including the narrow legacy ids ADR-0008 says parsers tolerate.

Nothing consumes it. Every project that was going to be imported has been, and
no legacy project remains. Its contract specification moved to
`docs/archive/migration/` on that basis, so aiwf now ships a verb whose only
format reference sits in the tier that tells a reader its cross-references are
not maintained.

Both halves are individually right and they do not hold together. Either the
verb retires, and the archived specification becomes the record of something
withdrawn, or a future consumer exists and the specification belongs back in the
normative tier.

## Why it matters

A consumer meets the inconsistency without warning. `aiwf --help` offers the
verb, `aiwf import --help` documents its usage, and the format reference a
manifest producer actually needs is filed where the project says not to rely on
it.

Retiring a verb is a kernel decision rather than a cleanup. It needs the
reversal question answered — what undoes a retirement, and what a consumer
holding a manifest does afterwards — and it reaches the command tree, the
skills-policy coverage rule, `internal/manifest/`, and the import scenarios in
the test suite. That is why it is filed rather than folded into the tier change
that surfaced it.

Whichever way it goes, the specification's tier follows the verb's status. They
must not be decided separately.
