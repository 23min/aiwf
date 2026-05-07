---
id: E-17
title: AC body prose chokepoint (closes G-058)
status: active
---

## Rescope note (per G-063, 2026-05-07)

This epic was originally scoped AC-only as "AC body prose chokepoint." **Rescoped 2026-05-07** to a kind-generalized "entity body prose chokepoint" covering all entity kinds whose body carries load-bearing prose. The rescope was forced by [G-063](../../gaps/G-063-no-defined-start-epic-ritual-epic-activation-is-a-deliberate-sovereign-act-with-preflight-optional-delegation-but-kernel-treats-it-as-a-one-line-fsm-flip.md) sub-decision #4: the start-epic preflight needs a "non-empty epic body" check, and the cleanest implementation is one rule (`entity-body-empty`) parameterized by kind rather than two parallel rules.

The frontmatter `title` field still says "AC body prose chokepoint (closes G-058)" — the slug has been renamed but the title cannot be mutated until [G-065](../../gaps/G-065-no-aiwf-retitle-verb-scope-refactors-that-change-an-entity-s-or-ac-s-intent-leave-frontmatter-title-fields-permanently-misleading-only-slug-rename-is-supported.md) (no `aiwf retitle` verb) closes. The body below carries the authoritative scope.

The epic still primarily closes [G-058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) (the AC body chokepoint is now one case of the generalized rule) and additionally satisfies the body-empty preflight requirement that G-063's start-epic ritual will need.

## Goal

Make non-empty body prose a kernel-enforced property across entity kinds. The design has always specified that each entity's load-bearing body sections carry prose detail (description, examples, edge cases, references — see [`docs/pocv3/plans/acs-and-tdd-plan.md:22`](../../../docs/pocv3/plans/acs-and-tdd-plan.md), [`docs/pocv3/design/design-decisions.md:139`](../../../docs/pocv3/design/design-decisions.md)) but no chokepoint enforces it: `aiwf add` verbs scaffold bare headings, existing coherence rules only check heading↔frontmatter pairing, and the `aiwf-add` skill never prompts the operator to fill the body in. Result is repo-wide skimping — every milestone M-049..M-061 shipped with empty AC bodies, many entities ship with bare body sections. See [G-058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) for the AC-side evidence.

End state: `aiwf check` reports `entity-body-empty` for any entity whose load-bearing body section is empty (warning by default; error under `aiwf.yaml: tdd.strict: true`); `aiwf add ac` accepts `--body-file` per AC so the body lands in the same atomic commit (the analogous flag for other `aiwf add` verbs is captured as a follow-up gap); the `aiwf-add` skill names "fill in the body" as a required follow-up step across all kinds. Together these make the design intent mechanically enforceable rather than aspirational, for every kind that ships load-bearing body prose.

## Scope

- `aiwf check` finding `entity-body-empty` parameterized over kind (warning by default; error under `tdd.strict: true`). Shares the `tdd.strict` config field with [E-16](../E-16-tdd-policy-declaration-chokepoint-closes-g-055/epic.md)'s `milestone-tdd-undeclared` (single source of truth for the project's TDD strictness posture). Per-kind body-section list is hardcoded in the rule (epic: Goal/Scope/Out of scope; milestone: Goal/Approach/Acceptance criteria; AC: `### AC-N — <title>` body; gap: What's missing/Why it matters; adr: Context/Decision/Consequences; decision: Question/Decision/Reasoning; contract: Purpose/Stability). See [M-066](M-066-aiwf-check-finding-entity-body-empty.md) for the full design.
- `aiwf add ac` accepts `--body-file <path>` per AC for in-verb body scaffolding. Multi-AC form pairs `--body-file` positionally with `--title` (or accepts a directory of `AC-N.md` files). See [M-067](M-067-aiwf-add-ac-body-file-flag-for-in-verb-body-scaffolding.md).
- `aiwf-add` skill update: name "fill in the body" as the required next step across all `aiwf add` paths when `--body-file` was not used (or is not yet available for that kind); document the body shape per kind. See [M-068](M-068-aiwf-add-skill-names-fill-in-body-as-required-next-step.md).

## Out of scope

- **`--body-file` for non-AC entity-creation verbs.** Generalizing the flag to `aiwf add epic`, `aiwf add milestone`, `aiwf add gap`, `aiwf add adr`, `aiwf add decision`, `aiwf add contract` is a clear next step but is captured as a separate follow-up gap (filed alongside this rescope). The check rule fires for those kinds; operators currently rely on `aiwf edit-body` to fill the body post-add. AC has the highest authoring volume; the asymmetry is acceptable in the short term.
- **Retroactive backfill of historical bodies.** Existing milestones (M-049..M-061) and other entities surface as warnings under the new check finding; this epic does not write the prose for them. Authors who care can backfill kind-by-kind; that work is not blocking.
- **Schema for body content.** The body is markdown prose, and the kernel principle "prose is not parsed" applies (per `acs-and-tdd-plan.md:197`). The check rule asserts presence, not structure. No grammar, no required keywords, no required sub-headings beyond the per-kind body-section list.
- **Promote-time guard.** Refusing AC `open -> met`, milestone `draft -> in_progress`, or epic `proposed -> active` when bodies are empty is overkill once the check finding fires — same reasoning as G-055's deferred promote-time guard. YAGNI; revisit if drift shows up. (G-063's preflight is a different mechanism — skill-level conversation, not verb-level refusal.)
- **Title-length validator changes.** The existing 80-char limit on entity / AC titles is fine; this epic puts the detail in the body, not the title.
- **Render-side enforcement.** The I3 governance render reads what's written; if the body is empty, the page renders empty. Surfacing "this entity has an empty body" in the render is downstream of the check finding and not in this epic's scope.
