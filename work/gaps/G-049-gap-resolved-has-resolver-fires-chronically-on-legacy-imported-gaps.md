---
id: G-049
title: gap-resolved-has-resolver fires chronically on legacy-imported gaps
status: addressed
addressed_by_commit:
  - de39e01
---

## What's missing

The `gap-resolved-has-resolver` rule (`internal/check/...`) fires a warning whenever a gap is `addressed` but `addressed_by:` is empty. The rule's intent is sound — a closed gap should name the milestone that resolved it so the audit chain `G-NNN → addressed_by → M-NNN` is traversable. But it doesn't account for two real cases:

1. **Bulk-imported legacy gaps.** G38's bulk import (commit `4b89c36`) brought 47 historical gaps into the entity tree. Most were closed by specific commits (matrix references like `[x] 4ec5d84`), not by milestones. There were no milestones at the time the gaps closed, and there are still no milestones in the kernel tree today. `addressed_by:` was deliberately left empty per the user's instruction — commit SHAs aren't entities. Result: 43 chronic warnings on every `aiwf check` and `aiwf status`.

2. **Consumers that don't track milestones.** Some consumers (the kernel itself, repos doing lightweight gap-tracking only) may never adopt the epic/milestone surface. The rule assumes every gap has a milestone-shaped resolver, but that's not the only valid mental model.

## Why it matters

Chronic-noise warnings train consumers to ignore the row that exists to flag *real* problems. Same shape as G47's pin-skew chronic noise — 43 advisory rows on every `aiwf check`/`aiwf status` invocation, none actionable, all expected. The framework's signal-to-noise drops; over time consumers stop reading the warning list.

## Proposed fix

**Lean: weaken the rule for legacy-imported gaps via an `imported_at:` frontmatter stamp.**

`aiwf import` learns to stamp `imported_at: <ISO 8601 date>` (or `imported_via: bulk`) on each entity it creates. The `gap-resolved-has-resolver` rule consults the field: if present, the gap is exempt from the "must have addressed_by" requirement. New gaps (added via `aiwf add gap` and later promoted to `addressed`) still get the warning — that's the desired behavior for forward work.

Alternatives considered:

- **Allow `addressed_by_commit:` alongside `addressed_by:`.** Schema change to the gap kind; lets a commit SHA serve as a valid resolver. More flexible long-term but bigger surface; the rule has to recognize either field. Defer until a real consumer asks.
- **Suppress the rule when the tree has no milestones.** Heuristic; brittle once the consumer adds even one milestone for unrelated work. Reject.
- **Move addressed_by to optional, drop the warning.** Defeats the audit-chain promise. Reject.

Severity: **Low**. UX-only nag, not a correctness issue. But it landed on day one of dogfooding the kernel — the noise is real and immediate, and the lean fix is small.
