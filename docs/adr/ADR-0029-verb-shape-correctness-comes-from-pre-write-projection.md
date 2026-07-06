---
id: ADR-0029
title: Verb shape correctness comes from pre-write projection
status: proposed
---

# ADR-0029 — Verb shape correctness comes from pre-write projection

> **Date:** 2026-07-06 · **Decided by:** human/peter

## Context

`verb.Apply` (`internal/verb/apply.go`) performs zero content validation: it runs every `OpMove`/`OpWrite` from a `*Plan` as a pure filesystem operation, then commits. Nothing in `Apply` checks that a written entity's frontmatter has a valid `id`, a non-empty `status`, or its kind's required fields. A reader encountering this for the first time can reasonably ask: what stops a verb from writing a shape-invalid entity into git history?

The serialization path itself provides no such guarantee. `entity.Serialize` YAML-marshals an `Entity` struct with plain `string` fields; it will marshal an empty `Status` or a missing `id` exactly as readily as valid ones. The Go type system does not make an invalid entity unrepresentable.

The actual guarantee is architectural, not type-level: most mutating verbs (`add`, `promote`, `edit-body`, `rename`, `reallocate`, `cancel`) build a *projected* copy of the tree with the verb's change already applied, then run the full `check.Run` — including `frontmatterShape`, the rule that validates `id`, `status`, and per-kind required fields — against that projection (`internal/verb/common.go`'s `projectionFindings`). If the projection fails the check, the verb returns `Findings` and produces no `*Plan` at all; `verb.Apply` is never invoked, and `cliutil.FinishVerb` independently re-gates on the same condition before calling `Apply`. So the constraint lives at the validate-then-write boundary — before a `Plan` exists — not inside `Apply`, and not in the entity type itself.

This was independently confirmed while scoping M-0186/AC-6 (E-0045): M-0186 retrofit `verb.Apply`'s commit mechanism from `git commit` (which fires git hooks) to `git commit-tree` + `update-ref` (which fires none). It would be easy to assume this removed a shape-correctness backstop, since the old pre-commit hook ran `aiwf check --shape-only`. It did not: `--shape-only` only ever ran `TreeDiscipline` (stray-file tree-layout checking), never `frontmatterShape` — so the hook's removal from the verb-commit path has no bearing on frontmatter-shape enforcement, which was never wired through it. The two guarantees are, and always were, independent.

The guarantee is not perfectly uniform. `SetArea` and `RenameArea` skip `projectionFindings` entirely, serializing directly after mutating only the `Area` field — safe today only because `Area` is a field `frontmatterShape` never inspects, not because the gate ran and passed. `Rewidth` opts out explicitly and documents relying on `aiwf check` at the pre-push boundary as its backstop instead. No prior ADR, gap, or decision recorded any of this; it existed only as a package doc comment in `internal/verb/verb.go` and a policy docstring in `internal/policies/verbs_validate_then_write.go`.

## Decision

Entity shape correctness for the verb-commit path is guaranteed by a **pre-write projection check**, not by type-safe construction and not by any check running at or after commit time:

- A verb that mutates entity content builds a projected tree reflecting its change, then runs the full `check.Run` (including `frontmatterShape`) against that projection *before* producing a `*Plan`. A shape violation returns `Findings` with no `Plan` — nothing is written, and `verb.Apply` never runs.
- `verb.Apply` itself performs no content validation. It is deliberately "dumb": pure mechanical write-then-commit, trusting that any `*Plan` it receives already passed its producing verb's projection check. This is what keeps `Apply` a single, uniform seam (M-0186/AC-3, AC-5) rather than a place every verb must re-implement its own validation.
- `entity.Serialize` and the `Entity` struct carry no shape guarantees of their own; they are trusted to marshal whatever they are given. The guarantee is entirely in the calling discipline, not the type.
- The known exceptions — `SetArea`/`RenameArea` (skip the gate; safe only because they never touch a shape-relevant field) and `Rewidth` (opts out explicitly, relies on pre-push) — are accepted as-is. They are not violations of this decision; they are documented instances where a verb's own reasoning substitutes for the general gate.

## Consequences

- A new mutating verb that writes entity content must call `projectionFindings` (or equivalent: project the change, run `check.Run`, gate on `check.HasErrors`) before returning a `*Plan`. A verb that skips this and hand-waves "my change can't be shape-invalid" needs the same kind of argument `SetArea`/`RenameArea`/`Rewidth` document today, not silence.
- `verb.Apply` staying validation-free is intentional and should not be "fixed" by adding a defensive shape check inside it — that would duplicate the real gate and mask a bug in whichever verb skipped its own projection check, rather than surfacing it at the verb layer where it belongs.
- The pre-push `aiwf check` remains the authoritative backstop regardless of this decision — it catches any shape violation that reaches disk by any path (a verb bug, a hand-edit, a test harness bypassing verb construction). This decision explains why that backstop is rarely the *first* line of defense for verb-authored entities, not why it's unnecessary.
- If `SetArea`, `RenameArea`, or `Rewidth` are ever extended to touch a shape-relevant field, they must adopt the projection gate at that point — their current exemption is scoped to their current, narrow field-set, not a general precedent for skipping the gate.

## References

- Linked epics or milestones: `E-0045`, `M-0186` (AC-6 named the gap this ADR documents)
