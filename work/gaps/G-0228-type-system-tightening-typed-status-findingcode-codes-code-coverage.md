---
id: G-0228
title: 'Type-system tightening: typed Status, FindingCode, codes.Code coverage'
status: open
---
## What's missing

Four closed-set string fields at module boundaries that exist as untyped `string` despite the kernel having a closed enumerated set defined elsewhere — compensated today by runtime checks + `PolicyEnumLiteralAdoption`, but `type X string` would make the discipline structural instead of runtime-policed.

Specific retypings:

1. **`type Status string`** in `internal/entity/`. Retype `StatusActive`, `StatusDone`, `StatusDraft`, … as `Status`-valued. Propagate through:
   - `ValidateTransition(kind Kind, from, to Status) error`
   - `IsTerminal(kind Kind, s Status) bool`
   - the `transitions` map (keys/values both become `Status`)
   - every verb signature that takes/returns a status
   - frontmatter decode (the YAML tag stays string; the decoded field is `Status`)
2. **`type FindingCode string`** in `internal/check/`. Retype `Finding.Code` and the hint-table keys. Drops the bare-string at the most-publicly-emitted surface (JSON envelope findings).
3. **Extend `codes.Code` typed-descriptor pattern** to every kernel finding code — today only a subset of codes participate in the typed descriptor; the rest live as bare string constants. One pass enumerating them and migrating.
4. **`manifest.Entry.Kind`, `manifest.CommitSpec.Mode`** — promote to named types matching the existing constant lists.
5. **`workflows/spec` Predicate fields, `OutputFormat.Format`** — same treatment.

## Why it matters

Bare-string closed sets at module boundaries are the B1 smell the rubric calls out specifically. The runtime checks and `PolicyEnumLiteralAdoption` catch the obvious "did you misspell `'open'` as `'opne'`" case at CI time — but they don't catch the case where a new caller takes the field as `string` and rolls their own bare-string comparison against a literal. `type Status string` makes the comparison literal a compile-error rather than a policy-failure-later.

## Why this is one gap, not five

The retypings cascade through callers; doing them in one milestone with one CI cycle is cheaper than five sequential rolls. Each retype is a 1-day AC at most; the bundle is a 1-milestone scope.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §B1 (all three recommended moves; `smells_found` list).
