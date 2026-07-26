---
id: G-0228
title: Typed Status enum in internal/entity
status: addressed
priority: medium
addressed_by_commit:
    - 85a61c3963e1cb4150f8271fe8719fac84e19dbe
---
## What's missing

`type Status string` in `internal/entity/`. Today the status constants (`StatusActive`, `StatusDone`, `StatusDraft`, …) are bare untyped `string`, and `ValidateTransition(k Kind, from, to string)` / `IsTerminal(k Kind, s string)` take plain `string`. Retype the constants as `Status`-valued and propagate through the `transitions` map (keys and values), the transition / terminal signatures, the verb signatures that take or return a status, and frontmatter decode (the YAML tag stays `string`; the decoded field becomes `Status`).

What this buys, stated honestly: **argument-position safety and self-documenting signatures.** `ValidateTransition(k Kind, from, to string)` cannot today stop a caller transposing the two status arguments or passing an arbitrary string where a status belongs; `type Status string` makes those a compile error and makes each signature state what it means.

What it does **not** buy is comparison-literal safety — that already exists. `PolicyEnumLiteralAdoption` fails CI on any `ac.Status == "open"` / switch-case literal in production code, forcing the `entity.Status*` constants at every comparison site. So the retype is belt-and-suspenders over an existing chokepoint, not a new guarantee.

## Priority

This is the lowest-value item in the code-health cluster. The protection a reader would assume it adds (typo / comparison safety) is already delivered by `PolicyEnumLiteralAdoption`; what remains is argument-position safety and documentation, bought with a wide mechanical cascade through Strong-verdict code. Worth folding in when `internal/entity/` is already open for another reason — not worth a dedicated milestone ahead of the layering fix (G-0227) or the verb-UX uniformity (G-0230).

## Out of scope

- **`type FindingCode string`** on `internal/check`'s `Finding.Code`. The JSON wire tag stays `string` regardless, and the internal comparison surface is small — modest value for the churn.
- **Extending `codes.Code` to every finding code.** The typed descriptor (ID + Class) already covers the codes whose Class the legality enumeration consumes; migrating the rest is a large, vaguely-bounded "one pass enumerating them" with no consumer waiting on it — YAGNI.
- **`manifest.Entry.Kind` / `CommitSpec.Mode`, `workflows/spec` predicate fields, `OutputFormat.Format`.** Low-traffic tail fields; retype one only if it lands in a caller's way, not as a sweep.

## Source

`docs/archive/pocv3/health-scorecard-2026-06-04.md` §B1 (typed-Status move).
