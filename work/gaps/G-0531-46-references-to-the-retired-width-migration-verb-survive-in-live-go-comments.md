---
id: G-0531
title: 46 references to the retired width-migration verb survive in live Go comments
status: wontfix
priority: low
discovered_in: M-0290
---
## Problem

M-0290 deleted the width-migration verb. Forty-six references to it
survive in Go comments across twenty-three files the milestone never
opened.

They are not inert. Several assert something about the current code
that is false:

- `internal/cli/cliutil/apply.go` names the verb as a live multi-Plan
  caller alongside `archive` and `import`.
- `internal/cli/integration/trailer_shape_test.go` carries a block
  explaining why the verb is deliberately not covered there, and points
  at its integration tests under a path that no longer exists.
- `internal/cli/integration/archive_cmd_test.go` describes three of its
  own assertions as mirroring tests that were deleted.
- `internal/cli/render/resolver.go` and `internal/cli/importcmd/`
  describe a canonicalization pass as future work the verb will perform.

A maintainer following any of these looks for a symbol or a file that is
not there.

## Why it survived the milestone

The retirement swept every surface with a check behind it — shipped
skills, normative docs, the policy ledgers whose staleness is detected —
and stopped there. The ledgers that were missed are exactly the four
with no staleness detection, which is the same shape at a smaller
scale: the sweep tracked what the tests enforced rather than what the
deletion invalidated.

Nothing catches this class. `comment-history-attrition` matches
past-tense narration phrases, not references to symbols that stopped
existing, and it is diff-scoped, so a comment on an untouched line is
never examined.

## Direction

Two separable pieces:

- **The sweep itself** — mechanical, one pass, no design decisions. Each
  comment either drops the verb from a list, or states what is true now.
- **A detector, if one is cheap.** A reference to a deleted Go
  identifier is findable; a reference to a deleted *verb name* in prose
  is not, without a list of retired verbs to check against. Whether that
  list is worth maintaining is the open question — a ban that costs once
  is worth more than a per-subject mandate, and a retired-verb list is
  the latter.

## Not this gap

Shipped surfaces and normative docs are clean and pinned by
`TestM0290_AC4_*`. This is the tier below: comments in live source, with
no consumer reach.

## Provenance

Found 2026-08-03 by the wrap review for M-0290. The blocking subset —
false "shared with" claims, pointers to deleted files, and stale
allowlist keys, all in files the milestone edited — was fixed inside the
milestone. This gap is the residue in files it never opened.
